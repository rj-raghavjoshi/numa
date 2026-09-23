package vec

import "testing"

// ---------------------------------------------------------------------------
// SumSq.
// ---------------------------------------------------------------------------

func TestSumSqMatchesReference(t *testing.T) {
	for _, n := range boundaryLengths() {
		xs := randomSlice(n, int64(n)+31)

		if got, want := SumSq(xs), refSumSq(xs); !closeEnough(got, want) {
			t.Errorf("n=%d: SumSq = %v, reference = %v", n, got, want)
		}
	}
}

func TestSumSqEmpty(t *testing.T) {
	if got := SumSq(nil); got != 0 {
		t.Errorf("SumSq(nil) = %v, want 0", got)
	}
}

func TestSumSqKnownValues(t *testing.T) {
	// 1 + 4 + 9 + 16 = 30
	if got := SumSq([]float64{1, 2, 3, 4}); got != 30 {
		t.Errorf("SumSq = %v, want 30", got)
	}
}

func TestSumSqConsumesEveryElement(t *testing.T) {
	const n = 64
	base := randomSlice(n, 77)
	baseSum := SumSq(base)

	for i := range n {
		perturbed := make([]float64, n)
		copy(perturbed, base)
		perturbed[i] += 1.0

		if got := SumSq(perturbed); got == baseSum {
			t.Errorf("element %d appears to be ignored: SumSq unchanged", i)
		}
	}
}

// TestSumSqIsDotWithSelf checks an identity that must hold regardless of the
// individual implementations. It is a cross-check between two independently tuned
// loops, so it catches a mistake that happens to be consistent within one of them.
func TestSumSqIsDotWithSelf(t *testing.T) {
	for _, n := range boundaryLengths() {
		if n == 0 {
			continue
		}
		xs := randomSlice(n, int64(n)+101)

		if !closeEnough(SumSq(xs), Dot(xs, xs)) {
			t.Errorf("n=%d: SumSq = %v, Dot(xs,xs) = %v", n, SumSq(xs), Dot(xs, xs))
		}
	}
}
