package mat

import (
	"math"
)

// ---------------------------------------------------------------------------
// Factorisations and solves.
//
// Three factorisations, chosen by what the caller knows about the matrix:
//
//	LU with partial pivoting   any square matrix         A = P^-1 L U
//	Cholesky                   symmetric positive def.   A = L Lᵀ
//	Householder QR             any shape                 A = Q R
//
// The choice is not cosmetic. Cholesky is roughly half the work of LU and needs no pivoting, so
// it is the right answer whenever the matrix is known to be positive definite -- which a
// covariance or a Gram matrix is. LU is the general square case. QR is the only one of the three
// that works when the system is over-determined, which is what a regression with more rows than
// predictors is.
//
// # Errors are returned, not panicked
//
// A singular matrix and a non-positive-definite one are properties of the data, not of the call, so
// the factorisations return them. A shape mismatch is a property of the call, so it panics, like
// every other shape check in this package.
// ---------------------------------------------------------------------------

// luFactor factors a square matrix in place into an LU pair with partial pivoting, returning the
// permutation applied as a row-index slice and whether the matrix was non-singular.
//
// The returned slice holds, for each column k, the row that was swapped with row k. It is enough
// to apply the same permutation to a right-hand side, which is all the solvers need.
func luFactor(a *Mat) (pivots []int, ok bool) {
	n := a.Rows
	pivots = make([]int, n)
	ok = true
	for k := 0; k < n; k++ {
		// Partial pivoting: take the largest magnitude in the column as the pivot, so a small
		// diagonal entry does not amplify rounding error through the whole elimination.
		pivotRow := k
		best := math.Abs(a.Data[k*n+k])
		for i := k + 1; i < n; i++ {
			if v := math.Abs(a.Data[i*n+k]); v > best {
				best = v
				pivotRow = i
			}
		}
		pivots[k] = pivotRow
		if best == 0 {
			// The column is entirely zero below the diagonal, so the matrix is singular. The
			// factorisation continues so that the caller can still inspect it; the flag is what
			// matters.
			ok = false
			continue
		}
		if pivotRow != k {
			for j := 0; j < n; j++ {
				a.Data[k*n+j], a.Data[pivotRow*n+j] = a.Data[pivotRow*n+j], a.Data[k*n+j]
			}
		}
		pivot := a.Data[k*n+k]
		for i := k + 1; i < n; i++ {
			factor := a.Data[i*n+k] / pivot
			a.Data[i*n+k] = factor
			for j := k + 1; j < n; j++ {
				a.Data[i*n+j] -= factor * a.Data[k*n+j]
			}
		}
	}
	return pivots, ok
}

// luSolve solves a x = b given a factorisation produced by luFactor, applying the same row
// permutation to b.
func luSolve(lu *Mat, pivots []int, b []float64) []float64 {
	n := lu.Rows
	x := make([]float64, n)
	copy(x, b)
	for k := 0; k < n; k++ {
		if p := pivots[k]; p != k {
			x[k], x[p] = x[p], x[k]
		}
	}
	// Forward substitution through the unit-lower-triangular L.
	for i := 0; i < n; i++ {
		sum := x[i]
		for j := 0; j < i; j++ {
			sum -= lu.Data[i*n+j] * x[j]
		}
		x[i] = sum
	}
	// Back substitution through U.
	for i := n - 1; i >= 0; i-- {
		sum := x[i]
		for j := i + 1; j < n; j++ {
			sum -= lu.Data[i*n+j] * x[j]
		}
		d := lu.Data[i*n+i]
		if d == 0 {
			// Singular: the solution is not unique. Returning the partial result would hide
			// that, so a zero divisor produces a non-finite entry, which the caller can see.
			x[i] = math.NaN()
			continue
		}
		x[i] = sum / d
	}
	return x
}

// Solve returns the solution of the square system a x = b.
//
// It panics if a is not square or len(b) != a.Rows, and returns [ErrSingular] if a has no inverse.
// The right-hand side is not modified.
func Solve(a *Mat, b []float64) ([]float64, error) {
	if !a.IsSquare() {
		panic("mat: Solve requires a square matrix")
	}
	if len(b) != a.Rows {
		panic("mat: Solve right-hand side length mismatch")
	}
	if a.Rows == 0 {
		return nil, nil
	}
	lu := a.Clone()
	pivots, ok := luFactor(lu)
	if !ok {
		return nil, ErrSingular
	}
	return luSolve(lu, pivots, b), nil
}

// Inverse returns the inverse of a square matrix.
//
// It is computed by solving a x = e_j for each unit vector, which reuses one factorisation rather
// than doing n independent solves. It returns [ErrSingular] if a has no inverse.
//
// Inverting a matrix in order to solve a system is the wrong tool -- it costs more and is less
// accurate than [Solve] -- so this exists for the cases that genuinely need the inverse, such as
// forming a projection or a covariance of a linear map.
func Inverse(a *Mat) (*Mat, error) {
	if !a.IsSquare() {
		panic("mat: Inverse requires a square matrix")
	}
	n := a.Rows
	out := New(n, n)
	if n == 0 {
		return out, nil
	}
	lu := a.Clone()
	pivots, ok := luFactor(lu)
	if !ok {
		return nil, ErrSingular
	}
	b := make([]float64, n)
	for j := 0; j < n; j++ {
		for i := range b {
			b[i] = 0
		}
		b[j] = 1
		col := luSolve(lu, pivots, b)
		for i := 0; i < n; i++ {
			out.Data[i*n+j] = col[i]
		}
	}
	return out, nil
}

// Cholesky returns the lower-triangular L with A = L Lᵀ for a symmetric positive-definite matrix.
//
// It reads only the lower triangle of a, so a caller who has built a symmetric matrix from one
// triangle does not have to mirror it first; [IsSymmetric] is provided for the case where the
// caller wants to check.
//
// It returns [ErrNotSPD] if a diagonal entry is not positive, which is exactly the condition that
// the matrix is not positive definite. The partially factored result is discarded rather than
// returned, because a half-factored matrix is not a usable answer.
func Cholesky(a *Mat) (*Mat, error) {
	if !a.IsSquare() {
		panic("mat: Cholesky requires a square matrix")
	}
	n := a.Rows
	l := New(n, n)
	for i := 0; i < n; i++ {
		for j := 0; j <= i; j++ {
			sum := a.Data[i*n+j] - dotLower(l, i, j)
			if i == j {
				if sum <= 0 || sum != sum {
					return nil, ErrNotSPD
				}
				l.Data[i*n+j] = math.Sqrt(sum)
				continue
			}
			l.Data[i*n+j] = sum / l.Data[j*n+j]
		}
	}
	return l, nil
}

// dotLower returns the dot product of rows i and j of the lower-triangular factor over the first j
// columns, which is the correction the Cholesky recurrence subtracts.
//
// # Why there are four accumulators
//
// The textbook form of this loop is `for k { sum -= l[i][k]*l[j][k] }`, and that is a *reduction*:
// every iteration depends on the previous one's result, so the loop runs at the latency of a
// fused multiply-add rather than at the machine's throughput. The LU kernel next door has no such
// chain -- each column of its update is independent -- which is why the factorisation that does
// *twice* the arithmetic was measuring within 25% of this one instead of twice as slow.
//
// Four independent accumulators give the processor four chains to interleave, so the loop becomes
// throughput-bound instead of latency-bound. The summation order changes, which this repository
// permits: results are not promised to be bit-identical across architectures.
//
// The rows are contiguous, so both loads in each iteration are sequential.
func dotLower(l *Mat, i, j int) float64 {
	if j == 0 {
		return 0
	}
	n := l.Cols

	// Short rows are left to the simple loop, on direct indices rather than slices: the
	// four-accumulator prologue costs more than the latency it hides when there are only a handful of
	// products, and constructing the two slice headers costs more still at a size where the whole
	// call is a few nanoseconds. Measured, that was the difference between an n = 16 factorisation
	// matching its old time and being 15% slower.
	const unrollFrom = 16
	if j < unrollFrom {
		var sum float64
		ib, jb := i*n, j*n
		for k := 0; k < j; k++ {
			sum += l.Data[ib+k] * l.Data[jb+k]
		}
		return sum
	}

	li := l.Data[i*n : i*n+j]
	lj := l.Data[j*n : j*n+j]

	var s0, s1, s2, s3 float64
	k := 0
	for ; k+4 <= j; k += 4 {
		s0 += li[k] * lj[k]
		s1 += li[k+1] * lj[k+1]
		s2 += li[k+2] * lj[k+2]
		s3 += li[k+3] * lj[k+3]
	}
	for ; k < j; k++ {
		s0 += li[k] * lj[k]
	}
	// Pairing the partial sums keeps the final combination short and balanced.
	return (s0 + s1) + (s2 + s3)
}

// CholeskySolve solves A x = b given the lower-triangular factor L of A, by one forward and one
// back substitution.
//
// It panics unless l is square and len(b) == l.Rows.
func CholeskySolve(l *Mat, b []float64) []float64 {
	if !l.IsSquare() {
		panic("mat: CholeskySolve requires a square factor")
	}
	if len(b) != l.Rows {
		panic("mat: CholeskySolve right-hand side length mismatch")
	}
	n := l.Rows
	y := make([]float64, n)
	// Forward substitution through L.
	for i := 0; i < n; i++ {
		sum := b[i]
		for j := 0; j < i; j++ {
			sum -= l.Data[i*n+j] * y[j]
		}
		y[i] = sum / l.Data[i*n+i]
	}
	// Back substitution through Lᵀ.
	x := make([]float64, n)
	for i := n - 1; i >= 0; i-- {
		sum := y[i]
		for j := i + 1; j < n; j++ {
			sum -= l.Data[j*n+i] * x[j]
		}
		x[i] = sum / l.Data[i*n+i]
	}
	return x
}

// qrReduce applies Householder reflectors to r in place, zeroing its strict lower triangle and
// leaving the upper triangle holding R.
//
// It exists so that the reflectors can be applied to different things at once, and in particular so
// that a least-squares solve can apply them to the right-hand side *without forming Q*. That
// distinction is not cosmetic: accumulating an m x m Q costs O(n*m^2) for an m x n system, which for
// a tall regression dominates everything else. Applying Q^T b directly costs O(m*n) and produces
// the same vector.
//
//   - r must be m x n with m >= n and is overwritten with R in its upper triangle.
//   - q, when non-nil, must be the m x m identity and accumulates Q.
//   - b, when non-nil, must have m elements and receives Q^T b.
func qrReduce(r, q *Mat, b []float64) {
	m, n := r.Rows, r.Cols
	ref := make([]float64, m)
	for k := 0; k < n; k++ {
		var norm float64
		for i := k; i < m; i++ {
			norm += r.Data[i*n+k] * r.Data[i*n+k]
		}
		norm = math.Sqrt(norm)
		if norm == 0 {
			continue // the column is already zero below the diagonal
		}
		alpha := r.Data[k*n+k]
		if alpha >= 0 {
			norm = -norm
		}
		// v = x - norm*e_k, the reflector that maps the column onto -norm*e_k.
		for i := 0; i < m; i++ {
			ref[i] = 0
		}
		for i := k; i < m; i++ {
			ref[i] = r.Data[i*n+k]
		}
		ref[k] -= norm
		var vnorm float64
		for i := k; i < m; i++ {
			vnorm += ref[i] * ref[i]
		}
		if vnorm == 0 {
			continue
		}
		scale := 2 / vnorm

		// R <- H R, over the columns the reflector can still affect.
		for j := k; j < n; j++ {
			var dot float64
			for i := k; i < m; i++ {
				dot += ref[i] * r.Data[i*n+j]
			}
			dot *= scale
			for i := k; i < m; i++ {
				r.Data[i*n+j] -= dot * ref[i]
			}
		}

		// Q <- Q H, accumulated on the right so no separate transpose is needed.
		if q != nil {
			for i := 0; i < m; i++ {
				var dot float64
				for j := k; j < m; j++ {
					dot += q.Data[i*m+j] * ref[j]
				}
				dot *= scale
				for j := k; j < m; j++ {
					q.Data[i*m+j] -= dot * ref[j]
				}
			}
		}

		// b <- H b, which is applying Q^T incrementally instead of forming Q.
		if b != nil {
			var dot float64
			for i := k; i < m; i++ {
				dot += ref[i] * b[i]
			}
			dot *= scale
			for i := k; i < m; i++ {
				b[i] -= dot * ref[i]
			}
		}
	}
}

// QR returns the economy-size Householder factorisation A = Q R.
//
// For an m x n matrix with m >= n, Q is m x n with orthonormal columns and R is n x n upper
// triangular. The economy size is what least squares needs; a full m x m Q would carry m-n columns
// of information that nothing in this package uses.
//
// # It forms Q explicitly, which is the expensive part
//
// Accumulating Q costs O(n*m^2) for an m x n input, against O(m*n^2) for the reduction itself. For a
// square matrix that is the same order and does not matter, but for a tall-skinny matrix it
// dominates completely. A caller who wants the factors should use this function; a caller who wants
// to *solve* a least-squares problem should use [LeastSquares], which applies the reflectors to the
// right-hand side and never forms Q.
//
// It panics if a.Rows < a.Cols, since the factorisation as defined here requires at least as many
// rows as columns.
func QR(a *Mat) (q, r *Mat) {
	if a.Rows < a.Cols {
		panic("mat: QR requires at least as many rows as columns")
	}
	m, n := a.Rows, a.Cols
	r = a.Clone()
	full := Identity(m)
	qrReduce(r, full, nil)

	// Zero the strict lower triangle, which the reflectors leave holding round-off.
	for i := 1; i < n; i++ {
		for j := 0; j < i; j++ {
			r.Data[i*n+j] = 0
		}
	}
	// And shrink both factors to the economy size.
	qThin := New(m, n)
	for i := 0; i < m; i++ {
		for j := 0; j < n; j++ {
			qThin.Data[i*n+j] = full.Data[i*m+j]
		}
	}
	rThin := New(n, n)
	for i := 0; i < n; i++ {
		for j := 0; j < n; j++ {
			rThin.Data[i*n+j] = r.Data[i*n+j]
		}
	}
	return qThin, rThin
}

// LeastSquares returns the x minimising ||A x - b||2 for a matrix with at least as many rows as
// columns.
//
// It solves through the QR factorisation rather than the normal equations, which squares the
// conditioning of the problem but not, unlike forming A^T A explicitly, its condition *number* --
// that is the reason to use QR, and it matters whenever the columns are close to collinear.
//
// The reflectors are applied directly to the right-hand side, so Q is never formed and the cost is
// O(m*n^2) rather than the O(n*m^2) that [QR] pays. For a regression with many rows and few
// predictors that is the difference between usable and not.
//
// It panics if a.Rows < a.Cols or len(b) != a.Rows, and returns [ErrSingular] if a diagonal entry
// of R is negligible relative to the largest one, which means the columns are linearly dependent
// and the solution is not unique. The comparison is a rank tolerance rather than a test for exact
// zero, because the reflectors leave round-off where an exactly dependent column should produce a
// zero pivot; treating that round-off as a real pivot yields enormous values that are not a
// solution to anything.
func LeastSquares(a *Mat, b []float64) ([]float64, error) {
	if a.Rows < a.Cols {
		panic("mat: LeastSquares requires at least as many rows as columns")
	}
	if len(b) != a.Rows {
		panic("mat: LeastSquares right-hand side length mismatch")
	}
	m, n := a.Rows, a.Cols
	if n == 0 {
		return nil, nil
	}

	r := a.Clone()
	y := make([]float64, m)
	copy(y, b)
	qrReduce(r, nil, y)

	// Rank tolerance: a pivot is treated as zero when it is negligible relative to the size of
	// the problem, which is the standard rank-revealing criterion.
	//
	// Testing for exactly zero does not work. The reflectors leave round-off below the diagonal,
	// so exactly dependent columns produce a pivot around machineEpsilon * ||A|| rather than 0,
	// and dividing by it returns a solution with entries around 1e14 -- a number that looks like
	// an answer and is not one.
	//
	// # Why the scale is ||A||_F and not |R[0][0]|
	//
	// For the 3x2 exactly-dependent matrix [[1,2],[2,4],[3,6]] the residual pivot measured
	// 0.86 * eps * ||A||_F on arm64 and 1.07 * eps * ||A||_F on amd64: the round-off sits *at*
	// one epsilon of the matrix norm, and it differs by architecture because the reflector loop
	// may contract into an FMA on one and not the other. An earlier tolerance of n * eps * |R00|
	// landed between those two values and classified the same matrix differently on the two
	// targets.
	//
	// Scaling by max(m,n) * eps * ||A||_F gives a few times that headroom while still declaring
	// rank deficiency for any matrix whose condition number approaches 1/eps, which is the
	// correct boundary: such a matrix has no numerically meaningful solution to report.
	var fro float64
	for _, v := range a.Data {
		fro += v * v
	}
	fro = math.Sqrt(fro)
	if fro == 0 {
		return nil, ErrSingular
	}
	tol := float64(max(m, n)) * machineEpsilon * fro

	// Back substitution through the n x n upper-triangular block of R.
	x := make([]float64, n)
	for i := n - 1; i >= 0; i-- {
		sum := y[i]
		for j := i + 1; j < n; j++ {
			sum -= r.Data[i*n+j] * x[j]
		}
		d := r.Data[i*n+i]
		if math.Abs(d) <= tol {
			return nil, ErrSingular
		}
		x[i] = sum / d
	}
	return x, nil
}

// machineEpsilon is the difference between 1 and the next representable float64, written as a
// constant because the standard library does not export one.
const machineEpsilon = 1.0 / (1 << 52)

// EigenSym returns the eigenvalues and eigenvectors of a symmetric matrix using the cyclic Jacobi
// method.
//
// The eigenvalues are returned in the order the method converges to them, which is not sorted, and
// the eigenvectors are the *columns* of the returned matrix. Both are correct to the iteration's
// tolerance, which for a well-scaled matrix is near machine precision -- Jacobi is unusually
// accurate for a small dense symmetric problem, which is why it is used here rather than a
// tridiagonal reduction.
//
// It panics if the matrix is not square. A non-symmetric matrix is not rejected, because checking
// costs a pass and the method silently converges to something meaningless on one; the caller is
// expected to know, and [IsSymmetric] is available to confirm.
//
// It is iterative: the return value reports whether the off-diagonal mass reached the tolerance
// within the sweep cap. A false result still returns the partial decomposition, which is usually
// close, but a caller who depends on the accuracy should check.
//
// A tolerance of zero or less selects 1e-24 relative to the matrix's diagonal mass, which is near
// machine precision for a well-scaled problem.
func EigenSym(a *Mat, tolerance float64) (values []float64, vectors *Mat, converged bool) {
	if !a.IsSquare() {
		panic("mat: EigenSym requires a square matrix")
	}
	n := a.Rows
	values = make([]float64, n)
	if n == 0 {
		return values, Identity(0), true
	}

	// The eigenvectors are accumulated **transposed**, so that a rotation updates two contiguous
	// rows instead of two columns at stride n. Columns of V are what the algorithm updates, and a
	// column walk of an n x n matrix touches n cache lines; a row walk touches n/16. The matrix is
	// transposed back once at the end, which is O(n^2) against the O(n^3) the rotations cost.
	//
	// Measured, this is the single largest term in the rotation: at n = 128 each rotation touches 256
	// cache lines of eigenvector storage in the untransposed form against 32 here.
	vt := Identity(n)
	vectors = Identity(n)

	work := a.Clone()
	if tolerance <= 0 {
		tolerance = 1e-24
	}
	const maxSweeps = 100
	converged = false
	for sweep := 0; sweep < maxSweeps; sweep++ {
		off := 0.0
		diag := 0.0
		for p := 0; p < n; p++ {
			diag += work.Data[p*n+p] * work.Data[p*n+p]
			for q := p + 1; q < n; q++ {
				off += work.Data[p*n+q] * work.Data[p*n+q]
			}
		}
		// The criterion is relative to the diagonal mass, so it does not depend on the scale of
		// the matrix: a covariance matrix in units of 1e6 and one in units of 1e-6 both converge
		// at the same number of sweeps.
		if off <= tolerance*diag || off == 0 {
			converged = true
			break
		}
		for p := 0; p < n; p++ {
			for q := p + 1; q < n; q++ {
				apq := work.Data[p*n+q]
				if math.Abs(apq) < 1e-300 {
					continue
				}
				app := work.Data[p*n+p]
				aqq := work.Data[q*n+q]
				// The rotation angle that zeroes (p, q).
				theta := (aqq - app) / (2 * apq)
				t := math.Copysign(1, theta) / (math.Abs(theta) + math.Sqrt(theta*theta+1))
				c := 1 / math.Sqrt(t*t+1)
				s := t * c

				// Apply the rotation to the working matrix: A <- Jᵀ A J.
				for i := 0; i < n; i++ {
					aip := work.Data[i*n+p]
					aiq := work.Data[i*n+q]
					work.Data[i*n+p] = c*aip - s*aiq
					work.Data[i*n+q] = s*aip + c*aiq
				}
				for i := 0; i < n; i++ {
					api := work.Data[p*n+i]
					aqi := work.Data[q*n+i]
					work.Data[p*n+i] = c*api - s*aqi
					work.Data[q*n+i] = s*api + c*aqi
				}
				// And accumulate the rotation into the eigenvectors, in transposed storage.
				pRow := vt.Data[p*n : p*n+n]
				qRow := vt.Data[q*n : q*n+n]
				for i := 0; i < n; i++ {
					vpi := pRow[i]
					vqi := qRow[i]
					pRow[i] = c*vpi - s*vqi
					qRow[i] = s*vpi + c*vqi
				}
			}
		}
	}
	for i := 0; i < n; i++ {
		values[i] = work.Data[i*n+i]
	}
	// Transpose the accumulated storage into the column-major-by-convention result: the eigenvectors
	// are the *columns* of the returned matrix.
	for i := 0; i < n; i++ {
		for j := 0; j < n; j++ {
			vectors.Data[i*n+j] = vt.Data[j*n+i]
		}
	}
	return values, vectors, converged
}
