package converter

import (
	"testing"

	investapi "tinvest/internal/pb/v1"
)

func TestConvertShareFromPb_MapsAssetUidAndDivFlag(t *testing.T) {
	in := &investapi.Share{
		Uid:          "instr-uid-1",
		AssetUid:     "asset-uid-1",
		Ticker:       "LKOH",
		Currency:     "rub",
		DivYieldFlag: true,
	}

	got := ConvertShareFromPb(in)

	if got.AssetUID != "asset-uid-1" {
		t.Fatalf("AssetUID = %q, want %q", got.AssetUID, "asset-uid-1")
	}
	if !got.DivYieldFlag {
		t.Fatalf("DivYieldFlag = false, want true")
	}
	if got.ID != "instr-uid-1" {
		t.Fatalf("ID = %q, want %q", got.ID, "instr-uid-1")
	}
}

func TestConvertShareFromPb_MapsSector(t *testing.T) {
	got := ConvertShareFromPb(&investapi.Share{
		Ticker:   "SBER",
		Currency: "rub",
		Sector:   "financial",
	})
	if got.Sector != "financial" {
		t.Fatalf("Sector = %q, want %q", got.Sector, "financial")
	}
}

func TestConvertShareFromPb_MapsMinPriceIncrement(t *testing.T) {
	tests := []struct {
		name string
		in   *investapi.Quotation
		want float64
	}{
		{"nil", nil, 0},
		{"fraction only", &investapi.Quotation{Units: 0, Nano: 10000000}, 0.01},
		{"units and nano", &investapi.Quotation{Units: 1, Nano: 500000000}, 1.5},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := ConvertShareFromPb(&investapi.Share{
				Ticker: "SBER", Currency: "rub", MinPriceIncrement: tc.in,
			})
			if got.MinPriceIncrement != tc.want {
				t.Fatalf("MinPriceIncrement = %v, want %v", got.MinPriceIncrement, tc.want)
			}
		})
	}
}
