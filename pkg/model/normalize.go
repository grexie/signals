package model

import "math"

// Normalize a single value using a slice for min/max values
func normalizeValue(value float64, values []float64) float64 {
	if len(values) == 0 {
		return 0.0
	}

	if len(values) == 2 { // If values contain only min and max
		min, max := values[0], values[1]
		if max > min {
			return (value - min) / (max - min)
		}
		return 0.5 // Default to middle if min==max
	}

	// Find min and max in the slice
	min, max := values[0], values[0]
	for _, v := range values {
		if v < min {
			min = v
		}
		if v > max {
			max = v
		}
	}

	// Prevent division by zero and ensure values are in 0-1 range
	if max > min {
		normalized := (value - min) / (max - min)
		return math.Max(0, math.Min(1, normalized)) // Clamp between 0-1
	}
	return 0.5 // Default to middle if all values are the same
}
