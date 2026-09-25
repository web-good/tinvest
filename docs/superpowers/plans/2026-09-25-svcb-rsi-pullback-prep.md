# SVCB под `rsi_pullback` — план подготовки, калибровки и вердикта

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development
> (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use
> checkbox (`- [ ]`) syntax for tracking.

**Goal:** Довести SVCB (ПАО «Совкомбанк», обыкновенные акции) до вердикта по стратегии
`rsi_pullback`. Нужно собрать максимально широкие сетки, прогнать тематический walk-forward и
получить точку внутри риск-гейтов A и B, которая проходит семь пунктов стоп-условия. Если первый
круг провален — второй круг по правилам владельца. Результат — либо литерал в пакете и заведение
в боевую вселенную тридцать третьим тикером, либо протокол отказа.

**Architecture:** Процедура каноническая (спека §5). Пять отличий от каталога.

1. **Главный критерий владельца вынесен в стоп-условие:** годы 2024, 2025 и 2026 в плюс.
2. **Жёсткий пункт устойчивости:** хвост 6 месяцев против хвоста 12 месяцев по дате входа.
3. **История короче окна:** IPO 2023-12-15, 33 месяца данных; train первого фолда 36/12/6 — около
   девяти месяцев.
4. **Гейт B — просадка не выше baseline** (а не 1.3×): дефолты убыточны и их просадка велика.
5. **Две темы-арбитра связок со стопом** — `entry_stop` и `day_stop`.

Разведка показала: дефолты убыточны (PF 0.951), проваливают 2024 и 2025 годы; свежий хвост
прибылен. Внутри потолка стопа 1.0 одна ось стопа 2024 год не лечит. Рабочая зона — глубокий вход
`RSILower` 15–25 вместе со стопом 0.7–0.8 (все три года в плюс). Голоса фолдов по входу рассыпаны
между глубоким и мелким горбом оси, поэтому второй круг вероятен.

**Tech Stack:** Go 1.25, `cmd/backtest` (rolling walk-forward), `cmd/pullparity` (сверка живой
сборки с бэктестом), `./bin/mage ci` (lint + `go test -race ./...` + дрейф моков), `python3` для
разбора журналов сделок.

**Spec:** `docs/superpowers/specs/2026-09-25-svcb-rsi-pullback-prep-design.md` — план ссылается на её
параграфы (§N), исполнитель читает оба документа.
**Замеры:** `reports/_analysis/svcb_pullback_prep_measurements.md`, сырые отчёты разведки —
`reports/SVCB_prep/`.

## Global Constraints

**Ветка, кэш и окно**

- **Ветка:** `feat/svcb-pullback-prep` от `main` `2b97332` (в боевой вселенной 32 тикера, последний —
  `SFIN`). Ветка создана, спека закоммичена (`ee2c112`).
- **Таймфрейм `Minutes30` во всех прогонах.** Флаг **`-interval Minutes30` обязателен в каждой
  команде `cmd/backtest`**. Дефолт CLI — `Hour1`, и забытый флаг даёт чужие числа, а не ошибку.
- **`-refresh` НЕ запускать.** Кэш дотянут `-refresh` 2026-09-25 14:09:
  `data/candles/SVCB_Minutes30.json` (история с 2023-12-15 17:00), `SVCB_Day1.json`,
  `SVCB_Minutes30.json.meta` (`earliestKnown` 2023-12-15).
- **Хвост кэша дотягивается при каждом прогоне** (`internal/service/backtest/candles.go:124` —
  `CandleProvider.Load` добирает бары от последнего кэшированного до момента запуска). Следствия:
  - окно `-months N` кончается моментом запуска; начало окна до 2026-12-15 лежит раньше начала
    истории, поэтому **сдвигается только конец**;
  - все сделки с входом до 2026-09-25 13:30 воспроизводятся точно; новые бары добавляют сделки в
    конец;
  - границы фолдов walk-forward отсчитываются от момента запуска и сдвигаются вместе с ним;
  - **все прогоны одной задачи делаются в один календарный день**; дата прогона пишется в отчёт
    задачи и в строку `РЕЗУЛЬТАТ ПРОГОНА` `_comment`.
- **Прогоны `cmd/backtest` — строго последовательно, никогда параллельно.** Параллельные запуски
  одновременно дописывают один и тот же файл кэша и роняют друг друга (в разведке 2026-09-25 из
  22 параллельных прогонов половина не дала отчёта).
- **Окно тем и точки — `-months 36`.** Контроли точки идут с `-months 30` и `-months 24`.

**Схемы прогонов**

- **Тематические прогоны:** `-months 36 -train-months 12 -test-months 6 -min-trades 20
  -metric profit_factor`. У темы `screen` — `-min-trades 1`. Все двенадцать тем идут только по этой
  схеме.
- **Три схемы проверки точки** (не тем):

  | Схема | Флаги | Фолдов | Роль | Дефолты (2026-09-25) |
  |---|---|---|---|---|
  | **36/12/6** | `-months 36 -train-months 12 -test-months 6` | 4 | сборка и вердикт | **0.949** / пул 96; фолды 0.977/37, 0.453/12, 1.197/27, 1.250/20 |
  | **24/12/3** | `-months 24 -train-months 12 -test-months 3` | 4 | **пункт 3 стоп-условия** | **1.218** / пул 47 |
  | 30/12/6 | `-months 30 -train-months 12 -test-months 6` | 3 | только записывается | 0.927 / пул 59 |

- **Вырожденный фолд** — меньше пяти сделок OOS или ни одной убыточной сделки. В пользу тикера он не
  засчитывается: ни как PF, ни как **голос в большинство 3/4** (§4 спеки). Вердикт по схеме
  выносится по pooled OOS, оговорка записывается.
- **Число фолдов сверяется на первой же теме:** в шапке отчёта темы `screen` должно стоять
  «Фолдов: 4». Иначе остановиться и доложить владельцу (ловушка ASTR).
- **Голос фолда 1 схемы 36/12/6** получен на train около девяти месяцев (история с 2023-12-15);
  оговорка пишется в каждый отчёт темы.

**Инструмент**

- **Лот 100, шаг цены 0.005 ₽.** Реальный круг двух шагов — **0.104%** при цене 9.57 ₽. Модель
  (`-commission 0.0005`, круг 0.1%) **нейтральна**. Пункт 4 стоп-условия (`-commission 0.001`,
  круг 0.2%) поэтому содержателен.
- **Гэпы-прокси отсечек и новостных ступеней** — пять дат: **2024-06-13 (−4.7%), 2024-07-08
  (−5.8%), 2024-11-15 (−2.0%), 2025-04-07 (−2.1%), 2026-07-11 (−2.9%)**. Они выписаны в
  `EX_DATES` скрипта `reports/_analysis/svcb_journal.py`, не менять.

**Сетки**

Сетки максимально широкие: обрезок нет, предупреждения о краях идут в `_comment`. Жёсткие
инварианты (§5.1 спеки):

- `RSILower ≤ 50`, `RSIPeriod ≥ 2`, `StopDailyATR` нигде не ноль;
- ни один файл не порождает пар `EMAFast ≥ EMASlow`;
- ось `RSIPeriod` в `cal_entry.json` содержит 2 и 14, ось `RSILower` — 5 и 50;
- ось `EMASlow` в `cal_trend_low.json` содержит 12 и 45; в `cal_trend.json` — 250;
- ось `RSIUpper` в `cal_exit.json` содержит 35 и 95;
- ось `StopDailyATR` в `cal_risk.json` содержит 0.3 и 2.0, ось `TPDailyATR` — 0.1 и 2.5;
- ось `FreshDayATR` в `cal_day.json` содержит 0, 0.05 и 0.5; ось `SpentDayATR` — 0.4 и 2.0;
- ось `VolLookbackBars` в `cal_vol_window.json` содержит 32;
- ось `StopDailyATR` в `cal_entry_stop.json` и `cal_day_stop.json` не выше 1.0.

**Риск-гейты**

- **Гейт A:** `min(StopDailyATR, TrailDailyATR при UseTrail = 1)` ≤ **1.0**. Порог объявлен до
  прогонов и не двигается. Срабатывания стопа на дефолтах: 0.5 → 43 SL (33.1%), 0.7 → 22 (17.6%),
  0.8 → 16 (12.8%), **1.0 → 13 (10.4%)**, 1.3 → 5 (4.0%), 1.5 → 1, 2.0 → 1.
- **Гейт B:** max DD точки на полном окне ≤ **14 848.91 ₽ и ≤ 14.06%** (не выше дефолтов; если
  baseline переснят в Task 2 Step 6 — переснятые числа). Проверяются обе формы; пробита любая —
  гейт не пройден. Ничья в пользу риска: при разнице pooled OOS < 0.05 PF берётся вариант с меньшей
  просадкой; при разнице просадок < 0.5 п.п. — с меньшим числом убыточных полугодий.

**Семь пунктов стоп-условия** (§5.13 спеки). Круг останавливается, если точка даёт:

1. pooled OOS PF < 1.0 на 36/12/6;
2. меньше 20 сделок в пуле OOS 36/12/6;
3. pooled OOS PF < 1.0 на **24/12/3**;
4. pooled OOS PF < 1.0 на 36/12/6 при `-commission 0.001`;
5. хотя бы один из годов **2024, 2025, 2026** убыточен (net сделок, закрытых в году, ≤ 0). Год 2023
   не гейтится (сделок нет);
6. **хвост 6** (вход с 2026-03-25) с PF < 1.0, **или хвост 12** (вход с 2025-09-25) с PF < 1.0,
   **или** в хвосте 6 меньше пяти сделок. Дополнительно знак хвоста 6 (PF ≥ 1 или < 1) сверяется со
   знаком OOS PF фолда 4 одноточечного walk-forward 36/12/6. Расхождение — остановиться и доложить
   методику;
7. сделка, удержанная через любой из пяти гэпов-прокси (гейт D), **или** вход в выходные / в
   часы метки 02–06, не устранённый уводом `FreshDayATR` на 0.

**Числа дефолтов** (полное окно на 2026-09-25 14:09, против них меряется всё)

| Показатель | Значение |
|---|---|
| Сделок | **130** |
| PF | **0.951** |
| Net | **−3 663.14 ₽** |
| Max DD | **14 848.91 ₽ (14.06%)** |
| Win rate | 61.54% |
| Expectancy | **−28.18 ₽** |
| Выходы | RSI 74 (56.9%), **SL 43 (33.1%)**, TP 13 (10.0%) |
| Удержание медиана / p90 / максимум | **8 / 16 / 58** баров |
| Ночёвок | **50 (38.5%)**, переносов через 2+ дня **1** |
| Выходная сессия | входов **0**, выходов **6** (+340.59 ₽) |
| Часы метки 02–06 | входов **0**, выходов **6** (+4 350.77 ₽) |
| Гейт D | **0** сделок через все пять дат |

Календарные годы (по дате выхода):

| Год | Сделок | Net | PF |
|---|---|---|---|
| **2024** | 48 | **−3 986.48 ₽** | 0.874 |
| **2025** | 47 | **−2 284.90 ₽** | 0.923 |
| 2026 | 35 | +2 608.28 ₽ | 1.190 |

Хвосты: **хвост 6 — 20 сделок, +2 230.64 ₽, PF 1.247**; **хвост 12 — 47 сделок, +4 096.18 ₽,
PF 1.220**. При `-commission 0.001` на полном окне — 0.798.

**Скрипты протокола лежат в `reports/_analysis/`, в git не попадают** (`reports/` в `.gitignore`).
Не удалять и не перезаписывать:

- `svcb_recon.py` — свойства инструмента;
- `svcb_journal.py` — разбор журнала:
  - блоки: календарные годы, полугодия, анатомия выходов, удержание, ночёвки, выходы в выходные;
  - блок «ГЕЙТ C» (часы 02–06), блок «ГЕЙТ D» (пять дат), блок «ПУНКТ 6» (хвосты по
    фиксированным датам), пять худших сделок;
  - запуск: `python3 reports/_analysis/svcb_journal.py <отчёт>`;
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
   три env-файла (в репозитории его ещё нет: у RTKM задача прода не выполнялась).
2. **Скользящий конец окна.** Прогон в другой день даёт другие числа baseline, и исполнитель примет
   это за порчу кэша или, наоборот, пропустит настоящую порчу. Task 2 Step 6 сверяет **префикс**
   журнала (сделки с входом до 2026-09-25 13:30) со спекой побайтово по числу и net, а новые сделки
   записывает отдельно.
3. **Параллельные прогоны портят кэш.** Global Constraints запрещают параллельность; Task 3 Step 1
   проверяет, что после прогона кэш читается и в шапке «Фолдов: 4».
4. **Голос вырожденного фолда засчитан в большинство.** Правило новое для процедуры (решение RTKM
   поднято в спеку). Task 11 Step 1 требует в таблице сборки столбец «вырожденные фолды» и
   пересчёт большинства без них.
5. **Хвост 6 на единицах сделок, и его знак расходится с фолдом 4** (границы фолда 4 сдвинуты датой
   запуска). Task 11 Step 13 сверяет знак и записывает фактическое тест-окно фолда 4.

---

### Task 1: Каталог двенадцати сеток со сторожевым тестом осей

**Files:**
- Create: `data/params/rsi_pullback/svcb/cal_screen.json`, `cal_entry.json`, `cal_trend.json`,
  `cal_trend_low.json`, `cal_day.json`, `cal_volume.json`, `cal_vol_window.json`, `cal_risk.json`,
  `cal_exit.json`, `cal_trail.json`, `cal_entry_stop.json`, `cal_day_stop.json`
- Create: `internal/service/backtest/rsi_pullback_svcb_grid_test.go`

**Interfaces:**
- Consumes: хелперы пакета `backtest` — `rsiPullbackTickerGrid(t, ticker, file string)
  map[string][]float64` (`rsi_pullback_grid_test.go:40`) и `containsFloat(values []float64, want
  float64) bool` (`rsi_pullback_cnru_grid_test.go:121`).
- Produces: двенадцать путей `data/params/rsi_pullback/svcb/cal_*.json` для задач 3–10; тест
  `TestSVCBGridsStayWide`; переменная `svcbGridFiles` (её расширяет Task 12).

- [ ] **Step 1: Написать падающий сторожевой тест.**

```go
package backtest

import "testing"

// svcbGridFiles перечисляет сетки SVCB ПОИМЁННО, а не обходом каталога: задачи плана кладут в тот
// же каталог файлы-точки (plateau_*.json, probe_*.json), которые законно фиксируют ось одним
// значением и упали бы на инвариантах осей ниже. Прецеденты — IRKT, MAGN, X5, SFIN, RTKM.
var svcbGridFiles = []string{
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
	"cal_entry_stop.json",
	"cal_day_stop.json",
}

// svcbCoreEMAFast — дефолт ядра core.DefaultParams().EMAFast. Сетка, свипующая EMASlow без
// EMAFast, живёт на этом значении, и узел EMASlow <= 10 дал бы вырожденную пару.
const svcbCoreEMAFast = 10

// svcbStopCeiling — потолок риск-гейта A (§5.4 спеки): оба правила каталога, выживаемость 30% и
// частота срабатывания не реже 10% сделок, дают на SVCB одно значение.
const svcbStopCeiling = 1.0

// TestSVCBGridsStayWide держит инварианты §5.1 спеки
// docs/superpowers/specs/2026-09-25-svcb-rsi-pullback-prep-design.md. Владелец требует
// максимально широкие оси, поэтому тест запрещает УРЕЗАТЬ ось (проверяет обязательные узлы и
// запрещённые пары), но не запрещает лишних значений.
func TestSVCBGridsStayWide(t *testing.T) {
	for _, file := range svcbGridFiles {
		grid := rsiPullbackTickerGrid(t, "svcb", file)
		for _, v := range grid["RSILower"] {
			if v > 50 {
				t.Errorf("svcb/%s: RSILower=%v нарушает RSILower <= 50", file, v)
			}
		}
		for _, v := range grid["RSIPeriod"] {
			if v < 2 {
				t.Errorf("svcb/%s: RSIPeriod=%v нарушает RSIPeriod >= 2", file, v)
			}
		}
		for _, v := range grid["StopDailyATR"] {
			if v == 0 {
				t.Errorf("svcb/%s: StopDailyATR=0 запрещён — стопless многодневная сделка", file)
			}
		}
		fast := grid["EMAFast"]
		if len(fast) == 0 {
			fast = []float64{svcbCoreEMAFast}
		}
		for _, f := range fast {
			for _, s := range grid["EMASlow"] {
				if f >= s {
					t.Errorf("svcb/%s: пара EMAFast=%v, EMASlow=%v нарушает EMAFast < EMASlow", file, f, s)
				}
			}
		}
	}

	mustHave := []struct {
		file, axis string
		want       []float64
	}{
		{"cal_entry.json", "RSIPeriod", []float64{2, 14}},
		{"cal_entry.json", "RSILower", []float64{5, 50}},
		{"cal_trend_low.json", "EMASlow", []float64{12, 45}},
		{"cal_trend.json", "EMASlow", []float64{250}},
		{"cal_exit.json", "RSIUpper", []float64{35, 95}},
		{"cal_risk.json", "StopDailyATR", []float64{0.3, 2.0}},
		{"cal_risk.json", "TPDailyATR", []float64{0.1, 2.5}},
		{"cal_day.json", "FreshDayATR", []float64{0, 0.05, 0.5}},
		{"cal_day.json", "SpentDayATR", []float64{0.4, 2.0}},
		{"cal_vol_window.json", "VolLookbackBars", []float64{32}},
	}
	for _, m := range mustHave {
		grid := rsiPullbackTickerGrid(t, "svcb", m.file)
		for _, w := range m.want {
			if !containsFloat(grid[m.axis], w) {
				t.Errorf("svcb/%s: ось %s потеряла обязательный узел %v (есть %v)", m.file, m.axis, w, grid[m.axis])
			}
		}
	}

	// Арбитры меряют связку «ось × стоп» только внутри гейта A (§5.4 спеки).
	for _, file := range []string{"cal_entry_stop.json", "cal_day_stop.json"} {
		for _, v := range rsiPullbackTickerGrid(t, "svcb", file)["StopDailyATR"] {
			if v > svcbStopCeiling {
				t.Errorf("svcb/%s: StopDailyATR=%v выше потолка гейта A %v", file, v, svcbStopCeiling)
			}
		}
	}
}
```

- [ ] **Step 2: Убедиться, что тест падает.**
  Run: `go test ./internal/service/backtest/ -run SVCBGridsStayWide -v`
  Expected: FAIL — файлов сеток ещё нет.

- [ ] **Step 3: Написать двенадцать файлов сеток.** Формат (образец —
  `data/params/rsi_pullback/rtkm/cal_entry.json`):

```json
{
  "_comment": "data/params/rsi_pullback/svcb/cal_entry.json — тема entry для SVCB, 100 прогонов, поверх ДЕФОЛТОВ ЯДРА. <что меряет>. ЗАМЕР 2026-09-25 (окно -months 36, фактически 33 мес с IPO 2023-12-15, in-sample, §6.1 спеки docs/superpowers/specs/2026-09-25-svcb-rsi-pullback-prep-design.md): <числа оси>. ПРЕДУПРЕЖДЕНИЕ О КРАЯХ: <вырожденные узлы>. ЗАПУСК: go run ./cmd/backtest -ticker SVCB -strategy rsi_pullback -interval Minutes30 -calibrate data/params/rsi_pullback/svcb/cal_entry.json -out ./reports/SVCB_entry -months 36 -train-months 12 -test-months 6 -min-trades 20 -metric profit_factor. РЕЗУЛЬТАТ ПРОГОНА: (заполняется задачей плана).",
  "phases": [
    {
      "name": "entry",
      "grid": {
        "RSIPeriod": [2, 3, 4, 5, 6, 7, 8, 10, 12, 14],
        "RSILower": [5, 10, 15, 20, 25, 30, 35, 40, 45, 50]
      },
      "keepTop": 5
    }
  ]
}
```

  Угловые скобки в образце заполняются реальным текстом из §6 спеки — в файле их не остаётся.
  Имя фазы = имя темы. Оси всех файлов:

| Файл | Тема | Оси | Прогонов | `-min-trades` |
|---|---|---|---|---|
| `cal_screen.json` | `screen` | `UseDayATRGate` [0,1] × `UseVolume` [0,1] | 4 | 1 |
| `cal_entry.json` | `entry` | `RSIPeriod` [2,3,4,5,6,7,8,10,12,14] × `RSILower` [5,10,15,20,25,30,35,40,45,50] | 100 | 20 |
| `cal_trend.json` | `trend` | `EMAFast` [3,5,8,10,15,20,30,40] × `EMASlow` [50,75,100,150,200,250] | 48 | 20 |
| `cal_trend_low.json` | `trend_low` | `EMAFast` [3,5,8,10] × `EMASlow` [12,15,20,25,30,35,40,45] | 32 | 20 |
| `cal_day.json` | `day` | `FreshDayATR` [0,0.05,0.1,0.15,0.2,0.3,0.4,0.5] × `SpentDayATR` [0.4,0.5,0.6,0.7,0.8,0.9,1.0,1.25,1.5,2.0] | 80 | 20 |
| `cal_volume.json` | `volume` | `UseVolume` [1] × `VolMult` [1.0,1.2,1.5,2.0,2.5,3.0] × `VolBaseDays` [3,5,10,14,20,30] | 36 | 20 |
| `cal_vol_window.json` | `vol_window` | `UseVolume` [1] × `VolLookbackBars` [1,2,3,5,8,12,16,24,32] × `VolMult` [1.0,1.2,2.0] | 27 | 20 |
| `cal_risk.json` | `risk` | `StopDailyATR` [0.3,0.4,0.5,0.6,0.7,0.8,1.0,1.3,1.5,2.0] × `TPDailyATR` [0.1,0.15,0.2,0.25,0.3,0.4,0.5,0.6,0.8,1.0,1.5,2.5] | 120 | 20 |
| `cal_exit.json` | `exit` | `RSIUpper` [35,40,45,50,55,60,65,70,75,80,85,90,95] | 13 | 20 |
| `cal_trail.json` | `trail` | `UseRSIExit` [0,1] × `UseTrail` [1] × `TrailDailyATR` [0.3,0.4,0.5,0.7,1.0,1.5] | 12 | 20 |
| `cal_entry_stop.json` | `entry_stop` | `RSILower` [10,15,20,25,30,35,40,45,50] × `StopDailyATR` [0.5,0.6,0.7,0.8,1.0] | 45 | 20 |
| `cal_day_stop.json` | `day_stop` | `SpentDayATR` [0.5,0.6,0.7,0.8,0.9,1.0] × `StopDailyATR` [0.5,0.6,0.7,0.8,1.0] | 30 | 20 |

  `keepTop` — 5 во всех файлах, кроме `cal_screen.json` (4) и `cal_exit.json` (5).

  **Что обязательно написать в `_comment` каждого файла:**
  - что тема меряет и сколько в ней прогонов;
  - замер из §6 спеки, из которого получена каждая ось;
  - оговорку: история с 2023-12-15, train фолда 1 около девяти месяцев;
  - предупреждения о краях, по файлам:
    - `cal_entry.json`: `RSILower` 5 — 6 сделок за окно; `RSIPeriod` 14 — 11 сделок, 12 — 27;
      ось `RSILower` двугорбая (15 → 1.937, 45–50 → 1.30–1.35), дефолт 30 в провале;
    - `cal_trend_low.json`: короткая `EMASlow` чинит 2024 год, но ломает 2026 (15 → хвост 6
      0.353/9); тема нужна, чтобы правило §5.2 решало вопрос по pooled OOS против `trend`;
    - `cal_risk.json`: узел `TPDailyATR` 2.5 — контрольная строка асимметрии, которую требует
      `TestRSIPullbackGridControlPoints`, кандидатом он не считается (1.5 и 2.5 побайтово равны);
      стоп выше 1.0 режет гейт A; узел цели 0.1 убыточен (0.612/154);
    - `cal_day.json`: `FreshDayATR` вредна на всей оси; `SpentDayATR` 2.0 — 6 сделок за окно;
    - `cal_entry_stop.json`, `cal_day_stop.json`: тема — арбитр, а не источник полей (§5.2 спеки),
      стоп не выше потолка 1.0;
  - полную команду запуска с путём самого файла (`TestRSIPullbackCalFilesValid` требует, чтобы
    `_comment` содержал `svcb/<имя файла>`);
  - место под результат прогона.

- [ ] **Step 4: Прогнать тесты.**
  Run: `go test ./internal/service/backtest/ -run 'SVCBGridsStayWide|RSIPullbackCalFilesValid|RSIPullbackGridControlPoints|RSIPullbackPointFilesArePoints' -v`
  Expected: PASS.

- [ ] **Step 5: Коммит** `feat(rsi_pullback): каталог сеток SVCB`.

---

### Task 2: Пакет `strategy/svcb` в состоянии «калибровка не проводилась»

**Files:**
- Create: `internal/service/trading_strategy/rsi_pullback/strategy/svcb/svcb.go`
- Create: `internal/service/trading_strategy/rsi_pullback/strategy/svcb/svcb_test.go`
- Modify: `internal/service/backtest/rsi_pullback_registry.go` (импорт в блоке рядом со строкой 31 —
  сосед `rsipullbacksfin`; запись в карте рядом со строкой 96)
- Modify: `internal/service/backtest/rsi_pullback_registry_test.go`
- Create: `docs/superpowers/plans/task-2-report-svcb.md`

**Interfaces:**
- Produces: `svcb.Ticker` (константа `"SVCB"`) и `svcb.DefaultParams() core.Params` — их читают
  задачи 14–16.

- [ ] **Step 1: Написать падающий тест пакета.**

```go
package svcb

import (
	"testing"

	"tinvest/internal/service/trading_strategy/rsi_pullback/strategy/core"
)

// TestParamsTrackTheBaselineUntilCalibrated держит состояние «калибровка не проводилась»: до
// вердикта по спеке docs/superpowers/specs/2026-09-25-svcb-rsi-pullback-prep-design.md пакет
// отдаёт ровно дефолты ядра. Тест заменяется снимком литерала (Task 14) или остаётся как протокол
// отказа (Task 13).
func TestParamsTrackTheBaselineUntilCalibrated(t *testing.T) {
	if got, want := DefaultParams(), core.DefaultParams(); got != want {
		t.Fatalf("до калибровки SVCB обязан отдавать baseline ядра:\n got: %+v\nwant: %+v", got, want)
	}
}

func TestTickerIsSVCB(t *testing.T) {
	if Ticker != "SVCB" {
		t.Fatalf("Ticker = %q, want SVCB", Ticker)
	}
}
```

- [ ] **Step 2: Убедиться, что тест падает.**
  Run: `go test ./internal/service/trading_strategy/rsi_pullback/strategy/svcb/ -v`
  Expected: FAIL — пакета нет.

- [ ] **Step 3: Написать пакет.**

```go
// Package svcb supplies the ticker and rsi_pullback Params for SVCB (ПАО «Совкомбанк»,
// обыкновенные акции, лот 100).
//
// СОСТОЯНИЕ: калибровка не проводилась, параметры следуют дефолтам ядра. Процедура и критерии —
// docs/superpowers/specs/2026-09-25-svcb-rsi-pullback-prep-design.md.
package svcb

import "tinvest/internal/service/trading_strategy/rsi_pullback/strategy/core"

// Ticker is the instrument this package parameterises.
const Ticker = "SVCB"

// DefaultParams returns the rsi_pullback parameters for SVCB.
func DefaultParams() core.Params { return core.DefaultParams() }
```

- [ ] **Step 4: Зарегистрировать пакет в реестре бэктеста.** В
  `internal/service/backtest/rsi_pullback_registry.go`:
  - добавить импорт
    `rsipullbacksvcb "tinvest/internal/service/trading_strategy/rsi_pullback/strategy/svcb"`
    в алфавитном порядке блока импортов;
  - добавить в карту строку
    `rsipullbacksvcb.Ticker: rsiPullbackBindingFor(rsipullbacksvcb.Ticker, rsipullbacksvcb.DefaultParams),`
    после строки `rsipullbacksfin`.

  Выравнивание столбцов карты поправит `gofmt -w`. В `rsi_pullback_registry_test.go` добавить тест
  рядом с `TestRSIPullbackRTKMTracksBaseline` (строка 1028):

```go
// TestRSIPullbackSVCBTracksBaseline сторожит ЧЕСТНОЕ состояние: SVCB заведён в реестр до
// калибровки, чтобы прогоны шли через реестр, а не через generic-ветку, и обязан возвращать ровно
// baseline ядра. Заменяется снимком литерала (Task 14 плана) при положительном вердикте.
func TestRSIPullbackSVCBTracksBaseline(t *testing.T) {
	b, ok := rsiPullbackRegistry[rsipullbacksvcb.Ticker]
	if !ok {
		t.Fatal("SVCB отсутствует в rsiPullbackRegistry: тикер провалится в generic-ветку")
	}
	p, pok := b.DefaultParams().(core.Params)
	if !pok {
		t.Fatalf("SVCB: DefaultParams() вернул %T, want core.Params", b.DefaultParams())
	}
	if p != core.DefaultParams() {
		t.Fatalf("SVCB ещё не откалиброван, params обязаны совпадать с baseline:\n got: %+v\nwant: %+v", p, core.DefaultParams())
	}
	if got := b.Build(p).Ticker(); got != "SVCB" {
		t.Fatalf("Ticker() = %q, want SVCB", got)
	}
}
```

  Импорт `rsipullbacksvcb` добавить в тестовый файл тем же путём.

- [ ] **Step 5: Прогнать тесты.**
  Run: `go test ./internal/service/trading_strategy/rsi_pullback/... ./internal/service/backtest/ -run 'SVCB|RSIPullback' -v`
  Expected: PASS.

- [ ] **Step 6: Переснять baseline на дату исполнения и сверить префикс.**

```bash
go run ./cmd/backtest -ticker SVCB -strategy rsi_pullback -interval Minutes30 \
  -months 36 -metric profit_factor -out ./reports/SVCB_base
python3 reports/_analysis/svcb_journal.py reports/SVCB_base/SVCB_rsi_pullback_Minutes30_*.md
python3 - <<'EOF'
import glob, importlib.util
from datetime import datetime
spec = importlib.util.spec_from_file_location("j", "reports/_analysis/svcb_journal.py")
j = importlib.util.module_from_spec(spec); spec.loader.exec_module(j)
path = [p for p in glob.glob("reports/SVCB_base/SVCB_rsi_pullback_Minutes30_*.md")][-1]
trades, _ = j.parse(path)
cut = datetime(2026, 9, 25, 13, 30)
old = [t for t in trades if t["entry"] <= cut]
new = [t for t in trades if t["entry"] > cut]
print("префикс: сделок %d, net %+.2f, PF %.3f" % (len(old), sum(t["pnl"] for t in old), j.pf_of(old)))
print("новые после 2026-09-25 13:30: сделок %d, net %+.2f" % (len(new), sum(t["pnl"] for t in new)))
EOF
```

  **Префикс обязан совпасть со спекой ровно:** сделок **130**, net **−3 663.14**, PF **0.951**.
  Расхождение префикса означает порчу кэша или забытый флаг — остановиться и доложить владельцу.

  Новые сделки (после 2026-09-25 13:30) — не ошибка: окно кончается моментом запуска (Global
  Constraints). Если их больше нуля, в отчёт `task-2-report-svcb.md` записываются **переснятые**
  числа полного окна: сделок, PF, max DD в обеих формах, годы, хвосты, выходы. С этого момента
  решающими считаются они, а гейт B пересчитывается по переснятому max DD (§5.5 спеки). Если новых
  сделок ноль, в отчёт пишется «baseline совпал со спекой».

- [ ] **Step 7: Коммит** `feat(rsi_pullback): пакет SVCB до калибровки` (вместе с отчётом).

---

### Task 3: Тема `screen` — цена двух гейтов и сверка числа фолдов

**Files:** Modify `data/params/rsi_pullback/svcb/cal_screen.json` (`_comment`); Create
`docs/superpowers/plans/task-3-report-svcb.md`

**Interfaces:**
- Consumes: `cal_screen.json` из Task 1.
- Produces: голоса фолдов по `UseDayATRGate` и `UseVolume` — их читает Task 11.

- [ ] **Step 1: Прогнать тему.**

```bash
go run ./cmd/backtest -ticker SVCB -strategy rsi_pullback -interval Minutes30 \
  -calibrate data/params/rsi_pullback/svcb/cal_screen.json -out ./reports/SVCB_screen \
  -months 36 -min-trades 1 -train-months 12 -test-months 6 -metric profit_factor
```

- [ ] **Step 2: Сверить число фолдов.** В шапке `*_walkforward.md` должно стоять «Фолдов: 4». Иначе
  **остановиться и доложить владельцу**. Записать фактические train/test-окна четырёх фолдов: они
  отсчитываются от даты запуска и нужны Task 11 Step 13.

- [ ] **Step 3: Записать** pooled OOS PF, размер пула и голоса всех четырёх фолдов по обеим осям;
  пометить вырожденные фолды.

- [ ] **Step 4: Сверить с ожиданием** (§6.3 спеки): **ожидается 1×0** — дневной гейт включён,
  объёмный выключен. Точечный замер in-sample: 1×0 → 0.951/130, 0×1 → 0.966/255, 0×0 → 0.891/437,
  1×1 → 0.882/94. Разница 1×0 и 0×1 — 0.015 PF, поэтому голоса могут разойтись; неподтверждённое
  ожидание записать прямым текстом.

- [ ] **Step 5: Написать отчёт** `docs/superpowers/plans/task-3-report-svcb.md` (образец —
  `docs/superpowers/plans/task-3-report-rtkm.md`) и дописать результат в `_comment` сетки строкой
  `РЕЗУЛЬТАТ ПРОГОНА <дата>: …` с pooled OOS, пулом, фолдами и голосами.

- [ ] **Step 6: Коммит** `docs(rsi_pullback): SVCB, тема screen`.

---

### Task 4: Тема `entry` — первая половина планки

**Files:** Modify `data/params/rsi_pullback/svcb/cal_entry.json` (`_comment`); Create
`docs/superpowers/plans/task-4-report-svcb.md`

**Interfaces:**
- Consumes: `cal_entry.json` из Task 1.
- Produces: голоса по `RSIPeriod` и `RSILower`, вердикт первой половины планки — их читают Task 10,
  Task 11 и Task 12.

- [ ] **Step 1: Прогнать тему.**

```bash
go run ./cmd/backtest -ticker SVCB -strategy rsi_pullback -interval Minutes30 \
  -calibrate data/params/rsi_pullback/svcb/cal_entry.json -out ./reports/SVCB_entry \
  -months 36 -min-trades 20 -train-months 12 -test-months 6 -metric profit_factor
```

- [ ] **Step 2: Записать** pooled OOS PF, пул и голоса четырёх фолдов (`RSILower` — ведущая ось);
  пометить вырожденные фолды.

- [ ] **Step 3: Проверить первую половину планки** (§5.12): pooled OOS ≥ **1.5** при ≥ **20**
  сделках **и** `RSILower` одинаков в ≥ 3 фолдах из 4. Вырожденный фолд в пользу тикера не
  засчитывается.

- [ ] **Step 4: Сверить с ожиданием:** **планку тема НЕ возьмёт.** Разведочный walk-forward той же
  схемы с сеткой без `RSIPeriod` 14 дал 1.379/60, голоса `RSILower` 15/25/30/45, `RSIPeriod`
  3/7/10/6. Ожидается рассыпание между глубоким (15–25) и мелким (45–50) горбом.

- [ ] **Step 5: Отметить вырожденные узлы, если фолды их выбрали:** `RSILower` 5 (6 сделок за окно),
  `RSIPeriod` 14 (11 сделок), 12 (27 сделок).

- [ ] **Step 6: Написать отчёт** `task-4-report-svcb.md` и дописать результат в `_comment`.

- [ ] **Step 7: Коммит** `docs(rsi_pullback): SVCB, тема entry`.

---

### Task 5: Темы `trend` и `trend_low` — вторая половина планки

**Files:** Modify `data/params/rsi_pullback/svcb/cal_trend.json`, `cal_trend_low.json`
(`_comment`); Create `docs/superpowers/plans/task-5-report-svcb.md`

**Interfaces:**
- Consumes: оба файла из Task 1.
- Produces: голоса по `EMAFast` и `EMASlow` из обеих тем и их pooled OOS — их читает Task 11.

- [ ] **Step 1: Прогнать обе темы (последовательно).**

```bash
go run ./cmd/backtest -ticker SVCB -strategy rsi_pullback -interval Minutes30 \
  -calibrate data/params/rsi_pullback/svcb/cal_trend.json -out ./reports/SVCB_trend \
  -months 36 -min-trades 20 -train-months 12 -test-months 6 -metric profit_factor
go run ./cmd/backtest -ticker SVCB -strategy rsi_pullback -interval Minutes30 \
  -calibrate data/params/rsi_pullback/svcb/cal_trend_low.json -out ./reports/SVCB_trend_low \
  -months 36 -min-trades 20 -train-months 12 -test-months 6 -metric profit_factor
```

- [ ] **Step 2: Записать по каждой теме** pooled OOS, пул и голоса четырёх фолдов (`EMASlow` —
  ведущая ось); пометить вырожденные фолды и отдельно — OOS PF фолда 4.

- [ ] **Step 3: Проверить вторую половину планки** по **канонической** `trend`, не по `trend_low`.

- [ ] **Step 4: Сверить с ожиданиями** (§6.2 спеки).
  - `trend` планку не возьмёт: каноническая ось 50–250 лежит в 0.942–1.031 in-sample.
  - `trend_low`: разведочный walk-forward широкого тренда голосовал `EMASlow` 15/20/20/20 при
    фолде 4 — 0.224/5. **Риск записан заранее:** большинство 3/4 за короткую EMA возможно, а
    in-sample она ломает 2026 год. Решает правило §5.2 — поле тренда идёт из `trend_low` только при
    большинстве ≥ 3/4 **и** pooled OOS выше, чем у `trend`.

- [ ] **Step 5: Записать, какая тема дала больший pooled OOS**, и предварительный вывод по полю
  тренда для Task 11.

- [ ] **Step 6: Написать отчёт** `task-5-report-svcb.md` и дописать результаты в оба `_comment`.

- [ ] **Step 7: Коммит** `docs(rsi_pullback): SVCB, темы trend и trend_low`.

---

### Task 6: Тема `day`

**Files:** Modify `data/params/rsi_pullback/svcb/cal_day.json` (`_comment`); Create
`docs/superpowers/plans/task-6-report-svcb.md`

**Interfaces:**
- Consumes: `cal_day.json` из Task 1.
- Produces: голоса по `FreshDayATR` и `SpentDayATR` и pooled OOS — их читают Task 10 и Task 11.

- [ ] **Step 1: Прогнать тему.**

```bash
go run ./cmd/backtest -ticker SVCB -strategy rsi_pullback -interval Minutes30 \
  -calibrate data/params/rsi_pullback/svcb/cal_day.json -out ./reports/SVCB_day \
  -months 36 -min-trades 20 -train-months 12 -test-months 6 -metric profit_factor
```

- [ ] **Step 2: Записать** pooled OOS, пул и голоса четырёх фолдов по обеим осям; пометить
  вырожденные фолды.

- [ ] **Step 3: Записать предупреждение по `FreshDayATR`.** Ненулевое значение заводит вход на первый
  бар дня — кандидат на провал запрета входов в выходные и часы 02–06 в Task 11 Step 6. Опорный
  замер: 0 → 0.951/130, 0.05 и 0.1 → 0.930/145, 0.2 → 0.913/159, 0.4 → 0.764/254. Ось вредна
  целиком — голос за ненулевое значение записать как неожиданный.

- [ ] **Step 4: Записать опорный замер `SpentDayATR`:** 0.5 → 0.949/278, **0.6 → 1.140/225**,
  0.7 → 1.041/179, 0.8 → 0.951 (дефолт), 0.9 → 0.734/91, 1.0 → 0.625/72, 2.0 → 2.117/**6**. Голос
  за 2.0 записать как подгонку под фолд на вырожденной выборке.

- [ ] **Step 5: Написать отчёт** `task-6-report-svcb.md` и дописать результат в `_comment`.

- [ ] **Step 6: Коммит** `docs(rsi_pullback): SVCB, тема day`.

---

### Task 7: Темы `volume` и `vol_window`

**Files:** Modify `data/params/rsi_pullback/svcb/cal_volume.json`, `cal_vol_window.json`
(`_comment`); Create `docs/superpowers/plans/task-7-report-svcb.md`

**Interfaces:**
- Consumes: оба файла из Task 1.
- Produces: голоса по `VolMult`, `VolBaseDays`, `VolLookbackBars` и pooled OOS обеих тем — их читает
  Task 11.

- [ ] **Step 1: Прогнать обе темы (последовательно).**

```bash
go run ./cmd/backtest -ticker SVCB -strategy rsi_pullback -interval Minutes30 \
  -calibrate data/params/rsi_pullback/svcb/cal_volume.json -out ./reports/SVCB_volume \
  -months 36 -min-trades 20 -train-months 12 -test-months 6 -metric profit_factor
go run ./cmd/backtest -ticker SVCB -strategy rsi_pullback -interval Minutes30 \
  -calibrate data/params/rsi_pullback/svcb/cal_vol_window.json -out ./reports/SVCB_vol_window \
  -months 36 -min-trades 20 -train-months 12 -test-months 6 -metric profit_factor
```

- [ ] **Step 2: Записать по каждой теме** pooled OOS, пул и голоса четырёх фолдов по всем осям.

- [ ] **Step 3: Сверить с ожиданием.** Обе темы идут с `UseVolume` 1 и **поле `UseVolume` не
  решают** — его решает `screen` (Task 3). Опорный замер `vol_window` in-sample: лучший узел
  3 × 2.0 → 1.046/65, остальные 0.88–1.03. **Ожидание: объёмный гейт в точку не войдёт**; тогда
  голоса этих тем пишутся справочно.

- [ ] **Step 4: Написать отчёт** `task-7-report-svcb.md` и дописать результаты в оба `_comment`.

- [ ] **Step 5: Коммит** `docs(rsi_pullback): SVCB, темы volume и vol_window`.

---

### Task 8: Тема `risk`, потолок гейта A и зонды капкана

**Files:** Modify `data/params/rsi_pullback/svcb/cal_risk.json` (`_comment`); Create
`data/params/rsi_pullback/svcb/probe_stop_10.json`, `probe_stop_15.json`; Create
`docs/superpowers/plans/task-8-report-svcb.md`

**Interfaces:**
- Consumes: `cal_risk.json` из Task 1.
- Produces: голоса по `StopDailyATR` (уже пропущенные через потолок 1.0) и по `TPDailyATR` и pooled
  OOS — их читают Task 10 и Task 11.

- [ ] **Step 1: Прогнать тему.**

```bash
go run ./cmd/backtest -ticker SVCB -strategy rsi_pullback -interval Minutes30 \
  -calibrate data/params/rsi_pullback/svcb/cal_risk.json -out ./reports/SVCB_risk \
  -months 36 -min-trades 20 -train-months 12 -test-months 6 -metric profit_factor
```

- [ ] **Step 2: Записать** pooled OOS, пул и голоса четырёх фолдов по обеим осям.

- [ ] **Step 3: Применить потолок гейта A.** Голос выше **1.0** заменяется на 1.0, цена решения в PF
  записывается прямым текстом. Обоснование повторить в отчёте (§5.4 спеки): стоп 1.0 срабатывает в
  10.4% сделок, 1.3 — в 4.0%, 1.5 и 2.0 — в 0.8% (один SL-выход). Разведочный зонд голосовал
  1.5/1.5/1.5/0.8 при pooled 2.283/97 — замена почти наверняка понадобится.

- [ ] **Step 4: Снять два зонда капкана на полном окне.** Каждый файл — одна комбинация поверх
  дефолтов ядра:
  - `probe_stop_10.json`: `{"_comment":"…","phases":[{"name":"probe","grid":{"StopDailyATR":[1.0]},"keepTop":1}]}`;
  - `probe_stop_15.json` — то же со `StopDailyATR` [1.5].

  В `_comment` — назначение зонда, опорный замер и команда запуска. Команда (последовательно):

```bash
go run ./cmd/backtest -ticker SVCB -strategy rsi_pullback -interval Minutes30 \
  -calibrate data/params/rsi_pullback/svcb/probe_stop_10.json -out ./reports/SVCB_probe_stop_10 \
  -months 36 -min-trades 1 -metric profit_factor
python3 reports/_analysis/svcb_journal.py reports/SVCB_probe_stop_10/*_best.md
go run ./cmd/backtest -ticker SVCB -strategy rsi_pullback -interval Minutes30 \
  -calibrate data/params/rsi_pullback/svcb/probe_stop_15.json -out ./reports/SVCB_probe_stop_15 \
  -months 36 -min-trades 1 -metric profit_factor
python3 reports/_analysis/svcb_journal.py reports/SVCB_probe_stop_15/*_best.md
```

  По каждому зонду записать PF, число SL-выходов, обе формы DD и годы. Опорные числа: 1.0 → 1.153,
  13 SL, DD 14 143.77 ₽ (13.16%), 2024 −2 225 ₽; 1.5 → 1.621, 1 SL, DD 9 441.97 ₽ (7.78%), все годы в
  плюс. **Главный вывод для отчёта: внутри потолка стоп 2024 год не лечит;** годы в плюс даёт
  только номинальный стоп за потолком.

- [ ] **Step 5: Записать опорный замер `TPDailyATR`:** сигнала у оси нет (лучший узел 0.25 →
  1.008/137); узлы 1.5 и 2.5 побайтово равны, это контрольные строки.

- [ ] **Step 6: Написать отчёт** `task-8-report-svcb.md` и дописать результат в `_comment`.

- [ ] **Step 7: Коммит** `docs(rsi_pullback): SVCB, тема risk и зонды капкана`.

---

### Task 9: Темы `exit` и `trail`

**Files:** Modify `data/params/rsi_pullback/svcb/cal_exit.json`, `cal_trail.json` (`_comment`);
Create `docs/superpowers/plans/task-9-report-svcb.md`

**Interfaces:**
- Consumes: оба файла из Task 1.
- Produces: голоса по `RSIUpper`, `UseRSIExit`, `UseTrail`, `TrailDailyATR` — их читает Task 11.

- [ ] **Step 1: Прогнать обе темы (последовательно).**

```bash
go run ./cmd/backtest -ticker SVCB -strategy rsi_pullback -interval Minutes30 \
  -calibrate data/params/rsi_pullback/svcb/cal_exit.json -out ./reports/SVCB_exit \
  -months 36 -min-trades 20 -train-months 12 -test-months 6 -metric profit_factor
go run ./cmd/backtest -ticker SVCB -strategy rsi_pullback -interval Minutes30 \
  -calibrate data/params/rsi_pullback/svcb/cal_trail.json -out ./reports/SVCB_trail \
  -months 36 -min-trades 20 -train-months 12 -test-months 6 -metric profit_factor
```

- [ ] **Step 2: Записать** pooled OOS, пул и голоса четырёх фолдов по всем осям.

- [ ] **Step 3: Сверить с опорными замерами** (§6.6 спеки).
  - `RSIUpper`: горб 55–60 (55 → 1.130/136, 60 → 1.106), дефолт 70 → 0.951. Связка `RSIUpper` 55 +
    стоп 0.8 оставляет 2024 год в минусе (−5 225 ₽) — записать, если тема проголосует за 55.
  - Трейл вреден: при `UseRSIExit` 1 — 0.89–0.95. Если тема голосует за `UseTrail` 1, гейт A
    применяется к `min(StopDailyATR, TrailDailyATR)` (урок AFKS).

- [ ] **Step 4: Написать отчёт** `task-9-report-svcb.md` и дописать результаты в оба `_comment`.

- [ ] **Step 5: Коммит** `docs(rsi_pullback): SVCB, темы exit и trail`.

---

### Task 10: Темы-арбитры `entry_stop` и `day_stop`

**Files:** Modify `data/params/rsi_pullback/svcb/cal_entry_stop.json`, `cal_day_stop.json`
(`_comment`); Create `docs/superpowers/plans/task-10-report-svcb.md`

**Interfaces:**
- Consumes: оба файла из Task 1; pooled OOS и голоса `entry` (Task 4), `day` (Task 6), `risk`
  (Task 8).
- Produces: вердикты «рычаги складываются / конкурируют / арбитр справочный» по двум парам и голоса
  арбитров — их читают Task 11 Step 1 и Task 12 Step 1.

- [ ] **Step 1: Прогнать обе темы (последовательно).**

```bash
go run ./cmd/backtest -ticker SVCB -strategy rsi_pullback -interval Minutes30 \
  -calibrate data/params/rsi_pullback/svcb/cal_entry_stop.json -out ./reports/SVCB_entry_stop \
  -months 36 -min-trades 20 -train-months 12 -test-months 6 -metric profit_factor
go run ./cmd/backtest -ticker SVCB -strategy rsi_pullback -interval Minutes30 \
  -calibrate data/params/rsi_pullback/svcb/cal_day_stop.json -out ./reports/SVCB_day_stop \
  -months 36 -min-trades 20 -train-months 12 -test-months 6 -metric profit_factor
```

- [ ] **Step 2: Записать по каждой теме** pooled OOS, пул и голоса четырёх фолдов по обеим осям.

- [ ] **Step 3: Вынести вердикт каждого арбитра** (§5.2 спеки). Для пары `entry` × `risk` судит
  `entry_stop`, для пары `day` × `risk` — `day_stop`:
  - обе темы пары голосуют за уход от дефолта, и pooled OOS арбитра **выше** обеих — рычаги
    **складываются**, оба поля идут в точку по своим темам;
  - pooled OOS арбитра ниже любой из двух — рычаги **конкурируют**: в точку идёт один рычаг, тот,
    чья тема дала больший pooled OOS; цена решения пишется прямым текстом;
  - за уход от дефолта голосует не больше одной темы пары — арбитр **справочный**.

- [ ] **Step 4: Записать опорные числа разведки:** зонд `RSILower` [15,20,25,30,45,50] × стоп
  [0.5,0.6,0.7,0.8,1.0] дал 36/12/6 → 1.091/115, фолды 1.104/57, 0.800/8, 1.341/8, 1.181/42, голоса
  `RSILower` 50/15/15/50, стоп 0.7/0.8/0.6/0.5. In-sample: `RSILower` 20 + стоп 0.7 → 1.709/77,
  все годы в плюс; `SpentDayATR` 0.6 + стоп 0.8 → 1.367/214, все годы в плюс.

- [ ] **Step 5: Написать отчёт** `task-10-report-svcb.md` и дописать результаты в оба `_comment`.

- [ ] **Step 6: Коммит** `docs(rsi_pullback): SVCB, темы-арбитры entry_stop и day_stop`.

---

### Task 11: Сборка точки, гейты, три walk-forward и семь пунктов стоп-условия

**Files:** Create `data/params/rsi_pullback/svcb/plateau_point.json` и соседи плато
`plateau_<поле>_<значение>.json`; Create `docs/superpowers/plans/task-11-report-svcb.md`

**Interfaces:**
- Consumes: голоса и вердикты задач 3–10; скрипты `svcb_journal.py`, `sfin_sweep.py`.
- Produces: точку первого круга (восемнадцать полей) и вердикт — их читают Task 12 и Task 14.

- [ ] **Step 1: Собрать точку правилом большинства** (§5.2 спеки):
  - поле берётся из темы, которая его меряет, только при ≥ 3 голосах из 4 **невырожденных**
    фолдов; иначе — дефолт ядра; ничья 2/2 большинством не считается;
  - `trend_low` берёт поле тренда только при превосходстве pooled OOS над `trend`;
  - `vol_window` берёт поля объёма только при превосходстве над `volume` и при `UseVolume` 1 по
    `screen`;
  - `RSIUpper` — только из `exit`;
  - вердикты арбитров применяются по Task 10.

  Записать таблицу для всех восемнадцати полей: «поле → тема-источник → голоса по фолдам →
  вырожденные фолды → принятое значение → дефолт ядра».

- [ ] **Step 2: Применить гейт A:** `min(StopDailyATR, TrailDailyATR при UseTrail = 1)` ≤ 1.0.

- [ ] **Step 3: Написать `plateau_point.json`** — одна фаза, все восемнадцать полей по одному
  значению (образец — `data/params/rsi_pullback/rtkm/plateau_point.json`). Снять одиночный прогон:

```bash
go run ./cmd/backtest -ticker SVCB -strategy rsi_pullback -interval Minutes30 \
  -calibrate data/params/rsi_pullback/svcb/plateau_point.json -out ./reports/SVCB_point \
  -months 36 -min-trades 1 -metric profit_factor
python3 reports/_analysis/svcb_journal.py reports/SVCB_point/*_best.md
```

- [ ] **Step 4: Применить гейт B:** max DD ≤ **14 848.91 ₽ и ≤ 14.06%** (или переснятые числа Task 2
  Step 6), обе формы. При пробое в точку идёт ближайший вариант плато с меньшей просадкой, цена
  пишется прямым текстом.

- [ ] **Step 5: Гейт D:** из блока «ГЕЙТ D» — ноль сделок через все пять дат. Иначе сработал пункт 7.

- [ ] **Step 6: Запрет входов в выходные и в часы 02–06:** из вывода скрипта — входов в выходные 0 и
  входов 02–06 (блок «ГЕЙТ C») 0. При провале увести `FreshDayATR` на 0 и пересчитать с Step 3. Если
  провал остался — сработал пункт 7.

- [ ] **Step 7: Снять соседей плато.** Для каждого поля, ушедшего от дефолта, — прогон самого
  значения и двух соседей по оси, команда как в Step 3, файлы `plateau_<поле>_<значение>.json`.
  - Край сетки дополнительно проверяется зондом за краем.
  - **Для каждого соседа снять годы и хвосты** скриптом `svcb_journal.py` (§5.3 спеки).
  - Разница pooled OOS < 0.05 решается в пользу меньшей просадки, затем меньшего числа убыточных
    полугодий.
  - Плато уже 0.05 PF записывается как отсутствие сигнала.

- [ ] **Step 8: Третий контур.** Сравнить с дефолтами: SL 33.1%, удержание 8/16/58, ночёвок 38.5%,
  переносов 1, DD 14.06%, expectancy −28.18 ₽. **Падение доли SL вместе с ростом удержания и
  ночёвок — капкан**, тогда точка берёт более узкий стоп в пределах плато.

- [ ] **Step 9: Три walk-forward точки (последовательно).**

```bash
go run ./cmd/backtest -ticker SVCB -strategy rsi_pullback -interval Minutes30 \
  -calibrate data/params/rsi_pullback/svcb/plateau_point.json -out ./reports/SVCB_point_wf36 \
  -months 36 -min-trades 1 -train-months 12 -test-months 6 -metric profit_factor
go run ./cmd/backtest -ticker SVCB -strategy rsi_pullback -interval Minutes30 \
  -calibrate data/params/rsi_pullback/svcb/plateau_point.json -out ./reports/SVCB_point_wf24 \
  -months 24 -min-trades 1 -train-months 12 -test-months 3 -metric profit_factor
go run ./cmd/backtest -ticker SVCB -strategy rsi_pullback -interval Minutes30 \
  -calibrate data/params/rsi_pullback/svcb/plateau_point.json -out ./reports/SVCB_point_wf30 \
  -months 30 -min-trades 1 -train-months 12 -test-months 6 -metric profit_factor
```

  Ожидаемое число фолдов: **4, 4, 3**. Записать pooled OOS, пул и пофолдовые числа, пометить
  вырожденные фолды (меньше пяти сделок или без убыточных).

- [ ] **Step 10: Утяжелённые издержки.**

```bash
go run ./cmd/backtest -ticker SVCB -strategy rsi_pullback -interval Minutes30 \
  -calibrate data/params/rsi_pullback/svcb/plateau_point.json -out ./reports/SVCB_point_c001 \
  -months 36 -min-trades 1 -train-months 12 -test-months 6 -metric profit_factor -commission 0.001
```

- [ ] **Step 11: Календарные годы.** Из блока «календарные годы» Step 3: 2024, 2025, 2026 — net > 0.

- [ ] **Step 12: Хвосты.** Из блока «ПУНКТ 6» Step 3 записать хвост 6 и хвост 12 (сделок, net, PF) и
  долю крупнейшей сделки в net хвоста 6: найти в журнале отчёта сделку с максимальным |PnL| среди
  входов с 2026-03-25.

- [ ] **Step 13: Сверка знака хвоста 6 с фолдом 4.** Взять OOS PF фолда 4 из отчёта `SVCB_point_wf36`
  и его фактическое тест-окно (оно начинается за шесть месяцев до даты запуска, а не ровно
  2026-03-25). Знаки (≥ 1 или < 1) обязаны совпасть с хвостом 6. Расхождение — остановиться и
  доложить владельцу вместе с обоими числами и обоими окнами.

- [ ] **Step 14: Применить семь пунктов стоп-условия** (Global Constraints). По каждому пункту
  записать число и вердикт.

- [ ] **Step 15: Проверить планку** (Task 4 Step 3, Task 5 Step 3) и сверить с ожиданием «не
  возьмут обе».

- [ ] **Step 16: Написать отчёт** `task-11-report-svcb.md`. Содержание:
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

  **Ожидание, записанное до прогонов: второй круг вероятен** (§5.14 спеки): точка «дефолты +
  стоп 1.0» проваливает 2024 год.

- [ ] **Step 18: Коммит** `feat(rsi_pullback): SVCB, точка первого круга и вердикт`.

---

### Task 12: Второй круг (только при провале первого)

**Files:** Create `data/params/rsi_pullback/svcb/cal2_<тема>.json` (не больше шести),
`plateau_point2.json`, соседи `plateau_r2_<поле>_<значение>.json`; Modify
`internal/service/backtest/rsi_pullback_svcb_grid_test.go`; Create
`docs/superpowers/plans/task-12-report-svcb.md`

**Задача выполняется ТОЛЬКО при срабатывании хотя бы одного пункта стоп-условия в Task 11.**

**Interfaces:**
- Consumes: голоса задач 3–10 (включая арбитров), вердикт Task 11, `svcbGridFiles` из Task 1.
- Produces: точку второго круга — её читают Task 13 и Task 14.

- [ ] **Step 1: Выбрать зоны по фактическим голосам фолдов первого круга**, а не по in-sample рельефу.
  Шаг — половина шага первого круга. Не больше шести тем. **Каждое поле свипует ровно одна тема**
  второго круга, и она — его источник (§5.14 спеки): тема-связка забирает оба своих поля, одиночные
  темы тех же полей не заводятся. В `_comment` каждого файла записать, из каких голосов какого фолда
  получена зона.
  - Кандидат номер один — `cal2_entry_stop`, если голоса `entry` / `entry_stop` рассыпались по
    глубокому горбу: `RSILower` [12.5,15,17.5,20,22.5,25,27.5] × `StopDailyATR`
    [0.6,0.65,0.7,0.75,0.8,0.85,0.9,0.95,1.0]. Если фолды выбирали мелкий горб (45–50), зона
    `RSILower` строится вокруг него тем же шагом 2.5, не выше 50.
  - Кандидат номер два — `cal2_day_stop`: `SpentDayATR` [0.55,0.6,0.65,0.7] × `StopDailyATR`
    [0.7,0.75,0.8,0.85,0.9] — только если `SpentDayATR` не свипует другая тема второго круга.
  - Стоп ни в одном файле второго круга не выше 1.0.

- [ ] **Step 2: Расширить сторожевой тест.** В `rsi_pullback_svcb_grid_test.go` добавить переменную
  `svcbRound2GridFiles` с поимённым списком файлов `cal2_*.json` и функцию:

```go
func svcbAllGridFiles() []string {
	out := make([]string, 0, len(svcbGridFiles)+len(svcbRound2GridFiles))
	out = append(out, svcbGridFiles...)
	return append(out, svcbRound2GridFiles...)
}
```

  Заменить `for _, file := range svcbGridFiles` на `for _, file := range svcbAllGridFiles()` в первом
  цикле `TestSVCBGridsStayWide`. Добавить в конец теста проверку потолка по всем файлам второго круга:

```go
	for _, file := range svcbRound2GridFiles {
		for _, v := range rsiPullbackTickerGrid(t, "svcb", file)["StopDailyATR"] {
			if v > svcbStopCeiling {
				t.Errorf("svcb/%s: StopDailyATR=%v выше потолка гейта A %v", file, v, svcbStopCeiling)
			}
		}
	}
```

  И тест «одно поле — одна тема»:

```go
// TestSVCBRound2FieldsHaveOneSource держит правило §5.14 спеки: во втором круге каждое поле
// свипует ровно одна тема, иначе две темы проголосуют за одно поле по-разному.
func TestSVCBRound2FieldsHaveOneSource(t *testing.T) {
	owner := map[string]string{}
	for _, file := range svcbRound2GridFiles {
		for field, values := range rsiPullbackTickerGrid(t, "svcb", file) {
			if len(values) < 2 {
				continue // зафиксированное поле не голосует
			}
			if prev, ok := owner[field]; ok {
				t.Errorf("svcb: поле %s свипуют две темы второго круга: %s и %s", field, prev, file)
			}
			owner[field] = file
		}
	}
}
```

  Run: `go test ./internal/service/backtest/ -run 'SVCB|RSIPullbackCalFilesValid|RSIPullbackGridControlPoints' -v`
  Expected: PASS.

- [ ] **Step 3: Прогнать каждую узкую тему (последовательно).**

```bash
go run ./cmd/backtest -ticker SVCB -strategy rsi_pullback -interval Minutes30 \
  -calibrate data/params/rsi_pullback/svcb/cal2_<тема>.json -out ./reports/SVCB_r2_<тема> \
  -months 36 -min-trades 1 -train-months 12 -test-months 6 -metric profit_factor
```

  `-min-trades 1` — число сделок не критерий (указание владельца).

- [ ] **Step 4: Собрать точку второго круга** тем же правилом ≥ 3/4 невырожденных фолдов и прогнать
  её через Task 11 Steps 2–13 без изменений, с файлами `plateau_point2.json` / `plateau_r2_*` и
  каталогами `SVCB_point2*`. Поля, которые второй круг не свипует, берутся из точки первого круга.

- [ ] **Step 5: Стоп-условие второго круга.**
  - Пункт 2 снят; фактический размер пула записывается.
  - **Пул OOS меньше десяти сделок закрывает работу.**
  - Пункты 1, 3, 4, 5, 6, 7 — в силе полностью.

- [ ] **Step 6: Отчёт** `task-12-report-svcb.md` той же структуры, что Task 11, плюс таблица «зона
  узкой сетки → голоса первого круга». Прогнать тесты:
  `go test ./internal/service/backtest/ -run 'SVCB|RSIPullback'`.

- [ ] **Step 7: Вердикт.** Точка прошла — переход к Task 14. Иначе — Task 13.

- [ ] **Step 8: Коммит** `feat(rsi_pullback): SVCB, второй круг и вердикт`.

---

### Task 13: Протокол отказа (только при провале обоих кругов)

**Files:** Modify `internal/service/trading_strategy/rsi_pullback/strategy/svcb/svcb.go`
(doc-comment), `svcb_test.go` (комментарий теста); Create
`docs/superpowers/plans/task-13-report-svcb.md`

**Задача выполняется ТОЛЬКО при провале обоих кругов. Задачи 14–17 не выполняются.**

- [ ] **Step 1: Написать разбор отказа в doc-comment пакета.** Образец —
  `internal/service/trading_strategy/rsi_pullback/strategy/rtkm/rtkm.go`. Содержание:
  - всё из §5.16 спеки;
  - какой пункт сработал в каждом круге и с какими числами;
  - что на бумаге этому причиной.

  В doc-comment теста `TestParamsTrackTheBaselineUntilCalibrated` записать окончательное состояние
  «калибровка закрыта отказом». Литерал не ставится.

- [ ] **Step 2: Полный гейт.** Run: `./bin/mage ci`. Expected: PASS.

- [ ] **Step 3: Доложить владельцу** числа обоих кругов и причину отказа.

- [ ] **Step 4: Коммит** `docs(rsi_pullback): SVCB, протокол отказа`.

---

### Task 14: Литерал в пакете и снимок

**Files:** Modify `internal/service/trading_strategy/rsi_pullback/strategy/svcb/svcb.go`,
`svcb_test.go`, `internal/service/backtest/rsi_pullback_registry_test.go`

**Выполняется ТОЛЬКО при положительном вердикте Task 11 Step 17 или Task 12 Step 7.**

**Interfaces:**
- Consumes: принятую точку.
- Produces: `svcb.DefaultParams()` возвращает литерал — его читают Task 15 и Task 16.

- [ ] **Step 1: Заменить тест пакета снимком.** Удалить `TestParamsTrackTheBaselineUntilCalibrated`,
  написать `TestParamsMatchTheCalibratedSnapshot`. Все восемнадцать полей выписаны литералами.
  Образец — `internal/service/trading_strategy/rsi_pullback/strategy/sfin/sfin_test.go`. Скелет, где
  значения берутся из принятой точки (таблица отчёта Task 11 или 12):

```go
func TestParamsMatchTheCalibratedSnapshot(t *testing.T) {
	want := core.Params{
		RSIPeriod: 4, RSILower: 20, RSIUpper: 70,
		EMAFast: 10, EMASlow: 100,
		DailyATRPeriod: 14, UseDayATRGate: 1, FreshDayATR: 0, SpentDayATR: 0.8,
		StopDailyATR: 0.7, TPDailyATR: 0.6,
		UseVolume: 0, VolBaseDays: 14, VolLookbackBars: 3, VolMult: 1.2,
		UseRSIExit: 1, UseTrail: 0, TrailDailyATR: 0,
	}
	if got := DefaultParams(); got != want {
		t.Fatalf("литерал SVCB разошёлся со снимком калибровки:\n got: %+v\nwant: %+v", got, want)
	}
}
```

  Значения в скелете — **пример формы**, взятый из разведки (§6.6 спеки). Перед записью они
  заменяются числами принятой точки поле за полем. Целочисленные поля `core.Params`
  (`core/core.go:30`): `RSIPeriod`, `EMAFast`, `EMASlow`, `DailyATRPeriod`, `UseDayATRGate`,
  `UseVolume`, `VolBaseDays`, `VolLookbackBars`, `UseRSIExit`, `UseTrail`; остальные — `float64`.

- [ ] **Step 2: Убедиться, что тест падает.**
  Run: `go test ./internal/service/trading_strategy/rsi_pullback/strategy/svcb/ -v`
  Expected: FAIL.

- [ ] **Step 3: Поставить литерал.** `DefaultParams()` возвращает `core.Params` со всеми восемнадцатью
  полями **явно**, даже совпавшими с ядром.

- [ ] **Step 4: Обновить тест реестра.** `TestRSIPullbackSVCBTracksBaseline` заменить на
  `TestRSIPullbackSVCBIsRegisteredAndCalibrated` по образцу `TestRSIPullbackSFINIsRegisteredAndCalibrated`
  (`rsi_pullback_registry_test.go:1199`). Тест проверяет две вещи: реестр отдаёт
  `rsipullbacksvcb.DefaultParams()`, и это не `core.DefaultParams()`.

- [ ] **Step 5: Прогнать тесты.**
  Run: `go test ./internal/service/trading_strategy/rsi_pullback/... ./internal/service/backtest/ -run 'SVCB|RSIPullback' -v`
  Expected: PASS.

- [ ] **Step 6: Коммит** `feat(rsi_pullback): литерал SVCB`.

---

### Task 15: Реестр живого раннера

**Files:** Modify `internal/service/trading_strategy/rsi_pullback/live/registry.go`,
`internal/service/trading_strategy/rsi_pullback/live/registry_test.go`

**Interfaces:**
- Consumes: `svcb.Ticker`, `svcb.DefaultParams()` из Task 14.
- Produces: `ParamsFor("SVCB")` — его проверяет Task 18 через `cmd/pullparity`.

- [ ] **Step 1: Написать падающий тест** по образцу `TestRegistryHasSFIN`
  (`registry_test.go:214`):

```go
// TestRegistryHasSVCB держит связку «пакет — реестр живого раннера» для SVCB: раннер обязан
// торговать ровно литерал пакета.
func TestRegistryHasSVCB(t *testing.T) {
	p, ok := ParamsFor(svcb.Ticker)
	if !ok {
		t.Fatal("SVCB нет в реестре живого раннера: тикер не будет торговать вовсе")
	}
	if want := svcb.DefaultParams(); p != want {
		t.Fatalf("реестр отдаёт не литерал пакета:\n got: %+v\nwant: %+v", p, want)
	}
}
```

- [ ] **Step 2: Убедиться, что тест падает.**
  Run: `go test ./internal/service/trading_strategy/rsi_pullback/live/ -run SVCB -v`
  Expected: FAIL.

- [ ] **Step 3: Добавить в реестр.** Импорт
  `"tinvest/internal/service/trading_strategy/rsi_pullback/strategy/svcb"` (в алфавитном порядке
  блока) и строка `svcb.Ticker: svcb.DefaultParams(),` после `sfin.Ticker: sfin.DefaultParams(),`
  (`registry.go:712`).

- [ ] **Step 4: Прогнать тесты.**
  Run: `go test ./internal/service/trading_strategy/rsi_pullback/live/ -v`
  Expected: PASS.

- [ ] **Step 5: Коммит** `feat(rsi_pullback): SVCB в реестре живого раннера`.

---

### Task 16: Боевая вселенная и сторож env-файлов

**Files:** Modify `internal/config/rsi_pullback.go:605`, `internal/config/rsi_pullback_test.go:54-56`,
`env/prod.env:20`, `env/prod.env.example:30`, `env/local.env.example:28`

- [ ] **Step 1: Написать падающие тесты.**
  - Существующий тест `TestNewRSIPullbackConfig_Defaults` (`rsi_pullback_test.go:52`): дописать
    `"SVCB"` последним в `want` и заменить сообщение `want 32: SFIN заведён тридцать вторым
    2026-09-15` на `want 33: SVCB заведён тридцать третьим <дата исполнения>`.
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
  Expected: FAIL — `want 33` в первом тесте. Второй тест до правки Go-списка проходит (все четыре
  места кончаются на `SFIN`) — это нормально: он сторожит расхождение, а падает на Step 3, если
  правка пропустит хотя бы один файл.

- [ ] **Step 3: Добавить `"SVCB"` последним элементом** во все четыре места сразу:
  - `internal/config/rsi_pullback.go:605`;
  - `env/prod.env:20`;
  - `env/prod.env.example:30`;
  - `env/local.env.example:28`.

- [ ] **Step 4: Прогнать тесты.**
  Run: `go test ./internal/config/ -v`
  Expected: PASS.

- [ ] **Step 5: Мутационная проверка сторожа.** Временно убрать `,SVCB` из `env/prod.env`, прогнать
  `go test ./internal/config/ -run RSIPullbackTickersMatchEnvFiles` — Expected: FAIL с именем файла
  `env/prod.env`. Вернуть правку (`git checkout env/prod.env` не годится — файл ещё не закоммичен;
  восстановить строку вручную) и снова прогнать — PASS.

- [ ] **Step 6: Коммит** `feat(rsi_pullback): завести SVCB в боевую вселенную`.

---

### Task 17: Дока пакета — разбор калибровки и принятый риск

**Files:** Modify `internal/service/trading_strategy/rsi_pullback/strategy/svcb/svcb.go`
(doc-comment)

- [ ] **Step 1: Написать разбор в doc-comment пакета.** Образец формы —
  `internal/service/trading_strategy/rsi_pullback/strategy/sfin/sfin.go`. Обязательное содержимое
  §5.16 спеки:
  - короткая история (IPO 2023-12-15) и девятимесячный train фолда 1;
  - шаг цены 0.005 ₽ и реальный круг 0.104% против модельных 0.1%;
  - таблица сборки точки;
  - годы 2024–2026;
  - хвосты 6 и 12 с долей крупнейшей сделки хвоста 6 и сверкой знака с фолдом 4;
  - таблица срабатываний стопа, голос темы `risk` и цена урезания до 1.0;
  - гейт B в обеих формах;
  - третий контур против дефолтов;
  - гейт D с оговоркой «даты — прокси, движок дивиденды не моделирует, ручной контроль владельца в
    дни отсечки»;
  - три walk-forward с вырожденными фолдами;
  - пул OOS;
  - семь пунктов и планка;
  - условие пересмотра после прода (§8 №3 спеки): хвост 6 месяцев с PF < 1.0 на ≥ 10 сделках —
    вывести из вселенной.

  **Правило CLAUDE.md: `docs/rsi_pullback/` не трогается.**

- [ ] **Step 2: Линтер.** Run: `./bin/mage lint`. Expected: PASS.

- [ ] **Step 3: Коммит** `docs(rsi_pullback): разбор калибровки SVCB и принятый риск`.

---

### Task 18: Финальная проверка

**Files:** нет (только прогоны), кроме правок по найденному.

- [ ] **Step 1: Полный гейт.** Run: `./bin/mage ci`. Expected: PASS (lint + `go test -race ./...` +
  дрейф моков).

- [ ] **Step 2: Сверка живой сборки с бэктестом.**

```bash
go run ./cmd/pullparity -tickers SVCB -months 24
```

  Флаг — **`-tickers`** (множественное число). Без него команда молча возьмёт `UGLD,T,GAZP`.
  **`-months 24`, не 36:** на более длинном окне сверка даёт ложные расхождения (урок SFIN).
  Expected: ноль расхождений.

- [ ] **Step 3: Сверить вселенную.** Run: `go test ./internal/config/ -run RSIPullbackTickersMatchEnvFiles -v`.
  Expected: PASS, в списке 33 тикера, SVCB последний.

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
| §1 инструмент, кэш, скользящий конец окна, гэпы | Global Constraints; Task 2 Step 6 |
| §2 отношение к каталогу | Task 17 Step 1 |
| §3 baseline | Global Constraints; Task 2 Step 6 |
| §4 схемы, короткая история, ASTR, вырожденные фолды | Global Constraints; Task 3 Step 2; Task 11 Steps 1, 9 |
| §5.1 двенадцать тем и инварианты | Task 1; задачи 3–10 |
| §5.2 сборка и арбитры | Task 10 Step 3; Task 11 Step 1 |
| §5.3 соседи плато с годами и хвостами | Task 11 Step 7 |
| §5.4 гейт A (1.0) | Task 8 Step 3; Task 11 Step 2 |
| §5.5 гейт B (не выше baseline) и ничьи | Task 11 Steps 4, 7 |
| §5.6 третий контур, запрет входов | Task 11 Steps 6, 8 |
| §5.7 гейт D | Task 11 Step 5 |
| §5.8 скрипт журнала | Global Constraints |
| §5.9 календарные годы | Task 11 Step 11 |
| §5.10 хвосты и сверка знака | Task 11 Steps 12–13 |
| §5.11 схемы проверки | Task 11 Steps 9–10 |
| §5.12 планка | Task 4 Step 3; Task 5 Step 3; Task 11 Step 15 |
| §5.13 стоп-условие | Task 11 Step 14 |
| §5.14 второй круг, одно поле — одна тема | Task 12 |
| §5.15 правило прода | Task 11 Step 17; задачи 14–16 |
| §5.16 что записывается | Task 17; Task 13 (при отказе) |
| §6 свойства бумаги | опорные замеры в задачах 3–10 и `_comment` сеток |
| §7 артефакты | задачи 1, 2, 14–16 |
| §8 риски | Task 17 Step 1 |

**Плейсхолдеры.** Угловые скобки остались только в образцах, где исполнитель заполняет значения
из отчётов прогонов (`<тема>`, `<поле>`, `<дата исполнения>`). Скелет литерала в Task 14 явно
помечен как пример формы.

**Согласованность имён:**
- `svcbGridFiles`, `svcbStopCeiling` — Task 1, расширяются в Task 12 через `svcbAllGridFiles`,
  `svcbRound2GridFiles`;
- `svcb.Ticker` и `svcb.DefaultParams()` — Task 2, используются в задачах 14–16;
- `TestRSIPullbackSVCBTracksBaseline` — Task 2, заменяется в Task 14;
- `TestRSIPullbackTickersMatchEnvFiles` — Task 16, запускается в Task 18;
- номера тем в отчётах совпадают с номерами задач (`task-N-report-svcb.md`).

**Развилки:**
- Task 11 Step 17: точка прошла — задачи 12 и 13 пропускаются;
- иначе Task 12; при её провале Task 13, и задачи 14–18 (кроме `mage ci` в Task 13) не выполняются;
- Task 10 Step 3: при конфликте рычагов в точку идёт один из двух.
