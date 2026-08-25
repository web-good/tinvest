# YDEX под `rsi_pullback` — план подготовки, калибровки и вердикта

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Довести YDEX (МКПАО «Яндекс») до вердикта по стратегии `rsi_pullback`: каталог сеток с
осями по замерам инструмента, тематический walk-forward, принятая точка, литерал в пакете и
заведение в боевую вселенную семнадцатым тикером.

**Architecture:** Процедура каталога **каноническая, без отступлений**: все десять тем идут поверх
дефолтов ядра, поэтому каталог сеток создаётся целиком одной задачей, якоря нет, поздних файлов
нет. Отступление одно и оно в данных, а не в процедуре: расчётное окно — 25 месяцев после старта
YDEX, схема прогонов адаптированная 25/9/4 (прецедент IVAT).

**Tech Stack:** Go 1.25, `cmd/backtest` (rolling walk-forward), `cmd/pullparity` (сверка живой
сборки с бэктестом), `./bin/mage ci` (lint + `go test -race ./...` + дрейф моков).

**Spec:** `docs/superpowers/specs/2026-08-25-ydex-rsi-pullback-prep-design.md`

## Global Constraints

- **Схема прогонов — адаптированная:** `-months 25 -train-months 9 -test-months 4 -min-trades 20
  -metric profit_factor`, четыре фолда встык (9 + 4×4 = 25). У темы `screen` — `-min-trades 1`.
  Пропуск любого из трёх флагов окна даёт другую схему и несравнимые числа.
- **`-refresh` НЕ запускать ни на одном шаге.** Кэш перезалит 2026-08-25 до первого замера:
  `YDEX_Minutes30.json` — 27 991 бар (2024-07-25 … 2026-08-25), из них 18 864 будних;
  `YDEX_Day1.json` — 872 свечи, в окне 643 (530 будних). Запас истории — **ноль баров**: первый бар
  кэша совпадает с левой границей расчётного окна. Любой refresh сдвинет обе границы.
- **Окно исключает остановку торгов.** Торги YNDX остановлены 2024-06-14, торги YDEX начались
  2024-07-24 — в получасовой серии это дыра 40 дней с прыжком цены 4007 → 4542. Окно начинается
  2024-07-25, то есть вся YNDX-эпоха и сам разрыв вне расчёта.
- **Планка** (объявлена до прогонов, не пересматривается): темы `entry` и `trend` обе дают pooled
  OOS PF ≥ 1.5 при ≥ 20 сделках в пуле; ведущая ось (`RSILower` для `entry`, `EMASlow` для `trend`)
  выбрана одинаково в ≥ 3 фолдах из 4; вырожденный фолд (ни одной убыточной сделки) в пользу
  тикера не засчитывается; счёт по дефолтной комиссии (круг 0.1%).
- **Правило прода:** литерал ставится и YDEX заводится в `RSI_PULLBACK_TICKERS` семнадцатым
  **независимо от того, взята планка или нет**.
- **Стоп-условие:** работа останавливается, если принятая точка даёт pooled OOS PF < 1.0, **либо**
  меньше 20 сделок за расчётное окно, **либо** PF < 1.0 под удвоенными издержками
  (`-commission 0.001`). При срабатывании — числа владельцу, задачи 12–15 не выполняются.
- **Каждый `_comment` сетки** обязан содержать: что тема меряет и сколько в ней прогонов; замер, из
  которого получена каждая ось, с обоснованием края; полную команду запуска с путём
  `data/params/rsi_pullback/ydex/<файл>` (этого требует `TestRSIPullbackCalFilesValid`); место под
  строку `РЕЗУЛЬТАТ ПРОГОНА 2026-08-25: …`.
- **Коммит по завершении каждой задачи**, сообщения на русском в стиле существующей истории.
- **Дефолты ядра** (`core.DefaultParams()`), поверх которых считают ВСЕ темы: `RSIPeriod 4`,
  `RSILower 30`, `RSIUpper 70`, `EMAFast 10`, `EMASlow 100`, `DailyATRPeriod 14`,
  `UseDayATRGate 1`, `FreshDayATR 0`, `SpentDayATR 0.8`, `StopDailyATR 0.5`, `TPDailyATR 0.6`,
  `UseVolume 0`, `VolBaseDays 14`, `VolLookbackBars 3`, `VolMult 1.2`, `UseRSIExit 1`,
  `UseTrail 0`, `TrailDailyATR 0`. Контрольный прогон дефолтов на расчётном окне: **108 сделок,
  PF 1.778, net +28 302 ₽** — сильнейший baseline каталога.
- **Ветка:** `feat/ydex-pullback-prep` (создана от `main` `a55ee76`, спека закоммичена `03839b9`).

---

### Task 1: Каталог десяти сеток со сторожевым тестом осей

**Files:**
- Create: `data/params/rsi_pullback/ydex/cal_screen.json`
- Create: `data/params/rsi_pullback/ydex/cal_entry.json`
- Create: `data/params/rsi_pullback/ydex/cal_trend.json`
- Create: `data/params/rsi_pullback/ydex/cal_day.json`
- Create: `data/params/rsi_pullback/ydex/cal_day_spent.json`
- Create: `data/params/rsi_pullback/ydex/cal_volume.json`
- Create: `data/params/rsi_pullback/ydex/cal_vol_window.json`
- Create: `data/params/rsi_pullback/ydex/cal_risk.json`
- Create: `data/params/rsi_pullback/ydex/cal_exit.json`
- Create: `data/params/rsi_pullback/ydex/cal_trail.json`
- Create: `internal/service/backtest/rsi_pullback_ydex_grid_test.go`

**Interfaces:**
- Consumes: хелперы `rsiPullbackTickerGrid(t, ticker, file)`, `sameSet`, `containsValue` из
  `internal/service/backtest/rsi_pullback_grid_test.go`; общие тесты `TestRSIPullbackCalFilesValid`
  и `TestRSIPullbackGridControlPoints` того же файла подхватывают новый каталог сами.
- Produces: каталог `data/params/rsi_pullback/ydex/` и функцию `ydexGrid(t, file)` — ею пользуются
  только тесты этого файла.

- [ ] **Step 1: Написать падающий сторожевой тест осей**

Создать `internal/service/backtest/rsi_pullback_ydex_grid_test.go`:

```go
package backtest

import "testing"

// ydexGrid читает файл сеток YDEX через общий хелпер.
func ydexGrid(t *testing.T, file string) map[string][]float64 {
	t.Helper()
	return rsiPullbackTickerGrid(t, "ydex", file)
}

// TestYDEXGridsPinTheirMeasuredAxes сторожит оси всех десяти тем каталога ydex/. Каталог собран
// 2026-08-25 по замерам самого YDEX на расчётном окне 2024-07-25 … 2026-08-25 (27 991 получасовой
// бар, из них 18 864 будних; дневная серия 872 свечи, в окне 530 будних).
//
// Окно короче каталожного намеренно: торги YNDX остановлены 2024-06-14, торги YDEX начались
// 2024-07-24, и в получасовой серии это дыра 40 дней с прыжком цены 4007 -> 4542. Вся YNDX-эпоха
// и сам разрыв выброшены, схема прогонов адаптирована до 25/9/4 (прецедент IVAT).
//
// Шесть решений отличают этот каталог от образца bspb/, и каждое опирается на точечный замер
// сделками поверх дефолтов ядра (baseline: 108 сделок, PF 1.778 — сильнейший в каталоге):
//
//   - RSIPeriod РАСШИРЕН ВНИЗ ДО 3 — второй случай после BSPB. Кроссов у тройки вдвое больше, чем
//     у четвёрки (484 против 230 на уровне 10), то есть правило каталога формально применимо; но
//     RSI(3)@25 даёт 140 сделок при PF 2.050 против 87 при 1.762 у RSI(4)@25, и вся строка тройки
//     живая (1.53–2.05 на семи уровнях из семи).
//   - RSIPeriod ОБОРВАН НА 6: RSI(7)@10 — 1 сделка, RSI(8)@10 — НОЛЬ, RSI(8)@20 — 9 сделок при
//     PF 0.701. Ось периода на YDEX меряется сделками, а не кроссами.
//   - RSIUpper НЕ РАСШИРЕН до 85, в отличие от BSPB: замер даёт максимум на дефолте 70 (1.778),
//     75 рядом (1.769), 80 уже 1.581, а 85 мёртв (1.204). Разворот стоит ВНУТРИ каталожной оси.
//   - Ось EMA НЕ СДВИНУТА вниз, в отличие от BSPB: максимум оси (10/50 -> 1.802) стоит внутри
//     сетки, а расширение вниз замерено и отвергнуто (10/30 -> 1.393, 10/40 -> 1.434,
//     5/30 -> 1.295). Верх оси жив (20/200 -> 1.542, 40/200 -> 1.466).
//   - SpentDayATR ОБРЕЗАН СВЕРХУ ДО 1.3: при 1.5 остаётся 5.2% баров и 14 сделок за 25 месяцев =
//     5 на девятимесячное обучающее окно, вчетверо ниже -min-trades 20 (прецедент SIBN).
//   - TPDailyATR РАСШИРЕН ВНИЗ ДО 0.4 и ОБРЕЗАН СВЕРХУ ДО 1.5: максимум каждой строки риска стоит
//     на 0.5, а колонки 1.5, 2.0 и 2.5 совпадают ПОБАЙТОВО — цель шире 1.5 дневного ATR на YDEX
//     недостижима, RSI-выход или стоп срабатывают раньше.
func TestYDEXGridsPinTheirMeasuredAxes(t *testing.T) {
	screen := ydexGrid(t, "cal_screen.json")
	for _, field := range []string{"UseDayATRGate", "UseVolume"} {
		if got := screen[field]; !sameSet(got, 0, 1) {
			t.Errorf("cal_screen.json: %s = %v, want ровно {0,1} — тема меряет цену каждого гейта в сделках", field, got)
		}
	}

	entry := ydexGrid(t, "cal_entry.json")
	// Выше 50 отката нет по определению: 50 — средняя линия осциллятора.
	for _, v := range entry["RSILower"] {
		if v > 50 {
			t.Errorf("cal_entry.json свипует RSILower=%v: выше 50 отката нет по определению", v)
		}
	}
	// Уровень 10 обязан остаться: RSI(3)@10 даёт 53 сделки при PF 1.811 — второй результат строки.
	if !containsValue(entry["RSILower"], 10) {
		t.Errorf("cal_entry.json: RSILower = %v, не содержит 10 — на YDEX это 53 сделки при PF 1.811", entry["RSILower"])
	}
	// Тройка — сознательное отступление от правила каталога, держится замером сделками.
	if !containsValue(entry["RSIPeriod"], 3) {
		t.Errorf("cal_entry.json: RSIPeriod = %v, не содержит 3 — на YDEX тройка бьёт четвёрку и по сделкам (140 против 87), и по PF (2.050 против 1.762)", entry["RSIPeriod"])
	}
	for _, v := range entry["RSIPeriod"] {
		if v > 6 {
			t.Errorf("cal_entry.json свипует RSIPeriod=%v: на YDEX медленный RSI не оставляет сделок — RSI(7)@10 даёт 1, RSI(8)@10 ноль", v)
		}
		if v < 3 {
			t.Errorf("cal_entry.json свипует RSIPeriod=%v: короче тройки — уже не откат, а тик", v)
		}
	}
	// Полоса выхода НЕ расширяется до 85: там PF 1.204 против 1.778 на дефолте 70.
	for _, v := range entry["RSIUpper"] {
		if v > 80 {
			t.Errorf("cal_entry.json свипует RSIUpper=%v: на YDEX полоса выше 80 мертва (85 даёт PF 1.204 против 1.778 на 70)", v)
		}
	}

	trend := ydexGrid(t, "cal_trend.json")
	// Ось НЕ расширяется вниз: пары со EMASlow < 50 замерены хуже любой пары внутри оси.
	for _, v := range trend["EMASlow"] {
		if v < 50 {
			t.Errorf("cal_trend.json свипует EMASlow=%v: на YDEX низ оси хуже (10/30 -> 1.393, 10/40 -> 1.434) любой пары со EMASlow >= 50", v)
		}
	}
	// 50 обязан остаться: максимум оси — пара 10/50 (PF 1.802).
	if !containsValue(trend["EMASlow"], 50) {
		t.Errorf("cal_trend.json: EMASlow = %v, не содержит 50 — максимум замера (пара 10/50, PF 1.802) стоял бы вне сетки", trend["EMASlow"])
	}
	// Верх оси жив (20/200 -> 1.542, 30/200 -> 1.518), обрезать его нечем.
	if !containsValue(trend["EMASlow"], 200) {
		t.Errorf("cal_trend.json: EMASlow = %v, не содержит 200 — верх оси на YDEX жив (20/200 даёт PF 1.542)", trend["EMASlow"])
	}
	// Пара с EMAFast >= EMASlow либо вырождена (допуск 0.0%), либо инвертирована («медленная над
	// быстрой» — другой фильтр). При этой оси таких пар нет по построению; тест страхует правки.
	for _, f := range trend["EMAFast"] {
		for _, s := range trend["EMASlow"] {
			if f >= s {
				t.Errorf("cal_trend.json порождает пару EMAFast=%v >= EMASlow=%v: фильтр вырожден или инвертирован", f, s)
			}
		}
	}

	day := ydexGrid(t, "cal_day.json")
	// Ноль в ветке «свежий день» обязателен: на всех прод-тикерах каталога победил именно он, а на
	// YDEX свежая ветка замерена разбавляющей (0.3 -> PF 1.203 против 1.778 без неё).
	if !containsValue(day["FreshDayATR"], 0) {
		t.Errorf("cal_day.json: FreshDayATR = %v, не содержит 0 — на всех прод-тикерах каталога победил ноль", day["FreshDayATR"])
	}
	for _, name := range []string{"cal_day.json", "cal_day_spent.json"} {
		spent := ydexGrid(t, name)["SpentDayATR"]
		for _, v := range spent {
			if v > 1.3 {
				t.Errorf("%s свипует SpentDayATR=%v: при 1.5 остаётся 5.2%% баров и 14 сделок за 25 месяцев = 5 на обучающее окно при пороге 20", name, v)
			}
		}
		if !containsValue(spent, 1.3) {
			t.Errorf("%s: SpentDayATR = %v, не содержит 1.3 — верхний живой край оси (9.4%% баров, 25 сделок, PF 4.420)", name, spent)
		}
	}
	// Ось «дня исчерпанного» уплотнена живыми уровнями 0.7 (43.5% баров) и 1.1 (15.3%): между ними
	// PF растёт с 1.588 до 2.909.
	spentOnly := ydexGrid(t, "cal_day_spent.json")["SpentDayATR"]
	for _, v := range []float64{0.7, 1.1} {
		if !containsValue(spentOnly, v) {
			t.Errorf("cal_day_spent.json: SpentDayATR = %v, не содержит %v — уровень живой и разворот кривой лежит рядом", spentOnly, v)
		}
	}

	volume := ydexGrid(t, "cal_volume.json")
	for _, v := range volume["VolMult"] {
		if v > 2.5 {
			t.Errorf("cal_volume.json свипует VolMult=%v: при 3.0 остаётся 27 сделок (10 на обучающее окно) при PF 0.981", v)
		}
	}

	window := ydexGrid(t, "cal_vol_window.json")
	// Максимум оси стоит на 1 (41 сделка, PF 2.191), дефолт ядра 3 хуже на 0.204 PF.
	if !containsValue(window["VolLookbackBars"], 1) {
		t.Errorf("cal_vol_window.json: VolLookbackBars = %v, не содержит 1 — максимум замера стоял бы вне сетки", window["VolLookbackBars"])
	}
	for _, v := range window["VolLookbackBars"] {
		if v > 12 {
			t.Errorf("cal_vol_window.json свипует VolLookbackBars=%v: кривая там уже плоская и ниже дефолта (16 -> PF 1.682)", v)
		}
	}

	risk := ydexGrid(t, "cal_risk.json")
	// Стоп 0.3 ATR это 0.85% цены; реальный круг издержек 0.125% съедает 14.6% риска — под чертой
	// 17%, по которой строку вырезали из domrf/ и elfv/.
	if !containsValue(risk["StopDailyATR"], 0.3) {
		t.Errorf("cal_risk.json: StopDailyATR = %v, не содержит 0.3 — на YDEX круг издержек съедает там 14.6%% риска, под чертой 17%%", risk["StopDailyATR"])
	}
	for _, v := range risk["StopDailyATR"] {
		if v > 1.3 {
			t.Errorf("cal_risk.json свипует StopDailyATR=%v: при 1.5 стоп достаёт лишь 11.7%% дней и вытесняет убыток в RSI-выход", v)
		}
	}
	if !containsValue(risk["TPDailyATR"], 0.4) {
		t.Errorf("cal_risk.json: TPDailyATR = %v, не содержит 0.4 — максимум каждой строки стоит на 0.5, край оси обязан её накрыть", risk["TPDailyATR"])
	}
	for _, v := range risk["TPDailyATR"] {
		if v > 1.5 {
			t.Errorf("cal_risk.json свипует TPDailyATR=%v: колонки 1.5, 2.0 и 2.5 совпадают побайтово — цель шире 1.5 ATR на YDEX недостижима", v)
		}
	}

	exit := ydexGrid(t, "cal_exit.json")
	for _, v := range exit["RSIUpper"] {
		if v > 80 {
			t.Errorf("cal_exit.json свипует RSIUpper=%v: разворот полосы стоит внутри оси (70 -> 1.778, 75 -> 1.769, 80 -> 1.581, 85 -> 1.204)", v)
		}
	}
}
```

- [ ] **Step 2: Запустить тест и убедиться, что он падает**

Run: `go test ./internal/service/backtest/ -run TestYDEXGrids -v`
Expected: FAIL — каталога `data/params/rsi_pullback/ydex/` не существует.

- [ ] **Step 3: Создать десять файлов сеток**

Все файлы — в `data/params/rsi_pullback/ydex/`. Формат: `_comment` плюс `phases` с одной фазой.
`_comment` каждого файла обязан содержать собственный путь (этого требует
`TestRSIPullbackCalFilesValid`) и заканчиваться строкой-заготовкой
`РЕЗУЛЬТАТ ПРОГОНА 2026-08-25:`.

`cal_screen.json`:

```json
{
  "_comment": "data/params/rsi_pullback/ydex/cal_screen.json — тема screen для YDEX (МКПАО «Яндекс»), 4 прогона (UseDayATRGate x UseVolume). Тема меряет ЦЕНУ каждого из двух опциональных гейтов в сделках и в PF относительно дефолтов ядра. Точечные замеры на полной истории расчётного окна: день включён + объём выключен (дефолт) — 108 сделок, PF 1.778; день ВЫКЛЮЧЕН — 396 сделок, PF 0.960, то есть дневной гейт вырезает три четверти сигналов и разворачивает знак edge; объём включён при дефолтном VolMult 1.2 — 82 сделки, PF 1.656. Ось канонична и не расширяется: у обоих полей ровно два состояния. Команда: go run ./cmd/backtest -ticker YDEX -strategy rsi_pullback -interval Minutes30 -calibrate data/params/rsi_pullback/ydex/cal_screen.json -out ./reports/YDEX_screen -months 25 -train-months 9 -test-months 4 -min-trades 1 -metric profit_factor. РЕЗУЛЬТАТ ПРОГОНА 2026-08-25:",
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

`cal_entry.json`:

```json
{
  "_comment": "data/params/rsi_pullback/ydex/cal_entry.json — ПЕРВАЯ КЛЮЧЕВАЯ тема entry для YDEX, 168 прогонов (RSIUpper 6 x RSIPeriod 4 x RSILower 7). Тема меряет полосу RSI целиком поверх дефолтов ядра, на ней стоит планка. ОСЬ ПЕРИОДА РАСШИРЕНА ВНИЗ ДО 3 — второй случай в каталоге после BSPB. Кроссов вниз на будних барах окна: RSI(3) — 484/833/1223/1603/1900/2196/2425 по уровням 10..40, RSI(4) — 230/468/768/1108/1444/1750/2033, то есть формально применимо правило каталога «тройка — дыхание цены». Правило отменяет проверка сделками: RSI(3)@25 даёт 140 сделок при PF 2.050 против 87 при 1.762 у RSI(4)@25, и вся строка тройки живая (1.532 при 15, 1.652 при 20, 1.601 при 30, 1.773 при 35, 1.596 при 40). ОСЬ ОБОРВАНА СВЕРХУ НА 6: RSI(7)@10 — 1 сделка, RSI(8)@10 — ноль, RSI(8)@20 — 9 сделок при PF 0.701; дневной гейт и тренд-фильтр вырезают почти всё, что оставляет медленный RSI. RSILower не расширяется выше 40 (выше 50 отката нет по определению, а каталог дважды получил вред от растяжки до 50 — WUSH 2.000 -> 1.674) и не обрезается снизу (RSI(3)@10 — 53 сделки при 1.811). RSIUpper НЕ расширен до 85, в отличие от BSPB: замер даёт 70 -> 1.778, 75 -> 1.769, 80 -> 1.581, 85 -> 1.204, разворот стоит внутри оси. Команда: go run ./cmd/backtest -ticker YDEX -strategy rsi_pullback -interval Minutes30 -calibrate data/params/rsi_pullback/ydex/cal_entry.json -out ./reports/YDEX_entry -months 25 -train-months 9 -test-months 4 -min-trades 20 -metric profit_factor. РЕЗУЛЬТАТ ПРОГОНА 2026-08-25:",
  "phases": [
    {
      "name": "entry",
      "grid": {
        "RSIUpper": [55, 60, 65, 70, 75, 80],
        "RSIPeriod": [3, 4, 5, 6],
        "RSILower": [10, 15, 20, 25, 30, 35, 40]
      }
    }
  ]
}
```

`cal_trend.json`:

```json
{
  "_comment": "data/params/rsi_pullback/ydex/cal_trend.json — ВТОРАЯ КЛЮЧЕВАЯ тема trend для YDEX, 35 прогонов (EMAFast 5 x EMASlow 7). Тема меряет трендовый фильтр поверх дефолтов ядра, на ней стоит планка. ОСЬ КАНОНИЧНА И НЕ СДВИГАЕТСЯ — это решение замера, а не инерции. Максимум оси стоит ВНУТРИ сетки: 10/50 -> PF 1.802 при 99 сделках, дефолт 10/100 -> 1.778 при 108, 20/50 -> 1.731, 10/70 -> 1.715. Расширение вниз замерено и ОТВЕРГНУТО: 10/30 -> 1.393, 10/40 -> 1.434, 5/30 -> 1.295 — хуже любой пары со EMASlow >= 50. Верх оси жив и не обрезается: 20/200 -> 1.542, 30/200 -> 1.518, 40/200 -> 1.466, 10/200 -> 1.700. Трендовый допуск ровный на всей оси (доля будних баров с EMAFast > EMASlow — 47.9–51.8%), поэтому выбор пары не меняет объём выборки: число сделок держится в полосе 87–117. Пары с EMAFast >= EMASlow эта ось не порождает: её минимум (50) выше максимума EMAFast (40). Команда: go run ./cmd/backtest -ticker YDEX -strategy rsi_pullback -interval Minutes30 -calibrate data/params/rsi_pullback/ydex/cal_trend.json -out ./reports/YDEX_trend -months 25 -train-months 9 -test-months 4 -min-trades 20 -metric profit_factor. РЕЗУЛЬТАТ ПРОГОНА 2026-08-25:",
  "phases": [
    {
      "name": "trend",
      "grid": {
        "EMAFast": [5, 10, 20, 30, 40],
        "EMASlow": [50, 70, 100, 120, 150, 170, 200]
      }
    }
  ]
}
```

`cal_day.json`:

```json
{
  "_comment": "data/params/rsi_pullback/ydex/cal_day.json — тема day для YDEX, 24 прогона (FreshDayATR 4 x SpentDayATR 6). Тема меряет ОБЕ ветки двустороннего дневного гейта поверх дефолтов ядра. Замер долей будних баров (n = 18 367 с известным ATR предыдущего дня): свежая ветка отбирает 6.6% при 0.2, 14.9% при 0.3, 24.3% при 0.4, 35.3% при 0.5; исчерпанная — 52.0% при 0.6, 33.8% при 0.8, 26.1% при 0.9, 20.4% при 1.0, 10.2% при 1.25, 9.4% при 1.3. День проходит своё дневное ATR быстро: медиана размаха дня к текущему бару 0.62 ATR против 0.34 у BSPB. Точечные прогоны показывают, что свежая ветка РАЗБАВЛЯЕТ: FreshDayATR 0.2 -> 132 сделки/1.575, 0.3 -> 185/1.203, 0.4 -> 249/1.005, 0.5 -> 299/0.983 против 108/1.778 при нуле. Ноль в оси обязателен — на всех прод-тикерах каталога победил именно он. ВЕРХ ОСИ SpentDayATR ОБРЕЗАН ДО 1.3: при 1.5 остаётся 5.2% баров и 14 сделок за 25 месяцев = 5 на девятимесячное обучающее окно при пороге -min-trades 20 (прецедент обрезки — SIBN). Команда: go run ./cmd/backtest -ticker YDEX -strategy rsi_pullback -interval Minutes30 -calibrate data/params/rsi_pullback/ydex/cal_day.json -out ./reports/YDEX_day -months 25 -train-months 9 -test-months 4 -min-trades 20 -metric profit_factor. РЕЗУЛЬТАТ ПРОГОНА 2026-08-25:",
  "phases": [
    {
      "name": "day",
      "grid": {
        "UseDayATRGate": [1],
        "FreshDayATR": [0, 0.3, 0.4, 0.5],
        "SpentDayATR": [0.6, 0.8, 0.9, 1.0, 1.25, 1.3]
      }
    }
  ]
}
```

`cal_day_spent.json`:

```json
{
  "_comment": "data/params/rsi_pullback/ydex/cal_day_spent.json — тема day_spent для YDEX, 8 прогонов (SpentDayATR при выключенной свежей ветке). Тема меряет ТОЛЬКО ветку «день исчерпан» на уплотнённой оси поверх дефолтов ядра — приём SIBN, ELFV, DIAS и BSPB. Точечные прогоны: 0.6 -> 177 сделок/1.448, 0.7 -> 145/1.588, 0.8 (дефолт) -> 108/1.778, 0.9 -> 82/2.512, 1.0 -> 58/3.256, 1.1 -> 43/2.909, 1.25 -> 28/2.832, 1.3 -> 25/4.420. PF растёт с ужесточением гейта, выборка тает быстрее: YDEX даёт edge на исчерпанных днях. Уровни 0.7 (43.5% баров) и 1.1 (15.3%) добавлены, чтобы кривая между 0.7 и 1.1 была видна целиком. ВЕРХ 1.5 УБРАН: там 5.2% баров и 14 сделок = 5 на обучающее окно, и PF 11.762 — процедурный артефакт, а не edge. Значения выше 1.0 читать с оглядкой на колонку числа сделок: 1.3 это 9 сделок на обучающее окно при пороге 20. Команда: go run ./cmd/backtest -ticker YDEX -strategy rsi_pullback -interval Minutes30 -calibrate data/params/rsi_pullback/ydex/cal_day_spent.json -out ./reports/YDEX_day_spent -months 25 -train-months 9 -test-months 4 -min-trades 20 -metric profit_factor. РЕЗУЛЬТАТ ПРОГОНА 2026-08-25:",
  "phases": [
    {
      "name": "day_spent",
      "grid": {
        "UseDayATRGate": [1],
        "FreshDayATR": [0],
        "SpentDayATR": [0.6, 0.7, 0.8, 0.9, 1.0, 1.1, 1.25, 1.3]
      }
    }
  ]
}
```

`cal_volume.json`:

```json
{
  "_comment": "data/params/rsi_pullback/ydex/cal_volume.json — тема volume для YDEX, 20 прогонов (VolMult 5 x VolBaseDays 4). Тема меряет объёмный гейт поверх дефолтов ядра при дефолтном окне VolLookbackBars 3. Точечные прогоны: гейт выключен — 108 сделок/1.778; VolMult 1.0 -> 91/1.581, 1.2 -> 82/1.656, 1.5 -> 68/1.508, 2.0 -> 53/1.987, 2.5 -> 39/1.587, 3.0 -> 27/0.981, 4.0 -> 17/1.312. ОСЬ СУЖЕНА ДО 2.5: при 3.0 остаётся 27 сделок = 10 на девятимесячное обучающее окно (вдвое ниже порога -min-trades 20) и PF уходит под единицу; то же решение, что на DIAS и BSPB. Край 2.5 сохранён как последний, где видно, где ломается кривая (39 сделок = 14 на обучающее окно). База при VolMult 1.5: 3 дня -> 68 сделок/2.012, 5 -> 68/1.848, 10 -> 71/2.256, 14 -> 68/1.508 — разброс 0.75 PF при почти неизменном числе сделок, то есть шум базы, а не рельеф. Команда: go run ./cmd/backtest -ticker YDEX -strategy rsi_pullback -interval Minutes30 -calibrate data/params/rsi_pullback/ydex/cal_volume.json -out ./reports/YDEX_volume -months 25 -train-months 9 -test-months 4 -min-trades 20 -metric profit_factor. РЕЗУЛЬТАТ ПРОГОНА 2026-08-25:",
  "phases": [
    {
      "name": "volume",
      "grid": {
        "UseVolume": [1],
        "VolMult": [1.0, 1.2, 1.5, 2.0, 2.5],
        "VolBaseDays": [3, 5, 10, 14]
      }
    }
  ]
}
```

`cal_vol_window.json`:

```json
{
  "_comment": "data/params/rsi_pullback/ydex/cal_vol_window.json — тема vol_window для YDEX, 18 прогонов (VolLookbackBars 6 x VolMult 3). Четвёртый замер этой оси в каталоге (после ELFV, DIAS и BSPB) и первый на бумаге с 0.2% коротких дней — YDEX самый ликвидный тикер каталога (медиана оборота 2826 млн ₽/день, медиана 35 баров в буднем дне, дней короче 20 баров 1 из 542). Точечные прогоны при VolMult 2.0 и базе 14: окно 1 -> 41 сделка/2.191, 2 -> 48/1.924, 3 (дефолт ядра) -> 53/1.987, 5 -> 67/1.725, 8 -> 74/1.838, 12 -> 89/1.792, 16 -> 95/1.682. Максимум на 1, спуск пологий и монотонный по числу сделок. Нижний край 1 обязателен — там максимум; верхний край 12 берётся, чтобы спуск попал внутрь сетки; 16 не нужен (кривая плоская и ниже дефолта). VolMult сужен до трёх значений вокруг замеренного максимума: тема меряет ОКНО, полную ось множителя меряет cal_volume.json. ЧИТАТЬ ОСТОРОЖНО: VolLookbackBars ОСЛАБЛЯЕТ гейт, поэтому при выборе большого окна нужен точечный прогон, не эквивалентен ли выбор простому UseVolume = 0 — при неразличимых числах предпочитается выключенный гейт как более простая конфигурация. Команда: go run ./cmd/backtest -ticker YDEX -strategy rsi_pullback -interval Minutes30 -calibrate data/params/rsi_pullback/ydex/cal_vol_window.json -out ./reports/YDEX_vol_window -months 25 -train-months 9 -test-months 4 -min-trades 20 -metric profit_factor. РЕЗУЛЬТАТ ПРОГОНА 2026-08-25:",
  "phases": [
    {
      "name": "vol_window",
      "grid": {
        "UseVolume": [1],
        "VolLookbackBars": [1, 2, 3, 5, 8, 12],
        "VolMult": [1.2, 1.5, 2.0]
      }
    }
  ]
}
```

`cal_risk.json`:

```json
{
  "_comment": "data/params/rsi_pullback/ydex/cal_risk.json — тема risk для YDEX, 30 прогонов (StopDailyATR 5 x TPDailyATR 6). Тема меряет стоп и цель поверх дефолтов ядра. Дневной ATR(14) медианой 2.84% цены (p10 2.02, p90 3.98). СТРОКА 0.3 ВОЗВРАЩЕНА В СЕТКУ (у domrf/ и elfv/ она вырезана): стоп 0.3 ATR это 0.85% цены, реальный круг издержек 0.125% (моделируемые 0.1% плюс два тика по 0.0123% при шаге 0.5 ₽ и медианной цене 4081 ₽) съедает 14.6% риска — под чертой 17%, как это было на DIAS (14.1%) и BSPB (14.0%). Выживаемость уровней (доля будних дней, чей размах достаёт уровня): 0.3 -> 99.4%, 0.5 -> 92.3%, 0.7 -> 73.8%, 1.0 -> 37.7%, 1.3 -> 18.1%, 1.5 -> 11.7%. ВЕРХНИЙ КРАЙ 1.3 СОХРАНЁН: при 1.5 стоп достаёт лишь 11.7% дней и перестаёт быть защитой, превращаясь в способ вытеснить убыток в RSI-выход — капкан, разобранный на WUSH, LENT, LSNGP, IVAT, SVAV, SIBN и BSPB. Точечные PF по стопам (цель 0.5): 0.3 -> 1.543, 0.5 -> 1.806, 0.7 -> 2.087, 1.0 -> 2.027, 1.3 -> 2.362 при почти неизменном числе сделок (110 против 108) — это и есть след капкана. ОСЬ ЦЕЛИ РАСШИРЕНА ВНИЗ ДО 0.4: максимум каждой строки стоит на 0.5, а 0.8 и 1.0 теряют 0.2 PF, то есть нижний край каталожной оси оказался краем максимума. ОСЬ ЦЕЛИ ОБРЕЗАНА СВЕРХУ ДО 1.5: колонки 1.5, 2.0 и 2.5 совпадают ПОБАЙТОВО — цель шире 1.5 дневного ATR на YDEX недостижима, RSI-выход или стоп срабатывают раньше. Строка 1.5 оставлена, чтобы это было видно в отчёте и чтобы выполнялась контрольная точка TestRSIPullbackGridControlPoints (цель шире самого широкого стопа 1.3). Команда: go run ./cmd/backtest -ticker YDEX -strategy rsi_pullback -interval Minutes30 -calibrate data/params/rsi_pullback/ydex/cal_risk.json -out ./reports/YDEX_risk -months 25 -train-months 9 -test-months 4 -min-trades 20 -metric profit_factor. РЕЗУЛЬТАТ ПРОГОНА 2026-08-25:",
  "phases": [
    {
      "name": "risk",
      "grid": {
        "StopDailyATR": [0.3, 0.5, 0.7, 1.0, 1.3],
        "TPDailyATR": [0.4, 0.5, 0.6, 0.8, 1.0, 1.5]
      }
    }
  ]
}
```

`cal_exit.json`:

```json
{
  "_comment": "data/params/rsi_pullback/ydex/cal_exit.json — тема exit для YDEX, 6 прогонов (RSIUpper). Тема меряет полосу RSI-выхода поверх дефолтов ядра. Точечные прогоны: 55 -> 117 сделок/1.736, 60 -> 114/1.596, 65 -> 110/1.640, 70 (дефолт) -> 108/1.778, 75 -> 106/1.769, 80 -> 101/1.581, 85 -> 100/1.204. Максимум стоит на дефолте, разворот вниз — между 75 и 80, то есть ВНУТРИ каталожной оси, поэтому ось НЕ расширяется до 85. Это прямая противоположность BSPB, где тот же замер тянул ось вверх (75 -> 1.806, разворот на верхнем крае), и записывается как свойство бумаги, а не как каталожная константа. Выход несёт основную работу: без него (UseRSIExit = 0) остаётся 95 сделок при PF 1.058 против 108 при 1.778. Команда: go run ./cmd/backtest -ticker YDEX -strategy rsi_pullback -interval Minutes30 -calibrate data/params/rsi_pullback/ydex/cal_exit.json -out ./reports/YDEX_exit -months 25 -train-months 9 -test-months 4 -min-trades 20 -metric profit_factor. РЕЗУЛЬТАТ ПРОГОНА 2026-08-25:",
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

`cal_trail.json`:

```json
{
  "_comment": "data/params/rsi_pullback/ydex/cal_trail.json — тема trail для YDEX, 12 прогонов (UseRSIExit 2 x TrailDailyATR 6). Тема меряет ATR-трейл и его сочетание с RSI-выходом поверх дефолтов ядра. Ось каноническая. Точечные прогоны с включённым трейлом: 0.3 -> 116 сделок/1.484, 0.5 -> 111/1.783, 0.7 -> 109/1.678, 1.0 -> 108/1.778, 1.3 -> 108/1.778 — при 1.0 и выше трейл не срабатывает ни разу (числа совпадают с baseline), а узкий трейл режет прибыль. Тема прогоняется целиком, несмотря на отрицательное ожидание: она стоит 12 прогонов, а её отрицательный результат — часть протокола и основание не включать трейл в точку. Команда: go run ./cmd/backtest -ticker YDEX -strategy rsi_pullback -interval Minutes30 -calibrate data/params/rsi_pullback/ydex/cal_trail.json -out ./reports/YDEX_trail -months 25 -train-months 9 -test-months 4 -min-trades 20 -metric profit_factor. РЕЗУЛЬТАТ ПРОГОНА 2026-08-25:",
  "phases": [
    {
      "name": "trail",
      "grid": {
        "UseRSIExit": [0, 1],
        "UseTrail": [1],
        "TrailDailyATR": [0, 0.3, 0.5, 0.7, 1.0, 1.3]
      }
    }
  ]
}
```

- [ ] **Step 4: Запустить сторожевой тест и общие тесты каталога**

Run: `go test ./internal/service/backtest/ -run 'TestYDEXGrids|TestRSIPullbackCalFilesValid|TestRSIPullbackGridControlPoints' -v`
Expected: PASS. Если `TestRSIPullbackCalFilesValid` падает — проверить, что `_comment` каждого файла
содержит собственный путь. Если падает `TestRSIPullbackGridControlPoints` — проверить, что в
`cal_risk.json` максимум `TPDailyATR` (1.5) строго больше максимума `StopDailyATR` (1.3).

- [ ] **Step 5: Коммит**

```bash
git add data/params/rsi_pullback/ydex internal/service/backtest/rsi_pullback_ydex_grid_test.go
git commit -m "feat(rsi_pullback): каталог сеток YDEX с осями по замерам инструмента"
```

---

### Task 2: Пакет `strategy/ydex` в состоянии «калибровка не проводилась»

**Files:**
- Create: `internal/service/trading_strategy/rsi_pullback/strategy/ydex/ydex.go`
- Create: `internal/service/trading_strategy/rsi_pullback/strategy/ydex/ydex_test.go`
- Modify: `internal/service/backtest/rsi_pullback_registry.go`
- Modify: `internal/service/backtest/rsi_pullback_registry_test.go`

**Interfaces:**
- Consumes: `core.Params`, `core.DefaultParams()` из
  `internal/service/trading_strategy/rsi_pullback/strategy/core`.
- Produces: `ydex.Ticker` (строка `"YDEX"`) и `ydex.DefaultParams() core.Params` — их используют
  Task 12 (литерал) и Task 13 (реестр живого раннера).

- [ ] **Step 1: Написать падающий тест baseline-состояния**

Создать `internal/service/trading_strategy/rsi_pullback/strategy/ydex/ydex_test.go`:

```go
package ydex

import (
	"testing"

	"tinvest/internal/service/trading_strategy/rsi_pullback/strategy/core"
)

// TestParamsTrackTheBaselineUntilCalibrated фиксирует ЧЕСТНОЕ состояние: калибровка YDEX ещё не
// проводилась, поэтому пакет обязан возвращать ровно baseline ядра. Тест держит это состояние до
// Task 12, где его заменяет снимок литерала. Пока он стоит, ни одна правка не может тихо
// подсунуть в прод «почти откалиброванные» параметры.
func TestParamsTrackTheBaselineUntilCalibrated(t *testing.T) {
	if got, want := DefaultParams(), core.DefaultParams(); got != want {
		t.Fatalf("YDEX ещё не откалиброван, параметры обязаны совпадать с baseline:\n got: %+v\nwant: %+v", got, want)
	}
}

func TestTickerIsYDEX(t *testing.T) {
	if Ticker != "YDEX" {
		t.Fatalf("Ticker = %q, want YDEX", Ticker)
	}
}
```

- [ ] **Step 2: Запустить тест и убедиться, что он падает**

Run: `go test ./internal/service/trading_strategy/rsi_pullback/strategy/ydex/ -v`
Expected: FAIL — пакета `ydex` не существует.

- [ ] **Step 3: Создать пакет**

Создать `internal/service/trading_strategy/rsi_pullback/strategy/ydex/ydex.go`:

```go
// Package ydex supplies the ticker and rsi_pullback Params for YDEX (МКПАО «Яндекс», обыкновенные
// акции, лот 1).
//
// СОСТОЯНИЕ: КАЛИБРОВКА НЕ ПРОВОДИЛАСЬ. Пакет возвращает core.DefaultParams() — baseline ядра, не
// подобранный под этот инструмент. Так и должно быть до конца калибровки: пакет заведён заранее,
// чтобы прогоны шли через тот же реестр, что и у остальных шестнадцати тикеров, а не через
// generic-ветку. Состояние держит ydex_test.go.
//
// ОКНО ИСКЛЮЧАЕТ ОСТАНОВКУ ТОРГОВ. Торги YNDX (Yandex N.V.) остановлены 2024-06-14, торги YDEX
// (МКПАО «Яндекс») начались 2024-07-24: в получасовой серии это дыра 40 дней с прыжком цены
// 4007 -> 4542. Расчётное окно — 2024-07-25 … 2026-08-25 (25.0 месяца, 27 991 бар, из них 18 864
// будних), то есть вся история нынешней бумаги и ничего до неё. Половина прежнего окна
// принадлежала бумаге с иностранной пропиской и другим режимом ценообразования — она выброшена
// целиком, а не взвешена.
//
// СХЕМА ПРОГОНОВ АДАПТИРОВАННАЯ: -months 25 -train-months 9 -test-months 4 -min-trades 20
// -metric profit_factor, четыре фолда встык (9 + 4x4 = 25). Прецедент — IVAT (25/9/4) и DIAS
// (30/10/5). Числа YDEX сопоставимы по схеме только с IVAT; с 36-месячным каталогом — лишь
// качественно. Пофолдовая устойчивость при этом остаётся измеримой, ради чего схема и выбрана.
//
// ЗАПАС ИСТОРИИ НУЛЕВОЙ: первый бар кэша совпадает с левой границей расчётного окна. -refresh во
// время калибровки НЕ запускать — он сдвинет обе границы.
//
// АПРИОР, записанный ДО прогонов. Скринер (pullback_screen_Minutes30_20260804_232456.md, строка 14
// из 99 прошедших вселенную): оборот 2640 млн ₽, дневной ATR 3.04%, TradesMed 42, PFmed 1.55,
// Capped 0/24, SilentCfg 0/24, плато 62%, PFmed HO 1.76 на 8 сделках. Контрольный прогон дефолтов
// ядра на расчётных 25 месяцах: 108 сделок, PF 1.778, net +28 302 ₽ — САМЫЙ СИЛЬНЫЙ baseline
// каталога (BSPB 1.114, IVAT 1.432). Веса априору не придаётся: вопрос «предсказывают ли колонки
// скринера исход протокола» каталог закрыл четырежды подряд (SIBN, ELFV, DIAS, BSPB) — не
// предсказывают ни снизу, ни сверху.
//
// ПРОЦЕДУРА ТЕМ КАНОНИЧЕСКАЯ. Дефолты ядра стоят ВНУТРИ рабочей зоны, поэтому все десять тем идут
// поверх них, якоря нет, и вердикт сравним с каталогом по построению. Это противоположность BSPB,
// где дефолты стояли почти в мёртвой зоне и темы пришлось делить на ранние и поздние.
//
// ЧТО ЗНАЕМ ОБ ИНСТРУМЕНТЕ ДО ПРОГОНОВ (замеры на расчётном окне):
//
//   - Режим СБАЛАНСИРОВАННЫЙ: два растущих полугодия из пяти (+8.1% и +8.5%), итог окна −13.4%,
//     максимальная просадка −36.2% — самая мягкая в каталоге. Часть лонгового результата
//     принадлежит режиму, поэтому разбивка точки по полугодиям обязательна.
//   - Шаг цены 0.5 ₽ при медианной цене 4081 ₽ = 0.0123%. Реальный круг издержек 0.125% против
//     моделируемых 0.1%. Baseline держит утроенный круг: PF 1.778 -> 1.411 (круг 0.2%) -> 1.098
//     (0.3%) — противоположность BSPB, где удвоение уводило дефолты в убыток.
//   - Ликвидность ВНЕ СРАВНЕНИЯ с каталогом: медиана оборота 2826 млн ₽/день, p10 1500 млн
//     (прежний рекорд — BSPB с 333 млн). Дней короче 20 баров — 0.2% (1 из 542). Исполнительный
//     риск, записанный у ELFV, DIAS и LENT, снят полностью.
//   - Корпоративных гэпов почти нет: за 25 месяцев ОДИН гэп хуже −3% (2024-08-05, −3.30%) и один
//     лучше +3%. Дивидендный риск BSPB здесь не является ведущим. Обратная сторона: бумага моложе
//     двух лет, структурных событий история почти не содержит.
//   - Дневной ATR(14) 2.84% медианой (выше BSPB 2.53%, ниже IVAT 4.37%). День проходит своё ATR
//     быстро: медиана размаха дня к бару 0.62 ATR против 0.34 у BSPB.
//   - Edge усиливается в двух направлениях: быстрый RSI на среднем уровне (RSI(3)@25 — 140 сделок
//     при PF 2.050) и исчерпанный день (SpentDayATR 1.0 — 58 сделок при 3.256). Обе зоны тоньше
//     дефолтной по выборке, и это главный риск калибровки.
package ydex

import "tinvest/internal/service/trading_strategy/rsi_pullback/strategy/core"

// Ticker is the instrument this package parameterises.
const Ticker = "YDEX"

// DefaultParams returns the baseline until calibration replaces it with a literal.
func DefaultParams() core.Params {
	return core.DefaultParams()
}
```

- [ ] **Step 4: Зарегистрировать тикер в реестре бэктеста**

В `internal/service/backtest/rsi_pullback_registry.go` добавить импорт в группу `rsipullback*`
(алфавитный порядок — после `rsipullbackwush`):

```go
	rsipullbackydex "tinvest/internal/service/trading_strategy/rsi_pullback/strategy/ydex"
```

и строку в карту `rsiPullbackRegistry`:

```go
	rsipullbackydex.Ticker:  rsiPullbackBindingFor(rsipullbackydex.Ticker, rsipullbackydex.DefaultParams),
```

- [ ] **Step 5: Добавить сторожевой тест реестра**

В `internal/service/backtest/rsi_pullback_registry_test.go` — импорт
`rsipullbackydex "tinvest/internal/service/trading_strategy/rsi_pullback/strategy/ydex"` и тест по
образцу того, что стоял у BSPB до калибровки:

```go
// TestRSIPullbackYDEXTracksBaseline сторожит ЧЕСТНОЕ состояние: YDEX заведён в реестр 2026-08-25
// ДО калибровки и обязан отдавать ровно core.DefaultParams(). Тест заменяется снимком литерала в
// Task 12 плана 2026-08-25-ydex-rsi-pullback-prep.md.
func TestRSIPullbackYDEXTracksBaseline(t *testing.T) {
	b, ok := rsiPullbackRegistry[rsipullbackydex.Ticker]
	if !ok {
		t.Fatal("YDEX отсутствует в rsiPullbackRegistry: тикер провалится в generic-ветку")
	}
	p, pok := b.DefaultParams().(core.Params)
	if !pok {
		t.Fatalf("YDEX: DefaultParams() вернул %T, want core.Params", b.DefaultParams())
	}
	if want := core.DefaultParams(); p != want {
		t.Fatalf("YDEX ещё не откалиброван:\n got: %+v\nwant: %+v", p, want)
	}
	if got := b.Build(p).Ticker(); got != "YDEX" {
		t.Fatalf("Ticker() = %q, want YDEX", got)
	}
}
```

Форма взята из `TestRSIPullbackBSPBIsRegisteredAndCalibrated` того же файла: реестр —
карта `rsiPullbackRegistry`, `Binding.DefaultParams()` отдаёт `any`, приводится к `core.Params`.

- [ ] **Step 6: Запустить тесты**

Run: `go test ./internal/service/trading_strategy/rsi_pullback/... ./internal/service/backtest/ -v`
Expected: PASS.

- [ ] **Step 7: Проверить, что прогон идёт через реестр**

```bash
go run ./cmd/backtest -ticker YDEX -strategy rsi_pullback -interval Minutes30 \
  -out ./reports/YDEX_probe -months 25
```

Expected: `trades=108, net=28302.07, PF=1.778` — те же числа, что в спеке. Расхождение означает,
что кэш изменился или прогон идёт не через реестр; в этом случае остановиться и разобраться до
первой темы, потому что все замеры спеки посчитаны на этом кэше.

- [ ] **Step 8: Коммит**

```bash
git add internal/service/trading_strategy/rsi_pullback/strategy/ydex internal/service/backtest
git commit -m "feat(rsi_pullback): пакет и реестр YDEX в состоянии «калибровка не проводилась»"
```

---

### Task 3: Тема `screen` — цена двух опциональных гейтов

**Files:**
- Modify: `data/params/rsi_pullback/ydex/cal_screen.json` (только `_comment`)

**Interfaces:**
- Consumes: каталог из Task 1, реестр из Task 2.
- Produces: числа цены гейтов, на которые ссылаются доки пакета и разбор Task 15.

- [ ] **Step 1: Прогнать тему**

```bash
go run ./cmd/backtest -ticker YDEX -strategy rsi_pullback -interval Minutes30 \
  -calibrate data/params/rsi_pullback/ydex/cal_screen.json -out ./reports/YDEX_screen \
  -months 25 -train-months 9 -test-months 4 -min-trades 1 -metric profit_factor
```

- [ ] **Step 2: Прочитать отчёт**

Открыть свежий файл в `reports/YDEX_screen/`. Выписать для каждой из четырёх комбинаций: pooled OOS
PF, число сделок в пуле, выбор по фолдам. Ожидание из точечных замеров (полная история, без
walk-forward): день включён + объём выключен — 108 сделок/1.778; день выключен — 396/0.960; объём
включён при 1.2 — 82/1.656. Прогон темы считает то же по фолдам, и числа будут другими — сравнивать
надо порядок, а не значения.

- [ ] **Step 3: Записать результат в `_comment` сетки**

Дописать после `РЕЗУЛЬТАТ ПРОГОНА 2026-08-25:` числа всех четырёх комбинаций и вывод: сколько
стоит каждый гейт в сделках и в PF. Обязательно отметить, подтвердил ли прогон точечный замер, что
дневной гейт разворачивает знак edge (0.960 без него против 1.778 с ним).

- [ ] **Step 4: Коммит**

```bash
git add data/params/rsi_pullback/ydex/cal_screen.json
git commit -m "feat(rsi_pullback): YDEX, тема screen прогнана"
```

---

### Task 4: Тема `entry` — ключевая, тройка против четвёрки

**Files:**
- Modify: `data/params/rsi_pullback/ydex/cal_entry.json` (только `_comment`)

**Interfaces:**
- Consumes: каталог из Task 1, реестр из Task 2.
- Produces: победителя pooled OOS (`RSIPeriod`, `RSILower`, `RSIUpper`) — его читает Task 11 при
  сборке точки.

- [ ] **Step 1: Прогнать тему**

```bash
go run ./cmd/backtest -ticker YDEX -strategy rsi_pullback -interval Minutes30 \
  -calibrate data/params/rsi_pullback/ydex/cal_entry.json -out ./reports/YDEX_entry \
  -months 25 -train-months 9 -test-months 4 -min-trades 20 -metric profit_factor
```

- [ ] **Step 2: Выписать числа планки**

Из отчёта выписать: pooled OOS PF, число сделок в пуле, разбивку по четырём фолдам (PF и сделки
каждого), выбор ведущей оси `RSILower` в каждом фолде, выбор `RSIPeriod` в каждом фолде.

Критерий 1 планки: pooled OOS PF ≥ 1.5 при ≥ 20 сделках.
Критерий 2 планки: `RSILower` выбран одинаково в ≥ 3 фолдах из 4.
Критерий 3: фолд без единой убыточной сделки в пользу тикера не засчитывается — если такой есть,
отметить отдельной строкой.

- [ ] **Step 3: Проверить, что решение по тройке подтвердилось**

Отдельно выписать, какой `RSIPeriod` выбрал каждый фолд. Это прямая проверка отступления от правила
каталога (риск 6 спеки): если тройку не выбрал ни один фолд, записать это честно — правило каталога
устояло, а точечный замер оказался свойством полной истории, а не бумаги.

- [ ] **Step 4: Проверить край оси периода**

Если победил `RSIPeriod = 6` (верхний край оси), проверить точечным прогоном, не растёт ли результат
дальше (`<L>` — победивший `RSILower`):

```bash
cat > /tmp/ydex_p7.json <<'EOF'
{"RSIPeriod":7,"RSILower":30,"RSIUpper":70,"EMAFast":10,"EMASlow":100,"DailyATRPeriod":14,"UseDayATRGate":1,"FreshDayATR":0,"SpentDayATR":0.8,"StopDailyATR":0.5,"TPDailyATR":0.6,"UseVolume":0,"VolBaseDays":14,"VolLookbackBars":3,"VolMult":1.2,"UseRSIExit":1,"UseTrail":0,"TrailDailyATR":0}
EOF
go run ./cmd/backtest -ticker YDEX -strategy rsi_pullback -interval Minutes30 \
  -params /tmp/ydex_p7.json -out ./reports/YDEX_entry_edge -months 25
```

Заменить `"RSILower":30` на победивший уровень перед запуском. Ожидание из замеров спеки: RSI(7) на
уровне 30 даёт 50 сделок при PF 1.143 — ниже дефолта, край оси стоит правильно. Записать факт в
`_comment`.

- [ ] **Step 5: Записать результат в `_comment` сетки**

Дописать после `РЕЗУЛЬТАТ ПРОГОНА 2026-08-25:` все числа Steps 2–4 и **вердикт по обоим критериям
планки отдельными фразами** («критерий PF взят/не взят: …», «критерий устойчивости взят/не взят:
…»).

- [ ] **Step 6: Коммит**

```bash
git add data/params/rsi_pullback/ydex/cal_entry.json
git commit -m "feat(rsi_pullback): YDEX, тема entry прогнана"
```

---

### Task 5: Тема `trend` — вторая ключевая

**Files:**
- Modify: `data/params/rsi_pullback/ydex/cal_trend.json` (только `_comment`)

**Interfaces:**
- Consumes: каталог из Task 1, реестр из Task 2.
- Produces: победителя pooled OOS (`EMAFast`, `EMASlow`) — его читает Task 11.

- [ ] **Step 1: Прогнать тему**

```bash
go run ./cmd/backtest -ticker YDEX -strategy rsi_pullback -interval Minutes30 \
  -calibrate data/params/rsi_pullback/ydex/cal_trend.json -out ./reports/YDEX_trend \
  -months 25 -train-months 9 -test-months 4 -min-trades 20 -metric profit_factor
```

- [ ] **Step 2: Выписать числа планки и прочитать их правильно**

Выписать pooled OOS PF, сделки пула, разбивку по фолдам, выбор ведущей оси `EMASlow` в каждом фолде.

Ведущая ось темы `trend` — **`EMASlow`**, не `EMAFast`: так считает планка у всего каталога.

Отдельно отметить, попал ли выбор на **50** — нижний край оси, где стоит точечный максимум
(пара 10/50, PF 1.802). Если победил край, проверить точечным прогоном пару со `EMASlow = 40`,
чтобы убедиться, что решение спеки не расширять ось вниз держится:

```bash
cat > /tmp/ydex_ema40.json <<'EOF'
{"RSIPeriod":4,"RSILower":30,"RSIUpper":70,"EMAFast":10,"EMASlow":40,"DailyATRPeriod":14,"UseDayATRGate":1,"FreshDayATR":0,"SpentDayATR":0.8,"StopDailyATR":0.5,"TPDailyATR":0.6,"UseVolume":0,"VolBaseDays":14,"VolLookbackBars":3,"VolMult":1.2,"UseRSIExit":1,"UseTrail":0,"TrailDailyATR":0}
EOF
go run ./cmd/backtest -ticker YDEX -strategy rsi_pullback -interval Minutes30 \
  -params /tmp/ydex_ema40.json -out ./reports/YDEX_trend_edge -months 25
```

Ожидание из спеки: 100 сделок при PF 1.434 — хуже 10/50, ось обрезана правильно.

- [ ] **Step 3: Записать результат в `_comment` сетки**

Дописать числа и вердикт по обоим критериям планки отдельными фразами.

- [ ] **Step 4: Коммит**

```bash
git add data/params/rsi_pullback/ydex/cal_trend.json
git commit -m "feat(rsi_pullback): YDEX, тема trend прогнана"
```

---

### Task 6: Темы `day` и `day_spent` — дневной гейт

**Files:**
- Modify: `data/params/rsi_pullback/ydex/cal_day.json` (только `_comment`)
- Modify: `data/params/rsi_pullback/ydex/cal_day_spent.json` (только `_comment`)

**Interfaces:**
- Consumes: каталог из Task 1.
- Produces: `FreshDayATR` и `SpentDayATR` для сборки точки (Task 11).

- [ ] **Step 1: Прогнать обе темы**

```bash
go run ./cmd/backtest -ticker YDEX -strategy rsi_pullback -interval Minutes30 \
  -calibrate data/params/rsi_pullback/ydex/cal_day.json -out ./reports/YDEX_day \
  -months 25 -train-months 9 -test-months 4 -min-trades 20 -metric profit_factor

go run ./cmd/backtest -ticker YDEX -strategy rsi_pullback -interval Minutes30 \
  -calibrate data/params/rsi_pullback/ydex/cal_day_spent.json -out ./reports/YDEX_day_spent \
  -months 25 -train-months 9 -test-months 4 -min-trades 20 -metric profit_factor
```

- [ ] **Step 2: Выписать числа обеих тем**

Для каждой: pooled OOS PF, сделки пула, разбивку по фолдам, выбор `SpentDayATR` и `FreshDayATR` в
каждом фолде.

- [ ] **Step 3: Прочитать выбор с оглядкой на число сделок**

Риск 1 спеки: максимумы дневного гейта стоят там, где выборка тонка. Точечные замеры на полной
истории: `SpentDayATR` 0.9 — 82 сделки, 1.0 — 58 (21 на девятимесячное обучающее окно при пороге
20), 1.25 — 28, 1.3 — 25 (9 на обучающее окно). Если тема выбрала уровень выше 1.0, выписать рядом
с PF число сделок пула и отметить, что выбор стоит на грани порога — это понадобится Task 11 при
сборке точки и Task 14 при разговоре о риске.

- [ ] **Step 4: Проверить, победил ли ноль в свежей ветке**

Если тема `day` выбрала `FreshDayATR > 0`, это первый такой случай среди прод-тикеров каталога —
записать отдельной строкой и проверить точечным прогоном, что связка «свежая ветка + выбранный
`SpentDayATR`» бьёт ту же связку с нулём.

- [ ] **Step 5: Записать результаты в `_comment` обеих сеток**

- [ ] **Step 6: Коммит**

```bash
git add data/params/rsi_pullback/ydex/cal_day.json data/params/rsi_pullback/ydex/cal_day_spent.json
git commit -m "feat(rsi_pullback): YDEX, темы дневного гейта прогнаны"
```

---

### Task 7: Тема `volume` — объёмный гейт

**Files:**
- Modify: `data/params/rsi_pullback/ydex/cal_volume.json` (только `_comment`)

**Interfaces:**
- Consumes: каталог из Task 1.
- Produces: `UseVolume`, `VolMult`, `VolBaseDays` для сборки точки (Task 11).

- [ ] **Step 1: Прогнать тему**

```bash
go run ./cmd/backtest -ticker YDEX -strategy rsi_pullback -interval Minutes30 \
  -calibrate data/params/rsi_pullback/ydex/cal_volume.json -out ./reports/YDEX_volume \
  -months 25 -train-months 9 -test-months 4 -min-trades 20 -metric profit_factor
```

- [ ] **Step 2: Выписать числа**

pooled OOS PF, сделки пула, разбивку по фолдам, выбор `VolMult` и `VolBaseDays` в каждом фолде.

- [ ] **Step 3: Сравнить с выключенным гейтом**

Из темы `screen` (Task 3) уже известен pooled OOS выключенного гейта. Если лучший `VolMult` даёт
PF, неразличимый с выключенным (разница меньше 0.1 PF), в точку берётся **выключенный гейт** как
более простая конфигурация — правило каталога, введённое на DIAS. Записать это решение явно.

- [ ] **Step 4: Записать результат в `_comment` сетки**

Обязательно отметить, подтвердил ли прогон точечный замер: максимум на `VolMult` 2.0 (53 сделки,
PF 1.987) при том, что 2.5 уже даёт 39 сделок (14 на обучающее окно), а 3.0 — 27 при PF 0.981.

- [ ] **Step 5: Коммит**

```bash
git add data/params/rsi_pullback/ydex/cal_volume.json
git commit -m "feat(rsi_pullback): YDEX, тема volume прогнана"
```

---

### Task 8: Тема `vol_window` — окно объёмного гейта, четвёртый замер оси в каталоге

**Files:**
- Modify: `data/params/rsi_pullback/ydex/cal_vol_window.json` (только `_comment`)

**Interfaces:**
- Consumes: каталог из Task 1, числа темы `volume` (Task 7).
- Produces: `VolLookbackBars` для сборки точки (Task 11) и ответ на спор ELFV/DIAS/BSPB.

- [ ] **Step 1: Прогнать тему**

```bash
go run ./cmd/backtest -ticker YDEX -strategy rsi_pullback -interval Minutes30 \
  -calibrate data/params/rsi_pullback/ydex/cal_vol_window.json -out ./reports/YDEX_vol_window \
  -months 25 -train-months 9 -test-months 4 -min-trades 20 -metric profit_factor
```

- [ ] **Step 2: Выписать числа и ответить на вопрос каталога**

Выписать pooled OOS PF, сделки, выбор `VolLookbackBars` по фолдам. Затем ответить письменно на
вопрос, ради которого тема стоит в плане: ELFV (18.9% коротких дней) дал максимум на дефолте ядра
3, DIAS (11.9%) — на 12, BSPB (7.7%) — на 5 и сформулировал правило «чем ликвиднее бумага, тем
ближе оптимум окна к дефолту». YDEX — 0.2% коротких дней, самая ликвидная бумага каталога, и
точечный максимум стоит на 1. Записать, какая редакция правила выживает: «ближе к дефолту» (|1−3|=2
против |12−3|=9 у DIAS) или «сходится к 3» (опровергается).

- [ ] **Step 3: Проверить эквивалентность выключенному гейту**

Если тема выбрала окно 8 или 12 (гейт ослаблен почти до неработающего), прогнать точечно ту же
конфигурацию с `UseVolume = 0` и сравнить. При неразличимых числах в точку берётся выключенный
гейт.

- [ ] **Step 4: Записать результат в `_comment` сетки**

- [ ] **Step 5: Коммит**

```bash
git add data/params/rsi_pullback/ydex/cal_vol_window.json
git commit -m "feat(rsi_pullback): YDEX, тема окна объёмного гейта прогнана"
```

---

### Task 9: Тема `risk` — стоп и цель

**Files:**
- Modify: `data/params/rsi_pullback/ydex/cal_risk.json` (только `_comment`)

**Interfaces:**
- Consumes: каталог из Task 1.
- Produces: `StopDailyATR`, `TPDailyATR` для сборки точки (Task 11).

- [ ] **Step 1: Прогнать тему**

```bash
go run ./cmd/backtest -ticker YDEX -strategy rsi_pullback -interval Minutes30 \
  -calibrate data/params/rsi_pullback/ydex/cal_risk.json -out ./reports/YDEX_risk \
  -months 25 -train-months 9 -test-months 4 -min-trades 20 -metric profit_factor
```

- [ ] **Step 2: Выписать числа**

pooled OOS PF, сделки, выбор `StopDailyATR` и `TPDailyATR` по фолдам.

- [ ] **Step 3: Прочитать выбор стопа против капкана**

Если тема выбрала `StopDailyATR` 1.0 или 1.3, это ожидаемый след капкана вытеснения убытка: на
точечных замерах PF растёт со стопом (0.5 → 1.806, 0.7 → 2.087, 1.3 → 2.362) при почти неизменном
числе сделок (110 против 108), то есть широкий стоп не защищает, а переносит убыток в RSI-выход.
Каталог ловил этот капкан на WUSH, LENT, LSNGP, IVAT, SVAV, SIBN и BSPB. Записать отдельной
строкой: сколько сделок закрылось по стопу при выбранном значении и при 0.5 (это видно в отчёте
точечного прогона по колонке причин выхода).

Решение, какой стоп идёт в точку, принимается в Task 11 и обосновывается там же — тема лишь
предоставляет числа.

- [ ] **Step 4: Проверить нижний край цели**

Точечные замеры дают максимум каждой строки на цели 0.5, а 0.4 идёт второй. Если тема выбрала 0.4
(нижний край расширенной оси), проверить точечным прогоном цель 0.3 с выбранным стопом и записать
результат: если 0.3 лучше, ось цели придётся расширять дальше в следующей редакции каталога, и это
надо зафиксировать честно.

- [ ] **Step 5: Записать результат в `_comment` сетки**

Обязательно отметить, подтвердилось ли, что колонки 1.5, 2.0 и 2.5 неразличимы (цель шире 1.5 ATR
недостижима) — в отчёте это видно по совпадению строк.

- [ ] **Step 6: Коммит**

```bash
git add data/params/rsi_pullback/ydex/cal_risk.json
git commit -m "feat(rsi_pullback): YDEX, тема risk прогнана"
```

---

### Task 10: Темы `exit` и `trail` — выходы

**Files:**
- Modify: `data/params/rsi_pullback/ydex/cal_exit.json` (только `_comment`)
- Modify: `data/params/rsi_pullback/ydex/cal_trail.json` (только `_comment`)

**Interfaces:**
- Consumes: каталог из Task 1.
- Produces: `RSIUpper`, `UseRSIExit`, `UseTrail`, `TrailDailyATR` для сборки точки (Task 11).

- [ ] **Step 1: Прогнать обе темы**

```bash
go run ./cmd/backtest -ticker YDEX -strategy rsi_pullback -interval Minutes30 \
  -calibrate data/params/rsi_pullback/ydex/cal_exit.json -out ./reports/YDEX_exit \
  -months 25 -train-months 9 -test-months 4 -min-trades 20 -metric profit_factor

go run ./cmd/backtest -ticker YDEX -strategy rsi_pullback -interval Minutes30 \
  -calibrate data/params/rsi_pullback/ydex/cal_trail.json -out ./reports/YDEX_trail \
  -months 25 -train-months 9 -test-months 4 -min-trades 20 -metric profit_factor
```

- [ ] **Step 2: Выписать числа обеих тем**

Для `exit`: pooled OOS PF, сделки, выбор `RSIUpper` по фолдам. Отметить, попал ли выбор на верхний
край 80 — если да, записать, что решение НЕ расширять ось до 85 (замер 1.204) требует перепроверки
точечным прогоном, и выполнить эту перепроверку.

Для `trail`: pooled OOS PF, сделки, выбор `UseRSIExit` и `TrailDailyATR` по фолдам. Ожидание из
замеров: трейл не помогает (лучшее 0.5 → 1.783 против baseline 1.778, при 1.0 и выше не срабатывает
ни разу), а `UseRSIExit = 0` резко хуже (95 сделок при 1.058).

- [ ] **Step 3: Записать результаты в `_comment` обеих сеток**

- [ ] **Step 4: Коммит**

```bash
git add data/params/rsi_pullback/ydex/cal_exit.json data/params/rsi_pullback/ydex/cal_trail.json
git commit -m "feat(rsi_pullback): YDEX, темы выходов прогнаны"
```

---

### Task 11: Сборка точки, её walk-forward и проверка стоп-условия

**Files:**
- Create: `data/params/rsi_pullback/ydex/plateau_point.json`

**Interfaces:**
- Consumes: лидерборды всех десяти тем (Tasks 3–10).
- Produces: принятую точку — 18 полей `core.Params`, которые Task 12 переносит в литерал.

- [ ] **Step 1: Собрать точку из лидербордов тем**

Правило сборки, одинаковое для всего каталога: каждое поле берётся из темы, которая его меряет; там,
где выбор темы под остальными осями точки не выживает, поле проверяется точечным прогоном и решение
записывается в `_comment`.

Источники: `RSIPeriod`, `RSILower` — `cal_entry`; `RSIUpper` — `cal_exit` (тема `entry` свипует его
тоже, но `exit` меряет его отдельной осью); `EMAFast`, `EMASlow` — `cal_trend`; `FreshDayATR`,
`SpentDayATR` — `cal_day`/`cal_day_spent`; `UseVolume`, `VolMult`, `VolBaseDays`,
`VolLookbackBars` — `cal_volume`/`cal_vol_window`/`cal_screen`; `StopDailyATR`, `TPDailyATR` —
`cal_risk`; `UseRSIExit`, `UseTrail`, `TrailDailyATR` — `cal_trail`; `DailyATRPeriod` — 14, ось не
свипуется нигде.

- [ ] **Step 2: Создать файл точки**

`data/params/rsi_pullback/ydex/plateau_point.json` — все 18 полей списками из ОДНОГО значения
(этого требует `TestRSIPullbackPlateauFilesArePoints`). Форма и обязательные разделы `_comment` — по
образцу `data/params/rsi_pullback/bspb/plateau_point.json`: собственный путь файла; откуда взято
каждое поле; оговорка, что точка собрана человеком, видевшим всю историю, и потому её числа нельзя
сравнивать с pooled OOS тем; вердикт по планке пункт за пунктом; замеры Steps 3–7; полная команда
запуска со схемой 25/9/4.

- [ ] **Step 3: Прогнать точку тем же walk-forward**

```bash
go run ./cmd/backtest -ticker YDEX -strategy rsi_pullback -interval Minutes30 \
  -calibrate data/params/rsi_pullback/ydex/plateau_point.json -out ./reports/YDEX_point_oos \
  -months 25 -train-months 9 -test-months 4 -min-trades 20 -metric profit_factor
```

Выписать pooled OOS PF, сделки пула, PF и сделки каждого из четырёх фолдов.

- [ ] **Step 4: Проверить стоп-условие плана — пункты 1 и 2**

- pooled OOS PF < 1.0 → **СТОП**;
- меньше 20 сделок за расчётное окно → **СТОП**.

При срабатывании: числа приносятся владельцу, Tasks 12–15 НЕ выполняются.

- [ ] **Step 5: Проверить стоп-условие плана — пункт 3 (удвоенные издержки)**

```bash
go run ./cmd/backtest -ticker YDEX -strategy rsi_pullback -interval Minutes30 \
  -calibrate data/params/rsi_pullback/ydex/plateau_point.json -out ./reports/YDEX_point_comm \
  -months 25 -train-months 9 -test-months 4 -min-trades 20 -metric profit_factor -commission 0.001
```

PF < 1.0 → **СТОП**. Записать процент потери PF и сравнить с каталогом: у DIAS точка теряла 19.7%,
у ELFV 18.1%, а baseline YDEX теряет 20.6% (1.778 → 1.411). Реальный круг издержек здесь 0.125%,
то есть удвоение вдесятеро строже реальности.

- [ ] **Step 6: Замерить плато соседями**

Прогнать точечно 4–6 конфигураций-соседей точки (по одному шагу сетки в каждую сторону вдоль
ведущих осей: `RSILower` ±1 шаг, `SpentDayATR` ±1 шаг, `TPDailyATR` ±1 шаг). Записать в `_comment`:
стоит ли точка на плато или на пике. Точка на пике — риск, и он записывается, а не скрывается.

- [ ] **Step 7: Проверить, что результат не держится одним полугодием**

Разбить сделки точки по пяти полугодиям расчётного окна (2024-07-25 … 2025-01-23,
2025-01-23 … 2025-07-24, 2025-07-24 … 2026-01-22, 2026-01-22 … 2026-07-23, 2026-07-23 … 2026-08-25)
и записать вклад каждого. Обязательно из-за риска 3 спеки: два полугодия из пяти росли (+8.1% и
+8.5%), и часть результата может принадлежать режиму. Если больше половины net-результата даёт одно
полугодие — записать это прямо (прецедент IVAT, где 85% результата делала одна неделя июля).

- [ ] **Step 8: Коммит**

```bash
git add data/params/rsi_pullback/ydex/plateau_point.json
git commit -m "feat(rsi_pullback): YDEX, принятая точка и её замеры"
```

---

### Task 12: Литерал в пакете и снимок

**Files:**
- Modify: `internal/service/trading_strategy/rsi_pullback/strategy/ydex/ydex.go`
- Modify: `internal/service/trading_strategy/rsi_pullback/strategy/ydex/ydex_test.go`
- Modify: `internal/service/backtest/rsi_pullback_registry_test.go`

**Interfaces:**
- Consumes: точку из Task 11.
- Produces: `ydex.DefaultParams()`, возвращающий литерал, — его читают Tasks 13 и 14.

- [ ] **Step 1: Заменить тест baseline снимком литерала**

В `ydex_test.go` удалить `TestParamsTrackTheBaselineUntilCalibrated` и написать вместо него набор
тестов по образцу `bspb_test.go`, где каждое утверждение несёт число из прогонов:

```go
func TestCalibratedLiteralIsPinned(t *testing.T) {
	want := core.Params{ /* 18 полей принятой точки из Task 11 */ }
	if got := DefaultParams(); got != want {
		t.Fatalf("откалиброванный литерал YDEX изменился:\n got: %+v\nwant: %+v", got, want)
	}
}

func TestParamsDoNotTrackTheBaseline(t *testing.T) {
	if DefaultParams() == core.DefaultParams() {
		t.Fatal("YDEX вернул core.DefaultParams(): откалиброванный тикер не должен отслеживать baseline")
	}
}

func TestStopIsArmed(t *testing.T) {
	if p := DefaultParams(); p.StopDailyATR <= 0 {
		t.Fatalf("StopDailyATR = %v, want > 0", p.StopDailyATR)
	}
}

func TestStopStaysAboveTheCostFloor(t *testing.T) {
	if p := DefaultParams(); p.StopDailyATR < 0.3 {
		t.Fatalf("StopDailyATR = %v: на стопе 0.3 ATR реальный круг издержек 0.125%% съедает 14.6%% риска, уже — ещё больше", p.StopDailyATR)
	}
}

func TestTargetIsArmed(t *testing.T) {
	if p := DefaultParams(); p.TPDailyATR <= 0 {
		t.Fatalf("TPDailyATR = %v, want > 0", p.TPDailyATR)
	}
}

func TestTargetStaysReachable(t *testing.T) {
	if p := DefaultParams(); p.TPDailyATR > 1.5 {
		t.Fatalf("TPDailyATR = %v: на YDEX цель шире 1.5 дневного ATR недостижима — колонки 1.5, 2.0 и 2.5 замера совпадают побайтово", p.TPDailyATR)
	}
}

func TestTrendPairIsValid(t *testing.T) {
	if p := DefaultParams(); p.EMAFast >= p.EMASlow {
		t.Fatalf("EMAFast = %d >= EMASlow = %d: трендовый фильтр вырожден или инвертирован", p.EMAFast, p.EMASlow)
	}
}

func TestEntryBandIsBelowTheExitBand(t *testing.T) {
	if p := DefaultParams(); p.RSILower >= p.RSIUpper {
		t.Fatalf("RSILower = %v >= RSIUpper = %v: вход и выход перепутаны", p.RSILower, p.RSIUpper)
	}
}
```

- [ ] **Step 2: Запустить тесты и убедиться, что они падают**

Run: `go test ./internal/service/trading_strategy/rsi_pullback/strategy/ydex/ -v`
Expected: FAIL — пакет всё ещё возвращает baseline.

- [ ] **Step 3: Поставить литерал в пакет**

В `ydex.go` заменить тело `DefaultParams()` литералом принятой точки и переписать доку пакета:
убрать блок «СОСТОЯНИЕ: КАЛИБРОВКА НЕ ПРОВОДИЛАСЬ», вписать вместо него результат — вердикт по
планке пункт за пунктом, pooled OOS PF точки и число сделок, пофолдовую разбивку, замер под
удвоенными издержками, разбивку по полугодиям, оговорку о подглядывании. Остальные блоки доки
(окно, схема, априор, свойства инструмента) сохранить.

```go
// DefaultParams returns the calibrated literal accepted on 2026-08-25.
func DefaultParams() core.Params {
	return core.Params{ /* 18 полей принятой точки */ }
}
```

- [ ] **Step 4: Заменить сторожевой тест реестра снимком**

В `internal/service/backtest/rsi_pullback_registry_test.go` удалить
`TestRSIPullbackYDEXTracksBaseline` и написать вместо него
`TestRSIPullbackYDEXIsRegisteredAndCalibrated` по образцу
`TestRSIPullbackBSPBIsRegisteredAndCalibrated`: тикер обязан быть в `rsiPullbackRegistry`, его
параметры обязаны совпадать с `rsipullbackydex.DefaultParams()` и не совпадать с
`core.DefaultParams()`.

- [ ] **Step 5: Запустить тесты**

Run: `go test ./internal/service/trading_strategy/rsi_pullback/... ./internal/service/backtest/ -v`
Expected: PASS.

- [ ] **Step 6: Коммит**

```bash
git add internal/service/trading_strategy/rsi_pullback/strategy/ydex internal/service/backtest
git commit -m "feat(rsi_pullback): YDEX откалиброван — литерал вместо отслеживания baseline"
```

---

### Task 13: Реестр живого раннера

**Files:**
- Modify: `internal/service/trading_strategy/rsi_pullback/live/registry.go`
- Modify: `internal/service/trading_strategy/rsi_pullback/live/registry_test.go`

**Interfaces:**
- Consumes: `ydex.Ticker`, `ydex.DefaultParams()` из Task 12.
- Produces: запись в `paramsByTicker` — её читает `ParamsFor` живого раннера и `cmd/pullparity`.

- [ ] **Step 1: Добавить импорт и запись в карту**

Импорт `"tinvest/internal/service/trading_strategy/rsi_pullback/strategy/ydex"` и строка в
`paramsByTicker` рядом с `bspb.Ticker`:

```go
	ydex.Ticker:  ydex.DefaultParams(),
```

- [ ] **Step 2: Дописать комментарий карты**

Перед картой в `registry.go` стоят абзацы про каждый заведённый тикер. Дописать абзац про YDEX на
английском, в стиле соседних: адаптированная схема 25/9/4 и почему (остановка торгов 2024-06-14 …
2024-07-24, окно только после старта YDEX), вердикт по планке, сильнейший baseline каталога
(1.778), каноническая процедура тем, ликвидность как снятый исполнительный риск, короткая история
бумаги как остаточный риск.

- [ ] **Step 3: Запустить тесты пакета**

Run: `go test ./internal/service/trading_strategy/rsi_pullback/live/... -v`
Expected: PASS. В пакете есть два теста, которые прямо сторожат эту пару задач:
`TestEveryDefaultTickerIsRegistered` (каждый тикер боевой вселенной обязан быть в карте) и
`TestBaselineTrackingTickersStayOutOfTheDefaultUniverse` (тикер, всё ещё отслеживающий baseline, в
боевую вселенную попасть не может). Второй — причина, по которой Task 12 идёт раньше Task 14.

- [ ] **Step 4: Коммит**

```bash
git add internal/service/trading_strategy/rsi_pullback/live
git commit -m "feat(rsi_pullback): YDEX в реестре живого раннера"
```

---

### Task 14: Боевая вселенная

**Files:**
- Modify: `internal/config/rsi_pullback.go`
- Modify: `internal/config/rsi_pullback_test.go` (список `want` в тесте дефолта вселенной)
- Modify: `env/prod.env`, `env/prod.env.example`, `env/local.env.example`
- Modify: `docs/rsi_pullback/live.md` (§8, таблица боевой вселенной)

**Interfaces:**
- Consumes: литерал из Task 12, реестр из Task 13.
- Produces: `RSI_PULLBACK_TICKERS` из семнадцати тикеров.

- [ ] **Step 1: Проверить стоп-условие ещё раз**

Прежде чем заводить тикер в прод, перечитать числа Task 11 Steps 4–5. Все три пункта стоп-условия
не сработали — иначе эта задача не выполняется вовсе.

- [ ] **Step 2: Обновить тест дефолта**

В тесте пакета `internal/config`, который сторожит состав `Tickers`, добавить `"YDEX"` в ожидаемый
список семнадцатым.

- [ ] **Step 3: Запустить тест и убедиться, что падает**

Run: `go test ./internal/config/ -v`
Expected: FAIL — дефолт ещё из шестнадцати тикеров.

- [ ] **Step 4: Обновить дефолт и env-файлы**

В `internal/config/rsi_pullback.go` дописать `"YDEX"` в конец списка `Tickers` и добавить перед ним
комментарий-абзац в стиле соседних (BSPB, DIAS, ELFV): когда заведён, вердикт по планке, числа
точки, **тип риска** (у YDEX он не исполнительный, как у ELFV, и не дивидендный, как у BSPB, а
риск короткой истории и адаптированной схемы), каноническая процедура тем и сильнейший baseline
каталога.

Во всех трёх env-файлах дописать `,YDEX` в конец `RSI_PULLBACK_TICKERS`. Текущее значение:
`UGLD,T,GAZP,DOMRF,FESH,WUSH,LENT,RENI,NVTK,LSNGP,IVAT,SVAV,SIBN,ELFV,DIAS,BSPB`.

- [ ] **Step 5: Обновить `live.md` §8**

Дописать YDEX в таблицу боевой вселенной: тикер, дата заведения, взята ли планка, pooled OOS PF
точки и число сделок, ключевой риск, схема прогонов (25/9/4 — отметить отдельно, потому что она
отличается от штатной).

- [ ] **Step 6: Запустить тесты**

Run: `go test ./internal/config/ ./internal/service/trading_strategy/rsi_pullback/... -v`
Expected: PASS.

- [ ] **Step 7: Коммит**

```bash
git add internal/config env docs/rsi_pullback/live.md
git commit -m "feat(rsi_pullback): завести YDEX в боевую вселенную"
```

---

### Task 15: Документация — разбор калибровки и принятый риск

**Files:**
- Modify: `docs/rsi_pullback/strategy.md`
- Modify: `docs/rsi_pullback/live.md`

**Interfaces:**
- Consumes: все числа Tasks 3–11.
- Produces: раздел YDEX в справочнике стратегии и риск в `live.md` §10.

- [ ] **Step 1: Дописать §8.0.1 `strategy.md`**

В таблицу откалиброванных тикеров добавить строку YDEX: схема прогонов (25/9/4 — адаптированная,
как у IVAT), вердикт по планке, pooled OOS PF точки и сделки, дата.

- [ ] **Step 2: Дописать раздел с разбором прогонов**

Новый раздел про YDEX по образцу разделов BSPB и DIAS. Обязательно:

1. **Окно и остановка торгов** — почему расчёт начинается 2024-07-25, что именно выброшено (дыра
   40 дней и вся YNDX-эпоха), и чем за это заплачено (сравнимость с каталогом).
2. **Сильнейший baseline каталога** (1.778) и его следствие — каноническая процедура тем без
   якоря. Это зеркало случая BSPB, и следующий тикер должен уметь опознать оба.
3. **Чем кончилось отступление по `RSIPeriod = 3`** — выбрали ли фолды тройку, и если да, устояла
   ли она в точке. Правило каталога либо получает второе исключение с обоснованием, либо
   восстанавливается.
4. **Четвёртый замер оси `vol_window`** — какая редакция правила BSPB выжила («ближе к дефолту»
   против «сходится к дефолту»).
5. **Режим** — два растущих полугодия из пяти и что это значит для доверия к числам.

- [ ] **Step 3: Дописать риск в `live.md` §10**

Новый риск номером, следующим за последним занятым (у BSPB это 19) — **короткая история и
адаптированная схема YDEX**:

- бумага торгуется под этим тикером с 2024-07-24, расчётное окно 25 месяцев, схема 25/9/4;
  обучающее окно 9 месяцев вместо 12, и числа не сравнимы построчно с 36-месячным каталогом;
- за окно пришёлся ОДИН гэп хуже −3% и ни одной крупной дивидендной отсечки: замеры описывают
  спокойный участок биографии, а не устойчивое свойство бумаги. Корпоративные события впереди;
- условие пересмотра: первая живая просадка глубже той, что дала точка на бэктесте, либо первое
  корпоративное событие с гэпом хуже −5%.

Отдельным пунктом дописать, что ликвидность YDEX (медиана 2826 млн ₽/день, p10 1500 млн, дней
короче 20 баров — 0.2%) снимает исполнительный риск, записанный у ELFV, DIAS и LENT, и что это
самая ликвидная бумага вселенной — прежний рекорд принадлежал NVTK и BSPB.

- [ ] **Step 4: Проверить, что доки не противоречат числам**

Перечитать написанное рядом с `_comment` сеток и докой пакета: одни и те же числа обязаны совпадать
во всех трёх местах. Расхождение — ошибка, а не стилистика (на SVAV, DIAS и BSPB такие расхождения
находились и исправлялись отдельными коммитами).

- [ ] **Step 5: Коммит**

```bash
git add docs/rsi_pullback
git commit -m "docs(rsi_pullback): разбор калибровки YDEX и принятый риск"
```

---

### Task 16: Финальная проверка

**Files:** нет изменений, только проверки (кроме Step 3).

- [ ] **Step 1: Полный гейт качества**

Run: `./bin/mage ci`
Expected: lint + `go test -race ./...` + проверка дрейфа моков — зелёные.

- [ ] **Step 2: Сверка живой сборки с бэктестом**

```bash
go run ./cmd/pullparity -tickers YDEX -months 24
```

Expected: ноль расхождений. **24 месяца, а не 25:** живой сборщик тянет дневные свечи окном
`dailyFetchDays = 730` (`live/marketdata/marketdata.go:47`), и на большем горизонте появляются
ожидаемые расхождения длины `Daily*` рядов (`maxDailyHorizonMonths`, выяснено на IVAT). Расхождение
на 24 месяцах означает, что живой раннер и бэктест считают сигнал по-разному, и заведение в прод
откатывается до выяснения.

- [ ] **Step 3: Записать результат сверки в `live.md` §9**

Строка вида «YDEX заведён 2026-08-25 и сверяется за 24 месяца (`go run ./cmd/pullparity -tickers
YDEX -months 24` — <N> баров, **ноль расхождений**)» рядом с такими же строками про IVAT, SVAV,
SIBN, ELFV, DIAS и BSPB. Коммит: `docs(rsi_pullback): сверка YDEX — 24 месяца, ноль расхождений`.

- [ ] **Step 4: Сообщить владельцу результат**

Короткий отчёт: вердикт по планке пункт за пунктом; замеры принятой точки (pooled OOS, пофолдовая
разбивка, удвоенные издержки, плато, разбивка по полугодиям); что заведено в прод; какие риски
записаны; что осталось (первые живые сделки, условия пересмотра). Тремя отдельными строками:

1. **чем кончилось отступление по `RSIPeriod = 3`** — выбрали ли фолды тройку, устояла ли она в
   точке, и получает ли правило каталога второе исключение или восстанавливается;
2. **что дал сильнейший baseline каталога** — подтвердилось ли, что тикер с рабочими дефолтами
   проходит процедуру канонически, и по какому признаку опознавать такой тикер заранее;
3. **чем кончился четвёртый замер оси `vol_window`** — какая редакция правила BSPB о связи
   ликвидности и оптимального окна выжила.

---

## Self-review

**Покрытие спеки.** Окно и остановка торгов → Global Constraints, дока пакета (Task 2 Step 3),
сторожевой тест (Task 1 Step 1), Task 15 Step 2 пункт 1; схема 25/9/4 → Global Constraints и каждая
команда прогона, дока пакета, Task 14 Step 5; кэш и нулевой запас → Global Constraints, дока
пакета, Task 2 Step 7; априор → дока пакета; каноническая процедура тем → Architecture, Task 1
(все десять файлов сразу), дока пакета, Task 15 Step 2 пункт 2; Свойство 1 (режим) → дока пакета,
Task 11 Step 7, Task 15 Step 2 пункт 5; Свойство 2 (шаг цены и издержки) → `_comment` `cal_risk`,
Task 11 Step 5, Task 12 Step 1 (`TestStopStaysAboveTheCostFloor`); Свойство 3 (ликвидность) → дока
пакета, `_comment` `cal_vol_window`, Task 15 Step 3; Свойство 4 (гэпы) → дока пакета, Task 15
Step 3; Свойство 5 (трендовый допуск и вырожденные пары) → сторож Task 1, `_comment` `cal_trend`,
Task 12 Step 1 (`TestTrendPairIsValid`); Свойство 6 (ATR и выживаемость стопов) → `_comment`
`cal_risk`, Task 9 Step 3; Свойство 7 (объёмный гейт) → `cal_volume` и Task 7; Свойство 8 (окно
гейта) → `cal_vol_window` и Task 8; Свойство 9 (дневной гейт) → `cal_day`/`cal_day_spent` и Task 6;
Свойство 10 (где живёт edge) → оси всех сеток, Task 4 Step 3, Task 9 Step 3; планка → Global
Constraints, вердикт выносится в Tasks 4, 5, 11, 12, 15; правило прода и стоп-условие → Global
Constraints, Task 11 Steps 4–5, Task 14 Step 1; артефакты спеки → задачи 1, 2, 11, 12, 13, 14, 15;
порядок работы спеки → порядок задач; риск 1 спеки (малочисленная зона edge) → Task 6 Step 3,
Task 7 Step 3, Task 11 Step 6; риск 3 (режим) → Task 11 Step 7; риск 5 (короткая история) →
Task 15 Step 3; риск 6 (отступление по тройке) → Task 4 Step 3, Task 15 Step 2 пункт 3, Task 16
Step 4 пункт 1; риск 8 (издержки) → Task 11 Step 5.

**Плейсхолдеры.** В задачах 3–11 значения результатов прогонов не выдуманы: там, где число станет
известно только после запуска, стоит явное указание выписать его из отчёта — это данные, которых до
прогона не существует, а не плейсхолдер плана. Комментарии `/* 18 полей принятой точки */` в
Task 12 — единственная подстановка, и её содержимое определяется правилом сборки Task 11 Step 1;
раньше Task 11 этих значений не существует физически. Код сторожевого теста, теста baseline, доки
пакета и всех десяти сеток дан целиком.

**Согласованность имён.** `ydexGrid` определён и используется только в
`rsi_pullback_ydex_grid_test.go` (Task 1); `TestRSIPullbackYDEXTracksBaseline` (Task 2 Step 5)
заменяется на `TestRSIPullbackYDEXIsRegisteredAndCalibrated` (Task 12 Step 4);
`TestParamsTrackTheBaselineUntilCalibrated` (Task 2 Step 1) заменяется на
`TestCalibratedLiteralIsPinned` (Task 12 Step 1); `ydex.Ticker` и `ydex.DefaultParams()` объявлены
в Task 2 и потребляются в Tasks 12–14. Импорт в реестре бэктеста назван `rsipullbackydex` — как
соседние `rsipullbackbspb` и `rsipullbackdias`; в живом реестре — `ydex`, как соседний `bspb`.
