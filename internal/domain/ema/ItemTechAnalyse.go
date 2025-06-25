package ema

import "time"

type Quotation struct {
	Units int64
	Nano  int32
}

type ItemTechAnalyse struct {
	Date       time.Time
	SignalLine Quotation
}
