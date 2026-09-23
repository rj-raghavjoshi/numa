package vec

import "math"

// Abs returns the absolute value of each element of xs, as a new slice.
//
// It returns nil for an empty or nil input.
//
// Abs allocates. For hot loops that can supply their own destination buffer, use
// [AbsTo].
func Abs(xs []float64) []float64 {
	if len(xs) == 0 {
		return nil
	}
	return absArch(make([]float64, len(xs)), xs)
}

// AbsTo stores the absolute value of each element of xs into dst and returns dst.
//
// dst and xs must be the same length. AbsTo panics if they are not.
//
// dst may alias xs.
func AbsTo(dst, xs []float64) []float64 {
	checkSameLen1("AbsTo", dst, xs)
	return absArch(dst, xs)
}

// abs returns the absolute value of v.
//
// It exists so the tuned loops read uniformly, and so that the definition of
// "absolute value" lives in exactly one place. math.Abs compiles to a single
// instruction on both supported architectures, so the wrapper costs nothing.
func abs(v float64) float64 {
	return math.Abs(v)
}
