package ta

import (
	"math"
	"testing"
)

// ---------------------------------------------------------------------------
// Tests for the linear-regression indicators.
//
// A perfect straight line is the decisive test case, because every property of the fit
// is known exactly: the slope is the line's slope, the value at the current bar is the
// line's value, R² is 1, and the standard error is 0. A flat line then tests the
// opposite end, where the fit is exact but R² is undefined and must not be reported as 1.
// ---------------------------------------------------------------------------

func refLinReg(xs []float64, n int) (slope, last, r2, se []float64) {
	size := len(xs)
	slope = allNaN(size)
	last = allNaN(size)
	r2 = allNaN(size)
	se = allNaN(size)
	meanX := float64(n-1) / 2
	for i := n - 1; i < size; i++ {
		base := i - n + 1
		var sumY float64
		for j := 0; j < n; j++ {
			sumY += xs[base+j]
		}
		meanY := sumY / float64(n)
		var sxy, sxx, syy float64
		for j := 0; j < n; j++ {
			dx := float64(j) - meanX
			dy := xs[base+j] - meanY
			sxy += dx * dy
			sxx += dx * dx
			syy += dy * dy
		}
		sl := sxy / sxx
		intercept := meanY - sl*meanX
		slope[i] = sl
		last[i] = intercept + sl*float64(n-1)
		if syy != 0 {
			r2[i] = sxy * sxy / (sxx * syy)
		} else {
			r2[i] = 0
		}
		if n > 2 {
			resid := syy - sl*sxy
			if resid < 0 {
				resid = 0
			}
			se[i] = math.Sqrt(resid / float64(n-2))
		} else {
			se[i] = 0
		}
	}
	return
}

func TestLinearRegressionMatchesReference(t *testing.T) {
	for _, size := range []int{40, 120, 300} {
		xs := synthClose(size, int64(size)+2020)
		for _, n := range []int{2, 5, 14, 20, 50} {
			ws, wl, wr, we := refLinReg(xs, n)

			gs := LinearRegressionSlope(xs, n)
			gl := LinearRegression(xs, n)
			gr := R2(xs, n)
			ge := StandardError(xs, n)

			assertSeries(t, "LinRegSlope", gs, ws, false)
			assertSeries(t, "LinearRegression", gl, wl, false)
			assertSeries(t, "R2", gr, wr, false)
			assertSeries(t, "StandardError", ge, we, false)
		}
	}
}

// TestLinearRegressionOnAStraightLine checks the case where every property of the fit is
// known in closed form.
func TestLinearRegressionOnAStraightLine(t *testing.T) {
	const size = 60
	const slope, intercept = 2.5, 7.0
	xs := make([]float64, size)
	for i := range xs {
		xs[i] = intercept + slope*float64(i)
	}

	const n = 20
	gotSlope := LinearRegressionSlope(xs, n)
	gotLine := LinearRegression(xs, n)
	gotR2 := R2(xs, n)
	gotSE := StandardError(xs, n)

	for i := n - 1; i < size; i++ {
		if math.Abs(gotSlope[i]-slope) > 1e-9 {
			t.Fatalf("slope[%d] = %v, want %v", i, gotSlope[i], slope)
		}
		if math.Abs(gotLine[i]-xs[i]) > 1e-9 {
			t.Fatalf("regression[%d] = %v, want the current value %v", i, gotLine[i], xs[i])
		}
		if math.Abs(gotR2[i]-1) > 1e-9 {
			t.Fatalf("R²[%d] = %v, want 1 (an exact fit)", i, gotR2[i])
		}
		if math.Abs(gotSE[i]) > 1e-9 {
			t.Fatalf("standard error[%d] = %v, want 0 (all residuals zero)", i, gotSE[i])
		}
	}
}

// TestLinearRegressionOnAConstantSeries pins the degenerate case: the fit is exact but
// there is no variation, so R² is reported as 0 rather than as 1 or NaN.
func TestLinearRegressionOnAConstantSeries(t *testing.T) {
	const size = 50
	const c = 12.5
	xs := make([]float64, size)
	for i := range xs {
		xs[i] = c
	}

	const n = 10
	if got := LinearRegressionSlope(xs, n)[size-1]; got != 0 {
		t.Errorf("slope on a constant series = %v, want 0", got)
	}
	if got := LinearRegression(xs, n)[size-1]; got != c {
		t.Errorf("regression on a constant series = %v, want %v", got, c)
	}
	if got := R2(xs, n)[size-1]; got != 0 {
		t.Errorf("R² on a constant series = %v, want 0", got)
	}
	if got := StandardError(xs, n)[size-1]; got != 0 {
		t.Errorf("standard error on a constant series = %v, want 0", got)
	}
}

// TestRegressionInterceptAndEndpointDifferBySlope checks the internal consistency of the
// two endpoints, which any misalignment of x would break.
func TestRegressionInterceptAndEndpointDifferBySlope(t *testing.T) {
	const n = 14
	xs := synthClose(200, 3131)

	first := LinearRegressionIntercept(xs, n)
	last := LinearRegression(xs, n)
	slope := LinearRegressionSlope(xs, n)

	for i := n - 1; i < len(xs); i++ {
		want := slope[i] * float64(n-1)
		if !closeOrNaN(last[i]-first[i], want) {
			t.Fatalf("i=%d: last-first = %v, want slope*(n-1) = %v", i, last[i]-first[i], want)
		}
	}
}

func TestStandardErrorBandsRelation(t *testing.T) {
	const n = 20
	const k = 2.0
	xs := synthClose(200, 4141)

	upper, middle, lower := StandardErrorBands(xs, n, k)
	line := LinearRegression(xs, n)
	se := StandardError(xs, n)

	for i := range xs {
		if !closeOrNaN(middle[i], line[i]) {
			t.Fatalf("middle[%d] = %v, want LinearRegression = %v", i, middle[i], line[i])
		}
		if math.IsNaN(upper[i]) {
			continue
		}
		if !closeOrNaN(upper[i]-middle[i], k*se[i]) {
			t.Fatalf("upper gap[%d] = %v, want %v", i, upper[i]-middle[i], k*se[i])
		}
		if !closeOrNaN(middle[i]-lower[i], k*se[i]) {
			t.Fatalf("lower gap[%d] = %v, want %v", i, middle[i]-lower[i], k*se[i])
		}
	}
}

// TestStandardErrorTwoPointsIsZero pins the n = 2 case, where two points determine the
// line exactly and the degrees of freedom would be zero.
func TestStandardErrorTwoPointsIsZero(t *testing.T) {
	xs := []float64{1, 5, 2, 9, 4}
	got := StandardError(xs, 2)
	for i := 1; i < len(xs); i++ {
		if got[i] != 0 {
			t.Errorf("StandardError with n=2 at %d = %v, want 0", i, got[i])
		}
	}
}

func TestRegressionWarmupAndPanics(t *testing.T) {
	const size = 60
	xs := synthClose(size, 5151)
	for _, n := range []int{3, 10, 20} {
		for name, series := range map[string][]float64{
			"LinearRegression":          LinearRegression(xs, n),
			"LinearRegressionSlope":     LinearRegressionSlope(xs, n),
			"LinearRegressionIntercept": LinearRegressionIntercept(xs, n),
			"R2":                        R2(xs, n),
			"StandardError":             StandardError(xs, n),
		} {
			if got := FirstValid(series); got != n-1 {
				t.Errorf("%s(n=%d): first valid = %d, want %d", name, n, got, n-1)
			}
		}
	}

	u, m, l := StandardErrorBands(nil, 5, 2)
	if u != nil || m != nil || l != nil {
		t.Error("empty StandardErrorBands should return nils")
	}
	if LinearRegression(nil, 5) != nil || StandardError(nil, 5) != nil {
		t.Error("empty regression inputs should return nil")
	}

	for _, n := range []int{0, 1, -1} {
		func() {
			defer func() {
				if recover() == nil {
					t.Errorf("LinearRegression with n=%d did not panic", n)
				}
			}()
			LinearRegression(xs, n)
		}()
	}
}

// TestLinearRegressionPrecisionAcrossOffsets is the precision contract for the O(1) form.
//
// The rolling implementation computes the slope from `W - meanX*S`, a difference of two quantities of
// order n²*|y|, so it loses precision as the series carries a larger offset. The measured loss, against
// the windowed centred fit, is:
//
//	offset     slope relative error   line relative error
//	0          1.5e-14                 4.7e-16
//	1e2        3.0e-14                 5.2e-16
//	1e4        2.2e-12                 9.0e-16
//	1e6        1.8e-10                 8.1e-16
//	1e8        1.5e-08                 6.0e-16
//	1e10       1.7e-06                 7.6e-16
//
// Two things are worth reading off that table. The *slope* degrades with the offset, roughly as its
// square: at a price scale of 10^4 the slope is good to about eleven significant figures, which is far
// beyond what any indicator needs. The *line* does not degrade at all, because it is dominated by the
// mean and the slope's error enters multiplied by half the window.
//
// The bounds below carry roughly an order of magnitude of headroom over the measurements, so they
// catch a regression rather than pinning the last bit.
func TestLinearRegressionPrecisionAcrossOffsets(t *testing.T) {
	const size, n = 400, 20

	// The offsets a real series plausibly carries: a price, a price in cents, a price in a subdivided
	// unit. 1e10 is beyond any market and is included to show where the slope stops being usable.
	limits := []struct {
		offset     float64
		slopeRel   float64
		lineAbsRel float64
	}{
		{0, 1e-12, 1e-13},
		{1e2, 1e-12, 1e-13},
		{1e4, 1e-10, 1e-13},
		{1e6, 1e-8, 1e-13},
		{1e8, 1e-6, 1e-13},
		{1e10, 1e-4, 1e-13},
	}

	for _, lim := range limits {
		xs := make([]float64, size)
		for i := range xs {
			xs[i] = lim.offset + 0.3*float64(i) + 2*math.Sin(float64(i)*0.7)
		}
		gotSlope := LinearRegressionSlope(xs, n)
		gotLine := LinearRegression(xs, n)

		var worstSlope, worstLine, scaleSlope, scaleLine float64
		for i := n - 1; i < size; i++ {
			wantSlope, _, wantLast, _, _, ok := linregWindow(xs, i-n+1, n)
			if !ok {
				continue
			}
			if d := math.Abs(gotSlope[i] - wantSlope); d > worstSlope {
				worstSlope = d
			}
			if d := math.Abs(gotLine[i] - wantLast); d > worstLine {
				worstLine = d
			}
			scaleSlope = math.Max(scaleSlope, math.Abs(wantSlope))
			scaleLine = math.Max(scaleLine, math.Abs(wantLast))
		}
		if scaleSlope == 0 {
			scaleSlope = 1
		}
		if scaleLine == 0 {
			scaleLine = 1
		}
		if rel := worstSlope / scaleSlope; rel > lim.slopeRel {
			t.Errorf("offset %.0e: slope relative error %.3e exceeds %.0e", lim.offset, rel, lim.slopeRel)
		}
		// The line's error is measured against its own magnitude, which is what a caller compares.
		if rel := worstLine / scaleLine; rel > lim.lineAbsRel {
			t.Errorf("offset %.0e: line relative error %.3e exceeds %.0e", lim.offset, rel, lim.lineAbsRel)
		}
	}
}

// TestLinearRegressionEndpointsAgreeWithTheWindowedFit checks that all three O(1) outputs match the
// windowed fit to the same precision, so the split between the two implementations is invisible to a
// caller who does not read the precision table.
func TestLinearRegressionEndpointsAgreeWithTheWindowedFit(t *testing.T) {
	xs := synthClose(300, 41241)
	for _, n := range []int{2, 5, 20, 100} {
		slope := LinearRegressionSlope(xs, n)
		first := LinearRegressionIntercept(xs, n)
		last := LinearRegression(xs, n)
		for i := n - 1; i < len(xs); i++ {
			ws, wf, wl, _, _, ok := linregWindow(xs, i-n+1, n)
			if !ok {
				continue
			}
			if math.Abs(slope[i]-ws) > 1e-8*math.Max(1, math.Abs(ws)) {
				t.Fatalf("n=%d: slope[%d] = %v, want %v", n, i, slope[i], ws)
			}
			if math.Abs(first[i]-wf) > 1e-8*math.Max(1, math.Abs(wf)) {
				t.Fatalf("n=%d: first[%d] = %v, want %v", n, i, first[i], wf)
			}
			if math.Abs(last[i]-wl) > 1e-8*math.Max(1, math.Abs(wl)) {
				t.Fatalf("n=%d: last[%d] = %v, want %v", n, i, last[i], wl)
			}
		}
	}
}

// TestLinearRegressionRollingRecoversAfterNaN repeats the package-wide NaN policy for the incremental
// form, which is a different code path from the windowed one.
func TestLinearRegressionRollingRecoversAfterNaN(t *testing.T) {
	const n = 5
	xs := synthClose(60, 51251)
	xs[30] = math.NaN()

	line := LinearRegression(xs, n)
	slope := LinearRegressionSlope(xs, n)

	// The NaN sits at index 30, so it is inside the windows ending at 30 through 30+n-1, and the
	// first clean window ends at 30+n.
	const nanAt = 30
	for i := nanAt; i < nanAt+n; i++ {
		if !math.IsNaN(line[i]) || !math.IsNaN(slope[i]) {
			t.Fatalf("index %d should be NaN: the window contains the NaN at %d", i, nanAt)
		}
	}
	recovered := nanAt + n
	if math.IsNaN(line[recovered]) {
		t.Fatalf("index %d is NaN, but its window is %d..%d and holds no NaN", recovered, recovered-n+1, recovered)
	}
	// And the recovered value must match the windowed fit.
	_, _, want, _, _, ok := linregWindow(xs, recovered-n+1, n)
	if !ok {
		t.Fatal("the reference disagrees that the window has recovered")
	}
	if math.Abs(line[recovered]-want) > 1e-8*math.Max(1, math.Abs(want)) {
		t.Fatalf("recovered value %v, want %v", line[recovered], want)
	}
}
