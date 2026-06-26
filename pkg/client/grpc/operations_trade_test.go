package grpc

import (
	"testing"
	"time"

	"google.golang.org/protobuf/types/known/timestamppb"

	investapi "tinvest/internal/pb/v1"
)

func TestTradesFromCursorItems_FiltersByInstrumentAndDirection(t *testing.T) {
	ts := time.Date(2026, 6, 1, 10, 0, 0, 0, time.UTC)
	items := []*investapi.OperationItem{
		{
			InstrumentUid: "uid-1",
			Type:          investapi.OperationType_OPERATION_TYPE_BUY,
			Date:          timestamppb.New(ts),
			TradesInfo: &investapi.OperationItemTrades{Trades: []*investapi.OperationItemTrade{
				{Num: "t1", Date: timestamppb.New(ts), Quantity: 5,
					Price: &investapi.MoneyValue{Units: 100, Nano: 250000000}},
			}},
		},
		{InstrumentUid: "uid-OTHER", Type: investapi.OperationType_OPERATION_TYPE_BUY}, // filtered out
	}

	got := tradesFromCursorItems(items, "uid-1")
	if len(got) != 1 {
		t.Fatalf("got %d trades, want 1", len(got))
	}
	if !got[0].IsBuy || got[0].Quantity != 5 || got[0].Price != 100.25 {
		t.Fatalf("trade = %+v, want buy qty5 price100.25", got[0])
	}
}

func TestTradesFromCursorItems_SellDirection(t *testing.T) {
	ts := time.Date(2026, 6, 2, 12, 0, 0, 0, time.UTC)
	items := []*investapi.OperationItem{
		{
			InstrumentUid: "uid-1",
			Type:          investapi.OperationType_OPERATION_TYPE_SELL,
			Date:          timestamppb.New(ts),
			TradesInfo: &investapi.OperationItemTrades{Trades: []*investapi.OperationItemTrade{
				{Num: "t2", Date: timestamppb.New(ts), Quantity: 3,
					Price: &investapi.MoneyValue{Units: 200, Nano: 500000000}},
			}},
		},
	}

	got := tradesFromCursorItems(items, "uid-1")
	if len(got) != 1 {
		t.Fatalf("got %d trades, want 1", len(got))
	}
	if got[0].IsBuy || got[0].Quantity != 3 || got[0].Price != 200.5 {
		t.Fatalf("trade = %+v, want sell qty3 price200.5", got[0])
	}
}

func TestTradesFromCursorItems_FiltersNonBuySell(t *testing.T) {
	ts := time.Date(2026, 6, 3, 9, 0, 0, 0, time.UTC)
	items := []*investapi.OperationItem{
		{
			InstrumentUid: "uid-1",
			Type:          investapi.OperationType_OPERATION_TYPE_COUPON, // not buy/sell
			Date:          timestamppb.New(ts),
		},
	}

	got := tradesFromCursorItems(items, "uid-1")
	if len(got) != 0 {
		t.Fatalf("got %d trades, want 0 (coupon should be filtered)", len(got))
	}
}
