package grpc

import (
	"context"
	"fmt"

	googlegrpc "google.golang.org/grpc"

	"tinvest/internal/domain/backtest"
	investapi "tinvest/internal/pb/v1"
	"tinvest/internal/utils"
)

// NewM30CandleSource subscribes to the same candle source as GetCandles and requests
// delivery after the bar closes.
func NewM30CandleSource(conn googlegrpc.ClientConnInterface, token string) func(context.Context, []string, func(string, backtest.Candle)) error {
	client := investapi.NewMarketDataStreamServiceClient(conn)
	auth := NewAuth(token)
	return func(ctx context.Context, ids []string, receive func(string, backtest.Candle)) error {
		interval := investapi.SubscriptionInterval_SUBSCRIPTION_INTERVAL_30_MIN
		instruments := make([]*investapi.CandleInstrument, 0, len(ids))
		for _, id := range ids {
			instruments = append(instruments, &investapi.CandleInstrument{InstrumentId: id, Interval: interval})
		}
		source := investapi.GetCandlesRequest_CANDLE_SOURCE_UNSPECIFIED
		stream, err := client.MarketDataServerSideStream(ctx, &investapi.MarketDataServerSideStreamRequest{
			SubscribeCandlesRequest: &investapi.SubscribeCandlesRequest{
				SubscriptionAction: investapi.SubscriptionAction_SUBSCRIPTION_ACTION_SUBSCRIBE,
				Instruments:        instruments, WaitingClose: true, CandleSourceType: &source,
			},
		}, NewRPCCredential(auth))
		if err != nil {
			return err
		}
		confirmed := make(map[string]bool, len(ids))
		for {
			frame, err := stream.Recv()
			if err != nil {
				return err
			}
			for _, sub := range frame.GetSubscribeCandlesResponse().GetCandlesSubscriptions() {
				if sub.GetSubscriptionStatus() != investapi.SubscriptionStatus_SUBSCRIPTION_STATUS_SUCCESS || !sub.GetWaitingClose() {
					return fmt.Errorf("closed M30 subscription for %s: %s (waiting_close=%t)",
						sub.GetInstrumentUid(), sub.GetSubscriptionStatus(), sub.GetWaitingClose())
				}
				confirmed[sub.GetInstrumentUid()] = true
			}
			c := frame.GetCandle()
			if c == nil || !confirmed[c.GetInstrumentUid()] || c.GetInterval() != interval ||
				c.GetTime() == nil || c.GetTime().CheckValid() != nil ||
				c.GetOpen() == nil || c.GetHigh() == nil || c.GetLow() == nil || c.GetClose() == nil {
				continue
			}
			receive(c.GetInstrumentUid(), backtest.Candle{
				Time: c.GetTime().AsTime(), Volume: c.GetVolume(),
				Open:  utils.CombinePrice(c.GetOpen().GetUnits(), c.GetOpen().GetNano()),
				High:  utils.CombinePrice(c.GetHigh().GetUnits(), c.GetHigh().GetNano()),
				Low:   utils.CombinePrice(c.GetLow().GetUnits(), c.GetLow().GetNano()),
				Close: utils.CombinePrice(c.GetClose().GetUnits(), c.GetClose().GetNano()),
			})
		}
	}
}
