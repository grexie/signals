package ta

import "math"

// Candlestick pattern detection
func IsDoji(open, close, high, low float64) bool {
	bodySize := math.Abs(open - close)
	totalRange := high - low
	return totalRange > 0 && bodySize/totalRange < 0.1
}

func IsHammer(open, close, high, low float64) bool {
	bodySize := math.Abs(open - close)
	if bodySize == 0 {
		return false
	}

	upperShadow := high - math.Max(open, close)
	lowerShadow := math.Min(open, close) - low

	return lowerShadow > 2*bodySize && upperShadow < bodySize
}

func IsEngulfing(currentOpen, currentClose, prevOpen, prevClose float64) bool {
	currentBullish := currentClose > currentOpen
	prevBullish := prevClose > prevOpen

	return currentBullish != prevBullish &&
		math.Abs(currentClose-currentOpen) > math.Abs(prevClose-prevOpen)
}
