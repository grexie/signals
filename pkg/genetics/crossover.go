package genetics

import "math/rand/v2"

// Crossover (Breed new strategies from the best ones)
func crossover(parent1, parent2 Strategy) Strategy {
	if parent1.Instrument != parent2.Instrument {
		parent1 = parent2
	}

	// Helper function to select between parent1, parent2, or an average
	selectValue := func(a, b float64) float64 {
		r := rand.Float64()
		if r < 0.4 { // 40% chance inherit from parent 1
			return a
		} else if r < 0.8 { // 40% chance inherit from parent 2
			return b
		}
		// 20% chance take the average
		return (a + b) / 2
	}

	return Strategy{
		Instrument: parent1.Instrument,

		BatchSizeLog2:       selectValue(parent1.BatchSizeLog2, parent2.BatchSizeLog2),
		HiddenLayerSizeLog2: selectValue(parent1.HiddenLayerSizeLog2, parent2.HiddenLayerSizeLog2),
		L2Penalty:           selectValue(parent1.L2Penalty, parent2.L2Penalty),
		DropoutRate:         selectValue(parent1.DropoutRate, parent2.DropoutRate),
		LearnRate:           selectValue(parent1.LearnRate, parent2.LearnRate),
		TrainDays:           selectValue(parent1.TrainDays, parent2.TrainDays),

		WindowSize: selectValue(parent1.WindowSize, parent2.WindowSize),
		Candles:    selectValue(parent1.Candles, parent2.Candles),
		TakeProfit: selectValue(parent1.TakeProfit, parent2.TakeProfit),
		StopLoss:   selectValue(parent1.StopLoss, parent2.StopLoss),

		Cooldown: selectValue(parent1.Cooldown, parent2.Cooldown),

		MinTradeProbability: selectValue(parent1.MinTradeProbability, parent2.MinTradeProbability),

		ShortMovingAverageLength:   selectValue(parent1.ShortMovingAverageLength, parent2.ShortMovingAverageLength),
		LongMovingAverageLength:    selectValue(parent1.LongMovingAverageLength, parent2.LongMovingAverageLength),
		LongRSILength:              selectValue(parent1.LongRSILength, parent2.LongRSILength),
		ShortRSILength:             selectValue(parent1.ShortRSILength, parent2.ShortRSILength),
		ShortMACDWindowLength:      selectValue(parent1.ShortMACDWindowLength, parent2.ShortMACDWindowLength),
		LongMACDWindowLength:       selectValue(parent1.LongMACDWindowLength, parent2.LongMACDWindowLength),
		MACDSignalWindow:           selectValue(parent1.MACDSignalWindow, parent2.MACDSignalWindow),
		FastShortMACDWindowLength:  selectValue(parent1.FastShortMACDWindowLength, parent2.FastShortMACDWindowLength),
		FastLongMACDWindowLength:   selectValue(parent1.FastLongMACDWindowLength, parent2.FastLongMACDWindowLength),
		FastMACDSignalWindow:       selectValue(parent1.FastMACDSignalWindow, parent2.FastMACDSignalWindow),
		BollingerBandsWindow:       selectValue(parent1.BollingerBandsWindow, parent2.BollingerBandsWindow),
		BollingerBandsMultiplier:   selectValue(parent1.BollingerBandsMultiplier, parent2.BollingerBandsMultiplier),
		StochasticOscillatorWindow: selectValue(parent1.StochasticOscillatorWindow, parent2.StochasticOscillatorWindow),
		SlowATRPeriod:              selectValue(parent1.SlowATRPeriod, parent2.SlowATRPeriod),
		FastATRPeriod:              selectValue(parent1.FastATRPeriod, parent2.FastATRPeriod),
		OBVMovingAverageLength:     selectValue(parent1.OBVMovingAverageLength, parent2.OBVMovingAverageLength),
		VolumesMovingAverageLength: selectValue(parent1.VolumesMovingAverageLength, parent2.VolumesMovingAverageLength),
		ChaikinMoneyFlowPeriod:     selectValue(parent1.ChaikinMoneyFlowPeriod, parent2.ChaikinMoneyFlowPeriod),
		MoneyFlowIndexPeriod:       selectValue(parent1.MoneyFlowIndexPeriod, parent2.MoneyFlowIndexPeriod),
		RateOfChangePeriod:         selectValue(parent1.RateOfChangePeriod, parent2.RateOfChangePeriod),
		CCIPeriod:                  selectValue(parent1.CCIPeriod, parent2.CCIPeriod),
		WilliamsRPeriod:            selectValue(parent1.WilliamsRPeriod, parent2.WilliamsRPeriod),
		PriceChangeFastPeriod:      selectValue(parent1.PriceChangeFastPeriod, parent2.PriceChangeFastPeriod),
		PriceChangeMediumPeriod:    selectValue(parent1.PriceChangeMediumPeriod, parent2.PriceChangeMediumPeriod),
		PriceChangeSlowPeriod:      selectValue(parent1.PriceChangeSlowPeriod, parent2.PriceChangeSlowPeriod),
		RSIUpperBound:              selectValue(parent1.RSIUpperBound, parent2.RSIUpperBound),
		RSILowerBound:              selectValue(parent1.RSILowerBound, parent2.RSILowerBound),
		RSISlope:                   selectValue(parent1.RSISlope, parent2.RSISlope),
	}
}
