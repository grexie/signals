package model

import (
	"log"
	"math"
	"math/rand"
	"sort"

	"github.com/grexie/signals/pkg/ta"
	"github.com/jedib0t/go-pretty/v6/progress"
)

func PrepareForPrediction(candles []Candle, params ModelParams) [][]float64 {
	features := [][]float64{}

	if len(candles) <= params.WindowSize {
		log.Fatalf("Not enough candles for the specified window size")
	}

	// Extract time series data
	closes := make([]float64, len(candles))
	lows := make([]float64, len(candles))
	highs := make([]float64, len(candles))
	volumes := make([]float64, len(candles))
	opens := make([]float64, len(candles))

	for i, candle := range candles {
		closes[i] = candle.Close
		lows[i] = candle.Low
		highs[i] = candle.High
		volumes[i] = candle.Volume
		opens[i] = candle.Open
	}

	// Calculate base technical indicators
	ma50 := ta.MovingAverage(closes, params.ShortMovingAverageLength)
	ma200 := ta.MovingAverage(closes, params.LongMovingAverageLength)
	rsi14 := ta.RSI(closes, params.LongRSILength)
	rsi5 := ta.RSI(closes, params.ShortRSILength) // Short-term RSI for quick movements
	macd, macdSignal := ta.MACD(closes, params.ShortMACDWindowLength, params.LongMACDWindowLength, params.MACDSignalWindow)
	macdFast, macdFastSignal := ta.MACD(closes, params.FastShortMACDWindowLength, params.FastLongMACDWindowLength, params.FastMACDSignalWindow) // Faster MACD
	ma20, bbUpper, bbLower := ta.BollingerBands(closes, params.BollingerBandsWindow, params.BollingerBandsMultiplier)
	stochK, stochD := ta.StochasticOscillator(closes, lows, highs, params.StochasticOscillatorWindow)
	vwap := ta.VWAP(closes, volumes)

	// Additional technical indicators
	atr14 := ta.ATR(highs, lows, closes, params.FastATRPeriod)
	atr20 := ta.ATR(highs, lows, closes, params.SlowATRPeriod)
	obv := ta.OBV(closes, volumes)
	obvEma := ta.MovingAverage(obv, params.OBVMovingAverageLength)

	// Volume indicators
	vwma := ta.MovingAverage(volumes, params.VolumesMovingAverageLength)
	cmf := ta.ChaikinMoneyFlow(highs, lows, closes, volumes, params.ChaikinMoneyFlowPeriod)
	mfi := ta.MoneyFlowIndex(highs, lows, closes, volumes, params.MoneyFlowIndexPeriod)

	// Momentum indicators
	roc := ta.RateOfChange(closes, params.RateOfChangePeriod)
	cci := ta.CCI(highs, lows, closes, params.CCIPeriod)
	williamsR := ta.WilliamsR(highs, lows, closes, params.WilliamsRPeriod)

	// Price changes over different timeframes
	priceChange1h := ta.PriceChanges(closes, params.PriceChangeFastPeriod)
	priceChange4h := ta.PriceChanges(closes, params.PriceChangeMediumPeriod)
	priceChange1d := ta.PriceChanges(closes, params.PriceChangeSlowPeriod)

	// Feature extraction with sliding window
	for i := params.WindowSize; i < len(candles); i++ {

		rsiSlope := (rsi14[i]-rsi14[i-params.RSISlope])/float64(100*params.RSISlope) + 0.5
		divergence := ta.Divergence(candles, macd, i, 20)

		// Base features
		currentFeatures := []float64{
			normalizeValue(closes[i], closes[i-params.WindowSize:i+1]),
			normalizeValue(ma50[i], ma50[i-params.WindowSize:i+1]),
			normalizeValue(ma200[i], ma200[i-params.WindowSize:i+1]),
			rsi14[i] / 100.0,
			rsi5[i] / 100.0, // Added short-term RSI
			rsiSlope,
			normalizeValue(macd[i], macd[i-params.WindowSize:i+1]),
			normalizeValue(macdSignal[i], macdSignal[i-params.WindowSize:i+1]),
			normalizeValue(macdFast[i], macdFast[i-params.WindowSize:i+1]),
			normalizeValue(macdFastSignal[i], macdFastSignal[i-params.WindowSize:i+1]),
			normalizeValue(ma20[i], ma20[i-params.WindowSize:i+1]),
			float64(divergence)/2.0 + 0.5,
			normalizeValue(bbUpper[i], bbUpper[i-params.WindowSize:i+1]),
			normalizeValue(bbLower[i], bbLower[i-params.WindowSize:i+1]),
			stochK[i] / 100.0,
			stochD[i] / 100.0,
			normalizeValue(vwap[i], vwap[i-params.WindowSize:i+1]),
		}

		// Volume features
		volumeFeatures := []float64{
			normalizeValue(volumes[i], volumes[i-params.WindowSize:i+1]),
			normalizeValue(vwma[i], vwma[i-params.WindowSize:i+1]),
			normalizeValue(obv[i], obv[i-params.WindowSize:i+1]),
			normalizeValue(obvEma[i], obvEma[i-params.WindowSize:i+1]),
			normalizeValue(cmf[i], cmf[i-params.WindowSize:i+1]),
			normalizeValue(mfi[i], mfi[i-params.WindowSize:i+1]),
		}
		currentFeatures = append(currentFeatures, volumeFeatures...)

		// Momentum features
		momentumFeatures := []float64{
			normalizeValue(roc[i], roc[i-params.WindowSize:i+1]),
			normalizeValue(cci[i], cci[i-params.WindowSize:i+1]),
			normalizeValue(williamsR[i], williamsR[i-params.WindowSize:i+1]),
		}
		currentFeatures = append(currentFeatures, momentumFeatures...)

		// Volatility features
		volatilityFeatures := []float64{
			normalizeValue(atr14[i], atr14[i-params.WindowSize:i+1]),
			normalizeValue(atr20[i], atr20[i-params.WindowSize:i+1]),
		}
		currentFeatures = append(currentFeatures, volatilityFeatures...)

		// Price change features
		priceChangeFeatures := []float64{
			normalizeValue(priceChange1h[i], priceChange1h[i-params.WindowSize:i+1]),
			normalizeValue(priceChange4h[i], priceChange4h[i-params.WindowSize:i+1]),
			normalizeValue(priceChange1d[i], priceChange1d[i-params.WindowSize:i+1]),
		}
		currentFeatures = append(currentFeatures, priceChangeFeatures...)

		// Pattern recognition features
		patternFeatures := []float64{
			boolToFloat(ta.IsDoji(opens[i], closes[i], highs[i], lows[i])),
			boolToFloat(ta.IsHammer(opens[i], closes[i], highs[i], lows[i])),
			boolToFloat(ta.IsEngulfing(opens[i], closes[i], opens[i-1], closes[i-1])),
			(closes[i] - bbLower[i]) / (bbUpper[i] - bbLower[i]), // BB position
		}
		currentFeatures = append(currentFeatures, patternFeatures...)

		// Momentum and acceleration
		priceVelocity := ta.Momentum(closes[i-5 : i+1])
		priceAcceleration := ta.Acceleration(closes[i-10 : i+1])
		currentFeatures = append(currentFeatures,
			normalizeValue(priceVelocity, []float64{-0.05, 0.05}),
			normalizeValue(priceAcceleration, []float64{-0.01, 0.01}))

		features = append(features, currentFeatures)
	}

	return features
}

// Improved data preparation
func Prepare(pw progress.Writer, candles []Candle, params ModelParams) ([][]float64, []float64) {
	tracker := progress.Tracker{
		Message: "Preparing data",
		Total:   int64(len(candles)) + 5,
		Units:   progress.UnitsDefault,
	}
	pw.AppendTracker(&tracker)
	tracker.Start()

	features := [][]float64{}
	labels := []float64{}

	if len(candles) <= params.WindowSize {
		log.Fatalf("Not enough candles for the specified window size")
	}

	// Extract time series data
	closes := make([]float64, len(candles))
	lows := make([]float64, len(candles))
	highs := make([]float64, len(candles))
	volumes := make([]float64, len(candles))
	opens := make([]float64, len(candles))

	for i, candle := range candles {
		closes[i] = candle.Close
		lows[i] = candle.Low
		highs[i] = candle.High
		volumes[i] = candle.Volume
		opens[i] = candle.Open
	}

	// Calculate base technical indicators
	tracker.Message = "Calculating technical indicators"
	ma50 := ta.MovingAverage(closes, params.ShortMovingAverageLength)                                                                           // 50
	ma200 := ta.MovingAverage(closes, params.LongMovingAverageLength)                                                                           // 200
	rsi14 := ta.RSI(closes, params.LongRSILength)                                                                                               // 14
	rsi5 := ta.RSI(closes, params.ShortRSILength)                                                                                               // 5
	macd, macdSignal := ta.MACD(closes, params.ShortMACDWindowLength, params.LongMACDWindowLength, params.MACDSignalWindow)                     // 12, 26, 9
	macdFast, macdFastSignal := ta.MACD(closes, params.FastShortMACDWindowLength, params.FastLongMACDWindowLength, params.FastMACDSignalWindow) // 5, 35, 5
	ma20, bbUpper, bbLower := ta.BollingerBands(closes, params.BollingerBandsWindow, params.BollingerBandsMultiplier)                           // 20, 2.0
	stochK, stochD := ta.StochasticOscillator(closes, lows, highs, params.StochasticOscillatorWindow)                                           // 14
	vwap := ta.VWAP(closes, volumes)
	tracker.Increment(1)

	// Additional technical indicators
	tracker.Message = "Additional technical indicators"
	atr14 := ta.ATR(highs, lows, closes, params.SlowATRPeriod) // 14
	atr20 := ta.ATR(highs, lows, closes, params.FastATRPeriod) // 20
	obv := ta.OBV(closes, volumes)
	obvEma := ta.MovingAverage(obv, params.OBVMovingAverageLength) // 20
	tracker.Increment(1)

	// Volume indicators
	tracker.Message = "Volume technical indicators"
	vwma := ta.MovingAverage(volumes, params.VolumesMovingAverageLength)                    // 20
	cmf := ta.ChaikinMoneyFlow(highs, lows, closes, volumes, params.ChaikinMoneyFlowPeriod) // 20
	mfi := ta.MoneyFlowIndex(highs, lows, closes, volumes, params.MoneyFlowIndexPeriod)     // 14
	tracker.Increment(1)

	// Momentum indicators
	tracker.Message = "Rate of change"
	roc := ta.RateOfChange(closes, params.RateOfChangePeriod)              // 14
	cci := ta.CCI(highs, lows, closes, params.CCIPeriod)                   // 20
	williamsR := ta.WilliamsR(highs, lows, closes, params.WilliamsRPeriod) // 14
	tracker.Increment(1)

	// Price changes over different timeframes
	tracker.Message = "Price changes"
	priceChange1h := ta.PriceChanges(closes, params.PriceChangeFastPeriod)   // 60
	priceChange4h := ta.PriceChanges(closes, params.PriceChangeMediumPeriod) // 240
	priceChange1d := ta.PriceChanges(closes, params.PriceChangeSlowPeriod)   // 1440
	tracker.Increment(1)

	tracker.Message = "Feature extraction"

	// Feature extraction with sliding window
	for i := params.WindowSize; i < len(candles)-params.Candles; i++ {
		tracker.Increment(1)

		rsiSlope := (rsi14[i]-rsi14[i-params.RSISlope])/float64(100*params.RSISlope) + 0.5
		divergence := ta.Divergence(candles, macd, i, 20)

		// Base features
		currentFeatures := []float64{
			normalizeValue(closes[i], closes[i-params.WindowSize:i+1]),
			normalizeValue(ma50[i], ma50[i-params.WindowSize:i+1]),
			normalizeValue(ma200[i], ma200[i-params.WindowSize:i+1]),
			rsi14[i] / 100.0,
			rsi5[i] / 100.0, // Added short-term RSI
			rsiSlope,
			normalizeValue(macd[i], macd[i-params.WindowSize:i+1]),
			normalizeValue(macdSignal[i], macdSignal[i-params.WindowSize:i+1]),
			normalizeValue(macdFast[i], macdFast[i-params.WindowSize:i+1]),
			normalizeValue(macdFastSignal[i], macdFastSignal[i-params.WindowSize:i+1]),
			normalizeValue(ma20[i], ma20[i-params.WindowSize:i+1]),
			float64(divergence)/2.0 + 0.5,
			normalizeValue(bbUpper[i], bbUpper[i-params.WindowSize:i+1]),
			normalizeValue(bbLower[i], bbLower[i-params.WindowSize:i+1]),
			stochK[i] / 100.0,
			stochD[i] / 100.0,
			normalizeValue(vwap[i], vwap[i-params.WindowSize:i+1]),
		}

		// Volume features
		volumeFeatures := []float64{
			normalizeValue(volumes[i], volumes[i-params.WindowSize:i+1]),
			normalizeValue(vwma[i], vwma[i-params.WindowSize:i+1]),
			normalizeValue(obv[i], obv[i-params.WindowSize:i+1]),
			normalizeValue(obvEma[i], obvEma[i-params.WindowSize:i+1]),
			normalizeValue(cmf[i], cmf[i-params.WindowSize:i+1]),
			normalizeValue(mfi[i], mfi[i-params.WindowSize:i+1]),
		}
		currentFeatures = append(currentFeatures, volumeFeatures...)

		// Momentum features
		momentumFeatures := []float64{
			normalizeValue(roc[i], roc[i-params.WindowSize:i+1]),
			normalizeValue(cci[i], cci[i-params.WindowSize:i+1]),
			normalizeValue(williamsR[i], williamsR[i-params.WindowSize:i+1]),
		}
		currentFeatures = append(currentFeatures, momentumFeatures...)

		// Volatility features
		volatilityFeatures := []float64{
			normalizeValue(atr14[i], atr14[i-params.WindowSize:i+1]),
			normalizeValue(atr20[i], atr20[i-params.WindowSize:i+1]),
		}
		currentFeatures = append(currentFeatures, volatilityFeatures...)

		// Price change features
		priceChangeFeatures := []float64{
			normalizeValue(priceChange1h[i], priceChange1h[i-params.WindowSize:i+1]),
			normalizeValue(priceChange4h[i], priceChange4h[i-params.WindowSize:i+1]),
			normalizeValue(priceChange1d[i], priceChange1d[i-params.WindowSize:i+1]),
		}
		currentFeatures = append(currentFeatures, priceChangeFeatures...)

		// Pattern recognition features
		patternFeatures := []float64{
			boolToFloat(ta.IsDoji(opens[i], closes[i], highs[i], lows[i])),
			boolToFloat(ta.IsHammer(opens[i], closes[i], highs[i], lows[i])),
			boolToFloat(ta.IsEngulfing(opens[i], closes[i], opens[i-1], closes[i-1])),
			(closes[i] - bbLower[i]) / (bbUpper[i] - bbLower[i]), // BB position
		}
		currentFeatures = append(currentFeatures, patternFeatures...)

		// Momentum and acceleration
		priceVelocity := ta.Momentum(closes[i-5 : i+1])
		priceAcceleration := ta.Acceleration(closes[i-10 : i+1])
		currentFeatures = append(currentFeatures,
			normalizeValue(priceVelocity, []float64{-0.05, 0.05}),
			normalizeValue(priceAcceleration, []float64{-0.01, 0.01}))

		features = append(features, currentFeatures)

		// Enhanced labeling strategy
		label := StrategyHold
		basePrice := candles[i].Close

		// Look ahead window
		highestHigh := basePrice
		lowestLow := basePrice
		closingPrice := candles[i+params.Candles].Close

		for j := 1; j <= params.Candles; j++ {
			highestHigh = math.Max(highestHigh, candles[i+j].High)
			lowestLow = math.Min(lowestLow, candles[i+j].Low)

			potentialGain := (highestHigh - basePrice) / basePrice
			potentialLoss := (basePrice - lowestLow) / basePrice
			actualChange := (closingPrice - basePrice) / basePrice

			// Enhanced signal generation with trend confirmation
			if potentialGain >= params.TakeProfit &&
				potentialLoss < params.StopLoss &&
				actualChange > 0 &&
				rsi14[i] < params.RSILowerBound &&
				rsiSlope > 0.45 &&
				macd[i] > macdSignal[i] &&
				divergence == ta.DivergenceBullish {
				label = StrategyLong
				break
			} else if potentialLoss >= params.TakeProfit &&
				potentialGain < params.StopLoss &&
				actualChange < 0 &&
				rsi14[i] > params.RSIUpperBound &&
				rsiSlope < 0.55 &&
				macd[i] < macdSignal[i] &&
				divergence == ta.DivergenceBearish {
				label = StrategyShort
				break
			}
		}

		labels = append(labels, float64(label))
	}
	tracker.MarkAsDone()

	// Balance classes and augment data
	features, labels = balanceClasses(pw, features, labels)

	// randomize results
	outIdx := make([]int, len(features))
	for i := range len(features) {
		outIdx[i] = i
	}
	sort.Slice(outIdx, func(i, j int) bool {
		return rand.Intn(2) == 0
	})
	outFeatures, outLabels := make([][]float64, len(features)), make([]float64, len(features))
	for i, v := range outIdx {
		outFeatures[i] = features[v]
		outLabels[i] = labels[v]
	}
	return outFeatures, outLabels
}
