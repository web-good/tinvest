package model

import "time"

type Quotation struct {
	Units int64
	Nano  int32
}

type Position struct {
	InstrumentType string
	Figi           string
	ShareID        string
	Price          Quotation
	Quantity       int64
	PurchasePrice  Quotation
}

type Operation struct {
	Date         time.Time
	InstrumentID string
	Type         string
}

type CashOperation struct {
	Date    time.Time
	Type    string  // OperationType.String()
	TypeID  int32   // raw OperationType enum value, for robust classification
	Payment float64 // RUB amount from MoneyValue (units + nano/1e9)
	Yield   float64 // realized P&L of the operation (units + nano/1e9); meaningful for trades (SELL)
}

// Trade is one executed fill of an instrument (from GetOperationsByCursor trades).
type Trade struct {
	Date     time.Time
	Price    float64
	Quantity int64
	IsBuy    bool
}
