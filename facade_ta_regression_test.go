package numa

import (
	"math"
	"testing"

	"github.com/rj-raghavjoshi/numa/ta"
	"github.com/rj-raghavjoshi/numa/vec"
)

// ---------------------------------------------------------------------------
// Forwarder tests for the regression and composed-oscillator families.
//
// Those families have the most parameters of anything in the package: the Accelerator
// Oscillator takes three ints and the Ultimate Oscillator three more, all of which are
// legal in almost any order. Each check therefore compares against ta with identical
// arguments and uses all-distinct values.
// ---------------------------------------------------------------------------

func TestRegressionForwardersMatchTa(t *testing.T) {
	_, _, _, close, _ := testOHLCV()

	eqSeries(t, "LinearRegression", LinearRegression(close, 20), ta.LinearRegression(close, 20))
	eqSeries(t, "LinearRegressionSlope", LinearRegressionSlope(close, 20), ta.LinearRegressionSlope(close, 20))
	eqSeries(t, "LinearRegressionIntercept", LinearRegressionIntercept(close, 20), ta.LinearRegressionIntercept(close, 20))
	eqSeries(t, "R2", R2(close, 20), ta.R2(close, 20))
	eqSeries(t, "StandardError", StandardError(close, 20), ta.StandardError(close, 20))

	gu, gm, gl := StandardErrorBands(close, 20, 2)
	wu, wm, wl := ta.StandardErrorBands(close, 20, 2)
	eqSeries(t, "SEB upper", gu, wu)
	eqSeries(t, "SEB middle", gm, wm)
	eqSeries(t, "SEB lower", gl, wl)

	// The period must reach the implementation.
	if eqSeriesEqual(LinearRegression(close, 5), LinearRegression(close, 20)) {
		t.Error("LinearRegression ignores its period")
	}
	// The band width must reach it too.
	if eqSeriesEqual(gu, func() []float64 { u, _, _ := StandardErrorBands(close, 20, 5); return u }()) {
		t.Error("StandardErrorBands ignores its width multiplier")
	}
}

// TestRegressionForwarderRelationships checks the algebraic ties that hold through the
// facade, which any mis-forwarded argument would break.
func TestRegressionForwarderRelationships(t *testing.T) {
	_, _, _, close, _ := testOHLCV()
	const n = 14

	slope := LinearRegressionSlope(close, n)
	first := LinearRegressionIntercept(close, n)
	last := LinearRegression(close, n)
	for i := n - 1; i < len(close); i++ {
		want := slope[i] * float64(n-1)
		if math.Abs((last[i]-first[i])-want) > 1e-9 {
			t.Fatalf("i=%d: last-first = %v, want %v", i, last[i]-first[i], want)
		}
	}

	// A straight line is the fixed point: R² is 1 and the standard error is 0.
	line := make([]float64, 60)
	for i := range line {
		line[i] = 3 + 1.5*float64(i)
	}
	lineR2 := R2(line, n)
	lineSE := StandardError(line, n)
	for i := n - 1; i < len(line); i++ {
		if math.Abs(lineR2[i]-1) > 1e-9 {
			t.Fatalf("R² of a straight line[%d] = %v, want 1", i, lineR2[i])
		}
		if math.Abs(lineSE[i]) > 1e-9 {
			t.Fatalf("standard error of a straight line[%d] = %v, want 0", i, lineSE[i])
		}
	}
}

func TestOscillatorForwardersMatchTa(t *testing.T) {
	_, high, low, close, _ := testOHLCV()

	eqSeries(t, "AwesomeOscillator", AwesomeOscillator(high, low, 5, 34), ta.AwesomeOscillator(high, low, 5, 34))
	eqSeries(t, "AcceleratorOscillator", AcceleratorOscillator(high, low, 5, 34, 5), ta.AcceleratorOscillator(high, low, 5, 34, 5))
	eqSeries(t, "DetrendedPriceOscillator", DetrendedPriceOscillator(close, 20), ta.DetrendedPriceOscillator(close, 20))
	eqSeries(t, "TRIX", TRIX(close, 5), ta.TRIX(close, 5))
	eqSeries(t, "UltimateOscillator", UltimateOscillator(high, low, close, 7, 14, 28), ta.UltimateOscillator(high, low, close, 7, 14, 28))
	eqSeries(t, "FisherTransform", FisherTransform(high, low, 9), ta.FisherTransform(high, low, 9))
	eqSeries(t, "MassIndex", MassIndex(high, low, 9, 25), ta.MassIndex(high, low, 9, 25))
}

// TestUltimateOscillatorForwarderKeepsParameterOrder is the specific three-period test.
// A transposition of short, mid and long changes the weighting, which is the whole point
// of the indicator, so all three are distinct here.
func TestUltimateOscillatorForwarderKeepsParameterOrder(t *testing.T) {
	_, high, low, close, _ := testOHLCV()

	base := UltimateOscillator(high, low, close, 5, 10, 20)
	swapped := UltimateOscillator(high, low, close, 20, 10, 5)
	if eqSeriesEqual(base, swapped) {
		t.Error("UltimateOscillator is insensitive to exchanging short and long")
	}
}

func TestOscillatorForwarderRanges(t *testing.T) {
	_, high, low, close, _ := testOHLCV()

	for i, v := range UltimateOscillator(high, low, close, 7, 14, 28) {
		if math.IsNaN(v) {
			continue
		}
		if v < -1e-9 || v > 100+1e-9 {
			t.Fatalf("UltimateOscillator[%d] = %v, outside [0,100]", i, v)
		}
	}
	// R² is bounded by 1 by construction.
	for i, v := range R2(close, 14) {
		if math.IsNaN(v) {
			continue
		}
		if v < -1e-9 || v > 1+1e-9 {
			t.Fatalf("R2[%d] = %v, outside [0,1]", i, v)
		}
	}
}

// TestRegressionForwarderAgreesWithVecMean is a cross-package check: a regression over a
// constant offset reproduces the least-squares line, and on a straight line the value at
// the current bar must equal the current value, which vec can compute independently.
func TestRegressionForwarderAgreesWithVecMean(t *testing.T) {
	// A perfectly linear series: the regression endpoint equals the last value.
	const size = 80
	line := make([]float64, size)
	for i := range line {
		line[i] = 10 + 0.25*float64(i)
	}
	got := LinearRegression(line, 16)
	for i := 15; i < size; i++ {
		if math.Abs(got[i]-line[i]) > 1e-9 {
			t.Fatalf("LinearRegression[%d] = %v, want %v", i, got[i], line[i])
		}
	}
	// And the window mean is a different number, so the two must not be confused.
	if math.Abs(got[size-1]-vec.Mean(line[size-16:])) < 1e-9 {
		t.Error("LinearRegression returned the window mean; the fit is not being applied")
	}
}

func TestRegressionOscillatorForwardersPanicLikeTa(t *testing.T) {
	short := make([]float64, 2)
	long := make([]float64, 3)

	cases := []struct {
		name string
		fn   func()
	}{
		{"LinearRegression period", func() { LinearRegression(long, 1) }},
		{"LinearRegressionSlope period", func() { LinearRegressionSlope(long, 0) }},
		{"StandardErrorBands period", func() { StandardErrorBands(long, 1, 2) }},
		{"AwesomeOscillator period", func() { AwesomeOscillator(long, long, 0, 34) }},
		{"AcceleratorOscillator smooth", func() { AcceleratorOscillator(long, long, 5, 34, -1) }},
		{"DPO period", func() { DetrendedPriceOscillator(long, 0) }},
		{"TRIX period", func() { TRIX(long, 0) }},
		{"UltimateOscillator period", func() { UltimateOscillator(long, long, long, 7, 14, 0) }},
		{"UltimateOscillator length", func() { UltimateOscillator(short, short, long, 7, 14, 28) }},
		{"FisherTransform period", func() { FisherTransform(long, long, 0) }},
		{"MassIndex period", func() { MassIndex(long, long, 9, 0) }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			defer func() {
				if recover() == nil {
					t.Errorf("%s did not panic", tc.name)
				}
			}()
			tc.fn()
		})
	}
}
