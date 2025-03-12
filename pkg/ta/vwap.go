package ta

func VWAP(closes, volumes []float64) []float64 {
	vwap := make([]float64, len(closes))
	cumulativeVolume, cumulativeValue := 0.0, 0.0

	for i := range closes {
		cumulativeVolume += volumes[i]
		cumulativeValue += closes[i] * volumes[i]
		if cumulativeVolume != 0 {
			vwap[i] = cumulativeValue / cumulativeVolume
		} else {
			vwap[i] = 0
		}
	}
	return vwap
}
