package model

import (
	"math/rand"

	"github.com/jedib0t/go-pretty/v6/progress"
)

func balanceClasses(pw progress.Writer, features [][]float64, labels []float64) ([][]float64, []float64) {
	// Group samples by class
	classSamples := make(map[int][]int)
	for i, label := range labels {
		class := int(label)
		classSamples[class] = append(classSamples[class], i)
	}

	// Find majority class size
	majoritySize := 0
	for _, samples := range classSamples {
		if len(samples) > majoritySize {
			majoritySize = len(samples)
		}
	}

	// Balance dataset through augmentation
	balancedFeatures := make([][]float64, 0)
	balancedLabels := make([]float64, 0)

	for class, samples := range classSamples {
		currentSamples := len(samples)

		// Add original samples
		for _, idx := range samples {
			balancedFeatures = append(balancedFeatures, features[idx])
			balancedLabels = append(balancedLabels, float64(class))
		}

		// Add augmented samples if needed
		if currentSamples < majoritySize {
			numAugmented := majoritySize - currentSamples
			for i := 0; i < numAugmented; i++ {
				// Select a random sample to augment
				originalIdx := samples[rand.Intn(len(samples))]

				// Create augmented feature vector
				augmented := make([]float64, len(features[originalIdx]))
				copy(augmented, features[originalIdx])

				// Add noise to features (1% random noise)
				for j := range augmented {
					noise := (rand.Float64()*2 - 1) * 0.01
					augmented[j] *= (1 + noise)
				}

				balancedFeatures = append(balancedFeatures, augmented)
				balancedLabels = append(balancedLabels, float64(class))
			}
		}
	}

	return balancedFeatures, balancedLabels
}
