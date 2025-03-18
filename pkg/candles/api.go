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
	Response   chan candleResponse
}

var (
	apiClient = resty.New().
		SetRetryCount(10).
		SetRetryWaitTime(200 * time.Millisecond).
		SetRetryMaxWaitTime(5 * time.Second)
)

type candleResponse struct {
	Candle Candle
	Err    error
}

func fetchMissingCandles(db *leveldb.DB, pw progress.Writer, instrument string, network Network, candles []Candle, from time.Time, to time.Time) chan candleResponse {
	if !from.Equal(from.Truncate(time.Minute)) {
		from = from.Add(time.Minute).Truncate(time.Minute)
	}
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
	var previousTime time.Time
	if len(candles) == 0 || candles[0].Timestamp.After(from) {
		missingIntervals = append(missingIntervals, struct {
			start time.Time
			end   time.Time
		}{
			start: from, // Ensure fetching starts exactly at `from`
			end:   to,
		})
	} else {
		previousTime = from.Add(-time.Minute)

		for _, candle := range candles {
			if candle.Timestamp.After(to) {
				break
			}

			if !candle.Timestamp.Equal(previousTime.Add(time.Minute)) {
				missingIntervals = append(missingIntervals, struct {
					start time.Time
					end   time.Time
				}{
					start: previousTime.Add(time.Minute),      // Adjust to the next expected candle
					end:   candle.Timestamp.Add(-time.Minute), // Avoid overlap
				})
			}

			previousTime = candle.Timestamp
		}

		if previousTime.Add(time.Minute).Before(to) {
			missingIntervals = append(missingIntervals, struct {
				start time.Time
				end   time.Time
			}{
				start: previousTime.Add(time.Minute), // Start from the next expected minute
				end:   to,
			})
		}
	}

	out := make(chan candleResponse, 100)

	if len(missingIntervals) == 0 {
		close(out)
		return out
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

	switch network {
	case OKX:
		okxFetcherInitOnce.Do(func() {
			startOKXFetcher(db)
		})

		go func() {
			defer close(out)

			channels := make([]chan candleResponse, len(missingIntervals))

			for i, interval := range missingIntervals {
				channels[i] = make(chan candleResponse, 100)

				okxFetchQueue <- candleRequest{
					Instrument: instrument,
					Start:      interval.start,
					End:        interval.end,
					Response:   channels[i],
				}
			}

			for _, ch := range channels {
				for candleResponse := range ch {
					out <- candleResponse
					if tracker != nil {
						tracker.Increment(1)
					}
				}
			}

			if tracker != nil {
				tracker.MarkAsDone()
			}
		}()
	case Binance:
		binanceFetcherInitOnce.Do(func() {
			startBinanceFetcher(db)
		})

		go func() {
			defer close(out)

			channels := make([]chan candleResponse, len(missingIntervals))

			for i, interval := range missingIntervals {
				channels[i] = make(chan candleResponse, 1500)

				binanceFetchQueue <- candleRequest{
					Instrument: instrument,
					Start:      interval.start,
					End:        interval.end,
					Response:   channels[i],
				}
			}

			for _, ch := range channels {
				for candleResponse := range ch {
					out <- candleResponse
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
