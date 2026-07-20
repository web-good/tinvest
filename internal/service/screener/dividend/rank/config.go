package rank

// Config держит веса столпов (сумма = 1.0) и пороги ворот. Все значения —
// точки калибровки (Task 8 сверяет с живыми данными). Единицы: yield и
// payout в процентах (8.0 = 8%, 60 = 60%).
type Config struct {
	WeightSustainability float64
	WeightSafety         float64
	WeightDivGrowth      float64
	WeightQuality        float64
	WeightValuation      float64
	WeightYield          float64

	MaxNetDebtToEbitda float64 // выше — gate highLeverage
	MaxPayoutPct       float64 // выше — gate unsustainablePayout
	YieldTrapMinYield  float64 // ниже этого yield trap не рассматривается
	YieldCapPct        float64 // потолок для yield-подсчёта

	PayoutIdealLow  float64 // нижняя граница идеальной зоны payout
	PayoutIdealHigh float64 // верхняя граница идеальной зоны payout
}

func DefaultConfig() Config {
	return Config{
		WeightSustainability: 0.30,
		WeightSafety:         0.25,
		WeightDivGrowth:      0.15,
		WeightQuality:        0.15,
		WeightValuation:      0.10,
		WeightYield:          0.05,

		MaxNetDebtToEbitda: 4.0,
		MaxPayoutPct:       120.0,
		YieldTrapMinYield:  20.0,
		YieldCapPct:        14.0,

		PayoutIdealLow:  30.0,
		PayoutIdealHigh: 60.0,
	}
}
