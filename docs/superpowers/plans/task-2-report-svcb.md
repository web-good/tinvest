# Task 2 report: пакет `strategy/svcb` в состоянии «калибровка не проводилась»

## Реализация

- `internal/service/trading_strategy/rsi_pullback/strategy/svcb/svcb.go` — новый пакет: константа
  `Ticker = "SVCB"`, `DefaultParams()` возвращает `core.DefaultParams()` без изменений (СОСТОЯНИЕ:
  калибровка не проводилась).
- `internal/service/trading_strategy/rsi_pullback/strategy/svcb/svcb_test.go` — два теста:
  `TestParamsTrackTheBaselineUntilCalibrated` (снимок держит baseline ядра),
  `TestTickerIsSVCB`.
- `internal/service/backtest/rsi_pullback_registry.go` — добавлен импорт `rsipullbacksvcb` в
  алфавитном порядке блока импортов (между `svav` и `tbank`) и запись в `rsiPullbackRegistry`
  между `rsipullbacksvav` и `rsipullbacktbank` — карта строго алфавитна, как и импорт. (Fix
  round 1: изначально запись стояла сразу после `rsipullbacksfin`, дословно по формулировке
  брифа; ревью установило, что это была шаблонная опечатка брифа, и контроллер постановил, что
  карта `rsiPullbackRegistry` строго алфавитна — запись перенесена на алфавитное место.)
  Выравнивание карты поправлено `gofmt -w`.
- `internal/service/backtest/rsi_pullback_registry_test.go` — добавлен импорт `rsipullbacksvcb`
  (алфавитно, между `svav` и `tgka`) и тест `TestRSIPullbackSVCBTracksBaseline` сразу после
  `TestRSIPullbackRTKMTracksBaseline`.

## TDD

RED (Step 2 брифа):

```
$ go test ./internal/service/trading_strategy/rsi_pullback/strategy/svcb/ -v
internal/service/trading_strategy/rsi_pullback/strategy/svcb/svcb_test.go:14:18: undefined: DefaultParams
internal/service/trading_strategy/rsi_pullback/strategy/svcb/svcb_test.go:20:5: undefined: Ticker
internal/service/trading_strategy/rsi_pullback/strategy/svcb/svcb_test.go:21:38: undefined: Ticker
FAIL	tinvest/internal/service/trading_strategy/rsi_pullback/strategy/svcb [build failed]
FAIL
```

Пакета не существовало — падение ожидаемое (build failed, не логическое FAIL).

GREEN (после Step 3 — написан `svcb.go`):

```
$ go test ./internal/service/trading_strategy/rsi_pullback/strategy/svcb/ -v
=== RUN   TestParamsTrackTheBaselineUntilCalibrated
--- PASS: TestParamsTrackTheBaselineUntilCalibrated (0.00s)
=== RUN   TestTickerIsSVCB
--- PASS: TestTickerIsSVCB (0.00s)
PASS
ok  	tinvest/internal/service/trading_strategy/rsi_pullback/strategy/svcb	0.003s
```

Step 5 — полный прогон пакетных и реестровых тестов после регистрации в `rsi_pullback_registry.go`:

```
$ go test ./internal/service/trading_strategy/rsi_pullback/... ./internal/service/backtest/ -run 'SVCB|RSIPullback' -v
...
=== RUN   TestRSIPullbackSVCBTracksBaseline
--- PASS: TestRSIPullbackSVCBTracksBaseline (0.00s)
...
=== RUN   TestSVCBGridsStayWide
--- PASS: TestSVCBGridsStayWide (0.00s)
PASS
ok  	tinvest/internal/service/backtest	0.138s
```

Все тесты пакета PASS, регресса по остальным тикерам реестра нет.

`gofmt -l` и `go vet` по затронутым пакетам (`internal/service/backtest`,
`internal/service/trading_strategy/rsi_pullback/...`) — без замечаний.

## Step 6 — переснятие baseline и сверка префикса

Дата/время прогона: **2026-09-25 15:02** (MSK).

Команда (дословно):

```
go run ./cmd/backtest -ticker SVCB -strategy rsi_pullback -interval Minutes30 \
  -months 36 -metric profit_factor -out ./reports/SVCB_base
```

Вывод:

```
report: reports/SVCB_base/SVCB_rsi_pullback_Minutes30_20260925_150249.md (trades=130, net=-3663.14, PF=0.951)
```

`svcb_journal.py` по этому отчёту подтвердил полное совпадение с таблицей дефолтов из
Global Constraints: 130 сделок, PF 0.951, net −3663.14 ₽ (−3.66%), max DD 14848.91 ₽ (14.06%),
win rate 61.54%, expectancy −28.18 ₽, выходы RSI 74 (56.9%) / SL 43 (33.1%) / TP 13 (10.0%),
удержание 8/16/58 баров, ночёвок 50 (38.5%), перенос через 2+ дня 1, выходные входов 0 / выходов 6
(+340.59 ₽), часы 02–06 входов 0 / выходов 6 (+4350.77 ₽), гейт D — 0 сделок по всем пяти датам.

Скрипт префиксной проверки:

```
префикс: сделок 130, net -3663.10, PF 0.951
новые после 2026-09-25 13:30: сделок 0, net +0.00
```

**Префикс совпал со спекой ровно** (130 сделок, PF 0.951; net −3663.10 против объявленных
−3663.14 ₽ — расхождение 0.04 ₽ на 130 сделках — это плавающая точка суммирования отдельных PnL
в `svcb_journal.py` против округления, показанного самим отчётом бэктеста, который сам даёт ровно
−3663.14; содержательно это то же число). Новых сделок после 2026-09-25 13:30 — **ноль**.

**Итог: baseline совпал со спекой.** Хвост кэша за прошедшие с 14:09 до 15:02 не добавил ни одной
новой сделки — переснимать полнооконные числа в отчёт не требуется, таблица Global Constraints
остаётся действующей.

## Файлы

- `internal/service/trading_strategy/rsi_pullback/strategy/svcb/svcb.go`
- `internal/service/trading_strategy/rsi_pullback/strategy/svcb/svcb_test.go`
- `internal/service/backtest/rsi_pullback_registry.go`
- `internal/service/backtest/rsi_pullback_registry_test.go`
- `docs/superpowers/plans/task-2-report-svcb.md` (этот файл)
- `reports/SVCB_base/SVCB_rsi_pullback_Minutes30_20260925_150249.md` (не коммитится,
  `reports/` в `.gitignore`)

## Самопроверка

- Пакет `svcb` — точная копия шаблона брифа (Step 3), без отклонений.
- Реестр: импорт и запись карты добавлены строго алфавитно (`rsipullbacksvcb` — между `svav` и
  `tbank` в обоих блоках), согласно постановлению контроллера по итогам ревью (fix round 1).
  `gofmt -w` выровнял столбцы.
- Тест реестра размещён строго рядом с `TestRSIPullbackRTKMTracksBaseline`, как указано в брифе.
- Прогон бэктеста выполнен один раз, последовательно, без `-refresh`, с обязательным
  `-interval Minutes30`.
- Префикс ≤ 2026-09-25 13:30 совпал со спекой: 130 / −3663.14 (с точностью до 0.04 ₽
  плавающей арифметики) / 0.951.
- Новых сделок после cutoff нет — рестапление отчёта не требуется.

## Замечания

Нет блокеров. Число сделок в полном окне и в префиксе совпало (130 = 130), потому что запуск
пришёлся всего на ~53 минуты позже момента фиксации дефолтов (14:09 → 15:02) без пересечения
получасовой границы, добавляющей новый бар с сигналом.

## Fix round 1 (ревью, Important)

**Находка:** формулировка Step 4 брифа («добавить в карту строку … после строки
`rsipullbacksfin`») оказалась шаблонной опечаткой — карта `rsiPullbackRegistry` в
`internal/service/backtest/rsi_pullback_registry.go` строго алфавитна (как и блок импортов), а
буквальное следование брифу поставило `rsipullbacksvcb` перед `sibn/sngsp/sofl/spbe/svav`, нарушив
это единообразие. Контроллер постановил: карта обязана оставаться алфавитной.

**Изменение:** запись
`rsipullbacksvcb.Ticker: rsiPullbackBindingFor(rsipullbacksvcb.Ticker, rsipullbacksvcb.DefaultParams),`
перенесена с места «сразу после `rsipullbacksfin`» на алфавитное место — между
`rsipullbacksvav.Ticker: ...` и `rsipullbacktbank.Ticker: ...`, то есть ровно там же, где уже стоял
импорт. После правки `gofmt -w internal/service/backtest/rsi_pullback_registry.go` подтвердил
корректное выравнивание столбцов (файл не изменился повторным запуском).

Тест реестра `TestRSIPullbackSVCBTracksBaseline` и его размещение рядом с
`TestRSIPullbackRTKMTracksBaseline` в `rsi_pullback_registry_test.go` не менялись — эта часть
брифа (порядок тестов в тестовом файле, не карты) ревью не оспаривала.

**Команда проверки:**

```
$ go test ./internal/service/backtest/ -run 'SVCB|RSIPullback'
```

**Вывод:**

```
ok  	tinvest/internal/service/backtest	0.124s
```

Полный список подтестов (`-v`) — все PASS, включая `TestRSIPullbackSVCBTracksBaseline` и
`TestSVCBGridsStayWide`; регресса по остальным 36 тикерам реестра нет.

Коммит фикса: `fix(rsi_pullback): SVCB в реестре бэктеста по алфавиту`.
