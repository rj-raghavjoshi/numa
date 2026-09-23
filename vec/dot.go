package vec

// Dot returns the inner product of xs and ys, the sum of xs[i]*ys[i].
//
// It returns 0 if either slice is empty. If the two slices have different
// lengths, Dot uses only the first min(len(xs), len(ys)) elements; it does not
// pad or panic.
//
// # Why this is fast
//
// Dot is the reduction used by matrix multiplication, convolution, and
// neural-network inference, which is why it is tuned aggressively.
//
// A floating-point multiply has higher latency than an add, so the naive
// loop-carried dependency stalls harder here than it does in Sum. That means
// there is more for independent accumulators to recover, and Dot shows the
// largest speedup of any reduction in this package.
//
// Measured: 3.81x faster than the naive loop at n=4M. See ../docs/benchmarks.md.
func Dot(xs, ys []float64) float64 {
	n := min(len(xs), len(ys))
	if n == 0 {
		return 0.0
	}
	return dotArch(xs[:n], ys[:n])
}
