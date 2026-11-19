package live

import (
	"context"
	"fmt"
	"slices"
	"time"

	"github.com/robfig/cron/v3"

	"tinvest/internal/domain/backtest"
	imodel "tinvest/internal/model"
	"tinvest/internal/service/trading_strategy/livecore/adapter"
	"tinvest/internal/service/trading_strategy/livecore/statestore"
	"tinvest/internal/service/trading_strategy/rsi_pullback/live/marketdata"
	"tinvest/internal/service/trading_strategy/scalping/strategy"
	"tinvest/pkg/logger"
)

// ClosedCandleSource delivers completed M30 candles keyed by instrument UID.
type ClosedCandleSource func(context.Context, []string, func(string, backtest.Candle)) error

type closedCandlePass struct {
	ticker       string
	instrumentID string
	data         strategy.MarketData
}

type closedBarDecision struct {
	bar  time.Time
	busy bool // a position was managed or an entry signal took priority
}

func (p *closedCandlePass) freshAt(now time.Time) bool {
	return recentClose(p.data.Times[len(p.data.Times)-1], now)
}

func recentClose(open, now time.Time) bool {
	age := now.Sub(open.Add(barSpan))
	return age >= 0 && age <= time.Minute && open.Equal(open.Truncate(barSpan))
}

func hasPendingOrder(e statestore.Entry) bool {
	return e.EntryRequestID != "" || e.PendingExit != "" || e.FVGExitOrderID != "" || e.FVGExitStopID != "" ||
		e.FVGRecovering || e.FVGStopPending || e.FVGTargetPending
}

// SetClosedCandleSource is called by the service provider before starting the runner.
func (s *service) SetClosedCandleSource(source ClosedCandleSource) {
	s.candleSource = source
}

func (s *service) candleStreamTicker(ticker string) bool {
	return slices.Contains(s.cfg.CandleStreamTickers, ticker) && slices.Contains(s.slots[0].strat.Tickers(), ticker)
}

// RunCandleStream reconnects independently of cron. Both paths enter runPass, which serializes
// broker actions and skips a completed decision when the other path already processed its bar.
func (s *service) RunCandleStream(ctx context.Context) {
	if s.candleSource == nil || len(s.cfg.CandleStreamTickers) == 0 {
		return
	}
	schedule, err := cron.ParseStandard(s.cfg.Schedule)
	if err != nil {
		logger.ErrorContext(ctx, fmt.Sprintf("rsi_pullback: candle stream schedule: %v", err))
		return
	}
	for ctx.Err() == nil {
		err := s.runCandleSubscription(ctx, schedule)
		if ctx.Err() != nil {
			return
		}
		logger.InfoContext(ctx, fmt.Sprintf("rsi_pullback: closed M30 stream unavailable, cron active: %v", err))
		timer := time.NewTimer(5 * time.Second)
		select {
		case <-ctx.Done():
			timer.Stop()
			return
		case <-timer.C:
		}
	}
}

func (s *service) runCandleSubscription(ctx context.Context, schedule cron.Schedule) error {
	shares, err := s.sharesByTicker(ctx)
	if err != nil {
		return err
	}
	type instrument struct {
		ticker  string
		history marketdata.ClosedHistory
	}
	var ids []string
	watch := make(map[string]*instrument)
	for ticker, share := range shares {
		if !s.candleStreamTicker(ticker) {
			continue
		}
		dec, ok := s.slots[0].strat.Decider(ticker)
		if !ok {
			continue
		}
		h := marketdata.NewClosedHistory(s.market, share.ID, dec.Lookback())
		if err := h.Warm(ctx, s.now()); err != nil {
			logger.InfoContext(ctx, fmt.Sprintf("rsi_pullback: %s candle stream warmup: %v", ticker, err))
		}
		ids = append(ids, share.ID)
		watch[share.ID] = &instrument{ticker: ticker, history: h}
	}
	if len(ids) == 0 {
		return fmt.Errorf("no eligible RSI Pullback tickers for candle stream")
	}

	return s.candleSource(ctx, ids, func(uid string, bar backtest.Candle) {
		inst := watch[uid]
		now := s.now()
		if inst == nil || !recentClose(bar.Time, now) {
			return
		}
		// Accelerate only close events that the existing schedule covers.
		closeAt := bar.Time.Add(barSpan).In(now.Location())
		if schedule.Next(closeAt.Add(-time.Second)).After(closeAt.Add(time.Minute)) {
			return
		}
		md, err := inst.history.Assemble(ctx, bar)
		if err != nil {
			logger.InfoContext(ctx, fmt.Sprintf("rsi_pullback: %s closed M30 data: %v", inst.ticker, err))
			return
		}
		if err := s.runPass(ctx, &closedCandlePass{ticker: inst.ticker, instrumentID: uid, data: md}); err != nil {
			logger.ErrorContext(ctx, fmt.Sprintf("rsi_pullback: %s closed M30 pass: %v", inst.ticker, err))
		}
	})
}

func (s *service) passMarketData(ctx context.Context, event *closedCandlePass, c cand, ticker string,
	sh *imodel.Share, now time.Time, watched bool, decided time.Time) (strategy.MarketData, bool) {
	if event != nil && c.sl.strat.Name() == adapter.LegacyOwner {
		return event.data, event.freshAt(now)
	}
	return s.assemble(ctx, c.sl, ticker, sh, c.dec, now, watched, decided)
}
