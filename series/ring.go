package series

// ---------------------------------------------------------------------------
// Ring: a fixed-capacity circular buffer of float64.
//
// It is the storage primitive under every windowed roller. Its whole job is to
// answer two questions in O(1): "what value is leaving the window" and "what are the
// values in the window, oldest first".
//
// # Why the capacity is exactly the window
//
// The obvious speed trick for a circular buffer is to round the capacity up to a
// power of two and index with a mask instead of a modulo. That is wrong here: the
// buffer's capacity *is* the window length, so a window of 10 rounded to 16 would
// silently become a 16-element window and every rolling output would be wrong.
//
// Instead the wraparound is a compare-and-subtract rather than a division:
//
//	head++
//	if head == len(buf) { head = 0 }
//
// and the same for the write position, where `head + n < 2*capacity` bounds the
// arithmetic to a single conditional subtraction. So no integer division appears on
// the hot path.
//
// # Contract
//
// Push is the only mutator, and it reports what it evicted. At is safe for any i in
// [0, Len()). The zero Ring is empty and usable, and Push on it is a no-op that
// evicts nothing, so a roller built without a constructor cannot panic -- it simply
// never produces output.
// ---------------------------------------------------------------------------

// Ring is a fixed-capacity circular buffer. The zero value is an empty ring with no
// capacity.
type Ring struct {
	buf  []float64
	head int // index of the oldest element
	n    int // number of valid elements
}

// NewRing returns a ring holding at most capacity values.
//
// It panics if capacity is negative. A zero-capacity ring is legal and simply never
// retains anything.
func NewRing(capacity int) *Ring {
	if capacity < 0 {
		panic("series: negative ring capacity")
	}
	return &Ring{buf: make([]float64, capacity)}
}

// Cap returns the ring's capacity.
func (r *Ring) Cap() int { return len(r.buf) }

// Len returns the number of valid elements currently held.
func (r *Ring) Len() int { return r.n }

// Full reports whether the ring holds as many values as it has capacity for.
//
// A zero-capacity ring is never full, so the zero value stays consistently inert:
// it accepts pushes, retains nothing, and reports neither content nor fullness.
func (r *Ring) Full() bool { return len(r.buf) > 0 && r.n == len(r.buf) }

// At returns the i-th oldest element, so At(0) is the value that will be evicted
// next and At(Len()-1) is the most recently pushed.
//
// At panics if i is outside [0, Len()).
func (r *Ring) At(i int) float64 {
	if i < 0 || i >= r.n {
		panic("series: ring index out of range")
	}
	idx := r.head + i
	if idx >= len(r.buf) {
		idx -= len(r.buf)
	}
	return r.buf[idx]
}

// Push adds v to the ring, returning the evicted value and whether anything was
// evicted. Nothing is evicted until the ring is full.
func (r *Ring) Push(v float64) (evicted float64, ok bool) {
	c := len(r.buf)
	if c == 0 {
		return 0, false
	}
	if r.n < c {
		w := r.head + r.n
		if w >= c {
			w -= c
		}
		r.buf[w] = v
		r.n++
		return 0, false
	}
	// Full: overwrite the oldest in place and advance the head.
	old := r.buf[r.head]
	r.buf[r.head] = v
	r.head++
	if r.head == c {
		r.head = 0
	}
	return old, true
}

// Reset empties the ring without releasing its buffer, so it can be reused.
func (r *Ring) Reset() {
	r.head = 0
	r.n = 0
}
