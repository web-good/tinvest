# TGKA под `rsi_pullback` — план подготовки, калибровки и вердикта

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development
> (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use
> checkbox (`- [ ]`) syntax for tracking.

**Goal:** Довести TGKA (ПАО «ТГК-1») до вердикта по стратегии `rsi_pullback`: каталог максимально
широких сеток, канонический тематический walk-forward, принятая точка, прошедшая два риск-гейта,
литерал в пакете и заведение в боевую вселенную двадцать третьим тикером.

**Architecture:** Процедура тем **каноническая**: все одиннадцать тем идут поверх дефолтов ядра —
на TGKA дефолт входа стоит точно в максимуме своей оси (`RSILower` 30 → 1.967/167), поэтому якорь
(BSPB, SOFL) не нужен и числа тем сравнимы с каталогом построчно. Схема прогонов **36/12/6**
(четыре фолда, проверено контрольным прогоном до написания спеки), плюс **обязательный контрольный
прогон принятой точки по схеме 24/12/3** — внутри окна сессия расширилась почти вдвое (19 → 35
получасовых баров в буднем дне), а ликвидность упала вчетверо. Два риск-гейта: гейт выживаемости
применяется к эффективной защите `min(StopDailyATR, TrailDailyATR при UseTrail=1)`, гейт просадки —
max DD точки ≤ 1.3 × max DD baseline. **Особенность TGKA:** капкан широкого стопа здесь не
детектируется просадкой (стоп 2.0 даёт PF 2.740 при DD 8 303 против 1.967/10 388 у дефолта), поэтому
гейт A — единственный работающий фильтр.

**Tech Stack:** Go 1.25, `cmd/backtest` (rolling walk-forward), `cmd/pullparity` (сверка живой
сборки с бэктестом), `./bin/mage ci` (lint + `go test -race ./...` + дрейф моков).

**Spec:** `docs/superpowers/specs/2026-09-03-tgka-rsi-pullback-prep-design.md`

## Global Constraints

- **Таймфрейм `Minutes30`** во всех без исключения прогонах. Флаг `-interval Minutes30` обязателен в
  каждой команде: дефолт CLI — `Hour1`, и забытый флаг даёт чужие числа, а не ошибку.
- **Схема прогонов каноническая:** `-months 36 -train-months 12 -test-months 6 -min-trades 20
  -metric profit_factor`, четыре фолда встык. У темы `screen` — `-min-trades 1`.
- **Расчётное окно 2023-09-04 … 2026-09-03, 36.0 месяца.** Обрезки по началу истории нет.
- **Число фолдов сверяется на первой же теме.** Контрольный прогон 2026-09-03 уже дал «Фолдов: 4»
  (pooled OOS PF 1.991 на 36/12/6 и 1.914 на 24/12/3), но сверка повторяется: если в шапке отчёта
  темы `screen` окажется «Фолдов: 3» — **остановиться и доложить владельцу**, остальные темы не
  запускать.
- **`-refresh` НЕ запускать ни на одном шаге.** Кэш дотянут штатным top-up 2026-09-03:
  `TGKA_Minutes30.json` — 32 229 баров (2023-08-07 … 2026-09-03); `TGKA_Day1.json` — 1 154 свечи
  (2022-08-04 … 2026-09-02).
- **Дыр в серии нет.** Разрыв длиннее четырёх дней ровно один — новогодние каникулы
  2023-12-29 → 2024-01-03 (4.6 дня), календарный. Скачков «открытие против вчерашнего закрытия»
  глубже 12% ноль.
- **ПРОЦЕДУРА КАНОНИЧЕСКАЯ:** все темы поверх дефолтов ядра, якоря нет.
- **Сетки держатся максимально широкими** (решение владельца 2026-09-03). Обрезок осей не делается;
  замеры, которые в узком каталоге были бы основанием вырезать край, идут в `_comment` как
  **предупреждения**. Жёсткие инварианты: `RSILower ≤ 50`, `RSIPeriod ≥ 2`, `RSIUpper > RSILower` на
  всех парах, ось тренда не порождает пар `EMAFast ≥ EMASlow`, `StopDailyATR` нигде не равен нулю.
- **Правило сборки точки:** поле берётся из темы, которая его меряет, и принимается только если за
  значение высказались **не менее трёх фолдов из четырёх**; иначе поле остаётся на **дефолте ядра**.
  Ничья 2/2 большинством не считается.
- **РИСК-ГЕЙТ A — выживаемость эффективной защиты.** Принимается только защита, достижимая **не
  менее чем в 30% будних дней окна**, и гейт применяется к `min(StopDailyATR, TrailDailyATR при
  UseTrail=1)`. Таблица выживаемости TGKA (n = 748): 0.3 — 99.2%, 0.5 — 90.9%, 0.7 — 72.7%,
  0.8 — 60.4%, **1.0 — 43.6%**, 1.3 — 23.5%, 1.5 — 16.3%, 2.0 — 7.6%. **Потолок — 1.0.** Если тема
  голосует выше, в точку идёт ближайшее значение оси внутри гейта, а цена решения в PF пишется
  прямым текстом. Порог 30% после начала прогонов не двигается.
- **РИСК-ГЕЙТ B — просадка точки.** Max DD принятой точки на полном окне (одиночный прогон
  `-params`) **не превышает 1.3 × max DD baseline**: 10.21% × 1.3 = **13.27%**.
- **Третий контур — риск-профиль точки против baseline.** Опорные числа baseline: доля SL-выходов
  **21.0%**, удержание медиана **8** / p90 **21** / максимум **37** бара, ночёвок **40.7%**, выходов
  в выходную сессию **5.4%**, max DD **10.21%**. Точка меряется по всем пяти. Выходная сессия —
  отдельный риск: её медианный оборот 1.9 млн ₽, проскальзывание там не моделируется.
- **Планка** (объявлена до прогонов, не пересматривается): темы `entry` и `trend` (каноническая, не
  `trend_low`) обе дают pooled OOS PF ≥ 1.5 при ≥ 20 сделках в пуле; ведущая ось (`RSILower` для
  `entry`, `EMASlow` для `trend`) выбрана одинаково в ≥ 3 фолдах из 4; вырожденный фолд в пользу
  тикера не засчитывается.
- **Стоп-условие из ТРЁХ пунктов.** Работа останавливается, если принятая точка даёт: (1) pooled OOS
  PF < 1.0 на 36/12/6, **либо** (2) меньше 20 сделок в пуле OOS, **либо** (3) pooled OOS PF < 1.0 на
  контрольном прогоне 24/12/3. При срабатывании — числа владельцу, **задачи 12–16 не выполняются**.
- **Издержки — предупреждение, не стоп.** Настоящий круг TGKA **0.0663%** (шаг 0.000002 ₽ при
  медианной цене 0.006036 ₽) против моделируемых 0.1%: модель консервативна в 1.5 раза.
  Чувствительность baseline: 0.1% → 1.967, 0.2% → 1.614, 0.3% → 1.324, 0.4% → 1.080.
- **Правило прода:** литерал ставится и TGKA заводится в `RSI_PULLBACK_TICKERS` двадцать третьим
  **независимо от того, взята планка или нет**. Стоп-условие это правило перевешивает.
- **Дефолты ядра** (`core.DefaultParams()`), поверх которых считают все темы: `RSIPeriod 4`,
  `RSILower 30`, `RSIUpper 70`, `EMAFast 10`, `EMASlow 100`, `DailyATRPeriod 14`, `UseDayATRGate 1`,
  `FreshDayATR 0`, `SpentDayATR 0.8`, `StopDailyATR 0.5`, `TPDailyATR 0.6`, `UseVolume 0`,
  `VolBaseDays 14`, `VolLookbackBars 3`, `VolMult 1.2`, `UseRSIExit 1`, `UseTrail 0`,
  `TrailDailyATR 0`. Контрольный прогон дефолтов: **167 сделок, PF 1.967, net +75 401.33 ₽**, win
  rate 66.47%, max DD 10.21%, выходы RSI 104 / SL 35 (21.0%) / TP 28 — **первое место** в ряду
  baseline каталога. На 24 месяцах — 131 сделка, PF 2.518.
- **Полугодия baseline** (нужны в Task 11): 2023-09..2024-03 — 13 сделок, PF 0.418, −7 042;
  2024-03..2024-09 — 19, 1.363, +3 200; 2024-09..2025-03 — 49, 2.769, +33 518;
  2025-03..2025-09 — 26, 2.691, +13 855; 2025-09..2026-03 — 31, 1.125, +2 383;
  2026-03..2026-09 — 29, **3.701**, +29 487. Одно убыточное из шести.
- **Ликвидность — главный риск.** Медиана оборота будних дней (лот 100 000): 36.5 млн ₽ (36 мес),
  25.7 (24 мес), **22.4 (12 мес)**; по годам 106.8 → 44.1 → 27.7 → **26.0 (2026)**. Гейт вселенной
  скринера 50 млн пройден **средним** (75.9 млн, раздуто 2023 годом), по медиане 12 месяцев **не
  проходится**. Прецеденты: NKHP (14 млн) и LENT (38 млн) заведены. Выходная сессия: 254 дня,
  медиана 1.9 млн ₽. Дневной ATR(14) медиана 2.98%.
- **Сессия расширилась внутри окна**: медиана баров в буднем дне 19 → 29 → 34 → 35. Отсюда
  контрольный прогон 24/12/3 и пункт 3 стоп-условия.
- **Каждый `_comment` сетки** обязан содержать: что тема меряет и сколько в ней прогонов; замер, из
  которого получена каждая ось, с предупреждением о крае; полную команду запуска с путём
  `data/params/rsi_pullback/tgka/<файл>` (этого требует `TestRSIPullbackCalFilesValid`); место под
  строку `РЕЗУЛЬТАТ ПРОГОНА 2026-09-03: …`.
- **Пер-тикерных записей в `docs/rsi_pullback/` не делается** (правило CLAUDE.md). Пер-тикерный
  разбор живёт в доке пакета `strategy/tgka` и в `_comment` файлов сеток.
- **Каталог `reports/` — в `.gitignore`.**
- **Коммит по завершении каждой задачи**, сообщения на русском в стиле существующей истории.
- **Ветка:** `feat/tgka-pullback-prep` от `feat/sofl-pullback-prep` (`14419e6`).

---

### Task 1: Каталог одиннадцати сеток со сторожевым тестом осей

**Files:**
- Create: `data/params/rsi_pullback/tgka/cal_screen.json`, `cal_entry.json`, `cal_trend.json`,
  `cal_trend_low.json`, `cal_day.json`, `cal_day_spent.json`, `cal_volume.json`,
  `cal_vol_window.json`, `cal_risk.json`, `cal_exit.json`, `cal_trail.json`
- Create: `internal/service/backtest/rsi_pullback_tgka_grid_test.go`

- [ ] **Step 1: Написать падающий сторожевой тест осей** (`TestTGKAGridsStayWide` — проверяет
  НАЛИЧИЕ обязательных значений, лишние не запрещает; `TestTGKAEntryGridKeepsRSIUpperAboveRSILower`;
  `TestTGKATrendGridsKeepFastBelowSlow` — по обоим файлам тренда).
- [ ] **Step 2: Убедиться, что тест падает** (`go test ./internal/service/backtest/ -run TestTGKA`).
- [ ] **Step 3: Создать одиннадцать файлов сеток** по таблице спеки, с замерами в `_comment`.
- [ ] **Step 4: `go test ./internal/service/backtest/ -run 'TestTGKA|TestRSIPullback'` — PASS.**
- [ ] **Step 5: Коммит** `feat(rsi_pullback): каталог сеток TGKA`.

---

### Task 2: Пакет `strategy/tgka` в состоянии «калибровка не проводилась»

**Files:**
- Create: `internal/service/trading_strategy/rsi_pullback/strategy/tgka/tgka.go`, `tgka_test.go`
- Modify: `internal/service/backtest/rsi_pullback_registry.go`, `rsi_pullback_registry_test.go`

- [ ] **Step 1:** тест `TestParamsTrackTheBaselineUntilCalibrated` + `TestTickerIsTGKA`.
- [ ] **Step 2:** убедиться, что падает.
- [ ] **Step 3:** создать пакет с честной шапкой (состояние «КАЛИБРОВКА НЕ ПРОВОДИЛАСЬ», окно и
  схема, оба риск-гейта с числами, априор — сильнейший в каталоге, особенности бумаги: копеечная
  цена с масштабированным шагом, лот 100 000, падающая ликвидность, расширившаяся сессия).
- [ ] **Step 4:** завести тикер в `rsiPullbackRegistry` (алиас `rsipullbacktgka`, алфавитный
  порядок) + тест `TestRSIPullbackTGKATracksBaseline`.
- [ ] **Step 5:** `go test ./internal/service/backtest/ ./internal/service/trading_strategy/rsi_pullback/...`
- [ ] **Step 6: Коммит** `feat(rsi_pullback): пакет TGKA до калибровки`.

---

### Task 3: Тема `screen` — цена двух гейтов и проверка числа фолдов

```bash
go run ./cmd/backtest -ticker TGKA -strategy rsi_pullback -interval Minutes30 \
  -calibrate data/params/rsi_pullback/tgka/cal_screen.json -out ./reports/TGKA_screen \
  -months 36 -train-months 12 -test-months 6 -min-trades 1 -metric profit_factor
```

- [ ] **Step 1:** прогнать тему.
- [ ] **Step 2:** проверить «Фолдов: 4» — иначе остановиться и доложить владельцу.
- [ ] **Step 3:** выписать pooled OOS PF, сделки пула, пофолдовые `UseDayATRGate` / `UseVolume`,
  пофолдовые OOS PF / сделки / MaxDD%. Ожидание из точечного замера: `UseDayATRGate=1` почти
  наверняка (без гейта 1.053/447 при DD 45 393), по `UseVolume` согласия может не быть.
- [ ] **Step 4:** дописать строку `РЕЗУЛЬТАТ ПРОГОНА` в `_comment`.
- [ ] **Step 5: Коммит** `feat(rsi_pullback): TGKA, тема screen — цена гейтов`.

---

### Task 4: Тема `entry` — ключевая

```bash
go run ./cmd/backtest -ticker TGKA -strategy rsi_pullback -interval Minutes30 \
  -calibrate data/params/rsi_pullback/tgka/cal_entry.json -out ./reports/TGKA_entry \
  -months 36 -train-months 12 -test-months 6 -min-trades 20 -metric profit_factor
```

- [ ] **Step 1:** прогнать тему (504 прогона × 4 фолда).
- [ ] **Step 2:** проверить планку по критериям A (pooled OOS PF ≥ 1.5 при ≥ 20 сделках) и B
  (`RSILower` одинаков в ≥ 3 фолдах из 4). Записать «взят»/«провален» отдельно по каждому.
- [ ] **Step 3:** проверить края расширенной оси периода — победил ли `RSIPeriod` 10 (точечно
  2.094 на 38 сделках, порог `-min-trades 20` должен его отсечь) или 2 (1.303/405). Победа любого
  края пишется прямым текстом.
- [ ] **Step 4:** дописать результат в `_comment`, **Коммит** `feat(rsi_pullback): TGKA, тема entry`.

---

### Task 5: Темы `trend` и `trend_low` — вторая ключевая и нижний угол

```bash
go run ./cmd/backtest -ticker TGKA -strategy rsi_pullback -interval Minutes30 \
  -calibrate data/params/rsi_pullback/tgka/cal_trend.json -out ./reports/TGKA_trend \
  -months 36 -train-months 12 -test-months 6 -min-trades 20 -metric profit_factor

go run ./cmd/backtest -ticker TGKA -strategy rsi_pullback -interval Minutes30 \
  -calibrate data/params/rsi_pullback/tgka/cal_trend_low.json -out ./reports/TGKA_trend_low \
  -months 36 -train-months 12 -test-months 6 -min-trades 20 -metric profit_factor
```

- [ ] **Step 1:** прогнать обе темы.
- [ ] **Step 2:** планка проверяется **только по канонической** `cal_trend` (критерии A и B).
- [ ] **Step 3:** сравнить `trend_low` с `trend`: поле тренда идёт из `trend_low` **только** при
  большинстве ≥ 3/4 **и** превосходстве pooled OOS над канонической темой. Иначе — из `trend` (или
  дефолт ядра, если большинства нет). Решение пишется прямым текстом.
- [ ] **Step 4:** проверить края: победа `EMASlow` 50 (нижний край канонической оси) или `EMAFast` 3
  (нижний край) — подтверждение точечного замера; победа верхних краёв (250, 40) — против него.
- [ ] **Step 5:** дописать результаты в оба `_comment`, **Коммит**
  `feat(rsi_pullback): TGKA, темы trend и trend_low`.

---

### Task 6: Темы `day` и `day_spent` — дневной гейт

```bash
go run ./cmd/backtest -ticker TGKA -strategy rsi_pullback -interval Minutes30 \
  -calibrate data/params/rsi_pullback/tgka/cal_day.json -out ./reports/TGKA_day \
  -months 36 -train-months 12 -test-months 6 -min-trades 20 -metric profit_factor

go run ./cmd/backtest -ticker TGKA -strategy rsi_pullback -interval Minutes30 \
  -calibrate data/params/rsi_pullback/tgka/cal_day_spent.json -out ./reports/TGKA_day_spent \
  -months 36 -train-months 12 -test-months 6 -min-trades 20 -metric profit_factor
```

- [ ] **Step 1:** прогнать обе темы.
- [ ] **Step 2:** сверить темы между собой: расхождение пофолдовых `SpentDayATR` — взаимодействие
  веток, пишется прямым текстом.
- [ ] **Step 3:** проверить край 1.5 (точечно 2.471 на 28 сделках) — отсёк ли его порог внутри
  фолдов.
- [ ] **Step 4:** дописать результаты, **Коммит** `feat(rsi_pullback): TGKA, темы day и day_spent`.

---

### Task 7: Тема `volume`

```bash
go run ./cmd/backtest -ticker TGKA -strategy rsi_pullback -interval Minutes30 \
  -calibrate data/params/rsi_pullback/tgka/cal_volume.json -out ./reports/TGKA_volume \
  -months 36 -train-months 12 -test-months 6 -min-trades 20 -metric profit_factor
```

- [ ] **Step 1:** прогнать тему.
- [ ] **Step 2:** выписать пофолдовых победителей `VolMult` и `VolBaseDays`; проверить, победил ли
  край `VolBaseDays` 3 или 20.
- [ ] **Step 3:** записать место TGKA в каталожной гипотезе (оборот 22–36 млн, рядом с NKHP).
- [ ] **Step 4:** дописать результат, **Коммит** `feat(rsi_pullback): TGKA, тема volume`.

---

### Task 8: Тема `vol_window`

```bash
go run ./cmd/backtest -ticker TGKA -strategy rsi_pullback -interval Minutes30 \
  -calibrate data/params/rsi_pullback/tgka/cal_vol_window.json -out ./reports/TGKA_vol_window \
  -months 36 -train-months 12 -test-months 6 -min-trades 20 -metric profit_factor
```

- [ ] **Step 1:** прогнать тему.
- [ ] **Step 2:** сверить `VolMult` с темой `volume`; расхождение = форма гейта не откалибрована,
  `VolMult` идёт дефолтом ядра.
- [ ] **Step 3:** проверить верхний край 32 (точечно равен 48 побайтово — ось насыщается).
- [ ] **Step 4:** дописать результат, **Коммит** `feat(rsi_pullback): TGKA, тема vol_window`.

---

### Task 9: Тема `risk` — стоп, цель и обязательная проверка риск-гейта A

**Files:** также `plateau_stop_10.json`, `plateau_stop_20.json`

```bash
go run ./cmd/backtest -ticker TGKA -strategy rsi_pullback -interval Minutes30 \
  -calibrate data/params/rsi_pullback/tgka/cal_risk.json -out ./reports/TGKA_risk \
  -months 36 -train-months 12 -test-months 6 -min-trades 20 -metric profit_factor
```

- [ ] **Step 1:** прогнать тему.
- [ ] **Step 2:** применить риск-гейт A: если большинство фолдов выбрало `StopDailyATR` > 1.0,
  победитель темы **отвергается гейтом**, в точку идёт 1.0 (43.6% дней), цена решения в PF пишется
  прямым текстом.
- [ ] **Step 3:** снять два зонда капкана (`StopDailyATR` 1.0 и 2.0 при остальных полях темы) на
  полном окне через `-params`, выписать PF, сделки, **max DD**, выходы RSI/SL/TP с долей SL,
  удержание (медиана, p90, максимум), долю ночёвок, долю выходов в выходную сессию.
- [ ] **Step 4:** записать, что на TGKA подпись капкана «PF вверх, DD вверх» **не воспроизводится**
  (точечно 2.0 → 2.740 при DD 8 303 против 1.967/10 388), и потому решает гейт A, а не анатомия.
- [ ] **Step 5:** дописать результат, **Коммит**
  `feat(rsi_pullback): TGKA, тема risk и проверка риск-гейта стопа`.

---

### Task 10: Темы `exit` и `trail`

```bash
go run ./cmd/backtest -ticker TGKA -strategy rsi_pullback -interval Minutes30 \
  -calibrate data/params/rsi_pullback/tgka/cal_exit.json -out ./reports/TGKA_exit \
  -months 36 -train-months 12 -test-months 6 -min-trades 20 -metric profit_factor

go run ./cmd/backtest -ticker TGKA -strategy rsi_pullback -interval Minutes30 \
  -calibrate data/params/rsi_pullback/tgka/cal_trail.json -out ./reports/TGKA_trail \
  -months 36 -train-months 12 -test-months 6 -min-trades 20 -metric profit_factor
```

- [ ] **Step 1:** прогнать обе темы.
- [ ] **Step 2:** проверить, не обходит ли трейл риск-гейт A (урок AFKS): дистанция трейла ниже
  принятого стопа делает её эффективной защитой, и гейт применяется к ней.
- [ ] **Step 3:** применить правило упрощения: дистанция ≥ 1.0 → `UseTrail 0`, `TrailDailyATR 0`.
- [ ] **Step 4:** если тема выбрала `UseRSIExit 0` — записать отдельной строкой: точечно это
  1.741/158 при max DD 18 443 против 1.967/10 388, то есть выше net и вдвое выше риск.
- [ ] **Step 5:** дописать результаты, **Коммит** `feat(rsi_pullback): TGKA, темы exit и trail`.

---

### Task 11: Сборка точки, два риск-гейта, два walk-forward и три пункта стоп-условия

**Files:** `plateau_point.json`, соседи плато `plateau_<поле>_<значение>.json`,
`docs/superpowers/plans/task-11-report-tgka.md`

- [ ] **Step 1:** собрать точку по правилу большинства (≥ 3 из 4; иначе дефолт ядра). Для каждого
  поля записать голоса, решение и случайность совпадения с дефолтом.
- [ ] **Step 2:** применить риск-гейт A к собранной точке (эффективная защита, потолок 1.0).
- [ ] **Step 3:** прогнать точку схемой 36/12/6 (`-calibrate plateau_point.json`), выписать pooled
  OOS PF, сделки пула, пофолдовые in-sample → OOS PF / сделки / NetPnL% / MaxDD%.
- [ ] **Step 4:** проверить пункты 1 и 2 стоп-условия. Провал → **остановиться**, задачи 12–16 не
  выполнять.
- [ ] **Step 5:** прогнать контрольную схему 24/12/3 (пункт 3). Baseline на том же куске даёт 2.518
  на 131 сделке — если точка ниже baseline, это записывается прямым текстом.
- [ ] **Step 6:** применить риск-гейт B на полной истории (`-params plateau_point.json`), потолок
  max DD **13.27%**.
- [ ] **Step 7:** снять анатомию и полугодия точки; сравнить с baseline по пяти числам.
- [ ] **Step 8:** снять две строки издержек (`-commission 0.001` и `0.0015`) — предупреждение, не
  стоп.
- [ ] **Step 9:** снять соседей плато по каждому полю, принятому темой большинством (±один узел
  оси); совпадение побайтово = инертная ось.
- [ ] **Step 10:** написать рабочую записку `docs/superpowers/plans/task-11-report-tgka.md`.
- [ ] **Step 11: Коммит** `feat(rsi_pullback): TGKA, принятая точка и её замеры`.

---

### Task 12: Литерал в пакете и снимок

**Задача выполняется только если ни один пункт стоп-условия не сработал.**

- [ ] **Step 1:** заменить `TestParamsTrackTheBaselineUntilCalibrated` на
  `TestDefaultParamsIsTheCalibratedSnapshot` (все восемнадцать полей) +
  `TestDefaultParamsDiffersFromCoreBaseline`.
- [ ] **Step 2:** убедиться, что падает.
- [ ] **Step 3:** поставить литерал принятой точки.
- [ ] **Step 4:** заменить тест реестра на `TestRSIPullbackTGKAServesTheCalibratedPoint`.
- [ ] **Step 5:** тесты + `./bin/golangci-lint run ./internal/...`.
- [ ] **Step 6: Коммит** `feat(rsi_pullback): TGKA откалиброван — литерал вместо отслеживания baseline`.

---

### Task 13: Реестр живого раннера

- [ ] **Step 1:** импорт и строка `tgka.Ticker: tgka.DefaultParams()` в `paramsByTicker`.
- [ ] **Step 2:** абзац комментария карты в том же англоязычном стиле: вердикт по планке, оба
  риск-гейта и их цена, принятые риски (падающая ликвидность 22–26 млн ₽ против гейта 50, выходная
  сессия 1.9 млн ₽, режим −64% за окно, расширившаяся сессия).
- [ ] **Step 3:** `go test ./internal/service/trading_strategy/rsi_pullback/live/...`
- [ ] **Step 4: Коммит** `feat(rsi_pullback): TGKA в реестре живого раннера`.

---

### Task 14: Боевая вселенная

**Files:** `internal/config/rsi_pullback.go`, `internal/config/rsi_pullback_test.go`,
`env/prod.env`, `env/prod.env.example`, `env/local.env.example`, `docs/rsi_pullback/live.md`

- [ ] **Step 1:** перечитать числа Task 11 Steps 4–6 — стоп-условие не сработало, оба гейта пройдены.
- [ ] **Step 2:** добавить `"TGKA"` в `want` теста конфига двадцать третьим.
- [ ] **Step 3:** убедиться, что тест падает.
- [ ] **Step 4:** дописать `"TGKA"` в `Tickers` + комментарий-абзац в стиле соседних; дописать
  `,TGKA` в три env-файла.
- [ ] **Step 5:** обновить значение дефолта `RSI_PULLBACK_TICKERS` в таблице `docs/rsi_pullback/live.md`
  (только значение — пер-тикерного ничего).
- [ ] **Step 6:** `go test ./internal/config/ ./internal/service/trading_strategy/rsi_pullback/...`
- [ ] **Step 7: Коммит** `feat(rsi_pullback): завести TGKA в боевую вселенную`.

---

### Task 15: Дока пакета — разбор калибровки и принятый риск

- [ ] **Step 1:** переписать шапку пакета в итог калибровки: вердикт по планке по каждому критерию
  обеих ключевых тем, точка поле за полем с голосами фолдов, оба walk-forward, две строки издержек,
  соседи плато с пометкой инертных осей, шесть полугодий, анатомия против baseline, результат обоих
  риск-гейтов и их цена. Отдельными абзацами — принятые риски и условие пересмотра:
  1) **ликвидность**: медиана 22.4 млн ₽ за 12 месяцев против гейта вселенной 50 млн, падение
  106.8 → 26.0 млн по годам; риск исполнения, калибровкой не устраняемый; **условие пересмотра:
  падение медианы ниже 15 млн ₽ — вывести тикер из боевой вселенной, не дожидаясь планового цикла**;
  2) **выходная сессия** с медианным оборотом 1.9 млн ₽ и долей выходов точки;
  3) **режим −64% за окно** при двух растущих полугодиях из шести — стратегия проверена в основном
  против падения;
  4) **сессия расширилась внутри окна** (19 → 35 баров) — отсюда контрольный прогон 24/12/3;
  5) **капкан широкого стопа**, который на TGKA не детектируется просадкой, и роль гейта A;
  6) **высокий baseline измерен в основном на 2023–2024 годах**, когда оборот был вчетверо выше.
- [ ] **Step 2:** `go test ./internal/service/trading_strategy/rsi_pullback/...`
- [ ] **Step 3:** механический вывод в `docs/rsi_pullback/strategy.md` — **только если** TGKA дал
  вывод о механике (кандидат назван заранее: капкан широкого стопа, не детектируемый просадкой,
  делает гейт выживаемости обязательным, а не дополнительным). Пер-тикерных чисел, дат и вердиктов
  в `docs/rsi_pullback/` не писать.
- [ ] **Step 4: Коммит** `docs(rsi_pullback): разбор калибровки TGKA и принятый риск`.

---

### Task 16: Финальная проверка

- [ ] **Step 1:** `go run ./cmd/pullparity -tickers TGKA -months 36` — **ноль расхождений**.
- [ ] **Step 2:** `./bin/mage ci` — зелёный.
- [ ] **Step 3:** `git status --porcelain | grep -c '^.. reports/'` — `0`.
- [ ] **Step 4:** финальный коммит, если остались правки.

---

## Self-Review

**Покрытие спеки:** одиннадцать тем → Tasks 3–10; каноническая процедура (без якоря) → Global
Constraints; сетки и их ширина → Task 1 (+ сторожевые тесты); пакет и реестр бэктеста → Task 2;
правило сборки точки → Task 11 Step 1; риск-гейт A → Task 9 Step 2, Task 10 Step 2, Task 11 Step 2;
риск-гейт B → Task 11 Step 6; третий контур → Task 11 Step 7; три пункта стоп-условия → Task 11
Steps 4–5; контрольный прогон 24/12/3 → Task 11 Step 5; издержки как предупреждение → Task 11
Step 8; соседи плато → Task 11 Step 9; литерал → Task 12; реестр живого раннера → Task 13; боевая
вселенная двадцать третьим тикером → Task 14; пер-тикерная запись → Task 15; `pullparity` и
`mage ci` → Task 16; проверка числа фолдов (ловушка ASTR) → Task 3 Step 2.

**Плейсхолдеры:** один — литерал в Task 12 (точка известна только после Task 11). Указано, откуда
берётся значение и по какому правилу.

---

## Итог исполнения плана (2026-09-03)

Tasks 1–11 выполнены. **Tasks 12–16 ОТМЕНЕНЫ решением владельца 2026-09-03** после трёх провалов
подряд: стоп-условие сработало в каждом круге, и тикер закрыт без заведения в прод (прецеденты HEAD
и AFKS).

- Круг 1 (каноническая процедура, 36/12/6): точка 1.789/156, контроль 24/12/3 — **0.899/66**.
- Круг 2 (окно B, расчётная 24/12/3): точка вырождена по входу — **1 сделка в пуле OOS**, 6 сделок
  за три года. Здесь **впервые сработал риск-гейт A**: единогласный стоп 2.0 заменён на 1.0.
- Круг 3 (слепой holdout 2026-03-03…2026-09-03, голоса только из фолдов 1–3): точка **0.815/25,
  net −1 070** против дефолтов ядра **3.709/29, net +20 223** на том же куске.

**Дополнение того же дня:** после доклада владелец выбрал не закрытие, а **заведение TGKA в боевую
вселенную двадцать третьим НА ДЕФОЛТАХ ЯДРА** — единственной конфигурации, выигравшей все три среза.
Задачи 13–16 выполнены в этой редакции: запись в живом реестре, TGKA в `RSI_PULLBACK_TICKERS` и трёх
env-файлах, значение дефолта в `docs/rsi_pullback/live.md`, переписанная дока пакета. Task 12
(литерал) выполнен вырожденно: литерала нет, пакет отдаёт `core.DefaultParams()`, и это пинит
`TestParamsStayAtTheCoreBaselineByDesign`. Сторожевой тест боевой вселенной НЕ ослаблен: заведено
ИМЕННОЕ исключение `baselineByDesignTickers` с причиной, для таких тикеров тест проверяет обратное
условие, а отдельный тест запрещает пустую причину и забытую строку. `pullparity -months 24` — ноль
расхождений; `./bin/mage ci` зелёный. Полный разбор — `docs/superpowers/plans/task-11-report-tgka.md`;
вердикт, цена решения и условие пересмотра — в док-комментарии пакета.
