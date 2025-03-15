package candles_test

import (
	"fmt"
	"log"
	"os"
	"testing"
	"time"

	"github.com/grexie/signals/pkg/candles"
	"github.com/syndtr/goleveldb/leveldb"
)

var db *leveldb.DB

func TestMain(m *testing.M) {
	path := fmt.Sprintf("%s/signals-cache.db-test", os.TempDir())
	if err := os.RemoveAll(path); err != nil {
		log.Fatalf("failed to remove %s", path)
	} else if d, err := leveldb.OpenFile(path, nil); err != nil {
		log.Fatalf("failed to open %s: %v", path, err)
	} else {
		db = d
	}
	m.Run()
}

func checkMissing(candles []candles.Candle, start, end time.Time) error {
	if !start.Equal(start.Truncate(time.Minute)) {
		start = start.Add(time.Minute).Truncate(time.Minute)
	}
	if !end.Equal(end.Truncate(time.Minute)) {
		end = end.Add(-time.Minute).Truncate(time.Minute)
	}
	for i := start; i.Before(end); i = i.Add(time.Minute) {
		found := false
		for _, candle := range candles {
			if candle.Timestamp.Equal(i) {
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("couldn't find candle for %s", i.Format(time.RFC3339))
		}
	}
	return nil
}

func TestGetCandles(t *testing.T) {
	now := time.Now()
	start := now.Add(-7 * time.Hour)
	end := now.Add(-6 * time.Hour)
	c1 := candles.GetCandles(db, nil, "DOGE-USDT-SWAP", candles.OKX, start, end)
	if err := checkMissing(c1, start, end); err != nil {
		t.Fatalf("error getting candles c1: %v", err)
	}

	start = now.Add(-3 * time.Hour)
	end = now.Add(-2 * time.Hour)
	c2 := candles.GetCandles(db, nil, "DOGE-USDT-SWAP", candles.OKX, start, end)
	if err := checkMissing(c2, start, end); err != nil {
		t.Fatalf("error getting candles c2: %v", err)
	}

	start = now.Add(-8 * time.Hour)
	end = now
	c3 := candles.GetCandles(db, nil, "DOGE-USDT-SWAP", candles.OKX, start, end)
	if err := checkMissing(c3, start, end); err != nil {
		t.Fatalf("error getting candles c3: %v", err)
	}

}
