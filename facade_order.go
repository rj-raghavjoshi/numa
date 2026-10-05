package numa

import "github.com/rj-raghavjoshi/numa/vec"

// ---------------------------------------------------------------------------
// Facade forwarders for vec/arg.go and vec/order.go: arg reductions, order
// statistics, ranking, and the NaN probe.
//
// Each is a one-line forwarder to vec. See facade_math.go for why the forwarders
// are split per family, and facade_order_test.go for the forwarding tests.
// ---------------------------------------------------------------------------

// ArgMin returns the index of the smallest element of xs, or -1 for an empty or
// NaN-containing input. See [vec.ArgMin].
func ArgMin(xs []float64) int { return vec.ArgMin(xs) }

// ArgMax returns the index of the largest element of xs, or -1 for an empty or
// NaN-containing input. See [vec.ArgMax].
func ArgMax(xs []float64) int { return vec.ArgMax(xs) }

// ArgMinMax returns both extreme indices in a single pass. See [vec.ArgMinMax].
func ArgMinMax(xs []float64) (loIdx, hiIdx int) { return vec.ArgMinMax(xs) }

// HasNaN reports whether any element of xs is NaN. See [vec.HasNaN].
func HasNaN(xs []float64) bool { return vec.HasNaN(xs) }

// Median returns the median of xs, or NaN for an empty or NaN-containing input.
// See [vec.Median].
func Median(xs []float64) float64 { return vec.Median(xs) }

// MedianInto returns the median of xs using buf as scratch. See [vec.MedianInto].
func MedianInto(xs []float64, buf []float64) float64 { return vec.MedianInto(xs, buf) }

// Quantile returns the q-quantile of xs for q in [0,1]. See [vec.Quantile].
func Quantile(xs []float64, q float64) float64 { return vec.Quantile(xs, q) }

// QuantileInto returns the q-quantile of xs using buf as scratch.
// See [vec.QuantileInto].
func QuantileInto(xs []float64, q float64, buf []float64) float64 {
	return vec.QuantileInto(xs, q, buf)
}

// Percentile returns the p-th percentile of xs for p in [0,100].
// See [vec.Percentile].
func Percentile(xs []float64, p float64) float64 { return vec.Percentile(xs, p) }

// PercentileInto returns the p-th percentile of xs using buf as scratch.
// See [vec.PercentileInto].
func PercentileInto(xs []float64, p float64, buf []float64) float64 {
	return vec.PercentileInto(xs, p, buf)
}

// SortInPlace sorts xs ascending in place. See [vec.SortInPlace].
func SortInPlace(xs []float64) []float64 { return vec.SortInPlace(xs) }

// SortCopy returns a sorted copy of xs. See [vec.SortCopy].
func SortCopy(xs []float64) []float64 { return vec.SortCopy(xs) }

// Rank assigns ascending average ranks, with NaN receiving NaN. See [vec.Rank].
func Rank(xs []float64) []float64 { return vec.Rank(xs) }

// RankTo stores the ascending average ranks of xs into dst. See [vec.RankTo].
func RankTo(dst, xs []float64) []float64 { return vec.RankTo(dst, xs) }

// MAD returns the median absolute deviation of xs. See [vec.MAD].
func MAD(xs []float64) float64 { return vec.MAD(xs) }

// RankScratchTo stores the ascending average ranks of xs into dst using scratch as working space.
// See [vec.RankScratchTo].
func RankScratchTo(dst, xs []float64, scratch []int) []float64 {
	return vec.RankScratchTo(dst, xs, scratch)
}
