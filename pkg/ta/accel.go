package ta

// Calculate acceleration of price movement
func Acceleration(prices []float64) float64 {
	if len(prices) < 3 {
		return 0
	}

	// Calculate velocity at different points
	v1 := prices[len(prices)/2] - prices[0]
	v2 := prices[len(prices)-1] - prices[len(prices)/2]

	// Return the change in velocity
	return v2 - v1
}
