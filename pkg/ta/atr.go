package ta

import "math"

// ATR calculation
func ATR(highs, lows, closes []float64, period int) []float64 {
	tr := make([]float64, len(highs))

	// Calculate True Range
	for i := range highs {
		if i == 0 {
			tr[i] = highs[i] - lows[i]
		} else {
			highLow := highs[i] - lows[i]
			highPrevClose := math.Abs(highs[i] - closes[i-1])
			lowPrevClose := math.Abs(lows[i] - closes[i-1])

			tr[i] = math.Max(highLow, math.Max(highPrevClose, lowPrevClose))
		}
	}

	// Calculate Average True Range
	return MovingAverage(tr, period)
}
