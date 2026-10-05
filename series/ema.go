package series

import "math"

// ---------------------------------------------------------------------------
// Exponential smoothers: EMA and Wilder's RMA.
//
// Both are the same recurrence with a different smoothing factor:
//
//	y[i] = alpha*x[i] + (1 - alpha)*y[i-1]
//
//   - EMA(n):  alpha = 2/(n+1), the standard exponential moving average.
//   - RMA(n):  alpha = 1/n, Wilder's smoothing, used by ATR, RSI, ADX and the
//     rest of Wilder's family.
//
// # Seeding follows the reference implementation, not the textbook
//
// The recurrence needs a previous value, and the choice of seed decides every later
// output. This implementation seeds with the SMA of the first n values and reports
// NaN before that, which is what TradingView's `ta.ema` and `ta.rma` do. That matters
// because an indicator's value on a given bar is only reproducible if the seed
// convention is; seeding with the first value instead, as a textbook EMA often does,
// produces a series that converges to the same place but disagrees for a long time
// on the way there.
//
// # The seed sum is compensated too
//
// The seed is a plain sum of n values, and it is computed with the same Neumaier
// accumulation as [Sum]. A long window over a series with a large offset is exactly
// where an uncompensated seed would lose the low-order bits, and the seed error would
// then decay only exponentially rather than disappearing.
//
// # NaN
//
// A NaN during the seeding phase poisons the seed, and therefore every subsequent
// output. A NaN after seeding is absorbed into prev and propagates from there. Both
// follow from the recurrence rather than from an explicit check.
// ---------------------------------------------------------------------------

// recur is the shared state of the exponential smoothers.
type recur struct {
	window int
	alpha  float64

	seedTotal float64
	seedComp  float64
	count     int
	prev      float64
	seeded    bool
}

func newRecur(window int, alpha float64) recur {
	return recur{window: window, alpha: alpha}
}

// Warmup returns the window length.
func (r *recur) Warmup() int { return r.window }

// Window returns the window length.
func (r *recur) Window() int { return r.window }

// Reset returns the smoother to its post-construction state.
func (r *recur) Reset() {
	r.seedTotal = 0
	r.seedComp = 0
	r.count = 0
	r.prev = 0
	r.seeded = false
}

// Push consumes one value and returns the current smoothed value, or NaN until the
// window is full.
func (r *recur) Push(v float64) float64 {
	if !r.seeded {
		r.seedTotal, r.seedComp = neumaierAdd(r.seedTotal, r.seedComp, v)
		r.count++
		if r.count < r.window {
			return math.NaN()
		}
		r.prev = (r.seedTotal + r.seedComp) / float64(r.window)
		r.seeded = true
		return r.prev
	}
	r.prev = r.alpha*v + (1-r.alpha)*r.prev
	return r.prev
}

// EMA is an exponential moving average with alpha = 2/(n+1), seeded with the SMA of
// the first n values.
//
// Construct with [NewEMA]. The zero value is not usable.
type EMA struct{ recur }

// NewEMA returns an exponential moving average over a window of window values.
//
// It panics if window < 1.
func NewEMA(window int) *EMA {
	if window < 1 {
		panic("series: EMA window must be >= 1")
	}
	return &EMA{newRecur(window, 2.0/float64(window+1))}
}

// RMA is Wilder's smoothing, alpha = 1/n, seeded with the SMA of the first n values.
//
// It is the smoother behind ATR, RSI and ADX. Construct with [NewRMA]; the zero
// value is not usable.
type RMA struct{ recur }

// NewRMA returns a Wilder smoothing over a window of window values.
//
// It panics if window < 1.
func NewRMA(window int) *RMA {
	if window < 1 {
		panic("series: RMA window must be >= 1")
	}
	return &RMA{newRecur(window, 1.0/float64(window))}
}
