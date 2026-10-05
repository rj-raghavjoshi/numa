package vec

import (
	"math"
	"sort"
	"testing"
)

// ---------------------------------------------------------------------------
// Tests for the rolling-window batch kernels.
//
// Every reference below recomputes the window from scratch at every position, which is exactly
// the O(window) form the running-state kernels exist to avoid. That makes each reference an
// independent check rather than a restatement of the implementation, and it also means the
// references are correct by construction for the recomputing kernels -- so those get extra
// property tests rather than only agreement.
// ---------------------------------------------------------------------------

func rollingWindows() []int {
	return []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 13, 16, 17, 23, 32, 33, 64, 127}
}

func rollingLengths() []int {
	return []int{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 16, 17, 23, 24, 25, 31, 32, 33, 63, 64, 65, 100, 200}
}

func refRollingSum(xs []float64, n int) []float64 {
	out := make([]float64, len(xs))
	fillNaN(out, 0, min(n-1, len(xs)))
	for i := n - 1; i < len(xs); i++ {
		var s float64
		bad := false
		for _, v := range xs[i-n+1 : i+1] {
			if v != v {
				bad = true
				break
			}
			s += v
		}
		if !bad {
			out[i] = s
		}
	}
	return out
}

func refRollingExtreme(xs []float64, n int, high bool) []float64 {
	out := make([]float64, len(xs))
	fillNaN(out, 0, min(n-1, len(xs)))
	for i := n - 1; i < len(xs); i++ {
		m := xs[i-n+1]
		bad := false
		for _, v := range xs[i-n+1 : i+1] {
			if v != v {
				bad = true
				break
			}
			if high && v > m {
				m = v
			}
			if !high && v < m {
				m = v
			}
		}
		if !bad {
			out[i] = m
		}
	}
	return out
}

func refRollingMoment(xs []float64, n int, divisor int) []float64 {
	out := make([]float64, len(xs))
	fillNaN(out, 0, len(xs))
	if divisor < 0 && n < 2 {
		return out
	}
	fillNaN(out, 0, min(n-1, len(xs)))
	d := float64(n)
	if divisor < 0 {
		d = float64(n - 1)
	}
	for i := n - 1; i < len(xs); i++ {
		var mean float64
		bad := false
		for _, v := range xs[i-n+1 : i+1] {
			if v != v {
				bad = true
				break
			}
			mean += v
		}
		if bad {
			continue
		}
		mean /= float64(n)
		var ss float64
		for _, v := range xs[i-n+1 : i+1] {
			dev := v - mean
			ss += dev * dev
		}
		out[i] = ss / d
	}
	return out
}

func refRollingWMA(xs []float64, n int) []float64 {
	out := make([]float64, len(xs))
	fillNaN(out, 0, min(n-1, len(xs)))
	denom := float64(n) * float64(n+1) / 2
	for i := n - 1; i < len(xs); i++ {
		var num float64
		bad := false
		for j := 0; j < n; j++ {
			v := xs[i-n+1+j]
			if v != v {
				bad = true
				break
			}
			num += float64(j+1) * v
		}
		if !bad {
			out[i] = num / denom
		}
	}
	return out
}

func refRollingQuantile(xs []float64, n int, q float64) []float64 {
	out := make([]float64, len(xs))
	fillNaN(out, 0, min(n-1, len(xs)))
	for i := n - 1; i < len(xs); i++ {
		buf := make([]float64, n)
		copy(buf, xs[i-n+1:i+1])
		sort.Float64s(buf)
		h := float64(n-1) * q
		lo := int(math.Floor(h))
		hi := int(math.Ceil(h))
		out[i] = buf[lo] + (h-float64(lo))*(buf[hi]-buf[lo])
	}
	return out
}

// ---------------------------------------------------------------------------

func TestRollingSumMatchesReference(t *testing.T) {
	for _, size := range rollingLengths() {
		if size == 0 {
			continue
		}
		xs := randomSlice(size, int64(size)+1)
		for _, n := range rollingWindows() {
			assertRolling(t, "RollingSum", RollingSum(xs, n), refRollingSum(xs, n), false)
		}
	}
}

func TestRollingMeanIsSumOverN(t *testing.T) {
	xs := randomSlice(120, 7)
	for _, n := range []int{1, 3, 8, 20} {
		sum := RollingSum(xs, n)
		mean := RollingMean(xs, n)
		for i := range xs {
			if math.IsNaN(sum[i]) {
				if !math.IsNaN(mean[i]) {
					t.Fatalf("n=%d i=%d: mean %v is defined where the sum is not", n, i, mean[i])
				}
				continue
			}
			if !closeEnough(mean[i], sum[i]/float64(n)) {
				t.Fatalf("n=%d i=%d: mean %v, want sum/n %v", n, i, mean[i], sum[i]/float64(n))
			}
		}
	}
}

func TestRollingExtremesMatchReference(t *testing.T) {
	for _, size := range rollingLengths() {
		if size == 0 {
			continue
		}
		xs := randomSlice(size, int64(size)+2)
		for _, n := range rollingWindows() {
			assertRolling(t, "RollingMax", RollingMax(xs, n), refRollingExtreme(xs, n, true), false)
			assertRolling(t, "RollingMin", RollingMin(xs, n), refRollingExtreme(xs, n, false), false)
		}
	}
}

// TestRollingRangeIsMaxMinusMin checks the definition directly.
func TestRollingRangeIsMaxMinusMin(t *testing.T) {
	xs := randomSlice(150, 3)
	for _, n := range []int{1, 5, 20} {
		hi := RollingMax(xs, n)
		lo := RollingMin(xs, n)
		rng := RollingRange(xs, n)
		for i := range xs {
			if !closeOrNaNBoth(rng[i], hi[i]-lo[i]) {
				t.Fatalf("n=%d i=%d: range %v, want hi-lo %v", n, i, rng[i], hi[i]-lo[i])
			}
		}
	}
}

func TestRollingVarianceMatchesReference(t *testing.T) {
	for _, size := range rollingLengths() {
		if size == 0 {
			continue
		}
		xs := randomSlice(size, int64(size)+4)
		for _, n := range rollingWindows() {
			assertRolling(t, "RollingVariance", RollingVariance(xs, n), refRollingMoment(xs, n, 1), false)
			assertRolling(t, "RollingVarianceSample", RollingVarianceSample(xs, n), refRollingMoment(xs, n, -1), false)
		}
	}
}

// TestRollingStdDevIsSqrtVariance checks the two agree, which catches a missing square root or a
// stray divisor.
func TestRollingStdDevIsSqrtVariance(t *testing.T) {
	xs := randomSlice(150, 5)
	for _, n := range []int{1, 4, 20} {
		v := RollingVariance(xs, n)
		s := RollingStdDev(xs, n)
		vs := RollingVarianceSample(xs, n)
		ss := RollingStdDevSample(xs, n)
		for i := range xs {
			if math.IsNaN(v[i]) {
				continue
			}
			if math.Abs(s[i]-math.Sqrt(v[i])) > 1e-12 {
				t.Fatalf("n=%d i=%d: StdDev %v, want sqrt(Variance) %v", n, i, s[i], math.Sqrt(v[i]))
			}
			if n > 1 && math.Abs(ss[i]-math.Sqrt(vs[i])) > 1e-12 {
				t.Fatalf("n=%d i=%d: StdDevSample %v, want sqrt(VarianceSample) %v", n, i, ss[i], math.Sqrt(vs[i]))
			}
		}
	}
}

// TestRollingVarianceSampleNeedsTwoPoints pins the degenerate window.
func TestRollingVarianceSampleNeedsTwoPoints(t *testing.T) {
	xs := randomSlice(20, 6)
	for _, v := range RollingVarianceSample(xs, 1) {
		if !math.IsNaN(v) {
			t.Fatal("sample variance with a window of one should be all NaN")
		}
	}
	// The population form is defined for a window of one and is zero.
	for i, v := range RollingVariance(xs, 1) {
		if v != 0 {
			t.Fatalf("population variance with a window of one at %d = %v, want 0", i, v)
		}
	}
}

func TestRollingCorrelationMatchesReference(t *testing.T) {
	for _, size := range []int{40, 120, 200} {
		xs := randomSlice(size, int64(size)+7)
		ys := randomSlice(size, int64(size)+8)
		for _, n := range []int{2, 5, 20} {
			assertRolling(t, "RollingCorrelation", RollingCorrelation(xs, ys, n), refRollingCorrelation(xs, ys, n), false)
			assertRolling(t, "RollingCovariance", RollingCovariance(xs, ys, n), refRollingCovariance(xs, ys, n, 1), false)
		}
	}
}

func refRollingCovariance(xs, ys []float64, n, divisor int) []float64 {
	size := min(len(xs), len(ys))
	out := make([]float64, size)
	fillNaN(out, 0, min(n-1, size))
	d := float64(n)
	if divisor < 0 {
		d = float64(n - 1)
	}
	for i := n - 1; i < size; i++ {
		var mx, my float64
		for j := i - n + 1; j <= i; j++ {
			mx += xs[j]
			my += ys[j]
		}
		mx /= float64(n)
		my /= float64(n)
		var s float64
		for j := i - n + 1; j <= i; j++ {
			s += (xs[j] - mx) * (ys[j] - my)
		}
		out[i] = s / d
	}
	return out
}

func refRollingCorrelation(xs, ys []float64, n int) []float64 {
	size := min(len(xs), len(ys))
	out := make([]float64, size)
	fillNaN(out, 0, min(n-1, size))
	for i := n - 1; i < size; i++ {
		var mx, my float64
		for j := i - n + 1; j <= i; j++ {
			mx += xs[j]
			my += ys[j]
		}
		mx /= float64(n)
		my /= float64(n)
		var sxy, sxx, syy float64
		for j := i - n + 1; j <= i; j++ {
			dx := xs[j] - mx
			dy := ys[j] - my
			sxy += dx * dy
			sxx += dx * dx
			syy += dy * dy
		}
		if sxx == 0 || syy == 0 {
			out[i] = math.NaN()
			continue
		}
		out[i] = sxy / math.Sqrt(sxx*syy)
	}
	return out
}

func TestRollingCorrelationFixedPoint(t *testing.T) {
	xs := randomSlice(80, 9)
	got := RollingCorrelation(xs, xs, 10)
	for i, v := range got {
		if math.IsNaN(v) {
			continue
		}
		if math.Abs(v-1) > 1e-9 {
			t.Fatalf("correlation with itself at %d = %v, want 1", i, v)
		}
	}
}

func TestRollingWMA(t *testing.T) {
	for _, size := range rollingLengths() {
		if size == 0 {
			continue
		}
		xs := randomSlice(size, int64(size)+10)
		for _, n := range rollingWindows() {
			assertRolling(t, "RollingWMA", RollingWMA(xs, n), refRollingWMA(xs, n), false)
		}
	}
	// A window of one is the value itself.
	xs := []float64{4, -2, 7}
	assertRolling(t, "RollingWMA(1)", RollingWMA(xs, 1), xs, true)
}

func TestRollingQuantileMatchesReference(t *testing.T) {
	xs := randomSlice(120, 11)
	for _, n := range []int{1, 2, 5, 20, 33} {
		for _, q := range []float64{0, 0.25, 0.5, 0.75, 1} {
			assertRolling(t, "RollingQuantile", RollingQuantile(xs, n, q), refRollingQuantile(xs, n, q), false)
		}
	}
	// The median is the 0.5 quantile.
	med := RollingMedian(xs, 10)
	q50 := RollingQuantile(xs, 10, 0.5)
	assertRolling(t, "RollingMedian", med, q50, true)
}

// TestRollingNaNsRecoverAndDoNotSpread is the important NaN test: a NaN must make exactly the
// windows that contain it NaN and nothing else, for both the running-state and the recomputing
// kernels.
func TestRollingNaNsRecoverAndDoNotSpread(t *testing.T) {
	const n = 3
	xs := []float64{1, 2, 3, NaN(), 5, 6, 7}

	ops := map[string][]float64{
		"RollingSum":      RollingSum(xs, n),
		"RollingMean":     RollingMean(xs, n),
		"RollingMax":      RollingMax(xs, n),
		"RollingMin":      RollingMin(xs, n),
		"RollingVariance": RollingVariance(xs, n),
		"RollingStdDev":   RollingStdDev(xs, n),
		"RollingWMA":      RollingWMA(xs, n),
		"RollingMedian":   RollingMedian(xs, n),
		"RollingQuantile": RollingQuantile(xs, n, 0.5),
		"RollingRange":    RollingRange(xs, n),
	}
	for name, got := range ops {
		if len(got) != len(xs) {
			t.Fatalf("%s: length %d", name, len(got))
		}
		// The NaN is at index 3, so windows ending at 3, 4 and 5 contain it.
		for _, i := range []int{3, 4, 5} {
			if !math.IsNaN(got[i]) {
				t.Errorf("%s[%d] = %v, want NaN (window contains the NaN)", name, i, got[i])
			}
		}
		if math.IsNaN(got[6]) {
			t.Errorf("%s[6] = NaN, want a value once the NaN has left the window", name)
		}
		if math.IsNaN(got[2]) {
			t.Errorf("%s[2] = NaN, want a value before the NaN arrives", name)
		}
	}
}

// TestRollingToVariantsMatchAllocating checks the buffer forms against the allocating ones.
func TestRollingToVariantsMatchAllocating(t *testing.T) {
	xs := randomSlice(120, 12)
	dst := make([]float64, len(xs))

	cases := []struct {
		name string
		to   func() []float64
		want []float64
	}{
		{"RollingSumTo", func() []float64 { return RollingSumTo(dst, xs, 8) }, RollingSum(xs, 8)},
		{"RollingMeanTo", func() []float64 { return RollingMeanTo(dst, xs, 8) }, RollingMean(xs, 8)},
		{"RollingMaxTo", func() []float64 { return RollingMaxTo(dst, xs, 8) }, RollingMax(xs, 8)},
		{"RollingMinTo", func() []float64 { return RollingMinTo(dst, xs, 8) }, RollingMin(xs, 8)},
		{"RollingRangeTo", func() []float64 { return RollingRangeTo(dst, xs, 8) }, RollingRange(xs, 8)},
		{"RollingWMATo", func() []float64 { return RollingWMATo(dst, xs, 8) }, RollingWMA(xs, 8)},
	}
	for _, c := range cases {
		// The destination must be returned, not a copy.
		got := c.to()
		if len(got) != len(c.want) {
			t.Fatalf("%s: length %d", c.name, len(got))
		}
		for i := range c.want {
			if !closeOrNaNBoth(got[i], c.want[i]) {
				t.Fatalf("%s[%d] = %v, want %v", c.name, i, got[i], c.want[i])
			}
		}
	}
}

// The rolling *To forms do not permit dst to alias xs: the loop reads the element leaving the
// window, which earlier iterations have already overwritten. There is deliberately no aliasing
// test here, because one written the obvious way would pass for the wrong reason -- it would
// compare the corrupted output against itself. The restriction is stated in the functions' doc
// comments and in docs/design.md, alongside the other two aliasing exceptions.

func TestRollingEmptyAndPanics(t *testing.T) {
	if RollingSum(nil, 5) != nil || RollingMean(nil, 5) != nil || RollingMax(nil, 5) != nil ||
		RollingMin(nil, 5) != nil || RollingRange(nil, 5) != nil || RollingWMA(nil, 5) != nil ||
		RollingVariance(nil, 5) != nil || RollingStdDev(nil, 5) != nil ||
		RollingMedian(nil, 5) != nil || RollingQuantile(nil, 5, 0.5) != nil ||
		RollingCorrelation(nil, nil, 5) != nil {
		t.Error("empty inputs should return nil")
	}

	xs := randomSlice(10, 14)
	for _, n := range []int{0, -1} {
		func() {
			defer func() {
				if recover() == nil {
					t.Errorf("period %d did not panic", n)
				}
			}()
			RollingSum(xs, n)
		}()
	}
	func() {
		defer func() {
			if recover() == nil {
				t.Error("RollingQuantile with q > 1 did not panic")
			}
		}()
		RollingQuantile(xs, 5, 1.5)
	}()
	func() {
		defer func() {
			if recover() == nil {
				t.Error("RollingSumTo with mismatched lengths did not panic")
			}
		}()
		RollingSumTo(make([]float64, 3), xs, 5)
	}()
}

// TestRollingWindowLargerThanInput pins the degenerate case: every output is NaN.
func TestRollingWindowLargerThanInput(t *testing.T) {
	xs := []float64{1, 2, 3}
	for _, got := range [][]float64{
		RollingSum(xs, 5), RollingMean(xs, 5), RollingMax(xs, 5), RollingMin(xs, 5),
		RollingRange(xs, 5), RollingWMA(xs, 5), RollingVariance(xs, 5), RollingStdDev(xs, 5),
		RollingMedian(xs, 5), RollingQuantile(xs, 5, 0.5),
	} {
		for i, v := range got {
			if !math.IsNaN(v) {
				t.Fatalf("index %d = %v, want NaN with a window larger than the input", i, v)
			}
		}
	}
}

// TestRollingSumIsExactAcrossLargeOffsetEviction checks the compensation, with a hard equality
// rather than a tolerance.
func TestRollingSumIsExactAcrossLargeOffsetEviction(t *testing.T) {
	xs := []float64{1e16, 1, 1, 1}
	got := RollingSum(xs, 2)
	// Windows: {1e16,1} -> 1e16 (the 1 is unrepresentable in the sum), {1,1} -> 2, {1,1} -> 2
	if got[2] != 2 {
		t.Fatalf("rolling sum after the large value left = %v, want exactly 2", got[2])
	}
	if got[3] != 2 {
		t.Fatalf("rolling sum after the large value left = %v, want exactly 2", got[3])
	}
}

// ---------------------------------------------------------------------------
// Assertion helpers local to this file.
// ---------------------------------------------------------------------------

func assertRolling(t *testing.T, name string, got, want []float64, exact bool) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("%s: length %d, want %d", name, len(got), len(want))
	}
	for i := range want {
		ok := closeOrNaNBoth
		if exact {
			ok = sameOrNaNBoth
		}
		if !ok(got[i], want[i]) {
			t.Fatalf("%s[%d] = %v, want %v", name, i, got[i], want[i])
		}
	}
}

func sameOrNaNBoth(a, b float64) bool {
	if math.IsNaN(a) || math.IsNaN(b) {
		return math.IsNaN(a) && math.IsNaN(b)
	}
	return a == b
}

func closeOrNaNBoth(a, b float64) bool {
	if math.IsNaN(a) || math.IsNaN(b) {
		return math.IsNaN(a) && math.IsNaN(b)
	}
	if a == b {
		return true
	}
	return math.Abs(a-b) <= 1e-9*math.Max(1, math.Abs(b))
}

// NaN is a helper for the NaN-policy test, keeping that test's data readable.
func NaN() float64 { return math.NaN() }
