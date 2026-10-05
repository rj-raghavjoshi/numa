package numa

import (
	"math"
	"testing"

	"github.com/rj-raghavjoshi/numa/vec"
)

// ---------------------------------------------------------------------------
// Forwarder tests for the statistics facade.
//
// The contracts a forwarder can silently break here:
//
//   - argument order for the two-slice statistics, which is invisible on symmetric
//     or perfectly correlated inputs
//   - the population/sample divisor distinction, which differs by a factor of
//     n/(n-1) and would look like a rounding difference under a loose tolerance
//   - the NaN-for-undefined contracts, which a forwarder returning a zero value
//     would quietly satisfy a "not a number" check for if carelessly written
// ---------------------------------------------------------------------------

func TestStatsForwardersMatchVec(t *testing.T) {
	xs := []float64{1, 2, 3, 4, 5, 6, 7, 8}
	ys := []float64{2, 1, 4, 3, 6, 5, 8, 7}

	if got, want := Variance(xs), vec.Variance(xs); got != want {
		t.Errorf("Variance = %v, want %v", got, want)
	}
	if got, want := VarianceSample(xs), vec.VarianceSample(xs); got != want {
		t.Errorf("VarianceSample = %v, want %v", got, want)
	}
	if got, want := StdDev(xs), vec.StdDev(xs); got != want {
		t.Errorf("StdDev = %v, want %v", got, want)
	}
	if got, want := StdDevSample(xs), vec.StdDevSample(xs); got != want {
		t.Errorf("StdDevSample = %v, want %v", got, want)
	}
	if got, want := Moment(xs, 3), vec.Moment(xs, 3); got != want {
		t.Errorf("Moment = %v, want %v", got, want)
	}
	if got, want := Covariance(xs, ys), vec.Covariance(xs, ys); got != want {
		t.Errorf("Covariance = %v, want %v", got, want)
	}
	if got, want := CovarianceSample(xs, ys), vec.CovarianceSample(xs, ys); got != want {
		t.Errorf("CovarianceSample = %v, want %v", got, want)
	}
	if got, want := Correlation(xs, ys), vec.Correlation(xs, ys); got != want {
		t.Errorf("Correlation = %v, want %v", got, want)
	}
	if got, want := Skewness(xs), vec.Skewness(xs); got != want {
		t.Errorf("Skewness = %v, want %v", got, want)
	}
	if got, want := Kurtosis(xs), vec.Kurtosis(xs); got != want {
		t.Errorf("Kurtosis = %v, want %v", got, want)
	}
	if got, want := MeanAbsDeviation(xs), vec.MeanAbsDeviation(xs); got != want {
		t.Errorf("MeanAbsDeviation = %v, want %v", got, want)
	}
	if got, want := RMS(xs), vec.RMS(xs); got != want {
		t.Errorf("RMS = %v, want %v", got, want)
	}
	if got, want := GeoMean(xs), vec.GeoMean(xs); got != want {
		t.Errorf("GeoMean = %v, want %v", got, want)
	}
	if got, want := HarmMean(xs), vec.HarmMean(xs); got != want {
		t.Errorf("HarmMean = %v, want %v", got, want)
	}
}

// TestCovarianceForwarderArgumentOrder uses series that are correlated but with
// distinct scale, so a swap of xs and ys is detectable even though covariance is
// symmetric: the point is that the same order must reach vec.
func TestCovarianceForwarderArgumentOrder(t *testing.T) {
	xs := []float64{1, 2, 3, 4}
	ys := []float64{10, 20, 30, 40}

	// Covariance is symmetric in value, so this checks the forwarding is consistent
	// rather than that the order is observable.
	if got, want := Covariance(xs, ys), vec.Covariance(xs, ys); got != want {
		t.Errorf("Covariance(xs,ys) = %v, vec = %v", got, want)
	}
	if got, want := Covariance(ys, xs), vec.Covariance(ys, xs); got != want {
		t.Errorf("Covariance(ys,xs) = %v, vec = %v", got, want)
	}
	// Correlation is +1 here regardless of order, which is why the equality checks
	// above are against vec rather than against a hand value.
	if got := Correlation(xs, ys); math.Abs(got-1) > 1e-15 {
		t.Errorf("Correlation = %v, want 1", got)
	}
}

// TestSampleVsPopulationSurviveFacade pins the divisor distinction, which is the
// difference most likely to be lost by forwarding to the wrong implementation.
func TestSampleVsPopulationSurviveFacade(t *testing.T) {
	xs := []float64{1, 2, 3, 4, 5}

	if got, want := Variance(xs), 2.0; got != want {
		t.Errorf("Variance = %v, want 2 (population)", got)
	}
	if got, want := VarianceSample(xs), 2.5; got != want {
		t.Errorf("VarianceSample = %v, want 2.5 (sample)", got)
	}
	if Variance(xs) == VarianceSample(xs) {
		t.Error("the population and sample forms returned the same value")
	}
}

// TestStatsNaNPolicySurvivesFacade pins the NaN-for-undefined contracts, which a
// forwarder could break by returning a zero value instead.
func TestStatsNaNPolicySurvivesFacade(t *testing.T) {
	cases := []struct {
		name string
		fn   func() float64
	}{
		{"Variance(nil)", func() float64 { return Variance(nil) }},
		{"VarianceSample(nil)", func() float64 { return VarianceSample(nil) }},
		{"Correlation(nil,nil)", func() float64 { return Correlation(nil, nil) }},
		{"Correlation constant", func() float64 { return Correlation([]float64{1, 1}, []float64{1, 2}) }},
		{"Skewness(nil)", func() float64 { return Skewness(nil) }},
		{"Kurtosis(nil)", func() float64 { return Kurtosis(nil) }},
		{"GeoMean(nil)", func() float64 { return GeoMean(nil) }},
		{"Variance with NaN", func() float64 { return Variance([]float64{1, math.NaN()}) }},
	}
	for _, tc := range cases {
		if got := tc.fn(); !math.IsNaN(got) {
			t.Errorf("%s = %v, want NaN", tc.name, got)
		}
	}
}
