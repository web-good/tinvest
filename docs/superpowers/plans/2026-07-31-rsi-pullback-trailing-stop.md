# ATR-трейлинг в rsi_pullback — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Добавить в backtest-стратегию `rsi_pullback` выход по ATR-трейлингу и сделать RSI-выход отключаемым, чтобы калибровка могла удерживать движение вместо фиксации по первому касанию перекупленности.

**Architecture:** Чистая функция `desiredStop(p, entry, dailyATR, maxFav) (level, reason)` считает единый защитный уровень, связывая ближайший к цене компонент из фиксированного SL и трейла — форма зеркалит проверенную `reversion.DesiredStop`, но работает на своих `Params` и на дневном ATR, замороженном на входе (`Position.EntryATR`). Трейл читает `Position.PrevMaxFavorablePrice`, а не `MaxFavorablePrice`, потому что движок марк-ту-маркетит позицию до `Decide`, а стоп триггерится внутрибарно по `low`. Три новых поля `Params` метутся существующей рефлексивной сеткой калибровки без правок движка.

**Tech Stack:** Go 1.25, стандартный `testing` (table-driven), mage (`./bin/mage ci`), JSON-гриды в `data/params/rsi_pullback/`.

## Global Constraints

- Спека: `docs/superpowers/specs/2026-07-31-rsi-pullback-trailing-stop-design.md`. При расхождении плана со спекой права спека.
- **Дефолты обязаны сохранять текущее поведение побайтово.** `UseRSIExit: 1, UseTrail: 0, TrailDailyATR: 0`. Уже откалиброванные наборы по тикерам (`data/params/rsi_pullback/gazp`, `.../t`) не содержат новых ключей и получат дефолты; если те сдвинут поведение, все прошлые прогоны станут несопоставимыми молча.
- Все поля `core.Params` — только `int` или `float64`: иначе их не увидит рефлексивная сетка (`applyField` в `internal/service/backtest/calibrate.go`).
- Единица трейла — **дневной ATR, замороженный на входе** (`Position.EntryATR`), не текущий. `EntryATR <= 0` отключает трейл целиком.
- Трейл читает **`Position.PrevMaxFavorablePrice`**. Подмена на `MaxFavorablePrice` даёт систематически завышенный результат — см. Task 2, Step 1.
- Порядок выходов: **STOP(SL|TRAIL) → TP → RSI**. Стоп выигрывает ничью с целью на одном баре.
- `model.IsStopReason` уже содержит `"TRAIL"` — правок `internal/domain/backtest/` и `scalping/model/` в этом плане НЕТ.
- Комментарии в коде — на английском (как весь пакет `core`), сообщения тестов и коммиты — на русском (как в существующем `core_test.go`).
- Гейт после каждой задачи: `./bin/mage ci` (lint + `go test -race ./...` + проверка дрейфа моков).

## Файловая структура

| Файл | Ответственность | Задача |
|---|---|---|
| `internal/service/trading_strategy/rsi_pullback/strategy/core/core.go` | `Params` + `DefaultParams` + `desiredStop` + `manage` + `Explain` + шапка пакета | 1, 2, 3 |
| `internal/service/trading_strategy/rsi_pullback/strategy/core/core_test.go` | все тесты пакета | 1, 2, 3 |
| `data/params/rsi_pullback/grid.json` | основная фазовая сетка: фаза `exit` заменяется на `trail` | 4 |
| `data/params/rsi_pullback/cal_trail.json` | тематический грид только по трейлу (новый файл) | 4 |
| `data/params/rsi_pullback/t/params.json` | набор из отчёта 134407 для проверки воспроизводимости (новый файл) | 4 |
| `docs/rsi_pullback/strategy.md` | §4 «Три выхода», таблица параметров, описание фаз | 3, 4 |

Логика выхода целиком остаётся в `core.go` — файл уже держит и вход, и выход, разделять его этот план не станет: `manage()` вырастет примерно на 12 строк, а `desiredStop` — отдельная тестируемая единица внутри того же файла, как `dayStateOK` и `volumeOK` рядом.

---

### Task 1: Параметры и функция `desiredStop`

**Files:**
- Modify: `internal/service/trading_strategy/rsi_pullback/strategy/core/core.go:28-65` (Params, DefaultParams), новая функция после `manage`
- Test: `internal/service/trading_strategy/rsi_pullback/strategy/core/core_test.go:153-166` (TestDefaultParams) + новые тесты

**Interfaces:**
- Consumes: ничего от других задач.
- Produces: поля `Params.UseRSIExit int`, `Params.UseTrail int`, `Params.TrailDailyATR float64`; функция `desiredStop(p Params, entry, dailyATR, maxFav float64) (float64, string)`, возвращающая `(уровень, "SL"|"TRAIL")` либо `(0, "")`. Task 2 вызывает её из `manage`, Task 3 — из `Explain`.

- [ ] **Step 1: Написать падающий тест на `desiredStop`**

Добавить в конец `core_test.go`:

```go
// TestDesiredStopBindsTheNearestLevel фиксирует главное правило уровня: среди активных
// компонентов связывает ЧИСЛЕННО БОЛЬШИЙ — он ближе к цене, и падающая цена коснётся его
// первым. Таблица заодно пинует каждое условие отключения по отдельности: тест, который
// проверяет только «трейл выше SL», переживает удаление любой из guard-веток.
func TestDesiredStopBindsTheNearestLevel(t *testing.T) {
	base := DefaultParams()
	base.StopDailyATR = 0.5
	base.UseTrail = 1
	base.TrailDailyATR = 1.0

	tests := []struct {
		name       string
		mutate     func(p *Params)
		entry      float64
		dailyATR   float64
		maxFav     float64
		wantLevel  float64
		wantReason string
	}{
		{
			name:       "трейл ещё ниже SL — связывает SL",
			entry:      100,
			dailyATR:   10,
			maxFav:     100, // трейл = 100-10 = 90, SL = 100-5 = 95
			wantLevel:  95,
			wantReason: "SL",
		},
		{
			name:       "цена ушла вверх — трейл перехватывает",
			entry:      100,
			dailyATR:   10,
			maxFav:     112, // трейл = 112-10 = 102 > SL 95
			wantLevel:  102,
			wantReason: "TRAIL",
		},
		{
			name:       "трейл ровно на уровне SL — SL удерживает связь",
			entry:      100,
			dailyATR:   10,
			maxFav:     105, // трейл = 95 == SL 95, строгое > не срабатывает
			wantLevel:  95,
			wantReason: "SL",
		},
		{
			name:       "UseTrail выключен",
			mutate:     func(p *Params) { p.UseTrail = 0 },
			entry:      100,
			dailyATR:   10,
			maxFav:     112,
			wantLevel:  95,
			wantReason: "SL",
		},
		{
			name:       "TrailDailyATR=0 при включённом UseTrail",
			mutate:     func(p *Params) { p.TrailDailyATR = 0 },
			entry:      100,
			dailyATR:   10,
			maxFav:     112,
			wantLevel:  95,
			wantReason: "SL",
		},
		{
			name:       "maxFav=0 — трейлить не от чего",
			entry:      100,
			dailyATR:   10,
			maxFav:     0,
			wantLevel:  95,
			wantReason: "SL",
		},
		{
			name:       "дневной ATR не посчитан — стопов нет вовсе",
			entry:      100,
			dailyATR:   0,
			maxFav:     112,
			wantLevel:  0,
			wantReason: "",
		},
		{
			name:       "SL отключён, трейл несёт уровень один",
			mutate:     func(p *Params) { p.StopDailyATR = 0 },
			entry:      100,
			dailyATR:   10,
			maxFav:     112,
			wantLevel:  102,
			wantReason: "TRAIL",
		},
		{
			name:       "оба отключены",
			mutate:     func(p *Params) { p.StopDailyATR = 0; p.UseTrail = 0 },
			entry:      100,
			dailyATR:   10,
			maxFav:     112,
			wantLevel:  0,
			wantReason: "",
		},
		{
			name:       "уровень провалился в ноль — не пол, а голый лонг",
			mutate:     func(p *Params) { p.StopDailyATR = 20; p.UseTrail = 0 },
			entry:      100,
			dailyATR:   10,
			maxFav:     100,
			wantLevel:  0,
			wantReason: "",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			p := base
			if tc.mutate != nil {
				tc.mutate(&p)
			}
			level, reason := desiredStop(p, tc.entry, tc.dailyATR, tc.maxFav)
			if math.Abs(level-tc.wantLevel) > 1e-9 || reason != tc.wantReason {
				t.Fatalf("desiredStop = (%.4f, %q), want (%.4f, %q)", level, reason, tc.wantLevel, tc.wantReason)
			}
		})
	}
}
```

- [ ] **Step 2: Прогнать тест, убедиться что он не компилируется**

Run: `go test ./internal/service/trading_strategy/rsi_pullback/strategy/core/ -run TestDesiredStopBindsTheNearestLevel`
Expected: FAIL — `undefined: desiredStop`, `p.UseTrail undefined`, `p.TrailDailyATR undefined`.

- [ ] **Step 3: Добавить поля в `Params` и `DefaultParams`**

В `core.go` в конец структуры `Params` (после `VolMult`):

```go
	UseRSIExit      int     // 1 arms the RSI exit; any other value disables it (grid; default 1)
	UseTrail        int     // 1 arms the ATR trailing stop; any other value disables it (grid; default 0)
	TrailDailyATR   float64 // trail = maxFav - TrailDailyATR*dailyATR; 0 disables it (grid)
```

В `DefaultParams()` после `VolMult: 1.2,`:

```go
		UseRSIExit:      1,
		UseTrail:        0,
		TrailDailyATR:   0,
```

- [ ] **Step 4: Написать `desiredStop`**

В `core.go` сразу после функции `manage`:

```go
// desiredStop returns the single protective stop level for an open position and the reason of
// the binding component ("SL" | "TRAIL"), or (0, "") when no stop is enabled or the daily ATR
// could not be computed. maxFav is the monotonic max of closes the trail may trail from;
// callers pass Position.PrevMaxFavorablePrice, never MaxFavorablePrice — see manage() for why.
// dailyATR<=0 disables every price stop outright (the live-trading guard: EntryATR is not
// persisted there). Among the active components the numerically GREATEST level binds: it is the
// closest to price, and therefore the first one price would touch as it falls. A level at or
// below zero is not a floor but a naked long, and is reported as "no stop" rather than passed on.
func desiredStop(p Params, entry, dailyATR, maxFav float64) (float64, string) {
	if dailyATR <= 0 {
		return 0, ""
	}
	level, reason := 0.0, ""
	if p.StopDailyATR > 0 {
		level, reason = entry-p.StopDailyATR*dailyATR, "SL"
	}
	if p.UseTrail == 1 && p.TrailDailyATR > 0 && maxFav > 0 {
		if l := maxFav - p.TrailDailyATR*dailyATR; l > level {
			level, reason = l, "TRAIL"
		}
	}
	if level <= 0 {
		return 0, ""
	}
	return level, reason
}
```

- [ ] **Step 5: Прогнать тест, убедиться что он проходит**

Run: `go test ./internal/service/trading_strategy/rsi_pullback/strategy/core/ -run TestDesiredStopBindsTheNearestLevel -v`
Expected: PASS, все 10 подтестов.

- [ ] **Step 6: Обновить `TestDefaultParams` под новые поля**

Заменить литерал `want` в `TestDefaultParams` (core_test.go:155-162) на:

```go
	want := Params{
		RSIPeriod: 4, RSILower: 30, RSIUpper: 70,
		EMAFast: 10, EMASlow: 100,
		DailyATRPeriod: 14,
		UseDayATRGate:  1, FreshDayATR: 0, SpentDayATR: 0.8,
		StopDailyATR: 0.5, TPDailyATR: 0.6,
		UseVolume: 0, VolBaseDays: 14, VolLookbackBars: 3, VolMult: 1.2,
		UseRSIExit: 1, UseTrail: 0, TrailDailyATR: 0,
	}
```

- [ ] **Step 7: Прогнать весь пакет**

Run: `go test ./internal/service/trading_strategy/rsi_pullback/... -count=1`
Expected: PASS. Ни один существующий тест не должен упасть — `desiredStop` пока никем не вызывается, а дефолты нейтральны.

- [ ] **Step 8: Коммит**

```bash
git add internal/service/trading_strategy/rsi_pullback/strategy/core/core.go \
        internal/service/trading_strategy/rsi_pullback/strategy/core/core_test.go
git commit -m "feat(rsi_pullback): параметры трейла и функция desiredStop

Уровень защитного стопа считает одна чистая функция: среди активных
компонентов связывает ближайший к цене. Форма зеркалит reversion.DesiredStop,
но работает на дневном ATR. Дефолты нейтральны (UseTrail=0, UseRSIExit=1),
поведение выходов пока не меняется — desiredStop ещё никем не вызывается.

Co-Authored-By: Claude Opus 5 <noreply@anthropic.com>"
```

---

### Task 2: Встроить трейл в `manage()` и сделать RSI-выход отключаемым

**Files:**
- Modify: `internal/service/trading_strategy/rsi_pullback/strategy/core/core.go:446-486` (комментарий к `manage` и её тело)
- Test: `internal/service/trading_strategy/rsi_pullback/strategy/core/core_test.go` (новые тесты в конец)

**Interfaces:**
- Consumes: `desiredStop(p Params, entry, dailyATR, maxFav float64) (float64, string)` из Task 1; поля `Params.UseRSIExit`, `Params.UseTrail`, `Params.TrailDailyATR` из Task 1.
- Produces: `manage` выдаёт `sig.Reason` из множества `"SL" | "TRAIL" | "TP" | "RSI"`; при стоповом выходе `sig.StopLoss` равен сработавшему уровню.

- [ ] **Step 1: Написать падающий регресс на анти-lookahead**

Это главный тест задачи. Движок в `Run()` (`engine.go:167-173`) вызывает `p.mark(candles[i].Close)` ДО `Decide`, поэтому `MaxFavorablePrice` уже включает close текущего бара, а стоп триггерится по `low` того же бара. Порядок `low` и `close` внутри бара из OHLC неизвестен: если close случился после low, реальный ордер в момент прохода low стоял на уровне, ничего не знавшем про этот close. `MaxFavorablePrice >= PrevMaxFavorablePrice` всегда, поэтому отсчёт от MaxFav даёт уровень не ниже честного — срабатывает не реже и фиксирует по цене не хуже.

Добавить в конец `core_test.go`:

```go
// trailFixture строит позицию с включённым трейлом на плоской серии, где всё, кроме
// low последнего бара и двух максимумов позиции, зафиксировано. entry=100, EntryATR=10.
func trailFixture(p Params, low, maxFav, prevMaxFav float64) (*Strategy, strategy.MarketData) {
	s := NewWithParams("TEST", p)
	start := time.Date(2026, 3, 2, 7, 0, 0, 0, msk)
	md := barSeries([]float64{100, 100, 100, 105}, start)
	i := len(md.Closes) - 1
	md.Lows[i], md.Highs[i], md.Closes[i] = low, 105, 105
	md.Price = 105
	md.Position = &strategy.Position{
		PurchasePrice:         100,
		Quantity:              1,
		StopLoss:              0, // фиксированный SL выключен: изолируем трейл
		TakeProfit:            0,
		EntryATR:              10,
		MaxFavorablePrice:     maxFav,
		PrevMaxFavorablePrice: prevMaxFav,
		EntryTime:             md.Times[0],
	}
	return s, md
}

// TestTrailReadsPrevMaxFavorable — регресс на lookahead. Движок марк-ту-маркетит позицию
// ДО Decide, поэтому MaxFavorablePrice уже знает close текущего бара, а стоп проверяется по
// low того же бара. Отсчёт трейла от MaxFavorablePrice выдал бы уровень, которого биржевой
// ордер в момент прохода low знать не мог. Тест падает при подмене поля.
func TestTrailReadsPrevMaxFavorable(t *testing.T) {
	p := DefaultParams()
	p.StopDailyATR = 0
	p.UseTrail = 1
	p.TrailDailyATR = 0.5 // трейл = maxFav - 5
	p.UseRSIExit = 0      // изолируем стоп от RSI-ветки

	// Бар: low 96, close 105. maxFav после mark = 105 -> уровень 100, low 96 его пробил бы.
	// prevMaxFav = 100 -> уровень 95, low 96 его НЕ достаёт.
	s, md := trailFixture(p, 96, 105, 100)
	got := s.Decide(md)
	if got.Kind == model.SignalSell {
		t.Fatalf("выход %q по уровню от MaxFavorablePrice: трейл обязан считаться от "+
			"PrevMaxFavorablePrice (95), а low 96 его не достаёт", got.Reason)
	}
}

// TestExitTrailFiresOnLow проверяет само срабатывание, включая точное касание: `low <= level`
// должно сработать, когда low ложится РОВНО на уровень. Тест только со строгим пробоем
// пережил бы замену сравнения на `<`.
func TestExitTrailFiresOnLow(t *testing.T) {
	tests := []struct {
		name string
		low  float64
	}{
		{"low пробил трейл", 94.9},
		{"low коснулся трейла ровно", 95},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			p := DefaultParams()
			p.StopDailyATR = 0
			p.UseTrail = 1
			p.TrailDailyATR = 0.5
			p.UseRSIExit = 0

			// prevMaxFav = 100, EntryATR = 10 -> уровень 100 - 0.5*10 = 95.
			s, md := trailFixture(p, tc.low, 100, 100)
			got := s.Decide(md)
			if got.Kind != model.SignalSell || got.Reason != "TRAIL" {
				t.Fatalf("Kind/Reason = %v/%q, want Sell/TRAIL", got.Kind, got.Reason)
			}
			if math.Abs(got.StopLoss-95) > 1e-9 {
				t.Fatalf("StopLoss = %v, want 95: движок заливает по min(level, open), "+
					"и без уровня в сигнале он зальёт по close", got.StopLoss)
			}
		})
	}
}

// TestTrailNeverGoesBelowSL: широкий трейл не должен ослаблять фиксированный стоп.
func TestTrailNeverGoesBelowSL(t *testing.T) {
	p := DefaultParams()
	p.StopDailyATR = 0.5 // SL = 100 - 5 = 95
	p.UseTrail = 1
	p.TrailDailyATR = 3.0 // трейл = 100 - 30 = 70, сильно ниже
	p.UseRSIExit = 0

	s, md := trailFixture(p, 94.9, 100, 100)
	got := s.Decide(md)
	if got.Kind != model.SignalSell || got.Reason != "SL" {
		t.Fatalf("Kind/Reason = %v/%q, want Sell/SL: широкий трейл не ослабляет стоп",
			got.Kind, got.Reason)
	}
}

// TestTrailDisabledWithoutEntryATR — live-guard: без ATR на входе трейл не считается.
func TestTrailDisabledWithoutEntryATR(t *testing.T) {
	p := DefaultParams()
	p.StopDailyATR = 0
	p.UseTrail = 1
	p.TrailDailyATR = 0.5
	p.UseRSIExit = 0

	s, md := trailFixture(p, 50, 100, 100)
	md.Position.EntryATR = 0
	if got := s.Decide(md); got.Kind == model.SignalSell {
		t.Fatalf("выход %q без EntryATR: трейл обязан молчать, когда линейка не известна",
			got.Reason)
	}
}

// TestTrailWinsOverTakeProfitOnTheSameBar: приоритет стопа над целью держится и для трейла.
func TestTrailWinsOverTakeProfitOnTheSameBar(t *testing.T) {
	p := DefaultParams()
	p.StopDailyATR = 0
	p.UseTrail = 1
	p.TrailDailyATR = 0.5
	p.UseRSIExit = 0

	s, md := trailFixture(p, 94.9, 100, 100)
	i := len(md.Closes) - 1
	md.Highs[i] = 130
	md.Position.TakeProfit = 120 // бар задевает и цель, и трейл
	if got := s.Decide(md); got.Reason != "TRAIL" {
		t.Fatalf("Reason = %q, want TRAIL: внутрибарный порядок неизвестен, побеждает худший исход",
			got.Reason)
	}
}

// TestUseRSIExitZeroKeepsPosition: при UseRSIExit=0 крест RSI вверх позицию не закрывает.
func TestUseRSIExitZeroKeepsPosition(t *testing.T) {
	p := DefaultParams()
	p.UseRSIExit = 0
	s := NewWithParams("T", p)
	md := upperCrossFixture()
	i := len(md.Closes) - 1
	md = withPosition(md, md.Closes[i]*0.97, md.Lows[i]*0.5, 2)
	if got := s.Decide(md); got.Kind == model.SignalSell {
		t.Fatalf("выход %q при UseRSIExit=0: RSI-выход должен быть отключён", got.Reason)
	}
}

// TestDefaultsPreserveLegacyExits: на дефолтах трейла нет, RSI-выход есть — то же поведение,
// что до появления обоих параметров. Это страховка совместимости уже откалиброванных наборов
// по тикерам, в которых новых ключей нет и которые поэтому получат дефолты.
func TestDefaultsPreserveLegacyExits(t *testing.T) {
	s := NewWithParams("T", DefaultParams())

	// RSI-выход работает.
	md := upperCrossFixture()
	i := len(md.Closes) - 1
	md = withPosition(md, md.Closes[i]*0.97, md.Lows[i]*0.5, 2)
	if got := s.Decide(md); got.Kind != model.SignalSell || got.Reason != "RSI" {
		t.Fatalf("Kind/Reason = %v/%q, want Sell/RSI на дефолтах", got.Kind, got.Reason)
	}

	// Трейла нет: позиция глубоко в прибыли, но проседание её не закрывает.
	p := DefaultParams()
	p.StopDailyATR = 0
	p.UseRSIExit = 0
	s2, md2 := trailFixture(p, 50, 200, 200)
	if got := s2.Decide(md2); got.Kind == model.SignalSell {
		t.Fatalf("выход %q на дефолтах: UseTrail=0, трейла быть не должно", got.Reason)
	}
}

// TestTrailIsAStopReason пинует неявную связь со движком: "TRAIL" обязан числиться стоповой
// причиной, иначе бэктест зальёт выход по close и потеряет модель гэпа min(level, open).
func TestTrailIsAStopReason(t *testing.T) {
	if !model.IsStopReason("TRAIL") {
		t.Fatal(`model.IsStopReason("TRAIL") = false: движок зальёт трейл по close`)
	}
}
```

- [ ] **Step 2: Прогнать новые тесты, убедиться что падают**

Run: `go test ./internal/service/trading_strategy/rsi_pullback/strategy/core/ -run 'TestTrail|TestExitTrail|TestUseRSIExit|TestDefaultsPreserveLegacyExits' -v`
Expected: FAIL. `TestTrailIsAStopReason` пройдёт сразу (`IsStopReason` уже знает `"TRAIL"`) — это нормально, он сторожит существующее свойство. Остальные падают: трейл не реализован, `UseRSIExit` не читается.

- [ ] **Step 3: Переписать `manage()`**

Заменить тело `manage` (core.go, начиная со строки `i := n - 1` внутри функции) на:

```go
	i := n - 1
	high, low, closeP := md.Highs[i], md.Lows[i], md.Closes[i]

	// 1. protective stop: the fixed SL and the ATR trail resolved into one level by
	// desiredStop. It wins a same-bar tie with the target: the intrabar order of the two
	// touches is unknowable from OHLC, and assuming the worse of the two is the honest choice.
	// The trail reads PrevMaxFavorablePrice, NOT MaxFavorablePrice: the engine marks the
	// position to market before calling Decide, so MaxFavorablePrice already contains this
	// bar's close, while the level is tested against this bar's low. An exchange stop order
	// working during bar i was placed after bar i-1 closed and cannot know about a close that
	// may have happened after the low. Since MaxFavorablePrice >= PrevMaxFavorablePrice always,
	// reading the former yields a level never below the honest one — it fires no less often and
	// fills no worse, inflating the result in both directions at once.
	if level, reason := desiredStop(s.p, pos.PurchasePrice, pos.EntryATR, pos.PrevMaxFavorablePrice); level > 0 && low <= level {
		sig.Kind, sig.Reason = model.SignalSell, reason
		sig.StopLoss = level
		sig.ExitReason = fmt.Sprintf("%s: low %.4f ≤ уровень %.4f (вход %.4f)", reason, low, level, pos.PurchasePrice)
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
	if s.p.UseRSIExit != 1 {
		return sig
	}
	rsi := indicators.RSISeries(md.Closes, s.p.RSIPeriod)
	if len(rsi) == n && crossedUp(rsi, i, s.p.RSIUpper) {
		sig.Kind, sig.Reason = model.SignalSell, "RSI"
		sig.RSI = rsi[i]
		sig.ExitReason = fmt.Sprintf("RSI: RSI(%d) пересёк %.0f снизу вверх (%.1f), выход по %.4f (вход %.4f)",
			s.p.RSIPeriod, s.p.RSIUpper, rsi[i], closeP, pos.PurchasePrice)
	}
	return sig
```

**Внимание на три вещи.**

Первая: уровень теперь считает `desiredStop` от `pos.EntryATR`, а не читается из `pos.StopLoss`. Фиксированный SL пересчитывается из тех же входных данных, что заморозил вход (`entry` и дневной ATR те же), поэтому в бэктесте результат совпадает до последнего знака.

Вторая, как следствие: **`Position.StopLoss` перестаёт читаться в `manage`.** Поле остаётся заполненным (движок кладёт туда `sig.StopLoss` при открытии, и оно нужно живому слою других стратегий), но для `rsi_pullback` становится справочным. Не «чинить» это, возвращая чтение поля: тогда трейл и SL считались бы из разных источников, и `desiredStop` перестала бы быть единственным местом, где рождается уровень.

Третья: `TestExitStopLoss` строит позицию через `withPosition`, где `EntryATR = entryPrice * 0.003`, а `StopDailyATR` дефолтный 0.5 — значит пересчитанный SL окажется НЕ там, где `pos.StopLoss`. Этот тест придётся поправить в Step 5.

- [ ] **Step 4: Обновить комментарий к `manage`**

Заменить строки core.go:446-452 (док-комментарий перед `func (s *Strategy) manage`) на:

```go
// manage handles an open long. It exits on one of four signals, evaluated in precedence order
// STOP(SL|TRAIL) → TP → RSI. The protective stop and the target trigger INTRABAR (the bar's low
// touching the level, the bar's high reaching the target), because a real stop or limit order
// fills as soon as price trades through it during the bar; the engine handles their fill pricing
// via model.IsStopReason and the "TP" reason. The RSI exit fills at the bar close and can be
// disabled with UseRSIExit=0. There is no time stop and no end-of-day close — the position is
// held until one of the exits fires, across nights and weekends.
```

- [ ] **Step 5: Починить `TestExitStopLoss` под пересчёт уровня**

`manage` больше не читает `pos.StopLoss`, а пересчитывает уровень как `entry - StopDailyATR*EntryATR`. Фикстура должна задавать стоп через эти величины. Заменить `TestExitStopLoss` (core_test.go:608-632) на:

```go
// TestExitStopLoss covers the stop, including the exact-touch boundary: `low <= level` must
// fire when the low lands ON the level. A test that only puts the stop strictly inside the bar
// survives a shift of that comparison to `<`.
func TestExitStopLoss(t *testing.T) {
	p := DefaultParams()
	p.StopDailyATR = 0.5
	s := NewWithParams("T", p)
	tests := []struct {
		name  string
		lowAt func(level float64) float64
	}{
		{"low pierces the stop", func(level float64) float64 { return level * 0.999 }},
		{"low touches the stop exactly", func(level float64) float64 { return level }},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			start := time.Date(2026, 3, 2, 7, 0, 0, 0, msk)
			md := barSeries([]float64{100, 100, 100, 100}, start)
			i := len(md.Closes) - 1
			// entry 100, EntryATR 10, StopDailyATR 0.5 -> уровень 95.
			const level = 95.0
			md.Lows[i] = tc.lowAt(level)
			md.Position = &strategy.Position{
				PurchasePrice: 100, Quantity: 1, StopLoss: level,
				EntryATR: 10, MaxFavorablePrice: 100, PrevMaxFavorablePrice: 100,
				EntryTime: md.Times[0],
			}
			got := s.Decide(md)
			if got.Kind != model.SignalSell || got.Reason != "SL" {
				t.Fatalf("Kind/Reason = %v/%q, want Sell/SL", got.Kind, got.Reason)
			}
			if math.Abs(got.StopLoss-level) > 1e-9 {
				t.Fatalf("StopLoss = %v, want %v", got.StopLoss, level)
			}
		})
	}
}
```

- [ ] **Step 6: Прогнать весь пакет и починить оставшиеся фикстуры**

Run: `go test ./internal/service/trading_strategy/rsi_pullback/strategy/core/ -count=1 -v`

Причина возможных падений одна: `manage` больше не читает `pos.StopLoss`, а пересчитывает уровень как `entry − StopDailyATR × EntryATR`. Тесты делятся на две группы.

**Тесты с литералом `&strategy.Position{...}` без `EntryATR`** (`TestExitTakeProfit`, `TestPositionSurvivesOvernight`, `TestPositionSurvivesWeekend`, `TestExitTakeProfitDisabledAtZero`) получают `EntryATR = 0`, `desiredStop` вернёт `(0, "")`, стоповая ветка отключится — они продолжат проверять ровно свой выход и упасть не должны.

**Ровно одно исключение — `TestExitStopWinsOverTakeProfitOnTheSameBar`** (core_test.go:704-717): он ПРО приоритет стопа, а с `EntryATR=0` стопа не станет и выход уйдёт в `TP`. Заменить его на:

```go
func TestExitStopWinsOverTakeProfitOnTheSameBar(t *testing.T) {
	p := DefaultParams()
	p.StopDailyATR = 0.5
	s := NewWithParams("TEST", p)
	start := time.Date(2026, 3, 2, 7, 0, 0, 0, msk)
	md := barSeries([]float64{100, 100, 100, 100}, start)
	i := len(md.Closes) - 1
	md.Highs[i], md.Lows[i] = 110, 90 // бар задевает и цель, и стоп
	md.Position = &strategy.Position{
		PurchasePrice: 100, Quantity: 1, StopLoss: 95, TakeProfit: 105,
		// entry 100, EntryATR 10, StopDailyATR 0.5 -> уровень 95, low 90 его пробивает.
		EntryATR: 10, MaxFavorablePrice: 100, PrevMaxFavorablePrice: 100,
		EntryTime: md.Times[0],
	}
	if sig := s.Decide(md); sig.Reason != "SL" {
		t.Fatalf("Reason = %q, want SL: внутрибарный порядок неизвестен, побеждает худший исход", sig.Reason)
	}
}
```

**Тесты на фикстуре `withPosition`** (`TestExitOnRSIEnteringUpperBand`, `TestExitStopWinsOverRSIOnTheSameBar`, `TestNoLookaheadWithOpenPosition`) сейчас получают `EntryATR = entryPrice*0.003`, то есть пересчитанный уровень `entry*0.9985` — намного ближе к цене, чем задумывала фикстура, и стоп начнёт срабатывать там, где тест его не ждёт. Перевести фикстуру на выключенный стоп:

```go
func withPosition(md strategy.MarketData, entryPrice, stop float64, heldBars int) strategy.MarketData {
	last := md.Times[len(md.Times)-1]
	md.Position = &strategy.Position{
		PurchasePrice: entryPrice,
		StopLoss:      stop,
		// EntryATR=0 гасит защитный уровень целиком: фикстура обслуживает тесты о ДРУГИХ
		// выходах, и пересчитанный стоп там только мешал бы. Тесты про сам стоп задают
		// EntryATR явно.
		EntryATR:              0,
		MaxFavorablePrice:     entryPrice,
		PrevMaxFavorablePrice: entryPrice,
		EntryTime:             last.Add(-time.Duration(heldBars) * 30 * time.Minute),
	}
	return md
}
```

После этой правки `TestExitStopWinsOverRSIOnTheSameBar` перестанет проверять то, ради чего написан (стоп выключён `EntryATR=0`). Заменить его на:

```go
func TestExitStopWinsOverRSIOnTheSameBar(t *testing.T) {
	p := DefaultParams()
	p.StopDailyATR = 0.5
	s := NewWithParams("T", p)
	md := upperCrossFixture()
	i := len(md.Closes) - 1
	entry := md.Closes[i] * 0.97
	// EntryATR подобран так, чтобы уровень стопа лёг внутрь бара: level = entry - 0.5*atr.
	atr := (entry - md.Lows[i]*1.0001) / 0.5
	md = withPosition(md, entry, 0, 2)
	md.Position.EntryATR = atr
	md.Position.MaxFavorablePrice, md.Position.PrevMaxFavorablePrice = entry, entry
	if got := s.Decide(md); got.Reason != "SL" {
		t.Fatalf("Reason = %q, want SL to take precedence over RSI", got.Reason)
	}
}
```

- [ ] **Step 7: Прогнать пакет до зелёного**

Run: `go test ./internal/service/trading_strategy/rsi_pullback/... -count=1`
Expected: PASS полностью.

- [ ] **Step 8: Прогнать полный гейт**

Run: `./bin/mage ci`
Expected: зелёный.

- [ ] **Step 9: Коммит**

```bash
git add internal/service/trading_strategy/rsi_pullback/strategy/core/core.go \
        internal/service/trading_strategy/rsi_pullback/strategy/core/core_test.go
git commit -m "feat(rsi_pullback): выход по ATR-трейлингу и отключаемый RSI-выход

manage() резолвит защитный уровень через desiredStop и триггерит его по low
бара, отдавая уровень в sig.StopLoss — иначе движок зальёт выход по close и
потеряет модель гэпа min(level, open). Трейл считается от
PrevMaxFavorablePrice: движок марк-ту-маркетит позицию до Decide, и отсчёт от
MaxFavorablePrice дал бы уровень, которого биржевой ордер в момент прохода low
знать не мог. TestTrailReadsPrevMaxFavorable падает при подмене поля.

Фикстуры выходов переведены на EntryATR: manage() больше не читает
pos.StopLoss, а пересчитывает уровень из замороженного на входе ATR.

Co-Authored-By: Claude Opus 5 <noreply@anthropic.com>"
```

---

### Task 3: `Explain` и документация стратегии

**Files:**
- Modify: `internal/service/trading_strategy/rsi_pullback/strategy/core/core.go:1-8` (шапка пакета), `:542-551` (хвост `Explain`)
- Modify: `docs/rsi_pullback/strategy.md:11-17` (интро), `:36` и таблица параметров, `:102-119` (§4)
- Test: `internal/service/trading_strategy/rsi_pullback/strategy/core/core_test.go:764-773` (`TestExplainMentionsEveryGate`)

**Interfaces:**
- Consumes: `desiredStop` и поля `Params` из Task 1; коды причин из Task 2.
- Produces: ничего для последующих задач (документация и диагностика).

- [ ] **Step 1: Расширить `TestExplainMentionsEveryGate`**

Заменить список ожидаемых подстрок (core_test.go:769):

```go
	for _, want := range []string{"день", "RSI", "EMA", "дневной ATR", "состояние дня", "фон объёмов", "стоп", "цель", "трейл", "выход по RSI"} {
```

- [ ] **Step 2: Прогнать, убедиться что падает**

Run: `go test ./internal/service/trading_strategy/rsi_pullback/strategy/core/ -run TestExplainMentionsEveryGate -v`
Expected: FAIL — `Explain не упоминает "трейл"`.

- [ ] **Step 3: Дописать хвост `Explain`**

После блока про цель (core.go, перед `return sb.String()`) добавить:

```go
	if s.p.UseTrail == 1 && s.p.TrailDailyATR > 0 && atr > 0 {
		fmt.Fprintf(&sb, "трейл: максимум − %.2f×ATR (%.4f); отсчёт от PrevMaxFavorablePrice\n",
			s.p.TrailDailyATR, s.p.TrailDailyATR*atr)
	} else {
		sb.WriteString("трейл: выключен\n")
	}
	if s.p.UseRSIExit == 1 {
		fmt.Fprintf(&sb, "выход по RSI: крест вверх через %.0f\n", s.p.RSIUpper)
	} else {
		sb.WriteString("выход по RSI: выключен (UseRSIExit=0)\n")
	}
```

- [ ] **Step 4: Прогнать, убедиться что проходит**

Run: `go test ./internal/service/trading_strategy/rsi_pullback/strategy/core/ -run TestExplainMentionsEveryGate -v`
Expected: PASS.

- [ ] **Step 5: Обновить шапку пакета**

Заменить core.go:3-6 (фрагмент про выходы) так, чтобы описание читалось:

```go
// through its lower band on the current bar. The stop and target are sized off the daily ATR at
// entry and frozen on the position; the trade is closed on the first of: the protective stop
// (the fixed SL or the ATR trail, whichever binds), the target, or RSI crossing UP through the
// upper band. The RSI exit can be disabled with UseRSIExit=0, leaving the stop and the target to
// carry the trade. There is no time stop and no end-of-day close — the position is held across
// nights and weekends until one of those exits fires.
```

- [ ] **Step 6: Обновить `docs/rsi_pullback/strategy.md`**

Три правки.

Первая — строка 17, заменить `пока её не закроет SL, TP или RSI-выход.` на `пока её не закроет SL, TRAIL, TP или RSI-выход.`

Вторая — в таблицу параметров (после строки про `VolMult`) добавить:

```markdown
| `UseRSIExit` | 0/1 | 1 | да (trail: 0, 1) |
| `UseTrail` | 0/1 | 0 | да (trail: 0, 1) |
| `TrailDailyATR` | множитель дневного ATR | 0 | да (trail: 0.5, 0.8, 1.2) |
```

Третья — переписать §4, заменив заголовок `## 4. Три выхода и тай-брейк` и пункты 1–3 на:

```markdown
## 4. Четыре выхода и тай-брейк

Приоритет строго `STOP(SL|TRAIL) → TP → RSI`; на каждом баре с открытой позицией проверяются
по порядку, срабатывает первый подошедший. Тайм-стопа и закрытия по концу дня нет.

1. **`SL` / `TRAIL`** — единый защитный уровень, который считает `desiredStop`: среди активных
   компонентов связывает численно **больший**, то есть ближайший к цене. Фиксированный
   компонент — `вход − StopDailyATR×дневной ATR`, трейл — `максимум − TrailDailyATR×дневной ATR`.
   Срабатывает при `low ≤ уровень`, причина сигнала — код связавшего компонента. Оба меряются
   дневным ATR, замороженным на входе (`Position.EntryATR`); при `EntryATR ≤ 0` защитного
   стопа нет вовсе.
2. **`TP`** — `high ≥ pos.TakeProfit` при `pos.TakeProfit > 0`.
3. **`RSI`** — короткий RSI пересёк `RSIUpper` **снизу вверх** на текущем баре (отскок
   дошёл до перекупленности). Заполняется по закрытию бара. Отключается `UseRSIExit=0`.

**Трейл отсчитывается от `Position.PrevMaxFavorablePrice`, а не от `MaxFavorablePrice`.**
Движок марк-ту-маркетит позицию до вызова `Decide` (`engine.go:167-173`), поэтому
`MaxFavorablePrice` уже содержит close текущего бара, тогда как уровень проверяется по `low`
того же бара. Порядок `low` и `close` внутри бара из OHLC неизвестен: биржевой ордер,
работавший во время бара, был выставлен после закрытия предыдущего и знать о новом close не
мог. Смещение систематическое — `MaxFavorablePrice ≥ PrevMaxFavorablePrice` всегда, так что
уровень от MaxFav никогда не ниже честного: срабатывает не реже и фиксирует по цене не хуже.
Запинено тестом `TestTrailReadsPrevMaxFavorable`.
```

Строку 118 (`Коды причин выхода — ровно "SL", "TP", "RSI"...`) заменить на:

```markdown
Коды причин выхода — ровно `"SL"`, `"TRAIL"`, `"TP"`, `"RSI"`. `model.IsStopReason` знает про
`"SL"` и `"TRAIL"` (обе заливаются по `min(уровень, open)`), движок обрабатывает `"TP"`
отдельно, `"RSI"` заполняется по close. Связь неявная и легко ломается — запинена тестом
`TestTrailIsAStopReason`.
```

- [ ] **Step 7: Прогнать гейт**

Run: `./bin/mage ci`
Expected: зелёный.

- [ ] **Step 8: Коммит**

```bash
git add internal/service/trading_strategy/rsi_pullback/strategy/core/core.go \
        internal/service/trading_strategy/rsi_pullback/strategy/core/core_test.go \
        docs/rsi_pullback/strategy.md
git commit -m "docs(rsi_pullback): трейл в Explain, шапке пакета и справочнике стратегии

Explain печатает состояние трейла и RSI-выхода, чтобы -explain объяснял
бездействие обоих. §4 справочника переписан под четыре выхода и объясняет,
почему трейл считается от PrevMaxFavorablePrice.

Co-Authored-By: Claude Opus 5 <noreply@anthropic.com>"
```

---

### Task 4: Гриды и проверка обратной совместимости

**Files:**
- Modify: `data/params/rsi_pullback/grid.json` (фаза `exit` → `trail`, обновление `_comment`)
- Create: `data/params/rsi_pullback/cal_trail.json`
- Create: `data/params/rsi_pullback/t/params.json`
- Modify: `docs/rsi_pullback/strategy.md` (§ про фазы и таблицу `cal_*` файлов)

**Interfaces:**
- Consumes: поля `UseRSIExit`, `UseTrail`, `TrailDailyATR` из Task 1 — имена ключей в JSON обязаны совпадать с именами полей `Params` побуквенно, иначе рефлексивный `applyField` их молча не найдёт.
- Produces: ничего для последующих задач.

- [ ] **Step 1: Заменить фазу `exit` на `trail` в `grid.json`**

В `data/params/rsi_pullback/grid.json` заменить последнюю фазу

```json
    {
      "name": "exit",
      "grid": {
        "RSIUpper": [60, 70, 80]
      }
    }
```

на

```json
    {
      "name": "trail",
      "grid": {
        "UseRSIExit": [0, 1],
        "RSIUpper": [60, 70, 80],
        "UseTrail": [0, 1],
        "TrailDailyATR": [0.5, 0.8, 1.2]
      }
    }
```

- [ ] **Step 2: Обновить `_comment` в `grid.json`**

В конце `_comment` заменить предложение про фазу 6 (`Phase 6 (exit) sweeps the RSI band whose upward cross closes the trade.`) на:

```
Phase 6 (trail) sweeps the two upward exits together: the RSI band whose upward cross closes the trade, whether that exit is armed at all (UseRSIExit), and the ATR trailing stop that competes with the fixed stop for the protective level. The two are swept in one phase on purpose — RSIUpper is meaningless when UseRSIExit=0, and the trail only gets a chance to fire when the RSI exit is off, so calibrating them apart would score each against a fixed setting of the other. At UseTrail=0 the TrailDailyATR values collapse to one identical control config, as do the RSIUpper values at UseRSIExit=0, so duplicate leaderboard rows are expected within this phase. The combination UseRSIExit=0, UseTrail=0, TPDailyATR=0 would leave the stop as the only exit; it is reachable in this grid only if TPDailyATR=0 is introduced upstream, which the risk phase deliberately never does.
```

Там же заменить арифметику вычислений (`9 + 6x12 + 5x12 + 5x12 + 4x16 + 4x3 = up to 277 backtest evaluations`) на `9 + 6x12 + 5x12 + 5x12 + 4x16 + 4x36 = up to 409 backtest evaluations`.

- [ ] **Step 3: Создать `data/params/rsi_pullback/cal_trail.json`**

```json
{
  "_comment": "TRAIL band: the ATR trailing stop and the on/off switch of the RSI exit, swept together (36 evaluations). These two belong in one file: the trail can only bind while the RSI exit is off, because on the T calibration the RSI exit closed 45 of 67 trades and would keep firing first. TrailDailyATR is measured in DAILY ATR units, like the stop and the target, and is frozen at entry — 0.5 trails half a daily range behind the running max, 1.2 gives the position room to breathe through a normal pullback. The trail competes with the fixed stop for the protective level and the nearest one to price binds, so a trail wider than StopDailyATR never binds until price has moved up by the difference: calibrate risk first (cal_risk.json), then sweep this over the surviving stop/target pair rather than over DefaultParams. Watch the TRAIL share of exits, not just the profit factor: a set where the trail never fires is really just the old fixed-stop configuration wearing new parameters. Run: go run ./cmd/backtest -ticker GAZP -strategy rsi_pullback -interval Minutes30 -calibrate data/params/rsi_pullback/cal_trail.json -out ./reports/GAZP_trail -months 24 -min-trades 20 -test-months 6 -metric profit_factor. GAZP is only the example instrument here: this file sweeps one theme and is ticker-agnostic, so replace -ticker and -out to calibrate another name (e.g. -ticker T -out ./reports/T_trail, adding -refresh on a ticker whose candle cache is short).",
  "phases": [
    {
      "name": "trail",
      "grid": {
        "UseRSIExit": [0, 1],
        "UseTrail": [0, 1],
        "TrailDailyATR": [0.5, 0.8, 1.0, 1.2, 1.5],
        "RSIUpper": [60, 65, 70]
      }
    }
  ]
}
```

- [ ] **Step 4: Проверить, что гриды — валидный JSON**

Run: `python3 -m json.tool data/params/rsi_pullback/grid.json > /dev/null && python3 -m json.tool data/params/rsi_pullback/cal_trail.json > /dev/null && echo OK`
Expected: `OK`.

- [ ] **Step 5: Создать `data/params/rsi_pullback/t/params.json` с набором из отчёта**

Это набор, давший `reports/T/T_rsi_pullback_Minutes30_20260731_134407.md` (67 сделок, PF 1.312). Новые ключи в нём отсутствуют намеренно: они обязаны прийти из дефолтов и ничего не сдвинуть.

```json
{
  "RSIPeriod": 5,
  "RSILower": 20,
  "RSIUpper": 65,
  "EMAFast": 20,
  "EMASlow": 100,
  "DailyATRPeriod": 14,
  "UseDayATRGate": 1,
  "FreshDayATR": 0,
  "SpentDayATR": 0.9,
  "StopDailyATR": 0.5,
  "TPDailyATR": 1.5,
  "UseVolume": 0,
  "VolBaseDays": 5,
  "VolLookbackBars": 3,
  "VolMult": 1.2
}
```

- [ ] **Step 6: Проверить обратную совместимость прогоном**

Run:
```bash
go run ./cmd/backtest -ticker T -strategy rsi_pullback -interval Minutes30 \
  -params data/params/rsi_pullback/t/params.json \
  -out ./reports/T_compat -months 48 -metric profit_factor
```

Expected: в свежем отчёте `./reports/T_compat/*.md` сводка совпадает с `reports/T/T_rsi_pullback_Minutes30_20260731_134407.md` — **67 сделок, profit factor 1.312, чистый PnL 10528.08**, и ни одной строки с причиной `TRAIL` в журнале сделок.

Проверить числа командой:
```bash
grep -E "Всего сделок|Profit factor|Чистый PnL" ./reports/T_compat/*.md
grep -c "| TRAIL |" ./reports/T_compat/*_trades.csv || echo "TRAIL: 0 (ожидаемо)"
```

Если числа разошлись — дефолты сдвинули поведение, и это блокер: вернуться к Task 1 Step 3 и Task 2 Step 3, не продолжать.

- [ ] **Step 7: Проверить, что трейл вообще срабатывает**

Скопировать набор с включённым трейлом:
```bash
python3 - <<'EOF'
import json
p = json.load(open('data/params/rsi_pullback/t/params.json'))
p.update({"UseRSIExit": 0, "UseTrail": 1, "TrailDailyATR": 0.8})
json.dump(p, open('/tmp/t_trail_smoke.json', 'w'), indent=2)
EOF
go run ./cmd/backtest -ticker T -strategy rsi_pullback -interval Minutes30 \
  -params /tmp/t_trail_smoke.json -out ./reports/T_trailsmoke -months 48 -metric profit_factor
grep -c "TRAIL" ./reports/T_trailsmoke/*_trades.csv
```

Expected: счётчик строго больше нуля. Это дым-тест механики, а не оценка качества — судить о трейле можно только по walk-forward, который идёт отдельной задачей.

- [ ] **Step 8: Дописать фазу `trail` в справочник стратегии**

В `docs/rsi_pullback/strategy.md` в таблицу файлов калибровки (около строки 260, рядом с `| cal_exit.json | 6 | уровень выхода RSIUpper |`) добавить строку:

```markdown
| `cal_trail.json` | 36 | трейл `TrailDailyATR` + отключение RSI-выхода `UseRSIExit` + `RSIUpper` |
```

Заголовок раздела `**Фаза 6 — `exit`, выход по RSI** (последняя, `keepTop` не нужен).` (около строки 415) и его текст заменить на:

```markdown
**Фаза 6 — `trail`, оба верхних выхода** (последняя, `keepTop` не нужен).

- `UseRSIExit` — `0, 1`. При `0` RSI-выход отключён целиком, и сделку закрывают только
  защитный уровень и цель.
- `RSIUpper` — уровень, крест **вверх** через который закрывает сделку: `60, 70, 80`.
  Осмысленен только при `UseRSIExit=1`; при `0` все три значения дают одну конфигурацию.
- `UseTrail` — `0, 1`. При `0` значения `TrailDailyATR` схлопываются в один контроль.
- `TrailDailyATR` — ширина трейла в дневных ATR: `0.5, 0.8, 1.2`.

Обе оси метутся одной фазой намеренно: трейл получает шанс связать уровень в основном тогда,
когда RSI-выход выключен, а `RSIUpper` бессмысленен при `UseRSIExit=0`. Разведи их по разным
фазам — и каждая ось оценивалась бы при зафиксированном значении другой. Дубли в лидерборде
внутри фазы ожидаемы.
```

- [ ] **Step 9: Прогнать гейт**

Run: `./bin/mage ci`
Expected: зелёный.

- [ ] **Step 10: Коммит**

```bash
rm -rf ./reports/T_compat ./reports/T_trailsmoke
git add data/params/rsi_pullback/grid.json \
        data/params/rsi_pullback/cal_trail.json \
        data/params/rsi_pullback/t/params.json \
        docs/rsi_pullback/strategy.md
git commit -m "feat(rsi_pullback): фаза trail в гридах и набор T для проверки совместимости

Фаза exit заменена на trail: RSIUpper, UseRSIExit, UseTrail и TrailDailyATR
метутся вместе, потому что трейл получает шанс связать уровень в основном при
выключенном RSI-выходе, а RSIUpper бессмыслен при UseRSIExit=0.

t/params.json фиксирует набор из отчёта 20260731_134407 без новых ключей:
прогон на нём обязан воспроизводить 67 сделок и PF 1.312, доказывая, что
дефолты новых полей не сдвинули поведение откалиброванных наборов.

Co-Authored-By: Claude Opus 5 <noreply@anthropic.com>"
```

---

## Что НЕ входит в этот план

- **Модель проскальзывания на внутрибарных проколах стопа.** Движок уже учитывает гэп
  открытия через `min(level, open)` (`engine.go:190`), но не проскальзывание при проходе
  уровня внутри бара. На T за 4 года это 20 стопов, из них 6 с проколом глубже 0.5%; при
  фиксированных 0.1% PF падает 1.312 → 1.237. Правка затрагивает все стратегии сразу, и
  смешивать её с трейлом нельзя — иначе не отличить эффект одного от другого.
- **Калибровка и walk-forward.** Идут отдельной задачей после того, как этот план станет
  зелёным. Планка приёмки: pooled OOS profit factor ≥ 1.5 при ≥ 30 сделках, иначе стратегия
  закрывается, как `orb` и `vwap_rev`.
- **Живой слой.** `rsi_pullback` остаётся backtest-only.
