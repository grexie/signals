package model

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"math"
	"os"
	"strconv"
	"time"

	"github.com/grexie/signals/pkg/market"
	"github.com/jedib0t/go-pretty/table"
	"github.com/jedib0t/go-pretty/v6/progress"
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
	}, func(v float64) float64 {
		return BoundTakeProfit(v/Leverage()) * Leverage()
	})
	StopLoss = envFloat64("SIGNALS_STOP_LOSS", func() float64 {
		return DefaultModelParams.StrategyHold * Leverage()
	}, func(v float64) float64 {
		return BoundStopLoss(v/Leverage()) * Leverage()
	})
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
)

type ModelMetrics struct {
	Accuracy        float64
	ConfusionMatrix [][]float64
	ClassPrecision  []float64
	ClassRecall     []float64
	F1Scores        []float64
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
	t.AppendHeader(table.Row{"CLASS", "PRECISION", "RECALL", "F1 SCORE"})
	t.AppendRows([]table.Row{
		{"HOLD", fmt.Sprintf("%6.2f%%", m.ClassPrecision[0]), fmt.Sprintf("%6.2f%%", m.ClassRecall[0]), fmt.Sprintf("%6.2f%%", m.F1Scores[0])},
		{"LONG", fmt.Sprintf("%6.2f%%", m.ClassPrecision[1]), fmt.Sprintf("%6.2f%%", m.ClassRecall[1]), fmt.Sprintf("%6.2f%%", m.F1Scores[1])},
		{"SHORT", fmt.Sprintf("%6.2f%%", m.ClassPrecision[2]), fmt.Sprintf("%6.2f%%", m.ClassRecall[2]), fmt.Sprintf("%6.2f%%", m.F1Scores[2])},
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
	}

	// Calculate confusion matrix percentages
	classTotals := make([]int, numClasses)
	for i := 0; i < numClasses; i++ {
		metrics.ConfusionMatrix[i] = make([]float64, numClasses)
		for j := 0; j < numClasses; j++ {
			classTotals[i] += confusionMatrix[i][j]
		}
		for j := 0; j < numClasses; j++ {
			if classTotals[i] > 0 {
				metrics.ConfusionMatrix[i][j] = float64(confusionMatrix[i][j]) / float64(classTotals[i]) * 100
			}
		}
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
	ctx, ch := market.FetchCandles(ctx, pw, db, instrument, from.Truncate(time.Minute), to.Truncate(time.Minute), market.CandleBar1m, fetch)

	var candles []Candle
outer:
	for {
		select {
		case candle, ok := <-ch:
			if !ok {
				break outer
			}
			candles = append(candles, candle)
		case <-ctx.Done():
			if !errors.Is(ctx.Err(), context.Canceled) {
				return nil, fmt.Errorf("context error: %v", ctx.Err())
			}
			break outer
		}
	}

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
		tracker := progress.Tracker{
			Message: "Testing",
			Total:   int64(len(testingFeatures)),
			Units:   progress.UnitsDefault,
		}
		pw.AppendTracker(&tracker)
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

		return &Model{
			weights:    weights,
			db:         db,
			params:     params,
			Instrument: instrument,
			Metrics:    metrics,
		}, nil
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

func (m *Model) Predict(ctx context.Context, feature []float64, now time.Time, fetch bool) ([]float64, Strategy, error) {
	if feature == nil {
		from := now.Truncate(time.Minute).Add(-400 * time.Minute)
		ctx, ch := market.FetchCandles(context.Background(), nil, nil, m.Instrument, from, now, market.CandleBar1m, fetch)

		var candles []Candle
	outer:
		for {
			select {
			case candle, ok := <-ch:
				if !ok {
					break outer
				}
				candles = append(candles, candle)
			case <-ctx.Done():
				if !errors.Is(ctx.Err(), context.Canceled) {
					return nil, StrategyHold, fmt.Errorf("context error: %v", ctx.Err())
				}
				break outer
			}
		}

		feature = PrepareForPrediction(candles, m.params)
	}

	pred, err := Predict(m.weights, feature)
	if err != nil {
		return nil, StrategyHold, err
	}

	predictedClass := argmax(pred)
	return feature, Strategy(predictedClass), nil
}
