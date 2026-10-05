package ta

import (
	"math"
)

// ---------------------------------------------------------------------------
// Volume and range indicators.
//
// The indicators here all answer a question about the *bar* rather than about the close: who won
// inside it, how much movement it bought, whether volume is confirming the move, and how the range
// itself is changing. That makes them the ones that notice a market where price is going nowhere on
// falling volume, which no close-based indicator can see.
// ---------------------------------------------------------------------------

// BalanceOfPower returns the n-bar simple average of the raw balance of power:
//
//	raw[i] = (close - open) / (high - low)
//	BOP    = SMA(raw, n)
//
// The raw ratio is the share of the bar's range that the close captured, signed: +1 means the bar
// closed at its high after opening at its low, -1 the reverse. It is the simplest statement of who
// controlled the bar, and averaging it converts a per-bar reading into a trend.
//
// A bar with no range has no balance to measure and contributes 0 rather than an infinity, which is
// the neutral value: neither side moved.
//
// A period of 1 returns the raw ratio. The first n-1 values are NaN. TradingView's default period is
// 14.
func BalanceOfPower(open, high, low, close []float64, n int) []float64 {
	checkPeriod("BalanceOfPower", n)
	size := requireSameLen("BalanceOfPower", open, high, low, close)
	if size == 0 {
		return nil
	}
	raw := make([]float64, size)
	for i := 0; i < size; i++ {
		rng := high[i] - low[i]
		if rng == 0 || rng != rng {
			raw[i] = 0
			continue
		}
		raw[i] = (close[i] - open[i]) / rng
	}
	return SMA(raw, n)
}

// EaseOfMovement returns the n-bar average of the ease of movement:
//
//	distance = (high + low)/2 - (previous high + previous low)/2
//	ease     = distance * (high - low) / volume
//	EOM      = SMA(ease, n)
//
// The numerator is how far the bar's midpoint travelled and the denominator is what that travel
// cost in volume, so a large move on little volume reads as high and a small move on heavy volume
// reads as low. It is the standard way to ask whether a price change was *effortless*, which is what
// an early trend looks like.
//
// A bar with no range or no volume contributes 0: there was no movement to measure, or no
// participation to measure it against. The first n values are NaN, because the distance needs a
// previous bar. TradingView's default period is 14.
func EaseOfMovement(high, low, volume []float64, n int) []float64 {
	checkPeriod("EaseOfMovement", n)
	size := requireSameLen("EaseOfMovement", high, low, volume)
	if size == 0 {
		return nil
	}
	ease := make([]float64, size)
	ease[0] = math.NaN()
	for i := 1; i < size; i++ {
		rng := high[i] - low[i]
		mid := (high[i] + low[i]) / 2
		prevMid := (high[i-1] + low[i-1]) / 2
		if rng != rng || mid != mid || prevMid != prevMid || volume[i] != volume[i] {
			ease[i] = math.NaN()
			continue
		}
		if rng == 0 || volume[i] == 0 {
			ease[i] = 0
			continue
		}
		ease[i] = (mid - prevMid) * rng / volume[i]
	}
	return SMA(ease, n)
}

// KlingerOscillator returns Stephen Klinger's volume oscillator and its signal line:
//
//	trend    = +1 when (high+low+close) rose, else -1
//	dm       = high - low
//	cm       = cm[previous] + dm  when the trend is unchanged
//	           dm[previous] + dm   when it reversed
//	vf       = volume * |2*(dm/cm - 1)| * trend * 100
//	KVO      = EMA(vf, fast) - EMA(vf, slow)
//	signal   = EMA(KVO, signalPeriod)
//
// The running `cm` is the part worth understanding: it accumulates the day's range while the trend
// persists, so a sustained move divides the volume force by a growing number and the oscillator
// measures volume per unit of *trend*, not per bar. When the trend reverses the accumulation resets
// to just the previous and current ranges, which is what makes the oscillator spike at turning
// points.
//
// fast must be less than slow. The first value is at index 1, since the trend needs a previous bar.
// TradingView's defaults are 34, 55 and 13.
func KlingerOscillator(high, low, close, volume []float64, fast, slow, signalPeriod int) (kvo, signal []float64) {
	checkPeriod("KlingerOscillator fast", fast)
	checkPeriod("KlingerOscillator slow", slow)
	checkPeriod("KlingerOscillator signalPeriod", signalPeriod)
	if fast >= slow {
		panic("ta: KlingerOscillator fast period must be less than slow period")
	}
	size := requireSameLen("KlingerOscillator", high, low, close, volume)
	if size == 0 {
		return nil, nil
	}

	vf := allNaN(size)
	prevTrend := 0.0
	var cm float64
	for i := 1; i < size; i++ {
		tp, prevTP := high[i]+low[i]+close[i], high[i-1]+low[i-1]+close[i-1]
		if tp != tp || prevTP != prevTP || volume[i] != volume[i] {
			continue
		}
		trend := 1.0
		if tp < prevTP {
			trend = -1
		}
		dm := high[i] - low[i]
		prevDM := high[i-1] - low[i-1]
		if dm != dm || prevDM != prevDM {
			continue
		}
		switch {
		case i == 1 || prevTrend == 0:
			cm = dm
		case trend == prevTrend:
			cm += dm
		default:
			cm = prevDM + dm
		}
		prevTrend = trend
		if cm == 0 {
			vf[i] = 0
			continue
		}
		vf[i] = volume[i] * math.Abs(2*(dm/cm-1)) * trend * 100
	}

	from := 1
	fastEMA := applyEMA(vf, from, fast)
	slowEMA := applyEMA(vf, from, slow)
	kvo = make([]float64, size)
	for i := 0; i < size; i++ {
		kvo[i] = fastEMA[i] - slowEMA[i]
	}
	signal = applyEMA(kvo, FirstValid(kvo), signalPeriod)
	return kvo, signal
}

// ChaikinVolatility returns the rate of change of a smoothed trading range:
//
//	hl = EMA(high - low, n)
//	CV = 100 * (hl[i] - hl[i-n]) / hl[i-n]
//
// It measures whether the daily range is *expanding or contracting* rather than how large it is,
// which is the distinction that makes it useful: a market can be volatile and calming down or quiet
// and waking up, and the level alone cannot tell them apart. A reading near zero is a range that has
// not changed over the window.
//
// A zero value n bars ago yields 0, following the package's division convention. The first 2n-1
// values are NaN: the EMA needs n-1 bars and the lookback needs n more. TradingView's default period
// is 10.
func ChaikinVolatility(high, low []float64, n int) []float64 {
	checkPeriod("ChaikinVolatility", n)
	size := requireSameLen("ChaikinVolatility", high, low)
	if size == 0 {
		return nil
	}
	rng := make([]float64, size)
	for i := 0; i < size; i++ {
		rng[i] = high[i] - low[i]
	}
	smoothed := EMA(rng, n)

	out := allNaN(size)
	for i := 2*n - 1; i < size; i++ {
		base := smoothed[i-n]
		if base != base || smoothed[i] != smoothed[i] {
			continue
		}
		if base == 0 {
			out[i] = 0
			continue
		}
		out[i] = 100 * (smoothed[i] - base) / base
	}
	return out
}
