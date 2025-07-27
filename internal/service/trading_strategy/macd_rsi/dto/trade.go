package dto

import "tinvest/internal/service/trading_strategy/macd_rsi/enum"

type Trade struct {
	LocalInterval  enum.Interval
	GlobalInterval enum.Interval
	RSILength      int32
	MACDLength     int32
	SearchArea     int
	Scheduler      string
}
