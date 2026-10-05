package numa

import (
	"math"
	"testing"

	"github.com/rj-raghavjoshi/numa/ta"
)

// ---------------------------------------------------------------------------
// Forwarder tests for the Swing Index.
//
// The limit move is a single float that scales every output, so it is driven with two distinct values
// and checked to reach the implementation rather than silently defaulting.
// ---------------------------------------------------------------------------

func TestSwingForwardersMatchTa(t *testing.T) {
	_, high, low, close, _ := testOHLCV()
	open := make([]float64, len(close))
	for i := range close {
		open[i] = (high[i] + low[i]) / 2
	}

	eqSeries(t, "SwingIndex", SwingIndex(open, high, low, close, 3),
		ta.SwingIndex(open, high, low, close, 3))
	eqSeries(t, "AccumulativeSwingIndex", AccumulativeSwingIndex(open, high, low, close, 3),
		ta.AccumulativeSwingIndex(open, high, low, close, 3))

	// The limit move must reach the implementation.
	a := SwingIndex(open, high, low, close, 2)
	b := SwingIndex(open, high, low, close, 4)
	if eqSeriesEqual(a, b) {
		t.Error("SwingIndex ignores its limit move")
	}
}

// TestAccumulativeSwingIndexForwarderIsARunningTotal repeats the structural check through the facade.
func TestAccumulativeSwingIndexForwarderIsARunningTotal(t *testing.T) {
	_, high, low, close, _ := testOHLCV()
	open := make([]float64, len(close))
	for i := range close {
		open[i] = (high[i] + low[i]) / 2
	}

	si := SwingIndex(open, high, low, close, 3)
	asi := AccumulativeSwingIndex(open, high, low, close, 3)

	var total float64
	for i := range si {
		total += si[i]
		if math.Abs(asi[i]-total) > 1e-9 {
			t.Fatalf("ASI[%d] = %v, want the running total %v", i, asi[i], total)
		}
	}
}

func TestSwingForwardersPanicLikeTa(t *testing.T) {
	_, high, low, close, _ := testOHLCV()
	open := make([]float64, len(close))
	short := make([]float64, 2)
	long := make([]float64, 3)

	for _, tc := range []struct {
		name string
		fn   func()
	}{
		{"SwingIndex limit", func() { SwingIndex(open, high, low, close, 0) }},
		{"SwingIndex length", func() { SwingIndex(short, short, short, long, 3) }},
		{"ASI limit", func() { AccumulativeSwingIndex(open, high, low, close, -1) }},
		{"ASI length", func() { AccumulativeSwingIndex(short, short, short, long, 3) }},
	} {
		tc := tc
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

func TestSwingForwardersEmptyInputs(t *testing.T) {
	if SwingIndex(nil, nil, nil, nil, 3) != nil {
		t.Error("empty SwingIndex should be nil")
	}
	if AccumulativeSwingIndex(nil, nil, nil, nil, 3) != nil {
		t.Error("empty AccumulativeSwingIndex should be nil")
	}
}
