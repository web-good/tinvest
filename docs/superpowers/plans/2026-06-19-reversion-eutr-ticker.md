# EUTR reversion ticker — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Зарегистрировать EUTR (ЕвроТранс) как per-ticker reversion-тикер, проведя его через стадийную walk-forward калибровку на ~31 месяце Hour1-истории.

**Architecture:** EUTR повторяет структуру UGLD/SFIN — отдельный пакет `…/reversion/strategy/eutr` с `Ticker` + `DefaultParams()`, регистрация в `reversionRegistry`, набор staged grid-файлов в `data/params/eutr/`. Движок `core` не меняется. Калибровка идёт послойно: победитель каждого слоя фиксируется в `DefaultParams`, и следующий слой наследует его через `ParseParams` (стартует от defaults, накладывает частичный JSON грида).

**Tech Stack:** Go 1.25, `cmd/backtest` walk-forward (rolling, train/OOS), JSON grid-файлы, Tinkoff Invest gRPC (Hour1-свечи, токен `T_BANK` в `env/token.env`).

## Global Constraints

- Таймфрейм калибровки: **Hour1**.
- Окно walk-forward: **`-months 31 -train-months 12 -test-months 6`** (≈3 OOS-фолда; история EUTR с IPO 2023-11-21).
- Метрика: **`profit_factor`**, **`-min-trades 20`**.
- Hour1-история EUTR уже закеширована: `data/candles/EUTR_Hour1.json` (2023-11-21 → 2026-06-19). `-refresh` не нужен.
- Stage-gate: опциональный фильтр включается в `DefaultParams` **только если улучшает pooled OOS PF** относительно текущей накопленной базы; иначе остаётся OFF.
- Package/var naming следует существующему: пакет `eutr`, импорт-алиас `reversioneutr`, константа `Ticker = "EUTR"`.
- Все walk-forward отчёты сохраняются под `reports/EUTR_<layer>/`.

---

### Task 1: Grid-файлы калибровки для EUTR

**Files:**
- Create: `data/params/eutr/reversion_cal_entry.json`
- Create: `data/params/eutr/reversion_cal_trend.json`
- Create: `data/params/eutr/reversion_cal_htf.json`
- Create: `data/params/eutr/reversion_cal_regime.json`
- Create: `data/params/eutr/reversion_cal_volume.json`
- Create: `data/params/eutr/reversion_cal_overbought.json`
- Create: `data/params/eutr/reversion_cal_breakeven.json`
- Create: `data/params/eutr/reversion_cal_trail.json`
- Create: `data/params/eutr/reversion_cal_atrstop.json`
- Create: `data/params/eutr/reversion_cal_catstop.json`

**Interfaces:**
- Consumes: ugld grid-файлы как источник сеток (`data/params/ugld/reversion_cal_*.json`).
- Produces: 10 grid-файлов EUTR, потребляемых задачей 3 через `-calibrate`.

- [ ] **Step 1: Скопировать 10 ugld cal-гридов в каталог eutr**

```bash
mkdir -p data/params/eutr
for f in entry trend htf regime volume overbought breakeven trail atrstop catstop; do
  cp data/params/ugld/reversion_cal_$f.json data/params/eutr/reversion_cal_$f.json
done
ls data/params/eutr/
```

Expected: 10 файлов `reversion_cal_*.json`.

- [ ] **Step 2: Переписать `_comment` в каждом файле под EUTR/Hour1/31 мес**

В каждом из 10 файлов заменить значение поля `_comment`: тикер `NVTK`/`UGLD` → `EUTR`, путь `data/params/nvtk/…` → `data/params/eutr/…`, `-interval Day1` → `-interval Hour1`, `-months 50 -test-months 12` → `-months 31 -train-months 12 -test-months 6`, `-out ./reports/NVTK_<layer>` → `-out ./reports/EUTR_<layer>`. **Сетки параметров (`phases`/`grid`) не трогать** — они тикеро-независимы.

Пример итогового `_comment` для `reversion_cal_entry.json`:

```
"_comment": "ENTRY-CORE calibration (the dual-oscillator buy signal). Two staged phases: tune RSI first, then the Stochastic over the RSI survivors. Everything else inherits EUTR DefaultParams. This is the heart of the strategy — run it even if the screen says all filters are dead. Run: go run ./cmd/backtest -ticker EUTR -strategy reversion -interval Hour1 -calibrate data/params/eutr/reversion_cal_entry.json -out ./reports/EUTR_entry -months 31 -train-months 12 -test-months 6 -metric profit_factor -min-trades 20",
```

- [ ] **Step 3: Проверить, что все 10 файлов — валидный JSON**

Run:
```bash
for f in data/params/eutr/reversion_cal_*.json; do python3 -c "import json,sys;json.load(open('$f'))" && echo "OK $f"; done
```
Expected: 10 строк `OK …`, без исключений.

- [ ] **Step 4: Commit**

```bash
git add data/params/eutr/
git commit -m "feat(reversion): staged calibration grids for EUTR ticker

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>"
```

---

### Task 2: Регистрация EUTR с baseline-params

Регистрация ДО калибровки (зеркалит ugld-коммит b2c269d): чтобы `-ticker EUTR` использовал defaults EUTR и гриды наследовали от них. На этом этапе params — нейтральный baseline (как у sfin: голое ядро + всегда-включённые выходы, опциональные фильтры OFF). Задача 3 будет перезаписывать эти поля победителями слоёв.

**Files:**
- Create: `internal/service/trading_strategy/reversion/strategy/eutr/eutr.go`
- Modify: `internal/service/backtest/reversion_registry.go` (импорт + строка map)

**Interfaces:**
- Consumes: `core.Params` (структура движка), `reversionBindingFor` из `reversion_registry.go`.
- Produces: `eutr.Ticker` (`= "EUTR"`), `eutr.DefaultParams() core.Params`. Потребляются регистром и задачей 3.

- [ ] **Step 1: Создать пакет eutr с baseline-params**

`internal/service/trading_strategy/reversion/strategy/eutr/eutr.go`:

```go
// Package eutr supplies the ticker and reversion Params for EUTR (ЕвроТранс / EuroTrans).
package eutr

import "tinvest/internal/service/trading_strategy/reversion/strategy/core"

// Ticker is the instrument this package configures.
const Ticker = "EUTR"

// DefaultParams returns EUTR's reversion parameters.
//
// 2026-06-19: BASELINE — calibration pending. EUTR topped the reversion-fitness
// screen (score 0.869, mean ATR% 3.79, autocorr -0.112, classified mean-reverting),
// so it is registered as a reversion candidate. Hour1 history starts 2023-11-21
// (IPO), giving ~31 months — shorter than the 36-month window used for the other
// tickers, so its walk-forward runs train 12 / OOS 6 over -months 31 (~3 folds).
//
// These are neutral baseline params (bare entry core + always-on RSIOS/EMAX exits;
// every optional gate OFF). They will be replaced by the staged walk-forward winners.
func DefaultParams() core.Params {
	return core.Params{
		// Ядро входа + всегда-включённый выход EMAX.
		UseTrend: 0, FastEMA: 10, SlowEMA: 200,
		RSIPeriod: 14, RSIOversold: 25,
		StochKPeriod: 14, StochDSmooth: 3, StochOversold: 15,
		// RSI50 momentum-fade выход — ВКЛ (всегда-включённый, гридами не перебирается).
		UseRSI50: 1,
		// Опциональные гейты входа — ВЫКЛ.
		HTFTrendEMA: 0,
		UseRegime:   0, ADXPeriod: 14, ADXMax: 30,
		UseVolume: 0, VolAvgPeriod: 30, VolMult: 1.8,
		// Опциональные выходы/стопы — ВЫКЛ.
		UseOverbought: 0, RSIOverbought: 70, StochOverbought: 80,
		UseBreakeven: 0, BreakevenArmATR: 0.5,
		UseTrail: 0, TrailATRMult: 1.5,
		UseATRStop: 0, ATRPeriod: 14, StopATRMult: 1.0,
		CatStopATRMult: 0,
	}
}
```

- [ ] **Step 2: Добавить импорт в reversion_registry.go**

В блок импортов (после `reversiond...` алфавитно — между `reversionafks` и `reversiongazp`) добавить:

```go
	reversioneutr "tinvest/internal/service/trading_strategy/reversion/strategy/eutr"
```

- [ ] **Step 3: Добавить строку в reversionRegistry map**

В `var reversionRegistry = map[string]Binding{ … }` добавить строку:

```go
	reversioneutr.Ticker:  reversionBindingFor(reversioneutr.Ticker, reversioneutr.DefaultParams),
```

- [ ] **Step 4: Сборка**

Run: `go build ./...`
Expected: без ошибок.

- [ ] **Step 5: Проверить, что EUTR резолвится как зарегистрированный (не generic)**

Run:
```bash
go run ./cmd/backtest -ticker EUTR -strategy reversion -interval Hour1 -months 6 -out ./reports/EUTR_check
```
Expected: строка `report: reports/EUTR_check/EUTR_reversion_Hour1_*.md (trades=…)` без ошибки инструмента. (Число сделок неважно.)

- [ ] **Step 6: Commit**

```bash
git add internal/service/trading_strategy/reversion/strategy/eutr/ internal/service/backtest/reversion_registry.go
git commit -m "feat(reversion): register EUTR ticker with baseline params

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>"
```

---

### Task 3: Стадийная walk-forward калибровка

Послойный прогон. После каждого слоя: посмотреть `reports/EUTR_<layer>/*_walkforward.md`, определить победителя по pooled OOS PF, и **если он бьёт текущую базу — зафиксировать его поля в `eutr.go` DefaultParams** (пересборка перед следующим слоем подхватит обновлённую базу через наследование). Если ни одно значение не бьёт базу — слой остаётся OFF, `eutr.go` не меняется.

**Files:**
- Modify: `internal/service/trading_strategy/reversion/strategy/eutr/eutr.go` (поля `DefaultParams` по мере фиксации победителей)
- Create (как побочный продукт): `reports/EUTR_<layer>/*_walkforward.md` для каждого слоя

**Interfaces:**
- Consumes: grid-файлы из задачи 1, `eutr.DefaultParams` из задачи 2.
- Produces: накопленную калиброванную `DefaultParams` (потребляется задачей 4 для финального комментария).

- [ ] **Step 1: Слой entry (ядро — гоняется всегда)**

Run:
```bash
go run ./cmd/backtest -ticker EUTR -strategy reversion -interval Hour1 \
  -calibrate data/params/eutr/reversion_cal_entry.json -out ./reports/EUTR_entry \
  -months 31 -train-months 12 -test-months 6 -metric profit_factor -min-trades 20
```
Прочитать `reports/EUTR_entry/*_walkforward.md`. Зафиксировать модальные/победившие `RSIPeriod, RSIOversold, StochKPeriod, StochDSmooth, StochOversold, FastEMA, SlowEMA` в `eutr.go` DefaultParams. Это новая база.

- [ ] **Step 2: Слой trend**

Run:
```bash
go run ./cmd/backtest -ticker EUTR -strategy reversion -interval Hour1 \
  -calibrate data/params/eutr/reversion_cal_trend.json -out ./reports/EUTR_trend \
  -months 31 -train-months 12 -test-months 6 -metric profit_factor -min-trades 20
```
Если `UseTrend=1` бьёт pooled OOS PF базы — зафиксировать `UseTrend: 1` в `eutr.go`; иначе оставить `UseTrend: 0`.

- [ ] **Step 3: Слой htf**

Run:
```bash
go run ./cmd/backtest -ticker EUTR -strategy reversion -interval Hour1 \
  -calibrate data/params/eutr/reversion_cal_htf.json -out ./reports/EUTR_htf \
  -months 31 -train-months 12 -test-months 6 -metric profit_factor -min-trades 20
```
Грид включает `HTFTrendEMA=0`. Зафиксировать победивший `HTFTrendEMA` (если 0 побеждает — гейт остаётся OFF). Слой использует `Hour4`-свечи (движок тянет их сам).

- [ ] **Step 4: Слой regime**

Run:
```bash
go run ./cmd/backtest -ticker EUTR -strategy reversion -interval Hour1 \
  -calibrate data/params/eutr/reversion_cal_regime.json -out ./reports/EUTR_regime \
  -months 31 -train-months 12 -test-months 6 -metric profit_factor -min-trades 20
```
Если `UseRegime=1` (с победившими `ADXPeriod, ADXMax`) бьёт базу — зафиксировать; иначе OFF.

- [ ] **Step 5: Слой volume**

Run:
```bash
go run ./cmd/backtest -ticker EUTR -strategy reversion -interval Hour1 \
  -calibrate data/params/eutr/reversion_cal_volume.json -out ./reports/EUTR_volume \
  -months 31 -train-months 12 -test-months 6 -metric profit_factor -min-trades 20
```
Если `UseVolume=1` (с `VolAvgPeriod, VolMult`) бьёт базу — зафиксировать; иначе OFF.

- [ ] **Step 6: Слой overbought**

Run:
```bash
go run ./cmd/backtest -ticker EUTR -strategy reversion -interval Hour1 \
  -calibrate data/params/eutr/reversion_cal_overbought.json -out ./reports/EUTR_overbought \
  -months 31 -train-months 12 -test-months 6 -metric profit_factor -min-trades 20
```
Если `UseOverbought=1` (с `RSIOverbought, StochOverbought`) бьёт базу — зафиксировать; иначе OFF.

- [ ] **Step 7: Слой breakeven**

Run:
```bash
go run ./cmd/backtest -ticker EUTR -strategy reversion -interval Hour1 \
  -calibrate data/params/eutr/reversion_cal_breakeven.json -out ./reports/EUTR_breakeven \
  -months 31 -train-months 12 -test-months 6 -metric profit_factor -min-trades 20
```
Если `UseBreakeven=1` (с `ATRPeriod, BreakevenArmATR`) бьёт базу — зафиксировать; иначе OFF.

- [ ] **Step 8: Слой trail**

Run:
```bash
go run ./cmd/backtest -ticker EUTR -strategy reversion -interval Hour1 \
  -calibrate data/params/eutr/reversion_cal_trail.json -out ./reports/EUTR_trail \
  -months 31 -train-months 12 -test-months 6 -metric profit_factor -min-trades 20
```
Если `UseTrail=1` (с `ATRPeriod, TrailATRMult`) бьёт базу — зафиксировать; иначе OFF.

- [ ] **Step 9: Слой atrstop**

Run:
```bash
go run ./cmd/backtest -ticker EUTR -strategy reversion -interval Hour1 \
  -calibrate data/params/eutr/reversion_cal_atrstop.json -out ./reports/EUTR_atrstop \
  -months 31 -train-months 12 -test-months 6 -metric profit_factor -min-trades 20
```
Если `UseATRStop=1` (с `ATRPeriod, StopATRMult`) бьёт базу — зафиксировать; иначе OFF.

- [ ] **Step 10: Слой catstop**

Run:
```bash
go run ./cmd/backtest -ticker EUTR -strategy reversion -interval Hour1 \
  -calibrate data/params/eutr/reversion_cal_catstop.json -out ./reports/EUTR_catstop \
  -months 31 -train-months 12 -test-months 6 -metric profit_factor -min-trades 20
```
Если `CatStopATRMult>0` бьёт базу — зафиксировать; иначе `CatStopATRMult: 0` (fixed-fraction sizing).

- [ ] **Step 11: Финальный прогон базы для pooled OOS PF**

Прогнать итоговую `DefaultParams` без `-calibrate`, чтобы получить итоговый walk-forward:
```bash
go run ./cmd/backtest -ticker EUTR -strategy reversion -interval Hour1 \
  -out ./reports/EUTR_final -months 31 -train-months 12 -test-months 6 -metric profit_factor
```
Зафиксировать pooled OOS PF, compounded%, число сделок, win%, max DD, и PF по каждому фолду — для комментария в задаче 4.

- [ ] **Step 12: Commit накопленной конфигурации**

```bash
git add internal/service/trading_strategy/reversion/strategy/eutr/eutr.go
git commit -m "feat(reversion): calibrate EUTR via staged walk-forward

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>"
```

---

### Task 4: Финализация комментария, развилка исхода, верификация

**Files:**
- Modify: `internal/service/trading_strategy/reversion/strategy/eutr/eutr.go` (финальный doc-комментарий)
- Modify: `/home/oleg/.claude/projects/-home-oleg-GolandProjects-tinvest/memory/project_reversion_strategy.md` (заметка о EUTR)

**Interfaces:**
- Consumes: итоговую `DefaultParams` и метрики walk-forward из задачи 3.
- Produces: финальный задокументированный `eutr.go`.

- [ ] **Step 1: Применить развилку исхода к doc-комментарию**

Переписать doc-комментарий `DefaultParams` по фактическому результату задачи 3, в стиле ugld.go/sfin.go:
- **Если pooled OOS PF > 1 и фолды устойчиво в плюсе** (профиль UGLD): описать дату, окно (Hour1, 31 мес, ~3 фолда), pooled OOS PF, compounded%, число сделок, win%, max DD; перечислить какие гейты ON/OFF и почему (со ссылкой на конкретный слой). Шапка `⚠️` НЕ ставится.
- **Если pooled OOS PF ≤ 1 или фолды нестабильны** (профиль SFIN): шапка `⚠️ <дата>: CALIBRATION FAILED — DO NOT TRADE THIS TICKER LIVE on the reversion strategy.`, затем числа фолдов и почему ни один фильтр не спасает; `DefaultParams` оставить голым baseline-ядром.

- [ ] **Step 2: Сборка и тесты**

Run: `go build ./... && go test ./internal/service/...`
Expected: build OK; тесты PASS (новой логики нет — проверяем, что регистрация ничего не сломала).

- [ ] **Step 3: Обновить память проекта**

В `~/.claude/projects/.../memory/project_reversion_strategy.md` дописать строку об EUTR: дата (2026-06-19), что добавлен per-ticker reversion, исход калибровки (рабочий / DO NOT TRADE) с pooled OOS PF, окно 31 мес/~3 фолда. Обновить пойнтер в `MEMORY.md` при необходимости.

- [ ] **Step 4: Commit**

```bash
git add internal/service/trading_strategy/reversion/strategy/eutr/eutr.go
git commit -m "docs(reversion): document EUTR calibration outcome

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>"
```

---

## Self-Review

**Spec coverage:**
- Per-ticker пакет + DefaultParams → Task 2 (baseline) + Task 3/4 (финал). ✓
- Регистрация в reversionRegistry → Task 2. ✓
- 10 grid-файлов с поправленным `_comment` → Task 1. ✓
- Стадийная калибровка в заданном порядке слоёв + stage-gate → Task 3. ✓
- Адаптация окна 31 мес / ~3 фолда → Global Constraints + закреплено в комментарии (Task 2 baseline, Task 4 финал). ✓
- Развилка UGLD-vs-SFIN → Task 4 Step 1. ✓
- Тестирование (build + go test + отчёты как свидетельство) → Task 2 Step 4-5, Task 4 Step 2, отчёты в Task 3. ✓
- Обновление памяти → Task 4 Step 3. ✓
- Вне объёма (core/live/новые гриды) — ни одна задача их не трогает. ✓

**Placeholder scan:** Все шаги с кодом содержат полный код/команды. Значения params в Task 3 намеренно определяются прогоном (это исследовательская калибровка), но процедура и правило фиксации заданы явно — это не плейсхолдер, а data-driven шаг.

**Type consistency:** `Ticker`, `DefaultParams`, `core.Params`, `reversionBindingFor`, алиас `reversioneutr` — согласованы с существующим `reversion_registry.go` и полями `core.Params` из ugld.go/sfin.go.
