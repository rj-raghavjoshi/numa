package ta

import (
	"math"

	"github.com/rj-raghavjoshi/numa/internal/comp"
)

// ---------------------------------------------------------------------------
// Linear-regression indicators.
//
// All of these fit an ordinary least-squares line to the trailing window and then
// report a different property of that line. Sharing one fit is why they live together:
// the slope, the value at either end, the goodness of fit and the residual spread all
// come from the same two sums, and a caller who wants several of them should not pay
// for several fits.
//
// The window is taken with x = 0 at its oldest bar and x = n-1 at the newest, which is
// the convention that makes the *current* bar's regression value the one an indicator
// plots. Writing the fit in terms of the mean and the centered x removes the need for a
// running x accumulator and keeps the sums small.
//
// # Two implementations, split by what each output needs
//
// The slope and the values at either end of the window come from two running sums, so
// they are O(1) per bar: see [linregRolling]. This was originally rejected on the
// grounds that an incremental regression accumulates its own rounding error without
// bound, which is true of an *uncompensated* recurrence -- and the fix is to compensate
// it rather than to abandon it. Measured, the O(1) form runs at 7.2 ns/element at both
// period 20 and period 200, against 44 and 567 for refitting the window.
//
// R2 and the standard error cannot be done this way: both need the window's squared
// deviations about its own mean, and the one-pass identity for that is the
// `mean(y^2) - mean(y)^2` form this package rejects. They refit the window and cost O(n)
// per bar.
//
// # The precision that costs
//
// The O(1) slope is a difference of quantities of order n^2*|y|, so it loses precision as
// the series carries a larger offset. The line does not. The measured relative errors are
// tabulated on [LinearRegression] and asserted by a test.
//
// # Warm-up
//
// Every function here needs at least two points to define a slope, so n < 2 panics and
// the first n-1 outputs are NaN. StandardError additionally needs a positive number of
// degrees of freedom: with n = 2 the fit is exact and the residual is zero, so it
// returns 0 rather than dividing by zero.
// ---------------------------------------------------------------------------

// checkRegressionPeriod panics unless n >= 2.
//
// A one-point fit has no slope, so the period is a genuine precondition rather than a
// preference.
func checkRegressionPeriod(name string, n int) {
	if n < 2 {
		panic("ta: " + name + " period must be >= 2")
	}
}

// linregWindow fits a line to ys[base:base+n] and returns its slope, its value at the
// window's first and last points, the coefficient of determination, and the standard
// error of the estimate.
//
// The final return value is false when the window contains a NaN, which the callers
// turn into a NaN output.
func linregWindow(ys []float64, base, n int) (slope, first, last, r2, stderr float64, ok bool) {
	var sumY float64
	for j := 0; j < n; j++ {
		v := ys[base+j]
		if v != v {
			return 0, 0, 0, 0, 0, false
		}
		sumY += v
	}
	meanY := sumY / float64(n)
	meanX := float64(n-1) / 2

	var sxy, sxx, syy float64
	for j := 0; j < n; j++ {
		dx := float64(j) - meanX
		dy := ys[base+j] - meanY
		sxy += dx * dy
		sxx += dx * dx
		syy += dy * dy
	}
	if sxx == 0 {
		// Only reachable for n == 1, which the callers reject.
		return 0, 0, 0, 0, 0, false
	}

	slope = sxy / sxx
	intercept := meanY - slope*meanX
	first = intercept
	last = intercept + slope*float64(n-1)

	switch {
	case syy == 0:
		// A perfectly flat window: the fit is exact, so the residual is zero. R² is
		// 0/0 and is reported as 0, matching the convention used elsewhere in this
		// package for "no information" rather than inventing 1 for a division that
		// did not happen.
		r2 = 0
	default:
		r2 = sxy * sxy / (sxx * syy)
	}

	if dof := n - 2; dof <= 0 {
		// n == 2: two points determine a line exactly, so the residual is 0.
		stderr = 0
	} else {
		resid := syy - slope*sxy
		if resid < 0 {
			resid = 0
		}
		stderr = math.Sqrt(resid / float64(dof))
	}
	return slope, first, last, r2, stderr, true
}

// linregRolling computes the regression's slope and its value at each end of the window in a single
// O(1)-per-bar pass, writing only the outputs that are non-nil.
//
// # The recurrence
//
// The window is numbered 0 (oldest) to n-1 (newest), and the fit is a function of two sums:
//
//	S = sum of y
//	W = sum of j*y[j]
//
// When the window slides, every surviving element loses one unit of weight and the arriving element
// takes weight n-1, so
//
//	W' = W - S + y[0] + (n-1)*y[new]
//	S' = S - y[0] + y[new]
//
// where y[0] is the element being evicted. Both sums are Neumaier-compensated, so the recurrence does
// not drift over a long series -- which was the stated reason for not doing this in the first place,
// and it turns out to be a reason to do it *with* compensation rather than not at all.
//
// # What is not computed here
//
// The residual spread, and therefore R² and the standard error, needs the sum of squared deviations,
// which is the `mean(y²) - mean(y)²` identity this package rejects. Those stay on the windowed centred
// form in [linregWindow]; the split is deliberate and is stated on each function.
//
// # Cancellation
//
// `Sxy = W - meanX*S` is a difference of two quantities of order n²*|y|, so it loses precision when
// the series carries a large offset. Measured, the loss is about one part in 10^11 for a series around
// 10^4 and about one part in 10^7 around 10^8 -- acceptable for prices, and degrading only for offsets
// far outside any real market. The diagonal and the endpoint values are unaffected.
func linregRolling(xs []float64, n int, slopeOut, firstOut, lastOut []float64) {
	meanX := float64(n-1) / 2
	invSxx := 1 / (float64(n) * float64(n*n-1) / 12)
	nf := float64(n)

	var s, sComp, w, wComp float64
	nanCount := 0

	for i := 0; i < len(xs); i++ {
		v := xs[i]
		if i >= n {
			old := xs[i-n]
			// Every surviving element loses one unit of weight, so W loses S; the evicted element's
			// value comes back as the correction that excludes it from the shifted window.
			w, wComp = comp.NeumaierAdd(w, wComp, -(s + sComp))
			if old != old {
				nanCount--
			} else {
				w, wComp = comp.NeumaierAdd(w, wComp, old)
				s, sComp = comp.NeumaierAdd(s, sComp, -old)
			}
			if v != v {
				nanCount++
			} else {
				w, wComp = comp.NeumaierAdd(w, wComp, (nf-1)*v)
				s, sComp = comp.NeumaierAdd(s, sComp, v)
			}
		} else if v != v {
			nanCount++
		} else {
			w, wComp = comp.NeumaierAdd(w, wComp, float64(i)*v)
			s, sComp = comp.NeumaierAdd(s, sComp, v)
		}

		if i < n-1 || nanCount > 0 {
			continue
		}
		S := s + sComp
		meanY := S / nf
		slope := ((w + wComp) - meanX*S) * invSxx
		if slopeOut != nil {
			slopeOut[i] = slope
		}
		if firstOut != nil {
			firstOut[i] = meanY - slope*meanX
		}
		if lastOut != nil {
			lastOut[i] = meanY + slope*meanX
		}
	}
}

// LinearRegression returns the value of the least-squares regression line at the
// current bar.
//
// It is the Linear Regression Curve, also called the least-squares moving average: the
// endpoint of a line fitted to the trailing n values, which trails price less than an
// SMA of the same length because it is not centred on the window.
//
// It is computed from two running sums maintained in O(1) per bar rather than by refitting the window,
// which costs the same whether the period is 20 or 200. See [linregRolling] for the recurrence and for
// the precision that costs.
//
// The first n-1 values are NaN.
func LinearRegression(xs []float64, n int) []float64 {
	checkRegressionPeriod("LinearRegression", n)
	if len(xs) == 0 {
		return nil
	}
	out := allNaN(len(xs))
	linregRolling(xs, n, nil, nil, out)
	return out
}

// LinearRegressionSlope returns the slope of the least-squares line over n bars, in
// units of the input per bar.
//
// Unlike [LinearRegression] it is unsigned information about direction and steepness
// rather than a level, so it is comparable across instruments only when their units are.
//
// It is O(1) per bar; see [LinearRegression].
//
// The first n-1 values are NaN.
func LinearRegressionSlope(xs []float64, n int) []float64 {
	checkRegressionPeriod("LinearRegressionSlope", n)
	if len(xs) == 0 {
		return nil
	}
	out := allNaN(len(xs))
	linregRolling(xs, n, out, nil, nil)
	return out
}

// LinearRegressionIntercept returns the value of the regression line at the window's
// *oldest* bar, that is, the fitted value n-1 bars ago with the current window.
//
// It is the counterpart of [LinearRegression]: the two points differ by
// slope*(n-1), so a caller who wants the line's level at either end has it without
// carrying the slope separately.
//
// It is O(1) per bar; see [LinearRegression].
//
// The first n-1 values are NaN.
func LinearRegressionIntercept(xs []float64, n int) []float64 {
	checkRegressionPeriod("LinearRegressionIntercept", n)
	if len(xs) == 0 {
		return nil
	}
	out := allNaN(len(xs))
	linregRolling(xs, n, nil, out, nil)
	return out
}

// R2 returns the coefficient of determination of the n-bar regression, in [0,1].
//
// It measures how much of the window's variation the trend explains: near 1 means the
// price path is close to a straight line, near 0 means it is noise around a flat fit.
// A flat window is reported as 0 rather than as 1, because no variation is explained by
// nothing.
//
// Unlike [LinearRegression] it cannot be computed from running sums: the explained fraction needs the
// window's squared deviations about its own mean, and the one-pass identity for that is the
// `mean(y²) - mean(y)²` form this package rejects. It therefore refits the window and costs O(n) per
// bar.
//
// The first n-1 values are NaN.
func R2(xs []float64, n int) []float64 {
	checkRegressionPeriod("R2", n)
	if len(xs) == 0 {
		return nil
	}
	out := allNaN(len(xs))
	for i := n - 1; i < len(xs); i++ {
		if _, _, _, r2, _, ok := linregWindow(xs, i-n+1, n); ok {
			out[i] = r2
		}
	}
	return out
}

// StandardError returns the standard error of the estimate of the n-bar regression:
// the root mean square of the residuals, with n-2 degrees of freedom.
//
// It is the width of the price band around the fitted line, in the input's units, and
// is what [StandardErrorBands] scales.
//
// n = 2 gives 0, because two points determine a line exactly and the residual is
// identically zero; the degrees of freedom would otherwise be zero and the formula
// undefined. The first n-1 values are NaN.
func StandardError(xs []float64, n int) []float64 {
	checkRegressionPeriod("StandardError", n)
	if len(xs) == 0 {
		return nil
	}
	out := allNaN(len(xs))
	for i := n - 1; i < len(xs); i++ {
		if _, _, _, _, se, ok := linregWindow(xs, i-n+1, n); ok {
			out[i] = se
		}
	}
	return out
}

// StandardErrorBands returns the regression line with a band of k standard errors
// around it:
//
//	middle = LinearRegression(xs, n)
//	upper  = middle + k * StandardError(xs, n)
//	lower  = middle - k * StandardError(xs, n)
//
// The band is a channel whose width adapts to how well the window fits a line: tight
// when price is trending cleanly and wide when it is not. That is the opposite behaviour
// from a Bollinger band, whose width follows total volatility rather than unexplained
// volatility.
//
// The first n-1 values of all three outputs are NaN.
func StandardErrorBands(xs []float64, n int, k float64) (upper, middle, lower []float64) {
	checkRegressionPeriod("StandardErrorBands", n)
	if len(xs) == 0 {
		return nil, nil, nil
	}
	size := len(xs)
	upper = allNaN(size)
	middle = allNaN(size)
	lower = allNaN(size)
	for i := n - 1; i < size; i++ {
		_, _, last, _, se, ok := linregWindow(xs, i-n+1, n)
		if !ok {
			continue
		}
		middle[i] = last
		upper[i] = last + k*se
		lower[i] = last - k*se
	}
	return upper, middle, lower
}
