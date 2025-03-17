package genetics

import (
	"context"
	"fmt"
	"math"
	"math/rand/v2"
	"os"
	"runtime"
	"sort"
	"sync"
	"time"

	"github.com/grexie/signals/pkg/model"
	"github.com/jedib0t/go-pretty/v6/progress"
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/syndtr/goleveldb/leveldb"
	"gonum.org/v1/gonum/stat"
)

type Strategy struct {
	Instrument string
	WindowSize float64
	Candles    float64
	TakeProfit float64
	StopLoss   float64

	Cooldown float64

	MinTradeProbability float64

	ShortMovingAverageLength   float64
	LongMovingAverageLength    float64
	LongRSILength              float64
	ShortRSILength             float64
	ShortMACDWindowLength      float64
	LongMACDWindowLength       float64
	MACDSignalWindow           float64
	FastShortMACDWindowLength  float64
	FastLongMACDWindowLength   float64
	FastMACDSignalWindow       float64
	BollingerBandsWindow       float64
	BollingerBandsMultiplier   float64
	StochasticOscillatorWindow float64
	SlowATRPeriod              float64
	FastATRPeriod              float64
	OBVMovingAverageLength     float64
	VolumesMovingAverageLength float64
	ChaikinMoneyFlowPeriod     float64
	MoneyFlowIndexPeriod       float64
	RateOfChangePeriod         float64
	CCIPeriod                  float64
	WilliamsRPeriod            float64
	PriceChangeFastPeriod      float64
	PriceChangeMediumPeriod    float64
	PriceChangeSlowPeriod      float64
	RSIUpperBound              float64
	RSILowerBound              float64
	RSISlope                   float64

	L2Penalty   float64
	DropoutRate float64
	LearnRate   float64
	TrainDays   float64

	ModelMetrics *model.ModelMetrics
}

func randPercent(dev float64) float64 {
	return 1 + (rand.Float64()*(2*dev)-dev)/100
}

// Generate a strategy from configured values
func newStrategy(instrument string) Strategy {
	return Strategy{
		Instrument: instrument,

		WindowSize: model.BoundWindowSizeFloat64(float64(model.WindowSize())),
		Candles:    model.BoundCandlesFloat64(float64(model.Candles())),
		StopLoss:   model.BoundStopLoss(model.StopLoss()),
		TakeProfit: model.BoundTakeProfit(model.TakeProfit()),

		Cooldown: model.BoundCooldownFloat64(float64(model.Cooldown().Seconds())),

		MinTradeProbability: model.BoundMinTradeProbability(model.MinTradeProbability()),

		ShortMovingAverageLength:   model.BoundShortMovingAverageLengthFloat64(float64(model.ShortMovingAverageLength())),
		LongMovingAverageLength:    model.BoundLongMovingAverageLengthFloat64(float64(model.LongMovingAverageLength())),
		LongRSILength:              model.BoundLongRSILengthFloat64(float64(model.LongRSILength())),
		ShortRSILength:             model.BoundShortRSILengthFloat64(float64(model.ShortRSILength())),
		ShortMACDWindowLength:      model.BoundShortMACDWindowLengthFloat64(float64(model.ShortMACDWindowLength())),
		LongMACDWindowLength:       model.BoundLongMACDWindowLengthFloat64(float64(model.LongMACDWindowLength())),
		MACDSignalWindow:           model.BoundMACDSignalWindowFloat64(float64(model.MACDSignalWindow())),
		FastShortMACDWindowLength:  model.BoundFastShortMACDWindowLengthFloat64(float64(model.FastShortMACDWindowLength())),
		FastLongMACDWindowLength:   model.BoundFastLongMACDWindowLengthFloat64(float64(model.FastLongMACDWindowLength())),
		FastMACDSignalWindow:       model.BoundFastMACDSignalWindowFloat64(float64(model.FastMACDSignalWindow())),
		BollingerBandsWindow:       model.BoundBollingerBandsWindowFloat64(float64(model.BollingerBandsWindow())),
		BollingerBandsMultiplier:   model.BoundBollingerBandsMultiplier(float64(model.BollingerBandsMultiplier())),
		StochasticOscillatorWindow: model.BoundStochasticOscillatorWindowFloat64(float64(model.StochasticOscillatorWindow())),
		SlowATRPeriod:              model.BoundSlowATRPeriodFloat64(float64(model.SlowATRPeriod())),
		FastATRPeriod:              model.BoundFastATRPeriodFloat64(float64(model.FastATRPeriod())),
		OBVMovingAverageLength:     model.BoundOBVMovingAverageLengthFloat64(float64(model.OBVMovingAverageLength())),
		VolumesMovingAverageLength: model.BoundVolumesMovingAverageLengthFloat64(float64(model.VolumesMovingAverageLength())),
		ChaikinMoneyFlowPeriod:     model.BoundChaikinMoneyFlowPeriodFloat64(float64(model.ChaikinMoneyFlowPeriod())),
		MoneyFlowIndexPeriod:       model.BoundMoneyFlowIndexPeriodFloat64(float64(model.MoneyFlowIndexPeriod())),
		RateOfChangePeriod:         model.BoundRateOfChangePeriodFloat64(float64(model.RateOfChangePeriod())),
		CCIPeriod:                  model.BoundCCIPeriodFloat64(float64(model.CCIPeriod())),
		WilliamsRPeriod:            model.BoundWilliamsRPeriodFloat64(float64(model.WilliamsRPeriod())),
		PriceChangeFastPeriod:      model.BoundPriceChangeFastPeriodFloat64(float64(model.PriceChangeFastPeriod())),
		PriceChangeMediumPeriod:    model.BoundPriceChangeMediumPeriodFloat64(float64(model.PriceChangeMediumPeriod())),
		PriceChangeSlowPeriod:      model.BoundPriceChangeSlowPeriodFloat64(float64(model.PriceChangeSlowPeriod())),
		RSIUpperBound:              model.BoundRSIUpperBound(float64(model.RSIUpperBound())),
		RSILowerBound:              model.BoundRSILowerBound(float64(model.RSILowerBound())),
		RSISlope:                   model.BoundRSISlopeFloat64(float64(model.RSISlope())),

		L2Penalty:   model.BoundL2Penalty(model.L2Penalty()),
		DropoutRate: model.BoundDropoutRate(model.DropoutRate()),
		LearnRate:   model.BoundLearnRate(model.LearnRate()),
		TrainDays:   model.BoundTrainDaysFloat64(float64(model.TrainDays()) / (24 * float64(time.Hour))),
	}
}

func randomizeStrategy(s *Strategy, percent float64) {
	s.WindowSize = model.BoundWindowSizeFloat64(s.WindowSize * randPercent(percent))
	s.Candles = model.BoundCandlesFloat64(s.Candles * randPercent(percent))
	s.TakeProfit = model.BoundTakeProfit(s.TakeProfit * randPercent(percent))
	s.StopLoss = model.BoundStopLoss(s.StopLoss * randPercent(percent))

	s.Cooldown = model.BoundCooldownFloat64(s.Cooldown * randPercent(percent))

	s.MinTradeProbability = model.BoundMinTradeProbability(s.MinTradeProbability * randPercent(percent))

	s.ShortMovingAverageLength = model.BoundShortMovingAverageLengthFloat64(s.ShortMovingAverageLength * randPercent(percent))
	s.LongMovingAverageLength = model.BoundLongMovingAverageLengthFloat64(s.LongMovingAverageLength * randPercent(percent))
	s.LongRSILength = model.BoundLongRSILengthFloat64(s.LongRSILength * randPercent(percent))
	s.ShortRSILength = model.BoundShortRSILengthFloat64(s.ShortRSILength * randPercent(percent))
	s.ShortMACDWindowLength = model.BoundShortMACDWindowLengthFloat64(s.ShortMACDWindowLength * randPercent(percent))
	s.LongMACDWindowLength = model.BoundLongMACDWindowLengthFloat64(s.LongMACDWindowLength * randPercent(percent))
	s.MACDSignalWindow = model.BoundMACDSignalWindowFloat64(s.MACDSignalWindow * randPercent(percent))
	s.FastShortMACDWindowLength = model.BoundFastShortMACDWindowLengthFloat64(s.FastShortMACDWindowLength * randPercent(percent))
	s.FastLongMACDWindowLength = model.BoundFastLongMACDWindowLengthFloat64(s.FastLongMACDWindowLength * randPercent(percent))
	s.FastMACDSignalWindow = model.BoundFastMACDSignalWindowFloat64(s.FastMACDSignalWindow * randPercent(percent))
	s.BollingerBandsWindow = model.BoundBollingerBandsWindowFloat64(s.BollingerBandsWindow * randPercent(percent))
	s.BollingerBandsMultiplier = model.BoundBollingerBandsMultiplier(s.BollingerBandsMultiplier * randPercent(percent))
	s.StochasticOscillatorWindow = model.BoundStochasticOscillatorWindowFloat64(s.StochasticOscillatorWindow * randPercent(percent))
	s.SlowATRPeriod = model.BoundSlowATRPeriodFloat64(s.SlowATRPeriod * randPercent(percent))
	s.FastATRPeriod = model.BoundFastATRPeriodFloat64(s.FastATRPeriod * randPercent(percent))
	s.OBVMovingAverageLength = model.BoundOBVMovingAverageLengthFloat64(s.OBVMovingAverageLength * randPercent(percent))
	s.VolumesMovingAverageLength = model.BoundVolumesMovingAverageLengthFloat64(s.VolumesMovingAverageLength * randPercent(percent))
	s.ChaikinMoneyFlowPeriod = model.BoundChaikinMoneyFlowPeriodFloat64(s.ChaikinMoneyFlowPeriod * randPercent(percent))
	s.MoneyFlowIndexPeriod = model.BoundMoneyFlowIndexPeriodFloat64(s.MoneyFlowIndexPeriod * randPercent(percent))
	s.RateOfChangePeriod = model.BoundRateOfChangePeriodFloat64(s.RateOfChangePeriod * randPercent(percent))
	s.CCIPeriod = model.BoundCCIPeriodFloat64(s.CCIPeriod * randPercent(percent))
	s.WilliamsRPeriod = model.BoundWilliamsRPeriodFloat64(s.WilliamsRPeriod * randPercent(percent))
	s.PriceChangeFastPeriod = model.BoundPriceChangeFastPeriodFloat64(s.PriceChangeFastPeriod * randPercent(percent))
	s.PriceChangeMediumPeriod = model.BoundPriceChangeMediumPeriodFloat64(s.PriceChangeMediumPeriod * randPercent(percent))
	s.PriceChangeSlowPeriod = model.BoundPriceChangeSlowPeriodFloat64(s.PriceChangeSlowPeriod * randPercent(percent))
	s.RSIUpperBound = model.BoundRSIUpperBound(s.RSIUpperBound * randPercent(percent))
	s.RSILowerBound = model.BoundRSILowerBound(s.RSILowerBound * randPercent(percent))
	s.RSISlope = model.BoundRSISlopeFloat64(s.RSISlope * randPercent(percent))

	s.L2Penalty = model.BoundL2Penalty(s.L2Penalty * randPercent(percent))
	s.DropoutRate = model.BoundDropoutRate(s.DropoutRate * randPercent(percent))
	s.LearnRate = model.BoundLearnRate(s.LearnRate * randPercent(percent))
	s.TrainDays = model.BoundTrainDaysFloat64(s.TrainDays * randPercent(percent))
}

func StrategyToParams(s Strategy) model.ModelParams {
	return model.ModelParams{
		Instrument:      model.Instrument(),
		Leverage:        model.Leverage(),
		TradeMultiplier: model.TradeMultiplier(),
		Commission:      model.Commission(),
		Cooldown:        time.Duration(s.Cooldown * float64(time.Second)),

		WindowSize: int(s.WindowSize),
		Candles:    int(s.Candles),
		TakeProfit: s.TakeProfit / model.Leverage(),
		StopLoss:   s.StopLoss / model.Leverage(),

		MinTradeProbability: s.MinTradeProbability,

		ShortMovingAverageLength:   int(s.ShortMovingAverageLength),
		LongMovingAverageLength:    int(s.LongMovingAverageLength),
		LongRSILength:              int(s.LongRSILength),
		ShortRSILength:             int(s.ShortRSILength),
		ShortMACDWindowLength:      int(s.ShortMACDWindowLength),
		LongMACDWindowLength:       int(s.LongMACDWindowLength),
		MACDSignalWindow:           int(s.MACDSignalWindow),
		FastShortMACDWindowLength:  int(s.FastShortMACDWindowLength),
		FastLongMACDWindowLength:   int(s.FastLongMACDWindowLength),
		FastMACDSignalWindow:       int(s.FastMACDSignalWindow),
		BollingerBandsWindow:       int(s.BollingerBandsWindow),
		BollingerBandsMultiplier:   s.BollingerBandsMultiplier,
		StochasticOscillatorWindow: int(s.StochasticOscillatorWindow),
		SlowATRPeriod:              int(s.SlowATRPeriod),
		FastATRPeriod:              int(s.FastATRPeriod),
		OBVMovingAverageLength:     int(s.OBVMovingAverageLength),
		VolumesMovingAverageLength: int(s.VolumesMovingAverageLength),
		ChaikinMoneyFlowPeriod:     int(s.ChaikinMoneyFlowPeriod),
		MoneyFlowIndexPeriod:       int(s.MoneyFlowIndexPeriod),
		RateOfChangePeriod:         int(s.RateOfChangePeriod),
		CCIPeriod:                  int(s.CCIPeriod),
		WilliamsRPeriod:            int(s.WilliamsRPeriod),
		PriceChangeFastPeriod:      int(s.PriceChangeFastPeriod),
		PriceChangeMediumPeriod:    int(s.PriceChangeMediumPeriod),
		PriceChangeSlowPeriod:      int(s.PriceChangeSlowPeriod),
		RSIUpperBound:              s.RSIUpperBound,
		RSILowerBound:              s.RSILowerBound,
		RSISlope:                   int(s.RSISlope),

		L2Penalty:   s.L2Penalty,
		DropoutRate: s.DropoutRate,
		LearnRate:   s.LearnRate,
		TrainDays:   time.Duration(s.TrainDays * float64(time.Hour) * 24),
	}
}

// Evaluate fitness by composing a new model from the strategy
func evaluateFitness(ctx context.Context, pw progress.Writer, db *leveldb.DB, now time.Time, s Strategy) *model.ModelMetrics {
	params := StrategyToParams(s)

	if m, err := model.NewModel(ctx, pw, db, s.Instrument, params, now); err != nil {
		return &model.ModelMetrics{}
	} else {
		return &m.Metrics
	}
}

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

// Crossover (Breed new strategies from the best ones)
func crossover(parent1, parent2 Strategy) Strategy {
	if parent1.Instrument != parent2.Instrument {
		parent1 = parent2
	}

	// Helper function to select between parent1, parent2, or an average
	selectValue := func(a, b float64) float64 {
		r := rand.Float64()
		if r < 0.4 { // 40% chance inherit from parent 1
			return a
		} else if r < 0.8 { // 40% chance inherit from parent 2
			return b
		}
		// 20% chance take the average
		return (a + b) / 2
	}

	return Strategy{
		Instrument: parent1.Instrument,

		WindowSize: selectValue(parent1.WindowSize, parent2.WindowSize),
		Candles:    selectValue(parent1.Candles, parent2.Candles),
		TakeProfit: selectValue(parent1.TakeProfit, parent2.TakeProfit),
		StopLoss:   selectValue(parent1.StopLoss, parent2.StopLoss),

		Cooldown: selectValue(parent1.Cooldown, parent2.Cooldown),

		MinTradeProbability: selectValue(parent1.MinTradeProbability, parent2.MinTradeProbability),

		ShortMovingAverageLength:   selectValue(parent1.ShortMovingAverageLength, parent2.ShortMovingAverageLength),
		LongMovingAverageLength:    selectValue(parent1.LongMovingAverageLength, parent2.LongMovingAverageLength),
		LongRSILength:              selectValue(parent1.LongRSILength, parent2.LongRSILength),
		ShortRSILength:             selectValue(parent1.ShortRSILength, parent2.ShortRSILength),
		ShortMACDWindowLength:      selectValue(parent1.ShortMACDWindowLength, parent2.ShortMACDWindowLength),
		LongMACDWindowLength:       selectValue(parent1.LongMACDWindowLength, parent2.LongMACDWindowLength),
		MACDSignalWindow:           selectValue(parent1.MACDSignalWindow, parent2.MACDSignalWindow),
		FastShortMACDWindowLength:  selectValue(parent1.FastShortMACDWindowLength, parent2.FastShortMACDWindowLength),
		FastLongMACDWindowLength:   selectValue(parent1.FastLongMACDWindowLength, parent2.FastLongMACDWindowLength),
		FastMACDSignalWindow:       selectValue(parent1.FastMACDSignalWindow, parent2.FastMACDSignalWindow),
		BollingerBandsWindow:       selectValue(parent1.BollingerBandsWindow, parent2.BollingerBandsWindow),
		BollingerBandsMultiplier:   selectValue(parent1.BollingerBandsMultiplier, parent2.BollingerBandsMultiplier),
		StochasticOscillatorWindow: selectValue(parent1.StochasticOscillatorWindow, parent2.StochasticOscillatorWindow),
		SlowATRPeriod:              selectValue(parent1.SlowATRPeriod, parent2.SlowATRPeriod),
		FastATRPeriod:              selectValue(parent1.FastATRPeriod, parent2.FastATRPeriod),
		OBVMovingAverageLength:     selectValue(parent1.OBVMovingAverageLength, parent2.OBVMovingAverageLength),
		VolumesMovingAverageLength: selectValue(parent1.VolumesMovingAverageLength, parent2.VolumesMovingAverageLength),
		ChaikinMoneyFlowPeriod:     selectValue(parent1.ChaikinMoneyFlowPeriod, parent2.ChaikinMoneyFlowPeriod),
		MoneyFlowIndexPeriod:       selectValue(parent1.MoneyFlowIndexPeriod, parent2.MoneyFlowIndexPeriod),
		RateOfChangePeriod:         selectValue(parent1.RateOfChangePeriod, parent2.RateOfChangePeriod),
		CCIPeriod:                  selectValue(parent1.CCIPeriod, parent2.CCIPeriod),
		WilliamsRPeriod:            selectValue(parent1.WilliamsRPeriod, parent2.WilliamsRPeriod),
		PriceChangeFastPeriod:      selectValue(parent1.PriceChangeFastPeriod, parent2.PriceChangeFastPeriod),
		PriceChangeMediumPeriod:    selectValue(parent1.PriceChangeMediumPeriod, parent2.PriceChangeMediumPeriod),
		PriceChangeSlowPeriod:      selectValue(parent1.PriceChangeSlowPeriod, parent2.PriceChangeSlowPeriod),
		RSIUpperBound:              selectValue(parent1.RSIUpperBound, parent2.RSIUpperBound),
		RSILowerBound:              selectValue(parent1.RSILowerBound, parent2.RSILowerBound),
		RSISlope:                   selectValue(parent1.RSISlope, parent2.RSISlope),

		L2Penalty:   selectValue(parent1.L2Penalty, parent2.L2Penalty),
		DropoutRate: selectValue(parent1.DropoutRate, parent2.DropoutRate),
		LearnRate:   selectValue(parent1.LearnRate, parent2.LearnRate),
		TrainDays:   selectValue(parent1.TrainDays, parent2.TrainDays),
	}
}

// Mutation (Introduce small variations)
func mutate(s *Strategy, mutationRate float64) {
	if rand.Float64() < mutationRate {
		randomizeStrategy(s, 5)
	}
}

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
	// Initialize random population
	population := make([]Strategy, popSize)
	population[0] = newStrategy(instrument)
	for i := 1; i < popSize; i++ {
		population[i] = newStrategy(instrument)
		randomizeStrategy(&population[i], 25)
	}

	for gen := range generations {
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

		maxFloats := func(v []float64) float64 {
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
		minFloats := func(v []float64) float64 {
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

		t := table.NewWriter()
		t.SetOutputMirror(os.Stdout)
		t.SetTitle(fmt.Sprintf("Generation %d - Summary", gen))
		t.AppendHeader(table.Row{"", "MEAN", "MIN", "MAX", "STDDEV"})
		t.AppendRows([]table.Row{
			{"Fitness", fmt.Sprintf("%0.6f", stat.Mean(fitnesses, nil)), fmt.Sprintf("%0.6f", minFloats(fitnesses)), fmt.Sprintf("%0.6f", maxFloats(fitnesses)), fmt.Sprintf("%0.6f", stat.StdDev(fitnesses, nil))},
			{"PnL", fmt.Sprintf("%0.2f%%", stat.Mean(pnls, nil)), fmt.Sprintf("%0.2f%%", minFloats(pnls)), fmt.Sprintf("%0.2f%%", maxFloats(pnls)), fmt.Sprintf("%0.6f", stat.StdDev(pnls, nil))},
			{"Max Drawdown", fmt.Sprintf("%0.2f%%", stat.Mean(maxDrawdowns, nil)), fmt.Sprintf("%0.2f%%", minFloats(maxDrawdowns)), fmt.Sprintf("%0.2f%%", maxFloats(maxDrawdowns)), fmt.Sprintf("%0.6f", stat.StdDev(maxDrawdowns, nil))},
			{"Sharpe Ratio", fmt.Sprintf("%0.2f", stat.Mean(sharpes, nil)), fmt.Sprintf("%0.2f", minFloats(sharpes)), fmt.Sprintf("%0.2f", maxFloats(sharpes)), fmt.Sprintf("%0.6f", stat.StdDev(sharpes, nil))},
			{"Sortino Ratio", fmt.Sprintf("%0.2f", stat.Mean(sortinos, nil)), fmt.Sprintf("%0.2f", minFloats(sortinos)), fmt.Sprintf("%0.2f", maxFloats(sortinos)), fmt.Sprintf("%0.6f", stat.StdDev(sortinos, nil))},
			{"Trades", fmt.Sprintf("%0.2f", stat.Mean(trades, nil)), fmt.Sprintf("%0.2f", minFloats(trades)), fmt.Sprintf("%0.2f", maxFloats(trades)), fmt.Sprintf("%0.6f", stat.StdDev(trades, nil))},
		})
		t.AppendSeparator()
		t.AppendRows([]table.Row{
			{"Train Days", fmt.Sprintf("%0.2f", stat.Mean(trainDays, nil)), fmt.Sprintf("%0.2f", minFloats(trainDays)), fmt.Sprintf("%0.2f", maxFloats(trainDays)), fmt.Sprintf("%0.6f", stat.StdDev(trainDays, nil))},
		})
		t.Render()

		params := StrategyToParams(strategy)
		params.Write(os.Stdout, fmt.Sprintf("Generation %d - Best Strategy", gen), false)
		strategy.ModelMetrics.Write(os.Stdout)
	}

	return population[0] // Return the best-performing strategy
}
