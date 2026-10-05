package ta

import (
	"math"
	"testing"

	"github.com/rj-raghavjoshi/numa/vec"
)

// ---------------------------------------------------------------------------
// Tests for Ichimoku.
//
// The interesting part is not the midpoints -- those are Highest and Lowest, already tested --
// but the displacement. Each span is asserted against a shift of its own input, which is the
// only way a sign error in the displacement shows up.
// ---------------------------------------------------------------------------

func TestIchimokuUndisplacedLines(t *testing.T) {
	const size = 200
	high, low, close := synthHLC(size, 1717)
	const conv, base, spanB, disp = 9, 26, 52, 26

	conversion, baseLine, _, _, _ := Ichimoku(high, low, close, conv, base, spanB, disp)

	// The conversion and base lines are not displaced at all.
	refConv := midpointLine(high, low, conv)
	refBase := midpointLine(high, low, base)
	assertSeries(t, "Ichimoku conversion", conversion, refConv, false)
	assertSeries(t, "Ichimoku base", baseLine, refBase, false)

	if got := FirstValid(conversion); got != conv-1 {
		t.Errorf("conversion first valid = %d, want %d", got, conv-1)
	}
	if got := FirstValid(baseLine); got != base-1 {
		t.Errorf("base first valid = %d, want %d", got, base-1)
	}
}

func TestIchimokuDisplacement(t *testing.T) {
	const size = 200
	high, low, close := synthHLC(size, 2727)
	const conv, base, spanB, disp = 9, 26, 52, 26

	conversion, baseLine, spanA, spanBLine, lagging := Ichimoku(high, low, close, conv, base, spanB, disp)

	// spanA is the midpoint of the two short lines, displaced forward by disp.
	raw := make([]float64, size)
	for i := range close {
		raw[i] = (conversion[i] + baseLine[i]) / 2
	}
	assertSeries(t, "Ichimoku spanA", spanA, vec.Shift(raw, disp), false)

	// spanB is the long range midpoint, displaced forward by the same amount.
	assertSeries(t, "Ichimoku spanB", spanBLine, vec.Shift(midpointLine(high, low, spanB), disp), false)

	// The lagging span is the close displaced backward, so it is undefined at the end.
	assertSeries(t, "Ichimoku lagging", lagging, vec.Shift(close, -disp), false)
	for i := size - disp; i < size; i++ {
		if !math.IsNaN(lagging[i]) {
			t.Errorf("lagging[%d] = %v, want NaN (no future close)", i, lagging[i])
		}
	}
	if lagging[0] != close[disp] {
		t.Errorf("lagging[0] = %v, want close[%d] = %v", lagging[0], disp, close[disp])
	}

	// The leading spans must be NaN for the first disp bars after their inputs are ready,
	// which is the forward shift's whole effect.
	if !math.IsNaN(spanA[disp-1]) {
		t.Errorf("spanA[%d] = %v, want NaN inside the forward displacement", disp-1, spanA[disp-1])
	}
	if math.IsNaN(spanA[base-1+disp]) {
		t.Errorf("spanA[%d] = NaN, want a value once the shift has filled", base-1+disp)
	}
}

// TestIchimokuZeroDisplacement pins the degenerate case: no shift at all, so the lagging span
// is the close itself.
func TestIchimokuZeroDisplacement(t *testing.T) {
	const size = 80
	high, low, close := synthHLC(size, 3737)

	conversion, baseLine, spanA, spanBLine, lagging := Ichimoku(high, low, close, 9, 26, 52, 0)
	for i := range close {
		if !closeOrNaN(spanA[i], (conversion[i]+baseLine[i])/2) {
			t.Fatalf("spanA[%d] = %v, want the undisplaced midpoint", i, spanA[i])
		}
		if !closeOrNaN(lagging[i], close[i]) {
			t.Fatalf("lagging[%d] = %v, want close = %v", i, lagging[i], close[i])
		}
	}
	if !closeOrNaN(spanBLine[51], midpointLine(high, low, 52)[51]) {
		t.Error("spanB should be undisplaced when displacement is zero")
	}
}

func TestIchimokuEmptyAndPanics(t *testing.T) {
	if c, b, a, s, l := Ichimoku(nil, nil, nil, 9, 26, 52, 26); c != nil || b != nil || a != nil || s != nil || l != nil {
		t.Error("empty Ichimoku should return nils")
	}

	short := make([]float64, 2)
	long := make([]float64, 3)
	for _, tc := range []struct {
		name string
		fn   func()
	}{
		{"conversionPeriod", func() { Ichimoku(long, long, long, 0, 26, 52, 26) }},
		{"basePeriod", func() { Ichimoku(long, long, long, 9, 0, 52, 26) }},
		{"spanBPeriod", func() { Ichimoku(long, long, long, 9, 26, 0, 26) }},
		{"negative displacement", func() { Ichimoku(long, long, long, 9, 26, 52, -1) }},
		{"length", func() { Ichimoku(short, short, long, 9, 26, 52, 26) }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			defer func() {
				if recover() == nil {
					t.Errorf("Ichimoku %s did not panic", tc.name)
				}
			}()
			tc.fn()
		})
	}
}
