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

// manage carries an open position: the core's exits (SL/TRAIL/TP/RSI) and the exchange
// stop order that mirrors the protective level. The level is computed by core.DesiredStop —
// the same function the backtest exits on, so prod and backtest cannot drift apart on the
// most frequent exit mechanism.
func (s *service) manage(ctx context.Context, pc *passCtx, ticker string, sh *imodel.Share,
	st *core.Strategy, md strategy.MarketData, pos *grpcmodel.Position, isHeld bool) error {

	if !isHeld {
		s.settleGonePosition(ctx, pc, ticker)
		return nil
	}

	entry, hasState := pc.state[ticker]
	if !hasState {
		// Восстановления стейта по API тут ещё нет: без EntryATR и замороженной цели ядро
		// не может ни защитить позицию, ни закрыть её, а угаданные величины хуже
		// отсутствующих — они молча уводят уровень стопа от того, что задала стратегия.
		s.notify(notifier.Alert(alertLabel, ticker, "позиция без локального стейта — сопровождение невозможно, пропуск"))
		logger.ErrorContext(ctx, fmt.Sprintf("rsi_pullback: %s held without state, skip", ticker))
		return nil
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
		pc.state[ticker] = entry
		_ = pc.store.Save(pc.state)
		s.notify(notifier.Alert(alertLabel, ticker, fmt.Sprintf("позиция уменьшилась частично (стоп или ручная продажа), осталось %d", pos.Quantity)))
	}

	prevMaxFav := entry.MaxFav // уровень, от которого считалась стоящая на бирже заявка
	// Raise maxFav from the latest completed close, then persist (monotonic).
	if md.Price > entry.MaxFav {
		entry.MaxFav = md.Price
		pc.state[ticker] = entry
		if err := pc.store.Save(pc.state); err != nil {
			return fmt.Errorf("rsi_pullback: save maxFav %s: %w", ticker, err)
		}
	}

	md.Position = &strategy.Position{
		PurchasePrice:         entry.EntryPrice,
		Quantity:              pos.Quantity,
		EntryATR:              entry.EntryATR,
		TakeProfit:            entry.TakeProfit,
		MaxFavorablePrice:     entry.MaxFav,
		PrevMaxFavorablePrice: prevMaxFav,
	}

	sig := st.Decide(md)
	if sig.Kind == model.SignalSell {
		return s.sell(ctx, pc, ticker, sh, entry, pos, sig)
	}

	// Синхронизация стоп-заявки (только при работающем List).
	strayCancelFailed := false
	if pc.listErr == nil {
		if entry.StopOrderID != "" {
			if _, alive := pc.stopByID[entry.StopOrderID]; !alive {
				// Заявки нет в ACTIVE: сработала или снята вне раннера. Различить
				// обязательно — репост свежего стопа на уже проданную стопом позицию
				// (портфель может отставать от расчётов) обернулся бы фантомной
				// продажей/шортом при касании уровня.
				fired, ferr := s.stops.Executed(ctx, entry.StopOrderID)
				switch {
				case ferr != nil:
					// Не репостим вслепую — ретрай на следующем получасовом тике.
					s.notify(notifier.Alert(alertLabel, ticker, "стоп-заявка исчезла из ACTIVE, но EXECUTED недоступен — репост отложен: "+ferr.Error()))
					return nil
				case fired:
					s.notify(notifier.Exit(ticker, entry.StopReason, entry.StopPrice, entry.Quantity, false))
					delete(pc.state, ticker)
					_ = pc.store.Save(pc.state)
					return nil
				default:
					s.notify(notifier.Alert(alertLabel, ticker, "стоп-заявка снята вне раннера — перевыставляю"))
					entry.StopOrderID = ""
				}
			}
		} else if stray, ok := pc.stopByInstrument[sh.ID]; ok {
			// Чужая/устаревшая заявка (например, после восстановления стейта) — снять.
			if err := s.stops.Cancel(ctx, stray.StopOrderID); err != nil {
				s.notify(notifier.Alert(alertLabel, ticker, "не удалось снять неизвестную стоп-заявку: "+err.Error()))
				// Не ставим новую заявку в этом тике: stray-заявка всё ещё жива на
				// бирже и продолжает защищать позицию (см. guard ниже). Без этого
				// флага на бирже оказались бы ДВЕ живые SELL-заявки на один
				// инструмент — вторая продала бы уже не имеющиеся бумаги.
				strayCancelFailed = true
			}
		}
	}

	// Желаемый уровень от ОБНОВЛЁННОГО MaxFav — на гранулярности шага цены биржи:
	// сырой уровень может расти на доли шага каждые полчаса, а биржевая цена после
	// округления не меняется; сравнение сырых значений гоняло бы cancel+repost
	// по той же цене с окном без защиты на каждом тике.
	level, reason := core.DesiredStop(mustParams(ticker), entry.EntryPrice, entry.EntryATR, entry.MaxFav)
	desired := stoporders.RoundDownToIncrement(level, sh.MinPriceIncrement)

	// Текущий уровень и размер — от биржевого снапшота (источник истины);
	// локальный entry.StopPrice — только fallback, когда заявки нет в снапшоте
	// (в т.ч. при listErr != nil: stopByID пуст, alive всегда false — sizeMismatch
	// не форсит слепой cancel+repost, мы не знаем реального размера заявки).
	current := entry.StopPrice
	sizeMismatch := false
	if entry.StopOrderID != "" {
		if live, alive := pc.stopByID[entry.StopOrderID]; alive {
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
			// снятия на следующем получасовом тике.
			break
		}
		entry = s.replaceStop(ctx, ticker, sh, entry, level, reason)
	case sizeMismatch, desired > current:
		if err := s.stops.Cancel(ctx, entry.StopOrderID); err != nil {
			s.notify(notifier.Alert(alertLabel, ticker, "не удалось снять стоп для переноса: "+err.Error()))
			break // старая заявка продолжает защищать
		}
		entry.StopOrderID = ""
		entry = s.replaceStop(ctx, ticker, sh, entry, level, reason)
	}
	if prev := pc.state[ticker]; prev != entry {
		pc.state[ticker] = entry
		if err := pc.store.Save(pc.state); err != nil {
			return fmt.Errorf("rsi_pullback: save stop state %s: %w", ticker, err)
		}
	}
	return nil
}

// sell closes the position on any SELL the core emits: the exchange stop comes off first,
// then the market order. A stop left standing would later sell a NEW position in the same
// ticker; a sell that goes through without it would double-sell this one.
func (s *service) sell(ctx context.Context, pc *passCtx, ticker string, sh *imodel.Share,
	entry statestore.Entry, pos *grpcmodel.Position, sig model.Signal) error {

	// Guard ДО снятия стопа: без лота продать нельзя, а уже снятая заявка
	// оставила бы позицию без биржевой защиты навсегда (replaceStop с тем же
	// guard'ом её не вернёт).
	if sh.Lot <= 0 {
		s.notify(notifier.Alert(alertLabel, ticker, "sh.Lot == 0 — невозможно вычислить лоты для продажи, пропуск"))
		logger.ErrorContext(ctx, fmt.Sprintf("rsi_pullback: %s sh.Lot=%d, skipping sell to avoid divide-by-zero", ticker, sh.Lot))
		return nil
	}

	hadStop := entry.StopOrderID != ""
	if hadStop {
		if err := s.stops.Cancel(ctx, entry.StopOrderID); err != nil {
			s.notify(notifier.Alert(alertLabel, ticker, "не удалось снять стоп-заявку перед продажей: "+err.Error()))
			logger.ErrorContext(ctx, fmt.Sprintf("rsi_pullback: %s cancel before sell: %v", ticker, err))
			return nil // без снятия продавать нельзя — двойная продажа
		}
		entry.StopOrderID = ""
		pc.state[ticker] = entry
		_ = pc.store.Save(pc.state)
	}

	lots := pos.Quantity / int64(sh.Lot)
	res, err := s.exec.Sell(ctx, sh.ID, lots)
	if err != nil {
		s.notify(notifier.Alert(alertLabel, ticker, "ордер на продажу отклонён: "+err.Error()))
		logger.ErrorContext(ctx, fmt.Sprintf("rsi_pullback: %s sell rejected: %v", ticker, err))
		// Стоп уже снят, а продажа не прошла — позиция «голая» до следующего
		// тика. Возвращаем биржевую защиту на прежнем уровне (дубль StopSet
		// подавит changed-флаг replaceStop: уровень/причина не менялись).
		if hadStop && entry.StopReason != "" {
			entry = s.replaceStop(ctx, ticker, sh, entry, entry.StopPrice, entry.StopReason)
			pc.state[ticker] = entry
			_ = pc.store.Save(pc.state)
			if entry.StopOrderID != "" {
				s.notify(notifier.Alert(alertLabel, ticker, "стоп-заявка перевыставлена после отклонённой продажи"))
			}
		}
		return nil // retried next tick
	}

	exitPrice := sig.Price
	if res.Placed && res.FillPrice > 0 {
		exitPrice = res.FillPrice
	}
	delete(pc.state, ticker)
	if err := pc.store.Save(pc.state); err != nil {
		return fmt.Errorf("rsi_pullback: save state after sell %s: %w", ticker, err)
	}
	s.notify(notifier.Exit(ticker, sig.Reason, exitPrice, pos.Quantity, !res.Placed))
	return nil
}

// settleGonePosition reconciles a ticker the broker no longer holds: either our stop fired,
// or the position was sold outside the runner and the stop is orphaned. Telling the two
// apart matters — an orphaned stop left on the exchange would later sell a new position in
// the same ticker, and a false "stop fired" notification would misreport the exit.
func (s *service) settleGonePosition(ctx context.Context, pc *passCtx, ticker string) {
	entry, hadState := pc.state[ticker]
	switch {
	case !hadState:
		// Ничего не знаем про тикер — не наша забота.
	case pc.listErr != nil:
		// Не можем свериться с биржей, есть ли ещё живая заявка — значит не можем
		// отличить сработавший стоп от ручной продажи с осиротевшей заявкой.
		// Консервативно: alert, стейт не трогаем, повтор на следующем тике.
		s.notify(notifier.Alert(alertLabel, ticker, "позиция исчезла, но GetStopOrders недоступен — не могу подтвердить срабатывание стопа, стейт сохранён"))
	case entry.StopOrderID == "":
		// Нет заявки, за которой нужно присматривать, — просто чистим стейт.
		delete(pc.state, ticker)
		_ = pc.store.Save(pc.state)
	default:
		if _, alive := pc.stopByID[entry.StopOrderID]; alive {
			// Заявка ещё жива на бирже — значит, позицию продали НЕ через наш
			// стоп (например, вручную в приложении брокера). Снимаем осиротевшую
			// заявку, иначе она позже продаст новую позицию по этому тикеру.
			if err := s.stops.Cancel(ctx, entry.StopOrderID); err != nil {
				s.notify(notifier.Alert(alertLabel, ticker, "позиция продана вне раннера, не удалось снять осиротевший стоп: "+err.Error()))
				return // стейт не чистим — ретрай на следующем тике
			}
			s.notify(notifier.Alert(alertLabel, ticker, "позиция продана вне раннера, снял осиротевший стоп"))
			delete(pc.state, ticker)
			_ = pc.store.Save(pc.state)
			return
		}
		// Заявки нет в живых: либо сработал наш биржевой стоп, либо её сняли
		// вне раннера вместе с продажей позиции. Различаем по EXECUTED-списку;
		// при его недоступности считаем срабатыванием — на PnL это не влияет,
		// вопрос только в тексте уведомления.
		if fired, ferr := s.stops.Executed(ctx, entry.StopOrderID); ferr == nil && !fired {
			s.notify(notifier.Alert(alertLabel, ticker, "позиция закрыта и стоп-заявка снята вне раннера — чищу стейт"))
		} else {
			s.notify(notifier.Exit(ticker, entry.StopReason, entry.StopPrice, entry.Quantity, false))
		}
		delete(pc.state, ticker)
		_ = pc.store.Save(pc.state)
	}
}
