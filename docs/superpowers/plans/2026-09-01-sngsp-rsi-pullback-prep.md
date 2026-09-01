# SNGSP под `rsi_pullback` — план подготовки, калибровки и вердикта

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Довести SNGSP (Сургутнефтегаз, привилегированные акции) до вердикта по стратегии
`rsi_pullback`: каталог максимально широких сеток, тематический walk-forward, принятая точка,
литерал в пакете и заведение в боевую вселенную двадцатым тикером.

**Architecture:** Процедура тем **каноническая**: все десять тем идут поверх дефолтов ядра,
поэтому каталог сеток создаётся целиком одной задачей, якоря нет, поздних файлов нет. Схема
прогонов **штатная 36/12/6** — впервые за шесть тикеров подряд адаптация не нужна: обе серии кэша
длиннее запрошенного окна. Одно отступление, и оно в ширине осей: сетки держатся **максимально
широкими** по решению владельца, поэтому сторожевой тест осей запрещает урезать ось, а не
расширять её.

**Tech Stack:** Go 1.25, `cmd/backtest` (rolling walk-forward), `cmd/pullparity` (сверка живой
сборки с бэктестом), `./bin/mage ci` (lint + `go test -race ./...` + дрейф моков).

**Spec:** `docs/superpowers/specs/2026-09-01-sngsp-rsi-pullback-prep-design.md`

## Global Constraints

- **Схема прогонов штатная:** `-months 36 -train-months 12 -test-months 6 -min-trades 20 -metric
  profit_factor`, четыре фолда встык (12 + 6·4 = 36). У темы `screen` — `-min-trades 1`. Пропуск
  любого из трёх флагов окна даёт другую схему и несравнимые числа. Расчётное окно —
  **2023-09-01 … 2026-09-01**.
- **Число фолдов сверяется на первой же теме.** Ловушка ASTR: `cmd/backtest/main.go` берёт левую
  границу как `time.Now().AddDate(0, -months, 0)`, `internal/service/backtest/walkforward.go`
  строит фолды тем же `AddDate` и отбрасывает фолд, чей правый край **строго позже** `to`. Схема
  36/12/6 встык без запаса безопасна только пока запуск идёт на дате ≤ 28-го числа: 1-е число
  переполнения не даёт. Если в шапке отчёта темы `screen` окажется «Фолдов: 3» — **остановиться и
  доложить владельцу**, не продолжать прогоны и не пересматривать планку задним числом.
- **`-refresh` НЕ запускать ни на одном шаге.** Кэш дотянут штатным top-up провайдера 2026-09-01:
  `SNGSP_Minutes30.json` — 36 685 баров (2023-08-04 … 2026-09-01), в окне 35 972, из них 25 052
  будних; `SNGSP_Day1.json` — 1156 свечей (2022-08-04 … 2026-08-31), в окне 881, из них 761
  будняя. Запас истории слева: 28 дней у получасовой серии, ~13 месяцев у дневной. Любой refresh
  сдвинет границы и сделает все числа спеки несравнимыми.
- **Сетки держатся максимально широкими** (решение владельца 2026-09-01, то же, что по BANEP
  2026-08-28 и ASTR 2026-08-30). Обрезок осей не делается; замеры, которые в узком каталоге были
  бы основанием вырезать край, идут в `_comment` как **предупреждения** о том, чего ждать на
  краю. Жёстких инвариантов три: `RSILower` ≤ 50, `RSIPeriod` ≥ 3, ось тренда не порождает пар
  `EMAFast ≥ EMASlow`.
- **Цена широкой оси названа заранее:** она повышает риск переоптимизации на обучающем окне,
  поэтому пофолдовая устойчивость читается строже — край, победивший в одном фолде и мёртвый в
  остальных, в точку не идёт. Правило сборки точки: **не менее трёх фолдов из четырёх** за одно
  значение оси, иначе ось остаётся на дефолте ядра.
- **Планка** (объявлена до прогонов, не пересматривается): темы `entry` и `trend` обе дают pooled
  OOS PF ≥ 1.5 при ≥ 20 сделках в пуле; ведущая ось (`RSILower` для `entry`, `EMASlow` для
  `trend`) выбрана одинаково в ≥ 3 фолдах из 4; вырожденный фолд (ни одной убыточной сделки) в
  пользу тикера не засчитывается; счёт по дефолтной комиссии (круг 0.1%).
- **Правило прода:** литерал ставится и SNGSP заводится в `RSI_PULLBACK_TICKERS` двадцатым
  **независимо от того, взята планка или нет**.
- **Стоп-условие:** работа останавливается, если принятая точка даёт pooled OOS PF < 1.0, **либо**
  меньше 20 сделок за расчётное окно, **либо** PF < 1.0 под удвоенными издержками
  (`-commission 0.001`). При срабатывании — числа владельцу, задачи 12–16 не выполняются.
- **Каждый `_comment` сетки** обязан содержать: что тема меряет и сколько в ней прогонов; замер, из
  которого получена каждая ось, с предупреждением о крае; полную команду запуска с путём
  `data/params/rsi_pullback/sngsp/<файл>` (этого требует `TestRSIPullbackCalFilesValid`); место
  под строку `РЕЗУЛЬТАТ ПРОГОНА 2026-09-01: …`.
- **Дефолты ядра** (`core.DefaultParams()`), поверх которых считают ВСЕ темы: `RSIPeriod 4`,
  `RSILower 30`, `RSIUpper 70`, `EMAFast 10`, `EMASlow 100`, `DailyATRPeriod 14`,
  `UseDayATRGate 1`, `FreshDayATR 0`, `SpentDayATR 0.8`, `StopDailyATR 0.5`, `TPDailyATR 0.6`,
  `UseVolume 0`, `VolBaseDays 14`, `VolLookbackBars 3`, `VolMult 1.2`, `UseRSIExit 1`,
  `UseTrail 0`, `TrailDailyATR 0`. Контрольный прогон дефолтов на расчётном окне: **183 сделки,
  PF 1.535, net +32 599.57 ₽** — четвёртое место в ряду baseline каталога (YDEX 1.778, LSNGP
  1.554, ELFV 1.546, SNGSP 1.535, IVAT 1.432, LENT 1.417, SVAV 1.368, BANEP 1.336, BSPB 1.114,
  ASTR 1.093, SIBN 1.027, NVTK 0.984).
- **Полугодия расчётного окна** (нужны в Task 11): 2023-09-04 … 2024-03-01 (+22.6%),
  2024-03-01 … 2024-08-30 (−22.0%), 2024-08-30 … 2025-02-28 (+18.2%), 2025-02-28 … 2025-08-29
  (−24.0%), 2025-08-29 … 2026-03-02 (+8.1%), 2026-03-02 … 2026-08-31 (−11.3%).
- **Пер-тикерных записей в `docs/rsi_pullback/` не делается.** Правило CLAUDE.md: `strategy.md`,
  `live.md`, `screener.md` описывают механику, а не калибровочные прогоны. Пер-тикерный разбор
  живёт в доке пакета `strategy/sngsp` и в `_comment` файлов сеток. В `docs/rsi_pullback/` уходит
  только механический вывод, если SNGSP такой даст (Task 15 Step 3).
- **Каталог `reports/` — в `.gitignore`.** Отчёты прогонов остаются локальными артефактами; в репозиторий едут только строки `РЕЗУЛЬТАТ ПРОГОНА` в `_comment` сеток, рабочая записка Task 11 и код. Ни один шаг не должен пытаться закоммитить `reports/`.
- **Коммит по завершении каждой задачи**, сообщения на русском в стиле существующей истории.
- **Ветка:** `feat/sngsp-pullback-prep` от `main` (`1438322`).

---

### Task 1: Каталог десяти сеток со сторожевым тестом осей

**Files:**
- Create: `data/params/rsi_pullback/sngsp/cal_screen.json`
- Create: `data/params/rsi_pullback/sngsp/cal_entry.json`
- Create: `data/params/rsi_pullback/sngsp/cal_trend.json`
- Create: `data/params/rsi_pullback/sngsp/cal_day.json`
- Create: `data/params/rsi_pullback/sngsp/cal_day_spent.json`
- Create: `data/params/rsi_pullback/sngsp/cal_volume.json`
- Create: `data/params/rsi_pullback/sngsp/cal_vol_window.json`
- Create: `data/params/rsi_pullback/sngsp/cal_risk.json`
- Create: `data/params/rsi_pullback/sngsp/cal_exit.json`
- Create: `data/params/rsi_pullback/sngsp/cal_trail.json`
- Create: `internal/service/backtest/rsi_pullback_sngsp_grid_test.go`

**Interfaces:**
- Consumes: хелперы пакета `backtest` — `rsiPullbackTickerGrid(t, ticker, file)`
  (`rsi_pullback_grid_test.go:40`), `sameSet` (`rsi_pullback_reni_grid_test.go:17`),
  `containsValue` (`rsi_pullback_ivat_grid_test.go:88`); объявлять их заново НЕЛЬЗЯ — пакет один,
  будет переопределение. Общие тесты `TestRSIPullbackCalFilesValid`
  (`rsi_pullback_grid_test.go:89`) и `TestRSIPullbackGridControlPoints` (там же, :147)
  подхватывают новый каталог сами.
- Produces: каталог `data/params/rsi_pullback/sngsp/` и функцию `sngspGrid(t, file)` — ею
  пользуются только тесты этого файла.

- [ ] **Step 1: Написать падающий сторожевой тест осей**

Создать `internal/service/backtest/rsi_pullback_sngsp_grid_test.go`:

```go
package backtest

import "testing"

// sngspGrid читает файл сеток SNGSP через общий хелпер.
func sngspGrid(t *testing.T, file string) map[string][]float64 {
	t.Helper()
	return rsiPullbackTickerGrid(t, "sngsp", file)
}

// TestSNGSPGridsStayWide сторожит оси всех десяти тем каталога sngsp/. Каталог собран 2026-09-01
// по замерам самого SNGSP на расчётном окне 2023-09-01 … 2026-09-01 (36 685 получасовых баров в
// кэше, в окне 35 972, из них 25 052 будних; дневная серия 1156 свечей, в окне 881, из них 761
// будняя).
//
// РЕШЕНИЕ ВЛАДЕЛЬЦА 2026-09-01: сетки этого тикера держатся МАКСИМАЛЬНО ШИРОКИМИ, как у BANEP и
// ASTR. Поэтому тест сторожит оси с ДРУГОЙ стороны, чем у большинства тикеров каталога: он
// проверяет, что край оси не урезали, а не что его не расширили. Замеры, которые в узком каталоге
// были бы основанием обрезать край, живут в _comment каждой сетки как предупреждения.
//
// Инвариантов, которые остаются жёсткими, три, и все три — про смысл, а не про диапазон:
//
//   - RSILower не может быть выше 50: 50 — средняя линия осциллятора, выше неё отката нет по
//     определению.
//   - RSIPeriod не может быть короче 3: двойка — это не откат, а тик.
//   - Ось тренда не может порождать пар EMAFast >= EMASlow: при равенстве фильтр вырождается,
//     при инверсии становится другим фильтром.
//
// СХЕМА ПРОГОНОВ ШТАТНАЯ 36/12/6 — обе серии кэша длиннее запрошенного окна, адаптация не нужна.
func TestSNGSPGridsStayWide(t *testing.T) {
	screen := sngspGrid(t, "cal_screen.json")
	for _, field := range []string{"UseDayATRGate", "UseVolume"} {
		if got := screen[field]; !sameSet(got, 0, 1) {
			t.Errorf("cal_screen.json: %s = %v, want ровно {0,1} — тема меряет цену каждого гейта в сделках", field, got)
		}
	}

	entry := sngspGrid(t, "cal_entry.json")
	for _, v := range entry["RSILower"] {
		if v > 50 {
			t.Errorf("cal_entry.json свипует RSILower=%v: выше 50 отката нет по определению", v)
		}
	}
	for _, v := range entry["RSIPeriod"] {
		if v < 3 {
			t.Errorf("cal_entry.json свипует RSIPeriod=%v: короче тройки — уже не откат, а тик", v)
		}
	}
	// Рельеф уровня на SNGSP НЕ монотонный: максимум в середине оси (RSI(4): 10 -> 1.323,
	// 25 -> 1.674, 30 -> 1.535, 45 -> 0.997). Оба края обязаны остаться в сетке именно поэтому:
	// середина оси доказуема только при видимых краях.
	for _, v := range []float64{3, 8} {
		if !containsValue(entry["RSIPeriod"], v) {
			t.Errorf("cal_entry.json: RSIPeriod = %v, не содержит %v — ось входа держится широкой по решению владельца", entry["RSIPeriod"], v)
		}
	}
	for _, v := range []float64{10, 50} {
		if !containsValue(entry["RSILower"], v) {
			t.Errorf("cal_entry.json: RSILower = %v, не содержит %v — оба края оси обязаны остаться", entry["RSILower"], v)
		}
	}
	// Полоса выхода расширена ВНИЗ до 50: максимум оси стоит на 60, у нижнего края канона §8.1.
	for _, v := range []float64{50, 85} {
		if !containsValue(entry["RSIUpper"], v) {
			t.Errorf("cal_entry.json: RSIUpper = %v, не содержит %v — полоса выхода меряется целиком, вниз до 50", entry["RSIUpper"], v)
		}
	}

	trend := sngspGrid(t, "cal_trend.json")
	for _, fast := range trend["EMAFast"] {
		for _, slow := range trend["EMASlow"] {
			if fast >= slow {
				t.Errorf("cal_trend.json порождает пару EMAFast=%v >= EMASlow=%v: фильтр тренда вырождается", fast, slow)
			}
		}
	}
	// Ось медленной EMA расширена ВВЕРХ до 250: 200 бьёт 150 (1.400 против 1.355), максимум
	// верхнего участка стоял на краю канонической оси. Быстрая держится до 40 — на паре 40/50
	// стоит максимум таблицы (1.647), и его нельзя прятать обрезкой.
	for _, v := range []float64{50, 250} {
		if !containsValue(trend["EMASlow"], v) {
			t.Errorf("cal_trend.json: EMASlow = %v, не содержит %v — ось тренда меряется целиком", trend["EMASlow"], v)
		}
	}
	for _, v := range []float64{5, 40} {
		if !containsValue(trend["EMAFast"], v) {
			t.Errorf("cal_trend.json: EMAFast = %v, не содержит %v — оба края быстрой EMA обязаны остаться", trend["EMAFast"], v)
		}
	}

	day := sngspGrid(t, "cal_day.json")
	// Ветка свежего дня на SNGSP монотонно вредит (0 -> 1.535, 0.6 -> 1.033), но ноль и верхний
	// край обязаны остаться: без нуля тема не может выбрать «свежая ветка закрыта», без 0.6 —
	// показать, что вред монотонен.
	for _, v := range []float64{0, 0.6} {
		if !containsValue(day["FreshDayATR"], v) {
			t.Errorf("cal_day.json: FreshDayATR = %v, не содержит %v", day["FreshDayATR"], v)
		}
	}
	for _, v := range []float64{0.5, 1.5} {
		if !containsValue(day["SpentDayATR"], v) {
			t.Errorf("cal_day.json: SpentDayATR = %v, не содержит %v — оба края исчерпанной ветки обязаны остаться", day["SpentDayATR"], v)
		}
	}

	spent := sngspGrid(t, "cal_day_spent.json")
	if got := spent["FreshDayATR"]; !sameSet(got, 0) {
		t.Errorf("cal_day_spent.json: FreshDayATR = %v, want ровно {0} — тема меряет ТОЛЬКО исчерпанную ветку", got)
	}
	for _, v := range []float64{0.4, 1.5} {
		if !containsValue(spent["SpentDayATR"], v) {
			t.Errorf("cal_day_spent.json: SpentDayATR = %v, не содержит %v", spent["SpentDayATR"], v)
		}
	}

	volume := sngspGrid(t, "cal_volume.json")
	for _, v := range []float64{1, 3} {
		if !containsValue(volume["VolMult"], v) {
			t.Errorf("cal_volume.json: VolMult = %v, не содержит %v — ось множителя держится широкой, включая выброс 3.0 на 48 сделках", volume["VolMult"], v)
		}
	}
	for _, v := range []float64{3, 20} {
		if !containsValue(volume["VolBaseDays"], v) {
			t.Errorf("cal_volume.json: VolBaseDays = %v, не содержит %v", volume["VolBaseDays"], v)
		}
	}

	window := sngspGrid(t, "cal_vol_window.json")
	for _, v := range []float64{1, 24} {
		if !containsValue(window["VolLookbackBars"], v) {
			t.Errorf("cal_vol_window.json: VolLookbackBars = %v, не содержит %v — оба края окна обязаны остаться", window["VolLookbackBars"], v)
		}
	}

	risk := sngspGrid(t, "cal_risk.json")
	// Ось стопа расширена ВВЕРХ до 2.0: рельеф не насыщается к 1.5 (1.5 -> 1.865, 2.0 -> 2.122
	// при неизменных 175 сделках). Цена расширения записана в _comment как предупреждение о
	// капкане, а не как обрезка.
	for _, v := range []float64{0.3, 2} {
		if !containsValue(risk["StopDailyATR"], v) {
			t.Errorf("cal_risk.json: StopDailyATR = %v, не содержит %v — оба края оси риска обязаны остаться", risk["StopDailyATR"], v)
		}
	}
	// Цель 2.5 — контрольная строка: TestRSIPullbackGridControlPoints требует от файла, свипующего
	// стоп до 2.0, хотя бы одну цель ВЫШЕ 2.0. Строка заведомо мертва (1.5, 2.0 и 2.5 совпадают
	// побайтово), и это записано в _comment.
	for _, v := range []float64{0.3, 2.5} {
		if !containsValue(risk["TPDailyATR"], v) {
			t.Errorf("cal_risk.json: TPDailyATR = %v, не содержит %v", risk["TPDailyATR"], v)
		}
	}

	exit := sngspGrid(t, "cal_exit.json")
	for _, v := range []float64{50, 85} {
		if !containsValue(exit["RSIUpper"], v) {
			t.Errorf("cal_exit.json: RSIUpper = %v, не содержит %v — ось выхода расширена вниз до 50", exit["RSIUpper"], v)
		}
	}

	trail := sngspGrid(t, "cal_trail.json")
	if got := trail["UseRSIExit"]; !sameSet(got, 0, 1) {
		t.Errorf("cal_trail.json: UseRSIExit = %v, want ровно {0,1} — тема меряет трейл ПРОТИВ RSI-выхода", got)
	}
	for _, v := range []float64{0.3, 1.5} {
		if !containsValue(trail["TrailDailyATR"], v) {
			t.Errorf("cal_trail.json: TrailDailyATR = %v, не содержит %v", trail["TrailDailyATR"], v)
		}
	}
}
```

- [ ] **Step 2: Запустить тест и убедиться, что он падает**

Run: `go test ./internal/service/backtest/ -run TestSNGSPGridsStayWide -v`
Expected: FAIL — каталога `data/params/rsi_pullback/sngsp/` не существует.

- [ ] **Step 3: Создать десять файлов сеток**

`data/params/rsi_pullback/sngsp/cal_screen.json`:

```json
{
  "_comment": "data/params/rsi_pullback/sngsp/cal_screen.json — тема screen для SNGSP (Сургутнефтегаз, привилегированные акции), двадцатый тикер каталога rsi_pullback. Тема меряет ЦЕНУ ДВУХ ОПЦИОНАЛЬНЫХ ГЕЙТОВ в сделках: 4 прогона (UseDayATRGate x UseVolume). РАННЯЯ ТЕМА — идёт поверх дефолтов ядра, как у всего каталога. Точечные замеры до прогона (36 месяцев, дефолты ядра): оба гейта в дефолтном положении (день включён, объём выключен) — 183 сделки, PF 1.535, net +32599.57; день ВЫКЛЮЧЕН — 501 сделка, PF 0.944; объём включён при VolMult 1.2 — 134 сделки, PF 1.519. Дневной гейт несёт ВЕСЬ edge: без него стратегия на SNGSP убыточна, и это самый сильный вклад одного фильтра в каталоге. Объёмный гейт неразличим с выключенным (вся его ось лежит в полосе 1.39–1.61 вокруг baseline). ЗАПУСК: go run ./cmd/backtest -ticker SNGSP -strategy rsi_pullback -interval Minutes30 -calibrate data/params/rsi_pullback/sngsp/cal_screen.json -out ./reports/SNGSP_screen -months 36 -train-months 12 -test-months 6 -min-trades 1 -metric profit_factor. У ЭТОЙ темы -min-trades 1, а не 20: выключенный дневной гейт даёт втрое больше сделок, а порог 20 отсёк бы часть смысла темы. ПРОВЕРИТЬ В ШАПКЕ ОТЧЁТА: фолдов должно быть ЧЕТЫРЕ; три означают ловушку переполнения AddDate и требуют остановки. РЕЗУЛЬТАТ ПРОГОНА 2026-09-01: ",
  "phases": [
    {
      "name": "screen",
      "grid": {
        "UseDayATRGate": [0, 1],
        "UseVolume": [0, 1]
      },
      "keepTop": 4
    }
  ]
}
```

`data/params/rsi_pullback/sngsp/cal_entry.json`:

```json
{
  "_comment": "data/params/rsi_pullback/sngsp/cal_entry.json — КЛЮЧЕВАЯ тема entry для SNGSP, 432 прогона (RSIUpper x RSIPeriod x RSILower = 8 x 6 x 9). РАННЯЯ ТЕМА — поверх дефолтов ядра: на ней стоит планка, и только прогон от дефолтов делает вердикт SNGSP сравнимым с девятнадцатью предшественниками. ОСИ МАКСИМАЛЬНО ШИРОКИЕ (решение владельца 2026-09-01). RSIPeriod 3..8: кроссов вниз хватает на всей оси (уровень 30: RSI(3) 2526, RSI(4) 1933, RSI(6) 1215, RSI(8) 843 события на 25 052 будних барах окна). RSILower 10..50 держится обоими краями, потому что рельеф уровня на SNGSP НЕ монотонный, а с максимумом В СЕРЕДИНЕ: RSI(4) даёт 10 -> 1.323/23 сделки, 15 -> 1.145/54, 20 -> 1.270/87, 25 -> 1.674/133, 30 -> 1.535/183, 35 -> 1.224/213, 40 -> 1.193/256, 45 -> 0.997/274, 50 -> 0.953/313 — это отличие от ASTR, где edge уходил в глубину. ПРЕДУПРЕЖДЕНИЕ О ГЛУБОКОМ КРАЕ: на длинных периодах глубокие уровни дают заоблачный PF на исчезающей выборке (RSI(6)@10 -> 3.722 на 4 сделках, RSI(6)@15 -> 2.090 на 12) — это шум, и порог -min-trades 20 его отсекает; строки остаются в сетке ради видимости рельефа, а не как кандидаты. RSIUpper РАСШИРЕН ВНИЗ ДО 50 (канон §8.1 начинается с 55): максимум оси выхода стоит на 60 (1.817/195 против 1.535/183 у дефолтных 70), а рельеф выше монотонно падает до 0.929 на 85 — ось, чей максимум стоит у края, каталог читать не умеет (ошибка, разобранная на WUSH). Кроссов вверх через 55 у RSI(4) 2962, ниже их только больше — уровень 50 живой. ЗАПУСК: go run ./cmd/backtest -ticker SNGSP -strategy rsi_pullback -interval Minutes30 -calibrate data/params/rsi_pullback/sngsp/cal_entry.json -out ./reports/SNGSP_entry -months 36 -train-months 12 -test-months 6 -min-trades 20 -metric profit_factor. РЕЗУЛЬТАТ ПРОГОНА 2026-09-01: ",
  "phases": [
    {
      "name": "entry",
      "grid": {
        "RSIUpper": [50, 55, 60, 65, 70, 75, 80, 85],
        "RSIPeriod": [3, 4, 5, 6, 7, 8],
        "RSILower": [10, 15, 20, 25, 30, 35, 40, 45, 50]
      },
      "keepTop": 6
    }
  ]
}
```

`data/params/rsi_pullback/sngsp/cal_trend.json`:

```json
{
  "_comment": "data/params/rsi_pullback/sngsp/cal_trend.json — ВТОРАЯ КЛЮЧЕВАЯ тема trend для SNGSP, 25 прогонов (EMAFast x EMASlow = 5 x 5). РАННЯЯ ТЕМА — поверх дефолтов ядра. ОСЬ МЕДЛЕННОЙ EMA РАСШИРЕНА ВВЕРХ ДО 250: при EMAFast 10 медленная 20 -> 1.145/157, 30 -> 1.187/165, 50 -> 1.341/170, 100 -> 1.535/183, 150 -> 1.355/194, 200 -> 1.400/201, 250 -> 1.283/200 — двухсотка бьёт полторашку, то есть максимум верхнего участка стоял на краю канонической оси §8.1. МЕДЛЕННАЯ 30 НЕ БЕРЁТСЯ, и это не обрезка, а выбор между двумя расширениями: с ней пришлось бы урезать быструю до 20 (инвариант EMAFast < EMASlow), а замер требует обратного — при EMASlow 50 быстрая 5 -> 1.246/151, 10 -> 1.341/170, 20 -> 1.500/176, 30 -> 1.610/186, 40 -> 1.647/191, то есть МАКСИМУМ ВСЕЙ ТАБЛИЦЫ стоит на паре 40/50; сама же медленная 30 сидит внизу рельефа (1.187). ПРЕДУПРЕЖДЕНИЕ: пара 40/50 — вырожденный фильтр (быстрая почти равна медленной), её победа читается как «фильтр тренда бумаге не нужен», а не как находка параметра. ТРЕНДОВЫЙ ДОПУСК РОВНЫЙ ДО ВЫРОЖДЕНИЯ: доля будних баров с EMAFast > EMASlow держится в полосе 48.0–49.3% по ВСЕЙ таблице 5..40 x 30..250 (n = 25 052, размах два процентных пункта — та же картина, что на BSPB). Ожидание: разброс победителей по фолдам. ЗАПУСК: go run ./cmd/backtest -ticker SNGSP -strategy rsi_pullback -interval Minutes30 -calibrate data/params/rsi_pullback/sngsp/cal_trend.json -out ./reports/SNGSP_trend -months 36 -train-months 12 -test-months 6 -min-trades 20 -metric profit_factor. РЕЗУЛЬТАТ ПРОГОНА 2026-09-01: ",
  "phases": [
    {
      "name": "trend",
      "grid": {
        "EMAFast": [5, 10, 20, 30, 40],
        "EMASlow": [50, 100, 150, 200, 250]
      },
      "keepTop": 5
    }
  ]
}
```

`data/params/rsi_pullback/sngsp/cal_day.json`:

```json
{
  "_comment": "data/params/rsi_pullback/sngsp/cal_day.json — тема day для SNGSP, 48 прогонов (FreshDayATR x SpentDayATR = 6 x 8), поверх дефолтов ядра. Тема меряет ДВУСТОРОННИЙ дневной гейт. Гейт несёт весь edge: без него 501 сделка при PF 0.944 против 183 при 1.535. Оси из замеров SNGSP. Ветка «день исчерпан» (при FreshDayATR 0): 0.5 -> 1.017/353, 0.6 -> 1.078/288, 0.7 -> 1.533/229, 0.8 -> 1.535/183, 0.9 -> 1.373/135, 1.0 -> 1.236/97, 1.25 -> 1.007/39, 1.5 -> 0.864/17 — плато 0.7–0.8, дальше PF падает вместе с выборкой. ПРЕДУПРЕЖДЕНИЕ О ВЕРХНЕМ КРАЕ: 1.25 и 1.5 стоят на 39 и 17 сделках за 36 месяцев, то есть 13 и 6 на двенадцатимесячное обучающее окно — под порогом -min-trades 20; строки остаются ради видимости спада, а не как кандидаты. Ветка «свежий день» (при SpentDayATR 0.8): 0 -> 1.535/183, 0.2 -> 1.397/218, 0.3 -> 1.213/260, 0.4 -> 1.100/306, 0.5 -> 1.075/364, 0.6 -> 1.033/421 — монотонно вниз, открытая свежая ветка на SNGSP добавляет мусорных сделок. Ноль в оси FreshDayATR обязателен: на всех прод-тикерах каталога победил именно он. Доли будних баров, проходящих ветки (n = 24 462): свежий день 0.2 -> 5.8%, 0.3 -> 12.2%, 0.4 -> 20.2%, 0.5 -> 30.6%, 0.6 -> 42.1%; день исчерпан 0.5 -> 69.4%, 0.6 -> 57.9%, 0.7 -> 47.1%, 0.8 -> 37.6%, 0.9 -> 28.4%, 1.0 -> 22.3%, 1.25 -> 11.4%, 1.5 -> 6.1%. ЗАПУСК: go run ./cmd/backtest -ticker SNGSP -strategy rsi_pullback -interval Minutes30 -calibrate data/params/rsi_pullback/sngsp/cal_day.json -out ./reports/SNGSP_day -months 36 -train-months 12 -test-months 6 -min-trades 20 -metric profit_factor. РЕЗУЛЬТАТ ПРОГОНА 2026-09-01: ",
  "phases": [
    {
      "name": "day",
      "grid": {
        "UseDayATRGate": [1],
        "FreshDayATR": [0, 0.2, 0.3, 0.4, 0.5, 0.6],
        "SpentDayATR": [0.5, 0.6, 0.7, 0.8, 0.9, 1.0, 1.25, 1.5]
      },
      "keepTop": 5
    }
  ]
}
```

`data/params/rsi_pullback/sngsp/cal_day_spent.json`:

```json
{
  "_comment": "data/params/rsi_pullback/sngsp/cal_day_spent.json — тема day_spent для SNGSP, 10 прогонов, поверх дефолтов ядра. Тема меряет ТОЛЬКО ветку «день исчерпан» на уплотнённой оси при FreshDayATR = 0; прямое сравнение с cal_day.json показывает, сколько стоит открытая ветка свежего дня. Уровни 0.4 и 1.1 добавлены к канону — приём, сработавший на SIBN, ELFV, DIAS и BSPB. Замер (36 месяцев, дефолты ядра): 0.4 -> 1.037/408, 0.5 -> 1.017/353, 0.6 -> 1.078/288, 0.7 -> 1.533/229, 0.8 -> 1.535/183, 0.9 -> 1.373/135, 1.0 -> 1.236/97, 1.1 -> 1.240/76, 1.25 -> 1.007/39, 1.5 -> 0.864/17. Ступенька между 0.6 и 0.7 — самая резкая на оси (0.455 PF за один шаг), и именно её тема обязана проверить пофолдово: одиночный замер не отличает настоящий порог от артефакта конкретной истории. ЗАПУСК: go run ./cmd/backtest -ticker SNGSP -strategy rsi_pullback -interval Minutes30 -calibrate data/params/rsi_pullback/sngsp/cal_day_spent.json -out ./reports/SNGSP_day_spent -months 36 -train-months 12 -test-months 6 -min-trades 20 -metric profit_factor. РЕЗУЛЬТАТ ПРОГОНА 2026-09-01: ",
  "phases": [
    {
      "name": "day_spent",
      "grid": {
        "UseDayATRGate": [1],
        "FreshDayATR": [0],
        "SpentDayATR": [0.4, 0.5, 0.6, 0.7, 0.8, 0.9, 1.0, 1.1, 1.25, 1.5]
      },
      "keepTop": 5
    }
  ]
}
```

`data/params/rsi_pullback/sngsp/cal_volume.json`:

```json
{
  "_comment": "data/params/rsi_pullback/sngsp/cal_volume.json — тема volume для SNGSP, 30 прогонов (VolMult x VolBaseDays = 6 x 5), поверх дефолтов ядра. Тема меряет объёмный гейт. ОСЬ МНОЖИТЕЛЯ ДЕРЖИТСЯ ШИРОКОЙ ДО 3.0, хотя замер велит ждать несходимости: множитель при базе 14 даёт 1.0 -> 1.607/146, 1.2 -> 1.519/134, 1.5 -> 1.561/115, 2.0 -> 1.472/84, 2.5 -> 1.602/68, 3.0 -> 2.129/48 — весь массив лежит в полосе вокруг baseline 1.535, то есть гейт НЕРАЗЛИЧИМ с выключенным. ПРЕДУПРЕЖДЕНИЕ О ВЕРХНЕМ КРАЕ: единственный выброс (3.0 -> 2.129) стоит на 48 сделках за 36 месяцев = 16 на обучающее окно, ниже -min-trades 20; такая строка может победить только процедурным артефактом. В узком каталоге это было бы основанием обрезать ось до 2.0 (решение DIAS и BSPB), здесь строка остаётся по решению о широких сетках. База при множителе 1.5: 3 -> 1.521/129, 5 -> 1.458/125, 10 -> 1.596/121, 14 -> 1.561/115, 20 -> 1.393/114 — разброс 0.20 PF, рельефа почти нет. ЛИКВИДНОСТЬ SNGSP — ЛУЧШАЯ В КАТАЛОГЕ (оборот медиана 1299 млн ₽, p10 520 млн; дней короче 20 баров 0.8%), поэтому объёмный гейт здесь имеет наименьшие шансы из всего каталога: тонких дней, которые он отсекает у других бумаг, тут почти нет. ЗАПУСК: go run ./cmd/backtest -ticker SNGSP -strategy rsi_pullback -interval Minutes30 -calibrate data/params/rsi_pullback/sngsp/cal_volume.json -out ./reports/SNGSP_volume -months 36 -train-months 12 -test-months 6 -min-trades 20 -metric profit_factor. РЕЗУЛЬТАТ ПРОГОНА 2026-09-01: ",
  "phases": [
    {
      "name": "volume",
      "grid": {
        "UseVolume": [1],
        "VolMult": [1.0, 1.2, 1.5, 2.0, 2.5, 3.0],
        "VolBaseDays": [3, 5, 10, 14, 20]
      },
      "keepTop": 5
    }
  ]
}
```

`data/params/rsi_pullback/sngsp/cal_vol_window.json`:

```json
{
  "_comment": "data/params/rsi_pullback/sngsp/cal_vol_window.json — тема vol_window для SNGSP, 24 прогона (VolLookbackBars x VolMult = 8 x 3), поверх дефолтов ядра. Тема меряет ОКНО объёмного гейта — сколько последних баров сравниваются с фоном. Замер при VolMult 2.0 и базе 14: окно 1 -> 1.496/66, 2 -> 1.431/79, 3 (дефолт ядра) -> 1.472/84, 5 -> 1.324/101, 8 -> 1.461/114, 12 -> 1.479/134, 16 -> 1.427/144, 24 -> 1.602/160. Рельеф плоский (размах 0.28 PF без монотонности), максимум на самом широком крае — а широкое окно ОСЛАБЛЯЕТ гейт, приближая его к выключенному. ЧИТАТЬ ОСТОРОЖНО: победитель на краю 24 обязан проверяться на эквивалентность UseVolume = 0, и при неразличимых числах предпочитается выключенный гейт (более простая конфигурация). ГИПОТЕЗА КАТАЛОГА, которую тема проверяет восьмой раз: чем ликвиднее бумага, тем ближе оптимум окна к дефолту ядра (ELFV с 18.9% коротких дней получил максимум ровно на дефолте, DIAS с 11.9% — на 12, BSPB с 7.7% — на 5). SNGSP ликвиднее всех троих (0.8% коротких дней, медиана 35 баров в дне), значит гипотеза предсказывает максимум у дефолта; замер её НЕ подтверждает — это и есть предмет темы. ЗАПУСК: go run ./cmd/backtest -ticker SNGSP -strategy rsi_pullback -interval Minutes30 -calibrate data/params/rsi_pullback/sngsp/cal_vol_window.json -out ./reports/SNGSP_vol_window -months 36 -train-months 12 -test-months 6 -min-trades 20 -metric profit_factor. РЕЗУЛЬТАТ ПРОГОНА 2026-09-01: ",
  "phases": [
    {
      "name": "vol_window",
      "grid": {
        "UseVolume": [1],
        "VolLookbackBars": [1, 2, 3, 5, 8, 12, 16, 24],
        "VolMult": [1.2, 2.0, 3.0]
      },
      "keepTop": 5
    }
  ]
}
```

`data/params/rsi_pullback/sngsp/cal_risk.json`:

```json
{
  "_comment": "data/params/rsi_pullback/sngsp/cal_risk.json — тема risk для SNGSP, 56 прогонов (StopDailyATR x TPDailyATR = 7 x 8), поверх дефолтов ядра. Тема меряет стоп и цель в дневных ATR. ОСЬ СТОПА РАСШИРЕНА ВВЕРХ ДО 2.0 (канон §8.1 кончается на 1.5): при цели 0.6 стоп 0.3 -> 1.479/191, 0.5 -> 1.535/183, 0.7 -> 1.514/179, 1.0 -> 1.728/176, 1.3 -> 1.703/176, 1.5 -> 1.865/175, 2.0 -> 2.122/175 — рельеф НЕ насыщается к полутора, в отличие от предшественников. ПРЕДУПРЕЖДЕНИЕ О КАПКАНЕ (восьмой тикер подряд): число сделок стоит на месте с 0.7, а PF растёт в полтора раза — это не защита капитала, а вытеснение убытка в RSI-выход; выживаемость стопов (доля будних дней, чей размах достаёт уровня, n = 747): 0.3 -> 99.3%, 0.5 -> 92.4%, 0.7 -> 74.4%, 1.0 -> 38.0%, 1.3 -> 16.6%, 1.5 -> 9.9%, то есть при 2.0 дневных ATR (~5.2% цены) стоп перестаёт быть защитой. Победитель темы ОБЯЗАН проверяться долей SL-выходов и удержанием сделок (см. Task 9 Step 3 и Task 11). ОСЬ ЦЕЛИ РАСШИРЕНА ВНИЗ ДО 0.3: максимум оси стоит на 0.5, то есть НИЖЕ дефолта ядра (при стопе 0.5: цель 0.3 -> 1.576/190, 0.4 -> 1.528/185, 0.5 -> 1.637/184, 0.6 -> 1.535/183, 0.8 -> 1.486/182, 1.0 -> 1.444/181), и весь edge живёт в коротких целях. СТРОКА 2.5 — КОНТРОЛЬНАЯ, а не кандидат: цели 1.5, 2.0 и 2.5 совпадают ПОБАЙТОВО (цель шире полутора дневных ATR на SNGSP недостижима — раньше срабатывает RSI-выход или стоп), но тест TestRSIPullbackGridControlPoints требует от файла, свипующего стоп до 2.0, хотя бы одной цели ВЫШЕ 2.0, иначе асимметрия риск/награда остаётся непроверенной. ЗАПУСК: go run ./cmd/backtest -ticker SNGSP -strategy rsi_pullback -interval Minutes30 -calibrate data/params/rsi_pullback/sngsp/cal_risk.json -out ./reports/SNGSP_risk -months 36 -train-months 12 -test-months 6 -min-trades 20 -metric profit_factor. РЕЗУЛЬТАТ ПРОГОНА 2026-09-01: ",
  "phases": [
    {
      "name": "risk",
      "grid": {
        "StopDailyATR": [0.3, 0.5, 0.7, 1.0, 1.3, 1.5, 2.0],
        "TPDailyATR": [0.3, 0.4, 0.5, 0.6, 0.8, 1.0, 1.5, 2.5]
      },
      "keepTop": 5
    }
  ]
}
```

`data/params/rsi_pullback/sngsp/cal_exit.json`:

```json
{
  "_comment": "data/params/rsi_pullback/sngsp/cal_exit.json — тема exit для SNGSP, 8 прогонов, поверх дефолтов ядра. Тема меряет уровень выхода RSIUpper. ОСЬ РАСШИРЕНА ВНИЗ ДО 50 (канон §8.1 начинается с 55): замер даёт 55 -> 1.665/199, 60 -> 1.817/195, 65 -> 1.686/189, 70 -> 1.535/183, 75 -> 1.360/181, 80 -> 1.147/179, 85 -> 0.929/175 — максимум стоит на 60, рельеф выше монотонно падает, картина ЗЕРКАЛЬНА BSPB (там максимум был на 75 и ось расширяли вверх). Ось, чей максимум стоит у края, каталог читать не умеет, поэтому нижний край опущен до 50. Уровень живой: кроссов вверх через 55 у RSI(4) 2962 за окно, через 50 их только больше. ПРЕДУПРЕЖДЕНИЕ О НИЖНЕМ КРАЕ: чем ниже уровень выхода, тем короче удержание, а на SNGSP оно и так внутридневное (медиана 0 календарных дней, ночь переживают 1.6% сделок) — победа края 50 означает ещё более быстрый выход, а не более выгодный. ЗАПУСК: go run ./cmd/backtest -ticker SNGSP -strategy rsi_pullback -interval Minutes30 -calibrate data/params/rsi_pullback/sngsp/cal_exit.json -out ./reports/SNGSP_exit -months 36 -train-months 12 -test-months 6 -min-trades 20 -metric profit_factor. РЕЗУЛЬТАТ ПРОГОНА 2026-09-01: ",
  "phases": [
    {
      "name": "exit",
      "grid": {
        "RSIUpper": [50, 55, 60, 65, 70, 75, 80, 85]
      },
      "keepTop": 5
    }
  ]
}
```

`data/params/rsi_pullback/sngsp/cal_trail.json`:

```json
{
  "_comment": "data/params/rsi_pullback/sngsp/cal_trail.json — тема trail для SNGSP, 10 прогонов (UseRSIExit x TrailDailyATR = 2 x 5), поверх дефолтов ядра. Тема меряет трейл ПРОТИВ RSI-выхода; UseTrail форсирован в 1 внутри темы, чтобы дистанция вообще измерялась. Замер при включённом RSI-выходе: трейл 0.3 -> 1.196/198, 0.5 -> 1.679/184, 0.7 -> 1.511/183, 1.0 -> 1.535/183, 1.5 -> 1.535/183 — с 1.0 трейл не срабатывает ни разу и побайтово равен его отсутствию, а 0.5 даёт прирост 0.144 PF над baseline. Это отличие от ASTR и BSPB, где трейл только портил. ВТОРАЯ ОСЬ ТЕМЫ ВАЖНЕЕ ПЕРВОЙ: при UseRSIExit = 0 стратегия на SNGSP разваливается на всей оси цели (0.5 -> 0.804/172, 0.6 -> 0.753/169, 1.0 -> 0.627/146, 1.5 -> 0.602/122, 2.0 -> 0.661/110, 2.5 -> 0.661/94) — RSI-выход критичен, и трейл его НЕ заменяет, а лишь дополняет. ПРЕДУПРЕЖДЕНИЕ: выключенный RSI-выход удлиняет удержание, а удлинение удержания на SNGSP будит спящий дивидендный риск (гэп отсечки до −14.06%, ~5.4 дневных ATR, стопом не держится). Победа UseRSIExit = 0 в любом фолде требует отдельной проверки экспозиции через июльские отсечки. ЗАПУСК: go run ./cmd/backtest -ticker SNGSP -strategy rsi_pullback -interval Minutes30 -calibrate data/params/rsi_pullback/sngsp/cal_trail.json -out ./reports/SNGSP_trail -months 36 -train-months 12 -test-months 6 -min-trades 20 -metric profit_factor. РЕЗУЛЬТАТ ПРОГОНА 2026-09-01: ",
  "phases": [
    {
      "name": "trail",
      "grid": {
        "UseRSIExit": [0, 1],
        "UseTrail": [1],
        "TrailDailyATR": [0.3, 0.5, 0.7, 1.0, 1.5]
      },
      "keepTop": 5
    }
  ]
}
```

- [ ] **Step 4: Запустить сторожевой и общие тесты каталога**

Run: `go test ./internal/service/backtest/ -run 'TestSNGSPGridsStayWide|TestRSIPullbackCalFilesValid|TestRSIPullbackGridControlPoints|TestRSIPullbackPlateauFilesArePoints' -v`
Expected: PASS все четыре. Если `TestRSIPullbackGridControlPoints` падает на `cal_risk.json` —
проверить, что цель 2.5 больше максимального стопа 2.0 (строка контрольная, удалять её нельзя).

- [ ] **Step 5: Коммит**

```bash
git add data/params/rsi_pullback/sngsp internal/service/backtest/rsi_pullback_sngsp_grid_test.go
git commit -m "feat(rsi_pullback): каталог максимально широких сеток SNGSP"
```

---

### Task 2: Пакет `strategy/sngsp` в состоянии «калибровка не проводилась»

**Files:**
- Create: `internal/service/trading_strategy/rsi_pullback/strategy/sngsp/sngsp.go`
- Create: `internal/service/trading_strategy/rsi_pullback/strategy/sngsp/sngsp_test.go`
- Modify: `internal/service/backtest/rsi_pullback_registry.go`
- Modify: `internal/service/backtest/rsi_pullback_registry_test.go`

**Interfaces:**
- Consumes: `core.Params`, `core.DefaultParams()` из
  `internal/service/trading_strategy/rsi_pullback/strategy/core`.
- Produces: `sngsp.Ticker` (строка `"SNGSP"`) и `sngsp.DefaultParams() core.Params` — их используют
  Task 12 (литерал) и Task 13 (реестр живого раннера).

- [ ] **Step 1: Написать падающий тест baseline-состояния**

Создать `internal/service/trading_strategy/rsi_pullback/strategy/sngsp/sngsp_test.go`:

```go
package sngsp

import (
	"testing"

	"tinvest/internal/service/trading_strategy/rsi_pullback/strategy/core"
)

// TestParamsTrackTheBaselineUntilCalibrated фиксирует ЧЕСТНОЕ состояние: калибровка SNGSP ещё не
// проводилась, поэтому пакет обязан возвращать ровно baseline ядра. Тест держит это состояние до
// Task 12, где его заменяет снимок литерала. Пока он стоит, ни одна правка не может тихо
// подсунуть в прод «почти откалиброванные» параметры.
func TestParamsTrackTheBaselineUntilCalibrated(t *testing.T) {
	if got, want := DefaultParams(), core.DefaultParams(); got != want {
		t.Fatalf("SNGSP ещё не откалиброван, параметры обязаны совпадать с baseline:\n got: %+v\nwant: %+v", got, want)
	}
}

func TestTickerIsSNGSP(t *testing.T) {
	if Ticker != "SNGSP" {
		t.Fatalf("Ticker = %q, want SNGSP", Ticker)
	}
}
```

- [ ] **Step 2: Запустить тест и убедиться, что он падает**

Run: `go test ./internal/service/trading_strategy/rsi_pullback/strategy/sngsp/ -v`
Expected: FAIL — пакета `sngsp` не существует.

- [ ] **Step 3: Создать пакет**

Создать `internal/service/trading_strategy/rsi_pullback/strategy/sngsp/sngsp.go`:

```go
// Package sngsp supplies the ticker and rsi_pullback Params for SNGSP (Сургутнефтегаз,
// привилегированные акции, лот 10).
//
// СОСТОЯНИЕ: КАЛИБРОВКА НЕ ПРОВОДИЛАСЬ. Пакет возвращает core.DefaultParams() — baseline ядра, не
// подобранный под этот инструмент. Так и должно быть до конца калибровки: пакет заведён заранее,
// чтобы прогоны шли через тот же реестр, что и у остальных девятнадцати тикеров, а не через
// generic-ветку. Состояние держит sngsp_test.go.
//
// ОКНО И СХЕМА ШТАТНЫЕ. Расчётное окно — 2023-09-01 … 2026-09-01 (36 месяцев; в окне 35 972
// получасовых бара, из них 25 052 будних; дневная серия в окне 881 свеча, из них 761 будняя).
// Схема прогонов -months 36 -train-months 12 -test-months 6 -min-trades 20 -metric profit_factor,
// четыре фолда встык. Адаптация не нужна: получасовая серия кэша начинается 2023-08-04 (запас 28
// дней), дневная — 2022-08-04 (запас ~13 месяцев). -refresh во время калибровки НЕ запускать: он
// сдвинет обе границы и сделает все замеры несравнимыми.
//
// СЕТКИ ЭТОГО ТИКЕРА ДЕРЖАТСЯ МАКСИМАЛЬНО ШИРОКИМИ — решение владельца 2026-09-01, то же, что по
// BANEP и ASTR. Обрезок осей нет; замеры, которые в узком каталоге были бы основанием вырезать
// край, живут в _comment сеток как предупреждения. Сторожевой тест TestSNGSPGridsStayWide
// запрещает УРЕЗАТЬ ось, а не расширять её. Жёстких инвариантов три: RSILower <= 50,
// RSIPeriod >= 3, ось тренда не порождает пар EMAFast >= EMASlow.
//
// АПРИОР, записанный ДО прогонов. Скринер (pullback_screen_Minutes30_20260804_232456.md, строка 18
// из 99 прошедших вселенную, сразу за отвергнутым KMAZ): оборот 1232 млн ₽, дневной ATR 2.68%,
// баров 35 648, TradesMed 41, PFmed 1.49, Capped 0/24, SilentCfg 0/24, плато 33%, PFmed HO 1.19 на
// 8 сделках. Контрольный прогон дефолтов ядра на расчётном окне: 183 сделки, PF 1.535,
// net +32 599.57 ₽ — ЧЕТВЁРТОЕ место в ряду baseline каталога (YDEX 1.778, LSNGP 1.554, ELFV
// 1.546, SNGSP 1.535, IVAT 1.432, LENT 1.417, SVAV 1.368, BANEP 1.336, BSPB 1.114, ASTR 1.093,
// SIBN 1.027, NVTK 0.984). Веса априору не придаётся: вопрос «предсказывают ли колонки скринера
// исход протокола» каталог закрыл семь раз подряд — не предсказывают ни снизу, ни сверху.
//
// ЛИНЕЙКА ИЗДЕРЖЕК (урок KMAZ) ПРИМЕНЕНА ПЕРВЫМ ДЕЙСТВИЕМ: шаг цены 0.005 ₽ при медианной цене
// 52.03 ₽ = 0.0096% цены — самая дешёвая доля шага в каталоге. Реальный круг заведомо ниже
// моделируемых 0.1%, тикер линейку проходит. Чувствительность baseline: круг 0.1% -> PF 1.535,
// 0.2% -> 1.171, 0.3% -> 0.872 (183 сделки во всех строках).
//
// ЧТО ЗНАЕМ ОБ ИНСТРУМЕНТЕ ДО ПРОГОНОВ (замеры на расчётном окне, полностью —
// reports/_analysis/sngsp_pullback_prep_measurements.md):
//
//   - РЕЖИМ САМЫЙ МЯГКИЙ В КАТАЛОГЕ и это ПИЛА, а не тренд: итог окна −17.5%, максимальная
//     просадка −51.0%, полугодия чередуются +22.6/−22.0/+18.2/−24.0/+8.1/−11.3. Все шесть
//     полугодий baseline прибыльны (PF 1.148, 1.709, 1.262, 1.653, 1.840, 1.871), но 62.8% net
//     дают два последних — разбивка принятой точки по полугодиям обязательна.
//   - ЛИКВИДНОСТЬ ЛУЧШАЯ В КАТАЛОГЕ: оборот по будним дням медиана 1299 млн ₽, p10 520, p90 3044
//     (n = 761); баров в дне медиана 35, дней короче 20 баров 0.8%. Исполнительного риска нет.
//   - ДИВИДЕНДНЫЙ ГЭП КРУПНЕЙШИЙ В КАТАЛОГЕ: 2025-07-17 −14.06%, 2024-07-18 −9.86% (обе — июльские
//     отсечки), 2026-07-16 −2.51%. Гэп −14% это ~5.4 дневных ATR: стопом не держится, движок
//     моделирует это честно (min(level, open)). Экспозиция baseline НУЛЕВАЯ — ни одна сделка не
//     была открыта через отсечку.
//   - СДЕЛКИ ВНУТРИДНЕВНЫЕ, а не многодневные: удержание медиана 9 баров, максимум 38; в
//     календарных днях медиана 0, максимум 1; ночь пережили 3 сделки из 183 (1.6%). Отсюда и
//     нулевая дивидендная экспозиция — риск СПЯЩИЙ, а не отсутствующий, и его будят конфигурации
//     с широким стопом, далёкой целью и выключенным RSI-выходом.
//   - ТРЕНДОВЫЙ ДОПУСК РОВНЫЙ ДО ВЫРОЖДЕНИЯ: доля будних баров с EMAFast > EMASlow в полосе
//     48.0–49.3% по всей таблице 5..40 x 30..250. Максимум PF стоит на паре 40/50 (1.647), где
//     фильтр почти выключен — победа этой пары читается как «тренд бумаге не нужен».
//   - ВОЛАТИЛЬНОСТЬ УМЕРЕННАЯ: дневной ATR(14) медиана 2.60% цены, p10 1.86, p90 3.49.
//   - КАПКАН ШИРОКОГО СТОПА, ВОСЬМОЙ ТИКЕР ПОДРЯД, И ЗДЕСЬ ОН НЕ НАСЫЩАЕТСЯ: при цели 0.6 стоп
//     0.3 -> 1.479/191, 0.5 -> 1.535/183, 1.0 -> 1.728/176, 1.5 -> 1.865/175, 2.0 -> 2.122/175.
//     Число сделок стоит с 0.7, PF растёт в полтора раза — убыток вытесняется в RSI-выход.
//   - ЦЕЛЬ: максимум на 0.5 (ниже дефолта ядра), шире 1.5 дневного ATR недостижима — колонки 1.5,
//     2.0 и 2.5 совпадают побайтово.
//   - ДНЕВНОЙ ГЕЙТ НЕСЁТ ВЕСЬ EDGE: без него 501 сделка при PF 0.944 против 183 при 1.535.
//     Исчерпанная ветка имеет плато 0.7–0.8; свежая ветка монотонно вредит (0 -> 1.535,
//     0.6 -> 1.033).
//   - ОБЪЁМНЫЙ ГЕЙТ НЕРАЗЛИЧИМ с выключенным: весь массив множителя и базы лежит в полосе
//     1.39–1.61 вокруг baseline 1.535.
//   - EDGE НА УМЕРЕННОЙ ГЛУБИНЕ ОТКАТА, рельеф с максимумом В СЕРЕДИНЕ оси: RSI(4)@25 -> 1.674/133,
//     RSI(5)@30 -> 1.720/132, а глубокие уровни на длинных периодах дают шум на 4–12 сделках.
//   - ВЫХОД: максимум полосы на 60 (1.817), рельеф выше монотонно падает до 0.929 на 85; сам
//     RSI-выход критичен — при UseRSIExit 0 PF ниже единицы на всей оси цели. Трейл 0.5 даёт
//     +0.144 PF над baseline, с 1.0 не срабатывает ни разу.
package sngsp

import "tinvest/internal/service/trading_strategy/rsi_pullback/strategy/core"

// Ticker is the instrument this package parameterises.
const Ticker = "SNGSP"

// DefaultParams returns the core baseline: SNGSP is not calibrated yet.
func DefaultParams() core.Params {
	return core.DefaultParams()
}
```

- [ ] **Step 4: Запустить тест пакета**

Run: `go test ./internal/service/trading_strategy/rsi_pullback/strategy/sngsp/ -v`
Expected: PASS оба теста.

- [ ] **Step 5: Зарегистрировать тикер в реестре бэктеста**

В `internal/service/backtest/rsi_pullback_registry.go` добавить импорт рядом с соседними
(порядок в блоке импортов — как у существующих строк):

```go
	rsipullbacksngsp "tinvest/internal/service/trading_strategy/rsi_pullback/strategy/sngsp"
```

и строку в карту реестра на алфавитную позицию, между `rsipullbacksibn` и `rsipullbacksvav`:

```go
	rsipullbacksngsp.Ticker: rsiPullbackBindingFor(rsipullbacksngsp.Ticker, rsipullbacksngsp.DefaultParams),
```

- [ ] **Step 6: Добавить сторожевой тест реестра**

Реестр сторожится не списком, а тестом на тикер (см. `TestRSIPullbackASTRIsRegisteredAndCalibrated`,
`rsi_pullback_registry_test.go:751`). Пока SNGSP отслеживает baseline, тест обратный. Дописать в
`internal/service/backtest/rsi_pullback_registry_test.go` (импорт
`rsipullbacksngsp "tinvest/internal/service/trading_strategy/rsi_pullback/strategy/sngsp"` уже
добавлен в Step 5 в `rsi_pullback_registry.go`, в тесте нужен свой):

```go
// TestRSIPullbackSNGSPTracksBaseline сторожит ЧЕСТНОЕ состояние: SNGSP заведён в реестр до
// калибровки, чтобы прогоны шли через реестр, а не через generic-ветку, и обязан возвращать ровно
// baseline ядра. Тест заменяется снимком литерала в Task 12.
func TestRSIPullbackSNGSPTracksBaseline(t *testing.T) {
	b, ok := rsiPullbackRegistry[rsipullbacksngsp.Ticker]
	if !ok {
		t.Fatal("SNGSP отсутствует в rsiPullbackRegistry: тикер провалится в generic-ветку")
	}
	p, pok := b.DefaultParams().(core.Params)
	if !pok {
		t.Fatalf("SNGSP: DefaultParams() вернул %T, want core.Params", b.DefaultParams())
	}
	if p != core.DefaultParams() {
		t.Fatalf("SNGSP ещё не откалиброван, params обязаны совпадать с baseline:\n got: %+v\nwant: %+v", p, core.DefaultParams())
	}
	if got := b.Build(p).Ticker(); got != "SNGSP" {
		t.Fatalf("Ticker() = %q, want SNGSP", got)
	}
}
```

- [ ] **Step 7: Запустить тесты**

Run: `go test ./internal/service/backtest/ ./internal/service/trading_strategy/rsi_pullback/... -v`
Expected: PASS.

- [ ] **Step 8: Коммит**

```bash
git add internal/service/trading_strategy/rsi_pullback/strategy/sngsp internal/service/backtest
git commit -m "feat(rsi_pullback): пакет SNGSP в состоянии отслеживания baseline"
```

---

### Task 3: Тема `screen` — цена двух гейтов и проверка числа фолдов

**Files:**
- Modify: `data/params/rsi_pullback/sngsp/cal_screen.json` (строка результата в `_comment`)
- Create: отчёты в `reports/SNGSP_screen/` (каталог `reports/` в `.gitignore` — отчёты остаются локальными, в коммит идут только строки результата в `_comment`)

**Interfaces:**
- Consumes: каталог сеток из Task 1, реестр из Task 2.
- Produces: пофолдовые победители пары (`UseDayATRGate`, `UseVolume`) — их читает Task 11 при
  сборке точки.

- [ ] **Step 1: Прогнать тему**

Run:
```bash
go run ./cmd/backtest -ticker SNGSP -strategy rsi_pullback -interval Minutes30 \
  -calibrate data/params/rsi_pullback/sngsp/cal_screen.json -out ./reports/SNGSP_screen \
  -months 36 -train-months 12 -test-months 6 -min-trades 1 -metric profit_factor
```
Expected: отчёт `reports/SNGSP_screen/SNGSP_rsi_pullback_Minutes30_<ts>_walkforward.md`.

- [ ] **Step 2: ПРОВЕРИТЬ ЧИСЛО ФОЛДОВ**

Открыть шапку отчёта. Строка «Фолдов:» обязана показывать **4**.

Если там **3** — это ловушка ASTR (переполнение конца месяца в `AddDate` роняет правый край
последнего фолда за `today`). В этом случае: **остановить работу**, не запускать остальные темы,
доложить владельцу с цитатой шапки отчёта. Схему менять самостоятельно нельзя — правило сборки
точки «≥3 фолда из 4» при трёх фолдах вырождается в «3 из 3», и это меняет планку задним числом.

- [ ] **Step 3: Выписать пофолдовых победителей**

Из отчёта выписать по каждому из четырёх фолдов: выбранную пару (`UseDayATRGate`, `UseVolume`),
in-sample PF, OOS PF, число сделок OOS. Плюс pooled OOS PF и pooled число сделок.

- [ ] **Step 4: Вписать результат в `_comment`**

Дописать после «РЕЗУЛЬТАТ ПРОГОНА 2026-09-01: » одной строкой: путь к отчёту, pooled OOS PF и
сделки, пофолдовые пары и их OOS PF, вывод про то, подтверждается ли точечный замер («без гейта
дня стратегия убыточна») пофолдово.

- [ ] **Step 5: Коммит**

```bash
git add data/params/rsi_pullback/sngsp/cal_screen.json
git commit -m "feat(rsi_pullback): SNGSP, тема screen — цена гейтов"
```

---

### Task 4: Тема `entry` — ключевая, рельеф с максимумом в середине

**Files:**
- Modify: `data/params/rsi_pullback/sngsp/cal_entry.json` (строка результата в `_comment`)
- Create: отчёты в `reports/SNGSP_entry/`

**Interfaces:**
- Consumes: каталог из Task 1, реестр из Task 2.
- Produces: пофолдовые победители `RSIPeriod`, `RSILower`, `RSIUpper` — их читает Task 11.

- [ ] **Step 1: Прогнать тему**

Run:
```bash
go run ./cmd/backtest -ticker SNGSP -strategy rsi_pullback -interval Minutes30 \
  -calibrate data/params/rsi_pullback/sngsp/cal_entry.json -out ./reports/SNGSP_entry \
  -months 36 -train-months 12 -test-months 6 -min-trades 20 -metric profit_factor
```
Expected: 432 комбинации на фолд, отчёт `..._walkforward.md`. Прогон длинный — это нормально.

- [ ] **Step 2: Записать вердикт по планке для этой темы**

Тема ключевая. Выписать: pooled OOS PF, число сделок в пуле, пофолдовые значения ведущей оси
`RSILower`. Планка по этой теме взята, если pooled OOS PF ≥ 1.5 при ≥ 20 сделках **и** `RSILower`
совпал в ≥ 3 фолдах из 4. Записать оба критерия раздельно — каталог знает случаи, где взят один и
провален другой (ASTR: PF провален, устойчивость взята; BSPB: провалены оба).

- [ ] **Step 3: Проверить, где сидят победители относительно точечного рельефа**

Точечный рельеф (полная история, поверх дефолтов): максимум `RSILower` в СЕРЕДИНЕ оси
(RSI(4)@25 → 1.674, RSI(5)@30 → 1.720), глубокие края — шум на 4–12 сделках. Записать, совпали ли
фолды с этим рельефом или разошлись, и в какую сторону. Рельеф на полной истории доказательством
не является (§8.0.1 `strategy.md`) — это сверка ожидания с фактом, а не критерий.

- [ ] **Step 4: Вписать результат в `_comment` и закоммитить**

```bash
git add data/params/rsi_pullback/sngsp/cal_entry.json
git commit -m "feat(rsi_pullback): SNGSP, тема entry — ключевая"
```

---

### Task 5: Тема `trend` — вторая ключевая, ожидается вырожденный победитель

**Files:**
- Modify: `data/params/rsi_pullback/sngsp/cal_trend.json`
- Create: отчёты в `reports/SNGSP_trend/`

**Interfaces:**
- Consumes: каталог из Task 1.
- Produces: пофолдовые победители `EMAFast`, `EMASlow` — их читает Task 11.

- [ ] **Step 1: Прогнать тему**

Run:
```bash
go run ./cmd/backtest -ticker SNGSP -strategy rsi_pullback -interval Minutes30 \
  -calibrate data/params/rsi_pullback/sngsp/cal_trend.json -out ./reports/SNGSP_trend \
  -months 36 -train-months 12 -test-months 6 -min-trades 20 -metric profit_factor
```

- [ ] **Step 2: Записать вердикт по планке для этой темы**

Ведущая ось — `EMASlow`. Критерии те же, что в Task 4 Step 2, записываются раздельно.

- [ ] **Step 3: Проверить вырожденность победителя**

Если победитель фолда стоит на паре, где быстрая близка к медленной (40/50, 30/50), записать это
прямым текстом как «фильтр тренда вырожден, бумаге он не нужен», а не как найденный параметр.
Опора: трендовый допуск ровный (48.0–49.3% баров по всей таблице), то есть пара EMA на SNGSP почти
не меняет то, сколько времени вход разрешён, — она меняет только форму границы.

- [ ] **Step 4: Вписать результат в `_comment` и закоммитить**

```bash
git add data/params/rsi_pullback/sngsp/cal_trend.json
git commit -m "feat(rsi_pullback): SNGSP, тема trend — вторая ключевая"
```

---

### Task 6: Темы `day` и `day_spent` — дневной гейт

**Files:**
- Modify: `data/params/rsi_pullback/sngsp/cal_day.json`, `.../cal_day_spent.json`
- Create: отчёты в `reports/SNGSP_day/`, `reports/SNGSP_day_spent/`

**Interfaces:**
- Consumes: каталог из Task 1.
- Produces: пофолдовые победители `FreshDayATR`, `SpentDayATR` из двух тем — их сравнивает Task 11.

- [ ] **Step 1: Прогнать обе темы**

Run:
```bash
go run ./cmd/backtest -ticker SNGSP -strategy rsi_pullback -interval Minutes30 \
  -calibrate data/params/rsi_pullback/sngsp/cal_day.json -out ./reports/SNGSP_day \
  -months 36 -train-months 12 -test-months 6 -min-trades 20 -metric profit_factor

go run ./cmd/backtest -ticker SNGSP -strategy rsi_pullback -interval Minutes30 \
  -calibrate data/params/rsi_pullback/sngsp/cal_day_spent.json -out ./reports/SNGSP_day_spent \
  -months 36 -train-months 12 -test-months 6 -min-trades 20 -metric profit_factor
```

- [ ] **Step 2: Сверить две темы между собой**

`cal_day.json` меряет обе ветки, `cal_day_spent.json` — только исчерпанную при `FreshDayATR 0`.
Выписать победителей `SpentDayATR` из обеих тем и проверить, согласны ли они. Расхождение — не
ошибка: тема с открытой свежей веткой видит другую выборку.

- [ ] **Step 3: Проверить ожидание «ноль побеждает»**

Точечный замер говорит, что свежая ветка вредит монотонно (0 → 1.535, 0.6 → 1.033). Записать,
подтвердили ли фолды. Каталог знает контрпример: на ASTR ноль впервые не победил устойчиво (2 из
4).

- [ ] **Step 4: Проверить шум на хвосте исчерпанной ветки**

Уровни 1.25 и 1.5 стоят на 39 и 17 сделках за 36 месяцев. Если фолд выбрал такой уровень —
записать число сделок OOS этого фолда рядом с его PF: победа на тающей выборке это шум, а не
находка.

- [ ] **Step 5: Вписать результаты в оба `_comment` и закоммитить**

```bash
git add data/params/rsi_pullback/sngsp/cal_day.json data/params/rsi_pullback/sngsp/cal_day_spent.json
git commit -m "feat(rsi_pullback): SNGSP, темы day и day_spent"
```

---

### Task 7: Тема `volume` — объёмный гейт на самой ликвидной бумаге каталога

**Files:**
- Modify: `data/params/rsi_pullback/sngsp/cal_volume.json`
- Create: отчёты в `reports/SNGSP_volume/`

**Interfaces:**
- Consumes: каталог из Task 1.
- Produces: пофолдовые победители `VolMult`, `VolBaseDays` — их читает Task 11.

- [ ] **Step 1: Прогнать тему**

Run:
```bash
go run ./cmd/backtest -ticker SNGSP -strategy rsi_pullback -interval Minutes30 \
  -calibrate data/params/rsi_pullback/sngsp/cal_volume.json -out ./reports/SNGSP_volume \
  -months 36 -train-months 12 -test-months 6 -min-trades 20 -metric profit_factor
```

- [ ] **Step 2: Сравнить с темой `screen`**

Тема `volume` форсирует `UseVolume = 1`, тема `screen` меряет гейт против выключенного. Сравнение
их pooled OOS PF — **неконтролируемое A/B** (в `screen` конфигурации гейта смешаны по фолдам), и
записывать его как «гейт выиграл/проиграл» нельзя. Единственный корректный вывод из пары —
сходится ли ось вообще.

- [ ] **Step 3: Проверить выброс на множителе 3.0**

Точечный замер даёт 3.0 → 2.129 на 48 сделках за 36 месяцев (16 на обучающее окно). Если фолд
выбрал 3.0, выписать число сделок его OOS-окна: победа под порогом `-min-trades 20` — процедурный
артефакт.

- [ ] **Step 4: Вписать результат в `_comment` и закоммитить**

```bash
git add data/params/rsi_pullback/sngsp/cal_volume.json
git commit -m "feat(rsi_pullback): SNGSP, тема volume"
```

---

### Task 8: Тема `vol_window` — восьмая проверка гипотезы «ликвидность → дефолт»

**Files:**
- Modify: `data/params/rsi_pullback/sngsp/cal_vol_window.json`
- Create: отчёты в `reports/SNGSP_vol_window/`

**Interfaces:**
- Consumes: каталог из Task 1.
- Produces: пофолдовые победители `VolLookbackBars` — их читает Task 11.

- [ ] **Step 1: Прогнать тему**

Run:
```bash
go run ./cmd/backtest -ticker SNGSP -strategy rsi_pullback -interval Minutes30 \
  -calibrate data/params/rsi_pullback/sngsp/cal_vol_window.json -out ./reports/SNGSP_vol_window \
  -months 36 -train-months 12 -test-months 6 -min-trades 20 -metric profit_factor
```

- [ ] **Step 2: Записать вердикт по гипотезе каталога**

Гипотеза: чем ликвиднее бумага, тем ближе оптимум окна к дефолту ядра (3 бара). Данные каталога:
ELFV (18.9% коротких дней) — максимум на дефолте; DIAS (11.9%) — на 12; BSPB (7.7%) — на 5; ASTR —
ось не сошлась вовсе. SNGSP ликвиднее всех (0.8% коротких дней), точечный максимум при этом стоит
на самом широком крае 24. Записать, что показали фолды, и подтверждают ли они гипотезу.

- [ ] **Step 3: Проверить эквивалентность выключенному гейту**

Широкое окно ослабляет гейт. Если победитель фолда стоит на 16 или 24, проверить, отличаются ли
его числа от `UseVolume = 0` заметно. При неразличимых числах в точку идёт **выключенный** гейт
(конфигурация проще при том же результате) — это прецедент ASTR (Ruling 5).

- [ ] **Step 4: Вписать результат в `_comment` и закоммитить**

```bash
git add data/params/rsi_pullback/sngsp/cal_vol_window.json
git commit -m "feat(rsi_pullback): SNGSP, тема vol_window"
```

---

### Task 9: Тема `risk` — стоп, цель и обязательная проверка капкана

**Files:**
- Modify: `data/params/rsi_pullback/sngsp/cal_risk.json`
- Create: отчёты в `reports/SNGSP_risk/`, `reports/SNGSP_risk_full/`

**Interfaces:**
- Consumes: каталог из Task 1.
- Produces: пофолдовые победители `StopDailyATR`, `TPDailyATR` + доля SL-выходов победителя — их
  читает Task 11.

- [ ] **Step 1: Прогнать тему**

Run:
```bash
go run ./cmd/backtest -ticker SNGSP -strategy rsi_pullback -interval Minutes30 \
  -calibrate data/params/rsi_pullback/sngsp/cal_risk.json -out ./reports/SNGSP_risk \
  -months 36 -train-months 12 -test-months 6 -min-trades 20 -metric profit_factor
```

- [ ] **Step 2: Выписать пофолдовых победителей**

По каждому фолду: `StopDailyATR`, `TPDailyATR`, in-sample PF, OOS PF, сделки OOS. Плюс pooled.

- [ ] **Step 3: ОБЯЗАТЕЛЬНАЯ ПРОВЕРКА КАПКАНА**

Если победившее значение стопа шире 0.7, прогнать полную историю на точке «победитель стопа +
победитель цели + остальные дефолты» и посчитать долю выходов по SL:

```bash
cat > /tmp/sngsp_stop_probe.json <<'JSON'
{"RSIPeriod":4,"RSILower":30,"RSIUpper":70,"EMAFast":10,"EMASlow":100,"DailyATRPeriod":14,
 "UseDayATRGate":1,"FreshDayATR":0,"SpentDayATR":0.8,"StopDailyATR":<победитель>,
 "TPDailyATR":<победитель>,"UseVolume":0,"VolBaseDays":14,"VolLookbackBars":3,"VolMult":1.2,
 "UseRSIExit":1,"UseTrail":0,"TrailDailyATR":0}
JSON

go run ./cmd/backtest -ticker SNGSP -strategy rsi_pullback -interval Minutes30 \
  -params /tmp/sngsp_stop_probe.json -out ./reports/SNGSP_risk_full -months 36
```

Взять `_trades.csv` из отчёта и посчитать долю строк с `reason == "SL"`:

```bash
python3 -c "
import csv,glob
f=sorted(glob.glob('reports/SNGSP_risk_full/*_trades.csv'))[-1]
rows=list(csv.DictReader(open(f)))
sl=[r for r in rows if r['reason']=='SL']
print(f'{len(sl)}/{len(rows)} = {100*len(sl)/len(rows):.1f}% выходов по SL')
"
```

Опорное число baseline: **37 из 183 = 20.2%** при стопе 0.5. Падение доли SL при росте PF означает
капкан: убыток не исчез, он утёк в RSI-выход. Записать оба числа рядом.

- [ ] **Step 4: ПРОВЕРКА УДЕРЖАНИЯ (специфика SNGSP)**

На том же `_trades.csv` посчитать удержание и ночную экспозицию:

```bash
python3 -c "
import csv,glob,datetime as dt,statistics as st
f=sorted(glob.glob('reports/SNGSP_risk_full/*_trades.csv'))[-1]
rows=list(csv.DictReader(open(f)))
days=[(dt.datetime.fromisoformat(r['exit_time'])-dt.datetime.fromisoformat(r['entry_time'])).days for r in rows]
print('календарных дней: медиана',st.median(days),'максимум',max(days),
      'доля сделок с ночёвкой',round(100*sum(1 for d in days if d>=1)/len(days),1),'%')
for c in ['2024-07-18','2025-07-17','2026-07-16']:
    d=dt.date.fromisoformat(c)
    hit=[r for r in rows if dt.datetime.fromisoformat(r['entry_time']).date()<d<=dt.datetime.fromisoformat(r['exit_time']).date()]
    print('открытых через отсечку',c,':',len(hit))
"
```

Опорные числа baseline: медиана 0 дней, максимум 1, ночёвка у 1.6% сделок, через отсечки ноль
открытых. Если победитель темы удлиняет удержание — это будит дивидендный риск, и факт идёт в
Task 15 как принятый риск с календарным предупреждением.

- [ ] **Step 5: Вписать результат в `_comment` и закоммитить**

```bash
git add data/params/rsi_pullback/sngsp/cal_risk.json
git commit -m "feat(rsi_pullback): SNGSP, тема risk и проверка капкана"
```

---

### Task 10: Темы `exit` и `trail` — выходы

**Files:**
- Modify: `data/params/rsi_pullback/sngsp/cal_exit.json`, `.../cal_trail.json`
- Create: отчёты в `reports/SNGSP_exit/`, `reports/SNGSP_trail/`

**Interfaces:**
- Consumes: каталог из Task 1.
- Produces: пофолдовые победители `RSIUpper` (тема exit), `UseRSIExit`, `TrailDailyATR` (тема
  trail) — их читает Task 11.

- [ ] **Step 1: Прогнать обе темы**

Run:
```bash
go run ./cmd/backtest -ticker SNGSP -strategy rsi_pullback -interval Minutes30 \
  -calibrate data/params/rsi_pullback/sngsp/cal_exit.json -out ./reports/SNGSP_exit \
  -months 36 -train-months 12 -test-months 6 -min-trades 20 -metric profit_factor

go run ./cmd/backtest -ticker SNGSP -strategy rsi_pullback -interval Minutes30 \
  -calibrate data/params/rsi_pullback/sngsp/cal_trail.json -out ./reports/SNGSP_trail \
  -months 36 -train-months 12 -test-months 6 -min-trades 20 -metric profit_factor
```

- [ ] **Step 2: Сверить тему `exit` с осью `RSIUpper` внутри темы `entry`**

`RSIUpper` свипуется дважды: в `cal_entry.json` (вместе с периодом и уровнем) и в `cal_exit.json`
(в одиночку). Выписать победителей обеих тем. Расхождение ожидаемо и означает взаимодействие осей;
при сборке точки поле берётся из **темы, которая его меряет отдельно** (`exit`), а результат
`entry` идёт в запись как контекст.

- [ ] **Step 3: Проверить нижний край оси выхода**

Если победитель стоит на 50 или 55, записать: удержание при низком уровне выхода сокращается ещё
сильнее, а на SNGSP оно и так внутридневное. Это меняет характер стратегии — из «многодневного
отката» в «внутридневной отскок», и в доке пакета (Task 15) это должно быть названо прямо.

- [ ] **Step 4: Проверить, не победил ли `UseRSIExit = 0`**

Точечный замер: без RSI-выхода PF ниже единицы на всей оси цели. Если какой-то фолд выбрал ноль —
выписать его OOS PF и сделки и записать это как аномалию фолда, требующую проверки экспозиции
через отсечки (Task 11 Step 8).

- [ ] **Step 5: Вписать результаты в оба `_comment` и закоммитить**

```bash
git add data/params/rsi_pullback/sngsp/cal_exit.json data/params/rsi_pullback/sngsp/cal_trail.json
git commit -m "feat(rsi_pullback): SNGSP, темы exit и trail"
```

---

### Task 11: Сборка точки, её walk-forward и проверка стоп-условия

**Files:**
- Create: `data/params/rsi_pullback/sngsp/plateau_point.json`
- Create: `data/params/rsi_pullback/sngsp/plateau_neighbour_*.json` (по одному на проверяемого
  соседа)
- Create: отчёты в `reports/SNGSP_point_oos/`, `reports/SNGSP_point_comm/`,
  `reports/SNGSP_point_full/`, `reports/SNGSP_neighbour_*/`
- Create: `docs/superpowers/plans/task-11-report-sngsp.md` (рабочая записка с полными выкладками)

**Interfaces:**
- Consumes: пофолдовых победителей всех десяти тем (Tasks 3–10).
- Produces: `plateau_point.json` — его читает Task 12 (литерал) и Task 16 (сверка).

- [ ] **Step 1: Собрать точку по правилу «≥3 фолда из 4»**

Для каждого из восемнадцати полей `core.Params`: взять тему, которая его меряет; посмотреть
победителей по четырём фолдам; принять значение, только если за него высказались **не менее трёх
фолдов из четырёх**; иначе оставить дефолт ядра. Соответствие «поле → тема»:

| Поле | Тема |
|---|---|
| `RSIPeriod`, `RSILower` | `entry` |
| `RSIUpper` | `exit` (в `entry` идёт как контекст) |
| `EMAFast`, `EMASlow` | `trend` |
| `DailyATRPeriod` | нигде не свипуется — дефолт 14 |
| `UseDayATRGate` | `screen` |
| `FreshDayATR`, `SpentDayATR` | `day` + `day_spent` |
| `StopDailyATR`, `TPDailyATR` | `risk` |
| `UseVolume` | `screen` |
| `VolMult`, `VolBaseDays` | `volume` |
| `VolLookbackBars` | `vol_window` |
| `UseRSIExit`, `UseTrail`, `TrailDailyATR` | `trail` |

Правило для `UseTrail`: если тема выбрала `TrailDailyATR = 0` (или значение, при котором трейл не
срабатывает — на SNGSP это 1.0 и выше), в точку идёт `UseTrail = 0` и `TrailDailyATR = 0`:
конфигурация проще при идентичном результате (прецедент ASTR).

- [ ] **Step 2: Записать деривацию поле за полем**

Создать `plateau_point.json` с сеткой из одного значения на ключ (это требование
`TestRSIPullbackPlateauFilesArePoints`) и `_comment`, где для КАЖДОГО поля написано: из какой темы,
какие значения выбрали четыре фолда, взято большинство или поле упало на дефолт. Плюс оговорка о
подглядывании: точку собрал человек, видевший всю историю, поэтому её числа нельзя сравнивать с
pooled OOS тем как замену процедуре отбора.

- [ ] **Step 3: Прогнать walk-forward точки**

Run:
```bash
go run ./cmd/backtest -ticker SNGSP -strategy rsi_pullback -interval Minutes30 \
  -calibrate data/params/rsi_pullback/sngsp/plateau_point.json -out ./reports/SNGSP_point_oos \
  -months 36 -train-months 12 -test-months 6 -min-trades 20 -metric profit_factor
```
Выписать: pooled OOS PF, число сделок, win rate, compounded, пофолдовые OOS PF и сделки, наличие
вырожденных фолдов.

- [ ] **Step 4: Проверить пункты 1 и 2 стоп-условия**

Пункт 1: pooled OOS PF ≥ 1.0. Пункт 2: ≥ 20 сделок. Если любой провален — **остановиться**,
записать числа, доложить владельцу; задачи 12–16 не выполняются.

- [ ] **Step 5: Проверить пункт 3 стоп-условия — удвоенные издержки**

Run:
```bash
go run ./cmd/backtest -ticker SNGSP -strategy rsi_pullback -interval Minutes30 \
  -calibrate data/params/rsi_pullback/sngsp/plateau_point.json -out ./reports/SNGSP_point_comm \
  -months 36 -train-months 12 -test-months 6 -min-trades 20 -metric profit_factor -commission 0.001
```
Записать pooled OOS PF и долю потери относительно основного прогона. Опорное число: baseline
теряет 23.7% PF при удвоении круга (1.535 → 1.171).

- [ ] **Step 6: Соседи плато**

Для трёх осей, которые дали принятые значения (как минимум для `StopDailyATR` и для ведущей оси
входа), создать файлы-соседи: копия точки, где ОДНО поле сдвинуто на соседний шаг сетки
соответствующей темы. Прогнать каждого той же командой, что точку, с `-out
./reports/SNGSP_neighbour_<имя>`. Записать вердикт по каждой оси раздельно: плато (числа рядом),
склон (сосед лучше при меньшей выборке), пик (оба соседа хуже). Пик записывается как риск —
структурная неустойчивость оси.

- [ ] **Step 7: Полугодия принятой точки**

Прогнать точку на полной истории без train/test, чтобы получить `_trades.csv`:

```bash
go run ./cmd/backtest -ticker SNGSP -strategy rsi_pullback -interval Minutes30 \
  -params /tmp/sngsp_point.json -out ./reports/SNGSP_point_full -months 36
```
(`/tmp/sngsp_point.json` — плоский JSON тех же восемнадцати полей.)

Разложить сделки по полугодиям расчётного окна (границы — в Global Constraints) и посчитать по
каждому: число сделок, net, PF, долю от общего net. Критерий: **ни одно полугодие не даёт больше
половины net**. Опорная картина baseline: 4.1 / 6.1 / 15.8 / 11.2 / 28.5 / 34.3%.

- [ ] **Step 8: Дивидендная экспозиция принятой точки**

На том же `_trades.csv` повторить проверку из Task 9 Step 4: удержание (медиана, максимум, доля
сделок с ночёвкой) и число сделок, открытых через 2024-07-18, 2025-07-17, 2026-07-16.

Если открытых через отсечку сделок **ноль** — риск остаётся спящим, и это записывается как
свойство точки, а не как гарантия. Если хотя бы одна — посчитать её PnL и записать как
реализованный дивидендный риск: в живой торговле такая сделка получает гэп −10…−14% ниже стопа.

- [ ] **Step 9: Записать рабочую записку**

Создать `docs/superpowers/plans/task-11-report-sngsp.md` со всеми выкладками шагов 1–8: команды,
пути к отчётам, таблицы фолдов, полугодий, соседей, чисел капкана и удержания. Записка —
источник для доки пакета (Task 15).

- [ ] **Step 10: Коммит**

```bash
git add data/params/rsi_pullback/sngsp docs/superpowers/plans/task-11-report-sngsp.md
git commit -m "feat(rsi_pullback): SNGSP, принятая точка и её замеры"
```

---

### Task 12: Литерал в пакете и снимок

**Files:**
- Modify: `internal/service/trading_strategy/rsi_pullback/strategy/sngsp/sngsp.go`
- Modify: `internal/service/trading_strategy/rsi_pullback/strategy/sngsp/sngsp_test.go`

**Interfaces:**
- Consumes: `plateau_point.json` из Task 11.
- Produces: `sngsp.DefaultParams()` как явный литерал — его читают Task 13 и Task 14.

- [ ] **Step 1: Заменить тест baseline-состояния снимком литерала**

В `sngsp_test.go` удалить `TestParamsTrackTheBaselineUntilCalibrated` и записать снимок:

```go
// TestParamsAreTheCalibratedSnapshot прибивает принятый 2026-09-01 литерал. Тест падает и на
// молчаливом дрейфе полей, и на откате к core.DefaultParams(): связь с baseline разорвана
// осознанно, и любое её восстановление обязано быть видимым в диффе.
func TestParamsAreTheCalibratedSnapshot(t *testing.T) {
	want := core.Params{
		RSIPeriod:       <из точки>,
		RSILower:        <из точки>,
		RSIUpper:        <из точки>,
		EMAFast:         <из точки>,
		EMASlow:         <из точки>,
		DailyATRPeriod:  14,
		UseDayATRGate:   <из точки>,
		FreshDayATR:     <из точки>,
		SpentDayATR:     <из точки>,
		StopDailyATR:    <из точки>,
		TPDailyATR:      <из точки>,
		UseVolume:       <из точки>,
		VolBaseDays:     <из точки>,
		VolLookbackBars: <из точки>,
		VolMult:         <из точки>,
		UseRSIExit:      <из точки>,
		UseTrail:        <из точки>,
		TrailDailyATR:   <из точки>,
	}
	if got := DefaultParams(); got != want {
		t.Fatalf("литерал SNGSP разошёлся со снимком:\n got: %+v\nwant: %+v", got, want)
	}
}

// TestParamsDifferFromTheCoreBaseline сторожит сам факт калибровки: если литерал вернулся к
// дефолтам ядра, значит правка откатила работу целиком, и это не должно проходить молча.
func TestParamsDifferFromTheCoreBaseline(t *testing.T) {
	if DefaultParams() == core.DefaultParams() {
		t.Fatal("SNGSP откалиброван — DefaultParams() не может совпадать с core.DefaultParams()")
	}
}
```

Если точка совпала с дефолтами ядра по всем полям (правило «≥3 из 4» не приняло ни одного поля),
второй тест не пишется, а в доке пакета это состояние описывается прямым текстом.

- [ ] **Step 2: Запустить тест и убедиться, что он падает**

Run: `go test ./internal/service/trading_strategy/rsi_pullback/strategy/sngsp/ -v`
Expected: FAIL — `DefaultParams()` всё ещё возвращает `core.DefaultParams()`.

- [ ] **Step 3: Поставить литерал**

В `sngsp.go` заменить тело `DefaultParams()` явным `core.Params{...}` со значениями точки.

- [ ] **Step 4: Заменить сторожевой тест реестра**

В `internal/service/backtest/rsi_pullback_registry_test.go` заменить
`TestRSIPullbackSNGSPTracksBaseline` (Task 2 Step 6) на снимковый вариант, по образцу
`TestRSIPullbackASTRIsRegisteredAndCalibrated` (`:751`):

```go
// TestRSIPullbackSNGSPIsRegisteredAndCalibrated сторожит, что SNGSP живёт в rsiPullbackRegistry и
// несёт СВОЙ литерал, а не baseline ядра. TestRSIPullbackSNGSPTracksBaseline, державший прежнее
// состояние, заменён этим снимком 2026-09-01.
func TestRSIPullbackSNGSPIsRegisteredAndCalibrated(t *testing.T) {
	b, ok := rsiPullbackRegistry[rsipullbacksngsp.Ticker]
	if !ok {
		t.Fatal("SNGSP отсутствует в rsiPullbackRegistry: тикер провалится в generic-ветку")
	}
	p, pok := b.DefaultParams().(core.Params)
	if !pok {
		t.Fatalf("SNGSP: DefaultParams() вернул %T, want core.Params", b.DefaultParams())
	}
	if p == core.DefaultParams() {
		t.Fatal("SNGSP вернул baseline: откалиброванный тикер обязан иметь собственный литерал")
	}
	if want := rsipullbacksngsp.DefaultParams(); p != want {
		t.Fatalf("SNGSP params = %+v, want литерал пакета %+v", p, want)
	}
	if got := b.Build(p).Ticker(); got != "SNGSP" {
		t.Fatalf("Ticker() = %q, want SNGSP", got)
	}
}
```

Если точка совпала с дефолтами ядра по всем полям, этот тест не пишется, а
`TestRSIPullbackSNGSPTracksBaseline` остаётся — вместе с записью в доке пакета о том, что правило
«≥3 из 4» не приняло ни одного поля.

- [ ] **Step 5: Запустить тесты**

Run: `go test ./internal/service/backtest/ ./internal/service/trading_strategy/rsi_pullback/... -v`
Expected: PASS.

- [ ] **Step 6: Коммит**

```bash
git add internal/service/trading_strategy/rsi_pullback/strategy/sngsp internal/service/backtest
git commit -m "feat(rsi_pullback): SNGSP откалиброван — литерал вместо отслеживания baseline"
```

---

### Task 13: Реестр живого раннера

**Files:**
- Modify: `internal/service/trading_strategy/rsi_pullback/live/registry.go`
- Modify: `internal/service/trading_strategy/rsi_pullback/live/registry_test.go` (если в нём есть
  список ожидаемых тикеров)

**Interfaces:**
- Consumes: `sngsp.Ticker`, `sngsp.DefaultParams()` из Task 12.
- Produces: запись в `paramsByTicker` — её читает `ParamsFor` живого раннера и `cmd/pullparity`.

- [ ] **Step 1: Добавить импорт и запись в карту**

Импорт `"tinvest/internal/service/trading_strategy/rsi_pullback/strategy/sngsp"` и строка в
`paramsByTicker` следом за `astr.Ticker` (`registry.go:295`):

```go
	sngsp.Ticker: sngsp.DefaultParams(),
```

- [ ] **Step 2: Дописать абзац комментария карты**

Перед картой стоят абзацы про каждый заведённый тикер, на английском. Дописать абзац про SNGSP в
том же стиле: штатная схема 36/12/6 (впервые за шесть тикеров без адаптации), решение о
максимально широких сетках, вердикт по планке, четвёртый baseline каталога (1.535), каноническая
процедура тем, снятые риски (лучшая ликвидность каталога, самая дешёвая доля шага цены), и
остаточные риски — дивидендный гэп июльской отсечки до −14.06% и капкан широкого стопа.

- [ ] **Step 3: Запустить тесты пакета**

Run: `go test ./internal/service/trading_strategy/rsi_pullback/live/... -v`
Expected: PASS. Два теста прямо сторожат эту пару задач: `TestEveryDefaultTickerIsRegistered`
(каждый тикер боевой вселенной обязан быть в карте) и
`TestBaselineTrackingTickersStayOutOfTheDefaultUniverse` (тикер, всё ещё отслеживающий baseline, в
боевую вселенную попасть не может). Второй — причина, по которой Task 12 идёт раньше Task 14.

- [ ] **Step 4: Коммит**

```bash
git add internal/service/trading_strategy/rsi_pullback/live
git commit -m "feat(rsi_pullback): SNGSP в реестре живого раннера"
```

---

### Task 14: Боевая вселенная

**Files:**
- Modify: `internal/config/rsi_pullback.go` (комментарий-абзац + список `Tickers`, строка 267)
- Modify: `internal/config/rsi_pullback_test.go` (список `want` в тесте дефолта вселенной)
- Modify: `env/prod.env`, `env/prod.env.example`, `env/local.env.example`
- Modify: `docs/rsi_pullback/live.md` (§8, значение дефолта `RSI_PULLBACK_TICKERS` в таблице
  конфигурации — это механика, а не пер-тикерная запись)

**Interfaces:**
- Consumes: литерал из Task 12, реестр из Task 13.
- Produces: `RSI_PULLBACK_TICKERS` из двадцати тикеров.

- [ ] **Step 1: Проверить стоп-условие ещё раз**

Перечитать числа Task 11 Steps 4–5. Все три пункта стоп-условия не сработали — иначе эта задача не
выполняется вовсе.

- [ ] **Step 2: Обновить тест дефолта**

В `internal/config/rsi_pullback_test.go` в список `want` добавить `"SNGSP"` двадцатым.

- [ ] **Step 3: Запустить тест и убедиться, что падает**

Run: `go test ./internal/config/ -v`
Expected: FAIL — дефолт ещё из девятнадцати тикеров.

- [ ] **Step 4: Обновить дефолт и env-файлы**

В `internal/config/rsi_pullback.go` дописать `"SNGSP"` в конец списка `Tickers` (строка 267) и
добавить перед ним комментарий-абзац в стиле соседних (ASTR, BANEP, YDEX): когда заведён, вердикт
по планке, числа точки, **тип риска** (у SNGSP он дивидендный — июльская отсечка до −14.06%, и
режимный он НЕ является: режим самый мягкий в каталоге), каноническая процедура тем, штатная схема
окна и решение о широких сетках.

Во всех трёх env-файлах дописать `,SNGSP` в конец `RSI_PULLBACK_TICKERS`. Текущее значение:
`UGLD,T,GAZP,DOMRF,FESH,WUSH,LENT,RENI,NVTK,LSNGP,IVAT,SVAV,SIBN,ELFV,DIAS,BSPB,YDEX,BANEP,ASTR`.

- [ ] **Step 5: Обновить значение дефолта в `live.md` §8**

В таблице конфигурации строка `RSI_PULLBACK_TICKERS` содержит дефолтный список. Дописать `,SNGSP`.
Ничего пер-тикерного (вердиктов, PF, дат) в `docs/rsi_pullback/` не добавлять — правило CLAUDE.md.

- [ ] **Step 6: Запустить тесты**

Run: `go test ./internal/config/ ./internal/service/trading_strategy/rsi_pullback/... -v`
Expected: PASS.

- [ ] **Step 7: Коммит**

```bash
git add internal/config env docs/rsi_pullback/live.md
git commit -m "feat(rsi_pullback): завести SNGSP в боевую вселенную"
```

---

### Task 15: Дока пакета — разбор калибровки и принятый риск

**Files:**
- Modify: `internal/service/trading_strategy/rsi_pullback/strategy/sngsp/sngsp.go` (док-комментарий
  пакета)
- Modify: `docs/rsi_pullback/strategy.md` — **только если** SNGSP дал механический вывод (§8.0.1)

**Interfaces:**
- Consumes: рабочую записку Task 11.
- Produces: пер-тикерную запись, которую читает следующий калибратор.

- [ ] **Step 1: Переписать док-комментарий пакета**

Заменить блок «СОСТОЯНИЕ: КАЛИБРОВКА НЕ ПРОВОДИЛАСЬ» на разбор в форме, принятой каталогом
(образец — `strategy/astr/astr.go`). Обязательные разделы:

- **СОСТОЯНИЕ: ОТКАЛИБРОВАН 2026-09-01** — как собран литерал, чем прибит;
- **ВЕРДИКТ ПО ПЛАНКЕ** — по каждой из двух ключевых тем раздельно оба критерия (PF и
  устойчивость), прямым текстом, без сглаживания;
- **ПРИНЯТАЯ ТОЧКА** — pooled OOS PF, сделки, win rate, пофолдовые числа, фолды-выбросы;
- **ПОЛУГОДИЯ** — таблица из Task 11 Step 7 и вердикт по критерию «ни одно полугодие не даёт
  больше половины net»;
- **ПЛАТО** — вердикт по каждой проверенной оси раздельно (плато / склон / пик);
- **КАПКАН ШИРОКОГО СТОПА** — доля SL-выходов точки против 20.2% у baseline;
- **УДЕРЖАНИЕ И ДИВИДЕНДНАЯ ЭКСПОЗИЦИЯ** — числа Task 11 Step 8;
- **ПРИНЯТЫЙ РИСК И УСЛОВИЕ ПЕРЕСМОТРА** — перечислить принятые риски (как минимум: дивидендный
  гэп июльской отсечки; капкан широкого стопа, если стоп широкий; непройденная планка, если она не
  пройдена; концентрация net в двух последних полугодиях, если она подтвердилась на точке).
  Условие пересмотра литерала: первая живая просадка глубже бэктестовой **либо** два подряд
  убыточных квартала живой торговли;
- **ОГОВОРКА О ПОДГЛЯДЫВАНИИ** — точку собрал человек, видевший всю историю.

Оставить в доке уже написанные блоки про окно, схему, широкие сетки, априор и свойства
инструмента — они не устаревают.

- [ ] **Step 2: Отдельно записать дивидендный риск для живой торговли**

В блоке принятых рисков назвать три даты известных отсечек (2024-07-18, 2025-07-17, 2026-07-16),
величину гэпов и вывод: SNGSP — первый тикер каталога, где календарь отсечек имеет смысл держать в
голове при живой торговле, даже если бэктест экспозиции не показал (прецедент противоположного
знака — ASTR, где дивидендный риск снят замером; прецедент того же знака — BANEP, где отсечки
стоили −12.74% / −10.67% / −5.28% за ночь).

- [ ] **Step 3: Механический вывод в `docs/rsi_pullback/strategy.md` — только если он есть**

В `docs/` идёт **только механика**, применимая к следующему тикеру, и только если SNGSP такую дал.
Кандидаты, известные заранее: (а) правило выбора между двумя расширениями оси тренда, когда
расширение вниз по `EMASlow` и расширение вверх по `EMAFast` несовместимы из-за инварианта
`EMAFast < EMASlow`; (б) контрольная строка цели, которую требует
`TestRSIPullbackGridControlPoints` при широкой оси стопа, и как её не спутать с кандидатом;
(в) внутридневное удержание у стратегии, объявленной многодневной, как отдельное свойство бумаги,
меняющее профиль риска.

Пер-тикерных записей (строк с вердиктом, PF, датами, реестров риска) в `docs/rsi_pullback/` не
делать ни в каком виде.

- [ ] **Step 4: Коммит**

```bash
git add internal/service/trading_strategy/rsi_pullback/strategy/sngsp docs/rsi_pullback
git commit -m "docs(rsi_pullback): разбор калибровки SNGSP и принятый риск"
```

---

### Task 16: Финальная проверка

**Files:** изменений нет, только проверки (кроме Step 3, где обновляется `_comment` точки).

**Interfaces:**
- Consumes: всё, сделанное Tasks 1–15.
- Produces: зелёный `mage ci` и нулевую сверку `pullparity` — условие готовности ветки к ревью.

- [ ] **Step 1: Сверка живой сборки с бэктестом**

Run:
```bash
go run ./cmd/pullparity -tickers SNGSP -months 36
```
Expected: ноль расхождений. Сверка проверяет, что живой раннер на тех же барах принимает те же
решения, что движок бэктеста, — то есть что литерал доехал до прода без искажений.

Если расхождения есть — **не чинить их правкой литерала**: это дефект сборки, и он разбирается
отдельно, с числами, на которых разошлись.

- [ ] **Step 2: Полный гейт качества**

Run:
```bash
./bin/mage ci
```
Expected: PASS (lint + `go test -race ./...` + проверка дрейфа моков).

- [ ] **Step 3: Дописать результат сверки в `_comment` точки**

В `plateau_point.json` дописать одной строкой: число баров, на которых шла сверка, и результат
(ноль расхождений).

- [ ] **Step 4: Финальный коммит**

```bash
git add data/params/rsi_pullback/sngsp/plateau_point.json
git commit -m "feat(rsi_pullback): SNGSP — сверка живой сборки и зелёный CI"
```

- [ ] **Step 5: Доложить владельцу итог**

Одним сообщением: вердикт по планке (по каждой ключевой теме раздельно, оба критерия), числа
принятой точки (pooled OOS PF, сделки, поведение под удвоенными издержками), принятые риски,
состояние ветки. Мерж и dry-run — решение владельца, план его не делает.
