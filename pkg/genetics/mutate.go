package genetics

import "math/rand/v2"

// Mutation (Introduce small variations)
func mutate(s *Strategy, mutationRate float64) {
	if rand.Float64() < mutationRate {
		randomizeStrategy(s, 5)
	}
}
