// Package numa is a high-throughput, hardware-aware numerical engine for
// quantitative modeling, time-series analysis, and algorithmic trading.
//
// # Contents
//
// This package is a thin facade over [github.com/rj-raghavjoshi/numa/vec], which
// holds the implementations. Every function here is a one-line forwarder, so
// there is no behaviour defined in this package that is not defined there.
//
// Using this package gives you the shorter call:
//
//	numa.Sum(xs)
//
// Using the implementation package directly avoids one indirection:
//
//	vec.Sum(xs)
//
// The indirection is a plain function call, and Go will inline most of these
// forwarders, but "most" is not "all". If you are calling these in a hot loop and
// want a guarantee rather than a hope, import vec directly and measure.
//
// # Documentation
//
// The full explanation of why the tuned loops are shaped the way they are —
// starting from what a CPU actually is — is in the tutorial under docs/learn/.
// The measured results are in docs/benchmarks.md, and the open decisions and
// unverified assumptions are in docs/next-steps.md. The design reference for
// modifying the code is docs/design.md.
//
// # Numerics
//
// All reductions are reassociated: results may differ in the last few bits from a
// plain left-to-right loop, and are not guaranteed to be bit-identical across
// architectures. See the vec package documentation for the full discussion.
//
// # Version
package numa

import (
	"github.com/rj-raghavjoshi/numa/vec"
)

// Version is the current semantic release of the numa engine.
const Version = "0.1.0"

// Sum returns the arithmetic sum of xs. See [vec.Sum].
func Sum(xs []float64) float64 { return vec.Sum(xs) }

// Mean returns the arithmetic mean of xs. See [vec.Mean].
func Mean(xs []float64) float64 { return vec.Mean(xs) }

// Dot returns the inner product of xs and ys. See [vec.Dot].
func Dot(xs, ys []float64) float64 { return vec.Dot(xs, ys) }

// SumSq returns the sum of squares of xs. See [vec.SumSq].
func SumSq(xs []float64) float64 { return vec.SumSq(xs) }

// Min returns the smallest element of xs, or +Inf if xs is empty.
// See [vec.Min].
func Min(xs []float64) float64 { return vec.Min(xs) }

// Max returns the largest element of xs, or -Inf if xs is empty.
// See [vec.Max].
func Max(xs []float64) float64 { return vec.Max(xs) }

// MinMax returns the smallest and largest elements of xs in a single pass.
// See [vec.MinMax].
func MinMax(xs []float64) (lo, hi float64) { return vec.MinMax(xs) }

// Add returns the elementwise sum of xs and ys. See [vec.Add].
func Add(xs, ys []float64) []float64 { return vec.Add(xs, ys) }

// Sub returns the elementwise difference xs-ys. See [vec.Sub].
func Sub(xs, ys []float64) []float64 { return vec.Sub(xs, ys) }

// Mul returns the elementwise product of xs and ys. See [vec.Mul].
func Mul(xs, ys []float64) []float64 { return vec.Mul(xs, ys) }

// Scale returns xs multiplied by the scalar k. See [vec.Scale].
func Scale(xs []float64, k float64) []float64 { return vec.Scale(xs, k) }

// AddScalar returns xs plus the scalar k. See [vec.AddScalar].
func AddScalar(xs []float64, k float64) []float64 { return vec.AddScalar(xs, k) }

// Abs returns the absolute value of each element of xs. See [vec.Abs].
func Abs(xs []float64) []float64 { return vec.Abs(xs) }

// AddTo stores the elementwise sum of xs and ys into dst. Panics if the lengths
// differ. See [vec.AddTo].
func AddTo(dst, xs, ys []float64) []float64 { return vec.AddTo(dst, xs, ys) }

// SubTo stores the elementwise difference xs-ys into dst. Panics if the lengths
// differ. See [vec.SubTo].
func SubTo(dst, xs, ys []float64) []float64 { return vec.SubTo(dst, xs, ys) }

// MulTo stores the elementwise product of xs and ys into dst. Panics if the
// lengths differ. See [vec.MulTo].
func MulTo(dst, xs, ys []float64) []float64 { return vec.MulTo(dst, xs, ys) }

// ScaleTo stores xs multiplied by k into dst. Panics if the lengths differ.
// See [vec.ScaleTo].
func ScaleTo(dst, xs []float64, k float64) []float64 { return vec.ScaleTo(dst, xs, k) }

// AddScalarTo stores xs plus k into dst. Panics if the lengths differ.
// See [vec.AddScalarTo].
func AddScalarTo(dst, xs []float64, k float64) []float64 {
	return vec.AddScalarTo(dst, xs, k)
}

// AbsTo stores the absolute value of each element of xs into dst. Panics if the
// lengths differ. See [vec.AbsTo].
func AbsTo(dst, xs []float64) []float64 { return vec.AbsTo(dst, xs) }
