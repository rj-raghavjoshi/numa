package vec

// Sub returns the elementwise difference xs-ys as a new slice.
//
// If the slices differ in length, the result has length min(len(xs), len(ys)). If
// either slice is empty the result is nil.
//
// Sub allocates. For hot loops that can supply their own destination buffer, use
// [SubTo].
//
// Note that `(xs+ys)-ys` is *not* exactly `xs` in floating point: adding ys[i] can
// round away low bits of xs[i] that subtracting it back cannot restore. The round
// trip is accurate only to within a relative epsilon, which the test suite asserts
// rather than pretending otherwise.
func Sub(xs, ys []float64) []float64 {
	n := min(len(xs), len(ys))
	if n == 0 {
		return nil
	}
	return subArch(make([]float64, n), xs[:n], ys[:n])
}

// SubTo stores the elementwise difference xs-ys into dst and returns dst.
//
// The three slices must be the same length. SubTo panics if they are not.
//
// dst may alias xs or ys.
func SubTo(dst, xs, ys []float64) []float64 {
	checkSameLen("SubTo", dst, xs, ys)
	return subArch(dst, xs, ys)
}
