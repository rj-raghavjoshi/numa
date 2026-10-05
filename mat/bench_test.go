package mat

import (
	"fmt"
	"testing"
)

// ---------------------------------------------------------------------------
// Benchmarks for the dense linear algebra kernels.
//
// GEMM is measured against an i-j-k triple loop rather than against a tuned BLAS, because that is
// the honest comparison available in pure Go: the naive form is what the tuned loop order replaces.
// The result is reported as ns/op, and the derived GFLOPS is worked out from 2*n^3 flops per
// product at report time.
//
// The factorisations are measured separately because their usefulness is not only speed: the
// interesting number is Cholesky against LU on the same matrix, which is the measurement that
// justifies having both.
// ---------------------------------------------------------------------------

var (
	benchSinkMat    *Mat
	benchSinkVec    []float64
	benchSinkScalar float64
)

func benchMatrices() []int { return []int{8, 32, 128, 256, 512} }

func matrixData(n int) *Mat {
	return randomish(n, n, uint64(n)+1)
}

func naiveMatMul(a, b *Mat) *Mat {
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

func BenchmarkMul(b *testing.B) {
	for _, n := range benchMatrices() {
		a, c := matrixData(n), matrixData(n)
		b.Run(fmt.Sprint(n), func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				benchSinkMat = Mul(a, c)
			}
		})
	}
}

func BenchmarkNaiveMul(b *testing.B) {
	for _, n := range benchMatrices() {
		a, c := matrixData(n), matrixData(n)
		b.Run(fmt.Sprint(n), func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				benchSinkMat = naiveMatMul(a, c)
			}
		})
	}
}

func BenchmarkMulVec(b *testing.B) {
	for _, n := range []int{32, 256, 1024, 4096} {
		a := matrixData(n)
		x := make([]float64, n)
		for i := range x {
			x[i] = float64(i) / float64(n)
		}
		b.Run(fmt.Sprint(n), func(b *testing.B) {
			b.SetBytes(int64(8 * n * n))
			for i := 0; i < b.N; i++ {
				benchSinkVec = MulVec(a, x)
			}
		})
	}
}

func BenchmarkTranspose(b *testing.B) {
	for _, n := range []int{64, 512, 2048} {
		a := matrixData(n)
		b.Run(fmt.Sprint(n), func(b *testing.B) {
			b.SetBytes(int64(8 * n * n))
			for i := 0; i < b.N; i++ {
				benchSinkMat = Transpose(a)
			}
		})
	}
}

// BenchmarkSolveLU and BenchmarkSolveCholesky solve the same positive-definite system, which is the
// comparison that justifies carrying both factorisations.
func BenchmarkSolveLU(b *testing.B) {
	for _, n := range []int{16, 64, 256} {
		a := spd(n, float64(n))
		bVec := rhs(n, float64(n))
		b.Run(fmt.Sprint(n), func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				x, err := Solve(a, bVec)
				if err != nil {
					b.Fatal(err)
				}
				benchSinkVec = x
			}
		})
	}
}

func BenchmarkSolveCholesky(b *testing.B) {
	for _, n := range []int{16, 64, 256} {
		a := spd(n, float64(n))
		bVec := rhs(n, float64(n))
		b.Run(fmt.Sprint(n), func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				l, err := Cholesky(a)
				if err != nil {
					b.Fatal(err)
				}
				benchSinkVec = CholeskySolve(l, bVec)
			}
		})
	}
}

func BenchmarkInverse(b *testing.B) {
	for _, n := range []int{16, 64, 256} {
		a := spd(n, float64(n))
		b.Run(fmt.Sprint(n), func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				inv, err := Inverse(a)
				if err != nil {
					b.Fatal(err)
				}
				benchSinkMat = inv
			}
		})
	}
}

func BenchmarkQR(b *testing.B) {
	for _, sh := range [][2]int{{64, 8}, {256, 16}, {512, 32}, {256, 256}} {
		a := randomish(sh[0], sh[1], 7)
		b.Run(fmt.Sprintf("%dx%d", sh[0], sh[1]), func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				q, r := QR(a)
				benchSinkMat = q
				benchSinkScalar = r.Data[0]
			}
		})
	}
}

func BenchmarkLeastSquares(b *testing.B) {
	for _, sh := range [][2]int{{1000, 5}, {10000, 20}, {1000, 50}} {
		a := randomish(sh[0], sh[1], 11)
		bVec := rhs(sh[0], 3)
		b.Run(fmt.Sprintf("%dx%d", sh[0], sh[1]), func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				x, err := LeastSquares(a, bVec)
				if err != nil {
					b.Fatal(err)
				}
				benchSinkVec = x
			}
		})
	}
}

func BenchmarkEigenSym(b *testing.B) {
	for _, n := range []int{8, 32, 64, 128} {
		a := spd(n, float64(n))
		b.Run(fmt.Sprint(n), func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				values, _, _ := EigenSym(a, 0)
				benchSinkScalar = values[0]
			}
		})
	}
}

// ---------------------------------------------------------------------------
// The scalar ceiling.
//
// GEMM is measured in GFLOPS against a naive triple loop, which says how much the loop order bought
// but not how much is left. This benchmark measures the machine's scalar fused-multiply-add rate with
// four independent chains, which is the ceiling any pure-Go kernel that cannot vectorise is bounded
// by. Comparing the two numbers is what decides whether blocking for cache is worth attempting.
// ---------------------------------------------------------------------------

func BenchmarkFMACeiling(b *testing.B) {
	const inner = 512
	// Four chains is not enough to reach the throughput ceiling on a machine whose FMA latency is
	// several cycles: the chains themselves become the limit. Eight and sixteen are measured so the
	// plateau is visible rather than assumed.
	b.Run("chains4", func(b *testing.B) {
		x0, x1, x2, x3 := 1.0000001, 1.0000002, 1.0000003, 1.0000004
		for i := 0; i < b.N; i++ {
			for j := 0; j < inner; j++ {
				x0 = x0*1.0000001 + 0.0000001
				x1 = x1*1.0000001 + 0.0000001
				x2 = x2*1.0000001 + 0.0000001
				x3 = x3*1.0000001 + 0.0000001
			}
		}
		benchSinkScalar = x0 + x1 + x2 + x3
	})
	b.Run("chains8", func(b *testing.B) {
		x0, x1, x2, x3 := 1.0000001, 1.0000002, 1.0000003, 1.0000004
		x4, x5, x6, x7 := 1.0000005, 1.0000006, 1.0000007, 1.0000008
		for i := 0; i < b.N; i++ {
			for j := 0; j < inner; j++ {
				x0 = x0*1.0000001 + 0.0000001
				x1 = x1*1.0000001 + 0.0000001
				x2 = x2*1.0000001 + 0.0000001
				x3 = x3*1.0000001 + 0.0000001
				x4 = x4*1.0000001 + 0.0000001
				x5 = x5*1.0000001 + 0.0000001
				x6 = x6*1.0000001 + 0.0000001
				x7 = x7*1.0000001 + 0.0000001
			}
		}
		benchSinkScalar = x0 + x1 + x2 + x3 + x4 + x5 + x6 + x7
	})
	b.Run("chains16", func(b *testing.B) {
		v := []float64{1.0000001, 1.0000002, 1.0000003, 1.0000004, 1.0000005, 1.0000006, 1.0000007, 1.0000008,
			1.0000009, 1.0000010, 1.0000011, 1.0000012, 1.0000013, 1.0000014, 1.0000015, 1.0000016}
		for i := 0; i < b.N; i++ {
			for j := 0; j < inner; j++ {
				for k := range v {
					v[k] = v[k]*1.0000001 + 0.0000001
				}
			}
		}
		var s float64
		for _, x := range v {
			s += x
		}
		benchSinkScalar = s
	})
}
