package ta

import (
	"math"
	"testing"

	"github.com/rj-raghavjoshi/numa/series"
)

// ---------------------------------------------------------------------------
// Tests for the extended moving-average family.
//
// The adaptive averages (KAMA, VIDYA, McGinley) are recursive, so a reference implementation written
// the same way would share any seeding mistake. Each is therefore checked against a *property* of
// its adaptation as well as against a reference: KAMA must reach its fastest constant on a straight
// line, VIDYA must freeze when momentum is absent, and McGinley must stay put on a constant series.
// ---------------------------------------------------------------------------

func refALMA(xs []float64, n int, offset, sigma float64) []float64 {
	out := allNaN(len(xs))
	if len(xs) < n {
		return out
	}
	weights := make([]float64, n)
	m := offset * float64(n-1)
	s := float64(n) / sigma
	var wsum float64
	for j := 0; j < n; j++ {
		d := float64(j) - m
		weights[j] = math.Exp(-(d * d) / (2 * s * s))
		wsum += weights[j]
	}
	for i := n - 1; i < len(xs); i++ {
		var acc float64
		for j := 0; j < n; j++ {
			acc += weights[j] * xs[i-n+1+j]
		}
		out[i] = acc / wsum
	}
	return out
}

func TestALMAMatchesReference(t *testing.T) {
	xs := synthClose(200, 9191)
	for _, n := range []int{5, 9, 20} {
		for _, tc := range [][2]float64{{0.85, 6}, {0.5, 6}, {0, 6}, {1, 6}, {0.85, 2}} {
			got := ALMA(xs, n, tc[0], tc[1])
			want := refALMA(xs, n, tc[0], tc[1])
			assertSeries(t, "ALMA", got, want, false)
		}
	}
}

// TestALMAOnALinearRamp pins the weight profile through its effect: on a straight line the average
// lands at the line's value at the weighted mean position, so the offset between the input and the
// output is exactly the profile's centre of mass.
func TestALMAOnALinearRamp(t *testing.T) {
	const size = 100
	xs := make([]float64, size)
	for i := range xs {
		xs[i] = float64(i)
	}
	for _, n := range []int{5, 10, 20} {
		for _, offset := range []float64{0, 0.5, 1} {
			got := ALMA(xs, n, offset, 6)
			// Recompute the centre of mass of the same weights.
			weights := make([]float64, n)
			m := offset * float64(n-1)
			s := float64(n) / 6.0
			var wsum, moment float64
			for j := 0; j < n; j++ {
				d := float64(j) - m
				weights[j] = math.Exp(-(d * d) / (2 * s * s))
				wsum += weights[j]
				moment += weights[j] * float64(j)
			}
			com := moment / wsum
			for i := n - 1; i < size; i++ {
				want := float64(i-n+1) + com
				if math.Abs(got[i]-want) > 1e-9 {
					t.Fatalf("n=%d offset=%v: ALMA[%d] = %v, want %v", n, offset, i, got[i], want)
				}
			}
		}
	}
	// And the centre of mass must actually move with the offset, or the test proves nothing.
	lo := ALMA(xs, 10, 0, 6)[size-1]
	hi := ALMA(xs, 10, 1, 6)[size-1]
	if hi <= lo {
		t.Fatalf("offset=1 gave %v and offset=0 gave %v; a higher offset must reduce lag", hi, lo)
	}
}

func TestALMAConstantSeries(t *testing.T) {
	const size = 60
	const c = 33.0
	xs := make([]float64, size)
	for i := range xs {
		xs[i] = c
	}
	got := ALMA(xs, 9, 0.85, 6)
	for i := 8; i < size; i++ {
		if math.Abs(got[i]-c) > 1e-9 {
			t.Fatalf("ALMA of a constant[%d] = %v, want %v", i, got[i], c)
		}
	}
}

func TestALMAWarmupAndPanics(t *testing.T) {
	xs := synthClose(50, 1)
	for _, n := range []int{3, 9, 20} {
		if got := FirstValid(ALMA(xs, n, 0.85, 6)); got != n-1 {
			t.Errorf("ALMA(n=%d) first valid = %d, want %d", n, got, n-1)
		}
	}
	if ALMA(nil, 5, 0.85, 6) != nil {
		t.Error("empty ALMA should be nil")
	}
	for _, tc := range []struct {
		name string
		fn   func()
	}{
		{"period", func() { ALMA(xs, 0, 0.85, 6) }},
		{"sigma", func() { ALMA(xs, 5, 0.85, 0) }},
		{"sigma negative", func() { ALMA(xs, 5, 0.85, -1) }},
		{"offset high", func() { ALMA(xs, 5, 1.5, 6) }},
		{"offset low", func() { ALMA(xs, 5, -0.1, 6) }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			defer func() {
				if recover() == nil {
					t.Errorf("ALMA %s did not panic", tc.name)
				}
			}()
			tc.fn()
		})
	}
}

func TestSWMAWeights(t *testing.T) {
	xs := synthClose(50, 2)
	got := SWMA(xs)
	if first := FirstValid(got); first != 3 {
		t.Fatalf("SWMA first valid = %d, want 3", first)
	}
	for i := 3; i < len(xs); i++ {
		want := (xs[i-3] + 2*xs[i-2] + 2*xs[i-1] + xs[i]) / 6
		if !closeOrNaN(got[i], want) {
			t.Fatalf("SWMA[%d] = %v, want %v", i, got[i], want)
		}
	}
	// The weights must sum to one, which a constant series confirms.
	const size = 20
	const c = 7.5
	flat := make([]float64, size)
	for i := range flat {
		flat[i] = c
	}
	flatOut := SWMA(flat)
	for i := 3; i < size; i++ {
		if math.Abs(flatOut[i]-c) > 1e-12 {
			t.Fatalf("SWMA of a constant[%d] = %v, want %v (weights must sum to 1)", i, flatOut[i], c)
		}
	}
	if SWMA(nil) != nil {
		t.Error("empty SWMA should be nil")
	}
}

func refKAMA(xs []float64, n, fast, slow int) []float64 {
	out := allNaN(len(xs))
	if len(xs) <= n {
		return out
	}
	fastSC := 2 / float64(fast+1)
	slowSC := 2 / float64(slow+1)
	prev := xs[n]
	out[n] = prev
	for i := n + 1; i < len(xs); i++ {
		var vol float64
		for j := i - n + 1; j <= i; j++ {
			vol += math.Abs(xs[j] - xs[j-1])
		}
		er := 0.0
		if vol != 0 {
			er = math.Abs(xs[i]-xs[i-n]) / vol
		}
		sc := er*(fastSC-slowSC) + slowSC
		sc *= sc
		prev += sc * (xs[i] - prev)
		out[i] = prev
	}
	return out
}

func TestKAMAMatchesReference(t *testing.T) {
	xs := synthClose(300, 3131)
	for _, n := range []int{5, 10} {
		got := KAMA(xs, n, 2, 30)
		want := refKAMA(xs, n, 2, 30)
		assertSeries(t, "KAMA", got, want, false)
	}
}

// TestKAMAIsFastestOnAStraightLine tests the adaptation directly: on a monotonic ramp the
// efficiency ratio is exactly 1, so the smoothing constant must be the fast one.
func TestKAMAIsFastestOnAStraightLine(t *testing.T) {
	const size = 200
	const n, fast, slow = 10, 2, 30
	xs := make([]float64, size)
	for i := range xs {
		xs[i] = float64(i)
	}
	got := KAMA(xs, n, fast, slow)

	fastSC := 2 / float64(fast+1)
	want := float64(n)
	for i := n + 1; i < size; i++ {
		want += fastSC * fastSC * (xs[i] - want)
		if math.Abs(got[i]-want) > 1e-6 {
			t.Fatalf("KAMA[%d] = %v, want the fast-constant recurrence %v", i, got[i], want)
		}
	}
}

// TestKAMAIsSlowestOnNoise: an oscillating series with no net progress has an efficiency ratio near
// zero, so the constant must approach the slow one and the average must barely move.
func TestKAMAIsSlowestOnNoise(t *testing.T) {
	const size = 400
	const n, fast, slow = 10, 2, 30
	xs := make([]float64, size)
	for i := range xs {
		if i%2 == 0 {
			xs[i] = 100
		} else {
			xs[i] = 101
		}
	}
	got := KAMA(xs, n, fast, slow)
	// The series returns to where it started every two bars, so the n-bar change is small relative
	// to the distance travelled.
	for i := n + 1; i < size; i++ {
		if math.Abs(got[i]-100.5) > 1 {
			t.Fatalf("KAMA[%d] = %v, want a value near the midpoint 100.5", i, got[i])
		}
	}
}

func TestKAMAWarmupAndPanics(t *testing.T) {
	xs := synthClose(100, 3)
	const n = 10
	if got := FirstValid(KAMA(xs, n, 2, 30)); got != n {
		t.Errorf("KAMA first valid = %d, want %d", got, n)
	}
	if KAMA(nil, 10, 2, 30) != nil {
		t.Error("empty KAMA should be nil")
	}
	for _, tc := range []struct {
		name string
		fn   func()
	}{
		{"period", func() { KAMA(xs, 0, 2, 30) }},
		{"fast >= slow", func() { KAMA(xs, 10, 30, 30) }},
		{"fast > slow", func() { KAMA(xs, 10, 40, 30) }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			defer func() {
				if recover() == nil {
					t.Errorf("KAMA %s did not panic", tc.name)
				}
			}()
			tc.fn()
		})
	}
}

func TestVIDYAMatchesReference(t *testing.T) {
	xs := synthClose(300, 4141)
	const n, cmoPeriod = 14, 9
	got := VIDYA(xs, n, cmoPeriod)

	cmo := CMO(xs, cmoPeriod)
	start := FirstValid(cmo)
	want := allNaN(len(xs))
	k := 2 / float64(n+1)
	prev := xs[start]
	want[start] = prev
	for i := start + 1; i < len(xs); i++ {
		alpha := k * math.Abs(cmo[i]) / 100
		prev = alpha*xs[i] + (1-alpha)*prev
		want[i] = prev
	}
	assertSeries(t, "VIDYA", got, want, false)
}

// TestVIDYAFreezesWithoutMomentum: a constant series has zero CMO, so alpha is zero and the average
// stays exactly at its seed.
func TestVIDYAFreezesWithoutMomentum(t *testing.T) {
	const size = 100
	const c = 55.0
	xs := make([]float64, size)
	for i := range xs {
		xs[i] = c
	}
	got := VIDYA(xs, 14, 9)
	start := FirstValid(got)
	if start < 0 {
		t.Fatal("VIDYA never became valid on a constant series")
	}
	for i := start; i < size; i++ {
		if math.Abs(got[i]-c) > 1e-12 {
			t.Fatalf("VIDYA[%d] = %v, want the frozen seed %v", i, got[i], c)
		}
	}
}

// TestVIDYAAlphaIsBounded checks that the adaptation stays a convex combination, which is what makes
// the average unable to overshoot on a single bar.
func TestVIDYAAlphaIsBounded(t *testing.T) {
	xs := synthClose(300, 5)
	const n, cmoPeriod = 14, 9
	got := VIDYA(xs, n, cmoPeriod)
	start := FirstValid(CMO(xs, cmoPeriod))
	lo, hi := math.Inf(1), math.Inf(-1)
	for i := start; i < len(xs); i++ {
		if math.IsNaN(got[i]) {
			continue
		}
		if got[i] < lo {
			lo = got[i]
		}
		if got[i] > hi {
			hi = got[i]
		}
	}
	inLo, inHi := minOf(xs), maxOf(xs)
	if lo < inLo-1e-9 || hi > inHi+1e-9 {
		t.Fatalf("VIDYA range [%v,%v] escapes the input range [%v,%v]", lo, hi, inLo, inHi)
	}
}

func minOf(xs []float64) float64 {
	m := math.Inf(1)
	for _, v := range xs {
		if v < m {
			m = v
		}
	}
	return m
}

func maxOf(xs []float64) float64 {
	m := math.Inf(-1)
	for _, v := range xs {
		if v > m {
			m = v
		}
	}
	return m
}

func TestVIDYAEmptyAndPanics(t *testing.T) {
	xs := synthClose(50, 6)
	if VIDYA(nil, 14, 9) != nil {
		t.Error("empty VIDYA should be nil")
	}
	for _, tc := range []struct {
		name string
		fn   func()
	}{
		{"period", func() { VIDYA(xs, 0, 9) }},
		{"cmoPeriod", func() { VIDYA(xs, 14, 0) }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			defer func() {
				if recover() == nil {
					t.Errorf("VIDYA %s did not panic", tc.name)
				}
			}()
			tc.fn()
		})
	}
}

func TestZLEMA(t *testing.T) {
	xs := synthClose(200, 7)
	for _, n := range []int{1, 5, 9, 20} {
		got := ZLEMA(xs, n)
		lag := (n - 1) / 2
		want := applyFrom(zlemaInput(xs, lag), lag, series.NewEMA(n))
		assertSeries(t, "ZLEMA", got, want, false)

		if first := FirstValid(got); first != lag+n-1 {
			t.Errorf("n=%d: ZLEMA first valid = %d, want lag+n-1 = %d", n, first, lag+n-1)
		}
	}
	// With n = 1 the lag is zero and ZLEMA is an EMA of the input with alpha 1, which is the input.
	got := ZLEMA(xs, 1)
	for i := range xs {
		if !closeOrNaN(got[i], xs[i]) {
			t.Fatalf("ZLEMA(n=1)[%d] = %v, want the input %v", i, got[i], xs[i])
		}
	}
	if ZLEMA(nil, 5) != nil {
		t.Error("empty ZLEMA should be nil")
	}
}

func zlemaInput(xs []float64, lag int) []float64 {
	out := allNaN(len(xs))
	for i := lag; i < len(xs); i++ {
		if xs[i] != xs[i] || xs[i-lag] != xs[i-lag] {
			continue
		}
		out[i] = 2*xs[i] - xs[i-lag]
	}
	return out
}

// TestT3WithZeroVolumeFactorIsTheThirdEMA pins the coefficient algebra: at v = 0 the weights are
// (0, 0, 0, 1), so only the third EMA survives.
func TestT3WithZeroVolumeFactorIsTheThirdEMA(t *testing.T) {
	xs := synthClose(300, 8)
	const n = 5
	got := T3(xs, n, 0)
	e1 := EMA(xs, n)
	e2 := applyFrom(e1, FirstValid(e1), series.NewEMA(n))
	e3 := applyFrom(e2, FirstValid(e2), series.NewEMA(n))
	assertSeries(t, "T3(v=0)", got, e3, false)
}

// TestT3CoefficientsSumToOne checks the algebraic identity that makes T3 a weighted *average* rather
// than an arbitrary combination: c1+c2+c3+c4 = 1 for every v.
func TestT3CoefficientsSumToOne(t *testing.T) {
	const size = 200
	const n = 5
	const c = 12.5
	flat := make([]float64, size)
	for i := range flat {
		flat[i] = c
	}
	for _, v := range []float64{0, 0.3, 0.7, 1} {
		got := T3(flat, n, v)
		for i := range flat {
			if math.IsNaN(got[i]) {
				continue
			}
			if math.Abs(got[i]-c) > 1e-9 {
				t.Fatalf("v=%v: T3 of a constant[%d] = %v, want %v (coefficients must sum to 1)", v, i, got[i], c)
			}
		}
	}
}

func TestT3WarmupAndPanics(t *testing.T) {
	xs := synthClose(200, 9)
	const n = 5
	if got := FirstValid(T3(xs, n, 0.7)); got != 6*(n-1) {
		t.Errorf("T3 first valid = %d, want 6(n-1) = %d", got, 6*(n-1))
	}
	if T3(nil, 5, 0.7) != nil {
		t.Error("empty T3 should be nil")
	}
	for _, tc := range []struct {
		name string
		fn   func()
	}{
		{"period", func() { T3(xs, 0, 0.7) }},
		{"v high", func() { T3(xs, 5, 1.5) }},
		{"v negative", func() { T3(xs, 5, -0.1) }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			defer func() {
				if recover() == nil {
					t.Errorf("T3 %s did not panic", tc.name)
				}
			}()
			tc.fn()
		})
	}
}

func refMcGinley(xs []float64, n int) []float64 {
	out := allNaN(len(xs))
	if len(xs) == 0 {
		return out
	}
	prev := xs[0]
	out[0] = prev
	for i := 1; i < len(xs); i++ {
		ratio := xs[i] / prev
		divisor := 0.6 * float64(n) * ratio * ratio * ratio * ratio
		prev += (xs[i] - prev) / divisor
		out[i] = prev
	}
	return out
}

func TestMcGinleyDynamic(t *testing.T) {
	xs := synthClose(200, 10)
	for _, n := range []int{5, 14, 20} {
		got := McGinleyDynamic(xs, n)
		assertSeries(t, "McGinleyDynamic", got, refMcGinley(xs, n), false)
		if got[0] != xs[0] {
			t.Errorf("n=%d: McGinleyDynamic[0] = %v, want the seed xs[0] = %v", n, got[0], xs[0])
		}
	}
	if McGinleyDynamic(nil, 14) != nil {
		t.Error("empty McGinleyDynamic should be nil")
	}
}

// TestMcGinleyDynamicConstantSeries: the ratio is 1, so the divisor is 0.6n and the update term is
// zero; the average must not drift.
func TestMcGinleyDynamicConstantSeries(t *testing.T) {
	const size = 80
	const c = 42.0
	xs := make([]float64, size)
	for i := range xs {
		xs[i] = c
	}
	got := McGinleyDynamic(xs, 14)
	for i := range xs {
		if math.Abs(got[i]-c) > 1e-12 {
			t.Fatalf("McGinleyDynamic[%d] = %v, want %v", i, got[i], c)
		}
	}
}

func TestAdaptiveIndicatorsEmptyAndPanics(t *testing.T) {
	xs := synthClose(30, 11)
	func() {
		defer func() {
			if recover() == nil {
				t.Error("McGinleyDynamic with a zero period did not panic")
			}
		}()
		McGinleyDynamic(xs, 0)
	}()
}
