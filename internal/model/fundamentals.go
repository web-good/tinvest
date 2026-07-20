package model

// Fundamentals — подмножество фундаментальных показателей актива из
// Tinkoff GetAssetFundamentals, что использует дивидендный скринер.
type Fundamentals struct {
	AssetUID string

	ForwardAnnualDividendYield       float64 // форвардная дивдоходность, %
	DividendYieldDailyTtm            float64 // текущая дивдоходность TTM, %
	DividendPayoutRatioFy            float64 // payout ratio, %
	FiveYearsAverageDividendYield    float64 // средняя дивдоходность за 5 лет, %
	FiveYearAnnualDividendGrowthRate float64 // среднегодовой рост дивиденда за 5 лет
	DividendRateTtm                  float64 // дивиденд на акцию TTM

	NetDebtToEbitda            float64
	TotalDebtToEquityMrq       float64
	FixedChargeCoverageRatioFy float64
	CurrentRatioMrq            float64

	Roic            float64
	Roe             float64
	NetMarginMrq    float64
	EbitdaTtm       float64
	RevenueTtm      float64
	FreeCashFlowTtm float64

	EvToEbitdaMrq          float64
	PeRatioTtm             float64
	PriceToBookTtm         float64
	PriceToFreeCashFlowTtm float64
}
