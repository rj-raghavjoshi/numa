package numa

import "github.com/rj-raghavjoshi/numa/vec"

// SubScalar returns xs[i] - k. See [vec.SubScalar].
func SubScalar(xs []float64, k float64) []float64 { return vec.SubScalar(xs, k) }

// SubScalarTo stores xs[i] - k into dst. See [vec.SubScalarTo].
func SubScalarTo(dst, xs []float64, k float64) []float64 { return vec.SubScalarTo(dst, xs, k) }

// RSubScalar returns k - xs[i]. See [vec.RSubScalar].
func RSubScalar(xs []float64, k float64) []float64 { return vec.RSubScalar(xs, k) }

// RSubScalarTo stores k - xs[i] into dst. See [vec.RSubScalarTo].
func RSubScalarTo(dst, xs []float64, k float64) []float64 {
	return vec.RSubScalarTo(dst, xs, k)
}

// DivScalar returns xs[i] / k. See [vec.DivScalar].
func DivScalar(xs []float64, k float64) []float64 { return vec.DivScalar(xs, k) }

// DivScalarTo stores xs[i] / k into dst. See [vec.DivScalarTo].
func DivScalarTo(dst, xs []float64, k float64) []float64 { return vec.DivScalarTo(dst, xs, k) }

// RDivScalar returns k / xs[i]. See [vec.RDivScalar].
func RDivScalar(xs []float64, k float64) []float64 { return vec.RDivScalar(xs, k) }

// RDivScalarTo stores k / xs[i] into dst. See [vec.RDivScalarTo].
func RDivScalarTo(dst, xs []float64, k float64) []float64 {
	return vec.RDivScalarTo(dst, xs, k)
}
