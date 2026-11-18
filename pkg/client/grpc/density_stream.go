package grpc

import (
	"context"
	"time"
	investapi "tinvest/internal/pb/v1"
	"tinvest/internal/service/screener/density"

	googlegrpc "google.golang.org/grpc"
)

func densityAction(subscribe bool) investapi.SubscriptionAction {
	if subscribe {
		return investapi.SubscriptionAction_SUBSCRIPTION_ACTION_SUBSCRIBE
	}
	return investapi.SubscriptionAction_SUBSCRIPTION_ACTION_UNSUBSCRIBE
}
func densityBookSubscription(ids []string, depth int32, subscribe bool) *investapi.MarketDataRequest {
	instruments := make([]*investapi.OrderBookInstrument, 0, len(ids))
	for _, id := range ids {
		instruments = append(instruments, &investapi.OrderBookInstrument{InstrumentId: id, Depth: depth, OrderBookType: investapi.OrderBookType_ORDERBOOK_TYPE_EXCHANGE})
	}
	return &investapi.MarketDataRequest{Payload: &investapi.MarketDataRequest_SubscribeOrderBookRequest{SubscribeOrderBookRequest: &investapi.SubscribeOrderBookRequest{SubscriptionAction: densityAction(subscribe), Instruments: instruments}}}
}
func densityTradeSubscription(ids []string, subscribe bool) *investapi.MarketDataRequest {
	instruments := make([]*investapi.TradeInstrument, 0, len(ids))
	for _, id := range ids {
		instruments = append(instruments, &investapi.TradeInstrument{InstrumentId: id})
	}
	return &investapi.MarketDataRequest{Payload: &investapi.MarketDataRequest_SubscribeTradesRequest{SubscribeTradesRequest: &investapi.SubscribeTradesRequest{SubscriptionAction: densityAction(subscribe), Instruments: instruments, TradeSource: investapi.TradeSourceType_TRADE_SOURCE_EXCHANGE}}}
}

// densityBook rejects incomplete or non-exchange snapshots so callers can reset the episode.
func densityBook(b *investapi.OrderBook, lot int64) (density.Book, bool) {
	if b == nil || !b.GetIsConsistent() || b.GetOrderBookType() != investapi.OrderBookType_ORDERBOOK_TYPE_EXCHANGE || b.GetTime() == nil || b.GetTime().CheckValid() != nil || lot <= 0 {
		return density.Book{}, false
	}
	out := density.Book{InstrumentID: b.GetInstrumentUid(), Lot: lot, Time: b.GetTime().AsTime()}
	convert := func(orders []*investapi.Order) ([]density.Level, bool) {
		levels := make([]density.Level, 0, len(orders))
		for _, order := range orders {
			if order == nil || order.GetPrice() == nil {
				return nil, false
			}
			q := order.GetPrice()
			levels = append(levels, density.Level{Price: density.Price{Units: q.GetUnits(), Nano: q.GetNano()}, Quantity: order.GetQuantity()})
		}
		return levels, true
	}
	var ok bool
	if out.Bids, ok = convert(b.GetBids()); !ok {
		return density.Book{}, false
	}
	if out.Asks, ok = convert(b.GetAsks()); !ok {
		return density.Book{}, false
	}
	return out, true
}

func densityEvents(r *investapi.MarketDataResponse, lots map[string]int64) []density.Event {
	if r == nil {
		return nil
	}
	var out []density.Event
	if v := r.GetSubscribeOrderBookResponse(); v != nil {
		for _, s := range v.GetOrderBookSubscriptions() {
			if s != nil {
				out = append(out, density.Event{Kind: density.EventBookAck, InstrumentID: s.GetInstrumentUid(), Success: s.GetSubscriptionStatus() == investapi.SubscriptionStatus_SUBSCRIPTION_STATUS_SUCCESS && s.GetOrderBookType() == investapi.OrderBookType_ORDERBOOK_TYPE_EXCHANGE})
			}
		}
	}
	if v := r.GetSubscribeTradesResponse(); v != nil {
		for _, s := range v.GetTradeSubscriptions() {
			if s != nil {
				out = append(out, density.Event{Kind: density.EventTradeAck, InstrumentID: s.GetInstrumentUid(), Success: s.GetSubscriptionStatus() == investapi.SubscriptionStatus_SUBSCRIPTION_STATUS_SUCCESS && v.GetTradeSource() == investapi.TradeSourceType_TRADE_SOURCE_EXCHANGE})
			}
		}
	}
	if v := r.GetSubscribeInfoResponse(); v != nil {
		for _, s := range v.GetInfoSubscriptions() {
			if s != nil {
				out = append(out, density.Event{Kind: density.EventInfoAck, InstrumentID: s.GetInstrumentUid(), Success: s.GetSubscriptionStatus() == investapi.SubscriptionStatus_SUBSCRIPTION_STATUS_SUCCESS})
			}
		}
	}
	if v := r.GetSubscribeCandlesResponse(); v != nil {
		for _, sub := range v.GetCandlesSubscriptions() {
			if sub != nil {
				out = append(out, density.Event{Kind: density.EventCandleAck, InstrumentID: sub.GetInstrumentUid(), Success: sub.GetSubscriptionStatus() == investapi.SubscriptionStatus_SUBSCRIPTION_STATUS_SUCCESS && sub.GetWaitingClose()})
			}
		}
	}
	if v := r.GetCandle(); v != nil && v.GetInterval() == investapi.SubscriptionInterval_SUBSCRIPTION_INTERVAL_ONE_MINUTE &&
		v.GetTime() != nil && v.GetTime().CheckValid() == nil && v.GetVolume() > 0 {
		price := func(q *investapi.Quotation) density.Price {
			return density.Price{Units: q.GetUnits(), Nano: q.GetNano()}
		}
		out = append(out, density.Event{Kind: density.EventCandle, InstrumentID: v.GetInstrumentUid(), Candle: density.Candle{
			Time: v.GetTime().AsTime(), Open: price(v.GetOpen()), High: price(v.GetHigh()), Low: price(v.GetLow()), Close: price(v.GetClose()),
		}})
	}
	if v := r.GetOrderbook(); v != nil {
		b, ok := densityBook(v, lots[v.GetInstrumentUid()])
		kind := density.EventBook
		if !ok {
			kind = density.EventInvalidBook
		}
		out = append(out, density.Event{Kind: kind, InstrumentID: v.GetInstrumentUid(), Book: b})
	}
	if v := r.GetTrade(); v != nil && v.GetTradeSource() == investapi.TradeSourceType_TRADE_SOURCE_EXCHANGE && v.GetPrice() != nil {
		q := v.GetPrice()
		var at time.Time
		if v.GetTime() != nil && v.GetTime().CheckValid() == nil {
			at = v.GetTime().AsTime()
		}
		out = append(out, density.Event{Kind: density.EventTrade, InstrumentID: v.GetInstrumentUid(), Price: density.Price{Units: q.GetUnits(), Nano: q.GetNano()}, Quantity: v.GetQuantity(), Time: at})
	}
	if v := r.GetTradingStatus(); v != nil {
		out = append(out, density.Event{Kind: density.EventTradingStatus, InstrumentID: v.GetInstrumentUid(), TradingOpen: v.GetTradingStatus() == investapi.SecurityTradingStatus_SECURITY_TRADING_STATUS_NORMAL_TRADING && v.GetLimitOrderAvailableFlag()})
	}
	return out
}

func densityInfoSubscription(ids []string, subscribe bool) *investapi.MarketDataRequest {
	instruments := make([]*investapi.InfoInstrument, 0, len(ids))
	for _, id := range ids {
		instruments = append(instruments, &investapi.InfoInstrument{InstrumentId: id})
	}
	return &investapi.MarketDataRequest{Payload: &investapi.MarketDataRequest_SubscribeInfoRequest{SubscribeInfoRequest: &investapi.SubscribeInfoRequest{SubscriptionAction: densityAction(subscribe), Instruments: instruments}}}
}

type densityStream struct {
	stream googlegrpc.BidiStreamingClient[investapi.MarketDataRequest, investapi.MarketDataResponse]
	lots   map[string]int64
}

func (d *DensityMarketData) OpenStream(ctx context.Context, lots map[string]int64) (density.Stream, error) {
	stream, err := d.stream.MarketDataStream(ctx, NewRPCCredential(d.auth))
	if err != nil {
		return nil, err
	}
	copied := make(map[string]int64, len(lots))
	for id, lot := range lots {
		copied[id] = lot
	}
	return &densityStream{stream: stream, lots: copied}, nil
}
func (s *densityStream) SubscribeBooks(ids []string, depth int32, subscribe bool) error {
	return s.stream.Send(densityBookSubscription(ids, depth, subscribe))
}
func (s *densityStream) SubscribeTrades(ids []string, subscribe bool) error {
	return s.stream.Send(densityTradeSubscription(ids, subscribe))
}
func (s *densityStream) SubscribeInfo(ids []string, subscribe bool) error {
	return s.stream.Send(densityInfoSubscription(ids, subscribe))
}
func (s *densityStream) SubscribeClosedCandles(ids []string) error {
	instruments := make([]*investapi.CandleInstrument, 0, len(ids))
	for _, uid := range ids {
		instruments = append(instruments, &investapi.CandleInstrument{InstrumentId: uid, Interval: investapi.SubscriptionInterval_SUBSCRIPTION_INTERVAL_ONE_MINUTE})
	}
	source := investapi.GetCandlesRequest_CANDLE_SOURCE_EXCHANGE
	return s.stream.Send(&investapi.MarketDataRequest{Payload: &investapi.MarketDataRequest_SubscribeCandlesRequest{SubscribeCandlesRequest: &investapi.SubscribeCandlesRequest{
		SubscriptionAction: densityAction(true), Instruments: instruments, WaitingClose: true, CandleSourceType: &source,
	}}})
}
func (s *densityStream) Receive() ([]density.Event, error) {
	r, err := s.stream.Recv()
	if err != nil {
		return nil, err
	}
	return densityEvents(r, s.lots), nil
}
func (s *densityStream) Close() error { return s.stream.CloseSend() }
