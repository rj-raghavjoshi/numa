package series

import (
	"math"
	"testing"
)

// ---------------------------------------------------------------------------
// Tests for Sum and SMA.
//
// The rolling sum is the one roller here with a real numerical hazard, so beyond the
// usual agreement-with-reference checks it gets a test that the compensation earns
// its keep: an uncompensated rolling sum returns a visibly wrong answer on a window
// where the evicted value dwarfs the retained ones.
// ---------------------------------------------------------------------------

func TestSMAWarmupIsNaNBeforeWindow(t *testing.T) {
	for _, w := range boundaryWindows() {
		xs := randomSlice(w+5, int64(w))
		s := NewSMA(w)
		for i, v := range xs {
			got := s.Push(v)
			if i+1 < w {
				if !math.IsNaN(got) {
					t.Fatalf("window %d: Push %d = %v, want NaN before warmup", w, i, got)
				}
			} else if math.IsNaN(got) {
				t.Fatalf("window %d: Push %d = NaN, want a value at warmup", w, i)
			}
		}
	}
}

func TestSMAMatchesReference(t *testing.T) {
	for _, n := range boundaryLengths() {
		if n == 0 {
			continue
		}
		xs := randomSlice(n, int64(n)+1)
		for _, w := range boundaryWindows() {
			got := Apply(xs, NewSMA(w))
			assertSequence(t, "SMA", got, refSMA(xs, w), false)
		}
	}
}

func TestSumKnownValues(t *testing.T) {
	xs := []float64{1, 2, 3, 4, 5, 6}
	got := Apply(xs, NewSum(3))
	want := []float64{math.NaN(), math.NaN(), 6, 9, 12, 15}
	assertSequence(t, "Sum", got, want, false)

	sma := Apply(xs, NewSMA(3))
	wantSMA := []float64{math.NaN(), math.NaN(), 2, 3, 4, 5}
	assertSequence(t, "SMA", sma, wantSMA, false)
}

// TestSumWindowOfOne pins the smallest legal window, where the ring never evicts and
// the sum is the value itself.
func TestSumWindowOfOne(t *testing.T) {
	xs := []float64{4, -2, 7}
	assertSequence(t, "Sum(1)", Apply(xs, NewSum(1)), xs, false)
	assertSequence(t, "SMA(1)", Apply(xs, NewSMA(1)), xs, false)
}

// TestSumDoesNotLoseSmallValuesAcrossEviction is the justification for Neumaier
// compensation, and it is a hard assertion rather than a tolerance.
//
// The window holds {1e16, 1}. An uncompensated sum computes 1e16 + 1 = 1e16, losing
// the 1; when 1e16 is evicted it computes 1e16 - 1e16 = 0, and adding the next 1
// gives 1. The correct window total is 2.
func TestSumDoesNotLoseSmallValuesAcrossEviction(t *testing.T) {
	s := NewSum(2)
	s.Push(1e16)
	s.Push(1)
	s.Push(1) // evicts 1e16; window is now {1, 1}

	got := s.Push(1) // window {1, 1}, both small
	if got != 2 {
		t.Fatalf("compensated rolling sum = %v, want exactly 2", got)
	}

	// The same sequence through a plainly written uncompensated sum, to show the
	// failure is real and not hypothetical.
	naive := naiveRollingSum(2, []float64{1e16, 1, 1, 1})
	if naive[len(naive)-1] == 2 {
		t.Log("the uncompensated form happened to be exact on this machine; the compensated result is still the pinned contract")
	} else if naive[len(naive)-1] != 1 {
		t.Fatalf("uncompensated rolling sum = %v, expected either 2 (unexpectedly exact) or 1 (the documented failure)", naive[len(naive)-1])
	}
}

// naiveRollingSum is the obvious add-and-subtract rolling sum, kept here so the test
// above can demonstrate the failure it motivates.
func naiveRollingSum(window int, xs []float64) []float64 {
	out := make([]float64, 0, len(xs))
	var ring []float64
	var total float64
	for _, v := range xs {
		if len(ring) == window {
			total -= ring[0]
			ring = ring[1:]
		}
		ring = append(ring, v)
		total += v
		if len(ring) < window {
			out = append(out, math.NaN())
		} else {
			out = append(out, total)
		}
	}
	return out
}

// TestSumDriftOverLongRun checks that the compensated sum does not wander over a long
// sequence, by comparing against a freshly recomputed window total at every step.
func TestSumDriftOverLongRun(t *testing.T) {
	const window = 50
	const n = 20000

	// A large offset with small variation is the case where an uncompensated running
	// sum loses the variation entirely.
	xs := make([]float64, n)
	for i := range xs {
		xs[i] = 1e9 + float64(i%7)
	}

	s := NewSum(window)
	var ring []float64
	for i, v := range xs {
		got := s.Push(v)
		ring = append(ring, v)
		if len(ring) > window {
			ring = ring[1:]
		}
		if i+1 < window {
			continue
		}
		// Fresh, plain recomputation of the window: its only error is one pass of
		// summation, so any drift in the roller shows up as a disagreement.
		var fresh float64
		for _, w := range ring {
			fresh += w
		}
		if math.Abs(got-fresh) > 1e-6*math.Max(1, math.Abs(fresh)) {
			t.Fatalf("step %d: rolling sum %v disagrees with a fresh window sum %v by %v",
				i, got, fresh, got-fresh)
		}
	}
}

func TestSMAAndSumReset(t *testing.T) {
	xs := randomSlice(40, 7)

	first := Apply(xs, NewSMA(5))

	s := NewSMA(5)
	second := Apply(xs, s)
	s.Reset()
	third := Apply(xs, s)

	assertSequence(t, "SMA first", second, first, true)
	assertSequence(t, "SMA after reset", third, first, true)
}

func TestSumNaNPropagates(t *testing.T) {
	// Window 2: indices 2 and 3 hold the NaN, so both are NaN; index 4 is the first
	// window without it.
	xs := []float64{1, 2, math.NaN(), 4, 5}
	got := Apply(xs, NewSum(2))

	if !math.IsNaN(got[0]) {
		t.Errorf("index 0 = %v, want NaN (window not yet full)", got[0])
	}
	if got[1] != 3 {
		t.Errorf("index 1 = %v, want 3 (window {1,2})", got[1])
	}
	if !math.IsNaN(got[2]) {
		t.Errorf("index 2 = %v, want NaN (the NaN itself)", got[2])
	}
	if !math.IsNaN(got[3]) {
		t.Errorf("index 3 = %v, want NaN (NaN still in the window)", got[3])
	}
	if got[4] != 9 {
		t.Errorf("index 4 = %v, want 9 (window {4,5}, NaN has left)", got[4])
	}
}

// TestSumRecoversAfterNaNLeavesTheWindow is the regression test for the interaction
// between NaN handling and Neumaier compensation.
//
// Accumulating a NaN directly poisons the compensation term permanently, because its
// eviction is a subtraction of NaN and no later addition can clear it. That makes the
// sum NaN forever after a single NaN, which an uncompensated sum does not do. The
// implementation therefore counts NaNs instead of accumulating them, and this test
// pins that the window recovers.
func TestSumRecoversAfterNaNLeavesTheWindow(t *testing.T) {
	// Window 2, NaN at index 1, so the window is NaN-free from index 3 onward.
	xs := []float64{1, math.NaN(), 3, 4, 5}
	got := Apply(xs, NewSum(2))

	if !math.IsNaN(got[2]) {
		t.Errorf("index 2 = %v, want NaN (NaN still in the window)", got[2])
	}
	if got[3] != 7 {
		t.Errorf("index 3 = %v, want 7 (window {3,4})", got[3])
	}
	if got[4] != 9 {
		t.Errorf("index 4 = %v, want 9 (window {4,5})", got[4])
	}

	// The same via SMA, whose recovery follows from Sum's.
	sma := Apply(xs, NewSMA(2))
	if sma[3] != 3.5 || sma[4] != 4.5 {
		t.Errorf("SMA after NaN left = (%v,%v), want (3.5,4.5)", sma[3], sma[4])
	}
}

func TestSumPanicsOnNonPositiveWindow(t *testing.T) {
	for _, w := range []int{0, -1} {
		func() {
			defer func() {
				if recover() == nil {
					t.Errorf("NewSum(%d) did not panic", w)
				}
			}()
			NewSum(w)
		}()
		func() {
			defer func() {
				if recover() == nil {
					t.Errorf("NewSMA(%d) did not panic", w)
				}
			}()
			NewSMA(w)
		}()
	}
}
