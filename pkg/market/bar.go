package market

import "time"

type CandleBar string

const (
	CandleBar1s  CandleBar = "1s"
	CandleBar1m  CandleBar = "1m"
	CandleBar5m  CandleBar = "5m"
	CandleBar15m CandleBar = "15m"
	CandleBar1h  CandleBar = "1h"
)

func CandleBarToDuration(bar CandleBar) time.Duration {
	switch bar {
	case CandleBar1s:
		return time.Second
	case CandleBar1m:
		return time.Minute
	case CandleBar5m:
		return 5 * time.Minute
	case CandleBar15m:
		return 15 * time.Minute
	case CandleBar1h:
		return time.Hour
	default:
		return time.Minute
	}
}
