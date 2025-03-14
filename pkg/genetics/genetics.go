package genetics

import (
	"context"
	"fmt"
	"math/rand/v2"
	"os"
	"sort"
	"sync"
	"time"

	"github.com/grexie/signals/pkg/model"
	"github.com/jedib0t/go-pretty/v6/progress"
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/syndtr/goleveldb/leveldb"
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

	ModelMetrics *model.ModelMetrics
}

func randPercent(dev float64) float64 {
	return 1 + (rand.Float64()*(2*dev)-dev)/100
}

// Generate a random strategy
func randomStrategy(instrument string) Strategy {
	return Strategy{
		Instrument: instrument,

		WindowSize: model.BoundWindowSizeFloat64(float64(model.WindowSize()) * randPercent(5)),
		Candles:    model.BoundCandlesFloat64(float64(model.Candles()) * randPercent(5)),
		StopLoss:   model.BoundStopLoss(model.StopLoss() * randPercent(5)),
		TakeProfit: model.BoundTakeProfit(model.TakeProfit() * randPercent(5)),

		ShortMovingAverageLength:   model.BoundShortMovingAverageLengthFloat64(float64(model.ShortMovingAverageLength()) * randPercent(5)),
		LongMovingAverageLength:    model.BoundLongMovingAverageLengthFloat64(float64(model.LongMovingAverageLength()) * randPercent(5)),
		LongRSILength:              model.BoundLongRSILengthFloat64(float64(model.LongRSILength()) * randPercent(5)),
		ShortRSILength:             model.BoundShortRSILengthFloat64(float64(model.ShortRSILength()) * randPercent(5)),
		ShortMACDWindowLength:      model.BoundShortMACDWindowLengthFloat64(float64(model.ShortMACDWindowLength()) * randPercent(5)),
		LongMACDWindowLength:       model.BoundLongMACDWindowLengthFloat64(float64(model.LongMACDWindowLength()) * randPercent(5)),
		MACDSignalWindow:           model.BoundMACDSignalWindowFloat64(float64(model.MACDSignalWindow()) * randPercent(5)),
		FastShortMACDWindowLength:  model.BoundFastShortMACDWindowLengthFloat64(float64(model.FastShortMACDWindowLength()) * randPercent(5)),
		FastLongMACDWindowLength:   model.BoundFastLongMACDWindowLengthFloat64(float64(model.FastLongMACDWindowLength()) * randPercent(5)),
		FastMACDSignalWindow:       model.BoundFastMACDSignalWindowFloat64(float64(model.FastMACDSignalWindow()) * randPercent(5)),
		BollingerBandsWindow:       model.BoundBollingerBandsWindowFloat64(float64(model.BollingerBandsWindow()) * randPercent(5)),
		BollingerBandsMultiplier:   model.BoundBollingerBandsMultiplier(float64(model.BollingerBandsMultiplier()) * randPercent(5)),
		StochasticOscillatorWindow: model.BoundStochasticOscillatorWindowFloat64(float64(model.StochasticOscillatorWindow()) * randPercent(5)),
		SlowATRPeriod:              model.BoundSlowATRPeriodFloat64(float64(model.SlowATRPeriod()) * randPercent(5)),
		FastATRPeriod:              model.BoundFastATRPeriodFloat64(float64(model.FastATRPeriod()) * randPercent(5)),
		OBVMovingAverageLength:     model.BoundOBVMovingAverageLengthFloat64(float64(model.OBVMovingAverageLength()) * randPercent(5)),
		VolumesMovingAverageLength: model.BoundVolumesMovingAverageLengthFloat64(float64(model.VolumesMovingAverageLength()) * randPercent(5)),
		ChaikinMoneyFlowPeriod:     model.BoundChaikinMoneyFlowPeriodFloat64(float64(model.ChaikinMoneyFlowPeriod()) * randPercent(5)),
		MoneyFlowIndexPeriod:       model.BoundMoneyFlowIndexPeriodFloat64(float64(model.MoneyFlowIndexPeriod()) * randPercent(5)),
		RateOfChangePeriod:         model.BoundRateOfChangePeriodFloat64(float64(model.RateOfChangePeriod()) * randPercent(5)),
		CCIPeriod:                  model.BoundCCIPeriodFloat64(float64(model.CCIPeriod()) * randPercent(5)),
		WilliamsRPeriod:            model.BoundWilliamsRPeriodFloat64(float64(model.WilliamsRPeriod()) * randPercent(5)),
		PriceChangeFastPeriod:      model.BoundPriceChangeFastPeriodFloat64(float64(model.PriceChangeFastPeriod()) * randPercent(5)),
		PriceChangeMediumPeriod:    model.BoundPriceChangeMediumPeriodFloat64(float64(model.PriceChangeMediumPeriod()) * randPercent(5)),
		PriceChangeSlowPeriod:      model.BoundPriceChangeSlowPeriodFloat64(float64(model.PriceChangeSlowPeriod()) * randPercent(5)),
		RSIUpperBound:              model.BoundRSIUpperBound(float64(model.RSIUpperBound()) * randPercent(5)),
		RSILowerBound:              model.BoundRSILowerBound(float64(model.RSILowerBound()) * randPercent(5)),
	}
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
	}

	if m, err := model.NewModel(ctx, pw, db, s.Instrument, params, now.AddDate(0, -1, 0), now, false); err != nil {
		return &model.ModelMetrics{}
	} else {
		return &m.Metrics
	}
}

// Selection (Choose the top performers)
func selection(population []Strategy, retainRate float64) []Strategy {
	n := int(float64(len(population)) * retainRate)
	elite := population[:n]

	// Stochastic selection for maintaining diversity
	roulette := make([]Strategy, 0, len(population))
	totalFitness := 0.0
	for _, s := range population {
		totalFitness += s.ModelMetrics.Accuracy
	}

	for _, s := range population[n:] {
		if rand.Float64() < (s.ModelMetrics.Accuracy / totalFitness) {
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

	return Strategy{
		Instrument:                 parent1.Instrument,
		WindowSize:                 (parent1.WindowSize + parent2.WindowSize) / 2,
		Candles:                    (parent1.Candles + parent2.Candles) / 2,
		TakeProfit:                 (parent1.TakeProfit + parent2.TakeProfit) / 2,
		StopLoss:                   (parent1.StopLoss + parent2.StopLoss) / 2,
		ShortMovingAverageLength:   (parent1.ShortMovingAverageLength + parent2.ShortMovingAverageLength) / 2,
		LongMovingAverageLength:    (parent1.LongMovingAverageLength + parent2.LongMovingAverageLength) / 2,
		LongRSILength:              (parent1.LongRSILength + parent2.LongRSILength) / 2,
		ShortRSILength:             (parent1.ShortRSILength + parent2.ShortRSILength) / 2,
		ShortMACDWindowLength:      (parent1.ShortMACDWindowLength + parent2.ShortMACDWindowLength) / 2,
		LongMACDWindowLength:       (parent1.LongMACDWindowLength + parent2.LongMACDWindowLength) / 2,
		MACDSignalWindow:           (parent1.MACDSignalWindow + parent2.MACDSignalWindow) / 2,
		FastShortMACDWindowLength:  (parent1.FastShortMACDWindowLength + parent2.FastShortMACDWindowLength) / 2,
		FastLongMACDWindowLength:   (parent1.FastLongMACDWindowLength + parent2.FastLongMACDWindowLength) / 2,
		FastMACDSignalWindow:       (parent1.FastMACDSignalWindow + parent2.FastMACDSignalWindow) / 2,
		BollingerBandsWindow:       (parent1.BollingerBandsWindow + parent2.BollingerBandsWindow) / 2,
		BollingerBandsMultiplier:   (parent1.BollingerBandsMultiplier + parent2.BollingerBandsMultiplier) / 2,
		StochasticOscillatorWindow: (parent1.StochasticOscillatorWindow + parent2.StochasticOscillatorWindow) / 2,
		SlowATRPeriod:              (parent1.SlowATRPeriod + parent2.SlowATRPeriod) / 2,
		FastATRPeriod:              (parent1.FastATRPeriod + parent2.FastATRPeriod) / 2,
		OBVMovingAverageLength:     (parent1.OBVMovingAverageLength + parent2.OBVMovingAverageLength) / 2,
		VolumesMovingAverageLength: (parent1.VolumesMovingAverageLength + parent2.VolumesMovingAverageLength) / 2,
		ChaikinMoneyFlowPeriod:     (parent1.ChaikinMoneyFlowPeriod + parent2.ChaikinMoneyFlowPeriod) / 2,
		MoneyFlowIndexPeriod:       (parent1.MoneyFlowIndexPeriod + parent2.MoneyFlowIndexPeriod) / 2,
		RateOfChangePeriod:         (parent1.RateOfChangePeriod + parent2.RateOfChangePeriod) / 2,
		CCIPeriod:                  (parent1.CCIPeriod + parent2.CCIPeriod) / 2,
		WilliamsRPeriod:            (parent1.WilliamsRPeriod + parent2.WilliamsRPeriod) / 2,
		PriceChangeFastPeriod:      (parent1.PriceChangeFastPeriod + parent2.PriceChangeFastPeriod) / 2,
		PriceChangeMediumPeriod:    (parent1.PriceChangeMediumPeriod + parent2.PriceChangeMediumPeriod) / 2,
		PriceChangeSlowPeriod:      (parent1.PriceChangeSlowPeriod + parent2.PriceChangeSlowPeriod) / 2,
		RSIUpperBound:              (parent1.RSIUpperBound + parent2.RSIUpperBound) / 2,
		RSILowerBound:              (parent1.RSILowerBound + parent2.RSILowerBound) / 2,
	}
}

// Mutation (Introduce small variations)
func mutate(s *Strategy, mutationRate float64) {
	if rand.Float64() < mutationRate {
		s.WindowSize = model.BoundWindowSizeFloat64(s.WindowSize * randPercent(2.5))
		s.Candles = model.BoundCandlesFloat64(s.Candles * randPercent(2.5))
		s.TakeProfit = model.BoundTakeProfit(s.TakeProfit * randPercent(2.5))
		s.StopLoss = model.BoundStopLoss(s.StopLoss * randPercent(2.5))

		s.ShortMovingAverageLength = model.BoundShortMovingAverageLengthFloat64(s.ShortMovingAverageLength * randPercent(2.5))
		s.LongMovingAverageLength = model.BoundLongMovingAverageLengthFloat64(s.LongMovingAverageLength * randPercent(2.5))
		s.LongRSILength = model.BoundLongRSILengthFloat64(s.LongRSILength * randPercent(2.5))
		s.ShortRSILength = model.BoundShortRSILengthFloat64(s.ShortRSILength * randPercent(2.5))
		s.ShortMACDWindowLength = model.BoundShortMACDWindowLengthFloat64(s.ShortMACDWindowLength * randPercent(2.5))
		s.LongMACDWindowLength = model.BoundLongMACDWindowLengthFloat64(s.LongMACDWindowLength * randPercent(2.5))
		s.MACDSignalWindow = model.BoundMACDSignalWindowFloat64(s.MACDSignalWindow * randPercent(2.5))
		s.FastShortMACDWindowLength = model.BoundFastShortMACDWindowLengthFloat64(s.FastShortMACDWindowLength * randPercent(2.5))
		s.FastLongMACDWindowLength = model.BoundFastLongMACDWindowLengthFloat64(s.FastLongMACDWindowLength * randPercent(2.5))
		s.FastMACDSignalWindow = model.BoundFastMACDSignalWindowFloat64(s.FastMACDSignalWindow * randPercent(2.5))
		s.BollingerBandsWindow = model.BoundBollingerBandsWindowFloat64(s.BollingerBandsWindow * randPercent(2.5))
		s.BollingerBandsMultiplier = model.BoundBollingerBandsMultiplier(s.BollingerBandsMultiplier * randPercent(2.5))
		s.StochasticOscillatorWindow = model.BoundStochasticOscillatorWindowFloat64(s.StochasticOscillatorWindow * randPercent(2.5))
		s.SlowATRPeriod = model.BoundSlowATRPeriodFloat64(s.SlowATRPeriod * randPercent(2.5))
		s.FastATRPeriod = model.BoundFastATRPeriodFloat64(s.FastATRPeriod * randPercent(2.5))
		s.OBVMovingAverageLength = model.BoundOBVMovingAverageLengthFloat64(s.OBVMovingAverageLength * randPercent(2.5))
		s.VolumesMovingAverageLength = model.BoundVolumesMovingAverageLengthFloat64(s.VolumesMovingAverageLength * randPercent(2.5))
		s.ChaikinMoneyFlowPeriod = model.BoundChaikinMoneyFlowPeriodFloat64(s.ChaikinMoneyFlowPeriod * randPercent(2.5))
		s.MoneyFlowIndexPeriod = model.BoundMoneyFlowIndexPeriodFloat64(s.MoneyFlowIndexPeriod * randPercent(2.5))
		s.RateOfChangePeriod = model.BoundRateOfChangePeriodFloat64(s.RateOfChangePeriod * randPercent(2.5))
		s.CCIPeriod = model.BoundCCIPeriodFloat64(s.CCIPeriod * randPercent(2.5))
		s.WilliamsRPeriod = model.BoundWilliamsRPeriodFloat64(s.WilliamsRPeriod * randPercent(2.5))
		s.PriceChangeFastPeriod = model.BoundPriceChangeFastPeriodFloat64(s.PriceChangeFastPeriod * randPercent(2.5))
		s.PriceChangeMediumPeriod = model.BoundPriceChangeMediumPeriodFloat64(s.PriceChangeMediumPeriod * randPercent(2.5))
		s.PriceChangeSlowPeriod = model.BoundPriceChangeSlowPeriodFloat64(s.PriceChangeSlowPeriod * randPercent(2.5))
		s.RSIUpperBound = model.BoundRSIUpperBound(s.RSIUpperBound * randPercent(2.5))
		s.RSILowerBound = model.BoundRSILowerBound(s.RSILowerBound * randPercent(2.5))
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
func NaturalSelection(db *leveldb.DB, instrument string, now time.Time, popSize, generations int, retainRate, mutationRate float64) Strategy {
	// Initialize random population
	population := make([]Strategy, popSize)
	for i := range population {
		population[i] = randomStrategy(instrument)
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
		numWorkers := 5 // Adjust based on available CPU cores
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
		newPopulation := make([]Strategy, 0, popSize)
		for s := range results {
			newPopulation = append(newPopulation, s)
		}

		tracker.MarkAsDone()

		pw.Stop()
		for pw.IsRenderInProgress() {
			time.Sleep(100 * time.Millisecond)
		}

		// Sort by fitness (higher is better)
		sort.Slice(newPopulation, func(i, j int) bool {
			return newPopulation[i].ModelMetrics.Accuracy > newPopulation[j].ModelMetrics.Accuracy
		})

		// Apply selection
		population = selection(newPopulation, retainRate)

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
			{"SIGNALS_RATE_OF_CHANGE_PERIOD", fmt.Sprintf("%0.0f", strategy.CCIPeriod)},
			{"SIGNALS_RATE_OF_CHANGE_PERIOD", fmt.Sprintf("%0.0f", strategy.WilliamsRPeriod)},
			{"SIGNALS_PRICE_CHANGE_FAST_PERIOD", fmt.Sprintf("%0.0f", strategy.PriceChangeFastPeriod)},
			{"SIGNALS_PRICE_CHANGE_MEDIUM_PERIOD", fmt.Sprintf("%0.0f", strategy.PriceChangeMediumPeriod)},
			{"SIGNALS_PRICE_CHANGE_SLOW_PERIOD", fmt.Sprintf("%0.0f", strategy.PriceChangeSlowPeriod)},
			{"SIGNALS_RSI_UPPER_BOUND", fmt.Sprintf("%0.02f", strategy.RSIUpperBound)},
			{"SIGNALS_RSI_LOWER_BOUND", fmt.Sprintf("%0.02f", strategy.RSILowerBound)},
		})
		t.Render()

		strategy.ModelMetrics.Write(os.Stdout)
	}

	return population[0] // Return the best-performing strategy
}
