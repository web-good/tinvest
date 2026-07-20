package converter

import (
	"testing"

	investapi "tinvest/internal/pb/v1"
)

func TestConvertFundamentalsFromPb(t *testing.T) {
	in := []*investapi.GetAssetFundamentalsResponse_StatisticResponse{
		{
			AssetUid:                   "asset-1",
			ForwardAnnualDividendYield: 8.5,
			DividendPayoutRatioFy:      55,
			NetDebtToEbitda:            1.2,
			Roic:                       0.19,
			EvToEbitdaMrq:              3.4,
			FreeCashFlowTtm:            1000,
		},
	}

	got := ConvertFundamentalsFromPb(in)

	if len(got) != 1 {
		t.Fatalf("len = %d, want 1", len(got))
	}
	if got[0].AssetUid != "asset-1" || got[0].ForwardAnnualDividendYield != 8.5 || got[0].NetDebtToEbitda != 1.2 {
		t.Fatalf("mismatch: %+v", got[0])
	}
}
