package ta

func WilliamsR(highs, lows, closes []float64, period int) []float64 {
	williamsr := make([]float64, len(closes))
	for i := period; i < len(closes); i++ {
		highest := highs[i]
		lowest := lows[i]
		for j := i - period + 1; j <= i; j++ {
			if highs[j] > highest {
				highest = highs[j]
			}
			if lows[j] < lowest {
				lowest = lows[j]
			}
		}
		if highest != lowest {
			williamsr[i] = ((highest - closes[i]) / (highest - lowest)) * -100
		}
	}
	return williamsr
}
