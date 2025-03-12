package ta

import "math"

func BollingerBands(prices []float64, window int, multiplier float64) ([]float64, []float64, []float64) {
	ma := MovingAverage(prices, window)
	upper, lower := make([]float64, len(prices)), make([]float64, len(prices))

	for i := range prices {
		if i < window {
			upper[i], lower[i] = 0, 0
			continue
		}
		sum := 0.0
		for j := i - window + 1; j <= i; j++ {
			sum += math.Pow(prices[j]-ma[i], 2)
		}
		stdDev := math.Sqrt(sum / float64(window))
		upper[i] = ma[i] + multiplier*stdDev
		lower[i] = ma[i] - multiplier*stdDev
	}
	return ma, upper, lower
}
