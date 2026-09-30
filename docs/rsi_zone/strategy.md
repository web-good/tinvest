# RSI zone (30m, лонг, многодневный)

Код: `internal/service/trading_strategy/rsi_zone/strategy/core/core.go`
Реестр бэктеста: `internal/service/backtest/rsi_zone_registry.go`
Грид калибровки: `data/params/rsi_zone/grid.json`
Пакеты тикеров: `internal/service/trading_strategy/rsi_zone/strategy/<ticker>/` — откалиброванный литерал
и разбор калибровки в доке пакета; гриды тикера — в `data/params/rsi_zone/<ticker>/`. Незарегистрированный
тикер получает `core.DefaultParams()`.
Спека: `docs/superpowers/specs/2026-09-28-rsi-zone-design.md`
Скринер тикеров: `docs/rsi_zone/screener.md` (`go run ./cmd/zonescreen`).

Живой раннер — гость раннера счёта rsi_pullback: [live.md](live.md).

## 1. Идея

Покупка критической перепроданности в тренде. Тренд — цена закрытия выше одной EMA
(`close > EMA(EMAPeriod)`), момент входа — пересечение коротким RSI нижней критической зоны
сверху вниз, выход — пересечение тем же RSI верхней критической зоны снизу вверх, защита —
стоп в дневных ATR, замороженный на входе. Целей, трейла, тайм-стопа и закрытия по концу дня
нет: позиция переносится через ночь и выходные.

Отличие от `rsi_pullback` (`docs/rsi_pullback/strategy.md`): тренд задаёт одна EMA против цены,
а не пара EMA; нет гейта состояния дня, цели, объёмного гейта и трейла.

## 2. Параметры

| Поле | Единица | Дефолт | В гриде |
|---|---|---|---|
| `RSIPeriod` | баров | 4 | entry: 4, 5, 6 |
| `RSILower` | пункты RSI | 25 | entry: 10, 15, 20, 25, 30 |
| `RSIUpper` | пункты RSI | 75 | exit: 60, 65, 70, 75, 80, 85 |
| `EMAPeriod` | баров | 200 | trend: 50, 100, 150, 200 |
| `DailyATRPeriod` | будних дневок | 14 | нет (фиксирован) |
| `StopDailyATR` | множитель дневного ATR | 1.0 | risk: 0.5, 0.7, 1.0, 1.5 |

`StopDailyATR = 0` отключает стоп в ядре, но в сетку намеренно не входит.

## 3. Вход

Позиции нет; проверки на закрытии последнего бара `i`, первый несработавший гейт отклоняет вход:

1. **Будний день** по MSK. Нулевое или невыровненное время бара гейт пропускает.
2. **RSI-крест вниз:** `i-1 ≥ RSIPeriod && rsi[i-1] ≥ RSILower && rsi[i] < RSILower`. Прогрев
   RSI отсекается по индексу, а не по значению: честный RSI 0.00 после серии падающих баров —
   валидное показание. Событие, не состояние: пока RSI сидит в зоне без нового пересечения,
   повторного входа нет.
3. **Тренд:** `EMA[i] > 0 && close[i] > EMA[i]`; равенство — не тренд.
4. **Дневной ATR > 0** — ATR Уайлдера по завершённым будним дневкам (субботы и воскресенья
   выброшены; без `DailyTimes` серия берётся как есть).
5. **Стоп выше нуля:** `stop = close − StopDailyATR × dailyATR`; при включённом стопе и
   `stop ≤ 0` вход отклоняется.

Неположительные `RSIPeriod`/`EMAPeriod` дают «нет сигнала», а не панику.

## 4. Выход

Приоритет `SL → RSI`:

1. **`SL`** — `low ≤ вход − StopDailyATR × EntryATR`. Уровень строится от дневного ATR,
   замороженного на входе, а не от текущего. Без `EntryATR` стопа нет. Цену исполнения
   определяет движок (`model.IsStopReason`: `min(стоп, open)`).
2. **`RSI`** — RSI пересёк `RSIUpper` снизу вверх на текущем баре; исполнение по close.

Календарного гейта у выходов нет: будний день ограничивает только вход (§3, гейт 1), а стоп и
RSI-выход срабатывают на любом баре, включая выходные сессии MOEX.

На общем баре стоп побеждает RSI-выход: внутрибарный порядок из OHLC неизвестен.

## 5. Запуск

```
go run ./cmd/backtest -ticker <TICKER> -strategy rsi_zone -interval Minutes30 -months 36
go run ./cmd/backtest -ticker <TICKER> -strategy rsi_zone -interval Minutes30 \
  -calibrate data/params/rsi_zone/grid.json -out ./reports/<TICKER> -months 36 \
  -min-trades 20 -test-months 6 -metric profit_factor
```

Калибровка — фазовая (entry → trend → exit → risk, keepTop 5, 85 комбинаций). Судить по pooled
OOS profit factor walk-forward, не по лучшей in-sample точке.

У зарегистрированного тикера поля, которых нет в гриде, берутся из его литерала, а не из дефолтов
ядра. Чтобы свипать оси поверх baseline, перечисляйте в гриде все поля.
