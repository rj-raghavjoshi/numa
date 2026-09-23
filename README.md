# numa

> **numa** is a high-throughput, hardware-aware numerical and statistical
> computing engine written in Go.

Engineered for quantitative modeling, time-series analysis, and algorithmic
trading systems where allocation overhead and floating-point stalls are
unacceptable.

## Documentation

| document | read it if you want to… |
|---|---|
| **[docs/learn/](docs/learn/)** | **understand *why*.** A three-part tutorial starting from "what is a CPU" — no background assumed. Explains the whole codebase through eight attempts at summing a list. |
| [docs/design.md](docs/design.md) | modify the code. The dispatch pattern, how to add an operation, and the invariants to preserve. |
| [docs/benchmarks.md](docs/benchmarks.md) | know *how much*. Measured throughput, reproduction commands, honest caveats including one negative result. |
| [docs/next-steps.md](docs/next-steps.md) | know what is unverified or undecided before relying on this. |

## Install

```sh
go get github.com/rj-raghavjoshi/numa
```

## Packages

| package | import path | use it when |
|---|---|---|
| `numa` | `github.com/rj-raghavjoshi/numa` | almost always. Shorter calls: `numa.Sum(xs)`. |
| `vec` | `github.com/rj-raghavjoshi/numa/vec` | you are calling these in a hot loop and want to skip one forwarding call, or you want the package that actually holds the implementations. |

`numa` is a thin facade: every function in it is a one-line forwarder to `vec`, so
there is no behaviour defined in `numa` that is not defined in `vec`. Go inlines
most of the forwarders, but not guaranteed-ly, which is the only reason to prefer
`vec` directly.

```
numa/
├── numa.go        facade — forwards every name to vec
└── vec/           the implementation
    ├── sum.go     Sum, Mean
    ├── dot.go     Dot
    ├── min.go     Min, Max, MinMax
    ├── add.go     Add, AddTo, ...
    ├── arch_*.go  per-architecture tuning (build tags)
    └── ...
```

## Usage

```go
package main

import (
	"fmt"
	"math"

	"github.com/rj-raghavjoshi/numa"
)

func main() {
	xs := []float64{1, 2, 3, 4, 5}

	fmt.Println(numa.Sum(xs))      // 15
	fmt.Println(numa.Mean(xs))     // 3
	fmt.Println(numa.Dot(xs, xs))  // 55
	fmt.Println(numa.MinMax(xs))   // 1 5
	fmt.Println(numa.Scale(xs, 2)) // [2 4 6 8 10]

	// NaN propagates through the scans, by design:
	fmt.Println(numa.Min([]float64{1, math.NaN()}))
	// NaN
}
```

### API

**Reductions** — walk the slice, return one number:

| function | returns | empty input |
|---|---|---|
| `Sum(xs)` | arithmetic sum | `0` |
| `Mean(xs)` | arithmetic mean | `0` |
| `Dot(xs, ys)` | inner product | `0` |
| `SumSq(xs)` | sum of squares | `0` |
| `Min(xs)` | smallest element | `+Inf` |
| `Max(xs)` | largest element | `-Inf` |
| `MinMax(xs)` | smallest and largest, one pass | `(+Inf, -Inf)` |

**Elementwise** — two forms each. The allocating form returns a new slice of
length `min(len(xs), len(ys))`; the `To` form writes into a caller-supplied
buffer and panics on a length mismatch:

| allocating | in place | computes |
|---|---|---|
| `Add(xs, ys)` | `AddTo(dst, xs, ys)` | `xs[i] + ys[i]` |
| `Sub(xs, ys)` | `SubTo(dst, xs, ys)` | `xs[i] - ys[i]` |
| `Mul(xs, ys)` | `MulTo(dst, xs, ys)` | `xs[i] * ys[i]` |
| `Scale(xs, k)` | `ScaleTo(dst, xs, k)` | `xs[i] * k` |
| `AddScalar(xs, k)` | `AddScalarTo(dst, xs, k)` | `xs[i] + k` |
| `Abs(xs)` | `AbsTo(dst, xs)` | `abs(xs[i])` |

All `*To` functions permit `dst` to alias the inputs.

## Highlights

* **Architecture-aware execution.** Loop pipelining and instruction-level
  parallelism tuned for ARM64 (2 accumulators) and x86-64 (4 accumulators),
  selected at compile time by build tags. No CGo, no assembly, no intrinsics —
  just Go arranged so the compiler's SSA backend generates the code we want.
* **Measured, not asserted.** **3.98×** faster than a plain loop on `Sum` at
  n=4M, **4.00×** on `Dot`, **1.97×** on `Min`/`Max`, **1.27×** on elementwise
  `Add`. The difference between those figures is the most interesting result in
  the repo, and [docs/benchmarks.md](docs/benchmarks.md) explains what each loop
  is actually waiting on.
* **Numerically stable.** Reassociated tree reductions, so the tuned result is
  typically *closer* to the exact answer than a naive left-to-right fold, not
  further from it. Checked against a 200-bit `big.Float` reference.
* **Zero dependencies.** Pure Go with standard-library testing only.

## Important caveats

These are deliberate design properties. Read them before using the package for
anything where the last bits matter.

1. **Results are reassociated.** `Sum` may differ in the last bits from a plain
   `for` loop. This is usually *more* accurate, but it is different, and it is a
   choice rather than an accident.

2. **No cross-architecture bit-reproducibility.** arm64 and x86-64 run different
   loop shapes and may return different last bits for identical input. If you need
   a reproducible audit trail, this package cannot provide one as designed.

3. **Tuned code is slower on short slices.** `Dot` at n=8 measures **0.80×** —
   25 % *slower* than the plain loop. If your data is short slices, benchmark
   before adopting these functions.

4. **The scans are ~2× faster than a naive loop, not ~4×.** `Sum` and `Dot` gain
   about 4×, but `Min`/`Max`/`MinMax` gain **1.97×**, because a compare is
   inherently latency-bound and no accumulator trick shortens the chain the way
   `s += a + b` shortens an add. That is the nature of the operation, not a weak
   implementation.

5. **`Min`/`Max`/`MinMax` return NaN** if any input element is NaN. This is
   specified and tested, unlike a naive implementation whose result depends on
   where the NaN sits.

The reductions are **not** memory-bandwidth-bound — a same-shape loop that moves
the same bytes but does no real arithmetic reaches 37.8 GB/s against `Sum`'s
25.4 GB/s. The remaining bottleneck is arithmetic latency, so FMA and SIMD have
real headroom. See [docs/benchmarks.md](docs/benchmarks.md).

## Testing

```sh
go test ./...                               # both packages
go test ./vec/                               # correctness, native arch
go test ./vec/ -bench=. -benchmem -count=10  # performance
GOARCH=amd64 go test ./vec/                  # cross-check the x86-64 path
```

The suite covers all boundary lengths (0–17, plus 23–25, 31–33, 63–65, 127–129),
per-element perturbation tests that catch skipped or double-counted elements,
destination aliasing, NaN propagation at every position, and accuracy against an
exact reference.

`GOARCH=amd64 go test ./vec/` works on Apple Silicon via Rosetta 2. It genuinely
verifies x86-64 correctness; the timings it produces are **not** representative of
native x86-64 and should not be quoted as such.

## License

See [LICENSE](LICENSE).
