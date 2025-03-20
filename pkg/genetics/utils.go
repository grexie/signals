package genetics

func maxFloats(v []float64) float64 {
	if len(v) == 0 {
		return 0
	}
	out := v[0]
	for i := 1; i < len(v); i++ {
		if out < v[i] {
			out = v[i]
		}
	}
	return out
}

func minFloats(v []float64) float64 {
	if len(v) == 0 {
		return 0
	}
	out := v[0]
	for i := 1; i < len(v); i++ {
		if out > v[i] {
			out = v[i]
		}
	}
	return out
}
