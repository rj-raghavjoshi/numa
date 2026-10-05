package ta

import (
	"math"
	"testing"
)

// ---------------------------------------------------------------------------
// Tests for the pivot points.
//
// The scalar primitive carries the formulas, so it is tested with hand-computed values at a
// single bar where every level can be worked out on paper. The series form is then tested only
// for the one thing it adds: reading the *previous* bar, which is a causality property rather
// than an arithmetic one.
// ---------------------------------------------------------------------------

func TestPivotLevelsClassicWorked(t *testing.T) {
	// H=10, L=8, C=9.5 -> P = 27.5/3 = 9.16667, span = 2.
	const h, l, c = 10.0, 8.0, 9.5
	p, r1, r2, r3, s1, s2, s3 := PivotLevels(h, l, c, PivotClassic)

	const pWant = 27.5 / 3
	if math.Abs(p-pWant) > 1e-12 {
		t.Errorf("pivot = %v, want %v", p, pWant)
	}
	if math.Abs(r1-(2*pWant-l)) > 1e-12 {
		t.Errorf("R1 = %v, want %v", r1, 2*pWant-l)
	}
	if math.Abs(s1-(2*pWant-h)) > 1e-12 {
		t.Errorf("S1 = %v, want %v", s1, 2*pWant-h)
	}
	if math.Abs(r2-(pWant+2)) > 1e-12 {
		t.Errorf("R2 = %v, want %v", r2, pWant+2)
	}
	if math.Abs(s2-(pWant-2)) > 1e-12 {
		t.Errorf("S2 = %v, want %v", s2, pWant-2)
	}
	if math.Abs(r3-(h+2*(pWant-l))) > 1e-12 {
		t.Errorf("R3 = %v, want %v", r3, h+2*(pWant-l))
	}
	if math.Abs(s3-(l-2*(h-pWant))) > 1e-12 {
		t.Errorf("S3 = %v, want %v", s3, l-2*(h-pWant))
	}
}

// TestPivotLevelsFibonacciRatios pins the 38.2/61.8/100 placements, which are the definition.
func TestPivotLevelsFibonacciRatios(t *testing.T) {
	const h, l, c = 10.0, 8.0, 9.5
	p, r1, r2, r3, s1, s2, s3 := PivotLevels(h, l, c, PivotFibonacci)
	const span = h - l

	if math.Abs(r1-(p+0.382*span)) > 1e-12 || math.Abs(s1-(p-0.382*span)) > 1e-12 {
		t.Errorf("first Fibonacci level = (%v,%v)", r1, s1)
	}
	if math.Abs(r2-(p+0.618*span)) > 1e-12 || math.Abs(s2-(p-0.618*span)) > 1e-12 {
		t.Errorf("second Fibonacci level = (%v,%v)", r2, s2)
	}
	if math.Abs(r3-(p+span)) > 1e-12 || math.Abs(s3-(p-span)) > 1e-12 {
		t.Errorf("third Fibonacci level = (%v,%v)", r3, s3)
	}
}

// TestPivotLevelsCamarillaAnchorsToClose pins the property that distinguishes Camarilla: the
// levels are placed relative to the close, not the pivot.
func TestPivotLevelsCamarillaAnchorsToClose(t *testing.T) {
	const h, l, c = 10.0, 8.0, 9.5
	_, r1, r2, r3, s1, s2, s3 := PivotLevels(h, l, c, PivotCamarilla)
	const span = h - l

	if math.Abs(r1-(c+span*1.1/12)) > 1e-12 {
		t.Errorf("R1 = %v, want %v", r1, c+span*1.1/12)
	}
	if math.Abs(r2-(c+span*1.1/6)) > 1e-12 {
		t.Errorf("R2 = %v, want %v", r2, c+span*1.1/6)
	}
	if math.Abs(r3-(c+span*1.1/4)) > 1e-12 {
		t.Errorf("R3 = %v, want %v", r3, c+span*1.1/4)
	}
	if math.Abs(s1-(c-span*1.1/12)) > 1e-12 {
		t.Errorf("S1 = %v, want %v", s1, c-span*1.1/12)
	}
	_ = s2
	_ = s3
}

// TestPivotLevelsWoodieWeightsCloseTwice pins the Woodie pivot, which is the only difference
// from Classic.
func TestPivotLevelsWoodieWeightsCloseTwice(t *testing.T) {
	const h, l, c = 10.0, 8.0, 9.5
	p, _, _, _, _, _, _ := PivotLevels(h, l, c, PivotWoodie)
	const want = (h + l + 2*c) / 4
	if math.Abs(p-want) > 1e-12 {
		t.Errorf("Woodie pivot = %v, want %v", p, want)
	}
	// And it must differ from Classic on this input, or the test proves nothing.
	pc, _, _, _, _, _, _ := PivotLevels(h, l, c, PivotClassic)
	if math.Abs(p-pc) < 1e-12 {
		t.Error("Woodie and Classic agree on an input chosen to separate them")
	}
}

// TestPivotPointsUsesThePreviousBar is the causality check: the level at index i must depend on
// index i-1 and not on index i, or the level would not be knowable when it is used.
func TestPivotPointsUsesThePreviousBar(t *testing.T) {
	high, low, close := synthHLC(120, 4747)

	pivot, r1, _, _, _, _, _ := PivotPoints(high, low, close, PivotClassic)

	for i := 1; i < len(close); i++ {
		wantP, wantR1, _, _, _, _, _ := PivotLevels(high[i-1], low[i-1], close[i-1], PivotClassic)
		if !closeOrNaN(pivot[i], wantP) {
			t.Fatalf("pivot[%d] = %v, want %v from bar %d", i, pivot[i], wantP, i-1)
		}
		if !closeOrNaN(r1[i], wantR1) {
			t.Fatalf("R1[%d] = %v, want %v", i, r1[i], wantR1)
		}
	}
	if !math.IsNaN(pivot[0]) {
		t.Errorf("pivot[0] = %v, want NaN (no completed period precedes it)", pivot[0])
	}

	// Changing only the current bar must not change the current bar's levels.
	alt := append([]float64(nil), close...)
	alt[len(alt)-1] *= 10
	altPivot, _, _, _, _, _, _ := PivotPoints(high, low, alt, PivotClassic)
	if !closeOrNaN(altPivot[len(alt)-1], pivot[len(pivot)-1]) {
		t.Error("the levels moved when only the current bar changed, which is look-ahead")
	}
}

func TestPivotMethodString(t *testing.T) {
	cases := map[PivotMethod]string{
		PivotClassic:    "Classic",
		PivotFibonacci:  "Fibonacci",
		PivotCamarilla:  "Camarilla",
		PivotWoodie:     "Woodie",
		PivotMethod(99): "Unknown",
	}
	for m, want := range cases {
		if got := m.String(); got != want {
			t.Errorf("PivotMethod(%d).String() = %q, want %q", m, got, want)
		}
	}
}

func TestPivotPointsEmptyAndUnknownMethod(t *testing.T) {
	if p, a, b, c, d, e, f := PivotPoints(nil, nil, nil, PivotClassic); p != nil || a != nil || b != nil || c != nil || d != nil || e != nil || f != nil {
		t.Error("empty PivotPoints should return nils")
	}

	func() {
		defer func() {
			if recover() == nil {
				t.Error("PivotLevels with an unknown method did not panic")
			}
		}()
		PivotLevels(1, 1, 1, PivotMethod(99))
	}()

	func() {
		defer func() {
			if recover() == nil {
				t.Error("PivotPoints with mismatched lengths did not panic")
			}
		}()
		PivotPoints(make([]float64, 2), make([]float64, 2), make([]float64, 3), PivotClassic)
	}()
}
