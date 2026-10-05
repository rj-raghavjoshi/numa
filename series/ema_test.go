package series

import (
	"math"
	"testing"
)

// ---------------------------------------------------------------------------
// Tests for EMA and RMA.
//
// The recurrence itself is simple; what is easy to get wrong is the seed and the
// warm-up boundary, so those get the most attention. A wrong seed produces a series
// that looks plausible and disagrees with the reference for a long time, which is
// exactly the kind of error a loose tolerance test would miss.
// ---------------------------------------------------------------------------

func TestEMAMatchesReference(t *testing.T) {
	for _, n := range boundaryLengths() {
		if n == 0 {
			continue
		}
		xs := randomSlice(n, int64(n)+2)
		for _, w := range boundaryWindows() {
			alpha := 2.0 / float64(w+1)
			got := Apply(xs, NewEMA(w))
			assertSequence(t, "EMA", got, refSmoother(xs, w, alpha), false)
		}
	}
}

func TestRMAMatchesReference(t *testing.T) {
	for _, n := range boundaryLengths() {
		if n == 0 {
			continue
		}
		xs := randomSlice(n, int64(n)+3)
		for _, w := range boundaryWindows() {
			alpha := 1.0 / float64(w)
			got := Apply(xs, NewRMA(w))
			assertSequence(t, "RMA", got, refSmoother(xs, w, alpha), false)
		}
	}
}

// TestSmootherSeedsWithSMANotFirstValue is the specific regression test for the seed
// convention. A first-value seed would make the output at index n-1 equal the first
// input rather than the SMA of the first n, and would then diverge.
func TestSmootherSeedsWithSMANotFirstValue(t *testing.T) {
	xs := []float64{10, 20, 30, 40, 50}
	const w = 3

	// SMA of the first three is 20.
	ema := Apply(xs, NewEMA(w))
	rma := Apply(xs, NewRMA(w))

	if math.IsNaN(ema[2]) || math.IsNaN(rma[2]) {
		t.Fatalf("index 2 should be the first real output: ema=%v rma=%v", ema[2], rma[2])
	}
	if ema[2] != 20 || rma[2] != 20 {
		t.Errorf("seed value = (%v,%v), want (20,20) -- the SMA of the first 3", ema[2], rma[2])
	}
	// And both must be NaN strictly before the seed.
	if !math.IsNaN(ema[1]) || !math.IsNaN(rma[1]) {
		t.Errorf("index 1 = (%v,%v), want NaN before the seed", ema[1], rma[1])
	}
}

// TestSmootherOfConstantSeriesIsThatConstant checks the fixed point of the
// recurrence, which any seed must reproduce.
func TestSmootherOfConstantSeriesIsThatConstant(t *testing.T) {
	const w = 5
	xs := make([]float64, 30)
	for i := range xs {
		xs[i] = 3.5
	}
	for name, roller := range map[string]Roller{"EMA": NewEMA(w), "RMA": NewRMA(w)} {
		got := Apply(xs, roller)
		for i := w - 1; i < len(got); i++ {
			if math.Abs(got[i]-3.5) > 1e-12 {
				t.Fatalf("%s[%d] = %v, want 3.5", name, i, got[i])
			}
		}
	}
}

// TestEMAWeightsRecentValuesMore pins the defining property that separates an EMA
// from an SMA: a step change moves it further, faster.
func TestEMAWeightsRecentValuesMore(t *testing.T) {
	const w = 10
	xs := make([]float64, 30)
	for i := 0; i < 15; i++ {
		xs[i] = 1
	}
	for i := 15; i < len(xs); i++ {
		xs[i] = 2
	}

	ema := Apply(xs, NewEMA(w))
	sma := Apply(xs, NewSMA(w))

	// Three bars after the step, the EMA must be closer to 2 than the SMA is.
	const probe = 17
	if ema[probe] <= sma[probe] {
		t.Errorf("EMA[%d] = %v, SMA[%d] = %v; the EMA should have moved further",
			probe, ema[probe], probe, sma[probe])
	}
}

func TestEMAReset(t *testing.T) {
	xs := randomSlice(50, 21)

	first := Apply(xs, NewEMA(7))

	e := NewEMA(7)
	second := Apply(xs, e)
	e.Reset()
	third := Apply(xs, e)

	assertSequence(t, "EMA first", second, first, true)
	assertSequence(t, "EMA after reset", third, first, true)
}

func TestEMAWarmupValue(t *testing.T) {
	for _, w := range boundaryWindows() {
		if got := NewEMA(w).Warmup(); got != w {
			t.Errorf("EMA(%d).Warmup() = %d, want %d", w, got, w)
		}
		if got := NewRMA(w).Warmup(); got != w {
			t.Errorf("RMA(%d).Warmup() = %d, want %d", w, got, w)
		}
	}
}

func TestSmootherNaNPropagation(t *testing.T) {
	xs := []float64{1, 2, 3, math.NaN(), 5, 6, 7}
	const w = 2

	for name, roller := range map[string]Roller{"EMA": NewEMA(w), "RMA": NewRMA(w)} {
		got := Apply(xs, roller)
		// A NaN inside the window poisons the seeded value and, through the
		// recurrence, every output after it.
		for i := 3; i < len(got); i++ {
			if !math.IsNaN(got[i]) {
				t.Errorf("%s[%d] = %v, want NaN to persist through the recurrence", name, i, got[i])
			}
		}
	}
}

func TestSmootherPanicsOnNonPositiveWindow(t *testing.T) {
	for _, w := range []int{0, -3} {
		func() {
			defer func() {
				if recover() == nil {
					t.Errorf("NewEMA(%d) did not panic", w)
				}
			}()
			NewEMA(w)
		}()
		func() {
			defer func() {
				if recover() == nil {
					t.Errorf("NewRMA(%d) did not panic", w)
				}
			}()
			NewRMA(w)
		}()
	}
}
