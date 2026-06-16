# HTF-фильтр тренда (4H) для входа reversion — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Добавить опциональный фильтр старшего тренда на 4-часовом таймфрейме, блокирующий вход стратегии reversion, когда 4H-тренд направлен вниз — чтобы не покупать провалы внутри настоящего нисходящего тренда (корневая причина стоп-аутов).

**Architecture:** Загружаем настоящие биржевые Hour4-свечи отдельной серией и прокидываем через движок бэктеста (`Run`/`Trace`) и калибровку (`RunPhases`/`RunGrid`) рядом с дневной серией, не затрагивая дневные поля. Движок на каждом hour1-баре наполняет новые поля `MarketData.HTFCloses/HTFHighs/HTFLows` по правилу no-lookahead. Чистое ядро `core` получает новый int-параметр `HTFTrendEMA` (0 = выкл), считает 4H-EMA в `buildInput` и вставляет gate шагом 0 в `decide`. Поле `int` автоматически попадает в reflection-грид калибровки.

**Tech Stack:** Go 1.25; индикаторы `internal/domain/ema`, `pkg/indicators`; движок бэктеста `internal/domain/backtest`; калибровка `internal/service/backtest`; CLI `cmd/backtest`. Table-driven тесты.

---

## Обзор затрагиваемых файлов

- `internal/domain/backtest/engine.go` — новая чистая функция `visibleCompletedHTF`; `htfInterval`; параметр `htfCandles` в `Run` и `Trace`; наполнение HTF-полей `md`.
- `internal/domain/backtest/engine_test.go` — `TestVisibleCompletedHTF`, `TestEngineSuppliesHTF`; обновление всех вызовов `Run`/`Trace`.
- `internal/service/trading_strategy/scalping/strategy/strategy.go` — поля `HTFCloses/HTFHighs/HTFLows` в `MarketData`.
- `internal/service/backtest/calibrate.go` — параметр `htfCandles` в `runCombos`, `RunGrid`, `RunPhases`.
- `internal/service/backtest/calibrate_test.go`, `internal/service/backtest/basket_test.go` — обновление вызовов.
- `cmd/backtest/main.go` — загрузка `enum.Hour4` для reversion; прокидывание `htfCandles` в `Run`/`Trace`/`RunPhases`/`SplitByTime`; `nil` в basket-пути.
- `internal/service/trading_strategy/reversion/strategy/core/core.go` — поле `Params.HTFTrendEMA`; поля `decideInput`; расчёт в `buildInput`; `htfUptrend`; gate шаг 0 в `decide`; HTF-часть в `entryReason` и `Explain`.
- `internal/service/trading_strategy/reversion/strategy/core/core_test.go` — тесты gate, buildInput, Explain.
- `data/params/{afks,rual,mdmg,plzl,nvtk,sber,ydex,gazp}/reversion_grid.json` — свип `HTFTrendEMA`.

> `reversion_registry.go` и per-ticker `DefaultParams` НЕ меняются: новое int-поле имеет нулевое значение по умолчанию (gate выкл) и подхватывается `json.Unmarshal` автоматически.

---

## Task 1: Чистая функция видимости 4H-баров (no-lookahead)

**Files:**
- Modify: `internal/domain/backtest/engine.go` (после `visibleDailyHighsLows`, ~line 54)
- Test: `internal/domain/backtest/engine_test.go`

- [ ] **Step 1: Написать падающий тест**

В `internal/domain/backtest/engine_test.go` добавить (рядом с `TestVisibleDailyCloses`):

```go
func TestVisibleCompletedHTF(t *testing.T) {
	// Четыре 4H-бара, открытия в UTC: 00:00, 04:00, 08:00, 12:00.
	base := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	htf := []Candle{
		{Time: base, High: 11, Low: 9, Close: 10},
		{Time: base.Add(4 * time.Hour), High: 21, Low: 19, Close: 20},
		{Time: base.Add(8 * time.Hour), High: 31, Low: 29, Close: 30},
		{Time: base.Add(12 * time.Hour), High: 41, Low: 39, Close: 40},
	}

	// На 09:00 закрылись бары 00:00 (→04:00) и 04:00 (→08:00); бар 08:00
	// закроется только в 12:00, текущий формирующийся невидим.
	cur := base.Add(9 * time.Hour)
	closes, highs, lows := visibleCompletedHTF(htf, cur, 4*time.Hour)
	if len(closes) != 2 || closes[0] != 10 || closes[1] != 20 {
		t.Fatalf("closes на 09:00 = %v, want [10 20]", closes)
	}
	if len(highs) != 2 || len(lows) != 2 || highs[1] != 21 || lows[1] != 19 {
		t.Fatalf("highs/lows на 09:00 = %v/%v, want выровнены с closes", highs, lows)
	}

	// Ровно на границе закрытия (08:00) видим только первый бар: правило
	// c.Time.Add(interval) <= cur, т.е. бар 04:00 закрывается В 08:00 включительно.
	if c, _, _ := visibleCompletedHTF(htf, base.Add(8*time.Hour), 4*time.Hour); len(c) != 2 {
		t.Fatalf("на 08:00 видимы %v, want 2 (бары 00:00 и 04:00 закрыты)", c)
	}

	// До первого закрытия — пусто.
	if c, _, _ := visibleCompletedHTF(htf, base.Add(time.Hour), 4*time.Hour); len(c) != 0 {
		t.Fatalf("на 01:00 видимы %v, want пусто", c)
	}
}
```

- [ ] **Step 2: Запустить тест — убедиться, что не компилируется/падает**

Run: `go test ./internal/domain/backtest/ -run TestVisibleCompletedHTF -v`
Expected: FAIL — `undefined: visibleCompletedHTF`.

- [ ] **Step 3: Реализовать функцию**

В `internal/domain/backtest/engine.go` добавить после `visibleDailyHighsLows` (line 54):

```go
// visibleCompletedHTF returns closes/highs/lows of higher-timeframe candles that have
// FULLY closed by cur. A bar opening at c.Time spanning `interval` is closed once
// c.Time.Add(interval) <= cur; the current, still-forming HTF bar is never visible
// (no-lookahead). The three series are index-aligned, oldest-first, so the last element
// is the most recent HTF bar closed at/before cur.
func visibleCompletedHTF(htf []Candle, cur time.Time, interval time.Duration) (closes, highs, lows []float64) {
	for _, c := range htf {
		if !c.Time.Add(interval).After(cur) { // c.Time+interval <= cur
			closes = append(closes, c.Close)
			highs = append(highs, c.High)
			lows = append(lows, c.Low)
		}
	}
	return closes, highs, lows
}
```

- [ ] **Step 4: Запустить тест — убедиться, что проходит**

Run: `go test ./internal/domain/backtest/ -run TestVisibleCompletedHTF -v`
Expected: PASS.

- [ ] **Step 5: Коммит**

```bash
git add internal/domain/backtest/engine.go internal/domain/backtest/engine_test.go
git commit -m "feat(backtest): add visibleCompletedHTF no-lookahead helper

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>"
```

---

## Task 2: HTF-поля MarketData + прокидывание htfCandles через всю pipeline

Самая механическая часть: добавляем позиционный параметр `htfCandles []Candle` в `Run`/`Trace`/`runCombos`/`RunGrid`/`RunPhases`, наполняем HTF-поля `MarketData` в движке и обновляем ВСЕ вызовы. Не-reversion стратегии и basket передают `nil` (поля остаются пустыми — поведение не меняется). Дерево должно компилироваться и все существующие тесты — оставаться зелёными в конце задачи.

**Files:**
- Modify: `internal/service/trading_strategy/scalping/strategy/strategy.go` (`MarketData`)
- Modify: `internal/domain/backtest/engine.go` (`Run`, `Trace`, `htfInterval`)
- Modify: `internal/service/backtest/calibrate.go` (`runCombos`, `RunGrid`, `RunPhases`)
- Modify: `cmd/backtest/main.go` (загрузка Hour4, прокидывание)
- Test: `internal/domain/backtest/engine_test.go` (новый `TestEngineSuppliesHTF` + обновление вызовов)
- Test: `internal/service/backtest/calibrate_test.go`, `internal/service/backtest/basket_test.go`

- [ ] **Step 1: Добавить HTF-поля в MarketData**

В `internal/service/trading_strategy/scalping/strategy/strategy.go`, в `type MarketData struct`, сразу после поля `DailyLows`:

```go
	// HTFCloses are oldest-first closes of COMPLETED higher-timeframe (4H) candles,
	// aligned so the last element is the most recent 4H bar fully closed at/before the
	// current bar. Empty if no HTF data is supplied or the filter is disabled.
	HTFCloses []float64
	// HTFHighs / HTFLows are oldest-first highs/lows of the same COMPLETED 4H candles
	// as HTFCloses (aligned index-for-index). Empty when no HTF data.
	HTFHighs []float64
	HTFLows  []float64
```

- [ ] **Step 2: Написать падающий тест наполнения HTF-серии движком**

В `internal/domain/backtest/engine_test.go` добавить (рядом с `TestEngineSuppliesDailyCloses`). Тест уже использует целевую сигнатуру `Run(s, candles, daily, htf, cfg)`:

```go
func TestEngineSuppliesHTF(t *testing.T) {
	base := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	candles := []Candle{
		{Time: base.Add(5 * time.Hour), Open: 10, High: 10, Low: 10, Close: 10, Volume: 1},
		{Time: base.Add(9 * time.Hour), Open: 10, High: 10, Low: 10, Close: 10, Volume: 1},
	}
	htf := []Candle{
		{Time: base, High: 11, Low: 9, Close: 10},                  // closes at 04:00
		{Time: base.Add(4 * time.Hour), High: 21, Low: 19, Close: 20}, // closes at 08:00
		{Time: base.Add(8 * time.Hour), High: 31, Low: 29, Close: 30}, // closes at 12:00 (unseen)
	}

	var seen [][]float64
	s := scriptedStrategy{lookback: 1, decide: func(md strategy.MarketData) model.Signal {
		seen = append(seen, append([]float64(nil), md.HTFCloses...))
		return model.Signal{Kind: model.SignalNone}
	}}
	Run(s, candles, nil, htf, Config{InitialCash: 100000, Fraction: 1.0, Commission: 0, Lot: 1})

	if len(seen) != 2 {
		t.Fatalf("decided %d bars, want 2", len(seen))
	}
	if len(seen[0]) != 1 || seen[0][0] != 10 {
		t.Errorf("bar 05:00 HTF = %v, want [10]", seen[0])
	}
	if len(seen[1]) != 2 || seen[1][1] != 20 {
		t.Errorf("bar 09:00 HTF = %v, want [10 20]", seen[1])
	}
}
```

- [ ] **Step 3: Запустить тест — убедиться, что не компилируется**

Run: `go test ./internal/domain/backtest/ -run TestEngineSuppliesHTF 2>&1 | head`
Expected: FAIL — слишком много аргументов в `Run` (сигнатура ещё старая).

- [ ] **Step 4: Изменить сигнатуру и тело Run/Trace в engine.go**

В `internal/domain/backtest/engine.go`:

Добавить константу интервала рядом с `mskLoc` (после line 20):

```go
// htfInterval is the higher-timeframe bar span used by visibleCompletedHTF to decide
// which 4H bars have fully closed at the current intraday bar.
const htfInterval = 4 * time.Hour
```

Изменить сигнатуру `Run` (line 84):

```go
func Run(s strategy.Strategy, candles []Candle, dailyCandles []Candle, htfCandles []Candle, cfg Config) Result {
```

В теле `Run`, сразу после строки с `md.TodayHigh, md.TodayLow = todayExtent(...)` (line 97):

```go
		md.HTFCloses, md.HTFHighs, md.HTFLows = visibleCompletedHTF(htfCandles, candles[i].Time, htfInterval)
```

Изменить сигнатуру `Trace` (line 149):

```go
func Trace(s strategy.Strategy, candles []Candle, dailyCandles []Candle, htfCandles []Candle, cfg Config, target time.Time) string {
```

В теле `Trace`, сразу после `md.TodayHigh, md.TodayLow = todayExtent(...)` (line 160):

```go
		md.HTFCloses, md.HTFHighs, md.HTFLows = visibleCompletedHTF(htfCandles, candles[i].Time, htfInterval)
```

- [ ] **Step 5: Обновить все вызовы Run/Trace в engine_test.go**

Найти оставшиеся старые вызовы:

Run: `grep -n "Run(\|Trace(" internal/domain/backtest/engine_test.go`

Каждому вызову `Run(s, candles, <daily>, cfg...)` добавить `nil` (htfCandles) ПОСЛЕ аргумента daily: `Run(s, candles, <daily>, nil, cfg...)`. Например `Run(s, candles, nil, Config{...})` → `Run(s, candles, nil, nil, Config{...})`, а `Run(s, candles, daily, Config{...})` (в `TestEngineSuppliesDailyCloses`) → `Run(s, candles, daily, nil, Config{...})`. `TestEngineSuppliesHTF` уже написан с новой сигнатурой — его не трогать.

- [ ] **Step 6: Изменить calibrate.go (runCombos/RunGrid/RunPhases)**

В `internal/service/backtest/calibrate.go`:

`runCombos` (line 62) — добавить `htfCandles`:

```go
func runCombos(b Binding, combos []any, candles, dailyCandles, htfCandles []backtest.Candle,
	cfg backtest.Config, periodDays float64,
) []CalibResult {
```

Внутри `runCombos` (line 75) изменить вызов:

```go
				res := backtest.Run(b.Build(combos[i]), candles, dailyCandles, htfCandles, cfg)
```

`RunGrid` (line 93):

```go
func RunGrid(b Binding, grid Grid, candles []backtest.Candle, dailyCandles, htfCandles []backtest.Candle,
	cfg backtest.Config, metric string, minTrades int, periodDays float64,
) ([]CalibResult, error) {
```

Внутри `RunGrid` (line 103):

```go
	results := runCombos(b, combos, candles, dailyCandles, htfCandles, cfg, periodDays)
```

`RunPhases` (line 127):

```go
func RunPhases(b Binding, phases []Phase, candles, dailyCandles, htfCandles []backtest.Candle,
	cfg backtest.Config, metric string, minTrades int, periodDays float64,
	onProgress func(PhaseProgress),
) ([]CalibResult, error) {
```

Внутри `RunPhases` (line 148):

```go
		results = rankResults(runCombos(b, combos, candles, dailyCandles, htfCandles, cfg, periodDays), metric, minTrades)
```

- [ ] **Step 7: Изменить cmd/backtest/main.go — загрузка Hour4 и прокидывание**

В `cmd/backtest/main.go`:

После блока загрузки дневных свечей (line 147-150, заканчивается `}`), добавить загрузку Hour4 только для reversion:

```go
	// Reversion's optional 4H HTF trend filter needs real Hour4 candles, loaded with the
	// same year lead-in as the daily series to warm the 4H EMA. Other strategies pass nil.
	var htfCandles []domain.Candle
	if strategyName == "reversion" {
		htfCandles, err = provider.Load(ctx, ticker, share.ID, enum.Hour4, dailyFrom, to, refresh)
		if err != nil {
			return err
		}
	}
```

Изменить вызов `Trace` (line 164):

```go
		fmt.Println(domain.Trace(binding.Build(binding.DefaultParams()), candles, dailyCandles, htfCandles, cfg, target))
```

Изменить вызов `runCalibration` (line 175):

```go
		return runCalibration(binding, calibratePath, candles, dailyCandles, htfCandles, cfg, metric, minTrades, testMonths, periodDays, base,
			metaCommon(ticker, interval, from, to, cfg), to)
```

Изменить вызов `runSingle` (line 179):

```go
	return runSingle(binding, paramsPath, candles, dailyCandles, htfCandles, cfg, periodDays, base,
		metaCommon(ticker, interval, from, to, cfg))
```

`runSingle` сигнатура (line 183) — добавить `htfCandles`:

```go
func runSingle(b svc.Binding, paramsPath string, candles []domain.Candle, dailyCandles, htfCandles []domain.Candle,
	cfg domain.Config, periodDays float64, base string, meta domain.Meta,
) error {
```

Внутри `runSingle` (line 202):

```go
	res := domain.Run(b.Build(params), candles, dailyCandles, htfCandles, cfg)
```

`runCalibration` сигнатура (line 221) — добавить `htfCandles`:

```go
func runCalibration(b svc.Binding, gridPath string, candles []domain.Candle, dailyCandles, htfCandles []domain.Candle,
	cfg domain.Config, metric string, minTrades, testMonths int, periodDays float64, base string, meta domain.Meta, to time.Time,
) error {
```

Внутри `runCalibration` — расширить блок сплита (line 233-248). Заменить:

```go
	gridCandles, gridDaily := candles, dailyCandles
	bestCandles, bestDaily := candles, dailyCandles
	bestDays := periodDays
	gridDays := periodDays
	var boundary time.Time
	if testMonths > 0 {
		boundary = to.AddDate(0, -testMonths, 0)
		testDays := to.Sub(boundary).Hours() / 24
		if testDays >= periodDays {
			return fmt.Errorf("test-months window (%.0f days) must be smaller than the backtest window (%.0f days)", testDays, periodDays)
		}
		gridCandles, bestCandles = svc.SplitByTime(candles, boundary)
		gridDaily, bestDaily = svc.SplitByTime(dailyCandles, boundary)
		bestDays = testDays
		gridDays = periodDays - testDays
	}
```

на:

```go
	gridCandles, gridDaily, gridHTF := candles, dailyCandles, htfCandles
	bestCandles, bestDaily, bestHTF := candles, dailyCandles, htfCandles
	bestDays := periodDays
	gridDays := periodDays
	var boundary time.Time
	if testMonths > 0 {
		boundary = to.AddDate(0, -testMonths, 0)
		testDays := to.Sub(boundary).Hours() / 24
		if testDays >= periodDays {
			return fmt.Errorf("test-months window (%.0f days) must be smaller than the backtest window (%.0f days)", testDays, periodDays)
		}
		gridCandles, bestCandles = svc.SplitByTime(candles, boundary)
		gridDaily, bestDaily = svc.SplitByTime(dailyCandles, boundary)
		gridHTF, bestHTF = svc.SplitByTime(htfCandles, boundary)
		bestDays = testDays
		gridDays = periodDays - testDays
	}
```

Изменить вызов `RunPhases` (line 261):

```go
	results, err := svc.RunPhases(b, phases, gridCandles, gridDaily, gridHTF, cfg, metric, minTrades, gridDays,
		func(p svc.PhaseProgress) {
```

Изменить `domain.Run` для best-комбинации (line 276):

```go
		res := domain.Run(b.Build(best), bestCandles, bestDaily, bestHTF, cfg)
```

- [ ] **Step 8: Обновить basket-путь в main.go (передаёт nil)**

Basket поддерживает только momentum, у которого HTF нет. Передаём `nil`.

В `runBasket`, вызов `RunPhases` (line 438):

```go
		results, err := svc.RunPhases(binding, phases, gridCandles, gridDaily, nil, cfg, metric, minTrades, gridDays, nil)
```

Вызов `domain.Run` (line 449):

```go
		res := domain.Run(binding.Build(best), bestCandles, bestDaily, nil, cfg)
```

- [ ] **Step 9: Обновить вызовы в calibrate_test.go и basket_test.go**

Найти затронутые вызовы:

Run: `grep -rn "RunPhases(\|RunGrid(\|backtest\.Run(\|domain\.Run(" internal/service/backtest/*_test.go`

Каждому вызову `RunPhases(...)` / `RunGrid(...)` добавить `nil` (htfCandles) ПОСЛЕ аргумента `dailyCandles`; каждому `backtest.Run(...)` — `nil` после dailyCandles. Сигнатуры:
- `RunGrid(b, grid, candles, dailyCandles, nil, cfg, metric, minTrades, periodDays)`
- `RunPhases(b, phases, candles, dailyCandles, nil, cfg, metric, minTrades, periodDays, onProgress)`
- `backtest.Run(s, candles, dailyCandles, nil, cfg)`

- [ ] **Step 10: Полная сборка и тесты**

Run: `go build ./... && go test ./internal/domain/backtest/ ./internal/service/backtest/ -run "HTF|Daily|Run|Phase|Grid|Basket|Split" -v 2>&1 | tail -30`
Expected: компиляция без ошибок; `TestEngineSuppliesHTF` PASS; `TestVisibleCompletedHTF` PASS; ранее зелёные тесты остаются зелёными.

Затем общий прогон:

Run: `go test ./... 2>&1 | tail -20`
Expected: PASS (или те же предсуществующие skip/PASS, что и до изменения).

- [ ] **Step 11: Коммит**

```bash
git add internal/service/trading_strategy/scalping/strategy/strategy.go \
        internal/domain/backtest/engine.go internal/domain/backtest/engine_test.go \
        internal/service/backtest/calibrate.go internal/service/backtest/calibrate_test.go \
        internal/service/backtest/basket_test.go cmd/backtest/main.go
git commit -m "feat(backtest): thread Hour4 HTF candle series through engine and calibration

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>"
```

---

## Task 3: Параметр HTFTrendEMA + расчёт 4H-EMA в buildInput

**Files:**
- Modify: `internal/service/trading_strategy/reversion/strategy/core/core.go` (`Params`, `decideInput`, `buildInput`)
- Test: `internal/service/trading_strategy/reversion/strategy/core/core_test.go`

- [ ] **Step 1: Написать падающий тест buildInput**

В `internal/service/trading_strategy/reversion/strategy/core/core_test.go` добавить:

```go
func TestBuildInputHTFGate(t *testing.T) {
	htf := []float64{10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20} // 11 точек, растущие

	// Минимальный валидный набор hour1-свечей для прочих индикаторов.
	closes := make([]float64, 60)
	for i := range closes {
		closes[i] = 100
	}
	md := strategy.MarketData{
		Price: 100, Closes: closes, Highs: closes, Lows: closes,
		Volumes: make([]int64, len(closes)),
		HTFCloses: htf, HTFHighs: htf, HTFLows: htf,
	}

	// Gate выключен (HTFTrendEMA=0): htf-значения не читаются.
	p := defaultParams()
	if in := NewWithParams("T", p).buildInput(md); in.htfOK || in.htfEMA != 0 || in.htfClose != 0 {
		t.Fatalf("HTFTrendEMA=0: want htfOK=false, htfEMA=0, htfClose=0; got %+v/%v/%v", in.htfOK, in.htfEMA, in.htfClose)
	}

	// Gate включён: EMA посчитана, htfClose = последний 4H-close, htfOK=true.
	p.HTFTrendEMA = 5
	in := NewWithParams("T", p).buildInput(md)
	if !in.htfOK {
		t.Fatalf("HTFTrendEMA=5 с достаточными данными: want htfOK=true")
	}
	if in.htfClose != 20 {
		t.Fatalf("htfClose = %v, want 20 (последний HTFCloses)", in.htfClose)
	}
	if in.htfEMA <= 0 {
		t.Fatalf("htfEMA = %v, want >0", in.htfEMA)
	}

	// Недостаточно данных (len < HTFTrendEMA): htfOK=false.
	p.HTFTrendEMA = 50
	if in := NewWithParams("T", p).buildInput(md); in.htfOK {
		t.Fatalf("HTFTrendEMA=50 при 11 точках: want htfOK=false")
	}
}
```

- [ ] **Step 2: Запустить тест — убедиться, что не компилируется**

Run: `go test ./internal/service/trading_strategy/reversion/strategy/core/ -run TestBuildInputHTFGate 2>&1 | head`
Expected: FAIL — `p.HTFTrendEMA undefined` и `in.htfOK undefined`.

- [ ] **Step 3: Добавить поле Params.HTFTrendEMA**

В `core.go`, в `type Params struct`, после поля `StochOverbought` (line 50):

```go
	HTFTrendEMA     int     // период EMA на 4H для фильтра старшего тренда; 0 = выкл
```

- [ ] **Step 4: Добавить поля decideInput**

В `core.go`, в `type decideInput struct`, после поля `volOK` (line 112):

```go
	htfClose    float64 // последний завершённый 4H-close (0 unless HTFTrendEMA>0 и данных хватает)
	htfEMA      float64 // EMA(HTFCloses, HTFTrendEMA), последнее значение; 0 unless gate active
	htfOK       bool    // true когда 4H-EMA прогрета; false -> старший тренд не подтверждён
```

- [ ] **Step 5: Считать 4H-EMA в buildInput**

В `core.go`, внутри `buildInput`, перед `return decideInput{` (line 163), добавить:

```go
	var htfClose, htfEMA float64
	htfOK := false
	if s.p.HTFTrendEMA > 0 && len(md.HTFCloses) >= s.p.HTFTrendEMA {
		if e := ema.Compute(md.HTFCloses, s.p.HTFTrendEMA); len(e) > 0 {
			// Prices are positive, so a real EMA is never 0; a 0 means "not warmed"
			// (same warm-up discipline as lastTwoEMA).
			if v := e[len(e)-1]; v != 0 {
				htfEMA = v
				htfClose = md.HTFCloses[len(md.HTFCloses)-1]
				htfOK = true
			}
		}
	}
```

И в литерале `return decideInput{...}` добавить три поля (после `volOK: volOK,`):

```go
		htfClose:    htfClose,
		htfEMA:      htfEMA,
		htfOK:       htfOK,
```

- [ ] **Step 6: Запустить тест — убедиться, что проходит**

Run: `go test ./internal/service/trading_strategy/reversion/strategy/core/ -run TestBuildInputHTFGate -v`
Expected: PASS.

- [ ] **Step 7: Прогнать весь пакет core (регрессия)**

Run: `go test ./internal/service/trading_strategy/reversion/strategy/core/ 2>&1 | tail`
Expected: PASS (новое поле с нулевым значением не влияет на существующие тесты).

- [ ] **Step 8: Коммит**

```bash
git add internal/service/trading_strategy/reversion/strategy/core/core.go \
        internal/service/trading_strategy/reversion/strategy/core/core_test.go
git commit -m "feat(reversion): compute 4H HTF EMA in buildInput behind HTFTrendEMA param

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>"
```

---

## Task 4: Gate-логика (decide шаг 0), entryReason и Explain

**Files:**
- Modify: `internal/service/trading_strategy/reversion/strategy/core/core.go` (`htfUptrend`, `decide`, `entryReason`, `explainFrom`)
- Test: `internal/service/trading_strategy/reversion/strategy/core/core_test.go`

- [ ] **Step 1: Написать падающие тесты gate**

В `core_test.go` добавить. Хелпер строит проходной вход с восходящим HTF:

```go
// htfPassingInput clears every gate AND has a confirmed up HTF trend.
func htfPassingInput() decideInput {
	in := passingInput()
	in.htfOK = true
	in.htfClose = 110
	in.htfEMA = 100 // close > EMA -> uptrend
	return in
}

func TestHTFGateOffWhenZero(t *testing.T) {
	// HTFTrendEMA=0: gate не читает HTF, вход определяется остальными фильтрами.
	s := NewWithParams("TEST", defaultParams()) // HTFTrendEMA=0
	in := passingInput()                        // htfOK=false, но gate выключен
	if sig := s.decide(in); sig.Kind != model.SignalBuy {
		t.Fatalf("HTFTrendEMA=0: want Buy, got %v", sig.Kind)
	}
}

func TestHTFGatePassesUptrend(t *testing.T) {
	p := defaultParams()
	p.HTFTrendEMA = 20
	s := NewWithParams("TEST", p)
	if sig := s.decide(htfPassingInput()); sig.Kind != model.SignalBuy {
		t.Fatalf("HTF up + все фильтры пройдены: want Buy, got %v", sig.Kind)
	}
}

func TestHTFGateBlocksDowntrend(t *testing.T) {
	p := defaultParams()
	p.HTFTrendEMA = 20
	s := NewWithParams("TEST", p)
	in := htfPassingInput()
	in.htfClose, in.htfEMA = 90, 100 // close < EMA -> downtrend
	if sig := s.decide(in); sig.Kind == model.SignalBuy {
		t.Fatalf("HTF down: want no Buy несмотря на пройденные hour1-фильтры")
	}
}

func TestHTFGateBlocksMissingData(t *testing.T) {
	p := defaultParams()
	p.HTFTrendEMA = 20
	s := NewWithParams("TEST", p)
	in := htfPassingInput()
	in.htfOK = false // не хватило 4H-данных
	if sig := s.decide(in); sig.Kind == model.SignalBuy {
		t.Fatalf("htfOK=false при HTFTrendEMA>0: want no Buy (защитный фильтр блокирует)")
	}
}

func TestHTFGateDoesNotAffectExits(t *testing.T) {
	// Открытая позиция: путь manage не должен зависеть от HTF gate.
	p := defaultParams()
	p.HTFTrendEMA = 20
	s := NewWithParams("TEST", p)
	in := openInput()
	in.htfOK, in.htfClose, in.htfEMA = false, 0, 0 // HTF "не подтверждён"
	if sig := s.decide(in); sig.Kind == model.SignalBuy {
		t.Fatalf("в позиции gate входа не применяется")
	}
	// Никакого ложного выхода от gate: openInput нейтрален -> SignalNone.
	if sig := s.decide(in); sig.Kind == model.SignalSell {
		t.Fatalf("HTF gate не должен порождать выход; got %v/%v", sig.Kind, sig.Reason)
	}
}

func TestExplainHTFBlock(t *testing.T) {
	p := defaultParams()
	p.HTFTrendEMA = 20
	s := NewWithParams("TEST", p)
	in := htfPassingInput()
	in.htfClose, in.htfEMA = 90, 100 // HTF вниз
	out := s.explainFrom(in)
	if !strings.Contains(out, "HTF") || !strings.Contains(out, "ВХОДА НЕТ") {
		t.Fatalf("Explain должен показать блок по HTF: %q", out)
	}
}
```

- [ ] **Step 2: Запустить тесты — убедиться, что падают**

Run: `go test ./internal/service/trading_strategy/reversion/strategy/core/ -run TestHTFGate -v 2>&1 | head`
Expected: FAIL — `TestHTFGateBlocksDowntrend`/`BlocksMissingData` дают Buy (gate ещё не реализован).

- [ ] **Step 3: Добавить htfUptrend и gate шаг 0 в decide**

В `core.go`, рядом с `uptrend` (после line 277):

```go
// htfUptrend reports the higher-timeframe (4H) trend gate: the last completed 4H close
// is above its 4H EMA. htfOK must be true (the EMA is warmed); when false the trend is
// NOT confirmed and the protective gate blocks the entry.
func htfUptrend(in decideInput) bool {
	return in.htfOK && in.htfClose > in.htfEMA
}
```

В `decide`, внутри блока входа (после `if in.pos != nil { return s.manage(in, sig) }`, перед `// 1. Optional trend filter.`, line 287):

```go
	// 0. Optional higher-timeframe (4H) trend filter. When enabled, block the buy unless
	// the 4H trend is confirmed up. Missing/un-warmed 4H data (htfOK=false) also blocks:
	// a protective filter must not trade when it cannot confirm the higher trend.
	if s.p.HTFTrendEMA > 0 && !htfUptrend(in) {
		return sig
	}
```

- [ ] **Step 4: Добавить HTF-часть в entryReason**

В `core.go`, в `entryReason` (line 308), перед `return fmt.Sprintf(...)`, собрать HTF-фрагмент и вставить его в строку. Заменить тело функции:

```go
func (s *Strategy) entryReason(in decideInput) string {
	trend := "выкл"
	if s.p.UseTrend == 1 {
		trend = fmt.Sprintf("EMA%d %.4f > EMA%d %.4f, close %.4f > EMA%d",
			s.p.FastEMA, in.emaFast, s.p.SlowEMA, in.emaSlow, in.price, s.p.SlowEMA)
	}
	htf := "выкл"
	if s.p.HTFTrendEMA > 0 {
		htf = fmt.Sprintf("EMA%d(4H): close %.4f > EMA %.4f", s.p.HTFTrendEMA, in.htfClose, in.htfEMA)
	}
	return fmt.Sprintf(
		"HTF: %s; Тренд: %s; двойное подтверждение перепроданности: RSI(%d) %.2f→%.2f (зона <%.0f) + Stoch%%D(%d,%d) %.2f→%.2f (зона <%.0f)",
		htf, trend,
		s.p.RSIPeriod, in.rsiPrev, in.rsiNow, s.p.RSIOversold,
		s.p.StochKPeriod, s.p.StochDSmooth, in.stochPrev, in.stochNow, s.p.StochOversold,
	)
}
```

- [ ] **Step 5: Добавить gate шаг 0 в Explain**

В `core.go`, в `explainFrom` (line 376), перед `// 1. Optional trend filter.` (line 389):

```go
	// 0. Optional higher-timeframe (4H) trend filter.
	if s.p.HTFTrendEMA > 0 {
		if !htfUptrend(in) {
			return block("HTF(4H): нужно close > EMA%d при прогретой 4H-EMA (htfOK=%v, close=%.4f, EMA=%.4f)",
				s.p.HTFTrendEMA, in.htfOK, in.htfClose, in.htfEMA)
		}
		pass("HTF↑(4H): close %.4f > EMA%d %.4f", in.htfClose, s.p.HTFTrendEMA, in.htfEMA)
	}
```

- [ ] **Step 6: Запустить тесты gate — убедиться, что проходят**

Run: `go test ./internal/service/trading_strategy/reversion/strategy/core/ -run "TestHTFGate|TestExplainHTF" -v`
Expected: все PASS.

- [ ] **Step 7: Прогнать весь пакет core (регрессия)**

Run: `go test ./internal/service/trading_strategy/reversion/strategy/core/ 2>&1 | tail`
Expected: PASS. (Существующие тесты, проверяющие `EntryReason`, ищут подстроки `RSI(14)`/`Stoch` — они остаются в строке, тесты зелёные.)

- [ ] **Step 8: Коммит**

```bash
git add internal/service/trading_strategy/reversion/strategy/core/core.go \
        internal/service/trading_strategy/reversion/strategy/core/core_test.go
git commit -m "feat(reversion): add 4H HTF trend gate as decide step 0 with explain/reason

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>"
```

---

## Task 5: Свип HTFTrendEMA в гридах калибровки (8 тикеров)

Добавляем `HTFTrendEMA` в каждый `data/params/<ticker>/reversion_grid.json`. Значения: `[0, 20, 50, 100]` (4H-периоды: 0=выкл, ~20≈3.3 дня, ~50≈8 дней, ~100≈17 дней). Калибровка per-ticker решит, помогает ли фильтр.

**Files:**
- Modify: `data/params/{afks,rual,mdmg,plzl,nvtk,sber,ydex,gazp}/reversion_grid.json`

- [ ] **Step 1: Осмотреть текущие гриды**

Run: `for f in data/params/*/reversion_grid.json; do echo "== $f =="; cat "$f"; done`
Expected: у каждого есть `phases[].grid` (фазовый формат) либо плоский объект полей.

- [ ] **Step 2: Добавить HTFTrendEMA в фазу `entry` каждого грида**

Для каждого файла добавить ключ `"HTFTrendEMA": [0, 20, 50, 100]` в объект `grid` фазы входа. Пример для `data/params/rual/reversion_grid.json` — было:

```json
{
  "phases": [
    {
      "name": "entry",
      "keepTop": 5,
      "grid": {
        "VolMult": [0.8, 0.9, 1.0, 1.1, 1.2],
        "StopATRMult": [0.8, 0.9, 1.0]
      }
    }
  ]
}
```

станет:

```json
{
  "phases": [
    {
      "name": "entry",
      "keepTop": 5,
      "grid": {
        "VolMult": [0.8, 0.9, 1.0, 1.1, 1.2],
        "StopATRMult": [0.8, 0.9, 1.0],
        "HTFTrendEMA": [0, 20, 50, 100]
      }
    }
  ]
}
```

Применить ту же правку (добавить `"HTFTrendEMA": [0, 20, 50, 100]` в grid первой фазы) ко всем восьми файлам. Для файлов в плоском формате — добавить ключ в корневой объект.

- [ ] **Step 3: Проверить валидность JSON всех гридов**

Run: `for f in data/params/*/reversion_grid.json; do python3 -c "import json,sys; json.load(open('$f'))" && echo "ok $f"; done`
Expected: `ok` для всех восьми файлов.

- [ ] **Step 4: Проверить, что грид парсится движком калибровки (smoke)**

Run: `grep -l "HTFTrendEMA" data/params/*/reversion_grid.json | wc -l`
Expected: `8`.

- [ ] **Step 5: Коммит**

```bash
git add data/params/*/reversion_grid.json
git commit -m "chore(reversion): add HTFTrendEMA sweep to per-ticker calibration grids

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>"
```

---

## Task 6: Финальная верификация сборки и тестов

**Files:** нет изменений — только проверки.

- [ ] **Step 1: Сборка всего проекта**

Run: `go build ./...`
Expected: без ошибок.

- [ ] **Step 2: Полный прогон тестов**

Run: `go test ./... 2>&1 | tail -30`
Expected: PASS (или те же предсуществующие skip, что и до изменения; новых FAIL нет).

- [ ] **Step 3: go vet**

Run: `go vet ./internal/domain/backtest/ ./internal/service/backtest/ ./internal/service/trading_strategy/reversion/... ./cmd/backtest/`
Expected: без замечаний.

- [ ] **Step 4 (опционально, требует токена и сети): валидационная калибровка одного тикера**

Run: `go run ./cmd/backtest -ticker RUAL -strategy reversion -calibrate data/params/rual/reversion_grid.json -out ./reports/RUAL -months 50 -test-months 12 -metric net_pnl`
Expected: отчёт `reports/RUAL/..._calibration.md`; сравнить `net_pnl` и бакеты `ATRSL`/`RSIOS` до/после. Следить, чтобы улучшение держалось на OOS (walk-forward `-test-months 12`), а не было переподгонкой.

> Этот шаг — ручная валидация качества стратегии, не часть автоматического CI. Калибровка не меняет код; победившие per-ticker значения `HTFTrendEMA` пользователь зашивает в соответствующие `DefaultParams` отдельным изменением после анализа отчётов.

---

## Self-Review (выполнено при написании плана)

**Покрытие спека:**
- §1 Новый параметр `HTFTrendEMA int` → Task 3 Step 3. ✓
- §2 Источник данных Hour4 через движок (MarketData-поля, `visibleCompletedHTF`, `Run`/`Trace`, calibrate, loader, basket) → Task 1 + Task 2. ✓
- §3 Логика gate (`htfUptrend`, decide шаг 0 перед UseTrend) → Task 4 Steps 3. ✓
- §4 Блокировка входа при нехватке HTF-данных → Task 4 Step 3 (условие `!htfUptrend` ловит `htfOK=false`); тест `TestHTFGateBlocksMissingData`. ✓
- §5 Прозрачность (entryReason + Explain шаг 0) → Task 4 Steps 4-5. ✓
- §6 Калибровка (свип `[0,20,50,100]`, DefaultParams остаются 0) → Task 5; DefaultParams не трогаем (нулевое значение int). ✓
- §7 Тесты (visibility, gate off/block/pass/missing-data, exits unaffected, buildInput warm-up) → Task 1/3/4 тесты. ✓
- §8 Валидация → Task 6 Step 4. ✓

**Сканирование плейсхолдеров:** код приведён полностью в каждом шаге; TODO/«добавить обработку ошибок» отсутствуют. ✓

**Согласованность типов:** `htfClose`/`htfEMA`/`htfOK` (decideInput) единообразны в Task 3 и 4; `htfCandles []Candle`/`[]domain.Candle` позиционно после `dailyCandles` во всех сигнатурах; `htfInterval = 4*time.Hour` используется только в движке; имя `htfUptrend` совпадает в реализации и тестах. ✓
