package ta

// Calculate price changes over a specific timeframe
func PriceChanges(prices []float64, period int) []float64 {
	changes := make([]float64, len(prices))
	for i := range prices {
		if i < period {
			changes[i] = 0
		} else {
			changes[i] = (prices[i] - prices[i-period]) / prices[i-period]
		}
	}
	return changes
}
