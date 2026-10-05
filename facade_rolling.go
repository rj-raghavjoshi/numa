package numa

import "github.com/rj-raghavjoshi/numa/vec"

// ---------------------------------------------------------------------------
// Facade forwarders for vec/rolling.go: rolling-window batch kernels.
//
// Each is a one-line forwarder. See facade_math.go for why the forwarders are split per family.
// ---------------------------------------------------------------------------

// RollingSum returns the trailing n-value sum at each position. See [vec.RollingSum].
func RollingSum(xs []float64, n int) []float64 { return vec.RollingSum(xs, n) }

// RollingSumTo stores the trailing n-value sum into dst; dst must not alias xs.
// See [vec.RollingSumTo].
func RollingSumTo(dst, xs []float64, n int) []float64 { return vec.RollingSumTo(dst, xs, n) }

// RollingMean returns the trailing n-value mean. See [vec.RollingMean].
func RollingMean(xs []float64, n int) []float64 { return vec.RollingMean(xs, n) }

// RollingMeanTo stores the trailing n-value mean into dst; dst must not alias xs.
// See [vec.RollingMeanTo].
func RollingMeanTo(dst, xs []float64, n int) []float64 { return vec.RollingMeanTo(dst, xs, n) }

// RollingMax returns the trailing n-value maximum. See [vec.RollingMax].
func RollingMax(xs []float64, n int) []float64 { return vec.RollingMax(xs, n) }

// RollingMaxTo stores the trailing n-value maximum into dst; dst must not alias xs.
// See [vec.RollingMaxTo].
func RollingMaxTo(dst, xs []float64, n int) []float64 { return vec.RollingMaxTo(dst, xs, n) }

// RollingMin returns the trailing n-value minimum. See [vec.RollingMin].
func RollingMin(xs []float64, n int) []float64 { return vec.RollingMin(xs, n) }

// RollingMinTo stores the trailing n-value minimum into dst; dst must not alias xs.
// See [vec.RollingMinTo].
func RollingMinTo(dst, xs []float64, n int) []float64 { return vec.RollingMinTo(dst, xs, n) }

// RollingRange returns the trailing n-value high-low range. See [vec.RollingRange].
func RollingRange(xs []float64, n int) []float64 { return vec.RollingRange(xs, n) }

// RollingRangeTo stores the trailing n-value range into dst. See [vec.RollingRangeTo].
func RollingRangeTo(dst, xs []float64, n int) []float64 { return vec.RollingRangeTo(dst, xs, n) }

// RollingWMA returns the trailing n-value weighted moving average. See [vec.RollingWMA].
func RollingWMA(xs []float64, n int) []float64 { return vec.RollingWMA(xs, n) }

// RollingWMATo stores the trailing n-value weighted average into dst; dst must not alias xs.
// See [vec.RollingWMATo].
func RollingWMATo(dst, xs []float64, n int) []float64 { return vec.RollingWMATo(dst, xs, n) }

// RollingVariance returns the trailing n-value population variance. See [vec.RollingVariance].
func RollingVariance(xs []float64, n int) []float64 { return vec.RollingVariance(xs, n) }

// RollingVarianceSample returns the trailing n-value sample variance.
// See [vec.RollingVarianceSample].
func RollingVarianceSample(xs []float64, n int) []float64 { return vec.RollingVarianceSample(xs, n) }

// RollingStdDev returns the trailing n-value population standard deviation.
// See [vec.RollingStdDev].
func RollingStdDev(xs []float64, n int) []float64 { return vec.RollingStdDev(xs, n) }

// RollingStdDevSample returns the trailing n-value sample standard deviation.
// See [vec.RollingStdDevSample].
func RollingStdDevSample(xs []float64, n int) []float64 { return vec.RollingStdDevSample(xs, n) }

// RollingCovariance returns the trailing n-value population covariance.
// See [vec.RollingCovariance].
func RollingCovariance(xs, ys []float64, n int) []float64 { return vec.RollingCovariance(xs, ys, n) }

// RollingCovarianceSample returns the trailing n-value sample covariance.
// See [vec.RollingCovarianceSample].
func RollingCovarianceSample(xs, ys []float64, n int) []float64 {
	return vec.RollingCovarianceSample(xs, ys, n)
}

// RollingCorrelation returns the trailing n-value Pearson correlation.
// See [vec.RollingCorrelation].
func RollingCorrelation(xs, ys []float64, n int) []float64 { return vec.RollingCorrelation(xs, ys, n) }

// RollingMedian returns the trailing n-value median. See [vec.RollingMedian].
func RollingMedian(xs []float64, n int) []float64 { return vec.RollingMedian(xs, n) }

// RollingQuantile returns the trailing n-value q-quantile. See [vec.RollingQuantile].
func RollingQuantile(xs []float64, n int, q float64) []float64 { return vec.RollingQuantile(xs, n, q) }

// RollingPercentRank returns the percentage of trailing values strictly below the current one.
// See [vec.RollingPercentRank].
func RollingPercentRank(xs []float64, n int) []float64 { return vec.RollingPercentRank(xs, n) }

// RollingEMA returns the exponential moving average with alpha = 2/(n+1), SMA-seeded.
// See [vec.RollingEMA].
func RollingEMA(xs []float64, n int) []float64 { return vec.RollingEMA(xs, n) }

// RollingEMATo stores the exponential moving average into dst; dst must not alias xs.
// See [vec.RollingEMATo].
func RollingEMATo(dst, xs []float64, n int) []float64 { return vec.RollingEMATo(dst, xs, n) }

// RollingRMA returns Wilder's smoothing with alpha = 1/n, SMA-seeded. See [vec.RollingRMA].
func RollingRMA(xs []float64, n int) []float64 { return vec.RollingRMA(xs, n) }

// RollingRMATo stores Wilder's smoothing into dst; dst must not alias xs. See [vec.RollingRMATo].
func RollingRMATo(dst, xs []float64, n int) []float64 { return vec.RollingRMATo(dst, xs, n) }
