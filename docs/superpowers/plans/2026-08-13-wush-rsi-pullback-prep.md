# WUSH под rsi_pullback — план реализации

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Завести WUSH как калибруемый тикер `rsi_pullback`, прогнать девять тем под rolling walk-forward и вынести явный вердикт по объявленной заранее планке — литерал есть либо литерала нет.

**Architecture:** Оснастка повторяет структуру `data/params/rsi_pullback/fesh/`: девять однотемных файлов сеток, два сторожевых теста на оси (сигнальные и рисковые), пакет параметров `strategy/wush` в состоянии «калибровка не проводилась» с регистрацией в двух реестрах. Значения осей взяты из замеров WUSH, а не скопированы: одна ось (`FreshDayATR`) отличается от FESH по измеренной причине. Прогоны идут после оснастки, walk-forward первым как вердикт, литерал — только если планка взята.

**Tech Stack:** Go 1.25, `go test`, `./bin/mage ci` (golangci-lint v2 + `go test -race` + mock-drift), `cmd/backtest` с флагами `-calibrate/-months/-train-months/-test-months/-min-trades/-metric`, Python 3 для оркестрации прогонов.

**Spec:** `docs/superpowers/specs/2026-08-13-wush-rsi-pullback-prep-design.md`

## Global Constraints

- Тексты `_comment` в JSON и сообщений об ошибках в тестах — **на русском**, как в `reni/`, `domrf/`, `fesh/`.
- Каждый `cal_*.json` обязан содержать в `_comment` строку `wush/<имя файла>.json` — это требование `TestRSIPullbackCalFilesValid`, оно ловит команду запуска, скопированную у соседнего тикера.
- `StopDailyATR = 0` не появляется ни в одном файле: калибровка не имеет права отключить стоп.
- Ввод WUSH в `RSI_PULLBACK_TICKERS` и правка `env/*.env` — **вне этого плана** при любом исходе прогонов.
- Параллелизм прогонов — **не более трёх** одновременно: каждый запуск делает `load shares` и упирается в rate-limit Tinkoff API (`RateLimit(10, 62)`). Поднимать нельзя.
- Замеры WUSH, на которые ссылаются все обоснования (кэш 30m 33 823 бара / 23 901 будний, 36.0 месяца с 2023-08-04; Day1 1033 свечи / 923 будних):
  - дневной ATR(14): медиана **4.25%**, среднее 4.52%, p10 3.00 / p25 3.58 / p75 5.00 / p90 6.07 / max 13.41; круг издержек **0.024 ATR**;
  - режимы: train 2023-08-04—2026-02-03 **−58.9%** (226.50 → 92.99), holdout 2026-02-04—2026-08-03 **−48.0%** (93.41 → 48.61), пик-минимум **−91.0%**;
  - дневной размах в долях ATR: медиана **0.83**; ≥0.6 — 74.7%, ≥0.8 — 52.9%, ≥1.0 — 34.6%, ≥1.25 — 20.3%, ≥1.5 — 12.8%, ≥1.75 — 8.6%;
  - выживаемость стопа (размах дня меньше k·ATR): 0.3 — **2.4%**, 0.5 — 16.1%, 0.7 — 36.8%, 1.0 — 65.4%, 1.3 — **81.9%**;
  - внутридневной прогресс: бар 2 — **0.33** ATR, бар 3 — 0.37, бар 5 — 0.43, бар 8 — 0.55, бар 12 — 0.65;
  - fresh-ветка (доля будних баров с размахом ≤ порога, n=23 306): ≤0.2 — 5.0%, ≤0.25 — 7.6%, **≤0.3 — 11.3%**, ≤0.35 — 15.3%, **≤0.4 — 21.2%**;
  - spent-ветка (≥ порога): ≥0.6 — **56.8%**, ≥0.8 — 35.8%, ≥1.0 — 22.5%, ≥1.25 — 11.2%, ≥1.5 — 6.2%, ≥1.75 — 4.1%;
  - кроссы RSI вниз, будние: RSI(4) 356/657/1036/1451/1929 через 10/15/20/25/30; RSI(5) 194/422/718/1085/1526; RSI(6) **118**/278/522/823/1242; RSI(7) **60**/206/389/677/1015;
  - объёмы, база 5 дней: ≥1.2 — 30.3%, ≥1.5 — 24.3%, ≥2.0 — 18.1%, ≥2.5 — 13.7%; база 10 дней: 26.9 / 21.5 / 15.7 / 12.0%;
  - трендовый фильтр `EMAFast > EMASlow`: **41–43%** во всех шестнадцати парах;
  - оборот 356 млн ₽/день, **лот 1**; скринер: место 4, `PFmed 1.94` / `TradesMed 36` / `Plateau 58%` / `Capped 4/24`.

---

### Task 1: Сигнальные сетки WUSH и сторожевой тест на их оси

**Files:**
- Create: `data/params/rsi_pullback/wush/cal_screen.json`
- Create: `data/params/rsi_pullback/wush/cal_entry.json`
- Create: `data/params/rsi_pullback/wush/cal_trend.json`
- Create: `data/params/rsi_pullback/wush/cal_exit.json`
- Test: `internal/service/backtest/rsi_pullback_wush_grid_test.go`

**Interfaces:**
- Consumes: хелперы из `internal/service/backtest/rsi_pullback_grid_test.go` — `rsiPullbackTickerGrid(t *testing.T, ticker, file string) map[string][]float64`; и `sameSet(got []float64, want ...float64) bool` из `rsi_pullback_reni_grid_test.go`. Оба уже существуют, создавать не нужно.
- Produces: функция `wushGrid(t *testing.T, file string) map[string][]float64` — ею пользуется Task 2 в том же файле теста.

- [ ] **Step 1: Написать падающий тест на сигнальные оси**

Создать `internal/service/backtest/rsi_pullback_wush_grid_test.go`:

```go
package backtest

import (
	"math"
	"testing"
)

// wushGrid читает файл сеток WUSH через общий хелпер.
func wushGrid(t *testing.T, file string) map[string][]float64 {
	t.Helper()
	return rsiPullbackTickerGrid(t, "wush", file)
}

// TestWUSHSignalGridsPinTheirMeasuredAxes сторожит оси, обоснованные замерами инструмента, а не
// вкусом. Каталог wush/ заводится копированием структуры fesh/, и типовая ошибка такой копии —
// притащить вместе с формой чужие обоснования. По ширине WUSH стоит рядом с FESH (медианный
// дневной ATR 4.25% против 4.42%), поэтому сужения DOMRF сюда не переносятся; оговорку RENI про
// мёртвые углы переносить тоже не нужно — слабейший угол RSI(7)@10 даёт 60 будних кроссов против
// 23 у RENI и 49 у FESH.
func TestWUSHSignalGridsPinTheirMeasuredAxes(t *testing.T) {
	screen := wushGrid(t, "cal_screen.json")
	for _, field := range []string{"UseDayATRGate", "UseVolume"} {
		if got := screen[field]; !sameSet(got, 0, 1) {
			t.Errorf("cal_screen.json: %s = %v, want ровно {0,1} — тема меряет цену каждого гейта в сделках, а любая другая пара не даёт сравнить гейт с его отсутствием", field, got)
		}
	}

	entry := wushGrid(t, "cal_entry.json")
	// Глубже 25 порог перестаёт отбирать откат: RSI(4) уходит под 30 1929 раз по будням за
	// 36 месяцев — это обычный шум, а не сетап.
	for _, v := range entry["RSILower"] {
		if v > 25 {
			t.Errorf("cal_entry.json свипует RSILower=%v: выше 25 порог перестаёт отбирать откат (1929 будних кроссов под 30)", v)
		}
	}
	// Ниже 10 выборка истончается быстрее, чем растёт качество сигнала: у RSI(7) на уровне 10
	// остаётся 60 будних кроссов за всю историю, и более глубокий порог режет и их.
	for _, v := range entry["RSILower"] {
		if v < 10 {
			t.Errorf("cal_entry.json свипует RSILower=%v: ниже 10 сигналов почти не остаётся (у RSI(7)@10 их 60)", v)
		}
	}
	// Уровень 10 обязан остаться: скринер выбрал для WUSH лучшей конфигурацией RSI 6/10, и
	// 118 будних кроссов RSI(6)@10 эту точку выдерживают. На DOMRF таких кроссов было 18, и там
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

	trend := wushGrid(t, "cal_trend.json")
	// EMASlow=200 исполнима: Lookback при ней равен 2*200+20 = 420 баров против 23901 будних
	// в кэше, то есть окно прогрева занимает 1.8% истории.
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

	exit := wushGrid(t, "cal_exit.json")
	// Это единственное место, где меряется полоса выхода: cal_entry.json её намеренно не свипует.
	if got := exit["RSIUpper"]; !sameSet(got, 55, 60, 65, 70, 75, 80) {
		t.Errorf("cal_exit.json: RSIUpper = %v, want ровно {55,60,65,70,75,80} — cal_entry.json полосу выхода не свипует, а любая точка вне шкалы RSI или пропуск внутри неё сужает единственное место, где эта полоса измеряется", got)
	}
}
```

- [ ] **Step 2: Запустить тест и убедиться, что он падает**

Run: `go test ./internal/service/backtest/ -run TestWUSHSignalGridsPinTheirMeasuredAxes -v`
Expected: FAIL — `read data/params/rsi_pullback/wush/cal_screen.json: no such file or directory`. Каталога `wush/` ещё нет.

- [ ] **Step 3: Создать `cal_screen.json`**

```json
{
  "_comment": "SCREEN: цена каждого из двух необязательных гейтов, 4 прогона. Тема отвечает на один вопрос — сколько сделок остаётся при каждой комбинации UseDayATRGate и UseVolume, — и потому обе оси обязаны быть ровно {0,1}: без точки «выключено» гейт не с чем сравнить. На WUSH этот вопрос важнее, чем на предыдущих тикерах: обучающее окно 2023-08-04—2026-02-03 это падение 226.50 -> 92.99 (-58.9%), holdout 2026-02-04—2026-08-03 — падение 93.41 -> 48.61 (-48.0%), и трендовый фильтр в таком режиме держит EMAFast > EMASlow лишь 41-43% времени. Сколько сделок переживёт добавление к этому ещё двух гейтов — первое, что надо знать про инструмент. Читать колонку сделок, а не profit factor. Порог -min-trades 1 обязателен: значение по умолчанию отфильтровало бы ровно те строки, которые тема измеряет. Запуск: go run ./cmd/backtest -ticker WUSH -strategy rsi_pullback -interval Minutes30 -calibrate data/params/rsi_pullback/wush/cal_screen.json -out ./reports/WUSH_screen -months 36 -train-months 12 -test-months 6 -min-trades 1 -metric profit_factor.",
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

```json
{
  "_comment": "ENTRY, форма отката: RSIPeriod x RSILower, 16 прогонов. Ось совпадает с fesh/cal_entry.json по форме, но обоснована собственным замером: кроссы RSI вниз через уровень, будние бары за 36.0 месяца — RSI(4) 356 через 10, 657 через 15, 1036 через 20, 1451 через 25; RSI(5) 194/422/718/1085; RSI(6) 118/278/522/823; RSI(7) 60/206/389/677. Живы ВСЕ шестнадцать углов, включая слабейший RSI(7)@10 с 60 кроссами — это больше, чем у FESH (49), и втрое больше, чем у RENI (23), где угол был помечен как заведомо мёртвый; ту оговорку сюда переносить не нужно. Уровень 10 оставлен: скринер выбрал для WUSH лучшей конфигурацией именно RSI 6/10 (PFmed 1.94 на TradesMed 36), и 118 будних кроссов RSI(6)@10 эту точку выдерживают. Уровень 30 не берётся: RSI(4) уходит под него 1929 раз по будням, это шум, а не откат. RSIUpper здесь НЕ свипуется: полоса выхода меряется отдельно, файлом cal_exit.json, иначе тема раздувается до 96 комбинаций. Читая лидерборд, помнить про режим: у WUSH падают ОБА окна протокола — train -58.9%, holdout -48.0%, пик-минимум -91.0%. Это значит, что низкий in-sample profit factor здесь не то же самое, что низкий PF на аптрендовом тикере, и что holdout впервые не завышен механически: в отличие от FESH, где holdout рос на 13.1% и почти любой купленный откат там закрывался в плюс, здесь завысить результат нечем. Обратная сторона — трендовый фильтр большую часть времени закрывает вход, поэтому счёт сделок читать раньше profit factor. Запуск: go run ./cmd/backtest -ticker WUSH -strategy rsi_pullback -interval Minutes30 -calibrate data/params/rsi_pullback/wush/cal_entry.json -out ./reports/WUSH_entry -months 36 -train-months 12 -test-months 6 -min-trades 20 -metric profit_factor.",
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

```json
{
  "_comment": "TREND, фильтр направления: EMAFast x EMASlow, полная сетка 4x4, 16 прогонов. Резать ось нечем: доля 30-минутных баров с EMAFast > EMASlow составляет 41-43% во ВСЕХ шестнадцати парах (5/50 — 43%, 30/200 — 41%), то есть ни одна пара не открывает и не закрывает вход постоянно. Диапазон заметно ниже, чем у FESH (46-47%), и это прямое следствие режима: у WUSH падают оба окна протокола, train -58.9% и holdout -48.0%, пик-минимум -91.0%. EMASlow=200 исполнима: Lookback при ней равен 2*200+20 = 420 баров против 23901 будних в кэше, окно прогрева занимает 1.8% истории. Эта тема — одна из двух, по которым спека выносит вердикт по тикеру (вторая — cal_entry.json): у FESH она дала pooled OOS PF 0.953 при EMASlow, гулявшем 100/50/150/100 по фолдам, и именно это закрыло вопрос о подтверждении. Смотреть не только pooled PF, но и то, повторяется ли выбранное значение хотя бы в трёх фолдах из четырёх. Запуск: go run ./cmd/backtest -ticker WUSH -strategy rsi_pullback -interval Minutes30 -calibrate data/params/rsi_pullback/wush/cal_trend.json -out ./reports/WUSH_trend -months 36 -train-months 12 -test-months 6 -min-trades 20 -metric profit_factor.",
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

- [ ] **Step 6: Создать `cal_exit.json`**

```json
{
  "_comment": "EXIT, полоса выхода: RSIUpper, 6 прогонов. Это единственное место, где ось выхода меряется — cal_entry.json её намеренно не свипует, чтобы не раздувать тему входа до 96 комбинаций. Шкала берётся целиком от 55 до 80 с шагом 5: пропуск внутри неё сузил бы единственное измерение этой оси, а точки вне шкалы RSI бессмысленны. Уровень выхода на WUSH стоит читать вместе со счётом сделок и медианой удержания: у FESH RSI-выход давал 82% всех выходов и всю прибыль, тогда как цель срабатывала 2 раза из 60 за три года — если картина повторится, ось TPDailyATR в cal_risk.json окажется инертной, и это надо будет заметить по этой теме, а не списать на шум. Режим инструмента: падают оба окна протокола (train -58.9%, holdout -48.0%), поэтому ранний выход по RSI здесь может выигрывать не потому, что он лучше, а потому, что удержание в падающем рынке дороже. Запуск: go run ./cmd/backtest -ticker WUSH -strategy rsi_pullback -interval Minutes30 -calibrate data/params/rsi_pullback/wush/cal_exit.json -out ./reports/WUSH_exit -months 36 -train-months 12 -test-months 6 -min-trades 20 -metric profit_factor.",
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

- [ ] **Step 7: Запустить тесты и убедиться, что они проходят**

Run: `go test ./internal/service/backtest/ -run 'TestWUSHSignalGridsPinTheirMeasuredAxes|TestRSIPullbackCalFilesValid|TestRSIPullbackGridControlPoints' -v`
Expected: PASS. `TestRSIPullbackCalFilesValid` подхватывает новый каталог автоматически (обход рекурсивный) и проверяет резолвимость полей и путь в `_comment`.

- [ ] **Step 8: Мутационная проверка сторожевого теста**

Временно поменять в `cal_entry.json` значение `RSILower` с `[10, 15, 20, 25]` на `[15, 20, 25, 30]` и убедиться, что тест краснеет двумя ошибками сразу (свип 30 и отсутствие 10). Затем вернуть как было.

Run: `go test ./internal/service/backtest/ -run TestWUSHSignalGridsPinTheirMeasuredAxes`
Expected: сначала FAIL с обоими сообщениями, после отката — PASS.

- [ ] **Step 9: Коммит**

```bash
git add data/params/rsi_pullback/wush/cal_screen.json data/params/rsi_pullback/wush/cal_entry.json data/params/rsi_pullback/wush/cal_trend.json data/params/rsi_pullback/wush/cal_exit.json internal/service/backtest/rsi_pullback_wush_grid_test.go
git commit -m "feat(rsi_pullback): сигнальные сетки WUSH с замеренными осями"
```

---

### Task 2: Сетки риска и гейтов WUSH и сторожевой тест на них

**Files:**
- Create: `data/params/rsi_pullback/wush/cal_day.json`
- Create: `data/params/rsi_pullback/wush/cal_day_spent.json`
- Create: `data/params/rsi_pullback/wush/cal_volume.json`
- Create: `data/params/rsi_pullback/wush/cal_risk.json`
- Create: `data/params/rsi_pullback/wush/cal_trail.json`
- Modify: `internal/service/backtest/rsi_pullback_wush_grid_test.go` (дописать вторую тест-функцию)

**Interfaces:**
- Consumes: `wushGrid` из Task 1.
- Produces: ничего для последующих задач.

- [ ] **Step 1: Написать падающий тест на рисковые оси**

Дописать в конец `internal/service/backtest/rsi_pullback_wush_grid_test.go`:

```go
// TestWUSHRiskGridsPinTheirMeasuredAxes сторожит оси стопа, цели и обоих гейтов. Каждое число
// ниже опирается на замер по кэшу WUSH, а не на перенос с соседнего тикера: медианный дневной
// ATR 4.25%, круг издержек 0.024 ATR, медианный дневной размах 0.83 ATR.
func TestWUSHRiskGridsPinTheirMeasuredAxes(t *testing.T) {
	day := wushGrid(t, "cal_day.json")
	if got := day["UseDayATRGate"]; len(got) != 1 || got[0] != 1 {
		t.Errorf("cal_day.json: UseDayATRGate = %v, want ровно [1] — цена отключения гейта меряется в cal_screen.json", got)
	}
	// Тикер-специфичное утверждение этого каталога. Медианный день WUSH проходит 0.33 ATR уже
	// ко второму бару против 0.28 у FESH, поэтому ветка «день только начался» здесь уже:
	// порог 0.25 оставляет ей 7.6% будних баров, 0.3 — 11.3%, 0.4 — 21.2%. Ось [0, 0.25, 0.35],
	// скопированная из fesh/, подсунула бы калибровке почти мёртвую ветку и дала ложный вывод
	// «ветка не нужна».
	for _, v := range day["FreshDayATR"] {
		if v > 0 && v < 0.3 {
			t.Errorf("cal_day.json свипует FreshDayATR=%v: ко второму бару медианный день WUSH прошёл 0.33 ATR, ветке остаётся меньше 8%% баров (у FESH порог 0.25 давал 8.8%%, здесь — 7.6%%)", v)
		}
	}
	// Порог 0.6 проходят 56.8% будних баров — на этом инструменте он перестаёт быть гейтом.
	for _, v := range day["SpentDayATR"] {
		if v < 0.8 {
			t.Errorf("cal_day.json свипует SpentDayATR=%v: порог достижим для 57%% баров, это не гейт", v)
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
	// RSILower в этой фазе не свипуется: глубина отката принадлежит cal_entry.json. Тема
	// обязана остаться однотемной.
	if got := day["RSILower"]; len(got) != 0 {
		t.Errorf("cal_day.json свипует RSILower=%v: глубина отката принадлежит cal_entry.json", got)
	}

	spent := wushGrid(t, "cal_day_spent.json")
	if got := spent["FreshDayATR"]; len(got) != 1 || got[0] != 0 {
		t.Errorf("cal_day_spent.json: FreshDayATR = %v, want ровно [0] — ветка «день начался» выключена целиком", got)
	}
	// Левый край 0.6 стоит контрольной строкой «гейт почти выключен» (56.8% баров). Точки
	// ниже на WUSH не гейтят вовсе и заняли бы прогоны впустую.
	for _, v := range spent["SpentDayATR"] {
		if v < 0.6 {
			t.Errorf("cal_day_spent.json свипует SpentDayATR=%v: ниже 0.6 порог не гейтит (0.6 проходят 57%% баров)", v)
		}
	}

	vol := wushGrid(t, "cal_volume.json")
	if got := vol["UseVolume"]; len(got) != 1 || got[0] != 1 {
		t.Errorf("cal_volume.json: UseVolume = %v, want ровно [1] — точка «гейт выключен» принадлежит cal_screen.json", got)
	}
	// Все четыре точки живые: при базе 5 дней гейт проходят 30.3% баров при 1.2, 24.3% при 1.5,
	// 18.1% при 2.0, 13.7% при 2.5. Выше 2.5 остаётся меньше седьмой части баров, и объёмный
	// гейт начинает резать выборку сильнее, чем несёт информации.
	for _, v := range vol["VolMult"] {
		if v > 2.5 {
			t.Errorf("cal_volume.json свипует VolMult=%v: выше 2.5 гейт проходит меньше 14%% баров", v)
		}
	}
	// Только две базы, 5 и 10: короткая быстрее реагирует на смену активности, длинная
	// устойчивее к одиночному всплеску. База 3 ловит один выброс объёма, база 14 — размывает;
	// на вторичном гейте лишние степени свободы не окупаются.
	for _, v := range vol["VolBaseDays"] {
		if v != 5 && v != 10 {
			t.Errorf("cal_volume.json свипует VolBaseDays=%v: обоснованы только 5 и 10 (3 ловит одиночный всплеск, 14 размывает)", v)
		}
	}

	risk := wushGrid(t, "cal_risk.json")
	// Круг издержек стоит 0.024 дневного ATR — почти как у FESH (0.023) и вдвое дешевле DOMRF
	// (0.052): на стопе 0.3 ATR (= 1.28% цены) комиссия съедает 8% риска. На DOMRF та же строка
	// стоила 17% и была оттуда вырезана — при копировании сеток это сужение легко притащить по
	// ошибке, поэтому присутствие строки проверяется явно.
	hasTightStop := false
	for _, v := range risk["StopDailyATR"] {
		if v == 0.3 {
			hasTightStop = true
		}
	}
	if !hasTightStop {
		t.Errorf("cal_risk.json: StopDailyATR = %v, не содержит 0.3 (издержки 0.024 ATR за круг эту строку лицензируют)", risk["StopDailyATR"])
	}
	// Нижняя граница оси: тот же круг издержек, который лицензирует строку 0.3 (8% риска),
	// запрещает идти уже. На 0.15 доля выросла бы до 16%, на 0.1 — до 24%: это уже та черта,
	// по которой DOMRF отверг свою строку 0.3 при 17%. «Попробуем стоп потуже» не должно
	// суметь добавить такую строку молча.
	for _, v := range risk["StopDailyATR"] {
		if v > 0 && v < 0.3 {
			t.Errorf("cal_risk.json свипует StopDailyATR=%v: издержки съели бы %.0f%% риска (круг 0.024 ATR) против 8%% на разрешённой строке 0.3; для сравнения, DOMRF отверг свою строку 0.3 при 17%%", v, 0.024/v*100)
		}
	}
	// Верх оси 1.3: медианный день покрывает 0.83 ATR, такой стоп переживает целиком 81.9%
	// дней. Шире — это уже не стоп, а отсутствие стопа, оплаченное размером позиции.
	for _, v := range risk["StopDailyATR"] {
		if v > 1.3 {
			t.Errorf("cal_risk.json свипует StopDailyATR=%v: шире 1.3 ATR стоп перестаёт быть стопом (медианный день 0.83 ATR)", v)
		}
	}

	trail := wushGrid(t, "cal_trail.json")
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

Run: `go test ./internal/service/backtest/ -run TestWUSHRiskGridsPinTheirMeasuredAxes -v`
Expected: FAIL — `read data/params/rsi_pullback/wush/cal_day.json: no such file or directory`.

- [ ] **Step 3: Создать `cal_day.json`**

```json
{
  "_comment": "DAY: обе ветки дневного гейта совместно, 12 прогонов. Гейт двусторонний — вход разрешён либо когда день ещё не раскрылся (размах в пределах FreshDayATR), либо когда он уже исчерпан (размах достиг SpentDayATR); полоса между ними отвергается. Ось FreshDayATR [0, 0.3, 0.4] — ЕДИНСТВЕННОЕ место, где каталог wush/ расходится с fesh/, и расхождение замерено, а не выбрано. Медиана доли ATR, пройденной к концу k-го бара дня (будние бары), у WUSH составляет 0.33 ко второму бару, 0.37 к третьему, 0.43 к пятому, 0.55 к восьмому, 0.65 к двенадцатому — против 0.27/0.32/0.37/0.51/0.58 у FESH. День здесь раскрывается быстрее, поэтому ветка «день только начался» уже: доля будних баров, у которых размах дня не превышает порога, равна 5.0% при 0.2, 7.6% при 0.25, 11.3% при 0.3, 15.3% при 0.35, 21.2% при 0.4. Точки 0.3 и 0.4 дают ветке те же доли, что 0.25 и 0.35 давали у FESH (8.8% и 16.4%); копирование чисел из fesh/ оставило бы ветке 7.6% баров и дало бы ложный вывод «ветка не нужна». Меряется по номеру бара, а не по часу, потому что утренняя сессия начинается то в 06:30 (44% дней), то в 09:30 (35%), то в 07:00 (18%) — абсолютный слот смешивал бы дни с разным началом торгов. Ноль в оси выключает ветку целиком и служит контролем. Ось SpentDayATR [0.8 ... 1.5]: доля будних баров, у которых размах дня достиг порога, равна 56.8% при 0.6, 35.8% при 0.8, 22.5% при 1.0, 11.2% при 1.25, 6.2% при 1.5. Порог 0.6 пропустил бы больше половины баров и перестал быть гейтом. RSILower здесь не свипуется: глубина отката принадлежит cal_entry.json. Доли баров по обеим осям посчитаны по всему 36-месячному пулу разом, а пул однороден иначе, чем у FESH: у WUSH падают ОБА окна протокола — train 2023-08-04—2026-02-03 это -58.9%, holdout 2026-02-04—2026-08-03 это -48.0%, пик-минимум -91.0%. Запуск: go run ./cmd/backtest -ticker WUSH -strategy rsi_pullback -interval Minutes30 -calibrate data/params/rsi_pullback/wush/cal_day.json -out ./reports/WUSH_day -months 36 -train-months 12 -test-months 6 -min-trades 20 -metric profit_factor.",
  "phases": [
    {
      "name": "day",
      "grid": {
        "UseDayATRGate": [1],
        "FreshDayATR": [0, 0.3, 0.4],
        "SpentDayATR": [0.8, 1.0, 1.25, 1.5]
      }
    }
  ]
}
```

- [ ] **Step 4: Создать `cal_day_spent.json`**

```json
{
  "_comment": "DAY_SPENT: одна ветка дневного гейта, «день исчерпан», широкой осью, 6 прогонов. Ветка «день только начался» выключена целиком (FreshDayATR ровно [0]), чтобы ось SpentDayATR мерилась без примеси второй полосы. Доля будних баров, у которых размах дня достиг порога: 56.8% при 0.6, 35.8% при 0.8, 22.5% при 1.0, 11.2% при 1.25, 6.2% при 1.5, 4.1% при 1.75. Левый край 0.6 стоит КОНТРОЛЬНОЙ строкой «гейт почти выключен»: если она выигрывает, ветка «день исчерпан» на WUSH не несёт информации, и это самостоятельный результат, а не повод расширять ось вниз. Правый край 1.75 отбирает 4.1% баров — его profit factor читать нельзя, только счёт сделок. На инструменте, где падают оба окна протокола (train -58.9%, holdout -48.0%), эта ветка проверяет ровно ту гипотезу, ради которой гейт и заводился: что после уже случившегося за день падения следующая нога вниз менее вероятна. Запуск: go run ./cmd/backtest -ticker WUSH -strategy rsi_pullback -interval Minutes30 -calibrate data/params/rsi_pullback/wush/cal_day_spent.json -out ./reports/WUSH_day_spent -months 36 -train-months 12 -test-months 6 -min-trades 20 -metric profit_factor.",
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

```json
{
  "_comment": "VOLUME, вторичный гейт: VolMult x VolBaseDays, 8 прогонов. Точка «гейт выключен» здесь не свипуется — она принадлежит cal_screen.json, — поэтому UseVolume ровно [1]. Замер отношения объёма бара к слотовой базе (база строится по завершённым будним дням с исключением текущего, как volumeBaseline в ядре): при базе 5 дней медиана 0.64, p75 1.46, p90 3.19, а гейт проходят 35.5% баров при 1.0, 30.3% при 1.2, 24.3% при 1.5, 18.1% при 2.0, 13.7% при 2.5; при базе 10 дней медиана 0.58, p75 1.30, p90 2.89 и доли 32.1/26.9/21.5/15.7/12.0%. Обе базы живые на всех четырёх множителях, поэтому ось VolBaseDays ограничена ровно ими: база 3 дня ловит один выброс объёма, 14 — размывает, и на вторичном гейте лишние степени свободы не окупаются. Выше 2.5 гейт проходит меньше седьмой части баров и начинает резать выборку сильнее, чем несёт информации. Запуск: go run ./cmd/backtest -ticker WUSH -strategy rsi_pullback -interval Minutes30 -calibrate data/params/rsi_pullback/wush/cal_volume.json -out ./reports/WUSH_volume -months 36 -train-months 12 -test-months 6 -min-trades 20 -metric profit_factor.",
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

```json
{
  "_comment": "RISK: StopDailyATR x TPDailyATR, 25 прогонов. Нижняя строка стопа 0.3 сохранена и лицензирована замером издержек: комиссия 0.05% за сторону даёт круг 0.1% цены = 0.024 дневного ATR, то есть на стопе 0.3 ATR (1.28% цены) круг съедает 8% риска. На DOMRF та же строка стоила 17% риска и была оттуда вырезана — при копировании сеток это сужение легко притащить по ошибке. Но читать строку 0.3 надо вместе со вторым замером: полный дневной размах меньше 0.3 ATR лишь у 2.4% дней, то есть такой стоп сидит глубоко внутри обычного внутридневного шума и будет снят сносом цены, а не провалом сетапа. Смотреть долю выходов по стопу, а не только profit factor. Верх оси 1.3 переживают 81.9% дней (медианный дневной размах 0.83 ATR); шире стоп перестаёт быть стопом. Ось целей до 2.5 свипуется в том числе НИЖЕ самого широкого стопа: цель меньше стопа требует win rate выше 50% просто чтобы выйти в ноль, и тема подтверждает или убивает такую асимметрию. Отдельно проверить, не инертна ли ось цели: у FESH цель срабатывала 2 раза из 60 за три года, и TPDailyATR 1.0/1.5/2.0/2.5 давали там одни и те же сделки — если то же повторится на WUSH, выбранное значение TPDailyATR будет стоять на плато, а не выбрано. StopDailyATR = 0 не появляется здесь ни при каких условиях: калибровка не имеет права отключить стоп. Запуск: go run ./cmd/backtest -ticker WUSH -strategy rsi_pullback -interval Minutes30 -calibrate data/params/rsi_pullback/wush/cal_risk.json -out ./reports/WUSH_risk -months 36 -train-months 12 -test-months 6 -min-trades 20 -metric profit_factor.",
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

- [ ] **Step 7: Создать `cal_trail.json`**

```json
{
  "_comment": "TRAIL, форма трейла против RSI-выхода: UseRSIExit x TrailDailyATR, 12 прогонов. UseTrail ровно [1] — тема меряет форму трейла, а не факт его включения. UseRSIExit свипуется на {0,1}, потому что трейл и RSI-выход конкурируют за одну и ту же сделку: без обеих точек нельзя сказать, вытесняет ли трейл основной выход или дополняет его. TrailDailyATR = 0 в оси означает трейл, взводящийся сразу, и служит левым контролем. Правый край 0.8, а не 0.6 как у ugld/: там потолок задавала узкая цель TPDailyATR=0.6, выше которой трейл не успевал взвестись, здесь же ось цели в cal_risk.json поднята до 2.5, и трейлу нужно пространство для позднего срабатывания. На WUSH эта тема важнее обычного: падают оба окна протокола (train -58.9%, holdout -48.0%, пик-минимум -91.0%), а в таком режиме трейл может выигрывать просто потому, что фиксирует прибыль до следующей ноги вниз, — то есть выигрывать по причине, которая не переживёт смену режима. Сравнивать не только profit factor, но и медиану удержания и долю выходов каждого типа. Запуск: go run ./cmd/backtest -ticker WUSH -strategy rsi_pullback -interval Minutes30 -calibrate data/params/rsi_pullback/wush/cal_trail.json -out ./reports/WUSH_trail -months 36 -train-months 12 -test-months 6 -min-trades 20 -metric profit_factor.",
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

- [ ] **Step 8: Запустить весь пакет тестов**

Run: `go test ./internal/service/backtest/ -v`
Expected: PASS, включая `TestRSIPullbackCalFilesValid` (все девять файлов `wush/` резолвятся, `_comment` каждого называет свой путь) и `TestRSIPullbackGridControlPoints` (в `cal_risk.json` есть цель выше самого широкого стопа: 2.5 > 1.3).

- [ ] **Step 9: Мутационная проверка тикер-специфичной оси**

Временно поменять в `cal_day.json` ось `FreshDayATR` с `[0, 0.3, 0.4]` на `[0, 0.25, 0.35]` — ровно ту ошибку, которую тест обязан ловить, то есть копирование оси из `fesh/`.

Run: `go test ./internal/service/backtest/ -run TestWUSHRiskGridsPinTheirMeasuredAxes`
Expected: FAIL с сообщением про «ко второму бару медианный день WUSH прошёл 0.33 ATR». Затем вернуть `[0, 0.3, 0.4]` и убедиться, что тест зелёный.

- [ ] **Step 10: Коммит**

```bash
git add data/params/rsi_pullback/wush/ internal/service/backtest/rsi_pullback_wush_grid_test.go
git commit -m "feat(rsi_pullback): сетки риска и гейтов WUSH с замеренными осями"
```

---

### Task 3: Пакет `strategy/wush` в состоянии «калибровка не проводилась»

**Files:**
- Create: `internal/service/trading_strategy/rsi_pullback/strategy/wush/wush.go`
- Modify: `internal/service/backtest/rsi_pullback_registry.go`
- Modify: `internal/service/trading_strategy/rsi_pullback/live/registry.go`
- Modify: `internal/service/trading_strategy/rsi_pullback/live/registry_test.go` (устаревший комментарий)
- Modify: `docs/rsi_pullback/strategy.md` (§8.0.1)
- Test: `internal/service/backtest/rsi_pullback_registry_test.go`

**Interfaces:**
- Consumes: `core.Params`, `core.DefaultParams()` из `internal/service/trading_strategy/rsi_pullback/strategy/core`.
- Produces: `wush.Ticker` (константа `"WUSH"`) и `wush.DefaultParams() core.Params` — их использует Task 6, если планка будет взята.

- [ ] **Step 1: Написать падающий тест на отслеживание baseline**

Дописать в `internal/service/backtest/rsi_pullback_registry_test.go` (импорт `rsipullbackwush "tinvest/internal/service/trading_strategy/rsi_pullback/strategy/wush"` добавить к остальным):

```go
// TestRSIPullbackWUSHTracksBaseline держит состояние «калибровка не проводилась». Пакет
// strategy/wush заведён 2026-08-13 ДО первого прогона — как место под будущий литерал и
// носитель замеров инструмента (спека docs/superpowers/specs/2026-08-13-wush-rsi-pullback-prep-design.md),
// и цена такого исключения в том, что по карте реестра неотличимо, настроен тикер или нет:
// запись WUSH выглядит ровно так же, как запись откалиброванного соседа. Поэтому состояние
// держит тест. Когда калибровка пройдёт объявленную планку и литерал появится, этот тест
// ЗАМЕНЯЕТСЯ на снимок литерала (как TestRSIPullbackFESHIsRegisteredAndCalibrated) — правка
// одного лишь пакета обязана валить сборку здесь.
func TestRSIPullbackWUSHTracksBaseline(t *testing.T) {
	b, ok := rsiPullbackRegistry["WUSH"]
	if !ok {
		t.Fatal("WUSH отсутствует в rsiPullbackRegistry: тикер провалится в generic-ветку")
	}
	p, ok := b.DefaultParams().(core.Params)
	if !ok {
		t.Fatalf("WUSH: DefaultParams() вернул %T, want core.Params", b.DefaultParams())
	}
	if p != core.DefaultParams() {
		t.Fatalf("WUSH params = %+v, want baseline %+v — калибровка не проводилась, литерала быть не должно; если он появился, замените этот тест на снимок литерала", p, core.DefaultParams())
	}
	if p != rsipullbackwush.DefaultParams() {
		t.Fatalf("реестр вернул %+v, а пакет — %+v: реестр обязан отдавать параметры своего пакета", p, rsipullbackwush.DefaultParams())
	}
	if got := b.Build(p).Ticker(); got != "WUSH" {
		t.Fatalf("Ticker() = %q, want WUSH", got)
	}
}
```

- [ ] **Step 2: Запустить тест и убедиться, что он падает**

Run: `go test ./internal/service/backtest/ -run TestRSIPullbackWUSHTracksBaseline -v`
Expected: FAIL на этапе компиляции — пакет `strategy/wush` не существует.

- [ ] **Step 3: Создать пакет**

Создать `internal/service/trading_strategy/rsi_pullback/strategy/wush/wush.go`:

```go
// Package wush supplies the ticker and starting rsi_pullback Params for WUSH (Whoosh).
//
// Calibration has NOT been run for this ticker: the body returns core.DefaultParams()
// unchanged rather than copying its fields, so a change to the baseline still reaches this
// ticker instead of silently drifting away from it. Once the calibration runs clear the bar
// declared in the spec, replace the body with an explicit literal — from that point the ticker
// must stop tracking the baseline, and TestRSIPullbackWUSHTracksBaseline must be replaced with
// a literal snapshot.
//
// Пакет заведён 2026-08-13 ДО калибровки — как место, куда ляжет литерал, и как носитель
// замеров инструмента. Спека: docs/superpowers/specs/2026-08-13-wush-rsi-pullback-prep-design.md,
// сетки: data/params/rsi_pullback/wush/.
//
// Что известно об инструменте (замеры по кэшу WUSH_Minutes30.json и WUSH_Day1.json,
// правилами ядра: будние дни MSK как weekdayDaily, сглаживание Уайлдера как в pkg/indicators):
//
//   - Истории достаточно: 33 823 30-минутных бара (23 901 будний) за 36.0 месяца с 2023-08-04.
//     Штатный протокол walk-forward §8 docs/rsi_pullback/strategy.md исполним целиком — как у
//     reni и fesh и в отличие от domrf.
//   - ПАДАЮТ ОБА ОКНА ПРОТОКОЛА, и это главное отличие WUSH от предшественников: обучающее окно
//     2023-08-04—2026-02-03 — падение 226.50 → 92.99 (−58.9%), holdout 2026-02-04—2026-08-03 —
//     падение 93.41 → 48.61 (−48.0%). Пик 337.48 (2024-03-25) → минимум 30.30 (2026-07-17), то
//     есть −91.0%; пять полугодий из семи отрицательные. У fesh holdout РОС на 13.1% и завышал
//     long-only механически, поэтому подтверждением служить не мог; здесь завысить результат
//     нечем — это самая честная проверка, какую ставил репозиторий. Обратная сторона: трендовый
//     фильтр в таком режиме держит EMAFast > EMASlow лишь 41–43% времени, вход закрыт большую
//     часть истории, и СЧЁТ СДЕЛОК надо читать раньше profit factor — красивый PF на шести
//     сделках здесь самый вероятный способ обмануться.
//   - Дневной ATR(14) идёт медианой 4.25% цены (среднее 4.52% — колонка скринера печатает
//     именно среднее, распределение скошено вправо хвостом до 13.41%). По ширине рядом с fesh
//     (4.42%) и ugld (4.28%), заметно выше reni (3.36%) и domrf (1.94%). Поэтому сетки в
//     data/params/rsi_pullback/wush/ пересчитаны от fesh/, а сужения domrf в них намеренно НЕ
//     перенесены.
//   - Круг издержек 0.024 дневного ATR; он лицензирует узкую строку стопа 0.3 ATR. Но тот же
//     стоп переживает целиком лишь 2.4% дней, то есть сидит внутри обычного внутридневного шума.
//   - День раскрывается БЫСТРЕЕ, чем у fesh: медиана доли ATR ко второму бару 0.33 против 0.28.
//     Отсюда единственное расхождение сеток с fesh/ — ось FreshDayATR [0, 0.3, 0.4] вместо
//     [0, 0.25, 0.35].
//   - Оборот 356 млн ₽/день при лоте 1 — вторая ликвидность из заведённых после domrf и
//     единственный тикер с лотом 1, то есть самая тонкая гранулярность позиции. На размер
//     позиции не давит.
//   - Слабое место, заявленное до прогонов: в отчёте скринера у WUSH Plateau 58% и Capped 4/24
//     против 83% и 2/24 у fesh. Рабочая зона узкая, и это прямо повышает шанс, что калибровка
//     сядет на случайный пик. Планка объявлена в спеке ДО первого прогона именно поэтому.
package wush

import "tinvest/internal/service/trading_strategy/rsi_pullback/strategy/core"

// Ticker is the instrument this package configures.
const Ticker = "WUSH"

// DefaultParams returns WUSH's starting rsi_pullback parameters (pre-calibration).
func DefaultParams() core.Params {
	return core.DefaultParams()
}
```

- [ ] **Step 4: Зарегистрировать пакет в реестре бэктеста**

В `internal/service/backtest/rsi_pullback_registry.go` добавить импорт `rsipullbackwush "tinvest/internal/service/trading_strategy/rsi_pullback/strategy/wush"` и строку в `rsiPullbackRegistry` (карта отсортирована по алфавиту, `wush` идёт последней):

```go
	rsipullbackugld.Ticker:  rsiPullbackBindingFor(rsipullbackugld.Ticker, rsipullbackugld.DefaultParams),
	rsipullbackwush.Ticker:  rsiPullbackBindingFor(rsipullbackwush.Ticker, rsipullbackwush.DefaultParams),
```

- [ ] **Step 5: Зарегистрировать пакет в живом реестре**

В `internal/service/trading_strategy/rsi_pullback/live/registry.go` добавить импорт `"tinvest/internal/service/trading_strategy/rsi_pullback/strategy/wush"`, строку в карту и поправить doc-комментарий `paramsByTicker`, который сейчас называет тикером без литерала только NVTK:

```go
// paramsByTicker maps every rsi_pullback ticker the runner knows to its params. The
// configured universe (RSI_PULLBACK_TICKERS) selects which of these actually trade;
// NVTK and WUSH are registered for completeness but have no calibrated literal yet — they
// return the baseline — and must not be put into the universe. WUSH was added 2026-08-13 as
// the place its literal will land if the calibration runs clear the bar declared in
// docs/superpowers/specs/2026-08-13-wush-rsi-pullback-prep-design.md. FESH got its literal
// 2026-08-13, which only lifts that mechanical block: the standard §8 protocol does not
// confirm the ticker (see the package doc), so putting it into the universe stays the owner's
// call.
var paramsByTicker = map[string]core.Params{
	ugld.Ticker:  ugld.DefaultParams(),
	tbank.Ticker: tbank.DefaultParams(),
	gazp.Ticker:  gazp.DefaultParams(),
	nvtk.Ticker:  nvtk.DefaultParams(),
	domrf.Ticker: domrf.DefaultParams(),
	reni.Ticker:  reni.DefaultParams(),
	fesh.Ticker:  fesh.DefaultParams(),
	wush.Ticker:  wush.DefaultParams(),
}
```

- [ ] **Step 6: Поправить устаревший комментарий в живом тесте**

В `internal/service/trading_strategy/rsi_pullback/live/registry_test.go` комментарий перед `TestBaselineTrackingTickersStayOutOfTheDefaultUniverse` утверждает «NVTK и RENI зарегистрированы, но возвращают baseline». Это устарело: RENI получил литерал 2026-08-12. Заменить фразу на:

```go
// Обратная сторона предыдущего теста: быть в реестре — не то же самое, что быть готовым к
// торговле. NVTK и WUSH зарегистрированы, но возвращают baseline, то есть параметры, которые
// никогда не проверялись на этих инструментах. Попадание такого тикера в дефолтную вселенную
// означало бы живые сделки по неоткалиброванной конфигурации, и заметить это по коду трудно:
// внешне запись в карте выглядит так же, как у откалиброванного соседа.
```

- [ ] **Step 7: Запустить тесты**

Run: `go test ./internal/service/backtest/ ./internal/service/trading_strategy/rsi_pullback/... -v`
Expected: PASS. Отдельно убедиться, что зелёные `TestRSIPullbackWUSHTracksBaseline`, `TestBaselineTrackingTickersStayOutOfTheDefaultUniverse` (WUSH не в дефолтной вселенной) и `TestRegisteredTickersKeepTheRSIExitArmed` (baseline несёт `UseRSIExit=1`).

- [ ] **Step 8: Мутационная проверка сторожевого теста**

Временно заменить тело `wush.DefaultParams()` на литерал, отличающийся от baseline одним полем:

```go
func DefaultParams() core.Params {
	p := core.DefaultParams()
	p.RSILower = 17
	return p
}
```

Run: `go test ./internal/service/backtest/ -run TestRSIPullbackWUSHTracksBaseline`
Expected: FAIL с «want baseline». Затем вернуть `return core.DefaultParams()` и убедиться, что тест зелёный.

- [ ] **Step 9: Обновить §8.0.1 документации**

В `docs/rsi_pullback/strategy.md`, в таблице состояний §8.0.1, в строку «калибровка не проводилась» добавить `wush` рядом с `nvtk`. В абзаце про исключение «пакет заводят до первого прогона» дописать после предложения про `fesh` (то, что заканчивается «тем же порядком, что и у RENI»):

```
2026-08-13 тем же порядком заведён `wush` (33 823 30-минутных бара за 36.0 месяца, дневной ATR
4.25% медианой, лот 1, оборот 356 млн ₽/день; спека
`docs/superpowers/specs/2026-08-13-wush-rsi-pullback-prep-design.md`), и его состояние держит
`TestRSIPullbackWUSHTracksBaseline`. У этого тикера впервые падают ОБА окна протокола — train
−58.9%, holdout −48.0%, пик-минимум −91.0%, — то есть holdout не завышен механически, как он был
завышен у `fesh` растущим окном. Планка для него объявлена в спеке ДО первого прогона: pooled
OOS PF ≥ 1.5 при ≥ 20 сделках на темах `entry` и `trend` плюс устойчивость выбора ведущей оси в
3 фолдах из 4. Не взята — литерала нет.
```

- [ ] **Step 10: Прогнать полный гейт**

Run: `./bin/mage ci`
Expected: EXIT 0 — линт чист, `go test -race ./...` зелёный, мок-дрейфа нет.

- [ ] **Step 11: Коммит**

```bash
git add internal/service/trading_strategy/rsi_pullback/strategy/wush/ internal/service/backtest/rsi_pullback_registry.go internal/service/backtest/rsi_pullback_registry_test.go internal/service/trading_strategy/rsi_pullback/live/registry.go internal/service/trading_strategy/rsi_pullback/live/registry_test.go docs/rsi_pullback/strategy.md
git commit -m "feat(rsi_pullback): пакет strategy/wush в состоянии «калибровка не проводилась»"
```

---

### Task 4: Прогон девяти тем под rolling walk-forward

Эта задача не пишет кода репозитория — она производит отчёты в `reports/` (каталог в `.gitignore`, коммитить нечего). Порядок против FESH перевёрнут сознательно: walk-forward идёт ПЕРВЫМ как вердикт, а не последним как проверка уже выбранного литерала.

**Files:**
- Create: `reports/WUSH_calib/run_themes.py` (оркестратор прогонов, локальный)
- Создаются прогонами: `reports/WUSH_screen/`, `reports/WUSH_entry/`, `reports/WUSH_trend/`, `reports/WUSH_exit/`, `reports/WUSH_day/`, `reports/WUSH_day_spent/`, `reports/WUSH_volume/`, `reports/WUSH_risk/`, `reports/WUSH_trail/`

**Interfaces:**
- Consumes: девять файлов `data/params/rsi_pullback/wush/*.json` из Task 1 и 2; регистрация тикера из Task 3 (без неё `RSIPullbackLookupOrGeneric` вернул бы generic-биндинг — прогон бы прошёл, но мерил бы не тот пакет).
- Produces: отчёты `*_walkforward.md` в перечисленных каталогах — их читает Task 5.

- [ ] **Step 1: Освежить кэш**

Кэш стоит на 2026-08-04. Один прогон с `-refresh` подтянет свежие бары; параллелить его нельзя.

```bash
go run ./cmd/backtest -ticker WUSH -strategy rsi_pullback -interval Minutes30 \
  -calibrate data/params/rsi_pullback/wush/cal_screen.json -out ./reports/WUSH_screen \
  -months 36 -train-months 12 -test-months 6 -min-trades 1 -metric profit_factor -refresh
```

Expected: прогон завершается, в `./reports/WUSH_screen/` появляются отчёт и `*_walkforward.md`. Проверить в шапке отчёта, что тикер разрешился именно в пакет `wush`, а число баров выросло против 33 823.

- [ ] **Step 2: Написать оркестратор оставшихся восьми тем**

Создать `reports/WUSH_calib/run_themes.py`:

```python
#!/usr/bin/env python3
"""Прогон тем WUSH под rolling walk-forward, не более трёх одновременно.

Лимит в три процесса — не тюнинг, а требование: каждый запуск cmd/backtest делает
load shares и упирается в rate-limit Tinkoff API (RateLimit(10, 62)). Поднимать нельзя.
"""
import subprocess
from concurrent.futures import ThreadPoolExecutor

REPO = "/home/oleg/GolandProjects/tinvest"

# (файл темы, каталог отчёта, -min-trades)
THEMES = [
    ("cal_entry.json", "WUSH_entry", 20),
    ("cal_trend.json", "WUSH_trend", 20),
    ("cal_exit.json", "WUSH_exit", 20),
    ("cal_day.json", "WUSH_day", 20),
    ("cal_day_spent.json", "WUSH_day_spent", 20),
    ("cal_volume.json", "WUSH_volume", 20),
    ("cal_risk.json", "WUSH_risk", 20),
    ("cal_trail.json", "WUSH_trail", 20),
]


def run(theme):
    grid, out, min_trades = theme
    cmd = [
        "go", "run", "./cmd/backtest",
        "-ticker", "WUSH", "-strategy", "rsi_pullback", "-interval", "Minutes30",
        "-calibrate", f"data/params/rsi_pullback/wush/{grid}",
        "-out", f"./reports/{out}",
        "-months", "36", "-train-months", "12", "-test-months", "6",
        "-min-trades", str(min_trades), "-metric", "profit_factor",
    ]
    p = subprocess.run(cmd, cwd=REPO, capture_output=True, text=True)
    status = "ok" if p.returncode == 0 else f"FAILED rc={p.returncode}"
    print(f"{grid}: {status}", flush=True)
    if p.returncode != 0:
        print(p.stderr[-2000:], flush=True)
    return p.returncode


if __name__ == "__main__":
    with ThreadPoolExecutor(max_workers=3) as ex:
        codes = list(ex.map(run, THEMES))
    print("неуспешных тем:", sum(1 for c in codes if c != 0))
```

- [ ] **Step 3: Прогнать восемь тем**

```bash
python3 reports/WUSH_calib/run_themes.py
```

Expected: восемь строк `ok`, итог «неуспешных тем: 0». Прогон длительный — 101 комбинация × 4 фолда.

- [ ] **Step 4: Проверить, что все девять отчётов на месте**

```bash
ls reports/WUSH_{screen,entry,trend,exit,day,day_spent,volume,risk,trail}/*_walkforward.md
```

Expected: девять файлов. Если какой-то отсутствует — тема упала по rate-limit, перезапустить её одну.

---

### Task 5: Разбор прогонов и вердикт по объявленной планке

**Files:**
- Create: `reports/WUSH_calib/VERDICT.md` (локальный разбор, `reports/` в `.gitignore`)

**Interfaces:**
- Consumes: девять `*_walkforward.md` из Task 4.
- Produces: решение «литерал есть» / «литерала нет». Task 6 выполняется ТОЛЬКО при «литерал есть».

- [ ] **Step 1: Свести pooled OOS PF и счёт сделок по девяти темам**

Прочитать девять `*_walkforward.md` и построить таблицу: тема, pooled OOS PF, число сделок в объединении тестовых окон, число вырожденных фолдов (PF в сотнях/тысячах из-за нуля убыточных сделок), выбранное значение ведущей оси по каждому из четырёх фолдов.

Вырожденный фолд считать отдельно и НЕ засчитывать в пользу тикера — при нулевом знаменателе pooled PF раздувается.

- [ ] **Step 2: Применить планку**

Планка из спеки, раздел 5, объявлена до прогонов:

- темы `entry` и `trend` обе показывают pooled OOS PF **≥ 1.5** при **≥ 20 сделках**;
- по ведущей оси темы (`RSILower` для `entry`, `EMASlow` для `trend`) одно и то же значение выбрано **не менее чем в 3 фолдах из 4**.

Для сравнения, FESH на этих же двух темах дал 1.029 и 0.953, а его ведущие оси гуляли 15/25/10/15 и 100/50/150/100 — то есть максимум два совпадения из четырёх. Планка подобрана так, чтобы FESH её не прошёл.

- [ ] **Step 3: Записать вердикт**

Создать `reports/WUSH_calib/VERDICT.md`: таблица из шага 1, применение планки по шагу 2, явный ответ «литерал есть» либо «литерала нет», и — обязательно — счёт сделок рядом с каждым PF. Отдельным абзацем отметить, инертна ли ось `TPDailyATR` (как у FESH, где цель срабатывала 2 раза из 60) и какую долю выходов даёт RSI-выход: это понадобится для доки пакета, если литерал появится.

- [ ] **Step 4: Доложить владельцу**

Сообщить вердикт с числами. Если планка не взята — работа на этом закончена: Task 6 НЕ выполняется, пакет остаётся в состоянии «калибровка не проводилась». Отрицательный результат записывается в doc-комментарий пакета `strategy/wush` и в §8.0.1 отдельным коммитом; это делает шаг 5.

- [ ] **Step 5: Если планка НЕ взята — зафиксировать отрицательный результат**

Дописать в doc-комментарий `wush.go` абзац с числами из `VERDICT.md`: какие pooled OOS PF дали девять тем, на скольких сделках, как гуляли ведущие оси. В §8.0.1 `docs/rsi_pullback/strategy.md` дописать, что WUSH прогнан 2026-08-13 и планку не взял.

```bash
git add internal/service/trading_strategy/rsi_pullback/strategy/wush/wush.go docs/rsi_pullback/strategy.md
git commit -m "docs(rsi_pullback): WUSH прогнан, объявленную планку не взял"
```

После этого коммита план завершён — Task 6 пропускается.

---

### Task 6: Литерал WUSH (выполняется ТОЛЬКО если планка взята в Task 5)

**Files:**
- Modify: `internal/service/trading_strategy/rsi_pullback/strategy/wush/wush.go`
- Create: `internal/service/trading_strategy/rsi_pullback/strategy/wush/wush_test.go`
- Modify: `internal/service/backtest/rsi_pullback_registry_test.go` (замена сторожевого теста)
- Modify: `internal/service/trading_strategy/rsi_pullback/live/registry.go` (комментарий)
- Modify: `internal/service/trading_strategy/rsi_pullback/live/registry_test.go` (комментарий)
- Modify: `docs/rsi_pullback/strategy.md` (§8.0.1)

**Interfaces:**
- Consumes: `wush.Ticker`, `wush.DefaultParams()` из Task 3; вердикт и выбранные значения из Task 5.
- Produces: литерал `core.Params` пакета `wush`.

- [ ] **Step 1: Подтвердить кандидата сеткой из одной точки**

Собрать `reports/WUSH_calib/point.json` — файл, где ВСЕ 18 полей `core.Params` заданы списком из одного значения. Калибратор при этом ничего не выбирает, поэтому profit factor принадлежит конфигурации, а не процедуре переоптимизации. Шаблон ниже полон — подставить в него значения кандидата из Task 5, ничего не удаляя: пропущенное поле молча возьмётся из `core.DefaultParams()`, и точка перестанет быть той конфигурацией, которую отчёт объявляет.

```json
{
  "_comment": "Сетка из ОДНОЙ точки: все 18 полей заданы списком длины 1, поэтому калибратор ничего не выбирает и pooled PF принадлежит самой конфигурации, а не процедуре переоптимизации. Граница приёма: для фиксированной точки это НЕ out-of-sample — конфигурацию выбрал человек, видевший всю историю.",
  "phases": [
    {
      "name": "point",
      "grid": {
        "RSIPeriod": [5],
        "RSILower": [15],
        "RSIUpper": [65],
        "EMAFast": [20],
        "EMASlow": [100],
        "DailyATRPeriod": [14],
        "UseDayATRGate": [1],
        "FreshDayATR": [0.3],
        "SpentDayATR": [0.8],
        "StopDailyATR": [0.5],
        "TPDailyATR": [1.0],
        "UseVolume": [0],
        "VolBaseDays": [14],
        "VolLookbackBars": [3],
        "VolMult": [1.2],
        "UseRSIExit": [1],
        "UseTrail": [0],
        "TrailDailyATR": [0.5]
      }
    }
  ]
}
```

```bash
go run ./cmd/backtest -ticker WUSH -strategy rsi_pullback -interval Minutes30 \
  -calibrate reports/WUSH_calib/point.json -out ./reports/WUSH_point \
  -months 36 -train-months 12 -test-months 6 -min-trades 20 -metric profit_factor
```

Expected: отчёт с pooled PF по четырём фолдам для одной конфигурации. Записать pooled PF, число сделок, win rate, худший фолд и число вырожденных фолдов — эти пять чисел пойдут в doc-комментарий.

Граница приёма, которую обязательно записать в доке: для фиксированной точки это **не** out-of-sample, потому что конфигурацию выбрал человек, видевший всю историю.

- [ ] **Step 2: Написать падающий снимок литерала**

Создать `internal/service/trading_strategy/rsi_pullback/strategy/wush/wush_test.go` по образцу `fesh/fesh_test.go`. Значения ниже — плейсхолдеры формы; подставить фактические из Task 5:

```go
package wush

import (
	"testing"

	"tinvest/internal/service/trading_strategy/rsi_pullback/strategy/core"
)

// TestCalibratedLiteralIsPinned прибивает литерал целиком. Снимок нужен потому, что параметры
// задаются литералом core.Params: забытое или «улучшенное на глаз» поле не ломает компиляцию и
// молча меняет то, чем торгует раннер. Отдельного обоснования требуют поля, которые естественнее
// всего захотеть подправить, — их значения подтверждены прогоном, а не выбраны по вкусу.
func TestCalibratedLiteralIsPinned(t *testing.T) {
	want := core.Params{
		RSIPeriod:       5,
		RSILower:        15,
		RSIUpper:        65,
		EMAFast:         20,
		EMASlow:         100,
		DailyATRPeriod:  14,
		UseDayATRGate:   1,
		FreshDayATR:     0.3,
		SpentDayATR:     0.8,
		StopDailyATR:    0.5,
		TPDailyATR:      1.0,
		UseVolume:       0,
		VolBaseDays:     14,
		VolLookbackBars: 3,
		VolMult:         1.2,
		UseRSIExit:      1,
		UseTrail:        0,
		TrailDailyATR:   0.5,
	}
	if got := DefaultParams(); got != want {
		t.Fatalf("DefaultParams() = %+v, want %+v", got, want)
	}
}

// Связь с baseline разорвана осознанно: правка core.DefaultParams() не должна доходить до
// откалиброванного тикера.
func TestParamsDoNotTrackTheBaseline(t *testing.T) {
	if DefaultParams() == core.DefaultParams() {
		t.Fatal("WUSH вернул baseline: откалиброванный тикер обязан иметь собственный литерал")
	}
}

// Ловушка нулевого значения: забытое поле даёт StopDailyATR=0, то есть позицию без стопа.
func TestStopIsArmed(t *testing.T) {
	if got := DefaultParams().StopDailyATR; got <= 0 {
		t.Fatalf("StopDailyATR = %v, want > 0 — позиция без стопа", got)
	}
}

// Та же ловушка для основного выхода.
func TestRSIExitIsArmed(t *testing.T) {
	if got := DefaultParams().UseRSIExit; got != 1 {
		t.Fatalf("UseRSIExit = %d, want 1", got)
	}
}
```

- [ ] **Step 3: Запустить и убедиться, что падает**

Run: `go test ./internal/service/trading_strategy/rsi_pullback/strategy/wush/ -v`
Expected: FAIL — `DefaultParams()` пока возвращает baseline.

- [ ] **Step 4: Поставить литерал**

Заменить тело `DefaultParams()` на литерал из шага 2 и переписать шапку пакета: строку «Calibration has NOT been run» — на описание состояния «откалиброван», с пятью числами из шага 1 (pooled PF, сделки, win rate, худший фолд, вырожденные фолды), с границей приёма «это не out-of-sample» и с механикой прибыли из `VERDICT.md`. Замеры инструмента в шапке сохранить: `reports/` в репозиторий не попадает, и все числа обязаны жить в коде.

- [ ] **Step 5: Заменить сторожевой тест на снимок**

В `internal/service/backtest/rsi_pullback_registry_test.go` удалить `TestRSIPullbackWUSHTracksBaseline` и написать вместо него `TestRSIPullbackWUSHIsRegisteredAndCalibrated` по образцу `TestRSIPullbackFESHIsRegisteredAndCalibrated`: тикер есть в `rsiPullbackRegistry`, `p != core.DefaultParams()`, `p == rsipullbackwush.DefaultParams()`, `b.Build(p).Ticker() == "WUSH"`.

- [ ] **Step 6: Обновить комментарии реестров и §8.0.1**

В `live/registry.go` и `live/registry_test.go` убрать WUSH из перечня тикеров без литерала (останется NVTK). В §8.0.1 перенести `wush` из строки «калибровка не проводилась» в строку «откалиброван» с числами прогона, и дописать в абзац про исключение, что сторожевой тест заменён на снимок.

- [ ] **Step 7: Мутационная проверка снимка**

Поменять в `wush.go` одно поле литерала (например `SpentDayATR` на 1.0) и убедиться, что краснеют оба теста — `TestCalibratedLiteralIsPinned` и `TestRSIPullbackWUSHIsRegisteredAndCalibrated`. Вернуть значение.

- [ ] **Step 8: Прогнать полный гейт**

Run: `./bin/mage ci`
Expected: EXIT 0.

- [ ] **Step 9: Коммит**

```bash
git add internal/service/trading_strategy/rsi_pullback/strategy/wush/ internal/service/backtest/rsi_pullback_registry_test.go internal/service/trading_strategy/rsi_pullback/live/registry.go internal/service/trading_strategy/rsi_pullback/live/registry_test.go docs/rsi_pullback/strategy.md
git commit -m "feat(rsi_pullback): WUSH откалиброван — литерал вместо отслеживания baseline"
```

- [ ] **Step 10: Остановиться**

Ввод WUSH в `RSI_PULLBACK_TICKERS` и правка `env/*.env` в этот план не входят при любом исходе. Доложить владельцу и ждать его решения.
