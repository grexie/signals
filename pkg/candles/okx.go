package candles

import (
	"encoding/json"
	"fmt"
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
			candles := fetchCandlesFromOKX(db, req.Instrument, req.Start, req.End)
			for candleResponse := range candles {
				req.Response <- candleResponse
			}
			close(req.Response)
			time.Sleep(time.Until(start.Add(200 * time.Millisecond)))
		}
	}()
}

func fetchCandlesFromOKX(db *leveldb.DB, instrument string, start, end time.Time) chan candleResponse {
	url := "https://www.okx.com/api/v5/market/history-candles"
	if !start.Equal(start.Truncate(time.Minute)) {
		start = start.Add(time.Minute).Truncate(time.Minute)
	}
	end = end.Truncate(time.Minute)

	out := make(chan candleResponse, 1000)

	go func() {
		defer close(out)

		notBefore := time.Now()
		for ; start.Before(end); start = start.Add(100 * time.Minute) {
			time.Sleep(time.Until(notBefore))

			params := map[string]string{
				"instId": instrument,
				"bar":    "1m",
				"limit":  "100",
				"after":  fmt.Sprintf("%d", start.Add(100*time.Minute).UTC().UnixMilli()),
				"before": fmt.Sprintf("%d", start.Add(-time.Second).UTC().UnixMilli()),
			}

			notBefore = time.Now().Add(200 * time.Millisecond)
			resp, err := apiClient.R().SetQueryParams(params).Get(url)
			if err != nil {
				out <- candleResponse{Err: err}
				return
			}

			var data struct {
				Code string     `json:"code"`
				Msg  string     `json:"msg"`
				Data [][]string `json:"data"`
			}

			if err := json.Unmarshal(resp.Body(), &data); err != nil {
				out <- candleResponse{Err: err}
				return
			}

			if data.Code != "0" {
				out <- candleResponse{Err: err}
				return
			}

			candles, err := newCandlesFromData(instrument, "okx", data.Data)
			if err != nil {
				out <- candleResponse{Err: err}
				return
			}

			for _, candle := range candles {
				key := fmt.Appendf([]byte{}, "%s-%s-1m-%s", instrument, "okx", candle.Timestamp.UTC().Format("2006-01-02T15:04"))
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
