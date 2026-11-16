package grpc

import (
	"context"
	"testing"
	investapi "tinvest/internal/pb/v1"

	googlegrpc "google.golang.org/grpc"
)

type allocationFake struct {
	investapi.OrdersServiceClient
	ownLots          int64
	available, total int64
	quoteCalls       int
}

func (f *allocationFake) GetMaxLots(_ context.Context, _ *investapi.GetMaxLotsRequest, _ ...googlegrpc.CallOption) (*investapi.GetMaxLotsResponse, error) {
	return &investapi.GetMaxLotsResponse{Currency: "rub", BuyLimits: &investapi.GetMaxLotsResponse_BuyLimitsView{BuyMaxLots: f.ownLots, BuyMoneyAmount: quote(f.available)}, BuyMarginLimits: &investapi.GetMaxLotsResponse_BuyLimitsView{BuyMaxLots: 100}}, nil
}
func (f *allocationFake) GetOrderPrice(_ context.Context, r *investapi.GetOrderPriceRequest, _ ...googlegrpc.CallOption) (*investapi.GetOrderPriceResponse, error) {
	f.quoteCalls++
	if r.Quantity != 1 || r.Direction != investapi.OrderDirection_ORDER_DIRECTION_BUY {
		panic("wrong pretrade quantity or side")
	}
	return &investapi.GetOrderPriceResponse{LotsRequested: 1, TotalOrderAmount: &investapi.MoneyValue{Currency: "rub", Units: f.total / 1e9, Nano: int32(f.total % 1e9)}}, nil
}
func TestDensityBrokerAllocationUsesOwnMoneyIncludingCommission(t *testing.T) {
	for _, tc := range []struct {
		name                   string
		lots, available, total int64
		allowed                bool
	}{{"commission_shortfall", 1, 1000e9, 1000500000000, false}, {"enough", 1, 1001e9, 1000500000000, true}, {"margin_only", 0, 1001e9, 1000500000000, false}} {
		t.Run(tc.name, func(t *testing.T) {
			b := brokerFixture()
			f := &allocationFake{ownLots: tc.lots, available: tc.available, total: tc.total}
			b.orders = f
			got, err := b.CanBuy(context.Background(), "uid", 100e9, 1)
			if err != nil || got != tc.allowed {
				t.Fatalf("allowed=%v err=%v", got, err)
			}
		})
	}
}

type ownedSharesFake struct {
	investapi.InstrumentsServiceClient
	request *investapi.InstrumentRequest
}

func (f *ownedSharesFake) ShareBy(_ context.Context, r *investapi.InstrumentRequest, _ ...googlegrpc.CallOption) (*investapi.ShareResponse, error) {
	f.request = r
	return &investapi.ShareResponse{Instrument: &investapi.Share{Uid: r.Id, Ticker: "HALTED", Lot: 10, MinPriceIncrement: quote(10000000), BuyAvailableFlag: false, ApiTradeAvailableFlag: false}}, nil
}
func TestRecoveryInstrumentMetadataIgnoresBuyAvailability(t *testing.T) {
	f := &ownedSharesFake{}
	m := &DensityMarketData{instruments: f, auth: NewAuth("test")}
	i, err := m.InstrumentByID(context.Background(), "owned")
	if err != nil || i.ID != "owned" || i.Lot != 10 || f.request.IdType != investapi.InstrumentIdType_INSTRUMENT_ID_TYPE_UID {
		t.Fatalf("%+v %v", i, err)
	}
}
