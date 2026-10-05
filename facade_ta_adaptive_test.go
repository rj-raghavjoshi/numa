package numa

import (
	"math"
	"testing"

	"github.com/rj-raghavjoshi/numa/ta"
)

// ---------------------------------------------------------------------------
// Forwarder tests for the extended moving averages, envelopes and price oscillator.
//
// ALMA and KAMA take parameters of mixed types and orders that are easy to transpose: ALMA's
// (offset, sigma) are both "shape" numbers, and KAMA's (fast, slow) are two periods whose meaning
// inverts if swapped. Both are driven with distinct values and compared against ta directly.
// ---------------------------------------------------------------------------

func TestAdaptiveForwardersMatchTa(t *testing.T) {
	_, _, _, close, _ := testOHLCV()

	eqSeries(t, "ALMA", ALMA(close, 20, 0.85, 6), ta.ALMA(close, 20, 0.85, 6))
	eqSeries(t, "SWMA", SWMA(close), ta.SWMA(close))
	eqSeries(t, "KAMA", KAMA(close, 10, 2, 30), ta.KAMA(close, 10, 2, 30))
	eqSeries(t, "VIDYA", VIDYA(close, 14, 9), ta.VIDYA(close, 14, 9))
	eqSeries(t, "ZLEMA", ZLEMA(close, 20), ta.ZLEMA(close, 20))
	eqSeries(t, "T3", T3(close, 5, 0.7), ta.T3(close, 5, 0.7))
	eqSeries(t, "McGinleyDynamic", McGinleyDynamic(close, 14), ta.McGinleyDynamic(close, 14))

	// Distinct parameters must actually change the answer.
	if eqSeriesEqual(ALMA(close, 20, 0.2, 6), ALMA(close, 20, 0.9, 6)) {
		t.Error("ALMA ignores its offset")
	}
	if eqSeriesEqual(ALMA(close, 20, 0.85, 2), ALMA(close, 20, 0.85, 8)) {
		t.Error("ALMA ignores its sigma")
	}
	// Swapping fast and slow is rejected rather than ignored, so the order is checked with two
	// pairs that are both valid but differ.
	if eqSeriesEqual(KAMA(close, 10, 2, 30), KAMA(close, 10, 4, 30)) {
		t.Error("KAMA ignores its fast period")
	}
	if eqSeriesEqual(KAMA(close, 10, 2, 30), KAMA(close, 10, 2, 20)) {
		t.Error("KAMA ignores its slow period")
	}
	if eqSeriesEqual(T3(close, 5, 0.2), T3(close, 5, 0.9)) {
		t.Error("T3 ignores its volume factor")
	}
}

func TestEnvelopeForwardersMatchTa(t *testing.T) {
	_, high, low, close, _ := testOHLCV()

	gu, gm, gl := MovingAverageEnvelope(close, 20, 0.05)
	wu, wm, wl := ta.MovingAverageEnvelope(close, 20, 0.05)
	eqSeries(t, "envelope upper", gu, wu)
	eqSeries(t, "envelope middle", gm, wm)
	eqSeries(t, "envelope lower", gl, wl)

	gu2, gl2 := MovingAverageChannel(high, low, 15)
	wu2, wl2 := ta.MovingAverageChannel(high, low, 15)
	eqSeries(t, "channel upper", gu2, wu2)
	eqSeries(t, "channel lower", gl2, wl2)

	eqSeries(t, "PriceOscillator", PriceOscillator(close, 10, 30), ta.PriceOscillator(close, 10, 30))
	eqSeries(t, "WeightedClose", WeightedClose(high, low, close), ta.WeightedClose(high, low, close))
}

// TestMovingAverageEnvelopeForwarderPercentUnit is the unit check: the parameter is a fraction, so
// 0.05 must give a two-percent-ish band and 5 must give a band ten times wider.
func TestMovingAverageEnvelopeForwarderPercentUnit(t *testing.T) {
	_, _, _, close, _ := testOHLCV()

	small, middle, _ := MovingAverageEnvelope(close, 20, 0.05)
	large, _, _ := MovingAverageEnvelope(close, 20, 0.5)
	for i := range close {
		if math.IsNaN(middle[i]) {
			continue
		}
		if !(large[i] > small[i]) {
			t.Fatalf("a 0.5 fraction must give a wider band than 0.05 at %d", i)
		}
		// The width is exactly proportional to the fraction.
		if math.Abs((large[i]-middle[i])/10-(small[i]-middle[i])) > 1e-9 {
			t.Fatalf("band width is not proportional to percent at %d", i)
		}
	}
}

func TestAdaptiveForwardersPanicLikeTa(t *testing.T) {
	_, _, low, close, _ := testOHLCV()
	short := make([]float64, 5)

	cases := []struct {
		name string
		fn   func()
	}{
		{"ALMA period", func() { ALMA(close, 0, 0.85, 6) }},
		{"ALMA sigma", func() { ALMA(close, 5, 0.85, 0) }},
		{"ALMA offset", func() { ALMA(close, 5, 2, 6) }},
		{"KAMA fast >= slow", func() { KAMA(close, 10, 30, 30) }},
		{"VIDYA period", func() { VIDYA(close, 14, 0) }},
		{"ZLEMA period", func() { ZLEMA(close, 0) }},
		{"T3 factor", func() { T3(close, 5, -1) }},
		{"McGinley period", func() { McGinleyDynamic(close, 0) }},
		{"envelope negative", func() { MovingAverageEnvelope(close, 10, -0.5) }},
		{"channel length", func() { MovingAverageChannel(short, low, 5) }},
		{"oscillator reversed", func() { PriceOscillator(close, 30, 10) }},
		{"weighted close length", func() { WeightedClose(short, low, close) }},
	}
	for _, tc := range cases {
		fn := tc.fn
		t.Run(tc.name, func(t *testing.T) {
			defer func() {
				if recover() == nil {
					t.Errorf("%s did not panic", tc.name)
				}
			}()
			fn()
		})
	}
}

func TestAdaptiveForwardersEmptyInputs(t *testing.T) {
	if ALMA(nil, 5, 0.85, 6) != nil || SWMA(nil) != nil || KAMA(nil, 5, 2, 30) != nil ||
		VIDYA(nil, 14, 9) != nil || ZLEMA(nil, 5) != nil || T3(nil, 5, 0.7) != nil ||
		McGinleyDynamic(nil, 5) != nil {
		t.Error("empty adaptive inputs should return nil")
	}
	if u, m, l := MovingAverageEnvelope(nil, 10, 0.05); u != nil || m != nil || l != nil {
		t.Error("empty MovingAverageEnvelope should return nils")
	}
	if WeightedClose(nil, nil, nil) != nil {
		t.Error("empty WeightedClose should be nil")
	}
}
