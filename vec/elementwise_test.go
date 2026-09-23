package vec

import (
	"math"
	"testing"
)

// ---------------------------------------------------------------------------
// Elementwise operations.
// ---------------------------------------------------------------------------

func TestElementwiseMatchReference(t *testing.T) {
	for _, n := range boundaryLengths() {
		xs := randomSlice(n, int64(n)+51)
		ys := randomSlice(n, int64(n)+52)

		check := func(name string, got []float64, want func(int) float64) {
			t.Helper()
			for i := range n {
				if !closeEnough(got[i], want(i)) {
					t.Errorf("%s n=%d i=%d: %v, want %v", name, n, i, got[i], want(i))
					return
				}
			}
		}

		check("Add", Add(xs, ys), func(i int) float64 { return xs[i] + ys[i] })
		check("Sub", Sub(xs, ys), func(i int) float64 { return xs[i] - ys[i] })
		check("Mul", Mul(xs, ys), func(i int) float64 { return xs[i] * ys[i] })
		check("Scale", Scale(xs, 2.5), func(i int) float64 { return xs[i] * 2.5 })
		check("AddScalar", AddScalar(xs, -1.25), func(i int) float64 { return xs[i] - 1.25 })
		check("Abs", Abs(xs), func(i int) float64 { return math.Abs(xs[i]) })
	}
}

// TestElementwiseConsumeEveryElement plants a spike at each index and requires
// the corresponding output position to reflect it. A stride bug in the unrolled
// body leaves some output positions untouched.
func TestElementwiseConsumeEveryElement(t *testing.T) {
	const n = 64

	for i := range n {
		xs := make([]float64, n)
		ys := make([]float64, n)
		xs[i] = 1

		if got := Add(xs, ys); got[i] != 1 {
			t.Errorf("Add: spike at %d missing, got %v", i, got[i])
		}
		if got := Sub(xs, ys); got[i] != 1 {
			t.Errorf("Sub: spike at %d missing, got %v", i, got[i])
		}
		// Mul with xs[i]=1, ys[i]=0 must be 0, and the spike must not leak out.
		if got := Mul(xs, ys); got[i] != 0 {
			t.Errorf("Mul: spike at %d produced %v, want 0", i, got[i])
		}
		if got := Scale(xs, 3); got[i] != 3 {
			t.Errorf("Scale: spike at %d missing, got %v", i, got[i])
		}
		if got := AddScalar(xs, 3); got[i] != 4 {
			t.Errorf("AddScalar: spike at %d missing, got %v", i, got[i])
		}
		if got := Abs(xs); got[i] != 1 {
			t.Errorf("Abs: spike at %d missing, got %v", i, got[i])
		}
	}
}

func TestElementwiseEmpty(t *testing.T) {
	if got := Add(nil, nil); got != nil {
		t.Errorf("Add(nil,nil) = %v, want nil", got)
	}
	if got := Sub(nil, nil); got != nil {
		t.Errorf("Sub(nil,nil) = %v, want nil", got)
	}
	if got := Mul(nil, nil); got != nil {
		t.Errorf("Mul(nil,nil) = %v, want nil", got)
	}
	if got := Scale(nil, 2); got != nil {
		t.Errorf("Scale(nil) = %v, want nil", got)
	}
	if got := AddScalar(nil, 2); got != nil {
		t.Errorf("AddScalar(nil) = %v, want nil", got)
	}
	if got := Abs(nil); got != nil {
		t.Errorf("Abs(nil) = %v, want nil", got)
	}
}

func TestElementwiseMismatchedLengths(t *testing.T) {
	xs := []float64{1, 2, 3, 4, 5}
	ys := []float64{10, 20, 30}

	if got := Add(xs, ys); len(got) != 3 {
		t.Errorf("Add mismatched length = %d, want 3", len(got))
	}
	if got := Sub(xs, ys); len(got) != 3 {
		t.Errorf("Sub mismatched length = %d, want 3", len(got))
	}
	if got := Mul(xs, ys); len(got) != 3 {
		t.Errorf("Mul mismatched length = %d, want 3", len(got))
	}
}

func TestAbsKnownValues(t *testing.T) {
	xs := []float64{-1.5, 0, 2.25, -0.0, 1e308, -1e308}
	got := Abs(xs)
	want := []float64{1.5, 0, 2.25, 0, 1e308, 1e308}

	for i := range want {
		if got[i] != want[i] {
			t.Errorf("Abs[%d] = %v, want %v", i, got[i], want[i])
		}
	}
}

// TestAbsIsNonNegative over all lengths checks the defining property rather than
// specific values.
func TestAbsIsNonNegative(t *testing.T) {
	for _, n := range boundaryLengths() {
		xs := randomSlice(n, int64(n)+91)
		for i := range xs {
			if i%2 == 0 {
				xs[i] = -xs[i]
			}
		}

		got := Abs(xs)
		for i := range n {
			// Sign bit must be clear. The `math.Signbit` check catches -0.0, which
			// a bare `got[i] < 0` would miss.
			if math.Signbit(got[i]) && got[i] != 0 {
				t.Errorf("n=%d: Abs[%d] = %v has sign bit set", n, i, got[i])
			}
			if math.Abs(got[i]) != math.Abs(xs[i]) {
				t.Errorf("n=%d: Abs[%d] = %v, want |%v|", n, i, got[i], xs[i])
			}
		}
	}
}

// ---------------------------------------------------------------------------
// The *To variants.
// ---------------------------------------------------------------------------

func TestToVariantsMatchAllocating(t *testing.T) {
	for _, n := range boundaryLengths() {
		// Test the *To variants directly at every length so the tuned loops are
		// exercised even where the allocating forms return early.
		xs := randomSlice(n, int64(n)+71)
		ys := randomSlice(n, int64(n)+72)
		dst := make([]float64, n)

		ops := []struct {
			name  string
			to    func() []float64
			alloc func() []float64
		}{
			{"AddTo", func() []float64 { return AddTo(dst, xs, ys) },
				func() []float64 { return Add(xs, ys) }},
			{"SubTo", func() []float64 { return SubTo(dst, xs, ys) },
				func() []float64 { return Sub(xs, ys) }},
			{"MulTo", func() []float64 { return MulTo(dst, xs, ys) },
				func() []float64 { return Mul(xs, ys) }},
			{"ScaleTo", func() []float64 { return ScaleTo(dst, xs, 4) },
				func() []float64 { return Scale(xs, 4) }},
			{"AddScalarTo", func() []float64 { return AddScalarTo(dst, xs, 4) },
				func() []float64 { return AddScalar(xs, 4) }},
			{"AbsTo", func() []float64 { return AbsTo(dst, xs) },
				func() []float64 { return Abs(xs) }},
		}

		for _, op := range ops {
			// Clear dst first: at n=0 the allocating forms return nil, so the
			// comparison must not depend on stale contents.
			clear(dst)

			got := op.to()
			want := op.alloc()

			if len(got) != len(want) {
				t.Errorf("%s n=%d: length %d, want %d", op.name, n, len(got), len(want))
				continue
			}
			for i := range want {
				if got[i] != want[i] {
					t.Errorf("%s n=%d i=%d: %v, want %v", op.name, n, i, got[i], want[i])
					break
				}
			}
		}
	}
}

// TestToVariantsAliasing checks the documented promise that dst may alias the
// inputs. This is the kind of guarantee that is easy to break while tuning,
// because a tuned body that reads ahead of its writes handles aliasing
// differently from one that does not.
func TestToVariantsAliasing(t *testing.T) {
	const n = 64

	xCopy := randomSlice(n, 81)
	yCopy := randomSlice(n, 82)

	// dst == xs
	xs := make([]float64, n)
	ys := make([]float64, n)
	copy(xs, xCopy)
	copy(ys, yCopy)

	AddTo(xs, xs, ys)
	for i := range n {
		if want := xCopy[i] + yCopy[i]; xs[i] != want {
			t.Errorf("AddTo aliasing xs at %d: %v, want %v", i, xs[i], want)
			break
		}
	}

	// dst == ys
	copy(xs, xCopy)
	copy(ys, yCopy)

	AddTo(ys, xs, ys)
	for i := range n {
		if want := xCopy[i] + yCopy[i]; ys[i] != want {
			t.Errorf("AddTo aliasing ys at %d: %v, want %v", i, ys[i], want)
			break
		}
	}

	// dst == xs == ys
	copy(xs, xCopy)

	MulTo(xs, xs, xs)
	for i := range n {
		if want := xCopy[i] * xCopy[i]; xs[i] != want {
			t.Errorf("MulTo full aliasing at %d: %v, want %v", i, xs[i], want)
			break
		}
	}

	// ScaleTo dst == xs
	copy(xs, xCopy)

	ScaleTo(xs, xs, 3)
	for i := range n {
		if want := xCopy[i] * 3; xs[i] != want {
			t.Errorf("ScaleTo aliasing at %d: %v, want %v", i, xs[i], want)
			break
		}
	}

	// AbsTo dst == xs
	copy(xs, xCopy)

	AbsTo(xs, xs)
	for i := range n {
		if want := math.Abs(xCopy[i]); xs[i] != want {
			t.Errorf("AbsTo aliasing at %d: %v, want %v", i, xs[i], want)
			break
		}
	}
}

func TestToVariantsPanicOnLengthMismatch(t *testing.T) {
	cases := []struct {
		name string
		fn   func()
	}{
		{"AddTo dst", func() { AddTo(make([]float64, 3), make([]float64, 4), make([]float64, 3)) }},
		{"AddTo ys", func() { AddTo(make([]float64, 3), make([]float64, 3), make([]float64, 4)) }},
		{"SubTo", func() { SubTo(make([]float64, 3), make([]float64, 4), make([]float64, 3)) }},
		{"MulTo", func() { MulTo(make([]float64, 3), make([]float64, 4), make([]float64, 3)) }},
		{"ScaleTo", func() { ScaleTo(make([]float64, 3), make([]float64, 4), 2) }},
		{"AddScalarTo", func() { AddScalarTo(make([]float64, 3), make([]float64, 4), 2) }},
		{"AbsTo", func() { AbsTo(make([]float64, 3), make([]float64, 4)) }},
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

// TestAddSubRoundTrip checks that (xs+ys)-ys comes back to xs.
//
// Note the tolerance: this is *not* an exact identity. Adding ys[i] can round away
// low bits of xs[i], and subtracting it back cannot conjure them up again. The
// round trip is only accurate to within a relative epsilon, which is precisely
// the kind of thing that looks like a bug and is not one.
func TestAddSubRoundTrip(t *testing.T) {
	for _, n := range boundaryLengths() {
		xs := randomSlice(n, int64(n)+121)
		ys := randomSlice(n, int64(n)+122)

		mid := Add(xs, ys)
		back := Sub(mid, ys)

		for i := range n {
			if !closeEnough(back[i], xs[i]) {
				t.Errorf("n=%d i=%d: (xs+ys)-ys = %v, want ~%v", n, i, back[i], xs[i])
				break
			}
		}
	}
}
