package model

import (
	"context"
	"fmt"
	"log"
	"math"
	"os"
	"slices"
	"sync"
	"time"

	"github.com/jedib0t/go-pretty/v6/progress"
	"github.com/syndtr/goleveldb/leveldb"
)

type EnsembleModel struct {
	mutex      sync.Mutex
	Models     []*Model
	Timestamps []time.Time
	Frequency  time.Duration
}

func NewEnsembleModel(ctx context.Context, db *leveldb.DB, instrument string, params ModelParams, frequency time.Duration, count int) (*EnsembleModel, error) {
	now := time.Now()

	log.Printf("creating ensemble with %d active generations with duration %s...", count, frequency.String())

	e := &EnsembleModel{
		Models:     []*Model{},
		Timestamps: []time.Time{},
		Frequency:  frequency,
	}

	log.Printf("training model: generation %d", 1)
	timestamp := now.Add(time.Duration(-count-1) * frequency)
	if err := e.AddModel(ctx, db, instrument, params, frequency, timestamp); err != nil {
		return nil, err
	}

	go func() {
		for i := range count {
			if i == 0 {
				continue
			}
			log.Printf("training model: generation %d", i+1)
			timestamp := now.Add(time.Duration(i-count-1) * frequency)
			e.AddModel(ctx, db, instrument, params, frequency, timestamp)
		}

		go func() {
			generation := count
			for {
				select {
				case <-ctx.Done():
					return
				case <-time.After(time.Until(now.Add(frequency))):
					log.Printf("training model: generation %d", generation+1)
					generation++
					now = now.Add(frequency)
					e.mutex.Lock()
					slices.SortFunc(e.Models, func(a *Model, b *Model) int {
						aw := 6*(math.Tanh(a.Metrics.Backtest.Mean.SharpeRatio/3)+1) + 12*(math.Tanh(a.Metrics.Backtest.Mean.SortinoRatio/3)+1)
						bw := 6*(math.Tanh(b.Metrics.Backtest.Mean.SharpeRatio/3)+1) + 12*(math.Tanh(b.Metrics.Backtest.Mean.SortinoRatio/3)+1)
						if aw < bw {
							return -1
						}
						if aw > bw {
							return 1
						}
						return 0
					})
					e.mutex.Unlock()
					e.AddModel(ctx, db, instrument, params, frequency, now)
					e.EvictModel(0)
				}
			}
		}()
	}()

	return e, nil
}

func (e *EnsembleModel) EvictModel(index int) {
	e.mutex.Lock()
	defer e.mutex.Unlock()

	ts := e.Timestamps[0]

	e.Models = slices.Delete(e.Models, index, index+1)
	e.Timestamps = slices.Delete(e.Timestamps, index, index+1)

	log.Printf("evicted model with timestamp %s, %d generations running", ts, len(e.Models))
}

func (e *EnsembleModel) AddModel(ctx context.Context, db *leveldb.DB, instrument string, params ModelParams, frequency time.Duration, timestamp time.Time) error {
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

	if m, err := NewModel(ctx, pw, db, instrument, params, timestamp.AddDate(0, -1, 0), timestamp, true); err != nil {
		return err
	} else {
		pw.Stop()
		for pw.IsRenderInProgress() {
			time.Sleep(100 * time.Millisecond)
		}

		m.Metrics.Write(os.Stdout)

		e.mutex.Lock()
		defer e.mutex.Unlock()

		e.Timestamps = append(e.Timestamps, timestamp)
		e.Models = append(e.Models, m)

		return nil
	}
}

type StrategyVotes map[Strategy]float64

func (e *EnsembleModel) Predict(pw progress.Writer, now time.Time) (Strategy, StrategyVotes, error) {
	e.mutex.Lock()
	models := append([]*Model{}, e.Models...)
	e.mutex.Unlock()

	votes := NewStrategyVotes()

	feature := []float64(nil)
	for _, m := range models {
		f, prediction, err := m.Predict(pw, feature, now)
		if err != nil {
			return StrategyHold, votes, err
		}
		feature = f

		weight := 6*(math.Tanh(m.Metrics.Backtest.Mean.SharpeRatio/3)+1) + 12*(math.Tanh(m.Metrics.Backtest.Mean.SortinoRatio/3)+1)
		for s, v := range prediction {
			votes.Vote(s, v*weight)
		}
	}

	return votes.Strategy(), votes, nil
}

func NewStrategyVotes() StrategyVotes {
	return StrategyVotes{
		StrategyHold:  0,
		StrategyLong:  0,
		StrategyShort: 0,
	}
}

func (s StrategyVotes) Strategy() Strategy {
	totalVotes := float64(0)
	for _, v := range s {
		totalVotes += v
	}

	p := MinTradeProbability()

	if s[StrategyLong] > totalVotes*p && s[StrategyShort] < totalVotes*p {
		return StrategyLong
	} else if s[StrategyShort] > totalVotes*p && s[StrategyLong] < totalVotes*p {
		return StrategyShort
	} else {
		return StrategyHold
	}
}

func (s StrategyVotes) Vote(strategy Strategy, votes float64) {
	s[strategy] += votes
}

func (s StrategyVotes) String() string {
	total := float64(s[StrategyHold] + s[StrategyLong] + s[StrategyShort])
	if total == 0 {
		return "[no votes]"
	}
	return fmt.Sprintf("[Hold: %0.02f%%, Long: %0.02f%%, Short: %0.02f%%]", float64(s[StrategyHold])/total*100, float64(s[StrategyLong])/total*100, float64(s[StrategyShort])/total*100)
}
