package ta

import "math"

func CCI(highs, lows, closes []float64, period int) []float64 {
	cci := make([]float64, len(closes))
	for i := period; i < len(closes); i++ {
		sum := 0.0
		tp := (highs[i] + lows[i] + closes[i]) / 3
		for j := i - period + 1; j <= i; j++ {
			sum += math.Abs(tp - ((highs[j] + lows[j] + closes[j]) / 3))
		}
		meanDev := sum / float64(period)
		if meanDev != 0 {
			cci[i] = (tp - MovingAverage(closes, period)[i]) / (0.015 * meanDev)
		}
	}
	return cci
}
