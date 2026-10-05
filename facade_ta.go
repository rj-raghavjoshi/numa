package numa

import "github.com/rj-raghavjoshi/numa/ta"

// ---------------------------------------------------------------------------
// Facade forwarders for the ta package: technical indicators.
//
// Each is a one-line forwarder. See facade_math.go for why the forwarders are split
// per family, and facade_ta_test.go for the forwarding tests.
//
// Indicators have many parameters, several of which are ints or floats in a row, so
// this is the family where a forwarding mistake is easiest to make and hardest to see:
// swapping `fast` and `slow`, or `n` and `k`, still compiles and still returns a
// plausible series. The tests below therefore compare against ta with the same
// arguments rather than only checking shapes.
// ---------------------------------------------------------------------------

// FirstValid returns the index of the first non-NaN element of xs.
// See [ta.FirstValid].
func FirstValid(xs []float64) int { return ta.FirstValid(xs) }

// MedianPrice returns (high + low) / 2. See [ta.MedianPrice].
func MedianPrice(high, low []float64) []float64 { return ta.MedianPrice(high, low) }

// TypicalPrice returns (high + low + close) / 3. See [ta.TypicalPrice].
func TypicalPrice(high, low, close []float64) []float64 { return ta.TypicalPrice(high, low, close) }

// AveragePrice returns (open + high + low + close) / 4. See [ta.AveragePrice].
func AveragePrice(open, high, low, close []float64) []float64 {
	return ta.AveragePrice(open, high, low, close)
}

// TrueRange returns Wilder's true range. See [ta.TrueRange].
func TrueRange(high, low, close []float64) []float64 { return ta.TrueRange(high, low, close) }

// Change returns the n-bar change of xs. See [ta.Change].
func Change(xs []float64, n int) []float64 { return ta.Change(xs, n) }

// SMA returns the n-period simple moving average. See [ta.SMA].
func SMA(xs []float64, n int) []float64 { return ta.SMA(xs, n) }

// EMA returns the n-period exponential moving average. See [ta.EMA].
func EMA(xs []float64, n int) []float64 { return ta.EMA(xs, n) }

// RMA returns Wilder's smoothing of xs. See [ta.RMA].
func RMA(xs []float64, n int) []float64 { return ta.RMA(xs, n) }

// WMA returns the n-period weighted moving average. See [ta.WMA].
func WMA(xs []float64, n int) []float64 { return ta.WMA(xs, n) }

// DEMA returns the double exponential moving average. See [ta.DEMA].
func DEMA(xs []float64, n int) []float64 { return ta.DEMA(xs, n) }

// TEMA returns the triple exponential moving average. See [ta.TEMA].
func TEMA(xs []float64, n int) []float64 { return ta.TEMA(xs, n) }

// TRIMA returns the triangular moving average. See [ta.TRIMA].
func TRIMA(xs []float64, n int) []float64 { return ta.TRIMA(xs, n) }

// HMA returns the Hull moving average. See [ta.HMA].
func HMA(xs []float64, n int) []float64 { return ta.HMA(xs, n) }

// Highest returns the rolling maximum over n values. See [ta.Highest].
func Highest(xs []float64, n int) []float64 { return ta.Highest(xs, n) }

// Lowest returns the rolling minimum over n values. See [ta.Lowest].
func Lowest(xs []float64, n int) []float64 { return ta.Lowest(xs, n) }

// Donchian returns the Donchian channel. See [ta.Donchian].
func Donchian(high, low []float64, n int) (upper, middle, lower []float64) {
	return ta.Donchian(high, low, n)
}

// ATR returns the n-period Average True Range. See [ta.ATR].
func ATR(high, low, close []float64, n int) []float64 { return ta.ATR(high, low, close, n) }

// BollingerBands returns the Bollinger bands. See [ta.BollingerBands].
func BollingerBands(close []float64, n int, k float64) (upper, middle, lower []float64) {
	return ta.BollingerBands(close, n, k)
}

// BBPercentB returns the Bollinger %B. See [ta.BBPercentB].
func BBPercentB(close []float64, n int, k float64) []float64 { return ta.BBPercentB(close, n, k) }

// BBWidth returns the Bollinger band width. See [ta.BBWidth].
func BBWidth(close []float64, n int, k float64) []float64 { return ta.BBWidth(close, n, k) }

// ---------------------------------------------------------------------------
// Momentum indicators.
//
// These are the most argument-dense forwarders in the package -- MACD takes three
// periods in a row and Stochastic takes three smoothing parameters -- so the tests
// drive them with all-distinct values. A transposed pair still compiles and still
// returns a plausible oscillator, which is exactly why it needs a test.
// ---------------------------------------------------------------------------

// RSI returns Wilder's Relative Strength Index. See [ta.RSI].
func RSI(close []float64, n int) []float64 { return ta.RSI(close, n) }

// MACD returns the MACD line, signal line and histogram. See [ta.MACD].
func MACD(close []float64, fast, slow, signal int) (macd, signalLine, hist []float64) {
	return ta.MACD(close, fast, slow, signal)
}

// Stochastic returns the stochastic oscillator %K and %D. See [ta.Stochastic].
func Stochastic(high, low, close []float64, kPeriod, kSmooth, dSmooth int) (k, d []float64) {
	return ta.Stochastic(high, low, close, kPeriod, kSmooth, dSmooth)
}

// StochRSI returns the stochastic oscillator applied to RSI. See [ta.StochRSI].
func StochRSI(close []float64, rsiPeriod, stochPeriod, kSmooth, dSmooth int) (k, d []float64) {
	return ta.StochRSI(close, rsiPeriod, stochPeriod, kSmooth, dSmooth)
}

// WilliamsPercentR returns the Williams %R. See [ta.WilliamsPercentR].
func WilliamsPercentR(high, low, close []float64, n int) []float64 {
	return ta.WilliamsPercentR(high, low, close, n)
}

// CCI returns the Commodity Channel Index. See [ta.CCI].
func CCI(high, low, close []float64, n int) []float64 { return ta.CCI(high, low, close, n) }

// CMO returns the Chande Momentum Oscillator. See [ta.CMO].
func CMO(close []float64, n int) []float64 { return ta.CMO(close, n) }

// ROC returns the n-period rate of change as a percentage. See [ta.ROC].
func ROC(close []float64, n int) []float64 { return ta.ROC(close, n) }

// Momentum returns the n-period price difference. See [ta.Momentum].
func Momentum(close []float64, n int) []float64 { return ta.Momentum(close, n) }

// ---------------------------------------------------------------------------
// Volume indicators.
//
// Every one of these takes four columns in the order (high, low, close, volume) or
// (close, volume). A forwarder that puts close before low, or volume before close,
// still compiles and still returns a plausible line, so the tests below compare each
// against ta with identical arguments and use columns whose values cannot be confused
// for each other.
// ---------------------------------------------------------------------------

// VWAP returns the cumulative volume-weighted average price. See [ta.VWAP].
func VWAP(high, low, close, volume []float64) []float64 {
	return ta.VWAP(high, low, close, volume)
}

// VWMA returns the n-period volume-weighted moving average. See [ta.VWMA].
func VWMA(close, volume []float64, n int) []float64 { return ta.VWMA(close, volume, n) }

// OBV returns the On Balance Volume line. See [ta.OBV].
func OBV(close, volume []float64) []float64 { return ta.OBV(close, volume) }

// AccumulationDistribution returns the A/D line. See [ta.AccumulationDistribution].
func AccumulationDistribution(high, low, close, volume []float64) []float64 {
	return ta.AccumulationDistribution(high, low, close, volume)
}

// ChaikinMoneyFlow returns the n-period Chaikin Money Flow. See [ta.ChaikinMoneyFlow].
func ChaikinMoneyFlow(high, low, close, volume []float64, n int) []float64 {
	return ta.ChaikinMoneyFlow(high, low, close, volume, n)
}

// ChaikinOscillator returns the Chaikin oscillator. See [ta.ChaikinOscillator].
func ChaikinOscillator(high, low, close, volume []float64, fast, slow int) []float64 {
	return ta.ChaikinOscillator(high, low, close, volume, fast, slow)
}

// MoneyFlowIndex returns the n-period Money Flow Index. See [ta.MoneyFlowIndex].
func MoneyFlowIndex(high, low, close, volume []float64, n int) []float64 {
	return ta.MoneyFlowIndex(high, low, close, volume, n)
}

// PriceVolumeTrend returns the Price Volume Trend line. See [ta.PriceVolumeTrend].
func PriceVolumeTrend(close, volume []float64) []float64 { return ta.PriceVolumeTrend(close, volume) }

// NetVolume returns volume signed by the direction of the close. See [ta.NetVolume].
func NetVolume(close, volume []float64) []float64 { return ta.NetVolume(close, volume) }

// VolumeOscillator returns the volume oscillator. See [ta.VolumeOscillator].
func VolumeOscillator(volume []float64, fast, slow int) []float64 {
	return ta.VolumeOscillator(volume, fast, slow)
}

// ForceIndex returns Elder's Force Index. See [ta.ForceIndex].
func ForceIndex(close, volume []float64, n int) []float64 { return ta.ForceIndex(close, volume, n) }

// ---------------------------------------------------------------------------
// Trend indicators.
//
// These take a mix of int periods and float factors. ParabolicSAR is the worst case:
// three floats in a row, two of which are typically 0.02 and 0.2, so a transposition
// changes the acceleration schedule subtly rather than obviously.
// ---------------------------------------------------------------------------

// DirectionalMovement returns ADX, +DI and -DI. See [ta.DirectionalMovement].
func DirectionalMovement(high, low, close []float64, n int) (adx, plusDI, minusDI []float64) {
	return ta.DirectionalMovement(high, low, close, n)
}

// Aroon returns the Aroon up line, down line and oscillator. See [ta.Aroon].
func Aroon(high, low []float64, n int) (up, down, osc []float64) {
	return ta.Aroon(high, low, n)
}

// Vortex returns the Vortex Indicator lines VI+ and VI-. See [ta.Vortex].
func Vortex(high, low, close []float64, n int) (viPlus, viMinus []float64) {
	return ta.Vortex(high, low, close, n)
}

// Choppiness returns the Choppiness Index. See [ta.Choppiness].
func Choppiness(high, low, close []float64, n int) []float64 {
	return ta.Choppiness(high, low, close, n)
}

// KeltnerChannels returns the Keltner channel. See [ta.KeltnerChannels].
func KeltnerChannels(high, low, close []float64, n, atrN int, mult float64) (upper, middle, lower []float64) {
	return ta.KeltnerChannels(high, low, close, n, atrN, mult)
}

// SuperTrend returns the SuperTrend line and its direction. See [ta.SuperTrend].
func SuperTrend(high, low, close []float64, atrN int, mult float64) (line, direction []float64) {
	return ta.SuperTrend(high, low, close, atrN, mult)
}

// ParabolicSAR returns Wilder's Parabolic SAR and its direction. See [ta.ParabolicSAR].
func ParabolicSAR(high, low []float64, start, increment, max float64) (sar, direction []float64) {
	return ta.ParabolicSAR(high, low, start, increment, max)
}

// ---------------------------------------------------------------------------
// Linear-regression and composed-oscillator indicators.
// ---------------------------------------------------------------------------

// LinearRegression returns the least-squares regression value at the current bar.
// See [ta.LinearRegression].
func LinearRegression(xs []float64, n int) []float64 { return ta.LinearRegression(xs, n) }

// LinearRegressionSlope returns the regression slope. See [ta.LinearRegressionSlope].
func LinearRegressionSlope(xs []float64, n int) []float64 { return ta.LinearRegressionSlope(xs, n) }

// LinearRegressionIntercept returns the regression value at the window's first bar.
// See [ta.LinearRegressionIntercept].
func LinearRegressionIntercept(xs []float64, n int) []float64 {
	return ta.LinearRegressionIntercept(xs, n)
}

// R2 returns the coefficient of determination of the regression. See [ta.R2].
func R2(xs []float64, n int) []float64 { return ta.R2(xs, n) }

// StandardError returns the standard error of the regression estimate.
// See [ta.StandardError].
func StandardError(xs []float64, n int) []float64 { return ta.StandardError(xs, n) }

// StandardErrorBands returns the regression line with a k-standard-error band.
// See [ta.StandardErrorBands].
func StandardErrorBands(xs []float64, n int, k float64) (upper, middle, lower []float64) {
	return ta.StandardErrorBands(xs, n, k)
}

// AwesomeOscillator returns the Awesome Oscillator. See [ta.AwesomeOscillator].
func AwesomeOscillator(high, low []float64, fast, slow int) []float64 {
	return ta.AwesomeOscillator(high, low, fast, slow)
}

// AcceleratorOscillator returns the Accelerator Oscillator.
// See [ta.AcceleratorOscillator].
func AcceleratorOscillator(high, low []float64, fast, slow, smooth int) []float64 {
	return ta.AcceleratorOscillator(high, low, fast, slow, smooth)
}

// DetrendedPriceOscillator returns the Detrended Price Oscillator.
// See [ta.DetrendedPriceOscillator].
func DetrendedPriceOscillator(close []float64, n int) []float64 {
	return ta.DetrendedPriceOscillator(close, n)
}

// TRIX returns the triple-smoothed rate of change. See [ta.TRIX].
func TRIX(close []float64, n int) []float64 { return ta.TRIX(close, n) }

// UltimateOscillator returns the Ultimate Oscillator. See [ta.UltimateOscillator].
func UltimateOscillator(high, low, close []float64, short, mid, long int) []float64 {
	return ta.UltimateOscillator(high, low, close, short, mid, long)
}

// FisherTransform returns Ehlers' Fisher Transform. See [ta.FisherTransform].
func FisherTransform(high, low []float64, n int) []float64 {
	return ta.FisherTransform(high, low, n)
}

// MassIndex returns the Mass Index. See [ta.MassIndex].
func MassIndex(high, low []float64, emaPeriod, sumPeriod int) []float64 {
	return ta.MassIndex(high, low, emaPeriod, sumPeriod)
}

// ---------------------------------------------------------------------------
// Correlation, realized volatility, and the long-period oscillator family.
// ---------------------------------------------------------------------------

// CorrelationCoefficient returns the rolling Pearson correlation. See [ta.CorrelationCoefficient].
func CorrelationCoefficient(xs, ys []float64, n int) []float64 {
	return ta.CorrelationCoefficient(xs, ys, n)
}

// CorrelationLog returns the rolling correlation of log returns. See [ta.CorrelationLog].
func CorrelationLog(xs, ys []float64, n int) []float64 { return ta.CorrelationLog(xs, ys, n) }

// RankCorrelation returns the rolling Spearman rank correlation. See [ta.RankCorrelation].
func RankCorrelation(xs, ys []float64, n int) []float64 { return ta.RankCorrelation(xs, ys, n) }

// HistoricalVolatility returns the annualized close-to-close volatility.
// See [ta.HistoricalVolatility].
func HistoricalVolatility(close []float64, n int, annual float64) []float64 {
	return ta.HistoricalVolatility(close, n, annual)
}

// VolatilityOHLC returns the annualized Garman-Klass volatility. See [ta.VolatilityOHLC].
func VolatilityOHLC(open, high, low, close []float64, n int, annual float64) []float64 {
	return ta.VolatilityOHLC(open, high, low, close, n, annual)
}

// CoppockCurve returns the Coppock Curve. See [ta.CoppockCurve].
func CoppockCurve(close []float64, rocLong, rocShort, wmaPeriod int) []float64 {
	return ta.CoppockCurve(close, rocLong, rocShort, wmaPeriod)
}

// KnowSureThing returns the Know Sure Thing oscillator and its signal line.
// See [ta.KnowSureThing].
func KnowSureThing(close []float64, roc, sma [4]int, signalLength int) (kst, signal []float64) {
	return ta.KnowSureThing(close, roc, sma, signalLength)
}

// TrueStrengthIndex returns the double-smoothed momentum ratio. See [ta.TrueStrengthIndex].
func TrueStrengthIndex(close []float64, long, short int) []float64 {
	return ta.TrueStrengthIndex(close, long, short)
}

// WilliamsAlligator returns the three shifted smoothed averages. See [ta.WilliamsAlligator].
func WilliamsAlligator(high, low []float64, jawPeriod, jawShift, teethPeriod, teethShift, lipsPeriod, lipsShift int) (jaw, teeth, lips []float64) {
	return ta.WilliamsAlligator(high, low, jawPeriod, jawShift, teethPeriod, teethShift, lipsPeriod, lipsShift)
}

// WilliamsFractal returns the up and down fractal masks. See [ta.WilliamsFractal].
func WilliamsFractal(high, low []float64, n int) (up, down []uint8) {
	return ta.WilliamsFractal(high, low, n)
}

// ---------------------------------------------------------------------------
// Ichimoku, pivot points, GMMA and ZigZag.
//
// PivotMethod is re-exported as a type alias, which costs nothing and means a caller can
// use numa.PivotClassic instead of importing ta for the constant alone.
// ---------------------------------------------------------------------------

// PivotMethod selects a pivot-point formula. See [ta.PivotMethod].
type PivotMethod = ta.PivotMethod

const (
	// PivotClassic is the original five-point pivot method.
	PivotClassic = ta.PivotClassic
	// PivotFibonacci places levels at the Fibonacci retracement ratios.
	PivotFibonacci = ta.PivotFibonacci
	// PivotCamarilla places levels relative to the close.
	PivotCamarilla = ta.PivotCamarilla
	// PivotWoodie weights the close twice in the pivot.
	PivotWoodie = ta.PivotWoodie
)

// Ichimoku returns the five Ichimoku lines. See [ta.Ichimoku].
func Ichimoku(high, low, close []float64, conversionPeriod, basePeriod, spanBPeriod, displacement int) (conversion, base, spanA, spanB, lagging []float64) {
	return ta.Ichimoku(high, low, close, conversionPeriod, basePeriod, spanBPeriod, displacement)
}

// PivotLevels returns the pivot and support/resistance levels for one period.
// See [ta.PivotLevels].
func PivotLevels(high, low, close float64, method PivotMethod) (pivot, r1, r2, r3, s1, s2, s3 float64) {
	return ta.PivotLevels(high, low, close, method)
}

// PivotPoints returns the pivot levels as series from the previous bar.
// See [ta.PivotPoints].
func PivotPoints(high, low, close []float64, method PivotMethod) (pivot, r1, r2, r3, s1, s2, s3 []float64) {
	return ta.PivotPoints(high, low, close, method)
}

// GMMA returns the Guppy Multiple Moving Average groups. See [ta.GMMA].
func GMMA(close []float64) (short, long [6][]float64) { return ta.GMMA(close) }

// ZigZag returns the confirmed pivot sequence. See [ta.ZigZag].
func ZigZag(high, low []float64, deviation float64) (pivot []float64, kind []int8) {
	return ta.ZigZag(high, low, deviation)
}

// ---------------------------------------------------------------------------
// Extended moving averages, envelopes and the price oscillator.
// ---------------------------------------------------------------------------

// ALMA returns the Arnaud Legoux Moving Average. See [ta.ALMA].
func ALMA(xs []float64, n int, offset, sigma float64) []float64 {
	return ta.ALMA(xs, n, offset, sigma)
}

// SWMA returns the Symmetrically Weighted Moving Average. See [ta.SWMA].
func SWMA(xs []float64) []float64 { return ta.SWMA(xs) }

// KAMA returns Kaufman's Adaptive Moving Average. See [ta.KAMA].
func KAMA(xs []float64, n, fast, slow int) []float64 { return ta.KAMA(xs, n, fast, slow) }

// VIDYA returns Chande's Variable Index Dynamic Average. See [ta.VIDYA].
func VIDYA(xs []float64, n, cmoPeriod int) []float64 { return ta.VIDYA(xs, n, cmoPeriod) }

// ZLEMA returns the Zero-Lag Exponential Moving Average. See [ta.ZLEMA].
func ZLEMA(xs []float64, n int) []float64 { return ta.ZLEMA(xs, n) }

// T3 returns Tillson's T3 moving average. See [ta.T3].
func T3(xs []float64, n int, v float64) []float64 { return ta.T3(xs, n, v) }

// McGinleyDynamic returns the McGinley Dynamic indicator. See [ta.McGinleyDynamic].
func McGinleyDynamic(xs []float64, n int) []float64 { return ta.McGinleyDynamic(xs, n) }

// MovingAverageEnvelope returns a percentage envelope around a moving average.
// See [ta.MovingAverageEnvelope].
func MovingAverageEnvelope(xs []float64, n int, percent float64) (upper, middle, lower []float64) {
	return ta.MovingAverageEnvelope(xs, n, percent)
}

// MovingAverageChannel returns the channel of high and low moving averages.
// See [ta.MovingAverageChannel].
func MovingAverageChannel(high, low []float64, n int) (upper, lower []float64) {
	return ta.MovingAverageChannel(high, low, n)
}

// PriceOscillator returns the percentage difference of two moving averages.
// See [ta.PriceOscillator].
func PriceOscillator(xs []float64, fast, slow int) []float64 {
	return ta.PriceOscillator(xs, fast, slow)
}

// WeightedClose returns (high + low + 2*close) / 4. See [ta.WeightedClose].
func WeightedClose(high, low, close []float64) []float64 {
	return ta.WeightedClose(high, low, close)
}

// ---------------------------------------------------------------------------
// Second-tier momentum, volume, stop and range indicators.
// ---------------------------------------------------------------------------

// PercentRank returns the percentage of trailing values below the current one.
// See [ta.PercentRank].
func PercentRank(xs []float64, n int) []float64 { return ta.PercentRank(xs, n) }

// StochasticMomentumIndex returns Blau's SMI and its signal. See [ta.StochasticMomentumIndex].
func StochasticMomentumIndex(high, low, close []float64, n, smoothK, smoothD int) (smi, signal []float64) {
	return ta.StochasticMomentumIndex(high, low, close, n, smoothK, smoothD)
}

// RelativeVolatilityIndex returns Dorsey's RVI. See [ta.RelativeVolatilityIndex].
func RelativeVolatilityIndex(close []float64, n int) []float64 {
	return ta.RelativeVolatilityIndex(close, n)
}

// ConnorsRSI returns the Connors Research composite. See [ta.ConnorsRSI].
func ConnorsRSI(close []float64, rsiPeriod, streakPeriod, rankPeriod int) []float64 {
	return ta.ConnorsRSI(close, rsiPeriod, streakPeriod, rankPeriod)
}

// ElderRay returns the bull and bear power. See [ta.ElderRay].
func ElderRay(high, low, close []float64, n int) (bullPower, bearPower []float64) {
	return ta.ElderRay(high, low, close, n)
}

// BalanceOfPower returns the smoothed balance of power. See [ta.BalanceOfPower].
func BalanceOfPower(open, high, low, close []float64, n int) []float64 {
	return ta.BalanceOfPower(open, high, low, close, n)
}

// EaseOfMovement returns the ease of movement. See [ta.EaseOfMovement].
func EaseOfMovement(high, low, volume []float64, n int) []float64 {
	return ta.EaseOfMovement(high, low, volume, n)
}

// KlingerOscillator returns the Klinger volume oscillator and its signal.
// See [ta.KlingerOscillator].
func KlingerOscillator(high, low, close, volume []float64, fast, slow, signalPeriod int) (kvo, signal []float64) {
	return ta.KlingerOscillator(high, low, close, volume, fast, slow, signalPeriod)
}

// ChaikinVolatility returns the rate of change of a smoothed range.
// See [ta.ChaikinVolatility].
func ChaikinVolatility(high, low []float64, n int) []float64 {
	return ta.ChaikinVolatility(high, low, n)
}

// ChandeKrollStop returns the two trailing stops. See [ta.ChandeKrollStop].
func ChandeKrollStop(high, low, close []float64, n int, x float64, q int) (stopLong, stopShort []float64) {
	return ta.ChandeKrollStop(high, low, close, n, x, q)
}

// FiftyTwoWeekHighLow returns the rolling extreme over bars periods.
// See [ta.FiftyTwoWeekHighLow].
func FiftyTwoWeekHighLow(high, low []float64, bars int) (highs, lows []float64) {
	return ta.FiftyTwoWeekHighLow(high, low, bars)
}

// SwingIndex returns the one-bar Swing Index. See [ta.SwingIndex].
func SwingIndex(open, high, low, close []float64, limitMove float64) []float64 {
	return ta.SwingIndex(open, high, low, close, limitMove)
}

// AccumulativeSwingIndex returns the running total of the Swing Index.
// See [ta.AccumulativeSwingIndex].
func AccumulativeSwingIndex(open, high, low, close []float64, limitMove float64) []float64 {
	return ta.AccumulativeSwingIndex(open, high, low, close, limitMove)
}

// ---------------------------------------------------------------------------
// Catalogue completion: spread statistics, vigour, chop, crosses, Hamming.
// ---------------------------------------------------------------------------

// StandardDeviation returns the trailing population standard deviation.
// See [ta.StandardDeviation].
func StandardDeviation(xs []float64, n int) []float64 { return ta.StandardDeviation(xs, n) }

// Ratio returns the elementwise ratio a/b. See [ta.Ratio].
func Ratio(a, b []float64) []float64 { return ta.Ratio(a, b) }

// Spread returns the elementwise difference a-b. See [ta.Spread].
func Spread(a, b []float64) []float64 { return ta.Spread(a, b) }

// RelativeVigorIndex returns Dorsey's RVI and its signal. See [ta.RelativeVigorIndex].
func RelativeVigorIndex(open, high, low, close []float64, n, signalPeriod int) (rvi, signal []float64) {
	return ta.RelativeVigorIndex(open, high, low, close, n, signalPeriod)
}

// ChopZone returns Chande's Chop Zone. See [ta.ChopZone].
func ChopZone(high, low, close []float64, n int) []float64 {
	return ta.ChopZone(high, low, close, n)
}

// HammingMA returns the Hamming-window weighted moving average. See [ta.HammingMA].
func HammingMA(xs []float64, n int) []float64 { return ta.HammingMA(xs, n) }

// SMIErgodic returns Blau's ergodic indicator, its signal and their difference.
// See [ta.SMIErgodic].
func SMIErgodic(close []float64, long, short, signalPeriod int) (ergodic, signal, oscillator []float64) {
	return ta.SMIErgodic(close, long, short, signalPeriod)
}

// AdvanceDecline returns the cumulative advance-decline line. See [ta.AdvanceDecline].
func AdvanceDecline(advances, declines []float64) []float64 {
	return ta.AdvanceDecline(advances, declines)
}

// MACross returns two simple moving averages and their crossing signal.
// See [ta.MACross].
func MACross(close []float64, fast, slow int) (fastLine, slowLine []float64, cross []int8) {
	return ta.MACross(close, fast, slow)
}

// EMACross returns two exponential moving averages and their crossing signal.
// See [ta.EMACross].
func EMACross(close []float64, fast, slow int) (fastLine, slowLine []float64, cross []int8) {
	return ta.EMACross(close, fast, slow)
}

// EMASMACross returns an exponential fast line, a simple slow line and their crossing signal.
// See [ta.EMASMACross].
func EMASMACross(close []float64, fast, slow int) (fastLine, slowLine []float64, cross []int8) {
	return ta.EMASMACross(close, fast, slow)
}
