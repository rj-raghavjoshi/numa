# Design

Reference documentation for the structure of this package. For the *reasoning*
behind the tuned loops, read [learn/](learn/) instead; this file is for people
modifying the code.

---

## File layout

```
numa/
├── go.mod
├── numa.go                 package docs, Version, and the original core forwarders
├── facade_math.go          forwarders for vec/math.go, one per family as the API grows
├── facade_math_test.go     verifies the forwarding (argument order, panics, NaN)
├── facade_mask.go          forwarders for vec/mask.go
├── facade_mask_test.go
├── facade_cum.go           forwarders for vec/cum.go
├── facade_cum_test.go
├── facade_signal.go        forwarders for vec/signal.go
├── facade_signal_test.go
├── facade_order.go         forwarders for vec/arg.go and vec/order.go
├── facade_order_test.go
├── facade_stats.go         forwarders for vec/stats.go
├── facade_stats_test.go
├── facade_rolling.go       forwarders for vec/rolling.go
├── facade_rolling_test.go
├── facade_ta*.go           forwarders for the ta indicator families
├── facade_ta*_test.go
├── numa_test.go            verifies the original core forwarding
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
│   ├── math.go             elementwise math maps (no arch variant — see below)
│   ├── mask.go             predicates, mask algebra, Where/Compress, masked reductions
│   ├── cum.go              cumulative scans (CumSum/Prod/Max/Min, CumCountTrue)
│   ├── signal.go           lag/difference/signal (Shift, Diff, Rate, Cross, BarsSince)
│   ├── arg.go              ArgMin/ArgMax/ArgMinMax public layer, HasNaN
│   ├── order.go            order statistics (Median/Quantile/Rank/MAD, quickselect)
│   ├── stats.go            descriptive statistics (Variance/Covariance/Correlation/Skew/Kurt)
│   ├── rolling.go          rolling-window batch kernels (sum/mean/extremes/stddev)
│   ├── orderstat.go        sliding Fenwick tree: rolling quantile/median/percent rank
│   ├── smoothing.go        rolling exponential smoothers (EMA, RMA)
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
│   ├── math_test.go        math maps + the FMA single-rounding contract
│   ├── mask_test.go        predicates, mask algebra, selection, masked reductions
│   ├── cum_test.go         prefix sums, carry handoff, forward NaN poisoning
│   ├── signal_test.go      lag boundaries, cross semantics, reset runs
│   ├── arg_test.go         first-occurrence ties, -Inf seeding, NaN sentinel
│   ├── order_test.go       quickselect correctness, interpolation edges, no input mutation
│   ├── stats_test.go       hand-computed moments, two-pass vs one-pass accuracy
│   ├── rolling_test.go     window references, NaN recovery, To-form equivalence
│   ├── orderstat_test.go   rescan references, ties, all-NaN, k-th order for every k
│   ├── smoothing_test.go   seed convention, streaming equivalence, boundary windows
│   ├── helpers_test.go     shared refs, boundaryLengths, closeEnough
│   ├── bench_test.go       benchmarks + naive baselines
│   └── ...
│
├── series/                 stateful streaming rollers (O(1) per element)
│   ├── doc.go              package documentation
│   ├── ring.go             Ring: fixed-capacity circular buffer
│   ├── roller.go           the Roller interface, Apply / ApplyTo
│   ├── compensate.go       Neumaier compensated addition
│   ├── sum.go              Sum, SMA
│   ├── ema.go              EMA, RMA (SMA-seeded smoothers)
│   ├── ring_test.go        eviction order, exact capacity, reset
│   ├── sum_test.go         reference agreement, drift, NaN recovery
│   ├── ema_test.go         seeding convention, warm-up boundary
│   ├── roller_test.go      interface vs concrete dispatch, driver contract
│   ├── helpers_test.go     boundaryWindows, references
│   └── bench_test.go       streaming vs window recomputation
│
├── ta/                     technical indicators (plain slices, NaN warm-up)
│   ├── doc.go              the six package-wide conventions
│   ├── warmup.go           FirstValid, applyFrom, shared checks
│   ├── price.go            Median/Typical/AveragePrice, TrueRange, Change
│   ├── moving.go           SMA/EMA/RMA/WMA/DEMA/TEMA/TRIMA/HMA
│   ├── extrema.go          Highest/Lowest (monotonic deque), Donchian
│   ├── volatility.go       ATR, BollingerBands, %B, width
│   ├── momentum.go         RSI, MACD, Stochastic, StochRSI, %R, CCI, CMO, ROC
│   ├── volume.go           VWAP, VWMA, OBV, A/D, CMF, Chaikin, MFI, PVT, ForceIndex
│   ├── trend.go            ADX/DMI, Aroon, Vortex, Choppiness, Keltner, SuperTrend, PSAR
│   ├── regression.go       least-squares fit and its slope, R², standard error, bands
│   ├── oscillator.go       Awesome, Accelerator, DPO, TRIX, UO, Fisher, Mass Index
│   ├── correlation.go      Pearson/log/Spearman correlation, realized volatility
│   ├── misc.go             Coppock, KST, TSI, Alligator (shifted), Fractal (causal)
│   ├── ichimoku.go         the five lines, displacement made explicit
│   ├── pivot.go            PivotMethod and the four pivot formulas
│   ├── gmma.go             GMMA (twelve EMAs) and the causal ZigZag
│   ├── adaptive.go         ALMA, SWMA, KAMA, VIDYA, ZLEMA, T3, McGinley
│   ├── envelope.go         MA envelope, MA channel, price oscillator
│   ├── momentum2.go        SMI, RVI, ConnorsRSI, ElderRay, PercentRank
│   ├── volume2.go          BalanceOfPower, EaseOfMovement, Klinger, Chaikin vol
│   ├── stop.go             ChandeKrollStop
│   ├── swing.go            Wilder's Swing Index and its accumulation
│   ├── catalogue.go        Ratio/Spread/StdDev, crosses, vigour, chop, Hamming
│   ├── helpers_test.go     naive references, synthetic OHLCV, synthHLC
│   ├── price_test.go       hand-computed transforms
│   ├── moving_test.go      warm-up boundaries, composite seeding
│   ├── volatility_test.go  extremes, ATR and Bollinger against references
│   ├── momentum_test.go    hand-computed oscillators, division-by-zero cases
│   ├── volume_test.go      cumulative lines, volume-weighted ratios, degenerate bars
│   ├── trend_test.go       invariants for recursive indicators, worked DI values
│   ├── regression_test.go  straight-line and constant fixed points, OLS reference
│   ├── oscillator_test.go  composition relationships, clamps, range bounds
│   ├── correlation_test.go fixed points (self/negation), monotonic invariance
│   ├── misc_test.go        shift delays, fractal confirmation index, strict ties
│   ├── ichimoku_test.go    displacement sign and direction, zero-displacement case
│   ├── pivot_test.go       worked levels per method, previous-bar causality
│   ├── gmma_test.go        twelve independent EMAs, hand-traced ZigZag confirmations
│   ├── adaptive_test.go    adaptation properties (fast on a line, frozen on noise)
│   ├── envelope_test.go    band/offset relations, ordering, unit of percent
│   ├── momentum2_test.go   bounds as properties, streak counter, composition
│   ├── volume2_test.go     degenerate divisors, stop invariant without monotonicity
│   ├── swing_test.go       hand-computed bar, running-total relation, NaN permanence
│   ├── catalogue_test.go   ratios of sums, log-domain symmetry, crossing events
│   └── bench_test.go       per-indicator, deque vs rescan, period sensitivity
│
├── mat/                    dense linear algebra (no facade; see below)
│   ├── mat.go              the Mat type, products, elementwise ops, norms
│   ├── linalg.go           LU, Cholesky, Householder QR, least squares, Jacobi
│   ├── mat_test.go         naive-product reference across shapes, panics
│   ├── linalg_test.go      residual and reconstruct-the-input identities
│   └── bench_test.go       GEMM vs naive, Cholesky vs LU, least-squares shapes
│
├── internal/comp/          Neumaier addition shared by vec and series
│   └── comp_test.go
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

### Why `series` is a separate package, and why it is not forwarded

`vec` and `series` answer the same questions about the same data with two different
access patterns: `vec` has the whole slice, `series` sees one value at a time and
carries state. Neither wraps the other. A full-history computation is usually faster
batch, because it can hold several independent accumulators; a per-step computation
must be streaming or it pays O(window) per element. Measured, the streaming cost is
flat in the window and the recomputation is not, but the streaming path loses below a
window of roughly 100 — see benchmarks.md.

`series` is **not** re-exported through the `numa` facade, unlike `vec`. The facade's
value comes from shortening calls to *slice-in, slice-out function kernels*
(`numa.Sum(xs)` instead of `vec.Sum(xs)`). `series`'s API is mainly *types* —
`SMA`, `EMA`, `Sum` rollers — and aliasing thirty types into the root package would be
noise, while forwarding only its two free functions (`Apply`, `ApplyTo`) would be an
arbitrary half-measure. So `series` is imported directly:

```go
s := series.NewSMA(20)
for _, v := range closes {
    out = append(out, s.Push(v))
}
```

`ta`, when it arrives, is function-shaped again (slices in, slices out) and *will* be
forwarded. The rule is about shape, not about package identity: **function kernels get
forwarders, type-bearing packages are imported directly.**

### How the three layers relate

```
ta     indicators, named as a charting layer names them   (ATR, RSI, MACD)
 ↓
series stateful rollers for incremental use                (SMA, EMA, Sum)
 ↓
vec    batch kernels, tuned per architecture               (Sum, Dot, Min2, quickselect)

mat    dense linear algebra, alongside the layers above     (Mul, Solve, Cholesky, QR, EigenSym)
```

Each layer may use the one below and none may use the one above. That direction is
what keeps `vec` free of any notion of an indicator and `series` free of any notion of
a price column.

The boundary is not a pure hierarchy, though, and the exceptions are deliberate:

* **`ta` delegates rather than reimplements.** `ta.SMA`, `ta.EMA` and `ta.RMA` are the
  series rollers driven over a whole slice; `ta.Highest`, `ta.Lowest` and `ta.WMA` are
  the `vec` rolling kernels. Two implementations of the same smoothing would drift
  apart, and the seeding convention is exactly the kind of detail that drifts. Every
  delegation is a one-line body, so a reader can see there is no second definition.
* **`series` does not depend on `vec`.** A rolling sum is not a batch sum, and forcing
  one through the other would add an indirection for nothing. The two are the same
  mathematics arranged for different access patterns, and both are tested against the
  same naive references.
* **The one thing they do share lives in `internal/comp`.** Neumaier compensated addition
  is six lines of arithmetic that both need and neither should own. It is an internal
  package rather than a duplication or an upward import, which keeps the
  `series`-does-not-import-`vec` rule intact while leaving one definition of the
  algorithm. It has its own test, because a defect in shared arithmetic would corrupt
  both packages while every test in either still passed — the same reasoning that gave
  the synthetic data generator its own test.
* **The rolling kernels exist in both shapes, deliberately.** `RollingSum` and
  `RollingMax` carry running state and are flat in the window; `RollingStdDev` and
  `RollingMedian` recompute the window and scale with it, because the incremental forms
  are the numerically unstable ones this package rejects. Which shape an operation has is
  an algorithmic decision with a measured cost (benchmarks.md), not an accident.
* **Batch is not automatically faster.** Measured, the streaming roller is *slower*
  than rescanning a window below roughly 100 elements (benchmarks.md), so `ta` uses
  the form that wins for each indicator rather than assuming either one.
* **A kernel-level speedup is not an indicator-level speedup.** Replacing a streaming
  sum with the batch kernel gave `SMA` 4× and `UltimateOscillator` — which has six of
  them — nothing at all. The reason is that **`series.Sum` is latency-bound**: it carries
  a dependency chain through `total`, `comp` and the ring, so one roller runs at the
  chain's latency and additional *independent* rollers overlap with it. Measured, six
  independent rollers cost 1.5× one, and the marginal cost of an extra roller is
  1.9 ns/element against 19.1 for the first. Six streaming sums therefore cost about the
  same as six batch ones, which is why the substitution changed nothing.
  The rule that came out of it: an indicator's own end-to-end benchmark is the evidence
  for a change to it, and kernel numbers are for finding candidates rather than for
  justifying them. `BenchmarkSumChainParallel`/`Serial` in `series` exist to keep this
  visible, because the mistake — multiplying a single-chain cost by the number of chains —
  is invisible in the single-chain number itself.

### Why `mat` has no facade

`series` is not forwarded either, and for the same reason: the facade rule is **shape-based**.
A package that is a set of functions over slices (`vec`, `ta`) gets forwarders, because
`numa.Sum(xs)` reading as one name is the point of a facade. A package that defines a *type*
(`series.Roller`, `mat.Mat`) does not, because a forwarder cannot re-export the type without an
alias and a caller ends up importing both names anyway. `PivotMethod` is the one aliased type,
and it is aliased because a *constant* of it is useful without the rest of the package.

The practical consequence is that `mat` is used as `mat.Solve(a, b)` rather than `numa.Solve`,
which is also better: a linear-algebra call in the middle of an indicator pipeline should say
which layer it comes from.

### Why the `vec` facade is split across files

`numa.go` originally held every forwarder, and the rule was one file. That stops
working at a few hundred names: the file becomes a wall of one-liners with no
structure. Forwarders are therefore grouped into `facade_<family>.go` files
(`facade_math.go`, `facade_rolling.go`, `facade_ta.go`), with a matching
`facade_<family>_test.go`. `numa.go` keeps the package documentation, `Version`,
and the original core forwarders.

The rule that matters is unchanged: **every exported free function in `vec` has
exactly one forwarder in `numa`, and the forwarder has a test.** Only the filing has
changed. `ta` will follow the same rule; `series` does not, for the shape reason given
above.


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

### Why some kernels have no arch variant

The dispatch pattern above exists to tune a **loop-carried dependency chain**: how
many independent accumulators the register file can hold, and therefore how long
the chain is. A map has no such chain — `dst[i]` depends only on `xs[i]` — so there
is nothing for an accumulator to hide and nothing for a build tag to choose between.

`vec/math.go` and `vec/mask.go` are therefore each a single untagged file. Their
wide unrolled loops *are* the tuned loops, and they are identical on every
architecture. Three byte-identical copies behind build tags would be duplication
dressed up as tuning.

The rule, stated so it can be applied to future operations:

| operation shape | loop-carried chain? | arch variant? |
|---|---|---|
| reduction (`Sum`, `Dot`) | yes — one chain per accumulator | **yes**, accumulator count differs |
| scan (`Min`, `Max`, `ArgMin`, `CumMax`) | yes — and not shorten-able by pairing | **yes**, except where blocking does not help (see below) |
| multipass statistic (`Variance`, `Correlation`) | each pass is a reduction | **yes**, each pass is arch-tuned separately |
| rolling window (`RollingSum`, `RollingMax`) | running state, O(1) per element | **no** — an algorithmic shape, not a register-count choice |
| rolling window, recomputing (`RollingStdDev`, `RollingMedian`) | O(window) per element by design | **no** — the incremental form is the unstable one (see below) |
| elementwise map (`Neg`, `Div`, `Min2`) | no | **no** — one untagged file |
| predicate / mask (`Greater`, `And`) | no | **no**, but still unrolled (compare-bound) |
| transcendental map (`Sqrt`, `Log`) | no | **no**, and not even worth unrolling; the library call dominates (measured in benchmarks.md) |

The **multipass** row is the one worth reading before writing a new statistic. The
instinct is to find the one-pass identity — `mean(x²) − mean(x)²` for variance — and
the instinct is wrong twice over. It is numerically catastrophic, *and* it is slower,
because a single untuned pass loses to two tuned ones. Measured: the two-pass
`Variance` is 3.36× faster than a naïve two-pass **and** 1.91× faster than the naïve
one-pass, while being the only accurate form of the three. A statistic that needs the
mean before it can accumulate deviations should read the input twice and tune both
passes; the second traversal is not the expensive part. See benchmarks.md.

`stats.go` holds only the public functions and the untagged third/fourth-power sums;
the deviation reductions it depends on (`dotDevArch`, `sumAbsDevArch`) live in the
three arch files, because those *are* ordinary reductions and get the usual tuning.

`Min2` and `GreaterTo` are the interesting middle cases: both are maps, but both
carry a compare, so the unroll buys far more than the ~1.3× of a pure arithmetic map
— **1.69×** for `Min2` and **1.93×** for `GreaterTo`, the latter despite writing
eight times less data than a float map. A compare that is *not* loop-carried still
benefits from having several in flight. That is the test for whether to unroll a map,
and it is measured, not assumed.

`vec/cum.go` is untagged for a third reason, and it is the one that would otherwise
look like a rule violation. Its loops **do** carry a dependency, so the map argument
does not apply — but a prefix scan cannot use independent accumulators at all,
because output i needs the running value and there is only ever one chain. It is
shortened by *blocking* instead, and the block size is an algorithmic choice rather
than a register-count choice, so it does not vary by architecture either. The
distinction worth keeping straight: arch tags exist for tuning that depends on the
machine; `cum.go` contains no such tuning.

`vec/arg.go` is the mirror case — a file that holds only the public layer because the
operations underneath it *do* get arch tuning. `ArgMin` and `ArgMax` are scans:
compare-bound, with a second compare for the NaN policy. The first single-chain
version measured **0.51×**, half the speed of a plain loop, because the two compares
contend for one port with nothing to hide them. The tuned implementations live in the
three arch files as `argMinArch`/`argMaxArch`/`argMinMaxArch`, with two chains on
arm64 and four on x86-64. **The lesson generalizes: adding a per-element test to a
latency-bound scan is not free, and the fix is always more independent chains, never
a better single chain.** See benchmarks.md.

Arg reductions also carry a constraint the value scans do not. Independent partial
scans can find the same extreme value at different indices, so the combine step must
break an equality by index; a combine that compares only values returns the wrong
index for a tie that straddles the chain boundary. `TestArgReductionsTiesResolveFirst`
exists for exactly that, and it caught a real amd64 combine bug during development.

### Naming

| suffix | meaning |
|---|---|
| `sumArch`, `dotArch`, `minArch`, … | private, one per architecture, no edge-case handling |
| `scanMin2`, `scanMin4` | private scan helpers, named for the accumulator count |
| `naiveSum`, `naiveDot` | benchmarks only, portable baseline |

The `2` / `4` suffix on scan helpers encodes the accumulator count, which is the
one thing that differs between the ARM64 and x86-64 versions.

There is deliberately no `*Arch` name for the untagged map kernels: the suffix
promises an architecture-specific implementation, and there isn't one to promise.

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

The default documented contract is that `dst` may alias `xs` and/or `ys`. This holds
because the loop writes `dst[i]` only after reading `xs[i]` and `ys[i]` in the
same iteration. **A tuned loop that reads ahead of its writes would break this.**
If you restructure an `*To` function to buffer ahead, you must update the
documentation *and* `TestToVariantsAliasing`.

There are now **three documented exceptions**, and they share a cause — the loop needs an
element it has already overwritten:

| exception | why |
|---|---|
| `ReverseTo` | writes `dst[i]` from `xs[n-1-i]`, so iteration i destroys an element a later iteration reads |
| every lagged `*To` in `signal.go` (`ShiftTo`, `DiffTo`, `RateTo`) | reads `xs[i-lag]`, which was overwritten at iteration `i-lag`; reading a block ahead does not help because the element was written several iterations earlier |
| every rolling `*To` in `rolling.go` (`RollingSumTo`, `RollingMeanTo`, `RollingMaxTo`, `RollingMinTo`, `RollingRangeTo`, `RollingWMATo`) | reads `xs[i-n]` as the value leaving the window, and by then iteration `i-n` has written the output over it |

Both are stated in the function's own documentation. Neither is detectable at
runtime without an allocation or a copy that every correct caller would pay for, so
the restriction is enforced by contract rather than by a check. **When adding a
lagged operation, state the restriction and do not add an aliasing test that would
pass for the wrong reason.**

### 3. The NaN check in scans is fused, not a second pass

An earlier revision of this file said `Min`, `Max` and `MinMax` call a `hasNaN`
helper that walks the input again. **That is no longer true, and the helper no
longer exists.** The NaN test is fused into the scan loop, one boolean flag per
accumulator, combined after the loop. Measured, fusion costs about 1% and a second
pass costs ~2× the tuning gain; see [next-steps.md](next-steps.md) §3.

The invariant that survives is the reason it was written: **do not add a second
pass to a latency-bound scan, and if you change what the scan loop carries, you must
re-benchmark.** `BenchmarkMinPureScan` exists as the upper bound and is the
regression detector — if `BenchmarkMin` drifts away from it, the fusion has been
undone.

The flags are per-accumulator rather than one shared flag so the accumulator chains
stay independent; a shared flag would be its own loop-carried dependency.

### 4. Generic implementations stay naive

`arch_generic.go` is the reference used to justify the tuning. **Do not optimize
it.** If it becomes clever, it stops being a baseline, and the benchmarks lose
their meaning.

### 5. A reduction loop runs at latency, not throughput

This one is not a correctness invariant but it is the most frequently useful thing in this file, and it
has now explained four separate slow kernels.

**If a loop's body accumulates into a single variable, the loop cannot go faster than the latency of
that accumulation.** `sum -= a[i]*b[i]` is a chain: iteration *k+1* needs iteration *k*'s result, so the
processor has nothing to overlap and each iteration costs the full latency of a fused multiply-add
rather than its throughput. The fix is almost always the same — **several independent accumulators**,
combined once at the end — and it turns a latency-bound loop into a throughput-bound one.

The four cases where this was the answer:

| kernel | symptom | fix |
|---|---|---|
| `vec.ArgMin`/`ArgMax` | fusing the NaN check into one chain halved throughput | two chains on arm64, four on amd64 |
| `mat.Cholesky` | 1.25× faster than LU instead of 2× | four accumulators in the dot product |
| `series.Sum` and its relatives | see below | — the chain is the *interface*, not the arithmetic |
| `RankCorrelation`, `RollingMedian` | a microsecond for forty values of work | the chain was redundant work, not latency |

There is a second, closely related limit: **the ratio of memory operations to arithmetic**. A loop
doing one multiply-add per two loads and a store cannot exceed half the machine's FMA rate no matter
how independent its iterations are, which is what `mat.Mul`'s inner loop was doing. Consuming four rows
of the right-hand operand per pass raised that ratio and gave 1.62×, with no change to the dependency
structure at all.

The diagnostic that matters: **when a kernel is far slower than its arithmetic suggests, suspect a
dependency chain, redundant work, or the memory-operation ratio, in that order, before suspecting the
memory system.** Cache behaviour is the usual first guess and, in this repository, has been the right
one only once.

**And the same caution applies to the benchmark you write to test for it.** The first version of the
FMA-ceiling benchmark used four accumulator chains, which is not enough to hide an FMA's latency, so it
measured the loop's dependency structure rather than the machine's throughput. That produced a ceiling
of 4.65 GFLOPS against a true 9.59, and a conclusion — "`mat.Mul` is already at 93% of what is
achievable" — that was wrong by a factor of two and would have ended the investigation. Any benchmark
whose *purpose* is to measure a limit has to show the limit being reached, not assume it was.

The converse is also true and is recorded in the streaming section: `series.Sum` is latency-bound on
purpose, and that is exactly why six independent rollers cost 1.5× one rather than six times as much.
The same property that makes a single chain slow makes several chains nearly free.

### 6. Compiler contraction changes results, and it is on by default

Go contracts `a + x*y` into a hardware fused multiply-add unless `GOFMAHASH` is
set — `useFMA` in `cmd/compile/internal/ssa/func.go` returns true by default — and
`math.FMA` is a compiler intrinsic. Two consequences for anyone touching these
loops:

* **A "two roundings" comment may be false.** `vec/math.go`'s `MulAdd` is written
  as a multiply and an add, but on arm64 it compiles to `FMADD`. Do not document a
  rounding count from the source. Disassemble it:

  ```sh
  go test -c -o /tmp/vec.test ./vec/
  go tool objdump -s 'vec\.dotArch$' /tmp/vec.test | grep FMADD
  ```

* **Use `vec.FMA`, not `MulAdd`, when one rounding must be guaranteed.**
  `TestFMAIsSingleRounding` pins the fused result independently of the platform.
  `TestMulAddContractionIsObservable` asserts only that `MulAdd` produced one of the
  two permitted roundings, and logs which — pinning one would fail on a platform
  that made the other choice, which is exactly the kind of "correct on my machine"
  test this repository avoids.

This is also further evidence for invariant §6.2 in next-steps: contraction is one
more reason results are not bit-identical across architectures.

**And a tolerance calibrated on one architecture is not a tolerance.** The rank check in
`mat.LeastSquares` was first written with a criterion of `n * eps * |R[0][0]|`, which classified
the exactly-dependent matrix `[[1,2],[2,4],[3,6]]` as rank-deficient on arm64 and full-rank on
amd64. Measuring the residual pivot explained why: it is `0.86 * eps * ||A||_F` on arm64 and
`1.07 * eps * ||A||_F` on amd64, so the round-off sits *at* one epsilon of the matrix norm and
the previous tolerance landed between the two values. The fix is a scale with real headroom
(`max(m,n) * eps * ||A||_F`), not a different magic number.

The general lesson is the one this file keeps returning to: when a comparison is decided by
round-off, its threshold has to be derived from the problem's size rather than chosen to make one
machine's tests pass. `GOARCH=amd64 go test ./...` is what caught it, and that run is not optional
for any change that introduces a numerical threshold.

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

### Test fixtures need their own test

`ta`'s synthetic OHLCV generator returns five slices in a fixed order:
`(open, high, low, close, volume)`. Several call sites destructured it as
`high, low, close, _, _`, so the "high" column was really the open and the "low"
column was really the high. **The bars had `high < low`, and every test that fed them
to both an implementation and its reference still passed**, because both sides saw the
same impossible input. It was found only when a trend indicator's invariants failed for
an unrelated reason.

Two changes came out of it, and both are the general lesson rather than a local fix:

1. **`synthHLC(n, seed)`** returns just the three columns a price indicator needs. A
   helper that cannot be destructured wrongly removes the class of bug instead of
   relying on callers to count return values.
2. **`TestSyntheticDataIsValidOHLC`** asserts the fixture's own invariant —
   `high >= open, close >= low`, and positive volume. A fixture that produces
   impossible data weakens every test using it *without failing any of them*, so the
   only way to hold it to its contract is to test it directly.

The same reasoning applies to any shared test helper: if it can be wrong in a way that
makes tests pass, it needs a test of its own.

---

## Further reading

| document | purpose |
|---|---|
| [learn/](learn/) | Why the loops are shaped this way, from first principles |
| [benchmarks.md](benchmarks.md) | Measured results and methodology |
| [next-steps.md](next-steps.md) | Open decisions and unverified assumptions |
