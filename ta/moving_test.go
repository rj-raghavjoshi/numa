package ta

import (
	"math"
	"testing"
)

// ---------------------------------------------------------------------------
// Tests for the moving averages.
//
// The interesting failures: a wrong warm-up boundary (off by one), a wrong seed for
// the exponential family, and a composite that feeds its inner indicator's warm-up
// NaNs into an outer smoother's seed and is therefore NaN forever.
// ---------------------------------------------------------------------------

func TestSimpleAveragesMatchReference(t *testing.T) {
	for _, n := range boundaryLengths() {
		if n == 0 {
			continue
		}
		xs := ramp(n)
		for _, p := range boundaryPeriods() {
			assertSeries(t, "SMA", SMA(xs, p), refSMA(xs, p), false)
			assertSeries(t, "WMA", WMA(xs, p), refWMA(xs, p), false)
		}
	}
}

func TestExponentialAveragesMatchReference(t *testing.T) {
	for _, n := range boundaryLengths() {
		if n == 0 {
			continue
		}
		xs := ramp(n)
		for _, p := range boundaryPeriods() {
			assertSeries(t, "EMA", EMA(xs, p), refSmoother(xs, p, 2.0/float64(p+1)), false)
			assertSeries(t, "RMA", RMA(xs, p), refSmoother(xs, p, 1.0/float64(p)), false)
		}
	}
}

// TestMovingAverageWarmup pins each warm-up length exactly, because an off-by-one here
// silently shifts every consumer of the indicator.
func TestMovingAverageWarmup(t *testing.T) {
	const size = 80
	xs := ramp(size)

	cases := []struct {
		name string
		got  []float64
		want int
	}{
		{"SMA", SMA(xs, 10), 9},
		{"WMA", WMA(xs, 10), 9},
		{"EMA", EMA(xs, 10), 9},
		{"RMA", RMA(xs, 10), 9},
		{"DEMA", DEMA(xs, 5), 2*5 - 2},
		{"TEMA", TEMA(xs, 5), 3*5 - 3},
	}
	for _, tc := range cases {
		if got := FirstValid(tc.got); got != tc.want {
			t.Errorf("%s: first valid index = %d, want %d", tc.name, got, tc.want)
		}
	}
}

// TestCompositesOfAConstantAreThatConstant checks the algebraic fixed point of each
// composite, which catches a wrong coefficient or a misaligned composition.
func TestCompositesOfAConstantAreThatConstant(t *testing.T) {
	const size = 120
	const c = 7.25
	xs := make([]float64, size)
	for i := range xs {
		xs[i] = c
	}

	for name, series := range map[string][]float64{
		"SMA":   SMA(xs, 9),
		"EMA":   EMA(xs, 9),
		"RMA":   RMA(xs, 9),
		"WMA":   WMA(xs, 9),
		"DEMA":  DEMA(xs, 9),
		"TEMA":  TEMA(xs, 9),
		"TRIMA": TRIMA(xs, 9),
		"HMA":   HMA(xs, 9),
	} {
		first := FirstValid(series)
		if first < 0 {
			t.Fatalf("%s produced no valid values", name)
		}
		for i := first; i < size; i++ {
			if math.Abs(series[i]-c) > 1e-9 {
				t.Fatalf("%s[%d] = %v, want %v (constant series)", name, i, series[i], c)
			}
		}
	}
}

// TestCompositeDoesNotPoisonItsOuterSeed is the specific regression test for the
// composition bug: feeding the inner indicator's leading NaNs to a seeded smoother
// makes the seed NaN and every later output NaN. applyFrom exists to prevent that.
func TestCompositeDoesNotPoisonItsOuterSeed(t *testing.T) {
	const size = 200
	open, _, _, close, _ := synthOHLCV(size, 5)
	_ = open

	for name, series := range map[string][]float64{
		"DEMA": DEMA(close, 12),
		"TEMA": TEMA(close, 12),
	} {
		first := FirstValid(series)
		if first < 0 {
			t.Fatalf("%s is entirely NaN, which means the outer seed was poisoned", name)
		}
		// Everything from the first valid index onward must be present too.
		for i := first; i < size; i++ {
			if math.IsNaN(series[i]) {
				t.Fatalf("%s[%d] = NaN after first valid %d", name, i, first)
			}
		}
	}
}

func TestTRIMAIsSymmetricComposition(t *testing.T) {
	const size = 100
	xs := ramp(size)
	for _, n := range []int{4, 5, 9, 10, 16} {
		len1 := (n + 1) / 2
		len2 := n/2 + 1
		want := SMA(SMA(xs, len1), len2)
		assertSeries(t, "TRIMA", TRIMA(xs, n), want, true)
	}
}

func TestMovingAverageEmptyAndPanics(t *testing.T) {
	for _, f := range []func([]float64, int) []float64{SMA, EMA, RMA, WMA, DEMA, TEMA, TRIMA, HMA} {
		if got := f(nil, 5); got != nil {
			t.Errorf("empty input returned %v, want nil", got)
		}
	}

	func() {
		defer func() {
			if recover() == nil {
				t.Error("SMA with period 0 did not panic")
			}
		}()
		SMA(ramp(10), 0)
	}()

	for _, f := range []func([]float64, int) []float64{SMA, EMA, RMA, WMA, DEMA, TEMA, TRIMA, HMA} {
		func() {
			defer func() {
				if recover() == nil {
					t.Error("a period below 1 did not panic")
				}
			}()
			f(ramp(10), -1)
		}()
	}
}

// TestWNANRecoversAfterTheWindowClears checks that a NaN in the middle of the series
// only affects the windows that contain it, for the direct-loop averages.
//
// With the NaN at index 3 and a window of 3, the affected output indices are 3, 4 and
// 5 -- the windows {2,3,NaN}, {3,NaN,5} and {NaN,5,6}. Index 6 is the first window
// free of it.
func TestWNANRecoversAfterTheWindowClears(t *testing.T) {
	xs := []float64{1, 2, 3, math.NaN(), 5, 6, 7}
	const n = 3

	sma := SMA(xs, n)
	wma := WMA(xs, n)

	for _, i := range []int{3, 4, 5} {
		if !math.IsNaN(sma[i]) {
			t.Errorf("SMA[%d] = %v, want NaN (window contains the NaN)", i, sma[i])
		}
		if !math.IsNaN(wma[i]) {
			t.Errorf("WMA[%d] = %v, want NaN (window contains the NaN)", i, wma[i])
		}
	}
	if math.IsNaN(sma[6]) || math.IsNaN(wma[6]) {
		t.Errorf("the averages did not recover once the NaN left the window: sma=%v wma=%v", sma, wma)
	}
}

// TestExponentialSmootherNaNIsPermanent documents the deliberate asymmetry: a
// recurrence cannot forget, so a NaN after seeding is permanent. It is a property of
// the mathematics, not a bug, but it differs from the direct-loop averages and so is
// pinned rather than left implicit.
func TestExponentialSmootherNaNIsPermanent(t *testing.T) {
	xs := []float64{1, 2, 3, 4, math.NaN(), 6, 7, 8}
	const n = 3

	ema := EMA(xs, n)
	for i := 4; i < len(xs); i++ {
		if !math.IsNaN(ema[i]) {
			t.Errorf("EMA[%d] = %v, want NaN (recurrence cannot clear)", i, ema[i])
		}
	}
}
