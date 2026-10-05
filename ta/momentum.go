package ta

import (
	"math"

	"github.com/rj-raghavjoshi/numa/vec"
)

// ---------------------------------------------------------------------------
// Momentum indicators.
//
// These measure how fast and in which direction price is moving. Most are ratios of
// a gain measure to a loss measure, which is where the divide-by-zero policies below
// come from: a flat stretch is routine in a price series, and an infinity produced
// there would poison everything downstream.
//
// # Division-by-zero policy
//
// Each function documents its own choice, because the neutral value differs by
// indicator and picking one globally would be wrong in most of them:
//
//	RSI    both averages zero -> 50 (neither side dominates)
//	RSI    only losses zero   -> 100 (all gain)
//	RSI    only gains zero    -> 0 (all loss)
//	Stoch  flat window        -> 50 (the close sits in the middle of nothing)
//	%R     flat window        -> 0
//	CCI    zero mean deviation-> 0
//	CMO    both sums zero     -> 0
//	ROC    zero base          -> 0, following vec.SafeDiv
//
// Every one of these is pinned by a test, because a "neutral" value that disagrees
// with the reference implementation is a silent, systematic error rather than a
// rounding difference.
// ---------------------------------------------------------------------------

// RSI returns Wilder's n-period Relative Strength Index of close, a value in [0,100].
//
// # Formula
//
//	change[i] = close[i] - close[i-1]
//	gain[i]   = max(change[i], 0)
//	loss[i]   = max(-change[i], 0)
//	avgGain   = RMA(gain, n)
//	avgLoss   = RMA(loss, n)
//	RSI       = 100 - 100/(1 + avgGain/avgLoss)
//
// # Warm-up and why this does not compose from RMA
//
// The first change is undefined because there is no previous close, so a plain
// RMA(change) would consume a leading NaN as part of its seed and report NaN forever.
// The smoothing is therefore inlined here, started at the first real change, which
// makes the first output land at index n rather than n-1: the first n changes are
// consumed to form Wilder's seed.
//
// "The first real change" is computed from the input rather than assumed to be at
// index 1, which is what makes RSI safe to feed a composed series. Given an input whose
// first element is NaN -- a streak count, a spread, the output of another indicator --
// assuming index 1 would seed the recurrence with a NaN and report NaN for every bar,
// permanently, because a seeded recurrence has no way to forget. The first change is
// therefore the first index at which both the value and its predecessor are defined, and
// the first output lands n-1 bars after that. For an ordinary price series the two
// readings coincide and the first output is at index n.
//
// # NaN
//
// A NaN in close makes the corresponding change NaN and poisons the running averages
// from that point on, which is inherent to the recurrence. See the note on
// [EMA] for the same asymmetry.
func RSI(close []float64, n int) []float64 {
	checkPeriod("RSI", n)
	size := len(close)
	if size == 0 {
		return nil
	}
	out := allNaN(size)

	// Find the first index at which a one-bar change is defined: the first valid value and then one
	// further, because a change also needs its predecessor.
	//
	// Reading changes unconditionally from index 1 is only safe when the input's first element is
	// valid. For a *composed* input -- a streak count, a spread, anything with a leading NaN -- it
	// feeds that NaN into the seed, and a seeded recurrence cannot forget: every later output is NaN
	// too. That is a permanent failure rather than a warm-up, so the start index is computed here
	// instead of assumed.
	start := FirstValid(close)
	if start < 0 {
		return out
	}
	if start == 0 {
		start = 1
	} else {
		start++
	}
	if start+n > size {
		return out
	}

	// Seed: the n changes at indices start..start+n-1.
	var gSum, lSum float64
	for i := start; i < start+n; i++ {
		c := close[i] - close[i-1]
		switch {
		case c != c:
			gSum, lSum = math.NaN(), math.NaN()
		case c > 0:
			gSum += c
		case c < 0:
			lSum -= c
		}
	}
	alpha := 1 / float64(n)
	avgGain := gSum / float64(n)
	avgLoss := lSum / float64(n)
	out[start+n-1] = rsiFromAverages(avgGain, avgLoss)

	for i := start + n; i < size; i++ {
		c := close[i] - close[i-1]
		var g, l float64
		switch {
		case c != c:
			g, l = math.NaN(), math.NaN()
		case c > 0:
			g = c
		case c < 0:
			l = -c
		}
		avgGain = alpha*g + (1-alpha)*avgGain
		avgLoss = alpha*l + (1-alpha)*avgLoss
		out[i] = rsiFromAverages(avgGain, avgLoss)
	}
	return out
}

// rsiFromAverages maps the smoothed gain and loss to an RSI value, applying the
// documented neutral cases rather than letting the division produce a NaN or an
// infinity.
func rsiFromAverages(avgGain, avgLoss float64) float64 {
	switch {
	case avgGain != avgGain || avgLoss != avgLoss:
		return math.NaN()
	case avgGain == 0 && avgLoss == 0:
		return 50
	case avgLoss == 0:
		return 100
	case avgGain == 0:
		return 0
	}
	return 100 - 100/(1+avgGain/avgLoss)
}

// MACD returns the moving average convergence divergence of close:
//
//	macd   = EMA(close, fast) - EMA(close, slow)
//	signal = EMA(macd, signal)
//	hist   = macd - signal
//
// The signal line is started at the MACD line's first valid index rather than being
// seeded with the MACD's leading NaNs; see [applyFrom]. The three outputs have the
// same length as close, and the histogram is NaN wherever the signal line is.
//
// All three are returned together so the shared EMA work is done once.
func MACD(close []float64, fast, slow, signal int) (macd, signalLine, hist []float64) {
	checkPeriod("MACD fast", fast)
	checkPeriod("MACD slow", slow)
	checkPeriod("MACD signal", signal)
	if len(close) == 0 {
		return nil, nil, nil
	}

	fastEMA := EMA(close, fast)
	slowEMA := EMA(close, slow)
	macd = make([]float64, len(close))
	for i := range close {
		macd[i] = fastEMA[i] - slowEMA[i]
	}

	signalLine = applyEMA(macd, FirstValid(macd), signal)
	hist = make([]float64, len(close))
	for i := range close {
		hist[i] = macd[i] - signalLine[i]
	}
	return macd, signalLine, hist
}

// Stochastic returns the stochastic oscillator %K and %D of a high/low/close series:
//
//	rawK = 100 * (close - lowest(low, kPeriod)) / (highest(high, kPeriod) - lowest(low, kPeriod))
//	K    = SMA(rawK, kSmooth)
//	D    = SMA(K, dSmooth)
//
// A window with no range at all (highest == lowest) yields rawK = 50, the neutral
// mid-point. The first output of K is at index kPeriod+kSmooth-2 and of D at
// kPeriod+kSmooth+dSmooth-3.
func Stochastic(high, low, close []float64, kPeriod, kSmooth, dSmooth int) (k, d []float64) {
	checkPeriod("Stochastic kPeriod", kPeriod)
	checkPeriod("Stochastic kSmooth", kSmooth)
	checkPeriod("Stochastic dSmooth", dSmooth)
	size := requireSameLen("Stochastic", high, low, close)
	if size == 0 {
		return nil, nil
	}

	rawK := rawStochastic(high, low, close, kPeriod)
	k = SMA(rawK, kSmooth)
	d = SMA(k, dSmooth)
	return k, d
}

// rawStochastic computes the unsmoothed %K, which is also what TradingView's
// `ta.stoch` returns on its own.
func rawStochastic(high, low, close []float64, n int) []float64 {
	hi := Highest(high, n)
	lo := Lowest(low, n)
	out := make([]float64, len(close))
	for i := range close {
		span := hi[i] - lo[i]
		switch {
		case hi[i] != hi[i] || lo[i] != lo[i]:
			out[i] = math.NaN()
		case span == 0:
			out[i] = 50
		default:
			out[i] = 100 * (close[i] - lo[i]) / span
		}
	}
	return out
}

// StochRSI returns the stochastic oscillator applied to [RSI] rather than to price:
//
//	rsi   = RSI(close, rsiPeriod)
//	rawK  = 100 * (rsi - lowest(rsi, stochPeriod)) / (highest(rsi, stochPeriod) - lowest(rsi, stochPeriod))
//	K     = SMA(rawK, kSmooth)
//	D     = SMA(K, dSmooth)
//
// It is a fast oscillator of an already oscillating series, so it reaches its extremes
// far more often than RSI does. The same flat-window rule applies: no spread yields 50.
//
// The warm-up is long because it is the sum of three stretches -- RSI's, then the
// stochastic window, then the two smoothings -- and the first valid index is reported
// by [FirstValid] rather than derived by hand.
func StochRSI(close []float64, rsiPeriod, stochPeriod, kSmooth, dSmooth int) (k, d []float64) {
	checkPeriod("StochRSI rsiPeriod", rsiPeriod)
	checkPeriod("StochRSI stochPeriod", stochPeriod)
	checkPeriod("StochRSI kSmooth", kSmooth)
	checkPeriod("StochRSI dSmooth", dSmooth)
	if len(close) == 0 {
		return nil, nil
	}

	r := RSI(close, rsiPeriod)
	// The stochastic window is taken over RSI, so high and low are both RSI. Leading
	// NaNs are handled by Highest/Lowest's NaN counting and by SMA's rolling sum.
	rawK := rawStochastic(r, r, r, stochPeriod)
	k = SMA(rawK, kSmooth)
	d = SMA(k, dSmooth)
	return k, d
}

// WilliamsPercentR returns the n-period Williams %R of a high/low/close series:
//
//	%R = -100 * (highest(high,n) - close) / (highest(high,n) - lowest(low,n))
//
// The result lies in [-100, 0], with -100 at the lowest low of the window and 0 at the
// highest high. A window with no range at all yields 0 rather than a division by zero.
//
// The first n-1 values are NaN. %R is the same quantity as the stochastic oscillator
// reflected about -50: %R = rawK - 100 for the same window.
func WilliamsPercentR(high, low, close []float64, n int) []float64 {
	checkPeriod("WilliamsPercentR", n)
	size := requireSameLen("WilliamsPercentR", high, low, close)
	if size == 0 {
		return nil
	}

	hi := Highest(high, n)
	lo := Lowest(low, n)
	out := make([]float64, size)
	for i := range close {
		span := hi[i] - lo[i]
		switch {
		case hi[i] != hi[i] || lo[i] != lo[i]:
			out[i] = math.NaN()
		case span == 0:
			out[i] = 0
		default:
			out[i] = -100 * (hi[i] - close[i]) / span
		}
	}
	return out
}

// CCI returns the n-period Commodity Channel Index of a high/low/close series:
//
//	TP       = (high + low + close) / 3
//	CCI      = (TP - SMA(TP, n)) / (0.015 * meanDeviation(TP, n))
//
// where the mean deviation is the average of |TP - SMA(TP,n)| over the window. The
// 0.015 constant is Lambert's, chosen to make roughly 70-80% of values fall inside
// [-100, 100]; it is a scaling convention, not a tunable.
//
// A window whose mean deviation is zero yields 0 rather than an infinity. The mean
// deviation is computed from the window mean directly, for the same accuracy reason as
// [BollingerBands].
func CCI(high, low, close []float64, n int) []float64 {
	checkPeriod("CCI", n)
	size := requireSameLen("CCI", high, low, close)
	if size == 0 {
		return nil
	}

	tp := TypicalPrice(high, low, close)
	mean := SMA(tp, n)
	out := allNaN(size)
	for i := n - 1; i < size; i++ {
		m := mean[i]
		var dev float64
		base := i - n + 1
		for j := 0; j < n; j++ {
			dev += math.Abs(tp[base+j] - m)
		}
		dev /= float64(n)
		if dev == 0 || dev != dev || m != m {
			out[i] = 0
			continue
		}
		out[i] = (tp[i] - m) / (0.015 * dev)
	}
	return out
}

// CMO returns the n-period Chande Momentum Oscillator of close, a value in [-100,100]:
//
//	CMO = 100 * (sumGain - sumLoss) / (sumGain + sumLoss)
//
// over the last n changes. Unlike [RSI] the gains and losses are plain sums rather than
// Wilder-smoothed, so the two indicators respond differently to the same data even
// though they measure the same thing.
//
// If every change in the window is zero the result is 0. The first n values are NaN,
// because n changes require n+1 closes.
//
// # Why this uses rolling sums
//
// A window of changes is a textbook use for [series.Sum]: the two sums are maintained
// in O(1) per element instead of resummed over the window. The first change has no
// previous close, so iteration starts at index 1; the window of the sum then covers
// indices i-n+1..i, which is exactly the CMO window once i >= n. Because the sum
// counts NaN changes rather than accumulating them, a bad value affects precisely the
// windows that contain it.
//
// The same shape applies to [ROC], which is a single lag rather than a sum.
func CMO(close []float64, n int) []float64 {
	checkPeriod("CMO", n)
	size := len(close)
	if size == 0 {
		return nil
	}
	out := allNaN(size)
	if size <= n {
		return out
	}

	// The gains and losses are materialised so the window sums can run as batch kernels. That costs
	// two extra slices, and it buys consistency with the rest of the package rather than a measured
	// win: the streaming sums this replaced had no state a caller could observe, and the batch form
	// is the one every other whole-slice indicator uses. See [SMA] for why the batch kernel is the
	// default and why its speed is not inferred from a sum-only micro-benchmark.
	gain := make([]float64, size)
	loss := make([]float64, size)
	gain[0], loss[0] = math.NaN(), math.NaN()
	for i := 1; i < size; i++ {
		c := close[i] - close[i-1]
		switch {
		case c != c:
			// A NaN change makes both windows that contain it NaN.
			gain[i], loss[i] = math.NaN(), math.NaN()
		case c > 0:
			gain[i], loss[i] = c, 0
		case c < 0:
			gain[i], loss[i] = 0, -c
		default:
			gain[i], loss[i] = 0, 0
		}
	}
	sg := vec.RollingSum(gain, n)
	sl := vec.RollingSum(loss, n)

	for i := n; i < size; i++ {
		switch {
		case sg[i] != sg[i] || sl[i] != sl[i]:
			out[i] = math.NaN()
		case sg[i] == 0 && sl[i] == 0:
			out[i] = 0
		default:
			out[i] = 100 * (sg[i] - sl[i]) / (sg[i] + sl[i])
		}
	}
	return out
}

// ROC returns the n-period rate of change of close as a percentage:
//
//	100 * (close[i] - close[i-n]) / close[i-n]
//
// It is [Momentum] normalized by the earlier price, so it is comparable across
// instruments at different levels. A zero base yields 0, following vec.SafeDiv; the
// first n values are NaN.
func ROC(close []float64, n int) []float64 {
	checkPeriod("ROC", n)
	if len(close) == 0 {
		return nil
	}
	rates := vec.Rate(close, n)
	out := make([]float64, len(close))
	for i, r := range rates {
		// Rate reports NaN for the warm-up and 0 for a zero base; scaling preserves
		// both, and NaN*100 is still NaN.
		out[i] = 100 * r
	}
	return out
}

// Momentum returns the n-period price difference close[i] - close[i-n].
//
// It is [ROC] without the normalization, so its scale depends on the instrument's
// price level. n = 0 returns zeros and a negative n is a forward difference, matching
// vec.Diff.
func Momentum(close []float64, n int) []float64 {
	if len(close) == 0 {
		return nil
	}
	return vec.Diff(close, n)
}
