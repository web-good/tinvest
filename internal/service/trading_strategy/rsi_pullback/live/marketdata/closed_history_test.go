package marketdata

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"tinvest/internal/domain/backtest"
)

func TestClosedHistoryAppendsStreamBarBeforeHistoricalAPIHasIt(t *testing.T) {
	at := time.Date(2026, 3, 10, 11, 30, 0, 0, msk)
	client := &fakeClient{
		m30: series30m(at.Add(-30*time.Minute), 20),
		day: seriesDaily(at, 20),
	}
	history := NewClosedHistory(client, "WUSH", 5)
	require.NoError(t, history.Warm(t.Context(), at))
	require.Len(t, client.windows, 2)
	md, err := history.Assemble(t.Context(), backtest.Candle{
		Time: at, Open: 100, High: 110, Low: 96, Close: 105, Volume: 1000,
	})
	require.NoError(t, err)
	require.Len(t, md.Times, 5)
	require.True(t, md.Times[4].Equal(at))
	require.Equal(t, 105.0, md.Price)
	require.Equal(t, 110.0, md.TodayHigh)
	require.Equal(t, 96.0, md.TodayLow)
	require.Len(t, client.windows, 2, "a contiguous close uses preloaded history")
}
