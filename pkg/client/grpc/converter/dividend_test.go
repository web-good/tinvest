package converter

import (
	"testing"
	"time"

	investapi "tinvest/internal/pb/v1"

	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestConvertDividendsFromPb(t *testing.T) {
	lastBuy := time.Date(2025, 7, 17, 0, 0, 0, 0, time.UTC)
	record := time.Date(2025, 7, 18, 0, 0, 0, 0, time.UTC)
	in := []*investapi.Dividend{
		{LastBuyDate: timestamppb.New(lastBuy), RecordDate: timestamppb.New(record), DividendType: "Regular Cash"},
		nil,
		{DividendType: "Cancelled"}, // no dates at all
	}
	got := ConvertDividendsFromPb(in)
	if len(got) != 2 {
		t.Fatalf("len = %d, want 2 (nil entry skipped)", len(got))
	}
	if !got[0].LastBuyDate.Equal(lastBuy) || !got[0].RecordDate.Equal(record) || got[0].DividendType != "Regular Cash" {
		t.Fatalf("got[0] = %+v", got[0])
	}
	if !got[1].LastBuyDate.IsZero() || !got[1].RecordDate.IsZero() || got[1].DividendType != "Cancelled" {
		t.Fatalf("got[1] = %+v, want zero dates for missing timestamps", got[1])
	}
}
