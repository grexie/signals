package candles

import (
	"slices"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/jedib0t/go-pretty/v6/progress"
	"github.com/syndtr/goleveldb/leveldb"
)

type Network string

type candleRequest struct {
	Instrument string
	Start      time.Time
	End        time.Time
	Response   chan Candle
}

var (
	apiClient = resty.New()
)

func fetchMissingCandles(db *leveldb.DB, pw progress.Writer, instrument string, network Network, candles []Candle, from time.Time, to time.Time) chan Candle {
	from = from.Truncate(time.Minute)
	to = to.Truncate(time.Minute)

	missingIntervals := []struct {
		start time.Time
		end   time.Time
	}{}
	candles = candles[:]
	slices.SortFunc(candles, func(a, b Candle) int {
		return a.Timestamp.Compare(b.Timestamp)
	})

	// identify missing intervals
	previousTime := from.Add(-1 * time.Minute)
	for _, candle := range candles {
		if candle.Timestamp.Equal(previousTime.Add(time.Minute)) {
			previousTime = candle.Timestamp
			continue
		} else if candle.Timestamp.After(to) {
			break
		} else {
			missingIntervals = append(missingIntervals, struct {
				start time.Time
				end   time.Time
			}{
				start: previousTime,
				end:   candle.Timestamp,
			})
			previousTime = candle.Timestamp
			continue
		}
	}
	if previousTime.Before(to) {
		missingIntervals = append(missingIntervals, struct {
			start time.Time
			end   time.Time
		}{
			start: previousTime,
			end:   to,
		})
	}

	var tracker *progress.Tracker
	if pw != nil {
		total := int64(0)
		for _, interval := range missingIntervals {
			total += int64(interval.end.Sub(interval.start).Minutes())
		}
		tracker = &progress.Tracker{
			Message: "Fetching candles",
			Total:   total,
			Units:   progress.UnitsDefault,
		}
		pw.AppendTracker(tracker)
		tracker.Start()
	}

	out := make(chan Candle, 100)

	switch network {
	case OKX:
		okxFetcherInitOnce.Do(func() {
			startOKXFetcher(db)
		})

		go func() {
			defer close(out)

			channels := make([]chan Candle, len(missingIntervals))

			for i, interval := range missingIntervals {
				channels[i] = make(chan Candle, 100)

				okxFetchQueue <- candleRequest{
					Instrument: instrument,
					Start:      interval.start,
					End:        interval.end,
					Response:   channels[i],
				}
			}

			for _, ch := range channels {
				for candle := range ch {
					out <- candle
					if tracker != nil {
						tracker.Increment(1)
					}
				}
			}

			if tracker != nil {
				tracker.MarkAsDone()
			}
		}()
	}

	return out
}
