package mat

import (
	"math"
	"testing"
)

// ---------------------------------------------------------------------------
// Tests for the dense matrix type and its elementwise and product kernels.
//
// The product is checked against a naive triple loop over a range of shapes, because a shape bug
// in a matrix product is the classic silent failure: the wrong index produces a plausible number
// rather than an error. The naive reference is deliberately written in the i-j-k order, which is
// the *opposite* traversal to the implementation's i-k-j, so a confusion between the two shows up.
// ---------------------------------------------------------------------------

func matClose(t *testing.T, name string, got, want *Mat, tol float64) {
	t.Helper()
	if got.Rows != want.Rows || got.Cols != want.Cols {
		t.Fatalf("%s: shape %dx%d, want %dx%d", name, got.Rows, got.Cols, want.Rows, want.Cols)
	}
	for i := 0; i < want.Rows; i++ {
		for j := 0; j < want.Cols; j++ {
			g, w := got.Data[i*want.Cols+j], want.Data[i*want.Cols+j]
			if math.IsNaN(g) || math.IsNaN(w) {
				if math.IsNaN(g) != math.IsNaN(w) {
					t.Fatalf("%s[%d,%d] = %v, want %v", name, i, j, g, w)
				}
				continue
			}
			if math.Abs(g-w) > tol*math.Max(1, math.Abs(w)) {
				t.Fatalf("%s[%d,%d] = %v, want %v", name, i, j, g, w)
			}
		}
	}
}

// filled returns a rows x cols matrix whose elements follow a deterministic pattern that makes
// the row and column indices visible, so a transposed or shifted result cannot match by accident.
func filled(rows, cols int, seed float64) *Mat {
	m := New(rows, cols)
	for i := 0; i < rows; i++ {
		for j := 0; j < cols; j++ {
			m.Data[i*cols+j] = math.Sin(seed + float64(i)*1.7 + float64(j)*0.3)
		}
	}
	return m
}

func naiveMul(a, b *Mat) *Mat {
	out := New(a.Rows, b.Cols)
	for i := 0; i < a.Rows; i++ {
		for j := 0; j < b.Cols; j++ {
			var sum float64
			for k := 0; k < a.Cols; k++ {
				sum += a.Data[i*a.Cols+k] * b.Data[k*b.Cols+j]
			}
			out.Data[i*b.Cols+j] = sum
		}
	}
	return out
}

func TestMulMatchesNaiveForManyShapes(t *testing.T) {
	shapes := [][3]int{
		{1, 1, 1}, {1, 2, 1}, {2, 1, 3}, {2, 2, 2}, {3, 4, 5}, {5, 3, 2},
		{8, 8, 8}, {7, 13, 11}, {16, 16, 16}, {1, 30, 1}, {30, 1, 30}, {4, 4, 1},
	}
	for _, sh := range shapes {
		a := filled(sh[0], sh[1], 1)
		b := filled(sh[1], sh[2], 2)
		got := Mul(a, b)
		want := naiveMul(a, b)
		matClose(t, "Mul", got, want, 1e-12)
	}
}

// TestMulAgainstHandComputed pins one product where every entry can be checked by hand.
func TestMulAgainstHandComputed(t *testing.T) {
	// [[1,2],[3,4]] * [[5,6],[7,8]] = [[19,22],[43,50]]
	a := FromRows([]float64{1, 2, 3, 4}, 2, 2)
	b := FromRows([]float64{5, 6, 7, 8}, 2, 2)
	got := Mul(a, b)
	want := FromRows([]float64{19, 22, 43, 50}, 2, 2)
	matClose(t, "Mul", got, want, 1e-12)
}

// TestMulIsAssociativeButNotCommutative is a property test rather than a value test: it catches a
// kernel that is accidentally symmetric in its arguments.
func TestMulIsAssociativeButNotCommutative(t *testing.T) {
	a := filled(6, 5, 3)
	b := filled(5, 7, 4)
	c := filled(7, 4, 5)

	left := Mul(Mul(a, b), c)
	right := Mul(a, Mul(b, c))
	matClose(t, "associativity", left, right, 1e-10)

	// Non-commutativity: the two orders must differ somewhere. Asserting that they are equal
	// would be asserting the wrong property, so the check is that they differ by a real amount.
	p := filled(5, 5, 6)
	q := filled(5, 5, 7)
	pq, qp := Mul(p, q), Mul(q, p)
	different := false
	for i := range pq.Data {
		if math.Abs(pq.Data[i]-qp.Data[i]) > 1e-9 {
			different = true
			break
		}
	}
	if !different {
		t.Error("pq and qp are identical; the test data cannot distinguish them")
	}
}

func TestMulIdentityAndZero(t *testing.T) {
	a := filled(7, 9, 8)
	id7 := Identity(7)
	id9 := Identity(9)
	matClose(t, "I*A", Mul(id7, a), a, 1e-15)
	matClose(t, "A*I", Mul(a, id9), a, 1e-15)
	matClose(t, "A*0", Mul(a, New(9, 4)), New(7, 4), 1e-15)
	matClose(t, "0*A", Mul(New(3, 7), a), New(3, 9), 1e-15)
}

func TestMulVecMatchesMulAndNaive(t *testing.T) {
	a := filled(9, 6, 9)
	x := make([]float64, 6)
	for i := range x {
		x[i] = math.Cos(float64(i) + 0.5)
	}
	got := MulVec(a, x)
	want := naiveMul(a, FromRows(x, 6, 1))
	for i := range got {
		if math.Abs(got[i]-want.Data[i]) > 1e-12 {
			t.Fatalf("MulVec[%d] = %v, want %v", i, got[i], want.Data[i])
		}
	}
	// An empty column dimension gives an empty result rather than a panic.
	if r := MulVec(New(3, 0), nil); len(r) != 3 {
		t.Fatalf("MulVec with zero columns returned length %d, want 3", len(r))
	}
}

func TestTransposeAndRoundTrip(t *testing.T) {
	a := filled(4, 7, 10)
	at := Transpose(a)
	if at.Rows != 7 || at.Cols != 4 {
		t.Fatalf("Transpose shape %dx%d", at.Rows, at.Cols)
	}
	for i := 0; i < a.Rows; i++ {
		for j := 0; j < a.Cols; j++ {
			if at.At(j, i) != a.At(i, j) {
				t.Fatalf("Transpose[%d,%d] mismatch", j, i)
			}
		}
	}
	matClose(t, "Aᵀᵀ", Transpose(at), a, 0)
}

func TestElementwiseOps(t *testing.T) {
	a := filled(5, 6, 11)
	b := filled(5, 6, 12)

	matClose(t, "Add", Add(a, b), naiveAdd(a, b), 1e-15)
	matClose(t, "Sub", Sub(a, b), naiveSub(a, b), 1e-15)
	matClose(t, "Scale", Scale(a, 2.5), naiveScale(a, 2.5), 1e-15)

	// AddTo must write through the destination, not return a copy.
	dst := New(5, 6)
	if AddTo(dst, a, b) != dst {
		t.Error("AddTo did not return dst")
	}
	matClose(t, "AddTo", dst, naiveAdd(a, b), 1e-15)
}

func naiveAdd(a, b *Mat) *Mat {
	out := New(a.Rows, a.Cols)
	for i := range out.Data {
		out.Data[i] = a.Data[i] + b.Data[i]
	}
	return out
}

func naiveSub(a, b *Mat) *Mat {
	out := New(a.Rows, a.Cols)
	for i := range out.Data {
		out.Data[i] = a.Data[i] - b.Data[i]
	}
	return out
}

func naiveScale(a *Mat, alpha float64) *Mat {
	out := New(a.Rows, a.Cols)
	for i := range out.Data {
		out.Data[i] = a.Data[i] * alpha
	}
	return out
}

func TestDotAxpyAndNorms(t *testing.T) {
	x := []float64{3, -4}
	y := []float64{1, 2}
	if got := Dot(x, y); got != 3-8 {
		t.Fatalf("Dot = %v, want -5", got)
	}
	if got := Norm2(x); math.Abs(got-5) > 1e-15 {
		t.Fatalf("Norm2 = %v, want 5", got)
	}
	got := Axpy(2, x, y)
	want := []float64{1 + 6, 2 - 8}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("Axpy[%d] = %v, want %v", i, got[i], want[i])
		}
	}

	m := FromRows([]float64{1, -2, 3, 4}, 2, 2)
	if got := NormInf(m); got != 7 {
		t.Fatalf("NormInf = %v, want 7 (max row sum 3+4)", got)
	}
	if got := Trace(m); got != 5 {
		t.Fatalf("Trace = %v, want 5", got)
	}
}

func TestIsSymmetric(t *testing.T) {
	sym := FromRows([]float64{4, 1, 2, 1, 3, 0, 2, 0, 5}, 3, 3)
	if !IsSymmetric(sym, 0) {
		t.Error("a symmetric matrix was reported asymmetric")
	}
	asym := sym.Clone()
	asym.Set(0, 1, 1.5)
	if IsSymmetric(asym, 0) {
		t.Error("an asymmetric matrix was reported symmetric")
	}
	if !IsSymmetric(asym, 0.6) {
		t.Error("a symmetric matrix within tolerance was reported asymmetric")
	}
	if IsSymmetric(New(2, 3), 0) {
		t.Error("a non-square matrix was reported symmetric")
	}
}

func TestAccessorsAndClone(t *testing.T) {
	m := New(2, 3)
	m.Set(0, 1, 5)
	if m.At(0, 1) != 5 {
		t.Fatalf("At after Set = %v", m.At(0, 1))
	}
	m.Row(1)[2] = 9
	if m.At(1, 2) != 9 {
		t.Error("Row did not share the backing array")
	}

	c := m.Clone()
	c.Set(0, 1, 100)
	if m.At(0, 1) == 100 {
		t.Error("Clone shares memory with the original")
	}

	m.Fill(3.5)
	for _, v := range m.Data {
		if v != 3.5 {
			t.Fatal("Fill did not set every element")
		}
	}

	if !New(3, 3).IsSquare() || New(3, 4).IsSquare() {
		t.Error("IsSquare is wrong")
	}
}

func TestShapesPanic(t *testing.T) {
	a := filled(3, 4, 1)
	c := filled(4, 3, 3)

	cases := []struct {
		name string
		fn   func()
	}{
		{"Mul inner mismatch", func() { Mul(a, a) }},
		{"MulTo shape", func() { MulTo(New(2, 2), a, c) }},
		{"MulVec length", func() { MulVec(a, make([]float64, 3)) }},
		{"MulVecTo length", func() { MulVecTo(make([]float64, 2), a, make([]float64, 4)) }},
		{"Add shape", func() { Add(a, c) }},
		{"Sub shape", func() { Sub(a, c) }},
		{"Copy shape", func() { Copy(a, c) }},
		{"ScaleTo shape", func() { ScaleTo(a, c, 1) }},
		{"Axpy length", func() { Axpy(1, make([]float64, 2), make([]float64, 3)) }},
		{"At out of range", func() { a.At(3, 0) }},
		{"At negative", func() { a.At(-1, 0) }},
		{"Set out of range", func() { a.Set(0, 4, 1) }},
		{"Row out of range", func() { a.Row(3) }},
		{"Trace non-square", func() { Trace(a) }},
		{"New negative", func() { New(-1, 2) }},
		{"FromRows length", func() { FromRows([]float64{1, 2}, 2, 2) }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			defer func() {
				if recover() == nil {
					t.Errorf("%s did not panic", tc.name)
				}
			}()
			tc.fn()
		})
	}
}

// TestEmptyMatricesAreUsable pins the degenerate case: a zero-dimension matrix must not be a
// special case everywhere.
func TestEmptyMatricesAreUsable(t *testing.T) {
	z := New(0, 0)
	if !z.IsSquare() {
		t.Error("the 0x0 matrix should be square")
	}
	matClose(t, "0*0", Mul(z, z), New(0, 0), 0)
	matClose(t, "0+0", Add(z, z), New(0, 0), 0)
	matClose(t, "0ᵀ", Transpose(z), New(0, 0), 0)
	if Trace(z) != 0 {
		t.Error("the trace of the 0x0 matrix should be 0")
	}
	if NormInf(z) != 0 {
		t.Error("the norm of the 0x0 matrix should be 0")
	}
	if !IsSymmetric(z, 0) {
		t.Error("the 0x0 matrix should be symmetric")
	}
	if got := Mul(New(3, 0), New(0, 4)); got.Rows != 3 || got.Cols != 4 {
		t.Errorf("3x0 * 0x4 = %dx%d, want 3x4", got.Rows, got.Cols)
	}
}
