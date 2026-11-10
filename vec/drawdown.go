package vec

import "math"

// ---------------------------------------------------------------------------
// Drawdown curves and peak-to-trough summaries.
//
// Drawdown measures the fractional decline of a strictly positive price or NAV
// series from its running peak. It is a scan (each output depends on the
// running peak up to i) and therefore shares the forward NaN-poisoning policy
// of [CumMax].
// ---------------------------------------------------------------------------

// Drawdown returns the underwater drawdown curve of xs at each index:
// (xs[i] - runningPeak) / runningPeak, where runningPeak = max(xs[0..i]).
//
// Values are non-positive (0 at a new peak, -0.25 for a 25% decline). Once a
// NaN or a non-positive running peak is encountered, every subsequent output is
// NaN (matching [CumMax] and refusing to divide by zero/negative peaks).
//
// Returns nil for an empty slice.
func Drawdown(xs []float64) []float64 {
	if len(xs) == 0 {
		return nil
	}
	return DrawdownTo(make([]float64, len(xs)), xs)
}

// DrawdownTo stores the underwater drawdown curve of xs into dst and returns dst.
//
// dst and xs must be the same length; DrawdownTo panics otherwise. dst may
// alias xs because each element xs[i] is read before dst[i] is written.
func DrawdownTo(dst, xs []float64) []float64 {
	checkSameLen1("DrawdownTo", dst, xs)
	peak := math.Inf(-1)
	poisoned := false
	for i, v := range xs {
		if poisoned || v != v {
			poisoned = true
			dst[i] = math.NaN()
			continue
		}
		if v > peak {
			peak = v
		}
		if peak <= 0 {
			dst[i] = math.NaN()
			continue
		}
		dst[i] = (v - peak) / peak
	}
	return dst
}

// MaxDrawdown returns the worst peak-to-trough fractional decline of xs in a
// single pass, along with the 0-based peak, trough, and recovery indices.
//
// maxDD is non-negative (e.g. 0.25 for a 25% decline from peak to trough).
// For a monotonically non-decreasing positive series, maxDD is 0 and
// peakIdx = troughIdx = recoveryIdx = -1. If the trough never recovers to or
// above the peak price, recoveryIdx is -1.
//
// Empty inputs, inputs containing NaN at any position, or series with no
// positive peak return (NaN, -1, -1, -1).
func MaxDrawdown(xs []float64) (maxDD float64, peakIdx, troughIdx, recoveryIdx int) {
	if len(xs) == 0 {
		return math.NaN(), -1, -1, -1
	}
	peakIdx, troughIdx, recoveryIdx = -1, -1, -1

	// NaN policy: any NaN anywhere makes the summary undefined, matching Min/Max.
	if HasNaN(xs) {
		return math.NaN(), -1, -1, -1
	}

	curPeakIdx := -1
	for i, v := range xs {
		if curPeakIdx < 0 {
			if v > 0 {
				curPeakIdx = i
			}
			continue
		}
		if v > xs[curPeakIdx] {
			curPeakIdx = i
			continue
		}
		peakVal := xs[curPeakIdx]
		if peakVal <= 0 {
			continue
		}
		decline := (peakVal - v) / peakVal
		if decline > maxDD {
			maxDD = decline
			peakIdx = curPeakIdx
			troughIdx = i
		}
	}
	if curPeakIdx < 0 {
		return math.NaN(), -1, -1, -1
	}
	if troughIdx >= 0 && peakIdx >= 0 {
		peakVal := xs[peakIdx]
		for i := troughIdx + 1; i < len(xs); i++ {
			if xs[i] >= peakVal {
				recoveryIdx = i
				break
			}
		}
	}
	return maxDD, peakIdx, troughIdx, recoveryIdx
}
