package ta

// Calculate momentum (velocity of price movement)
func Momentum(prices []float64) float64 {
	if len(prices) < 2 {
		return 0
	}
	return prices[len(prices)-1] - prices[0]
}
