package genetics

import (
	"context"
	"time"

	"github.com/grexie/signals/pkg/model"
	"github.com/jedib0t/go-pretty/v6/progress"
	"github.com/syndtr/goleveldb/leveldb"
)

// Evaluate fitness by composing a new model from the strategy
func evaluateFitness(ctx context.Context, pw progress.Writer, db *leveldb.DB, now time.Time, s Strategy) *model.ModelMetrics {
	params := StrategyToParams(s)

	if m, err := model.NewModel(ctx, pw, db, s.Instrument, params, now); err != nil {
		return &model.ModelMetrics{}
	} else {
		return &m.Metrics
	}
}
