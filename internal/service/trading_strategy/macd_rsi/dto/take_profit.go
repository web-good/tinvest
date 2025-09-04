package dto

import "tinvest/internal/enum"

type TakeProfit struct {
	Interval   enum.Interval
	RSILength  int32
	MACDLength int32
	Scheduler  string
}
