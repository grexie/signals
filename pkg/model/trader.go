package model

import (
	"fmt"
	"math"
	"time"

	"gonum.org/v1/gonum/stat"
)

// PaperTrader simulates trades based on model predictions
type PaperTrader struct {
	Capital           float64
	StartingCapital   float64
	OpenTrade         *Trade
	ClosedTrades      []Trade
	StopLossPercent   float64
	TakeProfitPercent float64
	TradeFeePercent   float64
	Leverage          float64
}

// Trade represents an open or closed trade
type Trade struct {
	EntryPrice       float64
	Size             float64
	IsLong           bool
	StopLoss         float64
	TakeProfit       float64
	EntryTime        time.Time
	ExitTime         *time.Time
	ExitPrice        *float64
	PercentageReturn *float64
}

// NewPaperTrader initializes a new paper trader
func NewPaperTrader(startingCapital, stopLossPercent, takeProfitPercent, tradeFeePercent, leverage float64) *PaperTrader {
	return &PaperTrader{
		Capital:           startingCapital,
		StartingCapital:   startingCapital,
		StopLossPercent:   stopLossPercent,
		TakeProfitPercent: takeProfitPercent,
		TradeFeePercent:   tradeFeePercent,
		Leverage:          leverage,
	}
}

// AddMoney adds funds to the paper trading account
func (pt *PaperTrader) AddMoney(amount float64) {
	pt.Capital += amount
	pt.StartingCapital += amount
}

// AddTrade opens a new trade
func (pt *PaperTrader) AddTrade(entryPrice float64, isLong bool) (*Trade, error) {
	if pt.OpenTrade != nil {
		return nil, fmt.Errorf("trade already open") // Prevent multiple open trades
	}

	// determine leveraged position size
	maxTradeCapital := pt.Capital / (1 + pt.TradeFeePercent*pt.Leverage)
	tradeSize := maxTradeCapital * pt.Leverage

	// compute stop loss and take profit levels
	stopLoss := entryPrice * (1 - pt.StopLossPercent)
	takeProfit := entryPrice * (1 + pt.TakeProfitPercent)

	if !isLong {
		stopLoss = entryPrice * (1 + pt.StopLossPercent)
		takeProfit = entryPrice * (1 - pt.TakeProfitPercent)
	}

	// calculate and deduct the entry fee
	fee := tradeSize * pt.TradeFeePercent
	if pt.Capital < fee {
		return nil, fmt.Errorf("insufficient capital for trade after fees")
	}
	pt.Capital -= fee

	// open the trade
	pt.OpenTrade = &Trade{
		EntryPrice: entryPrice,
		Size:       tradeSize,
		IsLong:     isLong,
		StopLoss:   stopLoss,
		TakeProfit: takeProfit,
		EntryTime:  time.Now(),
	}

	return pt.OpenTrade, nil
}

// Iterate processes a new candle
func (pt *PaperTrader) Iterate(candle Candle, predict func(Candle) Strategy) {
	if pt.OpenTrade == nil {
		signal := predict(candle)
		if signal != StrategyHold {
			pt.AddTrade(candle.Close, signal == StrategyLong)
		}
		return
	} else {
		// Check stop loss and take profit
		if (pt.OpenTrade.IsLong && candle.Low < pt.OpenTrade.StopLoss) ||
			(!pt.OpenTrade.IsLong && candle.High > pt.OpenTrade.StopLoss) {
			pt.CloseTrade(pt.OpenTrade.StopLoss, candle.Timestamp)
			return
		}

		if (pt.OpenTrade.IsLong && candle.High > pt.OpenTrade.TakeProfit) ||
			(!pt.OpenTrade.IsLong && candle.Low < pt.OpenTrade.TakeProfit) {
			pt.CloseTrade(pt.OpenTrade.TakeProfit, candle.Timestamp)
			return
		}
	}
}

// CloseTrade finalizes the trade
func (pt *PaperTrader) CloseTrade(exitPrice float64, exitTime time.Time) error {
	if pt.OpenTrade == nil {
		return fmt.Errorf("no trade open")
	}

	pnl := (exitPrice - pt.OpenTrade.EntryPrice) / pt.OpenTrade.EntryPrice
	if !pt.OpenTrade.IsLong {
		pnl *= -1 // Invert PnL for short trades
	}

	// profit and loss from leveraged position
	profitLoss := pt.OpenTrade.Size * pnl

	// apply closing trading fee
	fee := pt.OpenTrade.Size * pt.TradeFeePercent

	pt.Capital += profitLoss - fee

	if pt.Capital < 0 {
		pt.Capital = 0
	}

	pt.OpenTrade.ExitPrice = &exitPrice
	pt.OpenTrade.ExitTime = &exitTime
	percentageReturn := pnl * (1 - pt.TradeFeePercent*2*pt.Leverage)
	pt.OpenTrade.PercentageReturn = &percentageReturn
	pt.ClosedTrades = append(pt.ClosedTrades, *pt.OpenTrade)
	pt.OpenTrade = nil

	return nil
}

// PnLPercent returns the profit/loss percentage
func (pt *PaperTrader) PnL() float64 {
	return ((pt.Capital - pt.StartingCapital) / pt.StartingCapital) * 100
}

// MaxDrawdown calculates the worst peak-to-trough decline
func (pt *PaperTrader) MaxDrawdown() float64 {
	if len(pt.ClosedTrades) == 0 {
		return 0.0 // No trades, no drawdown
	}

	maxCapital := pt.StartingCapital // Peak capital
	maxDrawdown := 0.0               // Worst drawdown seen
	currentCapital := pt.StartingCapital

	for _, trade := range pt.ClosedTrades {
		// Calculate profit/loss
		var pnl float64
		if trade.IsLong {
			pnl = trade.Size * ((*trade.ExitPrice - trade.EntryPrice) / trade.EntryPrice)
		} else {
			pnl = trade.Size * ((trade.EntryPrice - *trade.ExitPrice) / trade.EntryPrice) // Invert for shorts
		}

		// Apply trading fees (charged on both entry and exit)
		tradeFees := trade.Size * pt.TradeFeePercent * 2

		// Update current capital
		currentCapital += pnl - tradeFees

		// Update peak capital
		if currentCapital > maxCapital {
			maxCapital = currentCapital
		}

		// Compute drawdown
		drawdown := (maxCapital - currentCapital) / maxCapital
		if drawdown > maxDrawdown {
			maxDrawdown = drawdown
		}
	}

	return maxDrawdown * 100 // Convert to percentage
}

func (p *PaperTrader) SharpeRatio(riskFreeRate float64) float64 {
	if len(p.ClosedTrades) == 0 {
		return 0
	}

	// Compute mean return
	var sum, variance float64
	returns := []float64{}
	for _, trade := range p.ClosedTrades {
		returns = append(returns, *trade.PercentageReturn)
		sum += *trade.PercentageReturn

	}

	if len(returns) == 0 {
		return 0
	}

	meanReturn := sum / float64(len(returns))

	// Compute standard deviation of returns
	for _, r := range returns {
		variance += math.Pow(r-meanReturn, 2)
	}
	stdDev := math.Sqrt(variance / float64(len(returns)))

	// Avoid divide-by-zero error
	if stdDev == 0 {
		return 0
	}

	// Sharpe Ratio Formula
	return (meanReturn - riskFreeRate) / stdDev
}

func (p *PaperTrader) SortinoRatio(riskFreeRate float64) float64 {
	if len(p.ClosedTrades) == 0 {
		return 0
	}

	returns := []float64{}
	negReturns := []float64{}
	for _, trade := range p.ClosedTrades {
		if *trade.PercentageReturn < 0 {
			negReturns = append(negReturns, *trade.PercentageReturn)
		}
		returns = append(returns, *trade.PercentageReturn)
	}

	if len(negReturns) == 0 {
		return math.Inf(1)
	}

	meanReturn := stat.Mean(returns, nil)

	// Compute standard deviation of returns
	downsideDeviation := stat.StdDev(negReturns, nil)

	// Avoid divide-by-zero error
	if downsideDeviation == 0 {
		return 0
	}

	// Sharpe Ratio Formula
	return (meanReturn - riskFreeRate) / downsideDeviation
}
