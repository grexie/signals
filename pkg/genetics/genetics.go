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
	}
}

func randomizeStrategy(s *Strategy, percent float64) {
	s.WindowSize = model.BoundWindowSizeFloat64(s.WindowSize * randPercent(percent))
	s.Candles = model.BoundCandlesFloat64(s.Candles * randPercent(percent))
	s.TakeProfit = model.BoundTakeProfit(s.TakeProfit * randPercent(percent))
	s.StopLoss = model.BoundStopLoss(s.StopLoss * randPercent(percent))

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
}

// Evaluate fitness by composing a new model from the strategy
func evaluateFitness(ctx context.Context, pw progress.Writer, db *leveldb.DB, now time.Time, s Strategy) *model.ModelMetrics {
	leverage := model.Leverage()
	commission := model.Commission()
	params := model.ModelParams{
		WindowSize:      int(s.WindowSize),
		StrategyCandles: int(s.Candles),
		StrategyLong:    s.TakeProfit / leverage,
		StrategyShort:   s.TakeProfit / leverage,
		StrategyHold:    s.StopLoss / leverage,
		TradeCommission: commission,

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
	}

	if m, err := model.NewModel(ctx, pw, db, s.Instrument, params, now.AddDate(0, -1, 0), now, false); err != nil {
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
		Instrument:                 parent1.Instrument,
		WindowSize:                 selectValue(parent1.WindowSize, parent2.WindowSize),
		Candles:                    selectValue(parent1.Candles, parent2.Candles),
		TakeProfit:                 selectValue(parent1.TakeProfit, parent2.TakeProfit),
		StopLoss:                   selectValue(parent1.StopLoss, parent2.StopLoss),
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
		newPopulation := make([]Strategy, 0, popSize)
		for s := range results {
			newPopulation = append(newPopulation, s)
			fitnesses = append(fitnesses, s.ModelMetrics.Fitness())
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
		t.SetTitle(fmt.Sprintf("Generation %d", gen))
		t.AppendRows([]table.Row{
			{"SIGNALS_INSTRUMENT", strategy.Instrument},
			{"SIGNALS_WINDOW_SIZE", fmt.Sprintf("%.0f", strategy.WindowSize)},
			{"SIGNALS_CANDLES", fmt.Sprintf("%.0f", strategy.Candles)},
			{"SIGNALS_TAKE_PROFIT", fmt.Sprintf("%.04f", strategy.TakeProfit)},
			{"SIGNALS_STOP_LOSS", fmt.Sprintf("%.04f", strategy.StopLoss)},
		})
		t.AppendSeparator()
		t.AppendRows([]table.Row{
			{"SIGNALS_SHORT_MOVING_AVERAGE_LENGTH", fmt.Sprintf("%0.0f", strategy.ShortMovingAverageLength)},
			{"SIGNALS_LONG_MOVING_AVERAGE_LENGTH", fmt.Sprintf("%0.0f", strategy.LongMovingAverageLength)},
			{"SIGNALS_LONG_RSI_LENGTH", fmt.Sprintf("%0.0f", strategy.LongRSILength)},
			{"SIGNALS_SHORT_RSI_LENGTH", fmt.Sprintf("%0.0f", strategy.ShortRSILength)},
			{"SIGNALS_SHORT_MACD_WINDOW_LENGTH", fmt.Sprintf("%0.0f", strategy.ShortMACDWindowLength)},
			{"SIGNALS_LONG_MACD_WINDOW_LENGTH", fmt.Sprintf("%0.0f", strategy.LongMACDWindowLength)},
			{"SIGNALS_MACD_SIGNAL_WINDOW", fmt.Sprintf("%0.0f", strategy.MACDSignalWindow)},
			{"SIGNALS_FAST_SHORT_MACD_WINDOW_LENGTH", fmt.Sprintf("%0.0f", strategy.FastShortMACDWindowLength)},
			{"SIGNALS_FAST_LONG_MACD_WINDOW_LENGTH", fmt.Sprintf("%0.0f", strategy.FastLongMACDWindowLength)},
			{"SIGNALS_FAST_MACD_SIGNAL_WINDOW", fmt.Sprintf("%0.0f", strategy.FastMACDSignalWindow)},
			{"SIGNALS_BOLLINGER_BANDS_WINDOW", fmt.Sprintf("%0.0f", strategy.BollingerBandsWindow)},
			{"SIGNALS_BOLLINGER_BANDS_MULTIPLIER", fmt.Sprintf("%0.02f", strategy.BollingerBandsMultiplier)},
			{"SIGNALS_STOCHASTIC_OSCILLATOR_WINDOW", fmt.Sprintf("%0.0f", strategy.StochasticOscillatorWindow)},
			{"SIGNALS_SLOW_ATR_PERIOD_WINDOW", fmt.Sprintf("%0.0f", strategy.SlowATRPeriod)},
			{"SIGNALS_FAST_ATR_PERIOD_WINDOW", fmt.Sprintf("%0.0f", strategy.FastATRPeriod)},
			{"SIGNALS_OBV_MOVING_AVERAGE_LENGTH", fmt.Sprintf("%0.0f", strategy.OBVMovingAverageLength)},
			{"SIGNALS_VOLUMES_MOVING_AVERAGE_LENGTH", fmt.Sprintf("%0.0f", strategy.VolumesMovingAverageLength)},
			{"SIGNALS_CHAIKIN_MONEY_FLOW_PERIOD", fmt.Sprintf("%0.0f", strategy.ChaikinMoneyFlowPeriod)},
			{"SIGNALS_MONEY_FLOW_INDEX_PERIOD", fmt.Sprintf("%0.0f", strategy.MoneyFlowIndexPeriod)},
			{"SIGNALS_RATE_OF_CHANGE_PERIOD", fmt.Sprintf("%0.0f", strategy.RateOfChangePeriod)},
			{"SIGNALS_CCI_PERIOD", fmt.Sprintf("%0.0f", strategy.CCIPeriod)},
			{"SIGNALS_WILLIAMS_R_PERIOD", fmt.Sprintf("%0.0f", strategy.WilliamsRPeriod)},
			{"SIGNALS_PRICE_CHANGE_FAST_PERIOD", fmt.Sprintf("%0.0f", strategy.PriceChangeFastPeriod)},
			{"SIGNALS_PRICE_CHANGE_MEDIUM_PERIOD", fmt.Sprintf("%0.0f", strategy.PriceChangeMediumPeriod)},
			{"SIGNALS_PRICE_CHANGE_SLOW_PERIOD", fmt.Sprintf("%0.0f", strategy.PriceChangeSlowPeriod)},
			{"SIGNALS_RSI_UPPER_BOUND", fmt.Sprintf("%0.02f", strategy.RSIUpperBound)},
			{"SIGNALS_RSI_LOWER_BOUND", fmt.Sprintf("%0.02f", strategy.RSILowerBound)},
			{"SIGNALS_RSI_SLOPE", fmt.Sprintf("%0.0f", strategy.RSISlope)},
		})
		t.Render()

		t = table.NewWriter()
		t.SetOutputMirror(os.Stdout)
		t.SetTitle(fmt.Sprintf("Generation %d - Summary", gen))
		t.AppendHeader(table.Row{"", "MEAN", "STDDEV"})
		t.AppendRows([]table.Row{
			{"Fitness", fmt.Sprintf("%0.04f", stat.Mean(fitnesses, nil)), fmt.Sprintf("%0.04f", stat.StdDev(fitnesses, nil))},
		})
		t.Render()

		strategy.ModelMetrics.Write(os.Stdout)
	}

	return population[0] // Return the best-performing strategy
}
