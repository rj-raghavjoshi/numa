package vec

import (
	"fmt"
	"math"
	"sort"
	"testing"
)

// ---------------------------------------------------------------------------
// Benchmark harness.
//
// Run with:
//
//	go test . -bench=. -benchmem -count=10
//
// then compare runs with benchstat rather than by eye:
//
//	go install golang.org/x/perf/cmd/benchstat@latest
//	go test . -bench=. -count=10 > new.txt
//	benchstat old.txt new.txt
//
// Three things make these numbers meaningful:
//
//  1. b.SetBytes reports throughput in GB/s, which is what tells you whether the
//     loop is limited by compute or by memory bandwidth. A tuned loop that is
//     already at bandwidth has nothing left to gain.
//
//  2. A small size (8) is included deliberately. The tuned paths should *lose*
//     there, because the setup and the tail loop dominate. If they win at n=8,
//     something is wrong with the measurement.
//
//  3. Portable baselines are included alongside the arch-specific paths, so that
//     "tuned beats naive" is measured on the machine actually running the test
//     rather than assumed from the build tag.
//
// The baselines are defined in this file rather than borrowed from the test
// files, so that the benchmarks compile independently of the test suite.
// ---------------------------------------------------------------------------

// benchSizes spans "fits in a register file" to "far exceeds L2 cache".
var benchSizes = []int{8, 1 << 10, 1 << 15, 1 << 22}

func benchData(n int) []float64 {
	xs := make([]float64, n)
	for i := range xs {
		// A deterministic, non-degenerate pattern. Avoiding all-equal values
		// matters for the reductions: some CPUs have data-dependent timing, and
		// all-equal inputs can be unrepresentatively fast.
		xs[i] = float64(i%17)*0.5 - 4.0
	}
	return xs
}

// sink prevents the compiler from eliminating the benchmarked call.
var sink float64

// ---------------------------------------------------------------------------
// Baselines.
//
// These are the numbers the tuned versions have to beat. They are written here,
// in the benchmark file, so they are available on every architecture and do not
// couple benchmarking to the test suite.
// ---------------------------------------------------------------------------

func naiveSum(xs []float64) float64 {
	var t float64
	for _, v := range xs {
		t += v
	}
	return t
}

func naiveDot(xs, ys []float64) float64 {
	var t float64
	for i, v := range xs {
		t += v * ys[i]
	}
	return t
}

func naiveMin(xs []float64) float64 {
	m := xs[0]
	for _, v := range xs[1:] {
		if v < m {
			m = v
		}
	}
	return m
}

func naiveMax(xs []float64) float64 {
	m := xs[0]
	for _, v := range xs[1:] {
		if v > m {
			m = v
		}
	}
	return m
}

func BenchmarkNaiveSum(b *testing.B) {
	for _, n := range benchSizes {
		xs := benchData(n)
		b.Run(fmt.Sprint(n), func(b *testing.B) {
			b.SetBytes(int64(8 * n))
			for i := 0; i < b.N; i++ {
				sink = naiveSum(xs)
			}
		})
	}
}

func BenchmarkNaiveDot(b *testing.B) {
	for _, n := range benchSizes {
		xs, ys := benchData(n), benchData(n)
		b.Run(fmt.Sprint(n), func(b *testing.B) {
			b.SetBytes(int64(16 * n)) // two reads per element
			for i := 0; i < b.N; i++ {
				sink = naiveDot(xs, ys)
			}
		})
	}
}

func BenchmarkNaiveMin(b *testing.B) {
	for _, n := range benchSizes {
		xs := benchData(n)
		b.Run(fmt.Sprint(n), func(b *testing.B) {
			b.SetBytes(int64(8 * n))
			for i := 0; i < b.N; i++ {
				sink = naiveMin(xs)
			}
		})
	}
}

func BenchmarkNaiveMax(b *testing.B) {
	for _, n := range benchSizes {
		xs := benchData(n)
		b.Run(fmt.Sprint(n), func(b *testing.B) {
			b.SetBytes(int64(8 * n))
			for i := 0; i < b.N; i++ {
				sink = naiveMax(xs)
			}
		})
	}
}

func BenchmarkNaiveAddTo(b *testing.B) {
	for _, n := range benchSizes {
		xs, ys := benchData(n), benchData(n)
		dst := make([]float64, n)
		b.Run(fmt.Sprint(n), func(b *testing.B) {
			b.SetBytes(int64(24 * n))
			for i := 0; i < b.N; i++ {
				for j := range dst {
					dst[j] = xs[j] + ys[j]
				}
			}
			sink = dst[0]
		})
	}
}

// ---------------------------------------------------------------------------
// Tuned reductions.
// ---------------------------------------------------------------------------

func BenchmarkSum(b *testing.B) {
	for _, n := range benchSizes {
		xs := benchData(n)
		b.Run(fmt.Sprint(n), func(b *testing.B) {
			b.SetBytes(int64(8 * n))
			for i := 0; i < b.N; i++ {
				sink = Sum(xs)
			}
		})
	}
}

func BenchmarkDot(b *testing.B) {
	for _, n := range benchSizes {
		xs, ys := benchData(n), benchData(n)
		b.Run(fmt.Sprint(n), func(b *testing.B) {
			b.SetBytes(int64(16 * n))
			for i := 0; i < b.N; i++ {
				sink = Dot(xs, ys)
			}
		})
	}
}

func BenchmarkSumSq(b *testing.B) {
	for _, n := range benchSizes {
		xs := benchData(n)
		b.Run(fmt.Sprint(n), func(b *testing.B) {
			b.SetBytes(int64(8 * n))
			for i := 0; i < b.N; i++ {
				sink = SumSq(xs)
			}
		})
	}
}

func BenchmarkMean(b *testing.B) {
	for _, n := range benchSizes {
		xs := benchData(n)
		b.Run(fmt.Sprint(n), func(b *testing.B) {
			b.SetBytes(int64(8 * n))
			for i := 0; i < b.N; i++ {
				sink = Mean(xs)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Scans.
//
// These are expected to scale worse than Sum. A min/max scan is latency-bound:
// each comparison needs the previous comparison's result, and there is no
// pairwise trick that shortens the chain the way `a + b` does for addition. The
// benchmarks are here to show that, not to hide it.
//
// Note the NaN check in Min/Max adds a second pass over the input. BenchmarkMinNoNaN
// and BenchmarkMinWithNaN measure that cost explicitly.
// ---------------------------------------------------------------------------

func BenchmarkMin(b *testing.B) {
	for _, n := range benchSizes {
		xs := benchData(n)
		b.Run(fmt.Sprint(n), func(b *testing.B) {
			b.SetBytes(int64(8 * n))
			for i := 0; i < b.N; i++ {
				sink = Min(xs)
			}
		})
	}
}

func BenchmarkMax(b *testing.B) {
	for _, n := range benchSizes {
		xs := benchData(n)
		b.Run(fmt.Sprint(n), func(b *testing.B) {
			b.SetBytes(int64(8 * n))
			for i := 0; i < b.N; i++ {
				sink = Max(xs)
			}
		})
	}
}

// BenchmarkMinPureScan measures the scan loop with no NaN handling at all, as the
// upper bound. Comparing it against BenchmarkMin isolates the cost of the fused NaN
// check: measured, the gap is about 1%, versus the 2x that a separate pass cost.
//
// This is the benchmark that justifies the fusion decision in nan.go. If a future
// change makes the gap large again, the fusion has been undone.
func BenchmarkMinPureScan(b *testing.B) {
	for _, n := range benchSizes {
		xs := benchData(n)
		b.Run(fmt.Sprint(n), func(b *testing.B) {
			b.SetBytes(int64(8 * n))
			for i := 0; i < b.N; i++ {
				sink = pureScanMin(xs)
			}
		})
	}
}

// naiveMinWithNaN is the plain one-accumulator scan that also honours the NaN
// policy, so the comparison against Min is apples-to-apples. The older
// BenchmarkNaiveMin did not check for NaN, which made the tuned version look
// better than it was when a separate pass was in play.
func naiveMinWithNaN(xs []float64) float64 {
	m := xs[0]
	nan := false
	for _, v := range xs {
		if v != v {
			nan = true
		}
		if v < m {
			m = v
		}
	}
	if nan {
		return math.NaN()
	}
	return m
}

// pureScanMin is the two-accumulator scan with the NaN flag removed, for isolating
// the check's cost.
func pureScanMin(xs []float64) float64 {
	n := len(xs)
	m0, m1 := xs[0], xs[0]
	i := 0
	limit := n - 1

	for i < limit {
		v0, v1 := xs[i], xs[i+1]
		if v0 < m0 {
			m0 = v0
		}
		if v1 < m1 {
			m1 = v1
		}
		i += 2
	}
	for ; i < n; i++ {
		if v := xs[i]; v < m0 {
			m0 = v
		}
	}
	if m1 < m0 {
		return m1
	}
	return m0
}

// BenchmarkNaiveMinWithNaN is the correct baseline for BenchmarkMin.
func BenchmarkNaiveMinWithNaN(b *testing.B) {
	for _, n := range benchSizes {
		xs := benchData(n)
		b.Run(fmt.Sprint(n), func(b *testing.B) {
			b.SetBytes(int64(8 * n))
			for i := 0; i < b.N; i++ {
				sink = naiveMinWithNaN(xs)
			}
		})
	}
}

// BenchmarkMinMax measures the combined scan against doing the two scans
// separately. If MinMax is not faster than two passes, the extra accumulators are
// costing more than the saved memory traffic.
func BenchmarkMinMax(b *testing.B) {
	for _, n := range benchSizes {
		xs := benchData(n)
		b.Run(fmt.Sprint(n), func(b *testing.B) {
			b.SetBytes(int64(8 * n))
			for i := 0; i < b.N; i++ {
				lo, hi := MinMax(xs)
				sink = lo + hi
			}
		})
	}
}

func BenchmarkMinThenMax(b *testing.B) {
	for _, n := range benchSizes {
		xs := benchData(n)
		b.Run(fmt.Sprint(n), func(b *testing.B) {
			b.SetBytes(int64(16 * n)) // two passes
			for i := 0; i < b.N; i++ {
				sink = Min(xs) + Max(xs)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Elementwise operations.
//
// These allocate a destination in the allocating forms, so those figures include
// allocation. The *To variants isolate the loop itself; compare the pairs to see
// how much of the cost is the loop and how much is the allocator.
// ---------------------------------------------------------------------------

func BenchmarkAdd(b *testing.B) {
	for _, n := range benchSizes {
		xs, ys := benchData(n), benchData(n)
		b.Run(fmt.Sprint(n), func(b *testing.B) {
			b.SetBytes(int64(24 * n)) // 2 reads + 1 write
			for i := 0; i < b.N; i++ {
				dst := Add(xs, ys)
				sink = dst[0]
			}
		})
	}
}

func BenchmarkAddTo(b *testing.B) {
	for _, n := range benchSizes {
		xs, ys := benchData(n), benchData(n)
		dst := make([]float64, n)
		b.Run(fmt.Sprint(n), func(b *testing.B) {
			b.SetBytes(int64(24 * n))
			for i := 0; i < b.N; i++ {
				AddTo(dst, xs, ys)
			}
			sink = dst[0]
		})
	}
}

func BenchmarkMulTo(b *testing.B) {
	for _, n := range benchSizes {
		xs, ys := benchData(n), benchData(n)
		dst := make([]float64, n)
		b.Run(fmt.Sprint(n), func(b *testing.B) {
			b.SetBytes(int64(24 * n))
			for i := 0; i < b.N; i++ {
				MulTo(dst, xs, ys)
			}
			sink = dst[0]
		})
	}
}

func BenchmarkScaleTo(b *testing.B) {
	for _, n := range benchSizes {
		xs := benchData(n)
		dst := make([]float64, n)
		b.Run(fmt.Sprint(n), func(b *testing.B) {
			b.SetBytes(int64(16 * n))
			for i := 0; i < b.N; i++ {
				ScaleTo(dst, xs, 2.5)
			}
			sink = dst[0]
		})
	}
}

func BenchmarkAddScalarTo(b *testing.B) {
	for _, n := range benchSizes {
		xs := benchData(n)
		dst := make([]float64, n)
		b.Run(fmt.Sprint(n), func(b *testing.B) {
			b.SetBytes(int64(16 * n))
			for i := 0; i < b.N; i++ {
				AddScalarTo(dst, xs, 2.5)
			}
			sink = dst[0]
		})
	}
}

func BenchmarkAbsTo(b *testing.B) {
	for _, n := range benchSizes {
		xs := benchData(n)
		dst := make([]float64, n)
		b.Run(fmt.Sprint(n), func(b *testing.B) {
			b.SetBytes(int64(16 * n))
			for i := 0; i < b.N; i++ {
				AbsTo(dst, xs)
			}
			sink = dst[0]
		})
	}
}

// ---------------------------------------------------------------------------
// Elementwise math kernels (math.go).
//
// These are maps, so the naive baseline is the same loop without the four-wide
// unroll. The comparison answers one question: does the unroll buy anything on
// this machine, or does the hardware already keep enough loads in flight?
//
// benchPos is used for the transcendental ops because benchData spans negative
// values, and sqrt/log of a negative is NaN. Timing a NaN-only loop would be
// measuring a branch, not the math library.
// ---------------------------------------------------------------------------

// benchPos is like benchData but strictly positive, for sqrt/log/pow benchmarks.
func benchPos(n int) []float64 {
	xs := make([]float64, n)
	for i := range xs {
		xs[i] = float64(i%17)*0.5 + 0.25
	}
	return xs
}

func naiveNegTo(dst, xs []float64) []float64 {
	for i, v := range xs {
		dst[i] = -v
	}
	return dst
}

func naiveSqrtTo(dst, xs []float64) []float64 {
	for i, v := range xs {
		dst[i] = math.Sqrt(v)
	}
	return dst
}

func naiveDivTo(dst, xs, ys []float64) []float64 {
	for i := range dst {
		dst[i] = xs[i] / ys[i]
	}
	return dst
}

func naiveMin2To(dst, xs, ys []float64) []float64 {
	for i := range dst {
		if xs[i] < ys[i] {
			dst[i] = xs[i]
		} else {
			dst[i] = ys[i]
		}
	}
	return dst
}

func naiveMulAddTo(dst, xs, ys, zs []float64) []float64 {
	for i := range dst {
		dst[i] = xs[i]*ys[i] + zs[i]
	}
	return dst
}

func BenchmarkNegTo(b *testing.B) {
	for _, n := range benchSizes {
		xs := benchData(n)
		dst := make([]float64, n)
		b.Run(fmt.Sprint(n), func(b *testing.B) {
			b.SetBytes(int64(16 * n))
			for i := 0; i < b.N; i++ {
				NegTo(dst, xs)
			}
			sink = dst[0]
		})
	}
}

func BenchmarkNaiveNegTo(b *testing.B) {
	for _, n := range benchSizes {
		xs := benchData(n)
		dst := make([]float64, n)
		b.Run(fmt.Sprint(n), func(b *testing.B) {
			b.SetBytes(int64(16 * n))
			for i := 0; i < b.N; i++ {
				naiveNegTo(dst, xs)
			}
			sink = dst[0]
		})
	}
}

func BenchmarkSqrtTo(b *testing.B) {
	for _, n := range benchSizes {
		xs := benchPos(n)
		dst := make([]float64, n)
		b.Run(fmt.Sprint(n), func(b *testing.B) {
			b.SetBytes(int64(16 * n))
			for i := 0; i < b.N; i++ {
				SqrtTo(dst, xs)
			}
			sink = dst[0]
		})
	}
}

func BenchmarkNaiveSqrtTo(b *testing.B) {
	for _, n := range benchSizes {
		xs := benchPos(n)
		dst := make([]float64, n)
		b.Run(fmt.Sprint(n), func(b *testing.B) {
			b.SetBytes(int64(16 * n))
			for i := 0; i < b.N; i++ {
				naiveSqrtTo(dst, xs)
			}
			sink = dst[0]
		})
	}
}

func BenchmarkDivTo(b *testing.B) {
	for _, n := range benchSizes {
		xs, ys := benchPos(n), benchPos(n)
		dst := make([]float64, n)
		b.Run(fmt.Sprint(n), func(b *testing.B) {
			b.SetBytes(int64(24 * n))
			for i := 0; i < b.N; i++ {
				DivTo(dst, xs, ys)
			}
			sink = dst[0]
		})
	}
}

func BenchmarkNaiveDivTo(b *testing.B) {
	for _, n := range benchSizes {
		xs, ys := benchPos(n), benchPos(n)
		dst := make([]float64, n)
		b.Run(fmt.Sprint(n), func(b *testing.B) {
			b.SetBytes(int64(24 * n))
			for i := 0; i < b.N; i++ {
				naiveDivTo(dst, xs, ys)
			}
			sink = dst[0]
		})
	}
}

func BenchmarkMin2To(b *testing.B) {
	for _, n := range benchSizes {
		xs, ys := benchData(n), benchPos(n)
		dst := make([]float64, n)
		b.Run(fmt.Sprint(n), func(b *testing.B) {
			b.SetBytes(int64(24 * n))
			for i := 0; i < b.N; i++ {
				Min2To(dst, xs, ys)
			}
			sink = dst[0]
		})
	}
}

func BenchmarkNaiveMin2To(b *testing.B) {
	for _, n := range benchSizes {
		xs, ys := benchData(n), benchPos(n)
		dst := make([]float64, n)
		b.Run(fmt.Sprint(n), func(b *testing.B) {
			b.SetBytes(int64(24 * n))
			for i := 0; i < b.N; i++ {
				naiveMin2To(dst, xs, ys)
			}
			sink = dst[0]
		})
	}
}

func BenchmarkMulAddTo(b *testing.B) {
	for _, n := range benchSizes {
		xs, ys, zs := benchPos(n), benchPos(n), benchPos(n)
		dst := make([]float64, n)
		b.Run(fmt.Sprint(n), func(b *testing.B) {
			b.SetBytes(int64(32 * n))
			for i := 0; i < b.N; i++ {
				MulAddTo(dst, xs, ys, zs)
			}
			sink = dst[0]
		})
	}
}

func BenchmarkNaiveMulAddTo(b *testing.B) {
	for _, n := range benchSizes {
		xs, ys, zs := benchPos(n), benchPos(n), benchPos(n)
		dst := make([]float64, n)
		b.Run(fmt.Sprint(n), func(b *testing.B) {
			b.SetBytes(int64(32 * n))
			for i := 0; i < b.N; i++ {
				naiveMulAddTo(dst, xs, ys, zs)
			}
			sink = dst[0]
		})
	}
}

// BenchmarkFMAEvidence measures the claim recorded in next-steps.md that
// hardware FMA is unreachable from pure Go.
//
// In this toolchain math.FMA is implemented entirely in Go (see math/fma.go,
// which has no assembly and no architecture fast path): it computes the fused
// result by integer manipulation of the operands. The comparison below is the
// evidence. If a future toolchain grows a hardware fast path, this benchmark is
// how that would be detected rather than assumed.
func BenchmarkFMAEvidence(b *testing.B) {
	n := 1 << 12
	xs, ys, zs := benchPos(n), benchPos(n), benchPos(n)
	dst := make([]float64, n)

	b.Run("math.FMA", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			for j := range dst {
				dst[j] = math.FMA(xs[j], ys[j], zs[j])
			}
		}
		sink = dst[0]
	})

	b.Run("separate", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			for j := range dst {
				dst[j] = xs[j]*ys[j] + zs[j]
			}
		}
		sink = dst[0]
	})
}

// ---------------------------------------------------------------------------
// Predicates and masked reductions (mask.go).
//
// A predicate reads 16 bytes and writes one, so the interesting question is
// whether the four-wide unroll matters at all when the store side is this light.
// These baselines answer it rather than assuming it.
// ---------------------------------------------------------------------------

func naiveGreaterTo(dst []uint8, xs, ys []float64) []uint8 {
	for i := range dst {
		if xs[i] > ys[i] {
			dst[i] = 1
		} else {
			dst[i] = 0
		}
	}
	return dst
}

func BenchmarkGreaterTo(b *testing.B) {
	for _, n := range benchSizes {
		xs, ys := benchData(n), benchPos(n)
		dst := make([]uint8, n)
		b.Run(fmt.Sprint(n), func(b *testing.B) {
			b.SetBytes(int64(16 * n))
			for i := 0; i < b.N; i++ {
				GreaterTo(dst, xs, ys)
			}
			sink = float64(dst[0])
		})
	}
}

func BenchmarkNaiveGreaterTo(b *testing.B) {
	for _, n := range benchSizes {
		xs, ys := benchData(n), benchPos(n)
		dst := make([]uint8, n)
		b.Run(fmt.Sprint(n), func(b *testing.B) {
			b.SetBytes(int64(16 * n))
			for i := 0; i < b.N; i++ {
				naiveGreaterTo(dst, xs, ys)
			}
			sink = float64(dst[0])
		})
	}
}

func BenchmarkMaskedSum(b *testing.B) {
	for _, n := range benchSizes {
		xs := benchData(n)
		mask := GreaterScalar(xs, 0)
		b.Run(fmt.Sprint(n), func(b *testing.B) {
			b.SetBytes(int64(9 * n))
			for i := 0; i < b.N; i++ {
				sink = MaskedSum(xs, mask)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Cumulative operations (cum.go).
//
// The whole claim of cum.go is that blocking the prefix scan shortens the carry
// chain from one add per element to one add per block. The baseline below is the
// plain sequential prefix sum, so the comparison measures exactly that claim and
// nothing else.
// ---------------------------------------------------------------------------

func naiveCumSum(dst, xs []float64) []float64 {
	var s float64
	for i, v := range xs {
		s += v
		dst[i] = s
	}
	return dst
}

func naiveCumMax(dst, xs []float64) []float64 {
	m := math.Inf(-1)
	for i, v := range xs {
		if v != v {
			m = math.NaN()
		} else if v > m {
			m = v
		}
		dst[i] = m
	}
	return dst
}

func BenchmarkCumSumTo(b *testing.B) {
	for _, n := range benchSizes {
		xs := benchData(n)
		dst := make([]float64, n)
		b.Run(fmt.Sprint(n), func(b *testing.B) {
			b.SetBytes(int64(16 * n))
			for i := 0; i < b.N; i++ {
				CumSumTo(dst, xs)
			}
			sink = dst[0]
		})
	}
}

func BenchmarkNaiveCumSumTo(b *testing.B) {
	for _, n := range benchSizes {
		xs := benchData(n)
		dst := make([]float64, n)
		b.Run(fmt.Sprint(n), func(b *testing.B) {
			b.SetBytes(int64(16 * n))
			for i := 0; i < b.N; i++ {
				naiveCumSum(dst, xs)
			}
			sink = dst[0]
		})
	}
}

func BenchmarkCumMaxTo(b *testing.B) {
	for _, n := range benchSizes {
		xs := benchData(n)
		dst := make([]float64, n)
		b.Run(fmt.Sprint(n), func(b *testing.B) {
			b.SetBytes(int64(16 * n))
			for i := 0; i < b.N; i++ {
				CumMaxTo(dst, xs)
			}
			sink = dst[0]
		})
	}
}

func BenchmarkNaiveCumMaxTo(b *testing.B) {
	for _, n := range benchSizes {
		xs := benchData(n)
		dst := make([]float64, n)
		b.Run(fmt.Sprint(n), func(b *testing.B) {
			b.SetBytes(int64(16 * n))
			for i := 0; i < b.N; i++ {
				naiveCumMax(dst, xs)
			}
			sink = dst[0]
		})
	}
}

// ---------------------------------------------------------------------------
// Lagged maps (signal.go).
//
// Shift and Diff read two streams a fixed distance apart and write one. There is
// no carry, so this is the same question as math.go's maps: does the four-wide
// unroll still pay when one of the loads is offset?
// ---------------------------------------------------------------------------

func naiveShiftTo(dst, xs []float64, lag int) []float64 {
	for i := lag; i < len(xs); i++ {
		dst[i] = xs[i-lag]
	}
	return dst
}

func naiveDiffTo(dst, xs []float64, lag int) []float64 {
	for i := lag; i < len(xs); i++ {
		dst[i] = xs[i] - xs[i-lag]
	}
	return dst
}

func BenchmarkShiftTo(b *testing.B) {
	for _, lag := range []int{1, 50} {
		for _, n := range benchSizes {
			xs := benchData(n)
			dst := make([]float64, n)
			b.Run(fmt.Sprintf("%d/lag%d", n, lag), func(b *testing.B) {
				b.SetBytes(int64(16 * n))
				for i := 0; i < b.N; i++ {
					ShiftTo(dst, xs, lag)
				}
				sink = dst[0]
			})
		}
	}
}

func BenchmarkNaiveShiftTo(b *testing.B) {
	for _, lag := range []int{1, 50} {
		for _, n := range benchSizes {
			xs := benchData(n)
			dst := make([]float64, n)
			b.Run(fmt.Sprintf("%d/lag%d", n, lag), func(b *testing.B) {
				b.SetBytes(int64(16 * n))
				for i := 0; i < b.N; i++ {
					naiveShiftTo(dst, xs, lag)
				}
				sink = dst[0]
			})
		}
	}
}

func BenchmarkDiffTo(b *testing.B) {
	for _, lag := range []int{1, 50} {
		for _, n := range benchSizes {
			xs := benchData(n)
			dst := make([]float64, n)
			b.Run(fmt.Sprintf("%d/lag%d", n, lag), func(b *testing.B) {
				b.SetBytes(int64(16 * n))
				for i := 0; i < b.N; i++ {
					DiffTo(dst, xs, lag)
				}
				sink = dst[0]
			})
		}
	}
}

func BenchmarkNaiveDiffTo(b *testing.B) {
	for _, lag := range []int{1, 50} {
		for _, n := range benchSizes {
			xs := benchData(n)
			dst := make([]float64, n)
			b.Run(fmt.Sprintf("%d/lag%d", n, lag), func(b *testing.B) {
				b.SetBytes(int64(16 * n))
				for i := 0; i < b.N; i++ {
					naiveDiffTo(dst, xs, lag)
				}
				sink = dst[0]
			})
		}
	}
}

// ---------------------------------------------------------------------------
// Order statistics and arg reductions (order.go, arg.go).
//
// The claim in order.go is that a quantile does not need the input sorted. The
// baseline is therefore the honest alternative -- copy and sort.Float64s, then
// index -- and not a strawman. If quickselect does not beat it, quickselect is not
// worth the code and this benchmark is how that would be discovered.
// ---------------------------------------------------------------------------

func naiveQuantile(xs []float64, q float64) float64 {
	buf := make([]float64, len(xs))
	copy(buf, xs)
	sort.Float64s(buf)
	n := len(buf)
	h := float64(n-1) * q
	lo := int(math.Floor(h))
	hi := int(math.Ceil(h))
	return buf[lo] + (h-float64(lo))*(buf[hi]-buf[lo])
}

// quantileSizes stops at 1<<20: the baseline sorts, so 4M elements would dominate
// the suite's runtime without changing the conclusion.
var quantileSizes = []int{1 << 10, 1 << 15, 1 << 20}

func BenchmarkQuantile(b *testing.B) {
	for _, n := range quantileSizes {
		xs := benchData(n)
		buf := make([]float64, n)
		b.Run(fmt.Sprint(n), func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				sink = QuantileInto(xs, 0.5, buf)
			}
		})
	}
}

func BenchmarkNaiveQuantile(b *testing.B) {
	for _, n := range quantileSizes {
		xs := benchData(n)
		b.Run(fmt.Sprint(n), func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				sink = naiveQuantile(xs, 0.5)
			}
		})
	}
}

func naiveArgMin(xs []float64) int {
	idx := 0
	for i := 1; i < len(xs); i++ {
		if xs[i] < xs[idx] {
			idx = i
		}
	}
	return idx
}

func BenchmarkArgMin(b *testing.B) {
	for _, n := range benchSizes {
		xs := benchData(n)
		b.Run(fmt.Sprint(n), func(b *testing.B) {
			b.SetBytes(int64(8 * n))
			for i := 0; i < b.N; i++ {
				sink = float64(ArgMin(xs))
			}
		})
	}
}

func BenchmarkNaiveArgMin(b *testing.B) {
	for _, n := range benchSizes {
		xs := benchData(n)
		b.Run(fmt.Sprint(n), func(b *testing.B) {
			b.SetBytes(int64(8 * n))
			for i := 0; i < b.N; i++ {
				sink = float64(naiveArgMin(xs))
			}
		})
	}
}

func BenchmarkRank(b *testing.B) {
	for _, n := range quantileSizes {
		xs := benchData(n)
		dst := make([]float64, n)
		b.Run(fmt.Sprint(n), func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				RankTo(dst, xs)
			}
			sink = dst[0]
		})
	}
}

// ---------------------------------------------------------------------------
// Statistics (stats.go).
//
// Two baselines are needed, because two different claims are being made:
//
//  1. The tuned two-pass against a naive two-pass, which measures the arch tuning
//     of the deviation sums. This is the usual "did the tuning pay off" question.
//  2. The two-pass against a one-pass sum-of-squares form, which measures what the
//     accuracy costs. That comparison is expected to favour one-pass on speed; the
//     point of recording it is to know the size of the trade rather than to pretend
//     it does not exist.
// ---------------------------------------------------------------------------

func naiveTwoPassVariance(xs []float64) float64 {
	var s float64
	for _, v := range xs {
		s += v
	}
	mean := s / float64(len(xs))
	var ss float64
	for _, v := range xs {
		d := v - mean
		ss += d * d
	}
	return ss / float64(len(xs))
}

func BenchmarkVariance(b *testing.B) {
	for _, n := range benchSizes {
		xs := benchData(n)
		b.Run(fmt.Sprint(n), func(b *testing.B) {
			// Two traversals of the input.
			b.SetBytes(int64(16 * n))
			for i := 0; i < b.N; i++ {
				sink = Variance(xs)
			}
		})
	}
}

func BenchmarkNaiveTwoPassVariance(b *testing.B) {
	for _, n := range benchSizes {
		xs := benchData(n)
		b.Run(fmt.Sprint(n), func(b *testing.B) {
			b.SetBytes(int64(16 * n))
			for i := 0; i < b.N; i++ {
				sink = naiveTwoPassVariance(xs)
			}
		})
	}
}

func BenchmarkNaiveOnePassVariance(b *testing.B) {
	for _, n := range benchSizes {
		xs := benchData(n)
		b.Run(fmt.Sprint(n), func(b *testing.B) {
			// One traversal, but numerically unsafe.
			b.SetBytes(int64(8 * n))
			for i := 0; i < b.N; i++ {
				sink = naiveOnePassVariance(xs)
			}
		})
	}
}

func BenchmarkCorrelation(b *testing.B) {
	for _, n := range benchSizes {
		xs, ys := benchData(n), benchPos(n)
		b.Run(fmt.Sprint(n), func(b *testing.B) {
			// Two traversals of two inputs, plus the deviations.
			b.SetBytes(int64(32 * n))
			for i := 0; i < b.N; i++ {
				sink = Correlation(xs, ys)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Rolling-window batch kernels (rolling.go).
//
// The file deliberately contains both shapes, so the benchmarks are arranged to show the
// difference rather than a single speedup number: RollingSum and RollingMax carry running state
// and should be flat in the window, while RollingStdDev (and the rest of the recomputing
// kernels) should scale with it. The naive baseline rescans the window for every output, which
// is what the running-state kernels exist to avoid.
// ---------------------------------------------------------------------------

func naiveRollingSum(xs []float64, n int) []float64 {
	out := make([]float64, len(xs))
	for i := n - 1; i < len(xs); i++ {
		var s float64
		for j := i - n + 1; j <= i; j++ {
			s += xs[j]
		}
		out[i] = s
	}
	return out
}

func naiveRollingMax(xs []float64, n int) []float64 {
	out := make([]float64, len(xs))
	for i := n - 1; i < len(xs); i++ {
		m := xs[i-n+1]
		for j := i - n + 2; j <= i; j++ {
			if xs[j] > m {
				m = xs[j]
			}
		}
		out[i] = m
	}
	return out
}

func BenchmarkRollingSum(b *testing.B) {
	for _, n := range benchSizes {
		xs := benchData(n)
		for _, w := range []int{20, 200} {
			b.Run(fmt.Sprintf("%d/window%d", n, w), func(b *testing.B) {
				b.SetBytes(int64(8 * n))
				for i := 0; i < b.N; i++ {
					sink = RollingSum(xs, w)[n-1]
				}
			})
		}
	}
}

func BenchmarkNaiveRollingSum(b *testing.B) {
	for _, n := range benchSizes {
		xs := benchData(n)
		for _, w := range []int{20, 200} {
			b.Run(fmt.Sprintf("%d/window%d", n, w), func(b *testing.B) {
				b.SetBytes(int64(8 * n))
				for i := 0; i < b.N; i++ {
					sink = naiveRollingSum(xs, w)[n-1]
				}
			})
		}
	}
}

func BenchmarkRollingMax(b *testing.B) {
	for _, n := range benchSizes {
		xs := benchData(n)
		for _, w := range []int{20, 200} {
			b.Run(fmt.Sprintf("%d/window%d", n, w), func(b *testing.B) {
				b.SetBytes(int64(8 * n))
				for i := 0; i < b.N; i++ {
					sink = RollingMax(xs, w)[n-1]
				}
			})
		}
	}
}

func BenchmarkNaiveRollingMax(b *testing.B) {
	for _, n := range benchSizes {
		xs := benchData(n)
		for _, w := range []int{20, 200} {
			b.Run(fmt.Sprintf("%d/window%d", n, w), func(b *testing.B) {
				b.SetBytes(int64(8 * n))
				for i := 0; i < b.N; i++ {
					sink = naiveRollingMax(xs, w)[n-1]
				}
			})
		}
	}
}

func BenchmarkRollingStdDev(b *testing.B) {
	for _, n := range benchSizes {
		xs := benchData(n)
		for _, w := range []int{20, 200} {
			b.Run(fmt.Sprintf("%d/window%d", n, w), func(b *testing.B) {
				b.SetBytes(int64(8 * n))
				for i := 0; i < b.N; i++ {
					sink = RollingStdDev(xs, w)[n-1]
				}
			})
		}
	}
}

func BenchmarkRollingMedian(b *testing.B) {
	for _, n := range benchSizes {
		xs := benchData(n)
		for _, w := range []int{20, 200} {
			b.Run(fmt.Sprintf("%d/window%d", n, w), func(b *testing.B) {
				b.SetBytes(int64(8 * n))
				for i := 0; i < b.N; i++ {
					sink = RollingMedian(xs, w)[n-1]
				}
			})
		}
	}
}

// ---------------------------------------------------------------------------
// Exponential smoothers (smoothing.go).
//
// The micro-benchmark below is deliberately *not* used to justify anything on its own -- see the
// cautionary section in docs/benchmarks.md. It is here so the streaming and batch forms can be
// compared in one run, where the comparison is meaningful, and it correctly predicted the indicator
// level result that the rolling-sum version did not: the batch smoother is about 2.7x the streaming
// one, and every EMA/RMA-driven indicator improved by 1.3-2.4x.
// ---------------------------------------------------------------------------

func BenchmarkRollingEMA(b *testing.B) {
	for _, n := range benchSizes {
		xs := benchData(n)
		for _, w := range []int{20, 200} {
			b.Run(fmt.Sprintf("%d/window%d", n, w), func(b *testing.B) {
				b.SetBytes(int64(8 * n))
				for i := 0; i < b.N; i++ {
					sink = RollingEMA(xs, w)[n-1]
				}
			})
		}
	}
}

func BenchmarkRollingRMA(b *testing.B) {
	for _, n := range benchSizes {
		xs := benchData(n)
		for _, w := range []int{14, 100} {
			b.Run(fmt.Sprintf("%d/window%d", n, w), func(b *testing.B) {
				b.SetBytes(int64(8 * n))
				for i := 0; i < b.N; i++ {
					sink = RollingRMA(xs, w)[n-1]
				}
			})
		}
	}
}

// BenchmarkNaiveRollingEMA is the baseline: the recurrence with the SMA seed, written out.
func BenchmarkNaiveRollingEMA(b *testing.B) {
	for _, n := range benchSizes {
		xs := benchData(n)
		b.Run(fmt.Sprint(n), func(b *testing.B) {
			b.SetBytes(int64(8 * n))
			for i := 0; i < b.N; i++ {
				sink = naiveRollingEMA(xs, 20)[n-1]
			}
		})
	}
}

func naiveRollingEMA(xs []float64, w int) []float64 {
	out := make([]float64, len(xs))
	for i := range out {
		out[i] = math.NaN()
	}
	if len(xs) < w {
		return out
	}
	var s float64
	for i := 0; i < w; i++ {
		s += xs[i]
	}
	prev := s / float64(w)
	out[w-1] = prev
	alpha := 2.0 / float64(w+1)
	for i := w; i < len(xs); i++ {
		prev = alpha*xs[i] + (1-alpha)*prev
		out[i] = prev
	}
	return out
}
