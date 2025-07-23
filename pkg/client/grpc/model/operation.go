package model

type Quotation struct {
	Units int64
	Nano  int32
}

type Position struct {
	ShareID       string
	Price         Quotation
	Quantity      int64
	PurchasePrice Quotation
}
