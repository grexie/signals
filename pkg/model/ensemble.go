package model

import (
	"context"
	"fmt"
	"log"
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

func NewEnsembleModel(ctx context.Context, db *leveldb.DB, instrument string, frequency time.Duration, count int) (*EnsembleModel, error) {
	now := time.Now()

	log.Printf("creating ensemble with %d active generations with duration %s...", count, frequency.String())

	e := &EnsembleModel{
		Models:     []*Model{},
		Timestamps: []time.Time{},
		Frequency:  frequency,
	}

	log.Printf("training model: generation %d", 1)
	timestamp := now.Add(time.Duration(-count-1) * frequency)
	if err := e.AddModel(ctx, db, instrument, frequency, timestamp); err != nil {
		return nil, err
	}

	go func() {
		for i := range count {
			if i == 0 {
				continue
			}
			log.Printf("training model: generation %d", i+1)
			timestamp := now.Add(time.Duration(i-count-1) * frequency)
			e.AddModel(ctx, db, instrument, frequency, timestamp)
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
					e.AddModel(ctx, db, instrument, frequency, now)
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

func (e *EnsembleModel) AddModel(ctx context.Context, db *leveldb.DB, instrument string, frequency time.Duration, timestamp time.Time) error {
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

	if m, err := NewModel(ctx, pw, db, instrument, timestamp.AddDate(0, -1, 0), timestamp); err != nil {
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

type StrategyVotes map[Strategy]int

func (e *EnsembleModel) Predict(ctx context.Context, now time.Time) (Strategy, StrategyVotes, error) {
	e.mutex.Lock()
	models := append([]*Model{}, e.Models...)
	e.mutex.Unlock()

	votes := NewStrategyVotes()

	feature := []float64(nil)
	for _, m := range models {
		f, s, err := m.Predict(ctx, feature, now)
		if err != nil {
			return StrategyHold, votes, err
		}
		feature = f

		votes.Vote(s)
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
	maxVotes := 0
	maxVotesStrategy := StrategyHold
	for s, v := range s {
		if maxVotes < v {
			maxVotes = v
			maxVotesStrategy = s
		}
	}
	return maxVotesStrategy
}

func (s StrategyVotes) Vote(strategy Strategy) {
	s[strategy]++
}

func (s StrategyVotes) String() string {
	total := float64(s[StrategyHold] + s[StrategyLong] + s[StrategyShort])
	if total == 0 {
		return "[no votes]"
	}
	return fmt.Sprintf("[Hold: %0.02f%%, Long: %0.02f%%, Short: %0.02f%%]", float64(s[StrategyHold])/total*100, float64(s[StrategyLong])/total*100, float64(s[StrategyShort])/total*100)
}
