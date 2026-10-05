package ta

import (
	"math"
	"testing"
)

// ---------------------------------------------------------------------------
// Tests for the batch splices.
//
// applyEMA and applyRMA exist so the composite indicators can use the batch smoothers instead of the
// streaming rollers. The warm-up region is where that splice can go wrong: an output slice is
// allocated zeroed, and a zero in the warm-up is a *value*, not a gap.
// ---------------------------------------------------------------------------

// TestApplyEMAWarmUpIsNaN pins the bug that made Klinger's oscillator report a defined value at index
// 0: the region before the start index must be NaN, because a caller composing on top of it cannot
// tell a zero from "not yet defined".
func TestApplyEMAWarmUpIsNaN(t *testing.T) {
	xs := make([]float64, 40)
	for i := range xs {
		xs[i] = float64(i) + 1
	}
	for _, from := range []int{0, 1, 5, 20, 40, 100} {
		for name, out := range map[string][]float64{
			"applyEMA": applyEMA(xs, from, 5),
			"applyRMA": applyRMA(xs, from, 5),
		} {
			if len(out) != len(xs) {
				t.Fatalf("%s(from=%d): length %d", name, from, len(out))
			}
			for i := 0; i < min(from, len(xs)); i++ {
				if !math.IsNaN(out[i]) {
					t.Fatalf("%s(from=%d): index %d = %v, want NaN", name, from, i, out[i])
				}
			}
		}
	}
}

// TestCompositesHaveNaNWarmUps is the indicator-level version: every composite that uses a batch
// splice must report its warm-up as NaN rather than as a number.
func TestCompositesHaveNaNWarmUps(t *testing.T) {
	size := 200
	high, low, close := synthHLC(size, 99)
	volume := make([]float64, size)
	for i := range volume {
		volume[i] = 1000 + float64(i)
	}

	checks := map[string][]float64{
		"MACD line":       func() []float64 { l, _, _ := MACD(close, 12, 26, 9); return l }(),
		"DEMA":            DEMA(close, 10),
		"TEMA":            TEMA(close, 10),
		"T3":              T3(close, 5, 0.7),
		"ZLEMA":           ZLEMA(close, 20),
		"ATR":             ATR(high, low, close, 14),
		"DirectionalMove": func() []float64 { _, _, adx := DirectionalMovement(high, low, close, 14); return adx }(),
		"TrueStrength":    TrueStrengthIndex(close, 25, 13),
		"KlingerOsc":      func() []float64 { k, _ := KlingerOscillator(high, low, close, volume, 34, 55, 13); return k }(),
	}
	for name, out := range checks {
		if len(out) == 0 {
			t.Fatalf("%s returned nothing", name)
		}
		if out[0] == out[0] {
			t.Errorf("%s[0] = %v, want NaN: a composite's first value is never defined", name, out[0])
		}
	}
}
