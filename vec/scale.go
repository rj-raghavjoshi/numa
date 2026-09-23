package vec

// Scale returns xs multiplied by the scalar k, as a new slice.
//
// It returns nil for an empty or nil input.
//
// Scale allocates. For hot loops that can supply their own destination buffer, use
// [ScaleTo].
//
// This is the workhorse of gradient-descent and normalization steps, where a whole
// parameter vector is multiplied by a learning rate in place. Prefer [ScaleTo] in
// those loops.
func Scale(xs []float64, k float64) []float64 {
	if len(xs) == 0 {
		return nil
	}
	return scaleArch(make([]float64, len(xs)), xs, k)
}

// ScaleTo stores xs multiplied by the scalar k into dst and returns dst.
//
// dst and xs must be the same length. ScaleTo panics if they are not.
//
// dst may alias xs, so `ScaleTo(w, w, 0.9)` decays a vector in place.
func ScaleTo(dst, xs []float64, k float64) []float64 {
	checkSameLen1("ScaleTo", dst, xs)
	return scaleArch(dst, xs, k)
}
