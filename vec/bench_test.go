package vec

import (
	"fmt"
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

// BenchmarkMinArchNoNaN measures the scan loop without the NaN check, so the cost
// of the documented NaN policy is a measured quantity rather than a guess.
func BenchmarkMinArchNoNaN(b *testing.B) {
	for _, n := range benchSizes {
		xs := benchData(n)
		b.Run(fmt.Sprint(n), func(b *testing.B) {
			b.SetBytes(int64(8 * n))
			for i := 0; i < b.N; i++ {
				sink = minArch(xs)
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
