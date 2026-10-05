package ta

import "github.com/rj-raghavjoshi/numa/vec"

// ---------------------------------------------------------------------------
// Rolling extremes and the channels built from them.
//
// The scans themselves live in vec (RollingMax and RollingMin), because a rolling window over a
// slice is a batch kernel and this package is for the indicators built on top of them. What is
// left here is the channel construction.
// ---------------------------------------------------------------------------

// Highest returns the rolling maximum of xs over a window of n values.
//
// The first n-1 values are NaN, and a window containing a NaN produces NaN.
//
// It delegates to [vec.RollingMax], which is the batch form of the same monotonic-deque scan
// the series rollers use; there is exactly one implementation of it in the engine.
func Highest(xs []float64, n int) []float64 {
	checkPeriod("Highest", n)
	if len(xs) == 0 {
		return nil
	}
	return vec.RollingMax(xs, n)
}

// Lowest returns the rolling minimum of xs over a window of n values.
//
// The first n-1 values are NaN, and a window containing a NaN produces NaN.
func Lowest(xs []float64, n int) []float64 {
	checkPeriod("Lowest", n)
	if len(xs) == 0 {
		return nil
	}
	return vec.RollingMin(xs, n)
}

// Donchian returns the n-period Donchian channel: the highest high, the midpoint of
// the channel, and the lowest low.
//
// It is the same construction TradingView lists as both "Donchian Channels" and
// "Price Channel". The first n-1 values of all three outputs are NaN.
func Donchian(high, low []float64, n int) (upper, middle, lower []float64) {
	checkPeriod("Donchian", n)
	size := requireSameLen("Donchian", high, low)
	if size == 0 {
		return nil, nil, nil
	}
	upper = Highest(high, n)
	lower = Lowest(low, n)
	middle = make([]float64, size)
	for i := range middle {
		middle[i] = (upper[i] + lower[i]) / 2
	}
	return upper, middle, lower
}

// FiftyTwoWeekHighLow returns the rolling extreme of the high and the low over bars periods.
//
// It is [Highest] and [Lowest] under the name the indicator is known by, and the period is a
// parameter rather than being fixed at 252 because the convention depends on the data: 252 is the
// number of trading days in a year for a daily equity series, but a weekly or intraday series has a
// different year and would be mislabelled by that constant.
//
// The values are the extremes *reached*, not closes, so a series that spiked and fell back still
// shows the spike. Both outputs are NaN until the window is full.
func FiftyTwoWeekHighLow(high, low []float64, bars int) (highs, lows []float64) {
	checkPeriod("FiftyTwoWeekHighLow", bars)
	size := requireSameLen("FiftyTwoWeekHighLow", high, low)
	if size == 0 {
		return nil, nil
	}
	return Highest(high, bars), Lowest(low, bars)
}
