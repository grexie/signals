package ta

func MovingAverage(prices []float64, window int) []float64 {
	ma := make([]float64, len(prices))
	for i := range prices {
		if i < window {
			ma[i] = 0
			continue
		}
		sum := 0.0
		for j := 0; j < window; j++ {
			sum += prices[i-j]
		}
		ma[i] = sum / float64(window)
	}
	return ma
}
