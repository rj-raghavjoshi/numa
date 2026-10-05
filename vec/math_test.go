package vec

import (
	"math"
	"testing"
)

// ---------------------------------------------------------------------------
// Tests for the elementwise math kernels.
//
// These ops are maps, so the interesting failures are not accumulator bugs but:
//
//   - a mismatched allocating/To pair (one of them wrong)
//   - aliasing corruption in a *To form
//   - NaN being silently dropped by Min2/Max2 (the trap math.Min/Max hide)
//   - the signed-zero edge in Sign
//   - a *To form that does not panic on a length mismatch
//
// Every unary op is checked against a scalar reference at every boundary length,
// which also proves the tail loop consumed each element exactly once.
// ---------------------------------------------------------------------------

type unaryCase struct {
	name  string
	alloc func([]float64) []float64
	to    func(dst, xs []float64) []float64
	ref   func(float64) float64
}

func unaryCases() []unaryCase {
	return []unaryCase{
		{"Neg", Neg, NegTo, func(v float64) float64 { return -v }},
		{"Recip", Recip, RecipTo, func(v float64) float64 { return 1 / v }},
		{"Sign", Sign, SignTo, func(v float64) float64 {
			switch {
			case v > 0:
				return 1
			case v < 0:
				return -1
			case v == 0:
				return v
			default:
				return math.NaN()
			}
		}},
		{"Floor", Floor, FloorTo, math.Floor},
		{"Ceil", Ceil, CeilTo, math.Ceil},
		{"Trunc", Trunc, TruncTo, math.Trunc},
		{"Round", Round, RoundTo, math.Round},
		{"Sqrt", Sqrt, SqrtTo, math.Sqrt},
		{"Exp", Exp, ExpTo, math.Exp},
		{"ExpM1", ExpM1, ExpM1To, math.Expm1},
		{"Log", Log, LogTo, math.Log},
		{"Log1p", Log1p, Log1pTo, math.Log1p},
		{"Log2", Log2, Log2To, math.Log2},
		{"Log10", Log10, Log10To, math.Log10},
	}
}

func TestUnaryMathMatchesReference(t *testing.T) {
	for _, tc := range unaryCases() {
		t.Run(tc.name, func(t *testing.T) {
			for _, n := range boundaryLengths() {
				xs := randomSlice(n, 1)
				got := tc.alloc(xs)
				if n == 0 {
					if got != nil {
						t.Fatalf("n=0: got %v, want nil", got)
					}
					continue
				}
				if len(got) != n {
					t.Fatalf("n=%d: length %d, want %d", n, len(got), n)
				}
				for i := range xs {
					want := tc.ref(xs[i])
					if !sameFloat(got[i], want) {
						t.Fatalf("n=%d i=%d: got %v, want %v", n, i, got[i], want)
					}
				}
			}
		})
	}
}

// TestUnaryToMatchesAlloc proves the two forms of each op agree, which is how a
// copy-paste error between them surfaces.
func TestUnaryToMatchesAlloc(t *testing.T) {
	for _, tc := range unaryCases() {
		t.Run(tc.name, func(t *testing.T) {
			for _, n := range boundaryLengths() {
				if n == 0 {
					continue
				}
				xs := randomSlice(n, 2)
				want := tc.alloc(xs)
				got := tc.to(make([]float64, n), xs)
				for i := range want {
					if !sameFloat(got[i], want[i]) {
						t.Fatalf("n=%d i=%d: To=%v alloc=%v", n, i, got[i], want[i])
					}
				}
			}
		})
	}
}

// TestUnaryToAliasesInput confirms the documented aliasing contract: writing over
// the input in place gives the same answer as allocating.
func TestUnaryToAliasesInput(t *testing.T) {
	for _, tc := range unaryCases() {
		t.Run(tc.name, func(t *testing.T) {
			xs := randomSlice(37, 3)
			want := tc.alloc(xs)
			got := tc.to(xs, xs) // dst aliases xs
			for i := range want {
				if !sameFloat(got[i], want[i]) {
					t.Fatalf("i=%d: aliased=%v alloc=%v", i, got[i], want[i])
				}
			}
		})
	}
}

func TestUnaryToPanicsOnMismatch(t *testing.T) {
	short := make([]float64, 2)
	long := make([]float64, 3)
	for _, tc := range unaryCases() {
		t.Run(tc.name, func(t *testing.T) {
			defer func() {
				if recover() == nil {
					t.Errorf("%s did not panic on length mismatch", tc.name)
				}
			}()
			tc.to(short, long)
		})
	}
}

func TestBinaryDivAndSafeDiv(t *testing.T) {
	xs := []float64{1, 2, 3, 0, -4}
	ys := []float64{2, 0, 4, 0, 0}

	div := Div(xs, ys)
	if div[0] != 0.5 || div[2] != 0.75 {
		t.Fatalf("Div finite values wrong: %v", div)
	}
	if !math.IsInf(div[1], 1) {
		t.Errorf("Div by zero: got %v, want +Inf", div[1])
	}
	if !math.IsNaN(div[3]) {
		t.Errorf("0/0: got %v, want NaN", div[3])
	}

	safe := SafeDiv(xs, ys)
	if safe[1] != 0 || safe[3] != 0 || safe[4] != 0 {
		t.Errorf("SafeDiv zero denominators: got %v, want zeros", safe)
	}
	if safe[0] != 0.5 {
		t.Errorf("SafeDiv finite value: got %v, want 0.5", safe[0])
	}
}

// TestSafeDivPropagatesNaN pins the deliberate asymmetry: a NaN denominator is
// not zero, so it is not masked.
func TestSafeDivPropagatesNaN(t *testing.T) {
	got := SafeDiv([]float64{1}, []float64{math.NaN()})
	if !math.IsNaN(got[0]) {
		t.Errorf("SafeDiv(1, NaN) = %v, want NaN", got[0])
	}
}

// TestMin2Max2PropagateNaN is the specific regression test against implementing
// these with math.Min/math.Max, which return the non-NaN operand and would hide
// a NaN sitting in either input.
func TestMin2Max2PropagateNaN(t *testing.T) {
	nan := math.NaN()
	cases := []struct {
		name string
		fn   func(a, b float64) float64
	}{
		{"Min2", func(a, b float64) float64 { return Min2([]float64{a}, []float64{b})[0] }},
		{"Max2", func(a, b float64) float64 { return Max2([]float64{a}, []float64{b})[0] }},
	}
	for _, tc := range cases {
		for _, in := range [][2]float64{{nan, 1}, {1, nan}, {nan, nan}} {
			if got := tc.fn(in[0], in[1]); !math.IsNaN(got) {
				t.Errorf("%s(%v,%v) = %v, want NaN", tc.name, in[0], in[1], got)
			}
		}
	}
}

func TestMin2Max2PlainValues(t *testing.T) {
	xs := []float64{3, -1, 4, -1, 5}
	ys := []float64{2, 7, 1, -8, 5}

	wantMin := []float64{2, -1, 1, -8, 5}
	wantMax := []float64{3, 7, 4, -1, 5}

	gotMin := Min2(xs, ys)
	gotMax := Max2(xs, ys)
	for i := range xs {
		if gotMin[i] != wantMin[i] {
			t.Errorf("Min2[%d] = %v, want %v", i, gotMin[i], wantMin[i])
		}
		if gotMax[i] != wantMax[i] {
			t.Errorf("Max2[%d] = %v, want %v", i, gotMax[i], wantMax[i])
		}
	}
}

// TestSignPreservesZeroSign pins the subtle contract. Sign(-0) must be -0, not
// +0, or a subsequent 1/sign(x) would flip to the wrong infinity.
func TestSignPreservesZeroSign(t *testing.T) {
	xs := []float64{0, math.Copysign(0, -1), -5, 5}
	got := Sign(xs)
	if math.Signbit(got[0]) {
		t.Errorf("Sign(+0) = %v, want +0", got[0])
	}
	if !math.Signbit(got[1]) {
		t.Errorf("Sign(-0) = %v, want -0", got[1])
	}
	if got[2] != -1 || got[3] != 1 {
		t.Errorf("Sign(±5) = (%v,%v), want (-1,1)", got[2], got[3])
	}
}

func TestClamp(t *testing.T) {
	xs := []float64{-5, 0, 0.5, 1, 9}
	got := Clamp(xs, 0, 1)
	want := []float64{0, 0, 0.5, 1, 1}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("Clamp[%d] = %v, want %v", i, got[i], want[i])
		}
	}
	// lo > hi is documented as min(max(x,lo),hi), which collapses to hi.
	if got := Clamp([]float64{5}, 1, 0)[0]; got != 0 {
		t.Errorf("Clamp with lo>hi = %v, want 0", got)
	}
}

func TestLerpExtrapolatesAndIsNotClamped(t *testing.T) {
	xs := []float64{0, 10}
	ys := []float64{10, 20}
	got := Lerp(xs, ys, 0.5)
	if got[0] != 5 || got[1] != 15 {
		t.Errorf("Lerp midpoint = %v, want [5 15]", got)
	}
	// t outside [0,1] extrapolates by design.
	got = Lerp([]float64{0}, []float64{10}, 2)
	if got[0] != 20 {
		t.Errorf("Lerp extrapolation = %v, want 20", got)
	}
}

func TestMulAddMatchesSeparateOps(t *testing.T) {
	xs := []float64{1, 2, 3}
	ys := []float64{4, 5, 6}
	zs := []float64{7, 8, 9}
	got := MulAdd(xs, ys, zs)
	want := []float64{11, 18, 27}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("MulAdd[%d] = %v, want %v", i, got[i], want[i])
		}
	}
}

func TestReverseAndCopy(t *testing.T) {
	xs := []float64{1, 2, 3, 4}
	rev := Reverse(xs)
	for i, want := range []float64{4, 3, 2, 1} {
		if rev[i] != want {
			t.Errorf("Reverse[%d] = %v, want %v", i, rev[i], want)
		}
	}
	// Reverse must not disturb the source.
	if xs[0] != 1 || xs[3] != 4 {
		t.Errorf("Reverse mutated its input: %v", xs)
	}

	inPlace := []float64{1, 2, 3, 4, 5}
	ReverseInPlace(inPlace)
	for i, want := range []float64{5, 4, 3, 2, 1} {
		if inPlace[i] != want {
			t.Errorf("ReverseInPlace[%d] = %v, want %v", i, inPlace[i], want)
		}
	}

	orig := []float64{1, 2, 3}
	cp := Copy(orig)
	cp[0] = 99
	if orig[0] != 1 {
		t.Error("Copy shares memory with its input")
	}
}

func TestFillAndEmptyResultNil(t *testing.T) {
	got := Fill(2.5, 4)
	for i := range got {
		if got[i] != 2.5 {
			t.Errorf("Fill[%d] = %v, want 2.5", i, got[i])
		}
	}
	if Fill(1, 0) != nil {
		t.Error("Fill(_, 0) should be nil")
	}
	if Neg(nil) != nil || Sqrt(nil) != nil || Div(nil, []float64{1}) != nil {
		t.Error("allocating forms should return nil for empty input")
	}
}

// TestFMAIsSingleRounding pins the one property FMA must have on every
// architecture, using an input where double rounding is observable.
//
// a = 1 + 2^-27 and b = 1 - 2^-27, so a*b is exactly 1 - 2^-54. That is below
// half an ulp of 1, so a twice-rounded product collapses to 1 and the following
// -1 gives 0. A fused multiply-add keeps the low bits and returns -2^-54. A test
// that only compared within a tolerance would never notice the difference.
//
// This assertion is architecture-independent: math.FMA is correctly rounded
// everywhere, whether it is a hardware instruction or the software fallback.
func TestFMAIsSingleRounding(t *testing.T) {
	const tiny = 1.0 / (1 << 27)
	a := []float64{1 + tiny}
	b := []float64{1 - tiny}
	z := []float64{-1}

	if got := FMA(a, b, z)[0]; got != -(tiny * tiny) {
		t.Errorf("FMA = %v, want %v (fused)", got, -(tiny * tiny))
	}
}

// TestMulAddContractionIsObservable records what the compiler actually does to a
// plain `a*b + c`, which is the finding that corrected an earlier claim in this
// repository that Go never contracts.
//
// The contract on MulAdd is deliberately weak: it may be contracted to a fused
// operation or left as two rounded operations, depending on architecture and
// GOAMD64 level. Both are correct. The test asserts the result is one of the two
// permitted values rather than pinning one, and logs which was observed, because
// pinning one would fail on a platform that made the other choice.
//
// FMA exists precisely because this choice is not guaranteed. Callers who need a
// single rounding must use FMA, not MulAdd.
func TestMulAddContractionIsObservable(t *testing.T) {
	const tiny = 1.0 / (1 << 27)
	a := []float64{1 + tiny}
	b := []float64{1 - tiny}
	z := []float64{-1}

	fused := -(tiny * tiny)
	twiceRounded := 0.0

	got := MulAdd(a, b, z)[0]
	switch got {
	case fused:
		t.Logf("MulAdd was contracted to a fused multiply-add on this platform (got %v)", got)
	case twiceRounded:
		t.Logf("MulAdd was left as two rounded operations on this platform (got %v)", got)
	default:
		t.Errorf("MulAdd = %v, want either %v (fused) or %v (twice rounded)", got, fused, twiceRounded)
	}
}

func TestFMAMatchesMathFMA(t *testing.T) {
	for _, n := range boundaryLengths() {
		if n == 0 {
			continue
		}
		xs := randomSlice(n, 11)
		ys := randomSlice(n, 12)
		zs := randomSlice(n, 13)
		got := FMA(xs, ys, zs)
		for i := range xs {
			want := math.FMA(xs[i], ys[i], zs[i])
			if got[i] != want {
				t.Fatalf("n=%d i=%d: got %v, want %v", n, i, got[i], want)
			}
		}
	}
}

// sameFloat compares two floats including the NaN case and the sign of zero,
// which a plain == would treat as equal.
func sameFloat(a, b float64) bool {
	if math.IsNaN(a) || math.IsNaN(b) {
		return math.IsNaN(a) && math.IsNaN(b)
	}
	if a == 0 && b == 0 {
		return math.Signbit(a) == math.Signbit(b)
	}
	return a == b
}
