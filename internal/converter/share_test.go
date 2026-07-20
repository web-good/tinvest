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

	if got.AssetUid != "asset-uid-1" {
		t.Fatalf("AssetUid = %q, want %q", got.AssetUid, "asset-uid-1")
	}
	if !got.DivYieldFlag {
		t.Fatalf("DivYieldFlag = false, want true")
	}
	if got.ID != "instr-uid-1" {
		t.Fatalf("ID = %q, want %q", got.ID, "instr-uid-1")
	}
}
