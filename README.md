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
| [docs/coverage.md](docs/coverage.md) | check what is missing. An audit against TradingView's published indicator list, with the entries that are deliberately not implemented and why. |
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
| `series` | `github.com/rj-raghavjoshi/numa/series` | you are computing incrementally, one value at a time, and want O(1) per element instead of recomputing a window. |
| `ta` | `github.com/rj-raghavjoshi/numa/ta` | you want an indicator by name: `ta.ATR(high, low, close, 14)`. Also re-exported from `numa`. |

`numa` is a thin facade: every function in it is a one-line forwarder to `vec`, so
there is no behaviour defined in `numa` that is not defined in `vec`. Go inlines
most of the forwarders, but not guaranteed-ly, which is the only reason to prefer
`vec` directly.

`series` is **not** re-exported through `numa`, because its API is types rather than
function kernels; import it directly. See [docs/design.md](docs/design.md).

```
numa/
├── numa.go          package docs and the original core forwarders
├── facade_*.go      one forwarder file per family (vec kernels, ta indicators)
└── vec/             the implementation
    ├── sum.go       Sum, Mean
    ├── dot.go       Dot
    ├── min.go       Min, Max, MinMax
    ├── add.go       Add, AddTo, ...
    ├── math.go      elementwise math maps (untagged — no dependency chain)
    ├── mask.go      predicates, mask algebra, Where/Compress, masked reductions
    ├── cum.go       cumulative scans (CumSum/Prod/Max/Min)
    ├── signal.go    lag, difference, cross, BarsSince, ValueWhen
    ├── arg.go       ArgMin/ArgMax/ArgMinMax, HasNaN
    ├── order.go     Median/Quantile/Percentile/Rank/MAD (quickselect)
    ├── stats.go     Variance/Covariance/Correlation/Skewness/Kurtosis (two-pass)
    ├── rolling.go   rolling windows (sum/mean/extremes/stddev)
    ├── orderstat.go rolling order statistics via a sliding Fenwick tree
    ├── smoothing.go rolling exponential smoothers (EMA, Wilder's RMA)
    ├── arch_*.go    per-architecture tuning (build tags)
    └── ...

series/              streaming rollers, imported directly
    ├── ring.go      fixed-capacity circular buffer
    ├── roller.go    the Roller interface, Apply / ApplyTo
    ├── sum.go       Sum, SMA (Neumaier-compensated rolling sum)
    └── ema.go       EMA, RMA (SMA-seeded smoothers)

ta/                  indicators, forwarded through numa
    ├── price.go     Median/Typical/AveragePrice, TrueRange, Change
    ├── moving.go    SMA/EMA/RMA/WMA/DEMA/TEMA/TRIMA/HMA
    ├── extrema.go   Highest/Lowest (monotonic deque), Donchian
    ├── volatility.go ATR, BollingerBands, %B, width
    ├── momentum.go  RSI, MACD, Stochastic, StochRSI, %R, CCI, CMO, ROC
    ├── volume.go    VWAP, VWMA, OBV, A/D, CMF, Chaikin, MFI, PVT, Force Index
    ├── trend.go     ADX/DMI, Aroon, Vortex, Choppiness, Keltner, SuperTrend, PSAR
    ├── regression.go LinReg + slope/intercept/R2/standard error/bands
    ├── oscillator.go Awesome, Accelerator, DPO, TRIX, UO, Fisher, Mass Index
    ├── correlation.go Pearson/log/Spearman, historical and Garman-Klass volatility
    ├── misc.go      Coppock, KST, TSI, Alligator, Fractal
    ├── ichimoku.go  the five Ichimoku lines with displacement
    ├── pivot.go     Classic/Fibonacci/Camarilla/Woodie pivot levels
    ├── gmma.go      Guppy multiple MAs, Zig Zag pivots
    ├── adaptive.go  ALMA, SWMA, KAMA, VIDYA, ZLEMA, T3, McGinley Dynamic
    ├── envelope.go  MA envelope, MA channel, price oscillator
    ├── momentum2.go SMI, RVI, Connors RSI, Elder Ray, percent rank
    ├── volume2.go   Balance of Power, Ease of Movement, Klinger, Chaikin volatility
    ├── stop.go      Chande Kroll stop
    ├── swing.go     Wilder's Swing Index and its accumulation
    └── catalogue.go spreads, crossings, vigour, chop, Hamming MA
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

All `*To` functions permit `dst` to alias the inputs, with one documented
exception: `ReverseTo` requires a separate destination, because the in-place case
is ambiguous. Use `ReverseInPlace(xs)` for that.

**Elementwise math** — the same two forms, for the operations a numerical engine
needs on raw slices:

| allocating | in place | computes |
|---|---|---|
| `Neg(xs)` | `NegTo(dst, xs)` | `-xs[i]` |
| `Recip(xs)` | `RecipTo(dst, xs)` | `1 / xs[i]` |
| `Sign(xs)` | `SignTo(dst, xs)` | `-1`, `0` or `+1` (sign of zero preserved) |
| `Sqrt`/`Exp`/`ExpM1`/`Log`/`Log1p`/`Log2`/`Log10` | `…To(dst, xs)` | the named function |
| `Floor`/`Ceil`/`Trunc`/`Round` | `…To(dst, xs)` | the named rounding |
| `Pow(xs, k)` | `PowTo(dst, xs, k)` | `xs[i] ** k` |
| `Div(xs, ys)` | `DivTo(dst, xs, ys)` | `xs[i] / ys[i]` |
| `SafeDiv(xs, ys)` | `SafeDivTo(dst, xs, ys)` | as `Div`, but a zero denominator gives `0` |
| `Min2(xs, ys)` | `Min2To(dst, xs, ys)` | elementwise minimum, **NaN propagates** |
| `Max2(xs, ys)` | `Max2To(dst, xs, ys)` | elementwise maximum, **NaN propagates** |
| `Clamp(xs, lo, hi)` | `ClampTo(dst, xs, lo, hi)` | `xs[i]` confined to `[lo, hi]` |
| `Lerp(xs, ys, t)` | `LerpTo(dst, xs, ys, t)` | `xs[i] + t*(ys[i]-xs[i])` |
| `MulAdd(xs, ys, zs)` | `MulAddTo(dst, xs, ys, zs)` | `xs[i]*ys[i] + zs[i]` |
| `FMA(xs, ys, zs)` | `FMATo(dst, xs, ys, zs)` | as `MulAdd`, **guaranteed one rounding** |
| `Copy(xs)`, `Fill(v, n)`, `Reverse(xs)` | `CopyTo`, `FillTo`, `ReverseTo` | the named move |

`Min2`/`Max2` propagate NaN deliberately, which is the opposite of the standard
library's `math.Min`/`math.Max`; see the package documentation for why.

`MulAdd` and `FMA` differ in exactly one property. `MulAdd` leaves the rounding
count to the compiler, which contracts it into a fused multiply-add on arm64 and on
FMA3 x86-64; `FMA` guarantees the single rounding on every target. Use `FMA` when
the result is audited, `MulAdd` otherwise.

**Predicates, masks and masked reductions** — a mask is a `[]uint8` of 0/1, so it
can be counted, summed and reused as a buffer instead of being re-derived:

| function | returns |
|---|---|
| `Greater(xs, ys)`, `Less`, `GreaterEqual`, `LessEqual`, `Equal`, `NotEqual` | `[]uint8` mask |
| `GreaterScalar(xs, k)`, `LessScalar`, `GreaterEqualScalar`, `LessEqualScalar`, `EqualScalar`, `NotEqualScalar` | mask against a constant threshold |
| `IsNaN(xs)`, `IsFinite(xs)`, `IsInf(xs, sign)` | value-class mask |
| `And(a, b)`, `Or`, `Xor`, `Not(a)` | mask algebra (non-zero means true) |
| `Where(cond, a, b)` | elementwise select |
| `Compress(xs, mask)` | gather the selected elements, in order |
| `CountTrue`, `AnyTrue`, `AllTrue` | mask reductions |
| `MaskedSum`, `MaskedMean`, `MaskedMin`, `MaskedMax` | reductions over selected elements |

Every one has a `*To` form. Predicates are plain IEEE comparisons, so a NaN operand
makes every ordered comparison false and `NotEqual` true — a mask can never claim an
ordering the hardware does not support. Ordered predicates are **1.93×** faster than
a plain loop, the largest map speedup in the package; `MaskedSum` is branch-bound at
11.2 GB/s and that limitation is documented rather than hidden. See
[docs/benchmarks.md](docs/benchmarks.md).

**Cumulative (prefix) operations** — one output per input, each depending on
everything up to it:

| allocating | in place | computes |
|---|---|---|
| `CumSum(xs)` | `CumSumTo(dst, xs)` | inclusive running sum |
| `CumProd(xs)` | `CumProdTo(dst, xs)` | inclusive running product |
| `CumMax(xs)` | `CumMaxTo(dst, xs)` | running maximum, NaN wins |
| `CumMin(xs)` | `CumMinTo(dst, xs)` | running minimum, NaN wins |
| `CumCountTrue(mask)` | `CumCountTrueTo(dst, mask)` | running count of non-zero mask entries |

A prefix scan is more latency-bound than a plain reduction, and the tuned `CumSum` is
**3.22×** faster than the naive prefix loop (41.0 vs 12.7 GB/s). `CumMax` is
deliberately *not* tuned — measured at exactly break-even, because blocking cannot
remove a per-element compare. Both numbers are in
[docs/benchmarks.md](docs/benchmarks.md).

**Lag and signal primitives** — comparing an element with one at a fixed offset:

| function | computes |
|---|---|
| `Shift(xs, lag)` | displaced copy; positive lag = past, negative = future, out of range = NaN |
| `Diff(xs, lag)` | `xs[i] - xs[i-lag]` |
| `Rate(xs, n)` | `(xs[i]-xs[i-n]) / xs[i-n]`, zero base gives 0 |
| `Rising(xs, n)`, `Falling(xs, n)` | mask for `xs[i] > xs[i-n]` / `<` |
| `CrossOver(a, b)`, `CrossUnder(a, b)` | mask for a crossing above / below b |
| `Cross(a, b)` | `+1` / `-1` / `0` signed crossing |
| `BarsSince(mask)` | bars since the last non-zero, `-1` before the first |
| `ValueWhen(cond, xs)` | forward-fill of the last selected value |
| `HighestSince(cond, xs)`, `LowestSince(cond, xs)` | running extreme since the last reset |

`Shift`, `Diff` and `Rate` **do not permit `dst` to alias the input** — unlike the
rest of the package. The reason is in [docs/design.md](docs/design.md): the loop reads
an element that an earlier iteration already overwrote.

`Shift` reaches **2.28×** by calling the built-in `copy` rather than any hand-written
loop (57.4 vs 25.2 GB/s), while `Diff` reaches 1.49× from the unroll. Getting that
number right required correcting a benchmark that was accidentally measuring a
compiler special case; the whole story is in
[docs/benchmarks.md](docs/benchmarks.md).

**Order statistics and arg reductions**

| function | returns |
|---|---|
| `ArgMin(xs)`, `ArgMax(xs)` | index of the extreme, `-1` if empty or if NaN is present |
| `ArgMinMax(xs)` | both indices in one pass |
| `HasNaN(xs)` | whether any element is NaN — needed to disambiguate the two meanings of `-1` |
| `Median(xs)` | median |
| `Quantile(xs, q)` | q-quantile for q in `[0,1]`, linear interpolation |
| `Percentile(xs, p)` | p-th percentile for p in `[0,100]` |
| `MedianInto`, `QuantileInto`, `PercentileInto` | the same, into a caller-supplied scratch buffer (no allocation) |
| `Rank(xs)` | ascending average ranks; ties share, NaN receives NaN |
| `MAD(xs)` | median absolute deviation |
| `SortInPlace(xs)`, `SortCopy(xs)` | ascending sort |

`Median`/`Quantile` use quickselect rather than sorting, for **3.1–5.1×** over the
copy-and-sort approach at 1K–32K elements. `ArgMin` was initially **0.51×** — *half*
the speed of a plain loop — because the NaN check is a second compare on a chain that
had nothing to hide it behind; splitting into independent partial scans restored
parity while keeping the NaN policy. Both results, and the tie-breaking subtlety that
independent chains introduce, are in [docs/benchmarks.md](docs/benchmarks.md).

**Descriptive statistics**

| function | returns |
|---|---|
| `Variance(xs)`, `VarianceSample(xs)` | population / unbiased sample variance |
| `StdDev(xs)`, `StdDevSample(xs)` | their square roots |
| `Moment(xs, k)` | k-th central moment |
| `Covariance(xs, ys)`, `CovarianceSample(xs, ys)` | population / sample covariance |
| `Correlation(xs, ys)` | Pearson correlation coefficient |
| `Skewness(xs)`, `Kurtosis(xs)` | third moment and excess fourth moment |
| `MeanAbsDeviation(xs)` | mean absolute deviation from the mean |
| `RMS(xs)`, `GeoMean(xs)`, `HarmMean(xs)` | other means |

Empty inputs and undefined quantities return **NaN**, deliberately unlike `Sum` and
`Mean`, which return 0 for an empty slice — a variance of 0 would assert "no spread"
about an unknown quantity. These functions centre the data before accumulating, so
they read the input twice. That makes them accurate **and**, measured, *faster* than
the one-pass `mean(x²)−mean(x)²` form: **1.91× faster** at n=4M, while the one-pass
result was wrong by a factor of 391 on data built to expose it. See
[docs/benchmarks.md](docs/benchmarks.md).

## Streaming (`series`)

For incremental work, `series` rollers carry state and cost O(1) per element:

```go
import "github.com/rj-raghavjoshi/numa/series"

s := series.NewSMA(20)
for _, v := range closes {
    out = append(out, s.Push(v)) // NaN until 20 values have arrived
}
```

| roller | computes | warm-up |
|---|---|---|
| `series.Sum(n)` | rolling sum, Neumaier-compensated | n |
| `series.SMA(n)` | simple moving average | n |
| `series.EMA(n)` | exponential moving average (α = 2/(n+1)), SMA-seeded | n |
| `series.RMA(n)` | Wilder's smoothing (α = 1/n), SMA-seeded | n |

Every roller implements `series.Roller` (`Warmup`, `Push`, `Reset`), and `Reset`
reuses the buffer, so one roller can process many series with zero allocations after
construction. `series.ApplyTo` and `series.Apply` drive a roller over a whole slice.

Streaming is **not** universally faster than recomputing: at window 20 the roller
*loses* to a naive window loop by 2×, while by window 200 it wins 9.3× and its cost is
flat in the window. Neumaier compensation doubles the rolling-sum cost (19.2 vs 9.3
ns/element) and is what prevents a silently wrong sum after a large-offset eviction.
All of it, including two corrections to earlier claims, is in
[docs/benchmarks.md](docs/benchmarks.md).

## Indicators (`ta`)

**100 of the 107 entries** in TradingView's published indicator list are implemented; the seven that
are not are named with their reasons in [docs/coverage.md](docs/coverage.md).

```go
import "github.com/rj-raghavjoshi/numa/ta"

atr := ta.ATR(high, low, close, 14)          // NaN for the first 13 bars
up, mid, lo := ta.BollingerBands(close, 20, 2)
```

Indicators take plain slices and return full-length slices, with NaN during warm-up;
`ta.FirstValid` reports where a series becomes usable. [ta/doc.go](ta/doc.go) states
the six conventions every function follows.

| category | functions |
|---|---|
| price transforms | `MedianPrice`, `TypicalPrice`, `AveragePrice`, `TrueRange`, `Change` |
| moving averages | `SMA`, `EMA`, `RMA`, `WMA`, `DEMA`, `TEMA`, `TRIMA`, `HMA` |
| extremes & channels | `Highest`, `Lowest`, `Donchian` |
| volatility | `ATR`, `BollingerBands`, `BBPercentB`, `BBWidth` |
| momentum | `RSI`, `MACD`, `Stochastic`, `StochRSI`, `WilliamsPercentR`, `CCI`, `CMO`, `ROC`, `Momentum` |
| volume | `VWAP`, `VWMA`, `OBV`, `AccumulationDistribution`, `ChaikinMoneyFlow`, `ChaikinOscillator`, `MoneyFlowIndex`, `PriceVolumeTrend`, `NetVolume`, `VolumeOscillator`, `ForceIndex` |
| trend | `DirectionalMovement` (ADX/+DI/-DI), `Aroon`, `Vortex`, `Choppiness`, `KeltnerChannels`, `SuperTrend`, `ParabolicSAR` |
| regression | `LinearRegression`, `LinearRegressionSlope`, `LinearRegressionIntercept`, `R2`, `StandardError`, `StandardErrorBands` |
| oscillators | `AwesomeOscillator`, `AcceleratorOscillator`, `DetrendedPriceOscillator`, `TRIX`, `UltimateOscillator`, `FisherTransform`, `MassIndex` |
| statistics | `CorrelationCoefficient`, `CorrelationLog`, `RankCorrelation`, `HistoricalVolatility`, `VolatilityOHLC` |
| patterns | `CoppockCurve`, `KnowSureThing`, `TrueStrengthIndex`, `WilliamsAlligator`, `WilliamsFractal` |
| structure | `Ichimoku`, `PivotLevels`/`PivotPoints` (Classic/Fibonacci/Camarilla/Woodie), `GMMA`, `ZigZag` |
| adaptive MAs | `ALMA`, `SWMA`, `KAMA`, `VIDYA`, `ZLEMA`, `T3`, `McGinleyDynamic` |
| envelopes | `MovingAverageEnvelope`, `MovingAverageChannel`, `PriceOscillator` |
| momentum II | `PercentRank`, `StochasticMomentumIndex`, `RelativeVolatilityIndex`, `ConnorsRSI`, `ElderRay` |
| volume II | `BalanceOfPower`, `EaseOfMovement`, `KlingerOscillator`, `ChaikinVolatility` |
| stops | `ChandeKrollStop`, `FiftyTwoWeekHighLow` |
| swing | `SwingIndex`, `AccumulativeSwingIndex` |
| spreads & cross | `Ratio`, `Spread`, `StandardDeviation`, `MACross`, `EMACross`, `EMASMACross` |
| other | `RelativeVigorIndex`, `ChopZone`, `HammingMA`, `SMIErgodic`, `AdvanceDecline` |
| utility | `FirstValid` |

**Rolling windows** (`vec`) — one output per input, summarising the trailing `n`:

| allocating | in place | computes |
|---|---|---|
| `RollingSum(xs, n)` | `RollingSumTo` | trailing sum, Neumaier-compensated |
| `RollingMean(xs, n)` | `RollingMeanTo` | trailing mean |
| `RollingMax` / `RollingMin` | `…To` | trailing extreme, monotonic deque |
| `RollingRange` | `RollingRangeTo` | trailing high-low range |
| `RollingWMA` | `RollingWMATo` | trailing weighted average |
| `RollingVariance`, `RollingVarianceSample` | — | population / sample variance |
| `RollingStdDev`, `RollingStdDevSample` | — | their square roots |
| `RollingCovariance`, `RollingCovarianceSample`, `RollingCorrelation` | — | two-series windows |
| `RollingMedian`, `RollingQuantile(xs, n, q)` | — | order statistics (sliding Fenwick tree) |
| `RollingPercentRank(xs, n)` | — | percentage of the window below the current value |
| `RollingEMA(xs, n)` | `RollingEMATo` | exponential average, alpha = 2/(n+1) |
| `RollingRMA(xs, n)` | `RollingRMATo` | Wilder's smoothing, alpha = 1/n |

The sum, the extremes and the order statistics all carry running state, so their cost is **flat in
the window**: at window 200 the deque beats a naive rescan by **58.6×**, the compensated sum by
**49.5×**, and the median by **25×** against the window rescan it replaced. The variance kernels
recompute their window and therefore scale with it — deliberately, because for a variance the
incremental form is the `mean(x²) − mean(x)²` identity this package rejects as numerically unstable.
Both shapes are measured in [docs/benchmarks.md](docs/benchmarks.md).

Unlike the elementwise `*To` forms, the rolling `*To` forms **do not permit `dst` to alias `xs`**;
see [docs/design.md](docs/design.md).

Indicators that receive a whole slice use these batch kernels rather than the `series` rollers:
`SMA` is `vec.RollingMean`, `EMA`/`RMA` are `vec.RollingEMA`/`RollingRMA`, and
`Highest`/`Lowest`/`WMA` delegate the same way. The gains were **4×** for `SMA`, about **2×** for the
EMA/RMA family (`MACD` 2.03×, `ATR` 1.90×), and **nothing at all** for `UltimateOscillator` — a null
result that turned out to be a real property of the streaming roller rather than a measurement
artefact. [docs/benchmarks.md](docs/benchmarks.md) records all three, and why they differ.

### `mat` — dense linear algebra

Imported directly (`mat.Solve`, not `numa.Solve`), because it defines a type; see
[docs/design.md](docs/design.md) for the facade rule.

| operation | functions |
|---|---|
| type | `Mat` (dense row-major), `New`, `FromRows`, `Identity`, `Clone`, `At`, `Set`, `Row` |
| products | `Mul`/`MulTo`, `MulVec`/`MulVecTo`, `Dot`, `Axpy`/`AxpyTo` |
| elementwise | `Add`, `Sub`, `Scale`, `Copy`, `Fill`, `Transpose` |
| norms | `Norm2`, `NormInf`, `Trace`, `IsSymmetric` |
| solves | `Solve` (LU, partial pivoting), `Inverse` |
| SPD | `Cholesky`, `CholeskySolve` |
| least squares | `QR`, `LeastSquares` |
| eigen | `EigenSym` (cyclic Jacobi) |

Numerical failures are **returned** (`ErrSingular`, `ErrNotSPD`) rather than panicked, because they
depend on the data; shape mismatches panic, because they depend on the call.

Measured highlights, all in [docs/benchmarks.md](docs/benchmarks.md): `LeastSquares` applies the
Householder reflectors to the right-hand side instead of forming Q, which is **650× faster** on a
10000×20 regression than forming it; `Mul` runs at **7.37 GFLOPS**, which is **77% of the machine's
scalar fused-multiply-add ceiling** and **5.87×** the naïve loop order; and Cholesky is **2.97×**
faster than LU on the same system, against an arithmetic ratio of 2×.

All of these are also on the `numa` facade. The catalogue is being filled in wave by
wave.

Two measured results worth knowing: `Highest` uses a monotonic deque and is **flat in
the window**, beating a window rescan by **52.7×** at window 200; `WMA` is deliberately
O(window) and therefore **19.8× slower** at window 200 than at window 20. Both are in
[docs/benchmarks.md](docs/benchmarks.md).

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
25.4 GB/s. The remaining bottleneck is arithmetic latency, so SIMD has real
headroom. FMA does not: Go already contracts `a + x*y` into a hardware fused
multiply-add by default, which was established by disassembling this package's own
`Dot` rather than by reading it. See [docs/benchmarks.md](docs/benchmarks.md) and
[docs/next-steps.md](docs/next-steps.md) §7.2.

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
