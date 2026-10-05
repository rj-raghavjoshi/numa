// Package ta implements technical indicators over plain float64 slices.
//
// It is the indicator layer of the numa compute engine: the other packages provide
// numerical primitives, and this one assembles them into the indicators a strategy or
// a charting layer asks for by name.
//
// # Conventions
//
// Every function here follows the same rules, so a caller does not have to read each
// doc comment to know what to expect.
//
//  1. **Plain slices in, plain slices out.** No bar or frame type exists. A function
//     that needs open, high, low, close and volume takes them as separate arguments:
//
//     ta.ATR(high, low, close, 14)
//
//     This keeps the package free of any container decision and lets a caller hold
//     its data in whatever layout it already has.
//
//  2. **Output has the same length as the input.** There is no "trim the warm-up"
//     variant and no offset index to track. Bars before the indicator is defined
//     carry NaN, which is the single marker for "not available".
//
//  3. **NaN wins.** A NaN anywhere in the relevant window makes that output NaN.
//     Nothing is silently skipped, because a NaNs almost always indicates an
//     upstream error and a plausible-looking number computed around it is worse than
//     a visibly missing one. This matches the vec scans' policy.
//
//  4. **Causal.** Output at index i depends only on inputs at indices <= i. An
//     indicator that needs a later value to compute an earlier one (centred moving
//     averages, for example) either does not exist here or reports NaN for the
//     positions it cannot fill. This is what makes the package usable for a
//     backtest: there is no look-ahead to guard against at the call site.
//
//  5. **Multi-output indicators return their series together.** MACD returns the
//     line, the signal and the histogram rather than three separate calls, so the
//     shared work is done once:
//
//     macd, signal, hist := ta.MACD(close, 12, 26, 9)
//
//  6. **Periods are normalized, not trusted blindly.** n < 1 panics, because it is a
//     programming error. n larger than the input is not an error: the result is all
//     NaN, which is the honest answer.
//
// # Warm-up lengths
//
// The number of leading NaNs is documented per function, and [FirstValid] reports
// where a computed series first becomes usable:
//
//	first := ta.FirstValid(ta.SMA(close, 20)) // 19
//
// Most indicators here warm up in exactly `n-1` bars for a single period `n`, but a
// few (the SEEDED ones, and any indicator composed from others) differ, which is why
// the helper exists rather than a table.
//
// # Relationship to series
//
// [github.com/rj-raghavjoshi/numa/series] holds the same mathematics as stateful
// rollers for incremental use. The batch functions here are the ones to call when the
// whole series is available; the rollers are the ones to call when it is not. For a
// full-history pass the batch form is usually faster, and it is what these functions
// use internally where one exists.
//
// # Accuracy
//
// The same two rules as the rest of the engine apply: results may differ in the last
// bits from a naive loop, and are not guaranteed bit-identical across architectures.
// Where an indicator has an unstable and a stable formulation, the stable one is used
// and the choice is documented on the function -- see ta.BollingerBands.
package ta
