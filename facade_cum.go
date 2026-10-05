package numa

import "github.com/rj-raghavjoshi/numa/vec"

// ---------------------------------------------------------------------------
// Facade forwarders for vec/cum.go: cumulative (prefix) operations.
//
// Each is a one-line forwarder to vec. See facade_math.go for why the forwarders
// are split per family, and facade_cum_test.go for the forwarding tests.
// ---------------------------------------------------------------------------

// CumSum returns the inclusive prefix sum of xs. See [vec.CumSum].
func CumSum(xs []float64) []float64 { return vec.CumSum(xs) }

// CumSumTo stores the inclusive prefix sum of xs into dst. See [vec.CumSumTo].
func CumSumTo(dst, xs []float64) []float64 { return vec.CumSumTo(dst, xs) }

// CumProd returns the inclusive prefix product of xs. See [vec.CumProd].
func CumProd(xs []float64) []float64 { return vec.CumProd(xs) }

// CumProdTo stores the inclusive prefix product of xs into dst. See [vec.CumProdTo].
func CumProdTo(dst, xs []float64) []float64 { return vec.CumProdTo(dst, xs) }

// CumMax returns the running maximum of xs, with NaN winning. See [vec.CumMax].
func CumMax(xs []float64) []float64 { return vec.CumMax(xs) }

// CumMaxTo stores the running maximum of xs into dst. See [vec.CumMaxTo].
func CumMaxTo(dst, xs []float64) []float64 { return vec.CumMaxTo(dst, xs) }

// CumMin returns the running minimum of xs, with NaN winning. See [vec.CumMin].
func CumMin(xs []float64) []float64 { return vec.CumMin(xs) }

// CumMinTo stores the running minimum of xs into dst. See [vec.CumMinTo].
func CumMinTo(dst, xs []float64) []float64 { return vec.CumMinTo(dst, xs) }

// CumCountTrue returns the running count of non-zero mask entries.
// See [vec.CumCountTrue].
func CumCountTrue(mask []uint8) []int { return vec.CumCountTrue(mask) }

// CumCountTrueTo stores the running count of non-zero mask entries into dst.
// See [vec.CumCountTrueTo].
func CumCountTrueTo(dst []int, mask []uint8) []int { return vec.CumCountTrueTo(dst, mask) }
