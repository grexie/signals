package model

import (
	"fmt"
	"io"
	"math"

	"github.com/jedib0t/go-pretty/v6/table"
)

type ModelMetrics struct {
	Accuracy        float64
	ConfusionMatrix [][]float64
	ClassPrecision  []float64
	ClassRecall     []float64
	F1Scores        []float64

	Samples []int

	Backtest DeepBacktestMetrics
}

func safeValue(v float64, def float64) float64 {
	if math.IsNaN(v) || math.IsInf(v, 0) {
		return def
	} else {
		return v
	}
}

func (m *ModelMetrics) Fitness() float64 {
	avgF1 := (m.F1Scores[0] + m.F1Scores[1] + m.F1Scores[2]) / 300

	// Offset tanh for smooth scaling (range: ~0.1 to 1.0)
	normPnL := 0.5 + 0.5*math.Tanh(safeValue(m.Backtest.Mean.PnL, 0)/50)
	sharpe := 0.5 + 0.5*math.Tanh(safeValue(m.Backtest.Mean.SharpeRatio, 0)/3)
	sortino := 0.5 + 0.5*math.Tanh(safeValue(m.Backtest.Mean.SortinoRatio, 0)/3)

	// Drawdown penalty (range: ~0.1 to 1.0)
	drawdownPenalty := 0.1 + 0.9*math.Exp(-safeValue(m.Backtest.Min.MaxDrawdown, 0)/25)

	// Variance penalty (range: ~0.2 to 1.0)
	variancePenalty := 1.0 / (1.0 + safeValue(m.Backtest.StdDev.PnL, 0)/10)

	// Trade Factor: Encourages balanced trading (range: ~0.5 to 1.5)
	tradeFactor := 0.5 + 1.0*math.Tanh(safeValue(m.Backtest.Mean.Trades, 0)*0.05)

	// Risk-Adjusted Return Modifier (range: ~0.8 to 1.2)
	riskRewardFactor := 0.8 + 0.4*math.Tanh((safeValue(m.Backtest.Mean.PnL, 0)/math.Max(safeValue(m.Backtest.Mean.Trades, 1), 1))*0.1)

	// PnL Reward Factor: Increases fitness for profitable models (range: ~0.8 to 1.5)
	pnlReward := 0.8 + 0.7*math.Exp(safeValue(m.Backtest.Mean.PnL, 0)/100)

	// Base fitness calculation (weighted sum)
	fitness := (avgF1 * 0.25) + (sortino * 0.25) + (sharpe * 0.2) + (normPnL * 0.3)

	// Apply smooth multipliers
	fitness *= drawdownPenalty
	fitness *= tradeFactor
	fitness *= variancePenalty
	fitness *= riskRewardFactor
	fitness *= pnlReward

	// Extreme penalty for full account wipeouts (range: 0.05 to 1.0)
	if m.Backtest.Min.MaxDrawdown >= 99.5 {
		fitness *= 0.05
	} else if m.Backtest.Min.MaxDrawdown >= 95 {
		fitness *= 0.2
	}

	// ✅ Ensure smooth scaling by adding a small **positive offset**
	fitness = safeValue(fitness+0.00001, 0.00001)

	return fitness
}

func (m ModelMetrics) Write(w io.Writer) error {
	t := table.NewWriter()
	t.SetOutputMirror(w)
	t.SetTitle("Confusion Matrix")
	t.AppendHeader(table.Row{"", "HOLD", "LONG", "SHORT"})
	for i := range 3 {
		var label string
		switch i {
		case 0:
			label = "HOLD"
		case 1:
			label = "LONG"
		case 2:
			label = "SHORT"
		}

		rowTotal := float64(m.ConfusionMatrix[i][0] + m.ConfusionMatrix[i][1] + m.ConfusionMatrix[i][2])
		holdPercent := float64(m.ConfusionMatrix[i][0]) / rowTotal * 100
		longPercent := float64(m.ConfusionMatrix[i][1]) / rowTotal * 100
		shortPercent := float64(m.ConfusionMatrix[i][2]) / rowTotal * 100

		if rowTotal == 0 {
			t.AppendRows([]table.Row{
				{label, "", "", ""},
			})
		} else {
			t.AppendRows([]table.Row{
				{label, fmt.Sprintf("%6.2f%%", holdPercent), fmt.Sprintf("%6.2f%%", longPercent), fmt.Sprintf("%6.2f%%", shortPercent)},
			})
		}

	}
	t.AppendFooter(table.Row{"ACCURACY", "", "", fmt.Sprintf("%0.02f%%", m.Accuracy)})

	t.Render()

	t = table.NewWriter()
	t.SetOutputMirror(w)
	t.SetTitle("Class Metrics")
	t.AppendHeader(table.Row{"CLASS", "PRECISION", "RECALL", "F1 SCORE", "SAMPLES"})
	t.AppendRows([]table.Row{
		{"HOLD", fmt.Sprintf("%6.2f%%", m.ClassPrecision[0]), fmt.Sprintf("%6.2f%%", m.ClassRecall[0]), fmt.Sprintf("%6.2f%%", m.F1Scores[0]), fmt.Sprintf("%d", m.Samples[0])},
		{"LONG", fmt.Sprintf("%6.2f%%", m.ClassPrecision[1]), fmt.Sprintf("%6.2f%%", m.ClassRecall[1]), fmt.Sprintf("%6.2f%%", m.F1Scores[1]), fmt.Sprintf("%d", m.Samples[1])},
		{"SHORT", fmt.Sprintf("%6.2f%%", m.ClassPrecision[2]), fmt.Sprintf("%6.2f%%", m.ClassRecall[2]), fmt.Sprintf("%6.2f%%", m.F1Scores[2]), fmt.Sprintf("%d", m.Samples[2])},
	})
	t.AppendSeparator()
	t.AppendRows([]table.Row{
		{"", fmt.Sprintf("%6.2f%%", (m.ClassPrecision[0]+m.ClassPrecision[1]+m.ClassPrecision[2])/3), fmt.Sprintf("%6.2f%%", (m.ClassRecall[0]+m.ClassRecall[1]+m.ClassRecall[2])/3), fmt.Sprintf("%6.2f%%", (m.F1Scores[0]+m.F1Scores[1]+m.F1Scores[2])/3), fmt.Sprintf("%d", m.Samples[0]+m.Samples[1]+m.Samples[2])},
	})
	t.Render()

	t = table.NewWriter()
	t.SetOutputMirror(w)
	t.SetTitle("Trading Metrics")
	t.AppendHeader(table.Row{"", "MEAN", "MIN", "MAX", "STDDEV"})
	t.AppendRows([]table.Row{
		{"PnL", fmt.Sprintf("%6.2f%%", m.Backtest.Mean.PnL), fmt.Sprintf("%6.2f%%", m.Backtest.Min.PnL), fmt.Sprintf("%6.2f%%", m.Backtest.Max.PnL), fmt.Sprintf("%6.2f", m.Backtest.StdDev.PnL)},
		{"Max Drawdown", fmt.Sprintf("%6.2f%%", m.Backtest.Mean.MaxDrawdown), fmt.Sprintf("%6.2f%%", m.Backtest.Min.MaxDrawdown), fmt.Sprintf("%6.2f%%", m.Backtest.Max.MaxDrawdown), fmt.Sprintf("%6.2f", m.Backtest.StdDev.MaxDrawdown)},
		{"Sharpe Ratio", fmt.Sprintf("%6.2f", m.Backtest.Mean.SharpeRatio), fmt.Sprintf("%6.2f", m.Backtest.Min.SharpeRatio), fmt.Sprintf("%6.2f", m.Backtest.Max.SharpeRatio), fmt.Sprintf("%6.2f", m.Backtest.StdDev.SharpeRatio)},
		{"Sortino Ratio", fmt.Sprintf("%6.2f", m.Backtest.Mean.SortinoRatio), fmt.Sprintf("%6.2f", m.Backtest.Min.SortinoRatio), fmt.Sprintf("%6.2f", m.Backtest.Max.SortinoRatio), fmt.Sprintf("%6.2f", m.Backtest.StdDev.SortinoRatio)},
		{"Trades", fmt.Sprintf("%6.2f", m.Backtest.Mean.Trades), fmt.Sprintf("%6.2f", m.Backtest.Min.Trades), fmt.Sprintf("%6.2f", m.Backtest.Max.Trades), fmt.Sprintf("%6.2f", m.Backtest.StdDev.Trades)},
	})
	t.AppendSeparator()
	t.AppendRow(table.Row{"Fitness", fmt.Sprintf("%.6f", m.Fitness())})
	t.Render()

	return nil
}

func calculateMetrics(confusionMatrix [][]int, total int) ModelMetrics {
	numClasses := len(confusionMatrix)
	metrics := ModelMetrics{
		ConfusionMatrix: make([][]float64, numClasses),
		ClassPrecision:  make([]float64, numClasses),
		ClassRecall:     make([]float64, numClasses),
		F1Scores:        make([]float64, numClasses),
		Samples:         make([]int, numClasses),
	}

	// Calculate confusion matrix percentages
	classTotals := make([]int, numClasses)
	for i := range numClasses {
		metrics.ConfusionMatrix[i] = make([]float64, numClasses)
		for j := 0; j < numClasses; j++ {
			classTotals[i] += confusionMatrix[i][j]
		}
		for j := 0; j < numClasses; j++ {
			if classTotals[i] > 0 {
				metrics.ConfusionMatrix[i][j] = float64(confusionMatrix[i][j]) / float64(classTotals[i]) * 100
			}
		}
		metrics.Samples[i] = confusionMatrix[i][i]
	}

	// Calculate precision and recall for each class
	for i := 0; i < numClasses; i++ {
		truePositives := confusionMatrix[i][i]
		falsePositives := 0
		falseNegatives := 0

		for j := 0; j < numClasses; j++ {
			if i != j {
				falsePositives += confusionMatrix[j][i]
				falseNegatives += confusionMatrix[i][j]
			}
		}

		// Calculate precision
		if truePositives+falsePositives > 0 {
			metrics.ClassPrecision[i] = float64(truePositives) / float64(truePositives+falsePositives) * 100
		}

		// Calculate recall
		if truePositives+falseNegatives > 0 {
			metrics.ClassRecall[i] = float64(truePositives) / float64(truePositives+falseNegatives) * 100
		}

		// Calculate F1 score
		if metrics.ClassPrecision[i]+metrics.ClassRecall[i] > 0 {
			metrics.F1Scores[i] = 2 * (metrics.ClassPrecision[i] * metrics.ClassRecall[i]) /
				(metrics.ClassPrecision[i] + metrics.ClassRecall[i])
		}
	}

	// Calculate overall accuracy
	correct := 0
	for i := range numClasses {
		correct += confusionMatrix[i][i]
	}
	metrics.Accuracy = float64(correct) / float64(total) * 100

	return metrics
}
