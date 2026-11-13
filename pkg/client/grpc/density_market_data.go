package grpc

import (
	"context"
	"fmt"
	"time"
	investapi "tinvest/internal/pb/v1"
	"tinvest/internal/service/screener/density"

	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type DensityMarketData struct {
	instruments investapi.InstrumentsServiceClient
	market      investapi.MarketDataServiceClient
	stream      investapi.MarketDataStreamServiceClient
	auth        *Auth
	now         func() time.Time
}

func NewDensityMarketData(conn grpc.ClientConnInterface, token string) *DensityMarketData {
	return &DensityMarketData{instruments: investapi.NewInstrumentsServiceClient(conn), market: investapi.NewMarketDataServiceClient(conn), stream: investapi.NewMarketDataStreamServiceClient(conn), auth: NewAuth(token), now: time.Now}
}
func densityShare(s *investapi.Share) density.UniverseInput {
	exchange := ""
	if s.GetRealExchange() == investapi.RealExchange_REAL_EXCHANGE_MOEX {
		exchange = "MOEX"
	}
	return density.UniverseInput{Instrument: density.Instrument{ID: s.GetUid(), Ticker: s.GetTicker(), Lot: int64(s.GetLot())}, Currency: s.GetCurrency(), Exchange: exchange, BuyAvailable: s.GetBuyAvailableFlag() && s.GetApiTradeAvailableFlag()}
}
func (d *DensityMarketData) Instruments(ctx context.Context) ([]density.UniverseInput, error) {
	ctx, cancel := context.WithTimeout(ctx, listTimeout)
	defer cancel()
	resp, err := d.instruments.Shares(ctx, &investapi.InstrumentsRequest{InstrumentStatus: investapi.InstrumentStatus_INSTRUMENT_STATUS_BASE.Enum(), InstrumentExchange: investapi.InstrumentExchangeType_INSTRUMENT_EXCHANGE_UNSPECIFIED.Enum()}, NewRPCCredential(d.auth))
	if err != nil {
		return nil, fmt.Errorf("shares: %w", err)
	}
	result := make([]density.UniverseInput, 0, len(resp.GetInstruments()))
	for _, s := range resp.GetInstruments() {
		if s != nil {
			result = append(result, densityShare(s))
		}
	}
	return result, nil
}
func (d *DensityMarketData) Candles(ctx context.Context, id string) ([]density.DailyCandle, error) {
	now := d.now()
	ctx, cancel := context.WithTimeout(ctx, requestTimeout)
	defer cancel()
	resp, err := d.market.GetCandles(ctx, &investapi.GetCandlesRequest{InstrumentId: &id, From: timestamppb.New(now.AddDate(0, 0, -70)), To: timestamppb.New(now), Interval: investapi.CandleInterval_CANDLE_INTERVAL_DAY}, NewRPCCredential(d.auth))
	if err != nil {
		return nil, fmt.Errorf("candles: %w", err)
	}
	result := make([]density.DailyCandle, 0, len(resp.GetCandles()))
	for _, c := range resp.GetCandles() {
		if c == nil || c.GetTime() == nil || c.GetTime().CheckValid() != nil || c.GetClose() == nil {
			continue
		}
		q := c.GetClose()
		result = append(result, density.DailyCandle{Time: c.GetTime().AsTime(), Close: density.Price{Units: q.GetUnits(), Nano: q.GetNano()}, Volume: c.GetVolume(), Complete: c.GetIsComplete()})
	}
	return result, nil
}
