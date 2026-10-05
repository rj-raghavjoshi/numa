package numa

import "github.com/rj-raghavjoshi/numa/vec"

// ---------------------------------------------------------------------------
// Facade forwarders for the elementwise math kernels in vec/math.go.
//
// Every function here is a one-line forwarder; there is no behaviour in this
// package that is not defined in vec. The forwarders are grouped into per-family
// files (this one, and later facade_*.go for other families) rather than all
// living in numa.go, because the facade now re-exports a few hundred names and a
// single file stopped being readable. numa.go still holds the package
// documentation, Version, and the original core forwarders.
//
// Forwarder tests live in facade_math_test.go. They exist because a forwarder
// can be wrong in a way the implementation's own tests cannot see: swapped
// arguments, a dropped length check, a lost NaN policy.
// ---------------------------------------------------------------------------

// Neg returns the elementwise negation of xs. See [vec.Neg].
func Neg(xs []float64) []float64 { return vec.Neg(xs) }

// NegTo stores the elementwise negation of xs into dst. See [vec.NegTo].
func NegTo(dst, xs []float64) []float64 { return vec.NegTo(dst, xs) }

// Recip returns the elementwise reciprocal of xs. See [vec.Recip].
func Recip(xs []float64) []float64 { return vec.Recip(xs) }

// RecipTo stores the elementwise reciprocal of xs into dst. See [vec.RecipTo].
func RecipTo(dst, xs []float64) []float64 { return vec.RecipTo(dst, xs) }

// Sign returns the elementwise sign of xs. See [vec.Sign].
func Sign(xs []float64) []float64 { return vec.Sign(xs) }

// SignTo stores the elementwise sign of xs into dst. See [vec.SignTo].
func SignTo(dst, xs []float64) []float64 { return vec.SignTo(dst, xs) }

// Floor returns the elementwise floor of xs. See [vec.Floor].
func Floor(xs []float64) []float64 { return vec.Floor(xs) }

// FloorTo stores the elementwise floor of xs into dst. See [vec.FloorTo].
func FloorTo(dst, xs []float64) []float64 { return vec.FloorTo(dst, xs) }

// Ceil returns the elementwise ceiling of xs. See [vec.Ceil].
func Ceil(xs []float64) []float64 { return vec.Ceil(xs) }

// CeilTo stores the elementwise ceiling of xs into dst. See [vec.CeilTo].
func CeilTo(dst, xs []float64) []float64 { return vec.CeilTo(dst, xs) }

// Trunc returns the elementwise truncation toward zero of xs. See [vec.Trunc].
func Trunc(xs []float64) []float64 { return vec.Trunc(xs) }

// TruncTo stores the elementwise truncation toward zero of xs into dst.
// See [vec.TruncTo].
func TruncTo(dst, xs []float64) []float64 { return vec.TruncTo(dst, xs) }

// Round returns the elementwise rounding of xs, halves away from zero.
// See [vec.Round].
func Round(xs []float64) []float64 { return vec.Round(xs) }

// RoundTo stores the elementwise rounding of xs into dst. See [vec.RoundTo].
func RoundTo(dst, xs []float64) []float64 { return vec.RoundTo(dst, xs) }

// Sqrt returns the elementwise square root of xs. See [vec.Sqrt].
func Sqrt(xs []float64) []float64 { return vec.Sqrt(xs) }

// SqrtTo stores the elementwise square root of xs into dst. See [vec.SqrtTo].
func SqrtTo(dst, xs []float64) []float64 { return vec.SqrtTo(dst, xs) }

// Exp returns the elementwise exponential of xs. See [vec.Exp].
func Exp(xs []float64) []float64 { return vec.Exp(xs) }

// ExpTo stores the elementwise exponential of xs into dst. See [vec.ExpTo].
func ExpTo(dst, xs []float64) []float64 { return vec.ExpTo(dst, xs) }

// ExpM1 returns exp(xs[i])-1 elementwise. See [vec.ExpM1].
func ExpM1(xs []float64) []float64 { return vec.ExpM1(xs) }

// ExpM1To stores exp(xs[i])-1 into dst. See [vec.ExpM1To].
func ExpM1To(dst, xs []float64) []float64 { return vec.ExpM1To(dst, xs) }

// Log returns the elementwise natural logarithm of xs. See [vec.Log].
func Log(xs []float64) []float64 { return vec.Log(xs) }

// LogTo stores the elementwise natural logarithm of xs into dst. See [vec.LogTo].
func LogTo(dst, xs []float64) []float64 { return vec.LogTo(dst, xs) }

// Log1p returns log(1+xs[i]) elementwise. See [vec.Log1p].
func Log1p(xs []float64) []float64 { return vec.Log1p(xs) }

// Log1pTo stores log(1+xs[i]) into dst. See [vec.Log1pTo].
func Log1pTo(dst, xs []float64) []float64 { return vec.Log1pTo(dst, xs) }

// Log2 returns the elementwise base-2 logarithm of xs. See [vec.Log2].
func Log2(xs []float64) []float64 { return vec.Log2(xs) }

// Log2To stores the elementwise base-2 logarithm of xs into dst. See [vec.Log2To].
func Log2To(dst, xs []float64) []float64 { return vec.Log2To(dst, xs) }

// Log10 returns the elementwise base-10 logarithm of xs. See [vec.Log10].
func Log10(xs []float64) []float64 { return vec.Log10(xs) }

// Log10To stores the elementwise base-10 logarithm of xs into dst.
// See [vec.Log10To].
func Log10To(dst, xs []float64) []float64 { return vec.Log10To(dst, xs) }

// Div returns the elementwise quotient xs[i]/ys[i]. See [vec.Div].
func Div(xs, ys []float64) []float64 { return vec.Div(xs, ys) }

// DivTo stores the elementwise quotient xs[i]/ys[i] into dst. See [vec.DivTo].
func DivTo(dst, xs, ys []float64) []float64 { return vec.DivTo(dst, xs, ys) }

// SafeDiv returns the elementwise quotient with a zero denominator mapped to 0.
// See [vec.SafeDiv].
func SafeDiv(xs, ys []float64) []float64 { return vec.SafeDiv(xs, ys) }

// SafeDivTo stores the safe elementwise quotient into dst. See [vec.SafeDivTo].
func SafeDivTo(dst, xs, ys []float64) []float64 { return vec.SafeDivTo(dst, xs, ys) }

// Pow returns xs[i] raised to the scalar exponent k. See [vec.Pow].
func Pow(xs []float64, k float64) []float64 { return vec.Pow(xs, k) }

// PowTo stores xs[i] raised to k into dst. See [vec.PowTo].
func PowTo(dst, xs []float64, k float64) []float64 { return vec.PowTo(dst, xs, k) }

// Min2 returns the elementwise minimum of xs and ys, propagating NaN.
// See [vec.Min2].
func Min2(xs, ys []float64) []float64 { return vec.Min2(xs, ys) }

// Min2To stores the elementwise minimum of xs and ys into dst. See [vec.Min2To].
func Min2To(dst, xs, ys []float64) []float64 { return vec.Min2To(dst, xs, ys) }

// Max2 returns the elementwise maximum of xs and ys, propagating NaN.
// See [vec.Max2].
func Max2(xs, ys []float64) []float64 { return vec.Max2(xs, ys) }

// Max2To stores the elementwise maximum of xs and ys into dst. See [vec.Max2To].
func Max2To(dst, xs, ys []float64) []float64 { return vec.Max2To(dst, xs, ys) }

// Clamp returns xs confined to [lo, hi]. See [vec.Clamp].
func Clamp(xs []float64, lo, hi float64) []float64 { return vec.Clamp(xs, lo, hi) }

// ClampTo stores xs confined to [lo, hi] into dst. See [vec.ClampTo].
func ClampTo(dst, xs []float64, lo, hi float64) []float64 { return vec.ClampTo(dst, xs, lo, hi) }

// Lerp returns the elementwise linear interpolation xs + t*(ys-xs).
// See [vec.Lerp].
func Lerp(xs, ys []float64, t float64) []float64 { return vec.Lerp(xs, ys, t) }

// LerpTo stores xs + t*(ys-xs) into dst. See [vec.LerpTo].
func LerpTo(dst, xs, ys []float64, t float64) []float64 { return vec.LerpTo(dst, xs, ys, t) }

// MulAdd returns the elementwise xs[i]*ys[i] + zs[i]. See [vec.MulAdd].
func MulAdd(xs, ys, zs []float64) []float64 { return vec.MulAdd(xs, ys, zs) }

// MulAddTo stores xs[i]*ys[i] + zs[i] into dst. See [vec.MulAddTo].
func MulAddTo(dst, xs, ys, zs []float64) []float64 { return vec.MulAddTo(dst, xs, ys, zs) }

// FMA returns the elementwise fused multiply-add xs[i]*ys[i] + zs[i], computed
// with one rounding. See [vec.FMA].
func FMA(xs, ys, zs []float64) []float64 { return vec.FMA(xs, ys, zs) }

// FMATo stores the fused xs[i]*ys[i] + zs[i] into dst. See [vec.FMATo].
func FMATo(dst, xs, ys, zs []float64) []float64 { return vec.FMATo(dst, xs, ys, zs) }

// Copy returns a new slice holding a copy of xs. See [vec.Copy].
func Copy(xs []float64) []float64 { return vec.Copy(xs) }

// CopyTo copies xs into dst. See [vec.CopyTo].
func CopyTo(dst, xs []float64) []float64 { return vec.CopyTo(dst, xs) }

// Fill returns a new slice of length n with every element set to v. See [vec.Fill].
func Fill(v float64, n int) []float64 { return vec.Fill(v, n) }

// FillTo sets every element of dst to v. See [vec.FillTo].
func FillTo(dst []float64, v float64) []float64 { return vec.FillTo(dst, v) }

// Reverse returns a new slice holding xs in reverse order. See [vec.Reverse].
func Reverse(xs []float64) []float64 { return vec.Reverse(xs) }

// ReverseTo stores xs in reverse order into dst; dst must not alias xs.
// See [vec.ReverseTo].
func ReverseTo(dst, xs []float64) []float64 { return vec.ReverseTo(dst, xs) }

// ReverseInPlace reverses xs into itself. See [vec.ReverseInPlace].
func ReverseInPlace(xs []float64) []float64 { return vec.ReverseInPlace(xs) }
