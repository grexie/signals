package ta

func RateOfChange(prices []float64, period int) []float64 {
	roc := make([]float64, len(prices))
	for i := period; i < len(prices); i++ {
		roc[i] = (prices[i] - prices[i-period]) / prices[i-period] * 100
	}
	return roc
}
