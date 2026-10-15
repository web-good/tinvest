# pdl_reclaim — возврат над вчерашним лоем на 5m

Дата: 2026-10-15. Ветка: `feat/pdl-reclaim` (от `feat/drop-bounce-rsi-windows`).

## Цель

Бэктест-стратегия, только лонг, внутридневная. Уровни поддержки и сопротивления — лой (PDL) и хай (PDH)
предыдущего буднего дня. Вход — на 5m свечах, когда цена ложно пробила PDL и вернулась над ним (вход A)
или когда под PDL случилось бычье поглощение (вход B). Выход — стоп в половину дневного ATR, цель PDH,
RSI(6) в верхней зоне, разворот гистограммы MACD или конец дня.

Стратегия устроена как остальные: общее ядро, пакет на каждый откалиброванный тикер с литералом
`Params()`, реестр в бэктесте, вселенная в env. Первые тикеры на калибровку — GAZP, SBER, NVTK
(калибровка — после реализации ядра, отдельным шагом). Live-раннера в этой работе нет.

## Уровни

- **PDL / PDH** — low и high последней завершённой будней дневки (`md.DailyLows` / `md.DailyHighs`,
  выходные дневки отброшены, как в `gap_fade.weekdayDaily`). В понедельник это пятница. Дневка
  биржевая — со всеми сессиями дня.
- **Дневной ATR** — Уайлдер `DailyATRPeriod` (14) по завершённым будним дневкам. Уходит в сигнал
  (`Signal.ATR`) и замораживается на позицию (`Position.EntryATR`).
- Нет дневок для PDL или ATR — входов нет.

## Вход

Проверяется на закрытии каждой 5m свечи `i` без открытой позиции. Общие условия:

- свеча `i` — Пн–Пт МСК, начинается не раньше `EntryFromHour:EntryFromMinute` (07:00);
- свеча покупки (`i+1`) начинается раньше `EntryToHour:EntryToMinute` (18:00) — то есть свеча `i`
  начинается раньше 17:55; покупка по open следующей свечи (`Config.EntryAtNextOpen`), свечи `i+1`
  в данных нет — сигнал отброшен;
- `close[i] < PDH`, когда цель включена (`TakeProfitPDH = 1`): иначе цели выше входа нет;
- стоп `close[i] − StopDailyATR · ATR` выше нуля.
- **RSI-фильтр** (добавлен владельцем 2026-10-15): RSI(`RSIPeriod`) по 5m close на свече `i` не выше
  `RSIEntryMax` (50); RSI выше 50 — входа нет. Недостаточно свечей для RSI — входа нет. `RSIEntryMax = 0`
  или `RSIPeriod = 0` выключает фильтр.

**Вход A — ложный пробой** (`EntryA = 1`): свеча `i` зелёная (`close > open`), `low[i] < PDL`,
`close[i] > PDL`.

**Вход B — поглощение под PDL** (`EntryB = 1`):

- свеча `i−1` красная (`close < open`), `close[i−1] < PDL`;
- свеча `i` зелёная, `close[i] > open[i−1]` (классическое поглощение тела), `low[i] < low[i−1]`;
- обе свечи одной даты МСК (свеча `i−1` может начаться раньше 07:00 — это условие только на `i`).

Сработали обе ветки — причина входа `A`. Сигнал несёт `StopLoss = close − StopDailyATR · ATR` (для
`-risk-pct`), `TakeProfit = PDH` (если `TakeProfitPDH = 1`), `ATR`.

После выхода перевход разрешён в тот же день на тех же правилах.

## Выход

На каждой 5m свече с открытой позицией, по приоритету:

1. **SL** — `low ≤ вход − StopDailyATR · EntryATR` (стоп от цены исполнения, заморожен на позиции).
   Движок исполняет по `min(стоп, open)`.
2. **TP** — `high ≥ PDH` (цель заморожена на позиции). Движок исполняет по `max(цель, open)`. Стоп и
   цель в одной свече — побеждает стоп.
3. **RSI** — `RSI(RSIPeriod)` по 5m close (Уайлдер) `≥ RSIExit` на закрытии свечи. Продажа по close.
4. **MACD** — гистограмма MACD(`MACDFast`, `MACDSlow`, `MACDSignal`) по 5m close хотя бы на одной свече
   от свечи входа (включительно) до предыдущей была зелёной (`≥ 0`), а на текущей — красная (`< 0`).
   Продажа по close. Зелёная уже на входе — считается.
5. **EOD** — первая свеча, начавшаяся не раньше `ExitHour:ExitMinute` (18:30, закрывается в 18:35),
   продаётся по close. Первая свеча новой даты МСК или свеча даты позже входа — тоже продажа по
   close (защита от переноса через ночь, как в `gap_fade.eodReason`); нет времени свечей — продажа.

Ядро без состояния между свечами: «MACD был зелёным в сделке» пересчитывается по гистограмме от
`Position.EntryTime` до текущей свечи.

## Параметры

| поле | по умолчанию | смысл |
|---|---|---|
| `EntryA` | 1 | 1 — вход A включён |
| `EntryB` | 1 | 1 — вход B включён |
| `StopDailyATR` | 0.5 | стоп ниже входа в дневных ATR; 0 — входов нет (нечем мерить риск) |
| `DailyATRPeriod` | 14 | длина дневного ATR Уайлдера по будням |
| `TakeProfitPDH` | 1 | 1 — цель PDH; 0 — без цели |
| `RSIPeriod` | 6 | длина RSI выхода на 5m; 0 — RSI-выход выключен |
| `RSIExit` | 70 | верхняя зона RSI; 0 — RSI-выход выключен |
| `RSIEntryMax` | 50 | RSI сигнальной свечи выше этого — входа нет; 0 — фильтр выключен |
| `MACDFast` / `MACDSlow` / `MACDSignal` | 12 / 26 / 9 | MACD выхода на 5m close; некорректные периоды — MACD-выход выключен |
| `EntryFromHour` / `EntryFromMinute` | 7 / 0 | сигнальная свеча начинается не раньше |
| `EntryToHour` / `EntryToMinute` | 18 / 0 | свеча покупки начинается раньше |
| `ExitHour` / `ExitMinute` | 18 / 30 | первая свеча не раньше — выход EOD по её close |

Все поля `int` / `float64` — их перебирает калибровка через рефлексию. `Lookback` — с запасом на
прогрев MACD и RSI и на свечи от входа до EOD: `max(3 · (MACDSlow + MACDSignal), 4 · RSIPeriod + 2) +
ExitBars`, где `ExitBars` — 5m свечей от 07:00 до 18:35 (139); дневки движок даёт отдельно.

## Устройство

- `internal/service/trading_strategy/pdl_reclaim/strategy/core/`:
  - `core.go` — `Params`, `DefaultParams`, `Strategy`, `NewWithParams`, `Lookback`, `Decide`
    (`enter` / `manage`);
  - `explain.go` — разбор свечи для `cmd/backtest -explain "YYYY-MM-DD HH:MM"`: каждое условие входа
    A/B и выхода со значением и ✓/✗, в конце итог `Decide`.
- `internal/service/trading_strategy/pdl_reclaim/strategy/<ticker>/` — пакет тикера: `Params()` с
  лучшей точкой калибровки, запись калибровки в doc-комментарии. Появляются после калибровки (GAZP,
  SBER, NVTK первыми); в этой работе пакетов нет.
- `internal/service/backtest/pdl_reclaim_registry.go` — `pdlReclaimRegistry` (`тикер → Params`),
  `PdlReclaimLookupOrGeneric(ticker, interval)`: тикер без пакета — `core.DefaultParams()`, JSON
  `-params` / `-calibrate` поверх; любой интервал кроме `Minutes5` — ошибка.
- `cmd/backtest`: `-strategy pdl_reclaim`, `EntryAtNextOpen`, `-explain`, `-risk-pct` (стоп в сигнале).
- `internal/config/pdl_reclaim.go` — `PdlReclaimConfig{TickersMinutes5 []string}` из
  `PDL_RECLAIM_TICKERS_M5`, `Enabled()`. Пустая переменная в `env/prod.env`, `env/prod.env.example`,
  `env/local.env.example`. Тест `TestPdlReclaimLiveTickersAreRegistered`: каждый тикер из
  `env/prod.env` имеет пакет в реестре.
- `data/params/pdl_reclaim/grid_example.json` — стартовая сетка: `StopDailyATR` (0.3/0.5/0.7) ×
  `RSIExit` (65/70/75/80) × `RSIPeriod` (4/6/8); веточные выключатели `EntryA`/`EntryB` проверяются
  отдельными прогонами.
- `docs/pdl_reclaim/strategy.md` — механика (по образцу `docs/drop_bounce/strategy.md`); строка в
  `CLAUDE.md`.

## Тесты

- Ядро, табличные на синтетических 5m свечах с дневками:
  - вход A: каждое условие по отдельности ломает вход; `close = PDL` и `low = PDL` — нет входа;
  - вход B: красная выше PDL, `close[i] = open[i−1]`, `low[i] = low[i−1]`, свечи разных дат — нет входа;
  - окно: 06:55 нет, 07:00 да, свеча 17:55 нет; суббота нет;
  - `close ≥ PDH` при включённой цели — нет входа; при выключенной — есть;
  - нет дневок / ATR — нет входа;
  - выходы: каждый по отдельности; стоп и цель в одной свече — SL; RSI раньше MACD; MACD зелёная на
    входе засчитывается, без зелёной красная не продаёт; EOD 18:25 нет, 18:30 да; первая свеча новой
    даты — EOD.
- Движок: полная сделка A и сделка B через `backtest.Run` с `EntryAtNextOpen`, перевход после выхода
  в тот же день.
- Реестр: `Minutes5` — binding, `Minutes15` — ошибка, частичный JSON поверх дефолтов.
- Конфиг: пустая переменная — `Enabled() == false`.

## Вне работы

Live-раннер, скринер, скилл калибровки, пакеты тикеров (GAZP, SBER, NVTK — следующий шаг после ядра).
