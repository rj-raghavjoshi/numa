package ta

import (
	"math"
	"testing"
)

// ---------------------------------------------------------------------------
// Tests for the envelopes, channels and the price oscillator.
//
// Each of these is a thin wrapper around a moving average, so the tests are mostly about the
// relationship to that average: the envelope's band must be the stated fraction of the middle line,
// the channel must be the averages of the two sides, and the oscillator must reduce to zero when
// the two averages coincide.
// ---------------------------------------------------------------------------

func TestMovingAverageEnvelope(t *testing.T) {
	xs := synthClose(150, 1212)
	const n = 20
	const pct = 0.05

	upper, middle, lower := MovingAverageEnvelope(xs, n, pct)
	sma := SMA(xs, n)

	for i := range xs {
		if !closeOrNaN(middle[i], sma[i]) {
			t.Fatalf("middle[%d] = %v, want SMA %v", i, middle[i], sma[i])
		}
		if math.IsNaN(sma[i]) {
			continue
		}
		if !closeOrNaN(upper[i], sma[i]*(1+pct)) {
			t.Fatalf("upper[%d] = %v, want %v", i, upper[i], sma[i]*(1+pct))
		}
		if !closeOrNaN(lower[i], sma[i]*(1-pct)) {
			t.Fatalf("lower[%d] = %v, want %v", i, lower[i], sma[i]*(1-pct))
		}
		if upper[i] < middle[i] || middle[i] < lower[i] {
			t.Fatalf("envelope is not ordered at %d: %v %v %v", i, upper[i], middle[i], lower[i])
		}
	}
}

// TestMovingAverageEnvelopeZeroPercentCollapses pins the degenerate width: the three lines coincide.
func TestMovingAverageEnvelopeZeroPercentCollapses(t *testing.T) {
	xs := synthClose(100, 1313)
	upper, middle, lower := MovingAverageEnvelope(xs, 10, 0)
	assertSeries(t, "upper", upper, middle, false)
	assertSeries(t, "lower", lower, middle, false)
}

// TestMovingAverageEnvelopeConstantSeries pins the fixed point: a flat series has a flat band.
func TestMovingAverageEnvelopeConstantSeries(t *testing.T) {
	const size = 60
	const c = 20.0
	xs := make([]float64, size)
	for i := range xs {
		xs[i] = c
	}
	upper, middle, lower := MovingAverageEnvelope(xs, 10, 0.02)
	for i := 9; i < size; i++ {
		if math.Abs(middle[i]-c) > 1e-12 || math.Abs(upper[i]-c*1.02) > 1e-12 || math.Abs(lower[i]-c*0.98) > 1e-12 {
			t.Fatalf("envelope of a constant at %d = (%v,%v,%v)", i, upper[i], middle[i], lower[i])
		}
	}
}

func TestMovingAverageChannel(t *testing.T) {
	high, low, _ := synthHLC(150, 1414)
	const n = 15

	upper, lower := MovingAverageChannel(high, low, n)
	wantUpper := SMA(high, n)
	wantLower := SMA(low, n)
	assertSeries(t, "MAC upper", upper, wantUpper, false)
	assertSeries(t, "MAC lower", lower, wantLower, false)

	// The highs must average above the lows on every bar, so the channel is never inverted.
	for i := n - 1; i < len(high); i++ {
		if upper[i] <= lower[i] {
			t.Fatalf("channel is inverted at %d: upper %v <= lower %v", i, upper[i], lower[i])
		}
	}
	if got := FirstValid(upper); got != n-1 {
		t.Errorf("channel first valid = %d, want %d", got, n-1)
	}
}

func TestPriceOscillator(t *testing.T) {
	xs := synthClose(200, 1515)
	const fast, slow = 10, 30

	got := PriceOscillator(xs, fast, slow)
	fastMA := SMA(xs, fast)
	slowMA := SMA(xs, slow)
	for i := range xs {
		if math.IsNaN(slowMA[i]) {
			continue
		}
		want := 100 * (fastMA[i] - slowMA[i]) / slowMA[i]
		if !closeOrNaN(got[i], want) {
			t.Fatalf("PriceOscillator[%d] = %v, want %v", i, got[i], want)
		}
	}
	if got := FirstValid(got); got != slow-1 {
		t.Errorf("PriceOscillator first valid = %d, want %d", got, slow-1)
	}
}

// TestPriceOscillatorConstantSeriesIsZero: equal averages give a zero difference.
func TestPriceOscillatorConstantSeriesIsZero(t *testing.T) {
	const size = 80
	xs := make([]float64, size)
	for i := range xs {
		xs[i] = 17
	}
	got := PriceOscillator(xs, 5, 20)
	for i := 19; i < size; i++ {
		if math.Abs(got[i]) > 1e-12 {
			t.Fatalf("PriceOscillator of a constant[%d] = %v, want 0", i, got[i])
		}
	}
}

// TestPriceOscillatorIsPositiveWhenFastLeads checks the sign convention on a rising series, where
// the faster average must be above the slower one.
func TestPriceOscillatorIsPositiveWhenFastLeads(t *testing.T) {
	const size = 100
	xs := make([]float64, size)
	for i := range xs {
		xs[i] = float64(i)
	}
	got := PriceOscillator(xs, 5, 20)
	for i := 19; i < size; i++ {
		if got[i] <= 0 {
			t.Fatalf("PriceOscillator on a rising series[%d] = %v, want positive", i, got[i])
		}
	}
}

func TestWeightedClose(t *testing.T) {
	high, low, close := synthHLC(120, 1616)
	got := WeightedClose(high, low, close)

	for i := range close {
		want := (high[i] + low[i] + 2*close[i]) / 4
		if !closeOrNaN(got[i], want) {
			t.Fatalf("WeightedClose[%d] = %v, want %v", i, got[i], want)
		}
		// It is a convex combination of values inside the bar, so it stays inside the bar.
		if got[i] < low[i]-1e-12 || got[i] > high[i]+1e-12 {
			t.Fatalf("WeightedClose[%d] = %v, outside the bar [%v,%v]", i, got[i], low[i], high[i])
		}
	}
}

// TestWeightedCloseHandComputed pins one bar by hand: H=10, L=6, C=9 -> (10+6+18)/4 = 8.5, which is
// above the typical price (25/3 = 8.33) because the close is weighted twice.
func TestWeightedCloseHandComputed(t *testing.T) {
	got := WeightedClose([]float64{10}, []float64{6}, []float64{9})
	if len(got) != 1 || math.Abs(got[0]-8.5) > 1e-12 {
		t.Fatalf("WeightedClose = %v, want [8.5]", got)
	}
	typical := TypicalPrice([]float64{10}, []float64{6}, []float64{9})[0]
	if math.Abs(typical-25.0/3) > 1e-12 {
		t.Fatalf("TypicalPrice = %v, want 25/3", typical)
	}
	if got[0] <= typical {
		t.Error("WeightedClose should exceed TypicalPrice when the close is above the midpoint")
	}
}

func TestEnvelopeIndicatorsEmptyAndPanics(t *testing.T) {
	if u, m, l := MovingAverageEnvelope(nil, 10, 0.05); u != nil || m != nil || l != nil {
		t.Error("empty MovingAverageEnvelope should return nils")
	}
	if u, l := MovingAverageChannel(nil, nil, 10); u != nil || l != nil {
		t.Error("empty MovingAverageChannel should return nils")
	}
	if PriceOscillator(nil, 5, 20) != nil {
		t.Error("empty PriceOscillator should be nil")
	}
	if WeightedClose(nil, nil, nil) != nil {
		t.Error("empty WeightedClose should be nil")
	}

	xs := synthClose(30, 17)
	high, low, _ := synthHLC(30, 18)
	short := make([]float64, 20)
	for _, tc := range []struct {
		name string
		fn   func()
	}{
		{"envelope period", func() { MovingAverageEnvelope(xs, 0, 0.05) }},
		{"envelope negative percent", func() { MovingAverageEnvelope(xs, 10, -0.01) }},
		{"channel period", func() { MovingAverageChannel(high, low, 0) }},
		{"channel length", func() { MovingAverageChannel(short, low, 5) }},
		{"oscillator period", func() { PriceOscillator(xs, 0, 20) }},
		{"oscillator reversed", func() { PriceOscillator(xs, 20, 20) }},
		{"oscillator fast > slow", func() { PriceOscillator(xs, 30, 20) }},
		{"weighted close length", func() { WeightedClose(short, low, xs) }},
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
