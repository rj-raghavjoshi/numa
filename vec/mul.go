package vec

// Mul returns the elementwise product of xs and ys as a new slice.
//
// If the slices differ in length, the result has length min(len(xs), len(ys)). If
// either slice is empty the result is nil.
//
// Mul allocates. For hot loops that can supply their own destination buffer, use
// [MulTo].
func Mul(xs, ys []float64) []float64 {
	n := min(len(xs), len(ys))
	if n == 0 {
		return nil
	}
	return mulArch(make([]float64, n), xs[:n], ys[:n])
}

// MulTo stores the elementwise product of xs and ys into dst and returns dst.
//
// The three slices must be the same length. MulTo panics if they are not.
//
// dst may alias xs or ys.
func MulTo(dst, xs, ys []float64) []float64 {
	checkSameLen("MulTo", dst, xs, ys)
	return mulArch(dst, xs, ys)
}
