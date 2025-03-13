package model

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/grexie/signals/pkg/market"
	"github.com/jedib0t/go-pretty/table"
	"github.com/jedib0t/go-pretty/v6/progress"
	"github.com/syndtr/goleveldb/leveldb"
	"gorgonia.org/tensor"
)

var (
	Candles = func() int {
		ca := 1
		if c, ok := os.LookupEnv("SIGNALS_CANDLES"); ok {
			if c, err := strconv.ParseInt(c, 10, 32); err != nil {
				log.Fatalf("failed to parse env.SIGNALS_CANDLES: %s", err)
			} else {
				ca = int(c)
			}
		}
		return ca
	}
)

var (
	TakeProfit = func() float64 {
		tp := 0.4
		if t, ok := os.LookupEnv("SIGNALS_TAKE_PROFIT"); ok {
			if t, err := strconv.ParseFloat(t, 64); err != nil {
				log.Fatalf("failed to parse env.SIGNALS_TAKE_PROFIT: %s", err)
			} else {
				tp = t
			}
		}
		return tp
	}
	StopLoss = func() float64 {
		sl := 0.1
		if s, ok := os.LookupEnv("SIGNALS_STOP_LOSS"); ok {
			if s, err := strconv.ParseFloat(s, 64); err != nil {
				log.Fatalf("failed to parse env.SIGNALS_STOP_LOSS: %s", err)
			} else {
				sl = s
			}
		}
		return sl
	}
	TradeMultiplier = func() float64 {
		tm := 1.0
		if t, ok := os.LookupEnv("SIGNALS_TRADE_MULTIPLIER"); ok {
			if t, err := strconv.ParseFloat(t, 64); err != nil {
				log.Fatalf("failed to parse env.SIGNALS_TRADE_MULTIPLIER: %s", err)
			} else {
				tm = t
			}
		}
		return tm
	}
	Leverage = func() float64 {
		lever := 50.0
		if l, ok := os.LookupEnv("SIGNALS_LEVERAGE"); ok {
			if l, err := strconv.ParseFloat(l, 64); err != nil {
				log.Fatalf("failed to parse env.SIGNALS_LEVERAGE: %s", err)
			} else {
				lever = l
			}
		}
		return lever
	}
	Commission = func() float64 {
		co := 0.001
		if c, ok := os.LookupEnv("SIGNALS_COMMISSION"); ok {
			if c, err := strconv.ParseFloat(c, 64); err != nil {
				log.Fatalf("failed to parse env.SIGNALS_COMMISSION: %s", err)
			} else {
				co = c
			}
		}
		return co
	}
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

func NewModel(ctx context.Context, pw progress.Writer, db *leveldb.DB, instrument string, from time.Time, to time.Time) (*Model, error) {
	ctx, ch := market.FetchCandles(ctx, pw, db, instrument, from.Truncate(time.Minute), to.Truncate(time.Minute), market.CandleBar1m)

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

	numCandles := Candles()
	tp, sl := TakeProfit(), StopLoss()
	commission := Commission()
	leverage := Leverage()

	// Ensure we have enough candle data (at least 200 window + 5 for prediction)
	required := 200 + numCandles
	if len(candles) < required {
		return nil, fmt.Errorf("insufficient candle data: need at least %d candles, got %d", required, len(candles))
	}

	features, labels := Prepare(
		pw,
		candles,
		GorgoniaParams{
			WindowSize:      200,
			StrategyCandles: numCandles,
			StrategyLong:    tp / leverage,
			StrategyShort:   tp / leverage,
			StrategyHold:    sl / leverage,
			TradeCommission: commission * leverage,
		},
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

func (m *Model) Predict(ctx context.Context, feature []float64, now time.Time) ([]float64, Strategy, error) {
	if feature == nil {
		from := now.Truncate(time.Minute).Add(-400 * time.Minute)
		ctx, ch := market.FetchCandles(context.Background(), nil, nil, m.Instrument, from, now, market.CandleBar1m)

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

		tp, sl := TakeProfit(), StopLoss()
		commission := Commission()
		leverage := Leverage()

		feature = PrepareForPrediction(candles, GorgoniaParams{
			WindowSize:      200,
			StrategyLong:    tp / leverage,
			StrategyShort:   tp / leverage,
			StrategyHold:    sl / leverage,
			TradeCommission: commission * leverage,
		})
	}

	pred, err := Predict(m.weights, feature)
	if err != nil {
		return nil, StrategyHold, err
	}

	predictedClass := argmax(pred)
	return feature, Strategy(predictedClass), nil
}
