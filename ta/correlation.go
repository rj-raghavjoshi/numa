package ta

import (
	"math"

	"github.com/rj-raghavjoshi/numa/vec"
)

// ---------------------------------------------------------------------------
// Correlation and realized-volatility indicators.
//
// These are the statistical indicators: rather than transforming price, they describe it.
// Correlation reports how two series move together over a window; the volatility family
// reports how much a single series moved, with different estimators trading accuracy for
// the assumption they make about the price path within a bar.
//
// # Why the windows are computed from the window
//
// Every spread here is centred on the window's own mean, taken directly from the window
// rather than from a sum of squares. That is the same choice vec/stats.go makes, and for
// the same reason: a correlation computed from `mean(xy) - mean(x)mean(y)` loses its
// significant digits exactly when the two series are close to each other, which is when
// the correlation matters most.
//
// # Zero variance
//
// A window in which either series does not move has no correlation. That is reported as
// NaN, matching [vec.Correlation], rather than as 0 -- "uncorrelated" and "undefined" are
// different claims and only one of them is true.
// ---------------------------------------------------------------------------

// CorrelationCoefficient returns the rolling Pearson correlation of xs and ys over n bars.
//
// The result lies in [-1, 1] where it is defined. A window in which either series has zero
// variance yields NaN, and so does a window containing a NaN.
//
// The first n-1 values are NaN. The cost is O(n) per bar, because the deviations are taken
// from the window mean; see the file header for why that is preferred to the O(1) form.
func CorrelationCoefficient(xs, ys []float64, n int) []float64 {
	checkPeriod("CorrelationCoefficient", n)
	size := requireSameLen("CorrelationCoefficient", xs, ys)
	if size == 0 {
		return nil
	}

	mx := SMA(xs, n)
	my := SMA(ys, n)
	out := allNaN(size)
	for i := n - 1; i < size; i++ {
		base := i - n + 1
		ax, ay := mx[i], my[i]
		if ax != ax || ay != ay {
			continue
		}
		var sxy, sxx, syy float64
		for j := 0; j < n; j++ {
			dx := xs[base+j] - ax
			dy := ys[base+j] - ay
			sxy += dx * dy
			sxx += dx * dx
			syy += dy * dy
		}
		if sxx == 0 || syy == 0 || sxx != sxx || syy != syy {
			continue
		}
		out[i] = sxy / math.Sqrt(sxx*syy)
	}
	return out
}

// CorrelationLog returns the rolling Pearson correlation of the two series' log returns
// over n bars.
//
// Correlating returns rather than levels is the standard treatment in finance, and it is
// not cosmetic: two prices that trend upward together at different rates have a level
// correlation near 1 that says nothing about whether their daily moves agree. The log
// return makes the comparison about co-movement rather than about shared drift.
//
// The first n values are NaN: the returns start at index 1 and the window needs n of them.
// A non-positive price produces NaN through the logarithm.
func CorrelationLog(xs, ys []float64, n int) []float64 {
	checkPeriod("CorrelationLog", n)
	size := requireSameLen("CorrelationLog", xs, ys)
	if size == 0 {
		return nil
	}
	rx := logReturns(xs)
	ry := logReturns(ys)
	return CorrelationCoefficient(rx, ry, n)
}

// RankCorrelation returns the rolling Spearman rank correlation of xs and ys over n bars:
// the Pearson correlation of the two series' ranks within the window.
//
// It measures whether the two series move in the same *order* rather than by the same
// amount, so it is insensitive to any monotonic transformation of either series. That makes
// it the right tool when the relationship is real but not linear, and it is why it tolerates
// outliers that would dominate a Pearson correlation.
//
// Ties are handled by [vec.Rank]'s average-rank rule. The cost is O(n log n) per bar,
// because each window is ranked independently; the first n-1 values are NaN.
func RankCorrelation(xs, ys []float64, n int) []float64 {
	checkPeriod("RankCorrelation", n)
	size := requireSameLen("RankCorrelation", xs, ys)
	if size == 0 {
		return nil
	}

	out := allNaN(size)
	// The four buffers are allocated once and reused for every window. Ranking a window would
	// otherwise allocate on every call, and at a million bars that is two million allocations for
	// no benefit. See docs/benchmarks.md.
	rx := make([]float64, n)
	ry := make([]float64, n)
	scratchX := make([]int, n)
	scratchY := make([]int, n)

	for i := n - 1; i < size; i++ {
		base := i - n + 1
		vec.RankScratchTo(rx, xs[base:i+1], scratchX)
		vec.RankScratchTo(ry, ys[base:i+1], scratchY)
		out[i] = spearmanFromRanks(rx, ry)
	}
	return out
}

// spearmanFromRanks computes the Pearson correlation of two already-ranked windows,
// returning NaN when either has no spread.
func spearmanFromRanks(rx, ry []float64) float64 {
	n := len(rx)
	meanX, meanY := 0.0, 0.0
	for i := 0; i < n; i++ {
		if rx[i] != rx[i] || ry[i] != ry[i] {
			return math.NaN()
		}
		meanX += rx[i]
		meanY += ry[i]
	}
	meanX /= float64(n)
	meanY /= float64(n)

	var sxy, sxx, syy float64
	for i := 0; i < n; i++ {
		dx := rx[i] - meanX
		dy := ry[i] - meanY
		sxy += dx * dy
		sxx += dx * dx
		syy += dy * dy
	}
	if sxx == 0 || syy == 0 {
		return math.NaN()
	}
	return sxy / math.Sqrt(sxx*syy)
}

// HistoricalVolatility returns the annualized volatility of close from its log returns:
//
//	100 * populationStdev(logReturns, n) * sqrt(annual)
//
// annual is the number of bars per year, so a daily series uses 252 (trading days) or 365
// (calendar days). The choice is the caller's because it is a property of the data, not of
// the formula, and getting it wrong scales the answer by the square root of the error.
//
// It is the Close-to-Close estimator: the simplest and the noisiest, because it ignores
// everything that happened within each bar. [VolatilityOHLC] uses the bar's range instead.
//
// The first n values are NaN: the returns start at index 1 and the window needs n of them.
func HistoricalVolatility(close []float64, n int, annual float64) []float64 {
	checkPeriod("HistoricalVolatility", n)
	if len(close) == 0 {
		return nil
	}
	rets := logReturns(close)
	out := allNaN(len(close))
	for i := n; i < len(close); i++ {
		base := i - n + 1
		// The window mean of the returns, then the population deviation about it.
		var sum float64
		for j := 0; j < n; j++ {
			if rets[base+j] != rets[base+j] {
				sum = math.NaN()
				break
			}
			sum += rets[base+j]
		}
		if sum != sum {
			continue
		}
		mean := sum / float64(n)
		out[i] = 100 * populationStdevWindow(rets, base, n, mean) * math.Sqrt(annual)
	}
	return out
}

// VolatilityOHLC returns the annualized Garman-Klass volatility of an OHLC series:
//
//	perBar = 0.5*ln(high/low)^2 - (2*ln2 - 1)*ln(close/open)^2
//	vol    = 100 * sqrt(max(mean(perBar, n), 0) * annual)
//
// The estimator uses the whole bar, so it is several times more efficient than the
// close-to-close one: it extracts the same information from fewer observations by assuming
// the price follows a geometric Brownian motion within the bar. That assumption is the
// price of the extra efficiency, and it is why the result can differ from
// [HistoricalVolatility] by a meaningful amount rather than by noise.
//
// The per-bar quantity can be negative for a bar whose open-to-close move was large
// relative to its range, so the window mean is clamped at zero before the square root; an
// unclamped negative would produce NaN and poison a series whose inputs are perfectly
// valid.
//
// The first n-1 values are NaN.
func VolatilityOHLC(open, high, low, close []float64, n int, annual float64) []float64 {
	checkPeriod("VolatilityOHLC", n)
	size := requireSameLen("VolatilityOHLC", open, high, low, close)
	if size == 0 {
		return nil
	}

	const twoLn2Minus1 = 2*math.Ln2 - 1
	perBar := make([]float64, size)
	for i := 0; i < size; i++ {
		switch {
		case low[i] <= 0 || open[i] <= 0 || high[i] != high[i] || low[i] != low[i]:
			perBar[i] = math.NaN()
			continue
		}
		lr := math.Log(high[i] / low[i])
		co := math.Log(close[i] / open[i])
		perBar[i] = 0.5*lr*lr - twoLn2Minus1*co*co
	}

	out := allNaN(size)
	for i := n - 1; i < size; i++ {
		base := i - n + 1
		var sum float64
		for j := 0; j < n; j++ {
			if perBar[base+j] != perBar[base+j] {
				sum = math.NaN()
				break
			}
			sum += perBar[base+j]
		}
		if sum != sum {
			continue
		}
		mean := sum / float64(n)
		if mean < 0 {
			mean = 0
		}
		out[i] = 100 * math.Sqrt(mean*annual)
	}
	return out
}
