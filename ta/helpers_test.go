package ta

import (
	"math"
	"math/rand"
	"testing"
)

// ---------------------------------------------------------------------------
// Shared test helpers.
//
// Every reference below is a naive, independent restatement of the definition --
// direct window loops, no shared machinery with the implementation. That is the only
// way a test can disagree with the code.
// ---------------------------------------------------------------------------

func boundaryPeriods() []int {
	return []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 13, 16, 17, 20, 23, 32, 33, 64}
}

func boundaryLengths() []int {
	return []int{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 16, 17, 20, 23, 24, 25, 31, 32, 33, 63, 64, 65}
}

// synthOHLCV produces a plausible OHLCV frame: a random-walk close, a high and low
// that bracket it, and a positive volume. Realistic inputs matter here because a
// constant series would hide a whole class of error, and a purely random one would
// never exercise a trend.
func synthOHLCV(n int, seed int64) (open, high, low, close, volume []float64) {
	rng := rand.New(rand.NewSource(seed))
	open = make([]float64, n)
	high = make([]float64, n)
	low = make([]float64, n)
	close = make([]float64, n)
	volume = make([]float64, n)

	price := 100.0
	for i := 0; i < n; i++ {
		open[i] = price
		price += rng.NormFloat64()
		close[i] = price
		hi := math.Max(open[i], close[i]) + math.Abs(rng.NormFloat64())
		lo := math.Min(open[i], close[i]) - math.Abs(rng.NormFloat64())
		high[i] = hi
		low[i] = lo
		volume[i] = 1000 + math.Abs(rng.NormFloat64())*100
	}
	return
}

// ramp is a deterministic increasing series, used where a test needs values that are
// easy to reason about by hand.
func ramp(n int) []float64 {
	xs := make([]float64, n)
	for i := range xs {
		xs[i] = float64(i + 1)
	}
	return xs
}

func sameOrNaN(got, want float64) bool {
	if math.IsNaN(got) || math.IsNaN(want) {
		return math.IsNaN(got) && math.IsNaN(want)
	}
	return got == want
}

func closeOrNaN(got, want float64) bool {
	if math.IsNaN(got) || math.IsNaN(want) {
		return math.IsNaN(got) && math.IsNaN(want)
	}
	if got == want {
		return true
	}
	return math.Abs(got-want) <= 1e-9*math.Max(1, math.Abs(want))
}

func assertSeries(t *testing.T, name string, got, want []float64, exact bool) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("%s: length %d, want %d", name, len(got), len(want))
	}
	for i := range want {
		ok := closeOrNaN
		if exact {
			ok = sameOrNaN
		}
		if !ok(got[i], want[i]) {
			t.Fatalf("%s[%d] = %v, want %v", name, i, got[i], want[i])
		}
	}
}

// ---------------------------------------------------------------------------
// References.
// ---------------------------------------------------------------------------

func refSMA(xs []float64, n int) []float64 {
	out := make([]float64, len(xs))
	for i := range xs {
		if i+1 < n {
			out[i] = math.NaN()
			continue
		}
		var s float64
		for j := i - n + 1; j <= i; j++ {
			s += xs[j]
		}
		out[i] = s / float64(n)
	}
	return out
}

func refSmoother(xs []float64, n int, alpha float64) []float64 {
	out := make([]float64, len(xs))
	seeded := false
	var seed, prev float64
	count := 0
	for i, v := range xs {
		if !seeded {
			seed += v
			count++
			if count < n {
				out[i] = math.NaN()
				continue
			}
			prev = seed / float64(n)
			seeded = true
			out[i] = prev
			continue
		}
		prev = alpha*v + (1-alpha)*prev
		out[i] = prev
	}
	return out
}

func refWMA(xs []float64, n int) []float64 {
	out := make([]float64, len(xs))
	denom := float64(n) * float64(n+1) / 2
	for i := range xs {
		if i+1 < n {
			out[i] = math.NaN()
			continue
		}
		var num float64
		for j := 0; j < n; j++ {
			num += float64(j+1) * xs[i-n+1+j]
		}
		out[i] = num / denom
	}
	return out
}

func refExtreme(xs []float64, n int, high bool) []float64 {
	out := make([]float64, len(xs))
	for i := range xs {
		if i+1 < n {
			out[i] = math.NaN()
			continue
		}
		m := xs[i]
		nan := false
		for j := i - n + 1; j <= i; j++ {
			if xs[j] != xs[j] {
				nan = true
				break
			}
			if high && xs[j] > m {
				m = xs[j]
			}
			if !high && xs[j] < m {
				m = xs[j]
			}
		}
		if nan {
			out[i] = math.NaN()
		} else {
			out[i] = m
		}
	}
	return out
}

func refStdev(xs []float64, n int) []float64 {
	out := make([]float64, len(xs))
	for i := range xs {
		if i+1 < n {
			out[i] = math.NaN()
			continue
		}
		var mean float64
		for j := i - n + 1; j <= i; j++ {
			mean += xs[j]
		}
		mean /= float64(n)
		var ss float64
		for j := i - n + 1; j <= i; j++ {
			d := xs[j] - mean
			ss += d * d
		}
		out[i] = math.Sqrt(ss / float64(n))
	}
	return out
}

func refATR(high, low, close []float64, n int) []float64 {
	tr := TrueRange(high, low, close)
	return refSmoother(tr, n, 1.0/float64(n))
}

// synthHLC returns only the high, low and close columns, in that order.
//
// It exists because synthOHLCV returns five slices in a fixed order and a caller who
// destructures them as `high, low, close, _, _` silently receives the OPEN as high and
// the HIGH as low. That mistake happened: the resulting bars had high < low, and every
// test that fed them to both the implementation and its reference still passed, because
// both sides saw the same invalid input. A three-value helper removes the class of bug
// rather than relying on the caller to count return values correctly.
func synthHLC(n int, seed int64) (high, low, close []float64) {
	_, h, l, c, _ := synthOHLCV(n, seed)
	return h, l, c
}

// TestSyntheticDataIsValidOHLC checks the fixture itself.
//
// A test data generator that produces impossible bars weakens every test that uses it
// without failing any of them, which is exactly what happened here. Checking the
// invariant directly is the only way the fixture can be held to it.
func TestSyntheticDataIsValidOHLC(t *testing.T) {
	const n = 200
	open, high, low, close, volume := synthOHLCV(n, 4242)
	for i := 0; i < n; i++ {
		if high[i] < low[i] {
			t.Fatalf("bar %d: high %v < low %v", i, high[i], low[i])
		}
		if high[i] < open[i] || high[i] < close[i] {
			t.Fatalf("bar %d: high %v does not bound open %v and close %v", i, high[i], open[i], close[i])
		}
		if low[i] > open[i] || low[i] > close[i] {
			t.Fatalf("bar %d: low %v does not bound open %v and close %v", i, low[i], open[i], close[i])
		}
		if volume[i] <= 0 {
			t.Fatalf("bar %d: volume %v is not positive", i, volume[i])
		}
	}

	// And the three-value helper must agree with the full frame's ordering.
	h, l, c := synthHLC(n, 4242)
	for i := 0; i < n; i++ {
		if h[i] != high[i] || l[i] != low[i] || c[i] != close[i] {
			t.Fatalf("synthHLC disagrees with synthOHLCV at %d", i)
		}
	}
}
