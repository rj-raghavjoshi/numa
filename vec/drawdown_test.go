package vec

import (
	"math"
	"testing"
)

func TestDrawdownKnownValues(t *testing.T) {
	xs := []float64{100, 120, 90, 96, 120, 150, 120}
	got := Drawdown(xs)
	want := []float64{0, 0, -0.25, -0.2, 0, 0, -0.2}
	cumCloseEnough(t, "Drawdown", got, want)

	// Aliasing safety: dst == xs must give the identical curve.
	aliased := append([]float64(nil), xs...)
	DrawdownTo(aliased, aliased)
	cumCloseEnough(t, "DrawdownTo aliased", aliased, want)

	maxDD, peak, trough, rec := MaxDrawdown(xs)
	if !closeEnough(maxDD, 0.25) || peak != 1 || trough != 2 || rec != 4 {
		t.Fatalf("MaxDrawdown = (%v, %d, %d, %d), want (0.25, 1, 2, 4)", maxDD, peak, trough, rec)
	}
}

func TestDrawdownUnrecoveredAndMonotonic(t *testing.T) {
	unrec := []float64{100, 80, 90}
	maxDD, peak, trough, rec := MaxDrawdown(unrec)
	if !closeEnough(maxDD, 0.2) || peak != 0 || trough != 1 || rec != -1 {
		t.Fatalf("unrecovered MaxDrawdown = (%v, %d, %d, %d), want (0.2, 0, 1, -1)", maxDD, peak, trough, rec)
	}

	rising := []float64{10, 12, 12, 15}
	maxDD, peak, trough, rec = MaxDrawdown(rising)
	if maxDD != 0 || peak != -1 || trough != -1 || rec != -1 {
		t.Fatalf("rising MaxDrawdown = (%v, %d, %d, %d), want (0, -1, -1, -1)", maxDD, peak, trough, rec)
	}
}

func TestDrawdownEdgeAndNaNPolicy(t *testing.T) {
	if Drawdown(nil) != nil {
		t.Error("Drawdown(nil) should be nil")
	}
	maxDD, peak, trough, rec := MaxDrawdown(nil)
	if !math.IsNaN(maxDD) || peak != -1 || trough != -1 || rec != -1 {
		t.Errorf("MaxDrawdown(nil) = (%v, %d, %d, %d), want (NaN, -1, -1, -1)", maxDD, peak, trough, rec)
	}

	// Non-positive prices cannot form a percentage peak drawdown.
	maxDD, peak, trough, rec = MaxDrawdown([]float64{0, -5, -1})
	if !math.IsNaN(maxDD) || peak != -1 || trough != -1 || rec != -1 {
		t.Errorf("MaxDrawdown(non-positive) = (%v, %d, %d, %d), want (NaN, -1, -1, -1)", maxDD, peak, trough, rec)
	}

	// Forward NaN poisoning in Drawdown, and whole-slice NaN policy in MaxDrawdown.
	for pos := 0; pos < 5; pos++ {
		xs := []float64{100, 120, 90, 110, 130}
		xs[pos] = math.NaN()

		dd := Drawdown(xs)
		for i := 0; i < pos; i++ {
			if math.IsNaN(dd[i]) {
				t.Errorf("pos=%d: Drawdown[%d] poisoned prematurely: %v", pos, i, dd[i])
			}
		}
		for i := pos; i < len(xs); i++ {
			if !math.IsNaN(dd[i]) {
				t.Errorf("pos=%d: Drawdown[%d] = %v, want NaN", pos, i, dd[i])
			}
		}

		mdd, p, tr, rc := MaxDrawdown(xs)
		if !math.IsNaN(mdd) || p != -1 || tr != -1 || rc != -1 {
			t.Errorf("pos=%d: MaxDrawdown = (%v, %d, %d, %d), want (NaN, -1, -1, -1)", pos, mdd, p, tr, rc)
		}
	}
}

func TestDrawdownToPanicsOnMismatch(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Error("DrawdownTo did not panic on length mismatch")
		}
	}()
	DrawdownTo(make([]float64, 2), make([]float64, 3))
}
