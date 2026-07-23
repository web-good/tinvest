# scalping_rsimacd: RSI-порог и трендовый фильтр H1 — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Добавить в стратегию `scalping_rsimacd` два контекстных гейта входа — минимальный RSI на баре MACD-кросса и трендовый фильтр часового таймфрейма — каждый включается/выключается через калибровочный грид.

**Architecture:** Оба гейта — новые числовые поля `core.Params` (грид свипает их рефлексией). Часовой HTF требует снять жёсткую константу 4h в движке бэктеста: добавляем `backtest.Config.HTFInterval` и новый конструктор `AssembleMarketDataWithHTFInterval`, старая `AssembleMarketData` делегирует ему с 4h, поэтому reversion (в том числе живой путь `reversion/live/marketdata`) не трогаем. CLI грузит Hour1-свечи для `scalping_rsimacd` и Hour4 — для `reversion`.

**Tech Stack:** Go 1.25, стандартный `testing`, `tinvest/pkg/indicators`, `tinvest/internal/domain/ema`, mage (`./bin/mage ci`).

## Global Constraints

- Спека: `docs/superpowers/specs/2026-07-23-scalping-rsimacd-entry-filters-design.md` — при расхождении плана и спеки прав план, но расхождение надо озвучить.
- TDD строго: сначала падающий тест, потом минимальная реализация. Каждая задача заканчивается коммитом.
- Никакого lookahead: стратегии виден только полностью закрытый HTF-бар (`c.Time.Add(interval) <= cur`).
- Гейт HTF при нехватке данных — **fail-closed** (вход запрещён). Это осознанное отличие от reversion.
- Комментарии в коде — на английском (как весь пакет `core`), пользовательские строки (`EntryReason`, `Explain`, docs) — на русском.
- Обратная совместимость: нулевой `Config.HTFInterval` означает 4 часа; поведение reversion не меняется.
- Существующие тесты `core_test.go` не переписывать; `testParams()` расширяется явным выключением новых гейтов.
- Проверка перед коммитом задачи: `go test ./internal/...` для затронутых пакетов; финальный гейт — `./bin/mage ci`.

## File Structure

| Файл | Ответственность | Действие |
|---|---|---|
| `internal/domain/backtest/types.go` | `Config` + новое поле `HTFInterval` и метод `htfSpan()` | Modify |
| `internal/domain/backtest/engine.go` | `AssembleMarketDataWithHTFInterval`, прокидка интервала в `Run`/`Trace` | Modify |
| `internal/domain/backtest/engine_test.go` | тесты видимости HTF при часовом интервале и дефолте | Modify |
| `internal/service/trading_strategy/scalping_rsimacd/strategy/core/core.go` | `Params.RSIEntryMin`, `Params.HTFTrendEMA`, гейты в `enter`, строки в `Explain`/`entryReason` | Modify |
| `internal/service/trading_strategy/scalping_rsimacd/strategy/core/core_test.go` | тесты обоих гейтов | Modify |
| `cmd/backtest/main.go` | загрузка Hour1 для `scalping_rsimacd`, `cfg.HTFInterval` | Modify |
| `data/params/scalping_rsimacd/grid.json` | фаза `context` | Modify |
| `internal/service/backtest/scalping_rsimacd_grid_test.go` | тест: все ключи грида существуют в `core.Params` | Create |
| `docs/scalping_rsimacd/strategy.md` | описание гейтов, дефолтов, fail-closed, H1-данных | Modify |

---

### Task 1: Настраиваемый интервал HTF в движке бэктеста

**Files:**
- Modify: `internal/domain/backtest/types.go:17-24`
- Modify: `internal/domain/backtest/engine.go:12-14`, `:104-115`, `:170-179`, `:258-269`
- Test: `internal/domain/backtest/engine_test.go`

**Interfaces:**
- Consumes: существующие `visibleCompletedHTF(htf []Candle, cur time.Time, interval time.Duration)`, `AssembleMarketData(window, daily, htf []Candle, cur time.Time) strategy.MarketData`.
- Produces:
  - `Config.HTFInterval time.Duration` — 0 означает 4 часа;
  - `(Config).htfSpan() time.Duration` — неэкспортируемый резолвер дефолта;
  - `AssembleMarketDataWithHTFInterval(window, daily, htf []Candle, cur time.Time, htfSpan time.Duration) strategy.MarketData`.

- [ ] **Step 1: Написать падающие тесты**

Добавить в конец `internal/domain/backtest/engine_test.go`:

```go
func TestConfigHTFSpanDefaultsTo4H(t *testing.T) {
	if got := (Config{}).htfSpan(); got != 4*time.Hour {
		t.Fatalf("zero HTFInterval must mean 4h, got %v", got)
	}
	if got := (Config{HTFInterval: time.Hour}).htfSpan(); got != time.Hour {
		t.Fatalf("explicit HTFInterval must win, got %v", got)
	}
}

func TestAssembleMarketDataWithHourlyHTF(t *testing.T) {
	base := time.Date(2026, 7, 20, 10, 0, 0, 0, time.UTC)
	window := []Candle{{Time: base.Add(2 * time.Hour), Open: 10, High: 11, Low: 9, Close: 10}}
	htf := []Candle{
		{Time: base, High: 2, Low: 1, Close: 1.5},
		{Time: base.Add(time.Hour), High: 3, Low: 2, Close: 2.5},
		{Time: base.Add(2 * time.Hour), High: 4, Low: 3, Close: 3.5},
	}
	cur := base.Add(2 * time.Hour) // третий часовой бар ещё формируется

	md := AssembleMarketDataWithHTFInterval(window, nil, htf, cur, time.Hour)
	if len(md.HTFCloses) != 2 {
		t.Fatalf("hourly HTF: got %d completed bars, want 2", len(md.HTFCloses))
	}
	if md.HTFCloses[len(md.HTFCloses)-1] != 2.5 {
		t.Fatalf("last completed hourly close = %v want 2.5", md.HTFCloses[len(md.HTFCloses)-1])
	}
	if len(md.HTFHighs) != 2 || len(md.HTFLows) != 2 {
		t.Fatalf("highs/lows must stay index-aligned with closes: %d/%d", len(md.HTFHighs), len(md.HTFLows))
	}

	// Прежняя 4-часовая семантика: ни один бар ещё не закрыт.
	if md4 := AssembleMarketData(window, nil, htf, cur); len(md4.HTFCloses) != 0 {
		t.Fatalf("4h default: got %d completed bars, want 0", len(md4.HTFCloses))
	}
}
```

- [ ] **Step 2: Запустить тесты и убедиться, что они падают**

Run: `go test ./internal/domain/backtest/ -run 'TestConfigHTFSpan|TestAssembleMarketDataWithHourlyHTF' -v`
Expected: FAIL — `undefined: AssembleMarketDataWithHTFInterval`, `Config.htfSpan undefined`, `unknown field HTFInterval`.

- [ ] **Step 3: Добавить поле и резолвер в `types.go`**

В структуру `Config` (после `RiskFractionPct`) добавить:

```go
	// HTFInterval is the bar-span of the higher-timeframe series passed to Run/Trace.
	// Zero means 4 hours — the legacy span reversion was built on.
	HTFInterval time.Duration
```

И следом за объявлением `Config`:

```go
// htfSpan returns the configured higher-timeframe bar span, defaulting to 4 hours so
// callers written before the field existed keep their behavior.
func (c Config) htfSpan() time.Duration {
	if c.HTFInterval > 0 {
		return c.HTFInterval
	}
	return defaultHTFInterval
}
```

- [ ] **Step 4: Прокинуть интервал в `engine.go`**

Переименовать константу и добавить новый конструктор. В `engine.go:12-14` заменить:

```go
// defaultHTFInterval is the HTF bar-span assumed when Config.HTFInterval is unset: the
// Hour4 series reversion was built on. Run/Trace pass Config.htfSpan() to
// visibleCompletedHTF at each bar to decide which HTF bars have fully closed.
const defaultHTFInterval = 4 * time.Hour
```

В `Run` (строка ~114) и `Trace` (строка ~178) заменить вызов
`AssembleMarketData(candles[i-l+1:i+1], dailyCandles, htfCandles, candles[i].Time)` на:

```go
		md := AssembleMarketDataWithHTFInterval(candles[i-l+1:i+1], dailyCandles, htfCandles, candles[i].Time, cfg.htfSpan())
```

В конце файла заменить тело `AssembleMarketData` на делегирование и добавить новый конструктор:

```go
// AssembleMarketData builds the per-bar snapshot with the default 4H higher-timeframe
// span. Kept for callers built on the Hour4 series (reversion, including its live
// market-data assembler).
func AssembleMarketData(window, daily, htf []Candle, cur time.Time) strategy.MarketData {
	return AssembleMarketDataWithHTFInterval(window, daily, htf, cur, defaultHTFInterval)
}

// AssembleMarketDataWithHTFInterval builds the per-bar snapshot from an oldest-first
// window plus completed-daily and higher-timeframe series, identically to Run's per-bar
// assembly — minus TodayHigh/TodayLow, which the caller sets separately. cur is the
// open-time of the current (latest) bar; it anchors the no-lookahead completeness test
// for the daily and HTF series. htfSpan is the bar-span of the htf series (e.g. 1h for
// scalping_rsimacd, 4h for reversion).
func AssembleMarketDataWithHTFInterval(window, daily, htf []Candle, cur time.Time, htfSpan time.Duration) strategy.MarketData {
	md := buildMarketData(window)
	md.DailyCloses = visibleDailyCloses(daily, cur, mskLoc)
	md.DailyHighs, md.DailyLows = visibleDailyHighsLows(daily, cur, mskLoc)
	md.HTFCloses, md.HTFHighs, md.HTFLows = visibleCompletedHTF(htf, cur, htfSpan)
	return md
}
```

Если после переименования константы `htfInterval` где-то остались ссылки — компилятор укажет; заменить их на `defaultHTFInterval`.

- [ ] **Step 5: Запустить тесты пакета**

Run: `go test ./internal/domain/backtest/ ./internal/service/trading_strategy/reversion/...`
Expected: PASS (в том числе прежние `TestVisibleCompletedHTF`, `TestAssembleMarketData_MatchesPerBarFields` и live-тесты reversion).

- [ ] **Step 6: Коммит**

```bash
git add internal/domain/backtest/types.go internal/domain/backtest/engine.go internal/domain/backtest/engine_test.go
git commit -m "feat(backtest): configurable higher-timeframe bar span via Config.HTFInterval"
```

---

### Task 2: Гейт RSIEntryMin на баре MACD-кросса

**Files:**
- Modify: `internal/service/trading_strategy/scalping_rsimacd/strategy/core/core.go` (структура `Params`, `DefaultParams`, `enter`, `entryReason`, `Explain`)
- Test: `internal/service/trading_strategy/scalping_rsimacd/strategy/core/core_test.go`

**Interfaces:**
- Consumes: `indicators.RSISeries(closes []float64, period int) []float64`; тестовые хелперы `testParams()`, `declineThenRally`, `sessionTimes`, `mdPrefix`, `firstBuy` — уже есть в `core_test.go`.
- Produces: `Params.RSIEntryMin float64` (0 = гейт выключен, дефолт 50).

- [ ] **Step 1: Написать падающие тесты**

Сначала расширить существующий хелпер `testParams()` (около строки 118 `core_test.go`), явно выключив оба новых гейта, чтобы старые сценарии продолжали проверять геометрию входа:

```go
// testParams are permissive defaults for entry tests: the risk sanity bounds are opened
// up so synthetic series never trip them, the stop sits exactly on the trigger low, and
// both context gates (RSI threshold, H1 trend) are off unless a test turns them on.
func testParams() Params {
	p := DefaultParams()
	p.MinRiskATR = 0
	p.MaxRiskATR = 100
	p.StopBufferATR = 0
	p.RSIEntryMin = 0
	p.HTFTrendEMA = 0
	return p
}
```

Добавить в конец `core_test.go`:

```go
// baselineEntryBar returns the series, times and the bar index on which the permissive
// testParams() strategy takes its first entry. Gate tests build on it so that a "no
// entry" assertion can never be vacuous.
func baselineEntryBar(t *testing.T) (highs, lows, closes []float64, times []time.Time, bar int) {
	t.Helper()
	highs, lows, closes = declineThenRally(60, 8, 100, 0.5, 0.2)
	times = sessionTimes(len(closes))
	_, bar, ok := firstBuy(NewWithParams("TEST", testParams()), highs, lows, closes, times)
	if !ok {
		t.Fatalf("baseline entry missing; gate assertions would be vacuous")
	}
	return highs, lows, closes, times, bar
}

func TestDefaultParamsEnableRSIEntryMin(t *testing.T) {
	if got := DefaultParams().RSIEntryMin; got != 50 {
		t.Fatalf("DefaultParams().RSIEntryMin = %v want 50", got)
	}
}

func TestRSIEntryMinBlocksWeakCross(t *testing.T) {
	highs, lows, closes, times, bar := baselineEntryBar(t)
	rsi := indicators.RSISeries(closes[:bar+1], testParams().RSIPeriod)
	at := rsi[bar]

	p := testParams()
	p.RSIEntryMin = at + 1 // порог выше фактического RSI на баре кросса
	sig := NewWithParams("TEST", p).Decide(mdPrefix(highs, lows, closes, times, bar))
	if sig.Kind == model.SignalBuy {
		t.Fatalf("entry fired with RSI %.2f below the %.2f threshold", at, p.RSIEntryMin)
	}
}

func TestRSIEntryMinAllowsStrongCross(t *testing.T) {
	highs, lows, closes, times, bar := baselineEntryBar(t)
	rsi := indicators.RSISeries(closes[:bar+1], testParams().RSIPeriod)
	at := rsi[bar]

	p := testParams()
	p.RSIEntryMin = at - 1 // порог ниже фактического RSI
	sig := NewWithParams("TEST", p).Decide(mdPrefix(highs, lows, closes, times, bar))
	if sig.Kind != model.SignalBuy {
		t.Fatalf("entry blocked though RSI %.2f clears the %.2f threshold", at, p.RSIEntryMin)
	}
	if !strings.Contains(sig.EntryReason, "RSI на кроссе") {
		t.Fatalf("EntryReason must mention the RSI threshold gate, got %q", sig.EntryReason)
	}
}

func TestRSIEntryMinZeroDisablesGate(t *testing.T) {
	highs, lows, closes, times, bar := baselineEntryBar(t)
	p := testParams()
	p.RSIEntryMin = 0
	sig := NewWithParams("TEST", p).Decide(mdPrefix(highs, lows, closes, times, bar))
	if sig.Kind != model.SignalBuy {
		t.Fatalf("RSIEntryMin=0 must not block anything")
	}
}

func TestExplainReportsRSIEntryMin(t *testing.T) {
	highs, lows, closes, times, bar := baselineEntryBar(t)
	p := testParams()
	p.RSIEntryMin = 50
	out := NewWithParams("TEST", p).Explain(mdPrefix(highs, lows, closes, times, bar))
	if !strings.Contains(out, "RSI на кроссе") {
		t.Fatalf("Explain must report the RSI threshold gate, got:\n%s", out)
	}
}
```

- [ ] **Step 2: Запустить тесты и убедиться, что они падают**

Run: `go test ./internal/service/trading_strategy/scalping_rsimacd/... -run 'RSIEntryMin' -v`
Expected: FAIL — `p.RSIEntryMin undefined (type Params has no field or method RSIEntryMin)`.

- [ ] **Step 3: Реализовать поле и гейт**

В `Params` (после `MACDConfirmBars`) добавить:

```go
	RSIEntryMin     float64 // minimum RSI on the MACD-cross bar; 0 disables the gate (grid 0/50/55)
```

В `DefaultParams()` (после `MACDConfirmBars: 3,`) добавить:

```go
		RSIEntryMin:     50,
```

В `enter`, сразу после вычисления ряда RSI и ПЕРЕД поиском триггера (после блока
`rsi := indicators.RSISeries(...)`/`if len(rsi) != n`), вставить:

```go
	// 3a. momentum gate: the cross must happen with RSI already above the threshold.
	if s.p.RSIEntryMin > 0 && rsi[i] <= s.p.RSIEntryMin {
		return sig
	}
```

Нумерация шагов-комментариев в `enter` после вставки: `// 3.` (ряд RSI + порог) остаётся
как есть, вставленный блок помечается `// 3a.`, остальные номера (`// 4.` ATR, `// 5.`
triggerAlive, `// 6.` риск) не трогаем.

В `entryReason` добавить параметр порога и фрагмент. Заменить сигнатуру и тело:

```go
// entryReason renders the rationale shown in the trade journal. barsAgo is the distance
// from the RSI trigger bar to the entry bar (0 = same bar). Optional gates contribute a
// fragment only when they are enabled.
func (s *Strategy) entryReason(barsAgo int, rsiNow, stop, entry, tp, atr float64) string {
	var extra string
	if s.p.RSIEntryMin > 0 {
		extra += fmt.Sprintf("; RSI на кроссе %.1f > порога %.0f", rsiNow, s.p.RSIEntryMin)
	}
	return fmt.Sprintf(
		"RSI(%d) вышел вверх из зоны %.0f (сейчас %.1f), через %d бар(ов) MACD(%d,%d,%d) пересёкся ниже нуля; вход %.4f, стоп %.4f (лоу свечи кросса, буфер %.2f×ATR), тейк %.4f (RR=%.2f), ATR=%.4f%s",
		s.p.RSIPeriod, s.p.RSIOversold, rsiNow, barsAgo, s.p.MACDFast, s.p.MACDSlow, s.p.MACDSignal,
		entry, stop, s.p.StopBufferATR, tp, s.p.RR, atr, extra,
	)
}
```

В `Explain`, сразу после строки, печатающей текущий RSI (`fmt.Fprintf(&sb, "RSI(%d) сейчас %.1f, зона %.0f\n", ...)`), добавить:

```go
	if s.p.RSIEntryMin > 0 {
		fmt.Fprintf(&sb, "RSI на кроссе %.1f > порога %.0f? %v\n",
			rsi[i], s.p.RSIEntryMin, rsi[i] > s.p.RSIEntryMin)
	} else {
		fmt.Fprintf(&sb, "RSI на кроссе: порог выключен (RSIEntryMin=0)\n")
	}
```

Тест `TestExplainReportsRSIEntryMin` требует подстроку «RSI на кроссе» — она есть в обеих ветках.

- [ ] **Step 4: Запустить тесты пакета**

Run: `go test ./internal/service/trading_strategy/scalping_rsimacd/... -v`
Expected: PASS, включая все ранее существовавшие тесты.

- [ ] **Step 5: Коммит**

```bash
git add internal/service/trading_strategy/scalping_rsimacd/strategy/core/
git commit -m "feat(scalping-rsimacd): RSI threshold gate on the MACD-cross bar"
```

---

### Task 3: Гейт тренда часового таймфрейма (HTFTrendEMA)

**Files:**
- Modify: `internal/service/trading_strategy/scalping_rsimacd/strategy/core/core.go` (`Params`, `DefaultParams`, `enter`, `entryReason`, `Explain`, новый метод `htfTrendOK`)
- Test: `internal/service/trading_strategy/scalping_rsimacd/strategy/core/core_test.go`

**Interfaces:**
- Consumes: `ema.Compute(closes []float64, period int) []float64` из `tinvest/internal/domain/ema` (результат той же длины, что вход; позиции до `period-1` — нули); `strategy.MarketData.HTFCloses`; хелпер `baselineEntryBar` из Task 2.
- Produces:
  - `Params.HTFTrendEMA int` (0 = гейт выключен, дефолт 0);
  - `(*Strategy).htfTrendOK(md strategy.MarketData) (ok bool, htfClose, htfEMA float64, haveData bool)` — `haveData=false` означает нехватку H1-истории (тогда `ok=false`, fail-closed).

- [ ] **Step 1: Написать падающие тесты**

Добавить в конец `core_test.go`:

```go
// htfCloses returns n synthetic hourly closes moving by step per bar from start
// (negative step = downtrend), so the last close sits above (up) or below (down) its EMA.
func htfCloses(n int, start, step float64) []float64 {
	out := make([]float64, n)
	for i := range out {
		out[i] = start + float64(i)*step
	}
	return out
}

func TestDefaultParamsLeaveHTFGateOff(t *testing.T) {
	if got := DefaultParams().HTFTrendEMA; got != 0 {
		t.Fatalf("DefaultParams().HTFTrendEMA = %d want 0 (gate off by default)", got)
	}
}

func TestHTFTrendGateAllowsUptrend(t *testing.T) {
	highs, lows, closes, times, bar := baselineEntryBar(t)
	p := testParams()
	p.HTFTrendEMA = 20

	md := mdPrefix(highs, lows, closes, times, bar)
	md.HTFCloses = htfCloses(60, 100, 0.5) // растущий H1: цена выше EMA
	sig := NewWithParams("TEST", p).Decide(md)
	if sig.Kind != model.SignalBuy {
		t.Fatalf("entry blocked though H1 close sits above its EMA")
	}
	if !strings.Contains(sig.EntryReason, "H1") {
		t.Fatalf("EntryReason must mention the H1 trend gate, got %q", sig.EntryReason)
	}
}

func TestHTFTrendGateBlocksDowntrend(t *testing.T) {
	highs, lows, closes, times, bar := baselineEntryBar(t)
	p := testParams()
	p.HTFTrendEMA = 20

	md := mdPrefix(highs, lows, closes, times, bar)
	md.HTFCloses = htfCloses(60, 130, -0.5) // падающий H1: цена ниже EMA
	if sig := NewWithParams("TEST", p).Decide(md); sig.Kind == model.SignalBuy {
		t.Fatalf("entry fired against a falling H1 trend")
	}
}

func TestHTFTrendGateFailsClosedWithoutData(t *testing.T) {
	highs, lows, closes, times, bar := baselineEntryBar(t)
	p := testParams()
	p.HTFTrendEMA = 20

	cases := map[string][]float64{
		"nil":   nil,
		"short": htfCloses(10, 100, 0.5), // короче периода EMA
	}
	for name, series := range cases {
		t.Run(name, func(t *testing.T) {
			md := mdPrefix(highs, lows, closes, times, bar)
			md.HTFCloses = series
			if sig := NewWithParams("TEST", p).Decide(md); sig.Kind == model.SignalBuy {
				t.Fatalf("gate must fail closed when H1 history is missing")
			}
		})
	}
}

func TestHTFTrendGateOffIgnoresMissingData(t *testing.T) {
	highs, lows, closes, times, bar := baselineEntryBar(t)
	p := testParams()
	p.HTFTrendEMA = 0

	md := mdPrefix(highs, lows, closes, times, bar)
	md.HTFCloses = nil
	if sig := NewWithParams("TEST", p).Decide(md); sig.Kind != model.SignalBuy {
		t.Fatalf("HTFTrendEMA=0 must not require H1 data")
	}
}

func TestExplainReportsHTFGate(t *testing.T) {
	highs, lows, closes, times, bar := baselineEntryBar(t)
	p := testParams()
	p.HTFTrendEMA = 20

	md := mdPrefix(highs, lows, closes, times, bar)
	md.HTFCloses = nil
	out := NewWithParams("TEST", p).Explain(md)
	if !strings.Contains(out, "нет данных H1") {
		t.Fatalf("Explain must report missing H1 data, got:\n%s", out)
	}

	md.HTFCloses = htfCloses(60, 100, 0.5)
	out = NewWithParams("TEST", p).Explain(md)
	if !strings.Contains(out, "тренд H1") {
		t.Fatalf("Explain must report the H1 trend verdict, got:\n%s", out)
	}
}
```

- [ ] **Step 2: Запустить тесты и убедиться, что они падают**

Run: `go test ./internal/service/trading_strategy/scalping_rsimacd/... -run 'HTF' -v`
Expected: FAIL — `p.HTFTrendEMA undefined (type Params has no field or method HTFTrendEMA)`.

- [ ] **Step 3: Реализовать поле, резолвер и гейт**

Добавить импорт `"tinvest/internal/domain/ema"` в `core.go`.

В `Params` (после `RSIEntryMin`) добавить:

```go
	HTFTrendEMA     int     // EMA period on completed H1 closes; entry needs close > EMA. 0 disables the gate (grid 0/20/50/100)
```

`DefaultParams()` оставить без поля (нулевое значение = гейт выключен), но добавить строку
для явности:

```go
		HTFTrendEMA:     0,
```

Новый метод (разместить рядом с `triggerAlive`):

```go
// htfTrendOK evaluates the higher-timeframe (H1) trend gate. It reports the verdict, the
// last completed H1 close, the EMA value, and whether there was enough H1 history at all.
// The gate is FAIL-CLOSED: with the gate enabled and insufficient H1 data it returns
// ok=false, haveData=false. Silently passing (as reversion does) would let a calibration
// run on a period without H1 history collect trades that were effectively unfiltered and
// inflate the profit factor.
func (s *Strategy) htfTrendOK(md strategy.MarketData) (ok bool, htfClose, htfEMA float64, haveData bool) {
	if s.p.HTFTrendEMA <= 0 {
		return true, 0, 0, true
	}
	if len(md.HTFCloses) < s.p.HTFTrendEMA {
		return false, 0, 0, false
	}
	e := ema.Compute(md.HTFCloses, s.p.HTFTrendEMA)
	if len(e) == 0 {
		return false, 0, 0, false
	}
	htfEMA = e[len(e)-1]
	htfClose = md.HTFCloses[len(md.HTFCloses)-1]
	if htfEMA <= 0 {
		return false, htfClose, htfEMA, false
	}
	return htfClose > htfEMA, htfClose, htfEMA, true
}
```

В `enter`, сразу ПОСЛЕ блока поиска RSI-триггера (`trig, ok := s.lastRSITrigger(...)`) и
ПЕРЕД расчётом ATR, вставить:

```go
	// higher-timeframe trend gate (fail-closed when H1 history is missing).
	htfOK, htfClose, htfEMA, _ := s.htfTrendOK(md)
	if !htfOK {
		return sig
	}
```

Внимание: локальная переменная `ok` уже занята результатом `lastRSITrigger` — использовать
именно `htfOK`, как показано.

В `entryReason` добавить фрагмент про H1. Заменить сигнатуру и вызов:

```go
func (s *Strategy) entryReason(barsAgo int, rsiNow, stop, entry, tp, atr, htfClose, htfEMA float64) string {
	var extra string
	if s.p.RSIEntryMin > 0 {
		extra += fmt.Sprintf("; RSI на кроссе %.1f > порога %.0f", rsiNow, s.p.RSIEntryMin)
	}
	if s.p.HTFTrendEMA > 0 {
		extra += fmt.Sprintf("; тренд H1 вверх (close %.4f > EMA(%d) %.4f)", htfClose, s.p.HTFTrendEMA, htfEMA)
	}
	return fmt.Sprintf(
		"RSI(%d) вышел вверх из зоны %.0f (сейчас %.1f), через %d бар(ов) MACD(%d,%d,%d) пересёкся ниже нуля; вход %.4f, стоп %.4f (лоу свечи кросса, буфер %.2f×ATR), тейк %.4f (RR=%.2f), ATR=%.4f%s",
		s.p.RSIPeriod, s.p.RSIOversold, rsiNow, barsAgo, s.p.MACDFast, s.p.MACDSlow, s.p.MACDSignal,
		entry, stop, s.p.StopBufferATR, tp, s.p.RR, atr, extra,
	)
}
```

Вызов в `enter` становится:

```go
	sig.EntryReason = s.entryReason(i-trig, rsi[i], stop, entry, tp, atr, htfClose, htfEMA)
```

В `Explain`, после блока RSI-триггера и перед строкой ATR, добавить:

```go
	if s.p.HTFTrendEMA > 0 {
		htfOK, htfClose, htfEMA, haveData := s.htfTrendOK(md)
		if !haveData {
			fmt.Fprintf(&sb, "тренд H1: нет данных H1 (нужно ≥ %d закрытых часовых баров, есть %d) -> вход запрещён\n",
				s.p.HTFTrendEMA, len(md.HTFCloses))
		} else {
			fmt.Fprintf(&sb, "тренд H1: close %.4f > EMA(%d) %.4f? %v\n", htfClose, s.p.HTFTrendEMA, htfEMA, htfOK)
		}
	} else {
		sb.WriteString("тренд H1: фильтр выключен (HTFTrendEMA=0)\n")
	}
```

Тест `TestExplainReportsHTFGate` требует подстроки «нет данных H1» и «тренд H1» — обе есть.

- [ ] **Step 4: Запустить тесты пакета**

Run: `go test ./internal/service/trading_strategy/scalping_rsimacd/... -v`
Expected: PASS.

- [ ] **Step 5: Коммит**

```bash
git add internal/service/trading_strategy/scalping_rsimacd/strategy/core/
git commit -m "feat(scalping-rsimacd): fail-closed H1 trend gate on the entry"
```

---

### Task 4: Проводка H1-свечей в CLI, грид и документация

**Files:**
- Modify: `cmd/backtest/main.go:186-195` (загрузка HTF) и `:194` (сборка `cfg`)
- Modify: `data/params/scalping_rsimacd/grid.json`
- Create: `internal/service/backtest/scalping_rsimacd_grid_test.go`
- Modify: `docs/scalping_rsimacd/strategy.md`

**Interfaces:**
- Consumes: `Config.HTFInterval` (Task 1), `core.Params.RSIEntryMin` (Task 2), `core.Params.HTFTrendEMA` (Task 3); `svc.ParsePhases(raw []byte) ([]Phase, error)`; неэкспортируемый `applyField(params any, name string, value float64) (any, error)` из `internal/service/backtest/calibrate.go` (тест лежит в том же пакете `backtest`).
- Produces: ничего для последующих задач — это финальная задача.

- [ ] **Step 1: Написать падающий тест на грид**

Создать `internal/service/backtest/scalping_rsimacd_grid_test.go`:

```go
package backtest

import (
	"os"
	"testing"

	"tinvest/internal/service/trading_strategy/scalping_rsimacd/strategy/core"
)

// TestScalpingRSIMACDGridFieldsExist parses the shipped calibration grid and applies every
// swept value to core.Params. A typo in the JSON would otherwise surface only mid-run.
func TestScalpingRSIMACDGridFieldsExist(t *testing.T) {
	raw, err := os.ReadFile("../../../data/params/scalping_rsimacd/grid.json")
	if err != nil {
		t.Fatalf("read grid: %v", err)
	}
	phases, err := ParsePhases(raw)
	if err != nil {
		t.Fatalf("parse phases: %v", err)
	}
	var names []string
	for _, ph := range phases {
		for name, values := range ph.Grid {
			names = append(names, name)
			for _, v := range values {
				if _, err := applyField(core.DefaultParams(), name, v); err != nil {
					t.Fatalf("phase %q: apply %s=%v: %v", ph.Name, name, v, err)
				}
			}
		}
	}
	wantSwept := []string{"RSIEntryMin", "HTFTrendEMA"}
	for _, want := range wantSwept {
		found := false
		for _, got := range names {
			if got == want {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("grid must sweep %s", want)
		}
	}
}
```

- [ ] **Step 2: Запустить тест и убедиться, что он падает**

Run: `go test ./internal/service/backtest/ -run TestScalpingRSIMACDGridFieldsExist -v`
Expected: FAIL — «grid must sweep RSIEntryMin».

- [ ] **Step 3: Добавить фазу `context` в грид**

В `data/params/scalping_rsimacd/grid.json` вставить фазу между `entry` и `risk` и обновить `_comment`:

```json
{
  "_comment": "scalping_rsimacd phased grid. Phase 1 sweeps the entry geometry (RSI length/oversold zone + the MACD confirmation window); phase 2 sweeps the context gates (minimum RSI on the MACD-cross bar and the H1 trend EMA, 0 = gate off); phase 3 sweeps risk and exits (RR, stop buffer, stochastic-exit ablation). MACD(3,6,9), the stochastic (14,3,3) and the session bounds are fixed by the strategy definition and are deliberately NOT swept — the grid stays at 27+12+12 combos to keep the fitting surface small. The context phase runs before risk because the gates change the trade sample RR is then tuned on. Judge on pooled OOS profit factor from a walk-forward, never on the in-sample best.",
  "phases": [
    {
      "name": "entry",
      "keepTop": 6,
      "grid": {
        "RSIPeriod": [3, 4, 5],
        "RSIOversold": [20, 25, 30],
        "MACDConfirmBars": [2, 3, 4]
      }
    },
    {
      "name": "context",
      "keepTop": 6,
      "grid": {
        "RSIEntryMin": [0, 50, 55],
        "HTFTrendEMA": [0, 20, 50, 100]
      }
    },
    {
      "name": "risk",
      "grid": {
        "RR": [2.0, 2.5, 3.0],
        "StopBufferATR": [0, 0.1],
        "EnableStochExit": [0, 1]
      }
    }
  ]
}
```

- [ ] **Step 4: Запустить тест на грид**

Run: `go test ./internal/service/backtest/ -run TestScalpingRSIMACDGridFieldsExist -v`
Expected: PASS.

- [ ] **Step 5: Прокинуть Hour1-свечи в CLI**

В `cmd/backtest/main.go` заменить блок загрузки HTF (строки ~186-195):

```go
	// The optional HTF trend filters need real higher-timeframe candles, loaded with the
	// same year lead-in as the daily series to warm the HTF EMA: Hour4 for reversion,
	// Hour1 for scalping_rsimacd. Other strategies pass nil.
	var (
		htfCandles  []domain.Candle
		htfInterval time.Duration
	)
	switch strategyName {
	case "reversion":
		htfInterval = 4 * time.Hour
		htfCandles, err = provider.Load(ctx, ticker, share.ID, enum.Hour4, dailyFrom, to, refresh)
	case "scalping_rsimacd":
		htfInterval = time.Hour
		htfCandles, err = provider.Load(ctx, ticker, share.ID, enum.Hour1, dailyFrom, to, refresh)
	}
	if err != nil {
		return err
	}
```

И в сборке конфига (следующая строка) добавить поле:

```go
	cfg := domain.Config{InitialCash: cash, Fraction: fraction, Commission: commission, Lot: share.Lot, RiskFractionPct: riskPct, HTFInterval: htfInterval}
```

Нулевой `htfInterval` для прочих стратегий безопасен: `Config.htfSpan()` вернёт дефолтные 4h, а `htfCandles` при этом пуст.

- [ ] **Step 6: Собрать проект и прогнать тесты**

Run: `go build ./internal/... ./pkg/... ./cmd/... && go test ./internal/... ./cmd/...`
Expected: сборка без ошибок, все тесты PASS.

- [ ] **Step 7: Обновить документацию**

В `docs/scalping_rsimacd/strategy.md` в разделе «Правила» после пункта 1 вставить новый пункт и перенумеровать остальные:

```markdown
2. На баре кросса RSI(`RSIPeriod`) выше порога `RSIEntryMin` (0 — фильтр выключен;
   по умолчанию 50).
3. Тренд старшего таймфрейма: последнее закрытие завершённой **часовой** свечи выше
   `EMA(H1, HTFTrendEMA)`. `HTFTrendEMA = 0` выключает фильтр (значение по умолчанию).
   Фильтр **fail-closed**: если включён, а часовой истории меньше `HTFTrendEMA` баров,
   вход запрещён.
```

В конце раздела «Правила» добавить абзац:

```markdown
Часовые свечи для фильтра тренда грузит `cmd/backtest` (интервал `Hour1`, годичный
lead-in для прогрева EMA); движок видит только полностью закрытые H1-бары
(`Config.HTFInterval = 1h`), заглядывания вперёд нет.
```

В разделе про калибровку/запуск добавить упоминание фазы:

```markdown
Грид `data/params/scalping_rsimacd/grid.json` состоит из трёх фаз: `entry` (геометрия
входа), `context` (`RSIEntryMin` × `HTFTrendEMA` — оба гейта включаются и выключаются
значением 0) и `risk` (RR, буфер стопа, аблация стохастик-выхода).
```

- [ ] **Step 8: Финальный гейт качества**

Run: `./bin/mage ci`
Expected: линт чистый, `go test -race ./...` PASS, mock-drift check чистый.

- [ ] **Step 9: Коммит**

```bash
git add cmd/backtest/main.go data/params/scalping_rsimacd/grid.json internal/service/backtest/scalping_rsimacd_grid_test.go docs/scalping_rsimacd/strategy.md
git commit -m "feat(scalping-rsimacd): load H1 candles for the trend gate, sweep both gates in the grid"
```

---

## После плана

Калибровка не входит в план — это отдельный прогон после мержа кода:

```
go run ./cmd/backtest -ticker SBER -strategy scalping_rsimacd -interval Minutes5 \
  -calibrate data/params/scalping_rsimacd/grid.json -out ./reports/SBER \
  -months 24 -test-months 6 -min-trades 20 -metric profit_factor
```

Критерий приёмки прежний (см. `docs/scalping_rsimacd/strategy.md`): решение по pooled OOS
profit factor, не по in-sample.
