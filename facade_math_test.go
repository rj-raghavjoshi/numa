package numa

import (
	"math"
	"testing"

	"github.com/rj-raghavjoshi/numa/vec"
)

// ---------------------------------------------------------------------------
// Forwarder tests for the elementwise math facade.
//
// A forwarder can be wrong in ways vec's own tests cannot catch: swapped
// arguments, a dropped length check, or a lost NaN policy. Each group below
// compares the facade against vec on the same input, and the panic test pins the
// contract that the facade does not add or remove a check.
// ---------------------------------------------------------------------------

func TestMathUnaryForwardersMatchVec(t *testing.T) {
	xs := []float64{-3.5, -0.0, 0, 0.25, 2, 16, 1e-8}

	cmp := func(name string, got, want []float64) {
		t.Helper()
		if len(got) != len(want) {
			t.Fatalf("%s: length %d, want %d", name, len(got), len(want))
		}
		for i := range want {
			if got[i] != want[i] && !(math.IsNaN(got[i]) && math.IsNaN(want[i])) {
				t.Errorf("%s[%d] = %v, want %v", name, i, got[i], want[i])
			}
		}
	}

	cmp("Neg", Neg(xs), vec.Neg(xs))
	cmp("Recip", Recip(xs), vec.Recip(xs))
	cmp("Floor", Floor(xs), vec.Floor(xs))
	cmp("Ceil", Ceil(xs), vec.Ceil(xs))
	cmp("Trunc", Trunc(xs), vec.Trunc(xs))
	cmp("Round", Round(xs), vec.Round(xs))
	cmp("Sqrt", Sqrt(xs), vec.Sqrt(xs))
	cmp("Exp", Exp(xs), vec.Exp(xs))
	cmp("ExpM1", ExpM1(xs), vec.ExpM1(xs))
	cmp("Log1p", Log1p(xs), vec.Log1p(xs))
	cmp("Log2", Log2(xs), vec.Log2(xs))
	cmp("Log10", Log10(xs), vec.Log10(xs))
	cmp("Abs", Abs(xs), vec.Abs(xs))
}

// TestMathArgumentOrderForwarders checks the non-commutative forwarders with
// asymmetric inputs, where a swapped argument is visible.
func TestMathArgumentOrderForwarders(t *testing.T) {
	xs := []float64{1, 2, 3, 4}
	ys := []float64{10, 20, 30, 40}
	zs := []float64{100, 200, 300, 400}

	cmp := func(name string, got, want []float64) {
		t.Helper()
		for i := range want {
			if got[i] != want[i] {
				t.Errorf("%s[%d] = %v, want %v", name, i, got[i], want[i])
			}
		}
	}

	cmp("Sub", Sub(xs, ys), vec.Sub(xs, ys))
	cmp("Div", Div(xs, ys), vec.Div(xs, ys))
	cmp("SafeDiv", SafeDiv(xs, ys), vec.SafeDiv(xs, ys))
	cmp("Min2", Min2(xs, ys), vec.Min2(xs, ys))
	cmp("Max2", Max2(xs, ys), vec.Max2(xs, ys))
	cmp("MinusOperandOrder", Div(ys, xs), vec.Div(ys, xs))
	cmp("Lerp", Lerp(xs, ys, 0.25), vec.Lerp(xs, ys, 0.25))
	cmp("MulAdd", MulAdd(xs, ys, zs), vec.MulAdd(xs, ys, zs))
	cmp("FMA", FMA(xs, ys, zs), vec.FMA(xs, ys, zs))

	if got := Pow(xs, 2)[3]; got != 16 {
		t.Errorf("Pow(xs,2)[3] = %v, want 16", got)
	}
	if got := Clamp(xs, 2, 3); got[0] != 2 || got[1] != 2 || got[2] != 3 || got[3] != 3 {
		t.Errorf("Clamp = %v, want [2 2 3 3]", got)
	}
}

func TestMathForwardersPanicLikeVec(t *testing.T) {
	short := make([]float64, 2)
	long := make([]float64, 3)

	cases := []struct {
		name string
		fn   func()
	}{
		{"NegTo", func() { NegTo(short, long) }},
		{"RecipTo", func() { RecipTo(short, long) }},
		{"SqrtTo", func() { SqrtTo(short, long) }},
		{"LogTo", func() { LogTo(short, long) }},
		{"DivTo", func() { DivTo(short, long, short) }},
		{"SafeDivTo", func() { SafeDivTo(short, long, short) }},
		{"Min2To", func() { Min2To(short, long, short) }},
		{"Max2To", func() { Max2To(short, long, short) }},
		{"LerpTo", func() { LerpTo(short, long, short, 1) }},
		{"MulAddTo", func() { MulAddTo(short, long, short, short) }},
		{"FMATo", func() { FMATo(short, long, short, short) }},
		{"PowTo", func() { PowTo(short, long, 2) }},
		{"ClampTo", func() { ClampTo(short, long, 0, 1) }},
		{"FillTo", func() { CopyTo(short, long) }},
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

// TestMathNaNPolicySurvivesFacade confirms the NaN-propagating pair survives
// forwarding, since that is the contract most easily lost by hand-writing a
// forwarder on top of math.Min/math.Max.
func TestMathNaNPolicySurvivesFacade(t *testing.T) {
	nan := math.NaN()
	if got := Min2([]float64{nan, 1}, []float64{2, nan}); !math.IsNaN(got[0]) || !math.IsNaN(got[1]) {
		t.Errorf("Min2 = %v, want NaNs", got)
	}
	if got := Max2([]float64{nan, 1}, []float64{2, nan}); !math.IsNaN(got[0]) || !math.IsNaN(got[1]) {
		t.Errorf("Max2 = %v, want NaNs", got)
	}
	if got := SafeDiv([]float64{1}, []float64{0}); got[0] != 0 {
		t.Errorf("SafeDiv by zero = %v, want 0", got[0])
	}
}
