# pdl_rsi_m30 — независимая 30-минутная версия RSI + stochastic

`pdl_rsi_m30` — экспериментальная backtest-only long-стратегия на закрытых 30-минутных свечах.
Запуск разрешён только с `-interval Minutes30`. Её движок, параметры, registry, литералы и сетки
находятся в каталоге `pdl_rsi_m30` и не импортируют действующую `pdl_rsi` Hour1.

Код ядра: `internal/service/trading_strategy/pdl_rsi_m30/strategy/core`. Привязка к бэктесту:
`internal/service/backtest/pdl_rsi_m30_registry.go`. У стратегии нет live-раннера.

## Вход

RSI — Wilder RSI(`RSIPeriod`) по M30-close. Стохастик — медленный %K: сырой %K за `StochK`
баров, сглаженный средним за `StochSmooth`; %D — среднее %K за `StochD`, используется только в
тексте причины входа. Порог %K равен 20 и зашит в ядро.

Режимы сигнала:

- `StochK > 0`: на одном закрытом M30-баре RSI ниже `RSIArm` и %K ниже 20. Каждый бар, на котором
  оба условия выполняются, является новым сигналом.
- `StochK == 0`: RSI пересёк `RSIArm` сверху вниз; предыдущий и текущий M30-бары принадлежат одной
  дате MSK.

На сигнальном баре также проверяются:

- RSI не ниже `RSIFloor`, если фильтр включён;
- close строго выше прогретой EMA(`TrendEMA`) по M30-close;
- объём M30-бара выше `VolumeMult` × среднего объёма `VolBars` предыдущих M30-баров основной
  сессии 10:00–18:30 MSK, если фильтр включён;
- запас дневного хода не ниже `MinRoomATR` дневного ATR, если фильтр включён;
- следующий M30-бар начинается в ту же буднюю биржевую дату MSK.

Будний день, чья первая видимая свеча начинается раньше 06:00 MSK, считается дилерской датой:
входы запрещены, а её бары не попадают в объёмную базу.

Покупка исполняется по open следующего M30-бара. Дневной Wilder ATR(`DailyATRPeriod`) считается
по завершённым будним дневкам и замораживается на сигнале. Сигнал несёт стоп и цель от close для
`-risk-pct`; открытая позиция перестраивает их от фактического fill:

- стоп: `StopATR` дневных ATR под fill, но не дальше `MaxStopPct` процентов цены;
- цель: `TargetR` расстояний до стопа над fill.

## Выход

На каждом закрытом M30-баре действует приоритет:

1. `SL` — low достиг стопа;
2. `TP` — high достиг цели;
3. `RSI` — RSI(`RSIPeriod`) пересёк `RSIExit` снизу вверх, продажа по close.

Пересечение `RSIExit` сверху вниз ничего не делает. RSI-лесенки и полей `RSITrail*` нет. Выхода
в конце дня нет; позиция переносится через ночь и выходные.

## Параметры

| поле | default | смысл |
|---|---:|---|
| `RSIPeriod` | 6 | период M30 RSI; 0 — входов нет |
| `RSIArm` | 30 | нижняя зона входного сигнала |
| `RSIFloor` | 22 | не покупать более глубокое падение; 0 — выкл |
| `RSIExit` | 70 | cross снизу вверх закрывает позицию; 0 — выкл |
| `StochK` | 14 | период сырого %K; 0 — режим без stochastic |
| `StochSmooth` | 1 | сглаживание %K |
| `StochD` | 3 | период %D только для причины входа |
| `VolumeMult` | 0 | множитель объёма сигнала; 0 — выкл |
| `VolBars` | 20 | M30-баров основной сессии в объёмной базе |
| `MinRoomATR` | 0 | минимальный остаток дневного хода в ATR; 0 — выкл |
| `StopATR` | 0.5 | расстояние стопа в дневных ATR; 0 — входов нет |
| `MaxStopPct` | 0 | максимальное расстояние стопа в процентах; 0 — выкл |
| `TargetR` | 2 | цель в расстояниях до стопа; 0 — без цели |
| `TrendEMA` | 200 | EMA по M30-close; 0 — выкл |
| `DailyATRPeriod` | 14 | период дневного Wilder ATR |

## RENI: пилотный baseline

На 2026-11-02 `RENI` — единственный тикер в пересечении live-вселенных `pdl_rsi`,
`rsi_pullback` и `rsi_zone`, поэтому пилот состоит из него. Его пакет переносит числа Hour1-литерала
только как стартовую точку (`data/params/pdl_rsi_m30/reni/baseline.json`). Это не результат
M30-калибровки.

Baseline:

```bash
go run ./cmd/backtest -ticker RENI -strategy pdl_rsi_m30 -interval Minutes30 \
  -months 36 -risk-pct 1 -params data/params/pdl_rsi_m30/reni/baseline.json \
  -out reports/RENI_pdl_rsi_m30/baseline
```

Rolling walk-forward со stochastic:

```bash
go run ./cmd/backtest -ticker RENI -strategy pdl_rsi_m30 -interval Minutes30 \
  -months 36 -risk-pct 1 -metric profit_factor -min-trades 3 -combos 0 \
  -train-months 3 -test-months 1 \
  -calibrate data/params/pdl_rsi_m30/reni/wf_stoch.json \
  -out reports/RENI_pdl_rsi_m30/wf_stoch
```

Для режима без stochastic используется тот же запуск с `wf_nostoch.json` и отдельным каталогом
`wf_nostoch`. Вердикт принимается по pooled OOS; полный 36-месячный результат не заменяет OOS.

## Честное сравнение

`pdl_rsi_m30`, `rsi_pullback` и `rsi_zone` запускаются на RENI за общий период с одинаковыми
`-cash`, `-commission` и `-risk-pct 1`. В сводке отдельно указываются baseline, in-sample и pooled
OOS, а также доходность, PF, максимальная просадка, число сделок, win rate, среднее удержание и
причины выходов.
