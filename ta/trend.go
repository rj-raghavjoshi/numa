package ta

import (
	"math"

	"github.com/rj-raghavjoshi/numa/series"
)

// ---------------------------------------------------------------------------
// Trend indicators.
//
// Two families live here, and they fail differently.
//
// **Trend strength** (ADX/DMI, Aroon, Vortex, Choppiness) reports how directional the
// market is, in a bounded range, without saying which way. These are ratios of a
// directional movement to a total movement, so every one of them has a
// divide-by-zero case on a motionless window.
//
// **Trailing stops and bands** (Keltner, SuperTrend, Parabolic SAR) produce an actual
// level that ratchets. These are recursive: the value at bar i depends on the value and
// the internal state at bar i-1, so they cannot be computed from a window and cannot be
// parallelised across bars. They also have an initialisation convention that shifts the
// whole series, which each function documents.
//
// # Warm-up conventions
//
// Wilder's directional family needs two nested smoothings, so its warm-up is roughly
// twice the period. Rather than derive each boundary by hand and risk an off-by-one,
// every function here states its convention and the tests pin the value that follows
// from it.
// ---------------------------------------------------------------------------

// DirectionalMovement returns Wilder's ADX together with the +DI and -DI lines.
//
//	upMove   = high[i] - high[i-1]
//	downMove = low[i-1] - low[i]
//	+DM      = upMove   if upMove > downMove and upMove > 0, else 0
//	-DM      = downMove if downMove > upMove and downMove > 0, else 0
//	ATR      = RMA(TrueRange, n)
//	+DI      = 100 * RMA(+DM, n) / ATR
//	-DI      = 100 * RMA(-DM, n) / ATR
//	DX       = 100 * |+DI - -DI| / (+DI + -DI)
//	ADX      = RMA(DX, n)
//
// # Initialisation convention
//
// The first bar has no previous high or low, so +DM and -DM start at index 1 while the
// true range starts at index 0 with high-low. The two smoothings are therefore offset
// by one bar, and the first output index follows:
//
//	+DI, -DI  index n
//	ADX       index 2n-1
//
// Other platforms differ by a bar or two at the very start of a series because their
// handling of the undefined first bar differs. The convention is stated so that a
// mismatch is diagnosable rather than mysterious; every implementation converges to the
// same values within about 2n bars.
//
// # Division by zero
//
// A zero true range makes the DI lines 0 rather than infinite, and a zero sum of the
// two DI lines makes DX 0. Both cases mean "no directional information", which 0
// expresses and an infinity does not.
func DirectionalMovement(high, low, close []float64, n int) (adx, plusDI, minusDI []float64) {
	checkPeriod("DirectionalMovement", n)
	size := requireSameLen("DirectionalMovement", high, low, close)
	if size == 0 {
		return nil, nil, nil
	}

	plusDM := make([]float64, size)
	minusDM := make([]float64, size)
	fillNaN(plusDM, 0, 1)
	fillNaN(minusDM, 0, 1)
	for i := 1; i < size; i++ {
		up := high[i] - high[i-1]
		down := low[i-1] - low[i]
		switch {
		case up != up || down != down:
			plusDM[i], minusDM[i] = math.NaN(), math.NaN()
		case up > down && up > 0:
			plusDM[i] = up
		case down > up && down > 0:
			minusDM[i] = down
		}
	}

	atr := ATR(high, low, close, n)
	// The directional moves are undefined at index 0, so their smoothers start at
	// index 1 rather than consuming that NaN as seed material.
	rmaPlus := applyRMA(plusDM, 1, n)
	rmaMinus := applyRMA(minusDM, 1, n)

	plusDI = make([]float64, size)
	minusDI = make([]float64, size)
	dx := allNaN(size)
	for i := 0; i < size; i++ {
		p, m, a := rmaPlus[i], rmaMinus[i], atr[i]
		if p != p || m != m || a != a {
			plusDI[i], minusDI[i] = math.NaN(), math.NaN()
			continue
		}
		if a == 0 {
			plusDI[i], minusDI[i] = 0, 0
		} else {
			plusDI[i] = 100 * p / a
			minusDI[i] = 100 * m / a
		}
		sum := plusDI[i] + minusDI[i]
		if plusDI[i] != plusDI[i] || minusDI[i] != minusDI[i] {
			continue
		}
		if sum == 0 {
			dx[i] = 0
			continue
		}
		dx[i] = 100 * math.Abs(plusDI[i]-minusDI[i]) / sum
	}

	adx = applyRMA(dx, FirstValid(dx), n)
	return adx, plusDI, minusDI
}

// Aroon returns the Aroon up line, down line and oscillator.
//
//	up[i]   = 100 * (n - barsSinceHighestHigh) / n
//	down[i] = 100 * (n - barsSinceLowestLow) / n
//	osc     = up - down
//
// where the "bars since" is measured within the trailing n-bar window, so that a new
// extreme on the current bar gives 100 and one at the start of the window gives 100/n.
// The oscillator lies in [-100, 100].
//
// # Ties
//
// When the extreme value occurs more than once in the window the *most recent*
// occurrence is used, which is the smaller offset and therefore the larger line. That
// is a choice: it makes a fresh retest of a level read as a stronger signal than a stale
// one, and it is what "bars since the high" most naturally means.
//
// # Cost and warm-up
//
// The window is scanned directly, O(n) per bar, rather than maintained with an
// index-tracking deque. The deque version is the obvious follow-up and is recorded in
// docs/benchmarks.md; the scan is kept because it makes the tie rule and the NaN rule
// explicit in the code that defines them.
//
// The first n-1 values of all three outputs are NaN, and a window containing a NaN
// produces NaN.
func Aroon(high, low []float64, n int) (up, down, osc []float64) {
	checkPeriod("Aroon", n)
	size := requireSameLen("Aroon", high, low)
	if size == 0 {
		return nil, nil, nil
	}

	up = allNaN(size)
	down = allNaN(size)
	osc = allNaN(size)
	for i := n - 1; i < size; i++ {
		base := i - n + 1
		hiIdx, loIdx := base, base
		nan := false
		for j := base; j <= i; j++ {
			h, l := high[j], low[j]
			if h != h || l != l {
				nan = true
				break
			}
			// >= and <= make a repeated extreme resolve to the most recent bar.
			if h >= high[hiIdx] {
				hiIdx = j
			}
			if l <= low[loIdx] {
				loIdx = j
			}
		}
		if nan {
			continue
		}
		upBars := i - hiIdx
		downBars := i - loIdx
		up[i] = 100 * float64(n-upBars) / float64(n)
		down[i] = 100 * float64(n-downBars) / float64(n)
		osc[i] = up[i] - down[i]
	}
	return up, down, osc
}

// Vortex returns the two Vortex Indicator lines, VI+ and VI-.
//
//	VI+ = sum(|high[i] - low[i-1]|, n) / sum(TrueRange, n)
//	VI- = sum(|low[i]  - high[i-1]|, n) / sum(TrueRange, n)
//
// A VI+ crossing above VI- is a bullish signal and vice versa. Both lines are
// non-negative and are usually close to 1, so the crossover matters more than the level.
//
// The sums need a previous bar, so the window starts at index 1 and the first n values
// are NaN. A zero true-range sum yields 0 for both lines.
func Vortex(high, low, close []float64, n int) (viPlus, viMinus []float64) {
	checkPeriod("Vortex", n)
	size := requireSameLen("Vortex", high, low, close)
	if size == 0 {
		return nil, nil
	}

	vmPlus := make([]float64, size)
	vmMinus := make([]float64, size)
	tr := TrueRange(high, low, close)
	fillNaN(vmPlus, 0, 1)
	fillNaN(vmMinus, 0, 1)
	for i := 1; i < size; i++ {
		vmPlus[i] = math.Abs(high[i] - low[i-1])
		vmMinus[i] = math.Abs(low[i] - high[i-1])
	}

	plusSum := series.NewSum(n)
	minusSum := series.NewSum(n)
	trSum := series.NewSum(n)
	viPlus = allNaN(size)
	viMinus = allNaN(size)
	for i := 0; i < size; i++ {
		p := plusSum.Push(vmPlus[i])
		m := minusSum.Push(vmMinus[i])
		t := trSum.Push(tr[i])
		if i+1 < n {
			continue
		}
		if p != p || m != m || t != t {
			viPlus[i], viMinus[i] = math.NaN(), math.NaN()
			continue
		}
		if t == 0 {
			viPlus[i], viMinus[i] = 0, 0
			continue
		}
		viPlus[i] = p / t
		viMinus[i] = m / t
	}
	return viPlus, viMinus
}

// Choppiness returns the n-period Choppiness Index, a value in [0,100]:
//
//	100 * log10(sum(TrueRange, n) / (highest(high,n) - lowest(low,n))) / log10(n)
//
// High values mean a ranging, choppy market; low values mean a trending one. It is
// scale-free, so it is comparable across instruments.
//
// A window whose total range is zero yields 0, and n = 1 is rejected by [checkPeriod]
// because log10(1) = 0 would make the index undefined for every window.
//
// # A boundary artifact above 100
//
// The index is usually quoted in [0,100], and it is in [0,100] for gap-free data. It
// can read slightly above 100 when the window's *first* bar gaps from a close outside
// the window: the true-range sum then includes a term larger than the window's own
// range. The value is deliberately not clamped, because clamping would hide a real gap
// and turn a signal into a rounded constant. The convention here matches the usual
// definition; a caller who needs a hard bound should clamp at the call site and say so.
//
// The first n-1 values are NaN.
func Choppiness(high, low, close []float64, n int) []float64 {
	checkPeriod("Choppiness", n)
	size := requireSameLen("Choppiness", high, low, close)
	if size == 0 {
		return nil
	}
	out := allNaN(size)

	tr := TrueRange(high, low, close)
	trSum := series.NewSum(n)
	hi := Highest(high, n)
	lo := Lowest(low, n)
	logN := math.Log10(float64(n))

	for i := 0; i < size; i++ {
		s := trSum.Push(tr[i])
		if i+1 < n {
			continue
		}
		span := hi[i] - lo[i]
		if s != s || span != span {
			continue
		}
		if span <= 0 {
			out[i] = 0
			continue
		}
		out[i] = 100 * math.Log10(s/span) / logN
	}
	return out
}

// KeltnerChannels returns the Keltner channel around close:
//
//	middle = EMA(close, n)
//	atr    = ATR(high, low, close, atrN)
//	upper  = middle + mult*atr
//	lower  = middle - mult*atr
//
// The ATR period is separate from the average period because that is how the indicator
// is specified; passing the same value for both is the common case, not the definition.
// The first max(n, atrN)-1 values are NaN.
func KeltnerChannels(high, low, close []float64, n, atrN int, mult float64) (upper, middle, lower []float64) {
	checkPeriod("KeltnerChannels", n)
	checkPeriod("KeltnerChannels atrN", atrN)
	size := requireSameLen("KeltnerChannels", high, low, close)
	if size == 0 {
		return nil, nil, nil
	}

	middle = EMA(close, n)
	atr := ATR(high, low, close, atrN)
	upper = make([]float64, size)
	lower = make([]float64, size)
	for i := 0; i < size; i++ {
		upper[i] = middle[i] + mult*atr[i]
		lower[i] = middle[i] - mult*atr[i]
	}
	return upper, middle, lower
}

// SuperTrend returns the SuperTrend line and its direction, +1 when the trend is up and
// -1 when it is down.
//
//	hl2        = (high + low) / 2
//	atr        = ATR(high, low, close, atrN)
//	basicUpper = hl2 + mult*atr
//	basicLower = hl2 - mult*atr
//
// The final bands ratchet: the upper band only moves down while price stays below it,
// and the lower band only moves up while price stays above it. The trend flips when the
// close crosses the opposite band, at which point the line jumps to the other band.
//
// # Initialisation
//
// The first bar with an ATR sets both final bands to their basic values and starts the
// trend up, which is the usual convention. Because the state then depends on that
// choice, the first few bars of a series can differ from an implementation that starts
// down; the two converge once the first genuine flip occurs.
//
// Before the ATR is available the output is NaN.
func SuperTrend(high, low, close []float64, atrN int, mult float64) (line, direction []float64) {
	checkPeriod("SuperTrend atrN", atrN)
	size := requireSameLen("SuperTrend", high, low, close)
	if size == 0 {
		return nil, nil
	}

	atr := ATR(high, low, close, atrN)
	line = allNaN(size)
	direction = allNaN(size)

	start := FirstValid(atr)
	if start < 0 {
		return line, direction
	}

	// Seed from the first bar that has an ATR: both bands start at their basic
	// values and the trend starts up.
	hl2 := (high[start] + low[start]) / 2
	finalUpper := hl2 + mult*atr[start]
	finalLower := hl2 - mult*atr[start]
	trendUp := true
	line[start] = finalLower
	direction[start] = 1

	for i := start + 1; i < size; i++ {
		hl2 := (high[i] + low[i]) / 2
		basicUpper := hl2 + mult*atr[i]
		basicLower := hl2 - mult*atr[i]

		// Ratchet: the bands may only move in the direction that tightens the stop.
		if basicUpper < finalUpper || close[i-1] > finalUpper {
			finalUpper = basicUpper
		}
		if basicLower > finalLower || close[i-1] < finalLower {
			finalLower = basicLower
		}

		switch {
		case trendUp && close[i] < finalLower:
			trendUp = false
		case !trendUp && close[i] > finalUpper:
			trendUp = true
		}

		if trendUp {
			line[i] = finalLower
			direction[i] = 1
		} else {
			line[i] = finalUpper
			direction[i] = -1
		}
	}
	return line, direction
}

// ParabolicSAR returns Wilder's Parabolic SAR and its direction, +1 when the trend is
// up and -1 when it is down.
//
//	SAR[i] = SAR[i-1] + af * (EP - SAR[i-1])
//
// where EP is the extreme point reached in the current trend and af is the acceleration
// factor, starting at start and increasing by increment on each new extreme up to a cap
// of max. A trend reverses when price penetrates the SAR, at which point the SAR is set
// to the extreme point and the acceleration factor resets.
//
// # The penetration clamp
//
// After the update, the SAR is clamped so that it cannot enter the range of the previous
// two bars:
//
//	uptrend:   SAR = min(SAR, low[i-1],  low[i-2])
//	downtrend: SAR = max(SAR, high[i-1], high[i-2])
//
// Without this clamp the indicator can place a stop inside recent price action, which
// makes it fire spuriously. It is part of Wilder's definition, not a refinement.
//
// # Initialisation
//
// The first bar starts an uptrend with the SAR at its low and the extreme point at its
// high. That is a convention; a series that opens falling will flip within a bar or two.
//
// # The reversal bar can contain its own SAR
//
// On the bar where the trend changes, the SAR is set to the extreme point of the trend
// that just ended. A wide reversal bar can therefore have its SAR *inside* its own range,
// so the usual invariant -- an uptrend's SAR below the low, a downtrend's above the high
// -- holds on every bar except that one. This is a property of Wilder's definition rather
// than an error, and it is why a caller should read the SAR as a level to be crossed
// rather than as a guaranteed stop on the bar it appears.
//
// The output is defined from index 0, so there is no warm-up. Both returned series are
// the same length as the input.
func ParabolicSAR(high, low []float64, start, increment, max float64) (sar, direction []float64) {
	size := requireSameLen("ParabolicSAR", high, low)
	if size == 0 {
		return nil, nil
	}
	if increment <= 0 {
		panic("ta: ParabolicSAR increment must be > 0")
	}
	if max < start {
		panic("ta: ParabolicSAR max must be >= start")
	}

	sar = make([]float64, size)
	direction = make([]float64, size)

	up := true
	af := start
	ep := high[0]
	sar[0] = low[0]
	direction[0] = 1

	for i := 1; i < size; i++ {
		prev := sar[i-1]
		cur := prev + af*(ep-prev)

		if up {
			if low[i] < cur {
				// Reversal: the new SAR is the extreme point of the old trend.
				up = false
				cur = ep
				ep = low[i]
				af = start
			} else if high[i] > ep {
				ep = high[i]
				af += increment
				if af > max {
					af = max
				}
			}
		} else {
			if high[i] > cur {
				up = true
				cur = ep
				ep = high[i]
				af = start
			} else if low[i] < ep {
				ep = low[i]
				af += increment
				if af > max {
					af = max
				}
			}
		}

		// Clamp so the SAR cannot sit inside the last two bars' range.
		if up {
			if i >= 1 && low[i-1] < cur {
				cur = low[i-1]
			}
			if i >= 2 && low[i-2] < cur {
				cur = low[i-2]
			}
		} else {
			if i >= 1 && high[i-1] > cur {
				cur = high[i-1]
			}
			if i >= 2 && high[i-2] > cur {
				cur = high[i-2]
			}
		}

		sar[i] = cur
		if up {
			direction[i] = 1
		} else {
			direction[i] = -1
		}
	}
	return sar, direction
}
