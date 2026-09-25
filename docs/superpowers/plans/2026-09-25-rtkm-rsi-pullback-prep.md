# RTKM под `rsi_pullback` — план подготовки, калибровки и вердикта

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development
> (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use
> checkbox (`- [ ]`) syntax for tracking.

**Goal:** Довести RTKM (ПАО «Ростелеком», обыкновенные акции) до вердикта по стратегии
`rsi_pullback`. Нужно собрать максимально широкие сетки, прогнать тематический walk-forward и
получить точку внутри риск-гейтов A и B, которая проходит семь пунктов стоп-условия. Если первый
круг провален — второй круг по правилам владельца. Результат — либо литерал в пакете и заведение
в боевую вселенную тридцать третьим тикером, либо протокол отказа.

**Architecture:** Процедура каноническая (спека §5). Три отличия от каталога.

1. **Главный критерий владельца вынесен в стоп-условие:** годы 2024, 2025 и 2026 в плюс.
2. **Жёсткий пункт устойчивости:** хвост 6 месяцев против хвоста 12 месяцев по дате входа
   (решение владельца 2026-09-25).
3. **Потолок гейта A = 0.8** — по частоте срабатывания стопа, а не по выживаемости.

Разведка показала: дефолты проваливают 2026 год, оба хвоста и контроль 24/12/3. Единственный
рычаг, который лечит 2026 год, — короткая `EMASlow` (горб 12–20). Рабочая зона — связка короткого
тренда и стопа 0.7–0.8; отсюда тема-арбитр `trend_stop`. Главный риск первого круга — рассыпание
голосов фолдов по `EMASlow`, поэтому второй круг вероятен.

**Tech Stack:** Go 1.25, `cmd/backtest` (rolling walk-forward), `cmd/pullparity` (сверка живой
сборки с бэктестом), `./bin/mage ci` (lint + `go test -race ./...` + дрейф моков), `python3` для
разбора журналов сделок.

**Spec:** `docs/superpowers/specs/2026-09-25-rtkm-rsi-pullback-prep-design.md` — план ссылается на её
параграфы (§N), исполнитель читает оба документа.
**Замеры:** `reports/_analysis/rtkm_pullback_prep_measurements.md`, сырые отчёты разведки —
`reports/RTKM_prep/`.

## Global Constraints

**Ветка, кэш и окно**

- **Ветка:** `feat/rtkm-pullback-prep` от `main` `35a4161` (в боевой вселенной 32 тикера, последний —
  `SFIN`). Ветка создана, спека закоммичена (`4c5057e`).
- **Таймфрейм `Minutes30` во всех прогонах.** Флаг **`-interval Minutes30` обязателен в каждой
  команде `cmd/backtest`**. Дефолт CLI — `Hour1`, и забытый флаг даёт чужие числа, а не ошибку.
- **`-refresh` НЕ запускать.** Кэш дотянут 2026-09-25:
  - `data/candles/RTKM_Minutes30.json` — 36 615 баров, 2023-09-25 09:30 … 2026-09-25 08:30;
  - `RTKM_Day1.json` — 1 374 бара.

  Повторный `-refresh` перезапишет кэш и сделает числа плана невоспроизводимыми.
- **Окно тем и точки — `-months 36`.** Контроли точки идут с `-months 30` и `-months 24`. Никакие
  другие прогоны окно не меняют.
- **`-months` отсчитывается от момента запуска.** Кэш не растёт, поэтому поздний запуск лишь
  отрезает начало окна. Первая сделка дефолтов — 2023-10-09. До **2026-10-09** числа не меняются.
  Если план исполняется позже и baseline в Task 2 Step 6 не сошёлся, остановиться и доложить
  владельцу.

**Схемы прогонов**

- **Тематические прогоны:** `-months 36 -train-months 12 -test-months 6 -min-trades 20
  -metric profit_factor`. У темы `screen` — `-min-trades 1`. Все одиннадцать тем идут только по этой
  схеме, иначе они несопоставимы.
- **Три схемы проверки точки** (не тем):

  | Схема | Флаги | Фолдов | Роль | Дефолты |
  |---|---|---|---|---|
  | **36/12/6** | `-months 36 -train-months 12 -test-months 6` | 4 | сборка и вердикт | **1.247** / пул 85; фолды 2.496/20, 1.223/25, 0.852/25, 0.678/15 |
  | **24/12/3** | `-months 24 -train-months 12 -test-months 3` | 4 | **пункт 3 стоп-условия** | **0.784** / пул 40 |
  | 30/12/6 | `-months 30 -train-months 12 -test-months 6` | 3 | только записывается | 0.975 |

- **Вырожденный фолд** — меньше пяти сделок или ни одной убыточной сделки. В пользу тикера он не
  засчитывается. Вердикт по схеме выносится по pooled OOS, оговорка записывается.
- **Число фолдов сверяется на первой же теме:** в шапке отчёта темы `screen` должно стоять
  «Фолдов: 4». Иначе остановиться и доложить владельцу (ловушка ASTR).

**Инструмент**

- **Лот 10, шаг цены 0.01 ₽.** Реальный круг издержек — **0.051%** при цене 39.10 ₽. Модель
  (`-commission 0.0005`, круг 0.1%) **пессимистична вдвое**. Пункта про реальный круг нет — отличие
  от RTKMP, где шаг 0.05 ₽.
- **Гэпы-прокси отсечек и новостных ступеней** — четыре даты: **2023-12-01 (−6.5%), 2024-09-27
  (−6.7%), 2025-08-13 (−3.5%), 2026-07-18 (−6.1%)**. Они выписаны в `EX_DATES` скрипта
  `reports/_analysis/rtkm_journal.py`, не менять.

**Сетки**

Сетки максимально широкие: обрезок нет, предупреждения о краях идут в `_comment`. Жёсткие
инварианты (§5.1 спеки):

- `RSILower ≤ 50`, `RSIPeriod ≥ 2`, `StopDailyATR` нигде не ноль;
- ни один файл не порождает пар `EMAFast ≥ EMASlow`;
- файл, свипующий `EMASlow` без `EMAFast`, держит `EMASlow` строго выше дефолта ядра `EMAFast` 10;
- ось `RSILower` в `cal_entry.json` содержит 5 и 50;
- ось `EMASlow` в `cal_trend_low.json` содержит 12 и 45;
- ось `EMASlow` в `cal_trend.json` содержит 250;
- ось `RSIUpper` в `cal_exit.json` содержит 35 и 95;
- ось `StopDailyATR` в `cal_risk.json` содержит 2.0;
- ось `TPDailyATR` в `cal_risk.json` содержит 0.15 и 2.5;
- ось `FreshDayATR` в `cal_day.json` содержит 0, 0.05, 0.15 и 0.5;
- ось `SpentDayATR` в `cal_day.json` содержит 2.0;
- ось `VolLookbackBars` в `cal_vol_window.json` содержит 32;
- ось `StopDailyATR` в `cal_trend_stop.json` не выше 0.8.

**Риск-гейты**

- **Гейт A:** `min(StopDailyATR, TrailDailyATR при UseTrail = 1)` ≤ **0.8**. Порог объявлен до
  прогонов и не двигается. Срабатывания стопа на дефолтах: 0.5 → 35 SL (25.5%), 0.7 → 21 (15.6%),
  **0.8 → 14 (10.4%)**, 1.0 → 9 (6.7%), 1.3 → 4, 2.0 → 0.
- **Гейт B:** max DD точки на полном 36-месячном окне ≤ **12 148.93 ₽ и ≤ 9.63%** (1.3 × дефолтов
  9 345.33 ₽ / 7.41%). Проверяются обе формы; пробита любая — гейт не пройден. Ничья в пользу риска:
  при разнице pooled OOS < 0.05 PF берётся вариант с меньшей просадкой.

**Семь пунктов стоп-условия** (§5.13 спеки). Круг останавливается, если точка даёт:

1. pooled OOS PF < 1.0 на 36/12/6;
2. меньше 20 сделок в пуле OOS 36/12/6;
3. pooled OOS PF < 1.0 на **24/12/3**;
4. pooled OOS PF < 1.0 на 36/12/6 при `-commission 0.001`;
5. хотя бы один из годов **2024, 2025, 2026** убыточен (net сделок, закрытых в году, ≤ 0). Год 2023
   не гейтится;
6. **хвост 6** (вход с 2026-03-25) с PF < 1.0, **или хвост 12** (вход с 2025-09-25) с PF < 1.0,
   **или** в хвосте 6 меньше пяти сделок. Дополнительно знак хвоста 6 (PF ≥ 1 или < 1) сверяется со
   знаком OOS PF фолда 4 одноточечного walk-forward 36/12/6. Расхождение — остановиться и доложить
   методику;
7. сделка, удержанная через любой из четырёх гэпов-прокси (гейт D), **или** вход в выходные / в
   часы метки 02–06, не устранённый уводом `FreshDayATR` на 0.

**Числа дефолтов** (полное окно, против них меряется всё)

| Показатель | Значение |
|---|---|
| Сделок | **137** |
| PF | **1.251** |
| Net | **+16 818.99 ₽** |
| Max DD | **9 345.33 ₽ (7.41%)** |
| Win rate | 68.61% |
| Expectancy | **+122.77 ₽** |
| Выходы | RSI 91 (66.4%), **SL 35 (25.5%)**, TP 11 (8.0%) |
| Удержание медиана / p90 / максимум | **8 / 21 / 38** баров |
| Ночёвок | **63 (46.0%)**, переносов через 2+ дня **2** |
| Выходная сессия | входов **0**, выходов **5** (+4 340.82 ₽) |
| Часы метки 02–06 | входов **0**, выходов **7** (+6 185.20 ₽) |
| Гейт D | **0** сделок через все четыре даты |

Календарные годы (по дате выхода):

| Год | Сделок | Net | PF |
|---|---|---|---|
| 2023 | 9 | −1 982.44 ₽ | 0.571 |
| 2024 | 50 | +16 704.84 ₽ | 2.124 |
| 2025 | 54 | +7 158.15 ₽ | 1.218 |
| **2026** | 24 | **−5 061.56 ₽** | 0.658 |

Хвосты: **хвост 6 — 15 сделок, −3 169.06 ₽, PF 0.678**; **хвост 12 — 40 сделок, −5 139.62 ₽,
PF 0.779**. При `-commission 0.001` на полном окне — 1.029.

**Скрипты протокола лежат в `reports/_analysis/`, в git не попадают** (`reports/` в `.gitignore`).
Не удалять и не перезаписывать:

- `rtkm_recon.py` — свойства инструмента;
- `rtkm_journal.py` — разбор журнала:
  - блоки: календарные годы, полугодия, анатомия выходов, удержание, ночёвки, выходы в выходные;
  - блок «ГЕЙТ C» (часы 02–06), блок «ГЕЙТ D» (четыре даты), блок «ПУНКТ 6» (хвосты по
    фиксированным датам), пять худших сделок;
  - запуск: `python3 reports/_analysis/rtkm_journal.py <отчёт>`;
- `sfin_sweep.py` — печать свипа из калибровочного отчёта; работает для любого тикера:
  `python3 reports/_analysis/sfin_sweep.py <ось[,ось2]> <отчёт>_calibration.md`.

**Коммиты.** Каждое сообщение коммита кончается строкой
`Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>`.

**Задачи прода (14–17) не выполняются, пока вердикт не вынесен** (§5.15 спеки).

## Review Focus

Пять режимов отказа, которые спека подразумевает, но которые легче всего пропустить. Каждый
закрыт тестом или шагом сверки в задаче-владельце.

1. **`env/prod.env` расходится с Go-списком и `*.example`.** Тикер тогда молча не торгует в проде —
   так было на SFIN и SMLT. Task 16 заводит тест `TestRSIPullbackTickersMatchEnvFiles`, читающий все
   три env-файла.
2. **Сетка, свипующая `EMASlow` без `EMAFast`** (арбитр `trend_stop`), при дефолтном `EMAFast` 10
   может породить вырожденную пару с узлом ниже 11. Task 1 тестом требует `EMASlow > 10` в таких
   файлах.
3. **Хвост 6 на единицах сделок, и его знак расходится с фолдом 4.** Task 11 Step 13 сверяет знак и
   считает пять сделок нижней границей.
4. **Забытый `-interval Minutes30` или «Фолдов: 3» на 36/12/6.** Task 2 Step 6 сверяет baseline
   числом, Task 3 Step 2 сверяет число фолдов.
5. **`cmd/pullparity` на окне длиннее 24 месяцев** даёт ложные расхождения (урок SFIN). Task 18
   запускает его с `-months 24`.

---

### Task 1: Каталог одиннадцати сеток со сторожевым тестом осей

**Files:**
- Create: `data/params/rsi_pullback/rtkm/cal_screen.json`, `cal_entry.json`, `cal_trend.json`,
  `cal_trend_low.json`, `cal_day.json`, `cal_volume.json`, `cal_vol_window.json`, `cal_risk.json`,
  `cal_exit.json`, `cal_trail.json`, `cal_trend_stop.json`
- Create: `internal/service/backtest/rsi_pullback_rtkm_grid_test.go`

**Interfaces:**
- Consumes: хелперы пакета `backtest` — `rsiPullbackTickerGrid(t, ticker, file string)
  map[string][]float64` (`rsi_pullback_grid_test.go:40`) и `containsFloat(values []float64, want
  float64) bool` (`rsi_pullback_cnru_grid_test.go:121`).
- Produces: одиннадцать путей `data/params/rsi_pullback/rtkm/cal_*.json` для задач 3–10; тест
  `TestRTKMGridsStayWide`; переменная `rtkmGridFiles` (её расширяет Task 12).

- [ ] **Step 1: Написать падающий сторожевой тест.**

```go
package backtest

import "testing"

// rtkmGridFiles перечисляет сетки RTKM ПОИМЁННО, а не обходом каталога: задачи плана кладут в тот
// же каталог файлы-точки (plateau_*.json, probe_*.json), которые законно фиксируют ось одним
// значением и упали бы на инвариантах осей ниже. Прецеденты — IRKT, MAGN, X5, SFIN.
var rtkmGridFiles = []string{
	"cal_screen.json",
	"cal_entry.json",
	"cal_trend.json",
	"cal_trend_low.json",
	"cal_day.json",
	"cal_volume.json",
	"cal_vol_window.json",
	"cal_risk.json",
	"cal_exit.json",
	"cal_trail.json",
	"cal_trend_stop.json",
}

// rtkmCoreEMAFast — дефолт ядра core.DefaultParams().EMAFast. Сетка, свипующая EMASlow без
// EMAFast, живёт на этом значении, и узел EMASlow <= 10 дал бы вырожденную пару.
const rtkmCoreEMAFast = 10

// TestRTKMGridsStayWide держит инварианты §5.1 спеки
// docs/superpowers/specs/2026-09-25-rtkm-rsi-pullback-prep-design.md. Владелец требует
// максимально широкие оси, поэтому тест запрещает УРЕЗАТЬ ось (проверяет обязательные узлы и
// запрещённые пары), но не запрещает лишних значений.
func TestRTKMGridsStayWide(t *testing.T) {
	for _, file := range rtkmGridFiles {
		grid := rsiPullbackTickerGrid(t, "rtkm", file)
		for _, v := range grid["RSILower"] {
			if v > 50 {
				t.Errorf("rtkm/%s: RSILower=%v нарушает RSILower <= 50", file, v)
			}
		}
		for _, v := range grid["RSIPeriod"] {
			if v < 2 {
				t.Errorf("rtkm/%s: RSIPeriod=%v нарушает RSIPeriod >= 2", file, v)
			}
		}
		for _, v := range grid["StopDailyATR"] {
			if v == 0 {
				t.Errorf("rtkm/%s: StopDailyATR=0 запрещён — стопless многодневная сделка", file)
			}
		}
		fast := grid["EMAFast"]
		if len(fast) == 0 {
			fast = []float64{rtkmCoreEMAFast}
		}
		for _, f := range fast {
			for _, s := range grid["EMASlow"] {
				if f >= s {
					t.Errorf("rtkm/%s: пара EMAFast=%v, EMASlow=%v нарушает EMAFast < EMASlow", file, f, s)
				}
			}
		}
	}

	mustHave := []struct {
		file, axis string
		want       []float64
	}{
		{"cal_entry.json", "RSILower", []float64{5, 50}},
		{"cal_trend_low.json", "EMASlow", []float64{12, 45}},
		{"cal_trend.json", "EMASlow", []float64{250}},
		{"cal_exit.json", "RSIUpper", []float64{35, 95}},
		{"cal_risk.json", "StopDailyATR", []float64{2.0}},
		{"cal_risk.json", "TPDailyATR", []float64{0.15, 2.5}},
		{"cal_day.json", "FreshDayATR", []float64{0, 0.05, 0.15, 0.5}},
		{"cal_day.json", "SpentDayATR", []float64{2.0}},
		{"cal_vol_window.json", "VolLookbackBars", []float64{32}},
	}
	for _, m := range mustHave {
		grid := rsiPullbackTickerGrid(t, "rtkm", m.file)
		for _, w := range m.want {
			if !containsFloat(grid[m.axis], w) {
				t.Errorf("rtkm/%s: ось %s потеряла обязательный узел %v (есть %v)", m.file, m.axis, w, grid[m.axis])
			}
		}
	}

	// Арбитр trend_stop меряет связку только внутри гейта A (потолок 0.8, §5.4 спеки).
	for _, v := range rsiPullbackTickerGrid(t, "rtkm", "cal_trend_stop.json")["StopDailyATR"] {
		if v > 0.8 {
			t.Errorf("rtkm/cal_trend_stop.json: StopDailyATR=%v выше потолка гейта A 0.8", v)
		}
	}
}
```

- [ ] **Step 2: Убедиться, что тест падает.**
  Run: `go test ./internal/service/backtest/ -run RTKMGridsStayWide -v`
  Expected: FAIL — файлов сеток ещё нет (`rsiPullbackPhases` не читает файл).

- [ ] **Step 3: Написать одиннадцать файлов сеток.** Формат (образец —
  `data/params/rsi_pullback/sfin/cal_entry.json`):

```json
{
  "_comment": "data/params/rsi_pullback/rtkm/cal_entry.json — тема entry для RTKM, 90 прогонов, поверх ДЕФОЛТОВ ЯДРА. <что меряет>. ЗАМЕР 2026-09-25 (36 мес, in-sample, §6.1 спеки): <числа оси>. ПРЕДУПРЕЖДЕНИЕ О КРАЯХ: <вырожденные узлы>. ЗАПУСК: go run ./cmd/backtest -ticker RTKM -strategy rsi_pullback -interval Minutes30 -calibrate data/params/rsi_pullback/rtkm/cal_entry.json -out ./reports/RTKM_entry -months 36 -train-months 12 -test-months 6 -min-trades 20 -metric profit_factor. РЕЗУЛЬТАТ ПРОГОНА: (заполняется задачей плана).",
  "phases": [
    {
      "name": "entry",
      "grid": {
        "RSIPeriod": [2, 3, 4, 5, 6, 7, 8, 10, 12],
        "RSILower": [5, 10, 15, 20, 25, 30, 35, 40, 45, 50]
      },
      "keepTop": 5
    }
  ]
}
```

  Угловые скобки в образце заполняются реальным текстом из §6 спеки — в файле их не остаётся.
  Оси всех файлов:

| Файл | Тема | Оси | Прогонов | `-min-trades` |
|---|---|---|---|---|
| `cal_screen.json` | `screen` | `UseDayATRGate` [0,1] × `UseVolume` [0,1] | 4 | 1 |
| `cal_entry.json` | `entry` | `RSIPeriod` [2,3,4,5,6,7,8,10,12] × `RSILower` [5,10,15,20,25,30,35,40,45,50] | 90 | 20 |
| `cal_trend.json` | `trend` | `EMAFast` [3,5,8,10,15,20,30,40] × `EMASlow` [50,75,100,150,200,250] | 48 | 20 |
| `cal_trend_low.json` | `trend_low` | `EMAFast` [3,5,8,10] × `EMASlow` [12,15,18,20,25,30,35,40,45] | 36 | 20 |
| `cal_day.json` | `day` | `FreshDayATR` [0,0.05,0.1,0.15,0.2,0.3,0.4,0.5] × `SpentDayATR` [0.5,0.6,0.7,0.8,0.9,1.0,1.1,1.25,1.5,2.0] | 80 | 20 |
| `cal_volume.json` | `volume` | `UseVolume` [1] × `VolMult` [1.0,1.2,1.5,2.0,2.5,3.0] × `VolBaseDays` [3,5,10,14,20,30] | 36 | 20 |
| `cal_vol_window.json` | `vol_window` | `UseVolume` [1] × `VolLookbackBars` [1,2,3,5,8,12,16,24,32] × `VolMult` [1.0,1.2,2.0] | 27 | 20 |
| `cal_risk.json` | `risk` | `StopDailyATR` [0.3,0.4,0.5,0.6,0.7,0.8,1.0,1.3,1.5,2.0] × `TPDailyATR` [0.15,0.2,0.25,0.3,0.4,0.5,0.6,0.8,1.0,1.5,2.5] | 110 | 20 |
| `cal_exit.json` | `exit` | `RSIUpper` [35,40,45,50,55,60,65,70,75,80,85,90,95] | 13 | 20 |
| `cal_trail.json` | `trail` | `UseRSIExit` [0,1] × `UseTrail` [1] × `TrailDailyATR` [0.3,0.4,0.5,0.7,1.0,1.5] | 12 | 20 |
| `cal_trend_stop.json` | `trend_stop` | `EMASlow` [12,15,20,30,50,100] × `StopDailyATR` [0.5,0.6,0.7,0.8] | 24 | 20 |

  **Что обязательно написать в `_comment` каждого файла:**
  - что тема меряет и сколько в ней прогонов;
  - замер из §6 спеки, из которого получена каждая ось;
  - предупреждения о краях, по файлам:
    - `cal_entry.json`: `RSILower` 5 — 6 сделок за окно; `RSIPeriod` 12 — 25 сделок;
    - `cal_risk.json`: узел `TPDailyATR` 2.5 — контрольная строка асимметрии, которую требует
      `TestRSIPullbackGridControlPoints`, кандидатом он не считается; стоп выше 0.8 режет гейт A;
    - `cal_day.json`: `SpentDayATR` выше 1.0 вырождает выборку (1.25 → 28 сделок, 2.0 → 5);
    - `cal_trend_stop.json`: тема — арбитр, а не источник полей (§5.2 спеки);
  - полную команду запуска с путём самого файла (`TestRSIPullbackCalFilesValid` требует, чтобы
    `_comment` содержал `rtkm/<имя файла>`);
  - место под результат прогона.

- [ ] **Step 4: Прогнать тесты.**
  Run: `go test ./internal/service/backtest/ -run 'RTKMGridsStayWide|RSIPullbackCalFilesValid|RSIPullbackGridControlPoints|RSIPullbackPointFilesArePoints' -v`
  Expected: PASS.

- [ ] **Step 5: Коммит** `feat(rsi_pullback): каталог сеток RTKM`.

---

### Task 2: Пакет `strategy/rtkm` в состоянии «калибровка не проводилась»

**Files:**
- Create: `internal/service/trading_strategy/rsi_pullback/strategy/rtkm/rtkm.go`
- Create: `internal/service/trading_strategy/rsi_pullback/strategy/rtkm/rtkm_test.go`
- Modify: `internal/service/backtest/rsi_pullback_registry.go` (импорт рядом со строкой 29, запись в
  карте рядом со строкой 93 — соседи `rtkmp`)
- Modify: `internal/service/backtest/rsi_pullback_registry_test.go`

**Interfaces:**
- Produces: `rtkm.Ticker` (константа `"RTKM"`) и `rtkm.DefaultParams() core.Params` — их читают
  задачи 14–16.

- [ ] **Step 1: Написать падающий тест пакета.**

```go
package rtkm

import (
	"testing"

	"tinvest/internal/service/trading_strategy/rsi_pullback/strategy/core"
)

// TestParamsTrackTheBaselineUntilCalibrated держит состояние «калибровка не проводилась»: до
// вердикта по спеке docs/superpowers/specs/2026-09-25-rtkm-rsi-pullback-prep-design.md пакет
// отдаёт ровно дефолты ядра. Тест заменяется снимком литерала (Task 14) или остаётся как протокол
// отказа (Task 13).
func TestParamsTrackTheBaselineUntilCalibrated(t *testing.T) {
	if got, want := DefaultParams(), core.DefaultParams(); got != want {
		t.Fatalf("до калибровки RTKM обязан отдавать baseline ядра:\n got: %+v\nwant: %+v", got, want)
	}
}

func TestTickerIsRTKM(t *testing.T) {
	if Ticker != "RTKM" {
		t.Fatalf("Ticker = %q, want RTKM", Ticker)
	}
}
```

- [ ] **Step 2: Убедиться, что тест падает.**
  Run: `go test ./internal/service/trading_strategy/rsi_pullback/strategy/rtkm/ -v`
  Expected: FAIL — пакета нет.

- [ ] **Step 3: Написать пакет.**

```go
// Package rtkm supplies the ticker and rsi_pullback Params for RTKM (ПАО «Ростелеком»,
// обыкновенные акции, лот 10).
//
// СОСТОЯНИЕ: калибровка не проводилась, параметры следуют дефолтам ядра. Процедура и критерии —
// docs/superpowers/specs/2026-09-25-rtkm-rsi-pullback-prep-design.md.
package rtkm

import "tinvest/internal/service/trading_strategy/rsi_pullback/strategy/core"

// Ticker is the instrument this package parameterises.
const Ticker = "RTKM"

// DefaultParams returns the rsi_pullback parameters for RTKM.
func DefaultParams() core.Params { return core.DefaultParams() }
```

- [ ] **Step 4: Зарегистрировать пакет в реестре бэктеста.** В
  `internal/service/backtest/rsi_pullback_registry.go`:
  - добавить импорт
    `rsipullbackrtkm "tinvest/internal/service/trading_strategy/rsi_pullback/strategy/rtkm"`;
  - добавить в карту строку
    `rsipullbackrtkm.Ticker: rsiPullbackBindingFor(rsipullbackrtkm.Ticker, rsipullbackrtkm.DefaultParams),`
    рядом со строкой `rsipullbackrtkmp`.

  Выравнивание столбцов карты поправит `gofmt`. В `rsi_pullback_registry_test.go` добавить тест:

```go
// TestRSIPullbackRTKMTracksBaseline сторожит ЧЕСТНОЕ состояние: RTKM заведён в реестр до
// калибровки, чтобы прогоны шли через реестр, а не через generic-ветку, и обязан возвращать ровно
// baseline ядра. Заменяется снимком литерала (Task 14 плана) при положительном вердикте.
func TestRSIPullbackRTKMTracksBaseline(t *testing.T) {
	b, ok := rsiPullbackRegistry[rsipullbackrtkm.Ticker]
	if !ok {
		t.Fatal("RTKM отсутствует в rsiPullbackRegistry: тикер провалится в generic-ветку")
	}
	p, pok := b.DefaultParams().(core.Params)
	if !pok {
		t.Fatalf("RTKM: DefaultParams() вернул %T, want core.Params", b.DefaultParams())
	}
	if p != core.DefaultParams() {
		t.Fatalf("RTKM ещё не откалиброван, params обязаны совпадать с baseline:\n got: %+v\nwant: %+v", p, core.DefaultParams())
	}
	if got := b.Build(p).Ticker(); got != "RTKM" {
		t.Fatalf("Ticker() = %q, want RTKM", got)
	}
}
```

- [ ] **Step 5: Прогнать тесты.**
  Run: `go test ./internal/service/trading_strategy/rsi_pullback/... ./internal/service/backtest/ -run 'RTKM|RSIPullback' -v`
  Expected: PASS.

- [ ] **Step 6: Проверить, что бэктест видит тикер, и сверить baseline.**

```bash
go run ./cmd/backtest -ticker RTKM -strategy rsi_pullback -interval Minutes30 \
  -months 36 -metric profit_factor -out ./reports/RTKM_base
python3 reports/_analysis/rtkm_journal.py reports/RTKM_base/RTKM_rsi_pullback_Minutes30_*.md
```

  Скрипт обязан выдать ровно числа дефолтов из Global Constraints:
  - сделок **137**, PF **1.251**, max DD **9 345.33 (7.41%)**;
  - годы **−1 982.44 / +16 704.84 / +7 158.15 / −5 061.56 ₽**;
  - выходы **RSI 91, SL 35, TP 11**, удержание **8/21/38**;
  - хвосты **0.678/15** и **0.779/40**;
  - гейт D — **ноль** сделок.

  Расхождение означает, что кэш перезаписан или запуск позже 2026-10-09 — остановиться и доложить.

- [ ] **Step 7: Коммит** `feat(rsi_pullback): пакет RTKM до калибровки`.

---

### Task 3: Тема `screen` — цена двух гейтов и сверка числа фолдов

**Files:** Modify `data/params/rsi_pullback/rtkm/cal_screen.json` (`_comment`); Create
`docs/superpowers/plans/task-3-report-rtkm.md`

**Interfaces:**
- Consumes: `cal_screen.json` из Task 1.
- Produces: голоса фолдов по `UseDayATRGate` и `UseVolume` — их читает Task 11.

- [ ] **Step 1: Прогнать тему.**

```bash
go run ./cmd/backtest -ticker RTKM -strategy rsi_pullback -interval Minutes30 \
  -calibrate data/params/rsi_pullback/rtkm/cal_screen.json -out ./reports/RTKM_screen \
  -months 36 -min-trades 1 -train-months 12 -test-months 6 -metric profit_factor
```

- [ ] **Step 2: Сверить число фолдов.** В шапке `*_walkforward.md` должно стоять «Фолдов: 4». Иначе
  **остановиться и доложить владельцу**.

- [ ] **Step 3: Записать** pooled OOS PF, размер пула и голоса всех четырёх фолдов по обеим осям.

- [ ] **Step 4: Сверить с ожиданием** (§6.3 спеки): **ожидается 1×0** — дневной гейт включён,
  объёмный выключен. Точечный замер: 1×0 → 1.251/137, 1×1 → 1.123/104, 0×0 → 0.935/472,
  0×1 → 0.914/273. Неподтверждённое ожидание записать прямым текстом.

- [ ] **Step 5: Написать отчёт** `docs/superpowers/plans/task-3-report-rtkm.md` (образец —
  `docs/superpowers/plans/task-3-report-sfin.md`) и дописать результат в `_comment` сетки строкой
  `РЕЗУЛЬТАТ ПРОГОНА <дата>: …` с pooled OOS, пулом, фолдами и голосами.

- [ ] **Step 6: Коммит** `docs(rsi_pullback): RTKM, тема screen`.

---

### Task 4: Тема `entry` — первая половина планки

**Files:** Modify `data/params/rsi_pullback/rtkm/cal_entry.json` (`_comment`); Create
`docs/superpowers/plans/task-4-report-rtkm.md`

**Interfaces:**
- Consumes: `cal_entry.json` из Task 1.
- Produces: голоса по `RSIPeriod` и `RSILower`, вердикт первой половины планки — их читает Task 11.

- [ ] **Step 1: Прогнать тему.**

```bash
go run ./cmd/backtest -ticker RTKM -strategy rsi_pullback -interval Minutes30 \
  -calibrate data/params/rsi_pullback/rtkm/cal_entry.json -out ./reports/RTKM_entry \
  -months 36 -min-trades 20 -train-months 12 -test-months 6 -metric profit_factor
```

- [ ] **Step 2: Записать** pooled OOS PF, пул и голоса четырёх фолдов (`RSILower` — ведущая ось).

- [ ] **Step 3: Проверить первую половину планки** (§5.12): pooled OOS ≥ **1.5** при ≥ **20**
  сделках **и** `RSILower` одинаков в ≥ 3 фолдах из 4. Вырожденный фолд в пользу тикера не
  засчитывается.

- [ ] **Step 4: Сверить с ожиданием:** **планку тема НЕ возьмёт.** Обе оси рваные (§6.1): `RSIPeriod`
  3 → 1.440, 4 → 1.251, 6 → 0.751; `RSILower` 20 → 0.902, 30 → 1.251, 45 → 1.321.

- [ ] **Step 5: Отметить вырожденные узлы, если фолды их выбрали:** `RSILower` 5 (6 сделок за окно),
  `RSIPeriod` 12 (25 сделок).

- [ ] **Step 6: Написать отчёт** `task-4-report-rtkm.md` и дописать результат в `_comment`.

- [ ] **Step 7: Коммит** `docs(rsi_pullback): RTKM, тема entry`.

---

### Task 5: Темы `trend` и `trend_low` — вторая половина планки

**Files:** Modify `data/params/rsi_pullback/rtkm/cal_trend.json`, `cal_trend_low.json`
(`_comment`); Create `docs/superpowers/plans/task-5-report-rtkm.md`

**Interfaces:**
- Consumes: оба файла из Task 1.
- Produces: голоса по `EMAFast` и `EMASlow` из обеих тем и их pooled OOS — их читают Task 10 и
  Task 11.

- [ ] **Step 1: Прогнать обе темы.**

```bash
go run ./cmd/backtest -ticker RTKM -strategy rsi_pullback -interval Minutes30 \
  -calibrate data/params/rsi_pullback/rtkm/cal_trend.json -out ./reports/RTKM_trend \
  -months 36 -min-trades 20 -train-months 12 -test-months 6 -metric profit_factor
go run ./cmd/backtest -ticker RTKM -strategy rsi_pullback -interval Minutes30 \
  -calibrate data/params/rsi_pullback/rtkm/cal_trend_low.json -out ./reports/RTKM_trend_low \
  -months 36 -min-trades 20 -train-months 12 -test-months 6 -metric profit_factor
```

- [ ] **Step 2: Записать по каждой теме** pooled OOS, пул и голоса четырёх фолдов (`EMASlow` —
  ведущая ось).

- [ ] **Step 3: Проверить вторую половину планки** по **канонической** `trend`, не по `trend_low`.

- [ ] **Step 4: Сверить с ожиданиями.**
  - `trend` планку не возьмёт: вся каноническая ось 50–250 лежит в 0.928–1.311, длинная EMA
    убыточна.
  - `trend_low` — главный рычаг бумаги, горб 12–20 (15 → 1.813/101). Риск записан заранее: голоса
    рассыплются по узлам горба, как в разведочном зонде (20/15/12/12).

- [ ] **Step 5: Записать, какая тема дала больший pooled OOS.** Поле тренда идёт из `trend_low` только
  при большинстве ≥ 3/4 **и** превосходстве над `trend` (§5.2).

- [ ] **Step 6: Написать отчёт** `task-5-report-rtkm.md` и дописать результаты в оба `_comment`.

- [ ] **Step 7: Коммит** `docs(rsi_pullback): RTKM, темы trend и trend_low`.

---

### Task 6: Тема `day`

**Files:** Modify `data/params/rsi_pullback/rtkm/cal_day.json` (`_comment`); Create
`docs/superpowers/plans/task-6-report-rtkm.md`

**Interfaces:**
- Consumes: `cal_day.json` из Task 1.
- Produces: голоса по `FreshDayATR` и `SpentDayATR` — их читает Task 11.

- [ ] **Step 1: Прогнать тему.**

```bash
go run ./cmd/backtest -ticker RTKM -strategy rsi_pullback -interval Minutes30 \
  -calibrate data/params/rsi_pullback/rtkm/cal_day.json -out ./reports/RTKM_day \
  -months 36 -min-trades 20 -train-months 12 -test-months 6 -metric profit_factor
```

- [ ] **Step 2: Записать** pooled OOS, пул и голоса четырёх фолдов по обеим осям.

- [ ] **Step 3: Записать предупреждение по `FreshDayATR`.** Ненулевое значение заводит вход на первый
  бар дня — это кандидат на провал запрета входов в выходные и часы 02–06 в Task 11 Step 6.
  Опорный замер: 0 → 1.251/137 (DD 9 345), **0.1 → 1.359/158 (DD 13 841)**, 0.2 → 1.305/170,
  0.4 → 0.995/280. Максимум оси растит просадку в полтора раза.

- [ ] **Step 4: Записать опорный замер `SpentDayATR`:** 0.7 → 1.079/184, **0.8 → 1.251/137
  (дефолт)**, 0.9 → 1.203/101, 1.0 → 1.359/66, 1.25 → 1.761/**28**, 1.5 → 1.528/**14**,
  2.0 → 0.843/**5**. Голос за узел выше 1.0 записать как подгонку под фолд на вырожденной выборке.

- [ ] **Step 5: Написать отчёт** `task-6-report-rtkm.md` и дописать результат в `_comment`.

- [ ] **Step 6: Коммит** `docs(rsi_pullback): RTKM, тема day`.

---

### Task 7: Темы `volume` и `vol_window`

**Files:** Modify `data/params/rsi_pullback/rtkm/cal_volume.json`, `cal_vol_window.json`
(`_comment`); Create `docs/superpowers/plans/task-7-report-rtkm.md`

**Interfaces:**
- Consumes: оба файла из Task 1.
- Produces: голоса по `VolMult`, `VolBaseDays`, `VolLookbackBars` и pooled OOS обеих тем — их читает
  Task 11.

- [ ] **Step 1: Прогнать обе темы.**

```bash
go run ./cmd/backtest -ticker RTKM -strategy rsi_pullback -interval Minutes30 \
  -calibrate data/params/rsi_pullback/rtkm/cal_volume.json -out ./reports/RTKM_volume \
  -months 36 -min-trades 20 -train-months 12 -test-months 6 -metric profit_factor
go run ./cmd/backtest -ticker RTKM -strategy rsi_pullback -interval Minutes30 \
  -calibrate data/params/rsi_pullback/rtkm/cal_vol_window.json -out ./reports/RTKM_vol_window \
  -months 36 -min-trades 20 -train-months 12 -test-months 6 -metric profit_factor
```

- [ ] **Step 2: Записать по каждой теме** pooled OOS, пул и голоса четырёх фолдов по всем осям.

- [ ] **Step 3: Сверить с ожиданием.** Обе темы идут с `UseVolume` 1 и **поле `UseVolume` не
  решают** — его решает `screen` (Task 3). Опорный замер: лучшие формы гейта — окно 16–24 бара
  (1.269–1.274), это в пределах шума дефолта 1.251 без гейта. **Ожидание: объёмный гейт в точку не
  войдёт**; тогда голоса этих тем пишутся справочно.

- [ ] **Step 4: Написать отчёт** `task-7-report-rtkm.md` и дописать результаты в оба `_comment`.

- [ ] **Step 5: Коммит** `docs(rsi_pullback): RTKM, темы volume и vol_window`.

---

### Task 8: Тема `risk` и потолок гейта A

**Files:** Modify `data/params/rsi_pullback/rtkm/cal_risk.json` (`_comment`); Create
`data/params/rsi_pullback/rtkm/probe_stop_10.json`, `probe_stop_20.json`; Create
`docs/superpowers/plans/task-8-report-rtkm.md`

**Interfaces:**
- Consumes: `cal_risk.json` из Task 1.
- Produces: голоса по `StopDailyATR` (уже пропущенные через потолок 0.8) и по `TPDailyATR` — их
  читают Task 10 и Task 11.

- [ ] **Step 1: Прогнать тему.**

```bash
go run ./cmd/backtest -ticker RTKM -strategy rsi_pullback -interval Minutes30 \
  -calibrate data/params/rsi_pullback/rtkm/cal_risk.json -out ./reports/RTKM_risk \
  -months 36 -min-trades 20 -train-months 12 -test-months 6 -metric profit_factor
```

- [ ] **Step 2: Записать** pooled OOS, пул и голоса четырёх фолдов по обеим осям.

- [ ] **Step 3: Применить потолок гейта A.** Голос выше **0.8** заменяется на 0.8, цена решения в PF
  записывается прямым текстом. Обоснование повторить в отчёте (§5.4 спеки): стоп 0.8 срабатывает в
  10.4% сделок, 1.0 — в 6.7%, 2.0 — ни разу. Разведочный зонд голосовал за 1.0 тремя фолдами из
  четырёх, поэтому замена почти наверняка понадобится.

- [ ] **Step 4: Снять два зонда капкана на полном окне.** Каждый файл — одна комбинация поверх
  дефолтов ядра:
  - `probe_stop_10.json`: `{"phases":[{"name":"probe","grid":{"StopDailyATR":[1.0]},"keepTop":1}],
    "_comment":"…"}`;
  - `probe_stop_20.json` — то же со `StopDailyATR` [2.0].

  Команда:

```bash
go run ./cmd/backtest -ticker RTKM -strategy rsi_pullback -interval Minutes30 \
  -calibrate data/params/rsi_pullback/rtkm/probe_stop_10.json -out ./reports/RTKM_probe_stop_10 \
  -months 36 -min-trades 1 -metric profit_factor
python3 reports/_analysis/rtkm_journal.py reports/RTKM_probe_stop_10/*_best.md
```

  То же повторить для `probe_stop_20`. По каждому зонду записать PF, число SL-выходов, обе формы
  DD и годы. Опорные числа: 1.0 → 1.694, 9 SL, DD 11 935 ₽ (7.81%), 2026 −7 285 ₽; 2.0 → 2.022,
  0 SL, 12 953 ₽ (7.85%), 2026 −8 083 ₽. **Главный вывод для отчёта: ни один стоп 2026 год не
  лечит.** Капкан поднимает PF за счёт 2024–2025.

- [ ] **Step 5: Записать опорный замер `TPDailyATR`:** плато 0.15–0.4 (1.263–1.311), сигнала у оси
  нет. Узлы 1.5 и 2.5 — контрольные строки.

- [ ] **Step 6: Написать отчёт** `task-8-report-rtkm.md` и дописать результат в `_comment`.

- [ ] **Step 7: Коммит** `docs(rsi_pullback): RTKM, тема risk и зонды капкана`.

---

### Task 9: Темы `exit` и `trail`

**Files:** Modify `data/params/rsi_pullback/rtkm/cal_exit.json`, `cal_trail.json` (`_comment`);
Create `docs/superpowers/plans/task-9-report-rtkm.md`

**Interfaces:**
- Consumes: оба файла из Task 1.
- Produces: голоса по `RSIUpper`, `UseRSIExit`, `UseTrail`, `TrailDailyATR` — их читает Task 11.

- [ ] **Step 1: Прогнать обе темы.**

```bash
go run ./cmd/backtest -ticker RTKM -strategy rsi_pullback -interval Minutes30 \
  -calibrate data/params/rsi_pullback/rtkm/cal_exit.json -out ./reports/RTKM_exit \
  -months 36 -min-trades 20 -train-months 12 -test-months 6 -metric profit_factor
go run ./cmd/backtest -ticker RTKM -strategy rsi_pullback -interval Minutes30 \
  -calibrate data/params/rsi_pullback/rtkm/cal_trail.json -out ./reports/RTKM_trail \
  -months 36 -min-trades 20 -train-months 12 -test-months 6 -metric profit_factor
```

- [ ] **Step 2: Записать** pooled OOS, пул и голоса четырёх фолдов по всем осям.

- [ ] **Step 3: Сверить с опорными замерами.**
  - `RSIUpper` двугорбая: **50 → 1.442/164**, 65 → 1.385, 70 → 1.251 (дефолт), **85 → 1.433/125**.
    Ожидается рассыпание голосов по двум горбам.
  - Трейл при `UseRSIExit` 1 инертен. Если тема голосует за `UseTrail` 1, гейт A применяется к
    `min(StopDailyATR, TrailDailyATR)` (урок AFKS).

- [ ] **Step 4: Написать отчёт** `task-9-report-rtkm.md` и дописать результаты в оба `_comment`.

- [ ] **Step 5: Коммит** `docs(rsi_pullback): RTKM, темы exit и trail`.

---

### Task 10: Тема `trend_stop` — арбитр связки

**Files:** Modify `data/params/rsi_pullback/rtkm/cal_trend_stop.json` (`_comment`); Create
`docs/superpowers/plans/task-10-report-rtkm.md`

**Interfaces:**
- Consumes: `cal_trend_stop.json` из Task 1; pooled OOS `trend_low` (Task 5) и `risk` (Task 8).
- Produces: вердикт «рычаги складываются / конкурируют» — его читает Task 11 Step 2.

- [ ] **Step 1: Прогнать тему.**

```bash
go run ./cmd/backtest -ticker RTKM -strategy rsi_pullback -interval Minutes30 \
  -calibrate data/params/rsi_pullback/rtkm/cal_trend_stop.json -out ./reports/RTKM_trend_stop \
  -months 36 -min-trades 20 -train-months 12 -test-months 6 -metric profit_factor
```

- [ ] **Step 2: Записать** pooled OOS, пул и голоса четырёх фолдов по обеим осям.

- [ ] **Step 3: Вынести вердикт арбитра** (§5.2 спеки).
  - pooled OOS темы **выше** и `trend_low`, и `risk` — рычаги **складываются**, оба поля идут в
    точку по своим темам.
  - Ниже любой из двух — рычаги **конкурируют**: в точку идёт один рычаг, тот, чья тема дала больший
    pooled OOS. Цена решения пишется прямым текстом.
  - Если за уход от дефолта голосует только одна из двух тем, арбитр справочный.

- [ ] **Step 4: Записать опорные числа разведки:** зонд с той же сеткой, но стопом до 1.0, дал
  36/12/6 → 2.516, фолды 27.039/20, 1.629/22, 1.496/19, **0.780/8**, голоса `EMASlow` 20/15/12/12.
  Точка 15 × 0.7 одноточечно: 2.792 / 24/12/3 → 1.525.

- [ ] **Step 5: Написать отчёт** `task-10-report-rtkm.md` и дописать результат в `_comment`.

- [ ] **Step 6: Коммит** `docs(rsi_pullback): RTKM, тема-арбитр trend_stop`.

---

### Task 11: Сборка точки, гейты, три walk-forward и семь пунктов стоп-условия

**Files:** Create `data/params/rsi_pullback/rtkm/plateau_point.json` и соседи плато
`plateau_<поле>_<значение>.json`; Create `docs/superpowers/plans/task-11-report-rtkm.md`

**Interfaces:**
- Consumes: голоса задач 3–10; скрипты `rtkm_journal.py`, `sfin_sweep.py`.
- Produces: точку первого круга (восемнадцать полей) и вердикт — их читают Task 12 и Task 14.

- [ ] **Step 1: Собрать точку правилом большинства** (§5.2 спеки):
  - поле берётся из темы, которая его меряет, только при ≥ 3 голосах из 4; иначе — дефолт ядра;
    ничья 2/2 большинством не считается;
  - `trend_low` берёт поле тренда только при превосходстве над `trend`;
  - `vol_window` берёт поля объёма только при превосходстве над `volume`;
  - `RSIUpper` — только из `exit`;
  - арбитр `trend_stop` применяется по вердикту Task 10.

  Записать таблицу для всех восемнадцати полей: «поле → тема-источник → голоса → принятое значение
  → дефолт ядра».

- [ ] **Step 2: Применить гейт A:** `min(StopDailyATR, TrailDailyATR при UseTrail = 1)` ≤ 0.8.

- [ ] **Step 3: Написать `plateau_point.json`** — одна фаза, все восемнадцать полей по одному
  значению (образец — `data/params/rsi_pullback/sfin/plateau_point.json`). Снять одиночный прогон:

```bash
go run ./cmd/backtest -ticker RTKM -strategy rsi_pullback -interval Minutes30 \
  -calibrate data/params/rsi_pullback/rtkm/plateau_point.json -out ./reports/RTKM_point \
  -months 36 -min-trades 1 -metric profit_factor
python3 reports/_analysis/rtkm_journal.py reports/RTKM_point/*_best.md
```

- [ ] **Step 4: Применить гейт B:** max DD ≤ **12 148.93 ₽ и ≤ 9.63%**, обе формы. При пробое в точку
  идёт ближайший вариант плато с меньшей просадкой, цена пишется прямым текстом.

- [ ] **Step 5: Гейт D:** из блока «ГЕЙТ D» — ноль сделок через все четыре даты. Иначе сработал
  пункт 7.

- [ ] **Step 6: Запрет входов в выходные и в часы 02–06:** из вывода скрипта — входов в выходные 0 и
  входов 02–06 (блок «ГЕЙТ C») 0. При провале увести `FreshDayATR` на 0 и пересчитать с Step 3. Если
  провал остался — сработал пункт 7.

- [ ] **Step 7: Снять соседей плато.** Для каждого поля, ушедшего от дефолта, — прогон самого
  значения и двух соседей по оси, команда как в Step 3.
  - Край сетки дополнительно проверяется зондом за краем.
  - **Для каждого соседа снять годы и хвосты** скриптом `rtkm_journal.py` (§5.3 спеки).
  - Разница pooled OOS < 0.05 решается в пользу меньшей просадки.
  - Плато уже 0.05 PF записывается как отсутствие сигнала.

- [ ] **Step 8: Третий контур.** Сравнить с дефолтами: SL 25.5%, удержание 8/21/38, ночёвок 46.0%,
  переносов 2, DD 7.41%, expectancy +122.77 ₽. **Падение доли SL вместе с ростом удержания и
  ночёвок — капкан**, тогда точка берёт более узкий стоп в пределах плато.

- [ ] **Step 9: Три walk-forward точки.**

```bash
go run ./cmd/backtest -ticker RTKM -strategy rsi_pullback -interval Minutes30 \
  -calibrate data/params/rsi_pullback/rtkm/plateau_point.json -out ./reports/RTKM_point_wf36 \
  -months 36 -min-trades 1 -train-months 12 -test-months 6 -metric profit_factor
go run ./cmd/backtest -ticker RTKM -strategy rsi_pullback -interval Minutes30 \
  -calibrate data/params/rsi_pullback/rtkm/plateau_point.json -out ./reports/RTKM_point_wf24 \
  -months 24 -min-trades 1 -train-months 12 -test-months 3 -metric profit_factor
go run ./cmd/backtest -ticker RTKM -strategy rsi_pullback -interval Minutes30 \
  -calibrate data/params/rsi_pullback/rtkm/plateau_point.json -out ./reports/RTKM_point_wf30 \
  -months 30 -min-trades 1 -train-months 12 -test-months 6 -metric profit_factor
```

  Ожидаемое число фолдов: **4, 4, 3**. Записать pooled OOS, пул и пофолдовые числа, пометить
  вырожденные фолды (меньше пяти сделок или без убыточных).

- [ ] **Step 10: Утяжелённые издержки.**

```bash
go run ./cmd/backtest -ticker RTKM -strategy rsi_pullback -interval Minutes30 \
  -calibrate data/params/rsi_pullback/rtkm/plateau_point.json -out ./reports/RTKM_point_c001 \
  -months 36 -min-trades 1 -train-months 12 -test-months 6 -metric profit_factor -commission 0.001
```

- [ ] **Step 11: Календарные годы.** Из блока «календарные годы» Step 3: 2024, 2025, 2026 — net > 0;
  2023 записать справочно.

- [ ] **Step 12: Хвосты.** Из блока «ПУНКТ 6» Step 3 записать хвост 6 и хвост 12 (сделок, net, PF) и
  долю крупнейшей сделки в net хвоста 6: найти в журнале отчёта сделку с максимальным |PnL| среди
  входов с 2026-03-25.

- [ ] **Step 13: Сверка знака хвоста 6 с фолдом 4.** Взять OOS PF фолда 4 из отчёта `RTKM_point_wf36`
  (тест-окно 2026-03-25 — 2026-09-25). Знаки (≥ 1 или < 1) обязаны совпасть с хвостом 6. Расхождение
  — остановиться и доложить владельцу вместе с обоими числами.

- [ ] **Step 14: Применить семь пунктов стоп-условия** (Global Constraints). По каждому пункту
  записать число и вердикт.

- [ ] **Step 15: Проверить планку** (Task 4 Step 3, Task 5 Step 3) и сверить с ожиданием «не
  возьмут обе».

- [ ] **Step 16: Написать отчёт** `task-11-report-rtkm.md`. Содержание:
  - таблица сборки;
  - гейты A, B, D и запрет входов с ценой решений;
  - соседи плато с годами и хвостами;
  - третий контур;
  - три walk-forward и издержки;
  - годы, хвосты и сверка знака;
  - семь пунктов и планка.

- [ ] **Step 17: Вердикт.**
  - Ни один пункт не сработал — переход к Task 14, задачи 12 и 13 пропускаются.
  - Иначе — переход к Task 12.

  **Ожидание, записанное до прогонов: второй круг вероятен** (§5.14 спеки).

- [ ] **Step 18: Коммит** `feat(rsi_pullback): RTKM, точка первого круга и вердикт`.

---

### Task 12: Второй круг (только при провале первого)

**Files:** Create `data/params/rsi_pullback/rtkm/cal2_<тема>.json` (не больше шести),
`plateau_point2.json`, соседи `plateau_r2_<поле>_<значение>.json`; Modify
`internal/service/backtest/rsi_pullback_rtkm_grid_test.go`; Create
`docs/superpowers/plans/task-12-report-rtkm.md`

**Задача выполняется ТОЛЬКО при срабатывании хотя бы одного пункта стоп-условия в Task 11.**

**Interfaces:**
- Consumes: голоса задач 3–10, вердикт Task 11, `rtkmGridFiles` из Task 1.
- Produces: точку второго круга — её читают Task 13 и Task 14.

- [ ] **Step 1: Выбрать зоны по фактическим голосам фолдов первого круга**, а не по in-sample рельефу.
  Шаг — половина шага первого круга. Не больше шести тем. В `_comment` каждого файла записать, из
  каких голосов какого фолда получена зона.
  - Кандидат номер один — `trend_low`, если голоса рассыпались по горбу 12–20: сетка `EMASlow`
    [12,13,14,15,16,17,18,20,22] × `EMAFast` [5,8,10].
  - Кандидат номер два — `risk` в зоне 0.6–0.8 с шагом 0.05.

- [ ] **Step 2: Расширить сторожевой тест.** В `rsi_pullback_rtkm_grid_test.go` добавить
  переменную `rtkmRound2GridFiles` с поимённым списком файлов `cal2_*.json`. Четыре жёстких
  инварианта первого цикла теста пустить по объединённому списку:

```go
func rtkmAllGridFiles() []string {
	out := make([]string, 0, len(rtkmGridFiles)+len(rtkmRound2GridFiles))
	out = append(out, rtkmGridFiles...)
	return append(out, rtkmRound2GridFiles...)
}
```

  Заменить `for _, file := range rtkmGridFiles` на `for _, file := range rtkmAllGridFiles()` в
  первом цикле `TestRTKMGridsStayWide`. Проверка стопа ≤ 0.8 распространяется на `cal2_risk.json`:
  узкая зона второго круга не выходит за потолок гейта A.

  Run: `go test ./internal/service/backtest/ -run 'RTKMGridsStayWide|RSIPullbackCalFilesValid|RSIPullbackGridControlPoints' -v`
  Expected: PASS.

- [ ] **Step 3: Прогнать каждую узкую тему.**

```bash
go run ./cmd/backtest -ticker RTKM -strategy rsi_pullback -interval Minutes30 \
  -calibrate data/params/rsi_pullback/rtkm/cal2_<тема>.json -out ./reports/RTKM_r2_<тема> \
  -months 36 -min-trades 1 -train-months 12 -test-months 6 -metric profit_factor
```

  `-min-trades 1` — число сделок не критерий (указание владельца).

- [ ] **Step 4: Собрать точку второго круга** тем же правилом ≥ 3/4 (Task 11 Step 1) и прогнать её
  через Task 11 Steps 2–13 без изменений, с файлами `plateau_point2.json` / `plateau_r2_*` и
  каталогами `RTKM_point2*`.

- [ ] **Step 5: Стоп-условие второго круга.**
  - Пункт 2 снят.
  - **Пул OOS меньше десяти сделок закрывает работу.**
  - Пункты 1, 3, 4, 5, 6, 7 — в силе полностью.

- [ ] **Step 6: Отчёт** `task-12-report-rtkm.md` той же структуры, что Task 11, плюс таблица «зона
  узкой сетки → голоса первого круга». Прогнать тесты:
  `go test ./internal/service/backtest/ -run 'RTKM|RSIPullback'`.

- [ ] **Step 7: Вердикт.** Точка прошла — переход к Task 14. Иначе — Task 13.

- [ ] **Step 8: Коммит** `feat(rsi_pullback): RTKM, второй круг и вердикт`.

---

### Task 13: Протокол отказа (только при провале обоих кругов)

**Files:** Modify `internal/service/trading_strategy/rsi_pullback/strategy/rtkm/rtkm.go`
(doc-comment), `rtkm_test.go` (комментарий теста); Create
`docs/superpowers/plans/task-13-report-rtkm.md`

**Задача выполняется ТОЛЬКО при провале обоих кругов. Задачи 14–17 не выполняются.**

- [ ] **Step 1: Написать разбор отказа в doc-comment пакета.** Образец —
  `internal/service/trading_strategy/rsi_pullback/strategy/rtkmp/rtkmp.go`. Содержание:
  - всё из §5.16 спеки;
  - какой пункт сработал в каждом круге и с какими числами;
  - что на бумаге этому причиной.

  В doc-comment теста `TestParamsTrackTheBaselineUntilCalibrated` записать окончательное состояние
  «калибровка закрыта отказом». Литерал не ставится.

- [ ] **Step 2: Полный гейт.** Run: `./bin/mage ci`. Expected: PASS.

- [ ] **Step 3: Доложить владельцу** числа обоих кругов и причину отказа.

- [ ] **Step 4: Коммит** `docs(rsi_pullback): RTKM, протокол отказа`.

---

### Task 14: Литерал в пакете и снимок

**Files:** Modify `internal/service/trading_strategy/rsi_pullback/strategy/rtkm/rtkm.go`,
`rtkm_test.go`, `internal/service/backtest/rsi_pullback_registry_test.go`

**Выполняется ТОЛЬКО при положительном вердикте Task 11 Step 17 или Task 12 Step 7.**

**Interfaces:**
- Consumes: принятую точку.
- Produces: `rtkm.DefaultParams()` возвращает литерал — его читают Task 15 и Task 16.

- [ ] **Step 1: Заменить тест пакета снимком.** Удалить `TestParamsTrackTheBaselineUntilCalibrated`,
  написать `TestParamsMatchTheCalibratedSnapshot`. Все восемнадцать полей выписаны литералами.
  Образец — `internal/service/trading_strategy/rsi_pullback/strategy/sfin/sfin_test.go`. Скелет, где
  значения берутся из принятой точки (таблица отчёта Task 11 или 12):

```go
func TestParamsMatchTheCalibratedSnapshot(t *testing.T) {
	want := core.Params{
		RSIPeriod: 4, RSILower: 30, RSIUpper: 70,
		EMAFast: 10, EMASlow: 15,
		DailyATRPeriod: 14, UseDayATRGate: 1, FreshDayATR: 0, SpentDayATR: 0.8,
		StopDailyATR: 0.7, TPDailyATR: 0.6,
		UseVolume: 0, VolBaseDays: 14, VolLookbackBars: 3, VolMult: 1.2,
		UseRSIExit: 1, UseTrail: 0, TrailDailyATR: 0,
	}
	if got := DefaultParams(); got != want {
		t.Fatalf("литерал RTKM разошёлся со снимком калибровки:\n got: %+v\nwant: %+v", got, want)
	}
}
```

  Значения в скелете — **пример формы**, взятый из разведки (§6.7 спеки). Перед записью они
  заменяются числами принятой точки поле за полем. Целочисленные поля `core.Params`
  (`core/core.go:30`): `RSIPeriod`, `EMAFast`, `EMASlow`, `DailyATRPeriod`, `UseDayATRGate`,
  `UseVolume`, `VolBaseDays`, `VolLookbackBars`, `UseRSIExit`, `UseTrail`; остальные — `float64`.

- [ ] **Step 2: Убедиться, что тест падает.**
  Run: `go test ./internal/service/trading_strategy/rsi_pullback/strategy/rtkm/ -v`
  Expected: FAIL.

- [ ] **Step 3: Поставить литерал.** `DefaultParams()` возвращает `core.Params` со всеми восемнадцатью
  полями **явно**, даже совпавшими с ядром.

- [ ] **Step 4: Обновить тест реестра.** `TestRSIPullbackRTKMTracksBaseline` заменить на
  `TestRSIPullbackRTKMIsRegisteredAndCalibrated` по образцу `TestRSIPullbackSFINIsRegisteredAndCalibrated`
  (`rsi_pullback_registry_test.go:1177`). Тест проверяет две вещи: реестр отдаёт
  `rsipullbackrtkm.DefaultParams()`, и это не `core.DefaultParams()`.

- [ ] **Step 5: Прогнать тесты.**
  Run: `go test ./internal/service/trading_strategy/rsi_pullback/... ./internal/service/backtest/ -run 'RTKM|RSIPullback' -v`
  Expected: PASS.

- [ ] **Step 6: Коммит** `feat(rsi_pullback): литерал RTKM`.

---

### Task 15: Реестр живого раннера

**Files:** Modify `internal/service/trading_strategy/rsi_pullback/live/registry.go`,
`internal/service/trading_strategy/rsi_pullback/live/registry_test.go`

**Interfaces:**
- Consumes: `rtkm.Ticker`, `rtkm.DefaultParams()` из Task 14.
- Produces: `ParamsFor("RTKM")` — его проверяет Task 18 через `cmd/pullparity`.

- [ ] **Step 1: Написать падающий тест** по образцу `TestRegistryHasSFIN`
  (`registry_test.go:214`):

```go
// TestRegistryHasRTKM держит связку «пакет — реестр живого раннера» для RTKM: раннер обязан
// торговать ровно литерал пакета.
func TestRegistryHasRTKM(t *testing.T) {
	p, ok := ParamsFor(rtkm.Ticker)
	if !ok {
		t.Fatal("RTKM нет в реестре живого раннера: тикер не будет торговать вовсе")
	}
	if want := rtkm.DefaultParams(); p != want {
		t.Fatalf("реестр отдаёт не литерал пакета:\n got: %+v\nwant: %+v", p, want)
	}
}
```

- [ ] **Step 2: Убедиться, что тест падает.**
  Run: `go test ./internal/service/trading_strategy/rsi_pullback/live/ -run RTKM -v`
  Expected: FAIL.

- [ ] **Step 3: Добавить в реестр.** Импорт
  `"tinvest/internal/service/trading_strategy/rsi_pullback/strategy/rtkm"` и строка
  `rtkm.Ticker: rtkm.DefaultParams(),` после `sfin.Ticker: sfin.DefaultParams(),` (`registry.go:712`).

- [ ] **Step 4: Прогнать тесты.**
  Run: `go test ./internal/service/trading_strategy/rsi_pullback/live/ -v`
  Expected: PASS.

- [ ] **Step 5: Коммит** `feat(rsi_pullback): RTKM в реестре живого раннера`.

---

### Task 16: Боевая вселенная и сторож env-файлов

**Files:** Modify `internal/config/rsi_pullback.go:605`, `internal/config/rsi_pullback_test.go:54`,
`env/prod.env:20`, `env/prod.env.example:30`, `env/local.env.example:28`

- [ ] **Step 1: Написать падающие тесты.**
  - Существующий тест в `rsi_pullback_test.go:54`: дописать `"RTKM"` последним в `want` и заменить
    сообщение `want 32` на `want 33: RTKM заведён тридцать третьим 2026-09-25`.
  - Новый тест в том же файле:

```go
// TestRSIPullbackTickersMatchEnvFiles сторожит, что боевой env/prod.env и оба образца несут ровно
// ту же вселенную, что Go-дефолт. Отслеживаемый env/prod.env дважды молча терял тикеры (SMLT,
// SFIN): правка только *.example оставляла прод без нового тикера.
func TestRSIPullbackTickersMatchEnvFiles(t *testing.T) {
	want := strings.Join(NewRSIPullbackConfig().Tickers, ",")
	for _, path := range []string{"../../env/prod.env", "../../env/prod.env.example", "../../env/local.env.example"} {
		raw, err := os.ReadFile(path) //nolint:gosec // fixed repository path
		if err != nil {
			t.Fatalf("read %s: %v", path, err)
		}
		var got string
		for _, line := range strings.Split(string(raw), "\n") {
			if v, ok := strings.CutPrefix(strings.TrimSpace(line), "RSI_PULLBACK_TICKERS="); ok {
				got = v
			}
		}
		if got != want {
			t.Errorf("%s: RSI_PULLBACK_TICKERS=%q, want %q (Go-дефолт internal/config/rsi_pullback.go)", path, got, want)
		}
	}
}
```

  `NewRSIPullbackConfig()` — тот же конструктор, что в `TestNewRSIPullbackConfig_Defaults`. Импорты
  `os` и `strings` добавить, если их ещё нет в файле.

- [ ] **Step 2: Убедиться, что оба теста падают.**
  Run: `go test ./internal/config/ -run RSIPullback -v`
  Expected: FAIL — `want 33` и расхождение env-файлов с дефолтом, где уже есть RTKM.

- [ ] **Step 3: Добавить `"RTKM"` последним элементом** во все четыре места сразу:
  - `internal/config/rsi_pullback.go:605`;
  - `env/prod.env:20`;
  - `env/prod.env.example:30`;
  - `env/local.env.example:28`.

  Сейчас все четыре кончаются на `SFIN`.

- [ ] **Step 4: Прогнать тесты.**
  Run: `go test ./internal/config/ -v`
  Expected: PASS.

- [ ] **Step 5: Коммит** `feat(rsi_pullback): завести RTKM в боевую вселенную`.

---

### Task 17: Дока пакета — разбор калибровки и принятый риск

**Files:** Modify `internal/service/trading_strategy/rsi_pullback/strategy/rtkm/rtkm.go`
(doc-comment)

- [ ] **Step 1: Написать разбор в doc-comment пакета.** Образец формы —
  `internal/service/trading_strategy/rsi_pullback/strategy/sfin/sfin.go`. Обязательное содержимое
  §5.16 спеки:
  - шаг цены 0.01 ₽ и реальный круг 0.051% против 0.234% у RTKMP;
  - таблица сборки точки;
  - годы 2023–2026 (2023 справочно);
  - хвосты 6 и 12 с долей крупнейшей сделки хвоста 6 и сверкой знака с фолдом 4;
  - таблица срабатываний стопа и решение по потолку 0.8 с ценой;
  - гейт B в обеих формах;
  - третий контур против дефолтов;
  - гейт D с оговоркой «даты — прокси, движок дивиденды не моделирует, ручной контроль владельца в
    дни отсечки»;
  - три walk-forward с вырожденными фолдами;
  - пул OOS;
  - семь пунктов и планка;
  - условие пересмотра после прода (§8 №1 спеки): хвост 6 месяцев с PF < 1.0 на ≥ 10 сделках —
    вывести из вселенной.

  **Правило CLAUDE.md: `docs/rsi_pullback/` не трогается.**

- [ ] **Step 2: Линтер.** Run: `./bin/mage lint`. Expected: PASS.

- [ ] **Step 3: Коммит** `docs(rsi_pullback): разбор калибровки RTKM и принятый риск`.

---

### Task 18: Финальная проверка

**Files:** нет (только прогоны), кроме правок по найденному.

- [ ] **Step 1: Полный гейт.** Run: `./bin/mage ci`. Expected: PASS (lint + `go test -race ./...` +
  дрейф моков).

- [ ] **Step 2: Сверка живой сборки с бэктестом.**

```bash
go run ./cmd/pullparity -tickers RTKM -months 24
```

  Флаг — **`-tickers`** (множественное число). Без него команда молча возьмёт `UGLD,T,GAZP`.
  **`-months 24`, не 36:** на более длинном окне сверка даёт ложные расхождения (урок SFIN).
  Expected: ноль расхождений.

- [ ] **Step 3: Сверить вселенную.** Run: `go test ./internal/config/ -run RSIPullbackTickersMatchEnvFiles -v`.
  Expected: PASS, в списке 33 тикера, RTKM последний.

- [ ] **Step 4: Доложить владельцу:**
  - точку;
  - три walk-forward;
  - издержки 0.2%;
  - годы;
  - хвосты и сверку знака;
  - гейты;
  - семь пунктов;
  - планку;
  - ручной контроль отсечек.

- [ ] **Step 5: Коммит** правок финальной проверки, если они были. Мерж делает владелец.

---

## Self-Review

**Покрытие спеки:**

| Раздел спеки | Задача |
|---|---|
| §1 инструмент, кэш, гэпы | Global Constraints; Task 2 Step 6 |
| §2 отношение к RTKMP | Task 17 Step 1 |
| §3 baseline | Global Constraints; Task 2 Step 6 |
| §4 схемы, ASTR, вырожденные фолды, дата запуска | Global Constraints; Task 3 Step 2; Task 11 Step 9 |
| §5.1 одиннадцать тем и инварианты | Task 1; задачи 3–10 |
| §5.2 сборка и арбитр | Task 10 Step 3; Task 11 Step 1 |
| §5.3 соседи плато с годами и хвостами | Task 11 Step 7 |
| §5.4 гейт A (0.8) | Task 8 Step 3; Task 11 Step 2 |
| §5.5 гейт B | Task 11 Step 4 |
| §5.6 третий контур, запрет входов | Task 11 Steps 6, 8 |
| §5.7 гейт D | Task 11 Step 5 |
| §5.8 скрипт журнала | Global Constraints |
| §5.9 календарные годы | Task 11 Step 11 |
| §5.10 хвосты и сверка знака | Task 11 Steps 12–13 |
| §5.11 схемы проверки | Task 11 Steps 9–10 |
| §5.12 планка | Task 4 Step 3; Task 5 Step 3; Task 11 Step 15 |
| §5.13 стоп-условие | Task 11 Step 14 |
| §5.14 второй круг | Task 12 |
| §5.15 правило прода | Task 11 Step 17; задачи 14–16 |
| §5.16 что записывается | Task 17; Task 13 (при отказе) |
| §6 свойства бумаги | опорные замеры в задачах 3–10 и `_comment` сеток |
| §7 артефакты | задачи 1, 2, 14–16 |
| §8 риски | Task 17 Step 1 |

**Плейсхолдеры.** Угловые скобки остались только в образцах, где исполнитель заполняет значения
из отчётов прогонов (`<тема>`, `<поле>`). Скелет литерала в Task 14 явно помечен как пример формы.

**Согласованность имён:**
- `rtkmGridFiles` — Task 1, расширяется в Task 12 через `rtkmAllGridFiles`;
- `rtkm.Ticker` и `rtkm.DefaultParams()` — Task 2, используются в задачах 14–16;
- `TestRSIPullbackRTKMTracksBaseline` — Task 2, заменяется в Task 14;
- `TestRSIPullbackTickersMatchEnvFiles` — Task 16, запускается в Task 18;
- номера тем в отчётах совпадают с номерами задач (`task-N-report-rtkm.md`).

**Развилки:**
- Task 11 Step 17: точка прошла — задачи 12 и 13 пропускаются;
- иначе Task 12; при её провале Task 13, и задачи 14–18 (кроме `mage ci` в Task 13) не выполняются;
- Task 10 Step 3: при конфликте рычагов в точку идёт один из двух.
