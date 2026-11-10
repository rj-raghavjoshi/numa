package vec

// ---------------------------------------------------------------------------
// Scalar elementwise kernels.
//
// AddScalar and Scale cover the two common scalar forms. The four operations
// here cover the rest of the matrix without forcing a caller to materialise a
// broadcast slice: xs-k, k-xs, xs/k, and k/xs. They are single-pass kernels and
// write through the same destination contract as the rest of the package.
// ---------------------------------------------------------------------------

// SubScalar returns xs[i] - k for each element of xs.
func SubScalar(xs []float64, k float64) []float64 {
	if len(xs) == 0 {
		return nil
	}
	return SubScalarTo(make([]float64, len(xs)), xs, k)
}

// SubScalarTo stores xs[i] - k into dst and returns dst.
// dst and xs must be the same length; dst may alias xs.
func SubScalarTo(dst, xs []float64, k float64) []float64 {
	checkSameLen1("SubScalarTo", dst, xs)
	for i := range dst {
		dst[i] = xs[i] - k
	}
	return dst
}

// RSubScalar returns k - xs[i] for each element of xs (the reversed form).
func RSubScalar(xs []float64, k float64) []float64 {
	if len(xs) == 0 {
		return nil
	}
	return RSubScalarTo(make([]float64, len(xs)), xs, k)
}

// RSubScalarTo stores k - xs[i] into dst and returns dst.
// dst and xs must be the same length; dst may alias xs.
func RSubScalarTo(dst, xs []float64, k float64) []float64 {
	checkSameLen1("RSubScalarTo", dst, xs)
	for i := range dst {
		dst[i] = k - xs[i]
	}
	return dst
}

// DivScalar returns xs[i] / k for each element of xs. Division by zero follows
// IEEE-754 (Inf or NaN), the same contract as [Div]; use [SafeDiv] when a zero
// denominator must map to zero instead.
func DivScalar(xs []float64, k float64) []float64 {
	if len(xs) == 0 {
		return nil
	}
	return DivScalarTo(make([]float64, len(xs)), xs, k)
}

// DivScalarTo stores xs[i] / k into dst and returns dst.
// dst and xs must be the same length; dst may alias xs.
func DivScalarTo(dst, xs []float64, k float64) []float64 {
	checkSameLen1("DivScalarTo", dst, xs)
	for i := range dst {
		dst[i] = xs[i] / k
	}
	return dst
}

// RDivScalar returns k / xs[i] for each element of xs (the reversed form).
// A zero denominator follows IEEE-754.
func RDivScalar(xs []float64, k float64) []float64 {
	if len(xs) == 0 {
		return nil
	}
	return RDivScalarTo(make([]float64, len(xs)), xs, k)
}

// RDivScalarTo stores k / xs[i] into dst and returns dst.
// dst and xs must be the same length; dst may alias xs.
func RDivScalarTo(dst, xs []float64, k float64) []float64 {
	checkSameLen1("RDivScalarTo", dst, xs)
	for i := range dst {
		dst[i] = k / xs[i]
	}
	return dst
}
