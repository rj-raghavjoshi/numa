package ta

// ---------------------------------------------------------------------------
// Volatility indicators.
// ---------------------------------------------------------------------------

// ATR returns the n-period Average True Range: Wilder's smoothing of [TrueRange].
//
// The warm-up is n bars, because [RMA] seeds with the SMA of the first n true-range
// values. The true range of the first bar is high-low, so the only NaNs are the
// smoothing seed's.
func ATR(high, low, close []float64, n int) []float64 {
	checkPeriod("ATR", n)
	requireSameLen("ATR", high, low, close)
	if len(high) == 0 {
		return nil
	}
	return RMA(TrueRange(high, low, close), n)
}

// BollingerBands returns the n-period Bollinger bands of close, at k population
// standard deviations from the middle band.
//
//	middle = SMA(close, n)
//	upper  = middle + k*stdev(close, n)
//	lower  = middle - k*stdev(close, n)
//
// The first n-1 values of all three outputs are NaN.
//
// # Population, not sample
//
// The deviation is the *population* standard deviation, dividing by n rather than
// n-1, which is what TradingView's `ta.stdev` and therefore `ta.bb` use. The
// difference is a factor of sqrt(n/(n-1)) -- about 2.6% at n=20 -- so choosing the
// wrong one would be a visible, systematic error rather than a rounding difference.
//
// # Why the deviations are computed from the window
//
// The middle band is the window mean, so the deviations are taken from it directly
// rather than from a running sum of squares. The `mean(x^2) - mean(x)^2` identity
// would be O(1) per element instead of O(window), and would also lose most of the
// significant digits whenever the price level dwarfs the bar-to-bar range, which is
// the normal case. The same reasoning as vec.Variance; see vec/stats.go.
func BollingerBands(close []float64, n int, k float64) (upper, middle, lower []float64) {
	checkPeriod("BollingerBands", n)
	if len(close) == 0 {
		return nil, nil, nil
	}
	middle = SMA(close, n)
	upper = make([]float64, len(close))
	lower = make([]float64, len(close))
	fillNaN(upper, 0, n-1)
	fillNaN(lower, 0, n-1)

	for i := n - 1; i < len(close); i++ {
		m := middle[i]
		sd := populationStdevWindow(close, i-n+1, n, m)
		upper[i] = m + k*sd
		lower[i] = m - k*sd
	}
	return upper, middle, lower
}

// BBPercentB returns the Bollinger %B of close:
//
//	(close - lower) / (upper - lower)
//
// It locates the close within the band: 0 at the lower band, 1 at the upper, and
// beyond that outside. A flat band (upper == lower, which happens when the window has
// zero deviation) yields 0 rather than an infinity, following vec.SafeDiv.
//
// The first n-1 values are NaN.
func BBPercentB(close []float64, n int, k float64) []float64 {
	checkPeriod("BBPercentB", n)
	if len(close) == 0 {
		return nil
	}
	upper, _, lower := BollingerBands(close, n, k)
	out := make([]float64, len(close))
	for i := range close {
		width := upper[i] - lower[i]
		// NaN propagates through the width and through the comparison, so the
		// warm-up region and any NaN window both fall out as NaN.
		if width == 0 {
			out[i] = 0
		} else {
			out[i] = (close[i] - lower[i]) / width
		}
	}
	return out
}

// BBWidth returns the Bollinger band width of close: (upper - lower) / middle.
//
// It is a normalized volatility measure, so it is comparable across instruments at
// different price levels. A zero middle band yields 0.
//
// The first n-1 values are NaN.
func BBWidth(close []float64, n int, k float64) []float64 {
	checkPeriod("BBWidth", n)
	if len(close) == 0 {
		return nil
	}
	upper, middle, lower := BollingerBands(close, n, k)
	out := make([]float64, len(close))
	for i := range close {
		if middle[i] == 0 {
			out[i] = 0
		} else {
			out[i] = (upper[i] - lower[i]) / middle[i]
		}
	}
	return out
}
