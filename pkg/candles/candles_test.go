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
	end = end.Truncate(time.Minute)

	// check all candles exist
	for i := start; i.Before(end.Add(time.Second)); i = i.Add(time.Minute) {
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

	for i := 1; i < len(candles); i++ {
		// check sorting errors, should be ascending
		if candles[i].Timestamp.Before(candles[i-1].Timestamp) {
			return fmt.Errorf("candles are not sorted")
		}

		// check for duplicate candles
		if candles[i].Timestamp.Equal(candles[i-1].Timestamp) {
			return fmt.Errorf("candles contain duplicates")
		}
	}

	return nil
}

func TestGetCandles(t *testing.T) {
	now := time.Now().Add(-5 * time.Minute)
	start := now.Add(-7 * time.Hour)
	end := now.Add(-6 * time.Hour)
	if c1, err := candles.GetCandles(db, nil, "DOGE-USDT-SWAP", candles.OKX, start, end); err != nil {
		t.Fatalf("error getting candles c1: %v", err)
	} else if err := checkMissing(c1, start, end); err != nil {
		t.Fatalf("error checking candles c1: %v", err)
	}

	start = now.Add(-3 * time.Hour)
	end = now.Add(-2 * time.Hour)
	if c2, err := candles.GetCandles(db, nil, "DOGE-USDT-SWAP", candles.OKX, start, end); err != nil {
		t.Fatalf("error getting candles c2: %v", err)
	} else if err := checkMissing(c2, start, end); err != nil {
		t.Fatalf("error checking candles c2: %v", err)
	}

	start = now.Add(-8 * time.Hour)
	end = now
	if c3, err := candles.GetCandles(db, nil, "DOGE-USDT-SWAP", candles.OKX, start, end); err != nil {
		t.Fatalf("error getting candles c3: %v", err)
	} else if err := checkMissing(c3, start, end); err != nil {
		t.Fatalf("error checking candles c3: %v", err)
	}

	start = now.Add(-8 * time.Hour)
	end = now
	if c4, err := candles.GetCandles(db, nil, "DOGE-USDT-SWAP", candles.OKX, start, end); err != nil {
		t.Fatalf("error getting candles c4: %v", err)
	} else if err := checkMissing(c4, start, end); err != nil {
		t.Fatalf("error checking candles c4: %v", err)
	}
}
