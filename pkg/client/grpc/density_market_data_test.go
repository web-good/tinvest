package grpc

import (
	"context"
	"errors"
	"slices"
	"strings"
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
	req     *investapi.GetCandlesRequest
	candles []*investapi.HistoricCandle
}

func TestDensitySupportCandlesUsesHourlyHistory(t *testing.T) {
	f := &densityMarketFake{}
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	d := &DensityMarketData{market: f, auth: NewAuth("test"), now: func() time.Time { return now }}
	got, err := d.SupportCandles(context.Background(), "uid", 60)
	if err != nil || f.req.GetInterval() != investapi.CandleInterval_CANDLE_INTERVAL_HOUR || !f.req.From.AsTime().Equal(now.AddDate(0, 0, -60)) || !f.req.To.AsTime().Equal(now) {
		t.Fatalf("request=%v err=%v", f.req, err)
	}
	if len(got) != 1 || got[0].Low.Units != 99 || got[0].High.Units != 101 || got[0].Close.Nano != 250000000 || !got[0].Complete {
		t.Fatalf("hourly conversion %+v", got)
	}
}

func (f *densityMarketFake) GetCandles(ctx context.Context, req *investapi.GetCandlesRequest, opts ...googlegrpc.CallOption) (*investapi.GetCandlesResponse, error) {
	f.req = req
	if f.candles != nil {
		return &investapi.GetCandlesResponse{Candles: f.candles}, nil
	}
	return &investapi.GetCandlesResponse{Candles: []*investapi.HistoricCandle{{Time: timestamppb.New(time.Date(2025, 12, 30, 0, 0, 0, 0, time.UTC)), Low: &investapi.Quotation{Units: 99}, High: &investapi.Quotation{Units: 101}, Close: &investapi.Quotation{Units: 100, Nano: 250000000}, Volume: 50, IsComplete: true}, {}}}, nil
}

func TestDensityShareRetainsPriceStep(t *testing.T) {
	s := &investapi.Share{MinPriceIncrement: &investapi.Quotation{Nano: 10000000}}
	if densityShare(s).Instrument.PriceStep.Nano != 10000000 {
		t.Fatal("missing tick size")
	}
}

type tickerSharesFake struct {
	investapi.InstrumentsServiceClient
	requests []*investapi.InstrumentRequest
	fail     error
}

func (f *tickerSharesFake) ShareBy(ctx context.Context, req *investapi.InstrumentRequest, _ ...googlegrpc.CallOption) (*investapi.ShareResponse, error) {
	f.requests = append(f.requests, req)
	if _, ok := ctx.Deadline(); !ok {
		panic("missing request deadline")
	}
	if f.fail != nil {
		return nil, f.fail
	}
	return &investapi.ShareResponse{Instrument: &investapi.Share{Uid: "uid-" + req.Id, Ticker: req.Id, ClassCode: "TQBR", Lot: 1, MinPriceIncrement: &investapi.Quotation{Nano: 10000000}, RealExchange: investapi.RealExchange_REAL_EXCHANGE_MOEX, Currency: "rub", BuyAvailableFlag: true, ApiTradeAvailableFlag: true}}, nil
}
func TestDensityTargetedTickersSkipBulkShares(t *testing.T) {
	f := &tickerSharesFake{}
	d := &DensityMarketData{instruments: f, auth: NewAuth("test")}
	got, err := d.InstrumentsByTickers(context.Background(), []string{"T", "SBER"})
	if err != nil || len(got) != 2 || len(f.requests) != 2 {
		t.Fatalf("%+v %v", got, err)
	}
	for _, r := range f.requests {
		if r.IdType != investapi.InstrumentIdType_INSTRUMENT_ID_TYPE_TICKER || r.GetClassCode() != "TQBR" {
			t.Fatal(r)
		}
	}
	if got[0].Instrument.ID != "uid-T" || got[1].Instrument.PriceStep.Nano != 10000000 {
		t.Fatal(got)
	}
}
func TestDensityTargetedTickersPropagateError(t *testing.T) {
	f := &tickerSharesFake{fail: context.DeadlineExceeded}
	d := &DensityMarketData{instruments: f, auth: NewAuth("test")}
	got, err := d.InstrumentsByTickers(context.Background(), []string{"T"})
	if len(got) != 0 || !errors.Is(err, context.DeadlineExceeded) || !strings.Contains(err.Error(), "T") {
		t.Fatalf("%+v %v", got, err)
	}
}

func TestDensityMinuteClosesRequireLatestCompletedMinute(t *testing.T) {
	end := time.Date(2026, 11, 18, 12, 0, 0, 0, time.FixedZone("MSK", 3*60*60))
	candle := func(offset time.Duration, close int64, complete bool) *investapi.HistoricCandle {
		return &investapi.HistoricCandle{Time: timestamppb.New(end.Add(offset)), Close: &investapi.Quotation{Units: close}, IsComplete: complete}
	}
	f := &densityMarketFake{candles: []*investapi.HistoricCandle{
		candle(-time.Minute, 102, true),
		candle(0, 103, false),
		candle(-2*time.Minute, 101, true),
		candle(-2*time.Hour, 55, true),
	}}
	d := &DensityMarketData{market: f, auth: NewAuth("test")}
	got, err := d.MinuteCloses(t.Context(), "uid", end)
	if err != nil || !slices.Equal(got, []float64{101e9, 102e9}) {
		t.Fatalf("minute closes=%v err=%v", got, err)
	}
	if f.req.GetInterval() != investapi.CandleInterval_CANDLE_INTERVAL_1_MIN ||
		f.req.GetInstrumentId() != "uid" || !f.req.GetTo().AsTime().Equal(end) {
		t.Fatalf("request=%v", f.req)
	}
	f.candles[0].IsComplete = false
	got, err = d.MinuteCloses(t.Context(), "uid", end)
	if err != nil || len(got) != 0 {
		t.Fatalf("incomplete latest candle: closes=%v err=%v", got, err)
	}
}
