package candles

import (
	"encoding/json"
	"fmt"
	"log"
	"slices"
	"sync"
	"time"

	"github.com/syndtr/goleveldb/leveldb"
)

var (
	okxFetchQueue      = make(chan candleRequest, 100)
	okxFetcherInitOnce sync.Once
)

const (
	OKX Network = "okx"
)

func startOKXFetcher(db *leveldb.DB) {
	go func() {
		for req := range okxFetchQueue {
			start := time.Now()
			candles, err := fetchCandlesFromOKX(db, req.Instrument, req.Start, req.End)
			if err != nil {
				log.Println("api fetch error:", err)
			}
			for _, candle := range candles {
				req.Response <- candle
			}
			close(req.Response)
			time.Sleep(time.Until(start.Add(200 * time.Millisecond)))
		}
	}()
}

func fetchCandlesFromOKX(db *leveldb.DB, instrument string, start, end time.Time) ([]Candle, error) {
	url := "https://www.okx.com/api/v5/market/history-candles"
	start = start.Truncate(time.Minute)
	end = end.Truncate(time.Minute)

	out := []Candle{}

	notBefore := time.Now()
	for ; start.Before(end); start = start.Add(100 * time.Minute) {
		time.Sleep(time.Until(notBefore))

		params := map[string]string{
			"instId": instrument,
			"bar":    "1m",
			"limit":  "100",
			"after":  fmt.Sprintf("%d", start.Add(100*time.Minute).Add(time.Second).UnixMilli()),
			"before": fmt.Sprintf("%d", start.Add(-time.Second).UnixMilli()),
		}

		notBefore = time.Now().Add(200 * time.Millisecond)
		resp, err := apiClient.R().SetQueryParams(params).Get(url)
		if err != nil {
			return nil, err
		}

		var data struct {
			Code string     `json:"code"`
			Msg  string     `json:"msg"`
			Data [][]string `json:"data"`
		}

		if err := json.Unmarshal(resp.Body(), &data); err != nil {
			return nil, err
		}

		if data.Code != "0" {
			return nil, fmt.Errorf("api error: %s", data.Msg)
		}

		candles, err := newCandlesFromData(instrument, "okx", data.Data)
		if err != nil {
			return nil, fmt.Errorf("error unmarshalling candle data: %v", err)
		}

		for _, candle := range candles {
			key := fmt.Appendf([]byte{}, "%s-%s-1m-%s", instrument, "okx", candle.Timestamp.UTC().Format("2006-01-02T15:04"))
			if b, err := json.Marshal(candle); err != nil {
				return nil, fmt.Errorf("error marshalling candle to json: %v", err)
			} else if err := db.Put(key, b, nil); err != nil {
				return nil, fmt.Errorf("error storing candle in db: %v", err)
			}
		}

		out = append(out, candles...)
	}

	slices.SortFunc(out, func(a, b Candle) int {
		return a.Timestamp.Compare(b.Timestamp)
	})

	out = slices.CompactFunc(out, func(a, b Candle) bool {
		return a.Timestamp.Equal(b.Timestamp)
	})

	return out, nil
}
