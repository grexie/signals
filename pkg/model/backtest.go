package model

import (
	"log"
	"math"
	"math/rand"
	"time"

	"github.com/grexie/signals/pkg/candles"
	"github.com/jedib0t/go-pretty/v6/progress"
	"gonum.org/v1/gonum/stat"
)

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
	trader := NewPaperTrader(10000, params.StopLoss, params.TakeProfit, params.Commission/2, Leverage(), params.Cooldown)

	for i := params.WindowSize; i < len(candles); i++ {
		trader.Iterate(candles[i], func(c Candle) Strategy {
			pred, err := Predict(m.weights, features[i-params.WindowSize])
			if err != nil {
				log.Println("prediction error:", err)
				return StrategyHold
			}

			if pred[1] >= params.MinTradeProbability && pred[2] < params.MinTradeProbability {
				return StrategyLong
			} else if pred[2] >= params.MinTradeProbability && pred[1] < params.MinTradeProbability {
				return StrategyShort
			} else {
				return StrategyHold
			}
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
