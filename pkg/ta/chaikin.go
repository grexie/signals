package ta

func ChaikinMoneyFlow(highs, lows, closes, volumes []float64, period int) []float64 {
	cmf := make([]float64, len(closes))
	for i := period; i < len(closes); i++ {
		sumMF := 0.0
		sumVol := 0.0
		for j := i - period + 1; j <= i; j++ {
			mf := ((closes[j] - lows[j]) - (highs[j] - closes[j])) / (highs[j] - lows[j]) * volumes[j]
			sumMF += mf
			sumVol += volumes[j]
		}
		if sumVol != 0 {
			cmf[i] = sumMF / sumVol
		}
	}
	return cmf
}

func MoneyFlowIndex(highs, lows, closes, volumes []float64, period int) []float64 {
	mfi := make([]float64, len(closes))
	for i := period; i < len(closes); i++ {
		posFlow := 0.0
		negFlow := 0.0
		for j := i - period + 1; j <= i; j++ {
			if closes[j] > closes[j-1] {
				posFlow += closes[j] * volumes[j]
			} else {
				negFlow += closes[j] * volumes[j]
			}
		}
		if negFlow != 0 {
			mfi[i] = 100 - (100 / (1 + posFlow/negFlow))
		} else {
			mfi[i] = 100
		}
	}
	return mfi
}
