# drop_bounce — отскок после большой красной 15m-свечи в перепроданности

Бэктест-стратегия, только лонг, таймфрейм 15m (`-interval Minutes15`; другие cmd/backtest отклоняет). Позиция может переноситься через ночь. Код: `internal/service/trading_strategy/drop_bounce/strategy/core`, привязка к бэктесту — `internal/service/backtest/drop_bounce_registry.go`. Спека: `docs/superpowers/specs/2026-10-09-drop-bounce-rework-design.md`.

## Вход

1. Последняя закрытая свеча красная и её тело не меньше `MinDropPct` процентов: `(open − close) / open · 100 ≥ MinDropPct`.
2. RSI(`RSIPeriod`) по Уайлдеру на close этой свечи ниже `RSIEntry`. Пока истории меньше `RSIPeriod + 1` свечей, сигнала нет.
3. Окно: свеча-сигнал — Пн–Пт МСК, начинается не раньше `StartHour:00`, а свеча исполнения (через 15 минут) лежит в ту же дату. Сигнал на последней свече дня на утро не переносится.
4. Фильтр тренда (опционально): при `EMAPeriod > 0` close свечи-сигнала должен быть строго выше EMA(`EMAPeriod`). Пока истории меньше `EMAPeriod` свечей, сигнала нет. `EMAPeriod = 0` выключает фильтр.
5. Покупка по open следующей свечи (`Config.EntryAtNextOpen`). Сигнал уровней не несёт: стоп и профит считаются от цены исполнения.
6. Исполнение только в ближайший 15m-слот: сигнал несёт `FillBy = время свечи-сигнала + 15 минут`, и если этой свечи в данных нет (дыра в вечерней сессии, тонкий тикер), движок сигнал отбрасывает, а не покупает по open следующего утра.

Пока позиция открыта, новые сигналы игнорируются.

## Выход

Проверяется на каждой свече удержания, включая свечу входа, по порядку; срабатывает первое условие.

1. **SL** — low свечи ≤ `вход · (1 − StopPct/100)`. Движок исполняет по `min(стоп, open)`: ночной гэп ниже стопа продаётся по open.
2. **PROFIT** — только на свече входа: её close ≥ `вход · (1 + ProfitPct/100)`. Продажа по close. На следующих свечах правило не действует.
3. **MACD** — гистограмма `MACD − signal` (`MACDFast/MACDSlow/MACDSignal`). Зелёная — больше нуля, красная — меньше. Цвет до входа не важен. Если за время сделки (со свечи входа включительно) гистограмма хоть раз закрылась зелёной, выход по close первой следующей красной свечи. Пока зелёной в сделке не было, красные свечи выхода не дают.
4. **RSI** — RSI(`RSIPeriod`) пересёк `RSIExit` снизу вверх: на прошлой свече ниже, на текущей не ниже. Продажа по close.
5. **STOCH** — сглаженный %K стохастика (`StochPeriod`, сглаживание SMA(`StochSmooth`), 1 — сырой %K) пересёк `StochExit` снизу вверх. Продажа по close.

Выхода по концу дня нет. Нулевой порог выключает свой выход (`StopPct`, `ProfitPct`, `RSIExit`, `StochExit`; у MACD — `MACDExit = 0`). Неверные периоды MACD или стохастика выключают этот выход.

## Параметры (`core.Params`)

| Поле | По умолчанию | Смысл |
|---|---|---|
| `MinDropPct` | 0.6 | минимальное падение тела свечи-сигнала, % |
| `RSIPeriod` | 6 | длина RSI для входа и выхода |
| `RSIEntry` | 30 | вход, если RSI ниже |
| `RSIExit` | 70 | выход при пересечении снизу вверх (0 — выключен) |
| `MACDExit` | 1 | 1 — выход MACD включён, 0 — выключен |
| `MACDFast`, `MACDSlow`, `MACDSignal` | 9, 14, 3 | периоды MACD |
| `StochPeriod`, `StochSmooth` | 14, 3 | период %K и его сглаживание |
| `StochExit` | 80 | выход при пересечении %K снизу вверх (0 — выключен) |
| `ProfitPct` | 1 | профит на close свечи входа, % (0 — выключен) |
| `StopPct` | 1 | стоп ниже цены входа, % (0 — выключен) |
| `StartHour` | 8 | начало окна сигналов, час МСК |
| `EMAPeriod` | 0 | вход только при close выше EMA этого периода (0 — фильтр выключен) |

История (`Lookback`) — `max(64, 4·RSIPeriod+2, 3·MACDSlow+MACDSignal+2, StochPeriod+StochSmooth+2, 2·EMAPeriod+20)` свечей: она задаёт первую торгуемую свечу и то, насколько далеко назад достаёт поиск зелёной гистограммы MACD в сделке.

Литерал тикера — пакет `internal/service/trading_strategy/drop_bounce/strategy/<ticker>/` с `DefaultParams()`, регистрация — `dropBounceRegistry` (тикер → литерал); без литерала берётся `core.DefaultParams`. JSON из `-params` / `-calibrate` частично переопределяет поля. Стартовая сетка — `data/params/drop_bounce/grid_example.json`, сетки тикера — `data/params/drop_bounce/<ticker>/`.

## Запуск

```
go run ./cmd/backtest -ticker GAZP -strategy drop_bounce -interval Minutes15 -months 12
go run ./cmd/backtest -ticker GAZP -strategy drop_bounce -interval Minutes15 -months 24 -calibrate data/params/drop_bounce/grid_example.json
```
