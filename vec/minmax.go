package vec

import "math"

// MinMax returns the smallest and largest elements of xs in a single pass.
//
// It returns (+Inf, -Inf) for an empty or nil slice. Callers that need both values
// should prefer MinMax over calling [Min] and [Max] separately, because it reads
// the input once instead of twice.
//
// MinMax returns (NaN, NaN) if any element of xs is NaN. See [NaN policy] for the
// reasoning.
//
// # Why this is fast
//
// Eight accumulators on x86-64: four scanning for the minimum and four for the
// maximum. Keeping the min and max chains distinct lets the compare circuitry stay
// busy, since the two directions are independent.
//
// The NaN check is fused into the same loop, so the single-pass advantage over
// `Min(xs)` plus `Max(xs)` is preserved rather than being spent on an extra pass.
//
// [NaN policy]: #nan-policy
func MinMax(xs []float64) (lo, hi float64) {
	if len(xs) == 0 {
		return math.Inf(1), math.Inf(-1)
	}
	return minMaxArch(xs)
}
