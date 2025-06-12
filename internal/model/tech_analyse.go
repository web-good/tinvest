package model

import "time"

type MacDItemTechAnalyse struct {
	Date       time.Time
	SignalLine Quotation
	MacDLine   Quotation
}

type Quotation struct {
	Units int64
	Nano  int32
}

type RsiItemTechAnalyse struct {
	Date       time.Time
	SignalLine Quotation
}

type EmaItemTechAnalyse struct {
	Date       time.Time
	SignalLine Quotation
}

type BbItemTechAnalyse struct {
	Date       time.Time
	MiddleBand Quotation
	UpperBand  Quotation
	LowerBand  Quotation
}

type CandleItemTechAnalyse struct {
	Time       time.Time
	Open       Quotation
	Close      Quotation
	Low        Quotation
	High       Quotation
	IsComplete bool
}
