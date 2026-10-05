package ta

import (
	"math"
	"testing"
)

// ---------------------------------------------------------------------------
// Tests for the price transforms.
// ---------------------------------------------------------------------------

func TestPriceTransforms(t *testing.T) {
	open := []float64{10, 20}
	high := []float64{12, 24}
	low := []float64{8, 16}
	close := []float64{11, 22}

	median := MedianPrice(high, low)
	if median[0] != 10 || median[1] != 20 {
		t.Errorf("MedianPrice = %v, want [10 20]", median)
	}

	typical := TypicalPrice(high, low, close)
	if typical[0] != (12+8+11)/3.0 {
		t.Errorf("TypicalPrice[0] = %v", typical[0])
	}

	average := AveragePrice(open, high, low, close)
	if average[0] != (10+12+8+11)/4.0 {
		t.Errorf("AveragePrice[0] = %v", average[0])
	}
}

func TestPriceTransformsEmptyAndPanics(t *testing.T) {
	if MedianPrice(nil, nil) != nil || TypicalPrice(nil, nil, nil) != nil ||
		AveragePrice(nil, nil, nil, nil) != nil || TrueRange(nil, nil, nil) != nil {
		t.Error("empty inputs should return nil")
	}

	cases := []struct {
		name string
		fn   func()
	}{
		{"MedianPrice", func() { MedianPrice(make([]float64, 2), make([]float64, 3)) }},
		{"TypicalPrice", func() { TypicalPrice(make([]float64, 2), make([]float64, 2), make([]float64, 3)) }},
		{"AveragePrice", func() { AveragePrice(make([]float64, 3), make([]float64, 3), make([]float64, 3), make([]float64, 2)) }},
		{"TrueRange", func() { TrueRange(make([]float64, 3), make([]float64, 3), make([]float64, 2)) }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			defer func() {
				if recover() == nil {
					t.Errorf("%s did not panic on a length mismatch", tc.name)
				}
			}()
			tc.fn()
		})
	}
}

// TestTrueRangeKnownValues pins the definition, including the deliberate choice for
// the first bar where no previous close exists.
func TestTrueRangeKnownValues(t *testing.T) {
	high := []float64{10, 12, 13}
	low := []float64{8, 9, 11}
	close := []float64{9, 11, 11}

	tr := TrueRange(high, low, close)
	// bar 0: no previous close -> high-low = 2
	if tr[0] != 2 {
		t.Errorf("TrueRange[0] = %v, want 2 (high-low)", tr[0])
	}
	// bar 1: max(3, |12-9|=3, |9-9|=0) = 3
	if tr[1] != 3 {
		t.Errorf("TrueRange[1] = %v, want 3", tr[1])
	}
	// bar 2: max(2, |13-11|=2, |11-11|=0) = 2
	if tr[2] != 2 {
		t.Errorf("TrueRange[2] = %v, want 2", tr[2])
	}
}

func TestChangeDelegatesToDiff(t *testing.T) {
	xs := []float64{1, 3, 6, 10}
	got := Change(xs, 1)
	if !math.IsNaN(got[0]) || got[1] != 2 || got[2] != 3 || got[3] != 4 {
		t.Errorf("Change = %v, want [NaN 2 3 4]", got)
	}
	if Change(nil, 1) != nil {
		t.Error("Change(nil) should be nil")
	}
}
