package ta

import (
	"math"
	"testing"
)

// ---------------------------------------------------------------------------
// Tests for the Swing Index and its accumulation.
//
// The formula has three branch conditions and a four-term numerator, so a reference written from the
// same description would share any misreading of it. The test therefore pins two things independently:
// a hand-computed bar where every term is worked out on paper, and the structural relation between the
// two exported functions, which must differ only by accumulation.
// ---------------------------------------------------------------------------

// refSwingBar computes the index for one bar directly from the description, with no shared code.
func refSwingBar(o1, h1, l1, c1, o, h, l, c, limitMove float64) float64 {
	_ = l1
	k := math.Max(h, c1) - math.Min(l, c1)
	absHC := math.Abs(h - c1)
	absLC := math.Abs(l - c1)
	absCO := math.Abs(c1 - o1)
	barRange := h - l
	var r float64
	switch {
	case absHC >= absLC && absHC >= barRange:
		r = absHC - 0.5*absLC + 0.25*absCO
	case absLC >= absHC && absLC >= barRange:
		r = absLC - 0.5*absHC + 0.25*absCO
	default:
		r = barRange + 0.25*absCO
	}
	if r == 0 {
		return 0
	}
	return 50 * (c - c1 + 0.5*(c-o) + 0.25*(c1-o1)) / r * (k / limitMove)
}

func TestSwingIndexHandComputed(t *testing.T) {
	// Bar 0: O=10, H=12, L=9,  C=11
	// Bar 1: O=11, H=14, L=10, C=13
	//
	// K       = max(14, 11) - min(10, 11) = 14 - 10 = 4
	// |H-C1|  = 3,  |L-C1| = 1,  range = 4,  |C1-O1| = 1
	// The bar's own range (4) dominates, so R = 4 + 0.25*1 = 4.25
	// numerator = (13-11) + 0.5*(13-11) + 0.25*(11-10) = 2 + 1 + 0.25 = 3.25
	// SI = 50 * 3.25 / 4.25 * (4 / limitMove)
	open := []float64{10, 11}
	high := []float64{12, 14}
	low := []float64{9, 10}
	close := []float64{11, 13}
	const limitMove = 4.0

	got := SwingIndex(open, high, low, close, limitMove)
	want := 50 * 3.25 / 4.25 * (4.0 / limitMove)
	if math.Abs(got[1]-want) > 1e-12 {
		t.Fatalf("SwingIndex[1] = %v, want %v", got[1], want)
	}
	if got[0] != 0 {
		t.Errorf("SwingIndex[0] = %v, want the seed 0", got[0])
	}

	// And the accumulation must be exactly the running total.
	asi := AccumulativeSwingIndex(open, high, low, close, limitMove)
	if asi[0] != 0 {
		t.Errorf("ASI[0] = %v, want 0", asi[0])
	}
	if math.Abs(asi[1]-want) > 1e-12 {
		t.Fatalf("ASI[1] = %v, want %v", asi[1], want)
	}
}

func TestSwingIndexMatchesReference(t *testing.T) {
	open, high, low, close, _ := synthOHLCV(200, 3131)
	for _, limitMove := range []float64{1, 3, 12.5} {
		got := SwingIndex(open, high, low, close, limitMove)
		for i := 1; i < len(close); i++ {
			want := refSwingBar(open[i-1], high[i-1], low[i-1], close[i-1],
				open[i], high[i], low[i], close[i], limitMove)
			if math.Abs(got[i]-want) > 1e-9 {
				t.Fatalf("limitMove=%v: SwingIndex[%d] = %v, want %v", limitMove, i, got[i], want)
			}
		}
	}
}

// TestAccumulativeSwingIndexIsTheRunningTotal pins the relation between the two exported functions,
// which is the one thing a shared-formula implementation could still get wrong.
func TestAccumulativeSwingIndexIsTheRunningTotal(t *testing.T) {
	open, high, low, close, _ := synthOHLCV(150, 4141)
	const limitMove = 3.0

	si := SwingIndex(open, high, low, close, limitMove)
	asi := AccumulativeSwingIndex(open, high, low, close, limitMove)

	var total float64
	for i := range si {
		total += si[i]
		if math.Abs(asi[i]-total) > 1e-9 {
			t.Fatalf("ASI[%d] = %v, want the running total %v", i, asi[i], total)
		}
	}
}

// TestSwingIndexScalesWithTheLimitMove pins limitMove's role: it divides, so doubling it halves every
// increment.
func TestSwingIndexScalesWithTheLimitMove(t *testing.T) {
	open, high, low, close, _ := synthOHLCV(100, 5151)

	a := SwingIndex(open, high, low, close, 4)
	b := SwingIndex(open, high, low, close, 8)
	for i := 1; i < len(close); i++ {
		if math.Abs(a[i]-2*b[i]) > 1e-9 {
			t.Fatalf("doubling limitMove did not halve the index at %d: %v vs %v", i, a[i], b[i])
		}
	}
}

// TestSwingIndexFlatBarContributesZero: a bar with no range, no body and no gap in any direction has
// no displacement, so R is zero and the documented answer is 0 rather than a division by zero.
func TestSwingIndexFlatBarContributesZero(t *testing.T) {
	const p = 25.0
	open := []float64{p, p}
	high := []float64{p, p}
	low := []float64{p, p}
	close := []float64{p, p}

	si := SwingIndex(open, high, low, close, 3)
	if si[1] != 0 {
		t.Errorf("SwingIndex of a flat bar = %v, want 0", si[1])
	}
	asi := AccumulativeSwingIndex(open, high, low, close, 3)
	if asi[1] != 0 {
		t.Errorf("ASI of a flat bar = %v, want 0", asi[1])
	}
}

// TestSwingIndexDirectionFollowsTheClose checks the sign convention: a bar closing above the previous
// close with a body in the same direction must give a positive index, and the mirror must give a
// negative one.
func TestSwingIndexDirectionFollowsTheClose(t *testing.T) {
	up := SwingIndex([]float64{10, 11.5}, []float64{12, 14}, []float64{9, 11.4}, []float64{11, 13.5}, 3)
	if !(up[1] > 0) {
		t.Errorf("a bar closing higher gave %v, want positive", up[1])
	}
	down := SwingIndex([]float64{10, 9.5}, []float64{12, 10.6}, []float64{9, 8}, []float64{11, 8.5}, 3)
	if !(down[1] < 0) {
		t.Errorf("a bar closing lower gave %v, want negative", down[1])
	}
}

// TestSwingIndexNaNIsPermanentForTheAccumulator pins the accumulator's asymmetry: unlike a rolling
// sum, it cannot forget a NaN.
func TestSwingIndexNaNIsPermanentForTheAccumulator(t *testing.T) {
	open := []float64{10, 11, 12, 13, 14}
	high := []float64{12, 14, math.NaN(), 15, 16}
	low := []float64{9, 10, 11, 12, 13}
	close := []float64{11, 13, 14, 15, 16}

	asi := AccumulativeSwingIndex(open, high, low, close, 3)
	if !math.IsNaN(asi[2]) {
		t.Errorf("ASI[2] = %v, want NaN", asi[2])
	}
	for i := 3; i < len(asi); i++ {
		if !math.IsNaN(asi[i]) {
			t.Errorf("ASI[%d] = %v, want NaN: an accumulator cannot forget", i, asi[i])
		}
	}
}

func TestSwingIndexEmptyAndPanics(t *testing.T) {
	if AccumulativeSwingIndex(nil, nil, nil, nil, 3) != nil {
		t.Error("empty AccumulativeSwingIndex should be nil")
	}
	if SwingIndex(nil, nil, nil, nil, 3) != nil {
		t.Error("empty SwingIndex should be nil")
	}

	long := make([]float64, 3)
	short := make([]float64, 2)
	for _, tc := range []struct {
		name string
		fn   func()
	}{
		{"ASI zero limit", func() { AccumulativeSwingIndex(long, long, long, long, 0) }},
		{"ASI negative limit", func() { AccumulativeSwingIndex(long, long, long, long, -1) }},
		{"ASI NaN limit", func() { AccumulativeSwingIndex(long, long, long, long, math.NaN()) }},
		{"ASI length", func() { AccumulativeSwingIndex(short, short, short, long, 3) }},
		{"SI zero limit", func() { SwingIndex(long, long, long, long, 0) }},
		{"SI length", func() { SwingIndex(short, short, short, long, 3) }},
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
