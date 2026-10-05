package ta

import (
	"math"

	"github.com/rj-raghavjoshi/numa/vec"
)

// ---------------------------------------------------------------------------
// Price transforms.
//
// These collapse the four price columns into one series, which most indicators then
// consume. They are ordinary elementwise maps with no warm-up: every output position
// is defined as long as the inputs are.
// ---------------------------------------------------------------------------

// MedianPrice returns (high + low) / 2.
//
// It is the midpoint of the bar range, and the price series the Awesome Oscillator
// and Williams Alligator are defined on.
func MedianPrice(high, low []float64) []float64 {
	n := requireSameLen("MedianPrice", high, low)
	if n == 0 {
		return nil
	}
	out := make([]float64, n)
	for i := 0; i < n; i++ {
		out[i] = (high[i] + low[i]) / 2
	}
	return out
}

// TypicalPrice returns (high + low + close) / 3.
//
// It is the price series behind the Commodity Channel Index, Money Flow Index and
// VWAP.
func TypicalPrice(high, low, close []float64) []float64 {
	n := requireSameLen("TypicalPrice", high, low, close)
	if n == 0 {
		return nil
	}
	out := make([]float64, n)
	for i := 0; i < n; i++ {
		out[i] = (high[i] + low[i] + close[i]) / 3
	}
	return out
}

// AveragePrice returns (open + high + low + close) / 4.
//
// TradingView calls this "Average Price" (OHLC/4). Note that some other platforms use
// the same name for the median price, which is why this doc comment spells out the
// formula rather than the name.
func AveragePrice(open, high, low, close []float64) []float64 {
	n := requireSameLen("AveragePrice", open, high, low, close)
	if n == 0 {
		return nil
	}
	out := make([]float64, n)
	for i := 0; i < n; i++ {
		out[i] = (open[i] + high[i] + low[i] + close[i]) / 4
	}
	return out
}

// TrueRange returns Wilder's true range:
//
//	max(high-low, |high-prevClose|, |low-prevClose|)
//
// The first bar has no previous close, so its true range is taken as high-low. That
// is the usual convention and it keeps the output fully defined rather than starting
// with a NaN; the value is documented here because it is a choice, not a derivation.
//
// TrueRange never has a warm-up. Its smoothed form, [ATR], does.
func TrueRange(high, low, close []float64) []float64 {
	n := requireSameLen("TrueRange", high, low, close)
	if n == 0 {
		return nil
	}
	out := make([]float64, n)
	out[0] = high[0] - low[0]
	for i := 1; i < n; i++ {
		hl := high[i] - low[i]
		hc := math.Abs(high[i] - close[i-1])
		lc := math.Abs(low[i] - close[i-1])
		m := hl
		if hc > m {
			m = hc
		}
		if lc > m {
			m = lc
		}
		out[i] = m
	}
	return out
}

// Change returns xs[i] - xs[i-n], the n-bar change.
//
// The first n positions are NaN. n = 0 returns zeros -- the change over no bars is
// exactly zero -- and a negative n is a forward difference, matching [vec.Diff], which
// this delegates to.
func Change(xs []float64, n int) []float64 {
	if len(xs) == 0 {
		return nil
	}
	return vec.Diff(xs, n)
}

// WeightedClose returns the weighted close price, (high + low + 2*close) / 4.
//
// It is the one price transform here that weights rather than averages: the close is given half
// the weight and the two extremes share the other half, on the view that where a bar ended matters
// more than how far it reached. Compare [TypicalPrice], which weights all three equally, and
// [MedianPrice], which ignores the close entirely.
//
// The result is always inside the bar's range, since it is a convex combination of values that are.
// All three inputs must have the same length; WeightedClose panics otherwise.
func WeightedClose(high, low, close []float64) []float64 {
	size := requireSameLen("WeightedClose", high, low, close)
	if size == 0 {
		return nil
	}
	out := make([]float64, size)
	for i := 0; i < size; i++ {
		out[i] = (high[i] + low[i] + 2*close[i]) / 4
	}
	return out
}
