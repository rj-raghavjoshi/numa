package vec

import (
	"math"
	"testing"
)

// ---------------------------------------------------------------------------
// Tests for the cumulative operations.
//
// The failure modes worth pinning:
//
//   - an off-by-one in the block boundary that drops or double-counts an element
//   - a carry that is not advanced correctly between the blocked loop and the tail
//   - a NaN that poisons the wrong direction, or stops poisoning after the first
//     output instead of persisting to the end
//   - aliasing corruption in the *To forms, which the blocked loops must survive
//     by reading a whole block before writing it
// ---------------------------------------------------------------------------

func refCumSum(xs []float64) []float64 {
	out := make([]float64, len(xs))
	var s float64
	for i, v := range xs {
		s += v
		out[i] = s
	}
	return out
}

func refCumProd(xs []float64) []float64 {
	out := make([]float64, len(xs))
	p := 1.0
	for i, v := range xs {
		p *= v
		out[i] = p
	}
	return out
}

// refCumMax implements the documented NaN-wins prefix maximum.
func refCumMax(xs []float64) []float64 {
	out := make([]float64, len(xs))
	m := math.Inf(-1)
	nan := false
	for i, v := range xs {
		if v != v {
			nan = true
		} else if v > m {
			m = v
		}
		if nan {
			out[i] = math.NaN()
		} else {
			out[i] = m
		}
	}
	return out
}

func refCumMin(xs []float64) []float64 {
	out := make([]float64, len(xs))
	m := math.Inf(1)
	nan := false
	for i, v := range xs {
		if v != v {
			nan = true
		} else if v < m {
			m = v
		}
		if nan {
			out[i] = math.NaN()
		} else {
			out[i] = m
		}
	}
	return out
}

// cumCloseEnough compares two same-length results elementwise. The cumulative ops
// reassociate, so exact equality is the wrong test for the arithmetic ones.
func cumCloseEnough(t *testing.T, name string, got, want []float64) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("%s: length %d, want %d", name, len(got), len(want))
	}
	for i := range want {
		if math.IsNaN(want[i]) {
			if !math.IsNaN(got[i]) {
				t.Fatalf("%s[%d] = %v, want NaN", name, i, got[i])
			}
			continue
		}
		if !closeEnough(got[i], want[i]) {
			t.Fatalf("%s[%d] = %v, want %v", name, i, got[i], want[i])
		}
	}
}

func TestCumulativeMatchReference(t *testing.T) {
	for _, n := range boundaryLengths() {
		if n == 0 {
			continue
		}
		xs := randomSlice(n, 41)

		cumCloseEnough(t, "CumSum", CumSum(xs), refCumSum(xs))
		cumCloseEnough(t, "CumMax", CumMax(xs), refCumMax(xs))
		cumCloseEnough(t, "CumMin", CumMin(xs), refCumMin(xs))
		// Products grow far too fast to compare at a relative tolerance once n is
		// large, so only the small boundary lengths are checked. The block
		// structure is still exercised there, which is the point of the test.
		if n <= 16 {
			cumCloseEnough(t, "CumProd", CumProd(xs), refCumProd(xs))
		}
	}
}

// TestCumulativeKnownValues pins exact results at the awkward lengths either side
// of the four-wide block boundary, where a carry error would show up.
func TestCumulativeKnownValues(t *testing.T) {
	xs := []float64{1, 2, 3, 4, 5, 6, 7}

	sum := CumSum(xs)
	wantSum := []float64{1, 3, 6, 10, 15, 21, 28}
	for i := range wantSum {
		if sum[i] != wantSum[i] {
			t.Errorf("CumSum[%d] = %v, want %v", i, sum[i], wantSum[i])
		}
	}

	max := CumMax(xs)
	wantMax := []float64{1, 2, 3, 4, 5, 6, 7}
	for i := range wantMax {
		if max[i] != wantMax[i] {
			t.Errorf("CumMax[%d] = %v, want %v", i, max[i], wantMax[i])
		}
	}

	desc := []float64{7, 6, 5, 4, 3, 2, 1}
	min := CumMin(desc)
	wantMin := []float64{7, 6, 5, 4, 3, 2, 1}
	for i := range wantMin {
		if min[i] != wantMin[i] {
			t.Errorf("CumMin[%d] = %v, want %v", i, min[i], wantMin[i])
		}
	}
}

// TestCumulativeTailCarry checks the handoff from the blocked loop to the tail
// loop. A carry that is not carried across that boundary produces results that
// are right up to the last partial block and wrong after it.
func TestCumulativeTailCarry(t *testing.T) {
	// Lengths 5..11 cross the block boundary with every possible remainder.
	for n := 5; n <= 11; n++ {
		xs := make([]float64, n)
		for i := range xs {
			xs[i] = float64(i + 1)
		}
		got := CumSum(xs)
		var s float64
		for i, v := range xs {
			s += v
			if got[i] != s {
				t.Fatalf("n=%d i=%d: CumSum = %v, want %v", n, i, got[i], s)
			}
		}
	}
}

// TestCumulativeNaNPoisonsForward pins the direction and persistence of the NaN
// policy: a NaN poisons itself and everything after it, and nothing before it.
func TestCumulativeNaNPoisonsForward(t *testing.T) {
	for n := 1; n <= 9; n++ {
		for pos := 0; pos < n; pos++ {
			xs := make([]float64, n)
			for i := range xs {
				xs[i] = float64(i) + 1
			}
			xs[pos] = math.NaN()

			gotMax := CumMax(xs)
			gotMin := CumMin(xs)
			want := refCumMax(xs)
			for i := range xs {
				if math.IsNaN(want[i]) != math.IsNaN(gotMax[i]) {
					t.Fatalf("CumMax n=%d pos=%d i=%d: got %v, want %v", n, pos, i, gotMax[i], want[i])
				}
				if math.IsNaN(want[i]) != math.IsNaN(gotMin[i]) {
					t.Fatalf("CumMin n=%d pos=%d i=%d: got %v, want %v", n, pos, i, gotMin[i], want[i])
				}
				// Before the NaN the value must be finite, not poisoned early.
				if i < pos && math.IsNaN(gotMax[i]) {
					t.Fatalf("CumMax n=%d pos=%d poisoned index %d before the NaN", n, pos, i)
				}
			}
		}
	}
}

// TestCumulativeInfinityIsNotNaN is the counter-check to the poisoning test: an
// infinity is a value, not a poison, and must persist without becoming NaN.
func TestCumulativeInfinityIsNotNaN(t *testing.T) {
	xs := []float64{1, math.Inf(1), 2}
	got := CumMax(xs)
	if math.IsNaN(got[2]) {
		t.Fatalf("CumMax after +Inf = %v, want +Inf", got[2])
	}
	if !math.IsInf(got[2], 1) {
		t.Fatalf("CumMax after +Inf = %v, want +Inf", got[1])
	}
}

func TestCumulativeAliasing(t *testing.T) {
	orig := []float64{1, 2, 3, 4, 5, 6, 7, 8, 9}

	check := func(name string, fn func(dst, xs []float64) []float64) {
		t.Helper()
		want := fn(make([]float64, len(orig)), orig)
		aliased := append([]float64(nil), orig...)
		got := fn(aliased, aliased)
		for i := range want {
			if !closeEnough(got[i], want[i]) {
				t.Fatalf("%s aliased[%d] = %v, want %v", name, i, got[i], want[i])
			}
		}
	}

	check("CumSumTo", CumSumTo)
	check("CumProdTo", CumProdTo)
	check("CumMaxTo", CumMaxTo)
	check("CumMinTo", CumMinTo)
}

func TestCumCountTrue(t *testing.T) {
	mask := []uint8{1, 0, 1, 1, 0, 0, 2}
	got := CumCountTrue(mask)
	want := []int{1, 1, 2, 3, 3, 3, 4}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("CumCountTrue[%d] = %d, want %d", i, got[i], want[i])
		}
	}
	if CumCountTrue(nil) != nil {
		t.Error("CumCountTrue(nil) should be nil")
	}
}

func TestCumulativeAllocatingFormsReturnNilWhenEmpty(t *testing.T) {
	if CumSum(nil) != nil || CumProd(nil) != nil || CumMax(nil) != nil || CumMin(nil) != nil {
		t.Error("cumulative allocating forms should return nil for empty input")
	}
}

func TestCumulativeToPanicsOnMismatch(t *testing.T) {
	short := make([]float64, 2)
	long := make([]float64, 3)
	shortI := make([]int, 2)
	longM := make([]uint8, 3)

	cases := []struct {
		name string
		fn   func()
	}{
		{"CumSumTo", func() { CumSumTo(short, long) }},
		{"CumProdTo", func() { CumProdTo(short, long) }},
		{"CumMaxTo", func() { CumMaxTo(short, long) }},
		{"CumMinTo", func() { CumMinTo(short, long) }},
		{"CumCountTrueTo", func() { CumCountTrueTo(shortI, longM) }},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			defer func() {
				if recover() == nil {
					t.Errorf("%s did not panic on length mismatch", tc.name)
				}
			}()
			tc.fn()
		})
	}
}
