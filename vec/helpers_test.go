package vec

import (
	"math"
	"math/big"
	"math/rand"
	"testing"
)

// ---------------------------------------------------------------------------
// Shared test helpers.
//
// These are used by more than one test file, so they live here rather than in
// whichever file happened to be written first.
//
// The reference implementations below are deliberately the dumbest possible code.
// A reference that is clever is not a reference.
// ---------------------------------------------------------------------------

func refSum(xs []float64) float64 {
	var t float64
	for _, v := range xs {
		t += v
	}
	return t
}

func refDot(xs, ys []float64) float64 {
	var t float64
	for i, v := range xs {
		t += v * ys[i]
	}
	return t
}

func refSumSq(xs []float64) float64 {
	var t float64
	for _, v := range xs {
		t += v * v
	}
	return t
}

func refMin(xs []float64) float64 {
	m := xs[0]
	for _, v := range xs[1:] {
		if v < m {
			m = v
		}
	}
	return m
}

func refMax(xs []float64) float64 {
	m := xs[0]
	for _, v := range xs[1:] {
		if v > m {
			m = v
		}
	}
	return m
}

// boundaryLengths are the lengths that exercise the awkward edges of the tuned
// loops: empty, shorter than one unrolled stride, exactly one stride, one stride
// plus one, and several strides plus remainders.
//
// These are the lengths where an off-by-one in the `limit := n - k` arithmetic or
// in the tail loop shows up. Random large lengths usually pass even when the loop
// bounds are wrong, which is why they are not sufficient.
//
// See ../docs/design.md for why these specific values.
func boundaryLengths() []int {
	return []int{
		0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17,
		23, 24, 25, 31, 32, 33, 63, 64, 65, 127, 128, 129,
	}
}

// randomSlice builds a deterministic pseudo-random slice of length n. The seed
// makes failures reproducible.
func randomSlice(n int, seed int64) []float64 {
	rng := rand.New(rand.NewSource(seed))
	xs := make([]float64, n)
	for i := range xs {
		xs[i] = rng.NormFloat64()
	}
	return xs
}

// closeEnough reports whether got and want agree within a relative tolerance.
//
// Bit equality is the wrong test for the reductions: the tuned loops reassociate,
// so they are allowed to differ in the last bits. See the package documentation.
func closeEnough(got, want float64) bool {
	if got == want {
		return true // exact equality, including matching infinities
	}
	if math.IsNaN(got) || math.IsNaN(want) {
		return false
	}
	if math.IsInf(got, 0) || math.IsInf(want, 0) {
		return false
	}
	return math.Abs(got-want) <= 1e-12*math.Max(1, math.Abs(want))
}

// exactSum accumulates the exact sum of xs at high precision, so it carries no
// rounding error of its own. It is the reference for the accuracy tests, as
// distinct from the naive loop, which has its own error.
func exactSum(xs []float64) *big.Float {
	acc := new(big.Float).SetPrec(200)
	for _, v := range xs {
		acc.Add(acc, new(big.Float).SetPrec(200).SetFloat64(v))
	}
	return acc
}

// ones returns n copies of 1.0. Used for spike tests, where a baseline of ones
// makes a perturbation unambiguous.
func ones(n int) []float64 {
	xs := make([]float64, n)
	for i := range xs {
		xs[i] = 1
	}
	return xs
}

// wideRange produces values spanning many orders of magnitude, which is the case
// that separates a stable summation from an unstable one.
func wideRange(n int) []float64 {
	xs := make([]float64, n)
	for i := range xs {
		xs[i] = math.Pow(10, float64(i%18)-9)
	}
	return xs
}

// cancelling produces large values that nearly cancel, leaving a small result.
// Naive summation loses precision badly here; a tree-shaped sum does not.
func cancelling(n int) []float64 {
	xs := make([]float64, n)
	for i := range xs {
		if i%2 == 0 {
			xs[i] = 1e8
		} else {
			xs[i] = -1e8
		}
	}
	xs[n-1] = 1
	return xs
}

// TestHelpersSanity checks the helpers themselves. A broken reference
// implementation would make every test in this package meaningless, and it is not
// otherwise covered.
func TestHelpersSanity(t *testing.T) {
	xs := []float64{1, 2, 3, 4}

	if got := refSum(xs); got != 10 {
		t.Errorf("refSum = %v, want 10", got)
	}
	if got := refDot(xs, xs); got != 30 {
		t.Errorf("refDot = %v, want 30", got)
	}
	if got := refSumSq(xs); got != 30 {
		t.Errorf("refSumSq = %v, want 30", got)
	}
	if got := refMin(xs); got != 1 {
		t.Errorf("refMin = %v, want 1", got)
	}
	if got := refMax(xs); got != 4 {
		t.Errorf("refMax = %v, want 4", got)
	}

	if !closeEnough(1.0, 1.0) {
		t.Error("closeEnough should accept exact equality")
	}
	if closeEnough(math.NaN(), math.NaN()) {
		t.Error("closeEnough should reject NaN")
	}
	// Matching infinities are accepted via the exact-equality path, which is the
	// intended behaviour: two +Inf results are equal, not merely close.
	if !closeEnough(math.Inf(1), math.Inf(1)) {
		t.Error("closeEnough should accept matching infinities")
	}
	// Opposite infinities must be rejected, and the IsInf guard is what does it.
	if closeEnough(math.Inf(1), math.Inf(-1)) {
		t.Error("closeEnough should reject opposite infinities")
	}
	if closeEnough(1.0, 1.5) {
		t.Error("closeEnough should reject a 50% difference")
	}

	// Every boundary length must be non-negative, or the tests that index into
	// slices built from it will panic misleadingly.
	for _, n := range boundaryLengths() {
		if n < 0 {
			t.Errorf("boundaryLengths contains a negative value: %d", n)
		}
	}
}
