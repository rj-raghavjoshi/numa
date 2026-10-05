package vec

import (
	"math"
	"sort"
	"testing"
)

// ---------------------------------------------------------------------------
// Tests for the sliding-window order statistics.
//
// The Fenwick structure compresses values to ranks, so its risky paths are the ones a continuous
// random series never reaches: many *equal* values (which collapse to one rank), a window with a
// single distinct value, and an input where every value is NaN. Those are tested explicitly rather
// than left to the random cases.
//
// The quantile kernels are also checked against the rescan they replaced, which is still available
// as [Quantile] over a copied window and is therefore an independent implementation.
// ---------------------------------------------------------------------------

// refRollingPercentRank rescans the window, which is the form the incremental kernel replaces.
func refRollingPercentRank(xs []float64, n int) []float64 {
	out := make([]float64, len(xs))
	for i := range out {
		out[i] = math.NaN()
	}
	for i := n - 1; i < len(xs); i++ {
		cur := xs[i]
		if cur != cur {
			continue
		}
		less := 0
		bad := false
		for j := i - n + 1; j <= i; j++ {
			v := xs[j]
			if v != v {
				bad = true
				break
			}
			if v < cur {
				less++
			}
		}
		if !bad {
			out[i] = 100 * float64(less) / float64(n)
		}
	}
	return out
}

// refRollingQuantileRescan computes the quantile by sorting each window, which is what
// [Quantile] does and what the incremental kernel has to reproduce.
func refRollingQuantileRescan(xs []float64, n int, q float64) []float64 {
	out := make([]float64, len(xs))
	for i := range out {
		out[i] = math.NaN()
	}
	for i := n - 1; i < len(xs); i++ {
		buf := make([]float64, n)
		bad := false
		for j := 0; j < n; j++ {
			v := xs[i-n+1+j]
			if v != v {
				bad = true
				break
			}
			buf[j] = v
		}
		if bad {
			continue
		}
		sort.Float64s(buf)
		h := float64(n-1) * q
		lo := int(math.Floor(h))
		hi := int(math.Ceil(h))
		out[i] = buf[lo] + (h-float64(lo))*(buf[hi]-buf[lo])
	}
	return out
}

func TestRollingPercentRankMatchesRescan(t *testing.T) {
	for _, size := range rollingLengths() {
		if size == 0 {
			continue
		}
		xs := randomSlice(size, int64(size)+21)
		for _, n := range rollingWindows() {
			got := RollingPercentRank(xs, n)
			want := refRollingPercentRank(xs, n)
			for i := range want {
				if !closeOrNaNBoth(got[i], want[i]) {
					t.Fatalf("size=%d n=%d: RollingPercentRank[%d] = %v, want %v", size, n, i, got[i], want[i])
				}
			}
		}
	}
}

// TestRollingPercentRankManyDuplicates exercises the compression's tie handling. With a six-value
// alphabet a 20-bar window holds very few distinct ranks, and a rank-vs-count mistake would show up
// as an off-by-one in the percentage rather than as an obviously wrong number.
func TestRollingPercentRankManyDuplicates(t *testing.T) {
	const size = 300
	xs := make([]float64, size)
	for i := range xs {
		xs[i] = float64(i % 6)
	}
	for _, n := range []int{2, 5, 20, 100} {
		got := RollingPercentRank(xs, n)
		want := refRollingPercentRank(xs, n)
		for i := range want {
			if !closeOrNaNBoth(got[i], want[i]) {
				t.Fatalf("n=%d: RollingPercentRank[%d] = %v, want %v", n, i, got[i], want[i])
			}
		}
	}
}

// TestOrderStatTiesAreStrictlyLess pins the tie rule directly: equal values are not less than each
// other, so a window of one repeated value ranks at zero rather than at the window size.
func TestOrderStatTiesAreStrictlyLess(t *testing.T) {
	const c = 7.0
	xs := make([]float64, 50)
	for i := range xs {
		xs[i] = c
	}
	got := RollingPercentRank(xs, 10)
	for i, v := range got {
		if math.IsNaN(v) {
			continue
		}
		if v != 0 {
			t.Fatalf("PercentRank of a constant[%d] = %v, want 0", i, v)
		}
	}
	// And the median of a constant window is the constant, not an interpolated value.
	med := RollingMedian(xs, 10)
	for i, v := range med {
		if math.IsNaN(v) {
			continue
		}
		if v != c {
			t.Fatalf("median of a constant[%d] = %v, want %v", i, v, c)
		}
	}
}

func TestRollingQuantileMatchesRescan(t *testing.T) {
	for _, size := range []int{1, 2, 3, 5, 8, 13, 20, 33, 64, 100} {
		xs := randomSlice(size, int64(size)+22)
		for _, n := range []int{1, 2, 3, 4, 5, 7, 11, 20, 33} {
			for _, q := range []float64{0, 0.1, 0.25, 0.5, 0.75, 0.9, 1} {
				got := RollingQuantile(xs, n, q)
				want := refRollingQuantileRescan(xs, n, q)
				for i := range want {
					if !closeOrNaNBoth(got[i], want[i]) {
						t.Fatalf("size=%d n=%d q=%v: [%d] = %v, want %v", size, n, q, i, got[i], want[i])
					}
				}
			}
		}
	}
}

// TestRollingOrderStatsAllNaN covers the path where the compressed domain is empty, so there is no
// rank to insert and the tree must never be descended.
func TestRollingOrderStatsAllNaN(t *testing.T) {
	xs := make([]float64, 40)
	for i := range xs {
		xs[i] = math.NaN()
	}
	for name, got := range map[string][]float64{
		"RollingMedian":      RollingMedian(xs, 5),
		"RollingQuantile":    RollingQuantile(xs, 5, 0.25),
		"RollingPercentRank": RollingPercentRank(xs, 5),
	} {
		for i, v := range got {
			if !math.IsNaN(v) {
				t.Fatalf("%s[%d] = %v, want NaN for an all-NaN input", name, i, v)
			}
		}
	}
}

// TestRollingOrderStatsNaNsRecover repeats the package-wide NaN policy for the new kernels: a NaN
// affects exactly the windows containing it.
func TestRollingOrderStatsNaNsRecover(t *testing.T) {
	const n = 3
	xs := []float64{5, 1, 9, math.NaN(), 2, 7, 3}

	for name, got := range map[string][]float64{
		"RollingMedian":      RollingMedian(xs, n),
		"RollingPercentRank": RollingPercentRank(xs, n),
	} {
		for _, i := range []int{3, 4, 5} {
			if !math.IsNaN(got[i]) {
				t.Errorf("%s[%d] = %v, want NaN (window contains the NaN)", name, i, got[i])
			}
		}
		if math.IsNaN(got[6]) {
			t.Errorf("%s[6] = NaN, want a value after the NaN left the window", name)
		}
		if math.IsNaN(got[2]) {
			t.Errorf("%s[2] = NaN, want a value before the NaN arrived", name)
		}
	}
}

func TestRollingOrderStatsWindowLargerThanInput(t *testing.T) {
	xs := []float64{3, 1, 2}
	for name, got := range map[string][]float64{
		"RollingMedian":      RollingMedian(xs, 10),
		"RollingQuantile":    RollingQuantile(xs, 10, 0.5),
		"RollingPercentRank": RollingPercentRank(xs, 10),
	} {
		for i, v := range got {
			if !math.IsNaN(v) {
				t.Fatalf("%s[%d] = %v, want NaN with a window larger than the input", name, i, v)
			}
		}
	}
}

func TestRollingOrderStatsEmptyAndPanics(t *testing.T) {
	if RollingMedian(nil, 5) != nil || RollingQuantile(nil, 5, 0.5) != nil ||
		RollingPercentRank(nil, 5) != nil {
		t.Error("empty inputs should return nil")
	}

	xs := randomSlice(20, 23)
	for _, n := range []int{0, -1} {
		func() {
			defer func() {
				if recover() == nil {
					t.Errorf("RollingPercentRank period %d did not panic", n)
				}
			}()
			RollingPercentRank(xs, n)
		}()
	}
	func() {
		defer func() {
			if recover() == nil {
				t.Error("RollingQuantile with q > 1 did not panic")
			}
		}()
		RollingQuantile(xs, 5, 1.5)
	}()
}

// TestRollingMedianAgreesWithRollingQuantile checks the two entry points share one implementation,
// which is what lets the median inherit the incremental kernel's cost.
func TestRollingMedianAgreesWithRollingQuantile(t *testing.T) {
	xs := randomSlice(200, 24)
	for _, n := range []int{1, 2, 5, 20} {
		a := RollingMedian(xs, n)
		b := RollingQuantile(xs, n, 0.5)
		for i := range a {
			if !sameOrNaNBoth(a[i], b[i]) {
				t.Fatalf("n=%d: median[%d] = %v, quantile 0.5 = %v", n, i, a[i], b[i])
			}
		}
	}
}

// TestOrderStatKthIsOrdered walks every k for a fixed window, which is the only way to be sure the
// Fenwick descent returns the k-th order statistic for every k rather than only for the median.
func TestOrderStatKthIsOrdered(t *testing.T) {
	xs := randomSlice(400, 25)
	const window = 32
	os := newOrderStat(xs)
	// Fill the window at position 100.
	for i := 100 - window + 1; i <= 100; i++ {
		os.add(int(os.rankOf(xs[i])), 1)
	}
	var got []float64
	for k := int32(1); k <= window; k++ {
		got = append(got, os.values[os.kth(k)-1])
	}
	want := append([]float64(nil), xs[100-window+1:101]...)
	sort.Float64s(want)
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("k-th order statistic %d = %v, want %v", i, got[i], want[i])
		}
	}
}
