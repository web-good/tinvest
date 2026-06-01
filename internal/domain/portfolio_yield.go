package domain

import "time"

// PortfolioYield is the computed calendar-year (YTD) return of the portfolio.
type PortfolioYield struct {
	PeriodStart    time.Time // beginning of the period (Jan 1)
	PeriodEnd      time.Time // end of the period (today)
	StartValue     float64   // portfolio value at PeriodStart (V_start)
	EndValue       float64   // current portfolio value (V_end)
	Deposits       float64   // total deposits during the period (positive)
	Withdrawals    float64   // total withdrawals during the period (positive)
	NetDeposits    float64   // Deposits - Withdrawals
	AnnualizedXIRR float64   // annualized money-weighted return, fraction (0.123 = 12.3%)
	PeriodReturn   float64   // non-annualized period return, fraction
	XIRRAvailable  bool      // false if XIRR could not be computed (same-sign / no convergence / insufficient data)
	Note           string    // human-readable note, e.g. when XIRR is unavailable
}
