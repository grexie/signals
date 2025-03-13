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

	"github.com/grexie/signals/pkg/db"
	"github.com/grexie/signals/pkg/market"
	"github.com/grexie/signals/pkg/model"
	"github.com/grexie/signals/pkg/trade"
	"github.com/jedib0t/go-pretty/v6/progress"
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/joho/godotenv"
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

	db, err := db.ConnectMongo()
	if err != nil {
		log.Fatalf("Failed to connect to MongoDB: %v", err)
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
	t.Render()

	t = table.NewWriter()
	t.SetOutputMirror(os.Stdout)
	t.SetTitle("Trade Info")
	t.AppendRows([]table.Row{
		{"Take Profit", fmt.Sprintf("%0.02f%%", 100*tp/tm)},
		{"Stop Loss", fmt.Sprintf("%0.02f%%", 100*sl*tm)},
		{"Leverage", fmt.Sprintf("%0.0f", leverage)},
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
	ctx, ch := market.FetchCandles(context.Background(), pw, db, instrument, now.AddDate(0, -1, -2), now, market.CandleBar1m)
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

	if m, err := model.NewEnsembleModel(context.Background(), db, instrument, generationsDuration, generations); err != nil {
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
