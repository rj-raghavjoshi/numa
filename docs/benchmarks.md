# Benchmarks

Measured results. Companion to [learn/](learn/), which explains *why* the loops
are shaped the way they are; this file is the evidence that the shape actually
pays off.

**Every number below was produced on this machine and can be reproduced with the
commands shown.** Nothing here is estimated or carried over from another
architecture.

---

## How to reproduce

```sh
cd /Users/raghav/Development/numa

# Full suite, all operations.
go test ./vec/ -bench=. -benchmem -count=10

# Just the headline comparison.
go test ./vec/ -bench='NaiveSum|^BenchmarkSum$' -benchmem -count=10

# Sub-second spot check.
go test ./vec/ -bench=. -benchtime=100x -run='^$'
```

For comparing two runs, use benchstat rather than eyeballing raw output:

```sh
go install golang.org/x/perf/cmd/benchstat@latest
go test ./vec/ -bench=. -benchmem -count=10 > new.txt
benchstat old.txt new.txt
```

`-count=10` is not optional. Single-run laptop numbers move by several percent
between invocations, which is larger than the differences being measured.

## Test environment

| | |
|---|---|
| Hardware | Apple Silicon (arm64), macOS |
| Toolchain | `go1.26.5 darwin/arm64` |
| x86-64 numbers | `GOARCH=amd64` under Rosetta 2 |

### Caveat on the amd64 numbers

The x86-64 results are measured through Rosetta 2, which translates x86-64 to
ARM64 rather than executing it natively. **They validate correctness and
direction, not absolute throughput.** Any tuning decision that depends on the
exact x86-64 magnitude needs re-measuring on real x86-64 hardware.

## Test data

All benchmarks use `benchData(n)`: `xs[i] = float64(i%17)*0.5 - 4.0`. A
non-constant pattern is deliberate — some CPUs have data-dependent timing, and
all-equal inputs can be unrepresentatively fast.

Sizes: 8 (register-resident), 1024 (L1), 32768 (L2), 4194304 (main memory).
Every benchmark calls `b.SetBytes`, so the reported throughput in GB/s tells you
whether a loop is compute-bound or bandwidth-bound.

---

## Headline result: `Sum`

The baseline `BenchmarkNaiveSum` is the plain one-accumulator loop, compiled on
the same machine and measured in the same run.

### arm64 (native)

| n | naïve ns/op | tuned ns/op | naïve GB/s | tuned GB/s | speedup |
|---|---|---|---|---|---|
| 8 | 5.02 | 4.17 | 12.7 | 15.4 | **1.20×** |
| 1 024 | 1 184 | 278.7 | 6.9 | 29.4 | **4.25×** |
| 32 768 | 40 725 | 10 195 | 6.4 | 25.7 | **3.99×** |
| 4 194 304 | 5 260 503 | 1 323 642 | 6.4 | 25.4 | **3.98×** |

### amd64 (Rosetta 2)

| n | tuned ns/op | tuned GB/s |
|---|---|---|
| 8 | 4.52 | 14.2 |
| 1 024 | 210.4 | 38.9 |
| 32 768 | 6 563 | 40.0 |
| 4 194 304 | 1 002 336 | 33.5 |

**Reading this:** the tuned loop is **~4× faster** at every size that matters.

> **Correction.** An earlier version of this file claimed the plateau at ~25 GB/s
> was "the memory-bandwidth ceiling". **That was wrong**, and the mistake survived
> because it was inferred from the shape of the curve rather than measured.
>
> `BenchmarkReadOnlyTuned` walks the same memory with the same loop shape but
> multiplies by zero instead of accumulating a real add. It reaches **37.8 GB/s** —
> 49 % faster than `Sum`. So `Sum` is not limited by memory bandwidth at all; it is
> limited by floating-point add latency still in the dependency chain.
>
> See [Bandwidth ceiling](#bandwidth-ceiling) below for the measurements, and
> [next-steps.md](next-steps.md) §6 for what this means for SIMD and FMA.

Note the small-size behaviour. At n=8 the tuned version is only 1.2× faster,
and part of that is the removed empty-check, not the unrolled loop. This matches
the prediction in [learn/02-the-buildup.md](learn/02-the-buildup.md): **the tuning
only pays off when the input is long enough to amortize the setup.**

## `Dot`

| n | naïve GB/s | tuned GB/s | speedup |
|---|---|---|---|
| 8 | 25.7 | 20.6 | 0.80× (**slower**) |
| 1 024 | 9.9 | 42.5 | **4.30×** |
| 32 768 | 9.6 | 39.5 | **4.10×** |
| 4 194 304 | 9.6 | 36.7 | **3.81×** |

`Dot` shows the largest speedup of any reduction here, as expected: a
floating-point multiply has higher latency than an add, so the naïve loop stalls
harder and there is more for the independent accumulators to recover.

It also shows the clearest small-size regression — **0.80×, i.e. 25 % slower at
n=8**. This is the honest counterexample to "tuned is always better". If your
data is short slices, measure before adopting these functions.

## `Min` / `Max`

arm64, n=4 194 304:

| variant | ns/op | GB/s | vs naive |
|---|---|---|---|
| `naiveMinWithNaN` (1 accumulator, fused check) | 6 967 355 | 4.81 | 1.00× |
| `Min` **before the fix** (tuned scan + separate NaN pass) | 7 053 507 | 4.68 | 0.97× |
| `Min` **after the fix** (tuned scan, fused NaN check) | 3 538 170 | **9.48** | **1.97×** |
| `MinPureScan` (tuned scan, no NaN check at all) | 3 502 250 | 9.58 | 1.99× |

> **This section previously reported `Min` as a failure**, measuring 0.98× — no
> faster than the naive loop. The cause was a second pass over the input for NaN
> detection, which cost about 2× and cancelled the entire benefit of the tuned
> scan. That is now fixed by fusing the check into the scan loop, and `Min` is
> **1.97× faster than naive**.
>
> Fusing costs about **1%**: 9.48 GB/s with the check versus 9.58 GB/s without any
> check at all. The reason it is nearly free is in the next section.

Min/max remains roughly **2.7× slower than summation** at the same size (9.5 GB/s
vs 25.4 GB/s), and that part *is* the nature of the problem:

> A compare cannot start until the previous compare's result is known. `min` has
> no equivalent of the `s += a + b` trick, because there is no way to "combine two
> elements at once" that shortens the chain. Splitting the input into independent
> partial scans is the *only* available parallelism, and it divides the chain
> length by the accumulator count rather than eliminating it.

So scans are **latency-bound** and reductions are **throughput-bound**, and they
respond to tuning differently.

### Why fusing the NaN check is nearly free

The NaN test accumulates a boolean flag, which the CPU evaluates with integer
operations on different execution ports than the floating-point compare. The scan
is latency-bound on the *compare* chain, and the flag does not lengthen that chain.
A second pass, by contrast, doubles the memory traffic and re-walks the
latency-bound loop — which is exactly the 2× that was measured.

**The arithmetic alternative does not work, and it is worth recording why** so
nobody retries it. The natural branch-free NaN test is a poison accumulator:

```
poison += v - v     // 0 for finite v, NaN for NaN
```

For finite values this adds 0; for NaN it adds NaN, which contaminates every later
addition, so `poison != poison` detects it without a compare. The problem: **`Inf −
Inf` is NaN too**, as is `Inf × 0`. Any array containing an infinity gets falsely
poisoned, and no branch-free float expression isolates NaN from infinity. Probing
each candidate against finite, ±0, NaN, +Inf and −Inf confirmed it. Since a compare
is unavoidable, folding it into the existing loop is the cheapest place to put it.

### The "NaN wins" policy

`Min`, `Max` and `MinMax` return NaN if any input element is NaN. The reasoning is
in `vec/nan.go`: an array containing NaN is almost always the result of an upstream
error, and silently returning the extreme of the *remaining* values would hide
that error while producing a plausible-looking number. In a domain where these
numbers feed decisions, failing loudly is safer.

The alternative ("ignore NaN") is defensible but is the *accidental* behaviour of
a naive scan, since `v < m` is false for NaN. That means a naive implementation's
result depends on where the NaN sits: one at index 0 gets overwritten by later
comparisons, one at the end gets returned. Neither policy justifies positional
behaviour, which is why the check exists.

## `MinMax` vs `Min` then `Max`

| arm64, n=4 194 304 | ns/op | GB/s |
|---|---|---|
| `MinMax` (one pass) | 5 260 681 | 6.4 |
| `Min` + `Max` (two passes) | 7 464 223 | 9.0 |

`MinMax` is **1.42× faster** than calling `Min` and `Max` separately, because it
reads the input once instead of twice. Note that the GB/s figure is *lower* for
`MinMax` even though it is faster — it moves less data, and the wall-clock time
is what matters. Comparing GB/s across those two would be misleading; the
`BenchmarkMinThenMax` variant reports `16*n` bytes precisely so the two are
comparable on the same basis.

(These two figures predate the NaN check, so both are now optimistic. The
comparison between them remains valid; the absolute values do not. Re-run
`go test ./vec/ -bench='MinMax|MinThenMax'` for current numbers.)

## Elementwise `Add`

| arm64 | naïve ns/op | tuned ns/op | naive GB/s | tuned GB/s |
|---|---|---|---|---|
| 8 | 5.44 | 7.48 | 35.3 | 25.7 |
| 1 024 | 676.7 | 546.5 | 36.3 | 45.0 |
| 32 768 | 20 557 | 16 189 | 38.3 | 48.6 |
| 4 194 304 | 2 722 034 | 2 142 221 | 37.0 | 47.0 |

Elementwise addition is a much smaller win than the reductions — **~1.27×**
rather than ~4×. This is also expected, and it is the clearest illustration of
the difference between the two kinds of loop:

> `dst[i] = xs[i] + ys[i]` has **no dependency chain at all**. Each output
> depends only on its own inputs. There is therefore no stall to eliminate, and
> the compiler was already free to overlap everything. The only thing tuning can
> do here is keep the load/store units saturated — and it is already close to
> memory bandwidth.

So: **tuning a reduction recovers stalled cycles; tuning a map recovers
nothing, because nothing was stalled.** Same syntax, completely different
problem.

---

## Bandwidth ceiling

The most useful measurement in this file, because it overturns a claim that was
previously made here.

Two loops that walk exactly the same memory with exactly the same loop shape
(four accumulator chains, eight elements per iteration):

```go
// readOnlyTuned: touches every element, multiplies by zero
s0 += xs[i] * 0
s1 += xs[i+2] * 0
...

// sumArch: touches every element, accumulates a real add
s0 += xs[i] + xs[i+1]
s1 += xs[i+2] + xs[i+3]
...
```

Same loads. Same bytes. Same unrolling. The only difference is the arithmetic.

arm64, n=4 194 304:

| loop | GB/s | per element |
|---|---|---|
| `ReadOnly` (naive, 1 accumulator, `v*0`) | 4.80 | 1.67 ns |
| `Sum` (tuned, 4×2 adds) | **25.4** | 0.32 ns |
| `ReadOnlyTuned` (tuned shape, 4×1 mul by zero) | **37.8** | 0.21 ns |
| `Dot` (tuned, 4×2 mul + add) | **38.4** | 0.21 ns |
| `Memcpy` (read + write) | 57.6 | — |

**`ReadOnlyTuned` is 49 % faster than `Sum` while doing strictly less arithmetic on
the same memory.** Therefore:

> **`Sum` is not bandwidth-bound.** It is bound by floating-point add latency that
> remains in the dependency chain. The plateau at ~25 GB/s is where *this loop
> shape* tops out, not where the memory system does.

This matters beyond bookkeeping, because the previous version of this file used the
"bandwidth-bound" claim to argue that SIMD would win nothing. That argument was
backwards. The remaining bottleneck is arithmetic, which is exactly what SIMD and
FMA address. Roughly **1.49× appears to be available on arm64** (37.8 / 25.4),
and `Dot` at 38.4 GB/s suggests it is already closer to the ceiling than `Sum` is.

---

---

## Elementwise math kernels

The `vec/math.go` kernels are maps with one output per input. They carry no build
tag, because a map has no accumulator chain for a register file to hide and no
architecture-specific loop shape to select — the wide unrolled loop *is* the tuned
loop on every architecture. That is an exception to the usual dispatch pattern, and
it is explained in [design.md](design.md).

The naive baselines are the same loop without the four-wide unroll. arm64,
n = 4 194 304:

| operation | naïve GB/s | tuned GB/s | speedup | limited by |
|---|---|---|---|---|
| `Neg` | 37.2 | **49.6** | 1.33× | memory bandwidth |
| `Div` | 37.9 | **52.6** | 1.38× | memory bandwidth |
| `Min2` | 28.3 | **47.8** | 1.69× | memory bandwidth |
| `MulAdd` | 37.8 | **50.0** | 1.32× | memory bandwidth |
| `Sqrt` | 19.3 | 19.2 | **1.00×** | the math library call |

Two things are worth reading out of that table:

1. **The cheap maps gain 1.3–1.7×; the transcendental one gains nothing.** That is
   the intended result, not a failure. `Sqrt` is dominated by the library call,
   which costs about an order of magnitude more than the loop around it, so the
   unroll has nothing left to recover. Leaving the transcendentals on a plain range
   loop is therefore justified by measurement rather than by taste — and the unroll
   is equally justified for the cheap maps.

2. **`Min2` gains the most of any map (1.69×), more than `Neg` or `Div`.** A
   lane-wise minimum carries a compare, and the four-wide unroll lets four compares
   be in flight at once instead of one. It is the same latency-versus-throughput
   story as the scans, reappearing here in a map.

### FMA evidence

`BenchmarkFMAEvidence` compares a loop calling `math.FMA` against a plain `x*y + z`.
On this machine the two are within 5 % of each other at n=4096 — the loop is
memory-bound at that size, so instruction count barely shows. The decisive evidence
is the disassembly: `go tool objdump -s 'vec\.dotArch$'` on the package's own test
binary shows `FMADD D`, because Go contracts `a + x*y` into a hardware fused
multiply-add by default. An earlier claim in this repository that Go does not
contract FMA, and that `math.FMA` is a slow software routine, was wrong; it is
corrected in [next-steps.md](next-steps.md) §7.2 along with the commands to
reproduce it.

### Predicates

A predicate reads 16 bytes and writes one, so it might be expected to be
read-bandwidth-bound with the store nearly free. Measured, the four-wide unroll
still matters a great deal, because the loop is **compare-bound, not store-bound** —
exactly like `Min2`:

| operation | naïve GB/s | tuned GB/s | speedup |
|---|---|---|---|
| `GreaterTo` | 18.5 | **35.7** | **1.93×** |

That is the largest map speedup recorded in this file. It makes the point that
"elementwise maps gain only ~1.3×" is a statement about *pure arithmetic* maps: a
map that carries a comparison gains like a scan, because the comparison is the
latency the unroll hides.

### MaskedSum is branch-bound, and that is recorded rather than hidden

`MaskedSum` reaches only **11.2 GB/s** at n=4M, against `Sum`'s 25.4 GB/s (and
`MaskedSum` reads *more* bytes, which makes the gap worse than the raw ratio
suggests). The cause is a data-dependent branch per element.

This is a known weakness, not a defensible result. A branchless formulation exists
(`total += xs[i] * float64(mask[i])`), but it adds a multiply and a conversion per
element and would change how a selected NaN propagates, so it is left as a
follow-up rather than changed speculatively. The branch version's contract is the
one this package wants; its cost is documented here so the next person does not
have to rediscover it.

### Cumulative operations

A prefix scan has a genuine loop-carried dependency: output i needs the running
value. It cannot use independent accumulators the way a reduction can, because
there is only ever one chain by definition. What shortens it is **blocking** —
computing a short local prefix inside a block, then adding the carried total to each
of its elements. The carry then advances once per block instead of once per element.

| operation | naïve GB/s | tuned GB/s | speedup |
|---|---|---|---|
| `CumSumTo` | 12.7 | **41.0** | **3.22×** |
| `CumMaxTo` | 9.66 | 9.65 | **1.00×** |

`CumSum` of a `float64` slice is not a reduction and should not be compared with
`Sum`'s 3.98×; the relevant comparison is against its own baseline. A prefix sum is
*more* latency-bound than a plain sum, because the running value must be stored as
well as carried, so there is more for blocking to recover.

`CumMax` is deliberately **not** blocked, and the measurement is why that is not
laziness: it lands at exactly break-even. A running maximum's per-element compare
cannot be removed by blocking — the local prefix maximum inside a block still needs
a compare per element — so only the carry shortens and nothing is saved. That is the
same wall `Min`/`Max` hit, and it is recorded as a negative result rather than
hidden.

### Lagged maps, and a benchmark that measured the wrong thing

`Shift` and `Diff` read two streams a fixed distance apart. The interesting result
here is not the speedup but the mistake that came first.

| operation | naïve GB/s | tuned GB/s | speedup |
|---|---|---|---|
| `ShiftTo` via built-in `copy` | 25.2 | **57.4** | **2.28×** |
| `ShiftTo` via four-wide unroll | 25.2 | 33.7 | 1.34× |
| `DiffTo` via four-wide unroll | 19.1 | **28.5** | **1.49×** |

The first benchmark run reported the naive `Shift` loop at **37 GB/s** — *faster* than
the unrolled version. That looks like a clean negative result and would have been
recorded as one. It was wrong. The naive baseline had `lag` as a compile-time constant
of 1, so the compiler recognised `dst[i] = xs[i-1]` as a shift and substituted a
memmove: the baseline was timing the runtime's SIMD copy routine, not the loop.
Re-running with the lag passed as a runtime value — which is what the real API does —
dropped the naive figure to 25.2 GB/s.

The correct conclusion was the opposite of the one the first run suggested. `Shift` is
pure data movement, so it should call the built-in `copy` deliberately. That reaches
**57.4 GB/s**, better than any hand-written loop, and it is the one place in the
package where "the tuned implementation" is a library call rather than a loop.

`DiffTo` has no such shortcut — it subtracts — so it keeps the unroll, at 1.49×.

**The lesson is recorded, not just the number:** a benchmark can accidentally measure
a compiler special case instead of the code under test, and a constant parameter is
how it happens. These benchmarks now take the lag as a variable and sweep two values.

### Order statistics: quickselect against sorting

A quantile does not need the input ordered — only the element at one or two
positions. The baseline is therefore the honest alternative (copy, `sort.Float64s`,
interpolate), not a strawman.

Median (q = 0.5), arm64:

| n | sort-based | quickselect | speedup |
|---|---|---|---|
| 1 024 | 10.2 µs | **3.3 µs** | **3.10×** |
| 32 768 | 480 µs | **93.6 µs** | **5.13×** |
| 1 048 576 | 9.78 ms | **6.48 ms** | **1.51×** |

The speedup *shrinks* as n grows, which is expected rather than disappointing:
quickselect is O(n) and the sort is O(n log n), but at large n both are memory-bound
and the constant factor of partitioning starts to dominate. At 1M elements the input
no longer fits in cache, so the asymptotic advantage is masked by the memory system —
the same effect §4 documents for the reductions.

The two guards in `selectK` (insertion sort below 12 elements, and a depth-limited
fallback to `sort.Float64s`) are what make this safe rather than merely fast. Plain
quickselect is quadratic on adversarial input; the depth cap restores an O(n log n)
worst case for one comparison per level.

### Arg reductions: a 2× regression, found and fixed

`ArgMin` is a scan, and the first implementation fused the NaN check into a
**single-chain** loop. It measured **0.51×** — half the speed of a plain loop:

| | GB/s | vs naïve |
|---|---|---|
| naïve loop (no NaN check) | 9.52 | 1.00× |
| first attempt: single chain + fused NaN check | **4.85** | **0.51×** |
| tuned: two chains + fused NaN check | **9.55** | **1.00×** |

The cause is that an arg reduction is compare-bound, and the NaN test is a second
compare competing for the same port. A single chain has nothing to hide it behind, so
the second compare roughly halves throughput. Splitting the input into independent
partial scans (two on arm64, four on x86-64), each with its own NaN flag, recovers the
loss exactly.

The tuned version lands at *parity* with the naïve loop rather than ahead of it, and
that is the correct result: 9.5 GB/s is the compare-latency ceiling that `Min` also
hits, and the naïve loop is already at it. The gain is not beating the naïve loop; it
is not being half as fast as it.

**A subtlety unique to arg reductions:** with independent partial scans, the same
extreme value can be found at different indices in different chains, so the combine
step must break an equality by index. A combine that compared only values would
return the wrong index for a tie across chains —
`TestArgReductionsTiesResolveFirst` and the exhaustive random cross-check cover it.
An amd64 combine bug of exactly this shape (comparing each chain against chain 0
rather than against the running best) was caught during development by those tests.

### Statistics: the accurate form is also the faster one

`Variance` here is two-pass: mean first, then the sum of squared deviations. The
obvious one-pass form `mean(x²) − mean(x)²` is numerically catastrophic, and its
usual defence is speed. On this machine that defence is false.

arm64, n = 4 194 304:

| implementation | ns/op | effective GB/s | accurate? |
|---|---|---|---|
| tuned two-pass `Variance` | **3.67 ms** | 18.3 | **yes** |
| naïve two-pass (one accumulator) | 12.3 ms | 5.46 | yes |
| naïve one-pass `mean(x²)−mean(x)²` | 7.00 ms | 4.79 | **no** |

Two results stand out:

1. **The tuned two-pass beats the naïve two-pass by 3.36×.** That is the arch tuning
   of `dotDevArch` doing the same thing the tuned reductions do elsewhere: splitting
   one dependency chain into independent ones.

2. **The tuned two-pass is also 1.91× faster than the naïve one-pass**, while reading
   the input twice where the one-pass reads it once. So the accuracy is not merely
   affordable here — it is free, and then some. Independent accumulators recover more
   than the second traversal costs.

The numerical failure is not marginal. `TestVarianceBeatsNaiveOnePass` uses 1000
values of 1e8 or 1e8+1, whose true variance is exactly 0.25. The one-pass form
returns **97.75** — wrong by a factor of 391 — while the two-pass returns 0.25. At
that magnitude `mean(x²)` and `mean(x)²` are both about 1e16, where one ulp is 2, so
the one-pass result is quantised to multiples of 2 and cannot represent 0.25 at all.

`Correlation` runs at 13.8 GB/s across two inputs.

This is the clearest instance in the package of a general lesson: **the "fast"
formulation was only faster in a setting that did not have tuned reductions.** The
one-pass identity trades accuracy for a speed that a vectorised deviation sum simply
gives back.

## series: streaming rollers

The `series` package maintains rolling state so each new value costs O(1) regardless
of the window, where recomputing over the window costs O(window). arm64, n = 65 536,
**ns per element**:

| operation | window 20 | window 200 |
|---|---|---|
| `SMA` (streaming) | 21.0 | 21.0 |
| naïve SMA (recompute the window) | **10.4** | 195 |
| `Sum` (streaming, Neumaier-compensated) | 19.2 | 19.4 |
| `SumNoCompensation` (identical structure) | **9.3** | 9.3 |
| `EMA` (streaming) | 4.6 | 4.6 |

Three things are worth reading out of that table.

1. **The streaming cost is flat in the window**, which is the whole claim: `SMA` is
   21.0 ns/element at window 20 and 21.0 at window 200. `EMA` is flat and much cheaper
   because it carries no window at all.

2. **Streaming is not universally faster, and at window 20 it loses by 2×.** Naïve
   recomputation of a 20-element window is a tight contiguous loop the compiler can
   vectorise, while the streaming path does ring bookkeeping, a fullness check, a NaN
   count and a compensated accumulator per element. The crossover lies between 20 and
   200; by 200 the recomputation is 9.3× slower. **A caller rolling a short window over
   in-memory data should measure rather than assume the roller wins.**

3. **Neumaier compensation doubles the rolling-sum cost**, isolated against an
   otherwise identical roller: 19.2 against 9.3 ns/element, **2.07×**. Two earlier
   figures in this file were wrong and are corrected rather than deleted: a claim that
   compensation was free (because the loop was "latency-bound on the ring arithmetic")
   and a 4.3× figure that came from a baseline with a different buffer structure. The
   honest number is 2.07×, and it is the price of not returning a silently wrong sum
   after a large-offset eviction — the `1e16, 1, 1` case in
   `TestSumDoesNotLoseSmallValuesAcrossEviction`.

Every roller reports **0 allocs/op** after construction; `Reset` reuses the ring
without reallocating, so one roller can process many series in a parameter sweep.

### Follow-up: periodic re-summation instead of per-element compensation

Neumaier pays on every element to bound drift. Periodic re-summation would bound it
differently: recompute the total from the ring once every `window` pushes, so the
error is at most one window's worth of summation instead of an unbounded random walk.
That is one amortized add per element rather than three comparision-dependent
operations, so it should be close to the uncompensated 9.3 ns/element while keeping
drift bounded. It is recorded here as a measured follow-up rather than implemented
speculatively, because the bound it gives is different in kind from compensation and
the trade should be measured before it is adopted.

## ta: indicators

arm64, **ns per element**. `SMA`, `EMA` and `ATR` delegate to the series rollers;
`Highest` is a monotonic deque; `WMA` and `BollingerBands` are window loops by design.

| indicator | n=1K | n=1M | method |
|---|---|---|---|
| `EMA(20)` | 5.3 | 4.8 | one multiply-add per element |
| `ATR(14)` | 7.3 | 6.5 | true range + Wilder smoothing |
| `Highest(20)` | 7.0 | 5.8 | monotonic deque |
| `Highest(200)` | 6.5 | 5.8 | monotonic deque — **flat in window** |
| `WMA(20)` | 13.2 | 12.8 | O(window) loop |
| `WMA(200)` | 206 | 254 | O(window) loop — 19.8× the window-20 cost |
| `SMA(20)` | 19.6 | 19.4 | the compensated series roller |
| `BollingerBands(20)` | 36.0 | 34.8 | O(window) centred deviations |

### The monotonic deque does what it promises

`Highest` is flat in the window, which is the whole claim: **5.8 ns/element at window
20 and 5.8 at window 200**. Rescanning the window instead grows linearly:

| window | deque ns/elem | rescan ns/elem | speedup |
|---|---|---|---|
| 20 | 5.8 | 16.9 | 2.9× |
| 200 | 5.8 | 305 | **52.7×** |

This is the one place in the package where the algorithmic difference — O(1) amortised
against O(window) — dwarfs every constant-factor tuning elsewhere. A 100-bar Donchian
channel or a 52-week high is not a micro-optimization target; it is a different
complexity class.

### The measured cost of WMA's NaN policy

`WMA` is deliberately O(window) rather than O(1), because the O(1) recurrence makes a
NaN permanently sticky (see the function's doc comment). The price is visible: **12.8
ns/element at window 20 against 254 at window 200, a 19.8× penalty**. That is a real
cost, recorded rather than excused. The O(1) form with NaN recovery is a follow-up.

### Bollinger's centred deviations

`BollingerBands` costs 34.8 ns/element at window 20, roughly 2.7× the `SMA` it is built
on, because the deviations are taken from the window mean directly rather than from a
running sum of squares. That is the accuracy choice documented on the function; the
O(1) alternative is the `mean(x²) − mean(x)²` identity that vec/stats.go rejects.

### Momentum indicators

arm64, **ns per element** at n = 1M:

| indicator | ns/element | method |
|---|---|---|
| `ROC(10)` | 1.8 | one lag and a divide (`vec.Rate`) |
| `RSI(14)` | 4.4 | inlined Wilder recurrence |
| `MACD(12,26,9)` | 15.8 | three EMA passes |
| `CMO(14)` | 26.2 | two rolling sums |
| `CMO(100)` | 26.8 | two rolling sums — **flat in period** |
| `CCI(20)` | 34.9 | O(window) centred mean deviation |
| `Stochastic(14,3,3)` | 52.7 | two monotonic deques + two SMAs |

`RSI` is the cheapest oscillator despite being the most intricate, because the
smoothing is inlined as a single recurrence rather than assembled from a general roller
— and because the first change has no previous close, so a general roller would have to
be started past it anyway.

**`CMO` was rewritten after being measured, and the measurement is the interesting
part.** The first version resummed the gains and losses over the window at every bar:
O(window), 26.2 ns/element at period 14. Replacing it with two `series.Sum` rollers makes
it O(1); the numbers are:

- at **period 14** the two are a wash — 26.2 before, 26.2 after — because a compensated
  rolling sum costs about what 14 direct adds cost;
- at **period 100** the rewrite holds the cost at **26.8 ns/element** where the rescan
  would have grown to roughly 100.

The change is invisible at the default period and decisive away from it, which is exactly
what a single-period benchmark would have hidden. Both periods are now measured.

`CCI` keeps its O(window) loop: the centred mean deviation is the accurate form and there
is no O(1) identity for it that this package would accept — the same reasoning as
BollingerBands. `Stochastic` is the most expensive of the group because it builds two
independent monotonic deques, one over the highs and one over the lows.

### Volume indicators

arm64, **ns per element** at n = 1M:

| indicator | ns/element | method |
|---|---|---|
| `AccumulationDistribution` | 1.7 | one pass, running total |
| `OBV` | 1.8 | one pass, running total |
| `PVT` | 1.8 | one pass, running total |
| `VWAP` | 1.8 | one pass, running total |
| `ForceIndex` | 5.7 | one EMA pass |
| `MFI(14)` | 21.5 | two rolling sums |
| `VWMA(20)` | 21.4 | two rolling sums |
| `VWMA(200)` | 21.3 | two rolling sums — **flat in period** |
| `CMF(20)` | 22–26 | two rolling sums |

The four cumulative indicators are the fastest in the package, around 1.8 ns/element,
because each is a single pass with one running total and no window at all. Anchoring is
the caller's job, which also means there is no anchor bookkeeping to pay for here.

`VWMA` is measured at two periods and is **flat**: 21.4 ns/element at period 20 and 21.3
at period 200. That is the rolling-sum construction paying off exactly as it did for
`CMO` in the momentum section. The absolute cost is dominated by the two
Neumaier-compensated `series.Sum` rollers, which is the 2.07× price documented in the
series section — the accuracy of the volume-weighted sums is bought, not free, and it is
the same trade.

### Trend indicators

arm64, **ns per element** at n = 1M:

| indicator | ns/element | method |
|---|---|---|
| `ParabolicSAR` | 4.6 | one multiply-add per bar plus state |
| `SuperTrend` | 10.5 | ATR plus a ratchet |
| `KeltnerChannels` | 12.3 | EMA + ATR |
| `Vortex` | 28.8 | three rolling sums |
| `DirectionalMovement` | 29.5 | three RMA passes + ATR |
| `Choppiness` | 33.7 | rolling sum + two deques |
| `Aroon(14)` | 36.1 | **window scan** |
| `Aroon(100)` | **201** | **window scan — 5.6× the period-14 cost** |

The two recursive indicators are the cheapest in the family, which is the expected shape:
each bar costs one multiply-add plus a few state updates, with no window and no second
pass. `DirectionalMovement` is the most expensive of the smooth ones because it runs three
independent RMA passes (true range, +DM, -DM) and then a fourth over DX for the ADX itself
— four sequential passes over the data, each carrying its own recurrence.

**`Aroon` is the one measured liability in this family.** It scans its window directly,
O(window) per bar, so its cost grows with the period: **36.1 ns/element at period 14 and
201 at period 100, a 5.6× penalty**. An index-tracking monotonic deque would make it O(1),
exactly as it did for `Highest` (52.7× at window 200). It is written as a scan for now
because the scan makes the tie rule — a repeated extreme resolves to the most recent bar —
and the NaN rule explicit in the code that defines them, and because a rewrite should be
measured against this number rather than assumed. The follow-up is recorded on the
function and here.

### Regression and composed oscillators

arm64, **ns per element** at n = 1M:

| indicator | ns/element | method |
|---|---|---|
| `TRIX(15)` | 16.7 | three recurrences |
| `FisherTransform(9)` | 24.1 | two deques + recurrence |
| `MassIndex(9,25)` | 32.1 | two EMAs + rolling sum |
| `UltimateOscillator(7,14,28)` | 39.7 | six rolling sums |
| `AwesomeOscillator(5,34)` | 41.3 | two rolling sums |
| `LinearRegression(20)` | **7.2** | two running sums, O(1) |
| `LinearRegression(200)` | **7.2** | **flat in the period** |
| `StandardErrorBands(20)` | 45.8 | refits the window for the residual spread |

**`LinearRegression` was the most period-sensitive indicator in the package, and is no longer.**
It used to refit the window at every bar, so its cost grew linearly with the period: 43.7 ns/element at
period 20 against 567 at period 200.

| n = 1 048 576 | before | after | ratio |
|---|---|---|---|
| `LinearRegression(20)` | 43.7 | **7.2** | 6.1× |
| `LinearRegression(200)` | 567 | **7.2** | **79×** |

It now maintains two running sums — the window total and the position-weighted total — so the cost is
flat in the period, the same property the rolling sum and the order statistics have.

**The change was previously rejected, and the reason for rejecting it was wrong.** The earlier entry
argued that an incremental regression "accumulates its own rounding error without bound", which is true
of an *uncompensated* recurrence. The conclusion should have been to compensate it rather than to
abandon it, and Neumaier compensation on both sums removes the drift entirely. What remains is a single
subtraction, `Sxy = W − meanX·S`, that loses precision with the series' offset — measured at 2.2e-12
relative for a slope on a series around 10^4, and tabulated at several offsets on the function. That is
a real cost, it is far smaller than the one it replaces, and it is asserted by a test rather than
merely described.

`R2` and `StandardError` still refit the window, because both need the window's squared deviations and
the one-pass identity for that is the `mean(y²) − mean(y)²` form this package rejects. The split is
deliberate and is stated on each function.

**The composed oscillators are expensive because of one shared component.** The Awesome
Oscillator needs two simple moving averages and the Ultimate Oscillator needs six rolling
sums, but both land near 40 ns/element — which is roughly *two compensated rolling sums*
at the 19.2 ns/element measured in the series section. The indicator itself is nearly
free; the cost is the Neumaier compensation inside `series.Sum`, paid once per window.
That is the clearest illustration of why the periodic-re-summation follow-up in the series
section matters beyond the series package: it would show up here as a large win in every
windowed indicator that uses two or more sums, which is most of them.

`TRIX` is the cheapest because it is three recurrences and no window at all, and
`FisherTransform` sits in between because its two monotonic deques are O(1) but it pays
for the transform's logarithm.

### Correlation, realized volatility, and long-period oscillators

arm64, **ns per element** at n = 1M:

| indicator | ns/element | method |
|---|---|---|
| `WilliamsFractal(2)` | 10.7 | one 5-bar scan per index |
| `VolatilityOHLC(20)` | 15.0 | per-bar estimate + window mean |
| `WilliamsAlligator` | 16.4 | three RMAs + shifts |
| `TrueStrengthIndex(25,13)` | 23.3 | four EMA passes |
| `HistoricalVolatility(20)` | 47.5 | log returns + window stdev |
| `CorrelationCoefficient(20)` | 60.8 | two SMAs + window deviations |
| `RankCorrelation(20)` | **431** | two ranked windows per bar, index-sorted |

**`RankCorrelation` is the most expensive indicator in the package**, and it took three attempts to
bring down, only one of which worked. It ranked two twenty-bar windows per bar by calling `vec.Rank`,
which copied each window, sorted the copy, and then binary-searched every value to find its rank.

| attempt | ns/element |
|---|---|
| original | 1097 |
| replace the per-element `sort.Search` closure with an inlined binary search | 1067 |
| reuse the ranked and scratch buffers across windows | 1067 |
| **sort indices once and walk the sorted order to assign ranks** | **431** |

**The two "obvious" fixes did nothing.** The closure was real but small, and the allocation was not
the bottleneck either, despite the arithmetic looking trivially cheap next to it — forty values of
work should not cost a microsecond, and the natural inference was that the allocation must be paying
for it.

**What worked was removing redundant work.** The original assigned ranks by binary-searching each
value twice, for the lower and upper ends of its tie group: two searches per element per series, so
eighty searches per output element. Sorting indices instead costs one sort and one linear scan, and
the ranks fall out of walking the sorted order once. That is a change in what the code *does*, not in
how tightly it does it — the same shape of result as `RollingMedian` earlier in this file, where the
win also came from replacing a rescan rather than from tuning one.

`vec.RankScratchTo` exposes the scratch-taking form so a caller that ranks repeatedly does not
allocate per window; `RankTo` remains the allocating convenience.

`VolatilityOHLC` is three times cheaper than `HistoricalVolatility` for the same window,
which is the estimator's whole point: it extracts more information from each bar, and here
that shows up directly as fewer bars needed for the same statistical power.

### Ichimoku, pivots, GMMA and Zig Zag

arm64, **ns per element** at n = 1M:

| indicator | ns/element | method |
|---|---|---|
| `ZigZag(2%)` | **1.9** | one pass, O(1) state |
| `PivotPoints` | 13.5 | one scalar formula per bar |
| `Ichimoku` | 39.3 | three range midpoints + two shifts |
| `GMMA` | 57.7 | twelve EMA passes |

**`ZigZag` is the cheapest indicator in the package**, at 1.9 ns/element — and the contrast
with `RankCorrelation`'s 431 is the whole lesson of this file. Both are stateful single-pass
algorithms over the same data; one carries a running extreme and a direction flag, the other sorts
two windows per bar. **The cost of an indicator is decided by its data structure, not by how
complicated its description sounds.** Zig Zag reads as the more sophisticated indicator and runs
over two hundred times faster.

`GMMA` at 57.7 ns/element is almost exactly twelve times a single `EMA` (4.8 ns/element from
the momentum section). That is not a defect to be optimized away — the indicator *is* twelve
lines, and the fan-out is the signal — but it is a number worth knowing before computing it
on every bar rather than on the bars where the groups are read.

`PivotPoints` is cheap because each bar is one scalar formula over three numbers: no window,
no recurrence, no state.

## vec rolling kernels: both shapes, measured

`vec/rolling.go` deliberately contains two implementation shapes, so the benchmarks are arranged
to show the difference rather than a single speedup. arm64, n = 4 194 304, **ns per element**:

| kernel | window 20 | window 200 | shape |
|---|---|---|---|
| `RollingSum` | 4.0 | **4.0** | running state, O(1) |
| `RollingMax` | 5.5 | **5.4** | monotonic deque, O(1) |
| naïve rescan of the sum | 10.8 | 198 | O(window) |
| naïve rescan of the maximum | 15.4 | 318 | O(window) |
| `RollingStdDev` | 33.9 | 545 | **recomputes the window** |
| `RollingMedian` | 39.4 | **37.0** | **incremental Fenwick, flat in the window** |

**The running-state kernels are flat in the window; the naïve rescan grows linearly.** At window
200 the deque is **58.6× faster** than rescanning for the maximum and the compensated running sum
is **49.5× faster** for the sum — both larger than any constant-factor tuning in this repository,
because they are a different complexity class rather than a better loop.

**The recomputing kernels scale with the window, and that is the deliberate part.**
`RollingStdDev` costs 33.9 ns/element at window 20 and 545 at window 200. It does not use the
incremental form, because for a variance the incremental form is the `mean(x²) − mean(x)²` identity
that `vec/stats.go` documents as catastrophic.

`RollingMedian` **used to** behave the same way — 134 ns/element at window 20 and 927 at window 200,
the most expensive kernel in the package — and no longer does. It now maintains the window's order in
a Fenwick tree over compressed value ranks (see `vec/orderstat.go`), which makes insert, delete and
k-th-smallest O(log U) for U distinct values:

| n = 4 194 304 | before | after | ratio |
|---|---|---|---|
| `RollingMedian`, window 20 | 134 | **39.4** | 3.4× |
| `RollingMedian`, window 200 | 927 | **37.0** | **25×** |

**The window scaling is gone entirely**: 39.4 ns/element at window 20 against 37.0 at window 200. The
same structure also answers a prefix count, so `RollingPercentRank` came with it:

| n = 1 048 576 | before | after | ratio |
|---|---|---|---|
| `PercentRank`, window 20 | 19.5 | 20.8 | **0.94×** |
| `PercentRank`, window 200 | 170 | **20.6** | **8.3×** |
| `ConnorsRSI(3,2,100)` | 102 | **39.5** | 2.6× |

**Window 20 got slightly slower, and that is the honest result.** At that size the rescan examines
20 values while the Fenwick pays a compression sort and a log-factor descent, and the rescan wins by
about 7%. The crossover is between 20 and 200; the reason the incremental form is still the right
default is that the *curve is flat*, so the cost no longer depends on a parameter the caller chooses
for analytical reasons rather than performance ones.

`ConnorsRSI` improves by 2.6× without a line of its own being touched, because its third component
was ~97 of its 102 ns/element. That is what fixing a shared primitive looks like from above.

The recomputing kernels handle NaN for free, because a window containing a NaN sums to NaN and
recovers when the NaN leaves. That is why they carry no NaN counter while the running-state ones
do.

### Where the rolling kernels fit against the rollers

The same operations exist as `series` rollers, and the comparison is not "batch beats streaming".
Measured earlier in this file, the *streaming* roller is slower than a naïve rescan below a window
of roughly 100 elements. These batch kernels are a third option: they carry running state like the
rollers but write a whole slice at once, so they get the O(1) scaling without a per-element `Push`
call. For a full-history computation they are the right default; for incremental work the rollers
are.

## mat: dense linear algebra

arm64. GEMM is measured against an i-j-k triple loop, which is the honest baseline available in
pure Go: the tuned i-k-j order is what that loop is replaced by.

### Matrix product

| size | `Mul` (i-k-j, 4 rows per pass) | naïve (i-j-k) | ratio | `Mul` GFLOPS |
|---|---|---|---|---|
| 8 | 319 ns | 645 ns | **2.02×** | 3.2 |
| 32 | 8.8 µs | 33.3 µs | 3.77× | 7.4 |
| 128 | 502 µs | 2.87 ms | 5.73× | 8.4 |
| 256 | 4.56 ms | 25.6 ms | 5.61× | 7.4 |
| 512 | 36.8 ms | 216 ms | **5.87×** | **7.37** |

The kernel now consumes four rows of `b` per pass over an output row, which raises the ratio of
multiplies to loads inside the inner loop. Against the previous form that is **1.62× at n = 512 and
2.52× at n = 8**, and the ratio against the naïve loop went from 3.56× to 5.87×.

**The earlier negative result at n = 8 is gone.** The old kernel was *slower than the naïve loop* at
small sizes (0.79×) — the zero-skip branch cost more than it saved when the row already fit in
registers. The unrolled form is 2.02× faster than naïve there, so the crossover no longer exists.

### The scalar ceiling, and a wrong answer obtained by measuring it

The number that decides whether any of this is worth doing is the machine's scalar fused-multiply-add
rate, since Go cannot emit vector instructions. Measuring it is less obvious than it looks:

| chains | GFLOPS |
|---|---|
| 4 | 4.65 |
| **8** | **9.59** |
| 16 (over a slice) | 3.75 |

**The four-chain measurement was itself latency-limited**, and using it as the ceiling gave the
conclusion that GEMM was already at 93% of what the machine could do — which would have closed the
optimisation as impossible. Eight chains is where the plateau appears, and at 9.59 GFLOPS the real
picture is that GEMM was at **47%**.

That is the same mistake design.md's invariant 5 describes, made *inside the measurement intended to
test for it*: four chains is not enough to hide an FMA latency of several cycles, so the benchmark was
measuring the loop's dependency structure rather than the machine's throughput. The sixteen-chain row
is slower again, for the opposite reason — iterating a slice adds enough overhead to mask the
arithmetic.

With the true ceiling, GEMM at **7.37 GFLOPS is at 77% of it**. The remaining quarter is load traffic,
which register blocking might reach and vector instructions would reach outright; neither is available
in portable Go.

Matrix-vector products are memory-bound rather than compute-bound at **20.5 GB/s** for n ≥ 256, so
there is nothing left to tune there.

### Factorisations

| operation | n = 16 | n = 64 | n = 256 |
|---|---|---|---|
| `Solve` (LU, partial pivoting) | 2.18 µs | 90.4 µs | 4.88 ms |
| `Solve` (Cholesky) | 2.04 µs | 41.1 µs | **1.64 ms** |
| ratio | 1.07× | **2.20×** | **2.97×** |

**Cholesky used to be only 1.25× faster at n = 256, against a theoretical 2×.** The arithmetic is half
of LU's, so it should have been about twice as fast, and it was not.

The cause was not the loop *order* — which is what the earlier entry in this file guessed — but the
loop's **dependency structure**. Cholesky's inner loop is a reduction,
`sum -= l[i][k]*l[j][k]`, so every iteration depends on the previous one's result and the loop runs at
the *latency* of a fused multiply-add. LU's inner loop updates independent columns and has no such
chain, which is why the factorisation doing twice the arithmetic was running within 25% of the one
doing half.

Four independent accumulators give the processor four chains to interleave. The result:

| n = 1 048 576-style solve | before | after | |
|---|---|---|---|
| n = 64 | 55.1 µs | **41.1 µs** | 1.34× |
| n = 256 | 3.86 ms | **1.64 ms** | **2.36×** |

**Cholesky is now 2.97× faster than LU at n = 256, which is more than the arithmetic ratio of 2×.** The
extra comes from what LU pays that Cholesky does not: a partial-pivot search over each column and a
full-row swap, both of which are O(n) per column and cache-hostile. The arithmetic difference is 2×;
the measured difference is 3×.

At n = 16 the two are level, and Cholesky is about 15% slower than it was before the change — the
four-accumulator prologue costs more than the latency it hides when there are only a handful of
products, and the kernel falls back to a plain loop below j = 16. The absolute cost is 200 ns on a
two-microsecond operation.

**The earlier guess in this file was wrong, and it is worth leaving visible.** It proposed that the
problem was cache behaviour and that a right-looking column update was the fix. The actual problem was
instruction-level, and the fix was six lines in the existing loop. The same class of mistake appears
elsewhere in this file — see the `RankCorrelation` entries — and the pattern is consistent: when a
kernel is much slower than its arithmetic suggests, the cause is usually a dependency chain or
redundant work, not the memory system.

### Least squares

| problem | `LeastSquares` | before the Q fix | ratio |
|---|---|---|---|
| 1000 x 5 | 77.1 µs | 15.1 ms | **196×** |
| 10000 x 20 | 8.17 ms | 5.31 s | **650×** |
| 1000 x 50 | 3.40 ms | 130 ms | 38× |

**This is the largest single speedup recorded in this repository, and it came from removing work
rather than tuning any.** The first implementation called `QR`, which accumulates an explicit
m x m Q at a cost of O(n·m²) — so a 10000 x 20 regression spent 5.3 seconds forming a
10000 x 10000 matrix in order to discard all but twenty of its columns. `LeastSquares` now applies
the Householder reflectors directly to the right-hand side, which needs no Q at all and costs
O(m·n²).

The gap grows with the aspect ratio, which is exactly the shape a regression has: the more rows per
predictor, the more of the old cost was pure waste. `QR` still forms Q, and its timings are
unchanged — `QR/512x32` at 20.9 ms is still dominated by the same accumulation, which is fine
because a caller who asks for Q wants it.

### Eigensolver

| size | before | after | ratio |
|---|---|---|---|
| 8 | 5.4 µs | 5.4 µs | 1.00× |
| 32 | 259 µs | **228 µs** | 1.14× |
| 64 | 1.99 ms | **1.71 ms** | 1.16× |
| 128 | 24.9 ms | **14.4 ms** | **1.73×** |

**Every rotation walked three columns at stride n.** The matrix update touches columns p and q, and
the eigenvector accumulation touches columns p and q of V — and a column walk of an n x n matrix
touches n cache lines where a row walk touches n/16. At n = 128 each rotation was touching 256 cache
lines of eigenvector storage to do 256 multiply-adds.

The eigenvector accumulator is now kept **transposed**, so the update is two contiguous row walks, and
it is transposed back once at the end for O(n²) against the O(n³) the rotations cost. That alone is the
1.73×.

**The orientation of that transpose is the thing that can go wrong**, and the existing tests would not
all have caught it: the eigenvalues are unchanged by it, and the per-pair `A v = λ v` checks are
satisfied by `Vᵀ` too when the matrix is diagonal. `TestEigenSymEigenvectorOrientation` therefore
checks the defining property `VᵀAV = diag(λ)`, which a transposed `V` does not satisfy.

### What is left in the eigensolver, and why it was not taken

The remaining factor is algorithmic rather than mechanical: dense cyclic Jacobi needs about n²/2
rotations per sweep, while a Householder tridiagonalisation followed by implicit QL needs O(n)
rotations per sweep and roughly n³ work in total against Jacobi's ~4n³. That is a factor of about
three.

It was not attempted, and the reason is not that it is hard. `EigenSym` is used by nothing else in the
library, a dense symmetric eigensolver is the kind of routine where a subtle error produces plausible
numbers, and the current implementation is verified against `A v = λ v`, `VᵀAV = diag(λ)`, the trace
identity and a chain of analytic cases. Trading a verified 3× for an unverified one at this stage
would be the wrong direction for a library whose stated value is that its numbers can be trusted.

It is recorded here so the next person knows the factor, the shape of the work, and that the reason
for stopping was verification rather than difficulty.

### Extended moving averages, envelopes and the price oscillator

arm64, **ns per element** at n = 1M:

| indicator | ns/element | shape |
|---|---|---|
| `WeightedClose` | **1.1** | elementwise |
| `SWMA` | 4.3 | fixed 4-bar window |
| `ZLEMA` | 6.2 | de-lag + EMA |
| `KAMA(10)` | 8.3 | **recursive, O(1) — flat in period** |
| `ALMA(9)` | 8.6 | fixed window |
| `McGinleyDynamic(14)` | 18.0 | recursive with a division |
| `VIDYA(14,9)` | 26.3 | CMO (two rolling sums) + recurrence |
| `MovingAverageEnvelope(20)` | 26.5 | SMA + a pass |
| `T3(5)` | 36.4 | six EMA passes |
| `MovingAverageChannel(20)` | 39.6 | two SMAs |
| `PriceOscillator(10,30)` | 41.3 | two SMAs |
| `ALMA(100)` | **134** | **O(window) — 15.6× the period-9 cost** |

**`KAMA` is flat in its period at 8.3 ns/element, and that is the rolling kernels paying off.** It
needs a rolling sum of absolute changes, and it gets one from `vec.RollingSum` — the O(1)
running-state kernel added in the rolling wave. Written as a rescan it would have scaled with the
period like `ALMA` does; instead the adaptation costs a constant per bar, which matters because an
adaptive average is exactly the indicator a caller wants on many instruments at once.

**`ALMA` is the period-sensitive one**, at 8.6 ns/element for a 9-bar window against 134 for a
100-bar one. Its Gaussian weights depend only on the period, so the weights are hoisted, but the
window still has to be walked per bar.

**An earlier version of this entry claimed that cost had "the same follow-up" as `LinearRegression`
and that a rolling weighted accumulator would make it O(1). That was wrong**, and the reason is worth
recording because it is easy to assume otherwise. The weights are fixed *relative to the window*, so
sliding the window by one re-weights every element still inside it. A sliding weighted sum therefore
collapses to a running sum only when the weights are a low-degree polynomial in position, because
then the re-weighting can be expressed as a combination of running power sums. The weighted moving
average is the linear case and has exactly such a recurrence; a Gaussian is not, so `ALMA` has no
O(1) form and its O(window) cost is inherent rather than an implementation choice.

**`T3` costs six EMA passes** (36.4 ns/element, against roughly 6 ns for a single EMA), which is
exactly what six chained `series.NewEMA` rollers should cost. Nothing is being wasted; the indicator
is expensive because it is a sixth-order smoother.

**`MovingAverageEnvelope`, `MovingAverageChannel` and `PriceOscillator` all land near 40 ns/element
because they are two `SMA` passes**, and a single `SMA` measures 19.2 ns/element — the price of the
Neumaier compensation inside `series.Sum`, paid twice. That connects directly to the
periodic-re-summation follow-up recorded in the series section.

`WeightedClose` at 1.1 ns/element is the cheapest indicator in the package: it is pure elementwise
arithmetic with no window, no recurrence and no state, and runs at **22 GB/s**.

### Second-tier momentum, volume, stop and range indicators

arm64, **ns per element** at n = 1M:

| indicator | ns/element | shape |
|---|---|---|
| `ElderRay(13)` | 6.3 | one EMA + two subtractions |
| `ChaikinVolatility(10)` | 8.9 | one EMA + a lookback |
| `FiftyTwoWeekHighLow(252)` | 10.7 | **deque, flat in the window** |
| `PercentRank(20)` | 19.5 | **O(window) rescan** |
| `BalanceOfPower(14)` | 26.7 | one SMA over a per-bar ratio |
| `EaseOfMovement(14)` | 27.0 | per-bar ratio + one SMA |
| `KlingerOscillator(34,55,13)` | 28.5 | two EMA passes over a trending volume |
| `RelativeVolatilityIndex(14)` | 37.1 | rolling stdev + two RMA passes |
| `ChandeKrollStop(10,1,9)` | 38.3 | ATR + two deques |
| `StochasticMomentumIndex(10,3,3)` | 40.2 | two deques + four EMA passes |
| `ConnorsRSI(3,2,100)` | 39.5 | RSI, RSI over a streak, `PercentRank(100)` |
| `PercentRank(200)` | **20.6** | **incremental — flat in the window** |

**`FiftyTwoWeekHighLow(252)` costs 10.7 ns/element, almost the same as `RollingMax` at 5.4**, and
that is the monotonic deque doing its job across a 252-bar window: the cost is flat in the window, so
asking for a 52-week extreme costs what asking for a 20-bar one does. A rescan would have been about
**318 ns/element** at that window, on the measurements in the rolling section — a 30× difference
between two ways of writing one line.

**`PercentRank` is no longer the rescan in this group.** It delegates to `vec.RollingPercentRank`,
which shares the Fenwick structure that made `RollingMedian` flat in the window, so its cost is
20.6 ns/element at both window 20 and window 200 — see the rolling section above for the
before-and-after.

**`ConnorsRSI` was dominated by its own third component** — `PercentRank(100)` was ~97 of its
102 ns/element, so the two RSI passes together cost about 5. Fixing the rank kernel took the
composite to 39.5 ns/element without touching a line of its own code.

`ElderRay` at 6.3 ns/element and 3.8 GB/s is the cheap end: a single EMA pass and two subtractions,
with no window and no second smoother.

`AccumulativeSwingIndex` measures **17.9 ns/element** at n = 1M (1.79 GB/s over its four input
series). It is a single pass with an accumulator and no window, so the cost is the per-bar arithmetic:
two absolute differences, an absolute body, a three-way branch and the ratio.

## The batch/streaming switch, and a cautionary result

`ta.SMA` used to be `series.Apply(xs, series.NewSMA(n))` — the streaming roller driven over a whole
slice. It now calls `vec.RollingMean`. arm64, **ns per element** at n = 1 048 576:

| indicator | streaming sum | batch kernel | ratio |
|---|---|---|---|
| `SMA` | 19.6 | **4.9** | **4.0×** |
| `MovingAverageEnvelope(20)` | 26.5 | **6.4** | 4.1× |
| `MovingAverageChannel(20)` | 39.6 | **10.0** | 4.0× |
| `PriceOscillator(10,30)` | 41.3 | **12.4** | 3.3× |
| `AwesomeOscillator(5,34)` | 41.3 | **13.2** | 3.1× |
| `UltimateOscillator(7,14,28)` | 39.7 | **40.3** | **1.0×** |

Every indicator whose cost was two or more simple moving averages gained about the same factor,
because `SMA` was the whole of its cost: the envelope, the channel and the price oscillator were
paying for two streaming sums and doing almost nothing else.

**`UltimateOscillator` gained nothing, and that is the interesting entry.** It went from six streaming
sums to six batch ones — the same substitution that gave 4× everywhere else — and moved by one
percent, in the wrong direction.

### Why the micro-benchmark could not have predicted that

Before making the change, a standalone benchmark of the two summation forms reported 20 against
4 ns/element for the batch kernel. That looked like a 5× win for `UltimateOscillator`, which had six
of them. It was not, and the reason turned out to be a property of the roller rather than a
measurement artefact.

Measuring the streaming path directly, with the number of *independent* rollers varied:

| variant | 1 roller | 2 | 4 | 6 |
|---|---|---|---|---|
| **parallel** — n independent rollers, each pushed every value | 19.1 | 21.5 | 27.2 | **28.8** |
| **chained** — one value fed through n rollers in sequence | 19.1 | 21.8 | 34.2 | — |

ns per element at n = 1M. **Six independent rollers cost 1.5× one, not 6×.** The marginal cost of an
extra roller is **1.9 ns/element** against **19.1 ns/element for the first one** — an order of
magnitude apart.

**The roller is latency-bound, and independent rollers overlap.** A rolling sum carries a dependency
chain through `total`, `comp` and the ring, and each element depends on the previous element's state.
With one roller there is exactly one such chain and the loop runs at the chain's *latency*, with
nothing for the processor to overlap it with. Adding a second, independent roller gives the hardware a
second chain to interleave, so the marginal cost of a roller is a fraction of the first one's. The
`chained` row confirms the reading from the other direction: folding the rollers into one longer chain
costs 1.8× at depth 4 where four separate chains cost 1.4×.

That closes the puzzle. Six streaming sums cost `19.1 + 5×1.9 = 28.8` ns/element, and six batch sums
cost `6×4.0 = 24`. Both land near the measured `UltimateOscillator` total of ~40 ns/element once the
per-bar buying-pressure and true-range construction is added — which is why replacing one with the
other changed nothing.

**The rule this repository now follows: an indicator's own end-to-end benchmark is the evidence for a
change to it.** A kernel-level number is used to *find* candidates and never to *justify* one. The
batch forms were kept in `CMO` and `UltimateOscillator` for consistency with the rest of the package,
not on the strength of a speedup that did not appear.

It is worth noting where the original reasoning went wrong, because the mistake is easy to repeat: the
single-roller benchmark *was* correct. It was the multiplication by six that was invalid, and nothing
about the number itself said so. Only measuring the multi-roller case revealed it, which is why the
`BenchmarkSumChainParallel`/`Serial` pair is now a permanent part of `series`' benchmarks rather than a
throwaway investigation.

## Exponential smoothers: the batch form, and the prediction that held

`EMA` and `RMA` are the same recurrence with a different smoothing factor, and they drive most of the
Wilder family: `ATR`, `DirectionalMovement`, `MACD`, `T3`, `TrueStrengthIndex`, `KlingerOscillator`,
`StochasticMomentumIndex`, `RelativeVolatilityIndex`. `ta.EMA` and `ta.RMA` now call
`vec.RollingEMA`/`RollingRMA`, and every composite that spliced a streaming smoother calls the batch
form instead.

arm64, **ns per element** at n = 1M, before and after:

| indicator | before | after | ratio |
|---|---|---|---|
| `EMA(20)` | 4.87 | **2.07** | **2.35×** |
| `MACD(12,26,9)` | 16.46 | **8.10** | **2.03×** |
| `ATR(14)` | 7.55 | **3.97** | 1.90× |
| `DirectionalMovement(14)` | 30.0 | **19.0** | 1.58× |
| `T3(5, 0.7)` | 36.7 | **23.9** | 1.54× |
| `KlingerOscillator(34,55,13)` | 28.5 | **21.0** | 1.36× |
| `StochasticMomentumIndex(10,3,3)` | 40.2 | **31.0** | 1.30× |

The kernel itself measures **1.84 ns/element**, flat in the window (1.84 at window 20 and 1.84 at
window 200 at n = 4M), against a naive uncompensated baseline of 2.2 — so the Neumaier-compensated
seed costs less than the run-to-run noise, which is what one expects when the seed pass is n
operations and the recurrence is N.

### This time the micro-benchmark predicted the indicator result

The kernel comparison, run as a single A/B rather than across separate runs, gave 4.9 ns/element for
the streaming form and 1.78 for the batch one — a factor of 2.7. The indicators then came in at
1.3-2.4×, with the ones dominated by a single smoother closest to the prediction (`EMA` at 2.35×) and
the ones with substantial non-smoother work furthest (`StochasticMomentumIndex` at 1.30×, which is two
deque passes plus four smoother passes).

That is the opposite of the rolling-sum case above, and the difference is worth stating: the rolling
sum's cost was a *latency* that overlapped between independent rollers, so multiplying it by the
number of rollers was invalid. A smoother's cost is mostly the per-element recurrence, which does not
overlap away. **The lesson from the earlier round still holds and is not "micro-benchmarks are
useless" — it is that a kernel number only transfers when the kernel is the bottleneck.**

### Two mistakes made and caught in this change

**A silent replacement.** The first attempt to switch `ta.EMA` rewrote `ta.RMA` and left `ta.EMA`
untouched, because the string being matched did not appear verbatim and the script printed success
without checking. It was caught by measurement — `ATR` improved 2.3× while `EMA` did not move at all,
which is not a combination the change could produce if both had applied. Scripted edits to source
files now assert that the replacement happened.

**A zeroed warm-up.** The new splice helper allocated its output slice with `make` and copied the
kernel output into `out[from:]`, leaving `out[0:from]` as **zeros rather than NaN**. A zero in a
warm-up region is a *value*, and the result was that Klinger's oscillator reported a defined value at
index 0, which seeded its signal line one step early and made two different signal-period settings
produce the same series. The facade test that checks each period reaches the implementation caught it.
`applyEMA`/`applyRMA` now fill the warm-up explicitly, and `TestApplyEMAWarmUpIsNaN` plus
`TestCompositesHaveNaNWarmUps` pin it across every composite that uses a splice.

### Catalogue-completion indicators

arm64, **ns per element** at n = 1M:

| indicator | ns/element | shape |
|---|---|---|
| `Ratio` | **1.0** | elementwise, 15.7 GB/s |
| `EMACross(12,26)` | 7.6 | two average passes + a sign comparison |
| `HammingMA(9)` | 8.6 | fixed 9-bar weight profile |
| `RelativeVigorIndex(10,4)` | 19.3 | two rolling sums over a per-bar ratio |
| `ChopZone(14)` | 25.3 | two rolling sums + a logarithm |
| `StandardDeviation(20)` | 32.6 | **recomputes the window** |
| `HammingMA(100)` | **146** | **O(window) — 17× the period-9 cost** |

`Ratio` at 1.0 ns/element and 15.7 GB/s is the memory-bandwidth floor for an elementwise kernel, and
it is worth having as a reference point: nothing that touches a window can approach it, and the gap
between it and the cheapest windowed kernel is the price of the window.

**`HammingMA` is the clearest illustration of a cost that is inherent rather than an
implementation choice.** Its cost scales linearly with the period — 8.6 ns/element at period 9
against 146 at period 100 — and unlike `ALMA` there is no plausible future fix. A fixed weight
profile re-weights every element still inside the window each time the window slides, so a sliding
weighted sum reduces to a running sum only for weights that are a low-degree polynomial in position.
`WMA` is the linear case and has exactly such a recurrence; the Hamming window is a raised cosine and
does not. The O(window) cost is what the formula *is*.

`StandardDeviation` at 32.6 ns/element scales with the window for the accuracy reason recorded in the
rolling section: it recomputes each window about its own mean rather than using the one-pass
`mean(x²) − mean(x)²` identity.

## Summary table

arm64, n=4 194 304, throughput in GB/s:

| operation | naïve | tuned | speedup | limited by |
|---|---|---|---|---|
| `Sum` | 6.4 | 25.4 | 3.98× | **add latency** (not bandwidth) |
| `Dot` | 9.6 | 38.4 | 4.00× | **mul latency** (near ceiling) |
| `Min` | 4.81 | **9.48** | **1.97×** | compare latency |
| `Min` (no NaN check) | 4.81 | 9.58 | 1.99× | compare latency |
| `MinMax` | — | 6.4 | 1.42× vs 2 passes | compare latency |
| `Add` | 37.0 | 47.0 | 1.27× | memory bandwidth |
| `ReadOnlyTuned` | — | 37.8 | — | **the practical ceiling for one pass** |

## Conclusions

1. **The tuning is real, for reductions.** ~4× on `Sum` and `Dot` at realistic
   sizes, measured rather than asserted.
2. **It is not free at small sizes.** `Dot` at n=8 is 25 % *slower* than naïve.
3. **Different loops are limited by different things**, so the same tuning strategy
   produces very different returns: reductions gain ~4×, elementwise maps gain
   ~1.3×, scans are limited by compare latency no matter what.
4. **The reductions are NOT bandwidth-bound** (corrected). A same-shape loop that
   moves the same bytes but does no real arithmetic reaches 37.8 GB/s against
   `Sum`'s 25.4 GB/s. The bottleneck is floating-point add latency, which means
   SIMD has real headroom. **FMA does not**: it is already active in the tuned
   loops, because the compiler contracts `a + x*y` by default — see the FMA
   evidence section above and [next-steps.md](next-steps.md) §7.2. The 1.49× in
   §4 is thus headroom *after* FMA, not before it.
5. **Two claims in earlier revisions of this file were wrong**, and both are
   recorded rather than deleted: the "bandwidth ceiling" reading above, and the
   assertion that `Min` was no faster than naive. The first was inferred from the
   shape of a curve instead of measured; the second was a real performance bug,
   now fixed by fusing the NaN check into the scan loop (0.98× → 1.97×).
6. **Everything here is reproducible** with the commands at the top of this file.
   If your numbers disagree with these, trust yours — the machine is the
   authority, and this file is only a record of one machine.
