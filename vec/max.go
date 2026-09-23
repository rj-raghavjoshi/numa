package vec

import "math"

// Max returns the largest element of xs.
//
// It returns -Inf for an empty or nil slice, which makes Max usable as the
// identity for a running maximum.
//
// Max returns NaN if any element of xs is NaN. See [NaN policy] for the reasoning
// behind that choice and its cost.
//
// # Why this is fast, and why that is not enough
//
// Same story as [Min]: a max scan is latency-bound, the chain cannot be shortened
// by pairing, and splitting the input into independent partial scans is the only
// available parallelism. The tuned scan loop alone measures 1.98x faster than the
// naive loop.
//
// Max enforces the "NaN wins" policy with a second pass over the input, which
// costs about 2x and consumes that gain entirely. See [Min] for the full
// discussion, and ../docs/next-steps.md for the open decision.
//
// [NaN policy]: #nan-policy
func Max(xs []float64) float64 {
	if len(xs) == 0 {
		return math.Inf(-1)
	}
	m := maxArch(xs)
	if hasNaN(xs) {
		return math.NaN()
	}
	return m
}
