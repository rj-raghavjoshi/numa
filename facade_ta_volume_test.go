package numa

import (
	"math"
	"testing"

	"github.com/rj-raghavjoshi/numa/ta"
)

// ---------------------------------------------------------------------------
// Forwarder tests for the volume indicators.
//
// These take up to four parallel columns, so the argument order is the whole risk:
// (high, low, close, volume) transposed to (open, high, low, close) compiles cleanly.
// The test data below deliberately gives each column a distinct magnitude, and the
// checks compare against ta with the same order rather than only against a hand value.
// ---------------------------------------------------------------------------

func TestVolumeForwardersMatchTa(t *testing.T) {
	_, high, low, close, volume := testOHLCV()

	eqSeries(t, "VWAP", VWAP(high, low, close, volume), ta.VWAP(high, low, close, volume))
	eqSeries(t, "VWMA", VWMA(close, volume, 10), ta.VWMA(close, volume, 10))
	eqSeries(t, "OBV", OBV(close, volume), ta.OBV(close, volume))
	eqSeries(t, "A/D", AccumulationDistribution(high, low, close, volume),
		ta.AccumulationDistribution(high, low, close, volume))
	eqSeries(t, "CMF", ChaikinMoneyFlow(high, low, close, volume, 20),
		ta.ChaikinMoneyFlow(high, low, close, volume, 20))
	eqSeries(t, "ChaikinOscillator", ChaikinOscillator(high, low, close, volume, 3, 10),
		ta.ChaikinOscillator(high, low, close, volume, 3, 10))
	eqSeries(t, "MFI", MoneyFlowIndex(high, low, close, volume, 14),
		ta.MoneyFlowIndex(high, low, close, volume, 14))
	eqSeries(t, "PVT", PriceVolumeTrend(close, volume), ta.PriceVolumeTrend(close, volume))
	eqSeries(t, "NetVolume", NetVolume(close, volume), ta.NetVolume(close, volume))
	eqSeries(t, "VolumeOscillator", VolumeOscillator(volume, 5, 20), ta.VolumeOscillator(volume, 5, 20))
	eqSeries(t, "ForceIndex", ForceIndex(close, volume, 13), ta.ForceIndex(close, volume, 13))
}

// TestVolumeForwardersDistinguishColumns would fail if a forwarder silently reordered
// the price columns, because swapping low and close changes the A/D term materially.
func TestVolumeForwardersDistinguishColumns(t *testing.T) {
	_, high, low, close, volume := testOHLCV()

	correct := AccumulationDistribution(high, low, close, volume)
	swapped := AccumulationDistribution(high, close, low, volume)
	if eqSeriesEqual(correct, swapped) {
		t.Error("A/D produced the same line with low and close swapped; the test data cannot detect a transposition")
	}
}

// TestVWAPIsSymmetricInItsPriceArguments documents a real property that makes one
// transposition undetectable: the typical price is the sum (high+low+close)/3, so
// permuting those three arguments cannot change VWAP. That is worth stating rather than
// leaving as a puzzle, and it means the only transposition VWAP can suffer is between
// the price columns and the volume column -- which this test does detect.
func TestVWAPIsSymmetricInItsPriceArguments(t *testing.T) {
	_, high, low, close, volume := testOHLCV()

	base := VWAP(high, low, close, volume)
	eqSeries(t, "VWAP high/low swapped", VWAP(low, high, close, volume), base)
	eqSeries(t, "VWAP close/high swapped", VWAP(close, low, high, volume), base)

	// Volume is not interchangeable with a price column.
	if eqSeriesEqual(base, VWAP(volume, low, close, high)) {
		t.Error("VWAP produced the same line with the volume column moved into a price slot")
	}
}

func TestVolumeForwarderRelationships(t *testing.T) {
	_, high, low, close, volume := testOHLCV()

	// The Chaikin oscillator is the difference of two EMAs of the A/D line.
	ad := AccumulationDistribution(high, low, close, volume)
	fe, se := EMA(ad, 3), EMA(ad, 10)
	co := ChaikinOscillator(high, low, close, volume, 3, 10)
	for i := range ad {
		if !sameOrNaNFloat(co[i], fe[i]-se[i]) {
			t.Fatalf("ChaikinOscillator[%d] = %v, want %v", i, co[i], fe[i]-se[i])
		}
	}

	// NetVolume must be the per-bar ingredient of OBV: OBV is its running total.
	nv := NetVolume(close, volume)
	var total float64
	obv := OBV(close, volume)
	for i := range close {
		if i > 0 {
			total += nv[i]
		}
		if !sameOrNaNFloat(obv[i], total) {
			t.Fatalf("OBV[%d] = %v, but the running sum of NetVolume = %v", i, obv[i], total)
		}
	}
}

func TestVolumeForwarderRanges(t *testing.T) {
	_, high, low, close, volume := testOHLCV()

	for i, v := range ChaikinMoneyFlow(high, low, close, volume, 20) {
		if math.IsNaN(v) {
			continue
		}
		if v < -1-1e-9 || v > 1+1e-9 {
			t.Fatalf("CMF[%d] = %v, outside [-1,1]", i, v)
		}
	}
	for i, v := range MoneyFlowIndex(high, low, close, volume, 14) {
		if math.IsNaN(v) {
			continue
		}
		if v < -1e-9 || v > 100+1e-9 {
			t.Fatalf("MFI[%d] = %v, outside [0,100]", i, v)
		}
	}
}

func TestVolumeForwardersPanicLikeTa(t *testing.T) {
	short := make([]float64, 2)
	long := make([]float64, 3)

	cases := []struct {
		name string
		fn   func()
	}{
		{"VWAP length", func() { VWAP(short, short, short, long) }},
		{"VWMA length", func() { VWMA(short, long, 2) }},
		{"VWMA period", func() { VWMA(long, long, 0) }},
		{"OBV length", func() { OBV(short, long) }},
		{"A/D length", func() { AccumulationDistribution(short, short, short, long) }},
		{"CMF period", func() { ChaikinMoneyFlow(long, long, long, long, 0) }},
		{"ChaikinOsc period", func() { ChaikinOscillator(long, long, long, long, 3, 0) }},
		{"MFI length", func() { MoneyFlowIndex(short, short, short, long, 5) }},
		{"PVT length", func() { PriceVolumeTrend(short, long) }},
		{"VolumeOsc period", func() { VolumeOscillator(long, 0, 5) }},
		{"ForceIndex period", func() { ForceIndex(long, long, -1) }},
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

func TestVolumeForwardersEmptyInputs(t *testing.T) {
	if VWAP(nil, nil, nil, nil) != nil || OBV(nil, nil) != nil ||
		PriceVolumeTrend(nil, nil) != nil || NetVolume(nil, nil) != nil {
		t.Error("empty inputs should return nil")
	}
	if AccumulationDistribution(nil, nil, nil, nil) != nil ||
		ChaikinOscillator(nil, nil, nil, nil, 3, 10) != nil {
		t.Error("empty multi-column inputs should return nil")
	}
}
