package ta

import "github.com/grexie/signals/pkg/candles"

type DivergenceStrategy int

const (
	DivergenceSideways DivergenceStrategy = 0
	DivergenceBullish  DivergenceStrategy = 1
	DivergenceBearish  DivergenceStrategy = -1
)

func Divergence(candles []candles.Candle, macd []float64, i int, window int) DivergenceStrategy {
	if i < window+2 || i >= len(candles)-1 {
		return DivergenceSideways
	}

	swingLows := findSwingLows(candles, i-window, i+1)
	if len(swingLows) >= 2 {
		// Use the two most recent
		last := swingLows[len(swingLows)-1]
		prev := swingLows[len(swingLows)-2]

		price1 := candles[prev].Low
		price2 := candles[last].Low
		macd1 := macd[prev]
		macd2 := macd[last]

		if price2 < price1 && macd2 > macd1 {
			return DivergenceBullish
		}
	}

	swingHighs := findSwingHighs(candles, i-window, i+1)
	if len(swingHighs) >= 2 {
		last := swingHighs[len(swingHighs)-1]
		prev := swingHighs[len(swingHighs)-2]

		price1 := candles[prev].High
		price2 := candles[last].High
		macd1 := macd[prev]
		macd2 := macd[last]

		if price2 > price1 && macd2 < macd1 {
			return DivergenceBearish
		}
	}

	return DivergenceSideways
}

func findSwingLows(candles []candles.Candle, start, end int) []int {
	var swingLows []int
	for i := start + 1; i < end-1; i++ {
		if candles[i].Low < candles[i-1].Low && candles[i].Low < candles[i+1].Low {
			swingLows = append(swingLows, i)
		}
	}
	return swingLows
}

func findSwingHighs(candles []candles.Candle, start, end int) []int {
	var swingHighs []int
	for i := start + 1; i < end-1; i++ {
		if candles[i].High > candles[i-1].High && candles[i].High > candles[i+1].High {
			swingHighs = append(swingHighs, i)
		}
	}
	return swingHighs
}
