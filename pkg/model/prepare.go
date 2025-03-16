package model

import (
	"fmt"
	"io"
	"log"
	"math"
	"math/rand"
	"os"
	"sort"
	"time"

	"github.com/grexie/signals/pkg/ta"
	"github.com/jedib0t/go-pretty/v6/progress"
	"github.com/jedib0t/go-pretty/v6/table"
)

type ModelParams struct {
	Instrument string

	WindowSize int
	Candles    int
	TakeProfit float64
	StopLoss   float64
	Leverage   float64

	TradeMultiplier float64
	Commission      float64
	Cooldown        time.Duration

	ShortMovingAverageLength   int
	LongMovingAverageLength    int
	LongRSILength              int
	ShortRSILength             int
	ShortMACDWindowLength      int
	LongMACDWindowLength       int
	MACDSignalWindow           int
	FastShortMACDWindowLength  int
	FastLongMACDWindowLength   int
	FastMACDSignalWindow       int
	BollingerBandsWindow       int
	BollingerBandsMultiplier   float64
	StochasticOscillatorWindow int
	SlowATRPeriod              int
	FastATRPeriod              int
	OBVMovingAverageLength     int
	VolumesMovingAverageLength int
	ChaikinMoneyFlowPeriod     int
	MoneyFlowIndexPeriod       int
	RateOfChangePeriod         int
	CCIPeriod                  int
	WilliamsRPeriod            int
	PriceChangeFastPeriod      int
	PriceChangeMediumPeriod    int
	PriceChangeSlowPeriod      int
	RSIUpperBound              float64
	RSILowerBound              float64
	RSISlope                   int

	L2Penalty   float64
	DropoutRate float64
	LearnRate   float64
}

func (m *ModelParams) Write(w io.Writer, title string) {
	t := table.NewWriter()
	t.SetOutputMirror(w)
	t.SetTitle(title)
	t.AppendRows([]table.Row{
		{"SIGNALS_INSTRUMENT", m.Instrument},
		{"SIGNALS_WINDOW_SIZE", fmt.Sprintf("%d", m.WindowSize)},
		{"SIGNALS_CANDLES", fmt.Sprintf("%d", m.Candles)},
		{"SIGNALS_TAKE_PROFIT", fmt.Sprintf("%0.04f", m.TakeProfit*m.Leverage)},
		{"SIGNALS_STOP_LOSS", fmt.Sprintf("%0.04f", m.StopLoss*m.Leverage)},
		{"SIGNALS_LEVERAGE", fmt.Sprintf("%0.0f", m.Leverage)},
		{"SIGNALS_TRADE_MULTIPLIER", fmt.Sprintf("%0.04f", m.TradeMultiplier)},
		{"SIGNALS_COMMISSION", fmt.Sprintf("%0.04f", m.Commission)},
		{"SIGNALS_COOLDOWN", fmt.Sprintf("%0.0f", m.Cooldown.Seconds())},
	})
	t.AppendSeparator()
	t.AppendRows([]table.Row{
		{"SIGNALS_L2_PENALTY", fmt.Sprintf("%.06f", m.L2Penalty)},
		{"SIGNALS_DROPOUT_RATE", fmt.Sprintf("%.06f", m.DropoutRate)},
		{"SIGNALS_LEARN_RATE", fmt.Sprintf("%.06f", m.LearnRate)},
	})
	t.AppendSeparator()
	t.AppendRows([]table.Row{
		{"SIGNALS_SHORT_MOVING_AVERAGE_LENGTH", fmt.Sprintf("%d", m.ShortMovingAverageLength)},
		{"SIGNALS_LONG_MOVING_AVERAGE_LENGTH", fmt.Sprintf("%d", m.LongMovingAverageLength)},
		{"SIGNALS_LONG_RSI_LENGTH", fmt.Sprintf("%d", m.LongRSILength)},
		{"SIGNALS_SHORT_RSI_LENGTH", fmt.Sprintf("%d", m.ShortRSILength)},
		{"SIGNALS_SHORT_MACD_WINDOW_LENGTH", fmt.Sprintf("%d", m.ShortMACDWindowLength)},
		{"SIGNALS_LONG_MACD_WINDOW_LENGTH", fmt.Sprintf("%d", m.LongMACDWindowLength)},
		{"SIGNALS_MACD_SIGNAL_WINDOW", fmt.Sprintf("%d", m.MACDSignalWindow)},
		{"SIGNALS_FAST_SHORT_MACD_WINDOW_LENGTH", fmt.Sprintf("%d", m.FastShortMACDWindowLength)},
		{"SIGNALS_FAST_LONG_MACD_WINDOW_LENGTH", fmt.Sprintf("%d", m.FastLongMACDWindowLength)},
		{"SIGNALS_FAST_MACD_SIGNAL_WINDOW", fmt.Sprintf("%d", m.FastMACDSignalWindow)},
		{"SIGNALS_BOLLINGER_BANDS_WINDOW", fmt.Sprintf("%d", m.BollingerBandsWindow)},
		{"SIGNALS_BOLLINGER_BANDS_MULTIPLIER", fmt.Sprintf("%0.02f", m.BollingerBandsMultiplier)},
		{"SIGNALS_STOCHASTIC_OSCILLATOR_WINDOW", fmt.Sprintf("%d", m.StochasticOscillatorWindow)},
		{"SIGNALS_SLOW_ATR_PERIOD_WINDOW", fmt.Sprintf("%d", m.SlowATRPeriod)},
		{"SIGNALS_FAST_ATR_PERIOD_WINDOW", fmt.Sprintf("%d", m.FastATRPeriod)},
		{"SIGNALS_OBV_MOVING_AVERAGE_LENGTH", fmt.Sprintf("%d", m.OBVMovingAverageLength)},
		{"SIGNALS_VOLUMES_MOVING_AVERAGE_LENGTH", fmt.Sprintf("%d", m.VolumesMovingAverageLength)},
		{"SIGNALS_CHAIKIN_MONEY_FLOW_PERIOD", fmt.Sprintf("%d", m.ChaikinMoneyFlowPeriod)},
		{"SIGNALS_MONEY_FLOW_INDEX_PERIOD", fmt.Sprintf("%d", m.MoneyFlowIndexPeriod)},
		{"SIGNALS_RATE_OF_CHANGE_PERIOD", fmt.Sprintf("%d", m.RateOfChangePeriod)},
		{"SIGNALS_CCI_PERIOD", fmt.Sprintf("%d", m.CCIPeriod)},
		{"SIGNALS_WILLIAMS_R_PERIOD", fmt.Sprintf("%d", m.WilliamsRPeriod)},
		{"SIGNALS_PRICE_CHANGE_FAST_PERIOD", fmt.Sprintf("%d", m.PriceChangeFastPeriod)},
		{"SIGNALS_PRICE_CHANGE_MEDIUM_PERIOD", fmt.Sprintf("%d", m.PriceChangeMediumPeriod)},
		{"SIGNALS_PRICE_CHANGE_SLOW_PERIOD", fmt.Sprintf("%d", m.PriceChangeSlowPeriod)},
		{"SIGNALS_RSI_UPPER_BOUND", fmt.Sprintf("%0.02f", m.RSIUpperBound)},
		{"SIGNALS_RSI_LOWER_BOUND", fmt.Sprintf("%0.02f", m.RSILowerBound)},
		{"SIGNALS_RSI_SLOPE", fmt.Sprintf("%d", m.RSISlope)},
	})
	t.Render()

	t = table.NewWriter()
	t.SetOutputMirror(os.Stdout)
	t.SetTitle("Trade Info")
	t.AppendRows([]table.Row{
		{"Take Profit", fmt.Sprintf("%0.02f%%", (100*m.TakeProfit*m.Leverage)/m.TradeMultiplier)},
		{"Stop Loss", fmt.Sprintf("%0.02f%%", (100 * m.StopLoss * m.TradeMultiplier))},
		{"Leverage", fmt.Sprintf("%0.0f", m.Leverage)},
	})
	t.AppendSeparator()
	t.AppendRows([]table.Row{
		{"TP %", fmt.Sprintf("%0.02f%%", 100*m.TakeProfit/(m.TradeMultiplier))},
		{"SL %", fmt.Sprintf("%0.02f%%", 100*m.StopLoss*m.TradeMultiplier)},
		{"Commission", fmt.Sprintf("%0.02f%%", 100*m.Commission*m.Leverage)},
	})
	t.Render()
}

func NewModelParamsFromDefaults() ModelParams {
	return ModelParams{
		Instrument: Instrument(),

		WindowSize: WindowSize(),
		Candles:    Candles(),
		TakeProfit: TakeProfit() / Leverage(),
		StopLoss:   StopLoss() / Leverage(),
		Leverage:   Leverage(),

		TradeMultiplier: TradeMultiplier(),
		Commission:      Commission(),
		Cooldown:        Cooldown(),

		ShortMovingAverageLength:   ShortMovingAverageLength(),
		LongMovingAverageLength:    LongMovingAverageLength(),
		LongRSILength:              LongRSILength(),
		ShortRSILength:             ShortRSILength(),
		ShortMACDWindowLength:      ShortMACDWindowLength(),
		LongMACDWindowLength:       LongMACDWindowLength(),
		MACDSignalWindow:           MACDSignalWindow(),
		FastShortMACDWindowLength:  FastShortMACDWindowLength(),
		FastLongMACDWindowLength:   FastLongMACDWindowLength(),
		FastMACDSignalWindow:       FastMACDSignalWindow(),
		BollingerBandsWindow:       BollingerBandsWindow(),
		BollingerBandsMultiplier:   BollingerBandsMultiplier(),
		StochasticOscillatorWindow: StochasticOscillatorWindow(),
		SlowATRPeriod:              SlowATRPeriod(),
		FastATRPeriod:              FastATRPeriod(),
		OBVMovingAverageLength:     OBVMovingAverageLength(),
		VolumesMovingAverageLength: VolumesMovingAverageLength(),
		ChaikinMoneyFlowPeriod:     ChaikinMoneyFlowPeriod(),
		MoneyFlowIndexPeriod:       MoneyFlowIndexPeriod(),
		RateOfChangePeriod:         RateOfChangePeriod(),
		CCIPeriod:                  CCIPeriod(),
		WilliamsRPeriod:            WilliamsRPeriod(),
		PriceChangeFastPeriod:      PriceChangeFastPeriod(),
		PriceChangeMediumPeriod:    PriceChangeMediumPeriod(),
		PriceChangeSlowPeriod:      PriceChangeSlowPeriod(),
		RSIUpperBound:              RSIUpperBound(),
		RSILowerBound:              RSILowerBound(),
		RSISlope:                   RSISlope(),

		L2Penalty:   L2Penalty(),
		DropoutRate: DropoutRate(),
		LearnRate:   LearnRate(),
	}
}

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

		rsiSlope := (rsi14[i] - rsi14[i-params.RSISlope]) / float64(params.RSISlope)

		// Base features
		currentFeatures := []float64{
			normalizeValue(closes[i], closes[i-params.WindowSize:i+1]),
			normalizeValue(ma50[i], ma50[i-params.WindowSize:i+1]),
			normalizeValue(ma200[i], ma200[i-params.WindowSize:i+1]),
			rsi14[i] / 100.0,
			rsi5[i] / 100.0, // Added short-term RSI
			rsiSlope / 100.0,
			normalizeValue(macd[i], macd[i-params.WindowSize:i+1]),
			normalizeValue(macdSignal[i], macdSignal[i-params.WindowSize:i+1]),
			normalizeValue(macdFast[i], macdFast[i-params.WindowSize:i+1]),
			normalizeValue(macdFastSignal[i], macdFastSignal[i-params.WindowSize:i+1]),
			normalizeValue(ma20[i], ma20[i-params.WindowSize:i+1]),
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

		rsiSlope := (rsi14[i] - rsi14[i-params.RSISlope]) / float64(params.RSISlope)

		// Base features
		currentFeatures := []float64{
			normalizeValue(closes[i], closes[i-params.WindowSize:i+1]),
			normalizeValue(ma50[i], ma50[i-params.WindowSize:i+1]),
			normalizeValue(ma200[i], ma200[i-params.WindowSize:i+1]),
			rsi14[i] / 100.0,
			rsi5[i] / 100.0, // Added short-term RSI
			rsiSlope / 100.0,
			normalizeValue(macd[i], macd[i-params.WindowSize:i+1]),
			normalizeValue(macdSignal[i], macdSignal[i-params.WindowSize:i+1]),
			normalizeValue(macdFast[i], macdFast[i-params.WindowSize:i+1]),
			normalizeValue(macdFastSignal[i], macdFastSignal[i-params.WindowSize:i+1]),
			normalizeValue(ma20[i], ma20[i-params.WindowSize:i+1]),
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
		}

		potentialGain := (highestHigh - basePrice) / basePrice
		potentialLoss := (basePrice - lowestLow) / basePrice
		actualChange := (closingPrice - basePrice) / basePrice

		// Enhanced signal generation with trend confirmation
		if potentialGain >= params.TakeProfit &&
			actualChange > 0 &&
			rsi14[i] > params.RSIUpperBound &&
			rsiSlope > 0 &&
			macd[i] > macdSignal[i] {
			label = StrategyLong
		} else if potentialLoss >= params.TakeProfit &&
			actualChange < 0 &&
			rsi14[i] < params.RSILowerBound &&
			rsiSlope < 0 &&
			macd[i] < macdSignal[i] {
			label = StrategyShort
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
