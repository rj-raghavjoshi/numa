package ta

import (
	"math"
	"testing"
)

// ---------------------------------------------------------------------------
// Tests for GMMA and ZigZag.
//
// GMMA is a fan-out of EMAs, so its test is that it really is twelve independent EMAs at the
// documented periods. ZigZag is the delicate one: its correctness is about *when* a pivot
// becomes visible, so the test walks a hand-traced series and asserts the confirmation bar of
// each pivot rather than only the final state.
// ---------------------------------------------------------------------------

func TestGMMAIsTwelveIndependentEMAs(t *testing.T) {
	close := synthClose(300, 5252)

	short, long := GMMA(close)

	for i := 0; i < 6; i++ {
		assertSeries(t, "GMMA short", short[i], EMA(close, gmmaShortPeriods[i]), false)
		assertSeries(t, "GMMA long", long[i], EMA(close, gmmaLongPeriods[i]), false)
	}

	// Every line must differ from the others, or the fan-out is not a fan-out.
	for i := 0; i < 6; i++ {
		for j := i + 1; j < 6; j++ {
			if sameSeries(short[i], short[j]) {
				t.Fatalf("GMMA short lines %d and %d are identical (%d and %d)",
					i, j, gmmaShortPeriods[i], gmmaShortPeriods[j])
			}
		}
	}

	// The longest line's warm-up is the indicator's.
	if got := FirstValid(long[5]); got != gmmaLongPeriods[5]-1 {
		t.Errorf("longest GMMA first valid = %d, want %d", got, gmmaLongPeriods[5]-1)
	}
}

func sameSeries(a, b []float64) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if !sameOrNaN(a[i], b[i]) {
			return false
		}
	}
	return true
}

func TestGMMAEmpty(t *testing.T) {
	s, l := GMMA(nil)
	for i := 0; i < 6; i++ {
		if s[i] != nil || l[i] != nil {
			t.Error("empty GMMA should return nil series")
		}
	}
}

// TestZigZagConfirmationTrace walks a hand-traced series.
//
// high = {10,12,15,13,11,14,18,16,12}, low = {9,11,14,12,10,13,17,15,11}, deviation 10%.
//
//	bar 0: assumed low pivot 9
//	bar 3: the running high 15 is retraced to 12 (<= 15*0.9=13.5), so 15 is confirmed
//	bar 4: the running low 10 is exceeded upward to 11 (>= 10*1.1=11), so 10 is confirmed
//	bar 7: the running high 18 is retraced to 15 (<= 18*0.9=16.2), so 18 is confirmed
func TestZigZagConfirmationTrace(t *testing.T) {
	high := []float64{10, 12, 15, 13, 11, 14, 18, 16, 12}
	low := []float64{9, 11, 14, 12, 10, 13, 17, 15, 11}

	pivot, kind := ZigZag(high, low, 0.10)

	checks := []struct {
		index int
		pivot float64
		kind  int8
		why   string
	}{
		{0, 9, -1, "the assumed initial low"},
		{2, 9, -1, "still unconfirmed before the retracement"},
		{3, 15, 1, "the high at bar 2 confirmed by the retracement at bar 3"},
		{4, 10, -1, "the low at bar 4 confirmed by the rally on the same bar"},
		{6, 10, -1, "still the last confirmed pivot"},
		{7, 18, 1, "the high at bar 6 confirmed by the retracement at bar 7"},
	}
	for _, c := range checks {
		if !closeOrNaN(pivot[c.index], c.pivot) {
			t.Errorf("pivot[%d] = %v, want %v (%s)", c.index, pivot[c.index], c.pivot, c.why)
		}
		if kind[c.index] != c.kind {
			t.Errorf("kind[%d] = %d, want %d (%s)", c.index, kind[c.index], c.kind, c.why)
		}
	}
}

// TestZigZagPivotsAlternate checks the structural invariant: a confirmed high is always
// followed by a confirmed low and vice versa, never two of the same in a row.
func TestZigZagPivotsAlternate(t *testing.T) {
	high, low, _ := synthHLC(500, 6262)
	_, kind := ZigZag(high, low, 0.02)

	last := int8(0)
	confirmations := 0
	for i := range kind {
		if kind[i] == 0 || kind[i] == last {
			continue
		}
		if last != 0 && kind[i] == last {
			t.Fatalf("two consecutive %d pivots at bar %d", kind[i], i)
		}
		last = kind[i]
		confirmations++
	}
	if confirmations < 4 {
		t.Errorf("only %d confirmations over a 500-bar walk; the threshold is too coarse for the test data", confirmations)
	}
}

// TestZigZagNeverConfirmsWithoutARetracement: a steadily rising series never retraces enough to
// confirm a high, so the assumed initial low stands for the whole series.
func TestZigZagNeverConfirmsWithoutARetracement(t *testing.T) {
	const size = 100
	high := make([]float64, size)
	low := make([]float64, size)
	for i := 0; i < size; i++ {
		high[i] = 100 + float64(i)
		low[i] = 99 + float64(i)
	}

	pivot, kind := ZigZag(high, low, 0.10)
	for i := range kind {
		if kind[i] != -1 {
			t.Fatalf("kind[%d] = %d, want -1 (no retracement ever occurs)", i, kind[i])
		}
		if pivot[i] != low[0] {
			t.Fatalf("pivot[%d] = %v, want the initial low %v", i, pivot[i], low[0])
		}
	}
}

// TestZigZagZeroDeviation confirms on every bar, which is legal but useless; the test pins that
// it does not panic or produce NaN.
func TestZigZagZeroDeviation(t *testing.T) {
	high, low, _ := synthHLC(50, 7272)
	pivot, kind := ZigZag(high, low, 0)
	for i := range pivot {
		if math.IsNaN(pivot[i]) {
			t.Fatalf("pivot[%d] = NaN with zero deviation", i)
		}
		if kind[i] != 1 && kind[i] != -1 {
			t.Fatalf("kind[%d] = %d with zero deviation, want ±1", i, kind[i])
		}
	}
}

func TestZigZagEmptyAndPanics(t *testing.T) {
	if p, k := ZigZag(nil, nil, 0.05); p != nil || k != nil {
		t.Error("empty ZigZag should return nils")
	}

	func() {
		defer func() {
			if recover() == nil {
				t.Error("ZigZag with a negative deviation did not panic")
			}
		}()
		ZigZag(make([]float64, 3), make([]float64, 3), -0.01)
	}()

	func() {
		defer func() {
			if recover() == nil {
				t.Error("ZigZag with mismatched lengths did not panic")
			}
		}()
		ZigZag(make([]float64, 2), make([]float64, 3), 0.05)
	}()
}
