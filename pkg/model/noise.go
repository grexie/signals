package model

import "math/rand"

func AugmentCandle(candle Candle, noisePercent float64) Candle {
	noise := func(value float64) float64 {
		return value * (1 + (rand.Float64()*2-1)*noisePercent)
	}

	return Candle{
		Timestamp: candle.Timestamp,
		Open:      noise(candle.Open),
		High:      noise(candle.High),
		Low:       noise(candle.Low),
		Close:     noise(candle.Close),
		Volume:    noise(candle.Volume),
	}
}
