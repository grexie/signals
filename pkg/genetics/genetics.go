package genetics

import (
	"context"
	"encoding/csv"
	"fmt"
	"log"
	"math/rand/v2"
	"os"
	"runtime"
	"sort"
	"sync"
	"time"

	"github.com/jedib0t/go-pretty/v6/progress"
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/syndtr/goleveldb/leveldb"
	"gonum.org/v1/gonum/stat"
)

// Worker function to evaluate fitness in parallel
func worker(ctx context.Context, db *leveldb.DB, pw progress.Writer, tracker *progress.Tracker, now time.Time, strategies []Strategy, results chan<- Strategy, wg *sync.WaitGroup) {
	defer wg.Done()
	for _, s := range strategies {
		s.ModelMetrics = evaluateFitness(ctx, pw, db, now, s)
		tracker.Increment(1)
		results <- s
	}
}

// Main Genetic Algorithm
func NaturalSelection(db *leveldb.DB, instrument string, now time.Time, popSize, generations int, retainRate, mutationRate float64, eliteCount int) Strategy {
	file, err := os.OpenFile(fmt.Sprintf("optimizer-%s.csv", now.Format("2006-01-02-15-04-05")), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		panic(err)
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	if stat, _ := file.Stat(); stat.Size() == 0 {
		WriteCSVHeader(writer)
	}

	// Initialize random population
	population := make([]Strategy, popSize)
	population[0] = newStrategy(instrument)
	for i := 1; i < popSize; i++ {
		population[i] = newStrategy(instrument)
		randomizeStrategy(&population[i], 25)
	}

	for gen := range generations {
		started := time.Now()

		// Evaluate fitness
		pw := progress.NewWriter()
		pw.SetMessageLength(40)
		pw.SetNumTrackersExpected(6)
		pw.SetSortBy(progress.SortByPercentDsc)
		pw.SetStyle(progress.StyleDefault)
		pw.SetTrackerLength(15)
		pw.SetTrackerPosition(progress.PositionRight)
		pw.SetUpdateFrequency(time.Millisecond * 100)
		pw.Style().Colors = progress.StyleColorsExample
		pw.Style().Options.PercentFormat = "%2.0f%%"
		go pw.Render()

		// Parallel fitness evaluation using worker pool
		tracker := progress.Tracker{
			Message: fmt.Sprintf("Evaluating fitness of generation %d", gen),
			Total:   int64(popSize),
			Units:   progress.UnitsDefault,
		}
		pw.AppendTracker(&tracker)
		tracker.Start()

		results := make(chan Strategy, popSize)
		var wg sync.WaitGroup
		numWorkers := runtime.NumCPU() - 1 // Adjust based on available CPU cores
		if numWorkers < 1 {
			numWorkers = 1
		}
		if numWorkers > popSize {
			numWorkers = popSize
		}
		chunkSize := popSize / numWorkers

		for i := 0; i < numWorkers; i++ {
			start := i * chunkSize
			end := start + chunkSize
			if i == numWorkers-1 {
				end = popSize
			}
			wg.Add(1)
			go worker(context.Background(), db, pw, &tracker, now, population[start:end], results, &wg)
		}

		go func() {
			wg.Wait()
			close(results)
		}()

		// Collect results
		fitnesses := []float64{}
		pnls := []float64{}
		maxDrawdowns := []float64{}
		sharpes := []float64{}
		sortinos := []float64{}
		trades := []float64{}
		trainDays := []float64{}
		f1Scores := []float64{}
		newPopulation := make([]Strategy, 0, popSize)
		for s := range results {
			newPopulation = append(newPopulation, s)
			fitnesses = append(fitnesses, s.ModelMetrics.Fitness())
			pnls = append(pnls, s.ModelMetrics.Backtest.Mean.PnL)
			maxDrawdowns = append(maxDrawdowns, s.ModelMetrics.Backtest.Mean.MaxDrawdown)
			sharpes = append(sharpes, s.ModelMetrics.Backtest.Mean.SharpeRatio)
			sortinos = append(sortinos, s.ModelMetrics.Backtest.Mean.SortinoRatio)
			trades = append(trades, s.ModelMetrics.Backtest.Mean.Trades)
			trainDays = append(trainDays, s.TrainDays)
			f1Scores = append(f1Scores, (s.ModelMetrics.F1Scores[0]+s.ModelMetrics.F1Scores[1]+s.ModelMetrics.F1Scores[2])/3)
		}

		tracker.MarkAsDone()

		pw.Stop()
		for pw.IsRenderInProgress() {
			time.Sleep(100 * time.Millisecond)
		}

		// Sort by fitness (higher is better)
		sort.Slice(newPopulation, func(i, j int) bool {
			return newPopulation[i].ModelMetrics.Fitness() > newPopulation[j].ModelMetrics.Fitness()
		})

		// Apply selection
		population = selection(newPopulation, retainRate, eliteCount)

		// Generate new population via crossover
		for len(population) < popSize {
			p1 := population[rand.IntN(len(population))]
			p2 := population[rand.IntN(len(population))]
			child := crossover(p1, p2)
			mutate(&child, mutationRate)
			population = append(population, child)
		}

		// Best strategy of this generation
		strategy := population[0]

		t := table.NewWriter()
		t.SetOutputMirror(os.Stdout)
		t.SetTitle(fmt.Sprintf("Generation %d - Summary", gen))
		t.AppendHeader(table.Row{"", "MEAN", "MIN", "25TH", "MEDIAN", "75TH", "MAX", "STDDEV"})
		t.AppendRows([]table.Row{
			{"Fitness", fmt.Sprintf("%0.6f", stat.Mean(fitnesses, nil)), fmt.Sprintf("%0.6f", minFloats(fitnesses)), fmt.Sprintf("%0.6f", CalculatePercentile(fitnesses, 25)), fmt.Sprintf("%0.6f", CalculatePercentile(fitnesses, 50)), fmt.Sprintf("%0.6f", CalculatePercentile(fitnesses, 75)), fmt.Sprintf("%0.6f", maxFloats(fitnesses)), fmt.Sprintf("%0.6f", stat.StdDev(fitnesses, nil))},
			{"PnL", fmt.Sprintf("%0.2f%%", stat.Mean(pnls, nil)), fmt.Sprintf("%0.2f%%", minFloats(pnls)), fmt.Sprintf("%0.2f%%", CalculatePercentile(pnls, 25)), fmt.Sprintf("%0.2f%%", CalculatePercentile(pnls, 50)), fmt.Sprintf("%0.2f%%", CalculatePercentile(pnls, 75)), fmt.Sprintf("%0.2f%%", maxFloats(pnls)), fmt.Sprintf("%0.6f", stat.StdDev(pnls, nil))},
			{"Max Drawdown", fmt.Sprintf("%0.2f%%", stat.Mean(maxDrawdowns, nil)), fmt.Sprintf("%0.2f%%", minFloats(maxDrawdowns)), fmt.Sprintf("%0.2f%%", CalculatePercentile(maxDrawdowns, 25)), fmt.Sprintf("%0.2f%%", CalculatePercentile(maxDrawdowns, 50)), fmt.Sprintf("%0.2f%%", CalculatePercentile(maxDrawdowns, 75)), fmt.Sprintf("%0.2f%%", maxFloats(maxDrawdowns)), fmt.Sprintf("%0.6f", stat.StdDev(maxDrawdowns, nil))},
			{"Sharpe Ratio", fmt.Sprintf("%0.2f", stat.Mean(sharpes, nil)), fmt.Sprintf("%0.2f", minFloats(sharpes)), fmt.Sprintf("%0.2f", CalculatePercentile(sharpes, 25)), fmt.Sprintf("%0.2f", CalculatePercentile(sharpes, 50)), fmt.Sprintf("%0.2f", CalculatePercentile(sharpes, 75)), fmt.Sprintf("%0.2f", maxFloats(sharpes)), fmt.Sprintf("%0.6f", stat.StdDev(sharpes, nil))},
			{"Sortino Ratio", fmt.Sprintf("%0.2f", stat.Mean(sortinos, nil)), fmt.Sprintf("%0.2f", minFloats(sortinos)), fmt.Sprintf("%0.2f", CalculatePercentile(sortinos, 25)), fmt.Sprintf("%0.2f", CalculatePercentile(sortinos, 50)), fmt.Sprintf("%0.2f", CalculatePercentile(sortinos, 75)), fmt.Sprintf("%0.2f", maxFloats(sortinos)), fmt.Sprintf("%0.6f", stat.StdDev(sortinos, nil))},
			{"Trades", fmt.Sprintf("%0.2f", stat.Mean(trades, nil)), fmt.Sprintf("%0.2f", minFloats(trades)), fmt.Sprintf("%0.2f", CalculatePercentile(trades, 25)), fmt.Sprintf("%0.2f", CalculatePercentile(trades, 50)), fmt.Sprintf("%0.2f", CalculatePercentile(trades, 75)), fmt.Sprintf("%0.2f", maxFloats(trades)), fmt.Sprintf("%0.6f", stat.StdDev(trades, nil))},
		})
		t.AppendSeparator()
		t.AppendRows([]table.Row{
			{"Train Days", fmt.Sprintf("%0.2f", stat.Mean(trainDays, nil)), fmt.Sprintf("%0.2f", minFloats(trainDays)), fmt.Sprintf("%0.2f", CalculatePercentile(trainDays, 25)), fmt.Sprintf("%0.2f", CalculatePercentile(trainDays, 50)), fmt.Sprintf("%0.2f", CalculatePercentile(trainDays, 75)), fmt.Sprintf("%0.2f", maxFloats(trainDays)), fmt.Sprintf("%0.6f", stat.StdDev(trainDays, nil))},
		})
		t.AppendSeparator()
		t.AppendRows([]table.Row{
			{"F1 Score", fmt.Sprintf("%0.2f%%", stat.Mean(f1Scores, nil)), fmt.Sprintf("%0.2f%%", minFloats(f1Scores)), fmt.Sprintf("%0.2f%%", CalculatePercentile(f1Scores, 25)), fmt.Sprintf("%0.2f%%", CalculatePercentile(f1Scores, 50)), fmt.Sprintf("%0.2f%%", CalculatePercentile(f1Scores, 75)), fmt.Sprintf("%0.2f%%", maxFloats(f1Scores)), fmt.Sprintf("%0.6f", stat.StdDev(f1Scores, nil))},
		})
		t.Render()

		params := StrategyToParams(strategy)
		params.Write(os.Stdout, fmt.Sprintf("Generation %d - Best Strategy", gen), false)
		strategy.ModelMetrics.Write(os.Stdout)

		if err := WriteCSVRow(writer, gen, started, time.Now(), fitnesses, pnls, maxDrawdowns, sharpes, sortinos, trades, trainDays, f1Scores, params, &strategy); err != nil {
			log.Fatalf("error writing csv: %v", err)
		}
	}

	return population[0] // Return the best-performing strategy
}
