package series

// ---------------------------------------------------------------------------
// The Roller interface and the batch driver over it.
// ---------------------------------------------------------------------------

// Roller is a stateful streaming computation over a sequence of float64 values.
//
// Implementations must satisfy these properties:
//
//   - Push returns NaN until exactly Warmup() values have been pushed, and a real
//     value from then on.
//   - Warmup() is constant for the life of the roller.
//   - Reset returns the roller to the state it had immediately after construction.
//   - A NaN input makes the affected outputs NaN, and the roller recovers neither
//     automatically nor silently.
//
// Rollers are not safe for concurrent use.
type Roller interface {
	// Warmup returns the number of elements that must be pushed before Push
	// produces a value other than NaN.
	Warmup() int

	// Push consumes one value and returns the current output.
	Push(v float64) float64

	// Reset returns the roller to its post-construction state without reallocating.
	Reset()
}

// ApplyTo pushes every element of xs through r, writing the outputs into dst, and
// returns dst.
//
// dst and xs must be the same length. ApplyTo panics if they are not.
//
// ApplyTo does NOT reset r first. That is deliberate: it lets a caller warm a roller
// on one series and then run it over the next without the seam being visible, which
// is what a streaming computation over concatenated data should do. Pass a fresh
// roller, or call Reset, when the series are meant to be independent.
func ApplyTo(dst, xs []float64, r Roller) []float64 {
	if len(dst) != len(xs) {
		panic("series: ApplyTo length mismatch")
	}
	for i, v := range xs {
		dst[i] = r.Push(v)
	}
	return dst
}

// Apply pushes every element of xs through r and returns a new slice of the outputs.
//
// It returns nil for an empty input. Like [ApplyTo] it does not reset r first.
func Apply(xs []float64, r Roller) []float64 {
	if len(xs) == 0 {
		return nil
	}
	return ApplyTo(make([]float64, len(xs)), xs, r)
}
