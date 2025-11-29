package model

import "time"

type Bond struct {
	Id                    string
	AciValue              float64
	CouponQuantityPerYear int32
	Name                  string
	Exchange              string
	MaturityDate          time.Time
	FloatingCouponFlag    bool
	Nominal               float64
	RiskLevel             string
}
