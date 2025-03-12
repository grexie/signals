package ta

func RSI(prices []float64, window int) []float64 {
	rsi := make([]float64, len(prices))
	for i := range prices {
		if i < window {
			continue
		}
		gains, losses := 0.0, 0.0
		for j := 0; j < window; j++ {
			change := prices[i-j] - prices[i-j-1]
			if change > 0 {
				gains += change
			} else {
				losses -= change
			}
		}
		avgGain := gains / float64(window)
		avgLoss := losses / float64(window)
		if avgLoss == 0 {
			rsi[i] = 100
		} else {
			rs := avgGain / avgLoss
			rsi[i] = 100 - (100 / (1 + rs))
		}
	}
	return rsi
}
