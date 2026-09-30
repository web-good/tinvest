---
description: Единая калибровка тикера rsi_pullback — входы pullback и zone, выбор режима UseZoneEntry 0/1/2 или отказ, выходы, walk-forward, пакет тикера
argument-hint: <TICKER> [заметки владельца]
---

# Единая калибровка rsi_pullback: $ARGUMENTS

Ты калибруешь rsi_pullback под тикер из аргумента: оба входа (pullback и zone), выбор режима
`UseZoneEntry` (0 — только pullback, 1 — оба, 2 — только zone) или отказ, общие выходы. Сам строишь
сетки, сам гоняешь бэктесты, сам выбираешь и доводишь тикер до пакета. Владелец хочет вердикт и
готовый код, а не меню: выбирай сам и обосновывай числами. Останавливайся только в точках **СТОП**.

Прочитай до начала:

- `docs/superpowers/specs/2026-09-30-rsi-pullback-unified-calibration-design.md` — спека процедуры;
- `docs/rsi_pullback/strategy.md` — §3, §3.1, §4, §7, §8–§8.2, §9;
- `internal/service/trading_strategy/rsi_pullback/strategy/core/core.go` — `Params`, режимы, `Lookback`;
- `docs/superpowers/specs/2026-09-26-mdmg-rsi-pullback-prep-design.md` — образец канона (темы входа,
  правило большинства, соседи плато, гейты A–D, стоп-условие, второй круг, правило прода);
- пакет тикера `internal/service/trading_strategy/rsi_pullback/strategy/<t>/`, `data/params/rsi_pullback/<t>/`
  и память проекта про тикер, если есть.

## 0. Жёсткие правила каждого прогона

- **`-strategy rsi_pullback -interval Minutes30`** в каждой команде `cmd/backtest` (дефолт CLI — `Hour1`).
- **`-months`, `-train-months`, `-test-months`** в каждой команде walk-forward (дефолт `-months` — 12).
- **`-refresh` — только в прогреве кэша** (§1, шаг 2).
- **Прогоны строго последовательно** — параллельные запуски ломают файл кэша.
- Сравниваемые прогоны — в один день; годы и хвосты режь фиксированными датами; дату прогона пиши в
  `_comment` и в отчёт.
- Прогон, упавший на ошибке API (`rpc error: code = Internal ...`), перезапусти, его числа не пиши.
- Отчёты — в `./reports/<TICKER>/...` (вне git).
- **Режим входа.** У незарегистрированного тикера `UseZoneEntry = 0`: сетка, забывшая режим, молча
  считает pullback, забывшая zone-поле при режиме 1/2 — молча даёт ноль сделок. Правила файлов —
  спека §5, сторож `TestRSIPullbackZoneFilesPinZone`.
- **Если литерал тикера уводит поля от дефолтов ядра**, каждая сетка перечисляет их явно значениями
  дефолтов ядра: калибровка идёт с нуля.
- Документация `docs/rsi_pullback/*.md` — только механика; результаты — в doc-comment пакета тикера и
  в `_comment` сеток.
- Ветка `feat/<ticker>-calibration` от `main` (или от ветки с единой процедурой, пока она не в `main`).
  Коммиты на русском, в конце строка `Co-Authored-By` из системной подсказки.

## 1. Разведка и три baseline

1. Ветка.
2. **Прогрев кэша** — единственный прогон с `-refresh`:
   ```
   go run ./cmd/backtest -ticker <T> -strategy rsi_pullback -interval Minutes30 -months 60 -refresh \
     -out ./reports/<T>/warmup
   ```
3. Длина истории — первый бар `data/candles/<T>_Minutes30.json`; схема и `<M>`:
   - ≥ 36 мес — основная **36/12/6** (4 фолда), контрольная **36/18/6** (3 фолда);
   - 24–36 мес — **24/12/3** (4 фолда), контрольная **24/15/3** (3 фолда);
   - меньше 24 мес — **СТОП**, доложи.
4. Файлы `data/params/rsi_pullback/<t>/plateau_base_{pullback,zone,both}.json`: одна фаза `point`,
   все поля выхода по `core.DefaultParams()`;
   - `pullback` — `UseZoneEntry 0`;
   - `zone` — `UseZoneEntry 2`, `ZoneRSIPeriod 4`, `ZoneRSILower 25`, `ZoneEMAPeriod 200`;
   - `both` — `UseZoneEntry 1` и те же zone-поля.
   Сторож ширины включится, когда на этапе 3 появится `plateau_mode_both.json`. `_comment` каждого
   файла — что это и команды.
5. На каждом baseline: полное окно (`-calibrate <файл> -months <M> -min-trades 1`), walk-forward
   основной, контрольной и основной с `-commission 0.001` (`-min-trades 1`). Сверь «Фолдов: N».
6. Сверка типа входа по колонке «Вход»: у `pullback` — только `pullback`, у `zone` — только `zone`, у
   `both` — оба типа. Иначе — **СТОП**.
7. Вход, чей baseline на полном окне дал меньше 20 сделок, — **неторгуемый**: его темы не гоняются,
   режимы с ним выбывают. Оба неторгуемы — **СТОП**, предложи отказ.
8. Профиль каждого baseline: сделки, PF, net, max DD (₽ и %), win rate, expectancy, доли выходов
   SL/TRAIL/TP/RSI, удержание (медиана / p90 / максимум), ночёвки, входы и выходы в выходные и в часы
   02–06, календарные годы и полугодия, хвосты 6 и 12 месяцев, раздел «По типу входа».
9. **Реальный круг издержек:** 2 × `MinPriceIncrement` / цена. Выше 0.2% — порог пункта 4 (§7)
   ужесточается до реального круга.
10. **Дивидендные отсечки** окна — из T-Invest `GetDividends` (образец `reports/_analysis/divprobe/`).
    Объяви до прогонов, не меняй.
11. **Скрипт журнала** `reports/_analysis/<t>_journal.py` (вне git) по образцу
    `reports/_analysis/mdmg_journal.py`: понимает выходы TP/TRAIL/SL/RSI и колонку `entry_kind`
    (последняя в CSV), считает всё из шага 8 и по типам входа. Сверь его вывод с шапкой отчёта baseline.

## 2. Сетки входов

Файлы в `data/params/rsi_pullback/<t>/`. `_comment` каждого: что меряет тема, полная команда запуска
с путём к самому файлу (требует `TestRSIPullbackCalFilesValid`), после прогона — строка
`РЕЗУЛЬТАТ ПРОГОНА <дата>: …` (pooled OOS, пул, пофолдовые PF, голоса фолдов).

Pullback-темы (`UseZoneEntry` не задан или 0): `cal_screen`, `cal_entry`, `cal_trend`, `cal_trend_low`,
`cal_day`, `cal_volume`, `cal_vol_window`, арбитры `cal_trend_spent`, `cal_trend_day`.
Zone-темы (каждая фаза — `UseZoneEntry 2` и три zone-поля): `cal_zone_zone_entry`, `cal_zone_zone_trend`.

Минимальные оси — таблица спеки §4.6 (в коде — `rsiPullbackUnifiedMinAxes`, сторож
`TestRSIPullbackUnifiedGridsStayWide`). Шире можно, уже нельзя. Инварианты: `StopDailyATR` нигде
не 0 и свипается только `cal_out_risk`; периоды ≥ 2 (кроме `EMAFast` 1 в `trend_low`/`trend_spent`);
узел, чьё окно `Lookback` съедает больше ~10% окна при истории < 36 мес, выбрасывается с записью в
`_comment`; упор в край — зонд за краем, не обрезка оси.

Первый коммит — baseline + сетки входов; `go test ./internal/service/backtest/ -run 'RSIPullback' -count=1`.

## 3. Этапы 1–2: walk-forward тем входов и сборка

Каждая тема:
```
go run ./cmd/backtest -ticker <T> -strategy rsi_pullback -interval Minutes30 \
  -calibrate data/params/rsi_pullback/<t>/<файл>.json -out ./reports/<T>/<тема> \
  -months <M> -train-months <TR> -test-months <TE> -min-trades 20 -metric profit_factor
```
(`cal_screen` — с `-min-trades 1`).

**Вырожденный фолд** — меньше пяти сделок OOS или ни одной убыточной: не голосует.
**Правило большинства:** поле уходит от baseline своего входа, только если за значение ≥ 3 из 4
невырожденных фолдов основной схемы (на 3 фолдах — единогласно); 2/2 — не большинство. Источники
полей — как в каноне; `ZoneRSIPeriod`, `ZoneRSILower` — `zone_entry`; `ZoneEMAPeriod` — `zone_trend`.
Таблица: поле → тема → голоса → вырожденные → принято → baseline.

**Соседи плато** для каждого ушедшего поля — значение и два соседа (`plateau_<поле>_<v>.json` или
`plateau_zone_<поле>_<v>.json`), годы и хвосты; плато уже 0.05 PF — «сигнала нет», прямым текстом.

Сборка: `plateau_entry_pullback.json` (baseline `pullback` + принятые поля входа) и
`plateau_entry_zone.json` (якорь + принятые zone-поля). Коммит.

## 4. Этап 3: выбор режима

`plateau_mode_pullback.json` (= `plateau_entry_pullback`, `UseZoneEntry 0`), `plateau_mode_zone.json`
(= `plateau_entry_zone`, `UseZoneEntry 2`), `plateau_mode_both.json` (поля обоих, `UseZoneEntry 1`),
выходы — дефолты ядра. Режимы с неторгуемым входом не создаются. Walk-forward основной схемы,
`-min-trades 1`, в один день.

- Допущен: pooled OOS PF ≥ 1.0 и ≥ 20 сделок пула.
- `both` дополнительно: в «Пул по типу входа» оба подпула PF ≥ 1.0 (с числом сделок в сводке).
- Выбор: больший PF → при разнице < 0.05 меньшая max DD % на полном окне → одиночный раньше `both`.
- Никто не допущен — дальше с режимом наибольшего PF; это пишется в сводку.

Таблица режимов (PF, пул, подпулы, DD) — в `_comment` `plateau_mode_both.json`. Коммит.

## 5. Этап 4: выходы

`cal_out_exit.json`, `cal_out_risk.json`, `cal_out_trail.json` — вход выбранного режима
зафиксирован, каждая фаза задаёт `UseZoneEntry` режима (и zone-поля при 1/2):

- `out_exit`: `RSIUpper`; в режиме 2 — `RSIPeriod` × `RSIUpper`;
- `out_risk`: `StopDailyATR` × `TPDailyATR`;
- `out_trail`: `UseTrail` 0/1 × `TrailDailyATR`. `UseRSIExit` не свипается.

Walk-forward и большинство — как в §3. **Гейт A** (потолок стопа по выживаемости уровня и таблице
срабатываний поверх зафиксированного входа, к `min(StopDailyATR, TrailDailyATR при UseTrail=1)`),
**гейт B** (max DD точки не выше baseline выбранного режима ни в ₽, ни в %), **гейт C** (профиль:
падение SL с ростом удержания и ночёвок — капкан, бери более узкий стоп в пределах плато; входов в
выходные — ноль), **гейт D** (сделки через отсечки — ноль) — как в каноне. Сборка — `plateau_point.json`
(режим задан явно). Коммит.

## 6. Этап 5: перепроверка режима

`plateau_final_{pullback,zone,both}.json` — вход каждого режима из §3, выходы из §5. Правило §4 заново.
Победил другой режим — переходи на него (выходы общие), выходы не перекалибровывай; переход один.
Точка = `plateau_final_<победитель>`, её копия — `plateau_point.json`.

## 7. Проверка точки — восемь пунктов стоп-условия

`plateau_point.json` (`-min-trades 1`), в один день: основная схема, контрольная, основная с
`-commission 0.001` (или реальным кругом), полное окно.

1. pooled OOS PF < 1.0 на основной схеме;
2. меньше 20 сделок в пуле OOS основной;
3. pooled OOS PF < 1.0 на контрольной;
4. pooled OOS PF < 1.0 при удвоенных (или реальных) издержках;
5. убыточен один из двух последних полных календарных лет или текущий год;
6. хвост 6 или 12 (по дате входа) с PF < 1.0, либо меньше пяти сделок в хвосте 6; знак хвоста 6 сверь
   с объединённым OOS PF двух последних фолдов (net = `OOS NetPnL%` × 1000, `GL = net/(PF−1)`,
   `GP = PF·GL`; формулу сначала проверь на baseline) — знаки разошлись: **СТОП**;
7. провал гейта C или D;
8. **ноль SL-выходов у точки на полном окне.**

Гейты A и B — ограничения сборки; точку, которую внутри них не собрать, считай сработавшей по пункту 1.

## 8. Второй круг — только при провале первого

Темы выбранного режима: `cal2_<тема>.json` (входа) и `cal2_out_<тема>.json` (выходов, режим задан
явно), точка — `r2_point.json`. Зоны — по фактическим голосам первого круга, шаг — половина шага
первого круга, не больше четырёх тем, каждое поле свипает ровно одна тема. Большинство — в зоне ±1 шаг.
`-min-trades 1`. Пункт 2 снят (пул меньше десяти закрывает работу); пункты 1, 3–8 в силе.

## 9. Правило прода

Кандидаты — точка (первого или второго круга) и три baseline; допущен прошедший все восемь пунктов.
Выбор: больший pooled OOS основной → при разнице < 0.05 меньшая max DD % → меньше убыточных полугодий
→ baseline `pullback`. Никто не допущен — **отказ**: литерал не меняется, сетки остаются протоколом,
в памяти — причина, сводка владельцу, без мержа.

## 10. Финал при допуске

1. Литерал пакета `internal/service/trading_strategy/rsi_pullback/strategy/<t>/<t>.go`: `UseZoneEntry`
   вердикта (`0`, `core.ZoneEntryAlso` или `core.ZoneEntryOnly`), zone-поля при 1/2, поля победителя;
   doc-comment — дата, протокол, таблицы голосов и выбора режима (этапы 3 и 5), плато, гейты A–D,
   восемь пунктов с числами, кандидат против baseline, риски. Тест-снимок литерала обновить. Нет
   пакета — завести по образцу соседей и алиас в `internal/service/backtest/rsi_pullback_registry.go`
   и `internal/service/trading_strategy/rsi_pullback/live/registry.go`.
2. При `UseZoneEntry` 1 или 2 — `go run ./cmd/pullparity -tickers <T> -months <M>`: ноль расхождений.
3. `go run ./cmd/backtest -ticker <T> -strategy rsi_pullback -interval Minutes30 -months <M>` без
   `-calibrate` — ровно числа точки (реестр взял литерал).
4. `./bin/mage ci` зелёный.
5. Коммиты по смыслу.
6. **СТОП:** сводка владельцу и вопрос про мерж и `RSI_PULLBACK_TICKERS` (`env/prod.env`,
   `.example`-файлы, сторож env).
7. Память: проектная запись тикера и строка в `MEMORY.md`.

## 11. Сводка владельцу

Коротко, по-русски: вердикт (режим 0/1/2 или отказ) и кандидат; литерал и отличие от baseline;
таблица выбора режима на этапах 3 и 5 с подпулами; таблица тем (pooled/пул/голоса); кандидат против
baseline на всех схемах; восемь пунктов с числами; главные риски строкой; что осталось владельцу.
