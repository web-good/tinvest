package scalping

import (
	"context"
	"strings"
	"testing"

	"github.com/golang/protobuf/ptypes/timestamp"

	imodel "tinvest/internal/model"
	"tinvest/internal/service/trading_strategy/scalping/dto"
	"tinvest/internal/service/trading_strategy/scalping/model"
	"tinvest/internal/service/trading_strategy/scalping/strategy"
	grpcmodel "tinvest/pkg/client/grpc/model"
	"tinvest/pkg/logger"
)

func TestMain(m *testing.M) {
	logger.Init()
	m.Run()
}

// --- fakes ---

type fakeStrategy struct {
	ticker   string
	lookback int
	sig      model.Signal
}

func (f fakeStrategy) Ticker() string                          { return f.ticker }
func (f fakeStrategy) Lookback() int                           { return f.lookback }
func (f fakeStrategy) Decide(strategy.MarketData) model.Signal { return f.sig }

type stubInstruments struct{ shares []*imodel.Share }

func (s stubInstruments) Shares(context.Context) ([]*imodel.Share, error) { return s.shares, nil }

type stubMarket struct {
	called  int
	candles []*imodel.CandleItemTechAnalyse
}

func (m *stubMarket) GetCandles(_ context.Context, _ *string, _ int32, _ *timestamp.Timestamp, _ *timestamp.Timestamp, _ *int32, _ bool) ([]*imodel.CandleItemTechAnalyse, error) {
	m.called++
	return m.candles, nil
}

type stubOps struct{ positions []*grpcmodel.Position }

func (o stubOps) GetPortfolio(context.Context, string) ([]*grpcmodel.Position, error) {
	return o.positions, nil
}

type stubTg struct{ msgs []string }

func (t *stubTg) SendMessage(msg string) error          { t.msgs = append(t.msgs, msg); return nil }
func (t *stubTg) SendMessageToChat(int64, string) error { return nil }

func tradableShare() *imodel.Share {
	return &imodel.Share{ID: "share-1", Ticker: "TEST", Name: "Test Co", Trading: true, Lot: 1}
}

func heldPosition() *grpcmodel.Position {
	return &grpcmodel.Position{InstrumentType: "share", ShareID: "share-1", Quantity: 10}
}

func oneCandle() []*imodel.CandleItemTechAnalyse {
	return []*imodel.CandleItemTechAnalyse{{
		Open:  imodel.Quotation{Units: 100},
		Close: imodel.Quotation{Units: 100},
		Low:   imodel.Quotation{Units: 99},
		High:  imodel.Quotation{Units: 101},
	}}
}

func TestTrade_SellOnly(t *testing.T) {
	tests := []struct {
		name         string
		sellOnly     bool
		positions    []*grpcmodel.Position
		sig          model.Signal
		wantMsgs     int
		wantFetched  bool
		wantContains string
	}{
		{
			name:         "held position with sell signal sends one alert",
			sellOnly:     true,
			positions:    []*grpcmodel.Position{heldPosition()},
			sig:          model.Signal{Kind: model.SignalSell, Reason: "TP"},
			wantMsgs:     1,
			wantFetched:  true,
			wantContains: "Мониторинг выхода",
		},
		{
			name:        "held position without signal stays silent",
			sellOnly:    true,
			positions:   []*grpcmodel.Position{heldPosition()},
			sig:         model.Signal{Kind: model.SignalNone},
			wantMsgs:    0,
			wantFetched: true,
		},
		{
			name:        "no position skips instrument before fetching candles",
			sellOnly:    true,
			positions:   nil,
			sig:         model.Signal{Kind: model.SignalSell, Reason: "TP"},
			wantMsgs:    0,
			wantFetched: false,
		},
		{
			name:        "buy signal is ignored in sell-only mode",
			sellOnly:    true,
			positions:   []*grpcmodel.Position{heldPosition()},
			sig:         model.Signal{Kind: model.SignalBuy},
			wantMsgs:    0,
			wantFetched: true,
		},
		{
			name:         "default mode still emits buy without a position",
			sellOnly:     false,
			positions:    nil,
			sig:          model.Signal{Kind: model.SignalBuy},
			wantMsgs:     1,
			wantFetched:  true,
			wantContains: "Скальпинг (1H)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			market := &stubMarket{candles: oneCandle()}
			tg := &stubTg{}
			svc := NewService(
				stubInstruments{shares: []*imodel.Share{tradableShare()}},
				market,
				stubOps{positions: tt.positions},
				tg,
				"acc-1",
				WithStrategies([]strategy.Strategy{fakeStrategy{ticker: "TEST", lookback: 1, sig: tt.sig}}),
			)

			if err := svc.Trade(context.Background(), dto.Trade{SellOnly: tt.sellOnly}); err != nil {
				t.Fatalf("Trade returned error: %v", err)
			}

			if len(tg.msgs) != tt.wantMsgs {
				t.Errorf("messages = %d, want %d (msgs=%v)", len(tg.msgs), tt.wantMsgs, tg.msgs)
			}
			if (market.called > 0) != tt.wantFetched {
				t.Errorf("candles fetched = %v, want %v (called=%d)", market.called > 0, tt.wantFetched, market.called)
			}
			if tt.wantContains != "" && (len(tg.msgs) == 0 || !strings.Contains(tg.msgs[0], tt.wantContains)) {
				t.Errorf("message missing %q (msgs=%v)", tt.wantContains, tg.msgs)
			}
		})
	}
}
