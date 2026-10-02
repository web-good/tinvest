package model

import "time"

// Dividend is one dividend event of a share as T-Invest reports it. Missing dates are zero.
type Dividend struct {
	LastBuyDate  time.Time // last day (inclusive, UTC) to buy and still receive the payout
	RecordDate   time.Time // register record date (UTC)
	DividendType string    // "Regular Cash", "Cancelled", "Daily Accrual", "Return of Capital", ...
}
