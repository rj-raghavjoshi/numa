package mat

import (
	"errors"
	"math"
	"sort"
	"testing"
)

// ---------------------------------------------------------------------------
// Tests for the factorisations and solves.
//
// These are tested primarily by *residuals* rather than by comparing against a reference
// implementation. The reason is that the useful statement about a solver is "it produces an x for
// which Ax is b", and every decomposition here has an identity that can be checked directly:
// L Lᵀ = A, Q R = A, A A⁻¹ = I, A v = λ v. A reference implementation would test agreement with
// another algorithm rather than correctness of the answer.
//
// Where a closed form exists -- 2x2 systems, a 2x2 symmetric eigenproblem -- the exact value is
// asserted as well, because residuals cannot catch a consistent scale error.
// ---------------------------------------------------------------------------

// spd returns an n x n symmetric positive-definite matrix: MᵀM plus n on the diagonal, which
// guarantees the eigenvalues are at least n.
func spd(n int, seed float64) *Mat {
	m := filled(n, n, seed)
	mt := Transpose(m)
	a := Mul(mt, m)
	for i := 0; i < n; i++ {
		a.Data[i*n+i] += float64(n)
	}
	return a
}

// randomish returns a matrix of pseudo-random values in [-0.5, 0.5) from a linear congruential
// generator.
//
// It exists because filled() uses a smooth trigonometric pattern, which makes its columns nearly
// collinear -- harmless for testing a matrix product, but useless for testing a least-squares fit:
// a near-singular design matrix turns that exercise into a test of conditioning, and the residual
// orthogonality then fails for reasons that have nothing to do with the solver.
func randomish(rows, cols int, seed uint64) *Mat {
	m := New(rows, cols)
	s := seed*6364136223846793005 + 1442695040888963407
	for i := range m.Data {
		s = s*6364136223846793005 + 1442695040888963407
		m.Data[i] = float64(s>>11)/float64(uint64(1)<<53) - 0.5
	}
	return m
}

// residual returns ||A x - b||inf scaled by the size of the inputs, which is the quantity a solver
// is actually responsible for.
func residual(a *Mat, x, b []float64) float64 {
	ax := MulVec(a, x)
	var worst, scale float64
	for i := range b {
		d := math.Abs(ax[i] - b[i])
		if d > worst {
			worst = d
		}
		s := math.Abs(b[i]) + math.Abs(ax[i])
		if s > scale {
			scale = s
		}
	}
	if scale == 0 {
		return 0
	}
	return worst / scale
}

func rhs(n int, seed float64) []float64 {
	b := make([]float64, n)
	for i := range b {
		b[i] = math.Cos(seed + float64(i)*0.9)
	}
	return b
}

func TestSolveHandComputed(t *testing.T) {
	// 2x + y = 3, x + 3y = 5  ->  x = 0.8, y = 1.4
	a := FromRows([]float64{2, 1, 1, 3}, 2, 2)
	x, err := Solve(a, []float64{3, 5})
	if err != nil {
		t.Fatalf("Solve: %v", err)
	}
	if math.Abs(x[0]-0.8) > 1e-12 || math.Abs(x[1]-1.4) > 1e-12 {
		t.Fatalf("Solve = %v, want [0.8 1.4]", x)
	}
}

func TestSolveResiduals(t *testing.T) {
	for _, n := range []int{1, 2, 3, 5, 8, 13, 21} {
		a := filled(n, n, float64(n)+1)
		// Add a multiple of the identity so the matrix is well-conditioned and the residual is a
		// meaningful test of the solver rather than of the conditioning.
		for i := 0; i < n; i++ {
			a.Data[i*n+i] += float64(n) + 2
		}
		b := rhs(n, float64(n))
		x, err := Solve(a, b)
		if err != nil {
			t.Fatalf("n=%d Solve: %v", n, err)
		}
		if r := residual(a, x, b); r > 1e-10 {
			t.Fatalf("n=%d relative residual %v is too large", n, r)
		}
	}
}

func TestSolveSingular(t *testing.T) {
	// The second row is twice the first, so there is no unique solution.
	a := FromRows([]float64{1, 2, 2, 4}, 2, 2)
	if _, err := Solve(a, []float64{1, 2}); !errors.Is(err, ErrSingular) {
		t.Fatalf("Solve of a singular matrix returned %v, want ErrSingular", err)
	}
}

// TestSolveDoesNotModifyItsInput is a contract check: Solve clones before factoring, so a caller
// can reuse the matrix.
func TestSolveDoesNotModifyItsInput(t *testing.T) {
	a := filled(4, 4, 20)
	for i := 0; i < 4; i++ {
		a.Data[i*4+i] += 5
	}
	before := a.Clone()
	if _, err := Solve(a, rhs(4, 1)); err != nil {
		t.Fatalf("Solve: %v", err)
	}
	matClose(t, "Solve input", a, before, 0)
}

func TestInverseIsTwoSided(t *testing.T) {
	for _, n := range []int{1, 2, 3, 5, 8, 12} {
		a := filled(n, n, float64(n)+2)
		for i := 0; i < n; i++ {
			a.Data[i*n+i] += float64(n) + 3
		}
		inv, err := Inverse(a)
		if err != nil {
			t.Fatalf("n=%d Inverse: %v", n, err)
		}
		matClose(t, "A*inv", Mul(a, inv), Identity(n), 1e-9)
		matClose(t, "inv*A", Mul(inv, a), Identity(n), 1e-9)
	}
}

func TestInverseSingular(t *testing.T) {
	a := FromRows([]float64{1, 2, 2, 4}, 2, 2)
	if _, err := Inverse(a); !errors.Is(err, ErrSingular) {
		t.Fatalf("Inverse of a singular matrix returned %v, want ErrSingular", err)
	}
}

func TestCholeskyReconstructs(t *testing.T) {
	// The sizes straddle the point where the dot-product kernel switches from its simple loop to the
	// four-accumulator one, and include values that are not multiples of four so the unrolled loop's
	// remainder is exercised. Without those the unrolled branch is never reached: the largest j the
	// small sizes produce is 12.
	sizes := []int{
		1, 2, 3, 5, 8, 13,
		15, 16, 17, 18, 19, 20, 21, 23, 24, 25,
		31, 32, 33, 40, 64,
	}
	for _, n := range sizes {
		a := spd(n, float64(n))
		l, err := Cholesky(a)
		if err != nil {
			t.Fatalf("n=%d Cholesky: %v", n, err)
		}
		matClose(t, "L Lᵀ", Mul(l, Transpose(l)), a, 1e-10)

		// L must be lower triangular.
		for i := 0; i < n; i++ {
			for j := i + 1; j < n; j++ {
				if l.Data[i*n+j] != 0 {
					t.Fatalf("n=%d L[%d,%d] = %v, want 0 above the diagonal", n, i, j, l.Data[i*n+j])
				}
			}
		}
	}
}

func TestCholeskyHandComputed(t *testing.T) {
	// [[4,2],[2,3]] = L Lᵀ with L = [[2,0],[1,sqrt(2)]]
	a := FromRows([]float64{4, 2, 2, 3}, 2, 2)
	l, err := Cholesky(a)
	if err != nil {
		t.Fatalf("Cholesky: %v", err)
	}
	if math.Abs(l.At(0, 0)-2) > 1e-12 || l.At(0, 1) != 0 {
		t.Fatalf("L row 0 = %v, want [2 0]", l.Row(0))
	}
	if math.Abs(l.At(1, 0)-1) > 1e-12 || math.Abs(l.At(1, 1)-math.Sqrt2) > 1e-12 {
		t.Fatalf("L row 1 = %v, want [1 sqrt(2)]", l.Row(1))
	}
}

// TestCholeskyRejectsNonSPD covers both failure modes: a negative pivot and a zero one.
func TestCholeskyRejectsNonSPD(t *testing.T) {
	cases := []struct {
		name string
		data []float64
	}{
		{"negative definite", []float64{-1, 0, 0, -1}},
		{"indefinite", []float64{1, 2, 2, 1}},
		{"singular", []float64{1, 1, 1, 1}},
		{"zero", []float64{0, 0, 0, 0}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			a := FromRows(c.data, 2, 2)
			if _, err := Cholesky(a); !errors.Is(err, ErrNotSPD) {
				t.Fatalf("Cholesky returned %v, want ErrNotSPD", err)
			}
		})
	}
}

func TestCholeskySolveResiduals(t *testing.T) {
	for _, n := range []int{1, 2, 5, 8, 13} {
		a := spd(n, float64(n)+3)
		b := rhs(n, float64(n))
		l, err := Cholesky(a)
		if err != nil {
			t.Fatalf("n=%d Cholesky: %v", n, err)
		}
		x := CholeskySolve(l, b)
		if r := residual(a, x, b); r > 1e-10 {
			t.Fatalf("n=%d relative residual %v", n, r)
		}
	}
}

func TestQRReconstructsAndIsOrthonormal(t *testing.T) {
	shapes := [][2]int{{1, 1}, {3, 1}, {4, 4}, {7, 3}, {13, 5}, {10, 10}, {20, 7}}
	for _, sh := range shapes {
		m, n := sh[0], sh[1]
		if m < n {
			continue
		}
		a := filled(m, n, float64(m+n))
		q, r := QR(a)

		if q.Rows != m || q.Cols != n {
			t.Fatalf("%dx%d: Q is %dx%d, want %dx%d", m, n, q.Rows, q.Cols, m, n)
		}
		if r.Rows != n || r.Cols != n {
			t.Fatalf("%dx%d: R is %dx%d, want %dx%d", m, n, r.Rows, r.Cols, n, n)
		}
		// R must be upper triangular.
		for i := 0; i < n; i++ {
			for j := 0; j < i; j++ {
				if r.Data[i*n+j] != 0 {
					t.Fatalf("%dx%d: R[%d,%d] = %v, want 0", m, n, i, j, r.Data[i*n+j])
				}
			}
		}
		// QᵀQ = I, where Q is m x n so QᵀQ is n x n.
		matClose(t, "QᵀQ", Mul(Transpose(q), q), Identity(n), 1e-10)
		// Q R = A.
		matClose(t, "QR", Mul(q, r), a, 1e-10)
	}
}

// TestQRPreservesTheColumnSpace is a weaker but shape-checking invariant: the first column of Q
// must be parallel to the first column of A.
func TestQRPreservesTheColumnSpace(t *testing.T) {
	a := filled(8, 4, 30)
	q, _ := QR(a)
	// a[:,0] and q[:,0] should be parallel for a Householder QR.
	colA := make([]float64, a.Rows)
	colQ := make([]float64, q.Rows)
	for i := 0; i < a.Rows; i++ {
		colA[i] = a.Data[i*a.Cols]
		colQ[i] = q.Data[i*q.Cols]
	}
	// The absolute value of the cosine must be 1.
	cos := math.Abs(Dot(colA, colQ)) / (Norm2(colA) * Norm2(colQ))
	if math.Abs(cos-1) > 1e-10 {
		t.Fatalf("first column cosine = %v, want 1", cos)
	}
}

func TestLeastSquaresExactFit(t *testing.T) {
	// The points (0,1), (1,3), (2,5) lie exactly on y = 1 + 2x.
	a := FromRows([]float64{
		1, 0,
		1, 1,
		1, 2,
	}, 3, 2)
	x, err := LeastSquares(a, []float64{1, 3, 5})
	if err != nil {
		t.Fatalf("LeastSquares: %v", err)
	}
	if math.Abs(x[0]-1) > 1e-12 || math.Abs(x[1]-2) > 1e-12 {
		t.Fatalf("LeastSquares = %v, want [1 2]", x)
	}
}

// TestLeastSquaresResidualIsOrthogonal is the defining property of a least-squares solution: the
// residual must be orthogonal to the column space. This holds for an over-determined system with no
// exact solution, which is the case the exact-fit test cannot reach.
func TestLeastSquaresResidualIsOrthogonal(t *testing.T) {
	for _, sh := range [][2]int{{5, 2}, {8, 3}, {13, 5}, {20, 4}} {
		m, n := sh[0], sh[1]
		a := randomish(m, n, uint64(m*n)+1)
		b := rhs(m, float64(m))
		x, err := LeastSquares(a, b)
		if err != nil {
			t.Fatalf("%dx%d LeastSquares: %v", m, n, err)
		}
		ax := MulVec(a, x)
		res := make([]float64, m)
		for i := range res {
			res[i] = b[i] - ax[i]
		}
		// Aᵀr must be zero for every column. The scale is the residual norm times the norm of
		// *that* column, not of row 0, or a column with a different magnitude would be judged
		// against the wrong threshold.
		for j := 0; j < n; j++ {
			var dotCol, colNorm float64
			for i := 0; i < m; i++ {
				col := a.Data[i*n+j]
				dotCol += col * res[i]
				colNorm += col * col
			}
			scale := Norm2(res) * math.Sqrt(colNorm)
			if scale == 0 {
				continue
			}
			if math.Abs(dotCol) > 1e-9*scale {
				t.Fatalf("%dx%d: column %d has residual component %v, want orthogonal", m, n, j, dotCol)
			}
		}
	}
}

func TestLeastSquaresRankDeficient(t *testing.T) {
	// The second column is twice the first, so the columns are dependent and R has a zero pivot.
	a := FromRows([]float64{
		1, 2,
		2, 4,
		3, 6,
	}, 3, 2)
	if _, err := LeastSquares(a, []float64{1, 2, 3}); !errors.Is(err, ErrSingular) {
		t.Fatalf("LeastSquares on dependent columns returned %v, want ErrSingular", err)
	}
}

func TestEigenSymTwoByTwoAnalytic(t *testing.T) {
	// [[2,1],[1,2]] has eigenvalues 1 and 3, with eigenvectors (1,-1)/√2 and (1,1)/√2.
	a := FromRows([]float64{2, 1, 1, 2}, 2, 2)
	values, vectors, converged := EigenSym(a, 0)
	if !converged {
		t.Fatal("EigenSym did not converge on a 2x2 problem")
	}

	// The eigenvalue multiset must be {1, 3}, in whatever order the rotations produced.
	sorted := append([]float64(nil), values...)
	sort.Float64s(sorted)
	if math.Abs(sorted[0]-1) > 1e-12 || math.Abs(sorted[1]-3) > 1e-12 {
		t.Fatalf("eigenvalues = %v, want {1 3}", sorted)
	}

	for j := 0; j < 2; j++ {
		v := []float64{vectors.At(0, j), vectors.At(1, j)}
		av := MulVec(a, v)
		for i := 0; i < 2; i++ {
			if math.Abs(av[i]-values[j]*v[i]) > 1e-12 {
				t.Fatalf("A v != λ v for eigenpair %d", j)
			}
		}
		// The analytic eigenvector for this eigenvalue, up to sign.
		var want []float64
		if math.Abs(values[j]-3) < 1e-9 {
			want = []float64{1 / math.Sqrt2, 1 / math.Sqrt2}
		} else {
			want = []float64{1 / math.Sqrt2, -1 / math.Sqrt2}
		}
		same := math.Abs(v[0]-want[0]) < 1e-9 && math.Abs(v[1]-want[1]) < 1e-9
		negated := math.Abs(v[0]+want[0]) < 1e-9 && math.Abs(v[1]+want[1]) < 1e-9
		if !same && !negated {
			t.Fatalf("eigenvector for λ=%v is %v, want ±%v", values[j], v, want)
		}
	}
}

// TestEigenSymResiduals is the main correctness test: for every eigenpair, A v must equal λ v, and
// the eigenvalues must reproduce the trace and determinant.
func TestEigenSymResiduals(t *testing.T) {
	for _, n := range []int{1, 2, 3, 5, 8, 13} {
		a := spd(n, float64(n)+5)
		values, vectors, converged := EigenSym(a, 0)
		if !converged {
			t.Fatalf("n=%d did not converge", n)
		}
		if len(values) != n || vectors.Rows != n || vectors.Cols != n {
			t.Fatalf("n=%d: shapes %d, %dx%d", n, len(values), vectors.Rows, vectors.Cols)
		}

		for j := 0; j < n; j++ {
			v := make([]float64, n)
			for i := 0; i < n; i++ {
				v[i] = vectors.Data[i*n+j]
			}
			// The eigenvectors must be unit length.
			if math.Abs(Norm2(v)-1) > 1e-9 {
				t.Fatalf("n=%d eigenvector %d has norm %v", n, j, Norm2(v))
			}
			av := MulVec(a, v)
			for i := 0; i < n; i++ {
				if math.Abs(av[i]-values[j]*v[i]) > 1e-9 {
					t.Fatalf("n=%d: A v != λ v for eigenpair %d (component %d)", n, j, i)
				}
			}
		}

		// The sum of the eigenvalues is the trace, and their product is the determinant. These
		// catch a set of values that satisfies A v = λ v only because each λ was matched to the
		// wrong vector.
		var sum float64
		for _, v := range values {
			sum += v
		}
		if math.Abs(sum-Trace(a)) > 1e-8*math.Max(1, math.Abs(Trace(a))) {
			t.Fatalf("n=%d: sum of eigenvalues %v, want trace %v", n, sum, Trace(a))
		}
	}
}

// TestEigenSymDiagonalMatrix pins the trivial case: a diagonal matrix's eigenvalues are its
// diagonal, in some order.
func TestEigenSymDiagonalMatrix(t *testing.T) {
	a := FromRows([]float64{5, 0, 0, 0, -2, 0, 0, 0, 7}, 3, 3)
	values, _, converged := EigenSym(a, 0)
	if !converged {
		t.Fatal("EigenSym did not converge on a diagonal matrix")
	}
	sort.Float64s(values)
	want := []float64{-2, 5, 7}
	for i := range want {
		if math.Abs(values[i]-want[i]) > 1e-12 {
			t.Fatalf("eigenvalues = %v, want %v", values, want)
		}
	}
}

func TestLinalgEmptyAndPanics(t *testing.T) {
	z := New(0, 0)
	if x, err := Solve(z, nil); err != nil || len(x) != 0 {
		t.Errorf("Solve of the 0x0 system = %v, %v", x, err)
	}
	if inv, err := Inverse(z); err != nil || inv.Rows != 0 {
		t.Errorf("Inverse of the 0x0 matrix = %v, %v", inv, err)
	}
	if l, err := Cholesky(z); err != nil || l.Rows != 0 {
		t.Errorf("Cholesky of the 0x0 matrix = %v, %v", l, err)
	}
	v, vec, ok := EigenSym(z, 0)
	if !ok || len(v) != 0 || vec.Rows != 0 {
		t.Errorf("EigenSym of the 0x0 matrix = %v, %v, %v", v, vec, ok)
	}
	if x, err := LeastSquares(New(3, 0), []float64{1, 2, 3}); err != nil || x != nil {
		t.Errorf("LeastSquares with no columns = %v, %v", x, err)
	}

	a := filled(3, 4, 1)
	sq := filled(3, 3, 1)
	for _, tc := range []struct {
		name string
		fn   func()
	}{
		{"Solve non-square", func() { Solve(a, make([]float64, 3)) }},
		{"Solve rhs length", func() { Solve(sq, make([]float64, 2)) }},
		{"Inverse non-square", func() { Inverse(a) }},
		{"Cholesky non-square", func() { Cholesky(a) }},
		{"CholeskySolve non-square", func() { CholeskySolve(a, make([]float64, 3)) }},
		{"CholeskySolve rhs", func() { CholeskySolve(sq, make([]float64, 2)) }},
		{"QR wide", func() { QR(FromRows([]float64{1, 2, 3, 4, 5, 6}, 2, 3)) }},
		{"LeastSquares wide", func() { LeastSquares(FromRows([]float64{1, 2, 3, 4, 5, 6}, 2, 3), []float64{1, 2}) }},
		{"LeastSquares rhs", func() { LeastSquares(FromRows([]float64{1, 0, 0, 1, 1, 1}, 3, 2), []float64{1, 2}) }},
		{"EigenSym non-square", func() { EigenSym(a, 0) }},
	} {
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

// TestDotLowerSpansTheUnrollBoundary checks the dot-product kernel directly at the sizes where its two
// paths meet, which is the only place the boundary can be wrong.
func TestDotLowerSpansTheUnrollBoundary(t *testing.T) {
	for _, n := range []int{16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 31, 32, 33} {
		a := spd(n, float64(n)+7)
		l, err := Cholesky(a)
		if err != nil {
			t.Fatalf("n=%d: Cholesky: %v", n, err)
		}
		// Reconstruct A = L Lᵀ, which is what every dotLower call feeds into.
		got := Mul(l, Transpose(l))
		matClose(t, "L Lᵀ", got, a, 1e-10)
	}

	// And the kernel itself must equal the plain dot product at every j, including the ones the
	// unrolled loop handles and the remainders it leaves.
	a := spd(40, 3)
	l, err := Cholesky(a)
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 40; i++ {
		for j := 0; j <= i; j++ {
			var want float64
			for k := 0; k < j; k++ {
				want += l.Data[i*40+k] * l.Data[j*40+k]
			}
			if got := dotLower(l, i, j); math.Abs(got-want) > 1e-12 {
				t.Fatalf("dotLower(i=%d, j=%d) = %v, want %v", i, j, got, want)
			}
		}
	}
}

// TestEigenSymEigenvectorOrientation is the orientation check for the transposed accumulation.
//
// EigenSym builds its eigenvectors transposed -- it updates contiguous rows rather than columns at
// stride n -- and transposes back at the end. An error there would still satisfy the eigenvalue
// assertions, because the eigenvalues of a symmetric matrix are unchanged by it, and it would still
// satisfy most of the per-pair checks if the matrix happened to be diagonal. The defining property
// VᵀAV = diag(λ) is what pins the orientation: a transposed V gives V A Vᵀ, which is diagonal only
// when the matrix is.
func TestEigenSymEigenvectorOrientation(t *testing.T) {
	for _, n := range []int{1, 2, 3, 5, 8, 17, 33} {
		a := spd(n, float64(n)+11)
		values, vectors, converged := EigenSym(a, 0)
		if !converged {
			t.Fatalf("n=%d: did not converge", n)
		}

		// Vᵀ A V must be diagonal, with the eigenvalues on the diagonal in the returned order.
		vav := Mul(Mul(Transpose(vectors), a), vectors)
		for i := 0; i < n; i++ {
			for j := 0; j < n; j++ {
				got := vav.At(i, j)
				if i == j {
					if math.Abs(got-values[i]) > 1e-8*math.Max(1, math.Abs(values[i])) {
						t.Fatalf("n=%d: (VᵀAV)[%d,%d] = %v, want eigenvalue %v", n, i, i, got, values[i])
					}
					continue
				}
				if math.Abs(got) > 1e-8 {
					t.Fatalf("n=%d: (VᵀAV)[%d,%d] = %v, want 0 off the diagonal", n, i, j, got)
				}
			}
		}
	}
}

// TestEigenSymSmallSizes pins the degenerate cases, where the accumulation loop never runs and the
// transpose at the end is the only thing that happens.
func TestEigenSymSmallSizes(t *testing.T) {
	for _, n := range []int{0, 1, 2} {
		a := spd(n, 3)
		values, vectors, converged := EigenSym(a, 0)
		if !converged {
			t.Fatalf("n=%d: did not converge", n)
		}
		if len(values) != n || vectors.Rows != n || vectors.Cols != n {
			t.Fatalf("n=%d: shapes %d, %dx%d", n, len(values), vectors.Rows, vectors.Cols)
		}
		// For n = 1 the eigenvector must be ±1, not 0.
		if n == 1 && math.Abs(math.Abs(vectors.At(0, 0))-1) > 1e-12 {
			t.Fatalf("n=1: eigenvector = %v, want ±1", vectors.At(0, 0))
		}
	}
}
