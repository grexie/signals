package model

func OneHotEncode(labels []float64, numClasses int) [][]float64 {
	oneHot := make([][]float64, len(labels))
	for i, label := range labels {
		row := make([]float64, numClasses)
		row[int(label)] = 1.0
		oneHot[i] = row
	}
	return oneHot
}

// Flatten the 2D one-hot encoded labels into a 1D slice
func FlattenOneHot(oneHot [][]float64) []float64 {
	flat := make([]float64, 0, len(oneHot)*len(oneHot[0]))
	for _, row := range oneHot {
		flat = append(flat, row...)
	}
	return flat
}
