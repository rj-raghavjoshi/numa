package numa

import "github.com/rj-raghavjoshi/numa/vec"

// ---------------------------------------------------------------------------
// Facade forwarders for vec/signal.go: lag, difference, and signal primitives.
//
// Each is a one-line forwarder to vec. See facade_math.go for why the forwarders
// are split per family, and facade_signal_test.go for the forwarding tests.
// ---------------------------------------------------------------------------

// Shift returns xs displaced by lag bars, with a positive lag looking backward.
// See [vec.Shift].
func Shift(xs []float64, lag int) []float64 { return vec.Shift(xs, lag) }

// ShiftTo stores xs displaced by lag bars into dst; dst must not alias xs.
// See [vec.ShiftTo].
func ShiftTo(dst, xs []float64, lag int) []float64 { return vec.ShiftTo(dst, xs, lag) }

// Diff returns xs[i] - xs[i-lag]. See [vec.Diff].
func Diff(xs []float64, lag int) []float64 { return vec.Diff(xs, lag) }

// DiffTo stores xs[i] - xs[i-lag] into dst; dst must not alias xs. See [vec.DiffTo].
func DiffTo(dst, xs []float64, lag int) []float64 { return vec.DiffTo(dst, xs, lag) }

// Rate returns the simple rate of change over n bars. See [vec.Rate].
func Rate(xs []float64, n int) []float64 { return vec.Rate(xs, n) }

// RateTo stores the simple rate of change over n bars into dst; dst must not alias
// xs. See [vec.RateTo].
func RateTo(dst, xs []float64, n int) []float64 { return vec.RateTo(dst, xs, n) }

// Rising returns a mask that is 1 where xs[i] > xs[i-n]. See [vec.Rising].
func Rising(xs []float64, n int) []uint8 { return vec.Rising(xs, n) }

// Falling returns a mask that is 1 where xs[i] < xs[i-n]. See [vec.Falling].
func Falling(xs []float64, n int) []uint8 { return vec.Falling(xs, n) }

// CrossOver returns a mask that is 1 where a crosses above b. See [vec.CrossOver].
func CrossOver(a, b []float64) []uint8 { return vec.CrossOver(a, b) }

// CrossUnder returns a mask that is 1 where a crosses below b. See [vec.CrossUnder].
func CrossUnder(a, b []float64) []uint8 { return vec.CrossUnder(a, b) }

// Cross returns +1 on a cross above, -1 on a cross below, and 0 elsewhere.
// See [vec.Cross].
func Cross(a, b []float64) []int8 { return vec.Cross(a, b) }

// BarsSince returns the number of bars since mask was last non-zero, or -1 before
// the first. See [vec.BarsSince].
func BarsSince(mask []uint8) []int { return vec.BarsSince(mask) }

// ValueWhen forward-fills the value of xs at the most recent non-zero cond.
// See [vec.ValueWhen].
func ValueWhen(cond []uint8, xs []float64) []float64 { return vec.ValueWhen(cond, xs) }

// HighestSince returns the running maximum since the last non-zero cond.
// See [vec.HighestSince].
func HighestSince(cond []uint8, xs []float64) []float64 { return vec.HighestSince(cond, xs) }

// LowestSince returns the running minimum since the last non-zero cond.
// See [vec.LowestSince].
func LowestSince(cond []uint8, xs []float64) []float64 { return vec.LowestSince(cond, xs) }
