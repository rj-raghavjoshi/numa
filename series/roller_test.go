package series

import (
	"math"
	"testing"
)

// ---------------------------------------------------------------------------
// Tests for the Roller interface and the Apply/ApplyTo driver.
//
// The driver is small, but it has one contract that is easy to get backwards: it does
// NOT reset the roller, so a caller can warm it on one series and continue into the
// next. These tests pin that, and pin that the interface path and the concrete-type
// path produce identical output.
// ---------------------------------------------------------------------------

func TestApplyToMatchesDirectPush(t *testing.T) {
	xs := randomSlice(64, 99)

	for _, w := range []int{1, 3, 8, 17} {
		direct := make([]float64, len(xs))
		r := NewSMA(w)
		for i, v := range xs {
			direct[i] = r.Push(v)
		}

		viaInterface := Apply(xs, NewSMA(w))
		assertSequence(t, "Apply vs direct Push", viaInterface, direct, true)
	}
}

func TestApplyToWritesIntoDst(t *testing.T) {
	xs := randomSlice(20, 4)
	dst := make([]float64, len(xs))
	out := ApplyTo(dst, xs, NewSMA(5))

	if len(out) != len(xs) {
		t.Fatalf("ApplyTo returned length %d, want %d", len(out), len(xs))
	}
	// The return value must be the same buffer, not a copy.
	if &out[0] != &dst[0] {
		t.Error("ApplyTo did not return dst")
	}
	assertSequence(t, "ApplyTo", out, refSMA(xs, 5), false)
}

// TestApplyToDoesNotReset is the contract that makes warm-up reuse possible. Calling
// ApplyTo twice must be equivalent to pushing the concatenated series once.
func TestApplyToDoesNotReset(t *testing.T) {
	head := randomSlice(10, 5)
	tail := randomSlice(10, 6)
	all := append(append([]float64(nil), head...), tail...)

	// One roller, two calls.
	r := NewSMA(4)
	Apply(head, r)
	second := Apply(tail, r)

	// One roller, one call over the concatenation. Compared with a tolerance: the
	// reference recomputes each window in order, while the roller accumulates and
	// evicts, so their last bits legitimately differ.
	want := refSMA(all, 4)[len(head):]

	assertSequence(t, "continued Apply", second, want, false)
}

func TestApplyEmptyAndPanics(t *testing.T) {
	if got := Apply(nil, NewSMA(3)); got != nil {
		t.Errorf("Apply(nil) = %v, want nil", got)
	}

	defer func() {
		if recover() == nil {
			t.Error("ApplyTo did not panic on a length mismatch")
		}
	}()
	ApplyTo(make([]float64, 2), make([]float64, 3), NewSMA(2))
}

// TestRollerInterfaceIsSatisfied is a compile-time assertion that each roller
// implements the interface, written so that a removal is a build failure rather than
// a test failure.
func TestRollerInterfaceIsSatisfied(t *testing.T) {
	rollers := map[string]Roller{
		"Sum": NewSum(5),
		"SMA": NewSMA(5),
		"EMA": NewEMA(5),
		"RMA": NewRMA(5),
	}
	for name, r := range rollers {
		if r.Warmup() != 5 {
			t.Errorf("%s.Warmup() = %d, want 5", name, r.Warmup())
		}
		if len(Apply(randomSlice(30, 1), r)) != 30 {
			t.Errorf("%s produced the wrong output length", name)
		}
	}
}

// TestTestRollerThroughTheInterface checks the driver against a Roller that is not an
// implementation from this package, so the interface is exercised rather than just the
// concrete types.
func TestTestRollerThroughTheInterface(t *testing.T) {
	xs := []float64{1, 2, 3, 4, 5}
	got := Apply(xs, &testRoller{n: 3})
	want := []float64{math.NaN(), math.NaN(), 6, 10, 15}
	assertSequence(t, "testRoller", got, want, true)

	r := &testRoller{n: 2}
	Apply([]float64{1, 1}, r) // warms r
	tail := Apply([]float64{3}, r)
	if tail[0] != 5 {
		t.Errorf("continued testRoller = %v, want 5", tail[0])
	}
}
