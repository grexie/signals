package market

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sort"
	"strconv"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/grexie/signals/pkg/db"
	"github.com/jedib0t/go-pretty/v6/progress"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

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

type candleData [][]string

func newCandlesFromData(instrument string, data [][]string) ([]Candle, error) {
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

func FetchCandles(ctx context.Context, pw progress.Writer, mdb *mongo.Database, instrument string, start time.Time, end time.Time, bar CandleBar) (context.Context, chan Candle) {
	client := resty.New()
	candles := 100
	out := make(chan Candle, candles*100)
	ctx, cancel := context.WithCancelCause(ctx)

	url := "https://www.okx.com/api/v5/market/history-candles"

	go func() {
		defer close(out)
		defer cancel(nil)

		if mdb != nil {
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
				if pw != nil {
					pw.AppendTracker(tracker)
				}
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
		}

		duration := time.Duration(candles) * CandleBarToDuration(bar)

		tracker := &progress.Tracker{
			Message: "Fetching candles from API",
			Units:   progress.UnitsDefault,
			Total:   int64((end.Sub(start) / CandleBarToDuration(bar)) + 1),
		}
		if pw != nil {
			pw.AppendTracker(tracker)
		}
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
					Data candleData `json:"data"`
				}

				if err := json.Unmarshal(resp.Body(), &data); err != nil {
					cancel(fmt.Errorf("failed to parse response body: %s", err))
					return
				} else if data.Code != "0" {
					cancel(fmt.Errorf("API Error: %s", data.Msg))
					return
				} else if candles, err := newCandlesFromData(instrument, data.Data); err != nil {
					cancel(fmt.Errorf("failed to convert data to candles: %s", err))
					return
				} else {
					for _, candle := range candles {
						tracker.Increment(1)
						if mdb != nil {
							if _, err := mdb.Collection("candles").InsertOne(ctx, candle); err != nil {
								cancel(err)
								return
							}
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
