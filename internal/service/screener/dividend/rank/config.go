package rank

// Config держит веса столпов (сумма = 1.0) и пороги ворот. Все значения —
// точки калибровки, сверенные с живыми данными Invest API. Единицы: yield и
// payout в процентах (8.0 = 8%, 60 = 60%); MarketCapitalization — в рублях.
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
	MinMarketCap       float64 // ниже (в т.ч. 0 = нет данных) — отсев как неликвид
	MinFreeFloat       float64 // доля акций в свободном обращении ниже этого (в т.ч. 0 = нет данных) — отсев как тонкий флоат

	BonusScoreT1 float64 // композит >= T1 → бонус +1
	BonusScoreT2 float64 // композит >= T2 → бонус +2
	BonusScoreT3 float64 // композит >= T3 → бонус +3

	PayoutIdealLow  float64 // нижняя граница идеальной зоны payout
	PayoutIdealHigh float64 // верхняя граница идеальной зоны payout

	BankPBIdealHigh float64 // P/B, до которого оценка банка максимальна (1.0)
	BankPBZero      float64 // P/B, при котором оценка банка падает до 0
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
		MaxPayoutPct:       140.0,
		YieldTrapMinYield:  20.0,
		YieldCapPct:        14.0,
		MinMarketCap:       50_000_000_000, // ₽50 млрд; live-калибровка 2026-07-21: единицы подтверждены как ₽ (не млн), curated-11 Golden X проходят с запасом (мин. TATNP ≈₽62.2 млрд), micro-cap (Мордовская энергосбытовая ≈₽1.2 млрд, ТНС энерго ≈₽1-14 млрд) отсеиваются
		MinFreeFloat:       0.07,           // live-калибровка 2026-07-21: разрыв 5–9% между неликвидом (UDMN 0, BANE/GCHE 3, LEAS 4, AKRN/SIBN 5) и легитимными (min 9%); 0.07 посередине

		BonusScoreT1: 56, // live-калибровка 2026-07-21 по перцентилям композита ликвидной вселенной (n=41): p25≈56.0
		BonusScoreT2: 67, // p50≈67.1
		BonusScoreT3: 73, // p75≈72.8 (округлено вверх, чтобы верхняя полоса точно отражала топ-квартиль)

		PayoutIdealLow:  30.0,
		PayoutIdealHigh: 60.0,

		// BankPBIdealHigh/BankPBZero — live-калибровка 2026-07-21 (cmd/divscreen
		// -top 0 / -probe SBER,BSPB,SVCB,VTBR,MOEX). Наблюдаемый P/B среди
		// финансового сектора (financial, n=9): SBERP 0.03, VTBR 0.13, BSPB 0.47,
		// SVCB 0.50, SBER 0.64, DOMRF 0.72, T 0.92, RGSS 0.97, MOEX 1.15 (MOEX
		// отсеян по yield trap, не по P/B). При исходных ориентирах 1.0/2.5 ВСЕ
		// живые банки получали бы Valuation=1.0 (P/B<=1.0 для всех, кроме MOEX) —
		// пилар не дифференцирует реальные имена. IdealHigh=0.5 (явно ниже
		// балансовой стоимости — бесспорно дёшево) и Zero=1.5 (запас над макс.
		// наблюдаемым P/B в выборке) дают реальный разброс: VTBR/SBERP/BSPB/SVCB
		// у потолка 1.0, SBER≈0.86, DOMRF≈0.78, T≈0.58, RGSS≈0.53, MOEX≈0.35.
		BankPBIdealHigh: 0.5,
		BankPBZero:      1.5,
	}
}
