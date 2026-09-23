package vec

// Sum returns the arithmetic sum of xs.
//
// The result is computed with architecture-tuned register pipelining and
// instruction-level parallelism, which means the input is reassociated: the
// result may differ in the last few bits from a plain left-to-right loop. See
// the package documentation for the reasoning.
//
// Sum returns 0 for an empty or nil slice.
//
// # Why this is fast
//
// The naive loop keeps a single accumulator, so every iteration's add depends on
// the previous iteration's add. That loop-carried dependency prevents the CPU
// from overlapping work, and it stalls once per element.
//
// The tuned loops keep two (ARM64) or four (x86-64) independent accumulators, so
// each one carries its own shorter chain. The partial results are combined at
// the end as a balanced tree rather than a fold.
//
// Measured: 3.98x faster than the naive loop at n=4M. See ../docs/benchmarks.md.
func Sum(xs []float64) float64 {
	if len(xs) == 0 {
		return 0.0
	}
	return sumArch(xs)
}

// Mean returns the arithmetic mean of xs.
//
// It returns 0 for an empty or nil slice.
//
// Mean sums the slice with the same tuned reduction as Sum and then divides, so
// it inherits Sum's reassociation behaviour. Note that summing and then dividing
// once is both faster and more accurate than accumulating `total += v/n` in the
// loop, which would introduce a rounding error per element.
func Mean(xs []float64) float64 {
	n := len(xs)
	if n == 0 {
		return 0.0
	}
	return sumArch(xs) / float64(n)
}
