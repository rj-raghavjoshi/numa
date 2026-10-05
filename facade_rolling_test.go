package numa

import (
	"math"
	"testing"

	"github.com/rj-raghavjoshi/numa/vec"
)

// ---------------------------------------------------------------------------
// Forwarder tests for the rolling kernels.
//
// The rolling family has the most `*To` forms of anything in the package, and their contract is
// unusual: unlike the elementwise `*To` functions, dst must NOT alias xs. That makes the
// allocating form the reference here, and the tests compare the two rather than exercising
// aliasing that is documented not to work.
// ---------------------------------------------------------------------------

func TestRollingForwardersMatchVec(t *testing.T) {
	xs := rollingTestData(80, 1)
	ys := rollingTestData(80, 2)

	for _, n := range []int{1, 5, 20} {
		eqSeries(t, "RollingSum", RollingSum(xs, n), vec.RollingSum(xs, n))
		eqSeries(t, "RollingMean", RollingMean(xs, n), vec.RollingMean(xs, n))
		eqSeries(t, "RollingMax", RollingMax(xs, n), vec.RollingMax(xs, n))
		eqSeries(t, "RollingMin", RollingMin(xs, n), vec.RollingMin(xs, n))
		eqSeries(t, "RollingRange", RollingRange(xs, n), vec.RollingRange(xs, n))
		eqSeries(t, "RollingWMA", RollingWMA(xs, n), vec.RollingWMA(xs, n))
		eqSeries(t, "RollingVariance", RollingVariance(xs, n), vec.RollingVariance(xs, n))
		eqSeries(t, "RollingVarianceSample", RollingVarianceSample(xs, n), vec.RollingVarianceSample(xs, n))
		eqSeries(t, "RollingStdDev", RollingStdDev(xs, n), vec.RollingStdDev(xs, n))
		eqSeries(t, "RollingStdDevSample", RollingStdDevSample(xs, n), vec.RollingStdDevSample(xs, n))
		eqSeries(t, "RollingCovariance", RollingCovariance(xs, ys, n), vec.RollingCovariance(xs, ys, n))
		eqSeries(t, "RollingCovarianceSample", RollingCovarianceSample(xs, ys, n), vec.RollingCovarianceSample(xs, ys, n))
		eqSeries(t, "RollingCorrelation", RollingCorrelation(xs, ys, n), vec.RollingCorrelation(xs, ys, n))
		eqSeries(t, "RollingMedian", RollingMedian(xs, n), vec.RollingMedian(xs, n))
		eqSeries(t, "RollingQuantile", RollingQuantile(xs, n, 0.25), vec.RollingQuantile(xs, n, 0.25))
		eqSeries(t, "RollingPercentRank", RollingPercentRank(xs, n), vec.RollingPercentRank(xs, n))
		eqSeries(t, "RollingEMA", RollingEMA(xs, n), vec.RollingEMA(xs, n))
		eqSeries(t, "RollingRMA", RollingRMA(xs, n), vec.RollingRMA(xs, n))
	}
}

// rollingTestData builds a deterministic non-degenerate series for the equivalence checks.
func rollingTestData(n, seed int) []float64 {
	xs := make([]float64, n)
	v := float64(seed)
	for i := range xs {
		v += math.Sin(v) + 0.1
		xs[i] = v
	}
	return xs
}

func TestRollingToForwardersMatchAllocating(t *testing.T) {
	xs := rollingTestData(80, 3)
	dst := make([]float64, len(xs))
	const n = 8

	eqSeries(t, "RollingSumTo", RollingSumTo(dst, xs, n), RollingSum(xs, n))
	eqSeries(t, "RollingMeanTo", RollingMeanTo(dst, xs, n), RollingMean(xs, n))
	eqSeries(t, "RollingMaxTo", RollingMaxTo(dst, xs, n), RollingMax(xs, n))
	eqSeries(t, "RollingMinTo", RollingMinTo(dst, xs, n), RollingMin(xs, n))
	eqSeries(t, "RollingRangeTo", RollingRangeTo(dst, xs, n), RollingRange(xs, n))
	eqSeries(t, "RollingWMATo", RollingWMATo(dst, xs, n), RollingWMA(xs, n))

	// The To forms must return the destination, not a copy.
	if &RollingSumTo(dst, xs, 5)[0] != &dst[0] {
		t.Error("RollingSumTo did not return dst")
	}
}

// TestRollingForwarderNaNPolicy repeats the recovery property through the facade, since the
// counter-based NaN handling is the easiest thing for a forwarder to disturb.
func TestRollingForwarderNaNPolicy(t *testing.T) {
	xs := []float64{1, 2, 3, math.NaN(), 5, 6, 7}
	const n = 3

	for name, got := range map[string][]float64{
		"RollingSum":         RollingSum(xs, n),
		"RollingMax":         RollingMax(xs, n),
		"RollingMin":         RollingMin(xs, n),
		"RollingStdDev":      RollingStdDev(xs, n),
		"RollingMedian":      RollingMedian(xs, n),
		"RollingPercentRank": RollingPercentRank(xs, n),
	} {
		for _, i := range []int{3, 4, 5} {
			if !math.IsNaN(got[i]) {
				t.Errorf("%s[%d] = %v, want NaN", name, i, got[i])
			}
		}
		if math.IsNaN(got[6]) {
			t.Errorf("%s[6] = NaN, want a value after the NaN left the window", name)
		}
	}
}

// TestRollingForwarderCorrelationFixedPoint checks the value a caller would sanity-check against.
func TestRollingForwarderCorrelationFixedPoint(t *testing.T) {
	xs := rollingTestData(60, 4)
	got := RollingCorrelation(xs, xs, 10)
	for i, v := range got {
		if math.IsNaN(v) {
			continue
		}
		if math.Abs(v-1) > 1e-9 {
			t.Fatalf("rolling correlation with itself[%d] = %v, want 1", i, v)
		}
	}
}

func TestRollingForwardersPanicLikeVec(t *testing.T) {
	xs := rollingTestData(10, 5)
	short := make([]float64, 3)

	cases := []struct {
		name string
		fn   func()
	}{
		{"RollingSum period", func() { RollingSum(xs, 0) }},
		{"RollingMean period", func() { RollingMean(xs, -1) }},
		{"RollingMax period", func() { RollingMax(xs, 0) }},
		{"RollingRange period", func() { RollingRange(xs, 0) }},
		{"RollingWMA period", func() { RollingWMA(xs, 0) }},
		{"RollingVariance period", func() { RollingVariance(xs, 0) }},
		{"RollingQuantile q", func() { RollingQuantile(xs, 5, 2) }},
		{"RollingPercentRank period", func() { RollingPercentRank(xs, 0) }},
		{"RollingEMA period", func() { RollingEMA(xs, 0) }},
		{"RollingEMATo length", func() { RollingEMATo(short, xs, 5) }},
		{"RollingRMA period", func() { RollingRMA(xs, 0) }},
		{"RollingRMATo length", func() { RollingRMATo(short, xs, 5) }},
		{"RollingSumTo length", func() { RollingSumTo(short, xs, 5) }},
		{"RollingMaxTo length", func() { RollingMaxTo(short, xs, 5) }},
		{"RollingCorrelation period", func() { RollingCorrelation(xs, xs, 0) }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			defer func() {
				if recover() == nil {
					t.Errorf("%s did not panic", tc.name)
				}
			}()
			tc.fn()
		})
	}
}

func TestRollingForwardersEmptyInputs(t *testing.T) {
	if RollingSum(nil, 5) != nil || RollingMean(nil, 5) != nil || RollingMax(nil, 5) != nil ||
		RollingMin(nil, 5) != nil || RollingRange(nil, 5) != nil || RollingWMA(nil, 5) != nil {
		t.Error("empty rolling inputs should return nil")
	}
	if RollingVariance(nil, 5) != nil || RollingStdDev(nil, 5) != nil ||
		RollingMedian(nil, 5) != nil || RollingQuantile(nil, 5, 0.5) != nil ||
		RollingPercentRank(nil, 5) != nil || RollingEMA(nil, 5) != nil || RollingRMA(nil, 5) != nil {
		t.Error("empty statistical rolling inputs should return nil")
	}
	if RollingCorrelation(nil, nil, 5) != nil {
		t.Error("empty RollingCorrelation should return nil")
	}
}
