package genetics

import (
	"math"
	"math/rand/v2"

	"gonum.org/v1/gonum/stat"
)

func selection(population []Strategy, retainRate float64, eliteCount int) []Strategy {
	fitnesses := make([]float64, len(population))
	for i, s := range population {
		fitnesses[i] = s.ModelMetrics.Fitness()
	}
	fitnessStdDev := stat.StdDev(fitnesses, nil)
	if fitnessStdDev > 0.05 {
		retainRate *= 0.9 // More selection pressure
	} else {
		retainRate *= 1.1 // Allow more exploration
	}

	n := int(float64(len(population)) * retainRate)
	elite := make([]Strategy, 0, eliteCount)

	// Explicitly retain the top 'eliteCount' best models no matter what
	for i := 0; i < eliteCount; i++ {
		elite = append(elite, population[i])
	}

	// Stochastic selection for maintaining diversity
	roulette := make([]Strategy, 0, len(population))
	totalFitness := 0.0
	for _, s := range population {
		totalFitness += s.ModelMetrics.Fitness()
	}

	for _, s := range population[n:] {
		scaledFitness := math.Exp(s.ModelMetrics.Fitness())
		if rand.Float64() < (scaledFitness / totalFitness) {
			roulette = append(roulette, s)
		}
	}

	return append(elite, roulette...)
}
