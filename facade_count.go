package numa

import "github.com/rj-raghavjoshi/numa/vec"

// CountGreaterScalar returns the number of elements strictly greater than k.
// See [vec.CountGreaterScalar].
func CountGreaterScalar(xs []float64, k float64) int { return vec.CountGreaterScalar(xs, k) }

// CountGreaterEqualScalar returns the number of elements greater than or equal
// to k. See [vec.CountGreaterEqualScalar].
func CountGreaterEqualScalar(xs []float64, k float64) int {
	return vec.CountGreaterEqualScalar(xs, k)
}
