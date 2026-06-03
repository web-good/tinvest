# Алгоритм RUSAL — адаптивная скальпинг-стратегия

Реализация: `internal/service/trading_strategy/scalping/strategy/rusal/rusal.go`.

## Идея

RUAL на H1 — среднеликвидная циклическая бумага MOEX (алюминий, USD/RUB, новости/санкции).
Бóльшую часть времени она ходит **в боковике** с **эпизодическими импульсными выносами**.
Одно-режимное правило ошибается в половине случаев: трендfølger «кровит» в боковике, а
mean-reverter попадает под каток в тренде.

Поэтому стратегия **адаптивная и режимо-зависимая**: сначала определяет режим рынка по
ADX, затем применяет mean-reversion в боковике и momentum в тренде. Стратегия —
**только лонг** и **stateless**: `Decide` переоценивается каждый час на свежих свечах,
без памяти между вызовами.

## Жизненный цикл сигнала

```
[свечи из Tinkoff, Lookback штук]
       ↓
[MarketData: Highs/Lows/Closes/Volumes/Price/Position]   ← раннер buildMarketData()
       ↓
[Decide(): расчёт индикаторов]
   ├── ema.Compute(closes, EMAPeriod)        → emaNow
   ├── indicators.RSISeries(closes, RSIPeriod) → rsiPrev, rsiNow
   ├── indicators.ATR(...)                    → atr
   ├── indicators.ADX(...)                    → adx, diPlus, diMinus
   ├── indicators.Donchian(...)               → donUpper, donLower
   ├── emaTouched(lows, ema, PullbackWindow)  → флаг «откат к EMA»
   └── recentHigh(highs, ChandelierWindow)    → chandelierHigh
       ↓
[decide(): чистое ядро решения]
   ├── regimeOf(adx)  → trend / range / dead
   ├── если есть позиция → выход по режиму
   └── если позиции нет  → вход по режиму
       ↓
[model.Signal] → раннер → notification.Trade() → Telegram
```

`Decide()` (`rusal.go:108`) считает индикаторы и упаковывает скаляры в `decideInput`,
после чего всю логику принимает **чистое ядро** `decide()` (`rusal.go:149`) — без
индикаторной математики, тривиально тестируемое таблицей. В конце `Decide` штампует
`sig.Ticker = "RUAL"`.

## Шаг 1. Определение режима по ADX

`regimeOf(adx)` (`rusal.go:77`) классифицирует рынок по значению ADX:

| Условие | Режим | Что делаем |
|---|---|---|
| `adx <= 0` | **dead** | ADX вернул 0 (мало истории/невалидно) — никаких новых входов |
| `adx >= ADXTrendLevel` (25) | **trend** | импульсная логика |
| `adx <= ADXRangeLevel` (20) | **range** | mean-reversion |
| между 20 и 25 | **dead** | «мёртвая зона»: новых входов нет |

Порог `adx <= 0` проверяется **первым** — это защита от того, чтобы нулевой ADX (признак
недостатка истории) случайно не попал в ветку range. Направление тренда берётся из
DI+/DI− (`diPlus`, `diMinus`), которые ADX отдаёт вместе со своим значением.

## Шаг 2. Вход (позиции нет)

Вход возможен только в режимах trend и range; в dead-зоне входов нет.

### Тренд: откат к EMA + разворот RSI вверх

Все условия должны выполняться одновременно (`rusal.go:184`):

- `diPlus > diMinus` — тренд направлен вверх;
- `emaTouched` — на одной из последних `PullbackWindow` (5) свечей `Low` касался EMA
  (`Low <= EMA*(1+EMATouchTol)`), т.е. был откат к скользящей;
- `rsiPrev < RSITrendLevel && rsiNow >= RSITrendLevel` — RSI пересёк уровень 45 снизу
  вверх (в тренде откаты неглубокие, поэтому порог высокий);
- `price > emaNow` — цена снова выше EMA, тренд цел.

→ **Buy**. `StopLoss = price - SLMult*ATR`; `TakeProfit = price + TrailMult*ATR`
(ориентир, в тренде реальный выход — трейлинг).

### Боковик: нижняя граница Donchian + разворот RSI вверх

Условия (`rusal.go:191`):

- `price <= donLower*(1+BandTol)` — цена у нижней границы канала (в пределах 0.3 %);
- `rsiPrev < RSIRangeLevel && rsiNow >= RSIRangeLevel` — RSI пересёк уровень 35 снизу
  вверх (перепроданность, порог ниже трендового).

→ **Buy**. `StopLoss = price - SLMult*ATR`; `TakeProfit = (donUpper + donLower)/2`
(середина канала).

## Шаг 3. Выход (позиция есть)

Выход — **режимо-зависимый**, но в обоих режимах действует жёсткий начальный ATR-стоп.
`hardSL = pos.PurchasePrice - SLMult*ATR`.

### В тренде: ATR-chandelier трейлинг (`rusal.go:156`)

`chandelier = chandelierHigh - TrailMult*ATR`, где `chandelierHigh` — максимум High за
последние `ChandelierWindow` (20) свечей.

| Условие | Результат |
|---|---|
| `price <= hardSL` | **Sell**, `Reason = "SL"` |
| `price <= chandelier` | **Sell**, `Reason = "TRAIL"`, `StopLoss = chandelier` |
| иначе | hold; в сигнал кладётся `StopLoss = hardSL` как защитный пол |

В тренде фиксированного тейк-профита нет — прибыль ведёт трейлинг, поэтому
`TakeProfit` остаётся 0.

### В боковике / dead-зоне: возврат к середине канала (`rusal.go:169`)

`mid = (donUpper + donLower)/2`.

| Условие | Результат |
|---|---|
| `price <= hardSL` | **Sell**, `Reason = "SL"` |
| `price >= mid && mid > 0` | **Sell**, `Reason = "TP"` |
| иначе | hold; в сигнал кладётся `StopLoss = hardSL`, `TakeProfit = mid` |

Условие `mid > 0` защищает от ложного TP на вырожденном Donchian (когда канала ещё нет).

## Statelessness

Стратегия не хранит память между часами:

- цена входа берётся из `pos.PurchasePrice` (портфель из gRPC);
- `chandelierHigh` — это скользящий `max(Highs)` за `ChandelierWindow`, т.е. стандартный
  N-барный chandelier, а **не** истинный максимум с момента входа — поэтому память
  не нужна;
- режим переоценивается каждый бар. Позиция, открытая в тренде, который затем выродился
  в боковик, корректно передаётся в боковой выход (фиксация у середины канала) — это
  желаемое поведение.

## Индикаторы

Все индикаторы — чистые функции, возвращают **последнее** значение и молча отдают нули
при недостатке истории или несовпадении длин срезов (как существующий `ATR`).

| Индикатор | Файл | Сигнатура | Возвращает 0 когда |
|---|---|---|---|
| ADX/DMI (Wilder) | `pkg/indicators/adx.go` | `ADX(highs, lows, closes, period) (adx, diPlus, diMinus)` | `period<=0`, разные длины, `len < 2*period+1` |
| Donchian | `pkg/indicators/donchian.go` | `Donchian(highs, lows, period) (upper, lower)` | `period<=0`, разные длины, `len < period` |
| ATR | `pkg/indicators` | `ATR(highs, lows, closes, period)` | (существующий) |
| RSI (Wilder, ряд) | `pkg/indicators` | `RSISeries(closes, period)` | (существующий) |
| EMA (ряд) | `internal/domain/ema` | `Compute(closes, period)` | (существующий) |

**ADX** считается по Уайлдеру: для каждого бара `+DM`/`−DM` и `TR`, двойное сглаживание
(сначала DI/DX за первые `period` приращений, затем ADX за следующие `period` значений
DX, далее сглаживание до последнего бара). Минимум `2*period+1` свечей покрывает оба
сглаживания — отсюда `Lookback() = 6*ADXPeriod + DonchianPeriod + 50` (≈ 154 с
дефолтами), с запасом на самый «прожорливый» индикатор.

## Тестирование

- `pkg/indicators/adx_test.go` — строгий тренд (ADX=100), нисходящий тренд, флэт (ADX=0),
  смешанный ряд (`0 < adx < 100`), охранные случаи (`period<=0`, мало истории,
  несовпадение длин, nil).
- `pkg/indicators/donchian_test.go` — известный max/min, охранные случаи.
- `rusal_test.go` — табличные тесты чистого `decide` (входы/выходы по обоим режимам,
  hold, вырожденный Donchian, продавленная цена→SL), плюс e2e `Decide` через реальный
  расчёт индикаторов с детерминированными `Params`.

## Происхождение и история

- Дизайн: `docs/superpowers/specs/2026-06-03-rusal-adaptive-scalping-design.md`.
- План реализации: `docs/superpowers/plans/2026-06-03-rusal-adaptive-scalping.md`.
- Параметры и их калибровка: [settings.md](settings.md).
