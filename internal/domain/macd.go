package domain

import "time"

type MACDItemTechAnalyse struct {
	Date       time.Time
	SignalLine Quotation
	MacDLine   Quotation
	UnderZero  bool
	Diff       float64
	IsCross    bool
}

type Quotation struct {
	Units int64
	Nano  int32
}
