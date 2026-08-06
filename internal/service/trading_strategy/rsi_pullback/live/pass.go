package live

import (
	"context"
	"fmt"
	"time"

	imodel "tinvest/internal/model"
	"tinvest/internal/service/trading_strategy/livecore/notifier"
	"tinvest/internal/service/trading_strategy/livecore/sizing"
	"tinvest/internal/service/trading_strategy/livecore/statestore"
	"tinvest/internal/service/trading_strategy/livecore/stoporders"
	"tinvest/internal/service/trading_strategy/rsi_pullback/live/marketdata"
	"tinvest/internal/service/trading_strategy/rsi_pullback/strategy/core"
	"tinvest/internal/service/trading_strategy/scalping/model"
	"tinvest/internal/service/trading_strategy/scalping/strategy"
	grpcmodel "tinvest/pkg/client/grpc/model"
	"tinvest/pkg/logger"
)

// maxBarAge is how stale the latest completed 30-minute bar may be before the runner
// refuses to act on it. The cron fires every half hour, so anything older than an hour
// means the feed is behind (holiday, halt, API outage) — and a decision taken on a stale
// bar is a decision taken on the wrong price.
const maxBarAge = 60 * time.Minute

// passCtx несёт состояние, общее для всех тикеров одного пасса: снапшот биржевых заявок
// берётся ОДИН раз на пасс, иначе повторный Cancel одной и той же заявки в одном тике
// дал бы ложный алерт.
type passCtx struct {
	state            map[string]statestore.Entry
	store            statestore.Store
	stopByID         map[string]stoporders.ActiveStop
	stopByInstrument map[string]stoporders.ActiveStop
	listErr          error
	now              time.Time
}

func (s *service) pass(ctx context.Context) error {
	shares, err := s.sharesByTicker(ctx)
	if err != nil {
		return err
	}
	held, err := s.heldByShareID(ctx)
	if err != nil {
		return err
	}
	store := s.stateStore()
	state, err := store.Load()
	if err != nil {
		return fmt.Errorf("rsi_pullback: load state: %w", err)
	}

	activeStops, listErr := s.stops.List(ctx) // один вызов на весь пасс
	if listErr != nil {
		s.notify(notifier.Alert(alertLabel, "", "GetStopOrders недоступен: "+listErr.Error()))
	}
	stopByInstrument := map[string]stoporders.ActiveStop{}
	stopByID := map[string]stoporders.ActiveStop{}
	for _, a := range activeStops {
		stopByInstrument[a.InstrumentUID] = a
		stopByID[a.StopOrderID] = a
	}

	now := s.now()
	for _, ticker := range s.cfg.Tickers {
		st, ok := StrategyFor(ticker)
		if !ok {
			s.notify(notifier.Alert(alertLabel, ticker, "тикер не зарегистрирован в rsi_pullback — пропуск"))
			continue
		}
		sh, ok := shares[ticker]
		if !ok || !sh.Trading {
			logger.ErrorContext(ctx, fmt.Sprintf("rsi_pullback: %s not tradable, skip", ticker))
			continue
		}
		md, err := marketdata.Assemble(ctx, s.market, sh.ID, st.Lookback(), now)
		if err != nil {
			logger.ErrorContext(ctx, fmt.Sprintf("rsi_pullback: %s marketdata: %v", ticker, err))
			continue
		}
		if n := len(md.Times); n == 0 || now.Sub(md.Times[n-1]) > maxBarAge {
			logger.ErrorContext(ctx, fmt.Sprintf("rsi_pullback: %s stale bar, skip", ticker))
			continue
		}

		pc := &passCtx{
			state: state, store: store, stopByID: stopByID,
			stopByInstrument: stopByInstrument, listErr: listErr, now: now,
		}
		pos, isHeld := held[sh.ID]
		if _, hasState := state[ticker]; isHeld || hasState {
			if err := s.manage(ctx, pc, ticker, sh, st, md, pos, isHeld); err != nil {
				return err
			}
			continue
		}
		if err := s.buy(ctx, pc, ticker, sh, st, md); err != nil {
			return err
		}
	}
	return nil
}

// buy opens a long when the core signals one: size from the configured percentage of the
// account, market BUY, state (with the frozen ATR and target), and immediately the
// protective exchange stop — the position must never live a single tick unprotected,
// because the next check is half an hour away while the stop is an intrabar one.
func (s *service) buy(ctx context.Context, pc *passCtx, ticker string, sh *imodel.Share,
	st *core.Strategy, md strategy.MarketData) error {

	md.Position = nil

	sig := st.Decide(md)
	if sig.Kind != model.SignalBuy {
		return nil
	}

	total, err := s.ops.GetPortfolioTotal(ctx, s.cfg.AccountID)
	if err != nil {
		return fmt.Errorf("rsi_pullback: portfolio total: %w", err)
	}
	cash, err := s.ops.GetAvailableCash(ctx, s.cfg.AccountID)
	if err != nil {
		return fmt.Errorf("rsi_pullback: cash: %w", err)
	}
	lots, ok, reason := sizing.Lots(s.cfg.BuyPct, total, cash, sig.Price, sh.Lot)
	if !ok {
		s.notify(notifier.Skip(ticker, reason))
		return nil
	}

	res, err := s.exec.Buy(ctx, sh.ID, lots)
	if err != nil {
		s.notify(notifier.Alert(alertLabel, ticker, "ордер на покупку отклонён: "+err.Error()))
		logger.ErrorContext(ctx, fmt.Sprintf("rsi_pullback: %s buy rejected: %v", ticker, err))
		return nil // state unchanged; retried next tick
	}

	fillPrice := sig.Price
	filledLots := lots
	if res.Placed {
		if res.FillPrice > 0 {
			fillPrice = res.FillPrice
		}
		if res.FilledLots > 0 {
			filledLots = res.FilledLots
		}
	}
	qty := filledLots * int64(sh.Lot)

	pc.state[ticker] = statestore.Entry{
		Ticker:     ticker,
		EntryTime:  pc.now,
		EntryPrice: fillPrice,
		EntryATR:   sig.ATR, // дневной ATR: им же меряются стоп, цель и обе границы гейта дня
		TakeProfit: sig.TakeProfit,
		MaxFav:     fillPrice,
		Quantity:   qty,
	}
	if err := pc.store.Save(pc.state); err != nil {
		return fmt.Errorf("rsi_pullback: save state after buy %s: %w", ticker, err)
	}
	s.notify(notifier.Entry(ticker, fillPrice, filledLots, qty, !res.Placed))

	pc.state[ticker] = s.placeInitialStop(ctx, pc, ticker, sh, pc.state[ticker])
	return nil
}

// placeInitialStop puts the protective exchange stop right after a fill so the position is
// never unprotected for the first half hour. There is no UseIntrabarStop switch here: the
// stop of this strategy is intrabar by definition, so it is always placed. Placement and
// stamping are delegated to replaceStop (same guard/rounding/notification path as manage);
// for a fresh entry StopPrice is 0, so the StopSet notification always fires. On failure the
// entry keeps an empty StopOrderID and the next pass retries.
func (s *service) placeInitialStop(ctx context.Context, pc *passCtx, ticker string,
	sh *imodel.Share, entry statestore.Entry) statestore.Entry {

	level, reason := core.DesiredStop(mustParams(ticker), entry.EntryPrice, entry.EntryATR, entry.MaxFav)
	if reason == "" {
		return entry
	}
	entry = s.replaceStop(ctx, ticker, sh, entry, level, reason)
	pc.state[ticker] = entry
	_ = pc.store.Save(pc.state)
	return entry
}

// mustParams: ParamsFor гарантированно ok — тикер прошёл StrategyFor выше.
func mustParams(ticker string) core.Params {
	p, _ := ParamsFor(ticker)
	return p
}

// replaceStop places a stop at level and stamps the entry (id only when actually placed;
// price/reason always). StopPrice is stamped ROUNDED to the instrument's price increment,
// so the state mirrors the exchange-side order (dry-run included).
func (s *service) replaceStop(ctx context.Context, ticker string, sh *imodel.Share,
	entry statestore.Entry, level float64, reason string) statestore.Entry {

	if sh.Lot <= 0 {
		s.notify(notifier.Alert(alertLabel, ticker, "sh.Lot == 0 — невозможно вычислить лоты для стоп-заявки, пропуск"))
		logger.ErrorContext(ctx, fmt.Sprintf("rsi_pullback: %s sh.Lot=%d, skipping stop placement to avoid divide-by-zero", ticker, sh.Lot))
		return entry
	}
	lots := entry.Quantity / int64(sh.Lot)
	res, err := s.stops.Place(ctx, sh.ID, lots, level, sh.MinPriceIncrement)
	if err != nil {
		s.notify(notifier.Alert(alertLabel, ticker, "стоп-заявка не выставлена: "+err.Error()))
		logger.ErrorContext(ctx, fmt.Sprintf("rsi_pullback: %s place stop: %v", ticker, err))
		return entry
	}
	if res.Placed {
		entry.StopOrderID = res.OrderID
	}
	rounded := stoporders.RoundDownToIncrement(level, sh.MinPriceIncrement)
	changed := rounded != entry.StopPrice || reason != entry.StopReason
	entry.StopPrice, entry.StopReason = rounded, reason
	if changed {
		s.notify(notifier.StopSet(ticker, rounded, reason, !res.Placed))
	}
	return entry
}

func (s *service) manage(ctx context.Context, pc *passCtx, ticker string, sh *imodel.Share,
	st *core.Strategy, md strategy.MarketData, pos *grpcmodel.Position, isHeld bool) error {
	return nil
}
