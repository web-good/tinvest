package grpc

import (
	"context"
	"testing"
	investapi "tinvest/internal/pb/v1"

	googlegrpc "google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestDensitySubscriptionSources(t *testing.T) {
	books := densityBookSubscription([]string{"uid"}, 50, true).GetSubscribeOrderBookRequest()
	if books.GetInstruments()[0].GetOrderBookType() != investapi.OrderBookType_ORDERBOOK_TYPE_EXCHANGE || books.GetInstruments()[0].GetDepth() != 50 {
		t.Fatal("wrong order book source")
	}
	trades := densityTradeSubscription([]string{"uid"}, true).GetSubscribeTradesRequest()
	if trades.GetTradeSource() != investapi.TradeSourceType_TRADE_SOURCE_EXCHANGE {
		t.Fatal("wrong trade source")
	}
}

func TestDensityBookRejectsUnreliableData(t *testing.T) {
	b := &investapi.OrderBook{InstrumentUid: "uid", IsConsistent: true, OrderBookType: investapi.OrderBookType_ORDERBOOK_TYPE_EXCHANGE, Time: timestamppb.Now(), Bids: []*investapi.Order{{Price: &investapi.Quotation{Units: 100}, Quantity: 10}}, Asks: []*investapi.Order{{Price: &investapi.Quotation{Units: 101}, Quantity: 10}}}
	got, ok := densityBook(b, 10)
	if !ok || got.InstrumentID != "uid" || got.Lot != 10 || len(got.Bids) != 1 {
		t.Fatalf("got %+v %v", got, ok)
	}
	b.IsConsistent = false
	if _, ok = densityBook(b, 10); ok {
		t.Fatal("inconsistent book accepted")
	}
	b.IsConsistent = true
	b.OrderBookType = investapi.OrderBookType_ORDERBOOK_TYPE_ALL
	if _, ok = densityBook(b, 10); ok {
		t.Fatal("non-exchange book accepted")
	}
}

func TestDensityStreamEventsIncludeAckAndTradingStatus(t *testing.T) {
	r := &investapi.MarketDataResponse{Payload: &investapi.MarketDataResponse_SubscribeTradesResponse{SubscribeTradesResponse: &investapi.SubscribeTradesResponse{TradeSource: investapi.TradeSourceType_TRADE_SOURCE_EXCHANGE, TradeSubscriptions: []*investapi.TradeSubscription{{InstrumentUid: "a", SubscriptionStatus: investapi.SubscriptionStatus_SUBSCRIPTION_STATUS_SUCCESS}, {InstrumentUid: "b", SubscriptionStatus: investapi.SubscriptionStatus_SUBSCRIPTION_STATUS_INSTRUMENT_NOT_FOUND}}}}}
	events := densityEvents(r, nil)
	if len(events) != 2 || events[0].InstrumentID != "a" || !events[0].Success || events[1].Success {
		t.Fatalf("events %+v", events)
	}
	r = &investapi.MarketDataResponse{Payload: &investapi.MarketDataResponse_TradingStatus{TradingStatus: &investapi.TradingStatus{InstrumentUid: "a", TradingStatus: investapi.SecurityTradingStatus_SECURITY_TRADING_STATUS_NORMAL_TRADING, LimitOrderAvailableFlag: true}}}
	events = densityEvents(r, nil)
	if len(events) != 1 || !events[0].TradingOpen {
		t.Fatalf("status %+v", events)
	}
}

func TestDensityInfoSubscription(t *testing.T) {
	req := densityInfoSubscription([]string{"uid"}, true).GetSubscribeInfoRequest()
	if req.GetSubscriptionAction() != investapi.SubscriptionAction_SUBSCRIPTION_ACTION_SUBSCRIBE || req.GetInstruments()[0].GetInstrumentId() != "uid" {
		t.Fatalf("request %+v", req)
	}
}

func TestDensityStreamUsesCallerContextAndExchangeRequests(t *testing.T) {
	fake := &fakeDensityStreamClient{stream: &fakeDensityStream{}}
	market := &DensityMarketData{stream: fake, auth: NewAuth("token")}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	stream, err := market.OpenStream(ctx, map[string]int64{"uid": 10})
	if err != nil {
		t.Fatal(err)
	}
	if len(fake.options) != 1 {
		t.Fatal("missing RPC credentials")
	}
	cred, ok := fake.options[0].(googlegrpc.PerRPCCredsCallOption)
	if !ok {
		t.Fatal("wrong RPC credential option")
	}
	md, err := cred.Creds.GetRequestMetadata(ctx)
	if err != nil || md["Authorization"] != "Bearer token" || !cred.Creds.RequireTransportSecurity() {
		t.Fatal("bad stream credentials")
	}
	if _, ok := fake.ctx.Deadline(); ok {
		t.Fatal("stream acquired unary deadline")
	}
	if err := stream.SubscribeBooks([]string{"uid"}, 50, true); err != nil {
		t.Fatal(err)
	}
	if got := fake.stream.sent.GetSubscribeOrderBookRequest().GetInstruments()[0].GetOrderBookType(); got != investapi.OrderBookType_ORDERBOOK_TYPE_EXCHANGE {
		t.Fatalf("source %v", got)
	}
}

type fakeDensityStreamClient struct {
	investapi.MarketDataStreamServiceClient
	ctx     context.Context
	stream  *fakeDensityStream
	options []googlegrpc.CallOption
}

func (f *fakeDensityStreamClient) MarketDataStream(ctx context.Context, opts ...googlegrpc.CallOption) (googlegrpc.BidiStreamingClient[investapi.MarketDataRequest, investapi.MarketDataResponse], error) {
	f.ctx = ctx
	f.options = opts
	return f.stream, nil
}

type fakeDensityStream struct {
	googlegrpc.BidiStreamingClient[investapi.MarketDataRequest, investapi.MarketDataResponse]
	sent *investapi.MarketDataRequest
}

func (f *fakeDensityStream) Send(r *investapi.MarketDataRequest) error { f.sent = r; return nil }
