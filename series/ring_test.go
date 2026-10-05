package series

import (
	"math"
	"testing"
)

// ---------------------------------------------------------------------------
// Tests for Ring.
//
// The failure modes worth pinning:
//
//   - a wraparound that loses the oldest-element ordering, which every windowed
//     roller depends on
//   - eviction happening one push too early or too late, which shifts every output
//   - Reset leaving the head behind so a reused ring returns the wrong order
// ---------------------------------------------------------------------------

func TestRingFillsThenEvictsInOrder(t *testing.T) {
	r := NewRing(3)
	if r.Cap() != 3 || r.Len() != 0 || r.Full() {
		t.Fatalf("fresh ring: cap=%d len=%d full=%v", r.Cap(), r.Len(), r.Full())
	}

	if _, ok := r.Push(1); ok {
		t.Error("evicted on the first push")
	}
	if _, ok := r.Push(2); ok {
		t.Error("evicted before full")
	}
	if _, ok := r.Push(3); ok {
		t.Error("evicted at the moment of becoming full")
	}
	if !r.Full() || r.Len() != 3 {
		t.Fatalf("after 3 pushes: len=%d full=%v", r.Len(), r.Full())
	}

	// The fourth push evicts the oldest, which is 1.
	if old, ok := r.Push(4); !ok || old != 1 {
		t.Fatalf("push 4 evicted (%v,%v), want (1,true)", old, ok)
	}
	for i, want := range []float64{2, 3, 4} {
		if got := r.At(i); got != want {
			t.Errorf("At(%d) = %v, want %v", i, got, want)
		}
	}

	// And the fifth evicts 2, continuing the order.
	if old, ok := r.Push(5); !ok || old != 2 {
		t.Fatalf("push 5 evicted (%v,%v), want (2,true)", old, ok)
	}
	for i, want := range []float64{3, 4, 5} {
		if got := r.At(i); got != want {
			t.Errorf("after wraparound At(%d) = %v, want %v", i, got, want)
		}
	}
}

// TestRingOrderSurvivesManyWrap makes the ordering check exhaustive rather than
// dependent on a single wraparound.
func TestRingOrderSurvivesManyWrap(t *testing.T) {
	const window = 7
	r := NewRing(window)
	for step := 0; step < 100; step++ {
		r.Push(float64(step))
		if step+1 < window {
			continue // still filling; the order check needs a full ring
		}
		if r.Len() != window {
			t.Fatalf("step %d: len=%d, want %d", step, r.Len(), window)
		}
		// The window must be the last `window` values, oldest first.
		for i := 0; i < window; i++ {
			want := float64(step - window + 1 + i)
			if got := r.At(i); got != want {
				t.Fatalf("step %d At(%d) = %v, want %v", step, i, got, want)
			}
		}
	}
}

func TestRingReset(t *testing.T) {
	r := NewRing(3)
	r.Push(1)
	r.Push(2)
	r.Push(3)
	r.Push(4) // wraps the head
	r.Reset()

	if r.Len() != 0 || r.Full() {
		t.Fatalf("after Reset: len=%d full=%v", r.Len(), r.Full())
	}
	// A reused ring must fill from the beginning in order.
	r.Push(10)
	r.Push(11)
	for i, want := range []float64{10, 11} {
		if got := r.At(i); got != want {
			t.Errorf("reused At(%d) = %v, want %v", i, got, want)
		}
	}
}

// TestZeroRingIsInert pins the documented property that a roller built without its
// constructor cannot panic: the zero Ring has no capacity and silently discards.
func TestZeroRingIsInert(t *testing.T) {
	var r Ring
	if old, ok := r.Push(1); ok || old != 0 {
		t.Fatalf("zero ring Push = (%v,%v), want (0,false)", old, ok)
	}
	if r.Len() != 0 || r.Full() {
		t.Fatal("zero ring reported content")
	}
	r.Reset() // must not panic
}

func TestRingPanics(t *testing.T) {
	func() {
		defer func() {
			if recover() == nil {
				t.Error("NewRing(-1) did not panic")
			}
		}()
		NewRing(-1)
	}()

	r := NewRing(2)
	r.Push(1)
	for _, i := range []int{-1, 1, 5} {
		func() {
			defer func() {
				if recover() == nil {
					t.Errorf("At(%d) did not panic", i)
				}
			}()
			r.At(i)
		}()
	}
}

// TestRingCapacityIsExactWindow is the regression test against the tempting
// power-of-two capacity trick. A window of 3 must hold exactly 3 values; rounding
// the capacity to 4 would silently change every rolling output.
func TestRingCapacityIsExactWindow(t *testing.T) {
	for _, w := range []int{1, 2, 3, 5, 6, 7, 9, 10, 11, 13} {
		r := NewRing(w)
		if r.Cap() != w {
			t.Fatalf("NewRing(%d).Cap() = %d, want exactly %d", w, r.Cap(), w)
		}
		for i := 0; i < w; i++ {
			if _, ok := r.Push(float64(i)); ok {
				t.Fatalf("window %d evicted at push %d, before full", w, i)
			}
		}
		if _, ok := r.Push(99); !ok {
			t.Fatalf("window %d did not evict when full", w)
		}
		if r.Len() != w {
			t.Fatalf("window %d holds %d elements after eviction", w, r.Len())
		}
	}
}

// testRoller is a minimal Roller used to exercise the interface and the driver
// without depending on any particular implementation. It is a cumulative sum that
// reports NaN until it has seen n values -- simple, and with no state a test could
// confuse for the roller under test.
type testRoller struct {
	n     int
	count int
	total float64
}

func (r *testRoller) Warmup() int { return r.n }

func (r *testRoller) Reset() { r.count = 0; r.total = 0 }

func (r *testRoller) Push(v float64) float64 {
	r.count++
	r.total += v
	if r.count < r.n {
		return math.NaN()
	}
	return r.total
}
