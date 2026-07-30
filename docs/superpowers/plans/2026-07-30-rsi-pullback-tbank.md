# rsi_pullback: подготовка ТБанка (T) к прогону — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [x]`) syntax for tracking.

**Goal:** Добавить в backtest-стратегию `rsi_pullback` по-тикерный пакет параметров ТБанка (тикер `T`), разложить тикер-специфичные гриды по подкаталогам, починить три красных теста и убрать молчаливое усечение окна свечей — чтобы прогон T запускался готовой командой и его отчёт был построен на полной истории.

**Architecture:** Тикер получает Go-пакет `tbank` с явным литералом `core.Params` (покомпонентная копия пост-грид конфига GAZP — осознанная гипотеза переносимости) и запись в `rsiPullbackRegistry`. Тикер-агностичные развёртки остаются в корне `data/params/rsi_pullback/`, тикер-специфичные фиксированные конфиги уезжают в подкаталоги `gazp/` и `t/`; грид-тесты начинают обходить каталог рекурсивно. Отдельно `CandleProvider.Load` учится добирать голову диапазона так же, как хвост.

**Tech Stack:** Go 1.25, стандартный `testing` (table-driven, без фреймворков), `mockery` v2 (мок `candleFetcher` уже сгенерирован: `internal/service/backtest/mocks`), `mage` для гейта качества.

## Global Constraints

- Спека: `docs/superpowers/specs/2026-07-30-rsi-pullback-tbank-design.md`. Ветка: `feat/rsi-pullback`.
- Движок стратегии `internal/service/trading_strategy/rsi_pullback/strategy/core/core.go` НЕ меняется, значения GAZP-конфига НЕ меняются.
- Прогон калибровки по сети — ВНЕ объёма. Ни один шаг плана не требует токена и не ходит в API.
- Live-обвязка GAZP — вне объёма.
- Комментарии в Go-коде и в `_comment` JSON — на английском (как во всём каталоге стратегии). Документация в `docs/` — на русском.
- Ни один грид-файл, включая подкаталоги, не может предлагать `StopDailyATR = 0`.
- Каждая задача заканчивается зелёным `go test ./internal/service/backtest/...`. Финальный гейт всей работы — `./bin/mage ci` (lint + `go test -race ./...` + проверка дрейфа моков).
- Сообщения коммитов: conventional-префикс + русский текст, как в истории ветки (`feat(rsi_pullback): ...`). В конце каждого коммита строка `Co-Authored-By: Claude Opus 5 <noreply@anthropic.com>`.
- Порядок задач обязателен: задачи 1–2 возвращают пакет `backtest` к зелёному, до этого любой новый тест невозможно отличить от уже сломанного.

**Исходная точка (проверено на HEAD `60ef593`):** три теста красные — `TestRSIPullbackBindingBuildsForTicker`, `TestRSIPullbackGridControlPoints`, `TestRSIPullbackGridEvaluationCost`.

## Файловая структура

| Файл | Ответственность | Задача |
| --- | --- | --- |
| `internal/service/backtest/rsi_pullback_registry_test.go` | тесты биндинга/реестра: baseline на некалиброванном тикере, литерал у калиброванного, наслоение JSON, гипотеза переносимости T←GAZP | 1, 4 |
| `internal/service/backtest/rsi_pullback_grid_test.go` | тесты грид-файлов: валидность полей, контрольные точки, стоимость прогона, рекурсивный обход, plateau-как-точка | 2, 5 |
| `data/params/rsi_pullback/grid.json` | тикер-агностичная фазная развёртка (`_comment` несёт формулу стоимости) | 2 |
| `internal/service/backtest/candles.go` | загрузка/кэш свечей; симметричный добор головы и хвоста | 3 |
| `internal/service/backtest/candles_test.go` | тесты провайдера свечей | 3 |
| `internal/service/trading_strategy/rsi_pullback/strategy/tbank/tbank.go` | тикер `T` и его стартовые параметры | 4 |
| `internal/service/backtest/rsi_pullback_registry.go` | реестр тикер→биндинг | 4 |
| `data/params/rsi_pullback/gazp/plateau_rsilower{15,20,25,30}.json` | пост-грид конфиг GAZP, по одной точке `RSILower` | 5 |
| `data/params/rsi_pullback/t/plateau_rsilower{15,20,25,30}.json` | перенесённый GAZP-конфиг на T, по одной точке `RSILower` | 5 |
| `data/params/rsi_pullback/cal_*.json` (8 файлов) | однотемные развёртки; команда в `_comment` перестаёт быть привязанной к GAZP | 5 |
| `docs/rsi_pullback/strategy.md` | справочник стратегии: раскладка параметров, порядок прогона нового тикера, фактические размеры гридов | 6 |

---

### Task 1: Тесты реестра перестают требовать, чтобы GAZP равнялся baseline

**Files:**
- Modify: `internal/service/backtest/rsi_pullback_registry_test.go:9-41`

**Interfaces:**
- Consumes: `RSIPullbackLookupOrGeneric(ticker string) Binding`, `core.DefaultParams() core.Params` — уже существуют.
- Produces: ничего для последующих задач (только правки тестов).

Причина: `gazp` уже несёт пост-грид литерал (`RSILower 25`, `EMASlow 70`, `SpentDayATR 0.9`, `TPDailyATR 0.7`, `VolBaseDays 7`, `VolLookbackBars 2`, `VolMult 1`), а тест сравнивает его с `core.DefaultParams()`. Проверку «пакет отслеживает baseline» надо вести на тикере, который его действительно отслеживает (`SBER`), а для калиброванного тикера пинить обратное — что он от baseline отвязан.

- [x] **Step 1: Заменить два теста в `rsi_pullback_registry_test.go`**

Полностью заменить `TestRSIPullbackBindingBuildsForTicker` (строки 9–25) и `TestRSIPullbackParseParamsLayersOverDefaults` (строки 27–41) на:

```go
// TestRSIPullbackBindingBuildsForTicker checks the wiring on a ticker whose package still
// tracks the baseline. SBER is deliberate: pinning this to a CALIBRATED ticker would turn
// every future calibration into a red test, which is exactly how this test broke before.
func TestRSIPullbackBindingBuildsForTicker(t *testing.T) {
	b := RSIPullbackLookupOrGeneric("SBER")
	p, ok := b.DefaultParams().(core.Params)
	if !ok {
		t.Fatalf("DefaultParams() returned %T, want core.Params", b.DefaultParams())
	}
	if p != core.DefaultParams() {
		t.Fatalf("DefaultParams() = %+v, want the baseline %+v", p, core.DefaultParams())
	}
	s := b.Build(p)
	if s.Ticker() != "SBER" {
		t.Fatalf("Ticker() = %q, want SBER", s.Ticker())
	}
	if s.Lookback() < 220 {
		t.Fatalf("Lookback() = %d, want >= 220", s.Lookback())
	}
}

// TestRSIPullbackCalibratedBindingKeepsItsOwnLiteral pins the opposite direction for a ticker
// that HAS been calibrated: GAZP must not drift back to the baseline. A package whose literal
// silently collapses into core.DefaultParams() would look calibrated while trading generic
// values, and the report would carry the ticker's name either way.
func TestRSIPullbackCalibratedBindingKeepsItsOwnLiteral(t *testing.T) {
	b := RSIPullbackLookupOrGeneric("GAZP")
	p, ok := b.DefaultParams().(core.Params)
	if !ok {
		t.Fatalf("DefaultParams() returned %T, want core.Params", b.DefaultParams())
	}
	if p == core.DefaultParams() {
		t.Fatal("GAZP returns the baseline: its calibrated literal was lost")
	}
	if got := b.Build(p).Ticker(); got != "GAZP" {
		t.Fatalf("Ticker() = %q, want GAZP", got)
	}
}

// TestRSIPullbackParseParamsLayersOverDefaults pins that partial calibration JSON overrides
// only the fields it names. It compares against the BINDING's own defaults, not the package
// baseline: for a calibrated ticker those differ, and the test is about layering, not baseline.
func TestRSIPullbackParseParamsLayersOverDefaults(t *testing.T) {
	b := RSIPullbackLookupOrGeneric("GAZP")
	base := b.DefaultParams().(core.Params)
	got, err := b.ParseParams([]byte(`{"RSILower": 10}`))
	if err != nil {
		t.Fatalf("ParseParams: %v", err)
	}
	p := got.(core.Params)
	if p.RSILower != 10 {
		t.Fatalf("RSILower = %v, want the JSON value 10", p.RSILower)
	}
	if p.RSIUpper != base.RSIUpper {
		t.Fatalf("RSIUpper = %v, want the binding default %v (partial JSON must not zero other fields)",
			p.RSIUpper, base.RSIUpper)
	}
	if p.EMASlow != base.EMASlow {
		t.Fatalf("EMASlow = %v, want the binding default %v", p.EMASlow, base.EMASlow)
	}
}
```

- [x] **Step 2: Прогнать тесты реестра**

Run: `go test ./internal/service/backtest/ -run 'TestRSIPullbackBinding|TestRSIPullbackCalibrated|TestRSIPullbackParseParams' -v`
Expected: PASS все четыре (`...BindingBuildsForTicker`, `...CalibratedBindingKeepsItsOwnLiteral`, `...ParseParamsLayersOverDefaults`, `...ParseParamsRejectsGarbage`).

- [x] **Step 3: Коммит**

```bash
git add internal/service/backtest/rsi_pullback_registry_test.go
git commit -m "$(cat <<'EOF'
test(rsi_pullback): тесты реестра не привязаны к baseline на калиброванном тикере
Проверка «пакет отслеживает baseline» переехала на SBER, для GAZP добавлена
обратная проверка — калиброванный литерал не должен схлопнуться в дефолты.
Наслоение частичного JSON сверяется с дефолтами биндинга, а не пакета core.

Co-Authored-By: Claude Opus 5 <noreply@anthropic.com>
EOF
)"
```

---

### Task 2: Контрольные точки и стоимость грида приводятся к факту

**Files:**
- Modify: `internal/service/backtest/rsi_pullback_grid_test.go:114-186`
- Modify: `data/params/rsi_pullback/grid.json` (только `_comment`)

**Interfaces:**
- Consumes: хелперы `rsiPullbackGrid(t)`, `rsiPullbackGridFiles(t)`, `rsiPullbackPhases(t, path)` из того же файла (существуют, строки 17–48).
- Produces: ничего для последующих задач.

Две причины падений. Первая: точка `UseDayATRGate = 0` живёт не в `grid.json`, а в `cal_screen.json` — так и записано в `_comment` файла `cal_day.json` («The UseDayATRGate=0 control point lives in cal_screen.json»). Тест должен требовать существование контроля в НАБОРЕ файлов стратегии, иначе он запрещает осмысленную реорганизацию. Вторая: risk-фаза выросла до 4×4, фактическая стоимость — 277 вызовов (9 + 6×12 + 5×12 + 5×12 + 4×16 + 4×3), а тест и `_comment` всё ещё называют 243.

- [x] **Step 1: Заменить `TestRSIPullbackGridControlPoints` (строки 114–160)**

```go
// TestRSIPullbackGridControlPoints pins the deliberate on/off points. The two optional gates
// must be sweepable to "off" SOMEWHERE in the shipped set — the control lives in
// cal_screen.json, not in grid.json, and pinning it to one file forbids that split — while the
// stop must never be sweepable to zero anywhere, and the full grid must test a target above the
// stop so the reward-to-risk asymmetry does not stay an assumption.
func TestRSIPullbackGridControlPoints(t *testing.T) {
	var sawDayOff, sawVolumeOff bool
	for _, path := range rsiPullbackGridFiles(t) {
		for _, ph := range rsiPullbackPhases(t, path) {
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
		}
	}
	if !sawDayOff {
		t.Fatal("no UseDayATRGate=0 control point in any grid file: the day gate can never be measured against off")
	}
	if !sawVolumeOff {
		t.Fatal("no UseVolume=0 control point in any grid file: the volume gate can never be measured against off")
	}

	var sawStop, sawTPAboveStop bool
	maxStop := 0.0
	for _, ph := range rsiPullbackGrid(t) {
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
	if !sawStop {
		t.Fatal("the grid never sweeps StopDailyATR")
	}
	if !sawTPAboveStop {
		t.Fatal("the grid never tests a target above the stop: the reward-to-risk asymmetry stays untested")
	}
}
```

- [x] **Step 2: Обновить ожидаемую стоимость в `TestRSIPullbackGridEvaluationCost`**

Заменить блок в конце теста (строка ~183):

```go
	if total != 277 {
		t.Fatalf("phased calibration costs %d evaluations, want the documented 277", total)
	}
```

- [x] **Step 3: Обновить формулу в `_comment` файла `data/params/rsi_pullback/grid.json`**

Найти подстроку `9 + 6x9 + 5x12 + 5x12 + 4x12 + 4x3 = up to 243 backtest evaluations` и заменить на:

```
9 + 6x12 + 5x12 + 5x12 + 4x16 + 4x3 = up to 277 backtest evaluations
```

Остальной текст комментария (включая абзац про отсутствие `StopDailyATR=0`) не трогать.

- [x] **Step 4: Прогнать все тесты пакета — он должен стать полностью зелёным**

Run: `go test ./internal/service/backtest/`
Expected: `ok` без FAIL. Если что-то падает — это регресс, а не наследство: три исходных красных теста закрыты задачами 1 и 2.

- [x] **Step 5: Коммит**

```bash
git add internal/service/backtest/rsi_pullback_grid_test.go data/params/rsi_pullback/grid.json
git commit -m "$(cat <<'EOF'
test(rsi_pullback): контрольные точки ищутся по набору гридов, стоимость 277
Точка UseDayATRGate=0 живёт в cal_screen.json — тест больше не требует её
именно в grid.json, но падает, если контроля нет ни в одном файле.
Стоимость фазного прогона приведена к факту: risk-фаза выросла до 4x4,
9 + 6x12 + 5x12 + 5x12 + 4x16 + 4x3 = 277, число обновлено и в _comment.

Co-Authored-By: Claude Opus 5 <noreply@anthropic.com>
EOF
)"
```

---

### Task 3: `CandleProvider.Load` добирает голову диапазона

**Files:**
- Modify: `internal/service/backtest/candles.go:81-108`
- Test: `internal/service/backtest/candles_test.go` (добавить в конец файла)

**Interfaces:**
- Consumes: `p.readCache`, `p.writeCache`, `p.fetchRange(ctx, instrumentID, interval, from, to)`, `mergeCandles(a, b)`, `sliceWindow(candles, from, to)`, `logger.Warn(msg string, fields ...any)` — все существуют.
- Produces: поведение `Load` — при тёплом, но коротком кэше возвращает полный запрошенный диапазон. Никаких новых экспортируемых имён.

Дефект: `Load` догружает только хвост (`last.Before(to)`), голову — никогда. Кэш T покрывает ~12 месяцев 30m, поэтому прогон с `-months 24` вернул бы усечённое окно, walk-forward построился бы на половине истории, и ни ошибки, ни предупреждения не было бы — отчёт выглядел бы валидным.

Отдельно оговорено поведение при легитимно короткой истории: `fetchRange` возвращает ошибку, если API не отдал ни одной свечи. Для головы это нормальная ситуация (инструмент не торговался так давно), поэтому голова не роняет прогон — пишется предупреждение через `logger.Warn`, и загрузка продолжается с тем, что есть. Молчание заменяется явным сигналом, но молодой инструмент по-прежнему прогоняется.

- [x] **Step 1: Написать падающий тест**

Добавить в конец `internal/service/backtest/candles_test.go`:

```go
// TestLoadFetchesMissingHead pins that a warm but SHORT cache does not silently truncate the
// requested window. Load used to top up only the tail, so a cache starting later than `from`
// returned a shorter series with no error at all — a walk-forward would then be built on part
// of the history while the report still claimed the full range.
func TestLoadFetchesMissingHead(t *testing.T) {
	base := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	// Cache holds hours 2..4 only; the caller asks for hours 0..4.
	cached := []backtest.Candle{
		{Time: base.Add(2 * time.Hour), Close: 12},
		{Time: base.Add(3 * time.Hour), Close: 13},
		{Time: base.Add(4 * time.Hour), Close: 14},
	}
	dir := t.TempDir()
	raw, err := json.Marshal(cached)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "T_Hour1.json"), raw, 0o600); err != nil {
		t.Fatal(err)
	}

	var reqFrom, reqTo time.Time
	var calls int
	m := mocks.NewMockcandleFetcher(t)
	m.EXPECT().GetCandles(mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		RunAndReturn(func(_ context.Context, _ *string, _ int32, from, to *timestamppb.Timestamp, _ *int32, _ bool) ([]*model.CandleItemTechAnalyse, error) {
			calls++
			reqFrom, reqTo = from.AsTime().UTC(), to.AsTime().UTC()
			return []*model.CandleItemTechAnalyse{
				bar(base, 10, true),
				bar(base.Add(time.Hour), 11, true),
			}, nil
		})

	p := NewCandleProvider(m, dir)
	got, err := p.Load(context.Background(), "T", "id-T", enum.Hour1, base, base.Add(4*time.Hour), false)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 5 {
		t.Fatalf("loaded %d candles, want 5 — the head of the window was not fetched", len(got))
	}
	if !got[0].Time.Equal(base) {
		t.Fatalf("first candle = %s, want the requested from %s", got[0].Time, base)
	}
	if calls != 1 {
		t.Fatalf("fetcher called %d times, want exactly 1 (head only; the tail is already cached)", calls)
	}
	if !reqFrom.Equal(base) || !reqTo.Equal(base.Add(2*time.Hour)) {
		t.Fatalf("head fetch asked for [%s, %s], want [%s, %s] — only the missing head",
			reqFrom, reqTo, base, base.Add(2*time.Hour))
	}
}

// TestLoadShortHistoryDoesNotFail pins the legitimate case behind the head top-up: an
// instrument that simply did not trade as far back as `from`. The head fetch returns nothing,
// and Load must still return the cached window instead of failing the whole run.
func TestLoadShortHistoryDoesNotFail(t *testing.T) {
	base := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	cached := []backtest.Candle{
		{Time: base.Add(2 * time.Hour), Close: 12},
		{Time: base.Add(3 * time.Hour), Close: 13},
	}
	dir := t.TempDir()
	raw, err := json.Marshal(cached)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "T_Hour1.json"), raw, 0o600); err != nil {
		t.Fatal(err)
	}

	m := mocks.NewMockcandleFetcher(t)
	m.EXPECT().GetCandles(mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(nil, nil) // no candles that far back

	p := NewCandleProvider(m, dir)
	got, err := p.Load(context.Background(), "T", "id-T", enum.Hour1, base, base.Add(3*time.Hour), false)
	if err != nil {
		t.Fatalf("Load failed on an instrument with a short history: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("loaded %d candles, want the 2 cached ones", len(got))
	}
}
```

Дополнить блок импортов файла `candles_test.go` (сейчас строки 3–15) до:

```go
import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/mock"
	"google.golang.org/protobuf/types/known/timestamppb"

	"tinvest/internal/domain/backtest"
	"tinvest/internal/enum"
	"tinvest/internal/model"
	"tinvest/internal/service/backtest/mocks"
)
```

- [x] **Step 2: Прогнать тест — он должен упасть**

Run: `go test ./internal/service/backtest/ -run 'TestLoadFetchesMissingHead|TestLoadShortHistoryDoesNotFail' -v`
Expected: FAIL. `TestLoadFetchesMissingHead` — «loaded 3 candles, want 5 — the head of the window was not fetched» (мок вообще не будет вызван, mockery дополнительно сообщит о неудовлетворённом ожидании). `TestLoadShortHistoryDoesNotFail` может пройти уже сейчас — это нормально, он охраняет поведение, вводимое следующим шагом.

- [x] **Step 3: Реализовать симметричный добор**

В `internal/service/backtest/candles.go` заменить хвост функции `Load` (строки 96–107, начиная с `last := cached[len(cached)-1].Time`) на:

```go
	// The head and the tail are topped up symmetrically. Topping up only the tail let a warm
	// but SHORT cache return a truncated window with no error: a run asking for 24 months on a
	// 12-month cache would quietly calibrate on half the history. A missing head is NOT fatal,
	// though — an instrument may simply not have traded that far back, so we warn and continue
	// with what exists instead of failing the run.
	dirty := false
	if first := cached[0].Time; first.After(from) {
		head, ferr := p.fetchRange(ctx, instrumentID, interval, from, first)
		if ferr != nil {
			logger.Warn(fmt.Sprintf("backtest: %s (%s) has no candles before %s — the window starts where its history does, not at %s",
				ticker, interval.String(), first, from))
		} else {
			cached = mergeCandles(cached, head)
			dirty = true
		}
	}
	if last := cached[len(cached)-1].Time; last.Before(to) {
		tail, ferr := p.fetchRange(ctx, instrumentID, interval, last, to)
		if ferr != nil {
			return nil, ferr
		}
		cached = mergeCandles(cached, tail)
		dirty = true
	}
	if dirty {
		if werr := p.writeCache(path, cached); werr != nil {
			return nil, werr
		}
	}
	return sliceWindow(cached, from, to), nil
```

Хвост сохраняет прежнюю семантику (ошибка возвращается): отсутствие свежих данных означает проблему с загрузкой, а не короткую историю инструмента.

- [x] **Step 4: Прогнать тесты провайдера**

Run: `go test ./internal/service/backtest/ -run 'TestLoad' -v`
Expected: PASS все три (`TestLoadNoFileFetchesAndCaches`, `TestLoadFetchesMissingHead`, `TestLoadShortHistoryDoesNotFail`). Важно: `TestLoadNoFileFetchesAndCaches` проверяет, что тёплый полный кэш не рефетчится — если он покраснел, добор головы срабатывает там, где нечего добирать.

- [x] **Step 5: Прогнать весь пакет**

Run: `go test ./internal/service/backtest/`
Expected: `ok`.

- [x] **Step 6: Коммит**

```bash
git add internal/service/backtest/candles.go internal/service/backtest/candles_test.go
git commit -m "$(cat <<'EOF'
fix(backtest): кэш свечей добирает голову диапазона, а не только хвост
Тёплый, но короткий кэш молча отдавал усечённое окно: прогон на -months 24
при 12-месячном кэше строил walk-forward на половине истории без единой
ошибки. Голова догружается симметрично хвосту; пустая голова не роняет
прогон (инструмент мог не торговаться так давно) — пишется предупреждение.

Co-Authored-By: Claude Opus 5 <noreply@anthropic.com>
EOF
)"
```

---

### Task 4: Пакет `tbank` и регистрация тикера `T`

**Files:**
- Create: `internal/service/trading_strategy/rsi_pullback/strategy/tbank/tbank.go`
- Modify: `internal/service/backtest/rsi_pullback_registry.go:7-20` (импорты), `:45-59` (карта)
- Test: `internal/service/backtest/rsi_pullback_registry_test.go` (добавить в конец файла)

**Interfaces:**
- Consumes: `core.Params`, `core.DefaultParams()`; `rsiPullbackBindingFor(ticker string, defaults func() core.Params) Binding`.
- Produces: `tbank.Ticker` (константа, `"T"`) и `tbank.DefaultParams() core.Params` — их использует реестр и тест переносимости.

Имя пакета — `tbank`, не `t`: однобуквенный пакет дал бы `t.DefaultParams()` на вызове и столкнулся бы с идиомой `t *testing.T`. Тикером в API остаётся `"T"`.

- [x] **Step 1: Написать падающий тест переносимости**

Добавить в конец `internal/service/backtest/rsi_pullback_registry_test.go`:

```go
// TestRSIPullbackTBankStartsFromGAZPConfig pins a DELIBERATE hypothesis, not a fact: T starts
// from GAZP's post-grid literal to test whether parameters transfer between liquid names. The
// equality is pinned so the link cannot dissolve unnoticed — once T is calibrated on its own
// data, this test must be rewritten to pin T's own literal, and the rewrite is the moment the
// hypothesis gets consciously retired.
func TestRSIPullbackTBankStartsFromGAZPConfig(t *testing.T) {
	tb := RSIPullbackLookupOrGeneric("T").DefaultParams().(core.Params)
	gz := RSIPullbackLookupOrGeneric("GAZP").DefaultParams().(core.Params)
	if tb != gz {
		t.Fatalf("T params = %+v, want GAZP's %+v — T is seeded from the GAZP config on purpose", tb, gz)
	}
	if tb == core.DefaultParams() {
		t.Fatal("T returns the baseline: the GAZP seed was lost")
	}
	if got := RSIPullbackLookupOrGeneric("T").Build(tb).Ticker(); got != "T" {
		t.Fatalf("Ticker() = %q, want T", got)
	}
}
```

- [x] **Step 2: Прогнать тест — он должен упасть**

Run: `go test ./internal/service/backtest/ -run TestRSIPullbackTBankStartsFromGAZPConfig -v`
Expected: FAIL — `T` не зарегистрирован, `RSIPullbackLookupOrGeneric` отдаёт generic-биндинг с `core.DefaultParams()`, и первое же сравнение с GAZP не проходит.

- [x] **Step 3: Создать пакет `tbank`**

Создать `internal/service/trading_strategy/rsi_pullback/strategy/tbank/tbank.go`:

```go
// Package tbank supplies the ticker and starting rsi_pullback Params for T (T-Bank).
//
// Calibration has NOT been run for this ticker. The values below are a COPY of GAZP's
// post-grid literal, taken as an explicit hypothesis that parameters transfer between liquid
// names — they are not a claim that T is tuned. Two consequences follow. First, this package
// does NOT track core.DefaultParams(): a change to the baseline must not reach it silently,
// because these values came from a different instrument's calibration. Second, once -calibrate
// picks a winning combination for T, this literal must be REPLACED — leaving GAZP's numbers in
// place after a run of T's own would ship a config the data never endorsed.
//
// The package is named tbank rather than t: a one-letter package reads as t.DefaultParams() at
// the call site and collides with the t *testing.T idiom. The exchange ticker stays "T".
package tbank

import "tinvest/internal/service/trading_strategy/rsi_pullback/strategy/core"

// Ticker is the instrument this package configures.
const Ticker = "T"

// DefaultParams returns T's starting rsi_pullback parameters: GAZP's calibrated config,
// unverified on T.
func DefaultParams() core.Params {
	return core.Params{
		RSIPeriod:       4,
		RSILower:        25,
		RSIUpper:        70,
		EMAFast:         10,
		EMASlow:         70,
		DailyATRPeriod:  14,
		UseDayATRGate:   1,
		FreshDayATR:     0,
		SpentDayATR:     0.9,
		StopDailyATR:    0.5,
		TPDailyATR:      0.7,
		UseVolume:       0,
		VolBaseDays:     7,
		VolLookbackBars: 2,
		VolMult:         1,
	}
}
```

- [x] **Step 4: Зарегистрировать тикер**

В `internal/service/backtest/rsi_pullback_registry.go` добавить импорт после строки с `rsipullbacksfin` (алфавитный порядок по алиасу):

```go
	rsipullbacktbank "tinvest/internal/service/trading_strategy/rsi_pullback/strategy/tbank"
```

и запись в `rsiPullbackRegistry` после строки `rsipullbacksfin.Ticker`:

```go
	rsipullbacktbank.Ticker: rsiPullbackBindingFor(rsipullbacktbank.Ticker, rsipullbacktbank.DefaultParams),
```

- [x] **Step 5: Прогнать тесты и `gofmt`**

Run: `gofmt -l ./internal && go test ./internal/service/backtest/ -run TestRSIPullback -v`
Expected: `gofmt -l` не печатает файлов; все `TestRSIPullback*` — PASS. Особо проверить `TestRSIPullbackRegistryEntriesMatchTheirTicker/T` (пакет зарегистрирован под тем тикером, который объявляет) и `TestRSIPullbackRegistryKeepsTheStopArmed` (стоп не нулевой).

- [x] **Step 6: Прогнать весь пакет**

Run: `go test ./internal/service/backtest/`
Expected: `ok`.

- [x] **Step 7: Коммит**

```bash
git add internal/service/trading_strategy/rsi_pullback/strategy/tbank/tbank.go \
        internal/service/backtest/rsi_pullback_registry.go \
        internal/service/backtest/rsi_pullback_registry_test.go
git commit -m "$(cat <<'EOF'
feat(rsi_pullback): пакет tbank под тикер T со стартом от GAZP-конфига
Значения — копия пост-грид литерала GAZP, взятая как гипотеза переносимости
параметров между ликвидными именами, а не как признак настроенного тикера:
это сказано в doc-комментарии, а равенство T==GAZP запинено тестом, чтобы
связь нельзя было расторгнуть молча. Имя пакета tbank, а не t: однобуквенный
пакет читается как t.DefaultParams() и конфликтует с идиомой t *testing.T.

Co-Authored-By: Claude Opus 5 <noreply@anthropic.com>
EOF
)"
```

---

### Task 5: Раскладка гридов по тикерам и плато-набор T

**Files:**
- Move: `data/params/rsi_pullback/plateau_rsilower{15,20,25,30}.json` → `data/params/rsi_pullback/gazp/`
- Create: `data/params/rsi_pullback/t/plateau_rsilower{15,20,25,30}.json`
- Modify: `data/params/rsi_pullback/cal_*.json` (8 файлов, только `_comment`)
- Modify: `internal/service/backtest/rsi_pullback_grid_test.go:38-48` (обход каталога) + новый тест в конец файла

**Interfaces:**
- Consumes: `rsiPullbackParamsDir` (константа, строка 13), `rsiPullbackPhases(t, path)`, `ParsePhases(raw []byte) ([]Phase, error)`, `applyField(p core.Params, name string, v float64) (core.Params, error)`.
- Produces: `rsiPullbackGridFiles(t)` начинает возвращать пути и из подкаталогов — на него опираются `TestRSIPullbackCalFilesValid`, `TestRSIPullbackGridControlPoints` (задача 2) и новый plateau-тест.

- [x] **Step 1: Сделать обход каталога рекурсивным**

В `internal/service/backtest/rsi_pullback_grid_test.go` заменить `rsiPullbackGridFiles` (строки 38–48) на:

```go
// rsiPullbackGridFiles lists every grid file shipped for the strategy, including the
// per-ticker subdirectories (gazp/, t/). The walk is recursive on purpose: ticker-specific
// fixed configs live in subdirectories, and a flat glob would silently exempt exactly those
// files from the field and stop checks below.
func rsiPullbackGridFiles(t *testing.T) []string {
	t.Helper()
	var files []string
	err := filepath.WalkDir(rsiPullbackParamsDir, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() || filepath.Ext(path) != ".json" {
			return nil
		}
		files = append(files, path)
		return nil
	})
	if err != nil {
		t.Fatalf("walk grids: %v", err)
	}
	if len(files) < 2 {
		t.Fatalf("expected the phased grid plus the cal_*.json files, found %d", len(files))
	}
	return files
}
```

Добавить `"io/fs"` в импорты файла (получится блок: `encoding/json`, `io/fs`, `os`, `path/filepath`, `strings`, `testing`, затем `tinvest/internal/service/trading_strategy/rsi_pullback/strategy/core`).

- [x] **Step 2: Добавить тест «плато-файл — это точка»**

Добавить в конец `internal/service/backtest/rsi_pullback_grid_test.go`:

```go
// TestRSIPullbackPlateauFilesArePoints pins what makes a plateau check meaningful: every key
// carries exactly ONE value, so each walk-forward fold has a single combo to rank and the
// calibrator makes no choice at all. The pooled OOS profit factor then belongs to that fixed
// configuration. Let any key carry two values and the number silently becomes the result of a
// selection procedure — which is the very thing a plateau check exists to rule out.
func TestRSIPullbackPlateauFilesArePoints(t *testing.T) {
	var seen int
	for _, path := range rsiPullbackGridFiles(t) {
		name := filepath.Base(path)
		if !strings.HasPrefix(name, "plateau_") {
			continue
		}
		seen++
		rel := filepath.Join(filepath.Base(filepath.Dir(path)), name)
		t.Run(rel, func(t *testing.T) {
			for _, ph := range rsiPullbackPhases(t, path) {
				for field, values := range ph.Grid {
					if len(values) != 1 {
						t.Fatalf("phase %q pins %s over %d values: a plateau file must carry exactly one value per key",
							ph.Name, field, len(values))
					}
				}
			}
		})
	}
	if seen == 0 {
		t.Fatal("no plateau_*.json files found: the plateau checks are part of the shipped set")
	}
}
```

- [x] **Step 3: Прогнать грид-тесты — они должны проходить на текущей плоской раскладке**

Run: `go test ./internal/service/backtest/ -run 'TestRSIPullbackGrid|TestRSIPullbackCalFiles|TestRSIPullbackPlateau' -v`
Expected: PASS. Рекурсивный обход на плоском каталоге даёт тот же набор файлов, а четыре существующих `plateau_*` уже фиксируют по одному значению на ключ — тест проверяет это до переезда, чтобы переезд можно было отличить от содержательной поломки.

- [x] **Step 4: Переместить GAZP-плато в подкаталог**

```bash
mkdir -p data/params/rsi_pullback/gazp
git mv data/params/rsi_pullback/plateau_rsilower15.json data/params/rsi_pullback/gazp/
git mv data/params/rsi_pullback/plateau_rsilower20.json data/params/rsi_pullback/gazp/
git mv data/params/rsi_pullback/plateau_rsilower25.json data/params/rsi_pullback/gazp/
git mv data/params/rsi_pullback/plateau_rsilower30.json data/params/rsi_pullback/gazp/
```

В каждом из четырёх файлов в `_comment` поправить путь в команде запуска: `data/params/rsi_pullback/plateau_rsilower15.json` → `data/params/rsi_pullback/gazp/plateau_rsilower15.json` (аналогично для 20, 25, 30). Остальной текст, включая «post-grid GAZP config as of 2026-07-30», не трогать — эти файлы GAZP-специфичны по содержанию, и `-ticker GAZP` в их команде остаётся правильным.

- [x] **Step 5: Создать плато-набор T**

Создать `data/params/rsi_pullback/t/plateau_rsilower25.json` — точка, равная GAZP-конфигу:

```json
{
  "_comment": "PLATEAU CHECK for T (T-Bank), point RSILower=25 — the GAZP post-grid config applied to T as-is. Every key carries exactly ONE value, so each walk-forward fold has a single combo to rank and the calibrator makes no choice: the pooled OOS profit factor belongs to this fixed configuration, not to a selection procedure. This file and its three siblings (15, 20, 30) test the TRANSFER hypothesis behind the tbank package before any calibration of T is run: a broad plateau above 1.5 means GAZP's parameters describe something real about 30-minute RSI pullbacks; a lone spike at 25 with neighbours near 1.0 means the config fits GAZP specifically and T needs its own grid. Run: go run ./cmd/backtest -ticker T -strategy rsi_pullback -interval Minutes30 -calibrate data/params/rsi_pullback/t/plateau_rsilower25.json -out ./reports/T_plateau -months 24 -train-months 12 -test-months 3 -min-trades 1 -metric profit_factor -refresh",
  "phases": [
    {
      "name": "fixed",
      "grid": {
        "RSILower": [25],
        "RSIPeriod": [4],
        "RSIUpper": [70],
        "EMAFast": [10],
        "EMASlow": [70],
        "DailyATRPeriod": [14],
        "UseDayATRGate": [1],
        "FreshDayATR": [0],
        "SpentDayATR": [0.9],
        "StopDailyATR": [0.5],
        "TPDailyATR": [0.7],
        "UseVolume": [0]
      }
    }
  ]
}
```

Создать три соседних файла — `t/plateau_rsilower15.json`, `t/plateau_rsilower20.json`, `t/plateau_rsilower30.json` — идентичных, с тремя отличиями в каждом: значение `"RSILower"` (`[15]`, `[20]`, `[30]` соответственно), число в начале `_comment` (`point RSILower=15` и т.д.) и путь к файлу в команде запуска внутри `_comment`. Во всех трёх фраза «the GAZP post-grid config applied to T as-is» заменяется на «a neighbour of the GAZP post-grid config (RSILower moved) applied to T», потому что as-is GAZP-конфигом является только точка 25.

Флаг `-refresh` в командах обязателен для первого прогона T: кэш `data/candles/T_*.json` покрывает лишь ~12 месяцев 30m и устаревший Day1.

- [x] **Step 6: Обобщить команды в тикер-агностичных развёртках**

В восьми файлах `data/params/rsi_pullback/cal_*.json` (`cal_day`, `cal_day_spent`, `cal_entry`, `cal_exit`, `cal_risk`, `cal_screen`, `cal_trend`, `cal_volume`) в конце `_comment` после существующей команды `Run: go run ./cmd/backtest -ticker GAZP ...` добавить предложение:

```
GAZP is only the example instrument here: this file sweeps one theme and is ticker-agnostic, so replace -ticker and -out to calibrate another name (e.g. -ticker T -out ./reports/T_<theme>, adding -refresh on a ticker whose candle cache is short).
```

`<theme>` в каждом файле заменить на его тему (`day`, `day_spent`, `entry`, `exit`, `risk`, `screen`, `trend`, `volume`). Имя самого файла из `_comment` не убирать — `TestRSIPullbackCalFilesValid` требует, чтобы комментарий называл свой файл.

- [x] **Step 7: Прогнать грид-тесты на новой раскладке**

Run: `go test ./internal/service/backtest/ -run 'TestRSIPullbackGrid|TestRSIPullbackCalFiles|TestRSIPullbackPlateau' -v`
Expected: PASS. `TestRSIPullbackPlateauFilesArePoints` должен показать восемь подтестов (`gazp/plateau_*` и `t/plateau_*`) — если их четыре, рекурсивный обход не подхватил подкаталог.

- [x] **Step 8: Проверить JSON-валидность и число точек глазами**

Run: `python3 -c "import json,glob;[json.load(open(f)) for f in glob.glob('data/params/rsi_pullback/**/*.json',recursive=True)];print('json ok', len(glob.glob('data/params/rsi_pullback/**/*.json',recursive=True)))"`
Expected: `json ok 17` (grid.json + 8 cal + 4 gazp/plateau + 4 t/plateau).

- [x] **Step 9: Коммит**

```bash
git add data/params/rsi_pullback internal/service/backtest/rsi_pullback_grid_test.go
git commit -m "$(cat <<'EOF'
feat(rsi_pullback): раскладка гридов по тикерам и плато-набор T

Плато-файлы GAZP уехали в gazp/ — их значения тикер-специфичны и в корне
вводили в заблуждение. Появился t/ с четырьмя точками вокруг RSILower:
точка 25 — GAZP-конфиг на T as-is, соседние показывают, плато это или
чужой спайк. Обход грид-файлов в тестах стал рекурсивным, иначе подкаталоги
молча выпали бы из проверок полей и запрета StopDailyATR=0; новый тест
требует от plateau-файлов ровно одного значения на ключ.

Co-Authored-By: Claude Opus 5 <noreply@anthropic.com>
EOF
)"
```

---

### Task 6: Документация — раскладка, порядок прогона T, фактические размеры гридов

**Files:**
- Modify: `docs/rsi_pullback/strategy.md:217-273` (раздел 8 и 8.0), `:335`

**Interfaces:**
- Consumes: факты, зафиксированные задачами 2–5.
- Produces: раздел, на который ссылается `CLAUDE.md` как на справочник стратегии.

Документация разошлась с файлами по всем размерам гридов — это проверено пересчётом: `grid.json` 243→**277**, `cal_day` 12→**120**, `cal_entry` 12→**20**, `cal_trend` 12→**36**, `cal_risk` 16→**24**, `cal_volume` 18→**24**. Совпадают только `cal_screen` (4), `cal_day_spent` (5), `cal_exit` (6).

- [x] **Step 1: Проверить размеры перед правкой (не доверять плану на слово)**

Run:
```bash
cd data/params/rsi_pullback && python3 - <<'EOF'
import json,glob
for f in sorted(glob.glob("*.json"))+sorted(glob.glob("*/*.json")):
    d=json.load(open(f)); seeds=1; total=0
    for ph in d["phases"]:
        n=1
        for v in ph["grid"].values(): n*=len(v)
        total+=seeds*n
        if ph.get("keepTop",0)>0: seeds=ph["keepTop"]
    print(f"{f:36s} {total}")
EOF
```
Expected: `grid.json 277`, `cal_day.json 120`, `cal_day_spent.json 5`, `cal_entry.json 20`, `cal_exit.json 6`, `cal_risk.json 24`, `cal_screen.json 4`, `cal_trend.json 36`, `cal_volume.json 24`, все восемь `plateau` — по `1`. Числа из вывода и идут в документацию.

- [x] **Step 2: Обновить формулу и число в разделе 8**

В `docs/rsi_pullback/strategy.md` строка ~230: `9 + 6·9 + 5·12 + 5·12 + 4·12 + 4·3 = 243` → `9 + 6·12 + 5·12 + 5·12 + 4·16 + 4·3 = 277`. В том же абзаце «243 вызова движка» → «277 вызовов движка». В строке ~264 «полный `grid.json` на 243 прогона» → «на 277 прогонов». В строке ~335 «тратить 243 прогона» → «тратить 277 прогонов».

- [x] **Step 3: Привести таблицу раздела 8.0 к факту и дополнить её плато-файлами**

Заменить таблицу (строки ~243–252) на:

```markdown
| Файл | Прогонов | Что меряет |
|---|---|---|
| `grid.json` | 277 | полная фазовая калибровка, шесть фаз подряд с наследованием seed'ов |
| `cal_screen.json` | 4 | цена двух опциональных гейтов в сделках (`UseDayATRGate` × `UseVolume`) |
| `cal_entry.json` | 20 | форма отката: `RSIPeriod` × `RSILower` |
| `cal_trend.json` | 36 | тренд: `EMAFast` × `EMASlow` |
| `cal_day.json` | 120 | пороги гейта дня при `UseDayATRGate = 1`, вместе с `RSILower`/`RSIPeriod` |
| `cal_day_spent.json` | 5 | только ветка «день исчерпан»: `FreshDayATR = 0` + свип `SpentDayATR` |
| `cal_volume.json` | 24 | фон объёмов при `UseVolume = 1`: `VolMult` × `VolBaseDays` |
| `cal_risk.json` | 24 | стоп и цель в дневных ATR |
| `cal_exit.json` | 6 | уровень выхода `RSIUpper` |
| `<ticker>/plateau_rsilower*.json` | 1 каждый | фиксированная точка: PF принадлежит конфигурации, а не отбору |
```

- [x] **Step 4: Добавить подраздел про по-тикерные пакеты и прогон нового тикера**

Вставить перед `### 8.1.` новый подраздел:

```markdown
### 8.0.1. По-тикерные пакеты и раскладка параметров

Стартовые параметры живут в Go, а не в JSON: каждому отслеживаемому тикеру принадлежит пакет
`internal/service/trading_strategy/rsi_pullback/strategy/<ticker>/`, и реестр
`rsiPullbackRegistry` связывает тикер с его биндингом. Незарегистрированный тикер прогоняется
на `core.DefaultParams()` (`RSIPullbackLookupOrGeneric`), а не падает.

Doc-комментарий пакета обязан честно называть одно из трёх состояний:

| Состояние | Тело `DefaultParams` | Пример |
|---|---|---|
| калибровка не проводилась | `return core.DefaultParams()` — тикер отслеживает baseline, правка дефолтов доходит до него | `sber`, ещё 11 тикеров |
| откалиброван | явный литерал; связь с baseline разорвана осознанно | `gazp` |
| засеян чужим конфигом | явный литерал-копия + пометка «гипотеза, не настроен» | `tbank` (тикер `T`, засеян GAZP-конфигом) |

Файлы параметров разложены по тому же принципу: тикер-агностичные развёртки (`grid.json`,
`cal_*.json`) лежат в корне `data/params/rsi_pullback/` и переиспользуются любым тикером —
в их командах GAZP только пример; тикер-специфичные фиксированные конфиги живут в
`data/params/rsi_pullback/<ticker>/`. Все файлы, включая подкаталоги, проходят
`TestRSIPullbackCalFilesValid` (имена полей резолвятся через `applyField`, `StopDailyATR = 0`
запрещён везде) и `TestRSIPullbackPlateauFilesArePoints` (в `plateau_*` ровно одно значение
на ключ).

Порядок прогона нового тикера — на примере T (ТБанк):

```bash
# 1. Цена гейтов в сделках + прогрев кэша до полных 24 месяцев.
#    -refresh обязателен на первом прогоне: кэш нового тикера обычно короче запрошенного окна.
go run ./cmd/backtest -ticker T -strategy rsi_pullback -interval Minutes30 \
  -calibrate data/params/rsi_pullback/cal_screen.json -out ./reports/T_screen \
  -months 24 -min-trades 1 -test-months 6 -metric profit_factor -refresh

# 2. Полная фазовая калибровка, 277 вызовов движка.
go run ./cmd/backtest -ticker T -strategy rsi_pullback -interval Minutes30 \
  -calibrate data/params/rsi_pullback/grid.json -out ./reports/T \
  -months 24 -min-trades 20 -test-months 6 -metric profit_factor

# 3. Плато вокруг перенесённого GAZP-конфига: четыре файла, сравнить pooled OOS PF.
go run ./cmd/backtest -ticker T -strategy rsi_pullback -interval Minutes30 \
  -calibrate data/params/rsi_pullback/t/plateau_rsilower25.json -out ./reports/T_plateau \
  -months 24 -train-months 12 -test-months 3 -min-trades 1 -metric profit_factor
```

Шаг 3 осмысленно запускать и ДО шага 2: `t/plateau_rsilower{15,20,25,30}.json` проверяют
гипотезу, ради которой `tbank` засеян конфигом GAZP. Точка 25 — этот конфиг as-is; широкое
плато выше 1.5 говорит, что параметры описывают саму механику 30-минутного отката, а
одинокий спайк при соседях около 1.0 означает, что конфиг подогнан под GAZP и T нужен свой
грид. После собственной калибровки T литерал в пакете `tbank` обязан быть перезаписан, а
`TestRSIPullbackTBankStartsFromGAZPConfig` — переписан под новые значения.
```

- [x] **Step 5: Обновить абзац про проверку файлов каталога**

В конце раздела 8.0 (строки ~269–273) в предложении «Все файлы каталога проверяются тестом `TestRSIPullbackCalFilesValid`…» добавить, что обход рекурсивный и покрывает подкаталоги `<ticker>/`, а требование «команда в `_comment` называет свой файл» относится к `cal_*.json`.

- [x] **Step 6: Проверить, что в документации не осталось расхождений**

Run: `grep -n "243\|4·12\|6·9" docs/rsi_pullback/strategy.md`
Expected: пусто (ни одного вхождения).

- [x] **Step 7: Финальный гейт качества**

Run: `./bin/mage ci`
Expected: lint без замечаний, `go test -race ./...` зелёный, проверка дрейфа моков зелёная. Если `./bin/mage` отсутствует — сначала `go run ./magefiles tools` согласно `docs/tooling/mage.md`.

- [x] **Step 8: Коммит**

```bash
git add docs/rsi_pullback/strategy.md
git commit -m "$(cat <<'EOF'
docs(rsi_pullback): раскладка параметров по тикерам и порядок прогона T

Размеры гридов приведены к факту (grid 243->277, cal_day 12->120,
cal_entry 12->20, cal_trend 12->36, cal_risk 16->24, cal_volume 18->24).
Новый подраздел: три состояния по-тикерного пакета, тикер-агностичные
развёртки против тикер-специфичных подкаталогов и готовые команды прогона
ТБанка, включая обязательный -refresh на первом прогоне.

Co-Authored-By: Claude Opus 5 <noreply@anthropic.com>
EOF
)"
```

---

## Проверка результата целиком

- [x] `./bin/mage ci` зелёный.
- [x] `go test ./internal/service/backtest/ -run TestRSIPullback -v` — все подтесты PASS, среди них `TestRSIPullbackRegistryEntriesMatchTheirTicker/T` и восемь подтестов `TestRSIPullbackPlateauFilesArePoints`.
- [x] `ls data/params/rsi_pullback/` — в корне только `grid.json` и восемь `cal_*.json`; подкаталоги `gazp/` и `t/` по четыре файла.
- [x] `git status` чистый, история ветки — шесть новых коммитов.

Прогон калибровки T (шаги из раздела 8.0.1 документации) выполняет владелец репозитория — он требует токена и сети и в этот план не входит.
