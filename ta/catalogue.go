package ta

import (
	"math"

	"github.com/rj-raghavjoshi/numa/vec"
)

// ---------------------------------------------------------------------------
// The remaining catalogue entries: spread statistics, the vigour and chop measures, the cross
// indicators, and the Hamming weighted average.
//
// These were identified by auditing this package against TradingView's published indicators list
// rather than by intuition, which is why they are grouped by their list position rather than by
// family. The audit and what it found is recorded in docs/benchmarks.md; the entries that are
// deliberately *not* implemented are listed there with their reasons, which are mostly that they need
// cross-sectional or bucketed data the numeric layer does not have.
// ---------------------------------------------------------------------------

// StandardDeviation returns the population standard deviation of the trailing n values.
//
// It is the spread measure behind [BollingerBands] and [StandardErrorBands], exposed on its own
// because a caller who wants the dispersion without a band around it should not have to compute the
// band and discard two thirds of it.
//
// It delegates to [vec.RollingStdDev], which recomputes each window about its own mean rather than
// using the one-pass `mean(x²) - mean(x)²` identity that loses precision. That choice is why the cost
// scales with the window; see docs/benchmarks.md.
//
// The first n-1 values are NaN, and a window containing a NaN produces NaN.
func StandardDeviation(xs []float64, n int) []float64 {
	checkPeriod("StandardDeviation", n)
	if len(xs) == 0 {
		return nil
	}
	return vec.RollingStdDev(xs, n)
}

// Ratio returns the elementwise ratio of two series, a/b.
//
// It is the relative-performance plot: dividing one instrument by another removes whatever they have
// in common and leaves the question of which is doing better. A zero denominator yields 0, following
// the package's division convention rather than producing an infinity that would swamp anything
// plotted with it.
//
// Both series must be the same length. The result has that length and no warm-up.
func Ratio(a, b []float64) []float64 {
	size := requireSameLen("Ratio", a, b)
	if size == 0 {
		return nil
	}
	out := make([]float64, size)
	for i := 0; i < size; i++ {
		if b[i] == 0 {
			out[i] = 0
			continue
		}
		out[i] = a[i] / b[i]
	}
	return out
}

// Spread returns the elementwise difference of two series, a-b.
//
// It is the arithmetic counterpart of [Ratio]: where the ratio answers "how many times", the spread
// answers "how much", and the two disagree about what matters when the series have different scales.
// A basis, a crack spread and a calendar spread are all this function.
//
// Both series must be the same length. The result has that length and no warm-up.
func Spread(a, b []float64) []float64 {
	size := requireSameLen("Spread", a, b)
	if size == 0 {
		return nil
	}
	out := make([]float64, size)
	for i := 0; i < size; i++ {
		out[i] = a[i] - b[i]
	}
	return out
}

// RelativeVigorIndex returns Donald Dorsey's Relative Vigor Index and its signal line:
//
//	vigour = (close - open) / (high - low)
//	RVI    = SMA(vigour, n) / SMA(high - low, n)
//	signal = SMA(RVI, signalPeriod)
//
// The name is the idea: a bar that closes far from its open relative to its range has vigour, and
// averaging that ratio over a window measures whether the market is finishing moves or stalling. It
// is smoothed as a ratio of sums rather than a sum of ratios so that a single quiet bar cannot
// dominate.
//
// **This is not [RelativeVolatilityIndex]**, which is also abbreviated RVI and measures something
// unrelated: that one separates up-volatility from down-volatility, this one measures where within
// its range price finished. Both names are in use and neither is wrong.
//
// A zero denominator yields 0. The first n-1 values are NaN, and the signal warms up a further
// signalPeriod-1 bars. signalPeriod is conventionally 4.
func RelativeVigorIndex(open, high, low, close []float64, n, signalPeriod int) (rvi, signal []float64) {
	checkPeriod("RelativeVigorIndex", n)
	checkPeriod("RelativeVigorIndex signalPeriod", signalPeriod)
	size := requireSameLen("RelativeVigorIndex", open, high, low, close)
	if size == 0 {
		return nil, nil
	}

	vigour := make([]float64, size)
	rng := make([]float64, size)
	for i := 0; i < size; i++ {
		r := high[i] - low[i]
		rng[i] = r
		vigour[i] = 0
		if r != 0 && r == r {
			vigour[i] = (close[i] - open[i]) / r
		}
	}

	num := vec.RollingSum(vigour, n)
	den := vec.RollingSum(rng, n)
	rvi = allNaN(size)
	for i := 0; i < size; i++ {
		if num[i] != num[i] || den[i] != den[i] {
			continue
		}
		rvi[i] = ratioOr0(num[i], den[i])
	}
	signal = applyEMA(rvi, FirstValid(rvi), signalPeriod)
	return rvi, signal
}

// ChopZone returns Chande's Chop Zone:
//
//	up   = sum over n of max(high - previousClose, 0)
//	down = sum over n of max(previousClose - low, 0)
//	chop = 100 * log10(up / down) / log10(n)
//
// It sorts the recent bars by how far above and below the previous close they reached, and reports
// the resulting balance on a logarithmic scale. The log is what makes it symmetric: a market spending
// twice as much effort upward reads as +30, and one spending twice as much downward as -30, rather
// than as 2 and 0.5.
//
// The output is unbounded in principle. A window with no downward reach has an undefined logarithm
// and is reported as 0, which is the neutral value rather than an extremum, because the ratio is
// undefined rather than infinite.
//
// The first n values are NaN: the comparison needs a previous close.
func ChopZone(high, low, close []float64, n int) []float64 {
	checkPeriod("ChopZone", n)
	size := requireSameLen("ChopZone", high, low, close)
	if size == 0 {
		return nil
	}
	out := allNaN(size)
	if size < n+1 {
		return out
	}

	up := make([]float64, size)
	down := make([]float64, size)
	up[0], down[0] = math.NaN(), math.NaN()
	for i := 1; i < size; i++ {
		up[i] = math.Max(high[i]-close[i-1], 0)
		down[i] = math.Max(close[i-1]-low[i], 0)
	}
	upSum := vec.RollingSum(up, n)
	downSum := vec.RollingSum(down, n)

	logN := math.Log10(float64(n))
	for i := 0; i < size; i++ {
		u, d := upSum[i], downSum[i]
		if u != u || d != d || u == 0 || d == 0 || logN == 0 {
			continue
		}
		out[i] = 100 * math.Log10(u/d) / logN
	}
	return out
}

// HammingMA returns the moving average weighted by a Hamming window.
//
//	w_k = 0.54 - 0.46*cos(2*pi*k/(n-1)),  k = 0 .. n-1
//	out  = sum(w_k * x_{oldest+k}) / sum(w_k)
//
// It is the same construction as [ALMA]: a fixed non-monotone weight profile over the window, chosen
// so the average's frequency response has low sidelobes rather than a sharp cutoff. The Hamming
// profile is the one from signal processing, and it is gentler than a Gaussian in the middle of the
// window and heavier at the edges.
//
// For n = 1 the profile is a single weight of 1, so the result is the input. The first n-1 values are
// NaN. The cost is O(n) per element, which is inherent: a general weight profile re-weights every
// element still in the window when the window slides, so there is no running-sum recurrence -- only
// polynomial weights such as [WMA]'s have one.
func HammingMA(xs []float64, n int) []float64 {
	checkPeriod("HammingMA", n)
	if len(xs) == 0 {
		return nil
	}
	size := len(xs)
	out := allNaN(size)
	if size < n {
		return out
	}

	weights := make([]float64, n)
	var wsum float64
	if n == 1 {
		weights[0] = 1
		wsum = 1
	} else {
		for k := 0; k < n; k++ {
			w := 0.54 - 0.46*math.Cos(2*math.Pi*float64(k)/float64(n-1))
			weights[k] = w
			wsum += w
		}
	}

	for i := n - 1; i < size; i++ {
		base := i - n + 1
		var acc float64
		bad := false
		for k := 0; k < n; k++ {
			v := xs[base+k]
			if v != v {
				bad = true
				break
			}
			acc += weights[k] * v
		}
		if bad {
			continue
		}
		out[i] = acc / wsum
	}
	return out
}

// SMIErgodic returns William Blau's SMI Ergodic Indicator and Oscillator:
//
//	ergodic    = TrueStrengthIndex(close, long, short)
//	signal     = EMA(ergodic, signalPeriod)
//	oscillator = ergodic - signal
//
// It is the [TrueStrengthIndex] with a signal line and the difference between them, which is what
// turns a bounded momentum reading into a pair of crossing lines. The naming is TradingView's: the
// "indicator" is the pair and the "oscillator" is their difference, so a caller plotting the
// histogram wants the third return value.
//
// All three have the same length, with NaN through the composed warm-up.
func SMIErgodic(close []float64, long, short, signalPeriod int) (ergodic, signal, oscillator []float64) {
	checkPeriod("SMIErgodic signalPeriod", signalPeriod)
	ergodic = TrueStrengthIndex(close, long, short)
	if ergodic == nil {
		return nil, nil, nil
	}
	signal = applyEMA(ergodic, FirstValid(ergodic), signalPeriod)
	oscillator = allNaN(len(close))
	for i := range close {
		if ergodic[i] != ergodic[i] || signal[i] != signal[i] {
			continue
		}
		oscillator[i] = ergodic[i] - signal[i]
	}
	return ergodic, signal, oscillator
}

// AdvanceDecline returns the cumulative advance-decline line from per-period counts of advancing and
// declining issues.
//
//	line[i] = line[i-1] + advances[i] - declines[i]
//
// It is a market-breadth measure: the index can rise on a handful of large names while most issues
// fall, and this line records the difference rather than the average. The two inputs are counts, so
// this function is the aggregation step and not the collection step -- tallying advances and declines
// across instruments needs a cross-section, which is a caller's job because it is a question about a
// universe rather than about a series.
//
// The line starts at the first period's net count rather than at zero, so it is comparable across
// windows that start at different places only by their shape.
func AdvanceDecline(advances, declines []float64) []float64 {
	size := requireSameLen("AdvanceDecline", advances, declines)
	if size == 0 {
		return nil
	}
	net := make([]float64, size)
	for i := 0; i < size; i++ {
		net[i] = advances[i] - declines[i]
	}
	return vec.CumSum(net)
}

// MACross returns two simple moving averages and the signal where the faster crosses the slower.
//
// The signal is +1 on the bar where fast crosses above slow, -1 on the bar where it crosses below,
// and 0 otherwise, matching [vec.CrossOver] and [vec.CrossUnder]. Reporting the crossing rather than
// the sign of the difference matters because a caller acting on the sign would trade every bar the
// lines are apart, which is not what the indicator means.
//
// fast must be less than slow. All three outputs have the same length as close.
func MACross(close []float64, fast, slow int) (fastLine, slowLine []float64, cross []int8) {
	return movingAverageCross(close, fast, slow, false, false)
}

// EMACross is [MACross] with exponential averages on both sides.
func EMACross(close []float64, fast, slow int) (fastLine, slowLine []float64, cross []int8) {
	return movingAverageCross(close, fast, slow, true, true)
}

// EMASMACross is [MACross] with an exponential average on the fast side and a simple one on the slow
// side, which is TradingView's "MA with EMA Cross".
func EMASMACross(close []float64, fast, slow int) (fastLine, slowLine []float64, cross []int8) {
	return movingAverageCross(close, fast, slow, true, false)
}

// movingAverageCross builds the two lines and the crossing signal. fastExponential and
// slowExponential select the average on each side, because the three published indicators differ only
// in that choice.
func movingAverageCross(close []float64, fast, slow int, fastExponential, slowExponential bool) (fastLine, slowLine []float64, cross []int8) {
	checkPeriod("movingAverageCross fast", fast)
	checkPeriod("movingAverageCross slow", slow)
	if fast >= slow {
		panic("ta: moving average cross needs fast < slow")
	}
	if len(close) == 0 {
		return nil, nil, nil
	}

	if fastExponential {
		fastLine = EMA(close, fast)
	} else {
		fastLine = SMA(close, fast)
	}
	if slowExponential {
		slowLine = EMA(close, slow)
	} else {
		slowLine = SMA(close, slow)
	}

	up := vec.CrossOver(fastLine, slowLine)
	down := vec.CrossUnder(fastLine, slowLine)
	cross = make([]int8, len(close))
	for i := range close {
		switch {
		case up[i] != 0:
			cross[i] = 1
		case down[i] != 0:
			cross[i] = -1
		}
	}
	return fastLine, slowLine, cross
}
