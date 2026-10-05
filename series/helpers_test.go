package series

import (
	"math"
	"math/rand"
	"testing"
)

// ---------------------------------------------------------------------------
// Shared test helpers.
//
// The references below are deliberately the dumbest possible code: recompute the
// whole window from scratch at every position. That is exactly what a streaming
// roller is supposed to avoid doing, which makes it an independent check rather
// than a restatement of the implementation.
// ---------------------------------------------------------------------------

// boundaryWindows are window lengths that exercise the awkward edges: 1, the
// smallest legal window, values around the unrolled stride, and values that do not
// divide common test lengths evenly.
func boundaryWindows() []int {
	return []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 13, 16, 17, 23, 32, 33, 64, 127}
}

// boundaryLengths are sequence lengths that leave every kind of remainder after
// subtracting a window.
func boundaryLengths() []int {
	return []int{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 16, 17, 23, 24, 25, 31, 32, 33, 63, 64, 65, 127, 128, 129}
}

func randomSlice(n int, seed int64) []float64 {
	rng := rand.New(rand.NewSource(seed))
	xs := make([]float64, n)
	for i := range xs {
		xs[i] = rng.NormFloat64()
	}
	return xs
}

// sameOrNaN reports whether got equals want, treating two NaNs as equal, which is
// what a rolling-window test needs: NaN is the documented "not available" marker, so
// NaN == NaN is the expected outcome, not a failure.
func sameOrNaN(got, want float64) bool {
	if math.IsNaN(got) || math.IsNaN(want) {
		return math.IsNaN(got) && math.IsNaN(want)
	}
	return got == want
}

// closeOrNaN is sameOrNaN with a relative tolerance, for the compensated sums whose
// last bits are not specified.
func closeOrNaN(got, want float64) bool {
	if math.IsNaN(got) || math.IsNaN(want) {
		return math.IsNaN(got) && math.IsNaN(want)
	}
	if got == want {
		return true
	}
	return math.Abs(got-want) <= 1e-12*math.Max(1, math.Abs(want))
}

// refSMA recomputes the average of the trailing window at every position, reporting
// NaN until the window is full.
func refSMA(xs []float64, n int) []float64 {
	out := make([]float64, len(xs))
	for i := range xs {
		if i+1 < n {
			out[i] = math.NaN()
			continue
		}
		var s float64
		for j := i - n + 1; j <= i; j++ {
			s += xs[j]
		}
		out[i] = s / float64(n)
	}
	return out
}

// refSmoother is the shared reference for EMA and RMA: SMA-seeded, then the
// recurrence.
func refSmoother(xs []float64, n int, alpha float64) []float64 {
	out := make([]float64, len(xs))
	seeded := false
	var seed, prev float64
	count := 0
	for i, v := range xs {
		if !seeded {
			seed += v
			count++
			if count < n {
				out[i] = math.NaN()
				continue
			}
			prev = seed / float64(n)
			seeded = true
			out[i] = prev
			continue
		}
		prev = alpha*v + (1-alpha)*prev
		out[i] = prev
	}
	return out
}

func assertSequence(t *testing.T, name string, got, want []float64, exact bool) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("%s: length %d, want %d", name, len(got), len(want))
	}
	ok := func(a, b float64) bool {
		if exact {
			return sameOrNaN(a, b)
		}
		return closeOrNaN(a, b)
	}
	for i := range want {
		if !ok(got[i], want[i]) {
			t.Fatalf("%s[%d] = %v, want %v", name, i, got[i], want[i])
		}
	}
}
