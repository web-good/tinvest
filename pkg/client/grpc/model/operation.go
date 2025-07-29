package model

import "time"

type Quotation struct {
	Units int64
	Nano  int32
}

type Position struct {
	Figi          string
	ShareID       string
	Price         Quotation
	Quantity      int64
	PurchasePrice Quotation
}

type Operation struct {
	Date         time.Time
	InstrumentID string
	Type         string
}
