package series

import "math"

// ---------------------------------------------------------------------------
// Rolling sum and simple moving average.
//
// A rolling sum is updated by adding the arriving value and subtracting the one
// leaving the window, so it is O(1) per element regardless of the window length.
// That is also its numerical hazard: an unbroken chain of add-and-subtract
// operations accumulates rounding error without bound, and the error never cancels
// because the subtractions are not the exact inverses of the additions.
//
// # Neumaier compensation
//
// Sum therefore carries a compensation term alongside the running total, using
// Neumaier's variant of Kahan summation: each add computes the rounding error it
// introduced and accumulates that separately, so the total keeps a few extra digits
// of the true sum.
//
// The failure it prevents is concrete. With window 2 over the series
//
//	1e16, 1, 1, 1, ...
//
// an uncompensated rolling sum returns 1 for the window {1, 1}, because it computed
// 1e16 + 1 = 1e16 (the 1 was rounded away), then 1e16 - 1e16 = 0, then 0 + 1 = 1.
// The compensated sum returns 2, having kept the discarded 1 in its compensation
// term. TestSumDoesNotLoseSmallValuesAcrossEviction pins exactly this.
//
// The compensation is not free, and an earlier draft of this comment claimed it was.
// Isolated against an otherwise identical roller -- same Ring, same fullness check,
// same NaN counter, plain accumulator -- it roughly **doubles** the cost of a push:
// 19.2 ns/element against 9.3 ns/element at a window of 20, 2.07x. The earlier
// figure of 4.3x came from comparing against a roller with a different buffer
// structure, which measured the whole streaming design rather than the compensation.
//
// That is a real price for correctness, and it is recorded as one rather than
// explained away. Periodic re-summation from the ring is the obvious cheaper
// alternative -- drift bounded to one window instead of compensated away, at one
// amortized add per element -- and is left as a measured follow-up in
// docs/benchmarks.md rather than guessed at here.
// ---------------------------------------------------------------------------

// Sum is a rolling window sum over the last n values.
//
// The zero value is not usable; construct with [NewSum].
type Sum struct {
	ring     Ring
	window   int
	total    float64
	comp     float64
	nanCount int
}

// NewSum returns a rolling sum over a window of window values.
//
// It panics if window < 1.
func NewSum(window int) *Sum {
	if window < 1 {
		panic("series: Sum window must be >= 1")
	}
	return &Sum{ring: *NewRing(window), window: window}
}

// Warmup returns the window length.
func (s *Sum) Warmup() int { return s.window }

// Window returns the window length.
func (s *Sum) Window() int { return s.window }

// Reset clears the accumulated state without reallocating.
func (s *Sum) Reset() {
	s.ring.Reset()
	s.total = 0
	s.comp = 0
	s.nanCount = 0
}

// Push adds v to the window and returns the current sum, or NaN until the window is
// full.
//
// # NaN values are counted, never accumulated
//
// A NaN is deliberately kept out of the compensated total and tracked with a counter
// instead. Adding it would poison the compensation term permanently: the eviction of
// a NaN is `add(-NaN)`, which leaves the compensation NaN even after every NaN has
// left the window, so the sum would never recover.
//
// Counting instead makes the poisoning last exactly as long as a NaN is inside the
// window, which is the behaviour a caller expects and the behaviour the
// uncompensated sum has. TestSumRecoversAfterNaNLeavesTheWindow pins it. The cost is
// one comparison and one increment per element, on a path that is already not the
// bottleneck.
func (s *Sum) Push(v float64) float64 {
	old, evicted := s.ring.Push(v)

	if v != v {
		s.nanCount++
	} else {
		s.add(v)
	}
	if evicted {
		if old != old {
			s.nanCount--
		} else {
			s.add(-old)
		}
	}

	if !s.ring.Full() {
		return math.NaN()
	}
	if s.nanCount > 0 {
		return math.NaN()
	}
	return s.total + s.comp
}

// add accumulates v into the running total with Neumaier compensation.
//
// Passing -v performs a compensated subtraction, which is how the evicted value
// leaves the window.
func (s *Sum) add(v float64) {
	s.total, s.comp = neumaierAdd(s.total, s.comp, v)
}

// SMA is a simple moving average: the rolling sum over n values divided by n.
//
// The zero value is not usable; construct with [NewSMA].
type SMA struct {
	sum    *Sum
	window int
}

// NewSMA returns a simple moving average over a window of window values.
//
// It panics if window < 1.
func NewSMA(window int) *SMA {
	return &SMA{sum: NewSum(window), window: window}
}

// Warmup returns the window length.
func (s *SMA) Warmup() int { return s.window }

// Window returns the window length.
func (s *SMA) Window() int { return s.window }

// Reset clears the accumulated state without reallocating.
func (s *SMA) Reset() { s.sum.Reset() }

// Push adds v to the window and returns the current average, or NaN until the window
// is full.
//
// The warm-up test is `math.IsNaN(total)`, which is safe because Sum returns NaN for
// exactly two reasons -- the window is not full, or the window contains a NaN -- and
// both should make the average NaN.
func (s *SMA) Push(v float64) float64 {
	total := s.sum.Push(v)
	if math.IsNaN(total) {
		return math.NaN()
	}
	return total / float64(s.window)
}
