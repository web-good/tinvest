# RSI Pullback на дневном ATR — план реализации

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Перевести backtest-стратегию `rsi_pullback` с внутридневного режима на многодневный: стоп и цель в единицах дневного ATR, двусторонний гейт по состоянию дня, слотовый фильтр фона объёмов, удержание через ночь.

**Architecture:** Стратегия остаётся чистой и stateless между барами: всё считается из `strategy.MarketData`, состояние позиции живёт только в `md.Position`. Дневной ATR берётся из уже существующих `md.DailyHighs/DailyLows/DailyCloses` (движок кладёт туда только завершённые дни), к ним добавляется новое поле `md.DailyTimes`, чтобы стратегия могла выбросить выходные сессии. Тейк-профит реализуется штатным механизмом движка (`sig.TakeProfit` + `Reason == "TP"`), новых возможностей движку не требуется.

**Tech Stack:** Go 1.25, `go test -race`, `./bin/mage ci` (golangci-lint v2 + тесты + проверка дрейфа моков).

## Global Constraints

- Спека: `docs/superpowers/specs/2026-07-28-rsi-pullback-daily-atr-design.md`. При расхождении плана и спеки — спрашивать, не выбирать молча.
- Все поля `core.Params` — только `int` или `float64`. Рефлексивная калибровка (`applyField` в `internal/service/backtest/calibrate.go`) другие типы не видит и молча их не свипует.
- Коды причин выхода — ровно `"SL"`, `"TP"`, `"RSI"`. `"SL"` обязан оставаться в `model.IsStopReason`, `"TP"` обрабатывается движком отдельно, `"RSI"` заполняется по закрытию бара.
- Никакого lookahead: решение на баре `i` использует только бары `0..i` и только завершённые дневные свечи.
- Любой гейт при отсутствующих или битых данных **пропускает** вход, а не блокирует его. Единственное исключение — дневной ATR: без него входа нет, потому что без него нет ни стопа, ни цели.
- Таймзона — `Europe/Moscow` с откатом на UTC (`mskLoc` уже есть в пакете).
- Выходные (суббота и воскресенье MSK) исключаются и из расчёта дневного ATR, и из базы объёмов, и из окна проверяемых баров.
- Стратегия backtest-only. Живой раннер к ней не подключается, в `internal/app` она не регистрируется.
- Каждая задача заканчивается зелёным `go test ./...` по затронутым пакетам и коммитом. Финальная задача — зелёным `./bin/mage ci`.
- Комментарии в коде — на английском (как в текущем `core.go`), тексты для владельца (`EntryReason`, `ExitReason`, `Explain`, документация) — на русском.

---

## File Structure

**Изменяются:**

- `internal/service/trading_strategy/scalping/strategy/strategy.go` — новое поле `DailyTimes` в `MarketData`.
- `internal/domain/backtest/engine.go` — объединённый хелпер `visibleDaily`, заполнение `DailyTimes` в трёх местах.
- `internal/domain/backtest/engine_test.go` — тесты объединённого хелпера.
- `internal/service/trading_strategy/rsi_pullback/strategy/core/core.go` — правила стратегии целиком.
- `internal/service/trading_strategy/rsi_pullback/strategy/core/core_test.go` — тесты стратегии.
- `internal/service/backtest/rsi_pullback_grid_test.go` — пин сетки.
- `data/params/rsi_pullback/grid.json` — сетка калибровки.
- `docs/rsi_pullback/strategy.md` — документация стратегии.
- `CLAUDE.md` — одна строка описания стратегии.

**Не изменяются:** `cmd/backtest/main.go` (реестр и флаги уже поддерживают `rsi_pullback`), `internal/service/backtest/registry.go`, живой `reversion` (он передаёт `daily = nil`).

---

## Task 1: Поле `DailyTimes` и объединённый хелпер движка

**Files:**
- Modify: `internal/service/trading_strategy/scalping/strategy/strategy.go` (после поля `DailyLows`, строки 55-58)
- Modify: `internal/domain/backtest/engine.go:33-58` (два хелпера → один), `:175-176`, `:243-244`, `:340-341` (места заполнения)
- Test: `internal/domain/backtest/engine_test.go:128-152` (заменить `TestVisibleDailyCloses`)

**Interfaces:**
- Produces: поле `strategy.MarketData.DailyTimes []time.Time` — открытия тех же завершённых дневных свечей, что и `DailyCloses`, индекс в индекс. Пусто, когда дневных данных нет.
- Produces: `visibleDaily(daily []Candle, t time.Time, loc *time.Location) (closes, highs, lows []float64, times []time.Time)` — заменяет `visibleDailyCloses` и `visibleDailyHighsLows`.

- [ ] **Step 1: Написать падающий тест**

Заменить целиком существующий `TestVisibleDailyCloses` в `internal/domain/backtest/engine_test.go` (строки 128-152) на:

```go
func TestVisibleDaily(t *testing.T) {
	msk, _ := time.LoadLocation("Europe/Moscow")
	day := func(y int, m time.Month, d int) Candle {
		return Candle{
			Time:  time.Date(y, m, d, 0, 0, 0, 0, time.UTC),
			High:  float64(d) + 0.5,
			Low:   float64(d) - 0.5,
			Close: float64(d),
		}
	}
	daily := []Candle{day(2026, 1, 1), day(2026, 1, 2), day(2026, 1, 3)}

	t3 := time.Date(2026, 1, 3, 12, 0, 0, 0, time.UTC)
	closes, highs, lows, times := visibleDaily(daily, t3, msk)
	if len(closes) != 2 || closes[0] != 1 || closes[1] != 2 {
		t.Fatalf("closes on Jan 3 = %v, want [1 2]", closes)
	}
	if len(highs) != 2 || len(lows) != 2 || len(times) != 2 {
		t.Fatalf("series not index-aligned: %d closes, %d highs, %d lows, %d times",
			len(closes), len(highs), len(lows), len(times))
	}
	if highs[1] != 2.5 || lows[1] != 1.5 {
		t.Fatalf("highs/lows on Jan 3 = %v/%v, want 2.5/1.5", highs[1], lows[1])
	}
	if !times[1].Equal(daily[1].Time) {
		t.Fatalf("times[1] = %v, want %v", times[1], daily[1].Time)
	}

	t1 := time.Date(2026, 1, 1, 10, 0, 0, 0, time.UTC)
	if closes, _, _, times := visibleDaily(daily, t1, msk); len(closes) != 0 || len(times) != 0 {
		t.Fatalf("visible on Jan 1 = %v / %v, want empty", closes, times)
	}

	t9 := time.Date(2026, 1, 9, 0, 0, 0, 0, time.UTC)
	if closes, _, _, times := visibleDaily(daily, t9, msk); len(closes) != 3 || len(times) != 3 {
		t.Fatalf("visible on Jan 9 = %d closes / %d times, want 3/3", len(closes), len(times))
	}
}

func TestRunPopulatesDailyTimes(t *testing.T) {
	base := time.Date(2026, 1, 5, 10, 0, 0, 0, time.UTC) // понедельник
	candles := make([]Candle, 0, 4)
	for i := 0; i < 4; i++ {
		candles = append(candles, Candle{
			Time: base.AddDate(0, 0, i), High: 101, Low: 99, Close: 100, Volume: 10,
		})
	}
	daily := []Candle{
		{Time: time.Date(2026, 1, 5, 0, 0, 0, 0, time.UTC), High: 102, Low: 98, Close: 100},
		{Time: time.Date(2026, 1, 6, 0, 0, 0, 0, time.UTC), High: 103, Low: 97, Close: 101},
	}
	var seen []time.Time
	var seenCloses []float64
	s := scriptedStrategy{lookback: 1, decide: func(md strategy.MarketData) model.Signal {
		seen = md.DailyTimes
		seenCloses = md.DailyCloses
		return model.Signal{Kind: model.SignalNone}
	}}
	Run(s, candles, daily, nil, Config{InitialCash: 1000, Fraction: 1.0, Lot: 1})
	if len(seen) != len(seenCloses) {
		t.Fatalf("DailyTimes len %d != DailyCloses len %d", len(seen), len(seenCloses))
	}
	if len(seen) != 2 {
		t.Fatalf("DailyTimes on the last bar = %d, want 2", len(seen))
	}
}
```

- [ ] **Step 2: Убедиться, что тест падает**

Run: `go test ./internal/domain/backtest/ -run 'TestVisibleDaily|TestRunPopulatesDailyTimes' -v`
Expected: FAIL — `undefined: visibleDaily`, `md.DailyTimes undefined`.

- [ ] **Step 3: Добавить поле в `MarketData`**

В `internal/service/trading_strategy/scalping/strategy/strategy.go` сразу после `DailyLows []float64`:

```go
	// DailyTimes are the open-times of the same COMPLETED daily candles as DailyCloses
	// (aligned index-for-index). Empty when no daily data is supplied. Consumers that
	// filter the daily series by calendar — e.g. dropping MOEX weekend sessions before
	// computing a daily ATR — must degrade gracefully when it is empty or not
	// length-aligned with the price slices.
	DailyTimes []time.Time
```

- [ ] **Step 4: Заменить два хелпера одним**

В `internal/domain/backtest/engine.go` удалить `visibleDailyCloses` (строки 33-45) и `visibleDailyHighsLows` (строки 47-58), на их место поставить:

```go
// visibleDaily returns the closes, highs, lows and open-times of daily candles whose
// calendar day (in loc) is strictly before t's calendar day — i.e. days that have fully
// closed by t. This is the no-lookahead rule: the current, still-forming day is never
// visible. The four series are index-aligned, oldest-first.
func visibleDaily(daily []Candle, t time.Time, loc *time.Location) (closes, highs, lows []float64, times []time.Time) {
	bound := startOfDay(t, loc)
	for _, c := range daily {
		if c.Time.Before(bound) {
			closes = append(closes, c.Close)
			highs = append(highs, c.High)
			lows = append(lows, c.Low)
			times = append(times, c.Time)
		}
	}
	return closes, highs, lows, times
}
```

- [ ] **Step 5: Перевести три места заполнения**

В `Run` (было 175-176), в `Trace` (было 243-244) и в `AssembleMarketDataWithHTFInterval` (было 340-341) заменить пару строк одной. В `Run` и `Trace`:

```go
		md.DailyCloses, md.DailyHighs, md.DailyLows, md.DailyTimes = visibleDaily(dailyCandles, candles[i].Time, mskLoc)
```

В `AssembleMarketDataWithHTFInterval`:

```go
	md.DailyCloses, md.DailyHighs, md.DailyLows, md.DailyTimes = visibleDaily(daily, cur, mskLoc)
```

- [ ] **Step 6: Прогнать тесты пакета**

Run: `go test ./internal/domain/backtest/ ./internal/service/trading_strategy/... -count=1`
Expected: PASS. Если какой-то тест ссылался на удалённые хелперы — переписать его на `visibleDaily`, поведение отбора баров не менялось.

- [ ] **Step 7: Коммит**

```bash
git add internal/domain/backtest/engine.go internal/domain/backtest/engine_test.go \
        internal/service/trading_strategy/scalping/strategy/strategy.go
git commit -m "feat(backtest): DailyTimes в MarketData и объединённый visibleDaily"
```

---

## Task 2: Дневной ATR по будним дням

**Files:**
- Modify: `internal/service/trading_strategy/rsi_pullback/strategy/core/core.go` (добавить `DailyATRPeriod` в `Params` и два новых хелпера)
- Test: `internal/service/trading_strategy/rsi_pullback/strategy/core/core_test.go`

**Interfaces:**
- Consumes: `strategy.MarketData.DailyTimes` из Task 1.
- Produces: `func weekdayDaily(highs, lows, closes []float64, times []time.Time) (h, l, c []float64)` — синхронная фильтрация выходных из дневных серий.
- Produces: `func (s *Strategy) dailyATR(md strategy.MarketData) float64` — ATR Уайлдера по будним завершённым дневкам; `0`, когда данных не хватает.
- Produces: поле `Params.DailyATRPeriod int`, дефолт `14`.

Остальные поля `Params` в этой задаче **не трогаются** — их перестройка идёт в Task 3 и Task 4, чтобы каждая задача оставляла пакет компилируемым.

- [ ] **Step 1: Написать падающие тесты**

Добавить в `core_test.go` рядом с существующими хелперами:

```go
// dailyBars builds `days` consecutive calendar days ending the day BEFORE `before` (MSK),
// oldest-first. Every bar closes at 100; a weekday bar spans `w` and a weekend bar spans
// `we`, so a test can prove weekend sessions never reach the ATR. With a flat close the true
// range of each bar equals its own width, so ATR over N equal weekday bars is exactly `w`.
func dailyBars(before time.Time, days int, w, we float64) (highs, lows, closes []float64, times []time.Time) {
	b := before.In(msk)
	start := time.Date(b.Year(), b.Month(), b.Day(), 0, 0, 0, 0, msk).AddDate(0, 0, -days)
	const price = 100.0
	for i := 0; i < days; i++ {
		d := start.AddDate(0, 0, i)
		width := w
		if d.Weekday() == time.Saturday || d.Weekday() == time.Sunday {
			width = we
		}
		highs = append(highs, price+width/2)
		lows = append(lows, price-width/2)
		closes = append(closes, price)
		times = append(times, d.Add(10*time.Hour))
	}
	return highs, lows, closes, times
}

func TestDailyATRIgnoresWeekends(t *testing.T) {
	s := NewWithParams("TEST", DefaultParams())
	now := time.Date(2026, 3, 2, 10, 0, 0, 0, msk) // понедельник

	// 40 календарных дней: будни шириной 2.0, выходные — намеренно узкие 0.2.
	h, l, c, ts := dailyBars(now, 40, 2.0, 0.2)
	withWeekend := strategy.MarketData{DailyHighs: h, DailyLows: l, DailyCloses: c, DailyTimes: ts}

	// Та же серия, но выходные вырезаны заранее.
	var wh, wl, wc []float64
	var wt []time.Time
	for i := range c {
		if wd := ts[i].In(msk).Weekday(); wd == time.Saturday || wd == time.Sunday {
			continue
		}
		wh = append(wh, h[i])
		wl = append(wl, l[i])
		wc = append(wc, c[i])
		wt = append(wt, ts[i])
	}
	weekdaysOnly := strategy.MarketData{DailyHighs: wh, DailyLows: wl, DailyCloses: wc, DailyTimes: wt}

	got, want := s.dailyATR(withWeekend), s.dailyATR(weekdaysOnly)
	if math.Abs(got-want) > 1e-9 {
		t.Fatalf("ATR с выходными %.6f != ATR без выходных %.6f", got, want)
	}
	if math.Abs(got-2.0) > 1e-6 {
		t.Fatalf("ATR = %.6f, want 2.0 (ширина буднего бара)", got)
	}
}

func TestDailyATRZeroWhenDataCannotSupportIt(t *testing.T) {
	s := NewWithParams("TEST", DefaultParams())
	now := time.Date(2026, 3, 2, 10, 0, 0, 0, msk)

	// Будних дней меньше, чем DailyATRPeriod+1.
	h, l, c, ts := dailyBars(now, 10, 2.0, 0.2)
	if got := s.dailyATR(strategy.MarketData{DailyHighs: h, DailyLows: l, DailyCloses: c, DailyTimes: ts}); got != 0 {
		t.Fatalf("ATR на короткой истории = %.6f, want 0", got)
	}
	if got := s.dailyATR(strategy.MarketData{}); got != 0 {
		t.Fatalf("ATR без дневных данных = %.6f, want 0", got)
	}
}

func TestDailyATRDegradesWhenTimesMisaligned(t *testing.T) {
	s := NewWithParams("TEST", DefaultParams())
	now := time.Date(2026, 3, 2, 10, 0, 0, 0, msk)
	h, l, c, ts := dailyBars(now, 40, 2.0, 0.2)

	// Времена короче ценовых серий: фильтровать нечем — серия должна пойти в ATR как есть,
	// а не обнулиться и не паниковать.
	md := strategy.MarketData{DailyHighs: h, DailyLows: l, DailyCloses: c, DailyTimes: ts[:5]}
	if got := s.dailyATR(md); got <= 0 {
		t.Fatalf("ATR при рассинхроне времён = %.6f, want > 0 (деградация, не отказ)", got)
	}
}
```

- [ ] **Step 2: Убедиться, что тесты падают**

Run: `go test ./internal/service/trading_strategy/rsi_pullback/... -run 'TestDailyATR' -v`
Expected: FAIL — `s.dailyATR undefined`.

- [ ] **Step 3: Добавить параметр и хелперы**

В `Params` добавить поле (пока рядом с существующими, порядок полей не важен):

```go
	DailyATRPeriod  int     // daily ATR length, computed over WEEKDAY completed daily candles (grid; default 14)
```

В `DefaultParams()` — `DailyATRPeriod: 14,`.

После `isWeekend` добавить:

```go
// weekdayDaily drops weekend (Sat/Sun MSK) bars from the daily series, keeping the three
// price slices aligned. MOEX runs weekend sessions and the candle cache contains them, but
// those bars are 3-4x narrower and 8-17x thinner than weekday ones: leaving them in
// understates the daily ATR by 9-16% (measured on GAZP/SBER/NVTK/RUAL), and that error would
// propagate into the stop, the target and both thresholds of the day gate at once. When times
// is empty or not aligned with the price slices there is nothing to filter by — the series is
// returned untouched rather than guessed at.
func weekdayDaily(highs, lows, closes []float64, times []time.Time) (h, l, c []float64) {
	n := len(closes)
	if n == 0 || len(highs) != n || len(lows) != n || len(times) != n {
		return highs, lows, closes
	}
	h = make([]float64, 0, n)
	l = make([]float64, 0, n)
	c = make([]float64, 0, n)
	for i := 0; i < n; i++ {
		if isWeekend(times[i].In(mskLoc)) {
			continue
		}
		h = append(h, highs[i])
		l = append(l, lows[i])
		c = append(c, closes[i])
	}
	return h, l, c
}

// dailyATR is the strategy's unit of risk: Wilder's ATR over COMPLETED weekday daily candles.
// The engine only ever exposes days that closed before the current bar, so no lookahead is
// possible here. Returns 0 whenever the data cannot support the calculation — the caller must
// then refuse the entry, because without it there is neither a stop nor a target.
func (s *Strategy) dailyATR(md strategy.MarketData) float64 {
	if s.p.DailyATRPeriod <= 0 {
		return 0
	}
	h, l, c := weekdayDaily(md.DailyHighs, md.DailyLows, md.DailyCloses, md.DailyTimes)
	if len(c) < s.p.DailyATRPeriod+1 {
		return 0
	}
	return indicators.ATR(h, l, c, s.p.DailyATRPeriod)
}
```

- [ ] **Step 4: Прогнать тесты**

Run: `go test ./internal/service/trading_strategy/rsi_pullback/... -count=1`
Expected: PASS (новые тесты зелёные, старые не затронуты).

- [ ] **Step 5: Коммит**

```bash
git add internal/service/trading_strategy/rsi_pullback/strategy/core/
git commit -m "feat(rsi_pullback): дневной ATR по будним завершённым дневкам"
```

---

## Task 3: Вход — двусторонний гейт по дню, стоп и цель в дневных ATR

**Files:**
- Modify: `internal/service/trading_strategy/rsi_pullback/strategy/core/core.go` (`Params`, `DefaultParams`, `Decide`, `enter`, `entryReason`, удаление `isDayEnd`/`barSpanMinutes`/`defaultBarSpanMin`)
- Test: `internal/service/trading_strategy/rsi_pullback/strategy/core/core_test.go`

**Interfaces:**
- Consumes: `s.dailyATR(md)` из Task 2.
- Produces: `func (s *Strategy) dayStateOK(md strategy.MarketData, atr float64) bool`.
- Produces: `Params` без `StopATR`, `ATRPeriod`, `DayEndMin`; с `UseDayATRGate int`, `FreshDayATR`, `SpentDayATR`, `StopDailyATR`, `TPDailyATR float64`.
- Produces: сигнал входа несёт `sig.StopLoss`, `sig.TakeProfit`, `sig.ATR` (дневной).
- Note: `MaxHoldBars` и выход `EOD` удаляются в Task 4, здесь они ещё живы.

- [ ] **Step 1: Написать падающие тесты**

В `core_test.go`: обновить `barSeries`, чтобы к серии можно было прицепить дневные данные и границы дня, и добавить тесты гейта. Добавить рядом с `barSeries`:

```go
// withDay attaches a completed weekday daily series (40 days, weekday width `atrWidth`) plus
// an explicit intraday extent to md, so entry tests can drive the day gate deterministically.
// The daily ATR of the attached series is exactly atrWidth.
func withDay(md strategy.MarketData, atrWidth, todayHigh, todayLow float64) strategy.MarketData {
	last := md.Times[len(md.Times)-1]
	h, l, c, ts := dailyBars(last, 40, atrWidth, atrWidth/10)
	md.DailyHighs, md.DailyLows, md.DailyCloses, md.DailyTimes = h, l, c, ts
	md.TodayHigh, md.TodayLow = todayHigh, todayLow
	return md
}
```

Тесты:

```go
func TestDayStateGateBothBranches(t *testing.T) {
	p := DefaultParams() // FreshDayATR 0.3, SpentDayATR 0.8
	s := NewWithParams("TEST", p)
	const atr = 10.0
	cases := []struct {
		name string
		used float64
		want bool
	}{
		{"день только начался", 1.0, true},
		{"ровно на границе свежести", 3.0, true},
		{"чуть выше границы свежести", 3.0001, false},
		{"мёртвая зона", 5.0, false},
		{"чуть ниже границы исчерпания", 7.9999, false},
		{"ровно на границе исчерпания", 8.0, true},
		{"день исчерпан", 12.0, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			md := strategy.MarketData{TodayHigh: 100 + tc.used, TodayLow: 100}
			if got := s.dayStateOK(md, atr); got != tc.want {
				t.Fatalf("used=%.4f: dayStateOK = %v, want %v", tc.used, got, tc.want)
			}
		})
	}
}

func TestDayStateGateDegradations(t *testing.T) {
	base := DefaultParams()
	dead := strategy.MarketData{TodayHigh: 105, TodayLow: 100} // ровно мёртвая зона при atr=10

	off := base
	off.UseDayATRGate = 0
	if !NewWithParams("TEST", off).dayStateOK(dead, 10) {
		t.Fatal("выключенный гейт обязан пропускать")
	}

	noThresholds := base
	noThresholds.FreshDayATR, noThresholds.SpentDayATR = 0, 0
	if !NewWithParams("TEST", noThresholds).dayStateOK(dead, 10) {
		t.Fatal("гейт без порогов обязан пропускать")
	}

	degenerate := base
	degenerate.FreshDayATR, degenerate.SpentDayATR = 0.8, 0.3 // ветки перекрываются
	if !NewWithParams("TEST", degenerate).dayStateOK(dead, 10) {
		t.Fatal("вырожденная конфигурация обязана пропускать всё")
	}

	s := NewWithParams("TEST", base)
	if !s.dayStateOK(strategy.MarketData{}, 10) {
		t.Fatal("без TodayHigh/TodayLow гейт обязан пропускать")
	}
	if !s.dayStateOK(dead, 0) {
		t.Fatal("без ATR гейт обязан пропускать (вход отсекается отдельной проверкой)")
	}
}

func TestEnterSetsStopAndTargetFromDailyATR(t *testing.T) {
	p := DefaultParams()
	s := NewWithParams("TEST", p)
	start := time.Date(2026, 3, 2, 7, 0, 0, 0, msk)
	md := barSeries(pullbackCloses(), start)
	md = withDay(md, 10.0, 101, 100) // used = 1 <= 0.3*10 -> ветка «день только начался»

	sig := s.Decide(md)
	if sig.Kind != model.SignalBuy {
		t.Fatalf("Kind = %v, want Buy", sig.Kind)
	}
	entry := md.Closes[len(md.Closes)-1]
	wantStop := entry - p.StopDailyATR*10.0
	wantTP := entry + p.TPDailyATR*10.0
	if math.Abs(sig.StopLoss-wantStop) > 1e-9 {
		t.Fatalf("StopLoss = %.6f, want %.6f", sig.StopLoss, wantStop)
	}
	if math.Abs(sig.TakeProfit-wantTP) > 1e-9 {
		t.Fatalf("TakeProfit = %.6f, want %.6f", sig.TakeProfit, wantTP)
	}
	if math.Abs(sig.ATR-10.0) > 1e-9 {
		t.Fatalf("ATR = %.6f, want 10.0 (дневной)", sig.ATR)
	}
}

func TestEnterRefusedWithoutDailyATR(t *testing.T) {
	s := NewWithParams("TEST", DefaultParams())
	start := time.Date(2026, 3, 2, 7, 0, 0, 0, msk)
	md := barSeries(pullbackCloses(), start) // дневных серий нет вовсе
	md.TodayHigh, md.TodayLow = 101, 100
	if sig := s.Decide(md); sig.Kind == model.SignalBuy {
		t.Fatal("вход без дневного ATR запрещён: нечем выставить стоп и цель")
	}
}

func TestEnterBlockedInTheDeadBand(t *testing.T) {
	s := NewWithParams("TEST", DefaultParams())
	start := time.Date(2026, 3, 2, 7, 0, 0, 0, msk)
	md := withDay(barSeries(pullbackCloses(), start), 10.0, 105, 100) // used = 5, мёртвая зона
	if sig := s.Decide(md); sig.Kind == model.SignalBuy {
		t.Fatal("вход в мёртвой зоне гейта запрещён")
	}
}
```

Кроме того, **обновить существующие тесты входа** (`TestEnterBuysThePullback`, `TestEnterGates`, `TestEnterAtSessionOpenBoundary`, `TestEnterCrossIsAnEventNotAState`, `TestEnterRejectsShortHistory`): везде, где ожидается `SignalBuy`, обернуть `md` в `withDay(md, 10.0, 101, 100)`, иначе вход теперь отсекается на гейте дневного ATR. В `TestEnterGates` подпункт про `StopATR` заменить на `StopDailyATR`.

- [ ] **Step 2: Убедиться, что тесты падают**

Run: `go test ./internal/service/trading_strategy/rsi_pullback/... -run 'TestDayState|TestEnter' -v`
Expected: FAIL — `s.dayStateOK undefined`, `p.StopDailyATR undefined`.

- [ ] **Step 3: Перестроить `Params`**

Заменить блок `Params` и `DefaultParams` (текущие строки 35-66) на:

```go
// Params holds every tunable. All fields are int or float64 so reflection grid calibration
// can sweep them.
type Params struct {
	RSIPeriod       int     // RSI length (grid; default 4)
	RSILower        float64 // lower band; a DOWNWARD cross of it is the entry (grid; default 15)
	RSIUpper        float64 // upper band; an UPWARD cross of it is the exit (grid; default 70)
	EMAFast         int     // fast EMA period (grid; default 10)
	EMASlow         int     // slow EMA period (grid; default 100)
	DailyATRPeriod  int     // daily ATR length, over WEEKDAY completed dailies (grid; default 14)
	UseDayATRGate   int     // 1 arms the two-sided day gate; any other value disables it (grid; default 1)
	FreshDayATR     float64 // "day barely started": range so far <= FreshDayATR*dailyATR (grid; default 0.3)
	SpentDayATR     float64 // "day spent": range so far >= SpentDayATR*dailyATR (grid; default 0.8)
	StopDailyATR    float64 // stop = entry - StopDailyATR*dailyATR; 0 disables it (grid; never 0 in the grid)
	TPDailyATR      float64 // target = entry + TPDailyATR*dailyATR; 0 disables it (grid)
	SessionStartMin int     // entry window start, minutes from MSK midnight (420 = 07:00)
	SessionEndMin   int     // entry window end, minutes from MSK midnight (1020 = 17:00)
}

// DefaultParams returns the spec's baseline; swept values come from calibration.
func DefaultParams() Params {
	return Params{
		RSIPeriod:       4,
		RSILower:        15,
		RSIUpper:        70,
		EMAFast:         10,
		EMASlow:         100,
		DailyATRPeriod:  14,
		UseDayATRGate:   1,
		FreshDayATR:     0.3,
		SpentDayATR:     0.8,
		StopDailyATR:    1.0,
		TPDailyATR:      0.6,
		SessionStartMin: 420,
		SessionEndMin:   1020,
	}
}
```

Поля объёмного фильтра (`UseVolume`, `VolBaseDays`, `VolLookbackBars`, `VolMult`) добавляются в Task 5.

- [ ] **Step 4: Реализовать гейт и перестроить `enter`**

Добавить после `inSession`:

```go
// dayStateOK reports whether the current day is in one of the two states this strategy trades.
// Either the day has barely started — its range so far is within FreshDayATR of the daily ATR,
// so the whole move is still ahead — or the day is spent: the range has already reached
// SpentDayATR, the sell-off has happened and a further leg down is less likely. The band
// between the two is refused: there the day has moved meaningfully but is neither fresh nor
// exhausted. The gate skips itself — never blocks — when disabled, when both thresholds are
// non-positive, when the thresholds overlap (FreshDayATR >= SpentDayATR makes every day pass
// anyway) or when the data cannot support it.
func (s *Strategy) dayStateOK(md strategy.MarketData, atr float64) bool {
	if s.p.UseDayATRGate != 1 || atr <= 0 {
		return true
	}
	if md.TodayHigh <= 0 || md.TodayLow <= 0 || md.TodayHigh < md.TodayLow {
		return true
	}
	fresh, spent := s.p.FreshDayATR, s.p.SpentDayATR
	if fresh <= 0 && spent <= 0 {
		return true
	}
	used := md.TodayHigh - md.TodayLow
	return (fresh > 0 && used <= fresh*atr) || (spent > 0 && used >= spent*atr)
}
```

Заменить `Decide` и `enter`:

```go
// Decide routes to entry (flat) or position management (open).
func (s *Strategy) Decide(md strategy.MarketData) model.Signal {
	sig := model.Signal{Ticker: s.ticker, Price: md.Price}
	if md.Position != nil {
		return s.manage(md, sig)
	}
	return s.enter(md, sig)
}

// enter emits a long when a short RSI crosses DOWN through its lower band on the current bar
// while the fast EMA sits above the slow one, the day is either fresh or spent, and the tape
// is busy. Everything is recomputed from md — no state survives between bars.
func (s *Strategy) enter(md strategy.MarketData, sig model.Signal) model.Signal {
	n := len(md.Closes)
	if n < 2 || len(md.Highs) != n || len(md.Lows) != n {
		return sig
	}
	// 1. entry window.
	if !s.inSession(s.barTime(md)) {
		return sig
	}
	i := n - 1
	// 2. RSI crosses down through the lower band on the current bar.
	rsi := indicators.RSISeries(md.Closes, s.p.RSIPeriod)
	if len(rsi) != n || !crossedDown(rsi, i, s.p.RSILower) {
		return sig
	}
	// 3. trend confirmation: fast EMA above slow EMA (both warmed).
	fast, slow, ok := s.emaPair(md.Closes)
	if !ok || fast[i] <= slow[i] {
		return sig
	}
	// 4. the daily ATR is the unit of both the stop and the target: no ATR, no trade.
	atr := s.dailyATR(md)
	if atr <= 0 {
		return sig
	}
	// 5. the day must be either fresh or spent.
	if !s.dayStateOK(md, atr) {
		return sig
	}
	entry := md.Closes[i]
	var stop, target float64
	if s.p.StopDailyATR > 0 {
		stop = entry - s.p.StopDailyATR*atr
	}
	if s.p.TPDailyATR > 0 {
		target = entry + s.p.TPDailyATR*atr
	}
	sig.Kind = model.SignalBuy
	sig.StopLoss = stop
	sig.TakeProfit = target
	sig.ATR = atr
	sig.RSI = rsi[i]
	sig.EntryReason = s.entryReason(rsi[i], fast[i], slow[i], entry, stop, target, atr, md)
	return sig
}

// entryReason renders the human-readable rationale shown in the trade journal.
func (s *Strategy) entryReason(rsiNow, fastNow, slowNow, entry, stop, target, atr float64, md strategy.MarketData) string {
	stopHow := "стоп выключен"
	if stop > 0 {
		stopHow = fmt.Sprintf("стоп %.4f (−%.2f ATR)", stop, s.p.StopDailyATR)
	}
	tpHow := "цель выключена"
	if target > 0 {
		tpHow = fmt.Sprintf("цель %.4f (+%.2f ATR)", target, s.p.TPDailyATR)
	}
	dayHow := "гейт дня выключен"
	if s.p.UseDayATRGate == 1 && md.TodayHigh > 0 && md.TodayLow > 0 && atr > 0 {
		dayHow = fmt.Sprintf("день прошёл %.2f ATR", (md.TodayHigh-md.TodayLow)/atr)
	}
	return fmt.Sprintf(
		"RSI(%d) ушёл под %.0f (%.1f) на откате, EMA(%d) %.4f > EMA(%d) %.4f, %s (дневной ATR %.4f); вход %.4f, %s, %s",
		s.p.RSIPeriod, s.p.RSILower, rsiNow, s.p.EMAFast, fastNow, s.p.EMASlow, slowNow,
		dayHow, atr, entry, stopHow, tpHow,
	)
}
```

- [ ] **Step 5: Убрать то, что стало неиспользуемым**

`manage` в этой задаче ещё принимает `span` — временно передать ему `barSpanMinutes(md.Times)` прямо в `Decide` нельзя, потому что сигнатура уже изменена. Поэтому на время Task 3 внутри `manage` первой строкой получить span самостоятельно:

```go
	span := barSpanMinutes(md.Times)
```

и изменить сигнатуру на `func (s *Strategy) manage(md strategy.MarketData, sig model.Signal) model.Signal`. Всё остальное в `manage` не трогать — оно переписывается в Task 4. `isDayEnd`, `barSpanMinutes` и `defaultBarSpanMin` пока остаются.

Из `enter` вызов `s.isDayEnd` убран — это осознанно: `SessionEndMin` (17:00) и так лежит раньше конца дня, а выхода EOD после Task 4 не будет вовсе.

- [ ] **Step 6: Прогнать тесты**

Run: `go test ./internal/service/trading_strategy/rsi_pullback/... -count=1`
Expected: PASS. Тесты выходов (`TestExitEndOfDay`, `TestExitTimeStop*`) должны продолжать проходить — они переписываются в Task 4.

- [ ] **Step 7: Коммит**

```bash
git add internal/service/trading_strategy/rsi_pullback/strategy/core/
git commit -m "feat(rsi_pullback): двусторонний гейт по дню, стоп и цель в дневных ATR"
```

---

## Task 4: Выходы SL → TP → RSI и удержание через ночь

**Files:**
- Modify: `internal/service/trading_strategy/rsi_pullback/strategy/core/core.go` (`manage`; удаление `isDayEnd`, `crossedIntoNewDay`, `barsHeld`, `holdUnknown`, `holdLabel`, `heldLabel`, `barSpanMinutes`, `defaultBarSpanMin`, `MaxHoldBars`, `DayEndMin`)
- Test: `internal/service/trading_strategy/rsi_pullback/strategy/core/core_test.go`

**Interfaces:**
- Produces: `func (s *Strategy) manage(md strategy.MarketData, sig model.Signal) model.Signal` — выходы в приоритете `SL` → `TP` → `RSI`, без ограничения по времени удержания.

- [ ] **Step 1: Написать падающие тесты**

Удалить из `core_test.go` тесты, проверяющие снятые правила: `TestExitTimeStop`, `TestExitTimeStopDisabledAtZero`, `TestExitTimeStopSilentOnUnknownEntryTime`, `TestExitEndOfDay`, `TestExitEndOfDayOnTruncatedSession`, `TestExitHoldsThroughTheEveningSession`, `TestExplainLabelsUnknownHold`. Добавить:

```go
func TestExitTakeProfit(t *testing.T) {
	s := NewWithParams("TEST", DefaultParams())
	start := time.Date(2026, 3, 2, 7, 0, 0, 0, msk)
	md := barSeries([]float64{100, 100, 100, 100}, start)
	i := len(md.Closes) - 1
	md.Highs[i] = 110
	md.Position = &strategy.Position{
		PurchasePrice: 100, Quantity: 1, StopLoss: 90, TakeProfit: 106,
		EntryTime: md.Times[0],
	}
	sig := s.Decide(md)
	if sig.Kind != model.SignalSell || sig.Reason != "TP" {
		t.Fatalf("Kind/Reason = %v/%q, want Sell/TP", sig.Kind, sig.Reason)
	}
	if sig.TakeProfit != 106 {
		t.Fatalf("TakeProfit = %.4f, want 106 (уровень из позиции)", sig.TakeProfit)
	}
}

func TestExitStopWinsOverTakeProfitOnTheSameBar(t *testing.T) {
	s := NewWithParams("TEST", DefaultParams())
	start := time.Date(2026, 3, 2, 7, 0, 0, 0, msk)
	md := barSeries([]float64{100, 100, 100, 100}, start)
	i := len(md.Closes) - 1
	md.Highs[i], md.Lows[i] = 110, 90 // бар задевает и цель, и стоп
	md.Position = &strategy.Position{
		PurchasePrice: 100, Quantity: 1, StopLoss: 95, TakeProfit: 105,
		EntryTime: md.Times[0],
	}
	if sig := s.Decide(md); sig.Reason != "SL" {
		t.Fatalf("Reason = %q, want SL: внутрибарный порядок неизвестен, побеждает худший исход", sig.Reason)
	}
}

func TestExitTakeProfitDisabledAtZero(t *testing.T) {
	p := DefaultParams()
	p.TPDailyATR = 0
	s := NewWithParams("TEST", p)
	start := time.Date(2026, 3, 2, 7, 0, 0, 0, msk)
	md := barSeries([]float64{100, 100, 100, 100}, start)
	md.Highs[len(md.Highs)-1] = 500
	md.Position = &strategy.Position{
		PurchasePrice: 100, Quantity: 1, StopLoss: 90, TakeProfit: 0,
		EntryTime: md.Times[0],
	}
	if sig := s.Decide(md); sig.Kind == model.SignalSell && sig.Reason == "TP" {
		t.Fatal("цель выключена (TakeProfit=0), выхода по TP быть не должно")
	}
}

func TestPositionSurvivesOvernight(t *testing.T) {
	s := NewWithParams("TEST", DefaultParams())
	// Понедельник 22:30 -> вторник 07:00: смена календарного дня и разрыв в серии.
	closes := []float64{100, 100, 100, 100}
	md := barSeries(closes, time.Date(2026, 3, 2, 21, 30, 0, 0, msk))
	md.Times[len(md.Times)-1] = time.Date(2026, 3, 3, 7, 0, 0, 0, msk)
	md.Position = &strategy.Position{
		PurchasePrice: 100, Quantity: 1, StopLoss: 90, TakeProfit: 110,
		EntryTime: md.Times[0],
	}
	if sig := s.Decide(md); sig.Kind == model.SignalSell {
		t.Fatalf("позиция закрыта на переходе через ночь (%q), а перенос разрешён", sig.Reason)
	}
}

func TestPositionSurvivesWeekend(t *testing.T) {
	s := NewWithParams("TEST", DefaultParams())
	// Пятница -> понедельник.
	md := barSeries([]float64{100, 100, 100, 100}, time.Date(2026, 3, 6, 21, 30, 0, 0, msk))
	md.Times[len(md.Times)-1] = time.Date(2026, 3, 9, 7, 0, 0, 0, msk)
	md.Position = &strategy.Position{
		PurchasePrice: 100, Quantity: 1, StopLoss: 90, TakeProfit: 110,
		EntryTime: md.Times[0],
	}
	if sig := s.Decide(md); sig.Kind == model.SignalSell {
		t.Fatalf("позиция закрыта на переходе через выходные (%q)", sig.Reason)
	}
}
```

- [ ] **Step 2: Убедиться, что тесты падают**

Run: `go test ./internal/service/trading_strategy/rsi_pullback/... -run 'TestExit|TestPositionSurvives' -v`
Expected: FAIL — `TestExitTakeProfit` не видит выхода `TP`, `TestPositionSurvives*` ловят `EOD`.

- [ ] **Step 3: Переписать `manage`**

```go
// manage handles an open long, exiting in precedence SL -> TP -> RSI. Both levels are read
// from the position (frozen at entry), never recomputed: the stop and target the trade was
// opened with are the ones it dies by. SL fills at the stop level and TP at the target (the
// engine handles that via model.IsStopReason and the "TP" reason); RSI fills at the bar close.
// There is no time stop and no end-of-day close — the position is held until one of the three
// exits fires, across nights and weekends.
func (s *Strategy) manage(md strategy.MarketData, sig model.Signal) model.Signal {
	pos := md.Position
	n := len(md.Closes)
	if pos == nil || n < 2 || len(md.Highs) != n || len(md.Lows) != n {
		return sig
	}
	i := n - 1
	high, low, closeP := md.Highs[i], md.Lows[i], md.Closes[i]

	// 1. hard stop. It wins a same-bar tie with the target: the intrabar order of the two
	// touches is unknowable from OHLC, and assuming the worse of the two is the honest choice.
	if pos.StopLoss > 0 && low <= pos.StopLoss {
		sig.Kind, sig.Reason = model.SignalSell, "SL"
		sig.StopLoss = pos.StopLoss
		sig.ExitReason = fmt.Sprintf("SL: low %.4f ≤ стоп %.4f (вход %.4f)", low, pos.StopLoss, pos.PurchasePrice)
		return sig
	}
	// 2. fixed target.
	if pos.TakeProfit > 0 && high >= pos.TakeProfit {
		sig.Kind, sig.Reason = model.SignalSell, "TP"
		sig.TakeProfit = pos.TakeProfit
		sig.ExitReason = fmt.Sprintf("TP: high %.4f ≥ цель %.4f (вход %.4f)", high, pos.TakeProfit, pos.PurchasePrice)
		return sig
	}
	// 3. RSI crosses UP through the upper band — the bounce reached overbought.
	rsi := indicators.RSISeries(md.Closes, s.p.RSIPeriod)
	if len(rsi) == n && crossedUp(rsi, i, s.p.RSIUpper) {
		sig.Kind, sig.Reason = model.SignalSell, "RSI"
		sig.RSI = rsi[i]
		sig.ExitReason = fmt.Sprintf("RSI: RSI(%d) пересёк %.0f снизу вверх (%.1f), выход по %.4f (вход %.4f)",
			s.p.RSIPeriod, s.p.RSIUpper, rsi[i], closeP, pos.PurchasePrice)
	}
	return sig
}
```

- [ ] **Step 4: Удалить мёртвый код**

Удалить целиком: `isDayEnd`, `crossedIntoNewDay`, `barSpanMinutes`, `barsHeld`, `holdUnknown`, `holdLabel`, `heldLabel`, константу `defaultBarSpanMin`. Из `Explain` убрать строки про конец дня и удержание (полностью `Explain` переписывается в Task 6, здесь достаточно убрать обращения к удалённым функциям). Проверить импорты: `math` и `sort` больше не нужны — удалить их из блока import.

- [ ] **Step 5: Прогнать тесты**

Run: `go test ./internal/service/trading_strategy/rsi_pullback/... -count=1 && go vet ./internal/service/trading_strategy/rsi_pullback/...`
Expected: PASS, vet чистый (никаких неиспользуемых импортов и функций).

- [ ] **Step 6: Коммит**

```bash
git add internal/service/trading_strategy/rsi_pullback/strategy/core/
git commit -m "feat(rsi_pullback): выходы SL/TP/RSI, удержание через ночь, снятие EOD и тайм-стопа"
```

---

## Task 5: Слотовый фильтр фона объёмов

**Files:**
- Modify: `internal/service/trading_strategy/rsi_pullback/strategy/core/core.go` (`Params`, `DefaultParams`, `Lookback`, `enter`, новые хелперы)
- Test: `internal/service/trading_strategy/rsi_pullback/strategy/core/core_test.go`

**Interfaces:**
- Produces: `Params.UseVolume int`, `Params.VolBaseDays int`, `Params.VolLookbackBars int`, `Params.VolMult float64`.
- Produces: `func volumeBaseline(vols []int64, times []time.Time, baseDays int) (bySlot map[int]float64, flat float64, ok bool)`.
- Produces: `func (s *Strategy) volumeOK(md strategy.MarketData) bool` — вызывается шестым гейтом в `enter`.

- [ ] **Step 1: Написать падающие тесты**

```go
// volSeries builds `days` weekday days of `perDay` 30-minute bars starting at 07:00 MSK, with
// a U-shaped volume profile: the first bar of each day is `openVol`, the rest are `midVol`.
// The last bar of the last day is the "current" bar.
func volSeries(firstDay time.Time, days, perDay int, openVol, midVol int64) strategy.MarketData {
	var md strategy.MarketData
	d := firstDay
	for added := 0; added < days; {
		if d.Weekday() == time.Saturday || d.Weekday() == time.Sunday {
			d = d.AddDate(0, 0, 1)
			continue
		}
		for b := 0; b < perDay; b++ {
			t := time.Date(d.Year(), d.Month(), d.Day(), 7, 0, 0, 0, msk).
				Add(time.Duration(b) * 30 * time.Minute)
			v := midVol
			if b == 0 {
				v = openVol
			}
			md.Times = append(md.Times, t)
			md.Volumes = append(md.Volumes, v)
			md.Closes = append(md.Closes, 100)
			md.Highs = append(md.Highs, 100.3)
			md.Lows = append(md.Lows, 99.7)
		}
		added++
		d = d.AddDate(0, 0, 1)
	}
	md.Price = 100
	return md
}

func TestVolumeGateComparesAgainstItsOwnSlot(t *testing.T) {
	p := DefaultParams()
	p.VolBaseDays, p.VolLookbackBars, p.VolMult = 5, 3, 1.5
	s := NewWithParams("TEST", p)

	// Профиль: открытие 10000, середина дня 1000. Текущий бар — середина дня с объёмом 2000:
	// это вдвое выше своего слота (1000), но впятеро ниже утреннего.
	md := volSeries(time.Date(2026, 3, 2, 0, 0, 0, 0, msk), 6, 8, 10000, 1000)
	last := len(md.Volumes) - 1
	md.Volumes[last] = 2000
	md.Volumes[last-1], md.Volumes[last-2] = 1000, 1000

	// Слотовая база для 10:30 равна 1000, значит 2000 — это 2.0x, гейт открыт.
	// На плоской базе тот же бар НЕ прошёл бы: она равна (10000 + 7*1000)/8 = 2125,
	// и 2000/2125 = 0.94 < 1.5. Именно это различает две реализации.
	if !s.volumeOK(md) {
		t.Fatal("бар вдвое выше своего слота обязан открывать гейт (плоская база дала бы отказ)")
	}

	md.Volumes[last] = 1000
	if s.volumeOK(md) {
		t.Fatal("бар ровно на уровне своего слота не должен открывать гейт при VolMult=1.5")
	}
}

func TestVolumeGateAnyOfTheLastThreeBars(t *testing.T) {
	p := DefaultParams()
	p.VolBaseDays, p.VolLookbackBars, p.VolMult = 5, 3, 1.5
	s := NewWithParams("TEST", p)
	md := volSeries(time.Date(2026, 3, 2, 0, 0, 0, 0, msk), 6, 8, 10000, 1000)
	last := len(md.Volumes) - 1

	md.Volumes[last], md.Volumes[last-1], md.Volumes[last-2] = 1000, 1000, 1000
	if s.volumeOK(md) {
		t.Fatal("три тихих бара не должны открывать гейт")
	}
	md.Volumes[last-2] = 2000 // всплеск на третьем баре назад
	if !s.volumeOK(md) {
		t.Fatal("всплеск на любом из последних трёх баров обязан открывать гейт")
	}
	md.Volumes[last-2] = 1000
	md.Volumes[last-3] = 5000 // четвёртый бар назад — уже вне окна
	if s.volumeOK(md) {
		t.Fatal("всплеск за пределами VolLookbackBars не должен открывать гейт")
	}
}

func TestVolumeGateIgnoresWeekendBars(t *testing.T) {
	p := DefaultParams()
	p.VolBaseDays, p.VolLookbackBars, p.VolMult = 5, 3, 1.5
	s := NewWithParams("TEST", p)
	const perDay = 8
	md := volSeries(time.Date(2026, 3, 2, 0, 0, 0, 0, msk), 6, perDay, 10000, 1000)

	// Текущий бар — 1.4x своего слота: ниже порога 1.5, гейт закрыт.
	last := len(md.Volumes) - 1
	md.Volumes[last] = 1400
	md.Volumes[last-1], md.Volumes[last-2] = 1000, 1000
	if s.volumeOK(md) {
		t.Fatal("1.4x от слотовой базы не должно открывать гейт при VolMult=1.5")
	}

	// Вклеиваем тонкую субботнюю сессию (те же слоты, объём 50) перед последним днём.
	// Если бы выходные попадали в базу, слотовая база упала бы с 1000 до (5*1000+50)/6 = 842,
	// и тот же бар дал бы 1400/842 = 1.66 ≥ 1.5 — гейт бы открылся. Он открыться не должен.
	sat := time.Date(2026, 3, 7, 7, 0, 0, 0, msk)
	insertAt := len(md.Times) - perDay
	for b := 0; b < perDay; b++ {
		bt := sat.Add(time.Duration(b) * 30 * time.Minute)
		md.Times = append(md.Times[:insertAt], append([]time.Time{bt}, md.Times[insertAt:]...)...)
		md.Volumes = append(md.Volumes[:insertAt], append([]int64{50}, md.Volumes[insertAt:]...)...)
		md.Closes = append(md.Closes[:insertAt], append([]float64{100}, md.Closes[insertAt:]...)...)
		md.Highs = append(md.Highs[:insertAt], append([]float64{100.3}, md.Highs[insertAt:]...)...)
		md.Lows = append(md.Lows[:insertAt], append([]float64{99.7}, md.Lows[insertAt:]...)...)
		insertAt++
	}
	if s.volumeOK(md) {
		t.Fatal("выходная сессия занизила базу: выходные обязаны выпадать из расчёта")
	}

	// И выходные не должны занимать места в окне последних трёх баров: всплеск на третьем
	// БУДНЕМ баре назад обязан открывать гейт.
	md.Volumes[len(md.Volumes)-1] = 1000
	md.Volumes[len(md.Volumes)-3] = 2000
	if !s.volumeOK(md) {
		t.Fatal("выходные бары не должны вытеснять будние из окна VolLookbackBars")
	}
}

func TestVolumeGateDegradations(t *testing.T) {
	base := DefaultParams()
	base.VolBaseDays, base.VolLookbackBars, base.VolMult = 5, 3, 1.5
	quiet := volSeries(time.Date(2026, 3, 2, 0, 0, 0, 0, msk), 6, 8, 10000, 1000)

	off := base
	off.UseVolume = 0
	if !NewWithParams("TEST", off).volumeOK(quiet) {
		t.Fatal("выключенный гейт обязан пропускать")
	}

	for name, mutate := range map[string]func(p *Params){
		"VolBaseDays=0":     func(p *Params) { p.VolBaseDays = 0 },
		"VolLookbackBars=0": func(p *Params) { p.VolLookbackBars = 0 },
		"VolMult=0":         func(p *Params) { p.VolMult = 0 },
	} {
		p := base
		mutate(&p)
		if !NewWithParams("TEST", p).volumeOK(quiet) {
			t.Fatalf("%s: сломанная конфигурация обязана пропускать", name)
		}
	}

	s := NewWithParams("TEST", base)
	if !s.volumeOK(strategy.MarketData{}) {
		t.Fatal("без объёмов гейт обязан пропускать")
	}
	noTimes := quiet
	noTimes.Times = nil
	if !s.volumeOK(noTimes) {
		t.Fatal("без времён гейт обязан пропускать")
	}
	// Только один день истории: базы нет вовсе.
	oneDay := volSeries(time.Date(2026, 3, 2, 0, 0, 0, 0, msk), 1, 8, 10000, 1000)
	if !s.volumeOK(oneDay) {
		t.Fatal("без завершённых дней в базе гейт обязан пропускать")
	}
}

func TestLookbackCoversVolumeBaseline(t *testing.T) {
	p := DefaultParams()
	p.UseVolume, p.VolBaseDays = 1, 10
	if got := NewWithParams("TEST", p).Lookback(); got < 11*maxBarsPerDay {
		t.Fatalf("Lookback = %d, want >= %d (11 дней по %d баров)", got, 11*maxBarsPerDay, maxBarsPerDay)
	}
	p.UseVolume = 0
	if got := NewWithParams("TEST", p).Lookback(); got >= 11*maxBarsPerDay {
		t.Fatalf("Lookback = %d: с выключенным гейтом окно не должно раздуваться", got)
	}
}
```

- [ ] **Step 2: Убедиться, что тесты падают**

Run: `go test ./internal/service/trading_strategy/rsi_pullback/... -run 'TestVolume|TestLookback' -v`
Expected: FAIL — `s.volumeOK undefined`, `maxBarsPerDay undefined`.

- [ ] **Step 3: Добавить параметры**

В `Params` после `TPDailyATR`:

```go
	UseVolume       int     // 1 arms the volume-background gate; any other value disables it (grid; default 1)
	VolBaseDays     int     // completed WEEKDAY days behind the baseline (grid; default 5)
	VolLookbackBars int     // how many recent weekday bars may open the gate (grid; default 3)
	VolMult         float64 // a bar opens the gate at volume >= VolMult * its slot baseline (grid; default 1.5)
```

В `DefaultParams()`: `UseVolume: 1, VolBaseDays: 5, VolLookbackBars: 3, VolMult: 1.5,`.

- [ ] **Step 4: Реализовать гейт**

```go
// maxBarsPerDay caps how many 30-minute bars a single calendar day can contribute (24h / 30m).
// It deliberately oversizes the window: the volume baseline needs whole days, and an
// undersized window would silently shrink the baseline instead of failing loudly.
const maxBarsPerDay = 48

// dayOf returns midnight of t's MSK calendar day — the grouping key for baseline days.
func dayOf(t time.Time) time.Time {
	tl := t.In(mskLoc)
	return time.Date(tl.Year(), tl.Month(), tl.Day(), 0, 0, 0, 0, mskLoc)
}

// slotOf returns the bar's intraday slot: minutes from MSK midnight. Bars sharing a slot are
// the same half-hour of the trading day across different days.
func slotOf(t time.Time) int {
	tl := t.In(mskLoc)
	return tl.Hour()*60 + tl.Minute()
}

// volumeBaseline builds the per-slot average volume over the last baseDays COMPLETED WEEKDAY
// days present in the window, plus a flat average over the same bars as a fallback for slots
// with fewer than two observations. The current day is excluded entirely, so the bars being
// judged never contaminate their own baseline and the baseline does not drift from bar to bar
// within a day. Weekend sessions are excluded: on MOEX they carry 8-17x less volume and would
// drag the baseline down, turning the gate into a free pass. Non-positive volumes are ignored.
// ok is false when no usable bar was found at all.
func volumeBaseline(vols []int64, times []time.Time, baseDays int) (bySlot map[int]float64, flat float64, ok bool) {
	n := len(vols)
	if n == 0 || len(times) != n || baseDays <= 0 {
		return nil, 0, false
	}
	current := dayOf(times[n-1])
	sums := make(map[int]float64)
	counts := make(map[int]int)
	var flatSum float64
	var flatCount int
	var lastDay time.Time
	days := 0
	for i := n - 1; i >= 0; i-- {
		t := times[i]
		if isWeekend(t.In(mskLoc)) {
			continue
		}
		d := dayOf(t)
		if !d.Before(current) {
			continue
		}
		if !d.Equal(lastDay) {
			if days == baseDays {
				break
			}
			days++
			lastDay = d
		}
		if vols[i] <= 0 {
			continue
		}
		sl := slotOf(t)
		sums[sl] += float64(vols[i])
		counts[sl]++
		flatSum += float64(vols[i])
		flatCount++
	}
	if flatCount == 0 {
		return nil, 0, false
	}
	bySlot = make(map[int]float64, len(sums))
	for sl, c := range counts {
		if c >= 2 {
			bySlot[sl] = sums[sl] / float64(c)
		}
	}
	return bySlot, flatSum / float64(flatCount), true
}

// volumeOK reports whether the recent tape is busier than usual FOR THIS TIME OF DAY: at least
// one of the last VolLookbackBars weekday bars must carry VolMult times the average volume of
// its own intraday slot. Comparing against a slot rather than a flat average matters because
// 30-minute volume is U-shaped — an opening bar dwarfs a midday one — so a flat baseline would
// measure the clock instead of the activity. The gate degrades to "allow" whenever it is
// disabled, misconfigured or unsupported by the data: missing volume must never block an entry.
func (s *Strategy) volumeOK(md strategy.MarketData) bool {
	if s.p.UseVolume != 1 || s.p.VolBaseDays <= 0 || s.p.VolLookbackBars <= 0 || s.p.VolMult <= 0 {
		return true
	}
	n := len(md.Volumes)
	if n == 0 || len(md.Times) != n {
		return true
	}
	bySlot, flat, ok := volumeBaseline(md.Volumes, md.Times, s.p.VolBaseDays)
	if !ok || flat <= 0 {
		return true
	}
	checked := 0
	for i := n - 1; i >= 0 && checked < s.p.VolLookbackBars; i-- {
		if isWeekend(md.Times[i].In(mskLoc)) {
			continue
		}
		checked++
		if md.Volumes[i] <= 0 {
			continue
		}
		base, hasSlot := bySlot[slotOf(md.Times[i])]
		if !hasSlot || base <= 0 {
			base = flat
		}
		if float64(md.Volumes[i]) >= base*s.p.VolMult {
			return true
		}
	}
	// No weekday bar to judge at all — allow, same as any other missing-data case.
	return checked == 0
}
```

- [ ] **Step 5: Подключить гейт и расширить `Lookback`**

В `enter` после проверки состояния дня:

```go
	// 6. the tape must be busier than usual for this time of day.
	if !s.volumeOK(md) {
		return sig
	}
```

Заменить `Lookback`:

```go
// Lookback sizes the candle window the engine feeds Decide on every bar. It must cover the
// hungriest indicator with room to converge: ema.Compute seeds on an SMA over the first
// `period` closes, so a window of exactly `period` bars yields a bare seed — and a window
// SHORTER than the period yields an all-zero series, which silently fails the trend gate for
// the whole run instead of erroring. Doubling the largest period leaves as many recursion steps
// as the seed span; the +20 covers the two-bar cross lookups. When the volume gate is armed the
// window must additionally hold VolBaseDays completed days plus the current one, which on
// 30-minute bars dominates everything else.
func (s *Strategy) Lookback() int {
	need := max(s.p.EMASlow, s.p.EMAFast, s.p.RSIPeriod)
	vol := 0
	if s.p.UseVolume == 1 && s.p.VolBaseDays > 0 {
		vol = (s.p.VolBaseDays + 1) * maxBarsPerDay
	}
	return max(minLookback, 2*need+20, vol)
}
```

Обновить существующий `TestLookbackCoversSlowEMA`, если он пинует точное значение: с дефолтными параметрами (`UseVolume=1`, `VolBaseDays=5`) окно теперь `6*48 = 288`.

- [ ] **Step 6: Прогнать тесты**

Run: `go test ./internal/service/trading_strategy/rsi_pullback/... -count=1 -race`
Expected: PASS.

- [ ] **Step 7: Коммит**

```bash
git add internal/service/trading_strategy/rsi_pullback/strategy/core/
git commit -m "feat(rsi_pullback): слотовый гейт фона объёмов с исключением выходных"
```

---

## Task 6: `Explain`, границы, сетка калибровки и документация

**Files:**
- Modify: `internal/service/trading_strategy/rsi_pullback/strategy/core/core.go` (`Explain`)
- Modify: `internal/service/trading_strategy/rsi_pullback/strategy/core/core_test.go`
- Rewrite: `data/params/rsi_pullback/grid.json`
- Rewrite: `internal/service/backtest/rsi_pullback_grid_test.go`
- Rewrite: `docs/rsi_pullback/strategy.md`
- Modify: `CLAUDE.md` (строка 23, описание `rsi_pullback`)

**Interfaces:**
- Consumes: всё из Task 2-5.

- [ ] **Step 1: Написать падающие тесты**

```go
func TestExplainMentionsEveryGate(t *testing.T) {
	s := NewWithParams("TEST", DefaultParams())
	start := time.Date(2026, 3, 2, 7, 0, 0, 0, msk)
	md := withDay(barSeries(pullbackCloses(), start), 10.0, 101, 100)
	got := s.Explain(md)
	for _, want := range []string{"сессия", "RSI", "EMA", "дневной ATR", "состояние дня", "фон объёмов", "стоп", "цель"} {
		if !strings.Contains(got, want) {
			t.Fatalf("Explain не упоминает %q:\n%s", want, got)
		}
	}
}

func TestCrossHelpersBoundaries(t *testing.T) {
	// crossedDown: предыдущий бар ровно НА уровне считается крестом, текущий ровно НА уровне — нет.
	if !crossedDown([]float64{15, 14}, 1, 15) {
		t.Fatal("crossedDown: пред=уровень, тек<уровень — крест должен считаться")
	}
	if crossedDown([]float64{16, 15}, 1, 15) {
		t.Fatal("crossedDown: тек ровно на уровне — креста ещё нет")
	}
	if crossedDown([]float64{14, 13}, 1, 15) {
		t.Fatal("crossedDown: пред уже под уровнем — это состояние, не событие")
	}
	// crossedUp — зеркально.
	if !crossedUp([]float64{70, 71}, 1, 70) {
		t.Fatal("crossedUp: пред=уровень, тек>уровень — крест должен считаться")
	}
	if crossedUp([]float64{69, 70}, 1, 70) {
		t.Fatal("crossedUp: тек ровно на уровне — креста ещё нет")
	}
	if crossedUp([]float64{71, 72}, 1, 70) {
		t.Fatal("crossedUp: пред уже над уровнем — это состояние, не событие")
	}
	// Прогревочный ноль RSI не должен изображать крест ни в одну сторону.
	if crossedDown([]float64{0, 14}, 1, 15) || crossedUp([]float64{0, 71}, 1, 70) {
		t.Fatal("прогревочный ноль не может быть началом креста")
	}
}
```

Плюс расширить существующий `TestNoLookaheadAcrossWindowCuts`: обрезать вместе с барами и дневные серии, и `TodayHigh/TodayLow`, убедившись, что вердикт на общем баре совпадает.

- [ ] **Step 2: Убедиться, что тесты падают**

Run: `go test ./internal/service/trading_strategy/rsi_pullback/... -run 'TestExplain|TestCrossHelpers|TestNoLookahead' -v`
Expected: FAIL — `Explain` ещё не упоминает новые гейты.

- [ ] **Step 3: Переписать `Explain`**

```go
// Explain returns a gate-by-gate verdict for one bar, consumed by the engine's Trace
// (-explain). It recomputes the same values enter()/manage() do and reports each gate.
func (s *Strategy) Explain(md strategy.MarketData) string {
	var sb strings.Builder
	n := len(md.Closes)
	if n < 2 || len(md.Highs) != n || len(md.Lows) != n {
		sb.WriteString("недостаточно свечей\n")
		return sb.String()
	}
	i := n - 1
	barT := s.barTime(md)
	fmt.Fprintf(&sb, "сессия: вход разрешён? %v (бар %v)\n", s.inSession(barT), barT)

	rsi := indicators.RSISeries(md.Closes, s.p.RSIPeriod)
	if len(rsi) == n {
		fmt.Fprintf(&sb, "RSI(%d) пред %.1f тек %.1f; вход-крест вниз через %.0f? %v; выход-крест вверх через %.0f? %v\n",
			s.p.RSIPeriod, rsi[i-1], rsi[i], s.p.RSILower, crossedDown(rsi, i, s.p.RSILower),
			s.p.RSIUpper, crossedUp(rsi, i, s.p.RSIUpper))
	} else {
		sb.WriteString("RSI: недостаточно истории\n")
	}

	if fast, slow, ok := s.emaPair(md.Closes); ok {
		fmt.Fprintf(&sb, "EMA(%d) %.4f vs EMA(%d) %.4f: тренд вверх? %v\n",
			s.p.EMAFast, fast[i], s.p.EMASlow, slow[i], fast[i] > slow[i])
	} else {
		sb.WriteString("EMA: не прогрето\n")
	}

	atr := s.dailyATR(md)
	if atr > 0 {
		fmt.Fprintf(&sb, "дневной ATR(%d) по будням: %.4f\n", s.p.DailyATRPeriod, atr)
	} else {
		sb.WriteString("дневной ATR: не посчитан — вход невозможен\n")
	}

	switch {
	case s.p.UseDayATRGate != 1:
		sb.WriteString("состояние дня: гейт выключен (UseDayATRGate=0)\n")
	case atr <= 0 || md.TodayHigh <= 0 || md.TodayLow <= 0:
		sb.WriteString("состояние дня: нет данных, гейт пропускает\n")
	default:
		used := (md.TodayHigh - md.TodayLow) / atr
		fmt.Fprintf(&sb, "состояние дня: пройдено %.2f ATR (свежий ≤%.2f, исчерпан ≥%.2f); пройден? %v\n",
			used, s.p.FreshDayATR, s.p.SpentDayATR, s.dayStateOK(md, atr))
	}

	if s.p.UseVolume != 1 {
		sb.WriteString("фон объёмов: гейт выключен (UseVolume=0)\n")
	} else {
		fmt.Fprintf(&sb, "фон объёмов: хотя бы один из %d баров ≥ %.2f× своего слота за %d дней? %v\n",
			s.p.VolLookbackBars, s.p.VolMult, s.p.VolBaseDays, s.volumeOK(md))
	}

	if s.p.StopDailyATR > 0 && atr > 0 {
		fmt.Fprintf(&sb, "стоп: вход − %.2f×ATR (%.4f)\n", s.p.StopDailyATR, s.p.StopDailyATR*atr)
	} else {
		sb.WriteString("стоп: выключен\n")
	}
	if s.p.TPDailyATR > 0 && atr > 0 {
		fmt.Fprintf(&sb, "цель: вход + %.2f×ATR (%.4f)\n", s.p.TPDailyATR, s.p.TPDailyATR*atr)
	} else {
		sb.WriteString("цель: выключена\n")
	}
	return sb.String()
}
```

- [ ] **Step 4: Переписать сетку**

`data/params/rsi_pullback/grid.json`:

```json
{
  "_comment": "rsi_pullback phased grid. Phase 1 (entry) sweeps the pullback trigger: RSI length and the lower band a DOWNWARD cross of which opens the trade. Phase 2 (trend) sweeps the EMA pair that must confirm an uptrend. Phase 3 (day) sweeps the two-sided day gate: entries are allowed either while the day is barely started (range <= FreshDayATR*dailyATR) or after it is spent (range >= SpentDayATR*dailyATR), never in between; UseDayATRGate=0 turns the whole gate off, and at 0 the Fresh/Spent values collapse to one identical control config, so duplicate leaderboard rows are expected. Phase 4 (volume) sweeps the slot-based volume-background gate, UseVolume=0 = off, with the same duplicate-row caveat. Phase 5 (risk) sweeps the stop and the target, both in daily-ATR units; TPDailyATR values above StopDailyATR are included on purpose so calibration can test whether the 0.6:1 reward-to-risk actually pays. StopDailyATR=0 is deliberately ABSENT: multi-day holding without a stop is not an option calibration may choose. Phase 6 (exit) sweeps the RSI band whose upward cross closes the trade. The session bounds and the daily ATR period are fixed by the strategy definition and NOT swept. RunPhases expands each phase over the previous phase's keepTop seeds: 9 + 6x9 + 5x12 + 5x12 + 4x9 + 4x3 = up to 231 backtest evaluations (fewer when minTrades floors a phase's ranking below keepTop). Judge on pooled OOS profit factor from a walk-forward, never on the in-sample best.",
  "phases": [
    {
      "name": "entry",
      "keepTop": 6,
      "grid": {
        "RSIPeriod": [4, 5, 6],
        "RSILower": [10, 15, 20]
      }
    },
    {
      "name": "trend",
      "keepTop": 5,
      "grid": {
        "EMAFast": [5, 10, 20],
        "EMASlow": [50, 100, 200]
      }
    },
    {
      "name": "day",
      "keepTop": 5,
      "grid": {
        "UseDayATRGate": [0, 1],
        "FreshDayATR": [0.3, 0.5],
        "SpentDayATR": [0.8, 1.0, 1.3]
      }
    },
    {
      "name": "volume",
      "keepTop": 4,
      "grid": {
        "UseVolume": [0, 1],
        "VolMult": [1.2, 1.5, 2.0],
        "VolBaseDays": [5, 10]
      }
    },
    {
      "name": "risk",
      "keepTop": 4,
      "grid": {
        "StopDailyATR": [0.8, 1.0, 1.3],
        "TPDailyATR": [0.6, 1.0, 1.5]
      }
    },
    {
      "name": "exit",
      "grid": {
        "RSIUpper": [60, 70, 80]
      }
    }
  ]
}
```

- [ ] **Step 5: Переписать пин-тест сетки**

Заменить `TestRSIPullbackGridHasTimeStopOffPoint` и `TestRSIPullbackGridCombos` в `internal/service/backtest/rsi_pullback_grid_test.go` на:

```go
// TestRSIPullbackGridControlPoints pins the deliberate on/off points: both optional gates must
// be sweepable to "off", and the stop must NOT be — calibration may never choose to hold a
// multi-day position without protection.
func TestRSIPullbackGridControlPoints(t *testing.T) {
	var sawDayOff, sawVolumeOff, sawStop, sawTPAboveStop bool
	maxStop := 0.0
	for _, ph := range rsiPullbackGrid(t) {
		for _, v := range ph.Grid["UseDayATRGate"] {
			if v == 0 {
				sawDayOff = true
			}
		}
		for _, v := range ph.Grid["UseVolume"] {
			if v == 0 {
				sawVolumeOff = true
			}
		}
		for _, v := range ph.Grid["StopDailyATR"] {
			sawStop = true
			if v == 0 {
				t.Fatal("StopDailyATR=0 is in the grid: calibration must not be able to disable the stop")
			}
			if v > maxStop {
				maxStop = v
			}
		}
	}
	for _, ph := range rsiPullbackGrid(t) {
		for _, v := range ph.Grid["TPDailyATR"] {
			if v > maxStop {
				sawTPAboveStop = true
			}
		}
	}
	if !sawDayOff {
		t.Fatal("no UseDayATRGate=0 control point in the grid")
	}
	if !sawVolumeOff {
		t.Fatal("no UseVolume=0 control point in the grid")
	}
	if !sawStop {
		t.Fatal("the grid never sweeps StopDailyATR")
	}
	if !sawTPAboveStop {
		t.Fatal("the grid never tests a target above the stop: the 0.6:1 asymmetry stays untested")
	}
}

// TestRSIPullbackGridEvaluationCost pins the real cost of a phased calibration. RunPhases
// expands every phase over the previous phase's keepTop seeds, so the number of backtest runs
// is NOT the sum of the grid sizes — an earlier revision of this file understated it fourfold.
func TestRSIPullbackGridEvaluationCost(t *testing.T) {
	phases := rsiPullbackGrid(t)
	seeds := 1
	total := 0
	for _, ph := range phases {
		n := 1
		for _, values := range ph.Grid {
			n *= len(values)
		}
		total += seeds * n
		if ph.KeepTop > 0 {
			seeds = ph.KeepTop
		}
	}
	if total != 231 {
		t.Fatalf("phased calibration costs %d evaluations, want the documented 231", total)
	}
}
```

- [ ] **Step 6: Прогнать все тесты**

Run: `go test ./internal/... ./pkg/... -count=1 -race`
Expected: PASS.

- [ ] **Step 7: Переписать документацию**

Переписать `docs/rsi_pullback/strategy.md` целиком под новые правила. Обязательные разделы: (1) идея и в чём отличие от `rsi_ema`; (2) все параметры с дефолтами; (3) шесть гейтов входа с формулами; (4) три выхода и правило тай-брейка SL/TP; (5) откуда берётся дневной ATR и почему выброшены выходные (с числами занижения 9-16% из спеки); (6) как считается фон объёмов и почему база слотовая; (7) `Lookback` и его зависимость от `VolBaseDays`; (8) команды калибровки с честной ценой в 231 прогон и планкой pooled OOS PF ≥ 1.5; (9) на что смотреть в отчёте: среднюю длительность сделки (тайм-стопа нет) и раздельную прибыльность двух веток гейта дня.

Команда калибровки для раздела 8:

```bash
go run ./cmd/backtest -ticker GAZP -strategy rsi_pullback -interval Minutes30 \
  -calibrate data/params/rsi_pullback/grid.json -out ./reports/GAZP \
  -months 24 -min-trades 20 -test-months 6 -metric profit_factor
```

В `CLAUDE.md` заменить описание `rsi_pullback` на: `rsi_pullback` — backtest-only 30m long multi-day RSI pullback strategy: daily-ATR stop/target, two-sided day-state gate, slot-based volume filter (`-strategy rsi_pullback`, docs: `docs/rsi_pullback/strategy.md`).

- [ ] **Step 8: Полный прогон CI**

Run: `./bin/mage ci`
Expected: EXIT=0 — линтер, `go test -race ./...` и проверка дрейфа моков зелёные.

- [ ] **Step 9: Коммит**

```bash
git add internal/ data/params/rsi_pullback/grid.json docs/rsi_pullback/strategy.md CLAUDE.md
git commit -m "feat(rsi_pullback): Explain, сетка калибровки и документация под дневной ATR"
```

---

## Проверка перед завершением

- [ ] `./bin/mage ci` зелёный на финальном дереве.
- [ ] `grep -rn "MaxHoldBars\|DayEndMin\|StopATR\|isDayEnd\|crossedIntoNewDay" internal/service/trading_strategy/rsi_pullback/` не находит ничего.
- [ ] `go run ./cmd/backtest -ticker GAZP -strategy rsi_pullback -interval Minutes30 -months 6` отрабатывает без ошибок и печатает сделки.
- [ ] Число сделок на 24 месяцах по GAZP — порядка 20-35. Если их единицы, что-то в гейтах строже задуманного: замер в спеке даёт 33 сигнала до учёта блокировки повторных входов открытой позицией.
