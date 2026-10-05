package ta

import (
	"math"
	"testing"
)

// ---------------------------------------------------------------------------
// Tests for the correlation and realized-volatility indicators.
//
// Correlation has exact fixed points that are worth asserting directly: a series against
// itself is 1, against its negation is -1, and against a monotonic transform of itself is
// still 1 for the rank version but not necessarily for the linear one. Those three cases
// separate the estimators better than any tolerance comparison.
// ---------------------------------------------------------------------------

func refCorrelationCoefficient(xs, ys []float64, n int) []float64 {
	size := len(xs)
	out := allNaN(size)
	for i := n - 1; i < size; i++ {
		base := i - n + 1
		var mx, my float64
		for j := base; j <= i; j++ {
			mx += xs[j]
			my += ys[j]
		}
		mx /= float64(n)
		my /= float64(n)
		var sxy, sxx, syy float64
		for j := base; j <= i; j++ {
			dx := xs[j] - mx
			dy := ys[j] - my
			sxy += dx * dy
			sxx += dx * dx
			syy += dy * dy
		}
		if sxx == 0 || syy == 0 {
			continue
		}
		out[i] = sxy / math.Sqrt(sxx*syy)
	}
	return out
}

func refHistoricalVolatility(close []float64, n int, annual float64) []float64 {
	size := len(close)
	rets := logReturns(close)
	out := allNaN(size)
	for i := n; i < size; i++ {
		base := i - n + 1
		var mean float64
		for j := base; j <= i; j++ {
			mean += rets[j]
		}
		mean /= float64(n)
		var ss float64
		for j := base; j <= i; j++ {
			d := rets[j] - mean
			ss += d * d
		}
		out[i] = 100 * math.Sqrt(ss/float64(n)) * math.Sqrt(annual)
	}
	return out
}

func TestCorrelationCoefficientMatchesReference(t *testing.T) {
	for _, size := range []int{60, 200} {
		xs := synthClose(size, int64(size)+5151)
		ys := synthClose(size, int64(size)+6161)
		for _, n := range []int{5, 14, 30} {
			got := CorrelationCoefficient(xs, ys, n)
			want := refCorrelationCoefficient(xs, ys, n)
			assertSeries(t, "CorrelationCoefficient", got, want, false)
		}
	}
}

// TestCorrelationFixedPoints pins the two exact values: a series against itself is 1 and
// against its negation is -1.
func TestCorrelationFixedPoints(t *testing.T) {
	xs := synthClose(120, 7171)
	neg := make([]float64, len(xs))
	for i := range xs {
		neg[i] = -xs[i]
	}
	const n = 20

	self := CorrelationCoefficient(xs, xs, n)
	anti := CorrelationCoefficient(xs, neg, n)
	for i := n - 1; i < len(xs); i++ {
		if math.Abs(self[i]-1) > 1e-9 {
			t.Fatalf("correlation with itself[%d] = %v, want 1", i, self[i])
		}
		if math.Abs(anti[i]+1) > 1e-9 {
			t.Fatalf("correlation with the negation[%d] = %v, want -1", i, anti[i])
		}
	}
}

// TestCorrelationZeroVarianceIsNaN pins the undefined case. Reporting 0 would claim the
// series are uncorrelated, which is a different and false statement about a series that
// did not move.
func TestCorrelationZeroVarianceIsNaN(t *testing.T) {
	const size = 30
	flat := make([]float64, size)
	for i := range flat {
		flat[i] = 5
	}
	other := synthClose(size, 8181)

	got := CorrelationCoefficient(flat, other, 10)
	for i := 9; i < size; i++ {
		if !math.IsNaN(got[i]) {
			t.Fatalf("correlation with a flat series[%d] = %v, want NaN", i, got[i])
		}
	}
}

// TestCorrelationLogUsesReturns shows what the log-return version measures. With b = a^2
// the log returns of b are exactly twice those of a, so the return correlation is 1 while
// the level correlation is not -- the two series are monotonically but not linearly related.
func TestCorrelationLogUsesReturns(t *testing.T) {
	const size = 200
	a := synthClose(size, 8181)
	b := make([]float64, size)
	for i := range a {
		b[i] = a[i] * a[i]
	}
	const n = 20

	level := CorrelationCoefficient(a, b, n)
	rets := CorrelationLog(a, b, n)

	for i := n; i < size; i++ {
		if math.Abs(rets[i]-1) > 1e-9 {
			t.Fatalf("log-return correlation[%d] = %v, want 1 (returns are exactly proportional)", i, rets[i])
		}
		if level[i] >= 1 {
			t.Fatalf("level correlation[%d] = %v, want < 1 (a and a^2 are monotonic but not linear)", i, level[i])
		}
	}
}

// TestCorrelationLogConstantSeriesIsNaN pins the undefined case through the return layer:
// a series that never moves has zero-variance returns.
func TestCorrelationLogConstantSeriesIsNaN(t *testing.T) {
	const size = 60
	flat := make([]float64, size)
	for i := range flat {
		flat[i] = 10
	}
	got := CorrelationLog(flat, flat, 10)
	for i := 10; i < size; i++ {
		if !math.IsNaN(got[i]) {
			t.Fatalf("log correlation of a constant series[%d] = %v, want NaN", i, got[i])
		}
	}
}

// TestRankCorrelationIsMonotonicInvariant is the defining property: a non-linear but
// monotonic transform leaves the rank correlation at 1 where the linear one drops.
func TestRankCorrelationIsMonotonicInvariant(t *testing.T) {
	const size = 200
	xs := synthClose(size, 9191)
	// A strictly increasing transform of xs: cubing preserves order but changes spacing.
	cubed := make([]float64, size)
	for i := range xs {
		cubed[i] = xs[i] * xs[i] * xs[i]
	}
	const n = 30

	rank := RankCorrelation(xs, cubed, n)
	pearson := CorrelationCoefficient(xs, cubed, n)
	for i := n - 1; i < size; i++ {
		if math.Abs(rank[i]-1) > 1e-9 {
			t.Fatalf("rank correlation with a monotonic transform[%d] = %v, want 1", i, rank[i])
		}
		// The contrast is the point: cubing preserves order exactly, so the rank
		// correlation is 1, but it is not a linear transform, so Pearson is close to 1
		// without being it. This assertion is what distinguishes the two estimators; a
		// test that only checked the rank would pass for a Pearson implementation too.
		if pearson[i] >= 1 {
			t.Fatalf("Pearson correlation with a cubic transform[%d] = %v, want < 1", i, pearson[i])
		}
		if pearson[i] < 0.9 {
			t.Fatalf("Pearson correlation with a cubic transform[%d] = %v, unexpectedly low for a monotonic map", i, pearson[i])
		}
	}
}

func TestHistoricalVolatilityMatchesReference(t *testing.T) {
	for _, size := range []int{60, 200} {
		close := synthClose(size, int64(size)+2222)
		for _, n := range []int{5, 20} {
			got := HistoricalVolatility(close, n, 252)
			want := refHistoricalVolatility(close, n, 252)
			assertSeries(t, "HistoricalVolatility", got, want, false)
		}
	}
}

// TestHistoricalVolatilityConstantSeriesIsZero pins the degenerate case: no price movement
// means no volatility.
func TestHistoricalVolatilityConstantSeriesIsZero(t *testing.T) {
	const size = 60
	const c = 50.0
	xs := make([]float64, size)
	for i := range xs {
		xs[i] = c
	}
	got := HistoricalVolatility(xs, 10, 252)
	for i := 10; i < size; i++ {
		if math.Abs(got[i]) > 1e-12 {
			t.Fatalf("volatility of a constant series[%d] = %v, want 0", i, got[i])
		}
	}
}

// TestHistoricalVolatilityAnnualizationScalesByRoot checks the annualization convention:
// quadrupling the bar count doubles the result, since it enters under a square root.
func TestHistoricalVolatilityAnnualizationScalesByRoot(t *testing.T) {
	close := synthClose(120, 3333)
	base := HistoricalVolatility(close, 20, 100)
	quad := HistoricalVolatility(close, 20, 400)
	for i := 20; i < len(close); i++ {
		if math.Abs(quad[i]-2*base[i]) > 1e-9 {
			t.Fatalf("annualization[%d]: %v with 400 != 2x %v with 100", i, quad[i], base[i])
		}
	}
}

func TestVolatilityOHLC(t *testing.T) {
	const size = 120
	open, high, low, close, _ := synthOHLCV(size, 4444)

	got := VolatilityOHLC(open, high, low, close, 20, 252)
	if first := FirstValid(got); first != 19 {
		t.Errorf("VolatilityOHLC first valid = %d, want 19", first)
	}
	for i, v := range got {
		if math.IsNaN(v) {
			continue
		}
		if v < 0 {
			t.Fatalf("VolatilityOHLC[%d] = %v, want non-negative", i, v)
		}
	}
}

// TestVolatilityOHLCFlatBarIsZero: a bar with no range and no open-to-close move gives a
// per-bar estimate of exactly zero, so the volatility is zero.
func TestVolatilityOHLCFlatBarIsZero(t *testing.T) {
	const size = 40
	const p = 25.0
	o := make([]float64, size)
	h := make([]float64, size)
	l := make([]float64, size)
	c := make([]float64, size)
	for i := range o {
		o[i], h[i], l[i], c[i] = p, p, p, p
	}
	got := VolatilityOHLC(o, h, l, c, 10, 252)
	for i := 9; i < size; i++ {
		if math.Abs(got[i]) > 1e-12 {
			t.Fatalf("VolatilityOHLC on a flat bar[%d] = %v, want 0", i, got[i])
		}
	}
}

func TestCorrelationEmptyAndPanics(t *testing.T) {
	if CorrelationCoefficient(nil, nil, 10) != nil {
		t.Error("empty CorrelationCoefficient should be nil")
	}
	if CorrelationLog(nil, nil, 10) != nil {
		t.Error("empty CorrelationLog should be nil")
	}
	if RankCorrelation(nil, nil, 10) != nil {
		t.Error("empty RankCorrelation should be nil")
	}
	if HistoricalVolatility(nil, 10, 252) != nil {
		t.Error("empty HistoricalVolatility should be nil")
	}
	if VolatilityOHLC(nil, nil, nil, nil, 10, 252) != nil {
		t.Error("empty VolatilityOHLC should be nil")
	}

	short := make([]float64, 2)
	long := make([]float64, 3)
	for _, tc := range []struct {
		name string
		fn   func()
	}{
		{"CorrelationCoefficient period", func() { CorrelationCoefficient(long, long, 0) }},
		{"CorrelationCoefficient length", func() { CorrelationCoefficient(short, long, 5) }},
		{"CorrelationLog period", func() { CorrelationLog(long, long, -1) }},
		{"RankCorrelation period", func() { RankCorrelation(long, long, 0) }},
		{"HistoricalVolatility period", func() { HistoricalVolatility(long, 0, 252) }},
		{"VolatilityOHLC period", func() { VolatilityOHLC(long, long, long, long, 0, 252) }},
		{"VolatilityOHLC length", func() { VolatilityOHLC(short, short, short, long, 5, 252) }},
	} {
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
