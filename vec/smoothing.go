package vec

import "github.com/rj-raghavjoshi/numa/internal/comp"

// ---------------------------------------------------------------------------
// Exponential smoothers over a whole slice.
//
// EMA and RMA are the same recurrence with a different smoothing factor:
//
//	y[i] = alpha*x[i] + (1-alpha)*y[i-1]
//
//   - EMA(n): alpha = 2/(n+1), the standard exponential moving average.
//   - RMA(n): alpha = 1/n, Wilder's smoothing, behind ATR, RSI and ADX.
//
// These are the streaming `series` smoothers written as a loop over a slice, for the same reason
// [RollingSum] exists: a caller holding the whole series does not need the incremental machinery.
//
// # The seeding convention is the whole contract
//
// The recurrence needs a previous value, and the seed decides every later output. Both seeds are the
// SMA of the first n values -- what TradingView's `ta.ema` and `ta.rma` do -- with the same Neumaier
// compensation the streaming version uses, so a caller can move between the two forms and get the
// same series. Seeding with the first value instead, as a textbook EMA often does, converges to the
// same place and disagrees for a long time on the way there.
//
// # NaN
//
// A NaN during the seeding phase poisons the seed and therefore every later output; a NaN after
// seeding is absorbed into the recurrence and propagates from there. Both fall out of the arithmetic
// rather than from a check, and both match the streaming roller exactly.
// ---------------------------------------------------------------------------

// RollingEMA returns the exponential moving average of xs with alpha = 2/(n+1), seeded with the
// simple average of the first n values.
//
// The first n-1 values are NaN. It panics if n < 1.
func RollingEMA(xs []float64, n int) []float64 {
	if len(xs) == 0 {
		return nil
	}
	return RollingEMATo(make([]float64, len(xs)), xs, n)
}

// RollingEMATo stores the exponential moving average into dst and returns dst.
//
// dst and xs must be the same length, and RollingEMATo panics otherwise, or if n < 1. dst must not
// alias xs: the seed pass reads xs[0:n] after earlier iterations have written dst.
func RollingEMATo(dst, xs []float64, n int) []float64 {
	checkRolling("RollingEMATo", dst, xs, n)
	return rollingSmoother(dst, xs, n, 2.0/float64(n+1))
}

// RollingRMA returns Wilder's smoothing of xs with alpha = 1/n, seeded with the simple average of
// the first n values.
//
// The first n-1 values are NaN. It panics if n < 1.
func RollingRMA(xs []float64, n int) []float64 {
	if len(xs) == 0 {
		return nil
	}
	return RollingRMATo(make([]float64, len(xs)), xs, n)
}

// RollingRMATo stores Wilder's smoothing into dst and returns dst.
//
// dst and xs must be the same length, and RollingRMATo panics otherwise, or if n < 1. dst must not
// alias xs; see [RollingEMATo].
func RollingRMATo(dst, xs []float64, n int) []float64 {
	checkRolling("RollingRMATo", dst, xs, n)
	return rollingSmoother(dst, xs, n, 1.0/float64(n))
}

// rollingSmoother is the shared recurrence behind both smoothers.
//
// The seed is accumulated with Neumaier compensation, matching the streaming roller: a long window
// over a series with a large offset is exactly where an uncompensated seed would lose its low-order
// bits, and a seed error decays only exponentially rather than disappearing.
func rollingSmoother(dst, xs []float64, n int, alpha float64) []float64 {
	size := len(dst)
	start := n - 1
	fillNaN(dst, 0, min(start, size))
	if size < n {
		return dst
	}

	var total, compensation float64
	for i := 0; i < n; i++ {
		total, compensation = comp.NeumaierAdd(total, compensation, xs[i])
	}
	prev := (total + compensation) / float64(n)
	dst[start] = prev

	oneMinusAlpha := 1 - alpha
	for i := n; i < size; i++ {
		prev = alpha*xs[i] + oneMinusAlpha*prev
		dst[i] = prev
	}
	return dst
}
