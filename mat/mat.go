// Package mat provides dense linear algebra for the numa engine: matrix products, triangular
// and orthogonal factorisations, linear solves, least squares and a symmetric eigensolver.
//
// It is the last layer of the compute engine, below the indicator library, and it exists for the
// parts of quantitative work that are not elementwise: regression with more than one predictor,
// portfolio construction, risk models, and any filtered or state-space method.
//
// # Representation
//
// A matrix is dense and row-major:
//
//	type Mat struct {
//	    Rows, Cols int
//	    Data       []float64 // len(Data) == Rows*Cols
//	}
//
// Row-major matches Go's slice model, so Row(i) is a zero-copy subslice and a matrix-vector
// product walks memory forward. The helper methods maintain the length invariant; the fields are
// exported because a caller doing something the API does not cover needs the raw buffer, and
// hiding it behind accessors would only add a call to the inner loop without preventing a
// mistake.
//
// # Errors rather than panics
//
// Shape mismatches panic, because they are programming errors and the rest of the engine treats
// them that way. Numerical failures are *returned*: a singular matrix, a factorisation that met a
// non-positive pivot, or an iteration that did not converge. Those depend on the data, not on the
// code, so a caller has to decide what they mean.
//
// # No CGo, no assembly
//
// The kernels here are Go loops arranged for the memory system, like every other kernel in this
// repository. That puts a ceiling on GEMM relative to a tuned BLAS, and the benchmarks record
// where the ceiling is rather than implying it does not exist.
package mat

import (
	"errors"
	"math"

	"github.com/rj-raghavjoshi/numa/vec"
)

// Errors returned by the factorisations. They are values rather than panics because each one
// describes the *data* rather than the call.
var (
	// ErrSingular reports a matrix that has no inverse, or a solve whose factorisation met a
	// zero pivot.
	ErrSingular = errors.New("mat: matrix is singular")

	// ErrNotSPD reports a matrix that is not symmetric positive definite, which is what a
	// Cholesky factorisation requires.
	ErrNotSPD = errors.New("mat: matrix is not symmetric positive definite")

	// ErrShape reports an operation whose operands do not fit together.
	ErrShape = errors.New("mat: shape mismatch")
)

// Mat is a dense row-major matrix of float64.
//
// Data holds Rows*Cols elements in row-major order: element (i, j) is Data[i*Cols+j]. The zero Mat
// has no rows and no columns and is valid for every operation that accepts an empty matrix.
type Mat struct {
	Rows, Cols int
	Data       []float64
}

// New returns a Rows x Cols matrix of zeros.
//
// It panics if either dimension is negative.
func New(rows, cols int) *Mat {
	if rows < 0 || cols < 0 {
		panic("mat: negative dimension")
	}
	return &Mat{Rows: rows, Cols: cols, Data: make([]float64, rows*cols)}
}

// FromRows wraps an existing slice as a Rows x Cols matrix without copying.
//
// It panics if len(data) != rows*cols. The matrix shares the caller's backing array, so writing
// through one is visible through the other; use Clone for an independent copy.
func FromRows(data []float64, rows, cols int) *Mat {
	if rows < 0 || cols < 0 || len(data) != rows*cols {
		panic("mat: FromRows length does not match the dimensions")
	}
	return &Mat{Rows: rows, Cols: cols, Data: data}
}

// Clone returns an independent copy of m.
func (m *Mat) Clone() *Mat {
	if m == nil {
		return nil
	}
	out := New(m.Rows, m.Cols)
	copy(out.Data, m.Data)
	return out
}

// At returns element (i, j). It panics if the index is out of range.
func (m *Mat) At(i, j int) float64 {
	if i < 0 || i >= m.Rows || j < 0 || j >= m.Cols {
		panic("mat: index out of range")
	}
	return m.Data[i*m.Cols+j]
}

// Set stores v at element (i, j). It panics if the index is out of range.
func (m *Mat) Set(i, j int, v float64) {
	if i < 0 || i >= m.Rows || j < 0 || j >= m.Cols {
		panic("mat: index out of range")
	}
	m.Data[i*m.Cols+j] = v
}

// Row returns row i as a subslice of the backing array, sharing memory with m.
//
// It panics if i is out of range.
func (m *Mat) Row(i int) []float64 {
	if i < 0 || i >= m.Rows {
		panic("mat: row out of range")
	}
	return m.Data[i*m.Cols : (i+1)*m.Cols]
}

// IsSquare reports whether m has as many rows as columns.
func (m *Mat) IsSquare() bool { return m.Rows == m.Cols }

// Identity returns the n x n identity matrix.
func Identity(n int) *Mat {
	m := New(n, n)
	for i := 0; i < n; i++ {
		m.Data[i*n+i] = 1
	}
	return m
}

// Fill sets every element to v.
func (m *Mat) Fill(v float64) {
	for i := range m.Data {
		m.Data[i] = v
	}
}

// Copy copies src into dst and returns dst.
//
// The shapes must match; Copy panics otherwise. Like the vec elementwise functions it is a
// programming-error check rather than a returned error.
func Copy(dst, src *Mat) *Mat {
	requireSameShape("Copy", dst, src)
	copy(dst.Data, src.Data)
	return dst
}

// Add returns a+b elementwise. It panics unless the shapes match.
//
// Add allocates; for hot loops that can supply a destination, use [AddTo].
func Add(a, b *Mat) *Mat {
	requireSameShape("Add", a, b)
	return AddTo(New(a.Rows, a.Cols), a, b)
}

// AddTo stores a+b into dst and returns dst. All three shapes must match.
func AddTo(dst, a, b *Mat) *Mat {
	requireSameShape("AddTo", dst, a)
	requireSameShape("AddTo", dst, b)
	vec.AddTo(dst.Data, a.Data, b.Data)
	return dst
}

// Sub returns a-b elementwise. It panics unless the shapes match.
func Sub(a, b *Mat) *Mat {
	requireSameShape("Sub", a, b)
	return SubTo(New(a.Rows, a.Cols), a, b)
}

// SubTo stores a-b into dst and returns dst. All three shapes must match.
func SubTo(dst, a, b *Mat) *Mat {
	requireSameShape("SubTo", dst, a)
	requireSameShape("SubTo", dst, b)
	vec.SubTo(dst.Data, a.Data, b.Data)
	return dst
}

// Scale returns a multiplied elementwise by alpha.
func Scale(a *Mat, alpha float64) *Mat {
	return ScaleTo(New(a.Rows, a.Cols), a, alpha)
}

// ScaleTo stores a*alpha into dst and returns dst.
func ScaleTo(dst, a *Mat, alpha float64) *Mat {
	requireSameShape("ScaleTo", dst, a)
	vec.ScaleTo(dst.Data, a.Data, alpha)
	return dst
}

// Transpose returns a new matrix that is m with its rows and columns exchanged.
func Transpose(m *Mat) *Mat {
	out := New(m.Cols, m.Rows)
	for i := 0; i < m.Rows; i++ {
		for j := 0; j < m.Cols; j++ {
			out.Data[j*out.Cols+i] = m.Data[i*m.Cols+j]
		}
	}
	return out
}

// Mul returns the matrix product a*b.
//
// It panics unless a.Cols == b.Rows. The kernel uses the i-k-j loop order, which for row-major
// storage streams a row of b and a row of the output together and touches each element of a once.
// That is the arrangement the memory system wants; a tuned BLAS would additionally block for cache
// and vectorise the inner loop, neither of which Go exposes.
//
// Mul allocates; for hot loops that can supply a destination, use [MulTo].
func Mul(a, b *Mat) *Mat {
	if a.Cols != b.Rows {
		panic("mat: Mul shape mismatch")
	}
	return MulTo(New(a.Rows, b.Cols), a, b)
}

// MulTo stores the product a*b into dst and returns dst.
//
// dst must have a.Rows rows and b.Cols columns, and a.Cols must equal b.Rows. MulTo panics
// otherwise. dst must not alias a or b.
func MulTo(dst, a, b *Mat) *Mat {
	if a.Cols != b.Rows || dst.Rows != a.Rows || dst.Cols != b.Cols {
		panic("mat: MulTo shape mismatch")
	}
	rows, inner, cols := a.Rows, a.Cols, b.Cols
	for i := range dst.Data {
		dst.Data[i] = 0
	}
	for i := 0; i < rows; i++ {
		aRow := a.Data[i*inner : (i+1)*inner]
		cRow := dst.Data[i*cols : (i+1)*cols]

		// Four rows of b are consumed per pass over the output row. The inner loop's cost is not the
		// multiply, it is the traffic: the un-unrolled form does one fused multiply-add per two loads
		// and a store, and measured, that leaves it at roughly half the machine's scalar FMA rate.
		// Accumulating four rows at once raises the ratio to four multiplies per five loads.
		//
		// The summation order changes, which this repository permits: results are not promised to be
		// bit-identical across architectures.
		k := 0
		for ; k+4 <= inner; k += 4 {
			a0, a1, a2, a3 := aRow[k], aRow[k+1], aRow[k+2], aRow[k+3]
			if a0 == 0 && a1 == 0 && a2 == 0 && a3 == 0 {
				continue
			}
			b0 := b.Data[k*cols : (k+1)*cols]
			b1 := b.Data[(k+1)*cols : (k+2)*cols]
			b2 := b.Data[(k+2)*cols : (k+3)*cols]
			b3 := b.Data[(k+3)*cols : (k+4)*cols]
			for j := 0; j < cols; j++ {
				cRow[j] += a0*b0[j] + a1*b1[j] + a2*b2[j] + a3*b3[j]
			}
		}
		for ; k < inner; k++ {
			av := aRow[k]
			if av == 0 {
				continue
			}
			bRow := b.Data[k*cols : (k+1)*cols]
			for j := 0; j < cols; j++ {
				cRow[j] += av * bRow[j]
			}
		}
	}
	return dst
}

// MulVec returns the matrix-vector product a*x as a new slice.
//
// It panics unless len(x) == a.Cols. MulVec allocates; use [MulVecTo] to supply a destination.
func MulVec(a *Mat, x []float64) []float64 {
	if len(x) != a.Cols {
		panic("mat: MulVec shape mismatch")
	}
	return MulVecTo(make([]float64, a.Rows), a, x)
}

// MulVecTo stores a*x into dst and returns dst.
//
// dst must have a.Rows elements and len(x) must equal a.Cols. MulVecTo panics otherwise. dst must
// not alias x.
func MulVecTo(dst []float64, a *Mat, x []float64) []float64 {
	if len(x) != a.Cols || len(dst) != a.Rows {
		panic("mat: MulVecTo shape mismatch")
	}
	for i := 0; i < a.Rows; i++ {
		dst[i] = vec.Dot(a.Data[i*a.Cols:(i+1)*a.Cols], x)
	}
	return dst
}

// Dot returns the inner product of two vectors, delegating to the tuned vec kernel.
func Dot(x, y []float64) float64 { return vec.Dot(x, y) }

// Axpy returns y + alpha*x, elementwise.
//
// It panics unless the lengths match. Axpy allocates; use [AxpyTo] for a destination.
func Axpy(alpha float64, x, y []float64) []float64 {
	if len(x) != len(y) {
		panic("mat: Axpy length mismatch")
	}
	out := make([]float64, len(y))
	copy(out, y)
	return AxpyTo(alpha, x, out)
}

// AxpyTo returns y + alpha*x with the result written into dst.
//
// All three lengths must match. dst may alias x or y.
func AxpyTo(alpha float64, x, dst []float64) []float64 {
	if len(x) != len(dst) {
		panic("mat: AxpyTo length mismatch")
	}
	for i := range dst {
		dst[i] += alpha * x[i]
	}
	return dst
}

// Norm2 returns the Euclidean norm of a vector.
func Norm2(x []float64) float64 { return math.Sqrt(vec.Dot(x, x)) }

// NormInf returns the maximum absolute row sum of a matrix, the induced infinity norm.
func NormInf(m *Mat) float64 {
	var best float64
	for i := 0; i < m.Rows; i++ {
		var sum float64
		for j := 0; j < m.Cols; j++ {
			sum += math.Abs(m.Data[i*m.Cols+j])
		}
		if sum > best {
			best = sum
		}
	}
	return best
}

// Trace returns the sum of the diagonal of a square matrix.
//
// It panics if the matrix is not square.
func Trace(m *Mat) float64 {
	if !m.IsSquare() {
		panic("mat: Trace requires a square matrix")
	}
	var t float64
	for i := 0; i < m.Rows; i++ {
		t += m.Data[i*m.Cols+i]
	}
	return t
}

// IsSymmetric reports whether m is square and equal to its transpose to within tolerance.
//
// The comparison is exact when tolerance is zero, which is the right test for a matrix that was
// constructed symmetric rather than computed.
func IsSymmetric(m *Mat, tolerance float64) bool {
	if !m.IsSquare() {
		return false
	}
	for i := 0; i < m.Rows; i++ {
		for j := i + 1; j < m.Cols; j++ {
			d := math.Abs(m.Data[i*m.Cols+j] - m.Data[j*m.Cols+i])
			a := m.Data[i*m.Cols+j]
			b := m.Data[j*m.Cols+i]
			scale := math.Max(math.Abs(a), math.Abs(b))
			if d > tolerance*math.Max(1, scale) {
				return false
			}
		}
	}
	return true
}

// requireSameShape panics unless a and b have the same dimensions.
func requireSameShape(name string, a, b *Mat) {
	if a.Rows != b.Rows || a.Cols != b.Cols {
		panic("mat: " + name + " shape mismatch")
	}
}
