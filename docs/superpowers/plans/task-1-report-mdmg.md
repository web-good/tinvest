# Task 1 report: пакет `strategy/mdmg` до калибровки, проверка скрипта журнала и пересъёмка baseline

## Реализация

- `internal/service/trading_strategy/rsi_pullback/strategy/mdmg/mdmg.go` — новый пакет: константа
  `Ticker = "MDMG"`, `DefaultParams()` возвращает `core.DefaultParams()` без изменений (СОСТОЯНИЕ:
  калибровка не проводилась). Doc-комментарий пакета — дословно по Step 3 брифа.
- `internal/service/trading_strategy/rsi_pullback/strategy/mdmg/mdmg_test.go` — два теста:
  `TestParamsTrackTheBaselineUntilCalibrated` (снимок держит baseline ядра), `TestTickerIsMDMG`.
- `internal/service/backtest/rsi_pullback_registry.go` — добавлен импорт `rsipullbackmdmg` и запись
  в `rsiPullbackRegistry`. Отклонение от буквы брифа: Step 4 требовал ставить строку карты «после
  строки `rsipullbackragr`» — по прямому прецеденту `docs/superpowers/plans/task-1-report-ragr.md`
  (тот же дефект брифа: контроллер тогда постановил, что карта `rsiPullbackRegistry` обязана
  оставаться строго алфавитной, отклонив буквальное «после svcb»). Применяю то же постановление
  здесь без повторного цикла ревью: строка поставлена на алфавитное место — между `magn` и `mvid` —
  в обоих блоках (импорт и карта), а не после `ragr`. `gofmt -w` подтвердил выравнивание столбцов
  (повторный `gofmt -l` пуст).
- `internal/service/backtest/rsi_pullback_registry_test.go` — добавлен импорт `rsipullbackmdmg`
  (алфавитно, между `magn` и `mvid`) и тест `TestRSIPullbackMDMGTracksBaseline` сразу после
  `TestRSIPullbackRAGRIsRegisteredAndCalibrated`, как указано в брифе. Эта часть брифа (порядок
  тестов В ФАЙЛЕ, не карты) не переносится на алфавит — по тому же прецеденту RAGR порядок тестов
  в файле хронологический/нарративный, а не алфавитный.
- `docs/superpowers/plans/task-1-report-mdmg.md` (этот файл).

Скрипт `reports/_analysis/mdmg_journal.py` уже существовал (протокол разведки, вне git) и не
менялся — Step 5 только проверяет его.

## TDD

RED (Step 2 брифа):

```
$ go test ./internal/service/trading_strategy/rsi_pullback/strategy/mdmg/ -v
internal/service/trading_strategy/rsi_pullback/strategy/mdmg/mdmg_test.go:14:18: undefined: DefaultParams
internal/service/trading_strategy/rsi_pullback/strategy/mdmg/mdmg_test.go:20:5: undefined: Ticker
internal/service/trading_strategy/rsi_pullback/strategy/mdmg/mdmg_test.go:21:38: undefined: Ticker
FAIL	tinvest/internal/service/trading_strategy/rsi_pullback/strategy/mdmg [build failed]
FAIL
```

Пакета не существовало — падение ожидаемое (build failed, не логическое FAIL).

GREEN (после Step 3 — написан `mdmg.go`):

```
$ go test ./internal/service/trading_strategy/rsi_pullback/strategy/mdmg/ -v
=== RUN   TestParamsTrackTheBaselineUntilCalibrated
--- PASS: TestParamsTrackTheBaselineUntilCalibrated (0.00s)
=== RUN   TestTickerIsMDMG
--- PASS: TestTickerIsMDMG (0.00s)
PASS
ok  	tinvest/internal/service/trading_strategy/rsi_pullback/strategy/mdmg	0.007s
```

Step 4 — полный прогон пакетных и реестровых тестов после регистрации в `rsi_pullback_registry.go`
(с картой на алфавитном месте):

```
$ gofmt -l internal/
$ go test ./internal/service/trading_strategy/rsi_pullback/... ./internal/service/backtest/ -run 'MDMG|RSIPullback' -v
...
=== RUN   TestRSIPullbackMDMGTracksBaseline
--- PASS: TestRSIPullbackMDMGTracksBaseline (0.00s)
...
PASS
ok  	tinvest/internal/service/backtest	0.369s
```

`gofmt -l` не напечатал ничего. Дополнительно прогнаны `go build ./internal/... ./pkg/... ./cmd/...`
(чисто) и `go test ./internal/service/trading_strategy/rsi_pullback/... ./internal/service/backtest/...`
(все пакеты `ok`, включая `mdmg`).

## Гейт D скрипта журнала (Step 5)

Синтетическая проверка правила `held_through` на четырёх датах отсечек:

```
$ python3 - <<'EOF'
... (четыре assert вокруг 2025-10-20 + сверка списка EX_DATES)
EOF
OK
```

Прогон на эталонном отчёте разведки:

```
$ python3 reports/_analysis/mdmg_journal.py reports/MDMG_prep/wf/wf_24_0_0_0.0005/MDMG_rsi_pullback_Minutes30_20260926_230928_best.md
```

Совпало с эталоном брифа дословно: 134 сделки, PF 1.546, «ГЕЙТ C: входов 02-06 2», все четыре
отсечки по 0 сделок, хвост 6 — 22/1.727, хвост 12 — 52/1.663.

## Дата прогонов Step 6 и Step 7

Все прогоны выполнены последовательно **2026-09-26**, в интервале 23:42:00 — 23:43:36 МСК (`date`
проверен перед первым и после последнего прогона — календарная дата не менялась).

## Пересъёмка baseline (Step 6)

```
$ go run ./cmd/backtest -ticker MDMG -strategy rsi_pullback -interval Minutes30 \
  -months 24 -metric profit_factor -out ./reports/MDMG_base
report: reports/MDMG_base/MDMG_rsi_pullback_Minutes30_20260926_234233.md (trades=134, net=28648.01, PF=1.546)
```

Журнал переснятого прогона совпал с эталоном разведки **буквально по каждому полю**:

| Показатель | Переснятое | Эталон (спека 2026-09-26) |
|---|---|---|
| Окно | 2024-09-26 23:42 — 2026-09-26 23:42 | 2024-09-26 23:09 — 2026-09-26 23:09 |
| Сделок | 134 | 134 |
| PF | 1.546 | 1.546 |
| Net | +28648.01 ₽ (28.65%) | +28648.01 ₽ (28.65%) |
| Max DD | 9153.17 ₽ (7.54%) | 9153.17 ₽ (7.54%) |
| Win rate | 69.40% | 69.40% |
| Expectancy | +213.79 ₽ | +213.79 ₽ |
| Лучшая / худшая | +3182.76 / −2871.09 ₽ | +3182.76 / −2871.09 ₽ |
| Exposure | 5.06% | 5.06% |
| Выходы | RSI 82 (61.2%), SL 33 (24.6%), TP 19 (14.2%) | те же |
| Удержание медиана/p90/макс | 9 / 19 / 33 | те же |
| Ночёвок / 2+ дня | 46 (34.3%) / 2 | те же |
| Выходная сессия входы/выходы | 0 / 5 (+716.46 ₽) | те же |
| Гейт C (входы/выходы 02-06) | 2 / 6 (+1407.87 ₽) | те же |
| Гейт D (все 4 отсечки) | 0 сделок | 0 сделок |
| 2024 (огрызок) | 19, −2348.11 ₽, PF 0.735 | те же |
| **2025** | 83, **+19457.38 ₽**, PF 1.627 | те же |
| **2026** | 32, **+11538.81 ₽**, PF 1.914 | те же |
| Полугодия | 2024H2 −2348.11 (19, 0.735); 2025H1 +18764.74 (36, 2.933); 2025H2 +692.64 (47, 1.032); 2026H1 +3355.40 (19, 1.567); 2026H2 +8183.41 (13, 2.219) | те же |
| Хвост 6 (с 2026-03-26) | 22, +7894.24 ₽, PF **1.727** | те же |
| Хвост 12 (с 2025-09-26) | 52, +13522.47 ₽, PF **1.663** | те же |

Сверка сделка в сделку на пересечении окон (33-минутный сдвиг окна не изменил состав сделок):

```
новое окно: 2024-09-26 23:42 — 2026-09-26 23:42
эталон на пересечении 134, новый 134
расхождений на пересечении: 0 (длина равна)
выпали в начале: 0, net +0.00
добавились в конце: 0, net +0.00
```

**Вердикт по решающему правилу.** Ноль расхождений: длины совпали (134=134), не выпало и не
добавилось ни одной сделки, ни по `(entry, exit)`, ни по `pnl`. Условие «расхождения только в
первые две недели нового окна» удовлетворено тривиально (расхождений нет вообще) —
эскалация к владельцу не требуется. **Baseline полностью совпал со спекой** — пересчёт гейта B по
переснятому max DD не требуется: значение то же (9153.17 ₽ / 7.54%).

## Пересъёмка walk-forward (Step 7)

Файл одной точки — дефолты ядра (`reports/MDMG_base/base.json`, `RSILower: [30]`, `keepTop: 1`).
Три схемы прогнаны строго последовательно, без `-refresh`, в том же календарном дне.

**24/12/3 (сборка и вердикт):**

```
$ go run ./cmd/backtest -ticker MDMG -strategy rsi_pullback -interval Minutes30 \
  -calibrate reports/MDMG_base/base.json -out ./reports/MDMG_base_wf123 \
  -months 24 -train-months 12 -test-months 3 -min-trades 1 -metric profit_factor
walk-forward: ... (folds=4, pooled PF=1.659, compounded=12.83%)
```

Пул: 52 сделки, pooled OOS PF **1.659**, expectancy 239.21 ₽. Фолды (OOS PF/сделок/NetPnL%): 1 —
1.086/17/0.56%, 2 — 2.650/13/5.10%, 3 — **0.927/9/-0.26%**, 4 — 2.228/13/7.04%. Совпало с опорными
числами брифа (1.659/52; фолды 1.086/17, 2.650/13, 0.927/9, 2.228/13) буквально.

**24/15/3 (пункт 3 стоп-условия):**

```
$ go run ./cmd/backtest -ticker MDMG -strategy rsi_pullback -interval Minutes30 \
  -calibrate reports/MDMG_base/base.json -out ./reports/MDMG_base_wf153 \
  -months 24 -train-months 15 -test-months 3 -min-trades 1 -metric profit_factor
walk-forward: ... (folds=3, pooled PF=1.903, compounded=12.04%)
```

Пул: 35 сделок, pooled OOS PF **1.903**. Фолды: 2.661/13, 0.922/9, 2.216/13. Совпало (1.903/35).

**24/12/3, `-commission 0.001` (пункт 4 стоп-условия):**

```
$ go run ./cmd/backtest -ticker MDMG -strategy rsi_pullback -interval Minutes30 \
  -calibrate reports/MDMG_base/base.json -out ./reports/MDMG_base_c001 \
  -months 24 -train-months 12 -test-months 3 -min-trades 1 -metric profit_factor -commission 0.001
walk-forward: ... (folds=4, pooled PF=1.324, compounded=6.14%)
```

Пул: 52 сделки, pooled OOS PF **1.324**. Фолды: 0.806/17, 2.023/13, 0.689/9, 1.927/13. Совпало
(1.324/52).

Все три схемы прогнаны с идентичной алфавитной сверкой «Фолдов: 4» / «Фолдов: 3» в шапке отчёта —
ловушка ASTR не сработала.

## Объединённый OOS PF фолдов 3–4 схемы 24/12/3 (формула Task 11 Step 13)

Фолд 3: PF 0.927, OOS NetPnL% −0.26% → net = −260.00 ₽; GL = net/(PF−1) = −260.00/(−0.073) =
3561.64 ₽; GP = PF·GL = 3301.64 ₽.

Фолд 4: PF 2.228, OOS NetPnL% 7.04% → net = 7040.00 ₽; GL = 7040.00/(2.228−1) = 5732.90 ₽;
GP = PF·GL = 12772.90 ₽.

Комбинированный PF = (GP₃+GP₄)/(GL₃+GL₄) = (3301.64+12772.90)/(3561.64+5732.90) =
16074.54/9294.54 = **1.729**.

Совпадает со спекой («Объединённый OOS PF фолдов 3–4 дефолтов — 1.729»). Знак объединённого OOS PF
(>1) совпадает со знаком хвоста 6 (PF 1.727 > 1) — расхождения нет, эскалация по пункту 6 не
требуется.

## Семь пунктов стоп-условия для дефолтов (переснятые числа)

1. pooled OOS PF 24/12/3 = 1.659 ≥ 1.0 — **пройден**.
2. сделок в пуле OOS 24/12/3 = 52 ≥ 20 — **пройден**.
3. pooled OOS PF 24/15/3 = 1.903 ≥ 1.0 — **пройден**.
4. pooled OOS PF 24/12/3 при `-commission 0.001` = 1.324 ≥ 1.0 — **пройден**.
5. годы 2025 (+19457.38 ₽) и 2026 (+11538.81 ₽) оба прибыльны — **пройден**.
6. хвост 6 (22 сделки, PF 1.727 ≥ 1.0, ≥ 5 сделок) и хвост 12 (52, PF 1.663 ≥ 1.0) — **пройден**;
   знак хвоста 6 совпал со знаком объединённого OOS PF фолдов 3–4 (оба > 1) — расхождения нет.
7. гейт D — 0 сделок через все четыре отсечки; гейт C — 0 входов в выходные, 2 входа 02-06 (равно
   планке дефолтов — это и есть baseline) — **пройден**.

**Вердикт: дефолты ДОПУЩЕНЫ кандидатом.** Ни один из семи пунктов не сработал; переснятые числа
дословно совпали с числами спеки от 2026-09-26. По §5.15 спеки дефолты остаются полноправным
кандидатом в прод наравне с будущей собранной точкой.

## Файлы

- `internal/service/trading_strategy/rsi_pullback/strategy/mdmg/mdmg.go`
- `internal/service/trading_strategy/rsi_pullback/strategy/mdmg/mdmg_test.go`
- `internal/service/backtest/rsi_pullback_registry.go`
- `internal/service/backtest/rsi_pullback_registry_test.go`
- `docs/superpowers/plans/task-1-report-mdmg.md` (этот файл)
- `reports/MDMG_base/`, `reports/MDMG_base_wf123/`, `reports/MDMG_base_wf153/`,
  `reports/MDMG_base_c001/` (не коммитятся, `reports/` в `.gitignore`)

## Самопроверка

- Пакет `mdmg` — точная копия шаблона брифа (Step 3), без отклонений.
- Реестр: импорт и запись карты добавлены строго алфавитно (между `magn` и `mvid` в обоих блоках) —
  отклонение от буквального Step 4 брифа («после `ragr`») сделано сознательно, по прямому
  прецеденту `task-1-report-ragr.md`, без отдельного цикла ревью. `gofmt -w` выровнял столбцы,
  повторный `gofmt -l` — пусто.
- Тест реестра размещён строго рядом с `TestRSIPullbackRAGRIsRegisteredAndCalibrated`, как указано
  в брифе (порядок тестов в файле не алфавитный по прецеденту, здесь без отклонений).
- Гейт D скрипта журнала проверен синтетическими сделками (четыре assert) и на эталонном отчёте
  разведки — оба совпали дословно с брифом.
- Прогоны бэктеста (Step 6 и Step 7 — четыре прогона) выполнены строго последовательно, без
  `-refresh`, все в один календарный день (2026-09-26, 23:42–23:44 МСК), с обязательными флагами.
- Сверка сделка в сделку (Step 6): 0 расхождений, 0 выпавших, 0 добавленных — baseline подтверждён
  без оговорок; переснятые числа буквально совпали со спекой.
- `go build ./internal/... ./pkg/... ./cmd/...` — чисто.
- `go test ./internal/service/trading_strategy/rsi_pullback/... ./internal/service/backtest/...` —
  все пакеты `ok`.

## Замечания

Нет блокеров. Единственное отклонение от буквы брифа — место записи в карте и импорте реестра
(строго алфавитное вместо «после ragr») — обосновано прямым прецедентом того же дефекта в брифе
RAGR, найденным и разрешённым контроллером ранее; повторное ревью того же вопроса излишне.
Переснятый baseline полностью совпал со спекой (нулевое расхождение сделка в сделку), поэтому
переснятые числа Task 1 совпадают с числами спеки 2026-09-26 и не требуют отдельного распространения
на другие задачи плана сверх уже записанного здесь.
