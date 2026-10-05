package vec

import "math"

// ---------------------------------------------------------------------------
// Predicates, masks, and masked reductions.
//
// A predicate is a map from float64 to a byte, and a mask is a []uint8 of 0/1.
// Any non-zero byte is treated as true on input, but the producers here always
// write exactly 0 or 1 so that a mask can be summed, counted, or compared against
// a length cheaply.
//
// # NaN semantics
//
// Predicates are plain IEEE-754 comparisons, so a NaN operand makes every
// ordered comparison false: Greater, Less, GreaterEqual and LessEqual all return
// 0, Equal returns 0, and NotEqual returns 1 because NaN != NaN is true. That is
// deliberate: it means a mask can never claim an ordering that the hardware does
// not support, and a caller who must exclude NaN uses [IsNaN] or [IsFinite]
// explicitly rather than relying on a silent skip.
//
// # Why predicates are unrolled
//
// A predicate reads 16 bytes and writes one per element, so it is far
// store-lighter than a float map. It is nonetheless compare-heavy, and the same
// effect that makes [Min2] gain more than [Neg] applies: several independent
// compares in flight are cheaper than one. The four-wide unroll is therefore kept.
//
// # Naming
//
// The Scalar suffix marks a comparison against a constant, which is the common
// case in an indicator (`rsi > 70`). It exists so that a constant threshold does
// not require materialising a filled slice to compare against.
// ---------------------------------------------------------------------------

// boolToU8 is the canonical mask producer. Written so the compiler emits a
// conditional select rather than a branch.
func boolToU8(b bool) uint8 {
	if b {
		return 1
	}
	return 0
}

// checkMask2 panics unless dst and the two float inputs have the same length.
func checkMask2(op string, dst []uint8, xs, ys []float64) {
	if len(dst) != len(xs) || len(dst) != len(ys) {
		panic("vec: " + op + " length mismatch")
	}
}

// checkMask1 panics unless dst and xs have the same length.
func checkMask1(op string, dst []uint8, xs []float64) {
	if len(dst) != len(xs) {
		panic("vec: " + op + " length mismatch")
	}
}

// checkMaskMask panics unless all three slices have the same length.
func checkMaskMask(op string, dst, a, b []uint8) {
	if len(dst) != len(a) || len(dst) != len(b) {
		panic("vec: " + op + " length mismatch")
	}
}

// ---------------------------------------------------------------------------
// Comparisons against a series.
// ---------------------------------------------------------------------------

// Greater returns a mask that is 1 where xs[i] > ys[i], over
// min(len(xs), len(ys)) elements.
//
// Greater allocates; for hot loops that can supply a buffer, use [GreaterTo].
func Greater(xs, ys []float64) []uint8 {
	n := min(len(xs), len(ys))
	if n == 0 {
		return nil
	}
	return GreaterTo(make([]uint8, n), xs[:n], ys[:n])
}

// GreaterTo stores the mask xs[i] > ys[i] into dst and returns dst.
//
// The three slices must be the same length. GreaterTo panics if they are not.
func GreaterTo(dst []uint8, xs, ys []float64) []uint8 {
	checkMask2("GreaterTo", dst, xs, ys)
	n := len(dst)
	i := 0
	limit := n - 3
	for i < limit {
		dst[i] = boolToU8(xs[i] > ys[i])
		dst[i+1] = boolToU8(xs[i+1] > ys[i+1])
		dst[i+2] = boolToU8(xs[i+2] > ys[i+2])
		dst[i+3] = boolToU8(xs[i+3] > ys[i+3])
		i += 4
	}
	for ; i < n; i++ {
		dst[i] = boolToU8(xs[i] > ys[i])
	}
	return dst
}

// GreaterEqual returns a mask that is 1 where xs[i] >= ys[i].
func GreaterEqual(xs, ys []float64) []uint8 {
	n := min(len(xs), len(ys))
	if n == 0 {
		return nil
	}
	return GreaterEqualTo(make([]uint8, n), xs[:n], ys[:n])
}

// GreaterEqualTo stores the mask xs[i] >= ys[i] into dst and returns dst.
//
// The three slices must be the same length. GreaterEqualTo panics if they are not.
func GreaterEqualTo(dst []uint8, xs, ys []float64) []uint8 {
	checkMask2("GreaterEqualTo", dst, xs, ys)
	n := len(dst)
	i := 0
	limit := n - 3
	for i < limit {
		dst[i] = boolToU8(xs[i] >= ys[i])
		dst[i+1] = boolToU8(xs[i+1] >= ys[i+1])
		dst[i+2] = boolToU8(xs[i+2] >= ys[i+2])
		dst[i+3] = boolToU8(xs[i+3] >= ys[i+3])
		i += 4
	}
	for ; i < n; i++ {
		dst[i] = boolToU8(xs[i] >= ys[i])
	}
	return dst
}

// Less returns a mask that is 1 where xs[i] < ys[i].
func Less(xs, ys []float64) []uint8 {
	n := min(len(xs), len(ys))
	if n == 0 {
		return nil
	}
	return LessTo(make([]uint8, n), xs[:n], ys[:n])
}

// LessTo stores the mask xs[i] < ys[i] into dst and returns dst.
//
// The three slices must be the same length. LessTo panics if they are not.
func LessTo(dst []uint8, xs, ys []float64) []uint8 {
	checkMask2("LessTo", dst, xs, ys)
	n := len(dst)
	i := 0
	limit := n - 3
	for i < limit {
		dst[i] = boolToU8(xs[i] < ys[i])
		dst[i+1] = boolToU8(xs[i+1] < ys[i+1])
		dst[i+2] = boolToU8(xs[i+2] < ys[i+2])
		dst[i+3] = boolToU8(xs[i+3] < ys[i+3])
		i += 4
	}
	for ; i < n; i++ {
		dst[i] = boolToU8(xs[i] < ys[i])
	}
	return dst
}

// LessEqual returns a mask that is 1 where xs[i] <= ys[i].
func LessEqual(xs, ys []float64) []uint8 {
	n := min(len(xs), len(ys))
	if n == 0 {
		return nil
	}
	return LessEqualTo(make([]uint8, n), xs[:n], ys[:n])
}

// LessEqualTo stores the mask xs[i] <= ys[i] into dst and returns dst.
//
// The three slices must be the same length. LessEqualTo panics if they are not.
func LessEqualTo(dst []uint8, xs, ys []float64) []uint8 {
	checkMask2("LessEqualTo", dst, xs, ys)
	n := len(dst)
	i := 0
	limit := n - 3
	for i < limit {
		dst[i] = boolToU8(xs[i] <= ys[i])
		dst[i+1] = boolToU8(xs[i+1] <= ys[i+1])
		dst[i+2] = boolToU8(xs[i+2] <= ys[i+2])
		dst[i+3] = boolToU8(xs[i+3] <= ys[i+3])
		i += 4
	}
	for ; i < n; i++ {
		dst[i] = boolToU8(xs[i] <= ys[i])
	}
	return dst
}

// Equal returns a mask that is 1 where xs[i] == ys[i].
//
// This is an exact comparison, deliberately: it is what a caller asking for
// equality means, and approximate equality belongs in a tolerance-taking
// function rather than hidden here. NaN is equal to nothing, including itself.
func Equal(xs, ys []float64) []uint8 {
	n := min(len(xs), len(ys))
	if n == 0 {
		return nil
	}
	return EqualTo(make([]uint8, n), xs[:n], ys[:n])
}

// EqualTo stores the mask xs[i] == ys[i] into dst and returns dst.
//
// The three slices must be the same length. EqualTo panics if they are not.
func EqualTo(dst []uint8, xs, ys []float64) []uint8 {
	checkMask2("EqualTo", dst, xs, ys)
	n := len(dst)
	i := 0
	limit := n - 3
	for i < limit {
		dst[i] = boolToU8(xs[i] == ys[i])
		dst[i+1] = boolToU8(xs[i+1] == ys[i+1])
		dst[i+2] = boolToU8(xs[i+2] == ys[i+2])
		dst[i+3] = boolToU8(xs[i+3] == ys[i+3])
		i += 4
	}
	for ; i < n; i++ {
		dst[i] = boolToU8(xs[i] == ys[i])
	}
	return dst
}

// NotEqual returns a mask that is 1 where xs[i] != ys[i].
//
// Because NaN != NaN is true, a NaN operand yields 1 here while every ordered
// comparison yields 0.
func NotEqual(xs, ys []float64) []uint8 {
	n := min(len(xs), len(ys))
	if n == 0 {
		return nil
	}
	return NotEqualTo(make([]uint8, n), xs[:n], ys[:n])
}

// NotEqualTo stores the mask xs[i] != ys[i] into dst and returns dst.
//
// The three slices must be the same length. NotEqualTo panics if they are not.
func NotEqualTo(dst []uint8, xs, ys []float64) []uint8 {
	checkMask2("NotEqualTo", dst, xs, ys)
	n := len(dst)
	i := 0
	limit := n - 3
	for i < limit {
		dst[i] = boolToU8(xs[i] != ys[i])
		dst[i+1] = boolToU8(xs[i+1] != ys[i+1])
		dst[i+2] = boolToU8(xs[i+2] != ys[i+2])
		dst[i+3] = boolToU8(xs[i+3] != ys[i+3])
		i += 4
	}
	for ; i < n; i++ {
		dst[i] = boolToU8(xs[i] != ys[i])
	}
	return dst
}

// ---------------------------------------------------------------------------
// Comparisons against a constant.
//
// These exist so that a threshold such as `rsi > 70` does not require filling a
// slice of 70s to compare against. That allocation is not merely wasteful; in a
// hot loop it is the allocation, because the comparison itself is one instruction.
// ---------------------------------------------------------------------------

// GreaterScalar returns a mask that is 1 where xs[i] > k.
func GreaterScalar(xs []float64, k float64) []uint8 {
	if len(xs) == 0 {
		return nil
	}
	return GreaterScalarTo(make([]uint8, len(xs)), xs, k)
}

// GreaterScalarTo stores the mask xs[i] > k into dst and returns dst.
//
// dst and xs must be the same length. GreaterScalarTo panics if they are not.
func GreaterScalarTo(dst []uint8, xs []float64, k float64) []uint8 {
	checkMask1("GreaterScalarTo", dst, xs)
	for i, v := range xs {
		dst[i] = boolToU8(v > k)
	}
	return dst
}

// LessScalar returns a mask that is 1 where xs[i] < k.
func LessScalar(xs []float64, k float64) []uint8 {
	if len(xs) == 0 {
		return nil
	}
	return LessScalarTo(make([]uint8, len(xs)), xs, k)
}

// LessScalarTo stores the mask xs[i] < k into dst and returns dst.
//
// dst and xs must be the same length. LessScalarTo panics if they are not.
func LessScalarTo(dst []uint8, xs []float64, k float64) []uint8 {
	checkMask1("LessScalarTo", dst, xs)
	for i, v := range xs {
		dst[i] = boolToU8(v < k)
	}
	return dst
}

// GreaterEqualScalar returns a mask that is 1 where xs[i] >= k.
func GreaterEqualScalar(xs []float64, k float64) []uint8 {
	if len(xs) == 0 {
		return nil
	}
	return GreaterEqualScalarTo(make([]uint8, len(xs)), xs, k)
}

// GreaterEqualScalarTo stores the mask xs[i] >= k into dst and returns dst.
//
// dst and xs must be the same length. GreaterEqualScalarTo panics if they are not.
func GreaterEqualScalarTo(dst []uint8, xs []float64, k float64) []uint8 {
	checkMask1("GreaterEqualScalarTo", dst, xs)
	for i, v := range xs {
		dst[i] = boolToU8(v >= k)
	}
	return dst
}

// LessEqualScalar returns a mask that is 1 where xs[i] <= k.
func LessEqualScalar(xs []float64, k float64) []uint8 {
	if len(xs) == 0 {
		return nil
	}
	return LessEqualScalarTo(make([]uint8, len(xs)), xs, k)
}

// LessEqualScalarTo stores the mask xs[i] <= k into dst and returns dst.
//
// dst and xs must be the same length. LessEqualScalarTo panics if they are not.
func LessEqualScalarTo(dst []uint8, xs []float64, k float64) []uint8 {
	checkMask1("LessEqualScalarTo", dst, xs)
	for i, v := range xs {
		dst[i] = boolToU8(v <= k)
	}
	return dst
}

// EqualScalar returns a mask that is 1 where xs[i] == k.
func EqualScalar(xs []float64, k float64) []uint8 {
	if len(xs) == 0 {
		return nil
	}
	return EqualScalarTo(make([]uint8, len(xs)), xs, k)
}

// EqualScalarTo stores the mask xs[i] == k into dst and returns dst.
//
// dst and xs must be the same length. EqualScalarTo panics if they are not.
func EqualScalarTo(dst []uint8, xs []float64, k float64) []uint8 {
	checkMask1("EqualScalarTo", dst, xs)
	for i, v := range xs {
		dst[i] = boolToU8(v == k)
	}
	return dst
}

// NotEqualScalar returns a mask that is 1 where xs[i] != k.
func NotEqualScalar(xs []float64, k float64) []uint8 {
	if len(xs) == 0 {
		return nil
	}
	return NotEqualScalarTo(make([]uint8, len(xs)), xs, k)
}

// NotEqualScalarTo stores the mask xs[i] != k into dst and returns dst.
//
// dst and xs must be the same length. NotEqualScalarTo panics if they are not.
func NotEqualScalarTo(dst []uint8, xs []float64, k float64) []uint8 {
	checkMask1("NotEqualScalarTo", dst, xs)
	for i, v := range xs {
		dst[i] = boolToU8(v != k)
	}
	return dst
}

// ---------------------------------------------------------------------------
// Value-class predicates.
// ---------------------------------------------------------------------------

// IsNaN returns a mask that is 1 where xs[i] is NaN.
func IsNaN(xs []float64) []uint8 {
	if len(xs) == 0 {
		return nil
	}
	return IsNaNTo(make([]uint8, len(xs)), xs)
}

// IsNaNTo stores the NaN mask of xs into dst and returns dst.
//
// dst and xs must be the same length. IsNaNTo panics if they are not.
func IsNaNTo(dst []uint8, xs []float64) []uint8 {
	checkMask1("IsNaNTo", dst, xs)
	for i, v := range xs {
		dst[i] = boolToU8(v != v)
	}
	return dst
}

// IsFinite returns a mask that is 1 where xs[i] is neither NaN nor an infinity.
func IsFinite(xs []float64) []uint8 {
	if len(xs) == 0 {
		return nil
	}
	return IsFiniteTo(make([]uint8, len(xs)), xs)
}

// IsFiniteTo stores the finiteness mask of xs into dst and returns dst.
//
// dst and xs must be the same length. IsFiniteTo panics if they are not.
func IsFiniteTo(dst []uint8, xs []float64) []uint8 {
	checkMask1("IsFiniteTo", dst, xs)
	for i, v := range xs {
		dst[i] = boolToU8(v == v && !math.IsInf(v, 0))
	}
	return dst
}

// IsInf returns a mask that is 1 where xs[i] is an infinity with the given sign.
//
// sign > 0 matches +Inf, sign < 0 matches -Inf, and sign == 0 matches either.
func IsInf(xs []float64, sign int) []uint8 {
	if len(xs) == 0 {
		return nil
	}
	return IsInfTo(make([]uint8, len(xs)), xs, sign)
}

// IsInfTo stores the infinity mask of xs into dst and returns dst.
//
// dst and xs must be the same length. IsInfTo panics if they are not.
func IsInfTo(dst []uint8, xs []float64, sign int) []uint8 {
	checkMask1("IsInfTo", dst, xs)
	for i, v := range xs {
		dst[i] = boolToU8(math.IsInf(v, sign))
	}
	return dst
}

// ---------------------------------------------------------------------------
// Mask algebra.
// ---------------------------------------------------------------------------

// And returns a mask that is 1 where both a and b are non-zero.
func And(a, b []uint8) []uint8 {
	n := min(len(a), len(b))
	if n == 0 {
		return nil
	}
	return AndTo(make([]uint8, n), a[:n], b[:n])
}

// AndTo stores the conjunction of a and b into dst and returns dst.
//
// The three slices must be the same length. AndTo panics if they are not.
func AndTo(dst, a, b []uint8) []uint8 {
	checkMaskMask("AndTo", dst, a, b)
	n := len(dst)
	i := 0
	limit := n - 3
	for i < limit {
		dst[i] = boolToU8(a[i] != 0 && b[i] != 0)
		dst[i+1] = boolToU8(a[i+1] != 0 && b[i+1] != 0)
		dst[i+2] = boolToU8(a[i+2] != 0 && b[i+2] != 0)
		dst[i+3] = boolToU8(a[i+3] != 0 && b[i+3] != 0)
		i += 4
	}
	for ; i < n; i++ {
		dst[i] = boolToU8(a[i] != 0 && b[i] != 0)
	}
	return dst
}

// Or returns a mask that is 1 where either a or b is non-zero.
func Or(a, b []uint8) []uint8 {
	n := min(len(a), len(b))
	if n == 0 {
		return nil
	}
	return OrTo(make([]uint8, n), a[:n], b[:n])
}

// OrTo stores the disjunction of a and b into dst and returns dst.
//
// The three slices must be the same length. OrTo panics if they are not.
func OrTo(dst, a, b []uint8) []uint8 {
	checkMaskMask("OrTo", dst, a, b)
	n := len(dst)
	i := 0
	limit := n - 3
	for i < limit {
		dst[i] = boolToU8(a[i] != 0 || b[i] != 0)
		dst[i+1] = boolToU8(a[i+1] != 0 || b[i+1] != 0)
		dst[i+2] = boolToU8(a[i+2] != 0 || b[i+2] != 0)
		dst[i+3] = boolToU8(a[i+3] != 0 || b[i+3] != 0)
		i += 4
	}
	for ; i < n; i++ {
		dst[i] = boolToU8(a[i] != 0 || b[i] != 0)
	}
	return dst
}

// Xor returns a mask that is 1 where exactly one of a and b is non-zero.
//
// It is defined on truth, not on bit patterns: two bytes that are both non-zero
// but unequal (say 1 and 2) are both true, so their exclusive-or is false. That
// makes the operation total over any non-zero-means-true input, rather than
// silently requiring the 0/1 convention.
func Xor(a, b []uint8) []uint8 {
	n := min(len(a), len(b))
	if n == 0 {
		return nil
	}
	return XorTo(make([]uint8, n), a[:n], b[:n])
}

// XorTo stores the exclusive-or of a and b into dst and returns dst.
//
// The three slices must be the same length. XorTo panics if they are not.
func XorTo(dst, a, b []uint8) []uint8 {
	checkMaskMask("XorTo", dst, a, b)
	for i := range dst {
		dst[i] = boolToU8((a[i] != 0) != (b[i] != 0))
	}
	return dst
}

// Not returns a mask that is 1 wherever a is 0, and 0 wherever a is non-zero.
func Not(a []uint8) []uint8 {
	if len(a) == 0 {
		return nil
	}
	return NotTo(make([]uint8, len(a)), a)
}

// NotTo stores the negation of a into dst and returns dst.
//
// dst and a must be the same length. NotTo panics if they are not.
func NotTo(dst, a []uint8) []uint8 {
	if len(dst) != len(a) {
		panic("vec: NotTo length mismatch")
	}
	for i, v := range a {
		dst[i] = boolToU8(v == 0)
	}
	return dst
}

// ---------------------------------------------------------------------------
// Selection.
// ---------------------------------------------------------------------------

// Where returns a new slice that takes a[i] where cond[i] is non-zero and b[i]
// otherwise, over min(len(cond), len(a), len(b)) elements.
//
// Where allocates; for hot loops that can supply a buffer, use [WhereTo].
func Where(cond []uint8, a, b []float64) []float64 {
	n := min(len(cond), min(len(a), len(b)))
	if n == 0 {
		return nil
	}
	return WhereTo(make([]float64, n), cond[:n], a[:n], b[:n])
}

// WhereTo stores the selection into dst and returns dst.
//
// The four slices must be the same length. WhereTo panics if they are not. dst
// may alias a or b.
func WhereTo(dst []float64, cond []uint8, a, b []float64) []float64 {
	if len(dst) != len(cond) || len(dst) != len(a) || len(dst) != len(b) {
		panic("vec: WhereTo length mismatch")
	}
	for i := range dst {
		if cond[i] != 0 {
			dst[i] = a[i]
		} else {
			dst[i] = b[i]
		}
	}
	return dst
}

// ---------------------------------------------------------------------------
// Mask reductions.
// ---------------------------------------------------------------------------

// CountTrue returns the number of non-zero entries in mask.
func CountTrue(mask []uint8) int {
	var n int
	for _, v := range mask {
		if v != 0 {
			n++
		}
	}
	return n
}

// AnyTrue reports whether mask contains any non-zero entry.
func AnyTrue(mask []uint8) bool {
	for _, v := range mask {
		if v != 0 {
			return true
		}
	}
	return false
}

// AllTrue reports whether mask is entirely non-zero.
//
// An empty mask is all-true, which is the identity for conjunction; callers who
// need to distinguish "no elements" from "all elements pass" must check the
// length themselves.
func AllTrue(mask []uint8) bool {
	for _, v := range mask {
		if v == 0 {
			return false
		}
	}
	return true
}

// Compress gathers the elements of xs whose mask is non-zero, in order, into a
// new slice of exactly the selected length.
//
// Compress does not forward to [CompressTo]: that function requires a destination
// the length of the input, and allocating one here would then hand the caller a
// subslice of a buffer that is larger than the result it appears to own. The
// gather is duplicated so the allocation matches the answer.
func Compress(xs []float64, mask []uint8) []float64 {
	n := min(len(xs), len(mask))
	if n == 0 {
		return nil
	}
	k := CountTrue(mask[:n])
	if k == 0 {
		return nil
	}
	out := make([]float64, k)
	j := 0
	for i := 0; i < n; i++ {
		if mask[i] != 0 {
			out[j] = xs[i]
			j++
		}
	}
	return out
}

// CompressTo gathers the selected elements of xs into dst and returns dst[:k],
// where k is the number of non-zero mask entries.
//
// dst, xs and mask must all be the same length; CompressTo panics if they are
// not, and the return value is a subslice of dst whose length is k. Requiring dst
// to be the full input length (rather than the surviving count) keeps the check
// uniform with the rest of the package and lets a caller reuse one buffer across
// calls with different selection counts.
func CompressTo(dst []float64, xs []float64, mask []uint8) []float64 {
	if len(dst) != len(xs) || len(dst) != len(mask) {
		panic("vec: CompressTo length mismatch")
	}
	k := 0
	for i, v := range xs {
		if mask[i] != 0 {
			dst[k] = v
			k++
		}
	}
	return dst[:k]
}

// MaskedSum returns the sum of the xs elements selected by mask.
//
// It returns 0 when nothing is selected. A selected NaN propagates, as it does in
// [Sum].
func MaskedSum(xs []float64, mask []uint8) float64 {
	n := min(len(xs), len(mask))
	var total float64
	for i := 0; i < n; i++ {
		if mask[i] != 0 {
			total += xs[i]
		}
	}
	return total
}

// MaskedMean returns the mean of the xs elements selected by mask.
//
// It returns 0 when nothing is selected, matching [Mean]'s empty-input contract.
func MaskedMean(xs []float64, mask []uint8) float64 {
	n := min(len(xs), len(mask))
	var total float64
	var count int
	for i := 0; i < n; i++ {
		if mask[i] != 0 {
			total += xs[i]
			count++
		}
	}
	if count == 0 {
		return 0
	}
	return total / float64(count)
}

// MaskedMin returns the smallest selected element of xs.
//
// It returns +Inf when nothing is selected, matching [Min]'s empty-input
// contract, and NaN if any selected element is NaN, matching the "NaN wins"
// scan policy.
func MaskedMin(xs []float64, mask []uint8) float64 {
	n := min(len(xs), len(mask))
	lo := math.Inf(1)
	nan := false
	for i := 0; i < n; i++ {
		if mask[i] != 0 {
			v := xs[i]
			nan = nan || v != v
			if v < lo {
				lo = v
			}
		}
	}
	if nan {
		return math.NaN()
	}
	return lo
}

// MaskedMax returns the largest selected element of xs.
//
// It returns -Inf when nothing is selected, matching [Max]'s empty-input
// contract, and NaN if any selected element is NaN, matching the "NaN wins"
// scan policy.
func MaskedMax(xs []float64, mask []uint8) float64 {
	n := min(len(xs), len(mask))
	hi := math.Inf(-1)
	nan := false
	for i := 0; i < n; i++ {
		if mask[i] != 0 {
			v := xs[i]
			nan = nan || v != v
			if v > hi {
				hi = v
			}
		}
	}
	if nan {
		return math.NaN()
	}
	return hi
}
