package numa

import (
	"math"
	"testing"

	"github.com/rj-raghavjoshi/numa/series"
	"github.com/rj-raghavjoshi/numa/vec"
)

// ---------------------------------------------------------------------------
// Cross-package equivalence: the batch kernels against the streaming rollers.
//
// design.md claims that `vec`'s rolling kernels and `series`' rollers are "the same mathematics
// arranged for different access patterns". That claim is load-bearing now that [SMA] runs on the
// batch form, because a caller who switches between a whole-slice computation and an incremental one
// expects the same numbers.
//
// The test lives in the root package because it needs both packages, and `vec` deliberately does not
// import `series`. That is also why nothing in either package's own tests could catch a divergence.
// ---------------------------------------------------------------------------

// equivalents builds a deterministic series with a NaN in the middle, so the tests cover the NaN
// policy of both implementations as well as their arithmetic.
func equivSeries(n int, seed float64) []float64 {
	xs := make([]float64, n)
	v := seed
	for i := range xs {
		v += math.Sin(v) * 0.7
		xs[i] = v
	}
	if n > 4 {
		xs[n/2] = math.NaN()
	}
	return xs
}

func TestRollingSumEqualsStreamingSum(t *testing.T) {
	for _, size := range []int{1, 2, 3, 5, 8, 13, 20, 33, 64, 100, 250} {
		xs := equivSeries(size, float64(size)+1)
		for _, n := range []int{1, 2, 3, 5, 8, 13, 20, 33, 64} {
			batch := vec.RollingSum(xs, n)
			stream := series.Apply(xs, series.NewSum(n))
			compareEquiv(t, "RollingSum vs series.Sum", size, n, batch, stream)
		}
	}
}

func TestRollingMeanEqualsStreamingSMA(t *testing.T) {
	for _, size := range []int{1, 2, 3, 5, 8, 13, 20, 33, 64, 100, 250} {
		xs := equivSeries(size, float64(size)+2)
		for _, n := range []int{1, 2, 3, 5, 8, 13, 20, 33, 64} {
			batch := vec.RollingMean(xs, n)
			stream := series.Apply(xs, series.NewSMA(n))
			compareEquiv(t, "RollingMean vs series.SMA", size, n, batch, stream)
		}
	}
}

// compareEquiv checks the two series agree, reporting the first divergence with enough context to
// tell a NaN-policy difference from an arithmetic one.
func compareEquiv(t *testing.T, name string, size, n int, batch, stream []float64) {
	t.Helper()
	if len(batch) != len(stream) {
		t.Fatalf("%s: size=%d n=%d: lengths %d and %d", name, size, n, len(batch), len(stream))
	}
	for i := range batch {
		a, b := batch[i], stream[i]
		aNaN, bNaN := math.IsNaN(a), math.IsNaN(b)
		if aNaN || bNaN {
			if aNaN != bNaN {
				t.Fatalf("%s: size=%d n=%d: index %d is %v in the batch form and %v in the streaming one",
					name, size, n, i, a, b)
			}
			continue
		}
		if a == b {
			continue
		}
		// A last-ulp difference is permitted: the two accumulate in the same order but divide by
		// the window differently, and this repository does not promise bit reproducibility.
		scale := math.Max(1, math.Abs(b))
		if math.Abs(a-b) > 1e-12*scale {
			t.Fatalf("%s: size=%d n=%d: index %d is %v in the batch form and %v in the streaming one",
				name, size, n, i, a, b)
		}
	}
}

// TestRollingSumAndStreamingSumAgreeUnderEviction checks the case the compensation exists for: a
// large value leaving the window must not corrupt the small ones, in either implementation.
func TestRollingSumAndStreamingSumAgreeUnderEviction(t *testing.T) {
	xs := []float64{1e16, 1, 1, 1, 1, 1, 1, 1}
	for _, n := range []int{2, 3, 5} {
		batch := vec.RollingSum(xs, n)
		stream := series.Apply(xs, series.NewSum(n))
		compareEquiv(t, "large-offset eviction", len(xs), n, batch, stream)
	}
	// And the eviction case must actually be handled: a window of the small values is exactly 2.
	got := vec.RollingSum(xs, 2)
	if got[2] != 2 {
		t.Fatalf("rolling sum after the large value left = %v, want exactly 2", got[2])
	}
}

// TestSMAMatchesTheStreamingRoller is the end-to-end version: the exported indicator, which now uses
// the batch kernel, must still agree with the roller it used to call.
func TestSMAMatchesTheStreamingRoller(t *testing.T) {
	for _, size := range []int{5, 20, 64, 200} {
		xs := equivSeries(size, float64(size)+3)
		for _, n := range []int{1, 3, 8, 20} {
			compareEquiv(t, "numa.SMA", size, n, SMA(xs, n), series.Apply(xs, series.NewSMA(n)))
		}
	}
}

// TestCMOAndUltimateOscillatorUnchangedByTheBatchRewrite is a value-level guard: the two indicators
// whose summing was rewritten must produce the numbers their definitions give, computed here from
// scratch without either summation.
func TestCMOAndUltimateOscillatorUnchangedByTheBatchRewrite(t *testing.T) {
	const size = 120
	close := equivSeries(size, 11)
	high := make([]float64, size)
	low := make([]float64, size)
	for i := range close {
		high[i] = close[i] + 1.5
		low[i] = close[i] - 1.5
	}

	const n = 9
	got := CMO(close, n)
	for i := n; i < size; i++ {
		var g, l float64
		for j := i - n + 1; j <= i; j++ {
			c := close[j] - close[j-1]
			switch {
			case c > 0:
				g += c
			case c < 0:
				l -= c
			}
		}
		want := 0.0
		if g+l != 0 {
			want = 100 * (g - l) / (g + l)
		}
		if math.Abs(got[i]-want) > 1e-9 {
			t.Fatalf("CMO[%d] = %v, want %v", i, got[i], want)
		}
	}

	const long = 20
	uo := UltimateOscillator(high, low, close, 7, 14, long)
	for i := long; i < size; i++ {
		var b1, t1, b2, t2, b3, t3 float64
		for j := i - 7 + 1; j <= i; j++ {
			pc := close[j-1]
			b1 += close[j] - math.Min(low[j], pc)
			t1 += math.Max(high[j], pc) - math.Min(low[j], pc)
		}
		for j := i - 14 + 1; j <= i; j++ {
			pc := close[j-1]
			b2 += close[j] - math.Min(low[j], pc)
			t2 += math.Max(high[j], pc) - math.Min(low[j], pc)
		}
		for j := i - long + 1; j <= i; j++ {
			pc := close[j-1]
			b3 += close[j] - math.Min(low[j], pc)
			t3 += math.Max(high[j], pc) - math.Min(low[j], pc)
		}
		r := func(num, den float64) float64 {
			if den == 0 {
				return 0
			}
			return num / den
		}
		want := 100 * (4*r(b1, t1) + 2*r(b2, t2) + r(b3, t3)) / 7
		if math.Abs(uo[i]-want) > 1e-9 {
			t.Fatalf("UltimateOscillator[%d] = %v, want %v", i, uo[i], want)
		}
	}
}

// TestRollingEMAEqualsStreamingEMA and the RMA form below are the equivalence checks for the
// exponential smoothers. They matter more than the sum's did, because the seed decides every later
// value: an implementation that seeded with the first observation instead of the SMA would converge
// to the same place and disagree for a long time on the way, and a test that only looked at the tail
// would not notice.
func TestRollingEMAEqualsStreamingEMA(t *testing.T) {
	for _, size := range []int{1, 2, 3, 5, 8, 13, 20, 33, 64, 100} {
		xs := equivSeries(size, float64(size)+4)
		for _, n := range []int{1, 2, 3, 5, 8, 13, 20, 33} {
			batch := vec.RollingEMA(xs, n)
			stream := series.Apply(xs, series.NewEMA(n))
			compareEquiv(t, "RollingEMA vs series.EMA", size, n, batch, stream)
		}
	}
}

func TestRollingRMAEqualsStreamingRMA(t *testing.T) {
	for _, size := range []int{1, 2, 3, 5, 8, 13, 20, 33, 64, 100} {
		xs := equivSeries(size, float64(size)+5)
		for _, n := range []int{1, 2, 3, 5, 8, 13, 20, 33} {
			batch := vec.RollingRMA(xs, n)
			stream := series.Apply(xs, series.NewRMA(n))
			compareEquiv(t, "RollingRMA vs series.RMA", size, n, batch, stream)
		}
	}
}

// TestEMASeedIsTheSMANotTheFirstValue pins the seeding convention, which is the one thing the two
// implementations have to agree on and the one thing a reader is most likely to assume differently.
func TestEMASeedIsTheSMANotTheFirstValue(t *testing.T) {
	xs := []float64{10, 20, 30, 40, 50}
	got := vec.RollingEMA(xs, 3)
	if math.Abs(got[2]-(10+20+30)/3.0) > 1e-12 {
		t.Fatalf("EMA[2] = %v, want the SMA of the first three values 20", got[2])
	}
	if !math.IsNaN(got[0]) || !math.IsNaN(got[1]) {
		t.Errorf("the warm-up must be NaN, not a value: got %v %v", got[0], got[1])
	}
	// And the seed must be the SMA rather than the first observation, which would give 10.
	if math.Abs(got[2]-10) < 1e-12 {
		t.Error("the EMA seeded with the first value instead of the SMA")
	}
}
