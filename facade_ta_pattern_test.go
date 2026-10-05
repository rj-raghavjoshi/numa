package numa

import (
	"math"
	"testing"

	"github.com/rj-raghavjoshi/numa/ta"
)

// ---------------------------------------------------------------------------
// Forwarder tests for Ichimoku, pivots, GMMA and ZigZag.
//
// The PivotMethod alias gets its own check: a re-exported *type* can be wrong in a way a
// function forwarder cannot, because the constants could alias different values while every
// call still compiles.
// ---------------------------------------------------------------------------

func TestIchimokuForwarderMatchesTa(t *testing.T) {
	open, high, low, close, _ := testOHLCV()
	_ = open

	gc, gb, ga, gs, gl := Ichimoku(high, low, close, 9, 26, 52, 26)
	wc, wb, wa, ws, wl := ta.Ichimoku(high, low, close, 9, 26, 52, 26)
	eqSeries(t, "Ichimoku conversion", gc, wc)
	eqSeries(t, "Ichimoku base", gb, wb)
	eqSeries(t, "Ichimoku spanA", ga, wa)
	eqSeries(t, "Ichimoku spanB", gs, ws)
	eqSeries(t, "Ichimoku lagging", gl, wl)

	// The displacement must reach the implementation.
	_, _, _, _, shifted := Ichimoku(high, low, close, 9, 26, 52, 5)
	if eqSeriesEqual(gl, shifted) {
		t.Error("Ichimoku ignored the displacement argument")
	}
}

func TestPivotMethodAlias(t *testing.T) {
	// The constants must be the same values as ta's, not merely defined.
	if PivotClassic != ta.PivotClassic || PivotFibonacci != ta.PivotFibonacci ||
		PivotCamarilla != ta.PivotCamarilla || PivotWoodie != ta.PivotWoodie {
		t.Fatal("the facade's PivotMethod constants do not match ta's")
	}
	// And a facade value must be accepted by ta, which is what the alias guarantees.
	p, _, _, _, _, _, _ := PivotLevels(10, 8, 9.5, PivotWoodie)
	wp, _, _, _, _, _, _ := ta.PivotLevels(10, 8, 9.5, ta.PivotWoodie)
	if p != wp {
		t.Errorf("PivotLevels through the alias = %v, ta = %v", p, wp)
	}
	// Method names must survive the alias too.
	for _, m := range []PivotMethod{PivotClassic, PivotFibonacci, PivotCamarilla, PivotWoodie} {
		if m.String() == "" || m.String() == "Unknown" {
			t.Errorf("PivotMethod %d has no name through the facade", m)
		}
	}
}

func TestPivotForwardersMatchTa(t *testing.T) {
	_, high, low, close, _ := testOHLCV()

	for _, m := range []PivotMethod{PivotClassic, PivotFibonacci, PivotCamarilla, PivotWoodie} {
		gp, gr1, gr2, gr3, gs1, gs2, gs3 := PivotPoints(high, low, close, m)
		wp, wr1, wr2, wr3, ws1, ws2, ws3 := ta.PivotPoints(high, low, close, m)
		eqSeries(t, "pivot", gp, wp)
		eqSeries(t, "R1", gr1, wr1)
		eqSeries(t, "R2", gr2, wr2)
		eqSeries(t, "R3", gr3, wr3)
		eqSeries(t, "S1", gs1, ws1)
		eqSeries(t, "S2", gs2, ws2)
		eqSeries(t, "S3", gs3, ws3)
	}
}

// TestPivotPointsForwarderUsesThePreviousBar repeats the causality check through the facade,
// because it is the one property a reordering of the arguments could break silently.
func TestPivotPointsForwarderUsesThePreviousBar(t *testing.T) {
	_, high, low, close, _ := testOHLCV()

	pivot, _, _, _, _, _, _ := PivotPoints(high, low, close, PivotClassic)
	if !math.IsNaN(pivot[0]) {
		t.Error("pivot[0] should be NaN")
	}
	want, _, _, _, _, _, _ := PivotLevels(high[0], low[0], close[0], PivotClassic)
	if !sameOrNaNFloat(pivot[1], want) {
		t.Errorf("pivot[1] = %v, want %v from bar 0", pivot[1], want)
	}
}

func TestGMMAForwarderMatchesTa(t *testing.T) {
	_, _, _, close, _ := testOHLCV()

	gs, gl := GMMA(close)
	ws, wl := ta.GMMA(close)
	for i := 0; i < 6; i++ {
		eqSeries(t, "GMMA short", gs[i], ws[i])
		eqSeries(t, "GMMA long", gl[i], wl[i])
	}
}

func TestZigZagForwarderMatchesTa(t *testing.T) {
	_, high, low, _, _ := testOHLCV()

	gp, gk := ZigZag(high, low, 0.05)
	wp, wk := ta.ZigZag(high, low, 0.05)
	eqSeries(t, "ZigZag pivot", gp, wp)
	if len(gk) != len(wk) {
		t.Fatalf("ZigZag kind length %d, want %d", len(gk), len(wk))
	}
	for i := range wk {
		if gk[i] != wk[i] {
			t.Fatalf("kind[%d] = %d, want %d", i, gk[i], wk[i])
		}
	}
}

func TestPatternForwardersPanicLikeTa(t *testing.T) {
	short := make([]float64, 2)
	long := make([]float64, 3)

	cases := []struct {
		name string
		fn   func()
	}{
		{"Ichimoku period", func() { Ichimoku(long, long, long, 0, 26, 52, 26) }},
		{"Ichimoku displacement", func() { Ichimoku(long, long, long, 9, 26, 52, -1) }},
		{"Ichimoku length", func() { Ichimoku(short, short, long, 9, 26, 52, 26) }},
		{"PivotLevels method", func() { PivotLevels(1, 1, 1, PivotMethod(99)) }},
		{"PivotPoints length", func() { PivotPoints(short, short, long, PivotClassic) }},
		{"ZigZag deviation", func() { ZigZag(long, long, -0.1) }},
		{"ZigZag length", func() { ZigZag(short, long, 0.1) }},
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

func TestPatternForwardersEmptyInputs(t *testing.T) {
	if c, b, a, s, l := Ichimoku(nil, nil, nil, 9, 26, 52, 26); c != nil || b != nil || a != nil || s != nil || l != nil {
		t.Error("empty Ichimoku should return nils")
	}
	if p, _, _, _, _, _, _ := PivotPoints(nil, nil, nil, PivotClassic); p != nil {
		t.Error("empty PivotPoints should return nils")
	}
	gs, gl := GMMA(nil)
	for i := 0; i < 6; i++ {
		if gs[i] != nil || gl[i] != nil {
			t.Error("empty GMMA should return nil series")
		}
	}
	if p, k := ZigZag(nil, nil, 0.05); p != nil || k != nil {
		t.Error("empty ZigZag should return nils")
	}
}
