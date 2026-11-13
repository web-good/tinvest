package grpc

import (
	"context"
	"testing"
	"time"
	investapi "tinvest/internal/pb/v1"

	googlegrpc "google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestDensityShareConversion(t *testing.T) {
	s := &investapi.Share{Uid: "uid", Ticker: "TEST", Lot: 10, Currency: "rub", RealExchange: investapi.RealExchange_REAL_EXCHANGE_MOEX, BuyAvailableFlag: true, ApiTradeAvailableFlag: true}
	got := densityShare(s)
	if got.Instrument.ID != "uid" || got.Exchange != "MOEX" || !got.BuyAvailable {
		t.Fatalf("unexpected %+v", got)
	}
	s.ApiTradeAvailableFlag = false
	if densityShare(s).BuyAvailable {
		t.Fatal("API unavailable accepted")
	}
	s.RealExchange = investapi.RealExchange_REAL_EXCHANGE_RTS
	if densityShare(s).Exchange == "MOEX" {
		t.Fatal("non-MOEX accepted")
	}
}

func TestDensityCandlesConversion(t *testing.T) {
	f := &densityMarketFake{}
	d := &DensityMarketData{market: f, auth: NewAuth("test"), now: func() time.Time { return time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC) }}
	candles, err := d.Candles(context.Background(), "uid")
	if err != nil || len(candles) != 1 || candles[0].Close.Nano != 250000000 || candles[0].Volume != 50 || !candles[0].Complete {
		t.Fatalf("candles=%+v err=%v", candles, err)
	}
	if f.req.GetInstrumentId() != "uid" || f.req.GetInterval() != investapi.CandleInterval_CANDLE_INTERVAL_DAY {
		t.Fatalf("request=%v", f.req)
	}
}

type densityMarketFake struct {
	investapi.MarketDataServiceClient
	req *investapi.GetCandlesRequest
}

func (f *densityMarketFake) GetCandles(ctx context.Context, req *investapi.GetCandlesRequest, opts ...googlegrpc.CallOption) (*investapi.GetCandlesResponse, error) {
	f.req = req
	return &investapi.GetCandlesResponse{Candles: []*investapi.HistoricCandle{{Time: timestamppb.New(time.Date(2025, 12, 30, 0, 0, 0, 0, time.UTC)), Close: &investapi.Quotation{Units: 100, Nano: 250000000}, Volume: 50, IsComplete: true}, {}}}, nil
}
