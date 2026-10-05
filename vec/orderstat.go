package vec

import (
	"math"
	"sort"
)

// ---------------------------------------------------------------------------
// Sliding-window order statistics, maintained incrementally.
//
// The quantile kernels used to copy each window and select from it with quickselect. That is O(n)
// per element and, for the median, measured 927 ns/element at a 256-bar window -- the most expensive
// kernel in the package. Recomputing a window's order from scratch throws away the fact that
// consecutive windows differ by two elements.
//
// This file keeps the order instead. Values are compressed to integer ranks once, and a binary
// indexed tree holds the *count* of each rank inside the current window. Inserting and removing an
// element is then O(log U) for U distinct values, and asking for the k-th smallest is a single
// descent of the tree.
//
// # Why a Fenwick tree rather than two heaps
//
// A pair of heaps gives the median in O(log n) but cannot remove an arbitrary value: the element
// leaving the window is not necessarily at a heap root, so a heap has to either store indices and
// sift, or delete lazily and carry dead entries. A Fenwick tree has neither problem, and it answers
// a general k-th smallest, which is what a quantile needs -- the median is just k = n/2.
//
// # Why compression is worth it
//
// The tree is indexed by rank rather than by value, so its size is the number of distinct values and
// not their range. The compression sort costs O(N log N) once per call and is dwarfed by the rolling
// pass when N is large; a caller with a short series and a short window is better served by
// [Quantile], which is why that function still exists.
// ---------------------------------------------------------------------------

// orderStat maintains the counts of a sliding window over a compressed value domain.
type orderStat struct {
	tree     []int32   // 1-based Fenwick tree over ranks
	values   []float64 // the distinct non-NaN values, ascending; values[r-1] is rank r
	size     int       // the number of distinct values
	top      int       // the largest power of two <= size, for the descent
	nanCount int       // NaNs currently inside the window
}

// newOrderStat builds the compressed domain for xs.
func newOrderStat(xs []float64) *orderStat {
	vals := make([]float64, 0, len(xs))
	for _, v := range xs {
		if v == v {
			vals = append(vals, v)
		}
	}
	sort.Float64s(vals)
	// In-place dedupe: the write index never passes the read index.
	n := 0
	for i, v := range vals {
		if i == 0 || v != vals[i-1] {
			vals[n] = v
			n++
		}
	}
	vals = vals[:n]

	top := 1
	for top<<1 <= n {
		top <<= 1
	}
	return &orderStat{
		tree:   make([]int32, n+1),
		values: vals,
		size:   n,
		top:    top,
	}
}

// rankOf returns the 1-based compressed rank of v, which must be an existing value.
func (o *orderStat) rankOf(v float64) int {
	return sort.SearchFloat64s(o.values, v) + 1
}

// add applies a count delta to a rank.
func (o *orderStat) add(rank int, delta int32) {
	for i := rank; i <= o.size; i += i & (-i) {
		o.tree[i] += delta
	}
}

// below returns how many window elements have a rank strictly less than the given rank.
func (o *orderStat) below(rank int) int32 {
	var sum int32
	for i := rank - 1; i > 0; i -= i & (-i) {
		sum += o.tree[i]
	}
	return sum
}

// kth returns the rank of the k-th smallest element, with k 1-based.
//
// The descent is the standard Fenwick search: at each step it tests whether the accumulated count
// up to the candidate index is still short of k, and if so moves there and subtracts. It visits one
// node per bit of the domain size, which is what makes it O(log U) rather than a scan.
func (o *orderStat) kth(k int32) int {
	idx := 0
	for bit := o.top; bit > 0; bit >>= 1 {
		next := idx + bit
		if next <= o.size && o.tree[next] < k {
			idx = next
			k -= o.tree[idx]
		}
	}
	return idx + 1
}

// compressedRanks returns the 1-based rank of every non-NaN element of xs, with 0 for the NaNs.
//
// Doing the binary searches once up front rather than per window is the difference between O(N)
// lookups and O(N) lookups that each cost log U; the ranks never change, so there is no reason to
// recompute them.
func (o *orderStat) compressedRanks(xs []float64) []int32 {
	ranks := make([]int32, len(xs))
	for i, v := range xs {
		if v == v {
			ranks[i] = int32(o.rankOf(v))
		}
	}
	return ranks
}

// RollingMedian returns the trailing n-value median at each position.
//
// It is the 0.5 quantile, computed by the same sliding order statistic as [RollingQuantile], so its
// cost is O(log U) per element rather than the O(n) a window rescan costs. For an even window it is
// the mean of the two middle values, matching [Median].
//
// The first n-1 values are NaN, and a window containing a NaN produces NaN.
func RollingMedian(xs []float64, n int) []float64 {
	return rollingQuantileOrderStat(xs, n, 0.5)
}

// RollingQuantile returns the trailing n-value q-quantile at each position, using the same linear
// interpolation between order statistics as [Quantile].
//
// It is O(log U) amortised per element for U distinct values in the input, against O(n) for a
// window rescan. The first n-1 values are NaN, and a window containing a NaN produces NaN.
//
// It panics if q is outside [0,1].
func RollingQuantile(xs []float64, n int, q float64) []float64 {
	return rollingQuantileOrderStat(xs, n, q)
}

// rollingQuantileOrderStat is the shared implementation behind [RollingMedian] and
// [RollingQuantile].
func rollingQuantileOrderStat(xs []float64, n int, q float64) []float64 {
	checkPeriodRolling("RollingQuantile", n)
	if q < 0 || q > 1 || q != q {
		panic("vec: RollingQuantile q must be in [0,1]")
	}
	if len(xs) == 0 {
		return nil
	}
	size := len(xs)
	out := make([]float64, size)
	// The whole output starts NaN: a window containing a NaN is skipped below and must read as NaN
	// rather than as the zero value.
	fillNaN(out, 0, size)
	if size < n {
		return out
	}

	os := newOrderStat(xs)
	if os.size == 0 {
		// Every value is NaN, so no window is ever defined.
		return out
	}
	ranks := os.compressedRanks(xs)

	// The interpolation weights are fixed by q and n, so they are computed once.
	pos := float64(n-1) * q
	lo := int(math.Floor(pos))
	hi := int(math.Ceil(pos))
	frac := pos - float64(lo)
	kLo := int32(lo + 1)
	kHi := int32(hi + 1)

	for i := 0; i < size; i++ {
		if r := ranks[i]; r != 0 {
			os.add(int(r), 1)
		} else {
			os.nanCount++
		}
		if i >= n {
			if r := ranks[i-n]; r != 0 {
				os.add(int(r), -1)
			} else {
				os.nanCount--
			}
		}
		if i < n-1 || os.nanCount > 0 {
			continue
		}
		vLo := os.values[os.kth(kLo)-1]
		if lo == hi {
			out[i] = vLo
			continue
		}
		vHi := os.values[os.kth(kHi)-1]
		out[i] = vLo + frac*(vHi-vLo)
	}
	return out
}

// RollingPercentRank returns the percentage of the trailing n values that are strictly less than the
// current one, at each position.
//
// The sliding order statistic answers this with a single prefix count, so the cost is O(log U) per
// element for U distinct values rather than the O(n) a window rescan costs. That matters more here
// than for the quantile: a caller reaching for a percent rank is usually asking about a long lookback,
// which is exactly where a rescan is worst.
//
// The window includes the current value, which cannot be less than itself, so the maximum output is
// 100*(n-1)/n. The first n-1 values are NaN, and a window containing a NaN produces NaN.
func RollingPercentRank(xs []float64, n int) []float64 {
	checkPeriodRolling("RollingPercentRank", n)
	if len(xs) == 0 {
		return nil
	}
	size := len(xs)
	out := make([]float64, size)
	fillNaN(out, 0, size)
	if size < n {
		return out
	}
	os := newOrderStat(xs)
	if os.size == 0 {
		return out
	}
	ranks := os.compressedRanks(xs)

	inv := 100 / float64(n)
	for i := 0; i < size; i++ {
		if r := ranks[i]; r != 0 {
			os.add(int(r), 1)
		} else {
			os.nanCount++
		}
		if i >= n {
			if r := ranks[i-n]; r != 0 {
				os.add(int(r), -1)
			} else {
				os.nanCount--
			}
		}
		if i < n-1 || os.nanCount > 0 {
			continue
		}
		out[i] = float64(os.below(int(ranks[i]))) * inv
	}
	return out
}
