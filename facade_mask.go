package numa

import "github.com/rj-raghavjoshi/numa/vec"

// ---------------------------------------------------------------------------
// Facade forwarders for vec/mask.go: predicates, mask algebra, selection, and
// masked reductions.
//
// Each is a one-line forwarder to vec. See facade_math.go for why the forwarders
// are split per family, and facade_mask_test.go for the forwarding tests.
// ---------------------------------------------------------------------------

// Greater returns a mask that is 1 where xs[i] > ys[i]. See [vec.Greater].
func Greater(xs, ys []float64) []uint8 { return vec.Greater(xs, ys) }

// GreaterTo stores the mask xs[i] > ys[i] into dst. See [vec.GreaterTo].
func GreaterTo(dst []uint8, xs, ys []float64) []uint8 { return vec.GreaterTo(dst, xs, ys) }

// GreaterEqual returns a mask that is 1 where xs[i] >= ys[i]. See [vec.GreaterEqual].
func GreaterEqual(xs, ys []float64) []uint8 { return vec.GreaterEqual(xs, ys) }

// GreaterEqualTo stores the mask xs[i] >= ys[i] into dst. See [vec.GreaterEqualTo].
func GreaterEqualTo(dst []uint8, xs, ys []float64) []uint8 {
	return vec.GreaterEqualTo(dst, xs, ys)
}

// Less returns a mask that is 1 where xs[i] < ys[i]. See [vec.Less].
func Less(xs, ys []float64) []uint8 { return vec.Less(xs, ys) }

// LessTo stores the mask xs[i] < ys[i] into dst. See [vec.LessTo].
func LessTo(dst []uint8, xs, ys []float64) []uint8 { return vec.LessTo(dst, xs, ys) }

// LessEqual returns a mask that is 1 where xs[i] <= ys[i]. See [vec.LessEqual].
func LessEqual(xs, ys []float64) []uint8 { return vec.LessEqual(xs, ys) }

// LessEqualTo stores the mask xs[i] <= ys[i] into dst. See [vec.LessEqualTo].
func LessEqualTo(dst []uint8, xs, ys []float64) []uint8 { return vec.LessEqualTo(dst, xs, ys) }

// Equal returns a mask that is 1 where xs[i] == ys[i]. See [vec.Equal].
func Equal(xs, ys []float64) []uint8 { return vec.Equal(xs, ys) }

// EqualTo stores the mask xs[i] == ys[i] into dst. See [vec.EqualTo].
func EqualTo(dst []uint8, xs, ys []float64) []uint8 { return vec.EqualTo(dst, xs, ys) }

// NotEqual returns a mask that is 1 where xs[i] != ys[i]. See [vec.NotEqual].
func NotEqual(xs, ys []float64) []uint8 { return vec.NotEqual(xs, ys) }

// NotEqualTo stores the mask xs[i] != ys[i] into dst. See [vec.NotEqualTo].
func NotEqualTo(dst []uint8, xs, ys []float64) []uint8 { return vec.NotEqualTo(dst, xs, ys) }

// GreaterScalar returns a mask that is 1 where xs[i] > k. See [vec.GreaterScalar].
func GreaterScalar(xs []float64, k float64) []uint8 { return vec.GreaterScalar(xs, k) }

// GreaterScalarTo stores the mask xs[i] > k into dst. See [vec.GreaterScalarTo].
func GreaterScalarTo(dst []uint8, xs []float64, k float64) []uint8 {
	return vec.GreaterScalarTo(dst, xs, k)
}

// LessScalar returns a mask that is 1 where xs[i] < k. See [vec.LessScalar].
func LessScalar(xs []float64, k float64) []uint8 { return vec.LessScalar(xs, k) }

// LessScalarTo stores the mask xs[i] < k into dst. See [vec.LessScalarTo].
func LessScalarTo(dst []uint8, xs []float64, k float64) []uint8 {
	return vec.LessScalarTo(dst, xs, k)
}

// GreaterEqualScalar returns a mask that is 1 where xs[i] >= k.
// See [vec.GreaterEqualScalar].
func GreaterEqualScalar(xs []float64, k float64) []uint8 { return vec.GreaterEqualScalar(xs, k) }

// GreaterEqualScalarTo stores the mask xs[i] >= k into dst.
// See [vec.GreaterEqualScalarTo].
func GreaterEqualScalarTo(dst []uint8, xs []float64, k float64) []uint8 {
	return vec.GreaterEqualScalarTo(dst, xs, k)
}

// LessEqualScalar returns a mask that is 1 where xs[i] <= k. See [vec.LessEqualScalar].
func LessEqualScalar(xs []float64, k float64) []uint8 { return vec.LessEqualScalar(xs, k) }

// LessEqualScalarTo stores the mask xs[i] <= k into dst. See [vec.LessEqualScalarTo].
func LessEqualScalarTo(dst []uint8, xs []float64, k float64) []uint8 {
	return vec.LessEqualScalarTo(dst, xs, k)
}

// EqualScalar returns a mask that is 1 where xs[i] == k. See [vec.EqualScalar].
func EqualScalar(xs []float64, k float64) []uint8 { return vec.EqualScalar(xs, k) }

// EqualScalarTo stores the mask xs[i] == k into dst. See [vec.EqualScalarTo].
func EqualScalarTo(dst []uint8, xs []float64, k float64) []uint8 {
	return vec.EqualScalarTo(dst, xs, k)
}

// NotEqualScalar returns a mask that is 1 where xs[i] != k. See [vec.NotEqualScalar].
func NotEqualScalar(xs []float64, k float64) []uint8 { return vec.NotEqualScalar(xs, k) }

// NotEqualScalarTo stores the mask xs[i] != k into dst. See [vec.NotEqualScalarTo].
func NotEqualScalarTo(dst []uint8, xs []float64, k float64) []uint8 {
	return vec.NotEqualScalarTo(dst, xs, k)
}

// IsNaN returns a mask that is 1 where xs[i] is NaN. See [vec.IsNaN].
func IsNaN(xs []float64) []uint8 { return vec.IsNaN(xs) }

// IsNaNTo stores the NaN mask of xs into dst. See [vec.IsNaNTo].
func IsNaNTo(dst []uint8, xs []float64) []uint8 { return vec.IsNaNTo(dst, xs) }

// IsFinite returns a mask that is 1 where xs[i] is neither NaN nor infinite.
// See [vec.IsFinite].
func IsFinite(xs []float64) []uint8 { return vec.IsFinite(xs) }

// IsFiniteTo stores the finiteness mask of xs into dst. See [vec.IsFiniteTo].
func IsFiniteTo(dst []uint8, xs []float64) []uint8 { return vec.IsFiniteTo(dst, xs) }

// IsInf returns a mask that is 1 where xs[i] is an infinity of the given sign.
// See [vec.IsInf].
func IsInf(xs []float64, sign int) []uint8 { return vec.IsInf(xs, sign) }

// IsInfTo stores the infinity mask of xs into dst. See [vec.IsInfTo].
func IsInfTo(dst []uint8, xs []float64, sign int) []uint8 { return vec.IsInfTo(dst, xs, sign) }

// And returns a mask that is 1 where both a and b are non-zero. See [vec.And].
func And(a, b []uint8) []uint8 { return vec.And(a, b) }

// AndTo stores the conjunction of a and b into dst. See [vec.AndTo].
func AndTo(dst, a, b []uint8) []uint8 { return vec.AndTo(dst, a, b) }

// Or returns a mask that is 1 where either a or b is non-zero. See [vec.Or].
func Or(a, b []uint8) []uint8 { return vec.Or(a, b) }

// OrTo stores the disjunction of a and b into dst. See [vec.OrTo].
func OrTo(dst, a, b []uint8) []uint8 { return vec.OrTo(dst, a, b) }

// Xor returns a mask that is 1 where exactly one of a and b is non-zero.
// See [vec.Xor].
func Xor(a, b []uint8) []uint8 { return vec.Xor(a, b) }

// XorTo stores the exclusive-or of a and b into dst. See [vec.XorTo].
func XorTo(dst, a, b []uint8) []uint8 { return vec.XorTo(dst, a, b) }

// Not returns the negation of a mask. See [vec.Not].
func Not(a []uint8) []uint8 { return vec.Not(a) }

// NotTo stores the negation of a into dst. See [vec.NotTo].
func NotTo(dst, a []uint8) []uint8 { return vec.NotTo(dst, a) }

// Where selects a[i] where cond[i] is non-zero and b[i] otherwise. See [vec.Where].
func Where(cond []uint8, a, b []float64) []float64 { return vec.Where(cond, a, b) }

// WhereTo stores the selection into dst. See [vec.WhereTo].
func WhereTo(dst []float64, cond []uint8, a, b []float64) []float64 {
	return vec.WhereTo(dst, cond, a, b)
}

// CountTrue returns the number of non-zero entries in mask. See [vec.CountTrue].
func CountTrue(mask []uint8) int { return vec.CountTrue(mask) }

// AnyTrue reports whether mask contains any non-zero entry. See [vec.AnyTrue].
func AnyTrue(mask []uint8) bool { return vec.AnyTrue(mask) }

// AllTrue reports whether mask is entirely non-zero. See [vec.AllTrue].
func AllTrue(mask []uint8) bool { return vec.AllTrue(mask) }

// Compress gathers the selected elements of xs into a new slice. See [vec.Compress].
func Compress(xs []float64, mask []uint8) []float64 { return vec.Compress(xs, mask) }

// CompressTo gathers the selected elements of xs into dst. See [vec.CompressTo].
func CompressTo(dst, xs []float64, mask []uint8) []float64 {
	return vec.CompressTo(dst, xs, mask)
}

// MaskedSum returns the sum of the selected elements of xs. See [vec.MaskedSum].
func MaskedSum(xs []float64, mask []uint8) float64 { return vec.MaskedSum(xs, mask) }

// MaskedMean returns the mean of the selected elements of xs. See [vec.MaskedMean].
func MaskedMean(xs []float64, mask []uint8) float64 { return vec.MaskedMean(xs, mask) }

// MaskedMin returns the smallest selected element of xs. See [vec.MaskedMin].
func MaskedMin(xs []float64, mask []uint8) float64 { return vec.MaskedMin(xs, mask) }

// MaskedMax returns the largest selected element of xs. See [vec.MaskedMax].
func MaskedMax(xs []float64, mask []uint8) float64 { return vec.MaskedMax(xs, mask) }
