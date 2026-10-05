package series

import (
	"fmt"
	"math"
	"testing"
)

// ---------------------------------------------------------------------------
// Benchmarks for the streaming rollers.
//
// The claim of this package is that a roller is O(1) per element while recomputing
// over the window is O(window). The naive baseline below does exactly the
// recomputation, so the comparison measures that claim and nothing else.
//
// Rolling a window is not a throughput competition with vec. A batch pass over a
// slice can keep independent accumulators and vectorise; a streaming roller must
// carry state between elements by definition. The relevant question is how the cost
// scales with the window, and whether it scales *at all*.
// ---------------------------------------------------------------------------

// sink prevents the compiler from eliminating the benchmarked work.
var sink float64

const benchN = 1 << 16

func benchData(n int) []float64 {
	xs := make([]float64, n)
	for i := range xs {
		xs[i] = float64(i%17)*0.5 - 4.0
	}
	return xs
}

// naiveSMA recomputes the whole window at every position, which is the O(window)
// alternative a streaming roller exists to replace.
func naiveSMA(xs []float64, w int) float64 {
	var last float64
	for i := range xs {
		if i+1 < w {
			continue
		}
		var s float64
		for j := i - w + 1; j <= i; j++ {
			s += xs[j]
		}
		last = s / float64(w)
	}
	return last
}

func BenchmarkSMA(b *testing.B) {
	xs := benchData(benchN)
	for _, w := range []int{20, 200} {
		b.Run(fmt.Sprintf("window%d", w), func(b *testing.B) {
			s := NewSMA(w)
			b.SetBytes(int64(8 * len(xs)))
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				s.Reset()
				for _, v := range xs {
					sink = s.Push(v)
				}
			}
		})
	}
}

func BenchmarkNaiveSMA(b *testing.B) {
	xs := benchData(benchN)
	for _, w := range []int{20, 200} {
		b.Run(fmt.Sprintf("window%d", w), func(b *testing.B) {
			b.SetBytes(int64(8 * len(xs)))
			for i := 0; i < b.N; i++ {
				sink = naiveSMA(xs, w)
			}
		})
	}
}

// BenchmarkSum measures the compensated rolling sum. Comparing it with
// BenchmarkNaiveSum shows what the Neumaier compensation costs.
func BenchmarkSum(b *testing.B) {
	xs := benchData(benchN)
	for _, w := range []int{20, 200} {
		b.Run(fmt.Sprintf("window%d", w), func(b *testing.B) {
			s := NewSum(w)
			b.SetBytes(int64(8 * len(xs)))
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				s.Reset()
				for _, v := range xs {
					sink = s.Push(v)
				}
			}
		})
	}
}

// naiveSumRoller is the same O(1) rolling sum without compensation, so the
// difference isolates the cost of the extra Neumaier arithmetic.
func naiveSumRoller(xs []float64, w int) float64 {
	ring := make([]float64, 0, w)
	var total, last float64
	for _, v := range xs {
		if len(ring) == w {
			total -= ring[0]
			ring = ring[1:]
		}
		ring = append(ring, v)
		total += v
		last = total
	}
	return last
}

func BenchmarkNaiveSum(b *testing.B) {
	xs := benchData(benchN)
	for _, w := range []int{20, 200} {
		b.Run(fmt.Sprintf("window%d", w), func(b *testing.B) {
			b.SetBytes(int64(8 * len(xs)))
			for i := 0; i < b.N; i++ {
				sink = naiveSumRoller(xs, w)
			}
		})
	}
}

func BenchmarkEMA(b *testing.B) {
	xs := benchData(benchN)
	for _, w := range []int{12, 26} {
		b.Run(fmt.Sprintf("window%d", w), func(b *testing.B) {
			e := NewEMA(w)
			b.SetBytes(int64(8 * len(xs)))
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				e.Reset()
				for _, v := range xs {
					sink = e.Push(v)
				}
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Isolating the cost of Neumaier compensation.
//
// BenchmarkSum above is compared against a naive roller that uses a slice rather
// than a Ring and has no NaN counter, so it does not isolate the compensation: it
// shows the cost of the whole streaming structure. sumNoComp mirrors Sum exactly --
// same Ring, same fullness check, same NaN counter -- and differs only in the
// accumulator, so BenchmarkSum minus BenchmarkSumNoCompensation is the compensation
// and nothing else.
// ---------------------------------------------------------------------------

type sumNoComp struct {
	ring     Ring
	window   int
	total    float64
	nanCount int
}

func newSumNoComp(window int) *sumNoComp {
	return &sumNoComp{ring: *NewRing(window), window: window}
}

func (s *sumNoComp) Warmup() int { return s.window }

func (s *sumNoComp) Reset() {
	s.ring.Reset()
	s.total = 0
	s.nanCount = 0
}

func (s *sumNoComp) Push(v float64) float64 {
	old, evicted := s.ring.Push(v)
	if v != v {
		s.nanCount++
	} else {
		s.total += v
	}
	if evicted {
		if old != old {
			s.nanCount--
		} else {
			s.total -= old
		}
	}
	if !s.ring.Full() {
		return math.NaN()
	}
	if s.nanCount > 0 {
		return math.NaN()
	}
	return s.total
}

func BenchmarkSumNoCompensation(b *testing.B) {
	xs := benchData(benchN)
	for _, w := range []int{20, 200} {
		b.Run(fmt.Sprintf("window%d", w), func(b *testing.B) {
			s := newSumNoComp(w)
			b.SetBytes(int64(8 * len(xs)))
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				s.Reset()
				for _, v := range xs {
					sink = s.Push(v)
				}
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Why a per-Push cost does not compose: the roller is latency-bound.
//
// A rolling sum carries a dependency chain. `total`, `comp` and the ring must be updated in order,
// and every element's result depends on the previous element's state. With one roller there is
// exactly one such chain, and the loop runs at the chain's latency rather than at the machine's
// throughput -- the CPU has nothing else to overlap it with.
//
// The two benchmarks below separate the two effects. `parallel` runs n *independent* rollers over
// each element, so there are n chains and the hardware can interleave them. `chained` feeds each
// roller's output into the next, so there is still only one chain but it is n times longer.
//
// If the cost per Push is a latency, parallel should be roughly flat in n and chained roughly linear.
// That is what makes a single-roller micro-benchmark useless for predicting what happens when an
// indicator drives six of them: the six chains overlap, so the indicator pays for one.
//
// This was found by contradicting a measurement. `UltimateOscillator` was expected to gain about 5x
// from batch rolling sums on the strength of a 20-versus-4 ns/element single-roller benchmark, and
// gained nothing, because six streaming rollers had never cost 6x20.
// ---------------------------------------------------------------------------

func BenchmarkSumChainParallel(b *testing.B) {
	xs := benchData(1 << 20)
	const window = 20

	b.Run("depth1", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			a := NewSum(window)
			var acc float64
			for _, v := range xs {
				acc += a.Push(v)
			}
			sink = acc
		}
	})
	b.Run("depth2", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			a, c := NewSum(window), NewSum(window)
			var acc float64
			for _, v := range xs {
				acc += a.Push(v) + c.Push(v)
			}
			sink = acc
		}
	})
	b.Run("depth4", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			a, c, d, e := NewSum(window), NewSum(window), NewSum(window), NewSum(window)
			var acc float64
			for _, v := range xs {
				acc += a.Push(v) + c.Push(v) + d.Push(v) + e.Push(v)
			}
			sink = acc
		}
	})
	b.Run("depth6", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			a, c, d, e, f, g := NewSum(window), NewSum(window), NewSum(window),
				NewSum(window), NewSum(window), NewSum(window)
			var acc float64
			for _, v := range xs {
				acc += a.Push(v) + c.Push(v) + d.Push(v) + e.Push(v) + f.Push(v) + g.Push(v)
			}
			sink = acc
		}
	})
}

func BenchmarkSumChainSerial(b *testing.B) {
	xs := benchData(1 << 20)
	const window = 20

	b.Run("depth1", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			a := NewSum(window)
			var x float64
			for _, v := range xs {
				x = a.Push(v)
			}
			sink = x
		}
	})
	b.Run("depth2", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			a, c := NewSum(window), NewSum(window)
			var x float64
			for _, v := range xs {
				x = a.Push(v)
				x = c.Push(x)
			}
			sink = x
		}
	})
	b.Run("depth4", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			a, c, d, e := NewSum(window), NewSum(window), NewSum(window), NewSum(window)
			var x float64
			for _, v := range xs {
				x = a.Push(v)
				x = c.Push(x)
				x = d.Push(x)
				x = e.Push(x)
			}
			sink = x
		}
	})
}
