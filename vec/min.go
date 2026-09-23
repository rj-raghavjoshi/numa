package vec

import "math"

// Min returns the smallest element of xs.
//
// It returns +Inf for an empty or nil slice, which makes Min usable as the
// identity for a running minimum.
//
// Min returns NaN if any element of xs is NaN. See [NaN policy] for the reasoning
// behind that choice and its cost.
//
// # Why this is fast, and why that is not enough
//
// A min scan is latency-bound rather than throughput-bound: a compare cannot
// start until the previous compare's result is known, so the chain is inherently
// the length of the input. Unlike addition, there is no `s += a + b` trick that
// shortens it -- min has no equivalent of combining two elements into one before
// touching the accumulator.
//
// Splitting the input into independent partial scans is therefore the only
// available parallelism, and it divides the chain length by the accumulator count
// rather than eliminating it. The tuned scan loop alone measures 1.98x faster than
// the naive loop at n=4M.
//
// # The NaN check costs that entire gain
//
// Min enforces the "NaN wins" policy with a second pass over the input, which
// costs about 2x. At n=4M that makes Min measure 0.98x -- no faster than the
// naive one-accumulator loop.
//
// This is a known, measured, and currently unresolved trade: correctness over
// throughput. It is flagged as the highest-priority open item in
// ../docs/next-steps.md. If you need scan throughput and can rule out NaN in your
// input, call the internal minArch directly within this package, or filter first.
//
// [NaN policy]: #nan-policy
func Min(xs []float64) float64 {
	if len(xs) == 0 {
		return math.Inf(1)
	}
	m := minArch(xs)
	if hasNaN(xs) {
		return math.NaN()
	}
	return m
}
