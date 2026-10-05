package vec

import (
	"math"
	"testing"
)

// ---------------------------------------------------------------------------
// Tests for predicates, masks, and masked reductions.
//
// The failure modes worth pinning here are:
//
//   - a predicate that claims an ordering for NaN, which the hardware does not
//     support, so a mask would silently be wrong
//   - mask algebra that only works when the bytes are exactly 0 and 1
//   - a masked reduction that disagrees with the corresponding unmasked one when
//     everything is selected, or that mishandles the empty selection
//   - Compress dropping or reordering elements
// ---------------------------------------------------------------------------

// TestPredicatesMatchReference checks every predicate against a scalar reference
// across the boundary lengths, which also proves each tail loop consumed every
// element exactly once.
func TestPredicatesMatchReference(t *testing.T) {
	cases := []struct {
		name string
		fn   func(x, y float64) uint8
		got  func(xs, ys []float64) []uint8
	}{
		{"Greater", func(x, y float64) uint8 { return boolToU8(x > y) }, Greater},
		{"GreaterEqual", func(x, y float64) uint8 { return boolToU8(x >= y) }, GreaterEqual},
		{"Less", func(x, y float64) uint8 { return boolToU8(x < y) }, Less},
		{"LessEqual", func(x, y float64) uint8 { return boolToU8(x <= y) }, LessEqual},
		{"Equal", func(x, y float64) uint8 { return boolToU8(x == y) }, Equal},
		{"NotEqual", func(x, y float64) uint8 { return boolToU8(x != y) }, NotEqual},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			for _, n := range boundaryLengths() {
				if n == 0 {
					continue
				}
				// Deliberately equal values at some positions, so equality cases
				// are exercised rather than only strict orderings.
				xs := randomSlice(n, 21)
				ys := randomSlice(n, 22)
				for i := 0; i < n; i += 3 {
					ys[i] = xs[i]
				}
				got := tc.got(xs, ys)
				for i := range xs {
					want := tc.fn(xs[i], ys[i])
					if got[i] != want {
						t.Fatalf("n=%d i=%d (%v,%v): got %d, want %d",
							n, i, xs[i], ys[i], got[i], want)
					}
				}
			}
		})
	}
}

// TestPredicateNaNSemantics pins the IEEE behaviour that makes masks safe: no
// ordered comparison involving NaN may claim an ordering.
func TestPredicateNaNSemantics(t *testing.T) {
	nan := math.NaN()
	xs := []float64{nan, 1, nan}
	ys := []float64{1, nan, nan}

	if got := Greater(xs, ys); got[0] != 0 || got[1] != 0 || got[2] != 0 {
		t.Errorf("Greater with NaN = %v, want all 0", got)
	}
	if got := Less(xs, ys); got[0] != 0 || got[1] != 0 || got[2] != 0 {
		t.Errorf("Less with NaN = %v, want all 0", got)
	}
	if got := GreaterEqual(xs, ys); got[0] != 0 || got[1] != 0 || got[2] != 0 {
		t.Errorf("GreaterEqual with NaN = %v, want all 0", got)
	}
	if got := LessEqual(xs, ys); got[0] != 0 || got[1] != 0 || got[2] != 0 {
		t.Errorf("LessEqual with NaN = %v, want all 0", got)
	}
	if got := Equal(xs, ys); got[0] != 0 || got[1] != 0 || got[2] != 0 {
		t.Errorf("Equal with NaN = %v, want all 0", got)
	}
	// NotEqual is the asymmetry: NaN != anything is true.
	if got := NotEqual(xs, ys); got[0] != 1 || got[1] != 1 || got[2] != 1 {
		t.Errorf("NotEqual with NaN = %v, want all 1", got)
	}
}

func TestScalarPredicates(t *testing.T) {
	xs := []float64{-2, 0, 2, math.NaN()}
	k := 1.0

	if got := GreaterScalar(xs, k); got[0] != 0 || got[1] != 0 || got[2] != 1 || got[3] != 0 {
		t.Errorf("GreaterScalar = %v, want [0 0 1 0]", got)
	}
	if got := LessScalar(xs, k); got[0] != 1 || got[1] != 1 || got[2] != 0 || got[3] != 0 {
		t.Errorf("LessScalar = %v, want [1 1 0 0]", got)
	}
	if got := GreaterEqualScalar(xs, k); got[2] != 1 || got[3] != 0 {
		t.Errorf("GreaterEqualScalar = %v", got)
	}
	if got := LessEqualScalar(xs, k); got[2] != 0 || got[3] != 0 {
		t.Errorf("LessEqualScalar = %v", got)
	}
	if got := EqualScalar([]float64{1, 2}, 1); got[0] != 1 || got[1] != 0 {
		t.Errorf("EqualScalar = %v, want [1 0]", got)
	}
	if got := NotEqualScalar([]float64{1, 2}, 1); got[0] != 0 || got[1] != 1 {
		t.Errorf("NotEqualScalar = %v, want [0 1]", got)
	}
}

func TestValueClassPredicates(t *testing.T) {
	xs := []float64{1, math.NaN(), math.Inf(1), math.Inf(-1), 0}

	if got := IsNaN(xs); got[0] != 0 || got[1] != 1 || got[2] != 0 || got[3] != 0 || got[4] != 0 {
		t.Errorf("IsNaN = %v, want only index 1 set", got)
	}
	if got := IsFinite(xs); got[0] != 1 || got[1] != 0 || got[2] != 0 || got[3] != 0 || got[4] != 1 {
		t.Errorf("IsFinite = %v, want indices 0 and 4 set", got)
	}
	if got := IsInf(xs, 0); got[2] != 1 || got[3] != 1 || got[1] != 0 {
		t.Errorf("IsInf(sign=0) = %v, want both infinities", got)
	}
	if got := IsInf(xs, 1); got[2] != 1 || got[3] != 0 {
		t.Errorf("IsInf(sign=+1) = %v, want only +Inf", got)
	}
	if got := IsInf(xs, -1); got[2] != 0 || got[3] != 1 {
		t.Errorf("IsInf(sign=-1) = %v, want only -Inf", got)
	}
}

// TestMaskAlgebraIsTruthBased is the regression test against bitwise
// implementations that only happen to work for the 0/1 convention. The mask
// values 1 and 2 are both true, so 1 XOR 2 must be false even though 1^2 == 3.
func TestMaskAlgebraIsTruthBased(t *testing.T) {
	a := []uint8{1, 1, 2, 0, 0, 255}
	b := []uint8{1, 0, 2, 1, 0, 0}

	wantAnd := []uint8{1, 0, 1, 0, 0, 0}
	wantOr := []uint8{1, 1, 1, 1, 0, 1}
	wantXor := []uint8{0, 1, 0, 1, 0, 1}
	wantNot := []uint8{0, 0, 0, 1, 1, 0}

	assertMask(t, "And", And(a, b), wantAnd)
	assertMask(t, "Or", Or(a, b), wantOr)
	assertMask(t, "Xor", Xor(a, b), wantXor)
	assertMask(t, "Not", Not(a), wantNot)
}

func assertMask(t *testing.T, name string, got, want []uint8) {
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

func TestMaskCounts(t *testing.T) {
	mask := []uint8{0, 1, 0, 2, 0}
	if got := CountTrue(mask); got != 2 {
		t.Errorf("CountTrue = %d, want 2", got)
	}
	if AnyTrue([]uint8{0, 0}) {
		t.Error("AnyTrue(empty mask of zeros) = true, want false")
	}
	if !AnyTrue(mask) {
		t.Error("AnyTrue = false, want true")
	}
	if AllTrue([]uint8{1, 2, 0}) {
		t.Error("AllTrue with a zero = true, want false")
	}
	if !AllTrue([]uint8{1, 2, 255}) {
		t.Error("AllTrue with all non-zero = false, want true")
	}
	if !AllTrue(nil) {
		t.Error("AllTrue(nil) = false, want true (empty conjunction)")
	}
}

func TestWhere(t *testing.T) {
	cond := []uint8{1, 0, 1, 0}
	a := []float64{10, 20, 30, 40}
	b := []float64{-1, -2, -3, -4}
	got := Where(cond, a, b)
	want := []float64{10, -2, 30, -4}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("Where[%d] = %v, want %v", i, got[i], want[i])
		}
	}
}

func TestCompressGathersInOrder(t *testing.T) {
	xs := []float64{10, 20, 30, 40, 50}
	mask := []uint8{0, 1, 0, 1, 1}
	got := Compress(xs, mask)
	want := []float64{20, 40, 50}
	if len(got) != len(want) {
		t.Fatalf("Compress length = %d, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("Compress[%d] = %v, want %v", i, got[i], want[i])
		}
	}

	// Nothing selected is documented as nil.
	if got := Compress(xs, make([]uint8, len(xs))); got != nil {
		t.Errorf("Compress with no selection = %v, want nil", got)
	}

	// All selected returns a full-length copy, not a view of the input.
	full := Compress(xs, []uint8{1, 1, 1, 1, 1})
	full[0] = 99
	if xs[0] != 10 {
		t.Error("Compress shares memory with its input")
	}
}

// TestCompressToReturnsSubslice pins the reuse contract: dst is sized to the
// input length and reused across calls with different selection counts, so the
// return value must be a subslice rather than the whole buffer.
func TestCompressToReturnsSubslice(t *testing.T) {
	xs := []float64{1, 2, 3, 4}
	dst := make([]float64, len(xs))

	got := CompressTo(dst, xs, []uint8{1, 0, 1, 0})
	if len(got) != 2 || got[0] != 1 || got[1] != 3 {
		t.Fatalf("CompressTo = %v (len %d), want [1 3]", got, len(got))
	}
	if len(dst) != 4 {
		t.Errorf("CompressTo resized dst to %d, want it left at 4", len(dst))
	}
}

// TestMaskedReductionsAgreeWithUnmaskedWhenAllSelected is the cross-check that
// catches a masked reduction that walks the wrong index or mishandles the tail.
func TestMaskedReductionsAgreeWithUnmaskedWhenAllSelected(t *testing.T) {
	xs := []float64{3, 1, 4, 1, 5, 9, 2, 6}
	all := make([]uint8, len(xs))
	for i := range all {
		all[i] = 1
	}

	if got, want := MaskedSum(xs, all), Sum(xs); !closeEnough(got, want) {
		t.Errorf("MaskedSum = %v, Sum = %v", got, want)
	}
	if got, want := MaskedMean(xs, all), Mean(xs); !closeEnough(got, want) {
		t.Errorf("MaskedMean = %v, Mean = %v", got, want)
	}
	if got, want := MaskedMin(xs, all), Min(xs); got != want {
		t.Errorf("MaskedMin = %v, Min = %v", got, want)
	}
	if got, want := MaskedMax(xs, all), Max(xs); got != want {
		t.Errorf("MaskedMax = %v, Max = %v", got, want)
	}
}

func TestMaskedReductionsSelectCorrectly(t *testing.T) {
	xs := []float64{3, 1, 4, 1, 5}
	mask := []uint8{1, 0, 1, 0, 1} // selects 3, 4, 5

	if got := MaskedSum(xs, mask); got != 12 {
		t.Errorf("MaskedSum = %v, want 12", got)
	}
	if got := MaskedMean(xs, mask); got != 4 {
		t.Errorf("MaskedMean = %v, want 4", got)
	}
	if got := MaskedMin(xs, mask); got != 3 {
		t.Errorf("MaskedMin = %v, want 3", got)
	}
	if got := MaskedMax(xs, mask); got != 5 {
		t.Errorf("MaskedMax = %v, want 5", got)
	}
}

// TestMaskedEmptySelection pins the identities, which mirror the unmasked
// empty-input contracts so a masked reduction composes with the rest of the
// package rather than inventing its own edge behaviour.
func TestMaskedEmptySelection(t *testing.T) {
	xs := []float64{3, 1, 4}
	none := make([]uint8, len(xs))

	if got := MaskedSum(xs, none); got != 0 {
		t.Errorf("MaskedSum with no selection = %v, want 0", got)
	}
	if got := MaskedMean(xs, none); got != 0 {
		t.Errorf("MaskedMean with no selection = %v, want 0", got)
	}
	if got := MaskedMin(xs, none); !math.IsInf(got, 1) {
		t.Errorf("MaskedMin with no selection = %v, want +Inf", got)
	}
	if got := MaskedMax(xs, none); !math.IsInf(got, -1) {
		t.Errorf("MaskedMax with no selection = %v, want -Inf", got)
	}
}

// TestMaskedScansPropagateNaN confirms the "NaN wins" policy survives masking,
// and only for selected elements.
func TestMaskedScansPropagateNaN(t *testing.T) {
	xs := []float64{1, math.NaN(), 3}

	// NaN selected: the scan must report NaN.
	if got := MaskedMin(xs, []uint8{0, 1, 1}); !math.IsNaN(got) {
		t.Errorf("MaskedMin over selected NaN = %v, want NaN", got)
	}
	if got := MaskedMax(xs, []uint8{0, 1, 1}); !math.IsNaN(got) {
		t.Errorf("MaskedMax over selected NaN = %v, want NaN", got)
	}
	// NaN not selected: it must not poison the result.
	if got := MaskedMin(xs, []uint8{1, 0, 1}); got != 1 {
		t.Errorf("MaskedMin over unselected NaN = %v, want 1", got)
	}
	if got := MaskedMax(xs, []uint8{1, 0, 1}); got != 3 {
		t.Errorf("MaskedMax over unselected NaN = %v, want 3", got)
	}
}

func TestMaskForwardersPanicOnMismatch(t *testing.T) {
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

func TestMaskToVariantsMatchAllocating(t *testing.T) {
	for _, n := range boundaryLengths() {
		if n == 0 {
			continue
		}
		xs := randomSlice(n, 31)
		ys := randomSlice(n, 32)

		pairs := []struct {
			name  string
			alloc func() []uint8
			to    func(dst []uint8)
		}{
			{"Greater", func() []uint8 { return Greater(xs, ys) }, func(d []uint8) { GreaterTo(d, xs, ys) }},
			{"Less", func() []uint8 { return Less(xs, ys) }, func(d []uint8) { LessTo(d, xs, ys) }},
			{"Equal", func() []uint8 { return Equal(xs, ys) }, func(d []uint8) { EqualTo(d, xs, ys) }},
			{"IsNaN", func() []uint8 { return IsNaN(xs) }, func(d []uint8) { IsNaNTo(d, xs) }},
		}
		for _, p := range pairs {
			want := p.alloc()
			got := make([]uint8, n)
			p.to(got)
			for i := range want {
				if got[i] != want[i] {
					t.Fatalf("%s n=%d i=%d: To=%d alloc=%d", p.name, n, i, got[i], want[i])
				}
			}
		}
	}
}
