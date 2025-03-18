package candles

import (
	"encoding/json"
	"time"
)

type Candle struct {
	Timestamp  time.Time
	Instrument string
	Network    string
	Open       float64
	High       float64
	Low        float64
	Close      float64
	Volume     float64
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
