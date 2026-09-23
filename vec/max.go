package vec

import "math"

// Max returns the largest element of xs.
//
// It returns -Inf for an empty or nil slice, which makes Max usable as the
// identity for a running maximum.
//
// Max returns NaN if any element of xs is NaN. See [NaN policy] for the reasoning
// behind that choice.
//
// # Why this is fast
//
// Same story as [Min]: a max scan is latency-bound, the chain cannot be shortened
// by pairing, and splitting the input into independent partial scans is the only
// available parallelism. The NaN check is fused into the scan loop and costs about
// 1% rather than the 2x a separate pass would cost. See [Min] for the measurements.
//
// [NaN policy]: #nan-policy
func Max(xs []float64) float64 {
	if len(xs) == 0 {
		return math.Inf(-1)
	}
	return maxArch(xs)
}
