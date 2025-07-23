package dto

import "tinvest/internal/service/trading_strategy/macd_rsi/enum"

type Trade struct {
	Interval      enum.Interval
	RSILength     int32
	RSIFastLength int32
	MACDLength    int32
	SearchArea    int
	Scheduler     string
}
