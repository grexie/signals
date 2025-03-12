package ta

import "math"

func StochasticOscillator(closes, lows, highs []float64, window int) ([]float64, []float64) {
	kValues := make([]float64, len(closes))
	dValues := make([]float64, len(closes))

	for i := range closes {
		if i < window {
			kValues[i], dValues[i] = 0, 0
			continue
		}
		low, high := lows[i], highs[i]
		for j := i - window + 1; j <= i; j++ {
			low = math.Min(low, lows[j])
			high = math.Max(high, highs[j])
		}
		kValues[i] = 100 * (closes[i] - low) / (high - low)
	}

	// Calculate %D as a 3-period moving average of %K
	dValues = MovingAverage(kValues, 3)

	return kValues, dValues
}
