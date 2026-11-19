package strategy

import "tinvest/internal/service/screener/bybitdensity/model"

// Executor may return a final fill synchronously, or a zero Execution and later
// deliver the final exchange result to Strategy.ConfirmExecution.
type Executor interface {
	ObserveBook(*model.Book)
	Submit(Order, *model.Book) Execution
}

// Order is an IOC for one instrument. LimitTicks zero selects a market exit.
// Direction is the position side (Bid long, Ask short), including on exit.
type Order struct {
	Key        model.Key
	ID         uint64
	Entry      bool
	Reason     string
	Direction  model.Side
	Steps      int64
	LimitTicks int64
}

// Execution is the cumulative result after the IOC's unfilled portion is
// cancelled. Quote is the positive executed notional before fees. An ID with
// zero steps acknowledges a completely unfilled IOC. IDs are instrument-local.
type Execution struct {
	OrderID uint64
	Steps   int64
	Quote   float64
}

type submittedOrder struct {
	order Order
	setup *setup
}
