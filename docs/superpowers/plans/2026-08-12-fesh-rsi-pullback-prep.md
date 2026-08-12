# FESH под rsi_pullback — план подготовки к калибровке

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Завести ДВМП (`FESH`) как калибруемый тикер `rsi_pullback`: пакет параметров в состоянии «калибровка не проводилась», каталог калибровочных сеток `data/params/rsi_pullback/fesh/` с осями, обоснованными замерами инструмента, и сторожевые тесты, которые эти оси удерживают.

**Architecture:** Только данные, один крошечный пакет и тесты. Пакет `strategy/fesh` повторяет форму `strategy/nvtk` (`return core.DefaultParams()`) и вносится в два реестра — бэктестовый и живой. Девять однотемных JSON-файлов повторяют структуру `data/params/rsi_pullback/reni/`; каждая ось прибита тестом, в тексте ошибки которого стоит замер, её обосновывающий. Прогоны калибровки в объём НЕ входят.

**Tech Stack:** Go 1.25, `go test`, `./bin/mage ci`, JSON-сетки, `cmd/backtest`.

## Global Constraints

- Спека: `docs/superpowers/specs/2026-08-12-fesh-rsi-pullback-prep-design.md`. Все числа осей и все замеры берутся оттуда — не пересчитывать и не «уточнять» по ходу.
- Каждый `cal_*.json` обязан нести `_comment`, содержащий подстроку `fesh/<имя файла>` — это проверяет `TestRSIPullbackCalFilesValid` (`internal/service/backtest/rsi_pullback_grid_test.go:89`). Команда запуска внутри `_comment` должна называть тот же самый файл.
- `StopDailyATR = 0` не появляется ни в одном файле — калибровка не имеет права отключить стоп.
- Каждая фаза непустая; все свипуемые поля обязаны резолвиться через `applyField`.
- В `cal_risk.json` хотя бы одна цель обязана превышать самый широкий стоп того же файла (`TestRSIPullbackGridControlPoints`): при стопах до 1.3 это обеспечивают цели 1.5, 2.0, 2.5.
- Тексты `_comment` и сообщений об ошибках в тестах — на русском, как в `data/params/rsi_pullback/reni/` и `rsi_pullback_reni_grid_test.go`.
- Хелперы `rsiPullbackTickerGrid` и `sameSet` уже существуют в пакете `backtest` (`rsi_pullback_grid_test.go:40`, `rsi_pullback_reni_grid_test.go:17`) — переиспользовать, не объявлять заново (пакет один, повторное объявление не скомпилируется).
- FESH НЕ вносится в `RSI_PULLBACK_TICKERS` и не получает литерал параметров. Это работа после walk-forward.
- Ни один шаг не запускает калибровочные прогоны. Единственный прогон во всём плане — smoke-запуск в Task 4.

## File Structure

**Создаются:**

| Файл | Ответственность |
|---|---|
| `internal/service/trading_strategy/rsi_pullback/strategy/fesh/fesh.go` | Тикер `FESH` + `DefaultParams()` в состоянии baseline-tracking; шапка несёт замеры инструмента |
| `data/params/rsi_pullback/fesh/cal_screen.json` | Цена двух опциональных гейтов в сделках (4 прогона) |
| `data/params/rsi_pullback/fesh/cal_entry.json` | Форма отката: `RSIPeriod` × `RSILower` (16) |
| `data/params/rsi_pullback/fesh/cal_trend.json` | Трендовый фильтр: `EMAFast` × `EMASlow` (16) |
| `data/params/rsi_pullback/fesh/cal_day.json` | Обе ветки дневного гейта совместно (12) |
| `data/params/rsi_pullback/fesh/cal_day_spent.json` | Только ветка «день исчерпан» (6) |
| `data/params/rsi_pullback/fesh/cal_volume.json` | Объёмный гейт (8) |
| `data/params/rsi_pullback/fesh/cal_risk.json` | Стоп и цель в долях дневного ATR (25) |
| `data/params/rsi_pullback/fesh/cal_exit.json` | Полоса выхода по RSI (6) |
| `data/params/rsi_pullback/fesh/cal_trail.json` | Форма трейла и его конкуренция с RSI-выходом (12) |
| `internal/service/backtest/rsi_pullback_fesh_grid_test.go` | Сторожевые тесты осей FESH |

**Модифицируются:**

| Файл | Что меняется |
|---|---|
| `internal/service/backtest/rsi_pullback_registry.go:8-13,40-47` | Импорт `fesh` + запись в `rsiPullbackRegistry` |
| `internal/service/trading_strategy/rsi_pullback/live/registry.go:1-24` | Импорт `fesh`, запись в `paramsByTicker`, актуализация комментария |
| `internal/service/backtest/rsi_pullback_registry_test.go` | Новый `TestRSIPullbackFESHTracksBaseline` |

---

### Task 1: Пакет `strategy/fesh` и обе записи в реестрах

**Files:**
- Create: `internal/service/trading_strategy/rsi_pullback/strategy/fesh/fesh.go`
- Modify: `internal/service/backtest/rsi_pullback_registry.go`
- Modify: `internal/service/trading_strategy/rsi_pullback/live/registry.go`
- Test: `internal/service/backtest/rsi_pullback_registry_test.go`

**Interfaces:**
- Consumes: `core.DefaultParams() core.Params`, `core.Params` (пакет `tinvest/internal/service/trading_strategy/rsi_pullback/strategy/core`); `rsiPullbackBindingFor(ticker string, defaults func() core.Params) Binding`; `RSIPullbackLookupOrGeneric(ticker string) Binding`.
- Produces: `fesh.Ticker` (константа `"FESH"`), `fesh.DefaultParams() core.Params`. Task 4 полагается на то, что `RSIPullbackLookupOrGeneric("FESH")` резолвится через реестр, а не через generic-ветку.

- [ ] **Step 1: Написать падающий тест**

В конец `internal/service/backtest/rsi_pullback_registry_test.go` добавить (импорт `"tinvest/internal/service/trading_strategy/rsi_pullback/strategy/fesh"` — в блок импортов рядом с `reni`):

```go
// TestRSIPullbackFESHTracksBaseline пинует состояние «калибровка не проводилась»: FESH обязан
// быть в карте (а не проваливаться в generic-ветку) И обязан возвращать РОВНО baseline. Оба
// факта снаружи неотличимы от откалиброванного соседа, и оба могут молча сломаться:
// пропавшая запись в карте даёт generic-биндинг с теми же значениями, а «улучшение» литерала
// без прогона даёт параметры, которые никогда не проверялись на этом инструменте.
// Когда калибровка FESH пройдёт и литерал появится, этот тест заменяется снимком литерала —
// по образцу TestRSIPullbackRENIIsRegisteredAndCalibrated.
func TestRSIPullbackFESHTracksBaseline(t *testing.T) {
	b, ok := rsiPullbackRegistry["FESH"]
	if !ok {
		t.Fatal("FESH отсутствует в rsiPullbackRegistry: тикер провалится в generic-ветку")
	}
	raw := b.DefaultParams()
	p, ok := raw.(core.Params)
	if !ok {
		t.Fatalf("FESH: DefaultParams() вернул %T, want core.Params", raw)
	}
	if p != core.DefaultParams() {
		t.Fatalf("FESH params = %+v, want baseline %+v — калибровка не проводилась, литерала быть не должно", p, core.DefaultParams())
	}
	if want := fesh.DefaultParams(); p != want {
		t.Fatalf("FESH params = %+v, want литерал пакета %+v", p, want)
	}
	if got := b.Build(p).Ticker(); got != "FESH" {
		t.Fatalf("Ticker() = %q, want FESH", got)
	}
}
```

- [ ] **Step 2: Запустить тест и убедиться, что он падает**

Run: `go test ./internal/service/backtest/ -run TestRSIPullbackFESHTracksBaseline -v`
Expected: FAIL — компиляция не проходит, `package .../strategy/fesh is not in std` (пакета ещё нет).

- [ ] **Step 3: Создать пакет `strategy/fesh`**

Создать `internal/service/trading_strategy/rsi_pullback/strategy/fesh/fesh.go`:

```go
// Package fesh supplies the ticker and starting rsi_pullback Params for FESH (ДВМП).
//
// Calibration has NOT been run for this ticker: the body returns core.DefaultParams()
// unchanged rather than copying its fields, so a change to the baseline still reaches this
// ticker instead of silently drifting away from it. Once -calibrate picks a winning
// combination for FESH, replace the body with an explicit literal — from that point the
// ticker must stop tracking the baseline.
//
// Что известно об инструменте (замеры 2026-08-12, спека
// docs/superpowers/specs/2026-08-12-fesh-rsi-pullback-prep-design.md):
//
//   - Истории достаточно: 35 062 30-минутных бара (25 321 будний) за 36.0 месяца с
//     2023-08-04. Штатный протокол walk-forward §8 docs/rsi_pullback/strategy.md исполним
//     целиком — как у reni и в отличие от domrf.
//   - Режимы разложены ЗЕРКАЛЬНО протоколу: обучающее окно 2023-08-04—2026-02-03 это падение
//     112.40 → 55.06 (−51.0%), а holdout 2026-02-04—2026-08-04 — рост 55.00 → 62.23 (+13.1%).
//     Для long-only покупки отката это значит, что in-sample PF занижен режимом, а holdout PF
//     завышен механически: подтверждением edge holdout здесь служить не может, его даёт только
//     rolling walk-forward, чьи фолды покрывают оба режима.
//   - Дневной ATR(14) идёт медианой 4.42% цены — самый широкий инструмент из заведённых
//     (ugld 4.28%, reni 3.36%, domrf 1.94%). Поэтому сетки в data/params/rsi_pullback/fesh/
//     пересчитаны от ugld/, а сужения, сделанные для domrf, в них намеренно НЕ перенесены.
//   - Круг издержек 0.023 дневного ATR — самый дешёвый из четырёх тикеров; он лицензирует
//     узкую строку стопа 0.3 ATR. Но тот же стоп переживает целиком лишь 1.0% дней, то есть
//     сидит внутри обычного внутридневного шума.
//   - Оборот 287 млн ₽/день при лоте 10 — втрое выше reni. На размер позиции не давит.
//
// Пакет заведён 2026-08-12 ДО калибровки — как место, куда ляжет литерал, и как носитель
// замеров выше. Попадание такого тикера в боевую вселенную ловит
// TestBaselineTrackingTickersStayOutOfTheDefaultUniverse в live/registry_test.go.
package fesh

import "tinvest/internal/service/trading_strategy/rsi_pullback/strategy/core"

// Ticker is the instrument this package configures.
const Ticker = "FESH"

// DefaultParams returns FESH's starting rsi_pullback parameters (pre-calibration).
func DefaultParams() core.Params {
	return core.DefaultParams()
}
```

- [ ] **Step 4: Внести FESH в бэктестовый реестр**

В `internal/service/backtest/rsi_pullback_registry.go` добавить импорт в общий блок (алфавитный порядок псевдонимов сохраняется — `fesh` идёт перед `gazp`):

```go
	rsipullbackfesh "tinvest/internal/service/trading_strategy/rsi_pullback/strategy/fesh"
```

и строку в `rsiPullbackRegistry` (перед `rsipullbackgazp`):

```go
	rsipullbackfesh.Ticker:  rsiPullbackBindingFor(rsipullbackfesh.Ticker, rsipullbackfesh.DefaultParams),
```

- [ ] **Step 5: Внести FESH в живой реестр и актуализировать его комментарий**

В `internal/service/trading_strategy/rsi_pullback/live/registry.go` добавить импорт `"tinvest/internal/service/trading_strategy/rsi_pullback/strategy/fesh"` (перед `gazp`), заменить комментарий над `paramsByTicker` и добавить запись в карту. Комментарий сейчас называет тикерами без литерала «NVTK и RENI», но RENI откалиброван 2026-08-12 (коммит `6eb8b51`) — перечень надо привести в актуальное состояние:

```go
// paramsByTicker maps every rsi_pullback ticker the runner knows to its params. The
// configured universe (RSI_PULLBACK_TICKERS) selects which of these actually trade;
// NVTK and FESH are registered for completeness but have no calibrated literal yet — both
// return the baseline — and must not be put into the universe.
var paramsByTicker = map[string]core.Params{
	ugld.Ticker:  ugld.DefaultParams(),
	tbank.Ticker: tbank.DefaultParams(),
	gazp.Ticker:  gazp.DefaultParams(),
	nvtk.Ticker:  nvtk.DefaultParams(),
	domrf.Ticker: domrf.DefaultParams(),
	reni.Ticker:  reni.DefaultParams(),
	fesh.Ticker:  fesh.DefaultParams(),
}
```

- [ ] **Step 6: Запустить тесты и убедиться, что они проходят**

Run: `go test ./internal/service/backtest/ ./internal/service/trading_strategy/rsi_pullback/... -run 'FESH|Baseline|Registered|Universe|RSIExit' -v`
Expected: PASS, в том числе `TestRSIPullbackFESHTracksBaseline`, `TestBaselineTrackingTickersStayOutOfTheDefaultUniverse` (FESH не в дефолтной вселенной), `TestRegisteredTickersKeepTheRSIExitArmed` (baseline несёт `UseRSIExit=1`).

- [ ] **Step 7: Коммит**

```bash
git add internal/service/trading_strategy/rsi_pullback/strategy/fesh/fesh.go \
        internal/service/backtest/rsi_pullback_registry.go \
        internal/service/backtest/rsi_pullback_registry_test.go \
        internal/service/trading_strategy/rsi_pullback/live/registry.go
git commit -m "feat(rsi_pullback): пакет strategy/fesh в состоянии «калибровка не проводилась»"
```

---

### Task 2: Сигнальные сетки FESH (screen, entry, trend)

**Files:**
- Create: `internal/service/backtest/rsi_pullback_fesh_grid_test.go`
- Create: `data/params/rsi_pullback/fesh/cal_screen.json`
- Create: `data/params/rsi_pullback/fesh/cal_entry.json`
- Create: `data/params/rsi_pullback/fesh/cal_trend.json`

**Interfaces:**
- Consumes: `rsiPullbackTickerGrid(t *testing.T, ticker, file string) map[string][]float64` (`rsi_pullback_grid_test.go:40`), `sameSet(got []float64, want ...float64) bool` (`rsi_pullback_reni_grid_test.go:17`).
- Produces: `feshGrid(t *testing.T, file string) map[string][]float64` — хелпер, которым пользуется Task 3.

- [ ] **Step 1: Написать падающий тест сигнальных осей**

Создать `internal/service/backtest/rsi_pullback_fesh_grid_test.go`:

```go
package backtest

import (
	"math"
	"testing"
)

// feshGrid читает файл сеток FESH через общий хелпер.
func feshGrid(t *testing.T, file string) map[string][]float64 {
	t.Helper()
	return rsiPullbackTickerGrid(t, "fesh", file)
}

// TestFESHSignalGridsPinTheirMeasuredAxes сторожит оси, обоснованные замерами инструмента, а не
// вкусом. Каталог fesh/ заводится копированием структуры reni/, и типовая ошибка такой копии —
// притащить вместе с формой чужие обоснования. FESH шире всех заведённых тикеров (дневной ATR
// 4.42% против 3.36% у RENI и 1.94% у DOMRF), и опасны обе стороны: и перенос сужений DOMRF,
// сделанных при дефиците сигналов, и перенос оговорок RENI про мёртвые углы — здесь их нет,
// слабейший угол RSI(7)@10 даёт 49 будних кроссов против 23 у RENI.
func TestFESHSignalGridsPinTheirMeasuredAxes(t *testing.T) {
	screen := feshGrid(t, "cal_screen.json")
	for _, field := range []string{"UseDayATRGate", "UseVolume"} {
		if got := screen[field]; !sameSet(got, 0, 1) {
			t.Errorf("cal_screen.json: %s = %v, want ровно {0,1} — тема меряет цену каждого гейта в сделках, а любая другая пара не даёт сравнить гейт с его отсутствием", field, got)
		}
	}

	entry := feshGrid(t, "cal_entry.json")
	// Глубже 25 порог перестаёт отбирать откат: RSI(4) уходит под 30 1986 раз по будням за
	// 36 месяцев — это обычный шум, а не сетап.
	for _, v := range entry["RSILower"] {
		if v > 25 {
			t.Errorf("cal_entry.json свипует RSILower=%v: выше 25 порог перестаёт отбирать откат (1986 будних кроссов под 30)", v)
		}
	}
	// Ниже 10 выборка истончается быстрее, чем растёт качество сигнала: у RSI(7) на уровне 10
	// остаётся 49 будних кроссов за всю историю, и более глубокий порог режет и их.
	for _, v := range entry["RSILower"] {
		if v < 10 {
			t.Errorf("cal_entry.json свипует RSILower=%v: ниже 10 сигналов почти не остаётся (у RSI(7)@10 их 49)", v)
		}
	}
	// Уровень 10 обязан остаться: скринер выбрал для FESH лучшей конфигурацией RSI 6/10, и
	// 81 будний кросс RSI(6)@10 эту точку выдерживает. На DOMRF таких кроссов было 18, и там
	// уровень 10 вырезали — при копировании сеток это сужение легко притащить по ошибке.
	hasDeep := false
	for _, v := range entry["RSILower"] {
		if v == 10 {
			hasDeep = true
		}
	}
	if !hasDeep {
		t.Errorf("cal_entry.json: RSILower = %v, не содержит 10 — основную гипотезу скринера", entry["RSILower"])
	}
	// RSIPeriod короче 4 покупает шум, а не откат: RSI(3) реагирует на любое дрожание цены.
	for _, v := range entry["RSIPeriod"] {
		if v < 4 {
			t.Errorf("cal_entry.json свипует RSIPeriod=%v: короче 4 это шум, а не откат", v)
		}
	}
	// RSIUpper здесь не свипуется: 4x4x6 = 96 комбинаций на одной теме — переобучение по
	// построению. Полоса выхода меряется отдельно, файлом cal_exit.json.
	if got := entry["RSIUpper"]; len(got) != 0 {
		t.Errorf("cal_entry.json свипует RSIUpper=%v: полоса выхода принадлежит cal_exit.json", got)
	}

	trend := feshGrid(t, "cal_trend.json")
	// EMASlow=200 исполнима: Lookback при ней равен 2*200+20 = 420 баров против 25321 будних
	// в кэше, то есть окно прогрева занимает 1.7% истории.
	for _, v := range trend["EMASlow"] {
		if v > 200 {
			t.Errorf("cal_trend.json свипует EMASlow=%v: окно прогрева растёт быстрее, чем польза от сглаживания", v)
		}
	}
	// Быстрая EMA обязана быть быстрее самой быстрой из медленных, иначе строка сетки означает
	// фильтр, который никогда не пропускает вход. Порог берётся из фактического минимума оси
	// EMASlow, а не зашивается константой: понижение оси EMASlow иначе тихо перестало бы
	// совпадать с тем, что здесь проверяется.
	minSlow := math.Inf(1)
	for _, v := range trend["EMASlow"] {
		if v < minSlow {
			minSlow = v
		}
	}
	if !math.IsInf(minSlow, 1) {
		for _, v := range trend["EMAFast"] {
			if v >= minSlow {
				t.Errorf("cal_trend.json свипует EMAFast=%v: минимум оси EMASlow сейчас %.0f, такая пара мертва", v, minSlow)
			}
		}
	}
}
```

- [ ] **Step 2: Запустить тест и убедиться, что он падает**

Run: `go test ./internal/service/backtest/ -run TestFESHSignalGridsPinTheirMeasuredAxes -v`
Expected: FAIL с `read ../../../data/params/rsi_pullback/fesh/cal_screen.json: no such file or directory` — каталога сеток ещё нет.

- [ ] **Step 3: Создать `cal_screen.json`**

Создать `data/params/rsi_pullback/fesh/cal_screen.json`:

```json
{
  "_comment": "SCREEN: цена двух опциональных гейтов в сделках, 4 прогона. Тема отвечает на один вопрос — сколько сделок остаётся, когда включён дневной гейт и объёмный гейт, — и запускается ПЕРВОЙ, до всех остальных тем. Порог -min-trades 1 обязателен именно здесь: при штатных 20 отфильтруются ровно те строки, которые тема измеряет, и лидерборд окажется пустым. Читать надо колонку сделок, а не profit factor: если при обоих включённых гейтах остаётся меньше 15 сделок за 36 месяцев, дальнейшие темы будут калибровать отдельные сделки, а не конфигурации. Замеры FESH, задающие ожидания: будних 30-минутных баров 25321 за 36.0 месяца, кроссов RSI(6) вниз через 15 — 276, дневной гейт при SpentDayATR=1.0 пропускает 21.3% баров, объёмный при VolMult=1.5 — 24.2% баров. Отдельно про holdout: обучающее окно FESH это падение −51.0%, а последние 6 месяцев — рост +13.1%, поэтому число сделок на holdout будет систематически выше при тех же гейтах, и сравнивать надо не с train, а между собой. Запуск: go run ./cmd/backtest -ticker FESH -strategy rsi_pullback -interval Minutes30 -calibrate data/params/rsi_pullback/fesh/cal_screen.json -out ./reports/FESH_screen -months 36 -min-trades 1 -metric profit_factor.",
  "phases": [
    {
      "name": "screen",
      "grid": {
        "UseDayATRGate": [0, 1],
        "UseVolume": [0, 1]
      }
    }
  ]
}
```

- [ ] **Step 4: Создать `cal_entry.json`**

Создать `data/params/rsi_pullback/fesh/cal_entry.json`:

```json
{
  "_comment": "ENTRY, форма отката: RSIPeriod x RSILower, 16 прогонов. Ось совпадает с reni/cal_entry.json по форме, но обоснована собственным замером: кроссы RSI вниз через уровень, будние бары за 36.0 месяца — RSI(4) 328 через 10, 700 через 15, 1082 через 20, 1510 через 25; RSI(5) 173/412/754/1121; RSI(6) 81/276/553/862; RSI(7) 49/189/394/695. Живы ВСЕ шестнадцать углов, включая слабейший RSI(7)@10 с 49 кроссами — у RENI тот же угол давал 23 и был помечен как заведомо мёртвый, эту оговорку сюда переносить не нужно. Уровень 10 оставлен: скринер выбрал для FESH лучшей конфигурацией именно RSI 6/10 (PFmed 2.03 на TradesMed 52, Plateau 83%), и 81 будний кросс RSI(6)@10 эту точку выдерживает. Уровень 30 не берётся: RSI(4) уходит под него 1986 раз по будням, это шум, а не откат. RSIUpper здесь НЕ свипуется: полоса выхода меряется отдельно, файлом cal_exit.json, иначе тема раздувается до 96 комбинаций. Читая лидерборд, помнить про режимы: обучающее окно 2023-08-04—2026-02-03 это падение 112.40 -> 55.06 (−51.0%), поэтому низкий in-sample profit factor здесь не то же самое, что низкий PF на аптрендовом тикере. Запуск: go run ./cmd/backtest -ticker FESH -strategy rsi_pullback -interval Minutes30 -calibrate data/params/rsi_pullback/fesh/cal_entry.json -out ./reports/FESH_entry -months 36 -min-trades 20 -test-months 6 -metric profit_factor.",
  "phases": [
    {
      "name": "entry",
      "grid": {
        "RSIPeriod": [4, 5, 6, 7],
        "RSILower": [10, 15, 20, 25]
      }
    }
  ]
}
```

- [ ] **Step 5: Создать `cal_trend.json`**

Создать `data/params/rsi_pullback/fesh/cal_trend.json`:

```json
{
  "_comment": "TREND: EMAFast x EMASlow, 16 прогонов. Ось совпадает с ugld/cal_trend.json и reni/cal_trend.json, и это единственная тема, где полная копия оправдана: периоды EMA задаются в барах, а не в единицах цены, поэтому разница в ширине инструментов (ATR 4.42% у FESH против 3.36% у RENI) на них не влияет. Замер, подтверждающий, что ось не вырождена: доля 30-минутных баров с EMAFast > EMASlow составляет 46-47% для ВСЕХ шестнадцати пар — ни одна пара не открывает и не закрывает вход постоянно. Проверено, что EMASlow=200 исполнима: Lookback при ней равен 2*200+20 = 420 баров против 25321 будних в кэше, то есть окно прогрева занимает 1.7% истории. Скринер на своей фиксированной сетке выбрал для FESH пару EMA 20/150 — она и есть основная гипотеза темы, остальные строки её проверяют. Эта тема на FESH важнее, чем на других тикерах: обучающее окно содержит падение 129.39 -> 40.11 (−69.0% от пика 2023-08-02 до минимума 2024-12-13), и трендовый фильтр обязан выключать вход именно там. Если 20/200 и 20/50 дают близкий profit factor, гейт на этих данных не работает вовсе, и это надо знать до настройки риска. Запуск: go run ./cmd/backtest -ticker FESH -strategy rsi_pullback -interval Minutes30 -calibrate data/params/rsi_pullback/fesh/cal_trend.json -out ./reports/FESH_trend -months 36 -min-trades 20 -test-months 6 -metric profit_factor.",
  "phases": [
    {
      "name": "trend",
      "grid": {
        "EMAFast": [5, 10, 20, 30],
        "EMASlow": [50, 100, 150, 200]
      }
    }
  ]
}
```

- [ ] **Step 6: Запустить тесты и убедиться, что они проходят**

Run: `go test ./internal/service/backtest/ -run 'TestFESHSignalGridsPinTheirMeasuredAxes|TestRSIPullbackCalFilesValid' -v`
Expected: PASS. `TestRSIPullbackCalFilesValid` должен показать подтесты `fesh/cal_screen.json`, `fesh/cal_entry.json`, `fesh/cal_trend.json`.

- [ ] **Step 7: Коммит**

```bash
git add internal/service/backtest/rsi_pullback_fesh_grid_test.go data/params/rsi_pullback/fesh/
git commit -m "feat(rsi_pullback): сигнальные сетки FESH с замеренными осями"
```

---

### Task 3: Сетки риска и гейтов FESH (day, day_spent, volume, risk, exit, trail)

**Files:**
- Modify: `internal/service/backtest/rsi_pullback_fesh_grid_test.go` (добавить вторую тестовую функцию)
- Create: `data/params/rsi_pullback/fesh/cal_day.json`
- Create: `data/params/rsi_pullback/fesh/cal_day_spent.json`
- Create: `data/params/rsi_pullback/fesh/cal_volume.json`
- Create: `data/params/rsi_pullback/fesh/cal_risk.json`
- Create: `data/params/rsi_pullback/fesh/cal_exit.json`
- Create: `data/params/rsi_pullback/fesh/cal_trail.json`

**Interfaces:**
- Consumes: `feshGrid(t *testing.T, file string) map[string][]float64` и `sameSet(got []float64, want ...float64) bool` из Task 2.
- Produces: полный каталог из девяти файлов — Task 4 запускает по нему smoke.

- [ ] **Step 1: Написать падающий тест риск-осей**

Дописать в конец `internal/service/backtest/rsi_pullback_fesh_grid_test.go`:

```go
// TestFESHRiskGridsPinTheirMeasuredAxes сторожит оси стопа, цели и обоих гейтов. Каждое число
// ниже опирается на замер по кэшу FESH, а не на перенос с соседнего тикера: дневной ATR 4.42%,
// круг издержек 0.023 ATR, медианный дневной размах 0.85 ATR.
func TestFESHRiskGridsPinTheirMeasuredAxes(t *testing.T) {
	day := feshGrid(t, "cal_day.json")
	if got := day["UseDayATRGate"]; len(got) != 1 || got[0] != 1 {
		t.Errorf("cal_day.json: UseDayATRGate = %v, want ровно [1] — цена отключения гейта меряется в cal_screen.json", got)
	}
	// Медианный день проходит 0.27 ATR уже ко второму бару, 0.32 к третьему. Пороги 0.1-0.2 из
	// ugld/ оставляют ветке «день только начался» 2.9-6.3% будних баров: она почти мертва.
	for _, v := range day["FreshDayATR"] {
		if v > 0 && v < 0.25 {
			t.Errorf("cal_day.json свипует FreshDayATR=%v: ко второму бару медианный день прошёл 0.27 ATR, ветке остаётся меньше 7%% баров", v)
		}
	}
	// Порог 0.6 проходят 54.3% будних баров — на этом инструменте он перестаёт быть гейтом.
	for _, v := range day["SpentDayATR"] {
		if v < 0.8 {
			t.Errorf("cal_day.json свипует SpentDayATR=%v: порог достижим для 54%% баров, это не гейт", v)
		}
	}
	// Соотношение двух веток гейта: положительный максимум FreshDayATR обязан быть строго
	// меньше минимума SpentDayATR. dayStateOK пропускает бар, когда день ещё не раскрылся
	// (used <= fresh*ATR) ИЛИ когда он уже исчерпан (used >= spent*ATR); если верх ветки
	// «свежий» дотягивается до низа ветки «исчерпан», обе полосы дают true почти на каждом
	// баре, и UseDayATRGate=1 в лидерборде продолжит формально числиться включённым, хотя
	// фактически не отсекает ничего.
	maxFresh := 0.0
	for _, v := range day["FreshDayATR"] {
		if v > maxFresh {
			maxFresh = v
		}
	}
	minSpent := math.Inf(1)
	for _, v := range day["SpentDayATR"] {
		if v < minSpent {
			minSpent = v
		}
	}
	if maxFresh > 0 && !math.IsInf(minSpent, 1) && maxFresh >= minSpent {
		t.Errorf("cal_day.json: max(FreshDayATR)=%.2f >= min(SpentDayATR)=%.2f — ветки «день начался» и «день исчерпан» перекрываются, dayStateOK почти всегда true, и гейт перестаёт что-либо отсекать несмотря на UseDayATRGate=1", maxFresh, minSpent)
	}
	// RSILower в этой фазе не свипуется: у ugld/ он раздувает тему до 60 прогонов, а глубина
	// отката принадлежит cal_entry.json. Тема обязана остаться однотемной.
	if got := day["RSILower"]; len(got) != 0 {
		t.Errorf("cal_day.json свипует RSILower=%v: глубина отката принадлежит cal_entry.json", got)
	}

	spent := feshGrid(t, "cal_day_spent.json")
	if got := spent["FreshDayATR"]; len(got) != 1 || got[0] != 0 {
		t.Errorf("cal_day_spent.json: FreshDayATR = %v, want ровно [0] — ветка «день начался» выключена целиком", got)
	}
	// Левый край 0.6 стоит контрольной строкой «гейт почти выключен» (54.3% баров). Точки
	// 0.4-0.5 из ugld/ на FESH не гейтят вовсе и заняли бы прогоны впустую.
	for _, v := range spent["SpentDayATR"] {
		if v < 0.6 {
			t.Errorf("cal_day_spent.json свипует SpentDayATR=%v: ниже 0.6 порог не гейтит (0.6 проходят 54%% баров)", v)
		}
	}

	vol := feshGrid(t, "cal_volume.json")
	if got := vol["UseVolume"]; len(got) != 1 || got[0] != 1 {
		t.Errorf("cal_volume.json: UseVolume = %v, want ровно [1] — точка «гейт выключен» принадлежит cal_screen.json", got)
	}
	// Все четыре точки живые: при базе 5 дней гейт проходят 29.9% баров при 1.2, 24.2% при 1.5,
	// 17.9% при 2.0, 14.0% при 2.5. Выше 2.5 остаётся меньше седьмой части баров, и объёмный
	// гейт начинает резать выборку сильнее, чем несёт информации.
	for _, v := range vol["VolMult"] {
		if v > 2.5 {
			t.Errorf("cal_volume.json свипует VolMult=%v: выше 2.5 гейт проходит меньше 14%% баров", v)
		}
	}
	// Только две базы, 5 и 10: короткая быстрее реагирует на смену активности, длинная
	// устойчивее к одиночному всплеску. База 3 из ugld/ ловит один выброс объёма, база 14 —
	// размывает; на вторичном гейте лишние степени свободы не окупаются.
	for _, v := range vol["VolBaseDays"] {
		if v != 5 && v != 10 {
			t.Errorf("cal_volume.json свипует VolBaseDays=%v: обоснованы только 5 и 10 (3 ловит одиночный всплеск, 14 размывает)", v)
		}
	}

	risk := feshGrid(t, "cal_risk.json")
	// Круг издержек стоит 0.023 дневного ATR — самый дешёвый из четырёх заведённых тикеров: на
	// стопе 0.3 ATR (= 1.33% цены) комиссия съедает 8% риска. На DOMRF та же строка стоила 17%
	// и была оттуда вырезана — при копировании сеток это сужение легко притащить по ошибке,
	// поэтому присутствие строки проверяется явно.
	hasTightStop := false
	for _, v := range risk["StopDailyATR"] {
		if v == 0.3 {
			hasTightStop = true
		}
	}
	if !hasTightStop {
		t.Errorf("cal_risk.json: StopDailyATR = %v, не содержит 0.3 (издержки 0.023 ATR за круг эту строку лицензируют)", risk["StopDailyATR"])
	}
	// Нижняя граница оси: тот же круг издержек, который лицензирует строку 0.3 (8% риска),
	// запрещает идти уже. На 0.15 доля выросла бы до 15%, на 0.1 — до 23%: это уже та черта,
	// по которой DOMRF отверг свою строку 0.3 при 17%. «Попробуем стоп потуже» не должно
	// суметь добавить такую строку молча.
	for _, v := range risk["StopDailyATR"] {
		if v > 0 && v < 0.3 {
			t.Errorf("cal_risk.json свипует StopDailyATR=%v: издержки съели бы %.0f%% риска (круг 0.023 ATR) против 8%% на разрешённой строке 0.3; для сравнения, DOMRF отверг свою строку 0.3 при 17%%", v, 0.023/v*100)
		}
	}
	// Верх оси 1.3: медианный день покрывает 0.85 ATR, такой стоп переживает целиком 79.7%
	// дней. Шире — это уже не стоп, а отсутствие стопа, оплаченное размером позиции.
	for _, v := range risk["StopDailyATR"] {
		if v > 1.3 {
			t.Errorf("cal_risk.json свипует StopDailyATR=%v: шире 1.3 ATR стоп перестаёт быть стопом (медианный день 0.85 ATR)", v)
		}
	}

	exit := feshGrid(t, "cal_exit.json")
	// Это единственное место, где меряется полоса выхода: cal_entry.json её намеренно не свипует.
	if got := exit["RSIUpper"]; !sameSet(got, 55, 60, 65, 70, 75, 80) {
		t.Errorf("cal_exit.json: RSIUpper = %v, want ровно {55,60,65,70,75,80} — cal_entry.json полосу выхода не свипует, а любая точка вне шкалы RSI или пропуск внутри неё сужает единственное место, где эта полоса измеряется", got)
	}

	trail := feshGrid(t, "cal_trail.json")
	if got := trail["UseTrail"]; len(got) != 1 || got[0] != 1 {
		t.Errorf("cal_trail.json: UseTrail = %v, want ровно [1] — тема меряет форму трейла, а не факт включения", got)
	}
	if got := trail["UseRSIExit"]; !sameSet(got, 0, 1) {
		t.Errorf("cal_trail.json: UseRSIExit = %v, want ровно {0,1} — трейл и RSI-выход конкурируют за одну сделку, и посторонняя точка не даёт замерить оба режима", got)
	}
	// Правый край 0.8, а не 0.6 как у ugld/: там потолок задавала цель TPDailyATR=0.6, выше
	// которой трейл не успевал взвестись; здесь ось цели поднята до 2.5, и трейлу нужно
	// пространство для по-настоящему позднего срабатывания.
	hasFarTrail := false
	for _, v := range trail["TrailDailyATR"] {
		if v == 0.8 {
			hasFarTrail = true
		}
	}
	if !hasFarTrail {
		t.Errorf("cal_trail.json: TrailDailyATR = %v, не содержит правый край 0.8 (цель поднята до 2.5)", trail["TrailDailyATR"])
	}
}
```

- [ ] **Step 2: Запустить тест и убедиться, что он падает**

Run: `go test ./internal/service/backtest/ -run TestFESHRiskGridsPinTheirMeasuredAxes -v`
Expected: FAIL с `read ../../../data/params/rsi_pullback/fesh/cal_day.json: no such file or directory`.

- [ ] **Step 3: Создать `cal_day.json`**

Создать `data/params/rsi_pullback/fesh/cal_day.json`:

```json
{
  "_comment": "DAY: обе ветки дневного гейта совместно, 12 прогонов. Гейт двусторонний — вход разрешён либо когда день ещё не раскрылся (размах в пределах FreshDayATR), либо когда он уже исчерпан (размах достиг SpentDayATR); полоса между ними отвергается. Ось FreshDayATR [0, 0.25, 0.35] отличается от ugld/ [0, 0.1, 0.2, 0.3] по замеру внутридневного прогресса: медиана доли ATR, пройденной к концу k-го бара дня (будние бары), составляет 0.27 ко второму бару, 0.32 к третьему, 0.37 к пятому, 0.51 к восьмому, 0.58 к двенадцатому. Меряется по номеру бара, а не по часу, потому что утренняя сессия есть у 99% дней и начинается то в 06:30, то в 07:00, то в 09:30 — абсолютный слот смешивал бы дни с разным началом торгов. Доля будних баров, у которых размах дня на этот момент не превышает порога: 2.9% при 0.1, 6.3% при 0.2, 8.8% при 0.25, 12.2% при 0.3, 16.4% при 0.35. Пороги 0.1-0.2 оставляют ветке «день только начался» меньше 7% баров — она при них почти мертва. Ноль в оси выключает ветку целиком и служит контролем. Ось SpentDayATR [0.8 ... 1.5]: доля будних баров, у которых размах дня достиг порога, равна 54.3% при 0.6, 35.6% при 0.8, 21.3% при 1.0, 10.8% при 1.25, 6.0% при 1.5. Порог 0.6 пропустил бы больше половины баров и перестал быть гейтом. RSILower, который у ugld/ раздувает эту тему до 60 прогонов, здесь не свипуется: глубина отката принадлежит cal_entry.json. Запуск: go run ./cmd/backtest -ticker FESH -strategy rsi_pullback -interval Minutes30 -calibrate data/params/rsi_pullback/fesh/cal_day.json -out ./reports/FESH_day -months 36 -min-trades 20 -test-months 6 -metric profit_factor.",
  "phases": [
    {
      "name": "day",
      "grid": {
        "UseDayATRGate": [1],
        "FreshDayATR": [0, 0.25, 0.35],
        "SpentDayATR": [0.8, 1.0, 1.25, 1.5]
      }
    }
  ]
}
```

- [ ] **Step 4: Создать `cal_day_spent.json`**

Создать `data/params/rsi_pullback/fesh/cal_day_spent.json`:

```json
{
  "_comment": "DAY-SPENT: только ветка «день исчерпан», 6 прогонов. FreshDayATR=0 выключает ветку «день только начался» целиком (dayStateOK защищает её условием fresh > 0, поэтому ровно ноль убирает её), и остаётся чистый свип нижней границы позднего входа. Разделять ветки нужно потому, что это две разные стратегии на одном коде — ранний вход по тренду против возврата после распродажи, — и общий profit factor их усредняет: прибыльная ветка оплачивает убыточную, а лидерборд cal_day.json этого не показывает. Ось шире, чем в cal_day.json, и уходит до 1.75, потому что здесь порог не конкурирует за прогоны с утренней веткой. Доля будних баров, у которых размах дня достиг порога (n=24766): 0.6 — 54.3%, 0.8 — 35.6%, 1.0 — 21.3%, 1.25 — 10.8%, 1.5 — 6.0%, 1.75 — 3.8%. В днях, а не в барах, те же пороги дают 77.0%, 55.5%, 37.3%, 22.0%, 13.7%, 8.9% (n=1001). Строка 0.6 оставлена именно как контроль «гейт почти выключен»: если она выигрывает, ветка «день исчерпан» на FESH не несёт информации. Правый край 1.75 отбирает меньше четырёх процентов баров — его profit factor читать нельзя, только счёт сделок. Точки 0.4-0.5 из ugld/ не переносятся: на FESH они не гейтят вовсе. Запуск: go run ./cmd/backtest -ticker FESH -strategy rsi_pullback -interval Minutes30 -calibrate data/params/rsi_pullback/fesh/cal_day_spent.json -out ./reports/FESH_day_spent -months 36 -min-trades 20 -test-months 6 -metric profit_factor.",
  "phases": [
    {
      "name": "day_spent",
      "grid": {
        "UseDayATRGate": [1],
        "FreshDayATR": [0],
        "SpentDayATR": [0.6, 0.8, 1.0, 1.25, 1.5, 1.75]
      }
    }
  ]
}
```

- [ ] **Step 5: Создать `cal_volume.json`**

Создать `data/params/rsi_pullback/fesh/cal_volume.json`:

```json
{
  "_comment": "VOLUME: объёмный гейт, 8 прогонов. Гейт требует, чтобы хотя бы один из последних VolLookbackBars будних баров нёс объём в VolMult раз выше среднего ДЛЯ СВОЕГО СЛОТА за последние VolBaseDays будних дней — сравнение со слотом, а не с плоским средним, обязательно, потому что 30-минутный объём U-образен и плоская база мерила бы время суток вместо активности. Замер по кэшу FESH, база 5 будних дней (n=25201): отношение объёма бара к слотовой базе имеет медиану 0.62, p75 1.45, p90 3.30; гейт проходят 35.2% баров при 1.0, 29.9% при 1.2, 24.2% при 1.5, 17.9% при 2.0, 14.0% при 2.5. База 10 дней (n=25056): медиана 0.56, p75 1.26, p90 2.82; доли 30.9 / 26.2 / 20.8 / 14.8 / 11.5%. Все четыре точки множителя живые на обеих базах, поэтому верхняя граница здесь 2.5, а не 2.0 как у domrf/, где выборка дефицитная. База ограничена парой [5, 10]: короткая быстрее реагирует на смену активности, длинная устойчивее к одиночному всплеску, а точки 3 и 14 из ugld/ отвергнуты по существу — база в три дня ловит один выброс объёма и превращает гейт в лотерею, база в четырнадцать размывает его до бесполезности. Отдельная оговорка про FESH: дневной оборот падал с 670 млн ₽ (медиана 2023) до 120 млн (2025) и восстановился до 208 млн (2026), поэтому длинная база здесь сравнивает бар с заметно другим режимом активности — это аргумент в пользу базы 5, но проверяется он прогоном, а не рассуждением. Точка «гейт выключен» принадлежит cal_screen.json и здесь отсутствует намеренно. Запуск: go run ./cmd/backtest -ticker FESH -strategy rsi_pullback -interval Minutes30 -calibrate data/params/rsi_pullback/fesh/cal_volume.json -out ./reports/FESH_volume -months 36 -min-trades 20 -test-months 6 -metric profit_factor.",
  "phases": [
    {
      "name": "volume",
      "grid": {
        "UseVolume": [1],
        "VolMult": [1.2, 1.5, 2.0, 2.5],
        "VolBaseDays": [5, 10]
      }
    }
  ]
}
```

- [ ] **Step 6: Создать `cal_risk.json`**

Создать `data/params/rsi_pullback/fesh/cal_risk.json`:

```json
{
  "_comment": "RISK: стоп и цель, оба в единицах дневного ATR, оба замораживаются на входе, 25 прогонов. Дневной ATR(14) у FESH идёт медианой 4.42% цены (p10 2.98, p25 3.51, p75 5.11, p90 5.88) — это самый широкий инструмент из заведённых (ugld 4.28%, reni 3.36%, domrf 1.94%), поэтому ось наследуется от ugld/, а не от domrf/. Строка стопа 0.3 СОХРАНЕНА и это осознанно: круг издержек (0.05% комиссии за сторону, тик при цене около 62 руб пренебрежим) стоит 0.1% оборота, то есть 0.023 дневного ATR — самая дешёвая цифра среди четырёх тикеров, — и на стопе 0.3 ATR (= 1.33% цены) издержки съедают 8% риска. На DOMRF та же строка стоила 17% и была оттуда вырезана; копировать это сужение сюда нельзя. Безопасной строка от этого не становится, и второй замер важнее первого: стоп 0.3 ATR переживает целиком лишь 1.0% дней (0.5 ATR — 13.1%, 0.7 — 34.8%, 1.0 — 62.7%, 1.3 — 79.7%), то есть сидит глубоко внутри обычного внутридневного шума и будет снят сносом цены, а не провалом сетапа. Читать долю выходов по стопу, а не только profit factor. Верх оси стопа 1.3 при медианном дневном размахе 0.85 ATR. Цель доходит до 2.5 (около 11% цены) и свипуется в том числе НИЖЕ самого широкого стопа: цель меньше стопа требует win rate выше 50% просто чтобы выйти в ноль, и эта тема подтверждает или убивает такую асимметрию. StopDailyATR=0 намеренно ОТСУТСТВУЕТ и не должен появиться: позиция, которую держат через ночи и выходные без стопа, — не та конфигурация, которую калибровка вправе выбрать. Следить за средним временем удержания: стоп 1.3 с целью 2.5 превращает стратегию в многонедельную, а holdout FESH (2026-02-04 и далее, рост +13.1%) такие конфигурации польстит сильнее прочих. Запуск: go run ./cmd/backtest -ticker FESH -strategy rsi_pullback -interval Minutes30 -calibrate data/params/rsi_pullback/fesh/cal_risk.json -out ./reports/FESH_risk -months 36 -min-trades 20 -test-months 6 -metric profit_factor.",
  "phases": [
    {
      "name": "risk",
      "grid": {
        "StopDailyATR": [0.3, 0.5, 0.7, 1.0, 1.3],
        "TPDailyATR": [0.5, 1.0, 1.5, 2.0, 2.5]
      }
    }
  ]
}
```

- [ ] **Step 7: Создать `cal_exit.json`**

Создать `data/params/rsi_pullback/fesh/cal_exit.json`:

```json
{
  "_comment": "EXIT: полоса выхода по RSI, 6 прогонов. Это единственное место, где меряется RSIUpper: cal_entry.json намеренно её не свипует, чтобы не отдавать 96 комбинаций одной теме. Выход срабатывает, когда RSI пересекает уровень снизу вверх, и потому конкурирует с целью: чем ниже полоса, тем чаще сделка закрывается до TP. На UGLD именно RSI-выход даёт 61% выходов, и его полоса — не косметика, а основной механизм фиксации. Ось 55..80 совпадает с ugld/cal_exit.json и reni/cal_exit.json: она задаётся не шириной инструмента, а шкалой самого RSI, поэтому пересчитывать её под ATR FESH не нужно. Нижний край 55 стоит контрольной строкой «выходим почти сразу»: если он выигрывает, стратегия на этом инструменте работает как скальп, а не как многодневное удержание, и это меняет читаемость всех остальных тем. Запуск: go run ./cmd/backtest -ticker FESH -strategy rsi_pullback -interval Minutes30 -calibrate data/params/rsi_pullback/fesh/cal_exit.json -out ./reports/FESH_exit -months 36 -min-trades 20 -test-months 6 -metric profit_factor.",
  "phases": [
    {
      "name": "exit",
      "grid": {
        "RSIUpper": [55, 60, 65, 70, 75, 80]
      }
    }
  ]
}
```

- [ ] **Step 8: Создать `cal_trail.json`**

Создать `data/params/rsi_pullback/fesh/cal_trail.json`:

```json
{
  "_comment": "TRAIL: форма трейла и его конкуренция с RSI-выходом, 12 прогонов. Файл честен только ПОСЛЕ cal_risk.json, с зафиксированной парой стоп/цель: трейл становится связывающим стопом лишь когда maxFav - Trail*ATR поднимается выше entry - Stop*ATR, то есть когда цена прошла (Trail - Stop) дневных ATR вверх. Пакет strategy/fesh существует, но возвращает core.DefaultParams() (калибровка не проводилась), поэтому команда из этого же _comment без -params исполняется поверх baseline (StopDailyATR=0.5, TPDailyATR=0.6) — и свип поверх него измеряет трейл, зажатый в окно 0.1 ATR (0.6-0.5), а не то пространство, которое даст победившая пара cal_risk.json. Наличие пакета этого не меняет: пока в нём стоит baseline, он тождественен generic-ветке. UseTrail зафиксирован в [1] — тема меряет форму трейла, а не факт его включения; UseRSIExit свипуется обеими точками, потому что трейл и RSI-выход борются за одну и ту же сделку, и их совместный эффект не выводится из раздельных замеров. TrailDailyATR=0 внутри включённого трейла означает подтяжку вплотную к максимуму и стоит левым краем оси. Правый край 0.8, а не 0.6 как у ugld/: потолок там задавала не собственная ось ugld/cal_risk.json, а baseline core.DefaultParams() с TPDailyATR=0.6, до которого трейл не успевал взвестись; на FESH действует ровно тот же baseline, пока cal_risk.json не запущен и его победившая пара не зафиксирована в пакете или через -params — отсюда и условная честность файла выше. Замер, задающий шаг оси: медианный дневной размах FESH равен 0.85 ATR, поэтому подтяжка на 0.3-0.5 ATR отстаёт от цены примерно на треть-половину обычного дня, а 0.8 — почти на целый день. Читать надо не только profit factor, но и распределение причин выхода: если трейл забирает больше половины сделок, цель из cal_risk.json перестала работать и обе темы нужно перечитать вместе. Запуск: go run ./cmd/backtest -ticker FESH -strategy rsi_pullback -interval Minutes30 -calibrate data/params/rsi_pullback/fesh/cal_trail.json -out ./reports/FESH_trail -months 36 -min-trades 20 -test-months 6 -metric profit_factor.",
  "phases": [
    {
      "name": "trail",
      "grid": {
        "UseRSIExit": [0, 1],
        "UseTrail": [1],
        "TrailDailyATR": [0, 0.3, 0.4, 0.5, 0.6, 0.8]
      }
    }
  ]
}
```

- [ ] **Step 9: Запустить тесты и убедиться, что они проходят**

Run: `go test ./internal/service/backtest/ -run 'TestFESH|TestRSIPullbackCalFilesValid|TestRSIPullbackGridControlPoints' -v`
Expected: PASS. `TestRSIPullbackCalFilesValid` показывает все девять подтестов `fesh/cal_*.json`; `TestRSIPullbackGridControlPoints` проходит (в `cal_risk.json` цели 1.5/2.0/2.5 превышают самый широкий стоп 1.3).

- [ ] **Step 10: Коммит**

```bash
git add internal/service/backtest/rsi_pullback_fesh_grid_test.go data/params/rsi_pullback/fesh/
git commit -m "feat(rsi_pullback): сетки риска и гейтов FESH с замеренными осями"
```

---

### Task 4: Приёмка

**Files:**
- Modify: ничего (только запуски); при падении линта правится файл, на который он указал.

**Interfaces:**
- Consumes: весь результат Task 1–3.
- Produces: подтверждение, что оснастка рабочая — зелёный `mage ci` и один отчёт smoke-прогона.

- [ ] **Step 1: Прогнать полный гейт качества**

Run: `./bin/mage ci`
Expected: лint чист, `go test -race ./...` зелёный, mock-drift отсутствует. Если `./bin/mage` отсутствует — сначала `go run mage.go tools` (см. `docs/tooling/mage.md`).

- [ ] **Step 2: Smoke-запуск команды из `cal_screen.json`**

Взять команду ровно из `_comment` файла `data/params/rsi_pullback/fesh/cal_screen.json` и выполнить её:

```bash
go run ./cmd/backtest -ticker FESH -strategy rsi_pullback -interval Minutes30 \
  -calibrate data/params/rsi_pullback/fesh/cal_screen.json -out ./reports/FESH_screen \
  -months 36 -min-trades 1 -metric profit_factor
```

Expected: команда отрабатывает без ошибок и пишет отчёт в `./reports/FESH_screen`. Проверяются ровно две вещи: что тикер резолвится через реестр (а не падает) и что строка запуска не содержит опечаток. **Числа отчёта не читаются, никуда не переносятся и не обсуждаются** — это проверка оснастки, а не измерение стратегии.

- [ ] **Step 3: Убедиться, что отчёт создан**

Run: `ls -la reports/FESH_screen/`
Expected: непустой каталог с markdown-отчётом прогона.

- [ ] **Step 4: Коммит (если что-то правилось)**

Если Step 1 потребовал правок (линт, форматирование), закоммитить их:

```bash
git add -A
git commit -m "chore(rsi_pullback): правки по гейту качества для оснастки FESH"
```

Если правок не было — шаг пропускается, коммита нет. Отчёты в `reports/` не коммитятся.

---

## Итог

После всех четырёх задач:

- `FESH` резолвится через `rsiPullbackRegistry` и через живой реестр, возвращая baseline; попадание в боевую вселенную заблокировано существующим тестом;
- в `data/params/rsi_pullback/fesh/` лежат девять однотемных сеток на 105 прогонов, каждая ось обоснована замером FESH из спеки и прибита сторожевым тестом;
- `./bin/mage ci` зелёный, команда запуска проверена smoke-прогоном.

Не сделано намеренно (и не должно быть сделано в рамках этого плана): калибровочные прогоны девяти тем, чтение лидербордов, литерал параметров, ввод FESH в `RSI_PULLBACK_TICKERS`.

Порядок запуска тем при калибровке (за владельцем): `cal_screen` → `cal_entry` → `cal_trend` → `cal_day` / `cal_day_spent` → `cal_volume` → `cal_risk` → `cal_exit` → `cal_trail`. Первый прогон — с `-refresh`: кэш стоит на 2026-08-04.
