package ta

import (
	"math"
	"testing"
)

// ---------------------------------------------------------------------------
// Tests for the rolling extremes, Donchian, ATR and the Bollinger bands.
// ---------------------------------------------------------------------------

func TestExtremesMatchReference(t *testing.T) {
	for _, n := range boundaryLengths() {
		if n == 0 {
			continue
		}
		xs := synthClose(n, int64(n))
		for _, p := range boundaryPeriods() {
			assertSeries(t, "Highest", Highest(xs, p), refExtreme(xs, p, true), false)
			assertSeries(t, "Lowest", Lowest(xs, p), refExtreme(xs, p, false), false)
		}
	}
}

func synthClose(n int, seed int64) []float64 {
	_, _, _, close, _ := synthOHLCV(n, seed)
	return close
}

func TestExtremesKnownValues(t *testing.T) {
	xs := []float64{3, 1, 4, 1, 5, 9, 2, 6}

	hi := Highest(xs, 3)
	wantHi := []float64{math.NaN(), math.NaN(), 4, 4, 5, 9, 9, 9}
	assertSeries(t, "Highest", hi, wantHi, true)

	lo := Lowest(xs, 3)
	wantLo := []float64{math.NaN(), math.NaN(), 1, 1, 1, 1, 2, 2}
	assertSeries(t, "Lowest", lo, wantLo, true)
}

// TestExtremesRecoverAfterNaNLeaves is the important test for the monotonic deque: a
// NaN must affect exactly the windows containing it and no others, which requires the
// deque's ordering to be correct again once the NaN is gone.
func TestExtremesRecoverAfterNaNLeaves(t *testing.T) {
	xs := []float64{5, 2, 8, math.NaN(), 3, 9, 1, 7}
	const n = 3

	hi := Highest(xs, n)
	lo := Lowest(xs, n)

	// Windows containing index 3 are indices 3, 4 and 5.
	for _, i := range []int{3, 4, 5} {
		if !math.IsNaN(hi[i]) {
			t.Errorf("Highest[%d] = %v, want NaN", i, hi[i])
		}
		if !math.IsNaN(lo[i]) {
			t.Errorf("Lowest[%d] = %v, want NaN", i, lo[i])
		}
	}
	// From index 6 the window {9,1,7} / {9,1,7} is NaN-free.
	if hi[6] != 9 {
		t.Errorf("Highest[6] = %v, want 9", hi[6])
	}
	if lo[6] != 1 {
		t.Errorf("Lowest[6] = %v, want 1", lo[6])
	}
	if hi[7] != 9 || lo[7] != 1 {
		t.Errorf("Highest/Lowest[7] = (%v,%v), want (9,1)", hi[7], lo[7])
	}
}

// TestExtremesLongerThanInput pins the degenerate case: a window larger than the data
// yields all NaN rather than a partial window or a panic.
func TestExtremesLongerThanInput(t *testing.T) {
	xs := []float64{1, 2, 3}
	for _, p := range []int{4, 10, 100} {
		for i, v := range Highest(xs, p) {
			if !math.IsNaN(v) {
				t.Errorf("Highest(p=%d)[%d] = %v, want NaN", p, i, v)
			}
		}
		for i, v := range Lowest(xs, p) {
			if !math.IsNaN(v) {
				t.Errorf("Lowest(p=%d)[%d] = %v, want NaN", p, i, v)
			}
		}
	}
}

func TestDonchian(t *testing.T) {
	high := []float64{10, 12, 11, 15}
	low := []float64{5, 6, 4, 7}

	upper, middle, lower := Donchian(high, low, 2)
	wantU := []float64{math.NaN(), 12, 12, 15}
	wantL := []float64{math.NaN(), 5, 4, 4}
	wantM := []float64{math.NaN(), 8.5, 8, 9.5}

	assertSeries(t, "Donchian upper", upper, wantU, true)
	assertSeries(t, "Donchian lower", lower, wantL, true)
	assertSeries(t, "Donchian middle", middle, wantM, true)

	if u, m, l := Donchian(nil, nil, 5); u != nil || m != nil || l != nil {
		t.Error("Donchian with empty input should return nils")
	}
}

func TestATRMatchesReference(t *testing.T) {
	for _, n := range boundaryLengths() {
		if n == 0 {
			continue
		}
		_, high, low, close, _ := synthOHLCV(n, int64(n)+40)
		for _, p := range boundaryPeriods() {
			assertSeries(t, "ATR", ATR(high, low, close, p), refATR(high, low, close, p), false)
		}
	}
}

// TestATRKnownValue uses a worked example to catch an error a reference that shares
// the same misunderstanding would miss.
func TestATRKnownValue(t *testing.T) {
	high := []float64{10, 11, 12}
	low := []float64{9, 10, 11}
	close := []float64{9.5, 10.5, 11.5}

	// True ranges: 1 (high-low), max(1, |11-9.5|, |10-9.5|) = 1.5,
	// max(1, |12-10.5|, |11-10.5|) = 1.5
	atr := ATR(high, low, close, 2)
	// RMA(2) seeds with the SMA of the first two true ranges: (1 + 1.5)/2 = 1.25
	if atr[1] != 1.25 {
		t.Errorf("ATR[1] = %v, want 1.25 (SMA seed of the first two true ranges)", atr[1])
	}
	// Then alpha = 1/2: 0.5*1.5 + 0.5*1.25 = 1.375
	if math.Abs(atr[2]-1.375) > 1e-15 {
		t.Errorf("ATR[2] = %v, want 1.375", atr[2])
	}
}

func TestBollingerBandsMatchReference(t *testing.T) {
	for _, n := range boundaryLengths() {
		if n == 0 {
			continue
		}
		close := synthClose(n, int64(n)+70)
		for _, p := range boundaryPeriods() {
			upper, middle, lower := BollingerBands(close, p, 2)
			wantMiddle := refSMA(close, p)
			wantSD := refStdev(close, p)
			assertSeries(t, "BB middle", middle, wantMiddle, false)
			for i := range close {
				if math.IsNaN(wantSD[i]) {
					if !math.IsNaN(upper[i]) || !math.IsNaN(lower[i]) {
						t.Fatalf("p=%d i=%d: bands should be NaN during warm-up", p, i)
					}
					continue
				}
				if !closeOrNaN(upper[i], wantMiddle[i]+2*wantSD[i]) {
					t.Fatalf("p=%d i=%d: upper = %v, want %v", p, i, upper[i], wantMiddle[i]+2*wantSD[i])
				}
				if !closeOrNaN(lower[i], wantMiddle[i]-2*wantSD[i]) {
					t.Fatalf("p=%d i=%d: lower = %v, want %v", p, i, lower[i], wantMiddle[i]-2*wantSD[i])
				}
			}
		}
	}
}

// TestBollingerUsesPopulationNotSample pins the divisor. At n=4 the two differ by a
// factor of sqrt(4/3) ~ 1.1547, which is far outside any tolerance.
func TestBollingerUsesPopulationNotSample(t *testing.T) {
	close := []float64{1, 2, 3, 4}
	upper, middle, lower := BollingerBands(close, 4, 1)

	// mean 2.5; population deviation sqrt(((1.5)^2+(0.5)^2+(0.5)^2+(1.5)^2)/4)
	// = sqrt(5/4) = sqrt(1.25); sample would be sqrt(5/3).
	sd := math.Sqrt(1.25)
	if math.Abs(middle[3]-2.5) > 1e-15 {
		t.Errorf("middle = %v, want 2.5", middle[3])
	}
	if math.Abs(upper[3]-(2.5+sd)) > 1e-12 {
		t.Errorf("upper = %v, want %v (population)", upper[3], 2.5+sd)
	}
	if math.Abs(lower[3]-(2.5-sd)) > 1e-12 {
		t.Errorf("lower = %v, want %v (population)", lower[3], 2.5-sd)
	}
}

// TestBollingerFlatWindowHasZeroWidth checks the degenerate case where every value in
// the window is equal: the bands collapse onto the middle and %B is defined as 0.
func TestBollingerFlatWindowHasZeroWidth(t *testing.T) {
	close := []float64{5, 5, 5, 5}
	upper, middle, lower := BollingerBands(close, 3, 2)
	if upper[3] != middle[3] || lower[3] != middle[3] {
		t.Errorf("flat window: upper=%v middle=%v lower=%v, want all equal",
			upper[3], middle[3], lower[3])
	}

	percentB := BBPercentB(close, 3, 2)
	if percentB[3] != 0 {
		t.Errorf("%%B on a flat window = %v, want 0", percentB[3])
	}
	width := BBWidth(close, 3, 2)
	if width[3] != 0 {
		t.Errorf("width on a flat window = %v, want 0", width[3])
	}
}

func TestBBPercentBAndWidth(t *testing.T) {
	close := []float64{1, 2, 3, 4, 5}
	const n = 3
	const k = 2

	upper, middle, lower := BollingerBands(close, n, k)
	percentB := BBPercentB(close, n, k)
	width := BBWidth(close, n, k)

	for i := n - 1; i < len(close); i++ {
		wantB := (close[i] - lower[i]) / (upper[i] - lower[i])
		if math.Abs(percentB[i]-wantB) > 1e-12 {
			t.Errorf("%%B[%d] = %v, want %v", i, percentB[i], wantB)
		}
		wantW := (upper[i] - lower[i]) / middle[i]
		if math.Abs(width[i]-wantW) > 1e-12 {
			t.Errorf("width[%d] = %v, want %v", i, width[i], wantW)
		}
	}
}

func TestVolatilityEmptyAndPanics(t *testing.T) {
	if ATR(nil, nil, nil, 5) != nil {
		t.Error("ATR(nil) should be nil")
	}
	u, m, l := BollingerBands(nil, 5, 2)
	if u != nil || m != nil || l != nil {
		t.Error("BollingerBands(nil) should return nils")
	}
	if BBPercentB(nil, 5, 2) != nil || BBWidth(nil, 5, 2) != nil {
		t.Error("BBPercentB/BBWidth(nil) should be nil")
	}

	for _, tc := range []struct {
		name string
		fn   func()
	}{
		{"ATR period", func() { ATR([]float64{1}, []float64{1}, []float64{1}, 0) }},
		{"ATR length", func() { ATR(make([]float64, 2), make([]float64, 2), make([]float64, 3), 2) }},
		{"Highest period", func() { Highest([]float64{1, 2}, 0) }},
		{"Bollinger period", func() { BollingerBands([]float64{1, 2}, 0, 2) }},
		{"Donchian period", func() { Donchian([]float64{1, 2}, []float64{1, 2}, 0) }},
		{"Donchian length", func() { Donchian(make([]float64, 2), make([]float64, 3), 2) }},
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

func TestFirstValid(t *testing.T) {
	if got := FirstValid([]float64{math.NaN(), math.NaN(), 1, 2}); got != 2 {
		t.Errorf("FirstValid = %d, want 2", got)
	}
	if got := FirstValid([]float64{math.NaN()}); got != -1 {
		t.Errorf("FirstValid all NaN = %d, want -1", got)
	}
	if got := FirstValid(nil); got != -1 {
		t.Errorf("FirstValid(nil) = %d, want -1", got)
	}
	if got := FirstValid([]float64{5}); got != 0 {
		t.Errorf("FirstValid = %d, want 0", got)
	}
}
