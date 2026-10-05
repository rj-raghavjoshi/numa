package ta

import (
	"math"
	"testing"
)

// ---------------------------------------------------------------------------
// Tests for the volume indicators.
//
// The volume family has more degenerate cases than the others, because a bar can have
// no range (high == low), no volume, or no change in typical price. Each produces a
// different neutral value, and each is pinned here rather than left to whatever the
// arithmetic happens to do.
// ---------------------------------------------------------------------------

func refVWAP(high, low, close, volume []float64) []float64 {
	out := make([]float64, len(close))
	var pv, vol float64
	for i := range close {
		tp := (high[i] + low[i] + close[i]) / 3
		pv += tp * volume[i]
		vol += volume[i]
		if vol == 0 {
			out[i] = 0
			continue
		}
		out[i] = pv / vol
	}
	return out
}

func refVWMA(close, volume []float64, n int) []float64 {
	out := allNaN(len(close))
	for i := n - 1; i < len(close); i++ {
		var cv, v float64
		for j := i - n + 1; j <= i; j++ {
			cv += close[j] * volume[j]
			v += volume[j]
		}
		if v == 0 {
			out[i] = 0
			continue
		}
		out[i] = cv / v
	}
	return out
}

func refOBV(close, volume []float64) []float64 {
	out := make([]float64, len(close))
	var total float64
	for i := 1; i < len(close); i++ {
		if close[i] > close[i-1] {
			total += volume[i]
		} else if close[i] < close[i-1] {
			total -= volume[i]
		}
		out[i] = total
	}
	return out
}

func refAD(high, low, close, volume []float64) []float64 {
	out := make([]float64, len(close))
	var total float64
	for i := range close {
		span := high[i] - low[i]
		var mfv float64
		if span != 0 {
			mfv = ((close[i] - low[i]) - (high[i] - close[i])) / span * volume[i]
		}
		total += mfv
		out[i] = total
	}
	return out
}

func refCMF(high, low, close, volume []float64, n int) []float64 {
	out := allNaN(len(close))
	for i := n - 1; i < len(close); i++ {
		var mfv, v float64
		for j := i - n + 1; j <= i; j++ {
			span := high[j] - low[j]
			if span != 0 {
				mfv += ((close[j] - low[j]) - (high[j] - close[j])) / span * volume[j]
			}
			v += volume[j]
		}
		if v == 0 {
			out[i] = 0
			continue
		}
		out[i] = mfv / v
	}
	return out
}

func refMFI(high, low, close, volume []float64, n int) []float64 {
	out := allNaN(len(close))
	for i := n; i < len(close); i++ {
		var p, q float64
		for j := i - n + 1; j <= i; j++ {
			tp := (high[j] + low[j] + close[j]) / 3
			tpPrev := (high[j-1] + low[j-1] + close[j-1]) / 3
			raw := tp * volume[j]
			if tp > tpPrev {
				p += raw
			} else if tp < tpPrev {
				q += raw
			}
		}
		switch {
		case p == 0 && q == 0:
			out[i] = 50
		case q == 0:
			out[i] = 100
		default:
			out[i] = 100 - 100/(1+p/q)
		}
	}
	return out
}

func refPVT(close, volume []float64) []float64 {
	out := make([]float64, len(close))
	var total float64
	for i := 1; i < len(close); i++ {
		if close[i-1] != 0 {
			total += (close[i] - close[i-1]) / close[i-1] * volume[i]
		}
		out[i] = total
	}
	return out
}

// ---------------------------------------------------------------------------

func TestVWAP(t *testing.T) {
	// Constant price and volume: VWAP is that price at every bar.
	high := []float64{10, 10, 10}
	low := []float64{10, 10, 10}
	close := []float64{10, 10, 10}
	volume := []float64{100, 200, 50}
	got := VWAP(high, low, close, volume)
	for i, v := range got {
		if v != 10 {
			t.Errorf("VWAP[%d] = %v, want 10", i, v)
		}
	}

	// Two bars with different typical prices, weighted by volume.
	h2 := []float64{10, 20}
	l2 := []float64{10, 20}
	c2 := []float64{10, 20}
	v2 := []float64{1, 3}
	got2 := VWAP(h2, l2, c2, v2)
	if got2[0] != 10 {
		t.Errorf("VWAP[0] = %v, want 10", got2[0])
	}
	// (10*1 + 20*3) / 4 = 17.5
	if math.Abs(got2[1]-17.5) > 1e-12 {
		t.Errorf("VWAP[1] = %v, want 17.5", got2[1])
	}
}

func TestVWAPZeroVolume(t *testing.T) {
	got := VWAP([]float64{5}, []float64{5}, []float64{5}, []float64{0})
	if got[0] != 0 {
		t.Errorf("VWAP with zero volume = %v, want 0", got[0])
	}
}

func TestVolumeIndicatorsMatchReferences(t *testing.T) {
	for _, n := range boundaryLengths() {
		if n == 0 {
			continue
		}
		_, high, low, close, volume := synthOHLCV(n, int64(n)+808)
		// Volume must be strictly positive for the ratio tests to be meaningful;
		// synthOHLCV already guarantees that.

		assertSeries(t, "VWAP", VWAP(high, low, close, volume), refVWAP(high, low, close, volume), false)
		assertSeries(t, "OBV", OBV(close, volume), refOBV(close, volume), true)
		assertSeries(t, "A/D", AccumulationDistribution(high, low, close, volume), refAD(high, low, close, volume), false)
		assertSeries(t, "PVT", PriceVolumeTrend(close, volume), refPVT(close, volume), false)

		for _, p := range []int{2, 3, 5, 9, 14} {
			assertSeries(t, "VWMA", VWMA(close, volume, p), refVWMA(close, volume, p), false)
			assertSeries(t, "CMF", ChaikinMoneyFlow(high, low, close, volume, p), refCMF(high, low, close, volume, p), false)
			assertSeries(t, "MFI", MoneyFlowIndex(high, low, close, volume, p), refMFI(high, low, close, volume, p), false)
		}
	}
}

func TestOBVKnownValues(t *testing.T) {
	close := []float64{1, 2, 1, 2, 2}
	volume := []float64{10, 20, 30, 40, 50}
	got := OBV(close, volume)
	// 0; +20 = 20; -30 = -10; +40 = 30; unchanged = 30
	want := []float64{0, 20, -10, 30, 30}
	assertSeries(t, "OBV", got, want, true)
}

func TestNetVolumeKnownValues(t *testing.T) {
	close := []float64{1, 2, 2, 1}
	volume := []float64{10, 20, 30, 40}
	got := NetVolume(close, volume)
	want := []float64{0, 20, 0, -40}
	assertSeries(t, "NetVolume", got, want, true)
}

// TestCMFExtremes pins the meaning of the indicator: a close pinned to the high of
// every bar is maximum accumulation (+1) and to the low is maximum distribution (-1).
func TestCMFExtremes(t *testing.T) {
	const n = 15
	high := make([]float64, n)
	low := make([]float64, n)
	closeAtHigh := make([]float64, n)
	closeAtLow := make([]float64, n)
	volume := make([]float64, n)
	for i := 0; i < n; i++ {
		high[i] = 11
		low[i] = 9
		closeAtHigh[i] = 11
		closeAtLow[i] = 9
		volume[i] = 100
	}

	atHigh := ChaikinMoneyFlow(high, low, closeAtHigh, volume, 5)
	if math.Abs(atHigh[n-1]-1) > 1e-12 {
		t.Errorf("CMF with the close at the high = %v, want +1", atHigh[n-1])
	}
	atLow := ChaikinMoneyFlow(high, low, closeAtLow, volume, 5)
	if math.Abs(atLow[n-1]+1) > 1e-12 {
		t.Errorf("CMF with the close at the low = %v, want -1", atLow[n-1])
	}
}

// TestMoneyFlowVolumeFlatBarIsZero pins the high == low case, which would otherwise
// divide by zero.
func TestMoneyFlowVolumeFlatBarIsZero(t *testing.T) {
	got := AccumulationDistribution([]float64{5}, []float64{5}, []float64{5}, []float64{1000})
	if got[0] != 0 {
		t.Errorf("A/D for a bar with no range = %v, want 0", got[0])
	}
}

// TestMFIDegenerateCases pins the three neutral values.
func TestMFIDegenerateCases(t *testing.T) {
	// Strictly rising typical price: all flow positive -> 100.
	n := 10
	high := make([]float64, n)
	low := make([]float64, n)
	close := make([]float64, n)
	volume := make([]float64, n)
	for i := 0; i < n; i++ {
		p := 10 + float64(i)
		high[i], low[i], close[i] = p+0.5, p-0.5, p
		volume[i] = 100
	}
	rising := MoneyFlowIndex(high, low, close, volume, 5)
	if rising[n-1] != 100 {
		t.Errorf("MFI of a rising series = %v, want 100", rising[n-1])
	}

	// Flat typical price: no flow either way -> 50, not 100.
	for i := range high {
		high[i], low[i], close[i] = 10.5, 9.5, 10
	}
	flat := MoneyFlowIndex(high, low, close, volume, 5)
	if flat[n-1] != 50 {
		t.Errorf("MFI of a flat series = %v, want 50", flat[n-1])
	}
}

func TestMFIWithinRange(t *testing.T) {
	_, high, low, close, volume := synthOHLCV(200, 909)
	got := MoneyFlowIndex(high, low, close, volume, 14)
	for i, v := range got {
		if math.IsNaN(v) {
			continue
		}
		if v < -1e-9 || v > 100+1e-9 {
			t.Fatalf("MFI[%d] = %v, outside [0,100]", i, v)
		}
	}
}

func TestPVTKnownValue(t *testing.T) {
	close := []float64{10, 11}
	volume := []float64{0, 100}
	got := PriceVolumeTrend(close, volume)
	if got[0] != 0 {
		t.Errorf("PVT[0] = %v, want 0", got[0])
	}
	// (11-10)/10 * 100 = 10
	if math.Abs(got[1]-10) > 1e-12 {
		t.Errorf("PVT[1] = %v, want 10", got[1])
	}
}

func TestVolumeOscillator(t *testing.T) {
	// Constant volume: every average is equal, so the oscillator is 0.
	flat := make([]float64, 30)
	for i := range flat {
		flat[i] = 500
	}
	got := VolumeOscillator(flat, 5, 20)
	for i := 19; i < len(flat); i++ {
		if got[i] != 0 {
			t.Fatalf("VolumeOscillator on constant volume[%d] = %v, want 0", i, got[i])
		}
	}

	// Rising volume: the fast average leads, so the oscillator is positive.
	rising := make([]float64, 30)
	for i := range rising {
		rising[i] = float64(i + 1)
	}
	got = VolumeOscillator(rising, 5, 20)
	if got[29] <= 0 {
		t.Errorf("VolumeOscillator on rising volume = %v, want positive", got[29])
	}
}

func TestChaikinOscillatorIsDifferenceOfEMAs(t *testing.T) {
	_, high, low, close, volume := synthOHLCV(120, 1010)
	const fast, slow = 3, 10

	ad := AccumulationDistribution(high, low, close, volume)
	fe := EMA(ad, fast)
	se := EMA(ad, slow)
	got := ChaikinOscillator(high, low, close, volume, fast, slow)

	for i := range ad {
		if !closeOrNaN(got[i], fe[i]-se[i]) {
			t.Fatalf("ChaikinOscillator[%d] = %v, want %v", i, got[i], fe[i]-se[i])
		}
	}
	// It must not be defined before the slow EMA is.
	if !math.IsNaN(got[slow-2]) {
		t.Errorf("ChaikinOscillator defined one bar too early: %v", got[slow-2])
	}
}

func TestForceIndexWarmup(t *testing.T) {
	close := synthClose(60, 1111)
	volume := make([]float64, len(close))
	for i := range volume {
		volume[i] = 1000
	}
	const n = 13

	got := ForceIndex(close, volume, n)
	// The first raw value is undefined, so the smoother starts at index 1 and its
	// own warm-up pushes the first output to index n.
	if first := FirstValid(got); first != n {
		t.Errorf("ForceIndex first valid = %d, want %d", first, n)
	}
	for i := 0; i < n; i++ {
		if !math.IsNaN(got[i]) {
			t.Errorf("ForceIndex[%d] = %v, want NaN", i, got[i])
		}
	}
	for i := n; i < len(got); i++ {
		if math.IsNaN(got[i]) {
			t.Fatalf("ForceIndex[%d] = NaN after its warm-up", i)
		}
	}
}

func TestVolumeIndicatorsEmptyAndPanics(t *testing.T) {
	empty := func(f func() []float64) {
		if f() != nil {
			t.Error("empty input should return nil")
		}
	}
	empty(func() []float64 { return VWAP(nil, nil, nil, nil) })
	empty(func() []float64 { return VWMA(nil, nil, 5) })
	empty(func() []float64 { return OBV(nil, nil) })
	empty(func() []float64 { return AccumulationDistribution(nil, nil, nil, nil) })
	empty(func() []float64 { return ChaikinMoneyFlow(nil, nil, nil, nil, 5) })
	empty(func() []float64 { return ChaikinOscillator(nil, nil, nil, nil, 3, 10) })
	empty(func() []float64 { return MoneyFlowIndex(nil, nil, nil, nil, 5) })
	empty(func() []float64 { return PriceVolumeTrend(nil, nil) })
	empty(func() []float64 { return NetVolume(nil, nil) })
	empty(func() []float64 { return VolumeOscillator(nil, 5, 10) })
	empty(func() []float64 { return ForceIndex(nil, nil, 5) })

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
		{"ChaikinOsc period", func() { ChaikinOscillator(long, long, long, long, 0, 5) }},
		{"MFI period", func() { MoneyFlowIndex(long, long, long, long, -1) }},
		{"PVT length", func() { PriceVolumeTrend(short, long) }},
		{"VolumeOsc slow", func() { VolumeOscillator(long, 5, 0) }},
		{"ForceIndex period", func() { ForceIndex(long, long, 0) }},
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

// TestVWMAFlatVolumeIsZero pins the zero-volume window case.
func TestVWMAFlatVolumeIsZero(t *testing.T) {
	close := []float64{10, 20, 30}
	volume := []float64{0, 0, 0}
	got := VWMA(close, volume, 2)
	if got[2] != 0 {
		t.Errorf("VWMA with zero volume in the window = %v, want 0", got[2])
	}
}
