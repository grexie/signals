package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/grexie/signals/pkg/genetics"
	"github.com/grexie/signals/pkg/market"
	"github.com/grexie/signals/pkg/model"
	"github.com/grexie/signals/pkg/trade"
	"github.com/jedib0t/go-pretty/v6/progress"
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/joho/godotenv"
	"github.com/syndtr/goleveldb/leveldb"
)

func loadEnv(filenames ...string) {
	for _, filename := range filenames {
		if s, err := os.Stat(filename); err == nil && !s.IsDir() {
			godotenv.Load(filename)
		}
	}
}

func main() {
	if _, ok := os.LookupEnv("ENV"); !ok {
		env := "development"
		os.Setenv("ENV", env)
	}
	loadEnv(".env."+os.Getenv("ENV")+".local", ".env."+os.Getenv("ENV"), ".env.local", ".env")

	db, err := leveldb.OpenFile("signals-cache.db", nil)
	if err != nil {
		log.Fatalf("failed to open signals-cache.db: %v", err)
	}

	generations := 24
	if g, ok := os.LookupEnv("SIGNALS_GENERATIONS"); ok {
		if g, err := strconv.ParseInt(g, 10, 64); err != nil {
			log.Fatalf("error parsing env.SIGNALS_GENERATIONS: %v", err)
		} else {
			generations = int(g)
		}
	}

	generationsDuration := time.Hour
	if g, ok := os.LookupEnv("SIGNALS_GENERATIONS_DURATION"); ok {
		if g, err := strconv.ParseInt(g, 10, 64); err != nil {
			log.Fatalf("error parsing env.SIGNALS_GENERATIONS_DURATION: %v", err)
		} else {
			generationsDuration = time.Duration(g) * time.Second
		}
	}

	cooldown := time.Duration(5) * time.Minute
	if c, ok := os.LookupEnv("SIGNALS_COOLDOWN"); ok {
		if c, err := strconv.ParseInt(c, 10, 64); err != nil {
			log.Fatalf("error parsing env.SIGNALS_COOLDOWN: %v", err)
		} else {
			cooldown = time.Duration(c) * time.Second
		}
	}

	instrument := "DOGE-USDT-SWAP"
	if i, ok := os.LookupEnv("SIGNALS_INSTRUMENT"); ok {
		instrument = i
	}

	candles := model.Candles()
	tp, sl := model.TakeProfit(), model.StopLoss()
	leverage := model.Leverage()
	tm := model.TradeMultiplier()
	commission := model.Commission()

	if len(os.Args) >= 2 && os.Args[1] == "optimize" {
		Optimize(db, instrument)
		return
	}

	t := table.NewWriter()
	t.SetOutputMirror(os.Stdout)
	t.SetTitle("Model Config")
	t.AppendRows([]table.Row{
		{"SIGNALS_INSTRUMENT", instrument},
		{"SIGNALS_CANDLES", fmt.Sprintf("%d", candles)},
		{"SIGNALS_TAKE_PROFIT", fmt.Sprintf("%0.04f", tp)},
		{"SIGNALS_STOP_LOSS", fmt.Sprintf("%0.04f", sl)},
		{"SIGNALS_LEVERAGE", fmt.Sprintf("%0.0f", leverage)},
		{"SIGNALS_TRADE_MULTIPLIER", fmt.Sprintf("%0.04f", tm)},
		{"SIGNALS_COMMISSION", fmt.Sprintf("%0.04f", commission)},
		{"SIGNALS_COOLDOWN", fmt.Sprintf("%0.0f", cooldown.Seconds())},
	})
	t.AppendSeparator()
	t.AppendRows([]table.Row{
		{"SIGNALS_SHORT_MOVING_AVERAGE_LENGTH", fmt.Sprintf("%d", model.ShortMovingAverageLength())},
		{"SIGNALS_LONG_MOVING_AVERAGE_LENGTH", fmt.Sprintf("%d", model.LongMovingAverageLength())},
		{"SIGNALS_LONG_RSI_LENGTH", fmt.Sprintf("%d", model.LongRSILength())},
		{"SIGNALS_SHORT_RSI_LENGTH", fmt.Sprintf("%d", model.ShortRSILength())},
		{"SIGNALS_SHORT_MACD_WINDOW_LENGTH", fmt.Sprintf("%d", model.ShortMACDWindowLength())},
		{"SIGNALS_LONG_MACD_WINDOW_LENGTH", fmt.Sprintf("%d", model.LongMACDWindowLength())},
		{"SIGNALS_MACD_SIGNAL_WINDOW", fmt.Sprintf("%d", model.MACDSignalWindow())},
		{"SIGNALS_FAST_SHORT_MACD_WINDOW_LENGTH", fmt.Sprintf("%d", model.FastShortMACDWindowLength())},
		{"SIGNALS_FAST_LONG_MACD_WINDOW_LENGTH", fmt.Sprintf("%d", model.FastLongMACDWindowLength())},
		{"SIGNALS_FAST_MACD_SIGNAL_WINDOW", fmt.Sprintf("%d", model.FastMACDSignalWindow())},
		{"SIGNALS_BOLLINGER_BANDS_WINDOW", fmt.Sprintf("%d", model.BollingerBandsWindow())},
		{"SIGNALS_BOLLINGER_BANDS_MULTIPLIER", fmt.Sprintf("%0.02f", model.BollingerBandsMultiplier())},
		{"SIGNALS_STOCHASTIC_OSCILLATOR_WINDOW", fmt.Sprintf("%d", model.StochasticOscillatorWindow())},
		{"SIGNALS_SLOW_ATR_PERIOD_WINDOW", fmt.Sprintf("%d", model.SlowATRPeriod())},
		{"SIGNALS_FAST_ATR_PERIOD_WINDOW", fmt.Sprintf("%d", model.FastATRPeriod())},
		{"SIGNALS_OBV_MOVING_AVERAGE_LENGTH", fmt.Sprintf("%d", model.OBVMovingAverageLength())},
		{"SIGNALS_VOLUMES_MOVING_AVERAGE_LENGTH", fmt.Sprintf("%d", model.VolumesMovingAverageLength())},
		{"SIGNALS_CHAIKIN_MONEY_FLOW_PERIOD", fmt.Sprintf("%d", model.ChaikinMoneyFlowPeriod())},
		{"SIGNALS_MONEY_FLOW_INDEX_PERIOD", fmt.Sprintf("%d", model.MoneyFlowIndexPeriod())},
		{"SIGNALS_RATE_OF_CHANGE_PERIOD", fmt.Sprintf("%d", model.RateOfChangePeriod())},
		{"SIGNALS_RATE_OF_CHANGE_PERIOD", fmt.Sprintf("%d", model.CCIPeriod())},
		{"SIGNALS_RATE_OF_CHANGE_PERIOD", fmt.Sprintf("%d", model.WilliamsRPeriod())},
		{"SIGNALS_PRICE_CHANGE_FAST_PERIOD", fmt.Sprintf("%d", model.PriceChangeFastPeriod())},
		{"SIGNALS_PRICE_CHANGE_MEDIUM_PERIOD", fmt.Sprintf("%d", model.PriceChangeMediumPeriod())},
		{"SIGNALS_PRICE_CHANGE_SLOW_PERIOD", fmt.Sprintf("%d", model.PriceChangeSlowPeriod())},
		{"SIGNALS_RSI_UPPER_BOUND", fmt.Sprintf("%0.02f", model.RSIUpperBound())},
		{"SIGNALS_RSI_LOWER_BOUND", fmt.Sprintf("%0.02f", model.RSILowerBound())},
	})
	t.Render()

	t = table.NewWriter()
	t.SetOutputMirror(os.Stdout)
	t.SetTitle("Trade Info")
	t.AppendRows([]table.Row{
		{"Take Profit", fmt.Sprintf("%0.02f%%", 100*tp/tm)},
		{"Stop Loss", fmt.Sprintf("%0.02f%%", 100*sl*tm)},
		{"Leverage", fmt.Sprintf("%0.0f", leverage)},
	})
	t.AppendSeparator()
	t.AppendRows([]table.Row{
		{"TP %", fmt.Sprintf("%0.02f%%", 100*tp/(tm*leverage))},
		{"SL %", fmt.Sprintf("%0.02f%%", 100*sl*tm/leverage)},
		{"Commission", fmt.Sprintf("%0.02f%%", 100*commission*leverage)},
	})
	t.Render()

	pw := progress.NewWriter()
	pw.SetMessageLength(40)
	pw.SetNumTrackersExpected(6)
	pw.SetSortBy(progress.SortByPercentDsc)
	pw.SetStyle(progress.StyleDefault)
	pw.SetTrackerLength(15)
	pw.SetTrackerPosition(progress.PositionRight)
	pw.SetUpdateFrequency(time.Millisecond * 100)
	pw.Style().Colors = progress.StyleColorsExample
	pw.Style().Options.PercentFormat = "%2.0f%%"
	go pw.Render()

	now := time.Now()
	ctx, ch := market.FetchCandles(context.Background(), pw, db, instrument, now.AddDate(0, -1, -2), now, market.CandleBar1m, true)
outer:
	for {
		select {
		case <-ch:
		case <-ctx.Done():
			if !errors.Is(ctx.Err(), context.Canceled) {
				log.Fatalf("context error: %v", ctx.Err())
			}
			break outer
		}
	}
	pw.Stop()
	for pw.IsRenderInProgress() {
		time.Sleep(100 * time.Millisecond)
	}

	notBefore := time.Time{}

	params := model.ModelParams{
		WindowSize:                 200,
		StrategyCandles:            candles,
		StrategyLong:               tp / leverage,
		StrategyShort:              tp / leverage,
		StrategyHold:               sl / leverage,
		TradeCommission:            commission * leverage,
		ShortMovingAverageLength:   model.ShortMovingAverageLength(),
		LongMovingAverageLength:    model.LongMovingAverageLength(),
		LongRSILength:              model.LongRSILength(),
		ShortRSILength:             model.ShortRSILength(),
		ShortMACDWindowLength:      model.ShortMACDWindowLength(),
		LongMACDWindowLength:       model.LongMACDWindowLength(),
		MACDSignalWindow:           model.MACDSignalWindow(),
		FastShortMACDWindowLength:  model.FastShortMACDWindowLength(),
		FastLongMACDWindowLength:   model.FastLongMACDWindowLength(),
		FastMACDSignalWindow:       model.FastMACDSignalWindow(),
		BollingerBandsWindow:       model.BollingerBandsWindow(),
		BollingerBandsMultiplier:   model.BollingerBandsMultiplier(),
		StochasticOscillatorWindow: model.StochasticOscillatorWindow(),
		SlowATRPeriod:              model.SlowATRPeriod(),
		FastATRPeriod:              model.FastATRPeriod(),
		OBVMovingAverageLength:     model.OBVMovingAverageLength(),
		VolumesMovingAverageLength: model.VolumesMovingAverageLength(),
		ChaikinMoneyFlowPeriod:     model.ChaikinMoneyFlowPeriod(),
		MoneyFlowIndexPeriod:       model.MoneyFlowIndexPeriod(),
		RateOfChangePeriod:         model.RateOfChangePeriod(),
		CCIPeriod:                  model.CCIPeriod(),
		WilliamsRPeriod:            model.WilliamsRPeriod(),
		PriceChangeFastPeriod:      model.PriceChangeFastPeriod(),
		PriceChangeMediumPeriod:    model.PriceChangeMediumPeriod(),
		PriceChangeSlowPeriod:      model.PriceChangeSlowPeriod(),
		RSIUpperBound:              model.RSIUpperBound(),
		RSILowerBound:              model.RSILowerBound(),
	}

	if m, err := model.NewEnsembleModel(context.Background(), db, instrument, params, generationsDuration, generations); err != nil {
		log.Fatalf("error instantiating ensemble model: %v", err)
	} else {

		for {
			nextTime := time.Now().Add(1 * time.Minute).Truncate(time.Minute)
			<-time.After(time.Until(nextTime))
			if strategy, votes, err := m.Predict(context.Background(), nextTime); err != nil {
				log.Println(err)
				continue
			} else {
				switch strategy {
				case model.StrategyHold:
					log.Printf("strategy: HOLD %s", votes)
				case model.StrategyLong:
					log.Printf("strategy: LONG %s", votes)
				case model.StrategyShort:
					log.Printf("strategy: SHORT %s", votes)
				}

				if hasPositions, positions, err := trade.CheckPositions(context.Background(), instrument); err != nil {
					log.Println(err)
					continue
				} else if hasPositions {
					for _, position := range positions.Data {
						if position.InstrumentID == instrument {
							if upnl, err := strconv.ParseFloat(position.UnrealisedPnL, 64); err != nil {
								log.Printf("error converting upnl %s to float: %v", position.UnrealisedPnL, err)
							} else {
								log.Printf("%s: %s %sx PX %s/%s UPnL %0.02f", instrument, strings.ToUpper(position.PositionSide), position.Leverage, position.Position, position.AveragePrice, upnl)
							}
						}
					}
				} else if equity, err := trade.GetEquity(context.Background()); err != nil {
					log.Println(err)
					continue
				} else if votes[model.StrategyLong] > votes[model.StrategyShort] && positions.HasShort(instrument) {
					for _, position := range positions.Short(instrument) {
						log.Printf("closing position as more votes for long than short\n%s", position)
						if err := trade.ClosePosition(instrument, position.Margin, position.PositionSide); err != nil {
							log.Println(err)
						}
					}
				} else if votes[model.StrategyShort] > votes[model.StrategyLong] && positions.HasLong(instrument) {
					for _, position := range positions.Long(instrument) {
						log.Printf("closing position as more votes for short than long\n%s", position)
						if err := trade.ClosePosition(instrument, position.Margin, position.PositionSide); err != nil {
							log.Println(err)
						}
					}
				} else if notBefore.Before(time.Now()) {
					switch strategy {
					case model.StrategyLong:
						if order, err := trade.PlaceOrder(context.Background(), instrument, true, equity, tp/tm, sl*tm, leverage); err != nil {
							log.Println(err)
							continue
						} else {
							log.Printf("placed LONG market order: %s %s", order.Instrument, order.OrderID)
							notBefore = time.Now().Add(cooldown)
							log.Printf("cooling down, next trade %s", notBefore)
						}
					case model.StrategyShort:
						if order, err := trade.PlaceOrder(context.Background(), instrument, false, equity, tp/tm, sl*tm, leverage); err != nil {
							log.Println(err)
							continue
						} else {
							log.Printf("placed SHORT market order: %s %s", order.Instrument, order.OrderID)
							notBefore = time.Now().Add(cooldown)
							log.Printf("cooling down, next trade %s", notBefore)
						}
					}
				}
			}
		}
	}
}

func Optimize(db *leveldb.DB, instrument string) {
	now := time.Now().Add(-5 * time.Minute)

	pw := progress.NewWriter()
	pw.SetMessageLength(40)
	pw.SetNumTrackersExpected(6)
	pw.SetSortBy(progress.SortByPercentDsc)
	pw.SetStyle(progress.StyleDefault)
	pw.SetTrackerLength(15)
	pw.SetTrackerPosition(progress.PositionRight)
	pw.SetUpdateFrequency(time.Millisecond * 100)
	pw.Style().Colors = progress.StyleColorsExample
	pw.Style().Options.PercentFormat = "%2.0f%%"
	go pw.Render()

	ctx, ch := market.FetchCandles(context.Background(), pw, db, instrument, now.AddDate(0, -1, -2), now, market.CandleBar1m, true)
outer:
	for {
		select {
		case <-ch:
		case <-ctx.Done():
			if !errors.Is(ctx.Err(), context.Canceled) {
				log.Fatalf("context error: %v", ctx.Err())
			}
			break outer
		}
	}
	pw.Stop()
	for pw.IsRenderInProgress() {
		time.Sleep(100 * time.Millisecond)
	}

	strategy := genetics.NaturalSelection(db, instrument, now, 50, 20, 0.4, 0.1)

	t := table.NewWriter()
	t.SetOutputMirror(os.Stdout)
	t.SetTitle("Best Strategy")
	t.AppendRows([]table.Row{
		{"SIGNALS_INSTRUMENT", strategy.Instrument},
		{"SIGNALS_WINDOW_SIZE", fmt.Sprintf("%.0f", strategy.WindowSize)},
		{"SIGNALS_CANDLES", fmt.Sprintf("%.0f", strategy.Candles)},
		{"SIGNALS_TAKE_PROFIT", fmt.Sprintf("%.04f", strategy.TakeProfit)},
		{"SIGNALS_STOP_LOSS", fmt.Sprintf("%.04f", strategy.StopLoss)},
	})
	t.AppendSeparator()
	t.AppendRows([]table.Row{
		{"SIGNALS_SHORT_MOVING_AVERAGE_LENGTH", fmt.Sprintf("%0.0f", strategy.ShortMovingAverageLength)},
		{"SIGNALS_LONG_MOVING_AVERAGE_LENGTH", fmt.Sprintf("%0.0f", strategy.LongMovingAverageLength)},
		{"SIGNALS_LONG_RSI_LENGTH", fmt.Sprintf("%0.0f", strategy.LongRSILength)},
		{"SIGNALS_SHORT_RSI_LENGTH", fmt.Sprintf("%0.0f", strategy.ShortRSILength)},
		{"SIGNALS_SHORT_MACD_WINDOW_LENGTH", fmt.Sprintf("%0.0f", strategy.ShortMACDWindowLength)},
		{"SIGNALS_LONG_MACD_WINDOW_LENGTH", fmt.Sprintf("%0.0f", strategy.LongMACDWindowLength)},
		{"SIGNALS_MACD_SIGNAL_WINDOW", fmt.Sprintf("%0.0f", strategy.MACDSignalWindow)},
		{"SIGNALS_FAST_SHORT_MACD_WINDOW_LENGTH", fmt.Sprintf("%0.0f", strategy.FastShortMACDWindowLength)},
		{"SIGNALS_FAST_LONG_MACD_WINDOW_LENGTH", fmt.Sprintf("%0.0f", strategy.FastLongMACDWindowLength)},
		{"SIGNALS_FAST_MACD_SIGNAL_WINDOW", fmt.Sprintf("%0.0f", strategy.FastMACDSignalWindow)},
		{"SIGNALS_BOLLINGER_BANDS_WINDOW", fmt.Sprintf("%0.0f", strategy.BollingerBandsWindow)},
		{"SIGNALS_BOLLINGER_BANDS_MULTIPLIER", fmt.Sprintf("%0.02f", strategy.BollingerBandsMultiplier)},
		{"SIGNALS_STOCHASTIC_OSCILLATOR_WINDOW", fmt.Sprintf("%0.0f", strategy.StochasticOscillatorWindow)},
		{"SIGNALS_SLOW_ATR_PERIOD_WINDOW", fmt.Sprintf("%0.0f", strategy.SlowATRPeriod)},
		{"SIGNALS_FAST_ATR_PERIOD_WINDOW", fmt.Sprintf("%0.0f", strategy.FastATRPeriod)},
		{"SIGNALS_OBV_MOVING_AVERAGE_LENGTH", fmt.Sprintf("%0.0f", strategy.OBVMovingAverageLength)},
		{"SIGNALS_VOLUMES_MOVING_AVERAGE_LENGTH", fmt.Sprintf("%0.0f", strategy.VolumesMovingAverageLength)},
		{"SIGNALS_CHAIKIN_MONEY_FLOW_PERIOD", fmt.Sprintf("%0.0f", strategy.ChaikinMoneyFlowPeriod)},
		{"SIGNALS_MONEY_FLOW_INDEX_PERIOD", fmt.Sprintf("%0.0f", strategy.MoneyFlowIndexPeriod)},
		{"SIGNALS_RATE_OF_CHANGE_PERIOD", fmt.Sprintf("%0.0f", strategy.RateOfChangePeriod)},
		{"SIGNALS_RATE_OF_CHANGE_PERIOD", fmt.Sprintf("%0.0f", strategy.CCIPeriod)},
		{"SIGNALS_RATE_OF_CHANGE_PERIOD", fmt.Sprintf("%0.0f", strategy.WilliamsRPeriod)},
		{"SIGNALS_PRICE_CHANGE_FAST_PERIOD", fmt.Sprintf("%0.0f", strategy.PriceChangeFastPeriod)},
		{"SIGNALS_PRICE_CHANGE_MEDIUM_PERIOD", fmt.Sprintf("%0.0f", strategy.PriceChangeMediumPeriod)},
		{"SIGNALS_PRICE_CHANGE_SLOW_PERIOD", fmt.Sprintf("%0.0f", strategy.PriceChangeSlowPeriod)},
		{"SIGNALS_RSI_UPPER_BOUND", fmt.Sprintf("%0.02f", strategy.RSIUpperBound)},
		{"SIGNALS_RSI_LOWER_BOUND", fmt.Sprintf("%0.02f", strategy.RSILowerBound)},
	})
	t.Render()
	strategy.ModelMetrics.Write(os.Stdout)
}
