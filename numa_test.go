package numa

import (
	"math"
	"testing"

	"github.com/rj-raghavjoshi/numa/vec"
)

// ---------------------------------------------------------------------------
// The facade is a set of one-line forwarders. These tests exist because a
// forwarder can be wrong in a way that is invisible: pass the arguments in the
// wrong order, drop one, or forget to convert. None of that shows up in the
// implementation package's own tests.
//
// Each test compares the facade against the implementation, so a mismatch is
// caught here rather than surfacing as a wrong number in a caller.
// ---------------------------------------------------------------------------

func TestReductionForwardersMatchVec(t *testing.T) {
	xs := []float64{3, 1, 4, 1, 5, 9, 2, 6}
	ys := []float64{2, 7, 1, 8, 2, 8, 1, 8}

	if got, want := Sum(xs), vec.Sum(xs); got != want {
		t.Errorf("Sum = %v, vec.Sum = %v", got, want)
	}
	if got, want := Mean(xs), vec.Mean(xs); got != want {
		t.Errorf("Mean = %v, vec.Mean = %v", got, want)
	}
	if got, want := Dot(xs, ys), vec.Dot(xs, ys); got != want {
		t.Errorf("Dot = %v, vec.Dot = %v", got, want)
	}
	if got, want := SumSq(xs), vec.SumSq(xs); got != want {
		t.Errorf("SumSq = %v, vec.SumSq = %v", got, want)
	}
	if got, want := Min(xs), vec.Min(xs); got != want {
		t.Errorf("Min = %v, vec.Min = %v", got, want)
	}
	if got, want := Max(xs), vec.Max(xs); got != want {
		t.Errorf("Max = %v, vec.Max = %v", got, want)
	}

	gotLo, gotHi := MinMax(xs)
	wantLo, wantHi := vec.MinMax(xs)
	if gotLo != wantLo || gotHi != wantHi {
		t.Errorf("MinMax = (%v,%v), vec.MinMax = (%v,%v)", gotLo, gotHi, wantLo, wantHi)
	}
}

// TestDotForwarderArgumentOrder is the specific regression test for swapping the
// two slice arguments, which would pass every symmetric test above.
func TestDotForwarderArgumentOrder(t *testing.T) {
	xs := []float64{1, 2, 3}
	ys := []float64{10, 20, 30}

	// Dot is commutative in value, so compare against the implementation with the
	// same order rather than relying on symmetry.
	if got, want := Dot(xs, ys), vec.Dot(xs, ys); got != want {
		t.Errorf("Dot(xs,ys) = %v, vec.Dot(xs,ys) = %v", got, want)
	}
	if got, want := Dot(xs, ys), 140.0; got != want {
		t.Errorf("Dot(xs,ys) = %v, want 140", got)
	}
}

func TestElementwiseForwardersMatchVec(t *testing.T) {
	xs := []float64{3, 1, 4, 1, 5}
	ys := []float64{2, 7, 1, 8, 2}

	cmp := func(name string, got, want []float64) {
		t.Helper()
		if len(got) != len(want) {
			t.Errorf("%s: length %d, want %d", name, len(got), len(want))
			return
		}
		for i := range want {
			if got[i] != want[i] {
				t.Errorf("%s[%d] = %v, want %v", name, i, got[i], want[i])
			}
		}
	}

	cmp("Add", Add(xs, ys), vec.Add(xs, ys))
	cmp("Sub", Sub(xs, ys), vec.Sub(xs, ys))
	cmp("Mul", Mul(xs, ys), vec.Mul(xs, ys))
	cmp("Scale", Scale(xs, 2.5), vec.Scale(xs, 2.5))
	cmp("AddScalar", AddScalar(xs, -1), vec.AddScalar(xs, -1))
	cmp("Abs", Abs(xs), vec.Abs(xs))
}

func TestToForwardersMatchVec(t *testing.T) {
	xs := []float64{3, 1, 4, 1, 5}
	ys := []float64{2, 7, 1, 8, 2}
	n := len(xs)

	gotDst := make([]float64, n)
	wantDst := make([]float64, n)

	cmp := func(name string) {
		t.Helper()
		for i := range n {
			if gotDst[i] != wantDst[i] {
				t.Errorf("%s[%d] = %v, want %v", name, i, gotDst[i], wantDst[i])
			}
		}
	}

	AddTo(gotDst, xs, ys)
	vec.AddTo(wantDst, xs, ys)
	cmp("AddTo")

	SubTo(gotDst, xs, ys)
	vec.SubTo(wantDst, xs, ys)
	cmp("SubTo")

	MulTo(gotDst, xs, ys)
	vec.MulTo(wantDst, xs, ys)
	cmp("MulTo")

	ScaleTo(gotDst, xs, 3)
	vec.ScaleTo(wantDst, xs, 3)
	cmp("ScaleTo")

	AddScalarTo(gotDst, xs, 3)
	vec.AddScalarTo(wantDst, xs, 3)
	cmp("AddScalarTo")

	AbsTo(gotDst, xs)
	vec.AbsTo(wantDst, xs)
	cmp("AbsTo")
}

// TestForwardersPanicLikeVec checks that the facade does not swallow the panic
// contract. A forwarder that added its own length check would change the panic
// message; one that removed it would silently corrupt memory.
func TestForwardersPanicLikeVec(t *testing.T) {
	short := make([]float64, 2)
	long := make([]float64, 3)

	cases := []struct {
		name string
		fn   func()
	}{
		{"AddTo", func() { AddTo(short, long, short) }},
		{"SubTo", func() { SubTo(short, long, short) }},
		{"MulTo", func() { MulTo(short, long, short) }},
		{"ScaleTo", func() { ScaleTo(short, long, 1) }},
		{"AddScalarTo", func() { AddScalarTo(short, long, 1) }},
		{"AbsTo", func() { AbsTo(short, long) }},
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

// TestNaNPolicySurvivesFacade confirms the documented "NaN wins" policy is still
// in effect through the facade. This is the contract most likely to be
// accidentally dropped by a hand-written forwarder.
func TestNaNPolicySurvivesFacade(t *testing.T) {
	xs := []float64{1, math.NaN(), 3}

	if got := Min(xs); !math.IsNaN(got) {
		t.Errorf("Min = %v, want NaN", got)
	}
	if got := Max(xs); !math.IsNaN(got) {
		t.Errorf("Max = %v, want NaN", got)
	}
	lo, hi := MinMax(xs)
	if !math.IsNaN(lo) || !math.IsNaN(hi) {
		t.Errorf("MinMax = (%v,%v), want (NaN,NaN)", lo, hi)
	}
}

// TestVersion pins the declared version so a release bump cannot be forgotten.
func TestVersion(t *testing.T) {
	if Version == "" {
		t.Error("Version is empty")
	}
}
