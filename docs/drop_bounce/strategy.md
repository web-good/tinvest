# drop_bounce — RSI-зона над EMA в окнах покупки

Бэктест-стратегия, только лонг, таймфрейм **только 1h** (`-interval Hour1`, другие cmd/backtest отклоняет). Позиция может переноситься через ночь и выходные. Код: `internal/service/trading_strategy/drop_bounce/strategy/core`, привязка к бэктесту — `internal/service/backtest/drop_bounce_registry.go`. Исходная спека 1h-версии: `docs/superpowers/specs/2026-10-13-drop-bounce-h1-rsi-ema-design.md`. 2026-10-14 владелец переписал вход и выход: окна покупки вместо `StartHour` и пятничного правила, отдельный RSI выхода; гейт хода дня (`DaySpentATR`), выход MACD и выход FRI убраны. Прежние версии — в git.

## Вход

На закрытии часовой свечи, все условия вместе:

1. **Окно покупки:** покупка идёт по open следующей свечи, и окно проверяется по ней. Свеча исполнения — Пн–Пт МСК, в ту же дату, что сигнал, и начинается в часе `h` утреннего `[MorningFrom, MorningTo)` или вечернего `[EveningFrom, EveningTo)` окна. По умолчанию покупки по open 07:00–10:00 и 17:00–23:00 (сигнальные свечи 06:00–09:00 и 16:00–22:00). Пятница — обычный будний день: вечерняя покупка в пятницу переносит позицию через выходные. Окно с `From ≥ To` пустое.
2. **Тренд:** `close > EMA(EMAPeriod)`, строго. Пока EMA не прогрелась, входа нет.
3. **Зона RSI** (`RSI(RSIPeriod)`, Уайлдер), одна из веток:
   - **ZONE** — `RSI < RSIZone`;
   - **CROSS** — RSI пересёк `RSIZone` снизу вверх на этой свече (`prev < RSIZone ≤ cur`) и `cur ≤ RSICap`.
4. **Дневной ATR посчитан** и стоп от close сигнала выше нуля.

Дневной ATR — Уайлдер `DailyATRPeriod` по завершённым дневкам Пн–Пт (выходные сессии отброшены: они в разы уже и занижают ATR). Он уходит в сигнал (`Signal.ATR`), движок замораживает его на позицию (`Position.EntryATR`).

`FillBy = время сигнала + 1h`: нет этой свечи в данных — сигнал отброшен. Пока позиция открыта, сигналы игнорируются. Паузы после стопа нет: следующая свеча может войти снова, если условия держатся.

## Выход

Выходы работают в любое время — окна ограничивают только покупку. По приоритету на каждой свече:

1. **SL** — внутри свечи: `low ≤ вход − StopDailyATR · ATR`, ATR замороженный на сигнале (если позиция его не несёт — сегодняшний). Движок исполняет по `min(стоп, open)`, так что ночной гэп ниже стопа закрывается по open. `StopDailyATR = 0` выключает стоп.
2. **RSI** — на close: RSI(`RSIExitPeriod`) пересёк `RSIExit` снизу вверх (`prev < RSIExit ≤ cur`). Продаёт при любой цене, причина `RSI`. Крест на свече исполнения считается. Период выхода отдельный от периода входа. `RSIExit = 0` или `RSIExitPeriod = 0` выключает выход.
3. **STALE** — на close, если `StaleBars > 0`: позиция держится `StaleBars` свечей (свеча исполнения считается) и ни один high с исполнения не дотянулся до `вход + StaleATR · ATR`. Продаёт, причина `STALE`. По умолчанию выключен (`StaleBars = 0`).

Выхода в конце дня нет.

## Параметры

| поле | по умолчанию | смысл |
|---|---|---|
| `RSIPeriod` | 6 | длина RSI входа; 0 — входов нет |
| `RSIZone` | 30 | уровень перепроданности: ZONE ниже, CROSS через него вверх |
| `RSICap` | 50 | CROSS засчитывается, только если RSI на свече креста ≤ этого |
| `EMAPeriod` | 200 | EMA тренда на часовых close; 0 — входов нет |
| `StopDailyATR` | 1.0 | стоп ниже цены входа в дневных ATR; 0 — выключен |
| `DailyATRPeriod` | 14 | длина дневного ATR Уайлдера по будним дням |
| `RSIExitPeriod` | 10 | длина RSI выхода; 0 — RSI-выход выключен |
| `RSIExit` | 70 | уровень креста RSI выхода вверх; 0 — RSI-выход выключен |
| `MorningFrom` / `MorningTo` | 7 / 11 | утреннее окно покупки, час open свечи исполнения МСК, `[From, To)` |
| `EveningFrom` / `EveningTo` | 17 / 24 | вечернее окно покупки, `[From, To)` |
| `StaleBars` | 0 | выход по времени STALE: свечей без хода в пользу; 0 — выключен |
| `StaleATR` | 0.3 | ход в дневных ATR над входом, который снимает выход STALE |

`Lookback = max(64, 3 · EMAPeriod, 4 · RSIPeriod + 2, 4 · RSIExitPeriod + 2)` — при EMA 200 это 600 часовых свечей.

## Калибровка

Один перебор по схеме hold-out (первые 30 из 36 месяцев, точка — центр плато, стресс, hold-out один раз): `RSIPeriod` × `RSIExitPeriod` × `StopDailyATR`, остальное — дефолты ядра. Списки значений и процедура — скилл `drop-bounce-calibrate`; стартовая сетка — `data/params/drop_bounce/grid_example.json`.

## Пакеты тикеров

Реестр `dropBounceRegistry` держит по одному литералу `Params()` на тикер, заведённый калибровкой на этих правилах; тикер без пакета работает на `core.DefaultParams()`; JSON `-params` / `-calibrate` ложится поверх. Поля прежних правил (`VolDailyMult`, `VolAvgPeriod`; до 2026-10-14 — `DaySpentATR`, `StartHour`, `MACDFast/Slow/Signal`) в файле `-params` молча игнорируются (прогон идёт на дефолтах), а сетка `-calibrate` с таким полем падает с ошибкой `unknown grid field`.

Кандидатов на калибровку подбирает скринер `cmd/dropscreen` — `docs/drop_bounce/screener.md`.

## Вселенная для live

Живого раннера пока нет; вселенная под него — `DROP_BOUNCE_TICKERS_H1` (`config.DropBounceConfig`). Список ведёт скилл `drop-bounce-calibrate` (`scripts/live_env.py`). Тест `TestDropBounceLiveTickersAreRegistered` требует, чтобы каждый тикер из `env/prod.env` имел пакет в `dropBounceRegistry`.

## Запуск

```bash
go run ./cmd/backtest -strategy drop_bounce -interval Hour1 -ticker GAZP -months 36
go run ./cmd/backtest -strategy drop_bounce -interval Hour1 -ticker GAZP -months 36 \
  -calibrate data/params/drop_bounce/grid_example.json -metric profit_factor
# почему на свече был (или не был) вход/выход: каждое условие со значением и ✓/✗, в конце — итог Decide
go run ./cmd/backtest -strategy drop_bounce -interval Hour1 -ticker GAZP -months 36 -explain "2026-09-03 09:00"
```
`-explain` берёт время начала свечи по МСК и те же параметры, что обычный прогон (литерал тикера или `-params`).
Для калибровочных прогонов — собранный бинарь на одном ядре (см. скилл `drop-bounce-calibrate`).
