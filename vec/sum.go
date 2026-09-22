package vec

// Sum returns the arithmetic sum of xs using architecture-optimized
// register pipelining and instruction-level parallelism.
func Sum(xs []float64) float64 {
	if len(xs) == 0 {
		return 0.0
	}
	return sumArch(xs)
}