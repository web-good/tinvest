# RENI под rsi_pullback — план подготовки к калибровке

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Завести каталог калибровочных сеток `data/params/rsi_pullback/reni/` с осями, обоснованными замерами Ренессанс Страхования, и сторожевые тесты, которые эти оси удерживают.

**Architecture:** Только данные и тесты. Пакет `strategy/reni` НЕ создаётся: `RSIPullbackLookupOrGeneric` (`internal/service/backtest/rsi_pullback_registry.go:48`) отдаёт незарегистрированному тикеру generic-биндинг с `core.DefaultParams()` — ровно то, что вернул бы baseline-tracking пакет, поэтому прогоны по сеткам работают без единой строки кода стратегии. Девять однотемных JSON-файлов повторяют структуру `data/params/rsi_pullback/ugld/`; каждое отклонение оси от UGLD прибито тестом с замером в тексте ошибки.

**Tech Stack:** Go 1.25, `go test`, `./bin/mage ci`, JSON-сетки, `cmd/backtest`.

## Global Constraints

- Спека: `docs/superpowers/specs/2026-08-10-reni-rsi-pullback-prep-design.md`. Все числа осей берутся оттуда.
- Каждый `cal_*.json` обязан нести `_comment`, содержащий подстроку `reni/<имя файла>` — это проверяет `TestRSIPullbackCalFilesValid` (`internal/service/backtest/rsi_pullback_grid_test.go:71`). Команда запуска внутри `_comment` должна называть тот же самый файл.
- `StopDailyATR = 0` не появляется ни в одном файле — калибровка не имеет права отключить стоп.
- Каждая фаза непустая; все свипуемые поля обязаны резолвиться через `applyField`.
- В `cal_risk.json` хотя бы одна цель обязана превышать самый широкий стоп того же файла (`TestRSIPullbackGridControlPoints`): при стопах до 1.3 это обеспечивают цели 1.5, 2.0, 2.5.
- Тексты `_comment` и сообщений об ошибках в тестах — на русском, как в `data/params/rsi_pullback/domrf/` и `rsi_pullback_domrf_grid_test.go`.
- Замеры RENI, на которые ссылаются обоснования: дневной ATR(14) медиана **3.36%**; круг издержек 0.1% = **0.030 ATR**; медианный дневной размах **0.88 ATR**; доли дней ≥0.6 — 79.4%, ≥0.8 — 59.1%, ≥1.0 — 35.7%, ≥1.25 — 21.7%, ≥1.5 — 13.1%, ≥1.75 — 7.3%; прогресс размаха 07:00 — 0.29 ATR, 10:00 — 0.49, 13:00 — 0.70, 16:00 — 0.82, 19:00 — 0.92; будние кроссы RSI вниз: RSI(4)@10 234, RSI(4)@15 521, RSI(5)@10 107, RSI(6)@10 51, RSI(6)@15 175, RSI(7)@10 23; доли баров, проходящих объёмный гейт: ≥1.2 — 31.1%, ≥1.5 — 24.7%, ≥2.0 — 17.6%, ≥2.5 — 13.3%; оборот 91 млн ₽/день; 23 071 будний 30-минутный бар за 35.9 месяца.
- Команды прогонов в `_comment`: `-interval Minutes30 -months 36 -test-months 6 -min-trades 20 -metric profit_factor`; исключение — `cal_screen.json` с `-min-trades 1`.

## File Structure

| Файл | Ответственность |
|---|---|
| `internal/service/backtest/rsi_pullback_grid_test.go` | **Modify.** Добавляется общий хелпер `rsiPullbackTickerGrid`; `domrfGrid` становится его обёрткой (появился второй потребитель). |
| `internal/service/backtest/rsi_pullback_domrf_grid_test.go` | **Modify.** Тело `domrfGrid` заменяется вызовом общего хелпера. Тесты DOMRF не меняются. |
| `internal/service/backtest/rsi_pullback_reni_grid_test.go` | **Create.** Хелпер `reniGrid` и два сторожевых теста: сигнальные оси и риск/гейтовые. |
| `data/params/rsi_pullback/reni/cal_screen.json` | **Create.** Цена двух гейтов в сделках. |
| `data/params/rsi_pullback/reni/cal_entry.json` | **Create.** Форма отката: период RSI × глубина. |
| `data/params/rsi_pullback/reni/cal_trend.json` | **Create.** Трендовый фильтр: пара EMA. |
| `data/params/rsi_pullback/reni/cal_day.json` | **Create.** Обе ветки дневного гейта совместно. |
| `data/params/rsi_pullback/reni/cal_day_spent.json` | **Create.** Только ветка «день исчерпан», ось шире. |
| `data/params/rsi_pullback/reni/cal_volume.json` | **Create.** Объёмный гейт: множитель × база. |
| `data/params/rsi_pullback/reni/cal_risk.json` | **Create.** Стоп × цель в единицах дневного ATR. |
| `data/params/rsi_pullback/reni/cal_exit.json` | **Create.** Полоса выхода по RSI. |
| `data/params/rsi_pullback/reni/cal_trail.json` | **Create.** Трейл и его взаимодействие с RSI-выходом. |

---

### Task 1: Общий хелпер и сигнальные сетки

**Files:**
- Modify: `internal/service/backtest/rsi_pullback_grid_test.go` (после `rsiPullbackPhases`, примерно строка 34)
- Modify: `internal/service/backtest/rsi_pullback_domrf_grid_test.go:9-22`
- Create: `internal/service/backtest/rsi_pullback_reni_grid_test.go`
- Create: `data/params/rsi_pullback/reni/cal_screen.json`
- Create: `data/params/rsi_pullback/reni/cal_entry.json`
- Create: `data/params/rsi_pullback/reni/cal_trend.json`

**Interfaces:**
- Consumes: `rsiPullbackParamsDir` (константа, `= "../../../data/params/rsi_pullback"`), `rsiPullbackPhases(t *testing.T, path string) []Phase` — оба уже есть в `rsi_pullback_grid_test.go`.
- Produces: `rsiPullbackTickerGrid(t *testing.T, ticker, file string) map[string][]float64` — сливает оси всех фаз одного файла в карту `поле → значения`; используется в Task 2. `reniGrid(t *testing.T, file string) map[string][]float64` — обёртка над ним с `ticker = "reni"`, используется в Task 2.

- [ ] **Step 1: Написать общий хелпер и переключить на него DOMRF**

В `internal/service/backtest/rsi_pullback_grid_test.go` сразу после функции `rsiPullbackPhases` добавить:

```go
// rsiPullbackTickerGrid читает один файл сеток тикера и сливает оси всех его фаз в одну карту
// «поле → значения». Файлы каталога однотемные, поэтому слияние не теряет информации, а
// сторожевым тестам не приходится знать имя фазы.
func rsiPullbackTickerGrid(t *testing.T, ticker, file string) map[string][]float64 {
	t.Helper()
	path := filepath.Join(rsiPullbackParamsDir, ticker, file)
	out := make(map[string][]float64)
	for _, ph := range rsiPullbackPhases(t, path) {
		for field, values := range ph.Grid {
			out[field] = append(out[field], values...)
		}
	}
	if len(out) == 0 {
		t.Fatalf("%s/%s: сетка пуста", ticker, file)
	}
	return out
}
```

В `internal/service/backtest/rsi_pullback_domrf_grid_test.go` заменить тело `domrfGrid` (строки 9-22) на обёртку, а импорт `path/filepath` из этого файла убрать — он больше не нужен:

```go
package backtest

import "testing"

// domrfGrid читает файл сеток DOMRF. Общий хелпер живёт в rsi_pullback_grid_test.go: у него
// появился второй потребитель (reni/), и копия разъехалась бы при первой же правке.
func domrfGrid(t *testing.T, file string) map[string][]float64 {
	t.Helper()
	return rsiPullbackTickerGrid(t, "domrf", file)
}
```

- [ ] **Step 2: Убедиться, что тесты DOMRF всё ещё зелёные**

Run: `go test ./internal/service/backtest/ -run 'TestDOMRF' -v 2>&1 | tail -5`
Expected: PASS обоих тестов `TestDOMRFSignalGridsPinTheirMeasuredAxes` и `TestDOMRFRiskGridsPinTheirMeasuredAxes` — рефакторинг хелпера не должен менять их поведение.

- [ ] **Step 3: Написать падающий тест сигнальных осей**

Создать `internal/service/backtest/rsi_pullback_reni_grid_test.go`:

```go
package backtest

import "testing"

// reniGrid читает файл сеток RENI через общий хелпер.
func reniGrid(t *testing.T, file string) map[string][]float64 {
	t.Helper()
	return rsiPullbackTickerGrid(t, "reni", file)
}

// TestRENISignalGridsPinTheirMeasuredAxes сторожит оси, обоснованные замерами инструмента, а не
// вкусом. Каталог reni/ заводится копированием структуры ugld/, и типовая ошибка такой копии —
// притащить чужие оси целиком. По ширине RENI действительно сосед UGLD (дневной ATR 3.36% против
// 4.28%), поэтому здесь опасна ОБРАТНАЯ ошибка: перенести сужения, сделанные для DOMRF, где
// ATR 1.94% и сигналов дефицит.
func TestRENISignalGridsPinTheirMeasuredAxes(t *testing.T) {
	screen := reniGrid(t, "cal_screen.json")
	for _, field := range []string{"UseDayATRGate", "UseVolume"} {
		if got := screen[field]; len(got) != 2 {
			t.Errorf("cal_screen.json: %s = %v, want обе точки [0,1] — тема меряет цену каждого гейта в сделках", field, got)
		}
	}

	entry := reniGrid(t, "cal_entry.json")
	// Глубже 25 порог перестаёт отбирать откат: RSI(4) уходит под 30 1765 раз за историю,
	// это обычный шум, а не сетап.
	for _, v := range entry["RSILower"] {
		if v > 25 {
			t.Errorf("cal_entry.json свипует RSILower=%v: выше 25 порог перестаёт отбирать откат (1765 кроссов под 30)", v)
		}
	}
	// Уровень 10 обязан остаться. Скринер выбрал для RENI лучшей конфигурацией RSI 6/10, и
	// 51 будний кросс RSI(6)@10 эту точку выдерживает. На DOMRF таких кроссов было 18, и там
	// уровень 10 вырезали — при копировании оттуда сужение легко притащить по ошибке.
	hasDeep := false
	for _, v := range entry["RSILower"] {
		if v == 10 {
			hasDeep = true
		}
	}
	if !hasDeep {
		t.Errorf("cal_entry.json: RSILower = %v, не содержит 10 — основную гипотезу скринера", entry["RSILower"])
	}
	// RSIUpper здесь не свипуется: 4x4x5 = 80 комбинаций на выборке около 36 сделок —
	// переобучение по построению. Полоса выхода меряется отдельно, файлом cal_exit.json.
	if got := entry["RSIUpper"]; len(got) != 0 {
		t.Errorf("cal_entry.json свипует RSIUpper=%v: полоса выхода принадлежит cal_exit.json", got)
	}

	trend := reniGrid(t, "cal_trend.json")
	// EMASlow=200 исполнима: Lookback при ней равен 2*200+20 = 420 баров против 23071 будних
	// в кэше, то есть окно прогрева занимает 1.8% истории.
	for _, v := range trend["EMASlow"] {
		if v > 200 {
			t.Errorf("cal_trend.json свипует EMASlow=%v: окно прогрева растёт быстрее, чем польза от сглаживания", v)
		}
	}
	// Быстрая EMA обязана быть быстрее самой быстрой из медленных, иначе строка сетки означает
	// фильтр, который никогда не пропускает вход.
	for _, v := range trend["EMAFast"] {
		if v >= 50 {
			t.Errorf("cal_trend.json свипует EMAFast=%v: минимум оси EMASlow равен 50, такая пара мертва", v)
		}
	}
}
```

- [ ] **Step 4: Запустить тест и убедиться, что он падает**

Run: `go test ./internal/service/backtest/ -run TestRENISignalGrids -v 2>&1 | tail -10`
Expected: FAIL — `read ../../../data/params/rsi_pullback/reni/cal_screen.json: no such file or directory`. Падение именно на отсутствии файла подтверждает, что тест смотрит в нужный каталог.

- [ ] **Step 5: Создать `cal_screen.json`**

```json
{
  "_comment": "SCREEN: цена двух опциональных гейтов в сделках, 4 прогона. Тема отвечает на один вопрос — сколько сделок остаётся, когда включён дневной гейт и объёмный гейт, — и запускается ПЕРВОЙ, до всех остальных тем. Порог -min-trades 1 обязателен именно здесь: при штатных 20 отфильтруются ровно те строки, которые тема измеряет, и лидерборд окажется пустым. Читать надо колонку сделок, а не profit factor: если при обоих включённых гейтах остаётся меньше 15 сделок за 36 месяцев, дальнейшие темы будут калибровать отдельные сделки, а не конфигурации. Замеры RENI, задающие ожидания: будних 30-минутных баров 23071 за 35.9 месяца, кроссов RSI(6) вниз через 15 — 175, дневной гейт при SpentDayATR=1.0 пропускает 35.7% дней, объёмный при VolMult=1.5 — 24.7% баров. Запуск: go run ./cmd/backtest -ticker RENI -strategy rsi_pullback -interval Minutes30 -calibrate data/params/rsi_pullback/reni/cal_screen.json -out ./reports/RENI_screen -months 36 -min-trades 1 -metric profit_factor.",
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

- [ ] **Step 6: Создать `cal_entry.json`**

```json
{
  "_comment": "ENTRY, форма отката: RSIPeriod x RSILower, 16 прогонов. Ось шире, чем у domrf/cal_entry.json (12 прогонов), и уже, чем у ugld/cal_entry.json (80): сигналов у RENI изобилие, но выборка сделок конечна. Кроссы RSI вниз через уровень, будние бары за всю историю (35.9 мес): RSI(4) 234 через 10, 521 через 15, 865 через 20, 1310 через 25; RSI(5) 107/299/561/927; RSI(6) 51/175/397/669; RSI(7) 23/112/277/517. Уровень 10 ОСТАВЛЕН, в отличие от domrf/, где его вырезали при 18 кроссах: здесь их 51 у RSI(6), а скринер выбрал для RENI лучшей конфигурацией именно RSI 6/10 — эта точка и есть основная гипотеза темы. Угол RSI(7)@10 с его 23 кроссами заведомо мёртв и будет отфильтрован порогом -min-trades; одна мёртвая строка из шестнадцати дешевле, чем резать ось ради неё. Уровень 30 не берётся: RSI(4) уходит под него 1765 раз, это шум, а не откат. RSIUpper здесь НЕ свипуется, в отличие от ugld/: там полоса выхода едет вместе с глубиной входа, и это оправдано изобилием сигналов, а здесь 4x4x5 степеней свободы на выборке около 36 сделок — переобучение по построению. Полоса выхода меряется отдельно, файлом cal_exit.json. Запуск: go run ./cmd/backtest -ticker RENI -strategy rsi_pullback -interval Minutes30 -calibrate data/params/rsi_pullback/reni/cal_entry.json -out ./reports/RENI_entry -months 36 -min-trades 20 -test-months 6 -metric profit_factor.",
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

- [ ] **Step 7: Создать `cal_trend.json`**

```json
{
  "_comment": "TREND: EMAFast x EMASlow, 16 прогонов. Ось совпадает с ugld/cal_trend.json, и это единственная тема, где полная копия оправдана: периоды EMA задаются в барах, а не в единицах цены, поэтому разница в ширине инструментов (ATR 3.36% у RENI против 4.28% у UGLD) на них не влияет. Проверено, что EMASlow=200 исполнима: Lookback при ней равен 2*200+20 = 420 баров против 23071 будних в кэше, то есть окно прогрева занимает 1.8% истории. Скринер на своей фиксированной сетке выбрал для RENI пару EMA 20/100 — она и есть основная гипотеза темы, остальные строки её проверяют. Отдельно стоит смотреть на пары с EMASlow=200 в окне 2025H2-2026H1: это единственный на всей истории продолжительный нисходящий режим (пик 141.06 от 2025-03-20, минимум 63.72 от 2026-07-20, просадка 54.8%), и медленная пара обязана выключать вход именно там. Если 20/200 и 20/50 дают близкий profit factor, трендовый гейт на этих данных не работает вовсе, и это надо знать до настройки риска. Запуск: go run ./cmd/backtest -ticker RENI -strategy rsi_pullback -interval Minutes30 -calibrate data/params/rsi_pullback/reni/cal_trend.json -out ./reports/RENI_trend -months 36 -min-trades 20 -test-months 6 -metric profit_factor.",
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

- [ ] **Step 8: Запустить тест и убедиться, что он проходит**

Run: `go test ./internal/service/backtest/ -run 'TestRENISignalGrids|TestRSIPullbackCalFilesValid|TestRSIPullbackGridControlPoints' -v 2>&1 | tail -20`
Expected: PASS. `TestRSIPullbackCalFilesValid` должен показать подтесты `reni/cal_screen.json`, `reni/cal_entry.json`, `reni/cal_trend.json` — это подтверждает, что `_comment` каждого файла называет свой собственный путь.

- [ ] **Step 9: Коммит**

```bash
git add internal/service/backtest/rsi_pullback_grid_test.go \
        internal/service/backtest/rsi_pullback_domrf_grid_test.go \
        internal/service/backtest/rsi_pullback_reni_grid_test.go \
        data/params/rsi_pullback/reni/cal_screen.json \
        data/params/rsi_pullback/reni/cal_entry.json \
        data/params/rsi_pullback/reni/cal_trend.json
git commit -m "test(rsi_pullback): сигнальные сетки RENI и сторож на их оси

Уровень RSILower=10 оставлен вопреки сужению, сделанному для DOMRF:
у RENI 51 будний кросс RSI(6)@10 против 18 у DOMRF, и скринер выбрал
именно RSI 6/10 лучшей конфигурацией. RSIUpper из темы входа убран —
80 комбинаций на выборке около 36 сделок переобучаются по построению,
полоса выхода меряется отдельным cal_exit.json.

Хелпер чтения сеток поднят в общий файл: у него появился второй
потребитель, и копия разъехалась бы при первой правке."
```

---

### Task 2: Сетки риска и гейтов

**Files:**
- Modify: `internal/service/backtest/rsi_pullback_reni_grid_test.go` (дописать второй тест в конец файла)
- Create: `data/params/rsi_pullback/reni/cal_day.json`
- Create: `data/params/rsi_pullback/reni/cal_day_spent.json`
- Create: `data/params/rsi_pullback/reni/cal_volume.json`
- Create: `data/params/rsi_pullback/reni/cal_risk.json`
- Create: `data/params/rsi_pullback/reni/cal_exit.json`
- Create: `data/params/rsi_pullback/reni/cal_trail.json`

**Interfaces:**
- Consumes: `reniGrid(t *testing.T, file string) map[string][]float64` из Task 1.
- Produces: ничего для последующих задач — Task 3 только запускает проверки.

- [ ] **Step 1: Написать падающий тест риск-осей**

Дописать в конец `internal/service/backtest/rsi_pullback_reni_grid_test.go`:

```go
// TestRENIRiskGridsPinTheirMeasuredAxes сторожит оси стопа, цели и обоих гейтов. Каждое число
// ниже опирается на замер по кэшу RENI, а не на перенос с соседнего тикера: дневной ATR 3.36%,
// круг издержек 0.030 ATR, медианный дневной размах 0.88 ATR.
func TestRENIRiskGridsPinTheirMeasuredAxes(t *testing.T) {
	day := reniGrid(t, "cal_day.json")
	if got := day["UseDayATRGate"]; len(got) != 1 || got[0] != 1 {
		t.Errorf("cal_day.json: UseDayATRGate = %v, want ровно [1] — цена отключения гейта меряется в cal_screen.json", got)
	}
	// К 07:00 MSK медианный день уже прошёл 0.29 ATR, к 10:00 — 0.49. Пороги 0.1-0.2 из ugld/
	// отсекают медианный день на первом же баре: ветка «день только начался» становится мёртвой.
	for _, v := range day["FreshDayATR"] {
		if v > 0 && v < 0.25 {
			t.Errorf("cal_day.json свипует FreshDayATR=%v: к 07:00 медианный день прошёл 0.29 ATR, порог мёртв", v)
		}
	}
	// Медианный день RENI покрывает 0.88 ATR, и порога 0.6 достигают 79.4% дней — на этом
	// инструменте он перестаёт быть гейтом.
	for _, v := range day["SpentDayATR"] {
		if v < 0.8 {
			t.Errorf("cal_day.json свипует SpentDayATR=%v: порог достижим для 79%% дней, это не гейт", v)
		}
	}
	// RSILower в этой фазе не свипуется: у ugld/ он раздувает тему до 60 прогонов, а глубина
	// отката принадлежит cal_entry.json. Тема обязана остаться однотемной.
	if got := day["RSILower"]; len(got) != 0 {
		t.Errorf("cal_day.json свипует RSILower=%v: глубина отката принадлежит cal_entry.json", got)
	}

	spent := reniGrid(t, "cal_day_spent.json")
	if got := spent["FreshDayATR"]; len(got) != 1 || got[0] != 0 {
		t.Errorf("cal_day_spent.json: FreshDayATR = %v, want ровно [0] — ветка «день начался» выключена целиком", got)
	}
	// Левый край 0.6 стоит контрольной строкой «гейт почти выключен» (79.4% дней). Точки
	// 0.4-0.5 из ugld/ на RENI не гейтят вовсе и заняли бы прогоны впустую.
	for _, v := range spent["SpentDayATR"] {
		if v < 0.6 {
			t.Errorf("cal_day_spent.json свипует SpentDayATR=%v: ниже 0.6 порог не гейтит (0.6 достигают 79%% дней)", v)
		}
	}

	vol := reniGrid(t, "cal_volume.json")
	if got := vol["UseVolume"]; len(got) != 1 || got[0] != 1 {
		t.Errorf("cal_volume.json: UseVolume = %v, want ровно [1] — точка «гейт выключен» принадлежит cal_screen.json", got)
	}
	// Все четыре точки живые: гейт проходят 31.1% баров при 1.2, 24.7% при 1.5, 17.6% при 2.0,
	// 13.3% при 2.5. Выше 2.5 остаётся меньше восьмой части баров, и объёмный гейт начинает
	// резать выборку сильнее, чем несёт информации.
	for _, v := range vol["VolMult"] {
		if v > 2.5 {
			t.Errorf("cal_volume.json свипует VolMult=%v: выше 2.5 гейт проходит меньше 13%% баров", v)
		}
	}
	// Только две базы, 5 и 10: короткая быстрее реагирует на смену активности, длинная
	// устойчивее к одиночному всплеску. База 3 из ugld/ ловит один выброс объёма, база 14 —
	// размывает его; на вторичном гейте лишние степени свободы не окупаются.
	for _, v := range vol["VolBaseDays"] {
		if v != 5 && v != 10 {
			t.Errorf("cal_volume.json свипует VolBaseDays=%v: обоснованы только 5 и 10 (3 ловит одиночный всплеск, 14 размывает)", v)
		}
	}

	risk := reniGrid(t, "cal_risk.json")
	// Круг издержек стоит 0.030 дневного ATR: на стопе 0.3 ATR (= 1.01% цены) комиссия съедает
	// 10%. На DOMRF та же строка стоила 17% и была оттуда вырезана — при копировании сеток это
	// сужение легко притащить по ошибке, поэтому присутствие строки проверяется явно.
	hasTightStop := false
	for _, v := range risk["StopDailyATR"] {
		if v == 0.3 {
			hasTightStop = true
		}
	}
	if !hasTightStop {
		t.Errorf("cal_risk.json: StopDailyATR = %v, не содержит 0.3 (издержки 0.030 ATR за круг эту строку лицензируют)", risk["StopDailyATR"])
	}
	// Верх оси 1.3: медианный день покрывает 0.88 ATR, такой стоп переживает целиком 87% дней.
	// Шире — это уже не стоп, а отсутствие стопа, оплаченное размером позиции.
	for _, v := range risk["StopDailyATR"] {
		if v > 1.3 {
			t.Errorf("cal_risk.json свипует StopDailyATR=%v: шире 1.3 ATR стоп перестаёт быть стопом (медианный день 0.88 ATR)", v)
		}
	}

	exit := reniGrid(t, "cal_exit.json")
	// Это единственное место, где меряется полоса выхода: cal_entry.json её намеренно не свипует.
	if len(exit["RSIUpper"]) < 5 {
		t.Errorf("cal_exit.json: RSIUpper = %v, want полную ось 55..80 — cal_entry.json полосу выхода не свипует", exit["RSIUpper"])
	}

	trail := reniGrid(t, "cal_trail.json")
	if got := trail["UseTrail"]; len(got) != 1 || got[0] != 1 {
		t.Errorf("cal_trail.json: UseTrail = %v, want ровно [1] — тема меряет форму трейла, а не факт включения", got)
	}
	if len(trail["UseRSIExit"]) != 2 {
		t.Errorf("cal_trail.json: UseRSIExit = %v, want обе точки [0,1] — трейл и RSI-выход конкурируют за одну сделку", trail["UseRSIExit"])
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

Run: `go test ./internal/service/backtest/ -run TestRENIRiskGrids -v 2>&1 | tail -10`
Expected: FAIL — `read ../../../data/params/rsi_pullback/reni/cal_day.json: no such file or directory`.

- [ ] **Step 3: Создать `cal_day.json`**

```json
{
  "_comment": "DAY: обе ветки дневного гейта совместно, 12 прогонов. Гейт двусторонний — вход разрешён либо когда день ещё не раскрылся (размах в пределах FreshDayATR), либо когда он уже исчерпан (размах достиг SpentDayATR); полоса между ними отвергается. Ось FreshDayATR [0, 0.25, 0.35] отличается от ugld/ [0, 0.1, 0.2, 0.3] по замеру внутридневного прогресса: медиана доли ATR, пройденной к концу слота (будние бары), составляет 07:00 — 0.29, 10:00 — 0.49, 13:00 — 0.70, 16:00 — 0.82, 19:00 — 0.92. Порог 0.2 отсекает медианный день уже на первом баре, то есть ветка «день только начался» при нём мертва; 0.25-0.35 оставляют ей утреннее окно. Ноль в оси выключает ветку целиком и служит контролем. Ось SpentDayATR [0.8 ... 1.5] сдвинута вверх относительно ugld/ [0.5 ... 1.2]: медианный день RENI покрывает 0.88 ATR против 0.67 у UGLD, и доля дней, достигающих порога, равна 79.4% при 0.6, 59.1% при 0.8, 35.7% при 1.0, 21.7% при 1.25, 13.1% при 1.5. Порог 0.6 пропустил бы четыре дня из пяти и перестал быть гейтом. RSILower, который у ugld/ раздувает эту тему до 60 прогонов, здесь не свипуется: глубина отката принадлежит cal_entry.json, тема обязана остаться однотемной. Запуск: go run ./cmd/backtest -ticker RENI -strategy rsi_pullback -interval Minutes30 -calibrate data/params/rsi_pullback/reni/cal_day.json -out ./reports/RENI_day -months 36 -min-trades 20 -test-months 6 -metric profit_factor.",
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

```json
{
  "_comment": "DAY-SPENT: только ветка «день исчерпан», 6 прогонов. FreshDayATR=0 выключает ветку «день только начался» целиком (dayStateOK защищает её условием fresh > 0, поэтому ровно ноль убирает её), и остаётся чистый свип нижней границы позднего входа. Разделять ветки нужно потому, что это две разные стратегии на одном коде — ранний вход по тренду против возврата после распродажи, — и общий profit factor их усредняет: прибыльная ветка оплачивает убыточную, а лидерборд cal_day.json этого не показывает. Ось шире, чем в cal_day.json, и уходит до 1.75, потому что здесь порог не конкурирует за прогоны с утренней веткой. Доля будних дней, достигающих порога (n=1001): 0.6 — 79.4%, 0.8 — 59.1%, 1.0 — 35.7%, 1.25 — 21.7%, 1.5 — 13.1%, 1.75 — 7.3%. Строка 0.6 оставлена именно как контроль «гейт почти выключен»: если она выигрывает, ветка «день исчерпан» на RENI не несёт информации. Правый край 1.75 отбирает семь процентов дней — его profit factor читать нельзя, только счёт сделок. Точки 0.4-0.5 из ugld/ не переносятся: на RENI они не гейтят вовсе. Запуск: go run ./cmd/backtest -ticker RENI -strategy rsi_pullback -interval Minutes30 -calibrate data/params/rsi_pullback/reni/cal_day_spent.json -out ./reports/RENI_day_spent -months 36 -min-trades 20 -test-months 6 -metric profit_factor.",
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
  "_comment": "VOLUME: объёмный гейт, 8 прогонов. Гейт требует, чтобы хотя бы один из последних VolLookbackBars будних баров нёс объём в VolMult раз выше среднего ДЛЯ СВОЕГО СЛОТА за последние VolBaseDays будних дней — сравнение со слотом, а не с плоским средним, обязательно, потому что 30-минутный объём U-образен и плоская база мерила бы время суток вместо активности. Замер по кэшу RENI (n=22976 будних баров): отношение объёма бара к слотовой базе за 5 дней имеет медиану 0.69, p75 1.49, p90 3.08; гейт проходят 37.0% баров при 1.0, 31.1% при 1.2, 24.7% при 1.5, 17.6% при 2.0, 13.3% при 2.5. Все четыре точки оси живые, поэтому верхняя граница здесь 2.5, а не 2.0 как у domrf/, где выборка дефицитная. База ограничена парой [5, 10]: короткая быстрее реагирует на смену активности, длинная устойчивее к одиночному всплеску, а точки 3 и 14 из ugld/ отвергнуты по существу — база в три дня ловит один выброс объёма и превращает гейт в лотерею, база в четырнадцать размывает его до бесполезности. Точка «гейт выключен» принадлежит cal_screen.json и здесь отсутствует намеренно. Запуск: go run ./cmd/backtest -ticker RENI -strategy rsi_pullback -interval Minutes30 -calibrate data/params/rsi_pullback/reni/cal_volume.json -out ./reports/RENI_volume -months 36 -min-trades 20 -test-months 6 -metric profit_factor.",
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
  "_comment": "RISK: стоп и цель, оба в единицах дневного ATR, оба замораживаются на входе, 25 прогонов. Дневной ATR(14) у RENI идёт медианой 3.36% цены (p10 1.88, p25 2.77, p75 4.02, p90 4.82) — инструмент той же ширины, что UGLD (4.28%), и вдвое шире DOMRF (1.94%), поэтому ось наследуется от ugld/, а не от domrf/. Строка стопа 0.3 СОХРАНЕНА и это осознанно: круг издержек (0.05% комиссии за сторону, тик при цене около 78 руб пренебрежим) стоит 0.1% оборота, то есть 0.030 дневного ATR, и на стопе 0.3 ATR (= 1.01% цены) издержки съедают 10% риска. На DOMRF та же строка стоила 17% и была оттуда вырезана — копировать это сужение сюда нельзя. Безопасной строка от этого не становится: медианный день покрывает 0.88 ATR, так что стоп 0.3 ATR сидит внутри обычного внутридневного шума и будет снят сносом цены, а не провалом сетапа. Читать долю выходов по стопу, а не только profit factor. Верх оси стопа 1.3: такой стоп переживает целиком 87% дней. Цель доходит до 2.5 (около 8.4% цены) и свипуется в том числе НИЖЕ самого широкого стопа: цель меньше стопа требует win rate выше 50% просто чтобы выйти в ноль, и эта тема подтверждает или убивает такую асимметрию. StopDailyATR=0 намеренно ОТСУТСТВУЕТ и не должен появиться: позиция, которую держат через ночи и выходные без стопа, — не та конфигурация, которую калибровка вправе выбрать. Следить за средним временем удержания: стоп 1.3 с целью 2.5 превращает стратегию в многонедельную. Запуск: go run ./cmd/backtest -ticker RENI -strategy rsi_pullback -interval Minutes30 -calibrate data/params/rsi_pullback/reni/cal_risk.json -out ./reports/RENI_risk -months 36 -min-trades 20 -test-months 6 -metric profit_factor.",
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

```json
{
  "_comment": "EXIT: полоса выхода по RSI, 6 прогонов. Это единственное место, где меряется RSIUpper: cal_entry.json намеренно её не свипует, чтобы не отдавать 80 комбинаций на выборку около 36 сделок. Выход срабатывает, когда RSI пересекает уровень снизу вверх, и потому конкурирует с целью: чем ниже полоса, тем чаще сделка закрывается до TP. На UGLD именно RSI-выход даёт 61% выходов, и его полоса — не косметика, а основной механизм фиксации. Ось 55..80 совпадает с ugld/cal_exit.json: она задаётся не шириной инструмента, а шкалой самого RSI, поэтому пересчитывать её под ATR RENI не нужно. Нижний край 55 стоит контрольной строкой «выходим почти сразу»: если он выигрывает, стратегия на этом инструменте работает как скальп, а не как многодневное удержание, и это меняет читаемость всех остальных тем. Запуск: go run ./cmd/backtest -ticker RENI -strategy rsi_pullback -interval Minutes30 -calibrate data/params/rsi_pullback/reni/cal_exit.json -out ./reports/RENI_exit -months 36 -min-trades 20 -test-months 6 -metric profit_factor.",
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

```json
{
  "_comment": "TRAIL: форма трейла и его конкуренция с RSI-выходом, 12 прогонов. UseTrail зафиксирован в [1] — тема меряет форму трейла, а не факт его включения; UseRSIExit свипуется обеими точками, потому что трейл и RSI-выход борются за одну и ту же сделку, и их совместный эффект не выводится из раздельных замеров. TrailDailyATR=0 внутри включённого трейла означает подтяжку вплотную к максимуму и стоит левым краем оси. Правый край 0.8, а не 0.6 как у ugld/: там потолок задавала цель TPDailyATR=0.6, выше которой трейл просто не успевал взвестись до закрытия сделки, а здесь ось цели в cal_risk.json поднята до 2.5, и трейл получает пространство для позднего срабатывания. Замер, задающий шаг оси: медианный дневной размах RENI равен 0.88 ATR, поэтому подтяжка на 0.3-0.5 ATR отстаёт от цены примерно на треть-половину обычного дня, а 0.8 — почти на целый день. Читать надо не только profit factor, но и распределение причин выхода: если трейл забирает больше половины сделок, цель из cal_risk.json перестала работать и обе темы нужно перечитать вместе. Запуск: go run ./cmd/backtest -ticker RENI -strategy rsi_pullback -interval Minutes30 -calibrate data/params/rsi_pullback/reni/cal_trail.json -out ./reports/RENI_trail -months 36 -min-trades 20 -test-months 6 -metric profit_factor.",
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

Run: `go test ./internal/service/backtest/ -run 'TestRENI|TestRSIPullbackCalFilesValid|TestRSIPullbackGridControlPoints' -v 2>&1 | tail -25`
Expected: PASS. Среди подтестов `TestRSIPullbackCalFilesValid` должны появиться все девять `reni/cal_*.json`.

- [ ] **Step 10: Коммит**

```bash
git add internal/service/backtest/rsi_pullback_reni_grid_test.go \
        data/params/rsi_pullback/reni/cal_day.json \
        data/params/rsi_pullback/reni/cal_day_spent.json \
        data/params/rsi_pullback/reni/cal_volume.json \
        data/params/rsi_pullback/reni/cal_risk.json \
        data/params/rsi_pullback/reni/cal_exit.json \
        data/params/rsi_pullback/reni/cal_trail.json
git commit -m "test(rsi_pullback): сетки риска и гейтов RENI со сторожем на оси

Дневной гейт сдвинут по замерам инструмента: FreshDayATR от 0.25, потому
что к 07:00 медианный день проходит 0.29 ATR и порог 0.2 из ugld/ мёртв;
SpentDayATR от 0.8, потому что порога 0.6 достигают 79.4% дней.

Узкий стоп 0.3 ATR в оси сохранён: круг издержек стоит 0.030 ATR, то есть
10% такого стопа против 17% на DOMRF, где строку вырезали."
```

---

### Task 3: Приёмка

**Files:**
- Изменений в файлах нет — задача проверяет уже сделанное.

**Interfaces:**
- Consumes: девять файлов каталога `data/params/rsi_pullback/reni/` и оба теста из Task 1-2.
- Produces: ничего.

- [ ] **Step 1: Прогнать полный гейт качества**

Run: `./bin/mage ci 2>&1 | tail -30; echo "EXIT=${PIPESTATUS[0]}"`
Expected: `EXIT=0`, ноль строк `FAIL`. Гейт включает линт, `go test -race ./...` и проверку дрейфа моков — тот же набор, что запускает CI.

- [ ] **Step 2: Smoke-запуск команды из `cal_screen.json`**

Взять команду ровно из `_comment` этого файла и выполнить её:

```bash
go run ./cmd/backtest -ticker RENI -strategy rsi_pullback -interval Minutes30 \
  -calibrate data/params/rsi_pullback/reni/cal_screen.json -out ./reports/RENI_screen \
  -months 36 -min-trades 1 -metric profit_factor
```

Expected: команда завершается без ошибки и пишет markdown-отчёт в `./reports/RENI_screen`. Проверяются ровно две вещи: что незарегистрированный тикер резолвится через generic-ветку `RSIPullbackLookupOrGeneric` и что строка запуска не содержит опечаток.

**Числа этого отчёта не читаются и никуда не переносятся** — это проверка оснастки, а не измерение стратегии. Единственное, на что стоит взглянуть, — что счётчик сделок не равен нулю: ноль означал бы, что кэш не покрывает окно и нужен `-refresh` (кэш стоит на 2026-08-04).

- [ ] **Step 3: Убедиться, что отчёт создан**

Run: `ls -la reports/RENI_screen/ | tail -5`
Expected: как минимум один свежий `.md`-файл с сегодняшней датой в имени.

- [ ] **Step 4: Коммит**

Отчёты smoke-запуска в репозиторий не добавляются — коммитить нечего, если Steps 1-3 прошли чисто. Если `mage ci` потребовал правок, они коммитятся здесь:

```bash
git add -A
git commit -m "chore(rsi_pullback): правки по итогам gate-проверки каталога reni"
```

---

## Итог

После трёх задач в репозитории появляются девять файлов сеток `data/params/rsi_pullback/reni/` (105 прогонов суммарно) и два сторожевых теста, удерживающих каждое отклонение оси от `ugld/` при его замере. Пакета `strategy/reni` и записей в реестрах нет — они появятся вместе с литералом, если walk-forward пройдёт.

Дальше владелец запускает темы по порядку: `cal_screen` (решает, продолжать ли вообще), затем `cal_entry` и `cal_trend`, затем `cal_day` / `cal_day_spent`, `cal_volume`, `cal_risk`, `cal_exit`, `cal_trail`. Holdout последних 6 месяцев виден в отчёте каждой темы; повторные прогоны по одному и тому же holdout превращают его в обучающую выборку, как это произошло с DOMRF 2026-08-10.
