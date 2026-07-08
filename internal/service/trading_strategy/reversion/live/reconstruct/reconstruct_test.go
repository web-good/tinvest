package reconstruct

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/mock"

	imodel "tinvest/internal/model"
	marketdatamocks "tinvest/internal/service/trading_strategy/reversion/live/marketdata/mocks"
	"tinvest/internal/service/trading_strategy/reversion/live/reconstruct/mocks"
	grpcmodel "tinvest/pkg/client/grpc/model"
)

func q(f float64) imodel.Quotation { return imodel.Quotation{Units: int64(f)} }

func TestEntry_RebuildsFromMostRecentBuy(t *testing.T) {
	entryTime := time.Date(2026, 6, 1, 10, 0, 0, 0, time.UTC)
	now := entryTime.Add(20 * time.Hour)

	tc := mocks.NewMockTradesClient(t)
	tc.EXPECT().
		GetInstrumentTrades(mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return([]grpcmodel.Trade{
			{Date: entryTime.Add(-48 * time.Hour), Price: 90, Quantity: 10, IsBuy: true}, // older buy
			{Date: entryTime, Price: 100, Quantity: 10, IsBuy: true},                     // most recent buy
		}, nil).
		Once()

	var candles []*imodel.CandleItemTechAnalyse
	for i := 0; i < 40; i++ {
		ts := entryTime.Add(time.Duration(i-20) * time.Hour)
		c := 100.0 + float64(i) // rising -> later closes are higher
		candles = append(candles, &imodel.CandleItemTechAnalyse{
			Time: ts, Open: q(c), High: q(c + 1), Low: q(c - 1), Close: q(c), Volume: 1000, IsComplete: true,
		})
	}
	cc := marketdatamocks.NewMockCandleClient(t)
	cc.EXPECT().
		GetCandles(mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(candles, nil).
		Once()

	got, err := Entry(context.Background(), tc, cc, "acc", "uid", "UGLD", 100, 14, 50, now)
	if err != nil {
		t.Fatalf("Entry: %v", err)
	}
	if !got.EntryTime.Equal(entryTime) {
		t.Fatalf("EntryTime = %v, want %v", got.EntryTime, entryTime)
	}
	if got.EntryPrice != 100 {
		t.Fatalf("EntryPrice = %v, want 100 (broker avg)", got.EntryPrice)
	}
	if got.MaxFav < got.EntryPrice {
		t.Fatalf("MaxFav %v < EntryPrice %v", got.MaxFav, got.EntryPrice)
	}
	if got.EntryATR <= 0 {
		t.Fatalf("EntryATR = %v, want > 0", got.EntryATR)
	}
}
