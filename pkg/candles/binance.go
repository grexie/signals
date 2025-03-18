package candles

import (
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"sync"
	"time"

	"github.com/syndtr/goleveldb/leveldb"
)

var (
	binanceFetchQueue      = make(chan candleRequest, 100)
	binanceFetcherInitOnce sync.Once
)

const (
	Binance Network = "binance"
)

func startBinanceFetcher(db *leveldb.DB) {
	go func() {
		for req := range binanceFetchQueue {
			start := time.Now()
			candles := fetchCandlesFromBinance(db, req.Instrument, req.Start, req.End)
			for candleResponse := range candles {
				req.Response <- candleResponse
			}
			close(req.Response)
			time.Sleep(time.Until(start.Add(200 * time.Millisecond)))
		}
	}()
}

func fetchCandlesFromBinance(db *leveldb.DB, symbol string, start, end time.Time) chan candleResponse {
	url := "https://api.binance.com/api/v3/klines"
	if !start.Equal(start.Truncate(time.Minute)) {
		start = start.Add(time.Minute).Truncate(time.Minute)
	}
	end = end.Truncate(time.Minute)

	out := make(chan candleResponse, 500)

	go func() {
		defer close(out)

		notBefore := time.Now()
		for ; start.Before(end); start = start.Add(500 * time.Minute) {
			time.Sleep(time.Until(notBefore))

			params := map[string]string{
				"symbol":    symbol,
				"interval":  "1m",
				"limit":     "500",
				"startTime": fmt.Sprintf("%d", start.Add(-time.Second).UTC().UnixMilli()),
				"endTime":   fmt.Sprintf("%d", start.Add(500*time.Minute).UTC().UnixMilli()),
			}

			notBefore = time.Now().Add(200 * time.Millisecond)
			resp, err := apiClient.R().SetQueryParams(params).Get(url)
			if err != nil {
				out <- candleResponse{Err: err}
				return
			}

			if resp.IsError() {
				out <- candleResponse{Err: fmt.Errorf("api error response: %s - %s", resp.Status(), string(resp.Body()))}
				continue
			}

			var klines [][]any

			if err := json.Unmarshal(resp.Body(), &klines); err != nil {
				out <- candleResponse{Err: err}
				return
			}

			candles, err := newCandlesFromDataBinance(symbol, "binance", klines)
			if err != nil {
				out <- candleResponse{Err: err}
				return
			}

			for _, candle := range candles {
				key := fmt.Appendf([]byte{}, "%s-%s-1m-%s", symbol, "binance", candle.Timestamp.UTC().Format("2006-01-02T15:04"))
				if b, err := json.Marshal(candle); err != nil {
					out <- candleResponse{Err: fmt.Errorf("error marshalling candle to json: %v", err)}
					return
				} else if err := db.Put(key, b, nil); err != nil {
					out <- candleResponse{Err: fmt.Errorf("error storing candle in db: %v", err)}
					return
				}

				out <- candleResponse{Candle: candle}
			}
		}
	}()

	return out
}

func newCandlesFromDataBinance(instrument string, network string, data [][]any) ([]Candle, error) {
	out := make([]Candle, len(data))

	for i, candle := range data {
		if len(candle) < 6 {
			return nil, fmt.Errorf("invalid candle data: %v", candle)
		}

		if timestamp, ok := candle[0].(float64); !ok {
			return nil, fmt.Errorf("timestamp not a float64")
		} else if open, err := strconv.ParseFloat(candle[1].(string), 64); err != nil {
			return nil, err
		} else if high, err := strconv.ParseFloat(candle[2].(string), 64); err != nil {
			return nil, err
		} else if low, err := strconv.ParseFloat(candle[3].(string), 64); err != nil {
			return nil, err
		} else if close, err := strconv.ParseFloat(candle[4].(string), 64); err != nil {
			return nil, err
		} else if volume, err := strconv.ParseFloat(candle[5].(string), 64); err != nil {
			return nil, err
		} else {
			out[i] = Candle{
				Timestamp:  time.UnixMilli(int64(timestamp)),
				Instrument: instrument,
				Network:    network,
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
