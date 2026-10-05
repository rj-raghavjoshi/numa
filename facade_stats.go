package numa

import "github.com/rj-raghavjoshi/numa/vec"

// ---------------------------------------------------------------------------
// Facade forwarders for vec/stats.go: descriptive statistics.
//
// Each is a one-line forwarder to vec. See facade_math.go for why the forwarders
// are split per family, and facade_stats_test.go for the forwarding tests.
// ---------------------------------------------------------------------------

// Variance returns the population variance of xs. See [vec.Variance].
func Variance(xs []float64) float64 { return vec.Variance(xs) }

// VarianceSample returns the unbiased sample variance of xs. See [vec.VarianceSample].
func VarianceSample(xs []float64) float64 { return vec.VarianceSample(xs) }

// StdDev returns the population standard deviation of xs. See [vec.StdDev].
func StdDev(xs []float64) float64 { return vec.StdDev(xs) }

// StdDevSample returns the sample standard deviation of xs. See [vec.StdDevSample].
func StdDevSample(xs []float64) float64 { return vec.StdDevSample(xs) }

// Moment returns the k-th central moment of xs. See [vec.Moment].
func Moment(xs []float64, k int) float64 { return vec.Moment(xs, k) }

// Covariance returns the population covariance of xs and ys. See [vec.Covariance].
func Covariance(xs, ys []float64) float64 { return vec.Covariance(xs, ys) }

// CovarianceSample returns the unbiased sample covariance. See [vec.CovarianceSample].
func CovarianceSample(xs, ys []float64) float64 { return vec.CovarianceSample(xs, ys) }

// Correlation returns the Pearson correlation coefficient. See [vec.Correlation].
func Correlation(xs, ys []float64) float64 { return vec.Correlation(xs, ys) }

// Skewness returns the population skewness of xs. See [vec.Skewness].
func Skewness(xs []float64) float64 { return vec.Skewness(xs) }

// Kurtosis returns the excess kurtosis of xs. See [vec.Kurtosis].
func Kurtosis(xs []float64) float64 { return vec.Kurtosis(xs) }

// MeanAbsDeviation returns the mean absolute deviation from the mean.
// See [vec.MeanAbsDeviation].
func MeanAbsDeviation(xs []float64) float64 { return vec.MeanAbsDeviation(xs) }

// RMS returns the root mean square of xs. See [vec.RMS].
func RMS(xs []float64) float64 { return vec.RMS(xs) }

// GeoMean returns the geometric mean of xs. See [vec.GeoMean].
func GeoMean(xs []float64) float64 { return vec.GeoMean(xs) }

// HarmMean returns the harmonic mean of xs. See [vec.HarmMean].
func HarmMean(xs []float64) float64 { return vec.HarmMean(xs) }
