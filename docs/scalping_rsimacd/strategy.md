# Scalping RSI+MACD (5m, лонг)

Дизайн: `docs/superpowers/specs/2026-07-22-scalping-rsimacd-design.md`
Код: `internal/service/trading_strategy/scalping_rsimacd/strategy/core`

## Правила

**Вход** (только лонг, только внутри сессии Пн–Чт 08:00–17:00 MSK, Пт 08:00–14:00):

1. На текущем баре MACD(3,6,9) пересекает сигнальную линию снизу вверх, **обе линии ниже нуля**.
2. В окне из `MACDConfirmBars` баров до текущего (включая сам бар кросса) RSI(`RSIPeriod`)
   пересёк уровень `RSIOversold` снизу вверх и закрылся выше него.
3. Уровень стопа держался с момента RSI-кросса, а закрытие текущего бара выше него.
4. Риск входа лежит в границах `[MinRiskATR×ATR, MaxRiskATR×ATR]`.

**Стоп:** минимум свечи RSI-кросса минус `StopBufferATR×ATR`.
**Тейк:** `вход + RR × риск`.

**Выходы** в порядке приоритета: `SL` → `TP` → `STOCH` (%K классического стохастика (14,3,3)
выходит вниз из зоны 80, закрытие по цене закрытия бара) → `EOD` (принудительное закрытие
на последнем баре сессии; позиция не переносится через ночь и выходные).

## Запуск

```
go run ./cmd/backtest -ticker <TICKER> -strategy scalping_rsimacd -interval Minutes5 \
  -calibrate data/params/scalping_rsimacd/grid.json -out ./reports/<TICKER> \
  -months 6 -test-months 3 -min-trades 20 -metric profit_factor
```

Диагностика одного бара: добавить `-explain "2026-07-20 12:35"` (время MSK).

## Критерий приёмки

Решение принимается только по pooled OOS profit factor на walk-forward: PF ≥ 1.5 при
≥ 30 OOS-сделок — кандидат на live; 1.0–1.5 — edge не подтверждён; сходимость всех комбо
или < 10 сделок — сетап слишком редок, диагностировать через `-explain`.

## Разведочный прогон (single-run, defaults, 2026-07-22)

`go run ./cmd/backtest -ticker SBER -strategy scalping_rsimacd -interval Minutes5 -out ./reports/SBER -months 6`
(defaults из `core.DefaultParams()`, без калибровки) дал 533 сделки за 6 месяцев,
profit factor 0.158, win rate 16.5%, чистый PnL −45.21%. Это не повод менять пороги
(они зафиксированы спекой) — калибровочный грид (`data/params/scalping_rsimacd/grid.json`)
и walk-forward должны решить, есть ли здесь эдж на дефолтных геометриях вообще, а не этот
разведочный прогон.
