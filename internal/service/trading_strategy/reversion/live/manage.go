package live

import (
	"context"
	"fmt"

	"tinvest/internal/service/trading_strategy/reversion/live/marketdata"
	"tinvest/internal/service/trading_strategy/reversion/live/notifier"
	"tinvest/internal/service/trading_strategy/reversion/live/reconstruct"
	"tinvest/internal/service/trading_strategy/scalping/model"
	"tinvest/internal/service/trading_strategy/scalping/strategy"
	"tinvest/internal/utils"
	"tinvest/pkg/logger"
)

func (s *service) managePass(ctx context.Context) error {
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
			continue
		}
		sh, ok := shares[ticker]
		if !ok || !sh.Trading {
			continue
		}
		pos, isHeld := held[sh.ID]
		if !isHeld {
			// Position gone (e.g. sold elsewhere) — drop any stale state entry.
			if _, ok := state[ticker]; ok {
				delete(state, ticker)
				_ = store.Save(state)
			}
			continue
		}

		entry, hasState := state[ticker]
		if !hasState {
			// Reconstruct from API + alert.
			rebuilt, err := reconstruct.Entry(ctx, s.ops, s.market, s.cfg.AccountID, sh.ID, ticker,
				utils.CombinePrice(pos.PurchasePrice.Units, pos.PurchasePrice.Nano),
				atrPeriodFor(ticker), st.Lookback(), now)
			if err != nil {
				s.notify(notifier.Alert(ticker, "позиция без локального стейта, реконструкция не удалась: "+err.Error()))
				logger.ErrorContext(ctx, fmt.Sprintf("reversion: reconstruct %s: %v", ticker, err))
				continue
			}
			rebuilt.Quantity = pos.Quantity
			entry = rebuilt
			state[ticker] = entry
			_ = store.Save(state)
			s.notify(notifier.Alert(ticker, fmt.Sprintf("стейт восстановлен из API: вход %.4f, ATR %.4f", entry.EntryPrice, entry.EntryATR)))
		}

		md, err := marketdata.Assemble(ctx, s.market, sh.ID, st.Lookback(), MaxHTFTrendEMA([]string{ticker}), now)
		if err != nil {
			logger.ErrorContext(ctx, fmt.Sprintf("reversion: %s marketdata: %v", ticker, err))
			continue
		}

		// Raise maxFav from the latest completed close, then persist (monotonic).
		if md.Price > entry.MaxFav {
			entry.MaxFav = md.Price
			state[ticker] = entry
			if err := store.Save(state); err != nil {
				return fmt.Errorf("reversion: save maxFav %s: %w", ticker, err)
			}
		}

		md.Position = &strategy.Position{
			PurchasePrice:     entry.EntryPrice,
			Quantity:          pos.Quantity,
			EntryATR:          entry.EntryATR,
			MaxFavorablePrice: entry.MaxFav,
		}

		sig := st.Decide(md)
		if sig.Kind != model.SignalSell {
			continue
		}

		lots := pos.Quantity / int64(sh.Lot)
		res, err := s.exec.Sell(ctx, sh.ID, lots)
		if err != nil {
			s.notify(notifier.Alert(ticker, "ордер на продажу отклонён: "+err.Error()))
			logger.ErrorContext(ctx, fmt.Sprintf("reversion: %s sell rejected: %v", ticker, err))
			continue // state unchanged; retried next tick
		}

		exitPrice := sig.Price
		if res.Placed && res.FillPrice > 0 {
			exitPrice = res.FillPrice
		}
		delete(state, ticker)
		if err := store.Save(state); err != nil {
			return fmt.Errorf("reversion: save state after sell %s: %w", ticker, err)
		}
		s.notify(notifier.Exit(ticker, sig.Reason, exitPrice, pos.Quantity, !res.Placed))
	}
	return nil
}

// atrPeriodFor returns the ticker's ATRPeriod for reconstruct's ATR recomputation.
func atrPeriodFor(ticker string) int {
	if p, ok := paramsByTicker[ticker]; ok && p.ATRPeriod > 0 {
		return p.ATRPeriod
	}
	return 14
}
