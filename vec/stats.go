package vec

import "math"

// ---------------------------------------------------------------------------
// Descriptive statistics over a whole slice.
//
// # Two passes, not one
//
// The tempting one-pass variance is
//
//	variance = mean(x^2) - mean(x)^2
//
// It is wrong in practice. Subtracting two large, nearly equal numbers destroys the
// significant digits exactly when the spread is small relative to the level, which
// is the normal case for a price series: a stock trading near 100 with a daily range
// of 0.5 loses roughly seven of the sixteen available digits to that subtraction,
// and the result can even come out *negative*.
//
// Every function here therefore centres the data first: it computes the mean in one
// pass, then accumulates moments of (x - mean) in a second. That costs a second read
// of the slice and buys back the accuracy completely. The deviation sums are
// arch-tuned (`dotDevArch`), so the second pass runs at whatever the machine's
// reduction throughput is rather than at the naive loop's.
//
// TestVarianceIsAccurateOnShiftedData pins this with data whose naive one-pass
// variance is badly wrong, and TestVarianceBeatsNaiveOnePass asserts the two-pass
// result is closer to a high-precision reference.
//
// # Empty and degenerate inputs
//
// Unlike Sum and Mean, which return 0 for an empty slice, these functions return
// **NaN** for an input on which the quantity is undefined: an empty slice, a sample
// statistic with fewer than two observations, or a correlation whose denominator is
// zero. A variance of 0 would assert "no spread", which is a different and false
// claim about an empty input. The divergence from Mean's contract is deliberate.
//
// # NaN propagation
//
// A NaN anywhere in the input makes every result NaN without any explicit check,
// because the mean becomes NaN and every deviation from NaN is NaN. That is the
// package's "NaN wins" policy falling out of the arithmetic rather than being
// re-implemented.
// ---------------------------------------------------------------------------

// Variance returns the population variance of xs.
//
// It returns NaN for an empty input or one containing NaN, and 0 for a
// single-element input (a single observation has no spread).
func Variance(xs []float64) float64 {
	n := len(xs)
	if n == 0 {
		return math.NaN()
	}
	mean := sumArch(xs) / float64(n)
	return dotDevArch(xs, xs, mean, mean) / float64(n)
}

// VarianceSample returns the unbiased sample variance of xs, dividing by n-1.
//
// It returns NaN for an input with fewer than two elements or one containing NaN.
func VarianceSample(xs []float64) float64 {
	n := len(xs)
	if n < 2 {
		return math.NaN()
	}
	mean := sumArch(xs) / float64(n)
	return dotDevArch(xs, xs, mean, mean) / float64(n-1)
}

// StdDev returns the population standard deviation of xs, the square root of
// [Variance].
func StdDev(xs []float64) float64 {
	return math.Sqrt(Variance(xs))
}

// StdDevSample returns the sample standard deviation of xs, the square root of
// [VarianceSample].
func StdDevSample(xs []float64) float64 {
	return math.Sqrt(VarianceSample(xs))
}

// Moment returns the k-th central moment of xs, the mean of (x - mean)^k.
//
// k = 0 returns 1, k = 1 returns 0, and k = 2 is [Variance]. The values 2, 3 and 4
// use tuned multi-accumulator loops; larger k falls back to math.Pow per element.
//
// It returns NaN for an empty input, for a negative k, or for an input containing
// NaN.
func Moment(xs []float64, k int) float64 {
	n := len(xs)
	if n == 0 || k < 0 {
		return math.NaN()
	}
	if k == 0 {
		return 1
	}
	mean := sumArch(xs) / float64(n)
	switch k {
	case 1:
		return 0
	case 2:
		return dotDevArch(xs, xs, mean, mean) / float64(n)
	case 3:
		return sumCubedDev(xs, mean) / float64(n)
	case 4:
		return sumFourthDev(xs, mean) / float64(n)
	}
	var total float64
	for _, v := range xs {
		total += math.Pow(v-mean, float64(k))
	}
	return total / float64(n)
}

// Covariance returns the population covariance of xs and ys, over
// min(len(xs), len(ys)) elements.
//
// It returns NaN for an empty input or one containing NaN.
func Covariance(xs, ys []float64) float64 {
	n := min(len(xs), len(ys))
	if n == 0 {
		return math.NaN()
	}
	mx := sumArch(xs[:n]) / float64(n)
	my := sumArch(ys[:n]) / float64(n)
	return dotDevArch(xs[:n], ys[:n], mx, my) / float64(n)
}

// CovarianceSample returns the unbiased sample covariance of xs and ys, dividing by
// n-1.
//
// It returns NaN for fewer than two elements or an input containing NaN.
func CovarianceSample(xs, ys []float64) float64 {
	n := min(len(xs), len(ys))
	if n < 2 {
		return math.NaN()
	}
	mx := sumArch(xs[:n]) / float64(n)
	my := sumArch(ys[:n]) / float64(n)
	return dotDevArch(xs[:n], ys[:n], mx, my) / float64(n-1)
}

// Correlation returns the Pearson correlation coefficient of xs and ys, a value in
// [-1, 1].
//
// It returns NaN for fewer than two elements, for an input containing NaN, or when
// either series has zero variance. The zero-variance case is NaN rather than 0
// because a constant series carries no information about co-movement; reporting 0
// would claim "uncorrelated", which is a different statement from "undefined".
func Correlation(xs, ys []float64) float64 {
	n := min(len(xs), len(ys))
	if n < 2 {
		return math.NaN()
	}
	mx := sumArch(xs[:n]) / float64(n)
	my := sumArch(ys[:n]) / float64(n)
	sxy := dotDevArch(xs[:n], ys[:n], mx, my)
	sxx := dotDevArch(xs[:n], xs[:n], mx, mx)
	syy := dotDevArch(ys[:n], ys[:n], my, my)
	den := math.Sqrt(sxx * syy)
	if den == 0 {
		return math.NaN()
	}
	return sxy / den
}

// Skewness returns the population skewness of xs, the third central moment divided
// by the 1.5 power of the second.
//
// It returns NaN for an empty input, for an input containing NaN, and for a
// constant input, whose third moment is 0 over a zero denominator.
func Skewness(xs []float64) float64 {
	n := len(xs)
	if n == 0 {
		return math.NaN()
	}
	mean := sumArch(xs) / float64(n)
	s2 := dotDevArch(xs, xs, mean, mean) / float64(n)
	if s2 == 0 {
		return math.NaN()
	}
	s3 := sumCubedDev(xs, mean) / float64(n)
	return s3 / math.Pow(s2, 1.5)
}

// Kurtosis returns the excess kurtosis of xs, the fourth central moment divided by
// the square of the second, minus 3.
//
// Excess rather than raw kurtosis so that a normal distribution scores 0, which is
// the convention that makes the number interpretable at a glance. It returns NaN for
// an empty input, for an input containing NaN, and for a constant input.
func Kurtosis(xs []float64) float64 {
	n := len(xs)
	if n == 0 {
		return math.NaN()
	}
	mean := sumArch(xs) / float64(n)
	s2 := dotDevArch(xs, xs, mean, mean) / float64(n)
	if s2 == 0 {
		return math.NaN()
	}
	s4 := sumFourthDev(xs, mean) / float64(n)
	return s4/(s2*s2) - 3
}

// MeanAbsDeviation returns the mean absolute deviation of xs from its mean:
// mean(|x - mean(x)|).
//
// It reads the input twice, like the other statistics here. It is a less robust
// spread measure than [MAD], which centres on the median, so a caller worried about
// outliers usually wants that instead.
//
// It returns NaN for an empty input or one containing NaN.
func MeanAbsDeviation(xs []float64) float64 {
	n := len(xs)
	if n == 0 {
		return math.NaN()
	}
	mean := sumArch(xs) / float64(n)
	return sumAbsDevArch(xs, mean) / float64(n)
}

// RMS returns the root mean square of xs.
//
// It returns NaN for an empty input or one containing NaN. Note that it does not
// centre the data, so a constant offset raises the RMS; that is the definition, not
// an error.
func RMS(xs []float64) float64 {
	n := len(xs)
	if n == 0 {
		return math.NaN()
	}
	return math.Sqrt(sumSqArch(xs) / float64(n))
}

// GeoMean returns the geometric mean of xs, the n-th root of their product.
//
// It is computed in log space, so it does not overflow where the product would. A
// non-positive element makes the result NaN, except for exact zeros, which drive it
// to 0 via log(0) = -Inf.
//
// It returns NaN for an empty input.
func GeoMean(xs []float64) float64 {
	n := len(xs)
	if n == 0 {
		return math.NaN()
	}
	var total float64
	for _, v := range xs {
		total += math.Log(v)
	}
	return math.Exp(total / float64(n))
}

// HarmMean returns the harmonic mean of xs, n divided by the sum of reciprocals.
//
// A zero element gives +Inf for its reciprocal and therefore 0 for the mean,
// following IEEE-754 rather than being special-cased. It returns NaN for an empty
// input.
func HarmMean(xs []float64) float64 {
	n := len(xs)
	if n == 0 {
		return math.NaN()
	}
	var total float64
	for _, v := range xs {
		total += 1 / v
	}
	return float64(n) / total
}

// ---------------------------------------------------------------------------
// Third and fourth deviation sums.
//
// These are untagged, unlike the deviation sum in the arch files. The reasoning is
// the same as for vec/math.go's maps: the four-wide split below is not an
// architecture-specific choice (there is no accumulator count that varies by
// machine), it is simply a wider unroll of one loop. Duplicating it behind three
// build tags would be duplication dressed up as tuning.
//
// They are separate from dotDevArch because a third or fourth power cannot be
// expressed as a dot product of two slices the caller already has, and
// materialising a deviations slice to feed Dot would cost a full extra memory
// traversal.
// ---------------------------------------------------------------------------

func sumCubedDev(xs []float64, center float64) float64 {
	n := len(xs)
	var s0, s1, s2, s3 float64
	i := 0
	limit := n - 3
	for i < limit {
		d0 := xs[i] - center
		d1 := xs[i+1] - center
		d2 := xs[i+2] - center
		d3 := xs[i+3] - center
		s0 += d0 * d0 * d0
		s1 += d1 * d1 * d1
		s2 += d2 * d2 * d2
		s3 += d3 * d3 * d3
		i += 4
	}
	for ; i < n; i++ {
		d := xs[i] - center
		s0 += d * d * d
	}
	return (s0 + s1) + (s2 + s3)
}

func sumFourthDev(xs []float64, center float64) float64 {
	n := len(xs)
	var s0, s1, s2, s3 float64
	i := 0
	limit := n - 3
	for i < limit {
		d0 := xs[i] - center
		d1 := xs[i+1] - center
		d2 := xs[i+2] - center
		d3 := xs[i+3] - center
		s0 += d0 * d0 * d0 * d0
		s1 += d1 * d1 * d1 * d1
		s2 += d2 * d2 * d2 * d2
		s3 += d3 * d3 * d3 * d3
		i += 4
	}
	for ; i < n; i++ {
		d := xs[i] - center
		s0 += d * d * d * d
	}
	return (s0 + s1) + (s2 + s3)
}
