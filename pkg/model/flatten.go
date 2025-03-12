package model

func flattenFeatures(features [][]float64) []float64 {
	totalSize := len(features) * len(features[0])
	flattened := make([]float64, 0, totalSize)
	for _, feature := range features {
		flattened = append(flattened, feature...)
	}
	return flattened
}

func flattenBatchFeatures(features [][]float64, indices []int) []float64 {
	batchSize := len(indices)
	if batchSize == 0 {
		return []float64{}
	}
	featureSize := len(features[0])
	flattened := make([]float64, batchSize*featureSize)

	for i, idx := range indices {
		copy(flattened[i*featureSize:], features[idx])
	}
	return flattened
}

func flattenBatchLabels(labels []float64, indices []int, numClasses int) []float64 {
	batchSize := len(indices)
	if batchSize == 0 {
		return []float64{}
	}
	flattened := make([]float64, batchSize*numClasses)

	for i, idx := range indices {
		label := int(labels[idx])
		flattened[i*numClasses+label] = 1.0
	}
	return flattened
}
