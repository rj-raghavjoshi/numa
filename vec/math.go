package vec

import "math"

// ---------------------------------------------------------------------------
// Elementwise math kernels: maps with one output per input.
//
// # Why these have no arch variant
//
// The rest of this package selects a tuned loop per architecture, because those
// operations carry a loop-carried dependency chain whose length depends on how
// many independent accumulators the register file can hold. A map has no such
// chain: dst[i] depends only on xs[i], so there is nothing for an accumulator to
// hide and nothing for a build tag to choose between. The wide unrolled loop
// below *is* the tuned loop, and it is identical on every architecture.
//
// Putting three byte-identical copies behind build tags would be duplication
// that merely looks like tuning. It is not, so there is one copy here. The
// exception is recorded in ../docs/design.md.
//
// # Cheap versus transcendental maps
//
// Operations whose body is a single arithmetic or rounding instruction (Neg,
// Sign, Floor, Ceil, Round, Trunc, Recip, Div, ...) are unrolled four wide, so
// the load/store units stay saturated. Operations that call into the math
// library (Sqrt, Log, Exp, ...) are dominated by that call, which costs an order
// of magnitude more than the loop overhead; unrolling them buys nothing
// measurable, so they use the plain range loop. Both are documented per family.
//
// # Conventions
//
//   - Allocating forms return nil for an empty input.
//   - *To forms write into a caller-supplied destination and panic on a length
//     mismatch, consistent with the rest of the package.
//   - Binary *To forms permit dst to alias xs or ys. ReverseTo is the one
//     documented exception; use ReverseInPlace for in-place reversal.
//   - NaN propagates through every operation here, because each is a plain
//     IEEE-754 instruction or a correctly-rounded library call.
// ---------------------------------------------------------------------------

// Neg returns the elementwise negation of xs.
func Neg(xs []float64) []float64 {
	if len(xs) == 0 {
		return nil
	}
	return NegTo(make([]float64, len(xs)), xs)
}

// NegTo stores the elementwise negation of xs into dst and returns dst.
//
// dst and xs must be the same length. NegTo panics if they are not. dst may
// alias xs.
func NegTo(dst, xs []float64) []float64 {
	checkSameLen1("NegTo", dst, xs)
	n := len(dst)
	i := 0
	limit := n - 3
	for i < limit {
		dst[i] = -xs[i]
		dst[i+1] = -xs[i+1]
		dst[i+2] = -xs[i+2]
		dst[i+3] = -xs[i+3]
		i += 4
	}
	for ; i < n; i++ {
		dst[i] = -xs[i]
	}
	return dst
}

// Recip returns the elementwise reciprocal of xs, 1/xs[i].
//
// Recip(0) is +Inf and Recip(-0) is -Inf, following IEEE-754. Use [SafeDiv]
// with a 1.0 numerator if a zero denominator should produce zero instead.
func Recip(xs []float64) []float64 {
	if len(xs) == 0 {
		return nil
	}
	return RecipTo(make([]float64, len(xs)), xs)
}

// RecipTo stores the elementwise reciprocal of xs into dst and returns dst.
//
// dst and xs must be the same length. RecipTo panics if they are not.
func RecipTo(dst, xs []float64) []float64 {
	checkSameLen1("RecipTo", dst, xs)
	n := len(dst)
	i := 0
	limit := n - 3
	for i < limit {
		dst[i] = 1 / xs[i]
		dst[i+1] = 1 / xs[i+1]
		dst[i+2] = 1 / xs[i+2]
		dst[i+3] = 1 / xs[i+3]
		i += 4
	}
	for ; i < n; i++ {
		dst[i] = 1 / xs[i]
	}
	return dst
}

// Sign returns the elementwise sign of xs: -1, 0 or +1.
//
// The sign of a zero is preserved: Sign(-0) is -0 and Sign(+0) is +0, because
// both are integer-valued and the distinction survives downstream divides.
// Sign of NaN is NaN.
func Sign(xs []float64) []float64 {
	if len(xs) == 0 {
		return nil
	}
	return SignTo(make([]float64, len(xs)), xs)
}

// SignTo stores the elementwise sign of xs into dst and returns dst.
//
// dst and xs must be the same length. SignTo panics if they are not.
func SignTo(dst, xs []float64) []float64 {
	checkSameLen1("SignTo", dst, xs)
	for i, v := range xs {
		switch {
		case v > 0:
			dst[i] = 1
		case v < 0:
			dst[i] = -1
		case v == 0:
			dst[i] = v // preserves the sign of zero
		default:
			dst[i] = math.NaN()
		}
	}
	return dst
}

// Floor returns the elementwise floor of xs.
func Floor(xs []float64) []float64 {
	if len(xs) == 0 {
		return nil
	}
	return FloorTo(make([]float64, len(xs)), xs)
}

// FloorTo stores the elementwise floor of xs into dst and returns dst.
//
// dst and xs must be the same length. FloorTo panics if they are not.
func FloorTo(dst, xs []float64) []float64 {
	checkSameLen1("FloorTo", dst, xs)
	n := len(dst)
	i := 0
	limit := n - 3
	for i < limit {
		dst[i] = math.Floor(xs[i])
		dst[i+1] = math.Floor(xs[i+1])
		dst[i+2] = math.Floor(xs[i+2])
		dst[i+3] = math.Floor(xs[i+3])
		i += 4
	}
	for ; i < n; i++ {
		dst[i] = math.Floor(xs[i])
	}
	return dst
}

// Ceil returns the elementwise ceiling of xs.
func Ceil(xs []float64) []float64 {
	if len(xs) == 0 {
		return nil
	}
	return CeilTo(make([]float64, len(xs)), xs)
}

// CeilTo stores the elementwise ceiling of xs into dst and returns dst.
//
// dst and xs must be the same length. CeilTo panics if they are not.
func CeilTo(dst, xs []float64) []float64 {
	checkSameLen1("CeilTo", dst, xs)
	n := len(dst)
	i := 0
	limit := n - 3
	for i < limit {
		dst[i] = math.Ceil(xs[i])
		dst[i+1] = math.Ceil(xs[i+1])
		dst[i+2] = math.Ceil(xs[i+2])
		dst[i+3] = math.Ceil(xs[i+3])
		i += 4
	}
	for ; i < n; i++ {
		dst[i] = math.Ceil(xs[i])
	}
	return dst
}

// Trunc returns the elementwise truncation toward zero of xs.
func Trunc(xs []float64) []float64 {
	if len(xs) == 0 {
		return nil
	}
	return TruncTo(make([]float64, len(xs)), xs)
}

// TruncTo stores the elementwise truncation toward zero of xs into dst and
// returns dst. dst and xs must be the same length. TruncTo panics if they are
// not.
func TruncTo(dst, xs []float64) []float64 {
	checkSameLen1("TruncTo", dst, xs)
	n := len(dst)
	i := 0
	limit := n - 3
	for i < limit {
		dst[i] = math.Trunc(xs[i])
		dst[i+1] = math.Trunc(xs[i+1])
		dst[i+2] = math.Trunc(xs[i+2])
		dst[i+3] = math.Trunc(xs[i+3])
		i += 4
	}
	for ; i < n; i++ {
		dst[i] = math.Trunc(xs[i])
	}
	return dst
}

// Round returns the elementwise rounding of xs, halves away from zero.
//
// Go's math.Round and Pine's math.round both round halves away from zero, so the
// two agree; banker's rounding would differ on exactly .5 and is not provided.
func Round(xs []float64) []float64 {
	if len(xs) == 0 {
		return nil
	}
	return RoundTo(make([]float64, len(xs)), xs)
}

// RoundTo stores the elementwise rounding of xs into dst and returns dst.
//
// dst and xs must be the same length. RoundTo panics if they are not.
func RoundTo(dst, xs []float64) []float64 {
	checkSameLen1("RoundTo", dst, xs)
	n := len(dst)
	i := 0
	limit := n - 3
	for i < limit {
		dst[i] = math.Round(xs[i])
		dst[i+1] = math.Round(xs[i+1])
		dst[i+2] = math.Round(xs[i+2])
		dst[i+3] = math.Round(xs[i+3])
		i += 4
	}
	for ; i < n; i++ {
		dst[i] = math.Round(xs[i])
	}
	return dst
}

// Sqrt returns the elementwise square root of xs.
//
// The result is NaN for negative inputs, by IEEE-754.
func Sqrt(xs []float64) []float64 {
	if len(xs) == 0 {
		return nil
	}
	return SqrtTo(make([]float64, len(xs)), xs)
}

// SqrtTo stores the elementwise square root of xs into dst and returns dst.
//
// dst and xs must be the same length. SqrtTo panics if they are not.
func SqrtTo(dst, xs []float64) []float64 {
	checkSameLen1("SqrtTo", dst, xs)
	for i, v := range xs {
		dst[i] = math.Sqrt(v)
	}
	return dst
}

// Exp returns the elementwise exponential of xs.
func Exp(xs []float64) []float64 {
	if len(xs) == 0 {
		return nil
	}
	return ExpTo(make([]float64, len(xs)), xs)
}

// ExpTo stores the elementwise exponential of xs into dst and returns dst.
//
// dst and xs must be the same length. ExpTo panics if they are not.
func ExpTo(dst, xs []float64) []float64 {
	checkSameLen1("ExpTo", dst, xs)
	for i, v := range xs {
		dst[i] = math.Exp(v)
	}
	return dst
}

// ExpM1 returns exp(xs[i])-1 elementwise, accurate for small inputs.
func ExpM1(xs []float64) []float64 {
	if len(xs) == 0 {
		return nil
	}
	return ExpM1To(make([]float64, len(xs)), xs)
}

// ExpM1To stores exp(xs[i])-1 into dst and returns dst.
//
// dst and xs must be the same length. ExpM1To panics if they are not.
func ExpM1To(dst, xs []float64) []float64 {
	checkSameLen1("ExpM1To", dst, xs)
	for i, v := range xs {
		dst[i] = math.Expm1(v)
	}
	return dst
}

// Log returns the elementwise natural logarithm of xs.
//
// The result is -Inf at 0 and NaN for negative inputs, by IEEE-754.
func Log(xs []float64) []float64 {
	if len(xs) == 0 {
		return nil
	}
	return LogTo(make([]float64, len(xs)), xs)
}

// LogTo stores the elementwise natural logarithm of xs into dst and returns dst.
//
// dst and xs must be the same length. LogTo panics if they are not.
func LogTo(dst, xs []float64) []float64 {
	checkSameLen1("LogTo", dst, xs)
	for i, v := range xs {
		dst[i] = math.Log(v)
	}
	return dst
}

// Log1p returns log(1+xs[i]) elementwise, accurate for small inputs.
func Log1p(xs []float64) []float64 {
	if len(xs) == 0 {
		return nil
	}
	return Log1pTo(make([]float64, len(xs)), xs)
}

// Log1pTo stores log(1+xs[i]) into dst and returns dst.
//
// dst and xs must be the same length. Log1pTo panics if they are not.
func Log1pTo(dst, xs []float64) []float64 {
	checkSameLen1("Log1pTo", dst, xs)
	for i, v := range xs {
		dst[i] = math.Log1p(v)
	}
	return dst
}

// Log2 returns the elementwise base-2 logarithm of xs.
func Log2(xs []float64) []float64 {
	if len(xs) == 0 {
		return nil
	}
	return Log2To(make([]float64, len(xs)), xs)
}

// Log2To stores the elementwise base-2 logarithm of xs into dst and returns dst.
//
// dst and xs must be the same length. Log2To panics if they are not.
func Log2To(dst, xs []float64) []float64 {
	checkSameLen1("Log2To", dst, xs)
	for i, v := range xs {
		dst[i] = math.Log2(v)
	}
	return dst
}

// Log10 returns the elementwise base-10 logarithm of xs.
func Log10(xs []float64) []float64 {
	if len(xs) == 0 {
		return nil
	}
	return Log10To(make([]float64, len(xs)), xs)
}

// Log10To stores the elementwise base-10 logarithm of xs into dst and returns
// dst. dst and xs must be the same length. Log10To panics if they are not.
func Log10To(dst, xs []float64) []float64 {
	checkSameLen1("Log10To", dst, xs)
	for i, v := range xs {
		dst[i] = math.Log10(v)
	}
	return dst
}

// ---------------------------------------------------------------------------
// Binary maps.
// ---------------------------------------------------------------------------

// Div returns the elementwise quotient xs[i]/ys[i] as a new slice of length
// min(len(xs), len(ys)).
//
// Division by zero yields ±Inf or NaN following IEEE-754. Use [SafeDiv] when a
// zero denominator should produce zero instead.
//
// Div allocates; for hot loops that can supply a buffer, use [DivTo].
func Div(xs, ys []float64) []float64 {
	n := min(len(xs), len(ys))
	if n == 0 {
		return nil
	}
	return DivTo(make([]float64, n), xs[:n], ys[:n])
}

// DivTo stores the elementwise quotient xs[i]/ys[i] into dst and returns dst.
//
// The three slices must be the same length. DivTo panics if they are not. dst
// may alias xs or ys.
func DivTo(dst, xs, ys []float64) []float64 {
	checkSameLen("DivTo", dst, xs, ys)
	n := len(dst)
	i := 0
	limit := n - 3
	for i < limit {
		dst[i] = xs[i] / ys[i]
		dst[i+1] = xs[i+1] / ys[i+1]
		dst[i+2] = xs[i+2] / ys[i+2]
		dst[i+3] = xs[i+3] / ys[i+3]
		i += 4
	}
	for ; i < n; i++ {
		dst[i] = xs[i] / ys[i]
	}
	return dst
}

// SafeDiv returns the elementwise quotient xs[i]/ys[i], except that a zero
// denominator produces 0 rather than ±Inf or NaN.
//
// This is the divisor convention used by the indicator layer, where a flat
// window or an absent volume routinely makes the denominator zero and an
// infinite value would poison the whole series. A NaN denominator is not zero,
// so it propagates rather than being masked.
//
// SafeDiv allocates; for hot loops that can supply a buffer, use [SafeDivTo].
func SafeDiv(xs, ys []float64) []float64 {
	n := min(len(xs), len(ys))
	if n == 0 {
		return nil
	}
	return SafeDivTo(make([]float64, n), xs[:n], ys[:n])
}

// SafeDivTo stores the safe elementwise quotient into dst and returns dst.
//
// The three slices must be the same length. SafeDivTo panics if they are not.
// dst may alias xs or ys.
func SafeDivTo(dst, xs, ys []float64) []float64 {
	checkSameLen("SafeDivTo", dst, xs, ys)
	n := len(dst)
	for i := 0; i < n; i++ {
		if ys[i] == 0 {
			dst[i] = 0
		} else {
			dst[i] = xs[i] / ys[i]
		}
	}
	return dst
}

// Pow returns xs[i] raised to the scalar exponent k, elementwise.
//
// Pow allocates; for hot loops that can supply a buffer, use [PowTo].
func Pow(xs []float64, k float64) []float64 {
	if len(xs) == 0 {
		return nil
	}
	return PowTo(make([]float64, len(xs)), xs, k)
}

// PowTo stores xs[i] raised to k into dst and returns dst.
//
// dst and xs must be the same length. PowTo panics if they are not.
func PowTo(dst, xs []float64, k float64) []float64 {
	checkSameLen1("PowTo", dst, xs)
	for i, v := range xs {
		dst[i] = math.Pow(v, k)
	}
	return dst
}

// Min2 returns the elementwise minimum of xs and ys.
//
// If either input is NaN the result is NaN, matching the package's "NaN wins"
// scan policy rather than a silent skip. Note that the standard library's
// math.Min returns the non-NaN operand instead; this function deliberately does
// not, so that a NaN anywhere in the input remains visible.
//
// Min2 allocates; for hot loops that can supply a buffer, use [Min2To].
func Min2(xs, ys []float64) []float64 {
	n := min(len(xs), len(ys))
	if n == 0 {
		return nil
	}
	return Min2To(make([]float64, n), xs[:n], ys[:n])
}

// Min2To stores the elementwise minimum of xs and ys into dst and returns dst.
//
// The three slices must be the same length. Min2To panics if they are not. dst
// may alias xs or ys.
func Min2To(dst, xs, ys []float64) []float64 {
	checkSameLen("Min2To", dst, xs, ys)
	for i := range dst {
		a, b := xs[i], ys[i]
		// Written as a comparison rather than math.Min so that NaN propagates:
		// every comparison against NaN is false, so a NaN in either operand
		// leaves the NaN result in place.
		if a < b {
			dst[i] = a
		} else if b < a {
			dst[i] = b
		} else if a != a {
			dst[i] = a
		} else if b != b {
			dst[i] = b
		} else {
			dst[i] = a
		}
	}
	return dst
}

// Max2 returns the elementwise maximum of xs and ys.
//
// NaN propagates; see [Min2] for why this differs from math.Max.
//
// Max2 allocates; for hot loops that can supply a buffer, use [Max2To].
func Max2(xs, ys []float64) []float64 {
	n := min(len(xs), len(ys))
	if n == 0 {
		return nil
	}
	return Max2To(make([]float64, n), xs[:n], ys[:n])
}

// Max2To stores the elementwise maximum of xs and ys into dst and returns dst.
//
// The three slices must be the same length. Max2To panics if they are not. dst
// may alias xs or ys.
func Max2To(dst, xs, ys []float64) []float64 {
	checkSameLen("Max2To", dst, xs, ys)
	for i := range dst {
		a, b := xs[i], ys[i]
		if a > b {
			dst[i] = a
		} else if b > a {
			dst[i] = b
		} else if a != a {
			dst[i] = a
		} else if b != b {
			dst[i] = b
		} else {
			dst[i] = a
		}
	}
	return dst
}

// ---------------------------------------------------------------------------
// Three-input maps.
// ---------------------------------------------------------------------------

// Clamp returns xs with every element confined to [lo, hi].
//
// The bound is applied as min(max(x, lo), hi), so when lo > hi the result is hi
// for every element. Callers should pass lo <= hi; the behaviour otherwise is
// defined but almost certainly not what was intended. NaN propagates.
//
// Clamp allocates; for hot loops that can supply a buffer, use [ClampTo].
func Clamp(xs []float64, lo, hi float64) []float64 {
	if len(xs) == 0 {
		return nil
	}
	return ClampTo(make([]float64, len(xs)), xs, lo, hi)
}

// ClampTo stores xs confined to [lo, hi] into dst and returns dst.
//
// dst and xs must be the same length. ClampTo panics if they are not.
func ClampTo(dst, xs []float64, lo, hi float64) []float64 {
	checkSameLen1("ClampTo", dst, xs)
	for i, v := range xs {
		if v < lo {
			v = lo
		}
		if v > hi {
			v = hi
		}
		dst[i] = v
	}
	return dst
}

// Lerp returns the elementwise linear interpolation xs + t*(ys-xs).
//
// t is not clamped: t < 0 and t > 1 extrapolate, which is what a caller asking
// for a weighted blend between two series usually wants. Clamp t first if not.
// NaN propagates from t as well as from the inputs.
//
// Lerp allocates; for hot loops that can supply a buffer, use [LerpTo].
func Lerp(xs, ys []float64, t float64) []float64 {
	n := min(len(xs), len(ys))
	if n == 0 {
		return nil
	}
	return LerpTo(make([]float64, n), xs[:n], ys[:n], t)
}

// LerpTo stores xs + t*(ys-xs) into dst and returns dst.
//
// The three slices must be the same length. LerpTo panics if they are not. dst
// may alias xs or ys.
func LerpTo(dst, xs, ys []float64, t float64) []float64 {
	checkSameLen("LerpTo", dst, xs, ys)
	for i := range dst {
		a, b := xs[i], ys[i]
		dst[i] = a + t*(b-a)
	}
	return dst
}

// MulAdd returns the elementwise xs[i]*ys[i] + zs[i].
//
// The source is a plain multiply followed by a plain add, but the compiler is
// free to contract the pair into one hardware fused multiply-add, and on arm64 it
// does: the disassembled loop contains FMADD. So whether a given element was
// rounded once or twice depends on the architecture and the GOAMD64 level --
// the same cross-architecture caveat the reductions already carry.
//
// Use [FMA] instead when a single rounding must be guaranteed rather than left
// to the compiler's contraction decision.
//
// MulAdd allocates; for hot loops that can supply a buffer, use [MulAddTo].
func MulAdd(xs, ys, zs []float64) []float64 {
	n := min(len(xs), min(len(ys), len(zs)))
	if n == 0 {
		return nil
	}
	return MulAddTo(make([]float64, n), xs[:n], ys[:n], zs[:n])
}

// MulAddTo stores xs[i]*ys[i] + zs[i] into dst and returns dst.
//
// The four slices must be the same length. MulAddTo panics if they are not. dst
// may alias any input.
func MulAddTo(dst, xs, ys, zs []float64) []float64 {
	checkSameLen4("MulAddTo", dst, xs, ys, zs)
	for i := range dst {
		dst[i] = xs[i]*ys[i] + zs[i]
	}
	return dst
}

// FMA returns the elementwise fused multiply-add xs[i]*ys[i] + zs[i], computed
// with one rounding.
//
// This is the guaranteed-fused counterpart of [MulAdd], and the distinction is
// not cosmetic: a fused result is both differently rounded and, in a
// multiply-accumulate reduction, strictly more accurate, because the product's
// low bits are not discarded before the addition.
//
// # Implementation
//
// Calls math.FMA, which is a compiler intrinsic, not a library call. On arm64 it
// lowers to the hardware FMADD instruction; on amd64 it lowers to a VFMADD when
// GOAMD64 >= v3 or when the CPU reports FMA3, and otherwise falls through to the
// correct pure-Go software routine that ships with the standard library. On
// architectures with neither, the software routine is used. So this is fast
// where the hardware exists and correct everywhere, with no CGo and no assembly
// in this repository.
//
// That the intrinsic is real was verified by disassembling this package's own
// Dot, whose loop already contracts to FMADD; see ../docs/benchmarks.md.
//
// FMA allocates; for hot loops that can supply a buffer, use [FMATo].
func FMA(xs, ys, zs []float64) []float64 {
	n := min(len(xs), min(len(ys), len(zs)))
	if n == 0 {
		return nil
	}
	return FMATo(make([]float64, n), xs[:n], ys[:n], zs[:n])
}

// FMATo stores the fused xs[i]*ys[i] + zs[i] into dst and returns dst.
//
// The four slices must be the same length. FMATo panics if they are not. dst may
// alias any input.
func FMATo(dst, xs, ys, zs []float64) []float64 {
	checkSameLen4("FMATo", dst, xs, ys, zs)
	for i := range dst {
		dst[i] = math.FMA(xs[i], ys[i], zs[i])
	}
	return dst
}

// ---------------------------------------------------------------------------
// Whole-array moves.
// ---------------------------------------------------------------------------

// Copy returns a new slice holding a copy of xs.
//
// It returns nil for an empty input, so a copy of an empty slice is nil rather
// than a zero-length non-nil slice. Use [CopyTo] into a pre-sized buffer when
// the distinction matters.
func Copy(xs []float64) []float64 {
	if len(xs) == 0 {
		return nil
	}
	return CopyTo(make([]float64, len(xs)), xs)
}

// CopyTo copies xs into dst and returns dst.
//
// dst and xs must be the same length. CopyTo panics if they are not.
//
// Like the standard library's copy, CopyTo is safe when dst and xs overlap,
// including when they are the same slice.
func CopyTo(dst, xs []float64) []float64 {
	checkSameLen1("CopyTo", dst, xs)
	copy(dst, xs)
	return dst
}

// Fill returns a new slice of length n with every element set to v.
//
// Fill returns nil when n <= 0.
func Fill(v float64, n int) []float64 {
	if n <= 0 {
		return nil
	}
	return FillTo(make([]float64, n), v)
}

// FillTo sets every element of dst to v and returns dst.
//
// FillTo never panics; an empty dst is a no-op.
func FillTo(dst []float64, v float64) []float64 {
	for i := range dst {
		dst[i] = v
	}
	return dst
}

// Reverse returns a new slice holding xs in reverse order.
func Reverse(xs []float64) []float64 {
	if len(xs) == 0 {
		return nil
	}
	return ReverseTo(make([]float64, len(xs)), xs)
}

// ReverseTo stores xs in reverse order into dst and returns dst.
//
// dst and xs must be the same length. ReverseTo panics if they are not.
//
// This is the one *To function in the package that does NOT permit aliasing:
// writing dst[i] overwrites the xs[n-1-i] that a later iteration still needs.
// Use [ReverseInPlace] to reverse a slice into itself.
func ReverseTo(dst, xs []float64) []float64 {
	checkSameLen1("ReverseTo", dst, xs)
	n := len(dst)
	for i := range dst {
		dst[i] = xs[n-1-i]
	}
	return dst
}

// ReverseInPlace reverses xs into itself and returns xs.
func ReverseInPlace(xs []float64) []float64 {
	for i, j := 0, len(xs)-1; i < j; i, j = i+1, j-1 {
		xs[i], xs[j] = xs[j], xs[i]
	}
	return xs
}
