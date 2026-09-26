# Task 1 report: пакет `strategy/ragr` в состоянии «калибровка не проводилась»

## Реализация

- `internal/service/trading_strategy/rsi_pullback/strategy/ragr/ragr.go` — новый пакет:
  константа `Ticker = "RAGR"`, `DefaultParams()` возвращает `core.DefaultParams()` без изменений
  (СОСТОЯНИЕ: калибровка не проводилась). Doc-комментарий пакета — дословно по Step 3 брифа.
- `internal/service/trading_strategy/rsi_pullback/strategy/ragr/ragr_test.go` — два теста:
  `TestParamsTrackTheBaselineUntilCalibrated` (снимок держит baseline ядра), `TestTickerIsRAGR`.
- `internal/service/backtest/rsi_pullback_registry.go` — добавлен импорт `rsipullbackragr` и
  запись в `rsiPullbackRegistry` СТРОГО АЛФАВИТНО, между `nvtk` и `reni` в обоих блоках.
  Отклонение от брифа: Step 4 буквально требовал ставить строку карты «после строки
  `rsipullbacksvcb`» — по прецеденту `docs/superpowers/plans/task-2-report-svcb.md` (Fix round 1)
  это тот же шаблонный дефект брифа, что уже был найден и отвергнут ревью SVCB: контроллер тогда
  постановил, что карта `rsiPullbackRegistry` обязана оставаться строго алфавитной. Применяю то же
  постановление здесь без повторного цикла ревью: строка поставлена на алфавитное место (между
  `nvtk` и `reni`), а не после `svcb`. `gofmt -w` подтвердил выравнивание столбцов (без изменений
  при повторном запуске).
- `internal/service/backtest/rsi_pullback_registry_test.go` — добавлен импорт `rsipullbackragr`
  (алфавитно, между `nvtk` и `reni`) и тест `TestRSIPullbackRAGRTracksBaseline` сразу после
  `TestRSIPullbackSVCBIsRegisteredAndCalibrated`, как указано в брифе. Эта часть брифа (порядок
  тестов В ФАЙЛЕ, не карты) не переносится на алфавит — по тому же прецеденту SVCB порядок тестов
  в файле хронологический/нарративный, а не алфавитный, и ревью SVCB его не оспаривало.
- `reports/_analysis/ragr_journal.py` (вне git) — гейт D переписан под §5.7 спеки: `EX_DATES`
  заменён на пять событийных шоков с временем бара (три внутридневных, два дневных гэпа), добавлена
  функция `held_through`, блок «ГЕЙТ D» переписан под тройку `(ex, bar, gap)` и печатает итоговую
  строку с планкой. Докстринг файла поправлен («событийным шокам» вместо «дивидендным отсечкам»,
  `ragr_journal.py` вместо `svcb_journal.py`).

## TDD

RED (Step 2 брифа):

```
$ go test ./internal/service/trading_strategy/rsi_pullback/strategy/ragr/ -v
internal/service/trading_strategy/rsi_pullback/strategy/ragr/ragr_test.go:14:18: undefined: DefaultParams
internal/service/trading_strategy/rsi_pullback/strategy/ragr/ragr_test.go:20:5: undefined: Ticker
internal/service/trading_strategy/rsi_pullback/strategy/ragr/ragr_test.go:21:38: undefined: Ticker
FAIL	tinvest/internal/service/trading_strategy/rsi_pullback/strategy/ragr [build failed]
FAIL
```

Пакета не существовало — падение ожидаемое (build failed, не логическое FAIL).

GREEN (после Step 3 — написан `ragr.go`):

```
$ go test ./internal/service/trading_strategy/rsi_pullback/strategy/ragr/ -v
=== RUN   TestParamsTrackTheBaselineUntilCalibrated
--- PASS: TestParamsTrackTheBaselineUntilCalibrated (0.00s)
=== RUN   TestTickerIsRAGR
--- PASS: TestTickerIsRAGR (0.00s)
PASS
ok  	tinvest/internal/service/trading_strategy/rsi_pullback/strategy/ragr	0.005s
```

Step 4 — полный прогон пакетных и реестровых тестов после регистрации в `rsi_pullback_registry.go`
(с исправленной алфавитной картой):

```
$ gofmt -l internal/
$ go test ./internal/service/trading_strategy/rsi_pullback/... ./internal/service/backtest/ -run 'RAGR|RSIPullback' -v
...
=== RUN   TestRSIPullbackRAGRTracksBaseline
--- PASS: TestRSIPullbackRAGRTracksBaseline (0.00s)
...
PASS
ok  	tinvest/internal/service/backtest	0.258s
```

`gofmt -l` не напечатал ничего — форматирование чистое. Дополнительно после переноса карты на
алфавитное место повторно прогнаны `go build ./internal/... ./pkg/... ./cmd/...` (чисто) и
`go test ./internal/service/trading_strategy/rsi_pullback/... ./internal/service/backtest/...`
(все пакеты `ok`, включая `ragr`).

## Гейт D скрипта журнала (Step 5)

Синтетическая проверка правила `held_through` на шести случаях внутридневного шока плюс двух
случаях дневного гэпа:

```
$ python3 - <<'EOF'
... (шесть кейсов вокруг бара 2025-03-26 15:30, два кейса вокруг гэпа 2025-04-07)
EOF
OK
```

Все `assert` прошли, включая `len(j.EX_DATES) == 5`.

## Прогон на эталонном baseline разведки (Step 6)

```
$ python3 reports/_analysis/ragr_journal.py reports/RAGR_prep/base19/RAGR_rsi_pullback_Minutes30_20260926_091823.md
  ГЕЙТ D: сделки, удерживаемые через событийный шок:
    2025-03-26 15:30 (шок -15.7%): сделок 0, PnL +0.00 ₽
    2025-04-07 гэп (шок -8.1%): сделок 0, PnL +0.00 ₽
    2025-05-14 10:00 (шок -13.8%): сделок 1, PnL -2822.39 ₽
      #3   2025-05-13 23:00:00 138.86 -> 2025-05-14 10:00:00 135.20 (SL) -2822.39 ₽
    2026-06-22 17:00 (шок -9.7%): сделок 0, PnL +0.00 ₽
    2026-08-06 гэп (шок -15.5%): сделок 0, PnL +0.00 ₽
    итого через шоки: сделок 1, PnL -2822.39 ₽ (планка: ≤ 1 сделки и ≥ −2822.39 ₽)
    без них: сделок 76, net +11345.57 ₽, PF 1.333
```

Ровно одна сделка через шок 2025-05-14 10:00 (вход 2025-05-13 23:00, SL), −2822.39 ₽; остальные
четыре шока — 0. Совпадает с ожиданием брифа буквально.

## Пересъёмка baseline (Step 7)

Прогон выполнен 2026-09-26 в 14:38 МСК (позже порога 09:30 из решающего правила брифа):

```
$ go run ./cmd/backtest -ticker RAGR -strategy rsi_pullback -interval Minutes30 \
  -months 19 -metric profit_factor -out ./reports/RAGR_base
report: reports/RAGR_base/RAGR_rsi_pullback_Minutes30_20260926_143814.md (trades=77, net=8523.21, PF=1.231)
```

Журнал переснятого прогона (сводка):

```
Период                     2025-02-26 14:38 — 2026-09-26 14:38
Всего сделок               77
Profit factor              1.231
Чистый PnL                 8523.21 (8.52%)
Макс. просадка             9642.93 (9.03%)
Win rate                   63.64%
Expectancy                 110.69
```

Числа буквально совпадают с дефолтами каталога из спеки (Сделок 77, PF 1.231, Net +8523.21 ₽,
Max DD 9642.93 ₽ / 9.03%). Гейт D переснятого прогона идентичен эталону (1 сделка через
2025-05-14 10:00, −2822.39 ₽).

Сверка сделка в сделку на пересечении окон:

```
новое окно: 2025-02-26 14:38 — 2026-09-26 14:38
эталон на пересечении 77, новый 77
расхождений на пересечении: 0 (длина равна)
выпали в начале: 0, net +0.00
добавились в конце: 0, net +0.00
```

**Вердикт по решающему правилу.** Запуск состоялся позже 09:30, поэтому формально применяется
вторая ветка правила («расхождения допустимы только в первых двух неделях нового окна, дальше —
остановиться и доложить владельцу»). Фактических расхождений нет ни одного: длины совпали (77=77),
не выпало и не добавилось ни одной сделки, гейт D идентичен. Ноль расхождений тривиально
удовлетворяет обеим ветвям правила — эскалация к владельцу не требуется.
**Baseline совпал со спекой.** Переснятые числа полного окна не отличаются от чисел в спеке;
пересчёт гейта B по переснятому max DD не требуется — значение то же (9642.93 ₽ / 9.03%).
Вердикт по пункту 5 стоп-условия (2025 год в минусе, PF 0.893) не изменился.

## Файлы

- `internal/service/trading_strategy/rsi_pullback/strategy/ragr/ragr.go`
- `internal/service/trading_strategy/rsi_pullback/strategy/ragr/ragr_test.go`
- `internal/service/backtest/rsi_pullback_registry.go`
- `internal/service/backtest/rsi_pullback_registry_test.go`
- `docs/superpowers/plans/task-1-report-ragr.md` (этот файл)
- `reports/_analysis/ragr_journal.py` (не коммитится, `reports/` в `.gitignore`)
- `reports/RAGR_base/RAGR_rsi_pullback_Minutes30_20260926_143814.md` (не коммитится)

## Самопроверка

- Пакет `ragr` — точная копия шаблона брифа (Step 3), без отклонений.
- Реестр: импорт и запись карты добавлены строго алфавитно (между `nvtk` и `reni` в обоих блоках) —
  отклонение от буквального Step 4 брифа («после `svcb`») сделано сознательно, по прямому
  прецеденту Fix round 1 из `task-2-report-svcb.md`, без отдельного цикла ревью.
  `gofmt -w` выровнял столбцы, повторный `gofmt -l` — пусто.
- Тест реестра размещён строго рядом с `TestRSIPullbackSVCBIsRegisteredAndCalibrated`, как указано
  в брифе (порядок тестов в файле не алфавитный по прецеденту, здесь без отклонений).
- Гейт D скрипта журнала проверен синтетическими сделками (шесть внутридневных кейсов + два
  гэп-кейса) и на эталонном отчёте разведки — оба совпали дословно с брифом.
- Прогон бэктеста выполнен один раз, последовательно, без `-refresh`, с обязательными
  `-interval Minutes30 -months 19 -metric profit_factor`.
- Сверка сделка в сделку: 0 расхождений, 0 выпавших, 0 добавленных — baseline подтверждён без
  оговорок.
- `go build ./internal/... ./pkg/... ./cmd/...` — чисто.
- `go test ./internal/service/trading_strategy/rsi_pullback/... ./internal/service/backtest/...` —
  все пакеты `ok`.

## Замечания

Нет блокеров. Единственное отклонение от буквы брифа — место записи в карте реестра (алфавитное
вместо «после svcb») — обосновано прямым прецедентом того же дефекта в брифе SVCB (Fix round 1),
найденным и разрешённым контроллером ранее; повторное ревью того же вопроса излишне.
