package converter

import (
	"testing"
	investapi "tinvest/internal/pb/v1"
)

func TestConvertBondModelFromBondPb_NewFields(t *testing.T) {
	pb := &investapi.Bond{
		Uid:                 "uid-1",
		Name:                "ОФЗ 26238",
		Nominal:             &investapi.MoneyValue{Units: 1000},
		AciValue:            &investapi.MoneyValue{Units: 15, Nano: 500000000},
		RiskLevel:           investapi.RiskLevel_RISK_LEVEL_LOW,
		LiquidityFlag:       true,
		SubordinatedFlag:    true,
		ForQualInvestorFlag: true,
		PerpetualFlag:       true,
		Sector:              "government",
		IssueSize:           5000000,
	}

	got := ConvertBondModelFromBondPb(pb)

	if !got.LiquidityFlag || !got.SubordinatedFlag || !got.ForQualInvestorFlag || !got.PerpetualFlag {
		t.Fatalf("флаги не замаплены: %+v", got)
	}
	if got.Sector != "government" {
		t.Fatalf("Sector = %q, ожидалось government", got.Sector)
	}
	if got.IssueSize != 5000000 {
		t.Fatalf("IssueSize = %d, ожидалось 5000000", got.IssueSize)
	}
}
