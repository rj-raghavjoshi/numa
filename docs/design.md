# Design

Reference documentation for the structure of this package. For the *reasoning*
behind the tuned loops, read [learn/](learn/) instead; this file is for people
modifying the code.

---

## File layout

```
numa/
├── go.mod
├── numa.go                 thin facade: forwards every exported name to vec
├── numa_test.go            verifies the forwarding (argument order, panics, NaN)
│
├── vec/                    the implementation package
│   ├── doc.go              package documentation (godoc entry point)
│   │
│   ├── sum.go              Sum, Mean
│   ├── dot.go              Dot
│   ├── sumsq.go            SumSq
│   ├── min.go              Min
│   ├── max.go              Max
│   ├── minmax.go           MinMax
│   ├── nan.go              NaN policy + hasNaN
│   ├── add.go              Add, AddTo, AddScalar, AddScalarTo
│   ├── sub.go              Sub, SubTo
│   ├── mul.go              Mul, MulTo
│   ├── scale.go            Scale, ScaleTo
│   ├── abs.go              Abs, AbsTo, and the abs() helper
│   ├── elementwise.go      shared length checks + the elementwise overview
│   │
│   ├── arch.go             (no code) documents the dispatch pattern
│   ├── arch_arm64.go       //go:build arm64
│   ├── arch_amd64.go       //go:build amd64
│   ├── arch_generic.go     //go:build !arm64 && !amd64
│   │
│   ├── sum_test.go         one test file per source file
│   ├── dot_test.go
│   ├── sumsq_test.go
│   ├── scan_test.go        Min/Max/MinMax together (they share the NaN policy)
│   ├── elementwise_test.go Add/Sub/Mul/Scale/Abs + aliasing + panics
│   ├── helpers_test.go     shared refs, boundaryLengths, closeEnough
│   ├── bench_test.go       benchmarks + naive baselines
│   └── ...
│
└── docs/
    ├── learn/              three-part tutorial, read in order
    │   ├── index.md
    │   ├── 01-fundamentals.md
    │   ├── 02-the-buildup.md
    │   └── 03-the-truth.md
    ├── design.md           this file
    ├── benchmarks.md       measured results
    └── next-steps.md       open decisions
```

### Why a facade, and why a subpackage

**Go has no partial packages.** One directory is one package, so a subdirectory
cannot contribute names to `numa`. If the implementations live in `vec/`, then
`numa.Sum` must be a forwarding function.

**Why keep the facade at all?** Because `numa.Sum(xs)` reads better than
`vec.Sum(xs)` and costs one function call. Go inlines most of these forwarders, so
the cost is usually zero — but it is not *guaranteed* to be zero, and that
distinction matters in a package whose entire purpose is speed. The facade
documentation says so, and points hot-loop callers at `vec` directly.

**The facade is a maintenance cost.** Every new exported function needs three
things: the implementation in `vec/`, a forwarder in `numa.go`, and a test in
`numa_test.go`. `numa_test.go` exists specifically because a forwarder can be
wrong in ways the implementation's own tests cannot catch — swapping argument
order, dropping a length check, losing the NaN policy.

### Why operations are grouped the way they are

**One file per operation, not one file per family.** `sum.go` holds `Sum` and
`Mean`; `dot.go` holds `Dot`. An earlier layout grouped all reductions into a
single `reduce.go`, which read well at four functions and would not at twenty. Per
operation scales, and `numa.Sum` maps directly onto `vec/sum.go`.

`scan_test.go` is the one exception: `Min`, `Max` and `MinMax` share the NaN policy
and the same latency-bound reasoning, so testing them together keeps that shared
contract in one place.

**Architecture files are grouped by *machine*.** `arch_arm64.go` holds every
operation's ARM64 tuning, because the tuning parameters (accumulator count, stride)
are properties of the machine, not of the operation. Adding an operation means
touching all three arch files together, which is the correct coupling.

**Tests mirror the source files.** `sum_test.go` tests `sum.go`. The shared helpers
(`closeEnough`, `boundaryLengths`, `randomSlice`, the reference implementations)
live in `helpers_test.go` rather than in whichever file happened to be written
first; Go makes package-level test identifiers visible across all `_test.go` files
in the package, so they can be used from anywhere without an import.

---

## The dispatch pattern

Every operation has a public entry point and a private architecture-specific
implementation:

```go
// vec/sum.go — public, architecture-independent
func Sum(xs []float64) float64 {
	if len(xs) == 0 {
		return 0.0
	}
	return sumArch(xs)
}
```

```go
// vec/arch_arm64.go — private, build-tagged
func sumArch(xs []float64) float64 { /* ARM64 tuning */ }
```

The public function owns:

* **Edge cases.** Empty and nil inputs, length mismatches, NaN policy.
* **The contract.** What the function guarantees, documented once.

The `*Arch` function owns:

* **The hot loop only.** It may assume a validated, non-empty input.

This split matters for performance: it keeps the branch on `len(xs) == 0` out of
the loop and out of the per-call path for the common case.

### Naming

| suffix | meaning |
|---|---|
| `sumArch`, `dotArch`, `minArch`, … | private, one per architecture, no edge-case handling |
| `scanMin2`, `scanMin4` | private scan helpers, named for the accumulator count |
| `naiveSum`, `naiveDot` | benchmarks only, portable baseline |

The `2` / `4` suffix on scan helpers encodes the accumulator count, which is the
one thing that differs between the ARM64 and x86-64 versions.

---

## How to add an operation

Say you want `Variance`. The order matters — test before implementation. There are
six steps, and step 6 is the one that is easy to forget because nothing fails if you
skip it.

### 1. Write the public function in its own file in `vec/`

Create `vec/variance.go`. One operation per file; see the layout rationale above.

```go
// vec/variance.go
package vec

func Variance(xs []float64) float64 {
	if len(xs) < 2 {
		return 0.0
	}
	return varianceArch(xs)
}
```

### 2. Add `varianceArch` to all three architecture files

The generic one is the reference — write it first, plainly:

```go
// vec/arch_generic.go
func varianceArch(xs []float64) float64 {
	mean := sumArch(xs) / float64(len(xs))
	var total float64
	for _, v := range xs {
		d := v - mean
		total += d * d
	}
	return total / float64(len(xs))
}
```

Then the tuned ones. **You cannot skip this step** — if `varianceArch` is missing
from any architecture file, the package fails to build on that architecture. The
compiler will tell you, which is the good news.

### 3. Decide which family it belongs to

| family | characteristic | tuning approach |
|---|---|---|
| reduction | one output, loop-carried chain | K accumulators + paired accumulate + tree |
| scan | one output, latency-bound | K independent partial scans, no pairwise trick |
| elementwise | one output per input | wide unroll, saturate load/store |
| **multipass** | **reads the input more than once** | **reduce passes before tuning the loop** |

`Variance` is *multipass*: it needs the mean before it can accumulate deviations.
No amount of loop tuning fixes reading the input twice. The real fix is a one-pass
algorithm (Welford's), which is a different piece of work. See
[next-steps.md](next-steps.md).

### 4. Write the tests

Create `vec/variance_test.go`. Copy the structure from an existing test file. At
minimum:

* agreement with a reference across `boundaryLengths()`
* a per-element perturbation test (catches skipped/double-counted elements)
* empty, nil, single-element, and mismatched-length cases
* any documented contract (NaN policy, aliasing)

Add a reference implementation to `helpers_test.go` if the existing ones don't
cover it.

### 5. Add benchmarks, with a baseline

A benchmark without a naive baseline proves nothing. Add both `BenchmarkVariance`
and `BenchmarkNaiveVariance` to `bench_test.go`.

### 6. Add the forwarder to `numa.go` and a test for it

**This step is easy to forget because nothing breaks if you skip it** — the
operation simply isn't reachable from the `numa` package, and no test fails.

```go
// numa.go
// Variance returns the variance of xs. See [vec.Variance].
func Variance(xs []float64) float64 { return vec.Variance(xs) }
```

Then add a case to `numa_test.go`. That test file exists precisely because a
forwarder can be wrong in ways the implementation's tests cannot catch: swapped
argument order, a dropped length check, a lost NaN policy.

### 7. Record the numbers

Update [benchmarks.md](benchmarks.md) with the measured result. If the tuning
didn't help, **say so** — that's a finding, not a failure. `Add` gaining 1.27×
instead of ~400 % is the most informative number in that file.

---

## Invariants

Break these and you introduce bugs that the test suite may not catch:

### 1. `limit := n - (stride - 1)`

For a loop body that reads `xs[i]` … `xs[i+stride-1]`:

```go
limit := n - (stride - 1)
```

Getting this wrong is the classic off-by-one. Too small and you drop elements;
too large and you read past the end. The boundary-length tests are specifically
sized to catch this.

### 2. Aliasing safety in `*To` functions

The documented contract is that `dst` may alias `xs` and/or `ys`. This holds
because the loop writes `dst[i]` only after reading `xs[i]` and `ys[i]` in the
same iteration. **A tuned loop that reads ahead of its writes would break this.**
If you restructure an `*To` function to buffer ahead, you must update the
documentation *and* `TestToVariantsAliasing`.

### 3. The NaN check in scans is a second pass

`Min`, `Max` and `MinMax` call `hasNaN`, which walks the input again. This is a
deliberate cost, documented in `vec/nan.go`. If you fold NaN detection into the scan
loop, you change the performance profile and must re-benchmark — the scan is
latency-bound, so adding work to it is not free.

`hasNaN` is written as a non-early-exit boolean map specifically so the compiler
can vectorize it. An early-return version would serialize on a branch.

### 4. Generic implementations stay naive

`arch_generic.go` is the reference used to justify the tuning. **Do not optimize
it.** If it becomes clever, it stops being a baseline, and the benchmarks lose
their meaning.

---

## Testing

```sh
go test ./vec/                                # correctness, native arch
go test ./vec/ -bench=. -benchmem -count=10   # performance
GOARCH=amd64 go test ./vec/                   # cross-check the x86-64 path
go test ./...                                 # both packages, including the facade
GOOS=linux GOARCH=386 go vet .               # generic path compiles
```

`GOARCH=amd64 go test ./vec/` works on Apple Silicon via Rosetta 2. It genuinely
verifies the x86-64 *correctness*; the *timings* it produces are not
representative and should not be quoted as x86-64 performance.

### Why the boundary lengths matter

`boundaryLengths()` returns 0–17, 23–25, 31–33, 63–65, 127–129. These cover:

* shorter than one unrolled stride
* exactly one stride
* one stride plus one
* several strides plus each possible remainder

Random large lengths will pass even when the loop bounds are wrong, because a
stride bug usually drops a whole block rather than everything. **The small
awkward lengths are where the bugs are.**

---

## Further reading

| document | purpose |
|---|---|
| [learn/](learn/) | Why the loops are shaped this way, from first principles |
| [benchmarks.md](benchmarks.md) | Measured results and methodology |
| [next-steps.md](next-steps.md) | Open decisions and unverified assumptions |
