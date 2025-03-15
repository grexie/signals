package model

import (
	"context"
	"fmt"
	"io"
	"log"
	"math"
	"os"
	"slices"
	"strconv"
	"time"

	"github.com/grexie/signals/pkg/candles"
	"github.com/jedib0t/go-pretty/v6/progress"
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/syndtr/goleveldb/leveldb"
	"gorgonia.org/tensor"
)

func envInt(name string, def func() int, dec func(v int) int) func() int {
	return func() int {
		value := def()
		if v, ok := os.LookupEnv(name); ok {
			if v, err := strconv.ParseInt(v, 10, 32); err != nil {
				log.Fatalf("failed to parse env.%s: %v", name, err)
			} else {
				value = int(v)
			}
		}
		return dec(value)
	}
}

func envFloat64(name string, def func() float64, dec func(v float64) float64) func() float64 {
	return func() float64 {
		value := def()
		if v, ok := os.LookupEnv(name); ok {
			if v, err := strconv.ParseFloat(v, 64); err != nil {
				log.Fatalf("failed to parse env.%s: %v", name, err)
			} else {
				value = v
			}
		}
		return dec(value)
	}
}

var (
	WindowSize = envInt("SIGNALS_WINDOW_SIZE", func() int {
		return DefaultModelParams.WindowSize
	}, BoundWindowSize)
	Candles = envInt("SIGNALS_CANDLES", func() int {
		return DefaultModelParams.StrategyCandles
	}, BoundCandles)
)

var (
	TakeProfit = envFloat64("SIGNALS_TAKE_PROFIT", func() float64 {
		return DefaultModelParams.StrategyLong * Leverage()
	}, BoundTakeProfit)
	StopLoss = envFloat64("SIGNALS_STOP_LOSS", func() float64 {
		return DefaultModelParams.StrategyHold * Leverage()
	}, BoundStopLoss)
	TradeMultiplier = envFloat64("SIGNALS_TRADE_MULTIPLIER", func() float64 {
		return 1.0
	}, func(v float64) float64 { return math.Max(0.5, math.Min(2, v)) })
	Leverage = envFloat64("SIGNALS_LEVERAGE", func() float64 {
		return 50.0
	}, func(v float64) float64 { return math.Max(1, math.Min(100, v)) })
	Commission = envFloat64("SIGNALS_COMMISSION", func() float64 {
		return DefaultModelParams.TradeCommission
	}, func(v float64) float64 { return math.Max(0, math.Min(0.5, v)) })
)

var (
	ShortMovingAverageLength   = envInt("SIGNALS_SHORT_MOVING_AVERAGE_LENGTH", func() int { return 50 }, BoundShortMovingAverageLength)
	LongMovingAverageLength    = envInt("SIGNALS_LONG_MOVING_AVERAGE_LENGTH", func() int { return 200 }, BoundLongMovingAverageLength)
	LongRSILength              = envInt("SIGNALS_LONG_RSI_LENGTH", func() int { return 14 }, BoundLongRSILength)
	ShortRSILength             = envInt("SIGNALS_SHORT_RSI_LENGTH", func() int { return 5 }, BoundShortRSILength)
	ShortMACDWindowLength      = envInt("SIGNALS_SHORT_MACD_WINDOW_LENGTH", func() int { return 12 }, BoundShortMACDWindowLength)
	LongMACDWindowLength       = envInt("SIGNALS_LONG_MACD_WINDOW_LENGTH", func() int { return 26 }, BoundLongMACDWindowLength)
	MACDSignalWindow           = envInt("SIGNALS_MACD_SIGNAL_WINDOW", func() int { return 9 }, BoundMACDSignalWindow)
	FastShortMACDWindowLength  = envInt("SIGNALS_FAST_SHORT_MACD_WINDOW_LENGTH", func() int { return 5 }, BoundFastShortMACDWindowLength)
	FastLongMACDWindowLength   = envInt("SIGNALS_FAST_LONG_MACD_WINDOW_LENGTH", func() int { return 35 }, BoundFastLongMACDWindowLength)
	FastMACDSignalWindow       = envInt("SIGNALS_FAST_MACD_SIGNAL_WINDOW", func() int { return 5 }, BoundFastMACDSignalWindow)
	BollingerBandsWindow       = envInt("SIGNALS_BOLLINGER_BANDS_WINDOW", func() int { return 20 }, BoundBollingerBandsWindow)
	BollingerBandsMultiplier   = envFloat64("SIGNALS_BOLLINGER_BANDS_MULTIPLIER", func() float64 { return 2.0 }, BoundBollingerBandsMultiplier)
	StochasticOscillatorWindow = envInt("SIGNALS_STOCHASTIC_OSCILLATOR_WINDOW", func() int { return 14 }, BoundStochasticOscillatorWindow)
	SlowATRPeriod              = envInt("SIGNALS_SLOW_ATR_PERIOD_WINDOW", func() int { return 14 }, BoundSlowATRPeriod)
	FastATRPeriod              = envInt("SIGNALS_FAST_ATR_PERIOD_WINDOW", func() int { return 20 }, BoundFastATRPeriod)
	OBVMovingAverageLength     = envInt("SIGNALS_OBV_MOVING_AVERAGE_LENGTH", func() int { return 20 }, BoundOBVMovingAverageLength)
	VolumesMovingAverageLength = envInt("SIGNALS_VOLUMES_MOVING_AVERAGE_LENGTH", func() int { return 20 }, BoundVolumesMovingAverageLength)
	ChaikinMoneyFlowPeriod     = envInt("SIGNALS_CHAIKIN_MONEY_FLOW_PERIOD", func() int { return 20 }, BoundChaikinMoneyFlowPeriod)
	MoneyFlowIndexPeriod       = envInt("SIGNALS_MONEY_FLOW_INDEX_PERIOD", func() int { return 14 }, BoundMoneyFlowIndexPeriod)
	RateOfChangePeriod         = envInt("SIGNALS_RATE_OF_CHANGE_PERIOD", func() int { return 14 }, BoundRateOfChangePeriod)
	CCIPeriod                  = envInt("SIGNALS_CCI_PERIOD", func() int { return 20 }, BoundCCIPeriod)
	WilliamsRPeriod            = envInt("SIGNALS_WILLIAMS_R_PERIOD", func() int { return 14 }, BoundWilliamsRPeriod)
	PriceChangeFastPeriod      = envInt("SIGNALS_PRICE_CHANGE_FAST_PERIOD", func() int { return 60 }, BoundPriceChangeFastPeriod)
	PriceChangeMediumPeriod    = envInt("SIGNALS_PRICE_CHANGE_MEDIUM_PERIOD", func() int { return 240 }, BoundPriceChangeMediumPeriod)
	PriceChangeSlowPeriod      = envInt("SIGNALS_PRICE_CHANGE_SLOW_PERIOD", func() int { return 1440 }, BoundPriceChangeSlowPeriod)
	RSIUpperBound              = envFloat64("SIGNALS_RSI_UPPER_BOUND", func() float64 { return 50.0 }, BoundRSIUpperBound)
	RSILowerBound              = envFloat64("SIGNALS_RSI_LOWER_BOUND", func() float64 { return 50.0 }, BoundRSILowerBound)
	RSISlope                   = envInt("SIGNALS_RSI_SLOPE", func() int { return 3 }, BoundRSISlope)
)

type ModelMetrics struct {
	Accuracy        float64
	ConfusionMatrix [][]float64
	ClassPrecision  []float64
	ClassRecall     []float64
	F1Scores        []float64

	Samples []int

	PnL          float64
	MaxDrawdown  float64
	SharpeRatio  float64
	SortinoRatio float64
	Trades       int
}

func (m *ModelMetrics) Fitness() float64 {
	avgF1 := (m.F1Scores[0] + m.F1Scores[1] + m.F1Scores[2]) / 3
	drawdownPenalty := 1 / (1 + m.MaxDrawdown/100) // Penalizes high drawdowns

	fitness := (avgF1*0.15 + m.SortinoRatio*0.3 + m.SharpeRatio*0.2 + m.PnL*0.15) * drawdownPenalty
	return fitness
}

type Model struct {
	weights    []tensor.Tensor
	db         *leveldb.DB
	params     ModelParams
	Instrument string
	Metrics    ModelMetrics
}

func (m ModelMetrics) Write(w io.Writer) error {
	t := table.NewWriter()
	t.SetOutputMirror(w)
	t.SetTitle("Confusion Matrix")
	t.AppendHeader(table.Row{"", "HOLD", "LONG", "SHORT"})
	for i := range 3 {
		var label string
		switch i {
		case 0:
			label = "HOLD"
		case 1:
			label = "LONG"
		case 2:
			label = "SHORT"
		}

		rowTotal := float64(m.ConfusionMatrix[i][0] + m.ConfusionMatrix[i][1] + m.ConfusionMatrix[i][2])
		holdPercent := float64(m.ConfusionMatrix[i][0]) / rowTotal * 100
		longPercent := float64(m.ConfusionMatrix[i][1]) / rowTotal * 100
		shortPercent := float64(m.ConfusionMatrix[i][2]) / rowTotal * 100

		if rowTotal == 0 {
			t.AppendRows([]table.Row{
				{label, "", "", ""},
			})
		} else {
			t.AppendRows([]table.Row{
				{label, fmt.Sprintf("%6.2f%%", holdPercent), fmt.Sprintf("%6.2f%%", longPercent), fmt.Sprintf("%6.2f%%", shortPercent)},
			})
		}

	}
	t.AppendFooter(table.Row{"ACCURACY", "", "", fmt.Sprintf("%0.02f%%", m.Accuracy)})

	t.Render()

	t = table.NewWriter()
	t.SetOutputMirror(w)
	t.SetTitle("Class Metrics")
	t.AppendHeader(table.Row{"CLASS", "PRECISION", "RECALL", "F1 SCORE", "SAMPLES"})
	t.AppendRows([]table.Row{
		{"HOLD", fmt.Sprintf("%6.2f%%", m.ClassPrecision[0]), fmt.Sprintf("%6.2f%%", m.ClassRecall[0]), fmt.Sprintf("%6.2f%%", m.F1Scores[0]), fmt.Sprintf("%d", m.Samples[0])},
		{"LONG", fmt.Sprintf("%6.2f%%", m.ClassPrecision[1]), fmt.Sprintf("%6.2f%%", m.ClassRecall[1]), fmt.Sprintf("%6.2f%%", m.F1Scores[1]), fmt.Sprintf("%d", m.Samples[1])},
		{"SHORT", fmt.Sprintf("%6.2f%%", m.ClassPrecision[2]), fmt.Sprintf("%6.2f%%", m.ClassRecall[2]), fmt.Sprintf("%6.2f%%", m.F1Scores[2]), fmt.Sprintf("%d", m.Samples[2])},
	})
	t.AppendSeparator()
	t.AppendRows([]table.Row{
		{"", fmt.Sprintf("%6.2f%%", (m.ClassPrecision[0]+m.ClassPrecision[1]+m.ClassPrecision[2])/3), fmt.Sprintf("%6.2f%%", (m.ClassRecall[0]+m.ClassRecall[1]+m.ClassRecall[2])/3), fmt.Sprintf("%6.2f%%", (m.F1Scores[0]+m.F1Scores[1]+m.F1Scores[2])/3), fmt.Sprintf("%d", m.Samples[0]+m.Samples[1]+m.Samples[2])},
	})
	t.Render()

	t = table.NewWriter()
	t.SetOutputMirror(w)
	t.SetTitle("Trading Metrics")
	t.AppendRows([]table.Row{
		{"PnL", fmt.Sprintf("%6.2f%%", m.PnL)},
		{"Max Drawdown", fmt.Sprintf("%6.2f%%", m.MaxDrawdown)},
		{"Sharpe Ratio", fmt.Sprintf("%6.2f", m.SharpeRatio)},
		{"Sortino Ratio", fmt.Sprintf("%6.2f", m.SortinoRatio)},
		{"Trades", fmt.Sprintf("%d", m.Trades)},
	})
	t.Render()

	return nil
}

func calculateMetrics(confusionMatrix [][]int, total int) ModelMetrics {
	numClasses := len(confusionMatrix)
	metrics := ModelMetrics{
		ConfusionMatrix: make([][]float64, numClasses),
		ClassPrecision:  make([]float64, numClasses),
		ClassRecall:     make([]float64, numClasses),
		F1Scores:        make([]float64, numClasses),
		Samples:         make([]int, numClasses),
	}

	// Calculate confusion matrix percentages
	classTotals := make([]int, numClasses)
	for i := range numClasses {
		metrics.ConfusionMatrix[i] = make([]float64, numClasses)
		for j := 0; j < numClasses; j++ {
			classTotals[i] += confusionMatrix[i][j]
		}
		for j := 0; j < numClasses; j++ {
			if classTotals[i] > 0 {
				metrics.ConfusionMatrix[i][j] = float64(confusionMatrix[i][j]) / float64(classTotals[i]) * 100
			}
		}
		metrics.Samples[i] = confusionMatrix[i][i]
	}

	// Calculate precision and recall for each class
	for i := 0; i < numClasses; i++ {
		truePositives := confusionMatrix[i][i]
		falsePositives := 0
		falseNegatives := 0

		for j := 0; j < numClasses; j++ {
			if i != j {
				falsePositives += confusionMatrix[j][i]
				falseNegatives += confusionMatrix[i][j]
			}
		}

		// Calculate precision
		if truePositives+falsePositives > 0 {
			metrics.ClassPrecision[i] = float64(truePositives) / float64(truePositives+falsePositives) * 100
		}

		// Calculate recall
		if truePositives+falseNegatives > 0 {
			metrics.ClassRecall[i] = float64(truePositives) / float64(truePositives+falseNegatives) * 100
		}

		// Calculate F1 score
		if metrics.ClassPrecision[i]+metrics.ClassRecall[i] > 0 {
			metrics.F1Scores[i] = 2 * (metrics.ClassPrecision[i] * metrics.ClassRecall[i]) /
				(metrics.ClassPrecision[i] + metrics.ClassRecall[i])
		}
	}

	// Calculate overall accuracy
	correct := 0
	for i := range numClasses {
		correct += confusionMatrix[i][i]
	}
	metrics.Accuracy = float64(correct) / float64(total) * 100

	return metrics
}

func NewModel(ctx context.Context, pw progress.Writer, db *leveldb.DB, instrument string, params ModelParams, from time.Time, to time.Time, fetch bool) (*Model, error) {
	candles := candles.GetCandles(db, pw, instrument, candles.OKX, from, to)

	if len(candles) == 0 {
		return nil, fmt.Errorf("no candle data received")
	}

	// Ensure we have enough candle data (at least 200 window + 5 for prediction)
	required := params.WindowSize + params.StrategyCandles
	if len(candles) < required {
		return nil, fmt.Errorf("insufficient candle data: need at least %d candles, got %d", required, len(candles))
	}

	features, labels := Prepare(
		pw,
		candles,
		params,
	)

	countTraining := int(float64(len(features)) * 0.8)
	trainingFeatures := features[:countTraining]
	trainingLabels := labels[:countTraining]
	testingFeatures := features[countTraining:]
	testingLabels := labels[countTraining:]

	if weights, err := Train(pw, trainingFeatures, trainingLabels, 100); err != nil {
		return nil, fmt.Errorf("training error: %v", err)
	} else {
		tracker := &progress.Tracker{
			Message: "Testing",
			Total:   int64(len(testingFeatures)),
			Units:   progress.UnitsDefault,
		}
		pw.AppendTracker(tracker)
		tracker.Start()

		confusionMatrix := make([][]int, 3)
		for i := range confusionMatrix {
			confusionMatrix[i] = make([]int, 3)
		}

		correct := 0
		total := len(testingFeatures)
		predictions := make([]int, total)

		for i, features := range testingFeatures {
			pred, err := Predict(weights, features)
			tracker.Increment(1)
			if err != nil {
				log.Printf("prediction error for sample %d: %v", i, err)
				continue
			}

			predictedClass := argmax(pred)
			actualClass := int(testingLabels[i])

			predictions[i] = predictedClass
			confusionMatrix[actualClass][predictedClass]++
			if predictedClass == actualClass {
				correct++
			}
		}
		tracker.MarkAsDone()

		// Calculate detailed metrics
		metrics := calculateMetrics(confusionMatrix, total)

		m := &Model{
			weights:    weights,
			db:         db,
			params:     params,
			Instrument: instrument,
			Metrics:    metrics,
		}

		backtestingCandles := []int{}
		end := to.Truncate(time.Minute)
		start := end.AddDate(0, 0, -7)

		for i, candle := range candles {
			if candle.Timestamp.After(start) && candle.Timestamp.Before(end) {
				backtestingCandles = append(backtestingCandles, i)
			}
		}
		slices.SortFunc(backtestingCandles, func(a, b int) int {
			return candles[a].Timestamp.Compare(candles[b].Timestamp)
		})

		tracker = &progress.Tracker{
			Message: "Backtesting for PnL",
			Total:   int64(len(backtestingCandles)),
			Units:   progress.UnitsDefault,
		}
		pw.AppendTracker(tracker)
		tracker.Start()
		trader := NewPaperTrader(10000, params.StrategyHold, params.StrategyLong, params.TradeCommission/2, Leverage())

		for _, i := range backtestingCandles {
			trader.Iterate(candles[i], func(c Candle) Strategy {
				features := PrepareForPrediction(candles[i-params.WindowSize*2:i+1], params)
				pred, err := Predict(m.weights, features)
				if err != nil {
					log.Println("prediction error:", err)
					return StrategyHold
				}

				prediction := argmax(pred)
				return Strategy(prediction)
			})
			tracker.Increment(1)
		}

		m.Metrics.PnL = trader.PnL()
		m.Metrics.MaxDrawdown = trader.MaxDrawdown()
		m.Metrics.SharpeRatio = trader.SharpeRatio(0.0)
		m.Metrics.Trades = len(trader.ClosedTrades)
		tracker.MarkAsDone()

		return m, nil
	}
}

func argmax(slice []float64) int {
	maxIndex := 0
	maxValue := slice[0]
	for i, value := range slice {
		if value > maxValue {
			maxValue = value
			maxIndex = i
		}
	}
	return maxIndex
}

type Prediction map[Strategy]float64

func (m *Model) Predict(pw progress.Writer, feature []float64, now time.Time, fetch bool) ([]float64, Prediction, error) {
	if feature == nil {
		from := now.Truncate(time.Minute).Add(-time.Duration(WindowSize()*2) * time.Minute)
		candles := candles.GetCandles(m.db, pw, m.Instrument, candles.OKX, from, now)
		feature = PrepareForPrediction(candles, m.params)
	}

	pred, err := Predict(m.weights, feature)
	if err != nil {
		return nil, nil, err
	}

	prediction := Prediction{}
	for i := range 3 {
		prediction[Strategy(i)] = pred[i]
	}
	return feature, prediction, nil
}
