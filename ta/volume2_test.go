package ta

import (
	"math"
	"testing"

	"github.com/rj-raghavjoshi/numa/series"
)

// ---------------------------------------------------------------------------
// Tests for the volume and stop indicators.
//
// The stop test is written around the property the formula actually guarantees, not the one it is
// commonly believed to guarantee: the second pass is a rolling extreme, so it cannot be claimed to
// ratchet monotonically. Asserting monotonicity would have been a test that fails on correct code.
// ---------------------------------------------------------------------------

func TestBalanceOfPower(t *testing.T) {
	open, high, low, close, _ := synthOHLCV(200, 2020)
	const n = 14

	got := BalanceOfPower(open, high, low, close, n)
	raw := make([]float64, len(close))
	for i := range close {
		rng := high[i] - low[i]
		if rng == 0 {
			raw[i] = 0
			continue
		}
		raw[i] = (close[i] - open[i]) / rng
	}
	assertSeries(t, "BalanceOfPower", got, SMA(raw, n), false)

	// It is a mean of values in [-1,1], so it stays in [-1,1].
	for i, v := range got {
		if math.IsNaN(v) {
			continue
		}
		if v < -1-1e-12 || v > 1+1e-12 {
			t.Fatalf("BalanceOfPower[%d] = %v, outside [-1,1]", i, v)
		}
	}
}

// TestBalanceOfPowerExtremes pins the two ends: a bar that opens at its low and closes at its high
// is +1, and the reverse is -1.
func TestBalanceOfPowerExtremes(t *testing.T) {
	open := []float64{1, 10}
	high := []float64{10, 10}
	low := []float64{1, 1}
	close := []float64{10, 1}

	got := BalanceOfPower(open, high, low, close, 1)
	if math.Abs(got[0]-1) > 1e-12 {
		t.Errorf("bar closing at its high = %v, want 1", got[0])
	}
	if math.Abs(got[1]+1) > 1e-12 {
		t.Errorf("bar closing at its low = %v, want -1", got[1])
	}

	// A bar with no range has no balance to measure.
	flat := BalanceOfPower([]float64{5}, []float64{5}, []float64{5}, []float64{5}, 1)
	if flat[0] != 0 {
		t.Errorf("zero-range bar = %v, want 0", flat[0])
	}
}

func TestEaseOfMovement(t *testing.T) {
	high, low, volume := func() ([]float64, []float64, []float64) {
		_, h, l, _, v := synthOHLCV(200, 3030)
		return h, l, v
	}()
	const n = 14

	got := EaseOfMovement(high, low, volume, n)

	ease := allNaN(len(high))
	for i := 1; i < len(high); i++ {
		rng := high[i] - low[i]
		mid := (high[i] + low[i]) / 2
		prevMid := (high[i-1] + low[i-1]) / 2
		if rng == 0 || volume[i] == 0 {
			ease[i] = 0
			continue
		}
		ease[i] = (mid - prevMid) * rng / volume[i]
	}
	assertSeries(t, "EaseOfMovement", got, SMA(ease, n), false)

	if got := FirstValid(got); got != n {
		t.Errorf("EaseOfMovement first valid = %d, want %d", got, n)
	}
}

// TestEaseOfMovementZeroVolumeIsZero pins the degenerate divisor.
func TestEaseOfMovementZeroVolumeIsZero(t *testing.T) {
	high := []float64{10, 11, 12}
	low := []float64{9, 10, 11}
	volume := []float64{100, 0, 100}
	got := EaseOfMovement(high, low, volume, 1)
	if got[1] != 0 {
		t.Errorf("a bar with no volume gave %v, want 0", got[1])
	}
}

func refKlinger(high, low, close, volume []float64, fast, slow, signalPeriod int) (kvo, signal []float64) {
	size := len(close)
	vf := allNaN(size)
	prevTrend := 0.0
	var cm float64
	for i := 1; i < size; i++ {
		tp := high[i] + low[i] + close[i]
		prevTP := high[i-1] + low[i-1] + close[i-1]
		trend := 1.0
		if tp < prevTP {
			trend = -1
		}
		dm := high[i] - low[i]
		prevDM := high[i-1] - low[i-1]
		switch {
		case i == 1 || prevTrend == 0:
			cm = dm
		case trend == prevTrend:
			cm += dm
		default:
			cm = prevDM + dm
		}
		prevTrend = trend
		if cm == 0 {
			vf[i] = 0
			continue
		}
		vf[i] = volume[i] * math.Abs(2*(dm/cm-1)) * trend * 100
	}
	from := 1
	fe := applyFrom(vf, from, series.NewEMA(fast))
	se := applyFrom(vf, from, series.NewEMA(slow))
	kvo = make([]float64, size)
	for i := range close {
		kvo[i] = fe[i] - se[i]
	}
	signal = applyFrom(kvo, FirstValid(kvo), series.NewEMA(signalPeriod))
	return kvo, signal
}

func TestKlingerOscillatorMatchesReference(t *testing.T) {
	_, high, low, close, volume := synthOHLCV(300, 4040)
	gotKVO, gotSignal := KlingerOscillator(high, low, close, volume, 34, 55, 13)
	wantKVO, wantSignal := refKlinger(high, low, close, volume, 34, 55, 13)
	assertSeries(t, "KVO", gotKVO, wantKVO, false)
	assertSeries(t, "KVO signal", gotSignal, wantSignal, false)
}

func TestChaikinVolatility(t *testing.T) {
	high, low := func() ([]float64, []float64) {
		_, h, l, _, _ := synthOHLCV(300, 5050)
		return h, l
	}()
	const n = 10

	got := ChaikinVolatility(high, low, n)
	rng := make([]float64, len(high))
	for i := range high {
		rng[i] = high[i] - low[i]
	}
	smoothed := EMA(rng, n)
	for i := 2*n - 1; i < len(high); i++ {
		base := smoothed[i-n]
		want := 0.0
		if base != 0 {
			want = 100 * (smoothed[i] - base) / base
		}
		if !closeOrNaN(got[i], want) {
			t.Fatalf("ChaikinVolatility[%d] = %v, want %v", i, got[i], want)
		}
	}
	if first := FirstValid(got); first != 2*n-1 {
		t.Errorf("ChaikinVolatility first valid = %d, want %d", first, 2*n-1)
	}
}

// TestChaikinVolatilityConstantRangeIsZero: a range that never changes has zero rate of change.
func TestChaikinVolatilityConstantRangeIsZero(t *testing.T) {
	const size = 100
	const width = 2.0
	high := make([]float64, size)
	low := make([]float64, size)
	for i := range high {
		high[i] = 100 + float64(i)
		low[i] = high[i] - width
	}
	got := ChaikinVolatility(high, low, 10)
	for i, v := range got {
		if math.IsNaN(v) {
			continue
		}
		if math.Abs(v) > 1e-9 {
			t.Fatalf("ChaikinVolatility with a constant range[%d] = %v, want 0", i, v)
		}
	}
}

func TestChandeKrollStop(t *testing.T) {
	high, low, close := synthHLC(300, 6060)
	const n, q = 10, 9
	const x = 1.0

	stopLong, stopShort := ChandeKrollStop(high, low, close, n, x, q)

	atr := ATR(high, low, close, n)
	hi := Highest(high, n)
	lo := Lowest(low, n)
	firstLong := allNaN(len(close))
	firstShort := allNaN(len(close))
	for i := range close {
		if atr[i] != atr[i] || hi[i] != hi[i] || lo[i] != lo[i] {
			continue
		}
		firstLong[i] = hi[i] - x*atr[i]
		firstShort[i] = lo[i] + x*atr[i]
	}
	assertSeries(t, "stopLong", stopLong, Highest(firstLong, q), false)
	assertSeries(t, "stopShort", stopShort, Lowest(firstShort, q), false)

	// The guaranteed property: the second pass contains the current first-pass value in its window,
	// so the long stop is never below it and the short stop never above it.
	for i := range close {
		if math.IsNaN(firstLong[i]) || math.IsNaN(stopLong[i]) {
			continue
		}
		if stopLong[i] < firstLong[i]-1e-9 {
			t.Fatalf("long stop %v is below the first-pass %v at %d", stopLong[i], firstLong[i], i)
		}
		if stopShort[i] > firstShort[i]+1e-9 {
			t.Fatalf("short stop %v is above the first-pass %v at %d", stopShort[i], firstShort[i], i)
		}
	}

	if first := FirstValid(stopLong); first != n+q-2 {
		t.Errorf("ChandeKrollStop first valid = %d, want %d", first, n+q-2)
	}
}

func TestFiftyTwoWeekHighLow(t *testing.T) {
	high, low := func() ([]float64, []float64) {
		_, h, l, _, _ := synthOHLCV(300, 7070)
		return h, l
	}()
	const bars = 252

	gotHigh, gotLow := FiftyTwoWeekHighLow(high, low, bars)
	assertSeries(t, "52w high", gotHigh, Highest(high, bars), false)
	assertSeries(t, "52w low", gotLow, Lowest(low, bars), false)

	for i := bars - 1; i < len(high); i++ {
		if gotHigh[i] < gotLow[i] {
			t.Fatalf("52-week high %v is below the low %v at %d", gotHigh[i], gotLow[i], i)
		}
	}
}

func TestVolume2EmptyAndPanics(t *testing.T) {
	_, high, low, close, volume := synthOHLCV(40, 8080)
	open := make([]float64, 40)

	if BalanceOfPower(nil, nil, nil, nil, 14) != nil {
		t.Error("empty BalanceOfPower should be nil")
	}
	if EaseOfMovement(nil, nil, nil, 14) != nil {
		t.Error("empty EaseOfMovement should be nil")
	}
	if k, s := KlingerOscillator(nil, nil, nil, nil, 34, 55, 13); k != nil || s != nil {
		t.Error("empty KlingerOscillator should return nils")
	}
	if ChaikinVolatility(nil, nil, 10) != nil {
		t.Error("empty ChaikinVolatility should be nil")
	}
	if l, s := ChandeKrollStop(nil, nil, nil, 10, 1, 9); l != nil || s != nil {
		t.Error("empty ChandeKrollStop should return nils")
	}
	if h, l := FiftyTwoWeekHighLow(nil, nil, 252); h != nil || l != nil {
		t.Error("empty FiftyTwoWeekHighLow should return nils")
	}

	short := make([]float64, 2)
	long := make([]float64, 3)
	for _, tc := range []struct {
		name string
		fn   func()
	}{
		{"BOP period", func() { BalanceOfPower(open, high, low, close, 0) }},
		{"BOP length", func() { BalanceOfPower(short, high, low, long, 5) }},
		{"EOM period", func() { EaseOfMovement(high, low, volume, 0) }},
		{"EOM length", func() { EaseOfMovement(short, low, long, 5) }},
		{"KVO reversed", func() { KlingerOscillator(high, low, close, volume, 55, 34, 13) }},
		{"KVO period", func() { KlingerOscillator(high, low, close, volume, 34, 55, 0) }},
		{"Chaikin period", func() { ChaikinVolatility(high, low, 0) }},
		{"Chaikin length", func() { ChaikinVolatility(short, long, 10) }},
		{"CKS period", func() { ChandeKrollStop(high, low, close, 0, 1, 9) }},
		{"CKS negative x", func() { ChandeKrollStop(high, low, close, 10, -1, 9) }},
		{"CKS length", func() { ChandeKrollStop(short, short, long, 10, 1, 9) }},
		{"52w period", func() { FiftyTwoWeekHighLow(high, low, 0) }},
		{"52w length", func() { FiftyTwoWeekHighLow(short, long, 10) }},
	} {
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
