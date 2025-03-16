package model

import "math"

// General Strategy Parameters
func BoundWindowSize(v int) int {
	return int(math.Max(50, math.Min(500, float64(v)))) // Default: 200
}

func BoundWindowSizeFloat64(v float64) float64 {
	return math.Max(50, math.Min(500, v))
}

func BoundCandles(v int) int {
	return int(math.Max(1, math.Min(50, float64(v)))) // Default: 5
}

func BoundCandlesFloat64(v float64) float64 {
	return math.Max(1, math.Min(50, v))
}

func BoundTakeProfit(v float64) float64 {
	return math.Max(Commission()*2*Leverage(), math.Min(0.05*Leverage(), v)) // Default: 0.008*Leverage
}

func BoundStopLoss(v float64) float64 {
	return math.Max(0.001*Leverage(), math.Min(0.01*Leverage(), v)) // Default: 0.002*Leverage
}

// Moving Averages
func BoundShortMovingAverageLength(v int) int {
	return int(math.Max(10, math.Min(100, float64(v)))) // Default: 50
}

func BoundShortMovingAverageLengthFloat64(v float64) float64 {
	return math.Max(10, math.Min(100, v))
}

func BoundLongMovingAverageLength(v int) int {
	return int(math.Max(50, math.Min(400, float64(v)))) // Default: 200
}

func BoundLongMovingAverageLengthFloat64(v float64) float64 {
	return math.Max(50, math.Min(400, v))
}

// RSI (Relative Strength Index)
func BoundLongRSILength(v int) int {
	return int(math.Max(10, math.Min(50, float64(v)))) // Default: 14
}

func BoundLongRSILengthFloat64(v float64) float64 {
	return math.Max(10, math.Min(50, v))
}

func BoundShortRSILength(v int) int {
	return int(math.Max(2, math.Min(14, float64(v)))) // Default: 5
}

func BoundShortRSILengthFloat64(v float64) float64 {
	return math.Max(2, math.Min(14, v))
}

func BoundRSIUpperBound(v float64) float64 {
	return math.Max(40, math.Min(80, v)) // Default: 50
}

func BoundRSILowerBound(v float64) float64 {
	return math.Max(20, math.Min(60, v)) // Default: 50
}

func BoundRSISlope(v int) int {
	return int(math.Max(1, math.Min(20, float64(v)))) // Default: 3
}

func BoundRSISlopeFloat64(v float64) float64 {
	return math.Max(1, math.Min(20, v))
}

// MACD (Moving Average Convergence Divergence)
func BoundShortMACDWindowLength(v int) int {
	return int(math.Max(5, math.Min(20, float64(v)))) // Default: 12
}

func BoundShortMACDWindowLengthFloat64(v float64) float64 {
	return math.Max(5, math.Min(20, v))
}

func BoundLongMACDWindowLength(v int) int {
	return int(math.Max(20, math.Min(50, float64(v)))) // Default: 26
}

func BoundLongMACDWindowLengthFloat64(v float64) float64 {
	return math.Max(20, math.Min(50, v))
}

func BoundMACDSignalWindow(v int) int {
	return int(math.Max(5, math.Min(15, float64(v)))) // Default: 9
}

func BoundMACDSignalWindowFloat64(v float64) float64 {
	return math.Max(5, math.Min(15, v))
}

// Fast MACD
func BoundFastShortMACDWindowLength(v int) int {
	return int(math.Max(3, math.Min(10, float64(v)))) // Default: 5
}

func BoundFastShortMACDWindowLengthFloat64(v float64) float64 {
	return math.Max(3, math.Min(10, v))
}

func BoundFastLongMACDWindowLength(v int) int {
	return int(math.Max(20, math.Min(50, float64(v)))) // Default: 35
}

func BoundFastLongMACDWindowLengthFloat64(v float64) float64 {
	return math.Max(20, math.Min(50, v))
}

func BoundFastMACDSignalWindow(v int) int {
	return int(math.Max(3, math.Min(12, float64(v)))) // Default: 5
}

func BoundFastMACDSignalWindowFloat64(v float64) float64 {
	return math.Max(3, math.Min(12, v))
}

// Bollinger Bands
func BoundBollingerBandsWindow(v int) int {
	return int(math.Max(10, math.Min(50, float64(v)))) // Default: 20
}

func BoundBollingerBandsWindowFloat64(v float64) float64 {
	return math.Max(10, math.Min(50, v))
}

func BoundBollingerBandsMultiplier(v float64) float64 {
	return math.Max(1.5, math.Min(3.5, v)) // Default: 2.0
}

// Stochastic Oscillator
func BoundStochasticOscillatorWindow(v int) int {
	return int(math.Max(5, math.Min(30, float64(v)))) // Default: 14
}

func BoundStochasticOscillatorWindowFloat64(v float64) float64 {
	return math.Max(5, math.Min(30, v))
}

// ATR (Average True Range)
func BoundSlowATRPeriod(v int) int {
	return int(math.Max(10, math.Min(50, float64(v)))) // Default: 14
}

func BoundSlowATRPeriodFloat64(v float64) float64 {
	return math.Max(10, math.Min(50, v))
}

func BoundFastATRPeriod(v int) int {
	return int(math.Max(10, math.Min(30, float64(v)))) // Default: 20
}

func BoundFastATRPeriodFloat64(v float64) float64 {
	return math.Max(10, math.Min(30, v))
}

// Volume-Based Indicators
func BoundOBVMovingAverageLength(v int) int {
	return int(math.Max(10, math.Min(50, float64(v)))) // Default: 20
}

func BoundOBVMovingAverageLengthFloat64(v float64) float64 {
	return math.Max(10, math.Min(50, v))
}

func BoundVolumesMovingAverageLength(v int) int {
	return int(math.Max(10, math.Min(50, float64(v)))) // Default: 20
}

func BoundVolumesMovingAverageLengthFloat64(v float64) float64 {
	return math.Max(10, math.Min(50, v))
}

// Money Flow Indicators
func BoundChaikinMoneyFlowPeriod(v int) int {
	return int(math.Max(10, math.Min(40, float64(v)))) // Default: 20
}

func BoundChaikinMoneyFlowPeriodFloat64(v float64) float64 {
	return math.Max(10, math.Min(40, v))
}

func BoundMoneyFlowIndexPeriod(v int) int {
	return int(math.Max(10, math.Min(40, float64(v)))) // Default: 14
}

func BoundMoneyFlowIndexPeriodFloat64(v float64) float64 {
	return math.Max(10, math.Min(40, v))
}

// Momentum Indicators
func BoundRateOfChangePeriod(v int) int {
	return int(math.Max(10, math.Min(50, float64(v)))) // Default: 14
}

func BoundRateOfChangePeriodFloat64(v float64) float64 {
	return math.Max(10, math.Min(50, v))
}

func BoundCCIPeriod(v int) int {
	return int(math.Max(10, math.Min(50, float64(v)))) // Default: 20
}

func BoundCCIPeriodFloat64(v float64) float64 {
	return math.Max(10, math.Min(50, v))
}

func BoundWilliamsRPeriod(v int) int {
	return int(math.Max(10, math.Min(30, float64(v)))) // Default: 14
}

func BoundWilliamsRPeriodFloat64(v float64) float64 {
	return math.Max(10, math.Min(30, v))
}

// Price Change Periods
func BoundPriceChangeFastPeriod(v int) int {
	return int(math.Max(10, math.Min(100, float64(v)))) // Default: 60
}

func BoundPriceChangeFastPeriodFloat64(v float64) float64 {
	return math.Max(10, math.Min(100, v))
}

func BoundPriceChangeMediumPeriod(v int) int {
	return int(math.Max(50, math.Min(500, float64(v)))) // Default: 240
}

func BoundPriceChangeMediumPeriodFloat64(v float64) float64 {
	return math.Max(50, math.Min(500, v))
}

func BoundPriceChangeSlowPeriod(v int) int {
	return int(math.Max(500, math.Min(2000, float64(v)))) // Default: 1440
}

func BoundPriceChangeSlowPeriodFloat64(v float64) float64 {
	return math.Max(500, math.Min(2000, v))
}

func BoundL2Penalty(v float64) float64 {
	return math.Max(0.0001, math.Min(0.1, v))
}

func BoundDropoutRate(v float64) float64 {
	return math.Max(0.1, math.Min(0.5, v))
}

func BoundLearnRate(v float64) float64 {
	return math.Max(1e-5, math.Min(1e-2, v))
}

func BoundMinTradeProbability(v float64) float64 {
	return math.Max(0.3, math.Min(1, v))
}
