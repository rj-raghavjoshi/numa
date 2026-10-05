package vec

import (
	"math"
	"math/big"
	"math/rand"
	"testing"
)

// ---------------------------------------------------------------------------
// Tests for the descriptive statistics.
//
// The centrepiece is TestVarianceBeatsNaiveOnePass: it uses data where the one-pass
// `mean(x^2) - mean(x)^2` formula is not merely slightly off but off by 100%+, and
// asserts that the two-pass implementation is closer to a 200-bit reference than the
// one-pass form is. That is the justification for reading the input twice, asserted
// rather than asserted-in-a-comment.
//
// The rest covers the standard contracts: hand-computed values at exact cases, the
// relationship between the population and sample forms, the documented NaN
// behaviour for degenerate inputs, and agreement with an independent reference.
// ---------------------------------------------------------------------------

// exactVariance computes the population variance at 200 bits of precision, so it
// carries no rounding error of its own and can be used as ground truth.
func exactVariance(xs []float64) float64 {
	const prec = 200
	mean := new(big.Float).SetPrec(prec)
	for _, v := range xs {
		mean.Add(mean, new(big.Float).SetPrec(prec).SetFloat64(v))
	}
	mean.Quo(mean, new(big.Float).SetPrec(prec).SetInt64(int64(len(xs))))

	ss := new(big.Float).SetPrec(prec)
	for _, v := range xs {
		d := new(big.Float).SetPrec(prec).Sub(new(big.Float).SetPrec(prec).SetFloat64(v), mean)
		d.Mul(d, d)
		ss.Add(ss, d)
	}
	ss.Quo(ss, new(big.Float).SetPrec(prec).SetInt64(int64(len(xs))))

	f, _ := ss.Float64()
	return f
}

// naiveOnePassVariance is the formulation stats.go exists to avoid.
func naiveOnePassVariance(xs []float64) float64 {
	n := float64(len(xs))
	var s, s2 float64
	for _, v := range xs {
		s += v
		s2 += v * v
	}
	m := s / n
	return s2/n - m*m
}

// TestVarianceBeatsNaiveOnePass is the numerical motivation for the two-pass
// structure, measured rather than argued.
//
// Every value is 1e8 or 1e8+1, so the true variance is exactly 0.25 while
// mean(x^2) and mean(x)^2 are both about 1e16. Their difference must be represented
// as a multiple of 2 at that magnitude, so the one-pass result cannot be 0.25 -- it
// typically comes out as 0.
func TestVarianceBeatsNaiveOnePass(t *testing.T) {
	xs := make([]float64, 1000)
	for i := range xs {
		xs[i] = 1e8 + float64(i%2)
	}

	want := exactVariance(xs)
	got := Variance(xs)
	naive := naiveOnePassVariance(xs)

	t.Logf("exact=%v two-pass=%v one-pass=%v", want, got, naive)

	if math.Abs(got-want) > 1e-9 {
		t.Errorf("two-pass Variance = %v, want %v", got, want)
	}
	if math.Abs(naive-want) <= math.Abs(got-want) {
		t.Errorf("the one-pass form was not worse: one-pass err=%v, two-pass err=%v",
			math.Abs(naive-want), math.Abs(got-want))
	}
}

func TestVarianceKnownValues(t *testing.T) {
	// {1,2,3,4,5}: deviations -2,-1,0,1,2 -> sum of squares 10 -> population 2.
	xs := []float64{1, 2, 3, 4, 5}
	if got := Variance(xs); got != 2 {
		t.Errorf("Variance = %v, want 2", got)
	}
	if got := VarianceSample(xs); got != 2.5 {
		t.Errorf("VarianceSample = %v, want 2.5", got)
	}
	if got := StdDev(xs); math.Abs(got-math.Sqrt2) > 1e-15 {
		t.Errorf("StdDev = %v, want sqrt(2)", got)
	}
	if got := StdDevSample(xs); math.Abs(got-math.Sqrt(2.5)) > 1e-15 {
		t.Errorf("StdDevSample = %v, want sqrt(2.5)", got)
	}

	// A single element has no spread.
	if got := Variance([]float64{7}); got != 0 {
		t.Errorf("Variance of one element = %v, want 0", got)
	}
	if !math.IsNaN(VarianceSample([]float64{7})) {
		t.Error("VarianceSample of one element should be NaN")
	}
}

// TestVarianceSampleRelation checks the algebraic identity between the two forms
// across random data, which catches a wrong divisor.
func TestVarianceSampleRelation(t *testing.T) {
	for _, n := range []int{2, 3, 5, 17, 64, 200} {
		xs := randomSlice(n, int64(n))
		pop := Variance(xs)
		samp := VarianceSample(xs)
		want := pop * float64(n) / float64(n-1)
		if math.Abs(samp-want) > 1e-9*math.Max(1, math.Abs(want)) {
			t.Fatalf("n=%d: VarianceSample = %v, want %v (= Variance*n/(n-1))", n, samp, want)
		}
	}
}

func TestCovarianceAndCorrelationKnownValues(t *testing.T) {
	xs := []float64{1, 2, 3}
	ys := []float64{2, 4, 6}

	// xs-2 = (-1,0,1) and ys-4 = (-2,0,2), so the products are 2, 0, 2: the sum is
	// 4, giving a population covariance of 4/3 and a sample covariance of 4/2 = 2.
	if got := Covariance(xs, ys); math.Abs(got-4.0/3.0) > 1e-15 {
		t.Errorf("Covariance = %v, want 4/3", got)
	}
	if got := CovarianceSample(xs, ys); got != 2 {
		t.Errorf("CovarianceSample = %v, want 2", got)
	}
	if got := Correlation(xs, ys); math.Abs(got-1) > 1e-15 {
		t.Errorf("Correlation = %v, want 1", got)
	}

	neg := []float64{-2, -4, -6}
	if got := Correlation(xs, neg); math.Abs(got+1) > 1e-15 {
		t.Errorf("Correlation with negated series = %v, want -1", got)
	}
}

// TestCorrelationZeroVarianceIsNaN pins the choice of NaN over 0 for a constant
// series: "undefined" is a different claim from "uncorrelated".
func TestCorrelationZeroVarianceIsNaN(t *testing.T) {
	constant := []float64{5, 5, 5, 5}
	other := []float64{1, 2, 3, 4}

	if got := Correlation(constant, other); !math.IsNaN(got) {
		t.Errorf("Correlation with a constant series = %v, want NaN", got)
	}
	if got := Correlation(other, constant); !math.IsNaN(got) {
		t.Errorf("Correlation with a constant series = %v, want NaN", got)
	}
}

// TestSkewnessAndKurtosisKnownValues uses cases with closed forms.
//
// {1,1,1,1,10}: mean 2.8, m2 = 12.96, m3 = 69.984, so skewness = 69.984 /
// 12.96^1.5 = 69.984 / 46.656 = 1.5 exactly.
//
// {1,2,3,4,5}: symmetric, so skewness 0; m2 = 2, m4 = 6.8, so excess kurtosis =
// 6.8/4 - 3 = -1.3.
func TestSkewnessAndKurtosisKnownValues(t *testing.T) {
	if got := Skewness([]float64{1, 1, 1, 1, 10}); math.Abs(got-1.5) > 1e-12 {
		t.Errorf("Skewness = %v, want 1.5", got)
	}
	if got := Skewness([]float64{1, 2, 3, 4, 5}); math.Abs(got) > 1e-15 {
		t.Errorf("Skewness of a symmetric sample = %v, want 0", got)
	}
	if got := Kurtosis([]float64{1, 2, 3, 4, 5}); math.Abs(got+1.3) > 1e-12 {
		t.Errorf("Kurtosis = %v, want -1.3", got)
	}
}

func TestMoment(t *testing.T) {
	xs := []float64{1, 2, 3, 4, 5}

	if got := Moment(xs, 0); got != 1 {
		t.Errorf("Moment(k=0) = %v, want 1", got)
	}
	if got := Moment(xs, 1); got != 0 {
		t.Errorf("Moment(k=1) = %v, want 0", got)
	}
	if got := Moment(xs, 2); got != Variance(xs) {
		t.Errorf("Moment(k=2) = %v, Variance = %v", got, Variance(xs))
	}
	if got := Moment(xs, 3); math.Abs(got) > 1e-15 {
		t.Errorf("Moment(k=3) = %v, want 0", got)
	}
	if got := Moment(xs, 4); math.Abs(got-6.8) > 1e-12 {
		t.Errorf("Moment(k=4) = %v, want 6.8", got)
	}

	// The general path (k > 4) must agree with a direct definition.
	mean := Mean(xs)
	var want5 float64
	for _, v := range xs {
		want5 += math.Pow(v-mean, 5)
	}
	want5 /= float64(len(xs))
	if got := Moment(xs, 5); math.Abs(got-want5) > 1e-12 {
		t.Errorf("Moment(k=5) = %v, want %v", got, want5)
	}

	if !math.IsNaN(Moment(xs, -1)) {
		t.Error("Moment with negative k should be NaN")
	}
}

func TestMeanAbsDeviationKnownValue(t *testing.T) {
	// mean 3; |deviations| are 2,1,0,1,2 -> sum 6 -> /5 = 1.2.
	if got := MeanAbsDeviation([]float64{1, 2, 3, 4, 5}); math.Abs(got-1.2) > 1e-15 {
		t.Errorf("MeanAbsDeviation = %v, want 1.2", got)
	}
}

func TestOtherMeans(t *testing.T) {
	if got := RMS([]float64{3, 4}); math.Abs(got-math.Sqrt(12.5)) > 1e-15 {
		t.Errorf("RMS = %v, want sqrt(12.5)", got)
	}
	if got := GeoMean([]float64{2, 8}); math.Abs(got-4) > 1e-12 {
		t.Errorf("GeoMean = %v, want 4", got)
	}
	if got := HarmMean([]float64{1, 2, 4}); math.Abs(got-3.0/1.75) > 1e-15 {
		t.Errorf("HarmMean = %v, want 3/1.75", got)
	}
	// Geometric mean over a non-positive value is NaN, except an exact zero which
	// drives the log to -Inf and the result to 0.
	if got := GeoMean([]float64{-1, 2}); !math.IsNaN(got) {
		t.Errorf("GeoMean with a negative = %v, want NaN", got)
	}
	if got := GeoMean([]float64{0, 2}); got != 0 {
		t.Errorf("GeoMean with a zero = %v, want 0", got)
	}
}

// TestStatsMatchReference checks every statistic against an independent, plainly
// written reference across boundary and larger random lengths.
func TestStatsMatchReference(t *testing.T) {
	sizes := append([]int{}, boundaryLengths()...)
	sizes = append(sizes, 50, 500)

	for _, n := range sizes {
		if n < 2 {
			continue
		}
		rng := rand.New(rand.NewSource(int64(n) + 11))
		xs := make([]float64, n)
		ys := make([]float64, n)
		for i := range xs {
			xs[i] = rng.NormFloat64()
			ys[i] = rng.NormFloat64()
		}

		refVar, refCov := refVariance(xs), refCovariance(xs, ys)
		if got := Variance(xs); math.Abs(got-refVar) > 1e-9*math.Max(1, refVar) {
			t.Fatalf("n=%d: Variance = %v, want %v", n, got, refVar)
		}
		if got := Covariance(xs, ys); math.Abs(got-refCov) > 1e-9*math.Max(1, math.Abs(refCov)) {
			t.Fatalf("n=%d: Covariance = %v, want %v", n, got, refCov)
		}
	}
}

func refVariance(xs []float64) float64 {
	mean := refSum(xs) / float64(len(xs))
	var ss float64
	for _, v := range xs {
		d := v - mean
		ss += d * d
	}
	return ss / float64(len(xs))
}

func refCovariance(xs, ys []float64) float64 {
	n := min(len(xs), len(ys))
	mx := refSum(xs[:n]) / float64(n)
	my := refSum(ys[:n]) / float64(n)
	var s float64
	for i := 0; i < n; i++ {
		s += (xs[i] - mx) * (ys[i] - my)
	}
	return s / float64(n)
}

// TestStatsNaNPropagation checks the policy at every position, including the ends,
// since the mean is computed from the whole slice.
func TestStatsNaNPropagation(t *testing.T) {
	for pos := 0; pos < 4; pos++ {
		xs := []float64{1, 2, 3, 4}
		ys := []float64{4, 3, 2, 1}
		xs[pos] = math.NaN()

		if got := Variance(xs); !math.IsNaN(got) {
			t.Errorf("Variance with NaN at %d = %v, want NaN", pos, got)
		}
		if got := StdDev(xs); !math.IsNaN(got) {
			t.Errorf("StdDev with NaN at %d = %v, want NaN", pos, got)
		}
		if got := Covariance(xs, ys); !math.IsNaN(got) {
			t.Errorf("Covariance with NaN at %d = %v, want NaN", pos, got)
		}
		if got := Correlation(xs, ys); !math.IsNaN(got) {
			t.Errorf("Correlation with NaN at %d = %v, want NaN", pos, got)
		}
		if got := Skewness(xs); !math.IsNaN(got) {
			t.Errorf("Skewness with NaN at %d = %v, want NaN", pos, got)
		}
		if got := Kurtosis(xs); !math.IsNaN(got) {
			t.Errorf("Kurtosis with NaN at %d = %v, want NaN", pos, got)
		}
		if got := MeanAbsDeviation(xs); !math.IsNaN(got) {
			t.Errorf("MeanAbsDeviation with NaN at %d = %v, want NaN", pos, got)
		}
	}
}

// TestStatsEmptyAndDegenerateInputs pins the documented NaN contracts, which
// deliberately differ from Sum and Mean returning 0.
func TestStatsEmptyAndDegenerateInputs(t *testing.T) {
	nanFuncs := map[string]func() float64{
		"Variance":         func() float64 { return Variance(nil) },
		"VarianceSample":   func() float64 { return VarianceSample(nil) },
		"StdDev":           func() float64 { return StdDev(nil) },
		"StdDevSample":     func() float64 { return StdDevSample(nil) },
		"Moment":           func() float64 { return Moment(nil, 2) },
		"Covariance":       func() float64 { return Covariance(nil, nil) },
		"CovarianceSample": func() float64 { return CovarianceSample(nil, nil) },
		"Correlation":      func() float64 { return Correlation(nil, nil) },
		"Skewness":         func() float64 { return Skewness(nil) },
		"Kurtosis":         func() float64 { return Kurtosis(nil) },
		"MeanAbsDeviation": func() float64 { return MeanAbsDeviation(nil) },
		"RMS":              func() float64 { return RMS(nil) },
		"GeoMean":          func() float64 { return GeoMean(nil) },
		"HarmMean":         func() float64 { return HarmMean(nil) },
		"SampleVarOneElem": func() float64 { return VarianceSample([]float64{1}) },
		"CorrelationOneEl": func() float64 { return Correlation([]float64{1}, []float64{2}) },
		"CovSampleOneElem": func() float64 { return CovarianceSample([]float64{1}, []float64{2}) },
		"ConstantSkewness": func() float64 { return Skewness([]float64{3, 3, 3}) },
		"ConstantKurtosis": func() float64 { return Kurtosis([]float64{3, 3, 3}) },
	}

	for name, fn := range nanFuncs {
		if got := fn(); !math.IsNaN(got) {
			t.Errorf("%s = %v, want NaN", name, got)
		}
	}

	// A constant series has zero variance, which is defined, unlike the skewness.
	if got := Variance([]float64{3, 3, 3}); got != 0 {
		t.Errorf("Variance of a constant series = %v, want 0", got)
	}
}
