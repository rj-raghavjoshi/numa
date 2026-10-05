package numa

import (
	"math"
	"testing"

	"github.com/rj-raghavjoshi/numa/vec"
)

// ---------------------------------------------------------------------------
// Forwarder tests for the mask facade.
//
// Beyond the usual swapped-argument checks, the mask family has one contract
// that is easy to lose in a forwarder and impossible to see in a symmetric test:
// the NaN asymmetry between the ordered predicates (false) and NotEqual (true).
// It is pinned explicitly below.
// ---------------------------------------------------------------------------

func TestMaskForwardersMatchVec(t *testing.T) {
	xs := []float64{3, 1, 4, 1, 5}
	ys := []float64{2, 7, 1, 8, 2}

	cmp := func(name string, got, want []uint8) {
		t.Helper()
		if len(got) != len(want) {
			t.Fatalf("%s: length %d, want %d", name, len(got), len(want))
		}
		for i := range want {
			if got[i] != want[i] {
				t.Errorf("%s[%d] = %d, want %d", name, i, got[i], want[i])
			}
		}
	}

	cmp("Greater", Greater(xs, ys), vec.Greater(xs, ys))
	cmp("GreaterEqual", GreaterEqual(xs, ys), vec.GreaterEqual(xs, ys))
	cmp("Less", Less(xs, ys), vec.Less(xs, ys))
	cmp("LessEqual", LessEqual(xs, ys), vec.LessEqual(xs, ys))
	cmp("Equal", Equal(xs, ys), vec.Equal(xs, ys))
	cmp("NotEqual", NotEqual(xs, ys), vec.NotEqual(xs, ys))
	cmp("GreaterScalar", GreaterScalar(xs, 3), vec.GreaterScalar(xs, 3))
	cmp("LessScalar", LessScalar(xs, 3), vec.LessScalar(xs, 3))
	cmp("GreaterEqualScalar", GreaterEqualScalar(xs, 3), vec.GreaterEqualScalar(xs, 3))
	cmp("LessEqualScalar", LessEqualScalar(xs, 3), vec.LessEqualScalar(xs, 3))
	cmp("EqualScalar", EqualScalar(xs, 3), vec.EqualScalar(xs, 3))
	cmp("NotEqualScalar", NotEqualScalar(xs, 3), vec.NotEqualScalar(xs, 3))
	cmp("IsNaN", IsNaN(xs), vec.IsNaN(xs))
	cmp("IsFinite", IsFinite(xs), vec.IsFinite(xs))
	cmp("IsInf", IsInf(xs, 0), vec.IsInf(xs, 0))

	cmp("And", And(GreaterScalar(xs, 1), LessScalar(xs, 5)), vec.And(vec.GreaterScalar(xs, 1), vec.LessScalar(xs, 5)))
	cmp("Or", Or(GreaterScalar(xs, 4), LessScalar(xs, 2)), vec.Or(vec.GreaterScalar(xs, 4), vec.LessScalar(xs, 2)))
	cmp("Xor", Xor(GreaterScalar(xs, 3), LessScalar(xs, 3)), vec.Xor(vec.GreaterScalar(xs, 3), vec.LessScalar(xs, 3)))
	cmp("Not", Not(GreaterScalar(xs, 3)), vec.Not(vec.GreaterScalar(xs, 3)))
}

// TestMaskArgumentOrderForwarders uses non-commutative inputs so a swapped
// argument is visible. Greater and Less swap into each other, which a symmetric
// test would never catch.
func TestMaskArgumentOrderForwarders(t *testing.T) {
	xs := []float64{1, 2, 3}
	ys := []float64{3, 2, 1}

	forward := []uint8{0, 0, 1}  // xs > ys
	backward := []uint8{1, 0, 0} // ys > xs

	got := Greater(xs, ys)
	for i := range forward {
		if got[i] != forward[i] {
			t.Errorf("Greater(xs,ys)[%d] = %d, want %d", i, got[i], forward[i])
		}
	}
	got = Greater(ys, xs)
	for i := range backward {
		if got[i] != backward[i] {
			t.Errorf("Greater(ys,xs)[%d] = %d, want %d", i, got[i], backward[i])
		}
	}
	got = Less(ys, xs)
	for i := range forward {
		if got[i] != forward[i] {
			t.Errorf("Less(ys,xs)[%d] = %d, want %d", i, got[i], forward[i])
		}
	}
}

func TestMaskSelectionForwardersMatchVec(t *testing.T) {
	xs := []float64{3, 1, 4, 1, 5}
	ys := []float64{2, 7, 1, 8, 2}
	mask := Greater(xs, ys)

	gotWhere := Where(mask, xs, ys)
	wantWhere := vec.Where(vec.Greater(xs, ys), xs, ys)
	for i := range wantWhere {
		if gotWhere[i] != wantWhere[i] {
			t.Errorf("Where[%d] = %v, want %v", i, gotWhere[i], wantWhere[i])
		}
	}

	gotC := Compress(xs, mask)
	wantC := vec.Compress(xs, vec.Greater(xs, ys))
	if len(gotC) != len(wantC) {
		t.Fatalf("Compress length %d, want %d", len(gotC), len(wantC))
	}
	for i := range wantC {
		if gotC[i] != wantC[i] {
			t.Errorf("Compress[%d] = %v, want %v", i, gotC[i], wantC[i])
		}
	}

	if got, want := CountTrue(mask), vec.CountTrue(mask); got != want {
		t.Errorf("CountTrue = %d, want %d", got, want)
	}
	if got, want := AnyTrue(mask), vec.AnyTrue(mask); got != want {
		t.Errorf("AnyTrue = %v, want %v", got, want)
	}
	if got, want := AllTrue(mask), vec.AllTrue(mask); got != want {
		t.Errorf("AllTrue = %v, want %v", got, want)
	}

	if got, want := MaskedSum(xs, mask), vec.MaskedSum(xs, mask); got != want {
		t.Errorf("MaskedSum = %v, want %v", got, want)
	}
	if got, want := MaskedMean(xs, mask), vec.MaskedMean(xs, mask); got != want {
		t.Errorf("MaskedMean = %v, want %v", got, want)
	}
	if got, want := MaskedMin(xs, mask), vec.MaskedMin(xs, mask); got != want {
		t.Errorf("MaskedMin = %v, want %v", got, want)
	}
	if got, want := MaskedMax(xs, mask), vec.MaskedMax(xs, mask); got != want {
		t.Errorf("MaskedMax = %v, want %v", got, want)
	}
}

// TestMaskNaNPolicySurvivesFacade pins the asymmetry through the facade: every
// ordered predicate is false at a NaN, and NotEqual is true.
func TestMaskNaNPolicySurvivesFacade(t *testing.T) {
	xs := []float64{math.NaN()}
	ys := []float64{1}

	for _, tc := range []struct {
		name string
		got  uint8
	}{
		{"Greater", Greater(xs, ys)[0]},
		{"GreaterEqual", GreaterEqual(xs, ys)[0]},
		{"Less", Less(xs, ys)[0]},
		{"LessEqual", LessEqual(xs, ys)[0]},
		{"Equal", Equal(xs, ys)[0]},
	} {
		if tc.got != 0 {
			t.Errorf("%s with NaN = %d, want 0", tc.name, tc.got)
		}
	}
	if got := NotEqual(xs, ys)[0]; got != 1 {
		t.Errorf("NotEqual with NaN = %d, want 1", got)
	}
	if got := IsNaN(xs)[0]; got != 1 {
		t.Errorf("IsNaN(NaN) = %d, want 1", got)
	}
}

func TestMaskForwardersPanicLikeVec(t *testing.T) {
	shortF := make([]float64, 2)
	longF := make([]float64, 3)
	shortM := make([]uint8, 2)
	longM := make([]uint8, 3)

	cases := []struct {
		name string
		fn   func()
	}{
		{"GreaterTo", func() { GreaterTo(shortM, longF, longF) }},
		{"GreaterScalarTo", func() { GreaterScalarTo(shortM, longF, 1) }},
		{"IsNaNTo", func() { IsNaNTo(shortM, longF) }},
		{"AndTo", func() { AndTo(shortM, longM, longM) }},
		{"NotTo", func() { NotTo(shortM, longM) }},
		{"WhereTo", func() { WhereTo(shortF, longM, longF, longF) }},
		{"CompressTo", func() { CompressTo(shortF, longF, longM) }},
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
