package live

import (
	"context"
	"fmt"
	"slices"
	"sort"
	"strings"
	"time"

	imodel "tinvest/internal/model"
	"tinvest/internal/service/trading_strategy/livecore/adapter"
	"tinvest/internal/service/trading_strategy/livecore/notifier"
	"tinvest/internal/service/trading_strategy/livecore/sizing"
	"tinvest/internal/service/trading_strategy/livecore/statestore"
	"tinvest/internal/service/trading_strategy/livecore/stoporders"
	"tinvest/internal/service/trading_strategy/rsi_pullback/live/marketdata"
	"tinvest/internal/service/trading_strategy/scalping/model"
	"tinvest/internal/service/trading_strategy/scalping/strategy"
	"tinvest/internal/utils"
	grpcmodel "tinvest/pkg/client/grpc/model"
	"tinvest/pkg/logger"
)

// maxBarAge is how stale the latest completed 30-minute bar may be before the runner
// refuses to act on it. The cron fires every half hour, so anything older than an hour
// means the feed is behind (holiday, halt, API outage) — and a decision taken on a stale
// bar is a decision taken on the wrong price.
const maxBarAge = 60 * time.Minute

// freshEntryGrace is how long after an entry the position's absence from the broker's
// portfolio is NOT taken as a sale. Settlement lags the filled order, so the pass right
// after an entry can still see an empty portfolio — and reading that as "sold outside the
// runner" would cancel the live protective stop and wipe the state of a position that is
// actually open. Two staleness windows (2*maxBarAge = 2 hours, i.e. the entry pass plus
// room for a skipped tick and a lagging feed) is the grace; beyond that a settlement lag is
// no longer plausible, and staying silent about a vanished position becomes the more
// dangerous of the two errors. The cost of the wide window is a delayed notification of a
// real stop-out in the first two hours — never a missed action: hasState still blocks a
// second entry, and the exchange stop keeps working.
const freshEntryGrace = 2 * maxBarAge

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

	activeStops, listErr := s.accountStops.List(ctx) // один вызов на весь пасс
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
	// Ошибка по одному тикеру не обрывает пасс: остальные позиции иначе остались бы без
	// сопровождения — трейл не подтягивается, выходы не проверяются — и так на каждом
	// пассе, пока держится причина (сбой записи стейта повторится и завтра). Наверх ошибка
	// всё равно уходит, чтобы планировщик её залогировал.
	pc := &passCtx{
		state: state, store: store, stopByID: stopByID,
		stopByInstrument: stopByInstrument, listErr: listErr, now: now,
	}
	var failed []string
	for _, ticker := range s.passTickers(state) {
		entry, hasState := state[ticker]

		// Кто вправе действовать по тикеру. Запись в стейте — владелец из неё; иначе —
		// стратегии, в чьей вселенной тикер, по приоритету.
		var cands []*slot
		if hasState {
			owner := s.slotByName(adapter.Owner(entry))
			if owner == nil {
				s.slots[0].alert(ticker, fmt.Sprintf("позиция принадлежит неизвестной стратегии %q — сопровождение пропущено", adapter.Owner(entry)))
				continue
			}
			cands = []*slot{owner}
		} else {
			cands = s.universeSlots(ticker)
		}
		// Незарегистрированный тикер — алерт от той стратегии, в чьей он вселенной (или
		// чья запись), и пропуск; свечи по нему не запрашиваются.
		var ready []cand
		for _, sl := range cands {
			dec, ok := sl.strat.Decider(ticker)
			if !ok {
				sl.alert(ticker, "тикер не зарегистрирован в "+sl.strat.Name()+" — пропуск")
				continue
			}
			ready = append(ready, cand{sl, dec})
		}
		if len(ready) == 0 {
			continue
		}

		// Пропуск тикера, за которым мы следим (позиция у брокера или запись в стейте), —
		// это пропущенное сопровождение: трейл не подтягивается, выходы не проверяются, а
		// биржевой стоп остаётся на уровне получасовой давности. Строчка в логе контейнера
		// об этом никому не сообщит, поэтому каждый такой пропуск уходит алертом. По тикеру
		// без позиции алерта нет: выходной или закрытая сессия — не повод будить владельца.
		sh, ok := shares[ticker]
		if !ok || !sh.Trading {
			if hasState {
				ready[0].sl.alert(ticker, "инструмент недоступен для торгов — сопровождение пропущено")
			}
			logger.ErrorContext(ctx, fmt.Sprintf("%s: %s not tradable, skip", ready[0].sl.strat.Name(), ticker))
			continue
		}
		pos, isHeld := held[sh.ID]

		if isHeld || hasState {
			// Позиция без стейта на тикере нескольких стратегий: чья она — из API не узнать.
			// Угаданный владелец повёл бы её чужим стопом и чужими выходами; реконструкция
			// и чужие стоп-заявки остаются нетронутыми до ручной разметки.
			if !hasState && len(cands) > 1 {
				s.slots[0].alert(ticker, "позиция без стейта на общем тикере, владелец неизвестен — нужна ручная разметка стейта; сопровождение пропущено")
				continue
			}
			c := ready[0]
			// Реальная позиция без стейта на тикере бумажной стратегии при боевой соседке:
			// бумажный слот восстановил бы стейт, но биржевой стоп не поставил бы (dry-run), а
			// бумажный SELL стёр бы запись, оставив бумаги у брокера, — и так каждые полчаса.
			// Реальные бумаги без защиты хуже пропуска: алерт, стоп-заявки не трогаются.
			if !hasState && s.paperBesideLive(c.sl) {
				s.slots[0].alert(ticker, fmt.Sprintf("позиция без стейта на тикере бумажной стратегии %s — нужна ручная разметка стейта; сопровождение пропущено", c.sl.strat.Name()))
				continue
			}
			md, ok := s.assemble(ctx, c.sl, ticker, sh, c.dec, now, true)
			if !ok {
				continue
			}
			if perr := s.manage(ctx, pc, c.sl, ticker, sh, c.dec, md, pos, isHeld); perr != nil {
				c.sl.alert(ticker, "пасс по тикеру прерван: "+perr.Error())
				logger.ErrorContext(ctx, fmt.Sprintf("%s: %s pass: %v", c.sl.strat.Name(), ticker, perr))
				failed = append(failed, ticker)
			}
			continue
		}

		// Свободный тикер: стратегии по приоритету; первая с BUY входит, остальных на
		// этом баре не спрашивают.
		for _, c := range ready {
			md, ok := s.assemble(ctx, c.sl, ticker, sh, c.dec, now, false)
			if !ok {
				continue
			}
			signaled, perr := s.buy(ctx, pc, c.sl, ticker, sh, c.dec, md)
			if perr != nil {
				c.sl.alert(ticker, "пасс по тикеру прерван: "+perr.Error())
				logger.ErrorContext(ctx, fmt.Sprintf("%s: %s pass: %v", c.sl.strat.Name(), ticker, perr))
				failed = append(failed, ticker)
			}
			if signaled {
				break
			}
		}
	}
	if len(failed) > 0 {
		return fmt.Errorf("rsi_pullback: пасс завершён с ошибками по тикерам: %s", strings.Join(failed, ", "))
	}
	return nil
}

// assemble собирает MarketData окном Lookback стратегии и отбрасывает протухший бар.
// Своя сборка на стратегию, а не срез общей: окно ровно то, что видит движок бэктеста.
// watched — по тикеру есть позиция или запись: только тогда пропуск будит владельца.
func (s *service) assemble(ctx context.Context, sl *slot, ticker string, sh *imodel.Share,
	dec adapter.Decider, now time.Time, watched bool) (strategy.MarketData, bool) {

	md, err := marketdata.Assemble(ctx, s.market, sh.ID, dec.Lookback(), now)
	if err != nil {
		if watched {
			sl.alert(ticker, "рыночные данные недоступны — сопровождение пропущено: "+err.Error())
		}
		logger.ErrorContext(ctx, fmt.Sprintf("%s: %s marketdata: %v", sl.strat.Name(), ticker, err))
		return strategy.MarketData{}, false
	}
	if n := len(md.Times); n == 0 || now.Sub(md.Times[n-1]) > maxBarAge {
		if watched {
			sl.alert(ticker, "последний завершённый бар протух — сопровождение пропущено")
		}
		logger.ErrorContext(ctx, fmt.Sprintf("%s: %s stale bar, skip", sl.strat.Name(), ticker))
		return strategy.MarketData{}, false
	}
	return md, true
}

// cand — стратегия, готовая решать по тикеру: тикер есть в её реестре.
type cand struct {
	sl  *slot
	dec adapter.Decider
}

// paperBesideLive — стратегия бумажная, а на счёте есть боевая. Такая стратегия не вправе
// занимать тикер: ни стейтом бумажного входа, ни реконструкцией реальной позиции.
func (s *service) paperBesideLive(sl *slot) bool {
	return !sl.strat.TradeEnabled() && s.anyLive
}

// passTickers — объединение вселенных в порядке приоритета стратегий, затем тикеры стейта
// вне всех вселенных (отсортированно): владелец ведёт позицию до выхода, даже если тикер
// убрали из его вселенной.
func (s *service) passTickers(state map[string]statestore.Entry) []string {
	seen := map[string]bool{}
	var out []string
	for _, sl := range s.slots {
		for _, t := range sl.strat.Tickers() {
			if !seen[t] {
				seen[t] = true
				out = append(out, t)
			}
		}
	}
	var extra []string
	for t := range state {
		if !seen[t] {
			extra = append(extra, t)
		}
	}
	sort.Strings(extra)
	return append(out, extra...)
}

// universeSlots — стратегии, в чьей вселенной тикер, в порядке приоритета.
func (s *service) universeSlots(ticker string) []*slot {
	var out []*slot
	for _, sl := range s.slots {
		if slices.Contains(sl.strat.Tickers(), ticker) {
			out = append(out, sl)
		}
	}
	return out
}

// buy opens a long when the core signals one: size from the configured percentage of the
// account, market BUY, state (with the frozen ATR and target), and immediately the
// protective exchange stop — the position must never live a single tick unprotected,
// because the next check is half an hour away while the stop is an intrabar one.
func (s *service) buy(ctx context.Context, pc *passCtx, sl *slot, ticker string, sh *imodel.Share,
	dec adapter.Decider, md strategy.MarketData) (signaled bool, err error) {

	md.Position = nil

	sig := dec.Decide(md)
	if sig.Kind != model.SignalBuy {
		return false, nil
	}

	total, err := s.ops.GetPortfolioTotal(ctx, s.cfg.AccountID)
	if err != nil {
		return true, fmt.Errorf("%s: portfolio total: %w", sl.strat.Name(), err)
	}
	cash, err := s.ops.GetAvailableCash(ctx, s.cfg.AccountID)
	if err != nil {
		return true, fmt.Errorf("%s: cash: %w", sl.strat.Name(), err)
	}
	// Бумажная стратегия при боевой соседке бар не занимает (signaled=false): цикл пасса
	// спросит следующую по приоритету, и её боевой вход на этом баре не потеряется. Так
	// бумажный режим ничего не блокирует — ради этого он и не пишет стейт (см. ниже).
	paper := s.paperBesideLive(sl)
	lots, ok, reason := sizing.Lots(sl.strat.BuyPct(), total, cash, sig.Price, sh.Lot)
	if !ok {
		sl.strat.Notify(notifier.Skip(ticker, reason))
		return !paper, nil
	}

	// Бумажный вход при боевой соседке — только уведомление. Запись в стейте заняла бы
	// тикер до чистки по freshEntryGrace и заблокировала бы боевой вход другой стратегии.
	// Когда боевых стратегий на счёте нет вовсе, работает прежний dry-run: стейт пишется.
	if paper {
		sl.strat.Notify(notifier.Entry(ticker, sig.Price, lots, lots*int64(sh.Lot), true))
		return false, nil
	}

	res, err := sl.exec.Buy(ctx, sh.ID, lots)
	if err != nil {
		sl.alert(ticker, "ордер на покупку отклонён: "+err.Error())
		logger.ErrorContext(ctx, fmt.Sprintf("%s: %s buy rejected: %v", sl.strat.Name(), ticker, err))
		return true, nil // state unchanged; retried next tick
	}

	fillPrice := sig.Price
	filledLots := lots
	if res.Placed {
		// Брокер принял ордер, но исполнил ноль лотов (снят биржей после приёма, закрытая
		// сессия, нет встречной ликвидности). Позиции нет — записать стейт на ЗАПРОШЕННЫЙ
		// объём значило бы повести несуществующую позицию и тут же выставить SELL-стоп на
		// бумаги, которых нет. Ничего не пишем: если бумаги всё же появятся, следующий пасс
		// увидит позицию без стейта и восстановит её через reconstruct.
		if res.FilledLots == 0 {
			sl.alert(ticker, "ордер принят, но не исполнен (0 лотов) — стейт не создан")
			logger.ErrorContext(ctx, fmt.Sprintf("%s: %s buy accepted with zero fill", sl.strat.Name(), ticker))
			return true, nil
		}
		if res.FillPrice > 0 {
			fillPrice = res.FillPrice
		}
		filledLots = res.FilledLots
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
		Strategy:   sl.strat.Name(),
	}
	if err := pc.store.Save(pc.state); err != nil {
		return true, fmt.Errorf("%s: save state after buy %s: %w", sl.strat.Name(), ticker, err)
	}
	sl.strat.Notify(notifier.Entry(ticker, fillPrice, filledLots, qty, !res.Placed))

	pc.state[ticker] = s.placeInitialStop(ctx, pc, sl, ticker, sh, pc.state[ticker])
	return true, nil
}

// placeInitialStop puts the protective exchange stop right after a fill so the position is
// never unprotected for the first half hour. There is no UseIntrabarStop switch here: the
// stop of this strategy is intrabar by definition, so it is always placed. Placement and
// stamping are delegated to replaceStop (same guard/rounding/notification path as manage);
// for a fresh entry StopPrice is 0, so the StopSet notification always fires. On failure the
// entry keeps an empty StopOrderID and the next pass retries.
func (s *service) placeInitialStop(ctx context.Context, pc *passCtx, sl *slot, ticker string,
	sh *imodel.Share, entry statestore.Entry) statestore.Entry {

	level, reason := sl.strat.DesiredStop(ticker, entry)
	if reason == "" {
		return entry
	}
	entry = s.replaceStop(ctx, sl, ticker, sh, entry, level, reason)
	pc.state[ticker] = entry
	_ = pc.store.Save(pc.state)
	return entry
}

// replaceStop places a stop at level and stamps the entry (id only when actually placed;
// price/reason always). StopPrice is stamped ROUNDED to the instrument's price increment,
// so the state mirrors the exchange-side order (dry-run included).
func (s *service) replaceStop(ctx context.Context, sl *slot, ticker string, sh *imodel.Share,
	entry statestore.Entry, level float64, reason string) statestore.Entry {

	if sh.Lot <= 0 {
		sl.alert(ticker, "sh.Lot == 0 — невозможно вычислить лоты для стоп-заявки, пропуск")
		logger.ErrorContext(ctx, fmt.Sprintf("%s: %s sh.Lot=%d, skipping stop placement to avoid divide-by-zero", sl.strat.Name(), ticker, sh.Lot))
		return entry
	}
	lots := entry.Quantity / int64(sh.Lot)
	res, err := sl.stops.Place(ctx, sh.ID, lots, level, sh.MinPriceIncrement)
	if err != nil {
		sl.alert(ticker, "стоп-заявка не выставлена: "+err.Error())
		logger.ErrorContext(ctx, fmt.Sprintf("%s: %s place stop: %v", sl.strat.Name(), ticker, err))
		return entry
	}
	if res.Placed {
		entry.StopOrderID = res.OrderID
	}
	rounded := stoporders.RoundDownToIncrement(level, sh.MinPriceIncrement)
	changed := rounded != entry.StopPrice || reason != entry.StopReason
	entry.StopPrice, entry.StopReason = rounded, reason
	if changed {
		sl.strat.Notify(notifier.StopSet(ticker, rounded, reason, !res.Placed))
	}
	return entry
}

// manage carries an open position: the core's exits (SL/TRAIL/TP/RSI) and the exchange
// stop order that mirrors the protective level. The level is computed by core.DesiredStop —
// the same function the backtest exits on, so prod and backtest cannot drift apart on the
// most frequent exit mechanism.
func (s *service) manage(ctx context.Context, pc *passCtx, sl *slot, ticker string, sh *imodel.Share,
	dec adapter.Decider, md strategy.MarketData, pos *grpcmodel.Position, isHeld bool) error {

	if !isHeld {
		s.settleGonePosition(ctx, pc, sl, ticker)
		return nil
	}

	entry, hasState := pc.state[ticker]
	if !hasState {
		// Позиция есть у брокера, локального стейта нет (переезд контейнера, потерянный
		// том). Восстанавливаем по API: без EntryATR и замороженной цели ядро не может ни
		// защитить позицию, ни закрыть её. Если реконструкция не удалась — пропуск, а не
		// угаданные величины: они молча увели бы уровень стопа от того, что задала
		// стратегия. Биржевой стоп при этом НЕ выставляется здесь: восстановленный entry
		// идёт обычным путём manage ниже, где пустой StopOrderID сначала снимет чужую
		// заявку по инструменту (если она осталась с прошлой жизни раннера), а затем
		// поставит свою.
		rebuilt, rerr := sl.strat.Reconstruct(ctx, adapter.ReconstructInput{
			Trades: s.ops, Candles: s.market, AccountID: s.cfg.AccountID,
			InstrumentID: sh.ID, Ticker: ticker,
			PurchasePrice: utils.CombinePrice(pos.PurchasePrice.Units, pos.PurchasePrice.Nano),
			Now:           pc.now,
		})
		if rerr != nil {
			sl.alert(ticker, "позиция без локального стейта, реконструкция не удалась: "+rerr.Error())
			logger.ErrorContext(ctx, fmt.Sprintf("%s: reconstruct %s: %v", sl.strat.Name(), ticker, rerr))
			return nil
		}
		rebuilt.Strategy = sl.strat.Name()
		rebuilt.Quantity = pos.Quantity
		entry = rebuilt
		pc.state[ticker] = entry
		if err := pc.store.Save(pc.state); err != nil {
			return fmt.Errorf("%s: save reconstructed state %s: %w", sl.strat.Name(), ticker, err)
		}
		sl.alert(ticker, fmt.Sprintf(
			"стейт восстановлен по API: вход %.4f от %s, дневной ATR %.4f, цель %.4f",
			entry.EntryPrice, entry.EntryTime.Format("02.01 15:04"), entry.EntryATR, entry.TakeProfit))
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
		sl.alert(ticker, fmt.Sprintf("позиция уменьшилась частично (стоп или ручная продажа), осталось %d", pos.Quantity))
	}

	// Судьба биржевой заявки разбирается ДО Decide — иначе сработавший стоп ведёт к
	// двойной продаже. Уровень заявки и уровень, по которому решает ядро, совпадают по
	// построению: заявка выставлена от того же prevMaxFav, а округление вниз делает
	// биржевой уровень не выше ядрового. Значит всякий раз, когда стоп срабатывает
	// внутри бара, Decide на этом же баре тоже вернёт SELL. Если портфель брокера при
	// этом ещё не обновился (тот же лаг расчётов, ради которого существует
	// freshEntryGrace), мы попадаем сюда с isHeld=true и, решая по Decide, пошли бы
	// продавать рыночным ордером бумаги, которых уже нет — отказ брокера в лучшем
	// случае, шорт на марже в худшем. Синхронизация возможна только при работающем List.
	strayCancelFailed := false
	if pc.listErr == nil {
		if entry.StopOrderID != "" {
			if _, alive := pc.stopByID[entry.StopOrderID]; !alive {
				// Заявки нет в ACTIVE: сработала или снята вне раннера. Различить
				// обязательно — репост свежего стопа на уже проданную стопом позицию
				// обернулся бы фантомной продажей/шортом при касании уровня.
				fired, ferr := sl.stops.Executed(ctx, entry.StopOrderID)
				switch {
				case ferr != nil:
					// Не репостим вслепую — ретрай на следующем получасовом тике.
					sl.alert(ticker, "стоп-заявка исчезла из ACTIVE, но EXECUTED недоступен — репост отложен: "+ferr.Error())
					return nil
				case fired:
					sl.strat.Notify(notifier.Exit(ticker, entry.StopReason, entry.StopPrice, entry.Quantity, false))
					delete(pc.state, ticker)
					_ = pc.store.Save(pc.state)
					return nil
				default:
					sl.alert(ticker, "стоп-заявка снята вне раннера — перевыставляю")
					entry.StopOrderID = ""
				}
			}
		} else if stray, ok := pc.stopByInstrument[sh.ID]; ok {
			// Чужая/устаревшая заявка (например, после восстановления стейта) — снять.
			if err := sl.stops.Cancel(ctx, stray.StopOrderID); err != nil {
				sl.alert(ticker, "не удалось снять неизвестную стоп-заявку: "+err.Error())
				// Не ставим новую заявку в этом тике: stray-заявка всё ещё жива на
				// бирже и продолжает защищать позицию (см. guard ниже). Без этого
				// флага на бирже оказались бы ДВЕ живые SELL-заявки на один
				// инструмент — вторая продала бы уже не имеющиеся бумаги.
				strayCancelFailed = true
			}
		}
	}

	prevMaxFav := entry.MaxFav // уровень, от которого считалась стоящая на бирже заявка
	// Raise maxFav from the latest completed close, then persist (monotonic).
	if md.Price > entry.MaxFav {
		entry.MaxFav = md.Price
		pc.state[ticker] = entry
		if err := pc.store.Save(pc.state); err != nil {
			return fmt.Errorf("%s: save maxFav %s: %w", sl.strat.Name(), ticker, err)
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

	sig := dec.Decide(md)
	if sig.Kind != model.SignalSell && entry.PendingExit != "" {
		// Выход уже принят стратегией на одном из прошлых баров, но брокер его не
		// исполнил. Выходы по индикатору — события ОДНОГО бара (крест RSI вверх через
		// порог), поэтому «повторим на следующем тике» без этой отметки означает не
		// повтор, а потерю выхода: позиция досидит до стопа или цели.
		sig = model.Signal{Kind: model.SignalSell, Reason: entry.PendingExit, Price: md.Price}
	}
	if sig.Kind == model.SignalSell {
		return s.sell(ctx, pc, sl, ticker, sh, entry, pos, sig)
	}

	// Желаемый уровень от ОБНОВЛЁННОГО MaxFav — на гранулярности шага цены биржи:
	// сырой уровень может расти на доли шага каждые полчаса, а биржевая цена после
	// округления не меняется; сравнение сырых значений гоняло бы cancel+repost
	// по той же цене с окном без защиты на каждом тике.
	level, reason := sl.strat.DesiredStop(ticker, entry)
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
		switch {
		case pc.listErr != nil:
			// List не ответил — мы не знаем, не висит ли уже заявка по этому
			// инструменту (осталась с прошлой жизни раннера, потерялся StopOrderID).
			// Две живые SELL-заявки на одну позицию хуже получаса без биржевой
			// защиты: вторая продаст бумаги, которых уже нет, то есть откроет шорт
			// на марже. Ретрай на следующем получасовом тике.
			sl.alert(ticker, "список заявок недоступен — постановка стопа отложена до следующего тика")
		case strayCancelFailed:
			// Stray-заявка не снялась и всё ещё жива на бирже — она продолжает
			// защищать позицию. НЕ ставим вторую: alert уже ушёл выше, ретрай
			// снятия на следующем получасовом тике.
		default:
			entry = s.replaceStop(ctx, sl, ticker, sh, entry, level, reason)
		}
	case sizeMismatch, desired > current:
		if err := sl.stops.Cancel(ctx, entry.StopOrderID); err != nil {
			sl.alert(ticker, "не удалось снять стоп для переноса: "+err.Error())
			break // старая заявка продолжает защищать
		}
		entry.StopOrderID = ""
		entry = s.replaceStop(ctx, sl, ticker, sh, entry, level, reason)
	}
	if prev := pc.state[ticker]; prev != entry {
		pc.state[ticker] = entry
		if err := pc.store.Save(pc.state); err != nil {
			return fmt.Errorf("%s: save stop state %s: %w", sl.strat.Name(), ticker, err)
		}
	}
	return nil
}

// sell closes the position on any SELL the core emits: the exchange stop comes off first,
// then the market order. A stop left standing would later sell a NEW position in the same
// ticker; a sell that goes through without it would double-sell this one.
//
// Every path that fails to close the position stamps entry.PendingExit before returning:
// the core's indicator exits are single-bar events, so a failure that is not remembered is
// a lost exit, not a retry.
func (s *service) sell(ctx context.Context, pc *passCtx, sl *slot, ticker string, sh *imodel.Share,
	entry statestore.Entry, pos *grpcmodel.Position, sig model.Signal) error {

	// Причина не должна быть пустой: "" в PendingExit неотличимо от «выхода не ждём».
	pending := sig.Reason
	if pending == "" {
		pending = "EXIT"
	}
	// remember ставит отметку на ТЕКУЩЕЙ записи в стейте (а не на локальной копии entry):
	// ветки ниже успевают её переписать, и затирать их значением из аргумента нельзя.
	remember := func() {
		if e, ok := pc.state[ticker]; ok && e.PendingExit != pending {
			e.PendingExit = pending
			pc.state[ticker] = e
			_ = pc.store.Save(pc.state)
		}
	}

	// Guard ДО снятия стопа: без лота продать нельзя, а уже снятая заявка
	// оставила бы позицию без биржевой защиты навсегда (replaceStop с тем же
	// guard'ом её не вернёт).
	if sh.Lot <= 0 {
		sl.alert(ticker, "sh.Lot == 0 — невозможно вычислить лоты для продажи, пропуск")
		logger.ErrorContext(ctx, fmt.Sprintf("%s: %s sh.Lot=%d, skipping sell to avoid divide-by-zero", sl.strat.Name(), ticker, sh.Lot))
		remember()
		return nil
	}

	hadStop := entry.StopOrderID != ""
	if hadStop {
		if err := sl.stops.Cancel(ctx, entry.StopOrderID); err != nil {
			sl.alert(ticker, "не удалось снять стоп-заявку перед продажей: "+err.Error())
			logger.ErrorContext(ctx, fmt.Sprintf("%s: %s cancel before sell: %v", sl.strat.Name(), ticker, err))
			remember()
			return nil // без снятия продавать нельзя — двойная продажа
		}
		entry.StopOrderID = ""
		pc.state[ticker] = entry
		_ = pc.store.Save(pc.state)
	}

	lots := pos.Quantity / int64(sh.Lot)
	res, err := sl.exec.Sell(ctx, sh.ID, lots)
	if err != nil {
		sl.alert(ticker, "ордер на продажу отклонён: "+err.Error())
		logger.ErrorContext(ctx, fmt.Sprintf("%s: %s sell rejected: %v", sl.strat.Name(), ticker, err))
		// Стоп уже снят, а продажа не прошла — позиция «голая» до следующего
		// тика. Возвращаем биржевую защиту на прежнем уровне (дубль StopSet
		// подавит changed-флаг replaceStop: уровень/причина не менялись).
		entry.PendingExit = pending
		if hadStop && entry.StopReason != "" {
			entry = s.replaceStop(ctx, sl, ticker, sh, entry, entry.StopPrice, entry.StopReason)
			if entry.StopOrderID != "" {
				sl.alert(ticker, "стоп-заявка перевыставлена после отклонённой продажи")
			}
		}
		pc.state[ticker] = entry
		_ = pc.store.Save(pc.state)
		return nil // повтор на следующем тике — по отметке PendingExit, а не по сигналу
	}

	if res.Placed && res.FilledLots == 0 {
		// Ордер принят, но не исполнен ни на один лот — позиция всё ещё открыта. Чистка
		// стейта здесь стёрла бы замороженную цель и ATR входа живой позиции, а выход
		// остался бы неисполненным и никем не запомненным.
		sl.alert(ticker, "ордер на продажу принят, но не исполнен (0 лотов) — выход отложен")
		logger.ErrorContext(ctx, fmt.Sprintf("%s: %s sell accepted with zero fill", sl.strat.Name(), ticker))
		entry.PendingExit = pending
		if hadStop && entry.StopReason != "" {
			entry = s.replaceStop(ctx, sl, ticker, sh, entry, entry.StopPrice, entry.StopReason)
		}
		pc.state[ticker] = entry
		_ = pc.store.Save(pc.state)
		return nil
	}

	exitPrice := sig.Price
	if res.Placed && res.FillPrice > 0 {
		exitPrice = res.FillPrice
	}
	delete(pc.state, ticker)
	if err := pc.store.Save(pc.state); err != nil {
		return fmt.Errorf("%s: save state after sell %s: %w", sl.strat.Name(), ticker, err)
	}
	sl.strat.Notify(notifier.Exit(ticker, sig.Reason, exitPrice, pos.Quantity, !res.Placed))
	return nil
}

// settleGonePosition reconciles a ticker the broker no longer holds: either our stop fired,
// or the position was sold outside the runner and the stop is orphaned. Telling the two
// apart matters — an orphaned stop left on the exchange would later sell a new position in
// the same ticker, and a false "stop fired" notification would misreport the exit.
func (s *service) settleGonePosition(ctx context.Context, pc *passCtx, sl *slot, ticker string) {
	entry, hadState := pc.state[ticker]
	switch {
	case !hadState:
		// Ничего не знаем про тикер — не наша забота.
	case pc.now.Sub(entry.EntryTime) < freshEntryGrace:
		// Позиция слишком свежая, чтобы её отсутствие в портфеле что-то доказывало:
		// расчёты брокера отстают от исполненного ордера. Трактовать это как продажу
		// нельзя — ниже такая трактовка снимает биржевой стоп и чистит стейт, то есть
		// оставила бы РЕАЛЬНО открытую позицию без защиты, а следующий пасс, увидев
		// пустой стейт, вошёл бы в неё второй раз.
		sl.alert(ticker, "портфель ещё не показывает свежую позицию — сопровождение отложено, стоп и стейт не тронуты")
	case pc.listErr != nil:
		// Не можем свериться с биржей, есть ли ещё живая заявка — значит не можем
		// отличить сработавший стоп от ручной продажи с осиротевшей заявкой.
		// Консервативно: alert, стейт не трогаем, повтор на следующем тике.
		sl.alert(ticker, "позиция исчезла, но GetStopOrders недоступен — не могу подтвердить срабатывание стопа, стейт сохранён")
	case entry.StopOrderID == "":
		// Нет заявки, за которой нужно присматривать, — просто чистим стейт.
		delete(pc.state, ticker)
		_ = pc.store.Save(pc.state)
	default:
		if _, alive := pc.stopByID[entry.StopOrderID]; alive {
			// Заявка ещё жива на бирже — значит, позицию продали НЕ через наш
			// стоп (например, вручную в приложении брокера). Снимаем осиротевшую
			// заявку, иначе она позже продаст новую позицию по этому тикеру.
			if err := sl.stops.Cancel(ctx, entry.StopOrderID); err != nil {
				sl.alert(ticker, "позиция продана вне раннера, не удалось снять осиротевший стоп: "+err.Error())
				return // стейт не чистим — ретрай на следующем тике
			}
			sl.alert(ticker, "позиция продана вне раннера, снял осиротевший стоп")
			delete(pc.state, ticker)
			_ = pc.store.Save(pc.state)
			return
		}
		// Заявки нет в живых: либо сработал наш биржевой стоп, либо её сняли
		// вне раннера вместе с продажей позиции. Различаем по EXECUTED-списку;
		// при его недоступности считаем срабатыванием — на PnL это не влияет,
		// вопрос только в тексте уведомления.
		if fired, ferr := sl.stops.Executed(ctx, entry.StopOrderID); ferr == nil && !fired {
			sl.alert(ticker, "позиция закрыта и стоп-заявка снята вне раннера — чищу стейт")
		} else {
			sl.strat.Notify(notifier.Exit(ticker, entry.StopReason, entry.StopPrice, entry.Quantity, false))
		}
		delete(pc.state, ticker)
		_ = pc.store.Save(pc.state)
	}
}
