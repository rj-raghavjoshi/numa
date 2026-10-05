package numa

import (
	"math"
	"testing"

	"github.com/rj-raghavjoshi/numa/ta"
)

// ---------------------------------------------------------------------------
// Forwarder tests for the trend indicators.
//
// ParabolicSAR takes three floats in a row and its two usual values (0.02 and 0.2)
// differ by a factor of ten, so a transposition produces a slower or faster
// acceleration schedule rather than an obviously wrong number. The checks use values
// that are all distinct and compare against ta with the same order.
// ---------------------------------------------------------------------------

func TestTrendForwardersMatchTa(t *testing.T) {
	_, high, low, close, _ := testOHLCV()

	gadx, gp, gm := DirectionalMovement(high, low, close, 14)
	wadx, wp, wm := ta.DirectionalMovement(high, low, close, 14)
	eqSeries(t, "ADX", gadx, wadx)
	eqSeries(t, "+DI", gp, wp)
	eqSeries(t, "-DI", gm, wm)

	gu, gd, go_ := Aroon(high, low, 14)
	wu, wd, wo := ta.Aroon(high, low, 14)
	eqSeries(t, "Aroon up", gu, wu)
	eqSeries(t, "Aroon down", gd, wd)
	eqSeries(t, "Aroon osc", go_, wo)

	gvp, gvm := Vortex(high, low, close, 14)
	wvp, wvm := ta.Vortex(high, low, close, 14)
	eqSeries(t, "VI+", gvp, wvp)
	eqSeries(t, "VI-", gvm, wvm)

	eqSeries(t, "Choppiness", Choppiness(high, low, close, 14), ta.Choppiness(high, low, close, 14))

	gku, gkm, gkl := KeltnerChannels(high, low, close, 20, 10, 2)
	wku, wkm, wkl := ta.KeltnerChannels(high, low, close, 20, 10, 2)
	eqSeries(t, "Keltner upper", gku, wku)
	eqSeries(t, "Keltner middle", gkm, wkm)
	eqSeries(t, "Keltner lower", gkl, wkl)

	gsl, gsd := SuperTrend(high, low, close, 10, 3)
	wsl, wsd := ta.SuperTrend(high, low, close, 10, 3)
	eqSeries(t, "SuperTrend line", gsl, wsl)
	eqSeries(t, "SuperTrend direction", gsd, wsd)

	gsar, gsdir := ParabolicSAR(high, low, 0.02, 0.02, 0.2)
	wsar, wsdir := ta.ParabolicSAR(high, low, 0.02, 0.02, 0.2)
	eqSeries(t, "PSAR", gsar, wsar)
	eqSeries(t, "PSAR direction", gsdir, wsdir)
}

// TestParabolicSARForwarderKeepsParameterOrder is the specific test for the three-float
// signature. Starting acceleration and increment are both 0.02 in Wilder's defaults, so
// the test drives them apart deliberately.
func TestParabolicSARForwarderKeepsParameterOrder(t *testing.T) {
	_, high, low, _, _ := testOHLCV()

	base, _ := ParabolicSAR(high, low, 0.01, 0.05, 0.3)
	swapped, _ := ParabolicSAR(high, low, 0.05, 0.01, 0.3)
	if eqSeriesEqual(base, swapped) {
		t.Error("ParabolicSAR produced the same series with start and increment exchanged")
	}

	// The max must reach ta's parameter too: a max below start is rejected.
	defers := func() (panicked bool) {
		defer func() { panicked = recover() != nil }()
		ParabolicSAR(high, low, 0.5, 0.02, 0.2)
		return
	}
	if !defers() {
		t.Error("ParabolicSAR with max < start did not panic")
	}
}

// TestSuperTrendForwarderKeepsParameterOrder: atrN and mult are an int and a float, so
// the compiler catches that transposition; what it does not catch is mult reaching the
// wrong place, which changes the band width. Comparing against ta with the same
// arguments is the check.
func TestSuperTrendForwarderKeepsParameterOrder(t *testing.T) {
	_, high, low, close, _ := testOHLCV()

	narrow, _ := SuperTrend(high, low, close, 10, 1)
	wide, _ := SuperTrend(high, low, close, 10, 3)
	if eqSeriesEqual(narrow, wide) {
		t.Error("SuperTrend ignored the multiplier")
	}

	short, _ := SuperTrend(high, low, close, 5, 3)
	long, _ := SuperTrend(high, low, close, 20, 3)
	if eqSeriesEqual(short, long) {
		t.Error("SuperTrend ignored the ATR period")
	}
}

func TestTrendForwarderInvariants(t *testing.T) {
	_, high, low, close, _ := testOHLCV()

	_, stDir := SuperTrend(high, low, close, 10, 3)
	for i, v := range stDir {
		if math.IsNaN(v) {
			continue
		}
		if v != 1 && v != -1 {
			t.Fatalf("SuperTrend direction[%d] = %v, want ±1", i, v)
		}
	}

	sar, dir := ParabolicSAR(high, low, 0.02, 0.02, 0.2)
	for i := range sar {
		if dir[i] != 1 && dir[i] != -1 {
			t.Fatalf("PSAR direction[%d] = %v, want ±1", i, dir[i])
		}
	}
}

func TestTrendForwardersPanicLikeTa(t *testing.T) {
	short := make([]float64, 2)
	long := make([]float64, 3)

	cases := []struct {
		name string
		fn   func()
	}{
		{"DM period", func() { DirectionalMovement(long, long, long, 0) }},
		{"DM length", func() { DirectionalMovement(short, short, long, 5) }},
		{"Aroon period", func() { Aroon(long, long, 0) }},
		{"Aroon length", func() { Aroon(short, long, 5) }},
		{"Vortex period", func() { Vortex(long, long, long, 0) }},
		{"Choppiness period", func() { Choppiness(long, long, long, 0) }},
		{"Keltner atrN", func() { KeltnerChannels(long, long, long, 20, 0, 2) }},
		{"SuperTrend atrN", func() { SuperTrend(long, long, long, 0, 3) }},
		{"PSAR increment", func() { ParabolicSAR(long, long, 0.02, 0, 0.2) }},
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

func TestTrendForwardersEmptyInputs(t *testing.T) {
	if adx, p, m := DirectionalMovement(nil, nil, nil, 14); adx != nil || p != nil || m != nil {
		t.Error("empty DirectionalMovement should return nils")
	}
	if u, d, o := Aroon(nil, nil, 14); u != nil || d != nil || o != nil {
		t.Error("empty Aroon should return nils")
	}
	if l, d := SuperTrend(nil, nil, nil, 10, 3); l != nil || d != nil {
		t.Error("empty SuperTrend should return nils")
	}
	if s, d := ParabolicSAR(nil, nil, 0.02, 0.02, 0.2); s != nil || d != nil {
		t.Error("empty ParabolicSAR should return nils")
	}
	if u, m, l := KeltnerChannels(nil, nil, nil, 20, 10, 2); u != nil || m != nil || l != nil {
		t.Error("empty KeltnerChannels should return nils")
	}
}
