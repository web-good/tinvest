package grpc

import (
	"context"
	"fmt"
	"strings"
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
	step := s.GetMinPriceIncrement()
	return density.UniverseInput{Instrument: density.Instrument{ID: s.GetUid(), Ticker: s.GetTicker(), Lot: int64(s.GetLot()), PriceStep: density.Price{Units: step.GetUnits(), Nano: step.GetNano()}}, Currency: s.GetCurrency(), Exchange: exchange, BuyAvailable: s.GetBuyAvailableFlag() && s.GetApiTradeAvailableFlag()}
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

// InstrumentsByTickers fetches only explicitly selected MOEX main-board shares.
// A bulk Shares response is unnecessarily large for a short ticker allowlist.
func (d *DensityMarketData) InstrumentsByTickers(ctx context.Context, tickers []string) ([]density.UniverseInput, error) {
	result := make([]density.UniverseInput, 0, len(tickers))
	classCode := "TQBR"
	for _, ticker := range tickers {
		ticker = strings.ToUpper(strings.TrimSpace(ticker))
		if ticker == "" {
			return nil, fmt.Errorf("empty share ticker")
		}
		requestCtx, cancel := context.WithTimeout(ctx, requestTimeout)
		resp, err := d.instruments.ShareBy(requestCtx, &investapi.InstrumentRequest{IdType: investapi.InstrumentIdType_INSTRUMENT_ID_TYPE_TICKER, ClassCode: &classCode, Id: ticker}, NewRPCCredential(d.auth))
		cancel()
		if err != nil {
			return nil, fmt.Errorf("share %s/%s: %w", ticker, classCode, err)
		}
		share := resp.GetInstrument()
		if share == nil || share.GetTicker() != ticker || share.GetUid() == "" {
			return nil, fmt.Errorf("share %s/%s: missing or mismatched instrument", ticker, classCode)
		}
		result = append(result, densityShare(share))
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

// SupportCandles loads existing hourly history, rather than recording order books.
func (d *DensityMarketData) SupportCandles(ctx context.Context, id string, days int) ([]density.SupportCandle, error) {
	if days < 1 || days > 90 {
		return nil, fmt.Errorf("support lookback must be 1..90 days")
	}
	now := d.now()
	ctx, cancel := context.WithTimeout(ctx, requestTimeout)
	defer cancel()
	resp, err := d.market.GetCandles(ctx, &investapi.GetCandlesRequest{InstrumentId: &id, From: timestamppb.New(now.AddDate(0, 0, -days)), To: timestamppb.New(now), Interval: investapi.CandleInterval_CANDLE_INTERVAL_HOUR}, NewRPCCredential(d.auth))
	if err != nil {
		return nil, fmt.Errorf("hourly candles: %w", err)
	}
	result := make([]density.SupportCandle, 0, len(resp.GetCandles()))
	for _, c := range resp.GetCandles() {
		if c == nil || c.GetTime() == nil || c.GetTime().CheckValid() != nil || c.GetLow() == nil || c.GetHigh() == nil || c.GetClose() == nil {
			continue
		}
		result = append(result, density.SupportCandle{Time: c.GetTime().AsTime(), Low: density.Price{Units: c.GetLow().GetUnits(), Nano: c.GetLow().GetNano()}, High: density.Price{Units: c.GetHigh().GetUnits(), Nano: c.GetHigh().GetNano()}, Close: density.Price{Units: c.GetClose().GetUnits(), Nano: c.GetClose().GetNano()}, Complete: c.GetIsComplete()})
	}
	return result, nil
}
