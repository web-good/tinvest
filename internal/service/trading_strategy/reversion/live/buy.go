package live

import (
	"context"
	"fmt"

	imodel "tinvest/internal/model"
	"tinvest/internal/service/trading_strategy/reversion/live/marketdata"
	"tinvest/internal/service/trading_strategy/reversion/live/notifier"
	"tinvest/internal/service/trading_strategy/reversion/live/sizing"
	"tinvest/internal/service/trading_strategy/reversion/live/statestore"
	"tinvest/internal/service/trading_strategy/reversion/strategy/core"
	"tinvest/internal/service/trading_strategy/scalping/model"
	"tinvest/pkg/logger"
)

func (s *service) buyPass(ctx context.Context) error {
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
		return fmt.Errorf("reversion: load state: %w", err)
	}

	now := nowMSK()
	for _, ticker := range s.cfg.Tickers {
		st, ok := StrategyFor(ticker)
		if !ok {
			s.notify(notifier.Alert(ticker, "тикер не зарегистрирован в reversion — пропуск"))
			continue
		}
		sh, ok := shares[ticker]
		if !ok || !sh.Trading {
			logger.ErrorContext(ctx, fmt.Sprintf("reversion: %s not tradable, skip", ticker))
			continue
		}
		// Already in a position (broker or our state) -> manage-pass handles it.
		if _, h := held[sh.ID]; h {
			continue
		}
		if _, h := state[ticker]; h {
			continue
		}

		md, err := marketdata.Assemble(ctx, s.market, sh.ID, st.Lookback(), MaxHTFTrendEMA([]string{ticker}), now)
		if err != nil {
			logger.ErrorContext(ctx, fmt.Sprintf("reversion: %s marketdata: %v", ticker, err))
			continue
		}
		md.Position = nil

		sig := st.Decide(md)
		if sig.Kind != model.SignalBuy {
			continue
		}

		total, err := s.ops.GetPortfolioTotal(ctx, s.cfg.AccountID)
		if err != nil {
			return fmt.Errorf("reversion: portfolio total: %w", err)
		}
		cash, err := s.ops.GetAvailableCash(ctx, s.cfg.AccountID)
		if err != nil {
			return fmt.Errorf("reversion: cash: %w", err)
		}
		lots, ok, reason := sizing.Lots(s.cfg.BuyPct, total, cash, sig.Price, sh.Lot)
		if !ok {
			s.notify(notifier.Skip(ticker, reason))
			continue
		}

		res, err := s.exec.Buy(ctx, sh.ID, lots)
		if err != nil {
			s.notify(notifier.Alert(ticker, "ордер на покупку отклонён: "+err.Error()))
			logger.ErrorContext(ctx, fmt.Sprintf("reversion: %s buy rejected: %v", ticker, err))
			continue // state unchanged; retried next tick
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

		state[ticker] = statestore.Entry{
			Ticker:     ticker,
			EntryTime:  now,
			EntryPrice: fillPrice,
			EntryATR:   sig.ATR,
			MaxFav:     fillPrice,
			Quantity:   qty,
		}
		if err := store.Save(state); err != nil {
			return fmt.Errorf("reversion: save state after buy %s: %w", ticker, err)
		}
		s.notify(notifier.Entry(ticker, fillPrice, filledLots, qty, !res.Placed))

		state[ticker] = s.placeInitialStop(ctx, ticker, sh, state[ticker], state, store)
	}
	return nil
}

// placeInitialStop puts the protective exchange stop right after a fill so the
// position is never unprotected for the first hour. On failure the entry keeps an
// empty StopOrderID and managePass retries next tick.
func (s *service) placeInitialStop(ctx context.Context, ticker string, sh *imodel.Share,
	entry statestore.Entry, state map[string]statestore.Entry, store statestore.Store) statestore.Entry {

	p, ok := ParamsFor(ticker)
	if !ok || p.UseIntrabarStop != 1 {
		return entry // close-модель: биржевой стоп не выставляется, выходы — по сигналам ядра на закрытии часа
	}
	level, reason := core.DesiredStop(p, entry.EntryPrice, entry.EntryATR, entry.MaxFav)
	if reason == "" {
		return entry
	}
	lots := entry.Quantity / int64(sh.Lot)
	res, err := s.stops.Place(ctx, sh.ID, lots, level, sh.MinPriceIncrement)
	if err != nil {
		s.notify(notifier.Alert(ticker, "стоп-заявка не выставлена: "+err.Error()))
		logger.ErrorContext(ctx, fmt.Sprintf("reversion: %s place stop: %v", ticker, err))
		return entry
	}
	if res.Placed {
		entry.StopOrderID = res.OrderID
	}
	entry.StopPrice, entry.StopReason = level, reason
	state[ticker] = entry
	_ = store.Save(state)
	s.notify(notifier.StopSet(ticker, level, reason, !res.Placed))
	return entry
}
