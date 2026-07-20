package converter

import (
	"testing"

	"tinvest/internal/model"
	investapi "tinvest/internal/pb/v1"
)

func TestConvertFundamentalsFromPb(t *testing.T) {
	in := []*investapi.GetAssetFundamentalsResponse_StatisticResponse{
		{
			AssetUid:                         "asset-1",
			ForwardAnnualDividendYield:       1.1,
			DividendYieldDailyTtm:            2.2,
			DividendPayoutRatioFy:            3.3,
			FiveYearsAverageDividendYield:    4.4,
			FiveYearAnnualDividendGrowthRate: 5.5,
			DividendRateTtm:                  6.6,
			NetDebtToEbitda:                  7.7,
			TotalDebtToEquityMrq:             8.8,
			FixedChargeCoverageRatioFy:       9.9,
			CurrentRatioMrq:                  10.1,
			Roic:                             11.11,
			Roe:                              12.12,
			NetMarginMrq:                     13.13,
			EbitdaTtm:                        14.14,
			RevenueTtm:                       15.15,
			FreeCashFlowTtm:                  16.16,
			EvToEbitdaMrq:                    17.17,
			PeRatioTtm:                       18.18,
			PriceToBookTtm:                   19.19,
			PriceToFreeCashFlowTtm:           20.20,
		},
		nil,
	}

	got := ConvertFundamentalsFromPb(in)

	if len(got) != 1 {
		t.Fatalf("len = %d, want 1", len(got))
	}

	want := &model.Fundamentals{
		AssetUid:                         "asset-1",
		ForwardAnnualDividendYield:       1.1,
		DividendYieldDailyTtm:            2.2,
		DividendPayoutRatioFy:            3.3,
		FiveYearsAverageDividendYield:    4.4,
		FiveYearAnnualDividendGrowthRate: 5.5,
		DividendRateTtm:                  6.6,
		NetDebtToEbitda:                  7.7,
		TotalDebtToEquityMrq:             8.8,
		FixedChargeCoverageRatioFy:       9.9,
		CurrentRatioMrq:                  10.1,
		Roic:                             11.11,
		Roe:                              12.12,
		NetMarginMrq:                     13.13,
		EbitdaTtm:                        14.14,
		RevenueTtm:                       15.15,
		FreeCashFlowTtm:                  16.16,
		EvToEbitdaMrq:                    17.17,
		PeRatioTtm:                       18.18,
		PriceToBookTtm:                   19.19,
		PriceToFreeCashFlowTtm:           20.20,
	}

	if got[0].AssetUid != want.AssetUid {
		t.Errorf("AssetUid = %v, want %v", got[0].AssetUid, want.AssetUid)
	}
	if got[0].ForwardAnnualDividendYield != want.ForwardAnnualDividendYield {
		t.Errorf("ForwardAnnualDividendYield = %v, want %v", got[0].ForwardAnnualDividendYield, want.ForwardAnnualDividendYield)
	}
	if got[0].DividendYieldDailyTtm != want.DividendYieldDailyTtm {
		t.Errorf("DividendYieldDailyTtm = %v, want %v", got[0].DividendYieldDailyTtm, want.DividendYieldDailyTtm)
	}
	if got[0].DividendPayoutRatioFy != want.DividendPayoutRatioFy {
		t.Errorf("DividendPayoutRatioFy = %v, want %v", got[0].DividendPayoutRatioFy, want.DividendPayoutRatioFy)
	}
	if got[0].FiveYearsAverageDividendYield != want.FiveYearsAverageDividendYield {
		t.Errorf("FiveYearsAverageDividendYield = %v, want %v", got[0].FiveYearsAverageDividendYield, want.FiveYearsAverageDividendYield)
	}
	if got[0].FiveYearAnnualDividendGrowthRate != want.FiveYearAnnualDividendGrowthRate {
		t.Errorf("FiveYearAnnualDividendGrowthRate = %v, want %v", got[0].FiveYearAnnualDividendGrowthRate, want.FiveYearAnnualDividendGrowthRate)
	}
	if got[0].DividendRateTtm != want.DividendRateTtm {
		t.Errorf("DividendRateTtm = %v, want %v", got[0].DividendRateTtm, want.DividendRateTtm)
	}
	if got[0].NetDebtToEbitda != want.NetDebtToEbitda {
		t.Errorf("NetDebtToEbitda = %v, want %v", got[0].NetDebtToEbitda, want.NetDebtToEbitda)
	}
	if got[0].TotalDebtToEquityMrq != want.TotalDebtToEquityMrq {
		t.Errorf("TotalDebtToEquityMrq = %v, want %v", got[0].TotalDebtToEquityMrq, want.TotalDebtToEquityMrq)
	}
	if got[0].FixedChargeCoverageRatioFy != want.FixedChargeCoverageRatioFy {
		t.Errorf("FixedChargeCoverageRatioFy = %v, want %v", got[0].FixedChargeCoverageRatioFy, want.FixedChargeCoverageRatioFy)
	}
	if got[0].CurrentRatioMrq != want.CurrentRatioMrq {
		t.Errorf("CurrentRatioMrq = %v, want %v", got[0].CurrentRatioMrq, want.CurrentRatioMrq)
	}
	if got[0].Roic != want.Roic {
		t.Errorf("Roic = %v, want %v", got[0].Roic, want.Roic)
	}
	if got[0].Roe != want.Roe {
		t.Errorf("Roe = %v, want %v", got[0].Roe, want.Roe)
	}
	if got[0].NetMarginMrq != want.NetMarginMrq {
		t.Errorf("NetMarginMrq = %v, want %v", got[0].NetMarginMrq, want.NetMarginMrq)
	}
	if got[0].EbitdaTtm != want.EbitdaTtm {
		t.Errorf("EbitdaTtm = %v, want %v", got[0].EbitdaTtm, want.EbitdaTtm)
	}
	if got[0].RevenueTtm != want.RevenueTtm {
		t.Errorf("RevenueTtm = %v, want %v", got[0].RevenueTtm, want.RevenueTtm)
	}
	if got[0].FreeCashFlowTtm != want.FreeCashFlowTtm {
		t.Errorf("FreeCashFlowTtm = %v, want %v", got[0].FreeCashFlowTtm, want.FreeCashFlowTtm)
	}
	if got[0].EvToEbitdaMrq != want.EvToEbitdaMrq {
		t.Errorf("EvToEbitdaMrq = %v, want %v", got[0].EvToEbitdaMrq, want.EvToEbitdaMrq)
	}
	if got[0].PeRatioTtm != want.PeRatioTtm {
		t.Errorf("PeRatioTtm = %v, want %v", got[0].PeRatioTtm, want.PeRatioTtm)
	}
	if got[0].PriceToBookTtm != want.PriceToBookTtm {
		t.Errorf("PriceToBookTtm = %v, want %v", got[0].PriceToBookTtm, want.PriceToBookTtm)
	}
	if got[0].PriceToFreeCashFlowTtm != want.PriceToFreeCashFlowTtm {
		t.Errorf("PriceToFreeCashFlowTtm = %v, want %v", got[0].PriceToFreeCashFlowTtm, want.PriceToFreeCashFlowTtm)
	}
}
