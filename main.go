package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/grexie/signals/pkg/candles"
	"github.com/grexie/signals/pkg/genetics"
	"github.com/grexie/signals/pkg/model"
	"github.com/grexie/signals/pkg/trade"
	"github.com/jedib0t/go-pretty/v6/progress"
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
	runtime.GOMAXPROCS(runtime.NumCPU())

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

	tp, sl := model.TakeProfit(), model.StopLoss()
	leverage := model.Leverage()
	tm := model.TradeMultiplier()

	if len(os.Args) >= 2 {
		if os.Args[1] == "optimize" {
			Optimize(db, instrument)
			return
		} else if os.Args[1] == "train" {
			Train(db, instrument)
			return
		} else {
			log.Fatalf("unknown command: %s", os.Args[1])
		}
	}

	params := model.NewModelParamsFromDefaults()
	params.Write(os.Stdout, "Model Config", true)

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

	notBefore := time.Time{}

	now := time.Now()
	if _, err := candles.GetCandles(db, pw, instrument, candles.Network(model.Network()), now.AddDate(-1, 0, 0), now); err != nil {
		log.Fatalf("error fetching candles: %v", err)
	}

	if m, err := model.NewEnsembleModel(context.Background(), db, instrument, params, generationsDuration, generations); err != nil {
		log.Fatalf("error instantiating ensemble model: %v", err)
	} else {

		for {
			nextTime := time.Now().Add(1 * time.Minute).Truncate(time.Minute)
			<-time.After(time.Until(nextTime))
			if strategy, votes, err := m.Predict(nil, nextTime); err != nil {
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
				} else {
					if votes[model.StrategyLong] > votes[model.StrategyShort] && positions.HasShort(instrument) {
						for _, position := range positions.Short(instrument) {
							log.Printf("closing position as more votes for long than short\n%s", position)
							if err := trade.ClosePosition(instrument, position.Margin, position.PositionSide); err != nil {
								log.Println(err)
							}
						}
					}

					if votes[model.StrategyShort] > votes[model.StrategyLong] && positions.HasLong(instrument) {
						for _, position := range positions.Long(instrument) {
							log.Printf("closing position as more votes for short than long\n%s", position)
							if err := trade.ClosePosition(instrument, position.Margin, position.PositionSide); err != nil {
								log.Println(err)
							}
						}
					}

					if notBefore.Before(time.Now()) {
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
}

func Train(db *leveldb.DB, instrument string) {
	params := model.NewModelParamsFromDefaults()
	params.Write(os.Stdout, "Model Config", false)

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

	if m, err := model.NewModel(context.Background(), pw, db, instrument, params, now); err != nil {
		log.Fatalf("error training model: %v", err)
	} else {
		pw.Stop()
		for pw.IsRenderInProgress() {
			time.Sleep(100 * time.Millisecond)
		}

		m.Metrics.Write(os.Stdout)
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

	if _, err := candles.GetCandles(db, pw, instrument, candles.Network(model.Network()), now.AddDate(-1, 0, 0), now); err != nil {
		log.Fatalf("error fetching candles: %v", err)
	}

	populationSize := 50
	if v, ok := os.LookupEnv("SIGNALS_OPTIMIZER_POPULATION_SIZE"); ok {
		if v, err := strconv.ParseInt(v, 10, 64); err != nil {
			log.Fatalf("error parsing SIGNALS_OPTIMIZER_POPULATION_SIZE: %v", err)
		} else {
			populationSize = int(v)
		}
	}

	generations := 20
	if v, ok := os.LookupEnv("SIGNALS_OPTIMIZER_GENERATIONS"); ok {
		if v, err := strconv.ParseInt(v, 10, 64); err != nil {
			log.Fatalf("error parsing SIGNALS_OPTIMIZER_GENERATIONS: %v", err)
		} else {
			generations = int(v)
		}
	}

	retainRate := 0.45
	if v, ok := os.LookupEnv("SIGNALS_OPTIMIZER_RETAIN_RATE"); ok {
		if v, err := strconv.ParseFloat(v, 64); err != nil {
			log.Fatalf("error parsing SIGNALS_OPTIMIZER_RETAIN_RATE: %v", err)
		} else {
			retainRate = v
		}
	}

	mutationRate := 0.25
	if v, ok := os.LookupEnv("SIGNALS_OPTIMIZER_MUTATION_RATE"); ok {
		if v, err := strconv.ParseFloat(v, 64); err != nil {
			log.Fatalf("error parsing SIGNALS_OPTIMIZER_MUTATION_RATE: %v", err)
		} else {
			mutationRate = v
		}
	}

	eliteCount := 3
	if v, ok := os.LookupEnv("SIGNALS_OPTIMIZER_ELITE_COUNT"); ok {
		if v, err := strconv.ParseInt(v, 10, 64); err != nil {
			log.Fatalf("error parsing SIGNALS_OPTIMIZER_ELITE_COUNT: %v", err)
		} else {
			eliteCount = int(v)
		}
	}

	title := "Optimizer Config"
	os.Stdout.Write(fmt.Appendf([]byte{}, "+-%s-+\n| %s |\n+-%s-+\n\n", strings.Repeat("-", len(title)), title, strings.Repeat("-", len(title))))

	params := []string{
		fmt.Sprintf("SIGNALS_OPTIMIZER_POPULATION_SIZE=%d", populationSize),
		fmt.Sprintf("SIGNALS_OPTIMIZER_GENERATIONS=%d", generations),
		fmt.Sprintf("SIGNALS_OPTIMIZER_RETAIN_RATE=%.4f", retainRate),
		fmt.Sprintf("SIGNALS_OPTIMIZER_MUTATION_RATE=%.4f", mutationRate),
		fmt.Sprintf("SIGNALS_OPTIMIZER_ELITE_COUNT=%d", eliteCount),
	}

	for _, param := range params {
		fmt.Printf("%s\n", param)
	}

	fmt.Println()

	genetics.NaturalSelection(db, instrument, now, populationSize, generations, retainRate, mutationRate, eliteCount)
}
