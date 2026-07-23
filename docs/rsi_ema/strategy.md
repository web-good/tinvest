# RSI + EMA trend (15m, лонг)

Дизайн: `docs/superpowers/specs/2026-07-23-rsi-ema-design.md`
Код: `internal/service/trading_strategy/rsi_ema/strategy/core`

## Правила

**Вход** (только лонг, только внутри сессии Пн–Чт 07:00–18:00 MSK, Пт 07:00–14:00, не на баре конца дня):

1. RSI(`RSIPeriod`, дефолт 12) пересекает уровень `RSIMid` (50) снизу вверх на текущем баре.
2. EMA(`EMAFast`, 10) выше EMA(`EMASlow`, 50) на текущем баре (обе прогреты).
3. Опциональный стоп: при `StopATR > 0` стоп = `вход − StopATR×ATR` (ATR Уайлдера,
   период `ATRPeriod`, на базовом таймфрейме). При `StopATR = 0` стопа нет. Тейка нет.

**Выходы** в порядке приоритета:

1. `SL` — стоп задет лоу бара (только при `StopATR > 0`). Всегда активен.
2. `EMAX` — EMA(`EMAFast`) пересекает EMA(`EMASlow`) сверху вниз. Всегда активен.
3. `RSI70` — RSI пересекает `RSIUpper` (70) **снизу вверх** — фиксация прибыли при входе в
   перекупленность. Всегда активен.
4. `RSI50` — RSI пересекает `RSIMid` (50) **сверху вниз**. **Глушится кулдауном**: учитывается
   только когда прошло ≥ `EntryCooldownBars` баров с момента входа. Причина — вход покупает
   крест RSI 50 вверх, поэтому сразу после входа RSI болтается около 50 и на первом же откате
   мог бы преждевременно закрыть позицию.
5. `EOD` — принудительное закрытие на последнем баре сессии. Всегда активно; позиция не
   переносится через ночь и выходные.

`EMAX`, `RSI70`, `RSI50`, `EOD` заполняются по цене закрытия бара; `SL` — по стоп-цене
(с поправкой на гэп вниз). HTF-фильтра тренда нет.

## Запуск

Таймфрейм — флаг `-interval` (референс `Minutes15`; правила timeframe-agnostic, длину бара
стратегия выводит из данных). H1-серия не нужна.

Разведочный прогон на дефолтах:

```
go run ./cmd/backtest -ticker SBER -strategy rsi_ema -interval Minutes15 \
  -out ./reports/SBER -months 6 -refresh
```

Калибровка (walk-forward):

```
go run ./cmd/backtest -ticker SBER -strategy rsi_ema -interval Minutes15 \
  -calibrate data/params/rsi_ema/grid.json -out ./reports/SBER \
  -months 24 -test-months 6 -min-trades 20 -metric profit_factor -refresh
```

Диагностика одного бара: `-explain "2026-07-20 12:35"` (время MSK; учитывает `-params`).

Грид `data/params/rsi_ema/grid.json`: фазы `entry` (RSIPeriod × EMAFast × EMASlow),
`exits` (RSIUpper × EntryCooldownBars), `risk` (StopATR). 27 + 6×12 + 6×4 = 123 комбинации.

## Критерий приёмки

Только pooled OOS profit factor на walk-forward: PF ≥ 1.5 при ≥ 30 OOS-сделок — кандидат
на live; 1.0–1.5 — edge не подтверждён; сходимость всех комбо или < 10 сделок — сетап
слишком редок (диагностировать через `-explain`).
