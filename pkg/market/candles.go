package market

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"slices"
	"sort"
	"strconv"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/jedib0t/go-pretty/v6/progress"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/util"
)

type Candle struct {
	Timestamp  time.Time `bson:"timestamp"`
	Instrument string    `bson:"instrument"`
	Network    string    `bson:"network"`
	Open       float64   `bson:"open"`
	High       float64   `bson:"high"`
	Low        float64   `bson:"low"`
	Close      float64   `bson:"close"`
	Volume     float64   `bson:"volume"`
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

func FetchCandles(ctx context.Context, pw progress.Writer, db *leveldb.DB, instrument string, start time.Time, end time.Time, bar CandleBar, fetch bool) (context.Context, chan Candle) {
	client := resty.New()
	candles := 100
	out := make(chan Candle, candles*100)
	ctx, cancel := context.WithCancelCause(ctx)

	url := "https://www.okx.com/api/v5/market/history-candles"

	go func() {
		defer close(out)

		if db != nil {
			tracker := &progress.Tracker{
				Message: "Fetching candles from cache",
				Units:   progress.UnitsDefault,
			}

			iter := db.NewIterator(util.BytesPrefix(fmt.Appendf([]byte{}, "%s-%s-", instrument, "okx")), nil)
			candles := []Candle{}
			for iter.Next() {
				var candle Candle
				if err := json.Unmarshal(iter.Value(), &candle); err != nil {
					log.Println(err)
					cancel(err)
					return
				}

				if candle.Timestamp.After(start) && candle.Timestamp.Before(end) {
					tracker.Increment(1)
					candles = append(candles, candle)
				}
			}
			slices.SortFunc(candles, func(a Candle, b Candle) int {
				return a.Timestamp.Compare(b.Timestamp)
			})
			if len(candles) > 0 {
				start = candles[len(candles)-1].Timestamp
			}
			iter.Release()

			for _, candle := range candles {
				out <- candle
			}

			tracker.MarkAsDone()
		}

		duration := time.Duration(candles) * CandleBarToDuration(bar)

		if fetch {
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
					log.Println(err)
					cancel(err)
					return
				} else if resp.IsError() {
					log.Println(string(resp.Body()))
					cancel(fmt.Errorf("error response: %v", resp.Status()))
					return
				} else {
					var data struct {
						Code string     `json:"code"`
						Msg  string     `json:"msg"`
						Data candleData `json:"data"`
					}

					if err := json.Unmarshal(resp.Body(), &data); err != nil {
						log.Println(err)
						cancel(fmt.Errorf("failed to parse response body: %s", err))
						return
					} else if data.Code != "0" {
						log.Println(data)
						cancel(fmt.Errorf("API Error: %s", data.Msg))
						return
					} else if candles, err := newCandlesFromData(instrument, data.Data); err != nil {
						log.Println(err)
						cancel(fmt.Errorf("failed to convert data to candles: %s", err))
						return
					} else {
						for _, candle := range candles {
							tracker.Increment(1)
							if db != nil {
								if b, err := json.Marshal(candle); err != nil {
									log.Println(err)
									cancel(fmt.Errorf("failed to cache candle: %v", err))
									return
								} else if err := db.Put(fmt.Appendf([]byte{}, "%s-%s-%d", instrument, "okx", candle.Timestamp.Unix()), b, nil); err != nil {
									log.Println(err)
									cancel(fmt.Errorf("failed to cache candle: %v", err))
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
		}
	}()

	return ctx, out
}
