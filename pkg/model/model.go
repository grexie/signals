package model

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/grexie/signals/pkg/candles"
	"github.com/jedib0t/go-pretty/v6/progress"
	"github.com/syndtr/goleveldb/leveldb"
	"gorgonia.org/tensor"
)

type Model struct {
	weights    []tensor.Tensor
	db         *leveldb.DB
	params     ModelParams
	Instrument string
	Metrics    ModelMetrics
}

func NewModel(ctx context.Context, pw progress.Writer, db *leveldb.DB, instrument string, params ModelParams, now time.Time) (*Model, error) {
	to := now
	from := to.Add(-params.TrainDays)

	candles, err := candles.GetCandles(db, nil, instrument, candles.Network(Network()), from, to)
	if err != nil {
		return nil, err
	}

	if len(candles) == 0 {
		return nil, fmt.Errorf("no candle data received")
	}

	// Ensure we have enough candle data (at least 200 window + 5 for prediction)
	required := params.WindowSize + params.Candles
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

	if weights, err := Train(pw, params, trainingFeatures, trainingLabels, 100); err != nil {
		return nil, fmt.Errorf("training error: %v", err)
	} else {
		tracker := &progress.Tracker{
			Message: "Validation",
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

		if backtest, err := m.DeepBacktest(pw, instrument, params, to); err != nil {
			return nil, err
		} else {
			m.Metrics.Backtest = backtest
		}

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

func (m *Model) Predict(pw progress.Writer, feature []float64, now time.Time) ([]float64, Prediction, error) {
	if feature == nil {
		from := now.Truncate(time.Minute).Add(-time.Duration(WindowSize()*2) * time.Minute)
		candles, err := candles.GetCandles(m.db, pw, m.Instrument, candles.Network(Network()), from, now)
		if err != nil {
			return nil, nil, err
		}
		features := PrepareForPrediction(candles, m.params)
		feature = features[len(features)-1]
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
