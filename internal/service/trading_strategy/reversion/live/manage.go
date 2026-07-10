package live

import (
	"context"
	"fmt"

	imodel "tinvest/internal/model"
	"tinvest/internal/service/trading_strategy/reversion/live/marketdata"
	"tinvest/internal/service/trading_strategy/reversion/live/notifier"
	"tinvest/internal/service/trading_strategy/reversion/live/reconstruct"
	"tinvest/internal/service/trading_strategy/reversion/live/statestore"
	"tinvest/internal/service/trading_strategy/reversion/live/stoporders"
	"tinvest/internal/service/trading_strategy/reversion/strategy/core"
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

	activeStops, listErr := s.stops.List(ctx) // один вызов на весь пасс
	if listErr != nil {
		s.notify(notifier.Alert("reversion", "GetStopOrders недоступен: "+listErr.Error()))
	}
	stopByInstrument := map[string]stoporders.ActiveStop{}
	stopByID := map[string]stoporders.ActiveStop{}
	for _, a := range activeStops {
		stopByInstrument[a.InstrumentUID] = a
		stopByID[a.StopOrderID] = a
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
			entry, hadState := state[ticker]
			if hadState && entry.StopOrderID != "" {
				// Наш биржевой стоп исполнился.
				s.notify(notifier.Exit(ticker, entry.StopReason, entry.StopPrice, entry.Quantity, false))
			}
			if hadState {
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

		// Частичное исполнение: биржа продала часть позиции. Bookkeeping-only — StopOrderID
		// НЕ трогаем и не отменяем здесь: снапшот stopByID/stopByInstrument взят один раз
		// на пасс, повторный Cancel той же заявки ниже привёл бы к двойному cancel в одном
		// тике (ложный alert). Реконсиляция размера заявки на изменившееся entry.Quantity
		// выполняется общим путём ниже, в уровень-switch (size-mismatch case).
		if pos.Quantity < entry.Quantity && entry.StopOrderID != "" {
			entry.Quantity = pos.Quantity
			state[ticker] = entry
			_ = store.Save(state)
			s.notify(notifier.Alert(ticker, fmt.Sprintf("стоп исполнился частично, осталось %d", pos.Quantity)))
		}

		md, err := marketdata.Assemble(ctx, s.market, sh.ID, st.Lookback(), MaxHTFTrendEMA([]string{ticker}), now)
		if err != nil {
			logger.ErrorContext(ctx, fmt.Sprintf("reversion: %s marketdata: %v", ticker, err))
			continue
		}

		prevMaxFav := entry.MaxFav // уровень, от которого считалась стоящая на бирже заявка
		// Raise maxFav from the latest completed close, then persist (monotonic).
		if md.Price > entry.MaxFav {
			entry.MaxFav = md.Price
			state[ticker] = entry
			if err := store.Save(state); err != nil {
				return fmt.Errorf("reversion: save maxFav %s: %w", ticker, err)
			}
		}

		md.Position = &strategy.Position{
			PurchasePrice:         entry.EntryPrice,
			Quantity:              pos.Quantity,
			EntryATR:              entry.EntryATR,
			MaxFavorablePrice:     entry.MaxFav,
			PrevMaxFavorablePrice: prevMaxFav,
		}

		sig := st.Decide(md)
		if sig.Kind == model.SignalSell {
			// Любой SELL ядра: сначала снять биржевой стоп, потом рыночная продажа.
			if entry.StopOrderID != "" {
				if err := s.stops.Cancel(ctx, entry.StopOrderID); err != nil {
					s.notify(notifier.Alert(ticker, "не удалось снять стоп-заявку перед продажей: "+err.Error()))
					logger.ErrorContext(ctx, fmt.Sprintf("reversion: %s cancel before sell: %v", ticker, err))
					continue // без снятия продавать нельзя — двойная продажа
				}
				entry.StopOrderID = ""
				state[ticker] = entry
				_ = store.Save(state)
			}

			if sh.Lot <= 0 {
				s.notify(notifier.Alert(ticker, "sh.Lot == 0 — невозможно вычислить лоты для продажи, пропуск"))
				logger.ErrorContext(ctx, fmt.Sprintf("reversion: %s sh.Lot=%d, skipping sell to avoid divide-by-zero", ticker, sh.Lot))
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
			continue
		}

		// Синхронизация стоп-заявки (только при работающем List).
		if listErr == nil {
			if entry.StopOrderID != "" {
				if _, alive := stopByID[entry.StopOrderID]; !alive {
					s.notify(notifier.Alert(ticker, "стоп-заявка исчезла с биржи — перевыставляю"))
					entry.StopOrderID = ""
				}
			} else if stray, ok := stopByInstrument[sh.ID]; ok {
				// Чужая/устаревшая заявка (например, после reconstruct) — снять.
				if err := s.stops.Cancel(ctx, stray.StopOrderID); err != nil {
					s.notify(notifier.Alert(ticker, "не удалось снять неизвестную стоп-заявку: "+err.Error()))
				}
			}
		}

		// Желаемый уровень от ОБНОВЛЁННОГО MaxFav.
		level, reason := core.DesiredStop(mustParams(ticker), entry.EntryPrice, entry.EntryATR, entry.MaxFav)

		// Реконсиляция размера: живая заявка на бирже держит не тот объём, что реально в
		// позиции (например, после частичного исполнения выше). Оверсайз опаснее
		// не подтянутого трейла, поэтому проверяется до сравнения уровня и форсирует
		// cancel+repost даже если level == StopPrice.
		sizeMismatch := false
		if sh.Lot > 0 && entry.StopOrderID != "" {
			if live, alive := stopByID[entry.StopOrderID]; alive && live.Lots != entry.Quantity/int64(sh.Lot) {
				sizeMismatch = true
			}
		}

		switch {
		case reason == "":
			// ценовые стопы выключены параметрами — нечего вести
		case entry.StopOrderID == "":
			entry = s.replaceStop(ctx, ticker, sh, entry, level, reason)
		case sizeMismatch, level > entry.StopPrice:
			if err := s.stops.Cancel(ctx, entry.StopOrderID); err != nil {
				s.notify(notifier.Alert(ticker, "не удалось снять стоп для переноса: "+err.Error()))
				break // старая заявка продолжает защищать
			}
			entry.StopOrderID = ""
			entry = s.replaceStop(ctx, ticker, sh, entry, level, reason)
		}
		state[ticker] = entry
		if err := store.Save(state); err != nil {
			return fmt.Errorf("reversion: save stop state %s: %w", ticker, err)
		}
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

// mustParams: ParamsFor гарантированно ok — тикер прошёл StrategyFor выше.
func mustParams(ticker string) core.Params {
	p, _ := ParamsFor(ticker)
	return p
}

// replaceStop places a stop at level and stamps the entry (id only when actually
// placed; price/reason always, so dry-run state mirrors what WOULD be on exchange).
func (s *service) replaceStop(ctx context.Context, ticker string, sh *imodel.Share,
	entry statestore.Entry, level float64, reason string) statestore.Entry {

	if sh.Lot <= 0 {
		s.notify(notifier.Alert(ticker, "sh.Lot == 0 — невозможно вычислить лоты для стоп-заявки, пропуск"))
		logger.ErrorContext(ctx, fmt.Sprintf("reversion: %s sh.Lot=%d, skipping stop placement to avoid divide-by-zero", ticker, sh.Lot))
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
	changed := level != entry.StopPrice || reason != entry.StopReason
	entry.StopPrice, entry.StopReason = level, reason
	if changed {
		s.notify(notifier.StopSet(ticker, level, reason, !res.Placed))
	}
	return entry
}
