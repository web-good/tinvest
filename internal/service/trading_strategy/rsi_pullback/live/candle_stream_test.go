package live

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"tinvest/internal/config"
	"tinvest/internal/service/trading_strategy/rsi_pullback/live/dto"
	"tinvest/internal/service/trading_strategy/scalping/strategy"
)

func TestClosedCandlePassEvaluatesTickerOnceAcrossStreamAndCron(t *testing.T) {
	env := newSharedEnv(t, sharedNow, []string{"WUSH", "GAZP"}, func(cfg *config.RSIPullbackConfig) {
		cfg.CandleStreamTickers = []string{"WUSH"}
	})
	pullback := guestFake("rsi_pullback", 0, "WUSH", "GAZP")
	env.svc.slots[0].strat = pullback
	env.instruments.EXPECT().Shares(mock.Anything).Return(sharesOf("WUSH", "GAZP"), nil).Twice()
	env.ops.EXPECT().GetPortfolio(mock.Anything, mock.Anything).Return(nil, nil).Twice()

	signalBar := strategy.MarketData{Times: []time.Time{sharedLastBar}, Price: 40}
	event := closedCandlePass{ticker: "WUSH", instrumentID: "uid-WUSH", data: signalBar}
	require.NoError(t, env.svc.runPass(context.Background(), &event))
	require.Equal(t, 1, pullback.decideCalls)

	// Leave only WUSH in the cron universe so any repeat would request candles or Decide.
	pullback.universe = []string{"WUSH"}
	require.NoError(t, env.svc.Run(context.Background(), dto.Run{}))
	require.Equal(t, 1, pullback.decideCalls)
}

func TestClosedCandlePassLeavesGuestEntriesToCron(t *testing.T) {
	guest := zoneFake(0, "WUSH")
	env := newSharedEnv(t, sharedNow, []string{"WUSH"}, func(cfg *config.RSIPullbackConfig) {
		cfg.CandleStreamTickers = []string{"WUSH"}
	}, guest)
	pullback := guestFake("rsi_pullback", 0, "WUSH")
	env.svc.slots[0].strat = pullback
	env.instruments.EXPECT().Shares(mock.Anything).Return(sharesOf("WUSH"), nil).Twice()
	env.ops.EXPECT().GetPortfolio(mock.Anything, mock.Anything).Return(nil, nil).Twice()
	env.expectCandles(flatAt30m(sharedLastBar, 400, 40), dailies(sharedNow, 60))

	event := closedCandlePass{
		ticker: "WUSH", instrumentID: "uid-WUSH",
		data: strategy.MarketData{Times: []time.Time{sharedLastBar}, Price: 40},
	}
	require.NoError(t, env.svc.runPass(context.Background(), &event))
	require.Equal(t, 1, pullback.decideCalls)
	require.Zero(t, guest.decideCalls)

	require.NoError(t, env.svc.Run(context.Background(), dto.Run{}))
	require.Equal(t, 1, pullback.decideCalls)
	require.Equal(t, 1, guest.decideCalls)
}

func TestClosedCandlePassRejectsOpenAndLateBars(t *testing.T) {
	env := newSharedEnv(t, sharedNow, []string{"WUSH"}, func(cfg *config.RSIPullbackConfig) {
		cfg.CandleStreamTickers = []string{"WUSH"}
	})
	for _, bar := range []time.Time{sharedLastBar.Add(30 * time.Minute), sharedLastBar.Add(-30 * time.Minute)} {
		event := closedCandlePass{
			ticker: "WUSH", instrumentID: "uid-WUSH",
			data: strategy.MarketData{Times: []time.Time{bar}, Price: 40},
		}
		require.NoError(t, env.svc.runPass(context.Background(), &event))
	}
}
