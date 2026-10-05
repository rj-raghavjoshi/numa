package vec

import (
	"math"
	"math/rand"
	"sort"
	"testing"
)

// ---------------------------------------------------------------------------
// Tests for order statistics.
//
// Two classes of bug matter here, and they need different tests:
//
//   - A wrong answer from the selection logic itself. Caught by comparing against a
//     sort-based reference at every boundary length, plus larger random inputs
//     where quickselect actually partitions rather than falling into insertion sort.
//   - A wrong interpolation at the edges (q=0, q=1, even length, single element).
//     Caught by hand-computed values, because a reference that shares the same
//     interpolation bug would agree with it.
//
// The invariant that xs is never modified is checked explicitly: quickselect works
// in place, so a missing copy would silently reorder the caller's data.
// ---------------------------------------------------------------------------

func refQuantile(xs []float64, q float64) float64 {
	buf := append([]float64(nil), xs...)
	sort.Float64s(buf)
	n := len(buf)
	h := float64(n-1) * q
	lo := int(math.Floor(h))
	hi := int(math.Ceil(h))
	return buf[lo] + (h-float64(lo))*(buf[hi]-buf[lo])
}

func refRank(xs []float64) []float64 {
	out := make([]float64, len(xs))
	for i, x := range xs {
		if x != x {
			out[i] = math.NaN()
			continue
		}
		less, equal := 0, 0
		for _, y := range xs {
			if y != y {
				continue
			}
			switch {
			case y < x:
				less++
			case y == x:
				equal++
			}
		}
		out[i] = float64(less) + (float64(equal)+1)/2
	}
	return out
}

func TestMedianKnownValues(t *testing.T) {
	cases := []struct {
		xs   []float64
		want float64
	}{
		{[]float64{5}, 5},
		{[]float64{3, 1}, 2},
		{[]float64{3, 1, 2}, 2},
		{[]float64{4, 1, 3, 2}, 2.5},
		{[]float64{10, 20, 20, 40, 50}, 20},
		// Unsorted input, to catch an implementation that forgets to order first.
		{[]float64{9, -1, 4, 4, 100, -1}, 4},
	}
	for _, tc := range cases {
		if got := Median(tc.xs); got != tc.want {
			t.Errorf("Median(%v) = %v, want %v", tc.xs, got, tc.want)
		}
	}
}

func TestQuantileKnownValues(t *testing.T) {
	xs := []float64{1, 2, 3, 4}

	cases := []struct {
		q    float64
		want float64
	}{
		{0, 1},
		{0.25, 1.75},
		{0.5, 2.5},
		{0.75, 3.25},
		{1, 4},
	}
	for _, tc := range cases {
		if got := Quantile(xs, tc.q); math.Abs(got-tc.want) > 1e-15 {
			t.Errorf("Quantile(%v, %v) = %v, want %v", xs, tc.q, got, tc.want)
		}
	}

	// A single element is every quantile of itself.
	for _, q := range []float64{0, 0.3, 0.5, 0.9, 1} {
		if got := Quantile([]float64{7}, q); got != 7 {
			t.Errorf("Quantile(single, %v) = %v, want 7", q, got)
		}
	}
}

// TestQuantileMatchesSortedReference checks the quickselect path against a
// sort-based reference. The larger sizes are important: below 12 elements
// selectK delegates to insertion sort, so a bug in partitioning would go unnoticed
// if only boundary lengths were tested.
func TestQuantileMatchesSortedReference(t *testing.T) {
	qs := []float64{0, 0.01, 0.1, 0.25, 0.333, 0.5, 0.666, 0.75, 0.9, 0.99, 1}

	sizes := append([]int{}, boundaryLengths()...)
	sizes = append(sizes, 12, 13, 40, 41, 100, 257, 1000)

	for _, n := range sizes {
		if n == 0 {
			continue
		}
		rng := rand.New(rand.NewSource(int64(n)))
		xs := make([]float64, n)
		for i := range xs {
			xs[i] = rng.NormFloat64()
		}
		for _, q := range qs {
			want := refQuantile(xs, q)
			got := Quantile(xs, q)
			if math.Abs(got-want) > 1e-9*math.Max(1, math.Abs(want)) {
				t.Fatalf("n=%d q=%v: Quantile = %v, want %v", n, q, got, want)
			}
		}
	}
}

// TestOrderStatsDoNotModifyInput is the specific check against forgetting the copy:
// quickselect reorders its working buffer, so if that buffer is the caller's slice
// the caller's data is silently permuted.
func TestOrderStatsDoNotModifyInput(t *testing.T) {
	xs := []float64{5, 3, 9, 1, 7, 2, 8}
	orig := append([]float64(nil), xs...)

	Median(xs)
	Quantile(xs, 0.25)
	Percentile(xs, 90)
	MAD(xs)
	SortCopy(xs)

	for i := range orig {
		if xs[i] != orig[i] {
			t.Fatalf("input modified at %d: %v, want %v", i, xs[i], orig[i])
		}
	}
}

func TestPercentile(t *testing.T) {
	xs := []float64{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}
	if got := Percentile(xs, 50); got != 5.5 {
		t.Errorf("Percentile(50) = %v, want 5.5", got)
	}
	if got := Percentile(xs, 0); got != 1 {
		t.Errorf("Percentile(0) = %v, want 1", got)
	}
	if got := Percentile(xs, 100); got != 10 {
		t.Errorf("Percentile(100) = %v, want 10", got)
	}
}

// TestOrderStatsNaNPolicy pins that any NaN makes the result NaN rather than being
// silently skipped, which would report a plausible statistic computed from a
// corrupted series.
func TestOrderStatsNaNPolicy(t *testing.T) {
	nan := math.NaN()
	for pos := 0; pos < 4; pos++ {
		xs := []float64{1, 2, 3, 4}
		xs[pos] = nan
		if got := Median(xs); !math.IsNaN(got) {
			t.Errorf("Median with NaN at %d = %v, want NaN", pos, got)
		}
		if got := Quantile(xs, 0.5); !math.IsNaN(got) {
			t.Errorf("Quantile with NaN at %d = %v, want NaN", pos, got)
		}
		if got := MAD(xs); !math.IsNaN(got) {
			t.Errorf("MAD with NaN at %d = %v, want NaN", pos, got)
		}
	}
	if !math.IsNaN(Median(nil)) {
		t.Error("Median(nil) should be NaN")
	}
	if !math.IsNaN(Quantile(nil, 0.5)) {
		t.Error("Quantile(nil, 0.5) should be NaN")
	}
}

// TestMADKnownValue checks the definition directly, including that the deviation
// median is taken over absolute deviations.
func TestMADKnownValue(t *testing.T) {
	// median of {1,1,2,2,4,6,9} is 2; deviations {1,1,0,0,2,4,7}; their median is 1.
	xs := []float64{1, 1, 2, 2, 4, 6, 9}
	if got := MAD(xs); got != 1 {
		t.Errorf("MAD(%v) = %v, want 1", xs, got)
	}

	// A single outlier moves the MAD not at all, which is the point of the measure.
	withOutlier := append(append([]float64(nil), xs...), 1000)
	if got := MAD(withOutlier); got != 2 {
		t.Errorf("MAD with outlier = %v, want 2", got)
	}

	if !math.IsNaN(MAD([]float64{math.Inf(1), math.Inf(1)})) {
		t.Error("MAD over infinities should be NaN")
	}
}

func TestRankAverageTies(t *testing.T) {
	xs := []float64{10, 20, 20, 30}
	got := Rank(xs)
	want := []float64{1, 2.5, 2.5, 4}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("Rank[%d] = %v, want %v", i, got[i], want[i])
		}
	}
}

func TestRankMatchesReference(t *testing.T) {
	sizes := append([]int{}, boundaryLengths()...)
	sizes = append(sizes, 50, 200)

	for _, n := range sizes {
		if n == 0 {
			continue
		}
		rng := rand.New(rand.NewSource(int64(n) + 7))
		xs := make([]float64, n)
		for i := range xs {
			// A small range of integers guarantees many ties, which is the case
			// the average-rank rule exists for.
			xs[i] = float64(rng.Intn(7))
		}
		got := Rank(xs)
		want := refRank(xs)
		for i := range want {
			if got[i] != want[i] {
				t.Fatalf("n=%d i=%d: Rank = %v, want %v", n, i, got[i], want[i])
			}
		}
	}
}

// TestRankNaNGetsNoRank checks that a NaN neither receives a rank nor shifts the
// ranks of the values around it.
func TestRankNaNGetsNoRank(t *testing.T) {
	xs := []float64{10, math.NaN(), 20, 30}
	got := Rank(xs)
	if !math.IsNaN(got[1]) {
		t.Errorf("Rank(NaN) = %v, want NaN", got[1])
	}
	want := []float64{1, math.NaN(), 2, 3}
	for i := range want {
		if math.IsNaN(want[i]) {
			continue
		}
		if got[i] != want[i] {
			t.Errorf("Rank[%d] = %v, want %v (NaN must not consume a rank)", i, got[i], want[i])
		}
	}
}

func TestSortInPlaceAndCopy(t *testing.T) {
	xs := []float64{3, 1, 2}
	cp := SortCopy(xs)
	if cp[0] != 1 || cp[1] != 2 || cp[2] != 3 {
		t.Errorf("SortCopy = %v, want [1 2 3]", cp)
	}
	if xs[0] != 3 {
		t.Error("SortCopy modified its input")
	}

	SortInPlace(xs)
	if xs[0] != 1 || xs[1] != 2 || xs[2] != 3 {
		t.Errorf("SortInPlace = %v, want [1 2 3]", xs)
	}
	if SortCopy(nil) != nil {
		t.Error("SortCopy(nil) should be nil")
	}
}

// TestSelectKExhaustive checks the selection primitive directly at every k for a
// range of sizes, including the sizes just above the insertion-sort cutoff where
// partitioning first runs.
func TestSelectKExhaustive(t *testing.T) {
	for n := 1; n <= 40; n++ {
		rng := rand.New(rand.NewSource(int64(n) * 31))
		base := make([]float64, n)
		for i := range base {
			base[i] = float64(rng.Intn(100))
		}
		sorted := append([]float64(nil), base...)
		sort.Float64s(sorted)

		for k := 0; k < n; k++ {
			buf := append([]float64(nil), base...)
			selectK(buf, k)
			if buf[k] != sorted[k] {
				t.Fatalf("n=%d k=%d: selectK left %v at k, want %v", n, k, buf[k], sorted[k])
			}
			// The partition invariant: everything left is <= and right is >=.
			for i := 0; i < k; i++ {
				if buf[i] > buf[k] {
					t.Fatalf("n=%d k=%d: left element %v exceeds pivot %v", n, k, buf[i], buf[k])
				}
			}
			for i := k + 1; i < n; i++ {
				if buf[i] < buf[k] {
					t.Fatalf("n=%d k=%d: right element %v is below pivot %v", n, k, buf[i], buf[k])
				}
			}
		}
	}
}

func TestOrderStatsPanics(t *testing.T) {
	xs := make([]float64, 4)
	small := make([]float64, 2)

	cases := []struct {
		name string
		fn   func()
	}{
		{"Quantile q<0", func() { Quantile(xs, -0.1) }},
		{"Quantile q>1", func() { Quantile(xs, 1.1) }},
		{"Percentile p<0", func() { Percentile(xs, -1) }},
		{"Percentile p>100", func() { Percentile(xs, 101) }},
		{"MedianInto small buf", func() { MedianInto(xs, small) }},
		{"QuantileInto small buf", func() { QuantileInto(xs, 0.5, small) }},
		{"PercentileInto small buf", func() { PercentileInto(xs, 50, small) }},
		{"RankTo length", func() { RankTo(small, xs) }},
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

// TestOrderStatsEmptyInputs covers the allocating entry points on an empty slice,
// where the contract is nil or NaN rather than a panic.
func TestOrderStatsEmptyInputs(t *testing.T) {
	if Rank(nil) != nil {
		t.Error("Rank(nil) should be nil")
	}
	if !math.IsNaN(Median(nil)) || !math.IsNaN(MAD(nil)) {
		t.Error("Median(nil) and MAD(nil) should be NaN")
	}
	if Percentile(nil, 50) == 0 {
		// NaN != 0; this branch documents that the result is NaN, not a zero value.
		if !math.IsNaN(Percentile(nil, 50)) {
			t.Error("Percentile(nil, 50) should be NaN")
		}
	}
}

// ---------------------------------------------------------------------------
// Tests for RankScratchTo, the scratch-taking ranking form.
//
// It exists because the allocating form is called twice per window by the rolling rank correlation.
// The risk in a scratch-taking function is that the scratch is *state*: if it is not cleared at the
// start of each call, a second call over a shorter input inherits the first call's tail.
// ---------------------------------------------------------------------------

func TestRankScratchToMatchesRankTo(t *testing.T) {
	for _, size := range []int{1, 2, 3, 5, 8, 13, 20, 33, 64, 100} {
		xs := randomSlice(size, int64(size)+41)
		scratch := make([]int, size)
		got := make([]float64, size)
		RankScratchTo(got, xs, scratch)
		assertRolling(t, "RankScratchTo", got, Rank(xs), false)
	}
}

// TestRankScratchToIsReusableAcrossShorterInputs is the state-leak test: calling it again with a
// shorter slice must not inherit the previous call's entries.
func TestRankScratchToIsReusableAcrossShorterInputs(t *testing.T) {
	long := randomSlice(50, 42)
	short := randomSlice(5, 43)
	scratch := make([]int, 50)
	dst := make([]float64, 50)

	RankScratchTo(dst, long, scratch)
	shortDst := dst[:5]
	RankScratchTo(shortDst, short, scratch)

	want := Rank(short)
	for i := range want {
		if !closeOrNaNBoth(shortDst[i], want[i]) {
			t.Fatalf("reused scratch: rank[%d] = %v, want %v", i, shortDst[i], want[i])
		}
	}
}

// TestRankHandlesTiesAndNaNs pins both special cases at once.
func TestRankHandlesTiesAndNaNs(t *testing.T) {
	xs := []float64{3, 1, 3, math.NaN(), 2, 3}
	got := Rank(xs)

	// The three 3s occupy sorted positions 3, 4 and 5 (1-based), averaging to 4.
	if got[0] != 4 || got[2] != 4 || got[5] != 4 {
		t.Fatalf("tied ranks = %v, %v, %v, want 4 each", got[0], got[2], got[5])
	}
	if got[1] != 1 {
		t.Errorf("rank of the smallest = %v, want 1", got[1])
	}
	if got[4] != 2 {
		t.Errorf("rank of the second smallest = %v, want 2", got[4])
	}
	if !math.IsNaN(got[3]) {
		t.Errorf("rank of NaN = %v, want NaN", got[3])
	}
}

// TestRankLargeInputUsesTheComparisonSort exercises the path above the insertion-sort limit, which a
// small-window test would never reach.
func TestRankLargeInputUsesTheComparisonSort(t *testing.T) {
	const size = 200
	xs := randomSlice(size, 44)
	got := Rank(xs)
	want := refRankBySorting(xs)
	for i := range want {
		if !closeOrNaNBoth(got[i], want[i]) {
			t.Fatalf("rank[%d] = %v, want %v", i, got[i], want[i])
		}
	}
}

// refRankBySorting is the obvious implementation: copy, sort, binary-search each value.
func refRankBySorting(xs []float64) []float64 {
	out := make([]float64, len(xs))
	tmp := make([]float64, 0, len(xs))
	for _, v := range xs {
		if v == v {
			tmp = append(tmp, v)
		}
	}
	sort.Float64s(tmp)
	for i, x := range xs {
		if x != x {
			out[i] = math.NaN()
			continue
		}
		lo := sort.SearchFloat64s(tmp, x)
		hi := lo
		for hi < len(tmp) && tmp[hi] == x {
			hi++
		}
		out[i] = float64(lo+hi-1)/2 + 1
	}
	return out
}
