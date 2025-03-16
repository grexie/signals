package ta

import "math"

// func CCI(highs, lows, closes []float64, period int) []float64 {
// 	cci := make([]float64, len(closes))
// 	for i := period; i < len(closes); i++ {
// 		sum := 0.0
// 		tp := (highs[i] + lows[i] + closes[i]) / 3
// 		for j := i - period + 1; j <= i; j++ {
// 			sum += math.Abs(tp - ((highs[j] + lows[j] + closes[j]) / 3))
// 		}
// 		meanDev := sum / float64(period)
// 		if meanDev != 0 {
// 			cci[i] = (tp - MovingAverage(closes[i-period:i+1], period)[i]) / (0.015 * meanDev)
// 		}
// 	}
// 	return cci
// }

func CCI(highs, lows, closes []float64, period int) []float64 {
	cci := make([]float64, len(closes))

	if len(closes) < period {
		return cci
	}

	rollingSumTP := 0.0
	rollingSumDev := 0.0
	tpBuffer := make([]float64, 0, period)

	// Precompute first rolling sum for Typical Price (TP)
	for i := 0; i < period; i++ {
		tp := (highs[i] + lows[i] + closes[i]) / 3
		rollingSumTP += tp
		tpBuffer = append(tpBuffer, tp)
	}

	for i := period; i < len(closes); i++ {
		// Compute new Typical Price (TP)
		tp := (highs[i] + lows[i] + closes[i]) / 3
		tpBuffer = append(tpBuffer, tp)
		rollingSumTP += tp

		// Remove oldest TP from rolling sum
		if len(tpBuffer) > period {
			rollingSumTP -= tpBuffer[0]
			tpBuffer = tpBuffer[1:]
		}

		// Calculate Mean TP
		meanTP := rollingSumTP / float64(period)

		// Compute Mean Deviation efficiently
		rollingSumDev = 0
		for _, val := range tpBuffer {
			rollingSumDev += math.Abs(val - meanTP)
		}
		meanDeviation := rollingSumDev / float64(period)

		// Compute CCI
		if meanDeviation != 0 {
			cci[i] = (tp - meanTP) / (0.015 * meanDeviation)
		}
	}

	return cci
}
