# RSI zone: подтверждение входа стохастиком — план реализации

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Добавить в `rsi_zone` необязательный гейт входа: перепроданность должна подтвердить и RSI, и Stoch %D в окне из N баров, а запускать вход может крест любого из двух.

**Architecture:** Вся логика — в чистом ядре `rsi_zone/strategy/core/core.go`: новые поля `Params`, разрешение нулей в константы, чистые помощники `stochDSeries` и `zoneTrigger`, ветка шага 2 в `enter`. При `UseStoch = 0` путь входа прежний. Это доказывает характеризационный golden-тест через настоящий движок бэктеста, записанный ДО правок ядра. Калибровка получает тему `stoch`.

**Tech Stack:** Go 1.25, `pkg/indicators` (`RSISeries`, `StochasticSeries`, `ATR`), движок `internal/domain/backtest` (`bt.Run`), калибратор `cmd/backtest -calibrate`.

**Spec:** `docs/superpowers/specs/2026-09-30-rsi-zone-stoch-confirm-design.md`

## Global Constraints

- Ветка `feat/rsi-zone-stoch-confirm` (уже создана, на ней коммит спеки).
- `core.DefaultParams()` НЕ меняется: новые поля в нём нули.
- При `UseStoch = 0` решения, `Lookback()` и текст `EntryReason` совпадают с нынешними бар в бар.
- Нулевые поля стохастика при `UseStoch = 1` разрешаются в `StochKPeriod` 14, `StochDSmooth` 3, `StochLower` 20; `ZoneWindowBars < 1` — в 1. Отрицательные `StochKPeriod`/`StochDSmooth` — отказ от входа.
- Не трогать: выход (`manage`), `StopLevel`, `rsi_zone/live`, реестр бэктеста, `cmd/pullparity`, литералы тикеров, `zone_screen.go`.
- `docs/rsi_zone/*.md` — только механика, без результатов прогонов (правило из CLAUDE.md). Результаты — в `_comment` сеток.
- Коммиты на русском, в стиле истории проекта, в конце строка `Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>`.
- Проверка: `./bin/mage ci` зелёный. `go build ./...` не использовать (падает на `magefiles`), только `go build ./internal/... ./pkg/... ./cmd/...`.
- Прогоны бэктеста — строго последовательно, никогда параллельно (общий файл кэша).

## Review Focus

1. **Литерал тикера без полей стохастика, включённый одним `UseStoch = 1`** (так делает сетка `cal_stoch.json` поверх литерала): ждём поведение как у явных 14/3/20/1, а не ноль входов. Тест — Task 4, `TestUseStochAloneResolvesZeroFields`.
2. **Плоский диапазон цены (high = low на всех K барах)**: `StochasticSeries` даёт %K = 0, и такой бар считается «в зоне». Поведение наследовано от reversion и зафиксировано тестом, чтобы оно не поменялось молча. Тест — Task 3, случай `flat range reads as zero`.
3. **Окно у начала ряда (`i − N + 1 < 0`)**: без паники, считаются только существующие бары. Тест — Task 3, случай `window clipped at series start`.
4. **Оба креста на одном баре**: один вход, в журнале триггером указан RSI. Тест — Task 3, случай `both cross on the same bar`.
5. **Окно live совпадает с окном бэктеста**: live берёт `Lookback()` у ядра; при `UseStoch = 1` окно вмещает прогрев %D плюс N, при `UseStoch = 0` не меняется. Тест — Task 2, `TestLookback`.

---

### Task 1: Характеризационный golden-тест нынешнего входа

Фиксирует сделки нынешнего кода на синтетической истории для `core.DefaultParams()` и литералов всех пяти тикеров. Пишется и коммитится ДО любых правок ядра. Потом он будет доказывать, что `UseStoch = 0` ничего не поменял.

**Files:**
- Create: `internal/service/trading_strategy/rsi_zone/strategy/core/legacy_golden_test.go`
- Create: `internal/service/trading_strategy/rsi_zone/strategy/core/testdata/legacy_trades.golden` (генерируется тестом)

**Interfaces:**
- Consumes: `bt.Run(s strategy.Strategy, candles, dailyCandles, htfCandles []bt.Candle, cfg bt.Config) bt.Result`; `core.NewWithParams(ticker string, p core.Params) *core.Strategy`; `<ticker>.DefaultParams() core.Params` для afks, baza, dias, domrf, lent.
- Produces: golden-файл `testdata/legacy_trades.golden` и флаг `-update` для его перезаписи. Позже на них опираются Task 2 и Task 4.

- [ ] **Step 1: Написать тест**

Файл во внешнем тестовом пакете `core_test`: пакеты тикеров импортируют `core`, а внешний тестовый пакет может импортировать их без цикла.

```go
// legacy_golden_test.go pins the trades the rsi_zone entry produced BEFORE the stochastic gate
// existed, replayed through the real backtest engine on a seeded synthetic history for the core
// baseline and every calibrated ticker literal. With UseStoch left at zero the gate must be
// invisible: any change to entries, exits, prices or journal text fails here. Regenerate only
// deliberately: go test ./internal/service/trading_strategy/rsi_zone/strategy/core/ -run TestLegacyTradesGolden -update
package core_test

import (
	"flag"
	"fmt"
	"math"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	bt "tinvest/internal/domain/backtest"
	"tinvest/internal/service/trading_strategy/rsi_zone/strategy/afks"
	"tinvest/internal/service/trading_strategy/rsi_zone/strategy/baza"
	"tinvest/internal/service/trading_strategy/rsi_zone/strategy/core"
	"tinvest/internal/service/trading_strategy/rsi_zone/strategy/dias"
	"tinvest/internal/service/trading_strategy/rsi_zone/strategy/domrf"
	"tinvest/internal/service/trading_strategy/rsi_zone/strategy/lent"
)

var update = flag.Bool("update", false, "rewrite testdata/*.golden from the current code")

// syntheticHistory builds a seeded random walk of 30-minute weekday bars (16 per day,
// 10:00–17:30 MSK) with a mild upward drift, plus one daily candle per weekday aggregated
// from them. The drift keeps the close above long EMAs often enough for entries; the noise
// makes RSI(4) cross its bands and the stop fire.
func syntheticHistory(seed int64, days int) (bars, daily []bt.Candle) {
	loc, err := time.LoadLocation("Europe/Moscow")
	if err != nil {
		loc = time.UTC
	}
	r := rand.New(rand.NewSource(seed))
	p := 100.0
	first := time.Date(2024, 1, 1, 0, 0, 0, 0, loc)
	for d := 0; len(daily) < days; d++ {
		day := first.AddDate(0, 0, d)
		if wd := day.Weekday(); wd == time.Saturday || wd == time.Sunday {
			continue
		}
		var dc bt.Candle
		for b := 0; b < 16; b++ {
			o := p
			p *= 1 + 0.0002 + 0.004*r.NormFloat64()
			hi := math.Max(o, p) * (1 + 0.001*r.Float64())
			lo := math.Min(o, p) * (1 - 0.001*r.Float64())
			c := bt.Candle{
				Time: day.Add(10*time.Hour + time.Duration(b)*30*time.Minute),
				Open: o, High: hi, Low: lo, Close: p, Volume: 1000,
			}
			bars = append(bars, c)
			if b == 0 {
				dc = bt.Candle{Time: day.Add(10 * time.Hour), Open: o, High: hi, Low: lo}
			}
			dc.High = math.Max(dc.High, hi)
			dc.Low = math.Min(dc.Low, lo)
			dc.Close = p
		}
		daily = append(daily, dc)
	}
	return bars, daily
}

// legacyCases are the parameter sets whose trades are pinned: the core baseline and every
// ticker literal registered today.
func legacyCases() []struct {
	name string
	p    core.Params
} {
	return []struct {
		name string
		p    core.Params
	}{
		{"core", core.DefaultParams()},
		{afks.Ticker, afks.DefaultParams()},
		{baza.Ticker, baza.DefaultParams()},
		{dias.Ticker, dias.DefaultParams()},
		{domrf.Ticker, domrf.DefaultParams()},
		{lent.Ticker, lent.DefaultParams()},
	}
}

// renderTrades formats every trade on one line: times, prices, exit code and both journal texts.
func renderTrades(name string, trades []bt.Trade) string {
	var sb strings.Builder
	fmt.Fprintf(&sb, "== %s: %d trades\n", name, len(trades))
	for _, tr := range trades {
		fmt.Fprintf(&sb, "%s %.6f -> %s %.6f %s | %s | %s\n",
			tr.EntryTime.Format(time.RFC3339), tr.EntryPrice,
			tr.ExitTime.Format(time.RFC3339), tr.ExitPrice, tr.Reason,
			tr.EntryReason, tr.ExitReason)
	}
	return sb.String()
}

func TestLegacyTradesGolden(t *testing.T) {
	bars, daily := syntheticHistory(20260930, 400)
	cfg := bt.Config{InitialCash: 1_000_000, Fraction: 1.0, Commission: 0.0005, Lot: 1}

	var sb strings.Builder
	for _, c := range legacyCases() {
		res := bt.Run(core.NewWithParams(c.name, c.p), bars, daily, nil, cfg)
		// The fixture must actually exercise the entry, or the golden would pin nothing.
		if len(res.Trades) < 10 {
			t.Fatalf("%s: %d trades on the synthetic history, want >= 10 — the fixture no longer exercises the entry", c.name, len(res.Trades))
		}
		sb.WriteString(renderTrades(c.name, res.Trades))
	}
	got := sb.String()

	path := filepath.Join("testdata", "legacy_trades.golden")
	if *update {
		if err := os.MkdirAll("testdata", 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(got), 0o644); err != nil {
			t.Fatal(err)
		}
		return
	}
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read golden (run with -update once on the pre-change code): %v", err)
	}
	if got != string(want) {
		t.Fatalf("rsi_zone trades drifted from %s.\nThe stochastic gate must be invisible with UseStoch=0.\n--- got ---\n%s", path, got)
	}
}
```

- [ ] **Step 2: Запустить без golden — убедиться, что тест падает**

Run: `go test ./internal/service/trading_strategy/rsi_zone/strategy/core/ -run TestLegacyTradesGolden -v`
Expected: FAIL с `read golden (run with -update once on the pre-change code)`. Если вместо этого `want >= 10` — фикстура не даёт сделок. Тогда поднять шум с `0.004` до `0.006` и повторить; сид не менять.

- [ ] **Step 3: Сгенерировать golden на НЕИЗМЕНЁННОМ коде**

Run: `git diff --quiet -- internal/service/trading_strategy/rsi_zone/strategy/core/core.go && go test ./internal/service/trading_strategy/rsi_zone/strategy/core/ -run TestLegacyTradesGolden -update`
Expected: exit 0. Первая часть команды гарантирует, что `core.go` не тронут. Проверить файл: шесть секций `== ...`, в каждой ≥ 10 строк, среди причин выхода встречаются и `SL`, и `RSI`:

Run: `grep -c '^== ' internal/service/trading_strategy/rsi_zone/strategy/core/testdata/legacy_trades.golden; grep -c ' SL |' internal/service/trading_strategy/rsi_zone/strategy/core/testdata/legacy_trades.golden; grep -c ' RSI |' internal/service/trading_strategy/rsi_zone/strategy/core/testdata/legacy_trades.golden`
Expected: `6`, затем два положительных числа.

- [ ] **Step 4: Запустить без `-update` — тест проходит, результат детерминирован**

Run: `go test ./internal/service/trading_strategy/rsi_zone/strategy/core/ -run TestLegacyTradesGolden -count=3`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/service/trading_strategy/rsi_zone/strategy/core/legacy_golden_test.go internal/service/trading_strategy/rsi_zone/strategy/core/testdata/legacy_trades.golden
git commit -m "test(rsi_zone): golden сделок нынешнего входа перед гейтом стохастика

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

---

### Task 2: Поля `Params`, разрешение нулей, `Lookback`

**Files:**
- Modify: `internal/service/trading_strategy/rsi_zone/strategy/core/core.go` (структура `Params` — строки 25–34, `Lookback` — строки 59–65)
- Test: `internal/service/trading_strategy/rsi_zone/strategy/core/core_test.go` (`TestDefaultParams`, `TestLookback`) и новый `TestStochConfigResolvesZeros`

**Interfaces:**
- Consumes: ничего нового.
- Produces:
  - поля `Params`: `UseStoch int`, `StochKPeriod int`, `StochDSmooth int`, `StochLower float64`, `ZoneWindowBars int`;
  - константы `defaultStochKPeriod = 14`, `defaultStochDSmooth = 3`, `defaultStochLower = 20.0`;
  - `func stochConfig(p Params) (k, d int, lower float64, window int)` — разрешает нули;
  - `Lookback()` с учётом стохастика при `UseStoch == 1`.

- [ ] **Step 1: Написать падающие тесты**

В `core_test.go` оставить `TestDefaultParams` как есть: `want` прежний, это и есть требование «`DefaultParams` не меняется». Добавить случаи в таблицу `TestLookback`:

```go
		{"stoch off ignores stoch fields", Params{RSIPeriod: 4, EMAPeriod: 50, StochKPeriod: 90, StochDSmooth: 3, ZoneWindowBars: 8}, 120},
		{"stoch on, EMA dominates, window adds", Params{RSIPeriod: 4, EMAPeriod: 50, UseStoch: 1, StochKPeriod: 14, StochDSmooth: 3, ZoneWindowBars: 5}, 125},
		{"stoch on, stoch span dominates", Params{RSIPeriod: 4, EMAPeriod: 10, UseStoch: 1, StochKPeriod: 60, StochDSmooth: 3, ZoneWindowBars: 8}, 154},
		{"stoch on, zero fields resolve to 14/3 and window 1", Params{RSIPeriod: 4, EMAPeriod: 200, UseStoch: 1}, 421},
```

Сверка ожиданий: 2·50+5+20 = 125; 2·63+8+20 = 154; 2·200+1+20 = 421; при выключенном гейте 2·50+20 = 120 → `minLookback` 120.

Новый тест:

```go
func TestStochConfigResolvesZeros(t *testing.T) {
	cases := []struct {
		name                string
		p                   Params
		wantK, wantD, wantW int
		wantLower           float64
	}{
		{"all zero -> defaults", Params{}, 14, 3, 1, 20},
		{"explicit values kept", Params{StochKPeriod: 9, StochDSmooth: 1, StochLower: 15, ZoneWindowBars: 5}, 9, 1, 5, 15},
		{"negative window -> 1", Params{ZoneWindowBars: -3}, 14, 3, 1, 20},
		{"negative periods pass through for refusal downstream", Params{StochKPeriod: -1, StochDSmooth: -2}, -1, -2, 1, 20},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			k, d, lower, w := stochConfig(c.p)
			if k != c.wantK || d != c.wantD || lower != c.wantLower || w != c.wantW {
				t.Fatalf("stochConfig = (%d, %d, %v, %d), want (%d, %d, %v, %d)", k, d, lower, w, c.wantK, c.wantD, c.wantLower, c.wantW)
			}
		})
	}
}
```

- [ ] **Step 2: Запустить — убедиться, что падает**

Run: `go test ./internal/service/trading_strategy/rsi_zone/strategy/core/ -run 'TestLookback|TestStochConfigResolvesZeros|TestDefaultParams'`
Expected: FAIL — ошибка компиляции `unknown field UseStoch` / `undefined: stochConfig`.

- [ ] **Step 3: Реализация**

В `core.go` заменить структуру `Params` целиком:

```go
// Params holds every tunable. All fields are int or float64 so reflection grid calibration
// can sweep them.
type Params struct {
	RSIPeriod      int     // RSI length (grid; default 4)
	RSILower       float64 // lower critical band; a DOWNWARD cross of it is the entry (grid; default 25)
	RSIUpper       float64 // upper critical band; an UPWARD cross of it is the exit (grid; default 75)
	EMAPeriod      int     // trend EMA period; the entry needs close > EMA (grid; default 200)
	DailyATRPeriod int     // daily ATR length, over WEEKDAY completed dailies (fixed; default 14)
	StopDailyATR   float64 // stop = entry - StopDailyATR*dailyATR; 0 disables it (grid; never 0 in the grid)

	// Stochastic confirmation gate (theme stoch). With UseStoch=1 the entry needs RSI and
	// Stoch %D to BOTH read oversold within the last ZoneWindowBars bars, and one of them to
	// cross down into its zone on the current bar. A zero knob resolves to its default (see
	// stochConfig), so a ticker literal written before the gate switches it on with UseStoch
	// alone; DefaultParams leaves them all zero and the gate off.
	UseStoch       int     // 1 = require the stochastic confirmation; 0 = off (grid: stoch)
	StochKPeriod   int     // %K lookback; 0 -> 14; negative refuses every entry (grid: stoch)
	StochDSmooth   int     // %D smoothing, 1 = raw %K; 0 -> 3; negative refuses every entry (grid: stoch)
	StochLower     float64 // Stoch %D lower critical band; 0 -> 20 (grid: stoch)
	ZoneWindowBars int     // bars, current one included, in which the other oscillator may have been in its zone; < 1 -> 1 (grid: stoch)
}

// Defaults the stochastic gate falls back to when its knobs are left at zero.
const (
	defaultStochKPeriod = 14
	defaultStochDSmooth = 3
	defaultStochLower   = 20.0
)

// stochConfig resolves the stochastic gate's knobs: a zero period or band falls back to its
// default and a window below one bar becomes one bar. Negative periods pass through untouched
// so stochDSeries refuses them — a misconfiguration must block entries, not be papered over.
func stochConfig(p Params) (k, d int, lower float64, window int) {
	k, d, lower, window = p.StochKPeriod, p.StochDSmooth, p.StochLower, p.ZoneWindowBars
	if k == 0 {
		k = defaultStochKPeriod
	}
	if d == 0 {
		d = defaultStochDSmooth
	}
	if lower == 0 {
		lower = defaultStochLower
	}
	if window < 1 {
		window = 1
	}
	return k, d, lower, window
}
```

`DefaultParams` не трогать.

`Lookback` заменить целиком:

```go
// Lookback sizes the candle window the engine feeds Decide on every bar. ema.Compute seeds on
// an SMA over the first `period` closes, so a window shorter than the period yields an all-zero
// series that silently fails the trend gate for the whole run. Doubling the largest period
// leaves as many recursion steps as the seed span; the +20 covers the two-bar cross lookups.
// With the stochastic gate on, the %K+%D warm-up joins the largest period and the confirmation
// window is added on top; with it off the window is exactly what it was before the gate existed.
func (s *Strategy) Lookback() int {
	span := max(s.p.EMAPeriod, s.p.RSIPeriod)
	if s.p.UseStoch != 1 {
		return max(minLookback, 2*span+20)
	}
	k, d, _, window := stochConfig(s.p)
	return max(minLookback, 2*max(span, k+d)+window+20)
}
```

- [ ] **Step 4: Запустить тесты пакета и golden**

Run: `go test ./internal/service/trading_strategy/rsi_zone/...`
Expected: PASS, включая `TestLegacyTradesGolden` и тесты пакетов тикеров (LENT и BAZA по-прежнему равны `core.DefaultParams()`).

- [ ] **Step 5: Проверить, что live берёт окно у ядра**

Run: `grep -rn 'Lookback()' internal/service/trading_strategy/rsi_zone/live/ internal/service/trading_strategy/rsi_pullback/live/*.go | grep -v _test`
Expected: окно для MarketData берётся из `Lookback()` стратегии (в `rsi_pullback/live/pass.go` — `dec.Lookback()`), а не из константы. Если для rsi_zone окно берётся из константы — **СТОП**, доложи: при `UseStoch = 1` live увидит обрезанное окно.

- [ ] **Step 6: Commit**

```bash
git add internal/service/trading_strategy/rsi_zone/strategy/core/core.go internal/service/trading_strategy/rsi_zone/strategy/core/core_test.go
git commit -m "feat(rsi_zone): поля гейта стохастика, разрешение нулей и окно Lookback

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

---

### Task 3: Чистые помощники `stochDSeries` и `zoneTrigger`

**Files:**
- Modify: `internal/service/trading_strategy/rsi_zone/strategy/core/core.go` (новые функции после `crossedUp`, около строки 157)
- Create: `internal/service/trading_strategy/rsi_zone/strategy/core/zone_test.go`

**Interfaces:**
- Consumes: `crossedDown(series []float64, i, period int, level float64) bool` (существующая; `period` — первый индекс, чьё значение считается настоящим для бара `i-1`); `indicators.StochasticSeries(highs, lows, closes []float64, kPeriod, dSmooth int) (ks, ds []float64)`.
- Produces:
  - `func stochDSeries(highs, lows, closes []float64, k, d int) (series []float64, warm int, ok bool)` — %D, разложенный в ряд длины `len(closes)`; `warm = k + d - 2` — первый прогретый бар; `ok = false` при `k <= 0`, `d <= 0`, пустом %D или рассогласовании длин;
  - `type oscillator struct { series []float64; warm int; level float64 }`;
  - `type zoneHit struct { by string; otherAt int }` — `by` равно `"RSI"` или `"Stoch"`, `otherAt` — бар, где второй осциллятор был в зоне;
  - `func seenInZone(o oscillator, i, window int) int` — самый свежий бар окна `[i-window+1, i]`, где `o` в зоне, или −1;
  - `func zoneTrigger(rsi, stoch oscillator, i, window int) (zoneHit, bool)`.

- [ ] **Step 1: Написать падающие тесты**

`zone_test.go`:

```go
package core

import (
	"reflect"
	"testing"

	"tinvest/pkg/indicators"
)

func TestStochDSeriesAlignsToBars(t *testing.T) {
	n := 30
	highs, lows, closes := make([]float64, n), make([]float64, n), make([]float64, n)
	for i := 0; i < n; i++ {
		c := 100 + float64(i%7) - float64(i%3)
		closes[i], highs[i], lows[i] = c, c+1, c-1
	}
	const k, d = 5, 3
	series, warm, ok := stochDSeries(highs, lows, closes, k, d)
	if !ok {
		t.Fatal("ok = false on a valid series")
	}
	if len(series) != n || warm != k+d-2 {
		t.Fatalf("len = %d, warm = %d; want %d, %d", len(series), warm, n, k+d-2)
	}
	_, ds := indicators.StochasticSeries(highs, lows, closes, k, d)
	if !reflect.DeepEqual(series[warm:], ds) {
		t.Fatalf("series[warm:] = %v, want the raw %%D %v", series[warm:], ds)
	}
	for b := 0; b < warm; b++ {
		if series[b] != 0 {
			t.Fatalf("warm-up bar %d = %v, want the unset 0", b, series[b])
		}
	}
	// d = 1 is raw %K: warm = k-1.
	if _, w, ok := stochDSeries(highs, lows, closes, k, 1); !ok || w != k-1 {
		t.Fatalf("d=1: warm = %d ok = %v, want %d true", w, ok, k-1)
	}
}

func TestStochDSeriesRefuses(t *testing.T) {
	h, l, c := []float64{2, 2, 2}, []float64{1, 1, 1}, []float64{1.5, 1.5, 1.5}
	for _, kd := range [][2]int{{0, 3}, {3, 0}, {-1, 3}, {3, -1}, {5, 1}} {
		if _, _, ok := stochDSeries(h, l, c, kd[0], kd[1]); ok {
			t.Errorf("k=%d d=%d: ok = true, want refusal", kd[0], kd[1])
		}
	}
}

// Review Focus 2: a collapsed high/low range makes StochasticSeries report %K = 0, which then
// reads as in-zone. Inherited from reversion; pinned so it never changes silently.
func TestStochDSeriesFlatRangeReadsAsZero(t *testing.T) {
	n := 10
	flat := make([]float64, n)
	for i := range flat {
		flat[i] = 100
	}
	series, warm, ok := stochDSeries(flat, flat, flat, 3, 1)
	if !ok {
		t.Fatal("ok = false")
	}
	if !(series[n-1] == 0 && seenInZone(oscillator{series, warm, 20}, n-1, 1) == n-1) {
		t.Fatalf("flat range: %%D = %v, in zone = %v; want 0 and in zone", series[n-1], seenInZone(oscillator{series, warm, 20}, n-1, 1))
	}
}

func TestZoneTrigger(t *testing.T) {
	// Series are hand-built; warm 0 means every index is a genuine reading. RSI band 25, Stoch 20.
	osc := func(s []float64, warm int, level float64) oscillator { return oscillator{s, warm, level} }
	cases := []struct {
		name      string
		rsi       oscillator
		stoch     oscillator
		i, window int
		wantOK    bool
		wantBy    string
		wantAt    int
	}{
		{"RSI cross, stoch in zone at window's oldest bar",
			osc([]float64{50, 50, 50, 50, 30, 20}, 0, 25), osc([]float64{50, 50, 50, 10, 30, 30}, 0, 20), 5, 3, true, "RSI", 3},
		{"RSI cross, stoch in zone one bar before the window",
			osc([]float64{50, 50, 50, 50, 30, 20}, 0, 25), osc([]float64{50, 50, 10, 30, 30, 30}, 0, 20), 5, 3, false, "", 0},
		{"stoch cross, RSI in zone without its own cross",
			osc([]float64{50, 50, 50, 20, 18, 15}, 0, 25), osc([]float64{50, 50, 50, 50, 30, 10}, 0, 20), 5, 2, true, "Stoch", 5},
		{"both in zone, nobody crosses now",
			osc([]float64{50, 50, 50, 20, 18, 15}, 0, 25), osc([]float64{50, 50, 50, 15, 12, 10}, 0, 20), 5, 5, false, "", 0},
		{"window 1: both in zone on the cross bar",
			osc([]float64{50, 50, 50, 50, 30, 20}, 0, 25), osc([]float64{50, 50, 50, 50, 30, 10}, 0, 20), 5, 1, true, "RSI", 5},
		{"window 1: stoch in zone only on the previous bar",
			osc([]float64{50, 50, 50, 50, 30, 20}, 0, 25), osc([]float64{50, 50, 50, 50, 10, 30}, 0, 20), 5, 1, false, "", 0},
		// Review Focus 4: one entry, reported as the RSI trigger.
		{"both cross on the same bar",
			osc([]float64{50, 50, 50, 50, 30, 20}, 0, 25), osc([]float64{50, 50, 50, 50, 30, 10}, 0, 20), 5, 3, true, "RSI", 5},
		{"stoch warm-up zero does not confirm",
			osc([]float64{50, 50, 50, 50, 30, 20}, 0, 25), osc([]float64{0, 0, 0, 0, 30, 30}, 4, 20), 5, 5, false, "", 0},
		{"genuine post-warm-up stoch zero confirms",
			osc([]float64{50, 50, 50, 50, 30, 20}, 0, 25), osc([]float64{0, 0, 0, 0, 0, 30}, 4, 20), 5, 5, true, "RSI", 4},
		{"RSI warm-up zero does not confirm a stoch cross",
			osc([]float64{0, 0, 0, 0, 30, 30}, 4, 25), osc([]float64{50, 50, 50, 50, 30, 10}, 0, 20), 5, 5, false, "", 0},
		// Review Focus 3: the window reaches before bar 0; only existing bars count, no panic.
		{"window clipped at series start",
			osc([]float64{30, 20}, 0, 25), osc([]float64{10, 30}, 0, 20), 1, 5, true, "RSI", 0},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			hit, ok := zoneTrigger(c.rsi, c.stoch, c.i, c.window)
			if ok != c.wantOK {
				t.Fatalf("ok = %v, want %v (hit %+v)", ok, c.wantOK, hit)
			}
			if ok && (hit.by != c.wantBy || hit.otherAt != c.wantAt) {
				t.Fatalf("hit = %+v, want by %q at %d", hit, c.wantBy, c.wantAt)
			}
		})
	}
}
```

Как читать случай «stoch cross, RSI in zone without its own cross»: RSI ниже 25 на барах 3–5, креста на баре 5 нет (20 → 18 → 15). Stoch пересекает 20 на баре 5 (30 → 10). Окно 2 = бары 4–5, самый свежий бар RSI в зоне — 5.

- [ ] **Step 2: Запустить — убедиться, что падает**

Run: `go test ./internal/service/trading_strategy/rsi_zone/strategy/core/ -run 'TestStochDSeries|TestZoneTrigger'`
Expected: FAIL — `undefined: stochDSeries`, `undefined: oscillator`, `undefined: zoneTrigger`.

- [ ] **Step 3: Реализация**

В `core.go` после `crossedUp`:

```go
// stochDSeries returns Stochastic %D laid over the full bar index: series[b] is the %D reading
// of bar b, and warm = k+d-2 is the first bar that has one (earlier slots are an unset zero,
// the same convention RSISeries uses). indicators.StochasticSeries returns a shorter,
// right-aligned slice; its j-th value belongs to bar j+k+d-2. ok is false when the periods are
// not positive or history is too short to produce a single %D value.
func stochDSeries(highs, lows, closes []float64, k, d int) (series []float64, warm int, ok bool) {
	if k <= 0 || d <= 0 {
		return nil, 0, false
	}
	_, ds := indicators.StochasticSeries(highs, lows, closes, k, d)
	n := len(closes)
	warm = k + d - 2
	if len(ds) == 0 || warm+len(ds) != n {
		return nil, 0, false
	}
	series = make([]float64, n)
	copy(series[warm:], ds)
	return series, warm, true
}

// oscillator is one confirmation series with its first genuine index and its lower band.
type oscillator struct {
	series []float64
	warm   int
	level  float64
}

// zoneHit records which oscillator crossed on the current bar and where the other one was seen
// in its zone.
type zoneHit struct {
	by      string // "RSI" or "Stoch": the oscillator that crossed down on the current bar
	otherAt int    // the most recent bar of the window where the other oscillator was in its zone
}

// seenInZone returns the most recent bar of [i-window+1, i] (clipped at 0) where o reads
// strictly below its band on a warmed index, or -1. Validity is gated on the index, as in
// crossedDown: a genuine 0.00 after warm-up counts, an unset warm-up zero does not.
func seenInZone(o oscillator, i, window int) int {
	for b := i; b >= 0 && b > i-window; b-- {
		if b >= o.warm && b < len(o.series) && o.series[b] < o.level {
			return b
		}
	}
	return -1
}

// zoneTrigger is the symmetric confirmation: an entry fires when RSI crosses down through its
// band on bar i and Stoch was in its zone somewhere in the window, or the other way round. When
// both cross on the same bar the RSI trigger is reported. crossedDown's period argument is the
// oscillator's warm index, so the previous bar must itself be a genuine reading.
func zoneTrigger(rsi, stoch oscillator, i, window int) (zoneHit, bool) {
	if crossedDown(rsi.series, i, rsi.warm, rsi.level) {
		if b := seenInZone(stoch, i, window); b >= 0 {
			return zoneHit{by: "RSI", otherAt: b}, true
		}
	}
	if crossedDown(stoch.series, i, stoch.warm, stoch.level) {
		if b := seenInZone(rsi, i, window); b >= 0 {
			return zoneHit{by: "Stoch", otherAt: b}, true
		}
	}
	return zoneHit{}, false
}
```

Замечание о прогреве RSI: `RSISeries` заполняет индексы `< period` нулём, первый настоящий индекс — `period`. Поэтому `warm` для RSI равен `RSIPeriod`, как в существующем вызове `crossedDown(rsi, i, s.p.RSIPeriod, …)`.

- [ ] **Step 4: Запустить тесты**

Run: `go test ./internal/service/trading_strategy/rsi_zone/strategy/core/`
Expected: PASS, включая golden.

- [ ] **Step 5: Commit**

```bash
git add internal/service/trading_strategy/rsi_zone/strategy/core/core.go internal/service/trading_strategy/rsi_zone/strategy/core/zone_test.go
git commit -m "feat(rsi_zone): ряд Stoch %D по барам и симметричный триггер в окне N

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

---

### Task 4: Гейт во входе, журнал, тесты `Decide` и движка

**Files:**
- Modify: `internal/service/trading_strategy/rsi_zone/strategy/core/core.go` (`enter`, строки ~191–228; новая `stochReason` после `entryReason`)
- Create: `internal/service/trading_strategy/rsi_zone/strategy/core/stoch_enter_test.go`

**Interfaces:**
- Consumes: из Task 2 — `stochConfig(p Params) (k, d int, lower float64, window int)`; из Task 3 — `stochDSeries`, `oscillator`, `zoneHit`, `zoneTrigger`; из существующих тестов (`core_test.go`, `engine_test.go`, пакет `core`) — `fixture`, `fixtureWithDaily`, `trendCloses`, `downtrendCloses`, `mondayNoon`, `saturdayNoon`, `dailyWidth`, `engineUptrendCloses`, `engineCandles`, `engineDailyCandles`.
- Produces: `enter` с веткой `UseStoch == 1`; `func (s *Strategy) stochReason(hit zoneHit, i int, rsiNow, stochNow float64) string`. Хвост журнала начинается с `"; подтверждение: крест дал "`.

- [ ] **Step 1: Написать падающие тесты**

`stoch_enter_test.go`:

```go
package core

import (
	"math"
	"strings"
	"testing"
	"time"

	bt "tinvest/internal/domain/backtest"
	"tinvest/internal/service/trading_strategy/scalping/model"
	"tinvest/internal/service/trading_strategy/scalping/strategy"
)

// stochOf computes the %D series the strategy will see for md with the given knobs.
func stochOf(t *testing.T, md strategy.MarketData, k, d int) []float64 {
	t.Helper()
	s, _, ok := stochDSeries(md.Highs, md.Lows, md.Closes, k, d)
	if !ok {
		t.Fatal("stochDSeries refused the fixture")
	}
	return s
}

func stochParams(lower float64, window int) Params {
	p := DefaultParams()
	p.UseStoch, p.StochKPeriod, p.StochDSmooth, p.StochLower, p.ZoneWindowBars = 1, 14, 3, lower, window
	return p
}

// trendCloses(3): RSI(4) crosses 25 on the last bar. The stochastic decides.
func TestStochGateOnRSITrigger(t *testing.T) {
	md := fixture(trendCloses(3), mondayNoon)
	st := stochOf(t, md, 14, 3)
	i := len(st) - 1

	t.Run("stoch in zone on the cross bar confirms", func(t *testing.T) {
		sig := NewWithParams("T", stochParams(st[i]+1, 1)).Decide(md)
		if sig.Kind != model.SignalBuy {
			t.Fatalf("Kind = %v, want Buy (stoch %.2f under band %.2f)", sig.Kind, st[i], st[i]+1)
		}
		if !strings.Contains(sig.EntryReason, "; подтверждение: крест дал RSI") {
			t.Fatalf("EntryReason = %q, want the RSI-trigger tail", sig.EntryReason)
		}
	})
	t.Run("stoch never in zone in the window blocks", func(t *testing.T) {
		lowest := math.Min(st[i], math.Min(st[i-1], st[i-2]))
		if sig := NewWithParams("T", stochParams(lowest, 3)).Decide(md); sig.Kind == model.SignalBuy {
			t.Fatalf("Buy with band %.2f at or under every stoch reading of the window", lowest)
		}
	})
}

// trendCloses(4): RSI(4) is already under 25 on the last two bars, so there is no RSI cross.
// Only a stoch cross on the last bar can fire.
func TestStochGateOnStochTrigger(t *testing.T) {
	md := fixture(trendCloses(4), mondayNoon)
	st := stochOf(t, md, 14, 3)
	i := len(st) - 1
	if !(st[i] < st[i-1]) {
		t.Fatalf("fixture drift: stoch %.2f -> %.2f is not falling on the last bar", st[i-1], st[i])
	}
	mid := (st[i-1] + st[i]) / 2 // the band the stoch crosses exactly on bar i

	t.Run("stoch cross with RSI in zone enters", func(t *testing.T) {
		sig := NewWithParams("T", stochParams(mid, 1)).Decide(md)
		if sig.Kind != model.SignalBuy {
			t.Fatalf("Kind = %v, want Buy on the stoch cross", sig.Kind)
		}
		if !strings.Contains(sig.EntryReason, "; подтверждение: крест дал Stoch") {
			t.Fatalf("EntryReason = %q, want the Stoch-trigger tail", sig.EntryReason)
		}
	})
	t.Run("RSI out of its zone blocks the stoch cross", func(t *testing.T) {
		p := stochParams(mid, 1)
		p.RSILower = 10 // RSI(4) is 15.6 on the last bar: no longer in the zone
		if sig := NewWithParams("T", p).Decide(md); sig.Kind == model.SignalBuy {
			t.Fatal("Buy although RSI never read under its band in the window")
		}
	})
	t.Run("gate off keeps the old behaviour: no RSI cross, no entry", func(t *testing.T) {
		if sig := NewWithParams("T", DefaultParams()).Decide(md); sig.Kind == model.SignalBuy {
			t.Fatal("Buy with UseStoch=0 on a bar without an RSI cross")
		}
	})
	t.Run("weekend blocks the stoch trigger", func(t *testing.T) {
		mdW := fixture(trendCloses(4), saturdayNoon)
		if sig := NewWithParams("T", stochParams(mid, 1)).Decide(mdW); sig.Kind == model.SignalBuy {
			t.Fatal("Buy on a Saturday")
		}
	})
	t.Run("unknown daily ATR blocks the stoch trigger", func(t *testing.T) {
		mdA := fixture(trendCloses(4), mondayNoon)
		mdA.DailyHighs, mdA.DailyLows, mdA.DailyCloses, mdA.DailyTimes = nil, nil, nil, nil
		if sig := NewWithParams("T", stochParams(mid, 1)).Decide(mdA); sig.Kind == model.SignalBuy {
			t.Fatal("Buy without a daily ATR")
		}
	})
}

// downtrendCloses: RSI crosses on the last bar but close < EMA(200). A band of 101 puts every
// warmed stoch reading in the zone, so only the trend gate can reject.
func TestStochGateKeepsTheTrendGate(t *testing.T) {
	md := fixture(downtrendCloses(), mondayNoon)
	if sig := NewWithParams("T", stochParams(101, 1)).Decide(md); sig.Kind == model.SignalBuy {
		t.Fatal("Buy below the trend EMA")
	}
}

// Review Focus 1: UseStoch alone on a literal without stochastic knobs behaves like explicit
// 14/3/20/1 — never like "no entries at all".
func TestUseStochAloneResolvesZeroFields(t *testing.T) {
	for _, closes := range [][]float64{trendCloses(3), trendCloses(4)} {
		md := fixture(closes, mondayNoon)
		bare := DefaultParams()
		bare.UseStoch = 1
		explicit := stochParams(20, 1)
		a := NewWithParams("T", bare).Decide(md)
		b := NewWithParams("T", explicit).Decide(md)
		if a.Kind != b.Kind || a.EntryReason != b.EntryReason {
			t.Fatalf("UseStoch alone = (%v, %q), explicit 14/3/20/1 = (%v, %q)", a.Kind, a.EntryReason, b.Kind, b.EntryReason)
		}
	}
}

func TestNegativeStochPeriodRefusesEntry(t *testing.T) {
	md := fixture(trendCloses(3), mondayNoon)
	for _, p := range []Params{
		func() Params { p := stochParams(101, 1); p.StochKPeriod = -1; return p }(),
		func() Params { p := stochParams(101, 1); p.StochDSmooth = -1; return p }(),
	} {
		if sig := NewWithParams("T", p).Decide(md); sig.Kind == model.SignalBuy {
			t.Fatalf("Buy with a negative stochastic period: %+v", p)
		}
	}
}

// The whole engine path: with the gate on, a stoch cross (no RSI cross) opens the trade.
func TestEngineEntersOnStochTrigger(t *testing.T) {
	probe := NewWithParams("TEST", stochParams(20, 1))
	lookback := probe.Lookback()
	closes := engineUptrendCloses(lookback, 4)
	entryBarTime := mondayNoon.Add(-30 * time.Minute) // a Monday; the exit bar lands on mondayNoon
	candles := engineCandles(closes, entryBarTime)

	highs, lows := make([]float64, len(candles)), make([]float64, len(candles))
	for i, c := range candles {
		highs[i], lows[i] = c.High, c.Low
	}
	st, _, ok := stochDSeries(highs, lows, closes, 14, 3)
	if !ok {
		t.Fatal("stochDSeries refused the engine fixture")
	}
	i := len(st) - 1
	if !(st[i] < st[i-1]) {
		t.Fatalf("fixture drift: stoch %.2f -> %.2f is not falling on the last bar", st[i-1], st[i])
	}
	s := NewWithParams("TEST", stochParams((st[i-1]+st[i])/2, 1))

	gapOpen := closes[len(closes)-1] - 3*dailyWidth
	candles = append(candles, bt.Candle{Time: mondayNoon, Open: gapOpen, High: gapOpen, Low: gapOpen - 1, Close: gapOpen - 0.5})
	daily := engineDailyCandles(mondayNoon, 40, dailyWidth, dailyWidth/10)
	res := bt.Run(s, candles, daily, nil, bt.Config{InitialCash: 1_000_000, Fraction: 1.0, Lot: 1})

	if len(res.Trades) != 1 {
		t.Fatalf("trades = %d, want 1", len(res.Trades))
	}
	tr := res.Trades[0]
	if math.Abs(tr.EntryPrice-closes[len(closes)-1]) > 1e-6 {
		t.Fatalf("EntryPrice = %v, want the last down bar's close %v", tr.EntryPrice, closes[len(closes)-1])
	}
	if !strings.Contains(tr.EntryReason, "крест дал Stoch") {
		t.Fatalf("EntryReason = %q, want the Stoch trigger", tr.EntryReason)
	}
}
```

- [ ] **Step 2: Запустить — убедиться, что падает**

Run: `go test ./internal/service/trading_strategy/rsi_zone/strategy/core/ -run 'TestStochGate|TestUseStochAlone|TestNegativeStoch|TestEngineEntersOnStoch'`
Expected: FAIL. Тесты подтверждения ждут Buy там, где старый вход без RSI-креста молчит, а хвоста `подтверждение` в журнале ещё нет.

- [ ] **Step 3: Реализация**

В `enter` заменить блок шага 2 (от `i := n - 1` до проверки `crossedDown` включительно):

```go
	i := n - 1
	// 2. the trigger. Gate off: RSI crosses down through the lower band on the current bar.
	// Gate on: RSI or Stoch %D crosses down into its zone now while the other one was in its
	// zone somewhere in the last ZoneWindowBars bars.
	rsi := indicators.RSISeries(md.Closes, s.p.RSIPeriod)
	if len(rsi) != n {
		return sig
	}
	var (
		hit   zoneHit
		stoch []float64
	)
	if s.p.UseStoch == 1 {
		k, d, lower, window := stochConfig(s.p)
		st, warm, ok := stochDSeries(md.Highs, md.Lows, md.Closes, k, d)
		if !ok {
			return sig
		}
		h, fired := zoneTrigger(
			oscillator{series: rsi, warm: s.p.RSIPeriod, level: s.p.RSILower},
			oscillator{series: st, warm: warm, level: lower},
			i, window,
		)
		if !fired {
			return sig
		}
		hit, stoch = h, st
	} else if !crossedDown(rsi, i, s.p.RSIPeriod, s.p.RSILower) {
		return sig
	}
```

Шаги 3–5 не менять. В конце `enter` после строки с `sig.EntryReason = s.entryReason(...)` добавить:

```go
	if s.p.UseStoch == 1 {
		sig.EntryReason += s.stochReason(hit, i, rsi[i], stoch[i])
	}
```

После `entryReason`:

```go
// stochReason is the journal tail of a stochastic-confirmed entry: which oscillator crossed,
// how many bars back the other one was in its zone, and both readings on the entry bar.
func (s *Strategy) stochReason(hit zoneHit, i int, rsiNow, stochNow float64) string {
	k, d, lower, window := stochConfig(s.p)
	other := "Stoch"
	if hit.by == "Stoch" {
		other = "RSI"
	}
	return fmt.Sprintf("; подтверждение: крест дал %s, %s в зоне %d бар(ов) назад (окно %d); RSI %.1f (зона <%.0f), Stoch%%D(%d,%d) %.1f (зона <%.0f)",
		hit.by, other, i-hit.otherAt, window, rsiNow, s.p.RSILower, k, d, stochNow, lower)
}
```

- [ ] **Step 4: Запустить весь пакет, включая golden**

Run: `go test ./internal/service/trading_strategy/rsi_zone/...`
Expected: PASS. Если `TestLegacyTradesGolden` падает, ветка `UseStoch = 0` изменила поведение. Исправляй код, НЕ перезаписывай golden.

- [ ] **Step 5: Commit**

```bash
git add internal/service/trading_strategy/rsi_zone/strategy/core/core.go internal/service/trading_strategy/rsi_zone/strategy/core/stoch_enter_test.go
git commit -m "feat(rsi_zone): вход с подтверждением стохастиком в окне N баров

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

---

### Task 5: Тема калибровки `stoch`, сторожевой тест, документация

**Files:**
- Create: `data/params/rsi_zone/lent/cal_stoch.json`
- Create: `data/params/rsi_zone/domrf/cal_stoch.json`
- Create: `internal/service/backtest/rsi_zone_stoch_grid_test.go`
- Modify: `.claude/commands/rsi-zone-calibrate.md` (§2: абзац «Шесть полей…», таблица тем, инварианты)
- Modify: `docs/rsi_zone/strategy.md` (§2 «Параметры», §3 «Вход»)

**Interfaces:**
- Consumes: `ParsePhases(raw []byte) ([]Phase, error)`, `applyField(p any, name string, v float64) (any, error)` из пакета `internal/service/backtest` (используются в `rsi_zone_grid_test.go`); поля `core.Params` из Task 2.
- Produces: сетки, которые запускает Task 6.

- [ ] **Step 1: Написать падающий сторожевой тест**

`internal/service/backtest/rsi_zone_stoch_grid_test.go`:

```go
package backtest

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"tinvest/internal/service/trading_strategy/rsi_zone/strategy/core"
)

// TestRSIZoneStochGrids guards every data/params/rsi_zone/*/cal_stoch.json: each value applies
// to core.Params, the gate is switched on, all four stochastic axes are listed (so no axis is
// silently measured at a zero-resolved default), and the axes carry the minimal values of the
// theme.
func TestRSIZoneStochGrids(t *testing.T) {
	files, err := filepath.Glob("../../../data/params/rsi_zone/*/cal_stoch.json")
	if err != nil {
		t.Fatal(err)
	}
	if len(files) == 0 {
		t.Fatal("no cal_stoch.json found — the stoch theme has no grid")
	}
	minimal := map[string][]float64{
		"StochKPeriod":   {5, 9, 14},
		"StochDSmooth":   {1, 3},
		"StochLower":     {10, 15, 20, 25, 30},
		"ZoneWindowBars": {1, 2, 3, 5, 8},
	}
	for _, f := range files {
		t.Run(filepath.Base(filepath.Dir(f)), func(t *testing.T) {
			raw, err := os.ReadFile(f)
			if err != nil {
				t.Fatal(err)
			}
			phases, err := ParsePhases(raw)
			if err != nil {
				t.Fatal(err)
			}
			if len(phases) != 1 {
				t.Fatalf("phases = %d, want 1", len(phases))
			}
			g := phases[0].Grid
			for name, values := range g {
				for _, v := range values {
					if _, err := applyField(core.DefaultParams(), name, v); err != nil {
						t.Fatalf("apply %s=%v: %v", name, v, err)
					}
				}
			}
			if !reflect.DeepEqual(g["UseStoch"], []float64{1}) {
				t.Errorf("UseStoch = %v, want [1]", g["UseStoch"])
			}
			for name, want := range minimal {
				have := map[float64]bool{}
				for _, v := range g[name] {
					have[v] = true
				}
				for _, v := range want {
					if !have[v] {
						t.Errorf("%s misses %v (have %v)", name, v, g[name])
					}
				}
			}
			for _, v := range g["StopDailyATR"] {
				if v <= 0 {
					t.Errorf("StopDailyATR=%v: the stop must never be switched off by calibration", v)
				}
			}
		})
	}
}
```

Сигнатуры из `internal/service/backtest/calibrate.go`: `type Grid map[string][]float64`, `func ParsePhases(raw []byte) ([]Phase, error)`, `func applyField(params any, name string, value float64) (any, error)`.

- [ ] **Step 2: Запустить — убедиться, что падает**

Run: `go test ./internal/service/backtest/ -run TestRSIZoneStochGrids`
Expected: FAIL `no cal_stoch.json found`.

- [ ] **Step 3: Создать сетки**

`data/params/rsi_zone/lent/cal_stoch.json` (литерал LENT: 4/25/75, EMA 200, ATR 14, стоп 1.0):

```json
{
  "_comment": "Тема stoch: подтверждение входа Stoch %D в окне ZoneWindowBars (симметричный триггер: крест RSI или Stoch на текущем баре, второй осциллятор в зоне в окне). Несвипуемые шесть полей прибиты к литералу LENT (4/25/75, EMA 200, ATR 14, стоп 1.0). Запуск: go run ./cmd/backtest -ticker LENT -strategy rsi_zone -interval Minutes30 -calibrate data/params/rsi_zone/lent/cal_stoch.json -out ./reports/LENT_zone/stoch -months 36 -train-months 12 -test-months 6 -min-trades 20 -metric profit_factor.",
  "phases": [
    {
      "name": "stoch",
      "grid": {
        "RSIPeriod": [4],
        "RSILower": [25],
        "RSIUpper": [75],
        "EMAPeriod": [200],
        "DailyATRPeriod": [14],
        "StopDailyATR": [1.0],
        "UseStoch": [1],
        "StochKPeriod": [5, 9, 14],
        "StochDSmooth": [1, 3],
        "StochLower": [10, 15, 20, 25, 30],
        "ZoneWindowBars": [1, 2, 3, 5, 8]
      }
    }
  ]
}
```

`data/params/rsi_zone/domrf/cal_stoch.json` (литерал DOMRF: 4/25/75, EMA 50, ATR 14, стоп 1.0; нештатный протокол из-за короткой истории):

```json
{
  "_comment": "Тема stoch: подтверждение входа Stoch %D в окне ZoneWindowBars (симметричный триггер: крест RSI или Stoch на текущем баре, второй осциллятор в зоне в окне). Несвипуемые шесть полей прибиты к литералу DOMRF (4/25/75, EMA 50, ATR 14, стоп 1.0). История DOMRF с ~2025-12 — протокол нештатный, как у прочих тем DOMRF. Запуск: go run ./cmd/backtest -ticker DOMRF -strategy rsi_zone -interval Minutes30 -calibrate data/params/rsi_zone/domrf/cal_stoch.json -out ./reports/DOMRF_zone/stoch -months 10 -train-months 3 -test-months 2 -min-trades 15 -metric profit_factor.",
  "phases": [
    {
      "name": "stoch",
      "grid": {
        "RSIPeriod": [4],
        "RSILower": [25],
        "RSIUpper": [75],
        "EMAPeriod": [50],
        "DailyATRPeriod": [14],
        "StopDailyATR": [1.0],
        "UseStoch": [1],
        "StochKPeriod": [5, 9, 14],
        "StochDSmooth": [1, 3],
        "StochLower": [10, 15, 20, 25, 30],
        "ZoneWindowBars": [1, 2, 3, 5, 8]
      }
    }
  ]
}
```

- [ ] **Step 4: Запустить сторожевой тест**

Run: `go test ./internal/service/backtest/ -run 'TestRSIZone'`
Expected: PASS (включая существующие `TestRSIZoneGridMatchesTheSpec` и тесты сеток тикеров).

- [ ] **Step 5: Команда калибровки**

В `.claude/commands/rsi-zone-calibrate.md`, §2:

Абзац «Шесть полей `core.Params`: …» заменить на:

```markdown
Шесть базовых полей `core.Params`: `RSIPeriod`, `RSILower`, `RSIUpper`, `EMAPeriod`, `DailyATRPeriod`,
`StopDailyATR`. Тема `stoch` добавляет пять полей гейта стохастика: `UseStoch`, `StochKPeriod`,
`StochDSmooth`, `StochLower`, `ZoneWindowBars`. Остальные темы их не перечисляют: там гейт выключен
(`UseStoch = 0`). Файлы кладутся в `data/params/rsi_zone/<t>/`. У каждого файла `_comment`: что
меряет тема, команда запуска, а после прогона — строка `РЕЗУЛЬТАТ <дата>` с pooled OOS, пулом,
пофолдовыми PF и голосами фолдов.
```

В таблицу тем после строки `phased` добавить:

```markdown
| `stoch` | `cal_stoch.json` | `UseStoch` 1 × `StochKPeriod` 5,9,14 × `StochDSmooth` 1,3 × `StochLower` 10,15,20,25,30 × `ZoneWindowBars` 1,2,3,5,8 (150 комбинаций, шесть базовых полей прибиты к baseline) |
```

В список «Жёсткие инварианты сеток» добавить пункт:

```markdown
- В `cal_stoch.json` перечислены все четыре оси стохастика и `UseStoch` = [1]. Ось, которой нет в
  файле, молча измерится на дефолте ядра (14/3/20/1). Сторож — `rsi_zone_stoch_grid_test.go`.
```

- [ ] **Step 6: Документация механики**

`docs/rsi_zone/strategy.md`, §2 «Параметры»: в таблицу после строки `StopDailyATR` добавить:

```markdown
| `UseStoch` | флаг | 0 | stoch: 1 |
| `StochKPeriod` | баров | 0 → 14 | stoch: 5, 9, 14 |
| `StochDSmooth` | баров | 0 → 3 | stoch: 1, 3 |
| `StochLower` | пункты Stoch | 0 → 20 | stoch: 10, 15, 20, 25, 30 |
| `ZoneWindowBars` | баров | 0 → 1 | stoch: 1, 2, 3, 5, 8 |
```

и под абзацем про `StopDailyATR = 0`:

```markdown
Поля стохастика действуют только при `UseStoch = 1`. Нулевое значение разрешается в дефолт после
стрелки, поэтому литерал тикера включает гейт одним `UseStoch = 1`. Отрицательный период
стохастика — ошибка конфигурации, вход не открывается.
```

§3 «Вход»: после пункта 2 «**RSI-крест вниз:** …» (перед пунктом 3 «**Тренд:**») добавить абзац с отступом списка:

```markdown
**Подтверждение стохастиком (`UseStoch = 1`).** Пункт выше заменяется симметричным триггером.
На текущем баре `i` вниз пересекает свою нижнюю границу RSI (`RSILower`) или Stoch %D
(`StochLower`). Второй осциллятор при этом должен быть строго ниже своей границы хотя бы на одном
баре окна `[i − ZoneWindowBars + 1 … i]`. `ZoneWindowBars = 1` значит «оба в зоне на одном баре».
Бары прогрева индикаторов не засчитываются; настоящий 0.00 после прогрева засчитывается. Если на
баре пересекли оба, это один вход, в журнале триггером указан RSI. Остальные пункты (будний день,
тренд, дневной ATR, стоп) действуют без изменений. Журнал входа получает хвост «подтверждение: крест
дал …, … в зоне k бар(ов) назад». Stoch %D — `indicators.StochasticSeries`; при плоском диапазоне
цены за K баров %K равен 0, и такой бар считается «в зоне».
```

`Lookback` в `docs/rsi_zone/strategy.md` не описан — там ничего не добавлять.

- [ ] **Step 7: Полная проверка**

Run: `./bin/mage ci`
Expected: lint, `go test -race ./...` и проверка дрейфа моков — всё зелёное.

- [ ] **Step 8: Commit**

```bash
git add data/params/rsi_zone/lent/cal_stoch.json data/params/rsi_zone/domrf/cal_stoch.json internal/service/backtest/rsi_zone_stoch_grid_test.go .claude/commands/rsi-zone-calibrate.md docs/rsi_zone/strategy.md
git commit -m "feat(rsi_zone): тема калибровки stoch, сторож сеток и описание гейта

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

---

### Task 6: Оценка на LENT и DOMRF (прогоны, без кода)

Не условие мержа: результат идёт владельцу для решения. Все прогоны — строго последовательно и в один день.

**Files:**
- Modify: `_comment` в `data/params/rsi_zone/lent/cal_stoch.json` и `data/params/rsi_zone/domrf/cal_stoch.json` (строка `РЕЗУЛЬТАТ <дата>`)

**Interfaces:**
- Consumes: сетки из Task 5; точки литералов `data/params/rsi_zone/lent/baseline_point.json` и `data/params/rsi_zone/domrf/point.json`.

- [ ] **Step 1: Сверить, что файлы точек равны литералам**

Run: `python3 -c "import json;[print(f, json.load(open(f))['phases'][0]['grid']) for f in ['data/params/rsi_zone/lent/baseline_point.json','data/params/rsi_zone/domrf/point.json']]"`
Expected: LENT 4/25/75/EMA 200/ATR 14/стоп 1.0; DOMRF 4/25/75/EMA 50/ATR 14/стоп 1.0. Если не так — **СТОП**, доложи.

- [ ] **Step 2: LENT — литерал и тема `stoch`**

```bash
go run ./cmd/backtest -ticker LENT -strategy rsi_zone -interval Minutes30 -calibrate data/params/rsi_zone/lent/baseline_point.json -out ./reports/LENT_zone/stoch_base -months 36 -train-months 12 -test-months 6 -min-trades 20 -metric profit_factor
go run ./cmd/backtest -ticker LENT -strategy rsi_zone -interval Minutes30 -calibrate data/params/rsi_zone/lent/cal_stoch.json -out ./reports/LENT_zone/stoch -months 36 -train-months 12 -test-months 6 -min-trades 20 -metric profit_factor
```

Из обоих walk-forward отчётов выписать: pooled OOS PF, сделок в пуле, пофолдовые OOS PF и сделки, выбранные по фолдам значения четырёх осей стохастика. Долю стоп-выходов посчитать по журналам OOS: число строк с `| SL |` делить на число сделок.

- [ ] **Step 3: DOMRF — литерал и тема `stoch`**

```bash
go run ./cmd/backtest -ticker DOMRF -strategy rsi_zone -interval Minutes30 -calibrate data/params/rsi_zone/domrf/point.json -out ./reports/DOMRF_zone/stoch_base -months 10 -train-months 3 -test-months 2 -min-trades 15 -metric profit_factor
go run ./cmd/backtest -ticker DOMRF -strategy rsi_zone -interval Minutes30 -calibrate data/params/rsi_zone/domrf/cal_stoch.json -out ./reports/DOMRF_zone/stoch -months 10 -train-months 3 -test-months 2 -min-trades 15 -metric profit_factor
```

Выписать то же, что в Step 2.

- [ ] **Step 4: Шесть стопов LENT**

Если на LENT у голосов фолдов есть точка большинства (≥ 3 из 4 по каждой оси), собрать её плоским JSON и прогнать год:

```bash
go run ./cmd/backtest -ticker LENT -strategy rsi_zone -interval Minutes30 -params <scratchpad>/lent_stoch_point.json -months 12 -out ./reports/LENT_zone/stoch_year
```

Проверить, есть ли в журнале входы 2025-10-17 13:30, 2025-12-26 13:30, 2026-01-26 07:00, 2026-06-08 09:00, 2026-06-29 06:30, 2026-08-12 17:00. Если большинства нет — так и записать, этот шаг пропустить.

- [ ] **Step 5: Записать результат и закоммитить**

В `_comment` каждой сетки дописать `РЕЗУЛЬТАТ <дата>: литерал pooled X/N, stoch pooled Y/M; фолды …; голоса осей …; доля SL литерал a% → stoch b%; критерий спеки (PF выше, пул ≥ 20, доля SL ниже) — выполнен/не выполнен`. Для LENT добавить, какие из шести стопов отсечены.

```bash
git add data/params/rsi_zone/lent/cal_stoch.json data/params/rsi_zone/domrf/cal_stoch.json
git commit -m "calib(rsi_zone): тема stoch на LENT и DOMRF — результат оценки

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

- [ ] **Step 6: Доложить владельцу**

Таблица: тикер → литерал (PF/пул/доля SL) → stoch (PF/пул/доля SL) → голоса осей → критерий выполнен или нет. Литералы тикеров не менять: решение за владельцем.
