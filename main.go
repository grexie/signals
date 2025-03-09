package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math"
	"os"
	"sort"
	"strconv"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/grexie/signals/pkg/db"
	"github.com/jedib0t/go-pretty/table"
	"github.com/jedib0t/go-pretty/v6/progress"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"gorgonia.org/gorgonia"
	"gorgonia.org/tensor"
)

const (
	TakeProfit = 0.2
	StopLoss   = 0.1
	Leverage   = 50
)

func main() {
	pw := progress.NewWriter()
	pw.SetMessageLength(30)
	pw.SetNumTrackersExpected(1)
	pw.SetSortBy(progress.SortByPercentDsc)
	pw.SetStyle(progress.StyleDefault)
	pw.SetTrackerLength(15)
	pw.SetTrackerPosition(progress.PositionRight)
	pw.SetUpdateFrequency(time.Millisecond * 100)
	pw.Style().Colors = progress.StyleColorsExample
	pw.Style().Options.PercentFormat = "%4.1f%%"
	go pw.Render()

	db, err := db.ConnectMongo()
	if err != nil {
		log.Fatalf("Failed to connect to MongoDB: %v", err)
	}

	ctx, ch := FetchMarketCandles(context.Background(), pw, db, "DOGE-USDT-SWAP", time.Now().AddDate(0, -3, 0), time.Now(), CandleBar1m)

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
				log.Fatalf("Context error: %v", ctx.Err())
			}
			break outer
		}
	}

	features, labels := PrepareDataForGorgonia(
		pw,
		candles,
		GorgoniaParams{
			WindowSize:    200,
			StrategyLong:  TakeProfit / Leverage,
			StrategyShort: TakeProfit / Leverage,
			StrategyHold:  StopLoss / Leverage,
		},
	)

	countTraining := int(float64(len(features)) * 0.8)
	trainingFeatures := features[:countTraining]
	trainingLabels := labels[:countTraining]
	testingFeatures := features[countTraining:]
	testingLabels := labels[countTraining:]

	if weights, err := BuildAndTrainNN(pw, trainingFeatures, trainingLabels, 100); err != nil {
		log.Fatalf("Training error: %v", err)
	} else {
		tracker := progress.Tracker{
			Message: "Testing",
			Total:   int64(len(testingFeatures)),
			Units:   progress.UnitsDefault,
		}
		pw.AppendTracker(&tracker)
		tracker.Start()

		confusion := make([][]int, 3)
		for i := range confusion {
			confusion[i] = make([]int, 3)
		}
		for i := range len(testingFeatures) {
			tracker.SetValue(int64(i))
			if value, err := Predict(weights, testingFeatures[i]); err != nil {
				log.Fatalf("Prediction error: %v", err)
			} else {
				predictedClass := argmax(value)
				actualClass := int(testingLabels[i])

				confusion[predictedClass][actualClass]++
			}
		}
		tracker.MarkAsDone()
		pw.Stop()
		for pw.IsRenderInProgress() {
			time.Sleep(100 * time.Millisecond)
		}

		t := table.NewWriter()
		t.SetOutputMirror(os.Stdout)
		t.AppendHeader(table.Row{"", "HOLD", "SHORT", "LONG"})
		t.AppendRows([]table.Row{
			{"HOLD", confusion[0][0], confusion[0][1], confusion[0][2]},
			{"SHORT", confusion[1][0], confusion[1][1], confusion[1][2]},
			{"LONG", confusion[2][0], confusion[2][1], confusion[2][2]},
		})
		t.AppendFooter(table.Row{"ACCURACY", "", "", fmt.Sprintf("%0.02f%%", (100.0*float64(confusion[0][0]+confusion[1][1]+confusion[2][2]))/float64(len(testingFeatures)))})
		t.Render()
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

type CandleData [][]string

type Candle struct {
	ID         primitive.ObjectID `bson:"_id"`
	Timestamp  time.Time          `bson:"timestamp"`
	Instrument string             `bson:"instrument"`
	Network    string             `bson:"network"`
	Open       float64            `bson:"open"`
	High       float64            `bson:"high"`
	Low        float64            `bson:"low"`
	Close      float64            `bson:"close"`
	Volume     float64            `bson:"volume"`
}

type CandleBar string

const (
	CandleBar1s  CandleBar = "1s"
	CandleBar1m  CandleBar = "1m"
	CandleBar5m  CandleBar = "5m"
	CandleBar15m CandleBar = "15m"
	CandleBar1h  CandleBar = "1h"
)

func CandleBarToDuration(bar CandleBar) time.Duration {
	switch bar {
	case CandleBar1s:
		return time.Second
	case CandleBar1m:
		return time.Minute
	case CandleBar5m:
		return 5 * time.Minute
	case CandleBar15m:
		return 15 * time.Minute
	case CandleBar1h:
		return time.Hour
	default:
		return time.Minute
	}
}

func NewCandlesFromData(instrument string, data [][]string) ([]Candle, error) {
	out := make([]Candle, len(data))

	for i, candle := range data {
		if len(candle) < 6 {
			return nil, fmt.Errorf("invalid candle data: %v", candle)
		}

		if timestamp, err := strconv.ParseInt(candle[0], 10, 64); err != nil {
			return nil, err
		} else if open, err := strconv.ParseFloat(candle[1], 64); err != nil {
			return nil, err
		} else if high, err := strconv.ParseFloat(candle[2], 64); err != nil {
			return nil, err
		} else if low, err := strconv.ParseFloat(candle[3], 64); err != nil {
			return nil, err
		} else if close, err := strconv.ParseFloat(candle[4], 64); err != nil {
			return nil, err
		} else if volume, err := strconv.ParseFloat(candle[5], 64); err != nil {
			return nil, err
		} else {
			out[i] = Candle{
				ID:         primitive.NewObjectID(),
				Timestamp:  time.UnixMilli(timestamp),
				Instrument: instrument,
				Network:    "okx",
				Open:       open,
				High:       high,
				Low:        low,
				Close:      close,
				Volume:     volume,
			}
		}
	}

	sort.Slice(out, func(i, j int) bool {
		return out[i].Timestamp.Before(out[j].Timestamp)
	})

	return out, nil
}

func FetchMarketCandles(ctx context.Context, pw progress.Writer, mdb *mongo.Database, instrument string, start time.Time, end time.Time, bar CandleBar) (context.Context, chan Candle) {
	client := resty.New()
	candles := 50
	out := make(chan Candle, candles*100)
	ctx, cancel := context.WithCancelCause(ctx)

	url := "https://www.okx.com/api/v5/market/history-candles"

	go func() {
		defer close(out)
		defer cancel(nil)

		if err := db.EnsureIndex(mdb, ctx, "candles", mongo.IndexModel{
			Keys: bson.D{
				bson.E{Key: "instrument", Value: 1},
				bson.E{Key: "network", Value: 1},
				bson.E{Key: "timestamp", Value: 1},
			},
			Options: options.Index().SetName("candles").SetUnique(true),
		}); err != nil {
			cancel(err)
			return
		}

		tracker := &progress.Tracker{
			Message: "Fetching candles from cache",
			Units:   progress.UnitsDefault,
		}

		if count, err := mdb.Collection("candles").CountDocuments(ctx, bson.M{
			"instrument": instrument,
			"network":    "okx",
			"timestamp": bson.M{
				"$gte": start,
				"$lte": end,
			},
		}); err != nil {
			log.Println("failed to count candles in database:", err)
			cancel(err)
			return
		} else if count > 0 {
			tracker.Total = count
			pw.AppendTracker(tracker)
			tracker.Start()
		}

		if cursor, err := mdb.Collection("candles").Find(ctx, bson.M{
			"instrument": instrument,
			"network":    "okx",
			"timestamp": bson.M{
				"$gte": start,
				"$lte": end,
			},
		}, options.Find().SetSort(bson.M{"timestamp": 1})); err != nil {
			log.Println("failed to fetch candles from database:", err)
			cancel(err)
			return
		} else {
			for cursor.Next(ctx) {
				var candle Candle
				if err := cursor.Decode(&candle); err != nil {
					cancel(err)
					return
				}

				tracker.Increment(1)
				start = candle.Timestamp

				select {
				case out <- candle:
				case <-ctx.Done():
					return
				}
			}
		}
		tracker.MarkAsDone()

		duration := time.Duration(candles) * CandleBarToDuration(bar)

		tracker = &progress.Tracker{
			Message: "Fetching candles from API",
			Units:   progress.UnitsDefault,
			Total:   int64((end.Sub(start) / duration) + 1),
		}
		pw.AppendTracker(tracker)
		tracker.Start()

		for ; start.Before(end); start = start.Add(duration) {
			params := map[string]string{
				"instId": instrument,
				"bar":    string(bar),
				"limit":  fmt.Sprintf("%d", candles),
				"after":  fmt.Sprintf("%d", start.Add(duration).UnixMilli()),
				"before": fmt.Sprintf("%d", start.UnixMilli()),
			}

			requested := time.Now()

			if resp, err := client.R().SetContext(ctx).SetQueryParams(params).Get(url); err != nil {
				cancel(err)
				return
			} else if resp.IsError() {
				cancel(fmt.Errorf("error response: %v", resp.Status()))
				return
			} else {
				var data struct {
					Code string     `json:"code"`
					Msg  string     `json:"msg"`
					Data CandleData `json:"data"`
				}

				if err := json.Unmarshal(resp.Body(), &data); err != nil {
					cancel(fmt.Errorf("failed to parse response body: %s", err))
					return
				} else if data.Code != "0" {
					cancel(fmt.Errorf("API Error: %s", data.Msg))
					return
				} else if candles, err := NewCandlesFromData(instrument, data.Data); err != nil {
					cancel(fmt.Errorf("failed to convert data to candles: %s", err))
					return
				} else {
					for _, candle := range candles {
						tracker.Increment(1)
						if _, err := mdb.Collection("candles").InsertOne(ctx, candle); err != nil {
							cancel(err)
							return
						}
						select {
						case out <- candle:
						case <-ctx.Done():
							return
						}
					}
				}
			}

			time.Sleep(time.Until(requested.Add(200 * time.Millisecond)))
		}

		tracker.MarkAsDone()
	}()

	return ctx, out
}

func MovingAverage(prices []float64, window int) []float64 {
	ma := make([]float64, len(prices))
	for i := range prices {
		if i < window {
			ma[i] = 0
			continue
		}
		sum := 0.0
		for j := 0; j < window; j++ {
			sum += prices[i-j]
		}
		ma[i] = sum / float64(window)
	}
	return ma
}

func RSI(prices []float64, window int) []float64 {
	rsi := make([]float64, len(prices))
	for i := range prices {
		if i < window {
			continue
		}
		gains, losses := 0.0, 0.0
		for j := 0; j < window; j++ {
			change := prices[i-j] - prices[i-j-1]
			if change > 0 {
				gains += change
			} else {
				losses -= change
			}
		}
		avgGain := gains / float64(window)
		avgLoss := losses / float64(window)
		if avgLoss == 0 {
			rsi[i] = 100
		} else {
			rs := avgGain / avgLoss
			rsi[i] = 100 - (100 / (1 + rs))
		}
	}
	return rsi
}

func MACD(prices []float64, shortWindow, longWindow, signalWindow int) ([]float64, []float64) {
	shortMA := MovingAverage(prices, shortWindow)
	longMA := MovingAverage(prices, longWindow)
	macd := make([]float64, len(prices))
	var signal []float64

	for i := range prices {
		macd[i] = shortMA[i] - longMA[i]
	}
	signal = MovingAverage(macd, signalWindow)

	return macd, signal
}

func BollingerBands(prices []float64, window int, multiplier float64) ([]float64, []float64, []float64) {
	ma := MovingAverage(prices, window)
	upper, lower := make([]float64, len(prices)), make([]float64, len(prices))

	for i := range prices {
		if i < window {
			upper[i], lower[i] = 0, 0
			continue
		}
		sum := 0.0
		for j := i - window + 1; j <= i; j++ {
			sum += math.Pow(prices[j]-ma[i], 2)
		}
		stdDev := math.Sqrt(sum / float64(window))
		upper[i] = ma[i] + multiplier*stdDev
		lower[i] = ma[i] - multiplier*stdDev
	}
	return ma, upper, lower
}

func StochasticOscillator(closes, lows, highs []float64, window int) ([]float64, []float64) {
	kValues := make([]float64, len(closes))
	dValues := make([]float64, len(closes))

	for i := range closes {
		if i < window {
			kValues[i], dValues[i] = 0, 0
			continue
		}
		low, high := lows[i], highs[i]
		for j := i - window + 1; j <= i; j++ {
			low = math.Min(low, lows[j])
			high = math.Max(high, highs[j])
		}
		kValues[i] = 100 * (closes[i] - low) / (high - low)
	}

	// Calculate %D as a 3-period moving average of %K
	dValues = MovingAverage(kValues, 3)

	return kValues, dValues
}

func VWAP(closes, volumes []float64) []float64 {
	vwap := make([]float64, len(closes))
	cumulativeVolume, cumulativeValue := 0.0, 0.0

	for i := range closes {
		cumulativeVolume += volumes[i]
		cumulativeValue += closes[i] * volumes[i]
		if cumulativeVolume != 0 {
			vwap[i] = cumulativeValue / cumulativeVolume
		} else {
			vwap[i] = 0
		}
	}
	return vwap
}

type Strategy float64

const (
	StrategyHold  Strategy = 0
	StrategyLong  Strategy = 1
	StrategyShort Strategy = 2
)

type GorgoniaParams struct {
	WindowSize    int
	StrategyLong  float64
	StrategyShort float64
	StrategyHold  float64
}

func PrepareDataForGorgonia(pw progress.Writer, candles []Candle, params GorgoniaParams) ([][]float64, []float64) {
	tracker := progress.Tracker{
		Message: "Preparing data",
		Total:   int64(len(candles)),
		Units:   progress.UnitsDefault,
	}
	pw.AppendTracker(&tracker)
	tracker.Start()

	features := [][]float64{}
	labels := []float64{}

	closes := make([]float64, len(candles))
	lows := make([]float64, len(candles))
	highs := make([]float64, len(candles))
	volumes := make([]float64, len(candles))

	for i, candle := range candles {
		closes[i] = candle.Close
		lows[i] = candle.Low
		highs[i] = candle.High
		volumes[i] = candle.Volume
	}

	ma50 := MovingAverage(closes, 50)
	ma200 := MovingAverage(closes, 200)
	rsi14 := RSI(closes, 14)
	macd, macdSignal := MACD(closes, 12, 26, 9)
	ma20, bbUpper, bbLower := BollingerBands(closes, 20, 2.0)
	stochK, stochD := StochasticOscillator(closes, lows, highs, 14)
	vwap := VWAP(closes, volumes)

	for i := params.WindowSize; i < len(candles)-5; i++ {
		window := candles[i-params.WindowSize : i]
		feature := []float64{}
		tracker.Increment(1)

		for j := 0; j > -params.WindowSize; j-- {
			// Feature extraction
			f := []float64{
				window[params.WindowSize-1+j].Close,  // Latest close price
				ma50[i+j],                            // 50-period MA
				ma200[i+j],                           // 200-period MA
				rsi14[i+j],                           // 14-period RSI
				macd[i+j],                            // MACD
				macdSignal[i+j],                      // MACD Signal line
				ma20[i+j],                            // 20-period MA (Bollinger Middle Band)
				bbUpper[i+j],                         // Bollinger Upper Band
				bbLower[i+j],                         // Bollinger Lower Band
				stochK[i+j],                          // Stochastic %K
				stochD[i+j],                          // Stochastic %D
				vwap[i+j],                            // Volume Weighted Average Price
				window[params.WindowSize-1+j].Volume, // Latest volume
			}

			feature = append(feature, f...)
		}
		features = append(features, feature)

		// Labeling strategy:
		// - StrategyLong if 40% gain within next 5 candles
		// - StrategyShort if 40% loss within next 5 candles
		// - StrategyHold if a loss is avoided
		label := StrategyHold
		basePrice := candles[i].Close
		low := candles[i].Low
		high := candles[i].High
		for j := 1; j <= 5; j++ {
			low = math.Min(low, candles[i+j].Low)
			high = math.Max(high, candles[i+j].High)
			if priceChange := (candles[i+j].High - basePrice) / basePrice; priceChange >= params.StrategyLong {
				if priceChange := (low - basePrice) / basePrice; priceChange > -params.StrategyHold {
					label = StrategyLong
				}
				break
			} else if priceChange := (candles[i+j].Low - basePrice) / basePrice; priceChange <= -params.StrategyShort {
				if priceChange := (high - basePrice) / basePrice; priceChange > params.StrategyHold {
					label = StrategyShort
				}
				break
			}
		}
		labels = append(labels, float64(label))
	}

	tracker.MarkAsDone()

	return NormalizeData(pw, features), labels
}

func NormalizeData(pw progress.Writer, features [][]float64) [][]float64 {
	tracker := progress.Tracker{
		Message: "Normalizing data",
		Total:   int64(len(features[0]) * len(features)),
		Units:   progress.UnitsDefault,
	}
	pw.AppendTracker(&tracker)
	tracker.Start()

	for i := range features[0] {
		min, max := math.Inf(1), math.Inf(-1)
		for j := range features {
			tracker.Increment(1)
			min = math.Min(min, features[j][i])
			max = math.Max(max, features[j][i])
		}
		if max > min {
			for j := range features {
				features[j][i] = (features[j][i] - min) / (max - min)
			}
		}
	}

	tracker.MarkAsDone()
	return features
}

func CategoricalCrossEntropy(pred, target *gorgonia.Node) (*gorgonia.Node, error) {
	logPred, err := gorgonia.Log(pred)
	if err != nil {
		return nil, err
	}
	ce, err := gorgonia.HadamardProd(target, logPred)
	if err != nil {
		return nil, err
	}
	meanCE, err := gorgonia.Mean(ce)
	if err != nil {
		return nil, err
	}
	return gorgonia.Neg(meanCE)
}

func OneHotEncode(labels []float64, numClasses int) [][]float64 {
	oneHot := make([][]float64, len(labels))
	for i, label := range labels {
		row := make([]float64, numClasses)
		row[int(label)] = 1.0
		oneHot[i] = row
	}
	return oneHot
}

// Flatten the 2D one-hot encoded labels into a 1D slice
func FlattenOneHot(oneHot [][]float64) []float64 {
	flat := make([]float64, 0, len(oneHot)*len(oneHot[0]))
	for _, row := range oneHot {
		flat = append(flat, row...)
	}
	return flat
}

func BuildAndTrainNN(pw progress.Writer, features [][]float64, labels []float64, epochs int) ([]tensor.Tensor, error) {
	g := gorgonia.NewGraph()

	tracker := progress.Tracker{
		Message: "Training",
		Total:   int64(epochs),
		Units:   progress.UnitsDefault,
	}
	pw.AppendTracker(&tracker)
	tracker.Start()

	inputSize := len(features[0])
	outputSize := 3
	batchSize := len(features)

	// Input and output tensors
	flatFeatures := make([]float64, batchSize*inputSize)
	for i := 0; i < batchSize; i++ {
		copy(flatFeatures[i*inputSize:(i+1)*inputSize], features[i])
	}

	// Explicitly set the tensor shape to avoid shape mismatch
	xVal := tensor.New(
		tensor.WithShape(batchSize, inputSize),
		tensor.Of(tensor.Float64),
		tensor.WithBacking(flatFeatures),
	)

	xTensor := gorgonia.NewTensor(
		g, tensor.Float64, 2,
		gorgonia.WithShape(batchSize, inputSize),
		gorgonia.WithName("x"),
		gorgonia.WithValue(xVal),
	)

	oneHotLabels := OneHotEncode(labels, 3) // Assuming 3 classes: 0, 1, 2
	flatLabels := FlattenOneHot(oneHotLabels)
	yVal := tensor.New(
		tensor.WithShape(batchSize, outputSize),
		tensor.Of(tensor.Float64),
		tensor.WithBacking(flatLabels),
	)

	yTensor := gorgonia.NewTensor(
		g, tensor.Float64, 2,
		gorgonia.WithShape(batchSize, outputSize),
		gorgonia.WithName("y"),
		gorgonia.WithValue(yVal),
	)

	// Weight and bias initialization with gradient tracking
	w0 := gorgonia.NewMatrix(
		g, tensor.Float64,
		gorgonia.WithShape(inputSize, 10),
		gorgonia.WithName("w0"),
		gorgonia.WithInit(gorgonia.GlorotU(1)),
		gorgonia.WithGrad(tensor.New(tensor.Of(tensor.Float64), tensor.WithShape(inputSize, 10))),
	)

	b0 := gorgonia.NewMatrix(
		g, tensor.Float64,
		gorgonia.WithShape(1, 10), // Bias is (1, 10) for correct broadcasting
		gorgonia.WithName("b0"),
		gorgonia.WithInit(gorgonia.Zeroes()),
		gorgonia.WithGrad(tensor.New(tensor.Of(tensor.Float64), tensor.WithShape(1, 10))),
	)

	w1 := gorgonia.NewMatrix(
		g, tensor.Float64,
		gorgonia.WithShape(10, outputSize),
		gorgonia.WithName("w1"),
		gorgonia.WithInit(gorgonia.GlorotU(1)),
		gorgonia.WithGrad(tensor.New(tensor.Of(tensor.Float64), tensor.WithShape(10, outputSize))),
	)

	b1 := gorgonia.NewMatrix(
		g, tensor.Float64,
		gorgonia.WithShape(1, outputSize), // Bias is (1, outputSize) for broadcasting
		gorgonia.WithName("b1"),
		gorgonia.WithInit(gorgonia.Zeroes()),
		gorgonia.WithGrad(tensor.New(tensor.Of(tensor.Float64), tensor.WithShape(1, outputSize))),
	)

	// Forward pass with bias broadcasting
	l0Raw := gorgonia.Must(gorgonia.Mul(xTensor, w0))
	l0 := gorgonia.Must(gorgonia.BroadcastAdd(l0Raw, b0, nil, []byte{0}))
	l0Act := gorgonia.Must(gorgonia.LeakyRelu(l0, 0.01)) // ReLU activation

	predRaw := gorgonia.Must(gorgonia.Mul(l0Act, w1))
	pred := gorgonia.Must(gorgonia.BroadcastAdd(predRaw, b1, nil, []byte{0}))
	predAct := gorgonia.Must(gorgonia.SoftMax(pred)) // Softmax for multi-class classification

	// Binary Cross Entropy Loss
	loss, err := CategoricalCrossEntropy(predAct, yTensor)
	if err != nil {
		return nil, fmt.Errorf("failed to compute binary cross entropy: %w", err)
	}

	gorgonia.WithName("loss")(loss)
	gorgonia.WithName("l0Raw")(l0Raw)
	gorgonia.WithName("predAct")(predAct)

	// Create a virtual machine and bind dual values automatically
	vm := gorgonia.NewTapeMachine(g, gorgonia.BindDualValues(w0, b0, w1, b1, xTensor, yTensor, loss))
	defer vm.Close()

	// Prepare input data
	// Prepare input data with explicit reshaping

	learningRate := 0.001

	for epoch := 0; epoch < epochs; epoch++ {
		vm.Reset()

		gorgonia.Let(xTensor, xVal)
		gorgonia.Let(yTensor, yVal)

		if err := vm.RunAll(); err != nil {
			return nil, fmt.Errorf("error during training: %w", err)
		}

		solver := gorgonia.NewVanillaSolver(gorgonia.WithLearnRate(learningRate))
		if err := solver.Step([]gorgonia.ValueGrad{w0, b0, w1, b1}); err != nil {
			return nil, fmt.Errorf("error during solver step: %w", err)
		}

		// gradW0, _ := w0.Grad()
		// gradB0, _ := b0.Grad()
		// gradW1, _ := w1.Grad()
		// gradB1, _ := b1.Grad()

		// fmt.Printf("Gradients - w0: %v, b0: %v, w1: %v, b1: %v\n", gradW0, gradB0, gradW1, gradB1)

		tracker.SetValue(int64(epoch))
		tracker.UpdateMessage(fmt.Sprintf("Training: %v", loss.Value()))
	}
	tracker.MarkAsDone()

	// for _, n := range g.AllNodes() {
	// 	grad, _ := n.Grad()
	// 	fmt.Printf("Node: %s, Op: %v, Has Value: %v, Has Gradient: %v\n",
	// 		n.Name(), n.Op(), n.Value() != nil, grad != nil)
	// }

	w0Val, _ := w0.Value().(tensor.Tensor)
	b0Val, _ := b0.Value().(tensor.Tensor)
	w1Val, _ := w1.Value().(tensor.Tensor)
	b1Val, _ := b1.Value().(tensor.Tensor)

	return []tensor.Tensor{w0Val, b0Val, w1Val, b1Val}, nil
}

func Predict(weights []tensor.Tensor, input []float64) ([]float64, error) {
	g := gorgonia.NewGraph()
	inputSize := len(input)

	// Input tensor
	xVal := tensor.New(
		tensor.WithShape(1, inputSize),
		tensor.Of(tensor.Float64),
		tensor.WithBacking(input),
	)
	xTensor := gorgonia.NewTensor(g, tensor.Float64, 2, gorgonia.WithShape(1, inputSize), gorgonia.WithName("input"), gorgonia.WithValue(xVal))

	// Load weights as constants
	w0 := gorgonia.NewTensor(g, tensor.Float64, 2, gorgonia.WithShape(weights[0].Shape()...), gorgonia.WithValue(weights[0]))
	b0 := gorgonia.NewTensor(g, tensor.Float64, 2, gorgonia.WithShape(weights[1].Shape()...), gorgonia.WithValue(weights[1]))
	w1 := gorgonia.NewTensor(g, tensor.Float64, 2, gorgonia.WithShape(weights[2].Shape()...), gorgonia.WithValue(weights[2]))
	b1 := gorgonia.NewTensor(g, tensor.Float64, 2, gorgonia.WithShape(weights[3].Shape()...), gorgonia.WithValue(weights[3]))

	// Forward pass with bias broadcasting
	l0Raw := gorgonia.Must(gorgonia.Mul(xTensor, w0))
	l0 := gorgonia.Must(gorgonia.BroadcastAdd(l0Raw, b0, nil, []byte{0}))
	l0Act := gorgonia.Must(gorgonia.LeakyRelu(l0, 0.01))

	predRaw := gorgonia.Must(gorgonia.Mul(l0Act, w1))
	pred := gorgonia.Must(gorgonia.BroadcastAdd(predRaw, b1, nil, []byte{0}))
	predAct := gorgonia.Must(gorgonia.SoftMax(pred))

	// Create VM to run the graph
	vm := gorgonia.NewTapeMachine(g)
	if err := vm.RunAll(); err != nil {
		return nil, err
	}

	return predAct.Value().Data().([]float64)[0:3], nil
}
