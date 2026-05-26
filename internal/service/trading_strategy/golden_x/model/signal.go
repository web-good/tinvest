package model

type Signal struct {
	RSI            float64
	Thresholds     Thresholds
	SellThresholds SellThresholds
	TrendStatus    TrendStatus
	GreenBuy       bool
	YellowBuy      bool
	DivergenceOK   bool
	VolumeOK       bool
}
