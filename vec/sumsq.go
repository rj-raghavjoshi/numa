package vec

// SumSq returns the sum of squares of xs.
//
// It returns 0 for an empty or nil slice.
//
// # Why this is here
//
// SumSq avoids the intermediate allocation that chaining a multiply with a Sum
// would require, and it keeps the tuned loop in a single pass over the input.
//
// It is the identity `Dot(xs, xs)`, and the test suite asserts that relationship
// holds, which keeps the two implementations honest about each other.
//
// # Why this is fast
//
// Same structure as Dot: the multiply latency is hidden by independent
// accumulator chains, which the naive loop cannot do.
func SumSq(xs []float64) float64 {
	if len(xs) == 0 {
		return 0.0
	}
	return sumSqArch(xs)
}
