# Next Steps

Companion to [learn/](learn/), [design.md](design.md) and
[benchmarks.md](benchmarks.md).

The tutorial explains the reasoning. The design doc covers the structure. The
benchmark file records the numbers. This file records what is verified, what is
open, and what should be done next.

---

## 1. A correction

An earlier revision of this documentation claimed a correctness bug in the ARM64
summation: that `i += 8` advanced past two of the four elements the loop body
read, silently skipping half the input.

**That claim was wrong.** It came from reading the source and reasoning about what
it "should" do, without checking what actually executes. The disassembly settles
it: the compiler emits `ADD $4, R3, R3`, because the extra `+4` is dead arithmetic
(nothing reads `xs[i+4]`) and is folded away.

The full story, with the disassembly and the lesson, is in
[learn/03-the-truth.md](learn/03-the-truth.md) section 3. The short version:

> **The source code is a hint. The compiled binary is the truth.** Verify by
> disassembling or by measuring — never by inspection.

`TestSumKnownValues` in `vec/sum_test.go` now pins the exact result at lengths 1
through 100, so this specific concern cannot silently regress.

## 2. What is verified

| check | command | status |
|---|---|---|
| arm64 tests | `go test ./vec/` | **pass** |
| amd64 tests (Rosetta 2) | `GOARCH=amd64 go test ./vec/` | **pass** |
| arm64 vet | `go vet .` | **pass** |
| amd64 vet | `GOARCH=amd64 go vet .` | **pass** |
| generic vet (386, riscv64, js/wasm) | `GOOS/GOARCH=… go vet .` | **pass** |
| formatting | `gofmt -l .` | **clean** |
| benchmarks | `go test ./vec/ -bench=. -benchmem` | **run** — see [benchmarks.md](benchmarks.md) |

Coverage, applied to every operation:

* all boundary lengths 0–17, plus 23–25, 31–33, 63–65, 127–129
* agreement with a reference implementation
* **per-element perturbation** (catches skipped or double-counted elements)
* hand-computed known values (catches errors hidden inside a relative tolerance)
* empty and nil inputs
* mismatched lengths
* `dst` aliasing `xs`, `ys`, and both simultaneously
* panic on length mismatch for every `*To` variant
* identity cross-checks: `SumSq(xs) == Dot(xs, xs)`, `Mean(xs) == Sum(xs)/n`
* accuracy against a 200-bit `big.Float` reference
* **NaN propagation at every position**, including index 0 and index n-1

## 3. Resolved: the NaN check cost the entire scan speedup

This was the highest-priority item in the repo. It is now fixed.

**The problem.** `Min`, `Max` and `MinMax` enforced the "NaN wins" policy by
calling `hasNaN`, which walked the input a second time. Measured at n=4M on arm64:

| | GB/s | vs naive |
|---|---|---|
| naive loop | 4.81 | 1.00× |
| tuned scan, no NaN check | 9.58 | 1.99× |
| tuned scan + separate NaN pass | 4.68 | 0.97× |

The tuning worked; the second pass ate all of it.

**The fix.** Fuse the NaN check into the scan loop. Now 9.48 GB/s, **1.97× vs
naive**, within 1% of a scan with no check at all.

**Why it works, and the false lead.** The NaN test accumulates a boolean flag,
evaluated with integer operations on different execution ports than the
floating-point compare. Since the scan is latency-bound on the compare chain, the
flag costs nothing. A second pass, by contrast, doubles the memory traffic.

The obvious branch-free alternative was tried first and does not work, which is
worth recording: `poison += v - v` is 0 for finite values and NaN for NaN, so it
looks like a comparison-free NaN detector. But **`Inf - Inf` is NaN too**, as is
`Inf × 0`, so any array containing an infinity is falsely poisoned. There is no
branch-free float expression that isolates NaN from infinity. Probing each
candidate against finite, ±0, NaN and ±Inf confirmed it. Since a compare is
unavoidable, the cheapest place for it is inside the loop that is already reading
the data.

**Verified by:** `TestScansPropagateNaN` (every position),
`TestScansPropagateNaNAtBoundaries` (index 0 and n-1, which the loop seeds from),
`TestScansSmallInputsNaNBehaviour` (±Inf not poisoned),
`TestScansInfinitiesAreNotNaN`, and `BenchmarkMinPureScan` as the upper bound.

**Follow-up if this regresses:** if `BenchmarkMin` ever drifts far from
`BenchmarkMinPureScan` again, the fusion has been undone. That gap is the
regression detector, which is why the pure-scan benchmark exists.

## 4. Answered: the reductions are not bandwidth-bound

This was listed as an open question (§6.6 in an earlier revision) on the grounds
that if the loops were bandwidth-bound, SIMD would win nothing. It was measured
rather than assumed, and the assumption was wrong.

`BenchmarkReadOnlyTuned` walks the same memory with the same loop shape but
multiplies by zero instead of accumulating a real add:

| arm64, n=4M | GB/s |
|---|---|
| `Sum` (4 chains, 2 adds each) | 25.4 |
| `ReadOnlyTuned` (4 chains, mul by zero) | **37.8** |
| `Memcpy` (read + write) | 57.6 |

`ReadOnlyTuned` is **49% faster while doing strictly less arithmetic on the same
bytes**. So `Sum` is limited by floating-point add latency still in the dependency
chain, not by memory bandwidth. Roughly **1.49× appears available** on arm64 by
reducing instruction count per element rather than bytes touched.

The earlier claim was inferred from the shape of a plateau instead of measured. It
is corrected in [benchmarks.md](benchmarks.md), where the mistake is recorded
rather than quietly edited out.

## 5. What is not verified

* **Native x86-64 throughput.** The amd64 figures were taken under Rosetta 2,
  which translates x86-64 to ARM64 rather than executing it natively. Correctness
  is genuinely verified there; absolute performance is not representative. Any
  decision resting on x86-64 magnitudes needs real hardware.
* **`GOAMD64`/`GOARM` variants.** `GOAMD64=v3` (AVX2) and above are untested. The
  tuning here is scalar-accumulator ILP rather than SIMD, so the variant should
  matter less than usual — a prediction, not a measurement.
* **Compensated summation.** No Kahan/Neumaier variant is offered.
* **Concurrency.** Nothing here is safe to call concurrently against a shared
  destination slice, and that is not documented outside the aliasing note.

## 6. Numerical properties to be aware of

Deliberate properties, not defects — but the kind that surprise people in
production:

1. **The reductions reassociate.** `Sum` may differ in the last bits from a plain
   `for` loop. `TestSumIsNotLessAccurateThanNaive` checks the tuned result is no
   *further* from the exact answer than the naïve one, which is a different
   property from agreeing with the naïve loop.

2. **No cross-architecture reproducibility.** arm64 and amd64 run different loop
   shapes and may return different last bits for identical input. If you need a
   reproducible audit trail, this package cannot provide one as currently designed.

3. **`(xs+ys)-ys != xs`, exactly.** The round trip is accurate only to within a
   relative epsilon. `TestAddSubRoundTrip` asserts the tolerance rather than
   equality, and documents why.

4. **`Min`/`Max` return `±Inf` on empty input**, making them usable as running
   identities — which also means an empty slice is not an error. Callers decide.

5. **`Min`/`Max`/`MinMax` return NaN if any input element is NaN.** Specified,
   implemented, tested, and now cheap. See §3.

## 7. Further open decisions

1. **Compensated summation.** Offer `SumKahan`/`SumNeumaier` for callers who need
   accuracy more than throughput. The test infrastructure already exists —
   `exactSum` in `vec/helpers_test.go` is a 200-bit reference.

2. **Fused multiply-add.** `Dot` and `SumSq` currently compile to a separate
   multiply and add. With hardware FMA this could be one instruction with one
   rounding, which is both faster and *more accurate*. Since §4 established that
   the reductions are arithmetic-bound rather than bandwidth-bound, FMA is now a
   well-motivated optimization rather than a speculative one. It changes results
   again, so it interacts with §6.2 and §7.3.

3. **Reproducible summation via assembly.** If cross-architecture
   bit-reproducibility is required, the only way to keep the tuned loops is for each
   architecture to emit *identical* arithmetic operations in an identical order,
   differing only in scheduling. That is a genuine design constraint and should be
   decided early if it matters.
5. **SIMD.** The current approach is scalar-accumulator ILP relying on the
   compiler's autovectorizer. Explicit NEON/AVX2 intrinsics would need CGo or
   assembly, breaking the zero-dependency pure-Go constraint in the README. A
   middle path is `GOAMD64=v3`-gated code, which preserves that constraint.

   §4 changed this from "probably pointless" to "worth trying": there is ~1.49×
   of arithmetic headroom, so reducing instructions per element is the right
   target. It is a larger change than FMA (§7.2) and has a lower expected payoff
   than the measured 1.49× cap, so try FMA first.

6. **Is there headroom left?** Yes — see §4. Measured, not assumed. Roughly 1.49×
   on arm64 for `Sum`, bounded by a same-shape loop that does no real arithmetic.

## 8. Suggested order of work

```mermaid
graph LR
    A["1. Fix the NaN cost (§3)"] --> B["2. Is it bandwidth-bound? (§4)"]
    B --> C["3. FMA for Dot/SumSq (§7.2)<br/>~1.49x headroom, or not"]
    C --> D["4. Decide reproducibility (§6.2)"]
    D --> E["5. SIMD (§7.5)<br/>only if FMA leaves room"]
    E --> F["6. Re-benchmark on<br/>native x86-64"]
    style A fill:#d4f4d4
    style B fill:#d4f4d4
    style C fill:#ffe9b3
    style F fill:#ffcccc
```

**Done:** §3 and §4. Both were resolved by measuring rather than reasoning, and both
overturned an assumption that had been stated as fact in this repository — the NaN
check turned out to be nearly free when fused, and the reductions turned out not to
be bandwidth-bound at all.

**Next:** FMA (§7.2). It now has a measured motivation rather than a speculative
one: the bottleneck is arithmetic latency, and FMA removes an operation per
element while also improving accuracy by rounding once instead of twice.

**Still blocked on hardware:** §8.6. The x86-64 figures in this repository come
from Rosetta 2, so any decision that depends on x86-64 magnitudes needs a native
machine to confirm.

**Still a decision, not work:** §6.2 (reproducibility). It gates what the
arithmetic is even allowed to do, so it is worth settling before adding more
operations rather than after.
