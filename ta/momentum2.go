package ta

import (
	"math"

	"github.com/rj-raghavjoshi/numa/vec"
)

// ---------------------------------------------------------------------------
// Second-tier momentum indicators.
//
// These are the momentum measures that are not a difference or a ratio of two prices: a
// double-smoothed position inside the range, a volatility-scaled index, a composite of three
// different readings, and the two components of a bar's power. They share no formula, and they are
// grouped here because they share a *testing* problem -- each has a degenerate case (a flat range, a
// zero divisor, an absent direction) whose value is a convention rather than a computation, so each
// documents the convention it chose and the test pins it.
// ---------------------------------------------------------------------------

// PercentRank returns the percentage of the trailing n values that are strictly less than the
// current one.
//
// It is the rank of the newest observation inside its own window, expressed as a percentage, and it
// is the building block of any indicator that asks "is this reading historically high or low" rather
// than "is it high or low". The window includes the current value, which cannot be less than itself,
// so the maximum output is 100*(n-1)/n rather than 100.
//
// It delegates to [vec.RollingPercentRank], which maintains the window's order in a binary indexed
// tree rather than rescanning it. That turns a kernel whose cost grew with the lookback into one
// that does not, which matters because a caller reaching for a percent rank is usually asking about
// a long one.
//
// A window containing a NaN produces NaN. The first n-1 values are NaN.
func PercentRank(xs []float64, n int) []float64 {
	checkPeriod("PercentRank", n)
	if len(xs) == 0 {
		return nil
	}
	return vec.RollingPercentRank(xs, n)
}

// StochasticMomentumIndex returns William Blau's Stochastic Momentum Index and its signal line:
//
//	midpoint = (highest(high, n) + lowest(low, n)) / 2
//	raw      = close - midpoint
//	rng      = highest(high, n) - lowest(low, n)
//	num      = EMA(EMA(raw, smoothK), smoothK)
//	den      = EMA(EMA(rng, smoothK), smoothK)
//	SMI      = 100 * num / den
//	signal   = EMA(SMI, smoothD)
//
// It is the stochastic oscillator with two changes: the position is measured from the range's
// *midpoint* rather than its low, which makes the result symmetric about zero instead of bounded to
// [0,100], and both the numerator and the denominator are smoothed twice, which removes the
// stochastic's characteristic jitter.
//
// Smoothing the denominator as well as the numerator matters: dividing by the raw range would let a
// single narrow bar produce a spike that the smoothing is there to prevent.
//
// A zero denominator yields 0 rather than an infinity, following the package's division convention.
// TradingView's defaults are n = 10, smoothK = 3 and smoothD = 3. Both outputs have the same length
// as the inputs, with NaN through the composed warm-up.
func StochasticMomentumIndex(high, low, close []float64, n, smoothK, smoothD int) (smi, signal []float64) {
	checkPeriod("StochasticMomentumIndex", n)
	checkPeriod("StochasticMomentumIndex smoothK", smoothK)
	checkPeriod("StochasticMomentumIndex smoothD", smoothD)
	size := requireSameLen("StochasticMomentumIndex", high, low, close)
	if size == 0 {
		return nil, nil
	}

	hi := Highest(high, n)
	lo := Lowest(low, n)
	raw := make([]float64, size)
	rng := make([]float64, size)
	for i := 0; i < size; i++ {
		if hi[i] != hi[i] || lo[i] != lo[i] || close[i] != close[i] {
			raw[i] = math.NaN()
			rng[i] = math.NaN()
			continue
		}
		raw[i] = close[i] - (hi[i]+lo[i])/2
		rng[i] = hi[i] - lo[i]
	}

	num := doubleEMA(raw, smoothK)
	den := doubleEMA(rng, smoothK)

	smi = allNaN(size)
	for i := 0; i < size; i++ {
		switch {
		case num[i] != num[i] || den[i] != den[i]:
			smi[i] = math.NaN()
		case den[i] == 0:
			smi[i] = 0
		default:
			smi[i] = 100 * num[i] / den[i]
		}
	}
	signal = applyEMA(smi, FirstValid(smi), smoothD)
	return smi, signal
}

// doubleEMA returns EMA(EMA(xs, n), n), starting each pass at its own first valid index.
func doubleEMA(xs []float64, n int) []float64 {
	first := applyEMA(xs, FirstValid(xs), n)
	return applyEMA(first, FirstValid(first), n)
}

// RelativeVolatilityIndex returns Donald Dorsey's Relative Volatility Index.
//
//	sd   = population standard deviation of close over n bars
//	up   = sd where close rose, else 0
//	down = sd where close fell, else 0
//	RVI  = 100 * RMA(up, n) / (RMA(up, n) + RMA(down, n))
//
// The insight is that *volatility* has a direction: the standard deviation of an advance and of a
// decline are different quantities, and measuring them separately says whether the current movement
// is being driven up or down. It is used as a trend filter rather than a timing signal, because it
// is slow.
//
// A window with no movement at all in either direction leaves both terms zero, and the index is
// reported as 50 -- the neutral midpoint, matching the convention [Stochastic] and [MoneyFlowIndex]
// use for a flat market rather than inventing a direction.
//
// The first 2n-2 values are NaN: the deviation needs n-1 bars, and the smoothing needs n-1 more.
func RelativeVolatilityIndex(close []float64, n int) []float64 {
	checkPeriod("RelativeVolatilityIndex", n)
	if len(close) == 0 {
		return nil
	}
	size := len(close)
	sd := vec.RollingStdDev(close, n)

	up := allNaN(size)
	down := allNaN(size)
	for i := 1; i < size; i++ {
		if sd[i] != sd[i] || close[i] != close[i] || close[i-1] != close[i-1] {
			continue
		}
		switch {
		case close[i] > close[i-1]:
			up[i], down[i] = sd[i], 0
		case close[i] < close[i-1]:
			up[i], down[i] = 0, sd[i]
		default:
			up[i], down[i] = 0, 0
		}
	}

	from := FirstValid(sd)
	if from < 1 {
		from = 1
	}
	upSmooth := applyRMA(up, from, n)
	downSmooth := applyRMA(down, from, n)

	out := allNaN(size)
	for i := 0; i < size; i++ {
		u, d := upSmooth[i], downSmooth[i]
		if u != u || d != d {
			continue
		}
		if sum := u + d; sum != 0 {
			out[i] = 100 * u / sum
		} else {
			out[i] = 50
		}
	}
	return out
}

// ConnorsRSI returns the Connors Research composite momentum indicator, the mean of three readings
// that are all bounded to [0,100]:
//
//	RSI(close, rsiPeriod)
//	RSI(streak, streakPeriod)      where streak counts consecutive up or down closes
//	PercentRank(ROC(close, 1), rankPeriod)
//
// The three components fail in different markets and the average is the point: the first is the
// conventional overbought/oversold reading, the second measures how *long* a run has lasted rather
// than how far it went, and the third places today's move in the context of recent ones. A reading
// near 0 or 100 therefore requires all three to agree.
//
// The streak is signed -- positive for consecutive rises, negative for falls, zero after an
// unchanged close -- and is fed to RSI as a series, which is why a run of any length produces a
// sensible reading rather than needing its own definition.
//
// TradingView's defaults are 3, 2 and 100. The warm-up is the longest of the three components'.
func ConnorsRSI(close []float64, rsiPeriod, streakPeriod, rankPeriod int) []float64 {
	checkPeriod("ConnorsRSI rsiPeriod", rsiPeriod)
	checkPeriod("ConnorsRSI streakPeriod", streakPeriod)
	checkPeriod("ConnorsRSI rankPeriod", rankPeriod)
	if len(close) == 0 {
		return nil
	}
	size := len(close)

	streak := make([]float64, size)
	streak[0] = math.NaN()
	for i := 1; i < size; i++ {
		if close[i] != close[i] || close[i-1] != close[i-1] {
			streak[i] = math.NaN()
			continue
		}
		switch {
		case close[i] > close[i-1]:
			if prev := streak[i-1]; prev > 0 && prev == prev {
				streak[i] = prev + 1
			} else {
				streak[i] = 1
			}
		case close[i] < close[i-1]:
			if prev := streak[i-1]; prev < 0 && prev == prev {
				streak[i] = prev - 1
			} else {
				streak[i] = -1
			}
		default:
			streak[i] = 0
		}
	}

	priceRSI := RSI(close, rsiPeriod)
	streakRSI := RSI(streak, streakPeriod)
	rank := PercentRank(ROC(close, 1), rankPeriod)

	out := allNaN(size)
	for i := 0; i < size; i++ {
		a, b, c := priceRSI[i], streakRSI[i], rank[i]
		if a != a || b != b || c != c {
			continue
		}
		out[i] = (a + b + c) / 3
	}
	return out
}

// ElderRay returns the two components of Alexander Elder's Ray: the bull power and the bear power.
//
//	ema       = EMA(close, n)
//	bullPower = high - ema
//	bearPower = low  - ema
//
// Both are measured against the same average, so their signs together describe who was in control
// during the bar: a positive bull power means the high exceeded the average, a negative bear power
// means the low fell below it. Elder's reading requires the trend to be established by the average's
// own slope first -- the ray says whether the trend has power, not whether it exists.
//
// The first n-1 values of both outputs are NaN.
func ElderRay(high, low, close []float64, n int) (bullPower, bearPower []float64) {
	checkPeriod("ElderRay", n)
	size := requireSameLen("ElderRay", high, low, close)
	if size == 0 {
		return nil, nil
	}
	ema := EMA(close, n)
	bullPower = make([]float64, size)
	bearPower = make([]float64, size)
	for i := 0; i < size; i++ {
		bullPower[i] = high[i] - ema[i]
		bearPower[i] = low[i] - ema[i]
	}
	return bullPower, bearPower
}
