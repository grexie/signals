package genetics

import (
	"math"
	"math/rand/v2"
	"time"

	"github.com/grexie/signals/pkg/model"
)

type Strategy struct {
	Instrument string
	WindowSize float64
	Candles    float64
	TakeProfit float64
	StopLoss   float64

	Cooldown float64

	MinTradeProbability float64

	ShortMovingAverageLength   float64
	LongMovingAverageLength    float64
	LongRSILength              float64
	ShortRSILength             float64
	ShortMACDWindowLength      float64
	LongMACDWindowLength       float64
	MACDSignalWindow           float64
	FastShortMACDWindowLength  float64
	FastLongMACDWindowLength   float64
	FastMACDSignalWindow       float64
	BollingerBandsWindow       float64
	BollingerBandsMultiplier   float64
	StochasticOscillatorWindow float64
	SlowATRPeriod              float64
	FastATRPeriod              float64
	OBVMovingAverageLength     float64
	VolumesMovingAverageLength float64
	ChaikinMoneyFlowPeriod     float64
	MoneyFlowIndexPeriod       float64
	RateOfChangePeriod         float64
	CCIPeriod                  float64
	WilliamsRPeriod            float64
	PriceChangeFastPeriod      float64
	PriceChangeMediumPeriod    float64
	PriceChangeSlowPeriod      float64
	RSIUpperBound              float64
	RSILowerBound              float64
	RSISlope                   float64

	BatchSizeLog2       float64
	HiddenLayerSizeLog2 float64
	L2Penalty           float64
	DropoutRate         float64
	LearnRate           float64
	TrainDays           float64

	ModelMetrics *model.ModelMetrics
}

func randPercent(dev float64) float64 {
	return 1 + (rand.Float64()*(2*dev)-dev)/100
}

// Generate a strategy from configured values
func newStrategy(instrument string) Strategy {
	return Strategy{
		Instrument: instrument,

		BatchSizeLog2:       model.BoundBatchSizeLog2Float64(math.Log2(float64(model.BatchSize()))),
		HiddenLayerSizeLog2: model.BoundHiddenLayerSizeLog2Float64(math.Log2(float64(model.HiddenLayerSize()))),
		WindowSize:          model.BoundWindowSizeFloat64(float64(model.WindowSize())),
		Candles:             model.BoundCandlesFloat64(float64(model.Candles())),
		StopLoss:            model.BoundStopLoss(model.StopLoss()),
		TakeProfit:          model.BoundTakeProfit(model.TakeProfit()),

		Cooldown: model.BoundCooldownFloat64(float64(model.Cooldown().Seconds())),

		MinTradeProbability: model.BoundMinTradeProbability(model.MinTradeProbability()),

		ShortMovingAverageLength:   model.BoundShortMovingAverageLengthFloat64(float64(model.ShortMovingAverageLength())),
		LongMovingAverageLength:    model.BoundLongMovingAverageLengthFloat64(float64(model.LongMovingAverageLength())),
		LongRSILength:              model.BoundLongRSILengthFloat64(float64(model.LongRSILength())),
		ShortRSILength:             model.BoundShortRSILengthFloat64(float64(model.ShortRSILength())),
		ShortMACDWindowLength:      model.BoundShortMACDWindowLengthFloat64(float64(model.ShortMACDWindowLength())),
		LongMACDWindowLength:       model.BoundLongMACDWindowLengthFloat64(float64(model.LongMACDWindowLength())),
		MACDSignalWindow:           model.BoundMACDSignalWindowFloat64(float64(model.MACDSignalWindow())),
		FastShortMACDWindowLength:  model.BoundFastShortMACDWindowLengthFloat64(float64(model.FastShortMACDWindowLength())),
		FastLongMACDWindowLength:   model.BoundFastLongMACDWindowLengthFloat64(float64(model.FastLongMACDWindowLength())),
		FastMACDSignalWindow:       model.BoundFastMACDSignalWindowFloat64(float64(model.FastMACDSignalWindow())),
		BollingerBandsWindow:       model.BoundBollingerBandsWindowFloat64(float64(model.BollingerBandsWindow())),
		BollingerBandsMultiplier:   model.BoundBollingerBandsMultiplier(float64(model.BollingerBandsMultiplier())),
		StochasticOscillatorWindow: model.BoundStochasticOscillatorWindowFloat64(float64(model.StochasticOscillatorWindow())),
		SlowATRPeriod:              model.BoundSlowATRPeriodFloat64(float64(model.SlowATRPeriod())),
		FastATRPeriod:              model.BoundFastATRPeriodFloat64(float64(model.FastATRPeriod())),
		OBVMovingAverageLength:     model.BoundOBVMovingAverageLengthFloat64(float64(model.OBVMovingAverageLength())),
		VolumesMovingAverageLength: model.BoundVolumesMovingAverageLengthFloat64(float64(model.VolumesMovingAverageLength())),
		ChaikinMoneyFlowPeriod:     model.BoundChaikinMoneyFlowPeriodFloat64(float64(model.ChaikinMoneyFlowPeriod())),
		MoneyFlowIndexPeriod:       model.BoundMoneyFlowIndexPeriodFloat64(float64(model.MoneyFlowIndexPeriod())),
		RateOfChangePeriod:         model.BoundRateOfChangePeriodFloat64(float64(model.RateOfChangePeriod())),
		CCIPeriod:                  model.BoundCCIPeriodFloat64(float64(model.CCIPeriod())),
		WilliamsRPeriod:            model.BoundWilliamsRPeriodFloat64(float64(model.WilliamsRPeriod())),
		PriceChangeFastPeriod:      model.BoundPriceChangeFastPeriodFloat64(float64(model.PriceChangeFastPeriod())),
		PriceChangeMediumPeriod:    model.BoundPriceChangeMediumPeriodFloat64(float64(model.PriceChangeMediumPeriod())),
		PriceChangeSlowPeriod:      model.BoundPriceChangeSlowPeriodFloat64(float64(model.PriceChangeSlowPeriod())),
		RSIUpperBound:              model.BoundRSIUpperBound(float64(model.RSIUpperBound())),
		RSILowerBound:              model.BoundRSILowerBound(float64(model.RSILowerBound())),
		RSISlope:                   model.BoundRSISlopeFloat64(float64(model.RSISlope())),

		L2Penalty:   model.BoundL2Penalty(model.L2Penalty()),
		DropoutRate: model.BoundDropoutRate(model.DropoutRate()),
		LearnRate:   model.BoundLearnRate(model.LearnRate()),
		TrainDays:   model.BoundTrainDaysFloat64(float64(model.TrainDays()) / (24 * float64(time.Hour))),
	}
}

func randomizeStrategy(s *Strategy, percent float64) {
	s.WindowSize = model.BoundWindowSizeFloat64(s.WindowSize * randPercent(percent))
	s.Candles = model.BoundCandlesFloat64(s.Candles * randPercent(percent))
	s.TakeProfit = model.BoundTakeProfit(s.TakeProfit * randPercent(percent))
	s.StopLoss = model.BoundStopLoss(s.StopLoss * randPercent(percent))

	s.Cooldown = model.BoundCooldownFloat64(s.Cooldown * randPercent(percent))

	s.MinTradeProbability = model.BoundMinTradeProbability(s.MinTradeProbability * randPercent(percent))

	s.ShortMovingAverageLength = model.BoundShortMovingAverageLengthFloat64(s.ShortMovingAverageLength * randPercent(percent))
	s.LongMovingAverageLength = model.BoundLongMovingAverageLengthFloat64(s.LongMovingAverageLength * randPercent(percent))
	s.LongRSILength = model.BoundLongRSILengthFloat64(s.LongRSILength * randPercent(percent))
	s.ShortRSILength = model.BoundShortRSILengthFloat64(s.ShortRSILength * randPercent(percent))
	s.ShortMACDWindowLength = model.BoundShortMACDWindowLengthFloat64(s.ShortMACDWindowLength * randPercent(percent))
	s.LongMACDWindowLength = model.BoundLongMACDWindowLengthFloat64(s.LongMACDWindowLength * randPercent(percent))
	s.MACDSignalWindow = model.BoundMACDSignalWindowFloat64(s.MACDSignalWindow * randPercent(percent))
	s.FastShortMACDWindowLength = model.BoundFastShortMACDWindowLengthFloat64(s.FastShortMACDWindowLength * randPercent(percent))
	s.FastLongMACDWindowLength = model.BoundFastLongMACDWindowLengthFloat64(s.FastLongMACDWindowLength * randPercent(percent))
	s.FastMACDSignalWindow = model.BoundFastMACDSignalWindowFloat64(s.FastMACDSignalWindow * randPercent(percent))
	s.BollingerBandsWindow = model.BoundBollingerBandsWindowFloat64(s.BollingerBandsWindow * randPercent(percent))
	s.BollingerBandsMultiplier = model.BoundBollingerBandsMultiplier(s.BollingerBandsMultiplier * randPercent(percent))
	s.StochasticOscillatorWindow = model.BoundStochasticOscillatorWindowFloat64(s.StochasticOscillatorWindow * randPercent(percent))
	s.SlowATRPeriod = model.BoundSlowATRPeriodFloat64(s.SlowATRPeriod * randPercent(percent))
	s.FastATRPeriod = model.BoundFastATRPeriodFloat64(s.FastATRPeriod * randPercent(percent))
	s.OBVMovingAverageLength = model.BoundOBVMovingAverageLengthFloat64(s.OBVMovingAverageLength * randPercent(percent))
	s.VolumesMovingAverageLength = model.BoundVolumesMovingAverageLengthFloat64(s.VolumesMovingAverageLength * randPercent(percent))
	s.ChaikinMoneyFlowPeriod = model.BoundChaikinMoneyFlowPeriodFloat64(s.ChaikinMoneyFlowPeriod * randPercent(percent))
	s.MoneyFlowIndexPeriod = model.BoundMoneyFlowIndexPeriodFloat64(s.MoneyFlowIndexPeriod * randPercent(percent))
	s.RateOfChangePeriod = model.BoundRateOfChangePeriodFloat64(s.RateOfChangePeriod * randPercent(percent))
	s.CCIPeriod = model.BoundCCIPeriodFloat64(s.CCIPeriod * randPercent(percent))
	s.WilliamsRPeriod = model.BoundWilliamsRPeriodFloat64(s.WilliamsRPeriod * randPercent(percent))
	s.PriceChangeFastPeriod = model.BoundPriceChangeFastPeriodFloat64(s.PriceChangeFastPeriod * randPercent(percent))
	s.PriceChangeMediumPeriod = model.BoundPriceChangeMediumPeriodFloat64(s.PriceChangeMediumPeriod * randPercent(percent))
	s.PriceChangeSlowPeriod = model.BoundPriceChangeSlowPeriodFloat64(s.PriceChangeSlowPeriod * randPercent(percent))
	s.RSIUpperBound = model.BoundRSIUpperBound(s.RSIUpperBound * randPercent(percent))
	s.RSILowerBound = model.BoundRSILowerBound(s.RSILowerBound * randPercent(percent))
	s.RSISlope = model.BoundRSISlopeFloat64(s.RSISlope * randPercent(percent))

	s.BatchSizeLog2 = model.BoundBatchSizeLog2Float64(s.BatchSizeLog2 * randPercent(percent))
	s.HiddenLayerSizeLog2 = model.BoundHiddenLayerSizeLog2Float64(s.HiddenLayerSizeLog2 * randPercent(percent))
	s.L2Penalty = model.BoundL2Penalty(s.L2Penalty * randPercent(percent))
	s.DropoutRate = model.BoundDropoutRate(s.DropoutRate * randPercent(percent))
	s.LearnRate = model.BoundLearnRate(s.LearnRate * randPercent(percent))
	s.TrainDays = model.BoundTrainDaysFloat64(s.TrainDays * randPercent(percent))
}

func StrategyToParams(s Strategy) model.ModelParams {
	return model.ModelParams{
		Instrument:      model.Instrument(),
		Leverage:        model.Leverage(),
		TradeMultiplier: model.TradeMultiplier(),
		Commission:      model.Commission(),
		Cooldown:        time.Duration(s.Cooldown * float64(time.Second)),

		BatchSize:       int(math.Pow(2, float64(int(s.BatchSizeLog2)))),
		HiddenLayerSize: int(math.Pow(2, float64(int(s.HiddenLayerSizeLog2)))),
		L2Penalty:       s.L2Penalty,
		DropoutRate:     s.DropoutRate,
		LearnRate:       s.LearnRate,
		TrainDays:       time.Duration(s.TrainDays * float64(time.Hour) * 24),

		WindowSize: int(s.WindowSize),
		Candles:    int(s.Candles),
		TakeProfit: s.TakeProfit / model.Leverage(),
		StopLoss:   s.StopLoss / model.Leverage(),

		MinTradeProbability: s.MinTradeProbability,

		ShortMovingAverageLength:   int(s.ShortMovingAverageLength),
		LongMovingAverageLength:    int(s.LongMovingAverageLength),
		LongRSILength:              int(s.LongRSILength),
		ShortRSILength:             int(s.ShortRSILength),
		ShortMACDWindowLength:      int(s.ShortMACDWindowLength),
		LongMACDWindowLength:       int(s.LongMACDWindowLength),
		MACDSignalWindow:           int(s.MACDSignalWindow),
		FastShortMACDWindowLength:  int(s.FastShortMACDWindowLength),
		FastLongMACDWindowLength:   int(s.FastLongMACDWindowLength),
		FastMACDSignalWindow:       int(s.FastMACDSignalWindow),
		BollingerBandsWindow:       int(s.BollingerBandsWindow),
		BollingerBandsMultiplier:   s.BollingerBandsMultiplier,
		StochasticOscillatorWindow: int(s.StochasticOscillatorWindow),
		SlowATRPeriod:              int(s.SlowATRPeriod),
		FastATRPeriod:              int(s.FastATRPeriod),
		OBVMovingAverageLength:     int(s.OBVMovingAverageLength),
		VolumesMovingAverageLength: int(s.VolumesMovingAverageLength),
		ChaikinMoneyFlowPeriod:     int(s.ChaikinMoneyFlowPeriod),
		MoneyFlowIndexPeriod:       int(s.MoneyFlowIndexPeriod),
		RateOfChangePeriod:         int(s.RateOfChangePeriod),
		CCIPeriod:                  int(s.CCIPeriod),
		WilliamsRPeriod:            int(s.WilliamsRPeriod),
		PriceChangeFastPeriod:      int(s.PriceChangeFastPeriod),
		PriceChangeMediumPeriod:    int(s.PriceChangeMediumPeriod),
		PriceChangeSlowPeriod:      int(s.PriceChangeSlowPeriod),
		RSIUpperBound:              s.RSIUpperBound,
		RSILowerBound:              s.RSILowerBound,
		RSISlope:                   int(s.RSISlope),
	}
}
