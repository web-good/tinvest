package backtest

import "time"

type Report struct {
	Win  int32
	Lose int32
	Logs []OrderLine
}

type OrderLine struct {
	InstrumentId  string
	PurchasePrice float64
	SellingPrice  float64
	TakeProfit    float64
	StopLoss      float64
	Quantity      int
	PurchaseTime  time.Time
	SellingTime   time.Time
}

type Account struct {
	Amount     float64
	OrderLines map[string]*OrderLine
	Report     Report
}
