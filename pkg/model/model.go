package model

import (
	"context"
	"fmt"
	"io"
	"log"
	"math"
	"math/rand/v2"
	"os"
	"strconv"
	"time"

	"github.com/grexie/signals/pkg/candles"
	"github.com/jedib0t/go-pretty/v6/progress"
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/syndtr/goleveldb/leveldb"
	"gonum.org/v1/gonum/stat"
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

	Backtest DeepBacktestMetrics
}

func safeValue(v float64, def float64) float64 {
	if math.IsNaN(v) || math.IsInf(v, 0) {
		return def
	} else {
		return v
	}
}

func tradeFactor(trades float64, maxTrades float64) float64 {
	trades = safeValue(trades, 0)
	if trades <= maxTrades {
		return math.Min(1.8*math.Tanh(trades*0.25), 1.5) // Normal scaling up to 30 trades
	}
	return 1.5 * math.Exp(-(trades-maxTrades)/5) // Exponential decay penalty for trades > 30
}

func (m *ModelMetrics) Fitness() float64 {
	avgF1 := (m.F1Scores[0] + m.F1Scores[1] + m.F1Scores[2]) / 300
	normPnL := math.Tanh(safeValue(m.Backtest.Mean.PnL, 0) / 100)      // smoother scaling
	sharpe := math.Tanh(safeValue(m.Backtest.Mean.SharpeRatio, 0) / 3) // Smoother scaling
	sortino := math.Tanh(safeValue(m.Backtest.Mean.SortinoRatio, 0) / 3)
	drawdownPenalty := math.Exp(-safeValue(m.Backtest.Min.MaxDrawdown, 0) / 25) // Less extreme penalty

	// Penalize high variance across backtests
	variancePenalty := 1 / (1 + safeValue(m.Backtest.StdDev.PnL, 0)/5) // The divisor controls penalty strength

	// Cap trade rewards to prevent overtrading dominance
	tradeFactor := tradeFactor(safeValue(m.Backtest.Mean.Trades, 0), 30) // Cap trade rewards

	// Risk-Adjusted Return Modifier: Rewards per-trade profitability
	riskRewardFactor := math.Tanh((safeValue(m.Backtest.Mean.PnL, 0) / math.Max(safeValue(m.Backtest.Mean.Trades, 1), 1)) * 0.1)

	// Apply PnL Penalty to Encourage Profitability
	pnlPenalty := 1 / (1 + math.Exp(-safeValue(m.Backtest.Mean.PnL, 0)/4))

	// Compute base fitness
	fitness := avgF1*0.15 + sortino*0.3 + sharpe*0.2 + normPnL*0.15

	// Apply all penalties
	fitness *= drawdownPenalty
	fitness *= tradeFactor
	fitness *= (1 + riskRewardFactor*0.15)
	fitness *= variancePenalty
	fitness *= (1 + pnlPenalty*0.2)

	fitness = safeValue(fitness, 0)

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
	t.AppendHeader(table.Row{"", "MEAN", "MIN", "MAX", "STDDEV"})
	t.AppendRows([]table.Row{
		{"PnL", fmt.Sprintf("%6.2f%%", m.Backtest.Mean.PnL), fmt.Sprintf("%6.2f%%", m.Backtest.Min.PnL), fmt.Sprintf("%6.2f%%", m.Backtest.Max.PnL), fmt.Sprintf("%6.2f", m.Backtest.StdDev.PnL)},
		{"Max Drawdown", fmt.Sprintf("%6.2f%%", m.Backtest.Mean.MaxDrawdown), fmt.Sprintf("%6.2f%%", m.Backtest.Min.MaxDrawdown), fmt.Sprintf("%6.2f%%", m.Backtest.Max.MaxDrawdown), fmt.Sprintf("%6.2f", m.Backtest.StdDev.MaxDrawdown)},
		{"Sharpe Ratio", fmt.Sprintf("%6.2f", m.Backtest.Mean.SharpeRatio), fmt.Sprintf("%6.2f", m.Backtest.Min.SharpeRatio), fmt.Sprintf("%6.2f", m.Backtest.Max.SharpeRatio), fmt.Sprintf("%6.2f", m.Backtest.StdDev.SharpeRatio)},
		{"Sortino Ratio", fmt.Sprintf("%6.2f", m.Backtest.Mean.SortinoRatio), fmt.Sprintf("%6.2f", m.Backtest.Min.SortinoRatio), fmt.Sprintf("%6.2f", m.Backtest.Max.SortinoRatio), fmt.Sprintf("%6.2f", m.Backtest.StdDev.SortinoRatio)},
		{"Trades", fmt.Sprintf("%6.2f", m.Backtest.Mean.Trades), fmt.Sprintf("%6.2f", m.Backtest.Min.Trades), fmt.Sprintf("%6.2f", m.Backtest.Max.Trades), fmt.Sprintf("%6.2f", m.Backtest.StdDev.Trades)},
	})
	t.AppendSeparator()
	t.AppendRow(table.Row{"Fitness", fmt.Sprintf("%6.8f", m.Fitness())})
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
	candles, err := candles.GetCandles(db, nil, instrument, candles.OKX, from, to)
	if err != nil {
		return nil, err
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
		candles, err := candles.GetCandles(m.db, pw, m.Instrument, candles.OKX, from, now)
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

type BacktestMetrics struct {
	PnL          float64
	MaxDrawdown  float64
	SharpeRatio  float64
	SortinoRatio float64
	Trades       float64
}

func (m *Model) CalculateCandlesForBacktest(params ModelParams, start time.Time, end time.Time) int {
	return int(end.Sub(start) / time.Minute)
}

func (m *Model) Backtest(pw progress.Writer, iterate func(), instrument string, params ModelParams, start time.Time, end time.Time) (BacktestMetrics, error) {
	candles, err := candles.GetCandles(m.db, pw, instrument, candles.OKX, start.Add(-time.Duration(params.WindowSize)*time.Minute), end)
	if err != nil {
		return BacktestMetrics{}, err
	}

	features := PrepareForPrediction(candles, params)
	trader := NewPaperTrader(10000, params.StrategyHold, params.StrategyLong, params.TradeCommission/2, Leverage())

	for i := params.WindowSize; i < len(candles); i++ {
		trader.Iterate(candles[i], func(c Candle) Strategy {
			pred, err := Predict(m.weights, features[i-params.WindowSize])
			if err != nil {
				log.Println("prediction error:", err)
				return StrategyHold
			}

			prediction := argmax(pred)
			return Strategy(prediction)
		})
		if iterate != nil {
			iterate()
		}
	}

	days := float64(end.Sub(start).Hours() / 24)
	return BacktestMetrics{
		PnL:          (math.Pow(1.0+trader.PnL()/100.0, 1.0/days) - 1) * 100,
		MaxDrawdown:  trader.MaxDrawdown(),
		SharpeRatio:  trader.SharpeRatio(0),
		SortinoRatio: trader.SortinoRatio(0),
		Trades:       float64(len(trader.ClosedTrades)) / days,
	}, nil
}

type backtest struct {
	Start time.Time
	End   time.Time
}

type DeepBacktestMetrics struct {
	Mean   BacktestMetrics
	Min    BacktestMetrics
	Max    BacktestMetrics
	StdDev BacktestMetrics
}

func NewDeepBacktestMetrics(metrics []BacktestMetrics) DeepBacktestMetrics {
	pnl := make([]float64, len(metrics))
	maxDrawdown := make([]float64, len(metrics))
	sharpeRatio := make([]float64, len(metrics))
	sortinoRatio := make([]float64, len(metrics))
	trades := make([]float64, len(metrics))

	out := DeepBacktestMetrics{}

	for i, r := range metrics {
		pnl[i] = r.PnL
		maxDrawdown[i] = r.MaxDrawdown
		sharpeRatio[i] = r.SharpeRatio
		sortinoRatio[i] = r.SortinoRatio
		trades[i] = r.Trades

		if i == 0 {
			out.Min.PnL = pnl[i]
			out.Min.MaxDrawdown = maxDrawdown[i]
			out.Min.SharpeRatio = sharpeRatio[i]
			out.Min.SortinoRatio = sortinoRatio[i]
			out.Min.Trades = trades[i]

			out.Max.PnL = pnl[i]
			out.Max.MaxDrawdown = maxDrawdown[i]
			out.Max.SharpeRatio = sharpeRatio[i]
			out.Max.SortinoRatio = sortinoRatio[i]
			out.Max.Trades = trades[i]
		} else {
			out.Min.PnL = math.Min(out.Min.PnL, pnl[i])
			out.Min.MaxDrawdown = math.Min(out.Min.MaxDrawdown, maxDrawdown[i])
			out.Min.SharpeRatio = math.Min(out.Min.SharpeRatio, sharpeRatio[i])
			out.Min.SortinoRatio = math.Min(out.Min.SortinoRatio, sortinoRatio[i])
			out.Min.Trades = math.Min(out.Min.Trades, trades[i])

			out.Max.PnL = math.Max(out.Max.PnL, pnl[i])
			out.Max.MaxDrawdown = math.Max(out.Max.MaxDrawdown, maxDrawdown[i])
			out.Max.SharpeRatio = math.Max(out.Max.SharpeRatio, sharpeRatio[i])
			out.Max.SortinoRatio = math.Max(out.Max.SortinoRatio, sortinoRatio[i])
			out.Max.Trades = math.Max(out.Max.Trades, trades[i])
		}
	}

	out.Mean.PnL = stat.Mean(pnl, nil)
	out.Mean.MaxDrawdown = stat.Mean(maxDrawdown, nil)
	out.Mean.SharpeRatio = stat.Mean(sharpeRatio, nil)
	out.Mean.SortinoRatio = stat.Mean(sortinoRatio, nil)
	out.Mean.Trades = stat.Mean(trades, nil)

	out.StdDev.PnL = stat.StdDev(pnl, nil)
	out.StdDev.MaxDrawdown = stat.StdDev(maxDrawdown, nil)
	out.StdDev.SharpeRatio = stat.StdDev(sharpeRatio, nil)
	out.StdDev.SortinoRatio = stat.StdDev(sortinoRatio, nil)
	out.StdDev.Trades = stat.StdDev(trades, nil)

	return out
}

func (m *Model) DeepBacktest(pw progress.Writer, instrument string, params ModelParams, now time.Time) (DeepBacktestMetrics, error) {
	now = now.Truncate(time.Minute)

	backtestCandles := 0
	backtests := []backtest{}
	backtestResults := []BacktestMetrics{}
	for q := range 4 {
		q := now.AddDate(0, -3*q, 0)

		for _, d := range []int{7, 14, 28} {
			start := q.AddDate(0, 0, -int(rand.Float64()*60)-d)
			end := start.AddDate(0, 0, d)
			backtests = append(backtests, backtest{Start: start, End: end})
			backtestCandles += m.CalculateCandlesForBacktest(params, start, end)
		}
	}

	tracker := &progress.Tracker{
		Message: "Backtesting",
		Total:   int64(backtestCandles),
		Units:   progress.UnitsDefault,
	}
	pw.AppendTracker(tracker)
	tracker.Start()

	for _, backtest := range backtests {
		if r, err := m.Backtest(pw, func() {
			tracker.Increment(1)
		}, instrument, params, backtest.Start, backtest.End); err != nil {
			return DeepBacktestMetrics{}, err
		} else {
			backtestResults = append(backtestResults, r)
		}
	}

	tracker.MarkAsDone()

	return NewDeepBacktestMetrics(backtestResults), nil
}
