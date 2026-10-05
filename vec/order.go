package vec

import (
	"math"
	"math/bits"
	"sort"
)

// ---------------------------------------------------------------------------
// Order statistics: median, quantile, percentile, rank, and sorting.
//
// # Why quickselect and not sort
//
// The obvious way to find the median is to sort a copy and index the middle. That
// is O(n log n) for a question that only needs O(n): the value at position k does
// not require every other value to be ordered.
//
// `selectK` is a quickselect -- Hoare partitioning around a median-of-three pivot,
// recursing only into the side that contains k. It places the k-th smallest element
// at index k and leaves the rest merely partitioned. The measured difference at
// n=4M is large enough to justify the extra code (docs/benchmarks.md).
//
// Two guards make it safe rather than merely fast:
//
//   - Ranges shorter than 12 elements fall back to insertion sort, which beats
//     partitioning on small inputs and terminates the recursion cheaply.
//   - A recursion depth limit falls back to `sort.Float64s` on the remaining range.
//     Plain quickselect is O(n^2) on adversarial input; the fallback restores an
//     O(n log n) worst case at the cost of one comparison and one branch per level.
//
// # NaN policy
//
// Any NaN in the input makes the result NaN for every function here. The check is
// fused into the copy that these functions already have to make, so it costs no
// extra pass. This is the same "NaN wins" policy as [Min] and [Max]; a median
// computed from the surviving elements would hide the upstream error that produced
// the NaN.
//
// # Interpolation convention
//
// [Quantile] uses linear interpolation between order statistics -- the same
// convention as Pine's `array.percentile_linear_interpolation`, NumPy's default,
// and R's type 7. For q in [0,1] and sorted values x, with h = (n-1)q:
//
//	result = x[floor(h)] + (h - floor(h)) * (x[ceil(h)] - x[floor(h)])
//
// This is the convention that makes the median of an even-length sample the
// average of the two middle values, which is what a caller expects.
// ---------------------------------------------------------------------------

// copyChecked copies src into dst and reports whether every element was non-NaN.
//
// It fuses the NaN test into the copy these order statistics need anyway, so the
// NaN policy costs nothing beyond the work already being done. It returns early on
// the first NaN, leaving the rest of dst untouched, because the caller discards dst
// in that case.
func copyChecked(dst, src []float64) bool {
	for i, v := range src {
		if v != v {
			return false
		}
		dst[i] = v
	}
	return true
}

// insertionSort sorts buf in place. It is used for short ranges inside selectK.
func insertionSort(buf []float64) {
	for i := 1; i < len(buf); i++ {
		v := buf[i]
		j := i - 1
		for j >= 0 && buf[j] > v {
			buf[j+1] = buf[j]
			j--
		}
		buf[j+1] = v
	}
}

// partition is a Hoare partition of buf[lo:hi+1] around a median-of-three pivot.
//
// It returns j such that buf[lo:j] <= pivot and buf[j+1:hi] >= pivot, and leaves
// the pivot value itself somewhere in that split. Median-of-three ordering of
// lo/mid/hi before partitioning also installs sentinels: buf[hi] >= pivot bounds
// the left scan and buf[lo] <= pivot bounds the right scan, so neither inner loop
// needs a bounds check.
func partition(buf []float64, lo, hi int) int {
	mid := lo + (hi-lo)/2
	if buf[mid] < buf[lo] {
		buf[lo], buf[mid] = buf[mid], buf[lo]
	}
	if buf[hi] < buf[lo] {
		buf[lo], buf[hi] = buf[hi], buf[lo]
	}
	if buf[hi] < buf[mid] {
		buf[mid], buf[hi] = buf[hi], buf[mid]
	}
	pivot := buf[mid]

	i, j := lo, hi
	for {
		for buf[i] < pivot {
			i++
		}
		for buf[j] > pivot {
			j--
		}
		if i >= j {
			return j
		}
		buf[i], buf[j] = buf[j], buf[i]
		i++
		j--
	}
}

// selectK places the k-th smallest element of buf at index k and partitions the
// rest around it. Elements before k are <= buf[k]; elements after are >= buf[k].
// Neither side is sorted.
//
// buf must contain no NaN: the public entry points reject NaN before calling this,
// so the comparisons below cannot meet one.
func selectK(buf []float64, k int) {
	n := len(buf)
	if n <= 1 {
		return
	}
	if k < 0 {
		k = 0
	}
	if k >= n {
		k = n - 1
	}

	// A depth cap of 2*log2(n) is the standard introselect bound. Exceeding it
	// means the pivots are degenerate, so the remaining range is delegated to a
	// guaranteed O(n log n) sort.
	maxDepth := 2 * bits.Len(uint(n))

	lo, hi := 0, n-1
	depth := 0
	for lo < hi {
		if hi-lo < 12 {
			insertionSort(buf[lo : hi+1])
			return
		}
		if depth > maxDepth {
			sort.Float64s(buf[lo : hi+1])
			return
		}
		j := partition(buf, lo, hi)
		// Recurse into the side containing k, iteratively, so the stack depth
		// stays constant.
		if k <= j {
			hi = j
		} else {
			lo = j + 1
		}
		depth++
	}
}

// quantileOwned computes the q-quantile of buf, mutating buf as scratch. buf must
// be non-empty and NaN-free.
func quantileOwned(buf []float64, q float64) float64 {
	n := len(buf)
	if n == 1 {
		return buf[0]
	}
	h := float64(n-1) * q
	lo := int(math.Floor(h))
	hi := int(math.Ceil(h))
	if lo < 0 {
		lo = 0
	}
	if hi > n-1 {
		hi = n - 1
	}

	selectK(buf, lo)
	vlo := buf[lo]
	if hi == lo {
		return vlo
	}
	selectK(buf, hi)
	vhi := buf[hi]
	return vlo + (h-float64(lo))*(vhi-vlo)
}

// Median returns the median of xs, or NaN if xs is empty or contains NaN.
//
// It allocates a scratch copy of xs. For hot loops that can supply a buffer, use
// [MedianInto].
func Median(xs []float64) float64 {
	if len(xs) == 0 {
		return math.NaN()
	}
	return MedianInto(xs, make([]float64, len(xs)))
}

// MedianInto returns the median of xs using buf as scratch, avoiding allocation.
//
// buf must have length at least len(xs); MedianInto panics otherwise. The contents
// of buf are destroyed, and xs is never modified.
func MedianInto(xs []float64, buf []float64) float64 {
	n := len(xs)
	if n == 0 {
		return math.NaN()
	}
	if len(buf) < n {
		panic("vec: MedianInto buffer too small")
	}
	b := buf[:n]
	if !copyChecked(b, xs) {
		return math.NaN()
	}
	return quantileOwned(b, 0.5)
}

// Quantile returns the q-quantile of xs for q in [0,1], using linear interpolation
// between order statistics.
//
// It returns NaN if xs is empty or contains NaN, and panics if q is outside [0,1].
// Quantile allocates a scratch copy of xs; for hot loops that can supply a buffer,
// use [QuantileInto].
func Quantile(xs []float64, q float64) float64 {
	if len(xs) == 0 {
		return math.NaN()
	}
	return QuantileInto(xs, q, make([]float64, len(xs)))
}

// QuantileInto returns the q-quantile of xs using buf as scratch.
//
// buf must have length at least len(xs); QuantileInto panics otherwise, or if q is
// outside [0,1]. The contents of buf are destroyed.
func QuantileInto(xs []float64, q float64, buf []float64) float64 {
	if q < 0 || q > 1 || q != q {
		panic("vec: Quantile q must be in [0,1]")
	}
	n := len(xs)
	if n == 0 {
		return math.NaN()
	}
	if len(buf) < n {
		panic("vec: QuantileInto buffer too small")
	}
	b := buf[:n]
	if !copyChecked(b, xs) {
		return math.NaN()
	}
	return quantileOwned(b, q)
}

// Percentile returns the p-th percentile of xs for p in [0,100].
//
// It is [Quantile] with the scale a caller is more likely to have: p/100. It
// returns NaN if xs is empty or contains NaN, and panics if p is outside [0,100].
func Percentile(xs []float64, p float64) float64 {
	if len(xs) == 0 {
		return math.NaN()
	}
	return PercentileInto(xs, p, make([]float64, len(xs)))
}

// PercentileInto returns the p-th percentile of xs using buf as scratch.
//
// buf must have length at least len(xs); PercentileInto panics otherwise, or if p
// is outside [0,100]. The contents of buf are destroyed.
func PercentileInto(xs []float64, p float64, buf []float64) float64 {
	if p < 0 || p > 100 || p != p {
		panic("vec: Percentile p must be in [0,100]")
	}
	return QuantileInto(xs, p/100, buf)
}

// SortInPlace sorts xs ascending in place and returns it.
//
// NaN values sort before every number, following sort.Float64s. A caller who needs
// NaN excluded or rejected should test with [HasNaN] first.
func SortInPlace(xs []float64) []float64 {
	sort.Float64s(xs)
	return xs
}

// SortCopy returns a sorted copy of xs. It returns nil for an empty input.
func SortCopy(xs []float64) []float64 {
	if len(xs) == 0 {
		return nil
	}
	out := make([]float64, len(xs))
	copy(out, xs)
	sort.Float64s(out)
	return out
}

// Rank assigns each element its 1-based rank in ascending order, with ties sharing
// the average of the ranks they span.
//
// For example {10, 20, 20, 30} ranks as {1, 2.5, 2.5, 4}. NaN positions receive
// NaN, and the remaining values are ranked among themselves; a NaN is not a value,
// so it cannot occupy a rank.
//
// Rank allocates both the result and a sorted scratch copy. For hot loops that can
// supply a destination, use [RankTo]; the scratch is still allocated, because
// ranking requires an ordered view of the values.
func Rank(xs []float64) []float64 {
	if len(xs) == 0 {
		return nil
	}
	return RankTo(make([]float64, len(xs)), xs)
}

// RankTo stores the ascending average ranks of xs into dst and returns dst.
//
// dst and xs must be the same length. RankTo panics if they are not.
func RankTo(dst, xs []float64) []float64 {
	if len(dst) != len(xs) {
		panic("vec: RankTo length mismatch")
	}
	return RankScratchTo(dst, xs, make([]int, len(xs)))
}

// RankScratchTo stores the ascending average ranks of xs into dst, using scratch as its working
// space, and returns dst.
//
// It exists for callers that rank a window repeatedly -- a rolling rank correlation, a rank
// statistic -- where the allocation RankTo performs on every call is pure overhead. scratch needs
// length at least len(xs); its contents are unspecified on return, and passing nil is legal and
// simply allocates.
//
// Ties receive the average of the ranks they span, and a NaN receives NaN rather than a rank,
// because a NaN has no position in an ordering and must not shift the ranks of the values around it.
//
// dst and xs must be the same length, or RankScratchTo panics.
func RankScratchTo(dst, xs []float64, scratch []int) []float64 {
	if len(dst) != len(xs) {
		panic("vec: RankScratchTo length mismatch")
	}
	// # Why this sorts indices rather than values
	//
	// The obvious implementation copies the values, sorts the copy, and then binary-searches each
	// original value to find its rank. That is O(n log n) plus O(n log n) -- two binary searches per
	// element for the lower and upper ends of its tie group -- and it was the dominant cost of the
	// rolling rank correlation, which ranks two windows per bar.
	//
	// Sorting indices instead costs one sort and one linear scan: the ranks fall out of walking the
	// sorted order once, and ties are one comparison per element.
	idx := scratch[:0]
	for i, v := range xs {
		if v == v {
			idx = append(idx, i)
		}
	}

	// Insertion sort for the window sizes this is normally called with -- fifteen to twenty orders
	// of magnitude fewer comparisons than a comparison sort, with no interface dispatch -- and a
	// comparison sort above that so a large input does not degrade quadratically.
	const insertionLimit = 24
	if len(idx) <= insertionLimit {
		for a := 1; a < len(idx); a++ {
			key := idx[a]
			kv := xs[key]
			b := a - 1
			for b >= 0 && xs[idx[b]] > kv {
				idx[b+1] = idx[b]
				b--
			}
			idx[b+1] = key
		}
	} else {
		sort.Sort(indexByValue{idx: idx, xs: xs})
	}

	for i := range dst {
		dst[i] = math.NaN()
	}
	for a := 0; a < len(idx); {
		b := a + 1
		for b < len(idx) && xs[idx[b]] == xs[idx[a]] {
			b++
		}
		// The tie group occupies sorted positions a .. b-1, whose average rank is (a+b-1)/2 in
		// 0-based terms, plus one.
		avg := float64(a+b-1)/2 + 1
		for k := a; k < b; k++ {
			dst[idx[k]] = avg
		}
		a = b
	}
	return dst
}

// indexByValue sorts a slice of indices by the values they point at, without allocating.
//
// It is defined as a type rather than used through sort.Slice because sort.Slice reflects over its
// operand on every comparison, which is far more expensive than the comparison itself at these
// sizes.
type indexByValue struct {
	idx []int
	xs  []float64
}

func (b indexByValue) Len() int           { return len(b.idx) }
func (b indexByValue) Less(i, j int) bool { return b.xs[b.idx[i]] < b.xs[b.idx[j]] }
func (b indexByValue) Swap(i, j int)      { b.idx[i], b.idx[j] = b.idx[j], b.idx[i] }

// MAD returns the median absolute deviation of xs: the median of |x - median(x)|.
//
// It is a robust spread measure, unlike the standard deviation, which a single
// outlier can dominate. It allocates two scratch buffers.
//
// It returns NaN for an empty input, for an input containing NaN, and for an input
// whose median is infinite -- a deviation from an infinity is not defined.
//
// The median it reports is the one [Median] computes, including the even-length
// averaging, so `MAD(xs) == Median(deviations)` holds by construction.
func MAD(xs []float64) float64 {
	n := len(xs)
	if n == 0 {
		return math.NaN()
	}
	buf := make([]float64, n)
	if !copyChecked(buf, xs) {
		return math.NaN()
	}
	med := quantileOwned(buf, 0.5)
	if math.IsInf(med, 0) {
		return math.NaN()
	}
	// buf was permuted by quantileOwned, so every element is overwritten here.
	for i, v := range xs {
		buf[i] = math.Abs(v - med)
	}
	return quantileOwned(buf, 0.5)
}
