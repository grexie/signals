package genetics

import (
	"encoding/csv"
	"fmt"

	"github.com/grexie/signals/pkg/model"
	"gonum.org/v1/gonum/stat"
)

func WriteCSVHeader(writer *csv.Writer) error {
	header := []string{
		"Generation",

		"Fitness (Mean)", "Fitness (Min)", "Fitness (Max)", "Fitness (StdDev)",
		"PnL (Mean)", "PnL (Min)", "PnL (Max)", "PnL (StdDev)",
		"Max Drawdown (Mean)", "Max Drawdown (Min)", "Max Drawdown (Max)", "Max Drawdown (StdDev)",
		"Sharpe Ratio (Mean)", "Sharpe Ratio (Min)", "Sharpe Ratio (Max)", "Sharpe Ratio (StdDev)",
		"Sortino Ratio (Mean)", "Sortino Ratio (Min)", "Sortino Ratio (Max)", "Sortino Ratio (StdDev)",
		"Trades (Mean)", "Trades (Min)", "Trades (Max)", "Trades (StdDev)",

		"Train Days (Mean)", "Train Days (Min)", "Train Days (Max)", "Train Days (StdDev)",

		"Fitness (Best Strategy)",

		"Accuracy (Best Strategy)",
		"Precision (Best Strategy)",
		"Recall (Best Strategy)",
		"F1 Score (Best Strategy)",
		"Samples (Best Strategy)",

		"PnL Mean (Best Strategy)", "PnL Min (Best Strategy)", "PnL Max (Best Strategy)", "PnL StdDev (Best Strategy)",
		"Max Drawdown Mean (Best Strategy)", "Max Drawdown Min (Best Strategy)", "Max Drawdown Max (Best Strategy)", "Max Drawdown StdDev (Best Strategy)",
		"Sharpe Ratio Mean (Best Strategy)", "Sharpe Ratio Min (Best Strategy)", "Sharpe Ratio Max (Best Strategy)", "Sharpe Ratio StdDev (Best Strategy)",
		"Sortino Ratio Mean (Best Strategy)", "Sortino Ratio Min (Best Strategy)", "Sortino Ratio Max (Best Strategy)", "Sortino Ratio StdDev (Best Strategy)",
		"Trades Mean (Best Strategy)", "Trades Min (Best Strategy)", "Trades Max (Best Strategy)", "Trades StdDev (Best Strategy)",

		"SIGNALS_INSTRUMENT (Best Strategy)",
		"SIGNALS_LEVERAGE (Best Strategy)",
		"SIGNALS_TRADE_MULTIPLIER (Best Strategy)",
		"SIGNALS_COMMISSION (Best Strategy)",
		"SIGNALS_COOLDOWN (Best Strategy)",

		"SIGNALS_WINDOW_SIZE (Best Strategy)",
		"SIGNALS_CANDLES (Best Strategy)",
		"SIGNALS_TAKE_PROFIT (Best Strategy)",
		"SIGNALS_STOP_LOSS (Best Strategy)",

		"SIGNALS_MIN_TRADE_PROBABILITY (Best Strategy)",

		"SIGNALS_L2_PENALTY (Best Strategy)",
		"SIGNALS_DROPOUT_RATE (Best Strategy)",
		"SIGNALS_LEARN_RATE (Best Strategy)",
		"SIGNALS_TRAIN_DAYS (Best Strategy)",

		"SIGNALS_SHORT_MOVING_AVERAGE_LENGTH (Best Strategy)",
		"SIGNALS_LONG_MOVING_AVERAGE_LENGTH (Best Strategy)",
		"SIGNALS_LONG_RSI_LENGTH (Best Strategy)",
		"SIGNALS_SHORT_RSI_LENGTH (Best Strategy)",
		"SIGNALS_SHORT_MACD_WINDOW_LENGTH (Best Strategy)",
		"SIGNALS_LONG_MACD_WINDOW_LENGTH (Best Strategy)",
		"SIGNALS_MACD_SIGNAL_WINDOW (Best Strategy)",
		"SIGNALS_FAST_SHORT_MACD_WINDOW_LENGTH (Best Strategy)",
		"SIGNALS_FAST_LONG_MACD_WINDOW_LENGTH (Best Strategy)",
		"SIGNALS_FAST_MACD_SIGNAL_WINDOW (Best Strategy)",
		"SIGNALS_BOLLINGER_BANDS_WINDOW (Best Strategy)",
		"SIGNALS_BOLLINGER_BANDS_MULTIPLIER (Best Strategy)",
		"SIGNALS_STOCHASTIC_OSCILLATOR_WINDOW (Best Strategy)",
		"SIGNALS_SLOW_ATR_PERIOD_WINDOW (Best Strategy)",
		"SIGNALS_FAST_ATR_PERIOD_WINDOW (Best Strategy)",
		"SIGNALS_OBV_MOVING_AVERAGE_LENGTH (Best Strategy)",
		"SIGNALS_VOLUMES_MOVING_AVERAGE_LENGTH (Best Strategy)",
		"SIGNALS_CHAIKIN_MONEY_FLOW_PERIOD (Best Strategy)",
		"SIGNALS_MONEY_FLOW_INDEX_PERIOD (Best Strategy)",
		"SIGNALS_RATE_OF_CHANGE_PERIOD (Best Strategy)",
		"SIGNALS_CCI_PERIOD (Best Strategy)",
		"SIGNALS_WILLIAMS_R_PERIOD (Best Strategy)",
		"SIGNALS_PRICE_CHANGE_FAST_PERIOD (Best Strategy)",
		"SIGNALS_PRICE_CHANGE_MEDIUM_PERIOD (Best Strategy)",
		"SIGNALS_PRICE_CHANGE_SLOW_PERIOD (Best Strategy)",
		"SIGNALS_RSI_UPPER_BOUND (Best Strategy)",
		"SIGNALS_RSI_LOWER_BOUND (Best Strategy)",
		"SIGNALS_RSI_SLOPE (Best Strategy)",
	}

	if err := writer.Write(header); err != nil {
		return err
	} else {
		writer.Flush()
		return nil
	}
}

func WriteCSVRow(writer *csv.Writer, generation int, fitnesses []float64, pnls []float64, maxDrawdowns []float64, sharpes []float64, sortinos []float64, trades []float64, trainDays []float64, params model.ModelParams, s *Strategy) error {
	row := []string{
		fmt.Sprintf("%d", generation),

		fmt.Sprintf("%0.6f", stat.Mean(fitnesses, nil)), fmt.Sprintf("%0.6f", minFloats(fitnesses)), fmt.Sprintf("%0.6f", maxFloats(fitnesses)), fmt.Sprintf("%0.6f", stat.StdDev(fitnesses, nil)),
		fmt.Sprintf("%0.2f%%", stat.Mean(pnls, nil)), fmt.Sprintf("%0.2f%%", minFloats(pnls)), fmt.Sprintf("%0.2f%%", maxFloats(pnls)), fmt.Sprintf("%0.6f", stat.StdDev(pnls, nil)),
		fmt.Sprintf("%0.2f%%", stat.Mean(maxDrawdowns, nil)), fmt.Sprintf("%0.2f%%", minFloats(maxDrawdowns)), fmt.Sprintf("%0.2f%%", maxFloats(maxDrawdowns)), fmt.Sprintf("%0.6f", stat.StdDev(maxDrawdowns, nil)),
		fmt.Sprintf("%0.2f", stat.Mean(sharpes, nil)), fmt.Sprintf("%0.2f", minFloats(sharpes)), fmt.Sprintf("%0.2f", maxFloats(sharpes)), fmt.Sprintf("%0.6f", stat.StdDev(sharpes, nil)),
		fmt.Sprintf("%0.2f", stat.Mean(sortinos, nil)), fmt.Sprintf("%0.2f", minFloats(sortinos)), fmt.Sprintf("%0.2f", maxFloats(sortinos)), fmt.Sprintf("%0.6f", stat.StdDev(sortinos, nil)),
		fmt.Sprintf("%0.2f", stat.Mean(trades, nil)), fmt.Sprintf("%0.2f", minFloats(trades)), fmt.Sprintf("%0.2f", maxFloats(trades)), fmt.Sprintf("%0.6f", stat.StdDev(trades, nil)),

		fmt.Sprintf("%0.2f", stat.Mean(trainDays, nil)), fmt.Sprintf("%0.2f", minFloats(trainDays)), fmt.Sprintf("%0.2f", maxFloats(trainDays)), fmt.Sprintf("%0.6f", stat.StdDev(trainDays, nil)),

		fmt.Sprintf("%.6f", s.ModelMetrics.Fitness()),
		fmt.Sprintf("%0.02f%%", s.ModelMetrics.Accuracy),
		fmt.Sprintf("%6.2f%%", (s.ModelMetrics.ClassPrecision[0]+s.ModelMetrics.ClassPrecision[1]+s.ModelMetrics.ClassPrecision[2])/3),
		fmt.Sprintf("%6.2f%%", (s.ModelMetrics.ClassRecall[0]+s.ModelMetrics.ClassRecall[1]+s.ModelMetrics.ClassRecall[2])/3),
		fmt.Sprintf("%6.2f%%", (s.ModelMetrics.F1Scores[0]+s.ModelMetrics.F1Scores[1]+s.ModelMetrics.F1Scores[2])/3),
		fmt.Sprintf("%d", s.ModelMetrics.Samples[0]+s.ModelMetrics.Samples[1]+s.ModelMetrics.Samples[2]),

		fmt.Sprintf("%6.2f%%", s.ModelMetrics.Backtest.Mean.PnL), fmt.Sprintf("%6.2f%%", s.ModelMetrics.Backtest.Min.PnL), fmt.Sprintf("%6.2f%%", s.ModelMetrics.Backtest.Max.PnL), fmt.Sprintf("%6.2f", s.ModelMetrics.Backtest.StdDev.PnL),
		fmt.Sprintf("%6.2f%%", s.ModelMetrics.Backtest.Mean.MaxDrawdown), fmt.Sprintf("%6.2f%%", s.ModelMetrics.Backtest.Min.MaxDrawdown), fmt.Sprintf("%6.2f%%", s.ModelMetrics.Backtest.Max.MaxDrawdown), fmt.Sprintf("%6.2f", s.ModelMetrics.Backtest.StdDev.MaxDrawdown),
		fmt.Sprintf("%6.2f", s.ModelMetrics.Backtest.Mean.SharpeRatio), fmt.Sprintf("%6.2f", s.ModelMetrics.Backtest.Min.SharpeRatio), fmt.Sprintf("%6.2f", s.ModelMetrics.Backtest.Max.SharpeRatio), fmt.Sprintf("%6.2f", s.ModelMetrics.Backtest.StdDev.SharpeRatio),
		fmt.Sprintf("%6.2f", s.ModelMetrics.Backtest.Mean.SortinoRatio), fmt.Sprintf("%6.2f", s.ModelMetrics.Backtest.Min.SortinoRatio), fmt.Sprintf("%6.2f", s.ModelMetrics.Backtest.Max.SortinoRatio), fmt.Sprintf("%6.2f", s.ModelMetrics.Backtest.StdDev.SortinoRatio),
		fmt.Sprintf("%6.2f", s.ModelMetrics.Backtest.Mean.Trades), fmt.Sprintf("%6.2f", s.ModelMetrics.Backtest.Min.Trades), fmt.Sprintf("%6.2f", s.ModelMetrics.Backtest.Max.Trades), fmt.Sprintf("%6.2f", s.ModelMetrics.Backtest.StdDev.Trades),

		params.Instrument,
		fmt.Sprintf("%0.0f", params.Leverage),
		fmt.Sprintf("%0.04f", params.TradeMultiplier),
		fmt.Sprintf("%0.04f", params.Commission),
		fmt.Sprintf("%0.0f", params.Cooldown.Seconds()),
		"",
		fmt.Sprintf("%d", params.WindowSize),
		fmt.Sprintf("%d", params.Candles),
		fmt.Sprintf("%0.04f", params.TakeProfit*params.Leverage),
		fmt.Sprintf("%0.04f", params.StopLoss*params.Leverage),
		"",
		fmt.Sprintf("%0.04f", params.MinTradeProbability),
		"",
		fmt.Sprintf("%.06f", params.L2Penalty),
		fmt.Sprintf("%.06f", params.DropoutRate),
		fmt.Sprintf("%.06f", params.LearnRate),
		fmt.Sprintf("%0.02f", params.TrainDays.Hours()/24),
		"",
		fmt.Sprintf("%d", params.ShortMovingAverageLength),
		fmt.Sprintf("%d", params.LongMovingAverageLength),
		fmt.Sprintf("%d", params.LongRSILength),
		fmt.Sprintf("%d", params.ShortRSILength),
		fmt.Sprintf("%d", params.ShortMACDWindowLength),
		fmt.Sprintf("%d", params.LongMACDWindowLength),
		fmt.Sprintf("%d", params.MACDSignalWindow),
		fmt.Sprintf("%d", params.FastShortMACDWindowLength),
		fmt.Sprintf("%d", params.FastLongMACDWindowLength),
		fmt.Sprintf("%d", params.FastMACDSignalWindow),
		fmt.Sprintf("%d", params.BollingerBandsWindow),
		fmt.Sprintf("%0.02f", params.BollingerBandsMultiplier),
		fmt.Sprintf("%d", params.StochasticOscillatorWindow),
		fmt.Sprintf("%d", params.SlowATRPeriod),
		fmt.Sprintf("%d", params.FastATRPeriod),
		fmt.Sprintf("%d", params.OBVMovingAverageLength),
		fmt.Sprintf("%d", params.VolumesMovingAverageLength),
		fmt.Sprintf("%d", params.ChaikinMoneyFlowPeriod),
		fmt.Sprintf("%d", params.MoneyFlowIndexPeriod),
		fmt.Sprintf("%d", params.RateOfChangePeriod),
		fmt.Sprintf("%d", params.CCIPeriod),
		fmt.Sprintf("%d", params.WilliamsRPeriod),
		fmt.Sprintf("%d", params.PriceChangeFastPeriod),
		fmt.Sprintf("%d", params.PriceChangeMediumPeriod),
		fmt.Sprintf("%d", params.PriceChangeSlowPeriod),
		fmt.Sprintf("%0.02f", params.RSIUpperBound),
		fmt.Sprintf("%0.02f", params.RSILowerBound),
		fmt.Sprintf("%d", params.RSISlope),
	}

	if err := writer.Write(row); err != nil {
		return err
	} else {
		writer.Flush()
		return nil
	}
}
