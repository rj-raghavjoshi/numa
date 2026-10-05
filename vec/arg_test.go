package vec

import (
	"math"
	"math/rand"
	"testing"
)

// ---------------------------------------------------------------------------
// Tests for the arg reductions.
//
// The failure modes worth pinning:
//
//   - seeding the running extreme with ±Inf, which silently fails on an all-±Inf
//     input because no element ever satisfies a strict comparison against it
//   - returning a plausible index for an input containing NaN instead of the
//     documented -1 sentinel
//   - resolving ties to the last occurrence rather than the first
// ---------------------------------------------------------------------------

func refArgMin(xs []float64) int {
	if len(xs) == 0 {
		return -1
	}
	idx := 0
	for i, v := range xs {
		if v != v {
			return -1
		}
		if v < xs[idx] {
			idx = i
		}
	}
	return idx
}

func refArgMax(xs []float64) int {
	if len(xs) == 0 {
		return -1
	}
	idx := 0
	for i, v := range xs {
		if v != v {
			return -1
		}
		if v > xs[idx] {
			idx = i
		}
	}
	return idx
}

func TestArgReductionsMatchReference(t *testing.T) {
	sizes := append([]int{}, boundaryLengths()...)
	sizes = append(sizes, 100, 1000)

	for _, n := range sizes {
		if n == 0 {
			continue
		}
		rng := rand.New(rand.NewSource(int64(n) + 3))
		xs := make([]float64, n)
		for i := range xs {
			// A small integer range produces many ties, which is where the
			// first-occurrence rule matters.
			xs[i] = float64(rng.Intn(5))
		}
		if got, want := ArgMin(xs), refArgMin(xs); got != want {
			t.Fatalf("n=%d: ArgMin = %d, want %d", n, got, want)
		}
		if got, want := ArgMax(xs), refArgMax(xs); got != want {
			t.Fatalf("n=%d: ArgMax = %d, want %d", n, got, want)
		}
		lo, hi := ArgMinMax(xs)
		if lo != refArgMin(xs) || hi != refArgMax(xs) {
			t.Fatalf("n=%d: ArgMinMax = (%d,%d), want (%d,%d)", n, lo, hi, refArgMin(xs), refArgMax(xs))
		}
	}
}

// TestArgReductionsWithInfiniteExtremes is the specific regression test for
// seeding with ±Inf. With an all-+Inf input, `v < math.Inf(1)` is never true, so a
// seed of +Inf would leave the index at its zero value or at -1 rather than
// pointing at the first element.
func TestArgReductionsWithInfiniteExtremes(t *testing.T) {
	allPos := []float64{math.Inf(1), math.Inf(1), math.Inf(1)}
	allNeg := []float64{math.Inf(-1), math.Inf(-1), math.Inf(-1)}

	if got := ArgMin(allPos); got != 0 {
		t.Errorf("ArgMin(all +Inf) = %d, want 0", got)
	}
	if got := ArgMax(allPos); got != 0 {
		t.Errorf("ArgMax(all +Inf) = %d, want 0", got)
	}
	if got := ArgMin(allNeg); got != 0 {
		t.Errorf("ArgMin(all -Inf) = %d, want 0", got)
	}
	if got := ArgMax(allNeg); got != 0 {
		t.Errorf("ArgMax(all -Inf) = %d, want 0", got)
	}

	// A genuine infinity mixed with finite values is a value, not a poison.
	mixed := []float64{5, math.Inf(-1), 3, math.Inf(1)}
	if got := ArgMin(mixed); got != 1 {
		t.Errorf("ArgMin(mixed) = %d, want 1", got)
	}
	if got := ArgMax(mixed); got != 3 {
		t.Errorf("ArgMax(mixed) = %d, want 3", got)
	}
}

func TestArgReductionsTiesResolveFirst(t *testing.T) {
	xs := []float64{5, 1, 5, 1, 5}
	if got := ArgMin(xs); got != 1 {
		t.Errorf("ArgMin = %d, want 1 (first occurrence)", got)
	}
	if got := ArgMax(xs); got != 0 {
		t.Errorf("ArgMax = %d, want 0 (first occurrence)", got)
	}
}

// TestArgReductionsNaNPolicy pins the -1 sentinel and its two meanings.
func TestArgReductionsNaNPolicy(t *testing.T) {
	for pos := 0; pos < 3; pos++ {
		xs := []float64{1, 2, 3}
		xs[pos] = math.NaN()
		if got := ArgMin(xs); got != -1 {
			t.Errorf("ArgMin with NaN at %d = %d, want -1", pos, got)
		}
		if got := ArgMax(xs); got != -1 {
			t.Errorf("ArgMax with NaN at %d = %d, want -1", pos, got)
		}
		lo, hi := ArgMinMax(xs)
		if lo != -1 || hi != -1 {
			t.Errorf("ArgMinMax with NaN at %d = (%d,%d), want (-1,-1)", pos, lo, hi)
		}
	}

	// Both meanings of -1 are reachable, which is why HasNaN exists.
	if ArgMin(nil) != -1 {
		t.Error("ArgMin(nil) should be -1")
	}
	if !HasNaN([]float64{math.NaN()}) {
		t.Error("HasNaN should detect a NaN")
	}
	if !HasNaN([]float64{1, 2, math.NaN()}) {
		t.Error("HasNaN should detect a NaN at the end")
	}
	if HasNaN([]float64{1, math.Inf(1), math.Inf(-1)}) {
		t.Error("HasNaN should not treat infinities as NaN")
	}
	if HasNaN(nil) {
		t.Error("HasNaN(nil) should be false")
	}
}

// TestArgMinMaxAgreesWithSeparatePasses is the cross-check that the fused single
// pass did not drop or update only one of the two running extremes.
func TestArgMinMaxAgreesWithSeparatePasses(t *testing.T) {
	xs := []float64{4, -1, 9, 9, -1, 0, 12, 12}
	lo, hi := ArgMinMax(xs)
	if lo != ArgMin(xs) || hi != ArgMax(xs) {
		t.Errorf("ArgMinMax = (%d,%d), ArgMin/ArgMax = (%d,%d)", lo, hi, ArgMin(xs), ArgMax(xs))
	}
	if lo != 1 || hi != 6 {
		t.Errorf("ArgMinMax = (%d,%d), want (1,6)", lo, hi)
	}
}
