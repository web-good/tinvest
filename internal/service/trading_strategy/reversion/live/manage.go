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
			switch {
			case !hadState:
				// Ничего не знаем про тикер — не наша забота.
			case listErr != nil:
				// Не можем свериться с биржей, есть ли ещё живая заявка — значит не можем
				// отличить сработавший стоп от ручной продажи с осиротевшей заявкой.
				// Консервативно: alert, стейт не трогаем, повтор на следующем часовом тике.
				s.notify(notifier.Alert(ticker, "позиция исчезла, но GetStopOrders недоступен — не могу подтвердить срабатывание стопа, стейт сохранён"))
			case entry.StopOrderID == "":
				// Нет заявки, за которой нужно присматривать, — просто чистим стейт.
				delete(state, ticker)
				_ = store.Save(state)
			default:
				if _, alive := stopByID[entry.StopOrderID]; alive {
					// Заявка ещё жива на бирже — значит, позицию продали НЕ через наш
					// стоп (например, вручную в приложении брокера). Снимаем осиротевшую
					// заявку, иначе она позже продаст новую позицию по этому тикеру.
					if err := s.stops.Cancel(ctx, entry.StopOrderID); err != nil {
						s.notify(notifier.Alert(ticker, "позиция продана вне раннера, не удалось снять осиротевший стоп: "+err.Error()))
						// Стейт не чистим — ретрай на следующем часовом тике.
					} else {
						s.notify(notifier.Alert(ticker, "позиция продана вне раннера, снял осиротевший стоп"))
						delete(state, ticker)
						_ = store.Save(state)
					}
				} else {
					// Заявки нет в живых: либо сработал наш биржевой стоп, либо её сняли
					// вне раннера вместе с продажей позиции. Различаем по EXECUTED-списку;
					// при его недоступности считаем срабатыванием (как раньше) — на PnL
					// это не влияет, вопрос только в тексте уведомления.
					if fired, ferr := s.stops.Executed(ctx, entry.StopOrderID); ferr == nil && !fired {
						s.notify(notifier.Alert(ticker, "позиция закрыта и стоп-заявка снята вне раннера — чищу стейт"))
					} else {
						s.notify(notifier.Exit(ticker, entry.StopReason, entry.StopPrice, entry.Quantity, false))
					}
					delete(state, ticker)
					_ = store.Save(state)
				}
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

		// Позиция усохла (частичный стоп или ручная продажа) — реконсилируем количество
		// независимо от того, числится ли заявка в стейте: replaceStop сайзит от
		// entry.Quantity, и протухшее значение дало бы переразмеренный SELL-стоп.
		// Bookkeeping-only — StopOrderID НЕ трогаем и не отменяем здесь: снапшот
		// stopByID/stopByInstrument взят один раз на пасс, повторный Cancel той же заявки
		// ниже привёл бы к двойному cancel в одном тике (ложный alert). Реконсиляция
		// размера живой заявки выполняется общим путём ниже (size-mismatch case).
		if pos.Quantity < entry.Quantity {
			entry.Quantity = pos.Quantity
			state[ticker] = entry
			_ = store.Save(state)
			s.notify(notifier.Alert(ticker, fmt.Sprintf("позиция уменьшилась частично (стоп или ручная продажа), осталось %d", pos.Quantity)))
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
			// Guard ДО снятия стопа: без лота продать нельзя, а уже снятая заявка
			// оставила бы позицию без биржевой защиты навсегда (replaceStop с тем же
			// guard'ом её не вернёт).
			if sh.Lot <= 0 {
				s.notify(notifier.Alert(ticker, "sh.Lot == 0 — невозможно вычислить лоты для продажи, пропуск"))
				logger.ErrorContext(ctx, fmt.Sprintf("reversion: %s sh.Lot=%d, skipping sell to avoid divide-by-zero", ticker, sh.Lot))
				continue
			}

			// Любой SELL ядра: сначала снять биржевой стоп, потом рыночная продажа.
			hadStop := entry.StopOrderID != ""
			if hadStop {
				if err := s.stops.Cancel(ctx, entry.StopOrderID); err != nil {
					s.notify(notifier.Alert(ticker, "не удалось снять стоп-заявку перед продажей: "+err.Error()))
					logger.ErrorContext(ctx, fmt.Sprintf("reversion: %s cancel before sell: %v", ticker, err))
					continue // без снятия продавать нельзя — двойная продажа
				}
				entry.StopOrderID = ""
				state[ticker] = entry
				_ = store.Save(state)
			}

			lots := pos.Quantity / int64(sh.Lot)
			res, err := s.exec.Sell(ctx, sh.ID, lots)
			if err != nil {
				s.notify(notifier.Alert(ticker, "ордер на продажу отклонён: "+err.Error()))
				logger.ErrorContext(ctx, fmt.Sprintf("reversion: %s sell rejected: %v", ticker, err))
				// Стоп уже снят, а продажа не прошла — позиция «голая» до следующего
				// тика. Возвращаем биржевую защиту на прежнем уровне (дубль StopSet
				// подавит changed-флаг replaceStop: уровень/причина не менялись).
				if hadStop && entry.StopReason != "" {
					entry = s.replaceStop(ctx, ticker, sh, entry, entry.StopPrice, entry.StopReason)
					state[ticker] = entry
					_ = store.Save(state)
					if entry.StopOrderID != "" {
						s.notify(notifier.Alert(ticker, "стоп-заявка перевыставлена после отклонённой продажи"))
					}
				}
				continue // retried next tick
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

		p := mustParams(ticker)
		if p.UseIntrabarStop != 1 {
			// Close-модель: биржевой стоп не ведём. Снять заявку, оставшуюся после
			// переключения модели; мёртвую (не в живых по List) — просто вычистить из
			// стейта. При недоступном List ничего не трогаем — ретрай на следующем тике.
			if entry.StopOrderID != "" && listErr == nil {
				if _, alive := stopByID[entry.StopOrderID]; alive {
					if err := s.stops.Cancel(ctx, entry.StopOrderID); err != nil {
						s.notify(notifier.Alert(ticker, "close-модель: не удалось снять оставшуюся стоп-заявку: "+err.Error()))
						continue // заявка жива — стейт не трогаем, ретрай на следующем тике
					}
					s.notify(notifier.Alert(ticker, "close-модель: снял оставшуюся биржевую стоп-заявку"))
				}
				entry.StopOrderID, entry.StopPrice, entry.StopReason = "", 0, ""
				state[ticker] = entry
				if err := store.Save(state); err != nil {
					return fmt.Errorf("reversion: save stop state %s: %w", ticker, err)
				}
			}
			continue
		}

		// Синхронизация стоп-заявки (только при работающем List).
		strayCancelFailed := false
		if listErr == nil {
			if entry.StopOrderID != "" {
				if _, alive := stopByID[entry.StopOrderID]; !alive {
					// Заявки нет в ACTIVE: сработала или снята вне раннера. Различить
					// обязательно — репост свежего стопа на уже проданную стопом позицию
					// (портфель может отставать от расчётов) обернулся бы фантомной
					// продажей/шортом при касании уровня.
					fired, ferr := s.stops.Executed(ctx, entry.StopOrderID)
					switch {
					case ferr != nil:
						// Не репостим вслепую — ретрай на следующем часовом тике.
						s.notify(notifier.Alert(ticker, "стоп-заявка исчезла из ACTIVE, но EXECUTED недоступен — репост отложен: "+ferr.Error()))
						continue
					case fired:
						s.notify(notifier.Exit(ticker, entry.StopReason, entry.StopPrice, entry.Quantity, false))
						delete(state, ticker)
						_ = store.Save(state)
						continue
					default:
						s.notify(notifier.Alert(ticker, "стоп-заявка снята вне раннера — перевыставляю"))
						entry.StopOrderID = ""
					}
				}
			} else if stray, ok := stopByInstrument[sh.ID]; ok {
				// Чужая/устаревшая заявка (например, после reconstruct) — снять.
				if err := s.stops.Cancel(ctx, stray.StopOrderID); err != nil {
					s.notify(notifier.Alert(ticker, "не удалось снять неизвестную стоп-заявку: "+err.Error()))
					// Не ставим новую заявку в этом тике: stray-заявка всё ещё жива на
					// бирже и продолжает защищать позицию (см. guard ниже). Без этого
					// флага на бирже оказались бы ДВЕ живые SELL-заявки на один
					// инструмент — вторая продала бы уже не имеющиеся бумаги.
					strayCancelFailed = true
				}
			}
		}

		// Желаемый уровень от ОБНОВЛЁННОГО MaxFav — на гранулярности шага цены биржи:
		// сырой уровень может расти на доли шага каждый час, а биржевая цена после
		// округления не меняется; сравнение сырых значений гоняло бы cancel+repost
		// по той же цене с окном без защиты на каждом тике.
		level, reason := core.DesiredStop(p, entry.EntryPrice, entry.EntryATR, entry.MaxFav)
		desired := stoporders.RoundDownToIncrement(level, sh.MinPriceIncrement)

		// Текущий уровень и размер — от биржевого снапшота (источник истины);
		// локальный entry.StopPrice — только fallback, когда заявки нет в снапшоте
		// (в т.ч. при listErr != nil: stopByID пуст, alive всегда false — sizeMismatch
		// не форсит слепой cancel+repost, мы не знаем реального размера заявки).
		current := entry.StopPrice
		sizeMismatch := false
		if entry.StopOrderID != "" {
			if live, alive := stopByID[entry.StopOrderID]; alive {
				current = live.StopPrice
				// Оверсайз опаснее не подтянутого трейла, поэтому размер проверяется
				// до сравнения уровня и форсирует cancel+repost даже при равном уровне.
				if sh.Lot > 0 && live.Lots != entry.Quantity/int64(sh.Lot) {
					sizeMismatch = true
				}
			}
		}

		switch {
		case reason == "":
			// ценовые стопы выключены параметрами — нечего вести
		case entry.StopOrderID == "":
			if strayCancelFailed {
				// Stray-заявка не снялась и всё ещё жива на бирже — она продолжает
				// защищать позицию. НЕ ставим вторую: alert уже ушёл выше, ретрай
				// снятия на следующем часовом тике.
				break
			}
			entry = s.replaceStop(ctx, ticker, sh, entry, level, reason)
		case sizeMismatch, desired > current:
			if err := s.stops.Cancel(ctx, entry.StopOrderID); err != nil {
				s.notify(notifier.Alert(ticker, "не удалось снять стоп для переноса: "+err.Error()))
				break // старая заявка продолжает защищать
			}
			entry.StopOrderID = ""
			entry = s.replaceStop(ctx, ticker, sh, entry, level, reason)
		}
		if prev := state[ticker]; prev != entry {
			state[ticker] = entry
			if err := store.Save(state); err != nil {
				return fmt.Errorf("reversion: save stop state %s: %w", ticker, err)
			}
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
// placed; price/reason always). StopPrice is stamped ROUNDED to the instrument's
// price increment, so the state mirrors the exchange-side order (dry-run included).
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
	rounded := stoporders.RoundDownToIncrement(level, sh.MinPriceIncrement)
	changed := rounded != entry.StopPrice || reason != entry.StopReason
	entry.StopPrice, entry.StopReason = rounded, reason
	if changed {
		s.notify(notifier.StopSet(ticker, rounded, reason, !res.Placed))
	}
	return entry
}
