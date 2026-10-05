package ta

import (
	"math"

	"github.com/rj-raghavjoshi/numa/vec"
)

// ---------------------------------------------------------------------------
// Moving averages.
//
// # Warm-up
//
// The simple and weighted averages warm up in n-1 bars: the first output is defined
// once a full window exists, at index n-1.
//
// The seeded exponential smoothers, EMA and RMA, also first produce a value at index
// n-1, but for a different reason: they consume the first n values to form their seed
// and emit that seed as the output at the last of those n positions. So the boundary
// coincides, while the value at it is an average of the window rather than a
// recursive update. The composites inherit the longest warm-up of their parts, which
// is where the two conventions stop coinciding.
//
// # Why SMA, EMA and RMA delegate to series
//
// Those three already exist as streaming rollers in the series package, with the
// seeding convention, the NaN policy and the tests. Reimplementing them in batch form
// here would create a second definition of the same mathematics that could drift out
// of step. The batch entry points simply drive the roller over the whole slice.
//
// WMA is different and is implemented here directly; see its doc comment.
// ---------------------------------------------------------------------------

// SMA returns the n-period simple moving average of xs.
//
// The first n-1 values are NaN. See [series.SMA] for the rolling implementation.
func SMA(xs []float64, n int) []float64 {
	checkPeriod("SMA", n)
	if len(xs) == 0 {
		return nil
	}
	// The batch kernel, not the streaming roller. Both compute the same thing -- a Neumaier
	// compensated running sum divided by the window -- and the batch form is measurably faster when
	// the whole slice is available: SMA went from 19.6 to 4.7 ns/element, which is the end-to-end
	// indicator measurement in docs/benchmarks.md rather than a micro-benchmark of the sum alone.
	//
	// That distinction matters. A standalone benchmark of the streaming roller against the batch
	// kernel reported roughly 20 against 4 ns/element, and then failed to predict what happened to
	// the *other* indicators that were switched over: see [UltimateOscillator], where replacing six
	// streaming sums with six batch ones changed the total by about five percent. A per-call cost
	// that does not compose should not be used to justify a change, so the evidence here is the
	// indicator's own measurement.
	return vec.RollingMean(xs, n)
}

// EMA returns the n-period exponential moving average of xs, alpha = 2/(n+1),
// seeded with the SMA of the first n values.
//
// The first n-1 values are NaN. The seeding convention is what makes the series
// reproducible against TradingView's ta.ema; see [series.EMA].
func EMA(xs []float64, n int) []float64 {
	checkPeriod("EMA", n)
	if len(xs) == 0 {
		return nil
	}
	return vec.RollingEMA(xs, n)
}

// RMA returns Wilder's smoothing of xs, alpha = 1/n, seeded with the SMA of the
// first n values.
//
// It is the smoother behind [ATR], [RSI] and ADX. The first n-1 values are NaN. See
// [series.RMA].
func RMA(xs []float64, n int) []float64 {
	checkPeriod("RMA", n)
	if len(xs) == 0 {
		return nil
	}
	return vec.RollingRMA(xs, n)
}

// WMA returns the n-period weighted moving average of xs, with weights 1..n from the
// oldest value to the newest.
//
// The first n-1 values are NaN.
//
// # Why this one is not a rolling recurrence
//
// A weighted average can be maintained in O(1) per element: the running numerator
// updates as `num += n*x[i] - previousWindowSum`. That form is not used here because
// it makes a NaN permanently sticky. Once a NaN enters the running numerator, every
// later output is NaN even after the NaN has left the window, because the recurrence
// can never subtract it back out cleanly.
//
// The direct window loop below propagates a NaN only while it is inside the window,
// which is the behaviour every other function in this package has. The cost is
// O(window) per element instead of O(1). Given the measured result in
// docs/benchmarks.md that recomputing a window beats a streaming recurrence below a
// window of roughly 100, that is the better trade for the window lengths a weighted
// average is usually asked for. The O(1) form with NaN recovery is recorded as a
// follow-up rather than implemented speculatively.
func WMA(xs []float64, n int) []float64 {
	checkPeriod("WMA", n)
	if len(xs) == 0 {
		return nil
	}
	return vec.RollingWMA(xs, n)
}

// DEMA returns the n-period double exponential moving average:
//
//	2*EMA(xs,n) - EMA(EMA(xs,n),n)
//
// It is designed to reduce the lag of a plain EMA. The inner EMA is NaN for its first
// n-1 values, and the outer smoother is started at the inner's first valid index
// rather than being fed those NaNs as seed values -- feeding them would poison the
// outer seed and make the whole output NaN. See [applyFrom].
//
// The warm-up is 2n-2 bars.
func DEMA(xs []float64, n int) []float64 {
	checkPeriod("DEMA", n)
	if len(xs) == 0 {
		return nil
	}
	e1 := EMA(xs, n)
	e2 := applyEMA(e1, FirstValid(e1), n)
	out := make([]float64, len(xs))
	for i := range xs {
		out[i] = 2*e1[i] - e2[i]
	}
	return out
}

// TEMA returns the n-period triple exponential moving average:
//
//	3*EMA - 3*EMA(EMA) + EMA(EMA(EMA))
//
// The warm-up is 3n-3 bars, and the same seeding consideration as [DEMA] applies at
// each level.
func TEMA(xs []float64, n int) []float64 {
	checkPeriod("TEMA", n)
	if len(xs) == 0 {
		return nil
	}
	e1 := EMA(xs, n)
	e2 := applyEMA(e1, FirstValid(e1), n)
	e3 := applyEMA(e2, FirstValid(e2), n)
	out := make([]float64, len(xs))
	for i := range xs {
		out[i] = 3*e1[i] - 3*e2[i] + e3[i]
	}
	return out
}

// TRIMA returns the triangular moving average of xs: an SMA of an SMA, with the two
// periods chosen so the result is symmetric.
//
//	len1 = ceil(n/2), len2 = floor(n/2) + 1, TRIMA = SMA(SMA(xs, len1), len2)
//
// This matches TradingView's `ta.trima`. The warm-up is len1+len2-2 bars.
//
// Unlike the exponential composites, no seeding correction is needed here: the
// rolling sum underneath [SMA] excludes NaN values rather than accumulating them, so
// the outer average starts producing values as soon as its window is free of the
// inner average's warm-up NaNs.
func TRIMA(xs []float64, n int) []float64 {
	checkPeriod("TRIMA", n)
	if len(xs) == 0 {
		return nil
	}
	len1 := (n + 1) / 2 // ceil(n/2)
	len2 := n/2 + 1     // floor(n/2) + 1
	return SMA(SMA(xs, len1), len2)
}

// HMA returns the Hull moving average of xs:
//
//	WMA(2*WMA(xs, n/2) - WMA(xs, n), round(sqrt(n)))
//
// It is designed to track price closely while staying smooth. n/2 is integer division
// and is floored at 1, so HMA(xs, 1) is defined. The warm-up is n + round(sqrt(n)) - 2
// bars.
func HMA(xs []float64, n int) []float64 {
	checkPeriod("HMA", n)
	if len(xs) == 0 {
		return nil
	}
	half := n / 2
	if half < 1 {
		half = 1
	}
	w1 := WMA(xs, half)
	w2 := WMA(xs, n)
	diff := make([]float64, len(xs))
	for i := range xs {
		diff[i] = 2*w1[i] - w2[i]
	}
	sq := int(math.Round(math.Sqrt(float64(n))))
	if sq < 1 {
		sq = 1
	}
	return WMA(diff, sq)
}
