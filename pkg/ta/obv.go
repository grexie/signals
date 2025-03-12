package ta

// OBV calculation
func OBV(closes, volumes []float64) []float64 {
	obv := make([]float64, len(closes))

	for i := range closes {
		if i == 0 {
			obv[i] = volumes[i]
			continue
		}

		if closes[i] > closes[i-1] {
			obv[i] = obv[i-1] + volumes[i]
		} else if closes[i] < closes[i-1] {
			obv[i] = obv[i-1] - volumes[i]
		} else {
			obv[i] = obv[i-1]
		}
	}

	return obv
}
