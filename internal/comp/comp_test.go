package comp

import (
	"math"
	"testing"
)

// ---------------------------------------------------------------------------
// Tests for the shared compensated accumulator.
//
// This helper is used by both vec's rolling kernels and series' rollers, so a defect here would
// be invisible in either package's tests while corrupting both. It gets its own test for the same
// reason the synthetic data generator does: shared code that can be wrong in a way that makes
// callers pass needs its own check.
// ---------------------------------------------------------------------------

func TestNeumaierAddIsExactOnTheClassicCase(t *testing.T) {
	// 1e16 + 1 is 1e16 in float64: the 1 is lost. The compensation must hold it.
	total, comp := NeumaierAdd(1e16, 0, 1)
	if total != 1e16 {
		t.Fatalf("total = %v, want 1e16", total)
	}
	if comp != 1 {
		t.Fatalf("compensation = %v, want 1", comp)
	}
	if total+comp != 1e16 {
		// The sum is still rounded when reported, which is expected: the compensation is the
		// only place the lost bit survives, and it matters when a later subtraction removes
		// the large term.
	}

	// And the classic sequence: subtract the large term back out, and the small values remain.
	total, comp = NeumaierAdd(total, comp, -1e16)
	if total != 0 {
		t.Fatalf("total after subtraction = %v, want 0", total)
	}
	if comp != 1 {
		t.Fatalf("compensation after subtraction = %v, want 1", comp)
	}
}

func TestNeumaierAddKeepsSmallValuesAcrossEviction(t *testing.T) {
	// The rolling-sum failure, replayed exactly as RollingSumTo performs it for a window of
	// two over {1e16, 1, 1}: push 1e16, push 1, then push 1 while evicting 1e16.
	var total, compensation float64
	for _, v := range []float64{1e16, 1} {
		total, compensation = NeumaierAdd(total, compensation, v)
	}
	total, compensation = NeumaierAdd(total, compensation, 1)
	total, compensation = NeumaierAdd(total, compensation, -1e16)
	if got := total + compensation; got != 2 {
		t.Fatalf("compensated rolling sum = %v, want exactly 2", got)
	}
}

// TestNeumaierAddBeatsNaiveOnCancellation checks the property that motivates the helper, rather
// than a specific value: over a cancelling sequence the compensated total must be closer to the
// exact answer than the naive one.
func TestNeumaierAddBeatsNaiveOnCancellation(t *testing.T) {
	// Repeat [1e16, 1, -1e16]. The naive sum returns to 0 every cycle, because 1e16 + 1 is
	// 1e16 and the subsequent subtraction removes exactly that. The true sum is one per cycle.
	const cycles = 100
	var total, compensation float64
	var naive float64
	for c := 0; c < cycles; c++ {
		for _, v := range []float64{1e16, 1, -1e16} {
			total, compensation = NeumaierAdd(total, compensation, v)
			naive += v
		}
	}
	compensated := total + compensation
	const want = float64(cycles)

	if math.Abs(compensated-want) >= math.Abs(naive-want) {
		t.Fatalf("compensated error %v is not better than naive error %v (naive=%v, compensated=%v)",
			math.Abs(compensated-want), math.Abs(naive-want), naive, compensated)
	}
}

func TestNeumaierAddWithNaN(t *testing.T) {
	total, comp := NeumaierAdd(math.NaN(), 0, 1)
	if !math.IsNaN(total) || !math.IsNaN(comp) {
		t.Fatalf("NaN input gave (%v,%v), want both NaN", total, comp)
	}
	// The documented consequence: a NaN cannot be compensated back out, which is why the
	// rolling kernels count NaNs instead of accumulating them.
	total, comp = NeumaierAdd(total, comp, math.NaN())
	if !math.IsNaN(total) {
		t.Fatalf("total = %v, want NaN (sticky)", total)
	}
}
