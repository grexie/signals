package candles

import (
	"encoding/json"
	"fmt"
	"slices"
	"time"

	"github.com/jedib0t/go-pretty/v6/progress"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/util"
)

func GetCandles(db *leveldb.DB, pw progress.Writer, instrument string, network Network, start, end time.Time) ([]Candle, error) {
	out := []Candle{}

	for i := start.Truncate(time.Hour); i.Before(end); i = i.Add(time.Hour) {
		iter := db.NewIterator(util.BytesPrefix(fmt.Appendf([]byte{}, "%s-%s-1m-%s", instrument, network, i.UTC().Format("2006-01-02T15:"))), nil)
		for iter.Next() {
			var candle Candle
			if err := json.Unmarshal(iter.Value(), &candle); err != nil {
				continue
			}
			if (candle.Timestamp.Equal(start) || candle.Timestamp.After(start)) && (candle.Timestamp.Equal(end) || candle.Timestamp.Before(end)) {
				out = append(out, candle)
			}
		}
	}

	slices.SortFunc(out, func(a, b Candle) int {
		return a.Timestamp.Compare(b.Timestamp)
	})

	out = slices.CompactFunc(out, func(a, b Candle) bool {
		return a.Timestamp.Equal(b.Timestamp)
	})

	for candleResponse := range fetchMissingCandles(db, pw, instrument, network, out, start, end) {
		if candleResponse.Err != nil {
			return nil, candleResponse.Err
		} else {
			candle := candleResponse.Candle
			if (candle.Timestamp.Equal(start) || candle.Timestamp.After(start)) && (candle.Timestamp.Equal(end) || candle.Timestamp.Before(end)) {
				out = append(out, candle)
			}
		}
	}

	slices.SortFunc(out, func(a, b Candle) int {
		return a.Timestamp.Compare(b.Timestamp)
	})

	out = slices.CompactFunc(out, func(a, b Candle) bool {
		return a.Timestamp.Equal(b.Timestamp)
	})

	return out, nil
}
