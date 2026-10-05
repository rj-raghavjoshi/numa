package ta

import (
	"math"

	"github.com/rj-raghavjoshi/numa/vec"
)

// ---------------------------------------------------------------------------
// Long-period oscillators and pivot patterns.
//
// These are the indicators that combine several primitives into something whose shape
// matters more than any single term: a weighted sum of smoothed rates of change, a
// double-smoothed momentum ratio, and a pair of shifted averages that form a "mouth". They
// are grouped because each one is a composition, and the composition is where an
// indicator's warm-up and NaN behaviour are actually decided.
// ---------------------------------------------------------------------------

// CoppockCurve returns the Coppock Curve:
//
//	WMA(ROC(close, rocLong) + ROC(close, rocShort), wmaPeriod)
//
// It was designed as a long-term buy signal: the two rates of change capture momentum at
// different horizons and the weighted average smooths the sum enough that a monthly chart
// produces only a few crossings per decade. TradingView's defaults are 14, 11 and 10.
//
// The rates of change are percentages, so their sum is defined only where both are; a
// window containing a NaN rate propagates it. The first
// max(rocLong, rocShort) + wmaPeriod - 2 values are NaN.
func CoppockCurve(close []float64, rocLong, rocShort, wmaPeriod int) []float64 {
	checkPeriod("CoppockCurve rocLong", rocLong)
	checkPeriod("CoppockCurve rocShort", rocShort)
	checkPeriod("CoppockCurve wmaPeriod", wmaPeriod)
	if len(close) == 0 {
		return nil
	}

	long := ROC(close, rocLong)
	short := ROC(close, rocShort)
	sum := make([]float64, len(close))
	for i := range close {
		switch {
		case long[i] != long[i] || short[i] != short[i]:
			sum[i] = math.NaN()
		default:
			sum[i] = long[i] + short[i]
		}
	}
	return WMA(sum, wmaPeriod)
}

// KnowSureThing returns the Know Sure Thing oscillator and its signal line:
//
//	KST = 1*SMA(ROC(close, roc[0]), sma[0])
//	    + 2*SMA(ROC(close, roc[1]), sma[1])
//	    + 3*SMA(ROC(close, roc[2]), sma[2])
//	    + 4*SMA(ROC(close, roc[3]), sma[3])
//	signal = SMA(KST, signalLength)
//
// Four smoothed rates of change at increasing horizons are weighted 1 through 4, so the
// longest horizon dominates. That weighting is the whole idea: it makes the oscillator
// agree with the primary trend while the shorter terms make it turn early.
//
// The periods are taken as fixed-size arrays rather than eight positional ints, because
// eight ints in a row is exactly the signature where a transposition cannot be seen.
// TradingView's defaults are roc = {10, 15, 20, 30}, sma = {10, 10, 10, 15}, signal = 9.
//
// Both outputs have the same length as close, with NaN until every term is defined.
func KnowSureThing(close []float64, roc, sma [4]int, signalLength int) (kst, signal []float64) {
	for _, v := range roc {
		checkPeriod("KnowSureThing roc", v)
	}
	for _, v := range sma {
		checkPeriod("KnowSureThing sma", v)
	}
	checkPeriod("KnowSureThing signalLength", signalLength)
	if len(close) == 0 {
		return nil, nil
	}

	size := len(close)
	kst = allNaN(size)
	for term := 0; term < 4; term++ {
		weight := float64(term + 1)
		smoothed := SMA(ROC(close, roc[term]), sma[term])
		for i := 0; i < size; i++ {
			switch {
			case smoothed[i] != smoothed[i]:
				kst[i] = math.NaN()
			case kst[i] != kst[i]:
				// Already poisoned by an earlier term; leave it.
			default:
				kst[i] += weight * smoothed[i]
			}
		}
	}
	signal = SMA(kst, signalLength)
	return kst, signal
}

// TrueStrengthIndex returns the n-period double-smoothed momentum ratio, in [-100,100]:
//
//	pc       = close - close[1]
//	smooth   = EMA(EMA(pc, long), short)
//	absSmooth= EMA(EMA(|pc|, long), short)
//	TSI      = 100 * smooth / absSmooth
//
// Smoothing the momentum twice is what makes it usable: a single smoothed rate of change is
// too noisy to read, and the ratio of smoothed momentum to smoothed absolute momentum keeps
// the result bounded without a separate scaling. TradingView's defaults are long = 25,
// short = 13.
//
// The first difference is undefined at index 0, so both smoothing chains start at index 1
// rather than consuming that NaN as seed material. A zero denominator yields 0.
//
// The first long+short-2 values are NaN.
func TrueStrengthIndex(close []float64, long, short int) []float64 {
	checkPeriod("TrueStrengthIndex long", long)
	checkPeriod("TrueStrengthIndex short", short)
	if len(close) == 0 {
		return nil
	}

	size := len(close)
	pc := make([]float64, size)
	absPC := make([]float64, size)
	fillNaN(pc, 0, 1)
	fillNaN(absPC, 0, 1)
	for i := 1; i < size; i++ {
		d := close[i] - close[i-1]
		pc[i] = d
		if d != d {
			absPC[i] = math.NaN()
			continue
		}
		absPC[i] = math.Abs(d)
	}

	// Both chains are the same composition, so their warm-ups are identical.
	smooth := applyEMA(pc, 1, long)
	smooth = applyEMA(smooth, FirstValid(smooth), short)
	absSmooth := applyEMA(absPC, 1, long)
	absSmooth = applyEMA(absSmooth, FirstValid(absSmooth), short)

	out := allNaN(size)
	for i := range close {
		switch {
		case smooth[i] != smooth[i] || absSmooth[i] != absSmooth[i]:
			out[i] = math.NaN()
		case absSmooth[i] == 0:
			out[i] = 0
		default:
			out[i] = 100 * smooth[i] / absSmooth[i]
		}
	}
	return out
}

// WilliamsAlligator returns the three Alligator lines -- jaw, teeth and lips -- as the
// shifted smoothed averages of the median price:
//
//	median = (high + low) / 2
//	jaw    = shift(RMA(median, jawPeriod), jawShift)
//	teeth  = shift(RMA(median, teethPeriod), teethShift)
//	lips   = shift(RMA(median, lipsPeriod), lipsShift)
//
// The shifts move each line *forward in time*, so the value reported at bar i is the average
// as it stood several bars earlier. That lag is deliberate: the three lines then diverge and
// converge in a way that reads as a jaw opening and closing, and the ordering of the lines
// is the signal. TradingView's defaults are 13/8, 8/5 and 5/3.
//
// A forward shift makes the first shift bars NaN, on top of each smoother's own warm-up.
// The shifts must be non-negative; a negative shift would look ahead and is rejected.
func WilliamsAlligator(high, low []float64, jawPeriod, jawShift, teethPeriod, teethShift, lipsPeriod, lipsShift int) (jaw, teeth, lips []float64) {
	checkPeriod("WilliamsAlligator jawPeriod", jawPeriod)
	checkPeriod("WilliamsAlligator teethPeriod", teethPeriod)
	checkPeriod("WilliamsAlligator lipsPeriod", lipsPeriod)
	if jawShift < 0 || teethShift < 0 || lipsShift < 0 {
		panic("ta: WilliamsAlligator shifts must be >= 0")
	}
	size := requireSameLen("WilliamsAlligator", high, low)
	if size == 0 {
		return nil, nil, nil
	}

	median := MedianPrice(high, low)
	jaw = vec.Shift(RMA(median, jawPeriod), jawShift)
	teeth = vec.Shift(RMA(median, teethPeriod), teethShift)
	lips = vec.Shift(RMA(median, lipsPeriod), lipsShift)
	return jaw, teeth, lips
}

// WilliamsFractal returns masks for the up and down fractals over a window of 2n+1 bars.
//
// An up fractal marks a bar whose high is strictly higher than the n bars on each side of
// it; a down fractal is the mirror for the low. TradingView's default is n = 2, the
// five-bar fractal.
//
// # The signal is reported at the bar that confirms it
//
// A fractal cannot be known at the bar it describes -- it needs n bars of hindsight -- so
// the mask is set at index i+n, the first bar at which the pattern is knowable, rather than
// at index i. Plotting it at i, as a chart does, would place a signal n bars before the
// information existed, and anything reading that mask causally would be trading on the
// future. This is a deliberate divergence from the way the indicator is drawn, and it is the
// reason the output is a mask rather than a price level.
//
// A tie disqualifies a fractal: an equal high on either side means the bar is not a distinct
// extreme, so the comparison is strict. A NaN in the window produces no signal, because no
// comparison against it can be satisfied.
//
// The first n and last n positions of the output are 0, since neither can be confirmed.
func WilliamsFractal(high, low []float64, n int) (up, down []uint8) {
	checkPeriod("WilliamsFractal", n)
	size := requireSameLen("WilliamsFractal", high, low)
	if size == 0 {
		return nil, nil
	}

	up = make([]uint8, size)
	down = make([]uint8, size)
	if 2*n+1 > size {
		return up, down
	}

	for i := n; i+n < size; i++ {
		isUp, isDown := true, true
		h, l := high[i], low[i]
		if h != h || l != l {
			continue
		}
		for j := i - n; j <= i+n; j++ {
			if j == i {
				continue
			}
			if high[j] != high[j] || high[j] >= h {
				isUp = false
			}
			if low[j] != low[j] || low[j] <= l {
				isDown = false
			}
		}
		if isUp {
			up[i+n] = 1
		}
		if isDown {
			down[i+n] = 1
		}
	}
	return up, down
}
