package ta

func MACD(prices []float64, shortWindow, longWindow, signalWindow int) ([]float64, []float64) {
	shortMA := MovingAverage(prices, shortWindow)
	longMA := MovingAverage(prices, longWindow)
	macd := make([]float64, len(prices))
	var signal []float64

	for i := range prices {
		macd[i] = shortMA[i] - longMA[i]
	}
	signal = MovingAverage(macd, signalWindow)

	return macd, signal
}
