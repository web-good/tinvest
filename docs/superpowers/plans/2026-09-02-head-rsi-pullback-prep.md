# HEAD под `rsi_pullback` — план подготовки, калибровки и вердикта

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Довести HEAD (МКПАО «Хэдхантер») до вердикта по стратегии `rsi_pullback`: каталог
максимально широких сеток, тематический walk-forward, принятая точка, литерал в пакете и заведение
в боевую вселенную двадцать первым тикером.

**Architecture:** Процедура тем **каноническая**: все десять тем идут поверх дефолтов ядра, поэтому
каталог сеток создаётся целиком одной задачей, якоря нет, поздних файлов нет. Схема прогонов
**адаптированная 24/12/3** — у нынешней бумаги 23.2 месяца истории, штатная 36/12/6 недоступна.
Два отступления от канона в осях, и оба посажены на замер: ось выхода расширена вниз до 50, а ось
тренда сдвинута вниз (`EMASlow` от 20 при `EMAFast` до 10). Сетки держатся **максимально широкими**
по решению владельца, поэтому сторожевой тест осей запрещает урезать ось, а не расширять её.

**Tech Stack:** Go 1.25, `cmd/backtest` (rolling walk-forward), `cmd/pullparity` (сверка живой
сборки с бэктестом), `./bin/mage ci` (lint + `go test -race ./...` + дрейф моков).

**Spec:** `docs/superpowers/specs/2026-09-02-head-rsi-pullback-prep-design.md`

## Global Constraints

- **Тикер HEAD, а не HHRU.** Заявка пришла на HHRU; его ряд обрывается 2024-08-09, бумага
  редомицилирована в МКПАО «Хэдхантер» и торгуется как HEAD с 2024-09-26. Решение владельца
  2026-09-02: калибруется HEAD. Ни один шаг плана не должен запускать прогоны по тикеру HHRU и не
  должен заводить HHRU ни в один реестр.
- **Схема прогонов адаптированная:** `-months 24 -train-months 12 -test-months 3 -min-trades 20
  -metric profit_factor`, четыре фолда встык (12 + 3·4 = 24). У темы `screen` — `-min-trades 1`.
  Пропуск любого из трёх флагов окна даёт другую схему и несравнимые числа. Расчётное окно —
  **2024-09-02 … 2026-09-02**; фактические данные внутри него начинаются 2024-09-26.
- **Число фолдов сверяется на первой же теме.** Контрольный прогон 2026-09-02 уже дал «Фолдов: 4»,
  но сверка повторяется: `cmd/backtest/main.go` берёт левую границу как
  `time.Now().AddDate(0, -months, 0)`, `internal/service/backtest/walkforward.go` строит фолды тем
  же `AddDate` и отбрасывает фолд, чей правый край **строго позже** `to`. Если в шапке отчёта темы
  `screen` окажется «Фолдов: 3» — **остановиться и доложить владельцу**, не запускать остальные
  темы. Правило сборки «≥3 фолда из 4» при трёх фолдах вырождается в «3 из 3» и меняет планку
  задним числом.
- **`-refresh` НЕ запускать ни на одном шаге.** Кэш дотянут штатным top-up провайдера 2026-09-02:
  `HEAD_Minutes30.json` — 32 285 баров (2023-08-04 … 2026-09-02), в окне 25 788, из них 17 507
  будних; `HEAD_Day1.json` — 1124 свечи (2022-08-04 … 2026-09-01), в окне 610, из них 491 будняя.
  Запас истории слева — больше года у обеих серий. Любой refresh сдвинет границы и сделает все
  числа спеки несравнимыми.
- **Дыра остановки торгов лежит СЛЕВА от окна.** Единственный разрыв длиннее четырёх дней в серии —
  2024-08-09 → 2024-09-26 (47 дней). Левая граница окна 2024-09-02 стоит внутри этого разрыва,
  поэтому первый бар окна — 2024-09-26, и ни одна EMA, ни один ATR и ни одна сделка не считаются
  через разрыв. Ни один шаг плана не должен расширять окно за 24 месяца.
- **Сетки держатся максимально широкими** (решение владельца 2026-09-02, то же, что по BANEP,
  ASTR и SNGSP). Обрезок осей не делается; замеры, которые в узком каталоге были бы основанием
  вырезать край, идут в `_comment` как **предупреждения** о том, чего ждать на краю. Жёстких
  инвариантов три: `RSILower` ≤ 50, `RSIPeriod` ≥ 3, ось тренда не порождает пар
  `EMAFast ≥ EMASlow`.
- **Цена широкой оси названа заранее:** она повышает риск переоптимизации на обучающем окне, и на
  24-месячном окне с трёхмесячными OOS-фолдами этот риск выше, чем на 36-месячных
  предшественниках. Правило сборки точки: **не менее трёх фолдов из четырёх** за одно значение оси,
  иначе ось остаётся на дефолте ядра. Ничья 2/2 большинством не считается.
- **Планка** (объявлена до прогонов, не пересматривается): темы `entry` и `trend` обе дают pooled
  OOS PF ≥ 1.5 при ≥ 20 сделках в пуле; ведущая ось (`RSILower` для `entry`, `EMASlow` для
  `trend`) выбрана одинаково в ≥ 3 фолдах из 4; вырожденный фолд (ни одной убыточной сделки) в
  пользу тикера не засчитывается; счёт по дефолтной комиссии (круг 0.1%).
- **Правило прода:** литерал ставится и HEAD заводится в `RSI_PULLBACK_TICKERS` двадцать первым
  **независимо от того, взята планка или нет**.
- **Стоп-условие:** работа останавливается, если принятая точка даёт pooled OOS PF < 1.0, **либо**
  меньше 20 сделок за расчётное окно, **либо** PF < 1.0 под удвоенными издержками
  (`-commission 0.001`). При срабатывании — числа владельцу, задачи 12–16 не выполняются.
- **Каждый `_comment` сетки** обязан содержать: что тема меряет и сколько в ней прогонов; замер, из
  которого получена каждая ось, с предупреждением о крае; полную команду запуска с путём
  `data/params/rsi_pullback/head/<файл>` (этого требует `TestRSIPullbackCalFilesValid`); место под
  строку `РЕЗУЛЬТАТ ПРОГОНА 2026-09-02: …`.
- **Дефолты ядра** (`core.DefaultParams()`), поверх которых считают ВСЕ темы: `RSIPeriod 4`,
  `RSILower 30`, `RSIUpper 70`, `EMAFast 10`, `EMASlow 100`, `DailyATRPeriod 14`,
  `UseDayATRGate 1`, `FreshDayATR 0`, `SpentDayATR 0.8`, `StopDailyATR 0.5`, `TPDailyATR 0.6`,
  `UseVolume 0`, `VolBaseDays 14`, `VolLookbackBars 3`, `VolMult 1.2`, `UseRSIExit 1`,
  `UseTrail 0`, `TrailDailyATR 0`. Контрольный прогон дефолтов на расчётном окне: **90 сделок,
  PF 1.311, net +10 965.97 ₽** — девятое место в ряду baseline каталога (YDEX 1.778, LSNGP 1.554,
  ELFV 1.546, SNGSP 1.535, IVAT 1.432, LENT 1.417, SVAV 1.368, BANEP 1.336, HEAD 1.311, BSPB
  1.114, ASTR 1.093, SIBN 1.027, NVTK 0.984). Выходы baseline: RSI 65, SL 18 (20.0%), TP 7.
- **Полугодия расчётного окна** (нужны в Task 11): 2024-09-26 … 2025-02-28 (инструмент −2.3%,
  baseline 21 сделка, PF 1.713, 62.9% net), 2025-03-03 … 2025-09-01 (−4.3%, 26, 2.003, 54.3%),
  2025-09-02 … 2026-02-27 (−16.0%, 19, **0.549**, −57.9%), 2026-03-02 … 2026-09-01 (−11.7%, 24,
  1.795, 40.7%).
- **Три известных гэпа** (нужны в Tasks 9 и 11): **2024-12-17 −9.99%**, **2026-05-12 −8.16%**,
  **2025-09-26 −6.14%**. Это 3.5, 2.9 и 2.1 дневных ATR — стопом не держатся. Экспозиция baseline:
  одна сделка (вход 2025-09-25) прошла через сентябрьский гэп и закрылась по стопу с −6.28%,
  худшей строкой журнала. Доля сделок baseline с ночёвкой — **40.0%** (36 из 90). Риск на этой
  бумаге **реализованный, а не спящий**.
- **Пер-тикерных записей в `docs/rsi_pullback/` не делается.** Правило CLAUDE.md: `strategy.md`,
  `live.md`, `screener.md` описывают механику, а не калибровочные прогоны. Пер-тикерный разбор
  живёт в доке пакета `strategy/head` и в `_comment` файлов сеток. В `docs/rsi_pullback/` уходит
  только механический вывод, если HEAD такой даст (Task 15 Step 3).
- **Каталог `reports/` — в `.gitignore`.** Отчёты прогонов остаются локальными артефактами; в
  репозиторий едут только строки `РЕЗУЛЬТАТ ПРОГОНА` в `_comment` сеток, рабочая записка Task 11 и
  код. Ни один шаг не должен пытаться закоммитить `reports/`.
- **Коммит по завершении каждой задачи**, сообщения на русском в стиле существующей истории.
- **Ветка:** `feat/head-pullback-prep` от `feat/sngsp-pullback-prep` (`1f8bb77`); спека уже
  закоммичена (`425d8a3`).

---

### Task 1: Каталог десяти сеток со сторожевым тестом осей

**Files:**
- Create: `data/params/rsi_pullback/head/cal_screen.json`
- Create: `data/params/rsi_pullback/head/cal_entry.json`
- Create: `data/params/rsi_pullback/head/cal_trend.json`
- Create: `data/params/rsi_pullback/head/cal_day.json`
- Create: `data/params/rsi_pullback/head/cal_day_spent.json`
- Create: `data/params/rsi_pullback/head/cal_volume.json`
- Create: `data/params/rsi_pullback/head/cal_vol_window.json`
- Create: `data/params/rsi_pullback/head/cal_risk.json`
- Create: `data/params/rsi_pullback/head/cal_exit.json`
- Create: `data/params/rsi_pullback/head/cal_trail.json`
- Create: `internal/service/backtest/rsi_pullback_head_grid_test.go`

**Interfaces:**
- Consumes: хелперы пакета `backtest` — `rsiPullbackTickerGrid(t, ticker, file)`
  (`rsi_pullback_grid_test.go:40`), `sameSet` (`rsi_pullback_reni_grid_test.go:17`),
  `containsValue` (`rsi_pullback_ivat_grid_test.go:88`); объявлять их заново НЕЛЬЗЯ — пакет один,
  будет переопределение. Общие тесты `TestRSIPullbackCalFilesValid`
  (`rsi_pullback_grid_test.go:89`) и `TestRSIPullbackGridControlPoints` (там же, :147)
  подхватывают новый каталог сами.
- Produces: каталог `data/params/rsi_pullback/head/` и функцию `headGrid(t, file)` — ею пользуются
  только тесты этого файла.

- [ ] **Step 1: Написать падающий сторожевой тест осей**

Создать `internal/service/backtest/rsi_pullback_head_grid_test.go`:

```go
package backtest

import "testing"

// headGrid читает файл сеток HEAD через общий хелпер.
func headGrid(t *testing.T, file string) map[string][]float64 {
	t.Helper()
	return rsiPullbackTickerGrid(t, "head", file)
}

// TestHEADGridsStayWide сторожит оси всех десяти тем каталога head/. Каталог собран 2026-09-02 по
// замерам самой бумаги на расчётном окне 2024-09-02 … 2026-09-02 (в окне 25 788 получасовых
// баров, из них 17 507 будних; дневная серия в окне 610 свечей, из них 491 будняя).
//
// ЗАЯВКА БЫЛА НА HHRU. Тикер мёртв: ряд обрывается 2024-08-09, бумага редомицилирована в МКПАО
// «Хэдхантер» и торгуется как HEAD с 2024-09-26. Калибруется HEAD — решение владельца 2026-09-02.
//
// РЕШЕНИЕ ВЛАДЕЛЬЦА 2026-09-02: сетки этого тикера держатся МАКСИМАЛЬНО ШИРОКИМИ, как у BANEP,
// ASTR и SNGSP. Поэтому тест сторожит оси с ДРУГОЙ стороны, чем у большинства тикеров каталога: он
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
// СХЕМА ПРОГОНОВ АДАПТИРОВАННАЯ 24/12/3: у нынешней бумаги 23.2 месяца истории, штатная 36/12/6
// недоступна. Четыре фолда встык, число проверено контрольным прогоном до раскладки сеток.
func TestHEADGridsStayWide(t *testing.T) {
	screen := headGrid(t, "cal_screen.json")
	for _, field := range []string{"UseDayATRGate", "UseVolume"} {
		if got := screen[field]; !sameSet(got, 0, 1) {
			t.Errorf("cal_screen.json: %s = %v, want ровно {0,1} — тема меряет цену каждого гейта в сделках", field, got)
		}
	}

	entry := headGrid(t, "cal_entry.json")
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
	// Рельеф уровня на HEAD немонотонный, максимум живой зоны в середине оси (RSI(4): 10 -> 0.447
	// на 11 сделках, 30 -> 1.311, 35 -> 1.578, 50 -> 1.044). Оба края обязаны остаться в сетке:
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

	// Ось тренда СДВИНУТА ВНИЗ: EMASlow начинается с 20, EMAFast кончается на 10. Максимум всей
	// таблицы EMA стоит на паре 3/20 (PF 2.669 на 57 сделках), рядом 3/30 (1.626/76) и 5/30
	// (1.485/84), тогда как вся половина EMAFast 20..40 лежит в 1.10-1.30, то есть на уровне
	// baseline 1.311 и ниже. Расширить вниз и сохранить быстрый край одновременно нельзя —
	// инвариант EMAFast < EMASlow этого не разрешает. Прецедент ровно такой: ASTR, BANEP и BSPB
	// держат EMAFast [5,10,20] при EMASlow от 30.
	trend := headGrid(t, "cal_trend.json")
	for _, f := range trend["EMAFast"] {
		for _, s := range trend["EMASlow"] {
			if f >= s {
				t.Errorf("cal_trend.json порождает пару EMAFast=%v >= EMASlow=%v: фильтр вырождается", f, s)
			}
		}
	}
	if !containsValue(trend["EMAFast"], 3) || !containsValue(trend["EMASlow"], 20) {
		t.Errorf("cal_trend.json: пара 3/20 обязана остаться в сетке (максимум таблицы, PF 2.669 на 57 сделках), got fast=%v slow=%v", trend["EMAFast"], trend["EMASlow"])
	}
	if !containsValue(trend["EMASlow"], 250) {
		t.Errorf("cal_trend.json: EMASlow = %v, не содержит 250 — верхний край оси не урезается", trend["EMASlow"])
	}

	// Ось выхода РАСШИРЕНА ВНИЗ ДО 50 (канон §8.1 начинается с 55): максимум оси на 80 (1.470), но
	// второе значение стоит на 55 (1.430), у самого края канонической оси, а ось с сильным краем
	// каталог читать не умеет (ошибка, разобранная на WUSH). Уровень 50 живой и по рельефу
	// (1.380/103), и по событиям (2163 кросса вверх у RSI(4)).
	exit := headGrid(t, "cal_exit.json")
	for _, v := range []float64{50, 85} {
		if !containsValue(exit["RSIUpper"], v) {
			t.Errorf("cal_exit.json: RSIUpper = %v, не содержит %v — оба края полосы выхода обязаны остаться", exit["RSIUpper"], v)
		}
	}

	// Ось риска РАСШИРЕНА ВВЕРХ ДО 2.0 (канон кончается на 1.5). Рельеф насыщается уже на 1.3:
	// колонки 1.3, 1.5 и 2.0 совпадают побайтово (PF 2.392 на 88 сделках). Верх оставлен как
	// КОНТРОЛЬ насыщения: 2.0 дневных ATR это ~5.7% цены, такой стоп достаётся размахом дня в
	// 1.9% дней и защитой быть перестаёт. Победитель проверяется долей SL-выходов, удержанием и
	// экспозицией под гэпом (Task 9).
	risk := headGrid(t, "cal_risk.json")
	if !containsValue(risk["StopDailyATR"], 2.0) {
		t.Errorf("cal_risk.json: StopDailyATR = %v, не содержит 2.0 — контроль насыщения оси", risk["StopDailyATR"])
	}
	if !containsValue(risk["StopDailyATR"], 0.3) {
		t.Errorf("cal_risk.json: StopDailyATR = %v, не содержит 0.3 — узкий край оси не урезается", risk["StopDailyATR"])
	}
	// Контрольная строка цели: TestRSIPullbackGridControlPoints требует цель ВЫШЕ самого широкого
	// стопа файла. Строка 2.5 заведомо мертва (колонки 1.5, 2.0 и 2.5 совпадают побайтово) и стоит
	// в сетке как контроль асимметрии, а не как кандидат.
	if !containsValue(risk["TPDailyATR"], 2.5) {
		t.Errorf("cal_risk.json: TPDailyATR = %v, не содержит контрольную 2.5", risk["TPDailyATR"])
	}

	day := headGrid(t, "cal_day.json")
	for _, v := range []float64{0, 0.6} {
		if !containsValue(day["FreshDayATR"], v) {
			t.Errorf("cal_day.json: FreshDayATR = %v, не содержит %v", day["FreshDayATR"], v)
		}
	}
	for _, v := range []float64{0.5, 1.5} {
		if !containsValue(day["SpentDayATR"], v) {
			t.Errorf("cal_day.json: SpentDayATR = %v, не содержит %v", day["SpentDayATR"], v)
		}
	}

	spent := headGrid(t, "cal_day_spent.json")
	if !sameSet(spent["FreshDayATR"], 0) {
		t.Errorf("cal_day_spent.json: FreshDayATR = %v, want ровно {0} — тема мерит исчерпанную ветку при закрытой свежей", spent["FreshDayATR"])
	}
	for _, v := range []float64{0.4, 1.5} {
		if !containsValue(spent["SpentDayATR"], v) {
			t.Errorf("cal_day_spent.json: SpentDayATR = %v, не содержит %v", spent["SpentDayATR"], v)
		}
	}

	volume := headGrid(t, "cal_volume.json")
	if !sameSet(volume["UseVolume"], 1) {
		t.Errorf("cal_volume.json: UseVolume = %v, want ровно {1} — тема мерит форму гейта, а не его наличие", volume["UseVolume"])
	}
	for _, v := range []float64{1.0, 3.0} {
		if !containsValue(volume["VolMult"], v) {
			t.Errorf("cal_volume.json: VolMult = %v, не содержит %v", volume["VolMult"], v)
		}
	}

	window := headGrid(t, "cal_vol_window.json")
	for _, v := range []float64{1, 24} {
		if !containsValue(window["VolLookbackBars"], v) {
			t.Errorf("cal_vol_window.json: VolLookbackBars = %v, не содержит %v — ось окна держится широкой, сходимости на ней замер не показал", window["VolLookbackBars"], v)
		}
	}

	trail := headGrid(t, "cal_trail.json")
	if !sameSet(trail["UseRSIExit"], 0, 1) {
		t.Errorf("cal_trail.json: UseRSIExit = %v, want ровно {0,1} — тема мерит, заменяет ли трейл RSI-выход", trail["UseRSIExit"])
	}
	if !sameSet(trail["UseTrail"], 1) {
		t.Errorf("cal_trail.json: UseTrail = %v, want ровно {1} — иначе дистанция трейла не измеряется вовсе", trail["UseTrail"])
	}
}
```

- [ ] **Step 2: Запустить тест и убедиться, что он падает**

Run: `go test ./internal/service/backtest/ -run TestHEADGridsStayWide -v`
Expected: FAIL — каталога `data/params/rsi_pullback/head/` не существует.

- [ ] **Step 3: Создать десять файлов сеток**

Каждый файл — формат `{"_comment": "...", "phases": [{"name": "...", "grid": {...}, "keepTop": 5}]}`.
`_comment` обязан называть собственный путь (этого требует `TestRSIPullbackCalFilesValid`),
описывать замер за каждой осью, предупреждение о крае и полную команду запуска, и оставлять место
под строку результата.

`data/params/rsi_pullback/head/cal_screen.json`:

```json
{
  "_comment": "data/params/rsi_pullback/head/cal_screen.json — тема screen для HEAD, 4 прогона, поверх дефолтов ядра. Тема меряет цену каждого из двух гейтов в сделках. ЗАМЕР 2026-09-02 (полное окно, in-sample): UseDayATRGate 0 -> PF 0.930 на 327 сделках при net -8825.5, UseDayATRGate 1 -> 1.311 на 90 при +10 966.0 — БЕЗ ДНЕВНОГО ГЕЙТА СТРАТЕГИЯ НА HEAD УБЫТОЧНА, гейт несёт весь edge. Объёмный гейт неразличим с выключенным: весь массив множителя и базы лежит в полосе 0.955-1.419 вокруг baseline 1.311. ОЖИДАНИЕ: UseDayATRGate 1 берут все или почти все фолды, UseVolume большинства не набирает. ЗАПУСК: go run ./cmd/backtest -ticker HEAD -strategy rsi_pullback -interval Minutes30 -calibrate data/params/rsi_pullback/head/cal_screen.json -out ./reports/HEAD_screen -months 24 -train-months 12 -test-months 3 -min-trades 1 -metric profit_factor. ПРОВЕРИТЬ ЧИСЛО ФОЛДОВ В ШАПКЕ ОТЧЁТА: обязано быть 4. РЕЗУЛЬТАТ ПРОГОНА 2026-09-02: ",
  "phases": [
    {
      "name": "screen",
      "grid": {
        "UseDayATRGate": [0, 1],
        "UseVolume": [0, 1]
      },
      "keepTop": 5
    }
  ]
}
```

`data/params/rsi_pullback/head/cal_entry.json`:

```json
{
  "_comment": "data/params/rsi_pullback/head/cal_entry.json — тема entry для HEAD, 432 прогона (RSIUpper 8 x RSIPeriod 6 x RSILower 9), поверх дефолтов ядра. КЛЮЧЕВАЯ ТЕМА: ведущая ось RSILower. ЗАМЕР 2026-09-02, рельеф при RSI(4): 10 -> 0.447/11, 15 -> 0.961/32, 20 -> 1.028/49, 25 -> 1.074/65, 30 -> 1.311/90, 35 -> 1.578/118, 40 -> 1.058/131, 45 -> 1.097/149, 50 -> 1.044/166 — максимум живой зоны на 35, рельеф НЕМОНОТОННЫЙ. Ось периода: RSI(3)@25 -> 1.321/112, RSI(5)@35 -> 1.443/86, RSI(6)@40 -> 1.354/93, RSI(7)@45 -> 1.403/99, RSI(8)@45 -> 1.330/86. ПРЕДУПРЕЖДЕНИЕ О КРАЕ: на длинных периодах глубокие уровни дают заоблачный PF на выборке в единицы сделок (RSI(6)@10 -> 1.578 на ТРЁХ сделках, RSI(7)@15 -> 2.252 на ЧЕТЫРЁХ, RSI(8)@10 -> ноль сделок) — это шум, и порог -min-trades 20 его отсечёт; края оси стоят в сетке ради доказуемости середины, а не как кандидаты. RSIUpper свипуется здесь В СВЯЗКЕ и идёт как контекст: при сборке точки поле берётся из темы exit, которая мерит его изолированно. Кроссов вниз хватает на всей оси периода: RSI(8)@25 — 356 событий за окно. ЗАПУСК: go run ./cmd/backtest -ticker HEAD -strategy rsi_pullback -interval Minutes30 -calibrate data/params/rsi_pullback/head/cal_entry.json -out ./reports/HEAD_entry -months 24 -train-months 12 -test-months 3 -min-trades 20 -metric profit_factor. РЕЗУЛЬТАТ ПРОГОНА 2026-09-02: ",
  "phases": [
    {
      "name": "entry",
      "grid": {
        "RSIUpper": [50, 55, 60, 65, 70, 75, 80, 85],
        "RSIPeriod": [3, 4, 5, 6, 7, 8],
        "RSILower": [10, 15, 20, 25, 30, 35, 40, 45, 50]
      },
      "keepTop": 5
    }
  ]
}
```

`data/params/rsi_pullback/head/cal_trend.json`:

```json
{
  "_comment": "data/params/rsi_pullback/head/cal_trend.json — тема trend для HEAD, 24 прогона, поверх дефолтов ядра. ВТОРАЯ КЛЮЧЕВАЯ ТЕМА: ведущая ось EMASlow. ОСЬ СДВИНУТА ВНИЗ ОТНОСИТЕЛЬНО КАНОНА §8.1: EMASlow начинается с 20, EMAFast кончается на 10. ЗАМЕР 2026-09-02: максимум ВСЕЙ таблицы EMA стоит на паре 3/20 — PF 2.669 на 57 сделках, рядом 3/30 -> 1.626/76 и 5/30 -> 1.485/84 и 10/20 -> 1.413/95; канонический участок при этом весь около baseline или ниже (5/100 -> 1.290, 10/100 -> 1.311 = baseline, 20/100 -> 1.301, 30/150 -> 1.139, 40/250 -> 1.100), а ВСЯ половина EMAFast 20..40 лежит в 1.10-1.30. Расширить ось вниз и сохранить быстрый край одновременно НЕЛЬЗЯ: жёсткий инвариант EMAFast < EMASlow этого не разрешает. Прецедент ровно такой — ASTR, BANEP и BSPB держат EMAFast [5,10,20] при EMASlow от 30. ЦЕНА РЕШЕНИЯ, названная заранее: пары EMAFast 20..40 в теме не измеряются вовсе; если фолды упрутся в верхний край быстрой оси (10), это означает, что ось надо было ставить иначе, и такой исход записывается прямым текстом. ПРЕДУПРЕЖДЕНИЕ О КРАЕ: допуск EMAFast > EMASlow держится в полосе 44.2-48.4% по ВСЕЙ таблице 3..40 x 20..250 (n = 17 507 будних баров) — ровнее, чем у любого тикера каталога. Победитель на паре 3/20 (разнос всего 6.7 раза) читается как 'трендовый фильтр бумаге почти не нужен', а не как находка. ЗАПУСК: go run ./cmd/backtest -ticker HEAD -strategy rsi_pullback -interval Minutes30 -calibrate data/params/rsi_pullback/head/cal_trend.json -out ./reports/HEAD_trend -months 24 -train-months 12 -test-months 3 -min-trades 20 -metric profit_factor. РЕЗУЛЬТАТ ПРОГОНА 2026-09-02: ",
  "phases": [
    {
      "name": "trend",
      "grid": {
        "EMAFast": [3, 5, 10],
        "EMASlow": [20, 30, 50, 70, 100, 150, 200, 250]
      },
      "keepTop": 5
    }
  ]
}
```

`data/params/rsi_pullback/head/cal_day.json`:

```json
{
  "_comment": "data/params/rsi_pullback/head/cal_day.json — тема day для HEAD, 48 прогонов, поверх дефолтов ядра. Тема меряет обе ветки двустороннего дневного гейта разом. ЗАМЕР 2026-09-02, свежая ветка при SpentDayATR 0.8: 0 -> 1.311/90, 0.2 -> 1.176/110, 0.3 -> 1.025/145, 0.4 -> 0.928/182, 0.5 -> 0.960/235, 0.6 -> 0.886/275 — МОНОТОННО ВНИЗ, открытая свежая ветка на HEAD просто добавляет мусорных сделок. Исчерпанная ветка при FreshDayATR 0: 0.5 -> 1.001/208, 0.6 -> 1.173/160, 0.7 -> 1.325/123, 0.8 -> 1.311/90, 0.9 -> 1.427/68, 1.0 -> 1.583/52, 1.1 -> 1.292/41, 1.25 -> 1.450/33, 1.5 -> 3.499/17 — максимум на 1.0. Доли будних баров, проходящих ветки (n = 17 068): свежий день 0.2 -> 6.5%, 0.6 -> 42.9%; день исчерпан 0.5 -> 67.7%, 0.8 -> 37.0%, 1.0 -> 22.7%, 1.5 -> 7.0%. ПРЕДУПРЕЖДЕНИЕ О КРАЕ: хвост исчерпанной ветки 1.25-1.5 стоит на 17-33 сделках за ВСЁ окно, то есть 5-11 на двенадцатимесячное обучающее окно — ниже порога -min-trades 20, и PF 3.499 на семнадцати сделках это шум, а не находка. ОЖИДАНИЕ: FreshDayATR 0 побеждает большинством, SpentDayATR тянет в зону 0.9-1.0. ЗАПУСК: go run ./cmd/backtest -ticker HEAD -strategy rsi_pullback -interval Minutes30 -calibrate data/params/rsi_pullback/head/cal_day.json -out ./reports/HEAD_day -months 24 -train-months 12 -test-months 3 -min-trades 20 -metric profit_factor. РЕЗУЛЬТАТ ПРОГОНА 2026-09-02: ",
  "phases": [
    {
      "name": "day",
      "grid": {
        "FreshDayATR": [0, 0.2, 0.3, 0.4, 0.5, 0.6],
        "SpentDayATR": [0.5, 0.6, 0.7, 0.8, 0.9, 1.0, 1.25, 1.5]
      },
      "keepTop": 5
    }
  ]
}
```

`data/params/rsi_pullback/head/cal_day_spent.json`:

```json
{
  "_comment": "data/params/rsi_pullback/head/cal_day_spent.json — тема day_spent для HEAD, 10 прогонов, поверх дефолтов ядра. Тема мерит ТОЛЬКО исчерпанную ветку при закрытой свежей (FreshDayATR зафиксирован на нуле), чтобы отделить вклад одной ветки от их взаимодействия в теме day. ЗАМЕР 2026-09-02: 0.4 -> 0.989/260, 0.5 -> 1.001/208, 0.6 -> 1.173/160, 0.7 -> 1.325/123, 0.8 -> 1.311/90 (дефолт), 0.9 -> 1.427/68, 1.0 -> 1.583/52, 1.1 -> 1.292/41, 1.25 -> 1.450/33, 1.5 -> 3.499/17. Доля будних баров, проходящих ветку: 0.4 -> 78.1%, 0.7 -> 45.3%, 1.0 -> 22.7%, 1.5 -> 7.0% (n = 17 068). ПРЕДУПРЕЖДЕНИЕ О КРАЕ: верхние два значения стоят на выборке 17-33 сделки за всё окно и в теме будут шуметь; нижние два дают PF около единицы, то есть гейт, впускающий почти всё, edge не создаёт. СВЕРКА С ТЕМОЙ day обязательна: если обе темы дают по SpentDayATR разные фолдовые значения, это означает взаимодействие веток, и оно записывается как свойство, а не сглаживается. ЗАПУСК: go run ./cmd/backtest -ticker HEAD -strategy rsi_pullback -interval Minutes30 -calibrate data/params/rsi_pullback/head/cal_day_spent.json -out ./reports/HEAD_day_spent -months 24 -train-months 12 -test-months 3 -min-trades 20 -metric profit_factor. РЕЗУЛЬТАТ ПРОГОНА 2026-09-02: ",
  "phases": [
    {
      "name": "day_spent",
      "grid": {
        "FreshDayATR": [0],
        "SpentDayATR": [0.4, 0.5, 0.6, 0.7, 0.8, 0.9, 1.0, 1.1, 1.25, 1.5]
      },
      "keepTop": 5
    }
  ]
}
```

`data/params/rsi_pullback/head/cal_volume.json`:

```json
{
  "_comment": "data/params/rsi_pullback/head/cal_volume.json — тема volume для HEAD, 30 прогонов, поверх дефолтов ядра. Тема мерит форму объёмного гейта (множитель и база) при ВКЛЮЧЁННОМ гейте; сам вопрос 'нужен ли гейт' мерит тема screen. ЗАМЕР 2026-09-02: весь массив 30 комбинаций лежит в полосе 0.955-1.419 вокруг baseline 1.311, то есть гейт НЕРАЗЛИЧИМ с выключенным. Опорные строки: VolMult 1.0/база 10 -> 1.266/74, 1.2/14 -> 1.293/66, 1.5/20 -> 1.207/56, 2.0/5 -> 1.261/58, 2.5/3 -> 1.217/56, 3.0/5 -> 1.419/41, 3.0/14 -> 1.352/35. ПРЕДУПРЕЖДЕНИЕ О КРАЕ: единственный выброс (3.0/5 -> 1.419) стоит на 41 сделке за всё окно, то есть около 20 на обучающее окно — ровно на пороге -min-trades 20, и большинства фолдов он набрать не должен. ОЖИДАНИЕ: ни одна из объёмных осей большинства не набирает, оба поля остаются на дефолтах ядра. ЗАПУСК: go run ./cmd/backtest -ticker HEAD -strategy rsi_pullback -interval Minutes30 -calibrate data/params/rsi_pullback/head/cal_volume.json -out ./reports/HEAD_volume -months 24 -train-months 12 -test-months 3 -min-trades 20 -metric profit_factor. РЕЗУЛЬТАТ ПРОГОНА 2026-09-02: ",
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

`data/params/rsi_pullback/head/cal_vol_window.json`:

```json
{
  "_comment": "data/params/rsi_pullback/head/cal_vol_window.json — тема vol_window для HEAD, 24 прогона, поверх дефолтов ядра. Тема мерит окно объёмного гейта (VolLookbackBars) на трёх уровнях множителя. ЗАМЕР 2026-09-02 при VolMult 1.2: окно 1 -> 1.269/54, 2 -> 1.335/63, 3 -> 1.293/66 (дефолт), 5 -> 1.311/69, 8 -> 1.379/74, 12 -> 1.335/83, 16 -> 1.355/84, 24 -> 1.367/85; при 2.0: 1 -> 1.126/35, 12 -> 1.321/70, 24 -> 1.319/81; при 3.0: 1 -> 1.194/24, 3 -> 1.352/35, 12 -> 1.439/52, 24 -> 1.352/67. СХОДИМОСТИ НЕТ НИ НА ОДНОЙ ОСИ: рельеф плоский и слегка растущий к широкому краю, размах внутри строки не превышает 0.25 PF. ПРЕДУПРЕЖДЕНИЕ О КРАЕ: узкий край (окно 1) при множителе 3.0 оставляет 24 сделки за всё окно — ниже порога; широкий край (24) не даёт максимума ни в одной строке и лишь возвращает выборку к baseline. Это девятая проверка гипотезы каталога 'чем ликвиднее бумага, тем ближе оптимум окна к дефолту'. ЗАПУСК: go run ./cmd/backtest -ticker HEAD -strategy rsi_pullback -interval Minutes30 -calibrate data/params/rsi_pullback/head/cal_vol_window.json -out ./reports/HEAD_vol_window -months 24 -train-months 12 -test-months 3 -min-trades 20 -metric profit_factor. РЕЗУЛЬТАТ ПРОГОНА 2026-09-02: ",
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

`data/params/rsi_pullback/head/cal_risk.json`:

```json
{
  "_comment": "data/params/rsi_pullback/head/cal_risk.json — тема risk для HEAD, 56 прогонов, поверх дефолтов ядра. Тема мерит стоп и цель. ОСЬ СТОПА РАСШИРЕНА ВВЕРХ ДО 2.0 (канон кончается на 1.5). ЗАМЕР 2026-09-02 при цели 0.6: 0.3 -> 1.488/91, 0.5 -> 1.311/90 (дефолт), 0.7 -> 1.503/89, 1.0 -> 1.742/89, 1.3 -> 2.392/88, 1.5 -> 2.392/88, 2.0 -> 2.392/88 — три верхние колонки совпадают ПОБАЙТОВО, ось НАСЫЩАЕТСЯ на 1.3. Верх оставлен как контроль насыщения. КАПКАН ШИРОКОГО СТОПА, ДЕВЯТЫЙ ТИКЕР ПОДРЯД, и здесь он опаснее обычного: при стопе 1.3 доля SL-выходов падает 20.0% -> 1.1%, ночёвка растёт 40.0% -> 48.9%, максимум удержания 31 -> 52 бара; со связкой RSILower 35 + RSIUpper 80 и тем же стопом — PF 1.975, SL 4.6%, НОЧЁВКА 74.3%, удержание до 138 баров. Убыток не исчезает, он утекает в RSI-выход, а позиция начинает ночевать на бумаге, где гэп -9.99% уже стоил стоп-лосса (сделка baseline со входом 2025-09-25, -6.28%). Выживаемость стопов (доля будних дней, чей размах достаёт уровня, n = 476): 0.3 -> 99.4%, 0.5 -> 91.6%, 0.7 -> 69.3%, 1.0 -> 39.3%, 1.3 -> 20.8%, 1.5 -> 11.8%, 2.0 -> 1.9%. ОСЬ ЦЕЛИ: 0.3 -> 1.205/92, 0.4 -> 1.309/92, 0.5 -> 1.337/91, 0.6 -> 1.311/90, 0.8 -> 1.296/90, 1.0 -> 1.312/90, 1.5 -> 1.270/90, 2.0 -> 1.270/90, 2.5 -> 1.270/90 — размах всей оси 0.13 PF, поле слабое, а колонки 1.5, 2.0 и 2.5 совпадают побайтово (цель шире полутора дневных ATR не достигается). Строка 2.5 стоит в сетке как КОНТРОЛЬ асимметрии риск/награда, которого требует TestRSIPullbackGridControlPoints от файла, свипующего стоп до 2.0, — а не как кандидат. ПОБЕДИТЕЛЬ ТЕМЫ ОБЯЗАН проверяться долей SL-выходов, удержанием и экспозицией под тремя гэпами. ЗАПУСК: go run ./cmd/backtest -ticker HEAD -strategy rsi_pullback -interval Minutes30 -calibrate data/params/rsi_pullback/head/cal_risk.json -out ./reports/HEAD_risk -months 24 -train-months 12 -test-months 3 -min-trades 20 -metric profit_factor. РЕЗУЛЬТАТ ПРОГОНА 2026-09-02: ",
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

`data/params/rsi_pullback/head/cal_exit.json`:

```json
{
  "_comment": "data/params/rsi_pullback/head/cal_exit.json — тема exit для HEAD, 8 прогонов, поверх дефолтов ядра. Тема мерит уровень выхода RSIUpper ИЗОЛИРОВАННО (в теме entry он идёт в связке и потому только контекст). ОСЬ РАСШИРЕНА ВНИЗ ДО 50 (канон §8.1 начинается с 55). ЗАМЕР 2026-09-02: 50 -> 1.380/103, 55 -> 1.430/97, 60 -> 1.185/94, 65 -> 1.357/93, 70 -> 1.311/90 (дефолт), 75 -> 1.349/89, 80 -> 1.470/89, 85 -> 1.261/88 — максимум на 80, но ВТОРОЕ значение стоит на 55, у самого края канонической оси, а ось с сильным краем каталог читать не умеет (ошибка, разобранная на WUSH). Уровень 50 живой и по рельефу, и по событиям: кроссов вверх у RSI(4) через 50 — 2163 за окно, через 85 — 401. ПРЕДУПРЕЖДЕНИЕ О НИЖНЕМ КРАЕ: чем ниже уровень выхода, тем короче удержание; на HEAD удержание baseline и так внутридневное по медиане (9 баров, 0 календарных дней), но ночует 40% сделок — победа края 50 означает более быстрый выход и МЕНЬШУЮ экспозицию под гэпом, то есть край здесь работает на снижение риска, а не наоборот. ПРЕДУПРЕЖДЕНИЕ О ВЕРХНЕМ КРАЕ: 85 даёт худший PF оси и удлиняет удержание. ЗАПУСК: go run ./cmd/backtest -ticker HEAD -strategy rsi_pullback -interval Minutes30 -calibrate data/params/rsi_pullback/head/cal_exit.json -out ./reports/HEAD_exit -months 24 -train-months 12 -test-months 3 -min-trades 20 -metric profit_factor. РЕЗУЛЬТАТ ПРОГОНА 2026-09-02: ",
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

`data/params/rsi_pullback/head/cal_trail.json`:

```json
{
  "_comment": "data/params/rsi_pullback/head/cal_trail.json — тема trail для HEAD, 10 прогонов, поверх дефолтов ядра. Тема мерит, заменяет ли трейл RSI-выход и на какой дистанции он вообще работает; UseTrail форсирован в 1, иначе дистанция не измеряется. ЗАМЕР 2026-09-02 при ВКЛЮЧЁННОМ RSI-выходе: 0.3 -> 1.327/97, 0.5 -> 1.131/91, 0.7 -> 1.340/90, 1.0 -> 1.311/90, 1.5 -> 1.311/90 — с 1.0 трейл не срабатывает ни разу и результат ПОБАЙТОВО равен его отсутствию; максимальный прирост над baseline 0.029 PF, то есть на HEAD трейл почти ничего не даёт (отличие от SNGSP, где 0.5 давал +0.144). При ВЫКЛЮЧЕННОМ RSI-выходе: 0.3 -> 1.020/96 при win rate 36.5%, 0.5 -> 0.944/90, 0.7 -> 1.269/87, 1.0 -> 1.137/86, 1.5 -> 1.137/86; отдельно по оси цели без RSI-выхода: 0.5 -> 1.147/88, 1.0 -> 1.392/81, 2.5 -> 0.955/45 при win rate 15.6% — ВЫКЛЮЧАТЬ RSI-ВЫХОД НЕЛЬЗЯ, конфигурация держится только на редких больших целях. ПРАВИЛО УПРОЩЕНИЯ при сборке точки: если тема выбрала дистанцию, на которой трейл не срабатывает (на HEAD это 1.0 и выше), в точку идёт UseTrail 0 и TrailDailyATR 0 — конфигурация проще при идентичном результате (прецедент ASTR). ЗАПУСК: go run ./cmd/backtest -ticker HEAD -strategy rsi_pullback -interval Minutes30 -calibrate data/params/rsi_pullback/head/cal_trail.json -out ./reports/HEAD_trail -months 24 -train-months 12 -test-months 3 -min-trades 20 -metric profit_factor. РЕЗУЛЬТАТ ПРОГОНА 2026-09-02: ",
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

- [ ] **Step 4: Запустить сторожевой тест и общие тесты каталога**

Run: `go test ./internal/service/backtest/ -run 'TestHEADGridsStayWide|TestRSIPullbackCalFilesValid|TestRSIPullbackGridControlPoints' -v`
Expected: PASS все три. Если `TestRSIPullbackCalFilesValid` ругается на `_comment` — в тексте
комментария нет собственного пути файла; если `TestRSIPullbackGridControlPoints` — в `cal_risk.json`
нет цели выше самого широкого стопа.

- [ ] **Step 5: Коммит**

```bash
git add data/params/rsi_pullback/head internal/service/backtest/rsi_pullback_head_grid_test.go
git commit -m "feat(rsi_pullback): каталог максимально широких сеток HEAD"
```

---

### Task 2: Пакет `strategy/head` в состоянии «калибровка не проводилась»

**Files:**
- Create: `internal/service/trading_strategy/rsi_pullback/strategy/head/head.go`
- Create: `internal/service/trading_strategy/rsi_pullback/strategy/head/head_test.go`
- Modify: `internal/service/backtest/rsi_pullback_registry.go`
- Modify: `internal/service/backtest/rsi_pullback_registry_test.go`

**Interfaces:**
- Consumes: `core.Params`, `core.DefaultParams()` из
  `internal/service/trading_strategy/rsi_pullback/strategy/core`.
- Produces: `head.Ticker` (строка `"HEAD"`) и `head.DefaultParams() core.Params` — их используют
  Task 12 (литерал) и Task 13 (реестр живого раннера).

- [ ] **Step 1: Написать падающий тест baseline-состояния**

Создать `internal/service/trading_strategy/rsi_pullback/strategy/head/head_test.go`:

```go
package head

import (
	"testing"

	"tinvest/internal/service/trading_strategy/rsi_pullback/strategy/core"
)

// TestParamsTrackTheBaselineUntilCalibrated фиксирует ЧЕСТНОЕ состояние: калибровка HEAD ещё не
// проводилась, поэтому пакет обязан возвращать ровно baseline ядра. Тест держит это состояние до
// Task 12, где его заменяет снимок литерала. Пока он стоит, ни одна правка не может тихо
// подсунуть в прод «почти откалиброванные» параметры.
func TestParamsTrackTheBaselineUntilCalibrated(t *testing.T) {
	if got, want := DefaultParams(), core.DefaultParams(); got != want {
		t.Fatalf("HEAD ещё не откалиброван, параметры обязаны совпадать с baseline:\n got: %+v\nwant: %+v", got, want)
	}
}

func TestTickerIsHEAD(t *testing.T) {
	if Ticker != "HEAD" {
		t.Fatalf("Ticker = %q, want HEAD", Ticker)
	}
}
```

- [ ] **Step 2: Запустить тест и убедиться, что он падает**

Run: `go test ./internal/service/trading_strategy/rsi_pullback/strategy/head/ -v`
Expected: FAIL — пакета `head` не существует.

- [ ] **Step 3: Создать пакет**

Создать `internal/service/trading_strategy/rsi_pullback/strategy/head/head.go`:

```go
// Package head supplies the ticker and rsi_pullback Params for HEAD (МКПАО «Хэдхантер»,
// обыкновенные акции, лот 1).
//
// ЗАЯВКА БЫЛА НА HHRU. Тикер мёртв: получасовой и дневной ряды обрываются 2024-08-09, в holdout
// скринера у него ноль сделок. Бумага редомицилирована (HeadHunter Group PLC -> МКПАО
// «Хэдхантер»), торги остановлены 2024-08-09 и возобновлены 2024-09-26 под тикером HEAD. Завести
// HHRU в боевую вселенную значило бы поставить туда неторгуемый инструмент. Решение владельца
// 2026-09-02: калибруется HEAD — та же бумага, живой тикер, непрерывный ряд.
//
// СОСТОЯНИЕ: КАЛИБРОВКА НЕ ПРОВОДИЛАСЬ. Пакет возвращает core.DefaultParams() — baseline ядра, не
// подобранный под этот инструмент. Так и должно быть до конца калибровки: пакет заведён заранее,
// чтобы прогоны шли через тот же реестр, что и у остальных двадцати тикеров, а не через
// generic-ветку. Состояние держит head_test.go.
//
// ОКНО И СХЕМА АДАПТИРОВАННЫЕ. Расчётное окно — 2024-09-02 … 2026-09-02 (24 месяца; фактические
// данные внутри него начинаются 2024-09-26, в день возобновления торгов). В окне 25 788
// получасовых баров, из них 17 507 будних; дневная серия в окне 610 свечей, из них 491 будняя.
// Схема прогонов -months 24 -train-months 12 -test-months 3 -min-trades 20 -metric profit_factor,
// четыре фолда встык (12 + 3*4 = 24); число фолдов проверено контрольным прогоном ДО раскладки
// сеток. Штатная 36/12/6 недоступна: у нынешней бумаги 23.2 месяца истории. Схему 24/8/4 (тоже
// четыре фолда) отвергли: восьмимесячное обучающее окно даёт около 30 сделок, и при -min-trades 20
// половина сеток тонет ещё до ранжирования. Цена выбора: OOS-фолд всего три месяца, около 11
// сделок, пул четырёх фолдов около 45. ЧИСЛА HEAD СНЯТЫ НА 24-МЕСЯЧНОМ ОКНЕ и построчно с
// 36-месячным каталогом НЕ СРАВНИМЫ (прецедент — IVAT и YDEX).
//
// ДЫРА ОСТАНОВКИ ТОРГОВ ЛЕЖИТ СЛЕВА ОТ ОКНА: единственный разрыв длиннее четырёх дней —
// 2024-08-09 … 2024-09-26 (47 дней), и левая граница окна стоит внутри него, поэтому первый бар
// окна 2024-09-26. Ни одна EMA, ни один ATR и ни одна сделка не считаются через разрыв.
// -refresh во время калибровки НЕ запускать: он сдвинет обе границы и сделает все замеры
// несравнимыми.
//
// СЕТКИ ЭТОГО ТИКЕРА ДЕРЖАТСЯ МАКСИМАЛЬНО ШИРОКИМИ — решение владельца 2026-09-02, то же, что по
// BANEP, ASTR и SNGSP. Обрезок осей нет; замеры, которые в узком каталоге были бы основанием
// вырезать край, живут в _comment сеток как предупреждения. Сторожевой тест TestHEADGridsStayWide
// запрещает УРЕЗАТЬ ось, а не расширять её. Жёстких инвариантов три: RSILower <= 50,
// RSIPeriod >= 3, ось тренда не порождает пар EMAFast >= EMASlow.
//
// АПРИОР, записанный ДО прогонов. Скринер (pullback_screen_Minutes30_20260804_232456.md, строка 47
// из 99 прошедших вселенную): оборот 418 млн ₽, дневной ATR 4.04%, баров 31 240, TradesMed 37,
// PFmed 1.14, Capped 0/24, SilentCfg 0/24, плато 17%, PFmed HO 8.65 на 8 сделках — априор слабый.
// Контрольный прогон дефолтов ядра на расчётном окне: 90 сделок, PF 1.311, net +10 965.97 ₽ —
// ДЕВЯТОЕ место в ряду baseline каталога (YDEX 1.778, LSNGP 1.554, ELFV 1.546, SNGSP 1.535,
// IVAT 1.432, LENT 1.417, SVAV 1.368, BANEP 1.336, HEAD 1.311, BSPB 1.114, ASTR 1.093, SIBN 1.027,
// NVTK 0.984). Веса априору не придаётся: вопрос «предсказывают ли колонки скринера исход
// протокола» каталог закрыл восемь раз подряд — не предсказывают ни снизу, ни сверху.
//
// ЛИНЕЙКА ИЗДЕРЖЕК (урок KMAZ) ПРИМЕНЕНА ПЕРВЫМ ДЕЙСТВИЕМ: шаг цены 1 ₽ при медианной цене 3066 ₽
// = 0.033% цены, круг из двух шагов 0.065% ниже моделируемых 0.1% — тикер линейку проходит.
// Чувствительность baseline: круг 0.1% -> PF 1.311, 0.2% -> 1.054, 0.3% -> 0.816 (90 сделок во
// всех строках). Запас тоньше, чем у предшественников.
//
// ЧТО ЗНАЕМ ОБ ИНСТРУМЕНТЕ ДО ПРОГОНОВ (замеры на расчётном окне, полностью —
// reports/_analysis/head_pullback_prep_measurements.md):
//
//   - РЕЖИМ БЕЗ ЕДИНОГО РАСТУЩЕГО ПОЛУГОДИЯ (форма DIAS): итог окна −32.2%, максимальная просадка
//     −51.7%, полугодия −2.3 / −4.3 / −16.0 / −11.7%. Три полугодия baseline прибыльны
//     (PF 1.713, 2.003, 0.549, 1.795); третье убыточно и съедает 57.9% чужого net.
//   - ЛИКВИДНОСТЬ ВЫСОКАЯ: оборот по будним дням медиана 522 млн ₽, p10 249, p90 1475 (n = 491);
//     баров в дне медиана 35, дней короче 20 баров 0.4%. Исполнительного риска нет.
//   - ГЭП РЕАЛИЗОВАННЫЙ, А НЕ СПЯЩИЙ — главное отличие HEAD от всего каталога. Гэпы 2024-12-17
//     −9.99%, 2026-05-12 −8.16%, 2025-09-26 −6.14% (3.5, 2.9 и 2.1 дневных ATR, стопом не
//     держатся; движок моделирует разрыв честно через min(level, open)). Доля сделок baseline с
//     НОЧЁВКОЙ 40.0% (36 из 90) против 1.6% у SNGSP, и одна сделка (вход 2025-09-25) уже прошла
//     через сентябрьский гэп и закрылась по стопу с −6.28% — худшей строкой журнала.
//   - УДЕРЖАНИЕ: медиана 9 баров, p90 18, максимум 31; в календарных днях медиана 0, максимум 1.
//   - ТРЕНДОВЫЙ ДОПУСК РОВНЫЙ: доля будних баров с EMAFast > EMASlow в полосе 44.2-48.4% по всей
//     таблице 3..40 x 20..250 — ровнее, чем у любого тикера каталога. Максимум PF стоит на паре
//     3/20 (2.669 на 57 сделках), то есть НИЖЕ канонической оси; вся половина EMAFast 20..40
//     лежит в 1.10-1.30.
//   - ДНЕВНОЙ ГЕЙТ НЕСЁТ ВЕСЬ EDGE: без него 327 сделок при PF 0.930 (убыточно) против 90 при
//     1.311. Исчерпанная ветка имеет максимум на 1.0 (1.583/52); свежая монотонно вредит
//     (0 -> 1.311, 0.6 -> 0.886).
//   - КАПКАН ШИРОКОГО СТОПА, ДЕВЯТЫЙ ТИКЕР ПОДРЯД, С НАСЫЩЕНИЕМ НА 1.3: при цели 0.6 стоп
//     0.5 -> 1.311/90, 1.0 -> 1.742/89, 1.3 -> 2.392/88, а 1.5 и 2.0 совпадают с 1.3 побайтово.
//     Доля SL-выходов при стопе 1.3 падает 20.0% -> 1.1%, ночёвка растёт до 48.9%.
//   - ЦЕЛЬ ПЛОСКАЯ: размах всей оси 0.13 PF, максимум на 0.5 (1.337), шире 1.5 дневного ATR
//     недостижима (колонки 1.5, 2.0 и 2.5 совпадают побайтово).
//   - ОБЪЁМНЫЙ ГЕЙТ НЕРАЗЛИЧИМ с выключенным: весь массив множителя, базы и окна лежит в полосе
//     0.955-1.439 вокруг baseline 1.311, сходимости нет ни на одной оси.
//   - ВЫХОД: максимум на 80 (1.470), второе значение на 55 (1.430) — у края канонической оси,
//     поэтому ось расширена вниз до 50 (1.380). Сам RSI-выход критичен: при UseRSIExit 0
//     конфигурации держатся только на редких больших целях и обваливают win rate до 15.6%.
//     Трейл как добавка даёт максимум +0.029 PF, с дистанции 1.0 не срабатывает вовсе.
package head

import "tinvest/internal/service/trading_strategy/rsi_pullback/strategy/core"

// Ticker is the instrument this package parameterises.
const Ticker = "HEAD"

// DefaultParams returns the core baseline: HEAD is not calibrated yet.
func DefaultParams() core.Params {
	return core.DefaultParams()
}
```

- [ ] **Step 4: Запустить тест пакета**

Run: `go test ./internal/service/trading_strategy/rsi_pullback/strategy/head/ -v`
Expected: PASS оба теста.

- [ ] **Step 5: Зарегистрировать тикер в реестре бэктеста**

В `internal/service/backtest/rsi_pullback_registry.go` добавить импорт рядом с соседними:

```go
	rsipullbackhead "tinvest/internal/service/trading_strategy/rsi_pullback/strategy/head"
```

и строку в карту реестра на алфавитную позицию, между `rsipullbackgazp` и `rsipullbackivat`:

```go
	rsipullbackhead.Ticker: rsiPullbackBindingFor(rsipullbackhead.Ticker, rsipullbackhead.DefaultParams),
```

- [ ] **Step 6: Добавить сторожевой тест реестра**

Дописать в `internal/service/backtest/rsi_pullback_registry_test.go` (импорт
`rsipullbackhead "tinvest/internal/service/trading_strategy/rsi_pullback/strategy/head"` в тестовом
файле нужен свой):

```go
// TestRSIPullbackHEADTracksBaseline сторожит ЧЕСТНОЕ состояние: HEAD заведён в реестр до
// калибровки, чтобы прогоны шли через реестр, а не через generic-ветку, и обязан возвращать ровно
// baseline ядра. Тест заменяется снимком литерала в Task 12.
func TestRSIPullbackHEADTracksBaseline(t *testing.T) {
	b, ok := rsiPullbackRegistry[rsipullbackhead.Ticker]
	if !ok {
		t.Fatal("HEAD отсутствует в rsiPullbackRegistry: тикер провалится в generic-ветку")
	}
	p, pok := b.DefaultParams().(core.Params)
	if !pok {
		t.Fatalf("HEAD: DefaultParams() вернул %T, want core.Params", b.DefaultParams())
	}
	if p != core.DefaultParams() {
		t.Fatalf("HEAD ещё не откалиброван, params обязаны совпадать с baseline:\n got: %+v\nwant: %+v", p, core.DefaultParams())
	}
	if got := b.Build(p).Ticker(); got != "HEAD" {
		t.Fatalf("Ticker() = %q, want HEAD", got)
	}
}
```

- [ ] **Step 7: Запустить тесты**

Run: `go test ./internal/service/backtest/ ./internal/service/trading_strategy/rsi_pullback/... -v`
Expected: PASS.

- [ ] **Step 8: Коммит**

```bash
git add internal/service/trading_strategy/rsi_pullback/strategy/head internal/service/backtest
git commit -m "feat(rsi_pullback): пакет HEAD в состоянии отслеживания baseline"
```

---

### Task 3: Тема `screen` — цена двух гейтов и проверка числа фолдов

**Files:**
- Modify: `data/params/rsi_pullback/head/cal_screen.json` (строка результата в `_comment`)
- Create: отчёты в `reports/HEAD_screen/` (каталог `reports/` в `.gitignore` — отчёты остаются
  локальными, в коммит идут только строки результата в `_comment`)

**Interfaces:**
- Consumes: каталог сеток из Task 1, реестр из Task 2.
- Produces: пофолдовые победители пары (`UseDayATRGate`, `UseVolume`) — их читает Task 11 при
  сборке точки.

- [ ] **Step 1: Прогнать тему**

Run:
```bash
go run ./cmd/backtest -ticker HEAD -strategy rsi_pullback -interval Minutes30 \
  -calibrate data/params/rsi_pullback/head/cal_screen.json -out ./reports/HEAD_screen \
  -months 24 -train-months 12 -test-months 3 -min-trades 1 -metric profit_factor
```
Expected: отчёт `reports/HEAD_screen/HEAD_rsi_pullback_Minutes30_<ts>_walkforward.md`.

- [ ] **Step 2: ПРОВЕРИТЬ ЧИСЛО ФОЛДОВ**

Открыть шапку отчёта. Строка «Фолдов:» обязана показывать **4**.

Если там **3** — остановить работу, не запускать остальные темы, доложить владельцу с цитатой
шапки отчёта. Схему менять самостоятельно нельзя: правило сборки точки «≥3 фолда из 4» при трёх
фолдах вырождается в «3 из 3», и это меняет планку задним числом.

- [ ] **Step 3: Выписать пофолдовых победителей**

Из отчёта выписать по каждому из четырёх фолдов: выбранную пару (`UseDayATRGate`, `UseVolume`),
in-sample PF, OOS PF, число сделок OOS. Плюс pooled OOS PF и pooled число сделок.

- [ ] **Step 4: Вписать результат в `_comment`**

Дописать после «РЕЗУЛЬТАТ ПРОГОНА 2026-09-02: » одной строкой: путь к отчёту, число фолдов, pooled
OOS PF и сделки, пофолдовые пары и их OOS PF, вывод о том, подтверждается ли точечный замер («без
дневного гейта стратегия убыточна») пофолдово.

- [ ] **Step 5: Коммит**

```bash
git add data/params/rsi_pullback/head/cal_screen.json
git commit -m "feat(rsi_pullback): HEAD, тема screen — цена гейтов"
```

---

### Task 4: Тема `entry` — ключевая, рельеф с максимумом на умеренной глубине

**Files:**
- Modify: `data/params/rsi_pullback/head/cal_entry.json` (строка результата в `_comment`)
- Create: отчёты в `reports/HEAD_entry/`

**Interfaces:**
- Consumes: каталог сеток из Task 1, реестр из Task 2.
- Produces: пофолдовых победителей `RSIPeriod`, `RSILower` (принимаются в точку) и `RSIUpper`
  (идёт как контекст, поле берётся из темы `exit`) — их читает Task 11.

- [ ] **Step 1: Прогнать тему**

Run:
```bash
go run ./cmd/backtest -ticker HEAD -strategy rsi_pullback -interval Minutes30 \
  -calibrate data/params/rsi_pullback/head/cal_entry.json -out ./reports/HEAD_entry \
  -months 24 -train-months 12 -test-months 3 -min-trades 20 -metric profit_factor
```
432 комбинации на четырёх фолдах — прогон занимает единицы минут, это нормально.

- [ ] **Step 2: Выписать пофолдовых победителей и pooled**

По каждому фолду: тройка (`RSIPeriod`, `RSILower`, `RSIUpper`), in-sample PF, OOS PF, сделки OOS.
Плюс pooled OOS PF, pooled сделки, win rate.

- [ ] **Step 3: Проверить планку по этой теме**

Оба критерия раздельно и прямым текстом:
1. pooled OOS PF ≥ 1.5 при ≥ 20 сделках в пуле — взят или нет;
2. ведущая ось `RSILower` выбрана одинаково в ≥ 3 фолдах из 4 — взят или нет.

Вырожденный фолд (ни одной убыточной сделки, `MaxDD` = 0) в пользу тикера не засчитывается.
Планку не пересматривать ни при каком исходе.

- [ ] **Step 4: Сверить с точечным замером**

Точечный максимум живой зоны — RSI(4)@35 (1.578/118). Записать, попали ли фолды в эту зону или
разошлись, и не ушёл ли хоть один фолд в шумный край (уровни 10–15 на периодах 6–8, где выборка
1–4 сделки за всё окно).

- [ ] **Step 5: Вписать результат в `_comment` и закоммитить**

```bash
git add data/params/rsi_pullback/head/cal_entry.json
git commit -m "feat(rsi_pullback): HEAD, тема entry — ключевая"
```

---

### Task 5: Тема `trend` — вторая ключевая, ось сдвинута вниз

**Files:**
- Modify: `data/params/rsi_pullback/head/cal_trend.json` (строка результата в `_comment`)
- Create: отчёты в `reports/HEAD_trend/`

**Interfaces:**
- Consumes: каталог сеток из Task 1, реестр из Task 2.
- Produces: пофолдовых победителей `EMAFast`, `EMASlow` — их читает Task 11.

- [ ] **Step 1: Прогнать тему**

Run:
```bash
go run ./cmd/backtest -ticker HEAD -strategy rsi_pullback -interval Minutes30 \
  -calibrate data/params/rsi_pullback/head/cal_trend.json -out ./reports/HEAD_trend \
  -months 24 -train-months 12 -test-months 3 -min-trades 20 -metric profit_factor
```

- [ ] **Step 2: Выписать пофолдовых победителей и pooled**

По каждому фолду: пара (`EMAFast`, `EMASlow`), in-sample PF, OOS PF, сделки OOS. Плюс pooled.

- [ ] **Step 3: Проверить планку по этой теме**

Те же два критерия раздельно, ведущая ось — `EMASlow`.

- [ ] **Step 4: Проверить два специальных исхода**

**Исход А — победа пары 3/20.** Точечный максимум стоит именно там (2.669/57) при допуске 48.4% и
разносе всего в 6.7 раза. Такой победитель читается как «трендовый фильтр бумаге почти не нужен» и
записывается как СВОЙСТВО, а не как параметр, которому доверяют.

**Исход Б — упор в верхний край быстрой оси (`EMAFast` = 10) в большинстве фолдов.** Это означает,
что ось поставлена не там: половина канонической таблицы (`EMAFast` 20…40) в теме не измерялась
ради расширения вниз. Записать вывод прямым текстом в `_comment` и в Task 11; решение о повторной
раскладке принимает владелец, самостоятельно ось не переставлять.

- [ ] **Step 5: Вписать результат в `_comment` и закоммитить**

```bash
git add data/params/rsi_pullback/head/cal_trend.json
git commit -m "feat(rsi_pullback): HEAD, тема trend — вторая ключевая"
```

---

### Task 6: Темы `day` и `day_spent` — дневной гейт

**Files:**
- Modify: `data/params/rsi_pullback/head/cal_day.json`, `cal_day_spent.json` (строки результата)
- Create: отчёты в `reports/HEAD_day/`, `reports/HEAD_day_spent/`

**Interfaces:**
- Consumes: каталог сеток из Task 1, реестр из Task 2.
- Produces: пофолдовых победителей `FreshDayATR` и `SpentDayATR` из обеих тем — их сверяет и
  читает Task 11.

- [ ] **Step 1: Прогнать обе темы**

Run:
```bash
go run ./cmd/backtest -ticker HEAD -strategy rsi_pullback -interval Minutes30 \
  -calibrate data/params/rsi_pullback/head/cal_day.json -out ./reports/HEAD_day \
  -months 24 -train-months 12 -test-months 3 -min-trades 20 -metric profit_factor
go run ./cmd/backtest -ticker HEAD -strategy rsi_pullback -interval Minutes30 \
  -calibrate data/params/rsi_pullback/head/cal_day_spent.json -out ./reports/HEAD_day_spent \
  -months 24 -train-months 12 -test-months 3 -min-trades 20 -metric profit_factor
```

- [ ] **Step 2: Выписать победителей обеих тем и сверить их между собой**

По каждому фолду каждой темы: значения (`FreshDayATR`, `SpentDayATR`), in-sample PF, OOS PF,
сделки. Затем сверить `SpentDayATR` между темами пофолдово: совпадение означает, что ветки не
взаимодействуют; расхождение записывается как свойство и разбирается в Task 11 (поле берётся из
темы `day`, значение `day_spent` идёт как контекст).

- [ ] **Step 3: Проверить ожидания замера**

Ожидание 1: `FreshDayATR` 0 побеждает большинством (точечный рельеф монотонно падает от нуля).
Ожидание 2: `SpentDayATR` тянет в зону 0.9–1.0. Ожидание 3: хвост 1.25–1.5 не выбирается ни одним
фолдом (там 17–33 сделки за всё окно). Каждое подтвердившееся и каждое опровергнутое ожидание
записывается прямым текстом — опровержение это находка, а не ошибка.

- [ ] **Step 4: Вписать результаты в оба `_comment` и закоммитить**

```bash
git add data/params/rsi_pullback/head/cal_day.json data/params/rsi_pullback/head/cal_day_spent.json
git commit -m "feat(rsi_pullback): HEAD, темы day и day_spent"
```

---

### Task 7: Тема `volume` — объёмный гейт

**Files:**
- Modify: `data/params/rsi_pullback/head/cal_volume.json` (строка результата)
- Create: отчёты в `reports/HEAD_volume/`

**Interfaces:**
- Consumes: каталог сеток из Task 1, реестр из Task 2.
- Produces: пофолдовых победителей `VolMult`, `VolBaseDays` — их читает Task 11.

- [ ] **Step 1: Прогнать тему**

Run:
```bash
go run ./cmd/backtest -ticker HEAD -strategy rsi_pullback -interval Minutes30 \
  -calibrate data/params/rsi_pullback/head/cal_volume.json -out ./reports/HEAD_volume \
  -months 24 -train-months 12 -test-months 3 -min-trades 20 -metric profit_factor
```

- [ ] **Step 2: Выписать победителей и pooled**

По каждому фолду: пара (`VolMult`, `VolBaseDays`), in-sample PF, OOS PF, сделки OOS. Плюс pooled.

- [ ] **Step 3: Проверить ожидание «гейт неразличим»**

Точечный замер даёт полосу 0.955–1.419 вокруг baseline 1.311. Ожидание: разброс победителей по
фолдам покрывает всю сетку, большинства 3 из 4 нет ни у одного значения, оба поля уходят на
дефолты. Записать, подтвердилось ли; если какое-то значение всё же набрало большинство — проверить
его выборку (сколько сделок в обучающем окне) и записать это число рядом.

- [ ] **Step 4: Вписать результат в `_comment` и закоммитить**

```bash
git add data/params/rsi_pullback/head/cal_volume.json
git commit -m "feat(rsi_pullback): HEAD, тема volume"
```

---

### Task 8: Тема `vol_window` — девятая проверка гипотезы «ликвидность → дефолт»

**Files:**
- Modify: `data/params/rsi_pullback/head/cal_vol_window.json` (строка результата)
- Create: отчёты в `reports/HEAD_vol_window/`

**Interfaces:**
- Consumes: каталог сеток из Task 1, реестр из Task 2.
- Produces: пофолдовых победителей `VolLookbackBars` (первичная ось) и `VolMult` (вторичная,
  контекст для сверки с темой `volume`) — их читает Task 11.

- [ ] **Step 1: Прогнать тему**

Run:
```bash
go run ./cmd/backtest -ticker HEAD -strategy rsi_pullback -interval Minutes30 \
  -calibrate data/params/rsi_pullback/head/cal_vol_window.json -out ./reports/HEAD_vol_window \
  -months 24 -train-months 12 -test-months 3 -min-trades 20 -metric profit_factor
```

- [ ] **Step 2: Выписать победителей и pooled**

По каждому фолду: пара (`VolLookbackBars`, `VolMult`), in-sample PF, OOS PF, сделки OOS.

- [ ] **Step 3: Сверить `VolMult` с темой `volume`**

Если обе темы дали по `VolMult` большинство, но на разных значениях — это КОНФЛИКТ ТЕМ, и он
записывается честно (прецедент SNGSP: `volume` дал 2.5, `vol_window` дал 1.2). Соответствие «поле →
тема» отдаёт `VolMult` теме `volume`; расхождение означает, что множитель чувствителен к окну, на
котором меряется, и это риск оси, а не выбор лучшего числа.

- [ ] **Step 4: Записать вердикт по гипотезе каталога**

Гипотеза: «чем ликвиднее бумага, тем ближе оптимум окна к дефолту 3». Точечный замер на HEAD её не
поддерживает — рельеф плоский и слегка растущий к широкому краю. Записать, что дал walk-forward.

- [ ] **Step 5: Вписать результат в `_comment` и закоммитить**

```bash
git add data/params/rsi_pullback/head/cal_vol_window.json
git commit -m "feat(rsi_pullback): HEAD, тема vol_window"
```

---

### Task 9: Тема `risk` — стоп, цель и обязательная проверка капкана

**Files:**
- Modify: `data/params/rsi_pullback/head/cal_risk.json` (строка результата)
- Create: отчёты в `reports/HEAD_risk/`, `reports/HEAD_risk_probe/`

**Interfaces:**
- Consumes: каталог сеток из Task 1, реестр из Task 2.
- Produces: пофолдовых победителей `StopDailyATR`, `TPDailyATR` и числа капкана (доля SL-выходов,
  доля ночёвок, удержание, экспозиция под гэпом) — их читает Task 11.

- [ ] **Step 1: Прогнать тему**

Run:
```bash
go run ./cmd/backtest -ticker HEAD -strategy rsi_pullback -interval Minutes30 \
  -calibrate data/params/rsi_pullback/head/cal_risk.json -out ./reports/HEAD_risk \
  -months 24 -train-months 12 -test-months 3 -min-trades 20 -metric profit_factor
```

- [ ] **Step 2: Выписать победителей и pooled**

По каждому фолду: пара (`StopDailyATR`, `TPDailyATR`), in-sample PF, OOS PF, сделки OOS.

- [ ] **Step 3: Прогнать зонд капкана**

Если хотя бы один фолд выбрал стоп ≥ 1.0, прогнать зонд с победившей парой на полной истории окна,
чтобы получить журнал сделок:

```bash
go run ./cmd/backtest -ticker HEAD -strategy rsi_pullback -interval Minutes30 \
  -params /tmp/head_risk_probe.json -out ./reports/HEAD_risk_probe -months 24
```

`/tmp/head_risk_probe.json` — плоский JSON восемнадцати полей: дефолты ядра, где `StopDailyATR` и
`TPDailyATR` заменены победившей парой. Пример структуры (значения стопа и цели подставить свои):

```json
{"RSIPeriod":4,"RSILower":30,"RSIUpper":70,"EMAFast":10,"EMASlow":100,"DailyATRPeriod":14,
 "UseDayATRGate":1,"FreshDayATR":0,"SpentDayATR":0.8,"StopDailyATR":1.3,"TPDailyATR":0.6,
 "UseVolume":0,"VolBaseDays":14,"VolLookbackBars":3,"VolMult":1.2,"UseRSIExit":1,
 "UseTrail":0,"TrailDailyATR":0}
```

- [ ] **Step 4: Посчитать четыре числа капкана по `_trades.csv` зонда**

1. **Доля SL-выходов** — сколько строк с `reason=SL` из всех. Опорные числа: baseline 20.0%
   (18 из 90), зонд стопа 1.3 — 1.1% (1 из 88).
2. **Доля ночёвок** — сколько сделок, у которых дата `exit_time` больше даты `entry_time`. Опорные:
   baseline 40.0%, зонд стопа 1.3 — 48.9%, связка «широкий вход + широкий стоп» — 74.3%.
3. **Удержание** — медиана и максимум `bars_held`. Опорные: baseline 9 и 31, зонд — 10.5 и 52.
4. **Экспозиция под гэпом** — сколько сделок удержаны через **2024-12-17**, **2026-05-12**,
   **2025-09-26** (дата входа строго раньше даты гэпа, дата выхода не раньше), и сколько вошли
   прямо в эти дни. Опорное: baseline — одна сделка через 2025-09-26 с результатом −6.28%.

Вывод формулируется прямым текстом: если PF вырос, а доля SL упала — рост объяснён уходом убытков
в RSI-выход, а не улучшением защиты. Если при этом выросла доля ночёвок — риск гэпа усилен.

- [ ] **Step 5: Вписать результат и числа капкана в `_comment`, закоммитить**

```bash
git add data/params/rsi_pullback/head/cal_risk.json
git commit -m "feat(rsi_pullback): HEAD, тема risk и проверка капкана"
```

---

### Task 10: Темы `exit` и `trail` — выходы

**Files:**
- Modify: `data/params/rsi_pullback/head/cal_exit.json`, `cal_trail.json` (строки результата)
- Create: отчёты в `reports/HEAD_exit/`, `reports/HEAD_trail/`

**Interfaces:**
- Consumes: каталог сеток из Task 1, реестр из Task 2.
- Produces: пофолдовых победителей `RSIUpper` (тема `exit`) и `UseRSIExit`, `UseTrail`,
  `TrailDailyATR` (тема `trail`) — их читает Task 11.

- [ ] **Step 1: Прогнать обе темы**

Run:
```bash
go run ./cmd/backtest -ticker HEAD -strategy rsi_pullback -interval Minutes30 \
  -calibrate data/params/rsi_pullback/head/cal_exit.json -out ./reports/HEAD_exit \
  -months 24 -train-months 12 -test-months 3 -min-trades 20 -metric profit_factor
go run ./cmd/backtest -ticker HEAD -strategy rsi_pullback -interval Minutes30 \
  -calibrate data/params/rsi_pullback/head/cal_trail.json -out ./reports/HEAD_trail \
  -months 24 -train-months 12 -test-months 3 -min-trades 20 -metric profit_factor
```

- [ ] **Step 2: Выписать победителей обеих тем**

Тема `exit`: по каждому фолду `RSIUpper`, in-sample PF, OOS PF, сделки. Тема `trail`: по каждому
фолду тройка (`UseRSIExit`, `UseTrail`, `TrailDailyATR`), те же числа.

- [ ] **Step 3: Сверить `RSIUpper` с темой `entry`**

Тема `entry` свипует `RSIUpper` в связке, `exit` — изолированно. Выписать оба набора пофолдовых
значений и записать, совпадают ли. Расхождение ожидаемо (оси взаимодействуют) и означает, что поле
берётся из темы `exit`, а значение из `entry` идёт как контекст.

- [ ] **Step 4: Проверить два ожидания темы `trail`**

Ожидание 1: `UseRSIExit` 1 побеждает во всех фолдах — при нуле конфигурация обваливает win rate до
15.6% на дальней цели. Ожидание 2: если победившая дистанция трейла ≥ 1.0, трейл не срабатывает
вовсе (результат побайтово равен его отсутствию), и по правилу упрощения в точку идёт `UseTrail` 0
и `TrailDailyATR` 0. Записать, какое из ожиданий подтвердилось.

- [ ] **Step 5: Вписать результаты в оба `_comment` и закоммитить**

```bash
git add data/params/rsi_pullback/head/cal_exit.json data/params/rsi_pullback/head/cal_trail.json
git commit -m "feat(rsi_pullback): HEAD, темы exit и trail"
```

---

### Task 11: Сборка точки, её walk-forward и проверка стоп-условия

**Files:**
- Create: `data/params/rsi_pullback/head/plateau_point.json`
- Create: `data/params/rsi_pullback/head/plateau_neighbour_*.json` (по одному на проверяемого
  соседа)
- Create: отчёты в `reports/HEAD_point_oos/`, `reports/HEAD_point_comm/`,
  `reports/HEAD_point_full/`, `reports/HEAD_neighbour_*/`
- Create: `docs/superpowers/plans/task-11-report-head.md` (рабочая записка с полными выкладками)

**Interfaces:**
- Consumes: пофолдовых победителей всех десяти тем (Tasks 3–10).
- Produces: `plateau_point.json` — его читает Task 12 (литерал) и Task 16 (сверка).

- [ ] **Step 1: Собрать точку по правилу «≥3 фолда из 4»**

Для каждого из восемнадцати полей `core.Params`: взять тему, которая его меряет; посмотреть
победителей по четырём фолдам; принять значение, только если за него высказались **не менее трёх
фолдов из четырёх**; иначе оставить дефолт ядра. Ничья 2/2 большинством не считается.
Соответствие «поле → тема»:

| Поле | Тема |
|---|---|
| `RSIPeriod`, `RSILower` | `entry` |
| `RSIUpper` | `exit` (в `entry` идёт как контекст) |
| `EMAFast`, `EMASlow` | `trend` |
| `DailyATRPeriod` | нигде не свипуется — дефолт 14 |
| `UseDayATRGate` | `screen` |
| `FreshDayATR`, `SpentDayATR` | `day` (+ `day_spent` как контекст) |
| `StopDailyATR`, `TPDailyATR` | `risk` |
| `UseVolume` | `screen` |
| `VolMult`, `VolBaseDays` | `volume` |
| `VolLookbackBars` | `vol_window` |
| `UseRSIExit`, `UseTrail`, `TrailDailyATR` | `trail` |

Правило для `UseTrail`: если тема выбрала `TrailDailyATR` ≥ 1.0 (дистанция, на которой трейл на
HEAD не срабатывает ни разу), в точку идёт `UseTrail` 0 и `TrailDailyATR` 0 — конфигурация проще
при идентичном результате (прецедент ASTR).

- [ ] **Step 2: Записать деривацию поле за полем**

Создать `plateau_point.json` с сеткой из одного значения на ключ (это требование
`TestRSIPullbackPlateauFilesArePoints`) и `_comment`, где для КАЖДОГО из восемнадцати полей
написано: из какой темы, какие значения выбрали четыре фолда, взято большинство или поле упало на
дефолт. Плюс: вердикт по планке по обеим ключевым темам раздельно; оговорка о подглядывании —
точку собрал исполнитель, видевший всю историю тем и точечные ряды на полном окне, поэтому её
числа нельзя сравнивать с pooled OOS тем как замену процедуре отбора, и единственная честная
проверка — walk-forward самой точки.

- [ ] **Step 3: Прогнать walk-forward точки**

Run:
```bash
go run ./cmd/backtest -ticker HEAD -strategy rsi_pullback -interval Minutes30 \
  -calibrate data/params/rsi_pullback/head/plateau_point.json -out ./reports/HEAD_point_oos \
  -months 24 -train-months 12 -test-months 3 -min-trades 20 -metric profit_factor
```
Выписать: pooled OOS PF, число сделок, win rate, compounded, пофолдовые in-sample и OOS PF,
сделки OOS, наличие вырожденных фолдов (`MaxDD` = 0).

- [ ] **Step 4: Проверить пункты 1 и 2 стоп-условия**

Пункт 1: pooled OOS PF ≥ 1.0. Пункт 2: ≥ 20 сделок. Если любой провален — **остановиться**,
записать числа, доложить владельцу; задачи 12–16 не выполняются.

- [ ] **Step 5: Проверить пункт 3 стоп-условия — удвоенные издержки**

Run:
```bash
go run ./cmd/backtest -ticker HEAD -strategy rsi_pullback -interval Minutes30 \
  -calibrate data/params/rsi_pullback/head/plateau_point.json -out ./reports/HEAD_point_comm \
  -months 24 -train-months 12 -test-months 3 -min-trades 20 -metric profit_factor -commission 0.001
```
Записать pooled OOS PF и долю потери относительно основного прогона. Опорное число: baseline
теряет 19.6% PF при удвоении круга (1.311 → 1.054). Если потеря точки заметно больше — это её
главный экономический риск, и он идёт в доку пакета.

- [ ] **Step 6: Соседи плато**

Для трёх осей, давших принятые значения (обязательно для `StopDailyATR` и для ведущей оси входа
`RSILower`; третья — любая другая принятая), создать файлы-соседи: копия точки, где ОДНО поле
сдвинуто на соседний шаг сетки соответствующей темы. Прогнать каждого той же командой, что точку, с
`-out ./reports/HEAD_neighbour_<имя>`. Записать вердикт по каждой оси раздельно: плато (числа
рядом), склон (сосед лучше при меньшей выборке), пик (оба соседа хуже). Пик записывается как риск —
структурная неустойчивость оси.

Отдельно проверить вырожденное плато: если в точке `UseTrail` 1 с малой дистанцией, соседи по
`StopDailyATR` могут совпасть побайтово — трейл перекрывает стоп. Такое совпадение записывается как
инертность оси, а не как устойчивость.

- [ ] **Step 7: Полугодия принятой точки**

Прогнать точку на полной истории без train/test, чтобы получить `_trades.csv`:

```bash
go run ./cmd/backtest -ticker HEAD -strategy rsi_pullback -interval Minutes30 \
  -params /tmp/head_point.json -out ./reports/HEAD_point_full -months 24
```
(`/tmp/head_point.json` — плоский JSON тех же восемнадцати полей.)

Разложить сделки по четырём полугодиям расчётного окна (границы — в Global Constraints) и
посчитать по каждому: число сделок, net, PF, долю от общего net. Критерий: **ни одно полугодие не
даёт больше половины net**. Опорная картина baseline: 62.9 / 54.3 / −57.9 / 40.7% — у baseline
критерий провален дважды, и это ожидаемо при четырёх полугодиях вместо шести; у точки записывается
факт, а не сглаживание. Отдельно проверить третье полугодие (2025-09-02 … 2026-02-27), убыточное у
baseline с PF 0.549.

- [ ] **Step 8: Удержание и экспозиция под гэпом**

На том же `_trades.csv` повторить измерения Task 9 Step 4 для принятой точки:

1. доля SL-выходов (опорное: baseline 20.0%);
2. доля ночёвок (опорное: baseline 40.0%);
3. удержание — медиана и максимум `bars_held` (опорное: baseline 9 и 31);
4. сделки, удержанные через **2024-12-17**, **2026-05-12**, **2025-09-26**, и сделки, вошедшие в
   эти дни; для каждой такой сделки — её PnL.

Вывод: если экспозиция выросла относительно baseline (одна сделка через один гэп), риск записывается
как УСИЛЕННЫЙ и идёт в доку пакета и в раздел рисков как календарное предупреждение. Если упала —
записывается как снятый риск с указанием, каким полем он снят.

- [ ] **Step 9: Записать рабочую записку**

Создать `docs/superpowers/plans/task-11-report-head.md` со всеми выкладками шагов 1–8: команды,
пути к отчётам, таблицы фолдов, полугодий, соседей, чисел капкана, удержания и гэповой экспозиции.
Записка — источник для доки пакета (Task 15).

- [ ] **Step 10: Коммит**

```bash
git add data/params/rsi_pullback/head docs/superpowers/plans/task-11-report-head.md
git commit -m "feat(rsi_pullback): HEAD, принятая точка и её замеры"
```

---

### Task 12: Литерал в пакете и снимок

**Files:**
- Modify: `internal/service/trading_strategy/rsi_pullback/strategy/head/head.go`
- Modify: `internal/service/trading_strategy/rsi_pullback/strategy/head/head_test.go`
- Modify: `internal/service/backtest/rsi_pullback_registry_test.go`

**Interfaces:**
- Consumes: `plateau_point.json` из Task 11.
- Produces: `head.DefaultParams()` как явный литерал — его читают Task 13 и Task 14.

- [ ] **Step 1: Заменить тест baseline-состояния снимком литерала**

В `head_test.go` удалить `TestParamsTrackTheBaselineUntilCalibrated` и записать снимок (значения
полей подставить из `plateau_point.json`):

```go
// TestParamsAreTheCalibratedSnapshot прибивает принятый 2026-09-02 литерал. Тест падает и на
// молчаливом дрейфе полей, и на откате к core.DefaultParams(): связь с baseline разорвана
// осознанно, и любое её восстановление обязано быть видимым в диффе.
func TestParamsAreTheCalibratedSnapshot(t *testing.T) {
	want := core.Params{
		RSIPeriod:       0, // подставить из точки
		RSILower:        0, // подставить из точки
		RSIUpper:        0, // подставить из точки
		EMAFast:         0, // подставить из точки
		EMASlow:         0, // подставить из точки
		DailyATRPeriod:  14,
		UseDayATRGate:   0, // подставить из точки
		FreshDayATR:     0, // подставить из точки
		SpentDayATR:     0, // подставить из точки
		StopDailyATR:    0, // подставить из точки
		TPDailyATR:      0, // подставить из точки
		UseVolume:       0, // подставить из точки
		VolBaseDays:     0, // подставить из точки
		VolLookbackBars: 0, // подставить из точки
		VolMult:         0, // подставить из точки
		UseRSIExit:      0, // подставить из точки
		UseTrail:        0, // подставить из точки
		TrailDailyATR:   0, // подставить из точки
	}
	if got := DefaultParams(); got != want {
		t.Fatalf("литерал HEAD разошёлся со снимком:\n got: %+v\nwant: %+v", got, want)
	}
}

// TestParamsAreNotTheCoreBaseline сторожит сам факт калибровки: совпадение с дефолтами ядра
// означало бы, что литерал откатили.
func TestParamsAreNotTheCoreBaseline(t *testing.T) {
	if DefaultParams() == core.DefaultParams() {
		t.Fatal("HEAD откалиброван, его params не могут совпадать с core.DefaultParams()")
	}
}
```

Если правило сборки оставило на дефолтах ядра ВСЕ восемнадцать полей (то есть ни одна тема не
набрала большинства), `TestParamsAreNotTheCoreBaseline` писать нельзя — вместо него оставить
`TestParamsTrackTheBaselineUntilCalibrated` с переписанным комментарием («калибровка проведена,
большинства не набрало ни одно поле») и доложить владельцу отдельной строкой: это тоже вердикт.

- [ ] **Step 2: Запустить тест и убедиться, что он падает**

Run: `go test ./internal/service/trading_strategy/rsi_pullback/strategy/head/ -v`
Expected: FAIL — `DefaultParams()` пока возвращает baseline.

- [ ] **Step 3: Поставить литерал в пакет**

В `head.go` заменить тело функции на явный литерал (значения — из точки):

```go
// DefaultParams returns the calibrated HEAD point accepted 2026-09-02.
func DefaultParams() core.Params {
	return core.Params{
		RSIPeriod:       0, // подставить из точки
		RSILower:        0, // подставить из точки
		RSIUpper:        0, // подставить из точки
		EMAFast:         0, // подставить из точки
		EMASlow:         0, // подставить из точки
		DailyATRPeriod:  14,
		UseDayATRGate:   0, // подставить из точки
		FreshDayATR:     0, // подставить из точки
		SpentDayATR:     0, // подставить из точки
		StopDailyATR:    0, // подставить из точки
		TPDailyATR:      0, // подставить из точки
		UseVolume:       0, // подставить из точки
		VolBaseDays:     0, // подставить из точки
		VolLookbackBars: 0, // подставить из точки
		VolMult:         0, // подставить из точки
		UseRSIExit:      0, // подставить из точки
		UseTrail:        0, // подставить из точки
		TrailDailyATR:   0, // подставить из точки
	}
}
```

Блок «СОСТОЯНИЕ: КАЛИБРОВКА НЕ ПРОВОДИЛАСЬ» в док-комментарии пока не трогать — его переписывает
Task 15.

- [ ] **Step 4: Заменить тест реестра**

В `internal/service/backtest/rsi_pullback_registry_test.go` заменить
`TestRSIPullbackHEADTracksBaseline` на проверку откалиброванного состояния, по образцу
`TestRSIPullbackSNGSPIsRegisteredAndCalibrated` (`rsi_pullback_registry_test.go:775`):

```go
// TestRSIPullbackHEADIsRegisteredAndCalibrated сторожит, что HEAD доехал до реестра бэктеста с
// откалиброванными параметрами, а не с baseline ядра.
func TestRSIPullbackHEADIsRegisteredAndCalibrated(t *testing.T) {
	b, ok := rsiPullbackRegistry[rsipullbackhead.Ticker]
	if !ok {
		t.Fatal("HEAD отсутствует в rsiPullbackRegistry: тикер провалится в generic-ветку")
	}
	p, pok := b.DefaultParams().(core.Params)
	if !pok {
		t.Fatalf("HEAD: DefaultParams() вернул %T, want core.Params", b.DefaultParams())
	}
	if p != rsipullbackhead.DefaultParams() {
		t.Fatalf("реестр отдаёт не литерал пакета:\n got: %+v\nwant: %+v", p, rsipullbackhead.DefaultParams())
	}
	if got := b.Build(p).Ticker(); got != "HEAD" {
		t.Fatalf("Ticker() = %q, want HEAD", got)
	}
}
```

- [ ] **Step 5: Запустить тесты**

Run: `go test ./internal/service/backtest/ ./internal/service/trading_strategy/rsi_pullback/... -v`
Expected: PASS.

- [ ] **Step 6: Коммит**

```bash
git add internal/service/trading_strategy/rsi_pullback/strategy/head internal/service/backtest
git commit -m "feat(rsi_pullback): HEAD откалиброван — литерал вместо отслеживания baseline"
```

---

### Task 13: Реестр живого раннера

**Files:**
- Modify: `internal/service/trading_strategy/rsi_pullback/live/registry.go`
- Modify: `internal/service/trading_strategy/rsi_pullback/live/registry_test.go` (если в нём есть
  список ожидаемых тикеров)

**Interfaces:**
- Consumes: `head.Ticker`, `head.DefaultParams()` из Task 12.
- Produces: запись в `paramsByTicker` — её читает `ParamsFor` живого раннера и `cmd/pullparity`.

- [ ] **Step 1: Добавить импорт и запись в карту**

Импорт `"tinvest/internal/service/trading_strategy/rsi_pullback/strategy/head"` и строка в
`paramsByTicker` следом за `sngsp.Ticker` (`registry.go:317`):

```go
	head.Ticker:  head.DefaultParams(),
```

- [ ] **Step 2: Дописать абзац комментария карты**

Перед картой стоят абзацы про каждый заведённый тикер, на английском. Дописать абзац про HEAD в том
же стиле: заявка была на мёртвый HHRU и почему калибруется HEAD; адаптированная схема 24/12/3 и
несравнимость чисел с 36-месячным каталогом; решение о максимально широких сетках; вердикт по
планке; девятый baseline каталога (1.311); каноническая процедура тем; снятые риски (высокая
ликвидность, дешёвая доля шага цены); остаточные риски — **гэпы до −9.99% при доле ночёвок 40%,
риск уже реализованный**, капкан широкого стопа с насыщением на 1.3, режим без единого растущего
полугодия.

- [ ] **Step 3: Запустить тесты пакета**

Run: `go test ./internal/service/trading_strategy/rsi_pullback/live/... -v`
Expected: PASS. Два теста прямо сторожат эту пару задач: `TestEveryDefaultTickerIsRegistered`
(каждый тикер боевой вселенной обязан быть в карте) и
`TestBaselineTrackingTickersStayOutOfTheDefaultUniverse` (тикер, всё ещё отслеживающий baseline, в
боевую вселенную попасть не может). Второй — причина, по которой Task 12 идёт раньше Task 14.

- [ ] **Step 4: Коммит**

```bash
git add internal/service/trading_strategy/rsi_pullback/live
git commit -m "feat(rsi_pullback): HEAD в реестре живого раннера"
```

---

### Task 14: Боевая вселенная

**Files:**
- Modify: `internal/config/rsi_pullback.go` (комментарий-абзац + список `Tickers`, строка 285)
- Modify: `internal/config/rsi_pullback_test.go` (список `want`, строка 54)
- Modify: `env/prod.env`, `env/prod.env.example`, `env/local.env.example`
- Modify: `docs/rsi_pullback/live.md` (§8, значение дефолта `RSI_PULLBACK_TICKERS` в таблице
  конфигурации — это механика, а не пер-тикерная запись)

**Interfaces:**
- Consumes: литерал из Task 12, реестр из Task 13.
- Produces: `RSI_PULLBACK_TICKERS` из двадцати одного тикера.

- [ ] **Step 1: Проверить стоп-условие ещё раз**

Перечитать числа Task 11 Steps 4–5. Все три пункта стоп-условия не сработали — иначе эта задача не
выполняется вовсе.

- [ ] **Step 2: Обновить тест дефолта**

В `internal/config/rsi_pullback_test.go` в список `want` (строка 54) добавить `"HEAD"` двадцать
первым, после `"SNGSP"`.

- [ ] **Step 3: Запустить тест и убедиться, что падает**

Run: `go test ./internal/config/ -v`
Expected: FAIL — дефолт ещё из двадцати тикеров.

- [ ] **Step 4: Обновить дефолт и env-файлы**

В `internal/config/rsi_pullback.go` дописать `"HEAD"` в конец списка `Tickers` (строка 285) и
добавить перед ним комментарий-абзац в стиле соседних (SNGSP, ASTR, BANEP): когда заведён, вердикт
по планке, числа точки, **тип риска** (у HEAD он гэповый и режимный: гэпы до −9.99% при доле
ночёвок 40% и режим без единого растущего полугодия), каноническая процедура тем, адаптированная
схема окна 24/12/3 и решение о широких сетках.

Во всех трёх env-файлах дописать `,HEAD` в конец `RSI_PULLBACK_TICKERS`. Текущее значение:
`UGLD,T,GAZP,DOMRF,FESH,WUSH,LENT,RENI,NVTK,LSNGP,IVAT,SVAV,SIBN,ELFV,DIAS,BSPB,YDEX,BANEP,ASTR,SNGSP`.

- [ ] **Step 5: Обновить значение дефолта в `live.md` §8**

В таблице конфигурации строка `RSI_PULLBACK_TICKERS` содержит дефолтный список. Дописать `,HEAD`.
Ничего пер-тикерного (вердиктов, PF, дат) в `docs/rsi_pullback/` не добавлять — правило CLAUDE.md.

- [ ] **Step 6: Запустить тесты**

Run: `go test ./internal/config/ ./internal/service/trading_strategy/rsi_pullback/... -v`
Expected: PASS.

- [ ] **Step 7: Коммит**

```bash
git add internal/config env docs/rsi_pullback/live.md
git commit -m "feat(rsi_pullback): завести HEAD в боевую вселенную"
```

---

### Task 15: Дока пакета — разбор калибровки и принятый риск

**Files:**
- Modify: `internal/service/trading_strategy/rsi_pullback/strategy/head/head.go` (док-комментарий
  пакета)
- Modify: `docs/rsi_pullback/strategy.md` — **только если** HEAD дал механический вывод (§8.0.1)

**Interfaces:**
- Consumes: рабочую записку Task 11.
- Produces: пер-тикерную запись, которую читает следующий калибратор.

- [ ] **Step 1: Переписать док-комментарий пакета**

Заменить блок «СОСТОЯНИЕ: КАЛИБРОВКА НЕ ПРОВОДИЛАСЬ» на разбор в форме, принятой каталогом
(образец — `strategy/sngsp/sngsp.go`). Обязательные разделы:

- **СОСТОЯНИЕ: ОТКАЛИБРОВАН 2026-09-02** — как собран литерал, чем прибит;
- **ВЕРДИКТ ПО ПЛАНКЕ** — по каждой из двух ключевых тем раздельно оба критерия (PF и
  устойчивость), прямым текстом, без сглаживания;
- **ПРИНЯТАЯ ТОЧКА** — pooled OOS PF, сделки, win rate, пофолдовые числа, фолды-выбросы;
- **ПОЛУГОДИЯ** — таблица из Task 11 Step 7 и вердикт по критерию «ни одно полугодие не даёт
  больше половины net», с оговоркой, что полугодий здесь четыре, а не шесть;
- **ПЛАТО** — вердикт по каждой проверенной оси раздельно (плато / склон / пик / инертность);
- **КАПКАН ШИРОКОГО СТОПА** — доля SL-выходов точки против 20.0% у baseline;
- **УДЕРЖАНИЕ И ЭКСПОЗИЦИЯ ПОД ГЭПОМ** — числа Task 11 Step 8 против опорных baseline (ночёвка
  40.0%, одна сделка через гэп 2025-09-26 с −6.28%);
- **ПРИНЯТЫЙ РИСК И УСЛОВИЕ ПЕРЕСМОТРА** — перечислить принятые риски (как минимум: гэпы до −9.99%
  при высокой доле ночёвок; капкан широкого стопа, если стоп широкий; непройденная планка, если
  она не пройдена; убыточное третье полугодие, если оно подтвердилось на точке; тонкий запас по
  издержкам; несравнимость 24-месячных чисел с 36-месячным каталогом). Условие пересмотра
  литерала: первая живая просадка глубже бэктестовой **либо** два подряд убыточных квартала живой
  торговли **либо** смена режима бумаги на растущий (все четыре полугодия окна падающие, и рельеф
  осей снят внутри падающего режима);
- **ОГОВОРКА О ПОДГЛЯДЫВАНИИ** — точку собрал исполнитель, видевший всю историю.

Оставить в доке уже написанные блоки про заявку на HHRU, окно, схему, дыру остановки торгов,
широкие сетки, априор и свойства инструмента — они не устаревают.

- [ ] **Step 2: Отдельно записать гэповый риск для живой торговли**

В блоке принятых рисков назвать три даты известных гэпов (2024-12-17 −9.99%, 2026-05-12 −8.16%,
2025-09-26 −6.14%), их величину в дневных ATR и вывод: HEAD — первый тикер каталога, где гэповый
риск **уже реализовался в бэктесте**, а не остался спящим (прецедент противоположного знака —
SNGSP, где экспозиция нулевая при гэпе −14%; прецедент того же знака — BANEP, где отсечки стоили
−12.74% / −10.67% / −5.28% за ночь). Записать долю ночёвок принятой точки рядом с baseline-овыми
40.0%.

- [ ] **Step 3: Механический вывод в `docs/rsi_pullback/strategy.md` — только если он есть**

В `docs/` идёт **только механика**, применимая к следующему тикеру, и только если HEAD такую дал.
Кандидаты, известные заранее: (а) проверка живости тикера до раскладки сеток — ряд, обрывающийся в
середине запрошенного окна, означает редомициляцию или делистинг, и работать надо с тикером-
преемником; (б) выбор окна, когда история короче 36 месяцев и внутри неё есть остановка торгов:
левая граница ставится так, чтобы разрыв остался слева от окна; (в) правило выбора между двумя
расширениями оси тренда, когда расширение вниз по `EMASlow` и сохранение верхнего края `EMAFast`
несовместимы из-за инварианта `EMAFast < EMASlow`; (г) капкан широкого стопа, который **насыщается**
(три верхние колонки оси совпадают побайтово), — как отличать его от ненасыщающегося и почему верх
оси всё равно оставляют как контроль.

Пер-тикерных записей (строк с вердиктом, PF, датами, реестров риска) в `docs/rsi_pullback/` не
делать ни в каком виде.

- [ ] **Step 4: Коммит**

```bash
git add internal/service/trading_strategy/rsi_pullback/strategy/head docs/rsi_pullback
git commit -m "docs(rsi_pullback): разбор калибровки HEAD и принятый риск"
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
go run ./cmd/pullparity -tickers HEAD -months 24
```
24 — и честный горизонт инструмента (`maxDailyHorizonMonths` в `cmd/pullparity/main.go`, за его
пределами ранние бары расходятся по построению из-за 730-дневного окна дневных свечей живого
сборщика), и ровно расчётное окно калибровки. Expected: ноль расхождений.

Если расхождения есть — **не чинить их правкой литерала**: это дефект сборки, и он разбирается
отдельно, с числами, на которых разошлись. Отдельно проверить, не приходятся ли расхождения на
границу возобновления торгов 2024-09-26 — там у живого сборщика и у кэша разная глубина прогрева.

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
git add data/params/rsi_pullback/head/plateau_point.json
git commit -m "feat(rsi_pullback): HEAD — сверка живой сборки и зелёный CI"
```

- [ ] **Step 5: Доложить владельцу итог**

Одним сообщением: вердикт по планке (по каждой ключевой теме раздельно, оба критерия), числа
принятой точки (pooled OOS PF, сделки, поведение под удвоенными издержками), доля ночёвок и
экспозиция под гэпом, принятые риски, состояние ветки. Мерж и dry-run — решение владельца, план
его не делает.
