package vec

// Add returns the elementwise sum of xs and ys as a new slice.
//
// If the slices differ in length, the result has length min(len(xs), len(ys)). If
// either slice is empty the result is nil.
//
// Add allocates. For hot loops that can supply their own destination buffer, use
// [AddTo].
//
// # On the modest speedup
//
// Add is only ~1.27x faster than the naive loop, versus ~4x for [Sum]. That is not
// a weaker implementation: it is because this loop has no dependency chain to fix.
// Every output depends only on its own inputs, so there was never a stall to
// eliminate, and the loop is bound by memory bandwidth instead. See the package
// documentation for the full comparison.
func Add(xs, ys []float64) []float64 {
	n := min(len(xs), len(ys))
	if n == 0 {
		return nil
	}
	return addArch(make([]float64, n), xs[:n], ys[:n])
}

// AddTo stores the elementwise sum of xs and ys into dst and returns dst.
//
// The three slices must be the same length. AddTo panics if they are not, so that a
// length mismatch surfaces at the call site rather than corrupting memory.
//
// dst may alias xs or ys: writing to dst[i] happens only after xs[i] and ys[i] have
// been read in the same iteration.
func AddTo(dst, xs, ys []float64) []float64 {
	checkSameLen("AddTo", dst, xs, ys)
	return addArch(dst, xs, ys)
}

// AddScalar returns xs plus the scalar k, as a new slice.
//
// It returns nil for an empty or nil input.
//
// AddScalar allocates. For hot loops that can supply their own destination buffer,
// use [AddScalarTo].
func AddScalar(xs []float64, k float64) []float64 {
	if len(xs) == 0 {
		return nil
	}
	return addScalarArch(make([]float64, len(xs)), xs, k)
}

// AddScalarTo stores xs plus the scalar k into dst and returns dst.
//
// dst and xs must be the same length. AddScalarTo panics if they are not.
//
// dst may alias xs.
func AddScalarTo(dst, xs []float64, k float64) []float64 {
	checkSameLen1("AddScalarTo", dst, xs)
	return addScalarArch(dst, xs, k)
}
