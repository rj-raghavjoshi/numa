package vec

import "testing"

// ---------------------------------------------------------------------------
// Dot.
// ---------------------------------------------------------------------------

func TestDotMatchesReference(t *testing.T) {
	for _, n := range boundaryLengths() {
		xs := randomSlice(n, int64(n)+21)
		ys := randomSlice(n, int64(n)+22)

		if got, want := Dot(xs, ys), refDot(xs, ys); !closeEnough(got, want) {
			t.Errorf("n=%d: Dot = %v, reference = %v", n, got, want)
		}
	}
}

func TestDotMismatchedLengths(t *testing.T) {
	xs := []float64{1, 2, 3, 4, 5}
	ys := []float64{1, 2, 3}

	// Should use the first min(len) elements, not panic or pad.
	// 1*1 + 2*2 + 3*3 = 14
	if got, want := Dot(xs, ys), 14.0; got != want {
		t.Errorf("Dot mismatched = %v, want %v", got, want)
	}
}

func TestDotEmpty(t *testing.T) {
	if got := Dot(nil, nil); got != 0 {
		t.Errorf("Dot(nil,nil) = %v, want 0", got)
	}
	if got := Dot([]float64{1, 2}, nil); got != 0 {
		t.Errorf("Dot(xs,nil) = %v, want 0", got)
	}
}

func TestDotKnownValues(t *testing.T) {
	xs := []float64{1, 2, 3, 4}
	ys := []float64{5, 6, 7, 8}
	// 5 + 12 + 21 + 32 = 70
	if got := Dot(xs, ys); got != 70 {
		t.Errorf("Dot = %v, want 70", got)
	}
}

// TestDotConsumesEveryElement is the Dot equivalent of the Sum structural test.
// Both operands are perturbed, because a bug could read one slice correctly and
// mis-stride the other.
func TestDotConsumesEveryElement(t *testing.T) {
	const n = 64

	xs := ones(n)
	ys := ones(n)
	base := Dot(xs, ys)

	for i := range n {
		perturbed := make([]float64, n)
		copy(perturbed, xs)
		perturbed[i] += 1.0
		if got := Dot(perturbed, ys); got == base {
			t.Errorf("Dot ignored xs[%d]", i)
		}

		copy(perturbed, ys)
		perturbed[i] += 1.0
		if got := Dot(xs, perturbed); got == base {
			t.Errorf("Dot ignored ys[%d]", i)
		}
	}
}

// TestDotIsIdentityForUnitBasis checks a property rather than a value: dotting
// with a unit vector must select the corresponding element back out.
func TestDotIsIdentityForUnitBasis(t *testing.T) {
	const n = 32
	xs := randomSlice(n, 5)
	e := make([]float64, n)

	for i := range n {
		clear(e)
		e[i] = 1
		if got := Dot(xs, e); got != xs[i] {
			t.Errorf("Dot(xs, e%d) = %v, want %v", i, got, xs[i])
		}
	}
}
