package ta

// ---------------------------------------------------------------------------
// Pivot points.
//
// A pivot calculation turns one completed period's high, low and close into a set of
// support and resistance levels for the next. That is a *scalar* computation on three
// numbers, so the primitive here takes three numbers, and the series form applies it to the
// previous bar of a bar-level frame.
//
// # Why the series form uses the previous bar
//
// The levels for a period must be known at its start, so they can only use the period that
// ended. On bar-level data the "previous period" is the previous bar, so PivotPoints reports
// at index i the levels derived from index i-1. A caller working with daily bars grouped into
// weeks should aggregate first and pass one row per period, which keeps the period definition
// where it belongs: outside the numeric layer.
// ---------------------------------------------------------------------------

// PivotMethod selects a pivot-point formula.
type PivotMethod uint8

const (
	// PivotClassic is the original five-point method: a pivot at the period's mean price,
	// with the second and third levels placed by the period's range.
	PivotClassic PivotMethod = iota
	// PivotFibonacci places the levels at 38.2%, 61.8% and 100% of the period range around
	// the pivot, matching the retracement ratios traders already watch.
	PivotFibonacci
	// PivotCamarilla places the levels as fractions of the range from the *close*, which
	// keeps them close to price and makes them intraday levels rather than targets.
	PivotCamarilla
	// PivotWoodie weights the close twice in the pivot, moving it towards the most recent
	// price.
	PivotWoodie
)

// String returns the method's name, for diagnostics and test failure messages.
func (m PivotMethod) String() string {
	switch m {
	case PivotClassic:
		return "Classic"
	case PivotFibonacci:
		return "Fibonacci"
	case PivotCamarilla:
		return "Camarilla"
	case PivotWoodie:
		return "Woodie"
	default:
		return "Unknown"
	}
}

// PivotLevels returns the pivot and the three support and resistance levels for one
// completed period, given its high, low and close.
//
// The returned order is (pivot, r1, r2, r3, s1, s2, s3). It panics if the method is not one
// of the defined constants, because an unknown method silently falling back to Classic would
// be a wrong answer rather than an error.
//
// The formulas are:
//
//	Classic    P=(H+L+C)/3;  R1=2P-L;  S1=2P-H;  R2=P+(H-L);  S2=P-(H-L);
//	                         R3=H+2(P-L); S3=L-2(H-P)
//	Fibonacci  P=(H+L+C)/3;  Rk=P+k(H-L), Sk=P-k(H-L) for k in {0.382, 0.618, 1}
//	Camarilla  P=(H+L+C)/3;  Rk=C+k(H-L), Sk=C-k(H-L) for k in {1.1/12, 1.1/6, 1.1/4}
//	Woodie     P=(H+L+2C)/4; R1=2P-L;  S1=2P-H;  R2=P+(H-L);  S2=P-(H-L);
//	                         R3=H+2(P-L); S3=L-2(H-P)
//
// The 0.382, 0.618 and 1.1/n constants are part of the definitions rather than tunables.
func PivotLevels(high, low, close float64, method PivotMethod) (pivot, r1, r2, r3, s1, s2, s3 float64) {
	span := high - low
	switch method {
	case PivotClassic:
		pivot = (high + low + close) / 3
		r1, s1 = 2*pivot-low, 2*pivot-high
		r2, s2 = pivot+span, pivot-span
		r3, s3 = high+2*(pivot-low), low-2*(high-pivot)
	case PivotFibonacci:
		pivot = (high + low + close) / 3
		r1, s1 = pivot+0.382*span, pivot-0.382*span
		r2, s2 = pivot+0.618*span, pivot-0.618*span
		r3, s3 = pivot+span, pivot-span
	case PivotCamarilla:
		pivot = (high + low + close) / 3
		r1, s1 = close+span*1.1/12, close-span*1.1/12
		r2, s2 = close+span*1.1/6, close-span*1.1/6
		r3, s3 = close+span*1.1/4, close-span*1.1/4
	case PivotWoodie:
		pivot = (high + low + 2*close) / 4
		r1, s1 = 2*pivot-low, 2*pivot-high
		r2, s2 = pivot+span, pivot-span
		r3, s3 = high+2*(pivot-low), low-2*(high-pivot)
	default:
		panic("ta: PivotLevels unknown method")
	}
	return pivot, r1, r2, r3, s1, s2, s3
}

// PivotPoints returns the pivot and support/resistance levels as series, each element
// derived from the *previous* bar's high, low and close.
//
// Index 0 is NaN for every output, because no completed period precedes it. The levels at
// index i are therefore knowable at the start of bar i, which is what makes them usable as
// levels rather than as hindsight.
//
// All seven outputs have the same length as the input.
func PivotPoints(high, low, close []float64, method PivotMethod) (pivot, r1, r2, r3, s1, s2, s3 []float64) {
	size := requireSameLen("PivotPoints", high, low, close)
	if size == 0 {
		return nil, nil, nil, nil, nil, nil, nil
	}
	pivot = allNaN(size)
	r1 = allNaN(size)
	r2 = allNaN(size)
	r3 = allNaN(size)
	s1 = allNaN(size)
	s2 = allNaN(size)
	s3 = allNaN(size)

	for i := 1; i < size; i++ {
		pivot[i], r1[i], r2[i], r3[i], s1[i], s2[i], s3[i] =
			PivotLevels(high[i-1], low[i-1], close[i-1], method)
	}
	return pivot, r1, r2, r3, s1, s2, s3
}
