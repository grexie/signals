package main

import (
	"context"
	"log"
	"math"
	"os"
	"strconv"
	"time"

	"github.com/grexie/signals/pkg/db"
	"github.com/grexie/signals/pkg/model"
	"github.com/grexie/signals/pkg/trade"
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

	if m, err := model.NewEnsembleModel(context.Background(), db, "DOGE-USDT-SWAP", generationsDuration, generations); err != nil {
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

				if hasPositions, err := trade.CheckPositions(context.Background(), "DOGE-USDT-SWAP"); err != nil {
					log.Println(err)
					continue
				} else if hasPositions {
					log.Println("position open for DOGE-USDT-SWAP")
				} else if equity, err := trade.GetEquity(context.Background()); err != nil {
					log.Println(err)
					continue
				} else {
					equity = math.Min(equity, 12500)
					switch strategy {
					case model.StrategyLong:
						if order, err := trade.PlaceOrder(context.Background(), "DOGE-USDT-SWAP", true, equity, model.TakeProfit, model.StopLoss, model.Leverage); err != nil {
							log.Println(err)
							continue
						} else {
							log.Printf("placed LONG market order: %s %s", order.Instrument, order.OrderID)
						}
					case model.StrategyShort:
						if order, err := trade.PlaceOrder(context.Background(), "DOGE-USDT-SWAP", false, equity, model.TakeProfit, model.StopLoss, model.Leverage); err != nil {
							log.Println(err)
							continue
						} else {
							log.Printf("placed SHORT market order: %s %s", order.Instrument, order.OrderID)
						}
					}
				}
			}
		}
	}
}
