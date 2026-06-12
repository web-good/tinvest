# Reversion v3 — RSI + Stochastic двойное подтверждение — Design Spec

**Дата:** 2026-06-12
**Ветка:** `feat/reversion-rsi-dip` (продолжение)
**Статус:** утверждён к реализации

## Цель

Сделать Stochastic постоянной частью ядра стратегии `reversion` (а не опциональным
гейтом) и заменить одноиндикаторный RSI-триггер на правило согласия двух осцилляторов:
**«один индикатор уже в критической зоне, второй только заходит в неё»**. Зоны и периоды
Stochastic подбираются через grid-калибровку.

## Контекст

После переработки `reversion` стал чистым дневным RSI mean-reversion
(`docs/superpowers/specs/2026-06-12-reversion-rsi-daily-rework-design.md`): вход/выход —
зонные кроссы RSI с переключателями `EntryMode`/`ExitMode` (вход/выход из зоны),
обязательный ATR-стоп, опциональный тренд-фильтр. Stochastic существует в
`pkg/indicators` (функция `Stochastic` → последние %K/%D), но в стратегии не используется.

Эта итерация добавляет Stochastic в логику решения и убирает `EntryMode`/`ExitMode`
(направление кросса теперь зафиксировано: всегда «вход в зону»).

## Семантика решения

Рабочая линия Stochastic — **%D** (SMA от %K за `StochDSmooth` баров; при
`StochDSmooth=1` равна сырому %K). Для детекции кросса нужны значения текущего и
предыдущего бара обоих осцилляторов.

«Уже в зоне» = текущее значение за порогом. «Только заходит» = кросс порога на этом баре
(`prev` снаружи, `now` внутри). Такое определение естественно покрывает случай, когда оба
индикатора входят в зону на одном баре (это тоже срабатывание: один кроссит, другой при
этом уже `now` за порогом).

### Вход (buy) — позиция закрыта

Гейты в порядке короткого замыкания:

1. **Тренд (опц., `UseTrend=1`):** `EMA(FastEMA) > EMA(SlowEMA)` И `close > EMA(SlowEMA)`.
   При `UseTrend=0` пропускается.
2. **Двойное подтверждение перепроданности** (хотя бы одно из):
   - `crossDown(RSI, RSIOversold)` И `stochD_now < StochOversold`
   - `crossDown(stochD, StochOversold)` И `RSI_now < RSIOversold`
3. **ATR-стоп обязателен:** `ATRMult > 0` И `ATR > 0`; `stop = entry − ATRMult×ATR`;
   риск `entry − stop > 0`. Стоп замораживается на входе.

Оба осциллятора обязательны: при `RSIPeriod <= 0`, `StochKPeriod <= 0` или
`StochDSmooth <= 0` вход не рассматривается (нет валидного чтения).

### Выход (sell) — позиция открыта

Защита проверяется первой (худший случай для позиции выигрывает ничью):

1. **SL:** `barLow ≤ frozen stop` → reason `SL`.
2. **Двойное подтверждение перекупленности** (хотя бы одно из) → reason `XOVER`:
   - `crossUp(RSI, RSIOverbought)` И `stochD_now > StochOverbought`
   - `crossUp(stochD, StochOverbought)` И `RSI_now > RSIOverbought`

## Params (новая форма)

| Поле | Тип | Назначение |
|------|-----|-----------|
| `UseTrend` | int 0/1 | требовать восходящий тренд перед покупкой |
| `FastEMA` | int | быстрая EMA режима |
| `SlowEMA` | int | медленная EMA режима + пол цены |
| `RSIPeriod` | int | длина RSI (обязательно >0) |
| `RSIOversold` | float64 | нижняя зона RSI (вход) |
| `RSIOverbought` | float64 | верхняя зона RSI (выход) |
| `StochKPeriod` | int | период %K (обязательно >0) |
| `StochDSmooth` | int | сглаживание %D (обязательно >0; 1 = сырой %K) |
| `StochOversold` | float64 | нижняя зона Stochastic (вход) |
| `StochOverbought` | float64 | верхняя зона Stochastic (выход) |
| `ATRPeriod` | int | длина ATR для стопа |
| `ATRMult` | float64 | множитель ATR-стопа (>0) |

**Удаляются:** `EntryMode`, `ExitMode`. Все поля — int/float64 для рефлексивной
grid-калибровки.

## Изменения в индикаторах

Текущая `indicators.Stochastic` возвращает только последние %K/%D — недостаточно для
кросса. Добавляется `StochasticSeries(highs, lows, closes, kPeriod, dSmooth) ([]float64,
[]float64)` (срезы %K и %D, выровненные по правому краю), по образцу `RSISeries`. Ядро
берёт `stochD_now`/`stochD_prev` из последних двух элементов %D-серии. Существующая
`Stochastic` остаётся (используется в тестах/иных местах), `StochasticSeries` может
выразить её через себя или жить рядом.

## Lookback

Окно свечей должно покрывать самого «прожорливого» потребителя:
`max(SlowEMA, FastEMA, RSIPeriod+1, StochKPeriod+StochDSmooth+1, ATRPeriod+1) + запас`.

## Explain

Диагностика `Explain` показывает оба осциллятора: тренд (если включён), затем
двойное-перепроданность с фактическими значениями RSI и %D (now/prev) и порогами, затем
ATR-стоп. Блокирует на первом непройденном гейте.

## Дефолты (baseline, одинаковый для 8 тикеров + generic)

```
UseTrend: 1, FastEMA: 50, SlowEMA: 200,
RSIPeriod: 14, RSIOversold: 20, RSIOverbought: 70,
StochKPeriod: 14, StochDSmooth: 3, StochOversold: 20, StochOverbought: 80,
ATRPeriod: 14, ATRMult: 1.0
```

Некалиброванный — пользователь гоняет `-calibrate` и хардкодит победителей по тикеру.

## Grid (3 фазы)

```json
{
  "phases": [
    { "name": "entry", "keepTop": 5, "grid": {
        "UseTrend": [0, 1], "FastEMA": [50], "SlowEMA": [200],
        "RSIPeriod": [7, 14], "RSIOversold": [15, 20, 30],
        "StochKPeriod": [9, 14], "StochDSmooth": [1, 3], "StochOversold": [15, 20, 25]
    }},
    { "name": "exit", "keepTop": 5, "grid": {
        "RSIOverbought": [70, 80], "StochOverbought": [75, 80, 85],
        "ATRMult": [1.0, 1.5, 2.0], "ATRPeriod": [14]
    }}
  ]
}
```

Зоны (`StochOversold`/`StochOverbought`) и периоды (`StochKPeriod`/`StochDSmooth`)
Stochastic свипаются grid-ом — основное требование пользователя.

## Затрагиваемые файлы

- `pkg/indicators/stochastic.go` — `+StochasticSeries`; `pkg/indicators/stochastic_test.go` — тест серии.
- `internal/service/trading_strategy/reversion/strategy/core/core.go` — новый `Params`,
  двойная логика входа/выхода, `buildInput` со Stoch-сериями, `Lookback`, `Explain`.
- `.../core/core_test.go` — переписанные табличные тесты под двойное подтверждение.
- `.../strategy/{afks,gazp,mdmg,nvtk,plzl,rusal,sber,ydex}/<ticker>.go` — `DefaultParams`.
- `internal/service/backtest/reversion_registry.go` — `genericReversionDefaults`.
- `internal/service/backtest/reversion_registry_test.go` — override-тест на `StochOversold`/`ATRMult`.
- `data/params/{afks,gazp,mdmg,nvtk,plzl,rual,sber,ydex}/reversion_grid.json` — новые сетки.
- `docs/reversion/strategy.md` — переписать объяснение.

## Тестирование

- `StochasticSeries`: фикстура-рампа, выравнивание длины, warm-up (короткая история → пустые/нулевые).
- Ядро: вход срабатывает по каждой из двух веток (RSI-кроссит+Stoch-уже / Stoch-кроссит+RSI-уже),
  одновременный вход обоих, отсутствие срабатывания если второй не в зоне; зеркальные тесты
  выхода `XOVER`; приоритет `SL` над `XOVER`; тренд-тумблер; санити ATR-стопа; `Explain`
  блокирует на тренде/RSI/Stoch.
- registry: generic-дефолты валидны, override через `ParseParams` слой поверх дефолтов.
- Полная сборка/vet/тесты зелёные; дымовая калибровка одного тикера (`-interval Day1`).

## Вне области (YAGNI)

- `-basket` walk-forward (runner всё ещё momentum-only).
- Кросс %K×%D как отдельный сигнал (используем только зонные кроссы %D).
- Тайм-стоп / volume-гейт (удалены ранее, не возвращаются).
