package vec

import (
	"fmt"
	"testing"
)

// ---------------------------------------------------------------------------
// Bandwidth ceiling probes.
//
// benchmarks.md previously claimed the tuned reductions "plateau at the machine's
// memory bandwidth", and used that to argue that SIMD would win nothing. The claim
// was inferred from the shape of a curve rather than measured.
//
// These benchmarks measure the ceiling directly, and the answer is that the claim
// was wrong. See BenchmarkReadOnlyTuned for the numbers and the consequences.
//
//   - BenchmarkReadOnly walks every element doing no real arithmetic, with the
//     naive one-accumulator shape.
//
//   - BenchmarkReadOnlyTuned does the same but with the *tuned* four-chain shape,
//     which isolates loop shape from arithmetic: it is the same bytes and the same
//     loads as Sum, differing only in what is computed.
//
//   - BenchmarkMemcpy is the machine's practical read+write limit, as a sanity
//     bound on what any single-pass loop could hope to reach.
// ---------------------------------------------------------------------------

// readOnlyLoop sums with a trick that prevents the compiler from vectorizing the
// arithmetic away while still touching every cache line exactly once. It adds into
// a single accumulator with no unrolling, so it is *not* tuned; it exists purely
// to establish how fast the memory can be walked.
//
//go:noinline
func readOnlyLoop(xs []float64) float64 {
	var t float64
	for _, v := range xs {
		t += v
	}
	return t
}

// touchEveryElement reads every element but only keeps one, so the arithmetic is
// trivial and the loop is limited by loads alone.
//
//go:noinline
func touchEveryElement(xs []float64) float64 {
	var acc float64
	for i := range xs {
		acc += xs[i] * 0 // forces a load, contributes nothing
	}
	return acc + xs[len(xs)-1]
}

func BenchmarkReadOnly(b *testing.B) {
	for _, n := range benchSizes {
		xs := benchData(n)
		b.Run(fmt.Sprint(n), func(b *testing.B) {
			b.SetBytes(int64(8 * n))
			for i := 0; i < b.N; i++ {
				sink = touchEveryElement(xs)
			}
		})
	}
}

func BenchmarkMemcpy(b *testing.B) {
	for _, n := range benchSizes {
		src := benchData(n)
		dst := make([]float64, n)
		b.Run(fmt.Sprint(n), func(b *testing.B) {
			// 8 bytes read + 8 bytes written per element.
			b.SetBytes(int64(16 * n))
			for i := 0; i < b.N; i++ {
				copy(dst, src)
			}
			sink = dst[0]
		})
	}
}

// BenchmarkReadOnlyTuned is the same memory walk but written the way the tuned
// loops are: four independent accumulator chains, eight elements per iteration.
//
// Comparing this against BenchmarkReadOnly shows how much of Sum's throughput is
// achieved by the *loop shape* as opposed to the arithmetic. Comparing it against
// BenchmarkSum shows whether the floating-point adds cost anything at all on top
// of the loads.
//
// # Result: this overturns the "bandwidth-bound" claim
//
// Measured on arm64 at n=4M:
//
//	ReadOnlyTuned (4 chains, v*0)   37.8 GB/s
//	Sum (4 chains, v+v)             25.4 GB/s
//	ReadOnly (naive, 1 chain)        4.8 GB/s
//	Memcpy (read + write)           57.6 GB/s
//
// ReadOnlyTuned and Sum move the same bytes with the same loop shape. The only
// difference is that ReadOnlyTuned multiplies by zero instead of accumulating a
// real add. It is 49% faster, which means Sum is *not* limited by memory
// bandwidth -- it is limited by floating-point add latency remaining in the
// dependency chain.
//
// Consequences:
//
//   - The claim in docs/benchmarks.md that the reductions "plateau at memory
//     bandwidth" was wrong. Corrected.
//   - docs/next-steps.md §6.5 (SIMD) and §6.6 (is there headroom) are answered:
//     there is headroom, roughly 37.8/25.4 = 1.49x on arm64. SIMD is worth
//     investigating, because the remaining bottleneck is arithmetic, not memory.
//   - FMA is also worth measuring, since it collapses a multiply and an add into
//     one operation and Dot is closer to the ceiling (38.4 GB/s) than Sum is.
func BenchmarkReadOnlyTuned(b *testing.B) {
	for _, n := range benchSizes {
		xs := benchData(n)
		b.Run(fmt.Sprint(n), func(b *testing.B) {
			b.SetBytes(int64(8 * n))
			for i := 0; i < b.N; i++ {
				sink = readOnlyTuned(xs)
			}
		})
	}
}

func readOnlyTuned(xs []float64) float64 {
	n := len(xs)
	var s0, s1, s2, s3 float64
	i := 0
	limit := n - 7

	for i < limit {
		s0 += xs[i] * 0
		s1 += xs[i+2] * 0
		s2 += xs[i+4] * 0
		s3 += xs[i+6] * 0
		i += 8
	}
	for ; i < n; i++ {
		s0 += xs[i] * 0
	}
	return (s0 + s1) + (s2 + s3)
}
