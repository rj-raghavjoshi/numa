# Coverage against TradingView's indicator list

`numa/ta` is audited against TradingView's published [Indicators List][list] rather than against an
impression of it. The list contains **107 entries**; **100 are implemented**. The seven that are not
are named below with the reason, because "not implemented" and "not implementable here" are different
statements and only one of them is a gap.

[list]: https://www.tradingview.com/charting-library-docs/latest/ui_elements/indicators/Indicators-List/

## Implemented

| TradingView entry | `ta` function |
|---|---|
| 52 Week High/Low | `FiftyTwoWeekHighLow` |
| Accelerator Oscillator | `AcceleratorOscillator` |
| Accumulation/Distribution | `AccumulationDistribution` |
| Accumulative Swing Index | `AccumulativeSwingIndex`, `SwingIndex` |
| Advance/Decline | `AdvanceDecline` |
| Arnaud Legoux Moving Average | `ALMA` |
| Aroon | `Aroon` |
| Average Directional Index | `DirectionalMovement` |
| Average Price | `AveragePrice` |
| Average True Range | `ATR` |
| Awesome Oscillator | `AwesomeOscillator` |
| Balance of Power | `BalanceOfPower` |
| Bollinger Bands | `BollingerBands` |
| Bollinger Bands %B | `BBPercentB` |
| Bollinger Bands Width | `BBWidth` |
| Chaikin Money Flow | `ChaikinMoneyFlow` |
| Chaikin Oscillator | `ChaikinOscillator` |
| Chaikin Volatility | `ChaikinVolatility` |
| Chande Kroll Stop | `ChandeKrollStop` |
| Chande Momentum Oscillator | `CMO` |
| Chop Zone | `ChopZone` |
| Choppiness Index | `Choppiness` |
| Commodity Channel Index | `CCI` |
| Connors RSI | `ConnorsRSI` |
| Coppock Curve | `CoppockCurve` |
| Correlation Coefficient | `CorrelationCoefficient` |
| Correlation - Log | `CorrelationLog` |
| Detrended Price Oscillator | `DetrendedPriceOscillator` |
| Directional Movement | `DirectionalMovement` |
| Donchian Channels | `Donchian` |
| Double EMA | `DEMA` |
| Ease of Movement | `EaseOfMovement` |
| Elder's Force Index | `ForceIndex` |
| EMA Cross | `EMACross` |
| Envelopes | `MovingAverageEnvelope` |
| Fisher Transform | `FisherTransform` |
| Guppy Multiple Moving Average | `GMMA` |
| Historical Volatility | `HistoricalVolatility` |
| Hull Moving Average | `HMA` |
| Ichimoku Cloud | `Ichimoku` |
| Keltner Channels | `KeltnerChannels` |
| Klinger Oscillator | `KlingerOscillator` |
| Know Sure Thing | `KnowSureThing` |
| Least Squares Moving Average | `LinearRegression` |
| Linear Regression Curve | `LinearRegression` |
| Linear Regression Slope | `LinearRegressionSlope` |
| MA Cross | `MACross` |
| MA with EMA Cross | `EMASMACross` |
| Mass Index | `MassIndex` |
| McGinley Dynamic | `McGinleyDynamic` |
| Median Price | `MedianPrice` |
| Momentum | `Momentum` |
| Money Flow Index | `MoneyFlowIndex` |
| Moving Average | `SMA` |
| Moving Average Channel | `MovingAverageChannel` |
| MACD | `MACD` |
| Moving Average Exponential | `EMA` |
| Moving Average Weighted | `WMA` |
| Moving Average Double | `DEMA` |
| Moving Average Triple | `TEMA` |
| Moving Average Adaptive | `KAMA` |
| Moving Average Hamming | `HammingMA` |
| Moving Average Multiple | `GMMA` |
| Net Volume | `NetVolume` |
| On Balance Volume | `OBV` |
| Parabolic SAR | `ParabolicSAR` |
| Pivot Points Standard | `PivotPoints`, `PivotLevels` |
| Price Channel | `Donchian` |
| Price Oscillator | `PriceOscillator` |
| Price Volume Trend | `PriceVolumeTrend` |
| Rank Correlation Index | `RankCorrelation` |
| Rate Of Change | `ROC` |
| Ratio | `Ratio` |
| Relative Strength Index | `RSI` |
| Relative Vigor Index | `RelativeVigorIndex` |
| Relative Volatility Index | `RelativeVolatilityIndex` |
| Standard Error | `StandardError` |
| Standard Error Bands | `StandardErrorBands` |
| SMI Ergodic Indicator/Oscillator | `SMIErgodic` |
| Smoothed Moving Average | `RMA` |
| Standard Deviation | `StandardDeviation` |
| Stochastic | `Stochastic` |
| Stochastic RSI | `StochRSI` |
| SuperTrend | `SuperTrend` |
| Spread | `Spread` |
| TRIX | `TRIX` |
| Triple EMA | `TEMA` |
| True Strength Indicator | `TrueStrengthIndex` |
| Typical Price | `TypicalPrice` |
| Ultimate Oscillator | `UltimateOscillator` |
| Volatility Close-to-Close | `HistoricalVolatility` |
| Volatility O-H-L-C | `VolatilityOHLC` |
| VWAP | `VWAP` |
| VWMA | `VWMA` |
| Volume Oscillator | `VolumeOscillator` |
| Vortex Indicator | `Vortex` |
| Williams %R | `WilliamsPercentR` |
| Williams Alligator | `WilliamsAlligator` |
| Williams Fractal | `WilliamsFractal` |
| Zig Zag | `ZigZag` |

Beyond the list, `ta` also provides the primitives the entries are built from and a few things the
library computes internally but does not name: `Highest`, `Lowest`, `TrueRange`, `Change`, `PercentRank`,
`SWMA`, `ZLEMA`, `VIDYA`, `T3`, `TRIMA`, `R2`, `StochasticMomentumIndex`, `ElderRay`,
`WeightedClose`, `StandardErrorBands` and `CorrelationCoefficient`'s variants.

## Not implemented, and why

| TradingView entry | reason |
|---|---|
| Volume | It is the input series itself. There is no computation to provide, and inventing a function that returns its argument would be worse than not having one. |
| Moving Average Multiple | It is the Guppy construction under another name; `GMMA` covers it. |
| Majority Rule | Its published description is prose rather than a formula, and the two readings of it differ on how the majority threshold is computed. An implementation would be a guess presented as the indicator. |
| Trend Strength Index | No authoritative formula was located. The name collides with several unrelated measures, and guessing which one TradingView means would produce a wrong answer under a confident name. |
| Volatility Zero Trend Close-to-Close | A variant of close-to-close volatility with the trend removed. The list gives no formula and the published variants detrend differently. |
| Volatility Index | Vague in the same way, and it is usually computed from option prices rather than from a price series at all. |
| Volume Profile Fixed Range | Needs volume bucketed by *price level*, not by bar. That is a different data shape from everything else here — a histogram over a price axis — and it belongs to whatever owns the price-level universe. |
| Volume Profile Visible Range | Same as above, over a caller-chosen range. |

**The five formula-shaped omissions are deliberate rather than pending.** Each is an indicator whose
formula is either unpublished or published in mutually incompatible forms, and writing one of those
variants would produce a number that looks authoritative and is a guess. That is a worse outcome than
an absent function, because an absent function is visible.

If a formula is established for any of them — a primary source rather than a secondary description —
it can be implemented against the same conventions as everything else: a naive reference in a test, a
hand-computed case, boundary lengths, and a benchmark recorded in [benchmarks.md](benchmarks.md).
