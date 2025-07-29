package dto

import "tinvest/internal/service/trading_strategy/macd_rsi/enum"

type TakeProfit struct {
	Interval   enum.Interval
	RSILength  int32
	MACDLength int32
	Scheduler  string
}
