# Backtesting Engine — What It Must Add (Prioritized)

> **Scope.** This is a requirements checklist for the *downstream* backtesting
> engine, which lives in a separate module and consumes `numa`. It is not part of
> numa's runtime and it proposes **no changes to numa**. It exists so the engine's
> first commit is scoped by what actually blocks a correct result, rather than by
> what is easiest to write.
>
> Companion to [next-steps.md](next-steps.md) (numa's own open items). For the
> capability comparison that motivates the P2/P3 entries, see the discussion of
> what numpy/pandas provide that numa does not.

## How to read the priorities

| Tier | Meaning | Rule |
|---|---|---|
| **P0** | Blocks a first *correct* single-instrument vectorized backtest | Do not ship a backtest without it |
| **P1** | Needed for results to be *credible and comparable* | Next milestone; without it numbers can be right but meaningless |
| **P2** | Needed to *scale or model richer* | Add when the strategy set demands it |
| **P3** | Defer, or legitimately never | Do not build speculatively |

The engine is `vec`/`ta`/`mat` consumers, not forks. Every P0 item below is
engine-owned; none of them is a numa gap that must be closed first.

---

## Reuse from numa (do not reimplement)

| Need | Use |
|---|---|
| Indicators/signals | `ta.*` — causal by contract, see [../ta/doc.go](../ta/doc.go) |
| Bar-to-bar return/change | `numa.Change`, `numa.Rate`, `numa.Diff`, `numa.Shift` |
| Crossings, streaks, forward-fill | `numa.Cross`/`CrossOver`/`CrossUnder`, `BarsSince`, `ValueWhen`, `Rising`/`Falling` |
| Running extremes for stops | `numa.CumMax`/`CumMin`, `HighestSince`/`LowestSince` |
| Drawdown | `1 - equity/CumMax(equity)` |
| Vol / Sharpe / Sortino inputs | `numa.StdDev`, `StdDevSample`, `Mean`, `Variance`, `MaskedMean`/`MaskedSum` |
| Rolling risk | `numa.RollingStdDev`, `RollingCorrelation`, `RollingMax`/`RollingMin` |
| Selection / ranking | `numa.Rank`, `ArgMin`/`ArgMax`, `SortInPlace`/`SortCopy`, `Median`/`Quantile` |
| Covariance / beta / risk models | `numa.Covariance`, `RollingCovariance`, `mat.*` (`Solve`, `QR`, `LeastSquares`, `Cholesky`, `EigenSym`) |
| Streaming (later, event-driven path) | `series.*` rollers |
| Warm-up location | `numa.FirstValid` |

**Do not push into numa:** engine types (bars, orders, positions, portfolios),
time/index handling, IO, RNG, or engine-specific metrics. numa's facade rule is
shape-based: packages that define *types* are imported directly, not forwarded.

---

## P0 — blocks a correct first backtest

| # | Must add | Why | Acceptance |
|---|---|---|---|
| 1 | **Bar/time model** — OHLCV slices + timestamps, with validation | numa has no time type at all | Lengths equal; timestamps strictly increasing; explicit error on gaps/duplicates |
| 2 | **Return convention** — one documented definition (simple vs log) | `numa.Rate` returns **0 on a zero base**, which silently distorts returns; the engine must own the rule | Documented rule for zero/negative base; tested at those boundaries |
| 3 | **Warm-up → position policy** | `ta` emits leading NaN; NaN must never reach P&L | One `validFrom`/warm-up helper; explicit flat/hold/skip choice; test that no NaN leaks into equity |
| 4 | **Execution simulator** — signal at bar `i` → fill at a declared bar and price; fees + slippage | No orders/fills/position/cash exists anywhere | Deterministic fills; cash + position accounting identity holds every bar |
| 5 | **Equity curve + core metrics** — total return, CAGR, max drawdown, vol, Sharpe, Sortino, trade count, exposure | Names don't exist in numa; only building blocks do | Metrics composed from numa primitives; annualization + risk-free rate fixed in one place |
| 6 | **Look-ahead guard + invariant tests** | numa is causal; nothing stops the *strategy* peeking | Test asserts `len(signal) == len(close)`; a canary "peek" strategy must be caught; shift test documents sensitivity |
| 7 | **Reproducibility contract** | numa is not bit-identical across architectures ([next-steps.md](next-steps.md) §6.2) | Runs record seed, numa version, and a stated tolerance or single-target constraint |

## P1 — makes results credible and comparable

| # | Must add | Why |
|---|---|---|
| 8 | **Input adapters** — CSV/columnar read, column mapping, sort/dedupe | numa has zero IO |
| 9 | **Panel/multi-asset alignment** — union timestamps, missing-bar policy | Universe-wide strategies are not expressible on 1-D slices |
| 10 | **Cross-sectional ops** — rank/z-score/top-k | `Rank` exists but there is **no `argsort`** returning a permutation, which top-k selection needs |
| 11 | **RNG with explicit seed** | numa has no RNG; required for sampling, bootstraps, synthetic tests |
| 12 | **Trade blotter analytics** — win rate, avg win/loss, profit factor, holding period | Turns an equity curve into explainable trades |
| 13 | **Risk exposure** — net/gross, rolling vol/beta, position limits | Needed before any leverage or short book is trusted |

## P2 — scale and richer modeling

| # | Must add | Why |
|---|---|---|
| 14 | **SVD/PCA** for factor risk | `mat` has `QR`/`LeastSquares`/`EigenSym` but **no SVD**; backfill via gonum or implement |
| 15 | **Resampling, sessions/calendars, corporate-action adjustment** | Splits/dividends make unadjusted backtests wrong |
| 16 | **Parameter sweeps / walk-forward with parallelism** | numa is single-threaded and not documented safe on a shared `dst`; concurrency is the engine's |
| 17 | **Bootstrap/permutation inference** | Confidence on Sharpe/trade stats; needs #11 |
| 18 | **Buffer reuse and throughput** | Use `*To` forms; benchmark short-slice paths — tuned kernels can lose at small `n` |

## P3 — defer or never

FFT/spectral analysis · plotting · categoricals/string ops · SQL/Excel IO ·
GPU · sparse matrices (unless the universe is large) · dedicated NaN-aware
reductions (compose from `IsNaN` + `Compress` + `Masked*`) · general
non-symmetric eigensolver · out-of-core/memory-mapped arrays.

---

## Cross-cutting engineering requirements

- **Layering.** Import `numa` (facade), `numa/series`, `numa/mat` directly. Never
  add engine types to numa.
- **Packaging.** numa currently has **no git tags** (`Version = "0.1.0"`). Tag a
  `v0.x` release or use a `replace` directive for local development before the
  engine depends on it.
- **Result schema.** Version the backtest output (params, seed, numa version,
  conventions) so runs are comparable and auditable.
- **Testing.** Golden-file tests for at least one known strategy, plus property
  tests for the accounting identity and for causality.

## Decisions to freeze before P0 code

1. **Fill/settlement convention** — e.g. signal at close `i` → fill at open `i+1`.
2. **Reproducibility** — per-machine tolerance vs single-target determinism.
3. **Metrics conventions** — annualization factor and risk-free rate source.
4. **First milestone scope** — single-instrument (recommended) vs universe-wide.

## Explicit non-goals

The engine is not a numpy/pandas replacement for general data work, and numa
should not grow engine features to accommodate it. Data loading, time handling,
execution accounting, and metrics conventions belong to the engine.
