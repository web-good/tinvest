package model

import "time"

type Bond struct {
	Id                    string
	AciValue              float64
	CouponQuantityPerYear int32
	Name                  string
	Exchange              string
	MaturityDate          time.Time
	AmortizationFlag      bool
	FloatingCouponFlag    bool
	Nominal               float64
	RiskLevel             string
	Nkd                   float64
}

type BondCoupon struct {
	CouponDate      time.Time
	PayOnBond       Quotation
	CouponStartDate time.Time
	CouponEndDate   time.Time
}
