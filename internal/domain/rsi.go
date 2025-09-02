package domain

import "time"

type RSIItemTechAnalyse struct {
	Date       time.Time
	SignalLine Quotation
}
