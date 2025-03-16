package candles

import (
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"time"
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

// Marshal to an array
func (c Candle) MarshalJSON() ([]byte, error) {
	return json.Marshal([]any{
		c.Timestamp.Format(time.RFC3339),
		c.Instrument,
		c.Network,
		c.Open, c.High, c.Low, c.Close, c.Volume,
	})
}

// Unmarshal from an array
func (c *Candle) UnmarshalJSON(data []byte) error {
	var arr [8]any
	if err := json.Unmarshal(data, &arr); err != nil {
		return err
	}

	ts, err := time.Parse(time.RFC3339, arr[0].(string))
	if err != nil {
		return err
	}

	c.Timestamp = ts
	c.Instrument = arr[1].(string)
	c.Network = arr[2].(string)
	c.Open = arr[3].(float64)
	c.High = arr[4].(float64)
	c.Low = arr[5].(float64)
	c.Close = arr[6].(float64)
	c.Volume = arr[7].(float64)
	return nil
}

func newCandlesFromData(instrument string, network string, data [][]string) ([]Candle, error) {
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
