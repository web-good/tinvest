package converter

import (
	"tinvest/internal/model"
	investapi "tinvest/internal/pb/v1"
)

func ConvertFundamentalsFromPb(in []*investapi.GetAssetFundamentalsResponse_StatisticResponse) []*model.Fundamentals {
	res := make([]*model.Fundamentals, 0, len(in))
	for _, f := range in {
		if f == nil {
			continue
		}
		res = append(res, &model.Fundamentals{
			AssetUID:                         f.AssetUid,
			ForwardAnnualDividendYield:       f.ForwardAnnualDividendYield,
			DividendYieldDailyTtm:            f.DividendYieldDailyTtm,
			DividendPayoutRatioFy:            f.DividendPayoutRatioFy,
			FiveYearsAverageDividendYield:    f.FiveYearsAverageDividendYield,
			FiveYearAnnualDividendGrowthRate: f.FiveYearAnnualDividendGrowthRate,
			DividendRateTtm:                  f.DividendRateTtm,
			NetDebtToEbitda:                  f.NetDebtToEbitda,
			TotalDebtToEquityMrq:             f.TotalDebtToEquityMrq,
			FixedChargeCoverageRatioFy:       f.FixedChargeCoverageRatioFy,
			CurrentRatioMrq:                  f.CurrentRatioMrq,
			Roic:                             f.Roic,
			Roe:                              f.Roe,
			NetMarginMrq:                     f.NetMarginMrq,
			EbitdaTtm:                        f.EbitdaTtm,
			RevenueTtm:                       f.RevenueTtm,
			FreeCashFlowTtm:                  f.FreeCashFlowTtm,
			EvToEbitdaMrq:                    f.EvToEbitdaMrq,
			PeRatioTtm:                       f.PeRatioTtm,
			PriceToBookTtm:                   f.PriceToBookTtm,
			PriceToFreeCashFlowTtm:           f.PriceToFreeCashFlowTtm,
			MarketCapitalization:             f.MarketCapitalization,
			FreeFloat:                        f.FreeFloat,
			AverageDailyVolumeLast4Weeks:     f.AverageDailyVolumeLast_4Weeks,
		})
	}
	return res
}
