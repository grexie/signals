package model

import (
	"fmt"
	"io"
	"log"
	"math"
	"os"
	"strconv"
	"time"

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

func envString(name string, def func() string) func() string {
	return func() string {
		value := def()
		if v, ok := os.LookupEnv(name); ok {
			value = v
		}
		return value
	}
}

func envDuration(name string, def func() time.Duration) func() time.Duration {
	return func() time.Duration {
		value := def()
		if v, ok := os.LookupEnv(name); ok {
			if v, err := strconv.ParseInt(v, 10, 32); err != nil {
				log.Fatalf("failed to parse env.%s: %v", name, err)
			} else {
				value = time.Duration(v) * time.Second
			}
		}
		return value
	}
}

var (
	Instrument = envString("SIGNALS_INSTRUMENT", func() string { return "DOGE-USDT-SWAP" })
	Cooldown   = envDuration("SIGNALS_COOLDOWN", func() time.Duration { return 5 * time.Minute })
)

var (
	WindowSize = envInt("SIGNALS_WINDOW_SIZE", func() int {
		return 200
	}, BoundWindowSize)
	Candles = envInt("SIGNALS_CANDLES", func() int {
		return 5
	}, BoundCandles)
)

var (
	TakeProfit = envFloat64("SIGNALS_TAKE_PROFIT", func() float64 {
		return 0.4
	}, BoundTakeProfit)
	StopLoss = envFloat64("SIGNALS_STOP_LOSS", func() float64 {
		return 0.1
	}, BoundStopLoss)
	TradeMultiplier = envFloat64("SIGNALS_TRADE_MULTIPLIER", func() float64 {
		return 1.0
	}, func(v float64) float64 { return math.Max(0.5, math.Min(2, v)) })
	Leverage = envFloat64("SIGNALS_LEVERAGE", func() float64 {
		return 50.0
	}, func(v float64) float64 { return math.Max(1, math.Min(100, v)) })
	Commission = envFloat64("SIGNALS_COMMISSION", func() float64 {
		return 0.001
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

var (
	DropoutRate = envFloat64("SIGNALS_DROPOUT_RATE", func() float64 { return 0.4 }, BoundDropoutRate)
	L2Penalty   = envFloat64("SIGNALS_L2_PENALTY", func() float64 { return 0.05 }, BoundL2Penalty)
	LearnRate   = envFloat64("SIGNALS_LEARN_RATE", func() float64 { return 0.00005 }, BoundLearnRate)
)
