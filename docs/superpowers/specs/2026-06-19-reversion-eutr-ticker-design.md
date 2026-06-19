# Добавление тикера EUTR в reversion-стратегию

**Дата:** 2026-06-19
**Ветка:** feat/reversion-rsi-dip
**Статус:** дизайн согласован, ожидает реализации

## Контекст и цель

EUTR (ЕвроТранс) занял **1-е место** в свежем reversion-fitness скрине
(`reports/screener/volatility_Day1_20260619_184105.md` и
`reports/volotility/volatility_Day1_20260619_184634.md`):

| Тикер | Score | ATR% | autocorr | Класс | Цена |
|-------|-------|------|----------|-------|------|
| EUTR  | 0.869 | 3.79 | −0.112   | mean-reverting | ~701 |

Высокий композитный score, отрицательная автокорреляция и классификация
`mean-reverting` делают EUTR естественным кандидатом для buy-the-dip reversion.
Цель — провести EUTR через тот же конвейер калибровки, что сегодня прошли
UGLD (успех, зарегистрирован рабочим) и SFIN (калибровка провалена,
зарегистрирован с пометкой DO NOT TRADE), и зарегистрировать его как
полноценный per-ticker reversion-тикер.

## Ключевое ограничение: глубина истории

EUTR провёл IPO в ноябре 2023. Доступная Hour1-история:

- **2023-11-21 → 2026-06-19, ~31 месяц** (15136 часовых свечей, закешировано в
  `data/candles/EUTR_Hour1.json`).

Это **короче** 36-месячного окна, на котором калибровались UGLD/SFIN. Поэтому
walk-forward адаптируется:

- Окно калибровки: `-months 31` (вместо 36).
- Train 12 / OOS 6, шаг 6 → **~3 OOS-фолда** вместо 4.
- Это сознательный компромисс: меньше фолдов = слабее статистическая
  устойчивость вывода. Факт явно фиксируется в комментарии `eutr.go`, чтобы при
  чтении конфигурации было видно, что EUTR валидирован на более коротком окне,
  чем остальные тикеры.

## Архитектура

Никаких изменений в движке. EUTR повторяет структуру UGLD/SFIN:

1. **Per-ticker пакет** `internal/service/trading_strategy/reversion/strategy/eutr/eutr.go`:
   - `const Ticker = "EUTR"`
   - `func DefaultParams() core.Params` — итоговая конфигурация после калибровки.
   - Подробный doc-комментарий в стиле ugld.go/sfin.go: дата, окно, число фолдов,
     pooled OOS PF, какие гейты ON/OFF и почему.
2. **Регистрация** в `internal/service/backtest/reversion_registry.go`:
   - импорт `reversioneutr "tinvest/internal/service/trading_strategy/reversion/strategy/eutr"`;
   - строка `reversioneutr.Ticker: reversionBindingFor(reversioneutr.Ticker, reversioneutr.DefaultParams)` в `reversionRegistry`.
3. **Grid-файлы** `data/params/eutr/reversion_cal_*.json` (10 шт.) — копии
   ugld/sfin сеток с поправленным `_comment` (тикер EUTR, `-interval Hour1`,
   окно `-months 31 -train-months 12 -test-months 6`). Сами сетки параметров не
   меняются — они тикеро-независимы.

## Процедура калибровки (rolling walk-forward)

Для каждого слоя:

```
go run ./cmd/backtest -ticker EUTR -strategy reversion -interval Hour1 \
  -calibrate data/params/eutr/reversion_cal_<layer>.json \
  -out ./reports/EUTR_<layer> \
  -months 31 -train-months 12 -test-months 6 \
  -metric profit_factor -min-trades 20
```

Порядок слоёв (база накапливается: каждый следующий слой наследует выигравшую
конфигурацию предыдущих):

1. `entry` — ядро входа (RSI → Stoch → EMA, staged-фазы внутри грида). Гоняется
   всегда, даже если скрин говорит «фильтры мертвы»: это сердце стратегии.
2. `trend` — фильтр тренда (UseTrend).
3. `htf` — 4H higher-timeframe тренд (HTFTrendEMA; грид включает 0, т.е.
   проверяет в т.ч. «выключить лучше всех»).
4. `regime` — ADX range-режим (UseRegime).
5. `volume` — объёмный гейт (UseVolume).
6. `overbought` — OB-тейк-профит выход (UseOverbought).
7. `breakeven` — безубыток (UseBreakeven).
8. `trail` — ATR-трейл (UseTrail).
9. `atrstop` — ATR-стоп (UseATRStop).
10. `catstop` — катастрофический стоп / risk-sizing (CatStopATRMult).

**Правило стадии-гейта (stage gate):** опциональный фильтр включается в базу
**только если он улучшает pooled OOS PF** относительно текущей накопленной базы.
Если ни одно значение грида не бьёт базу — фильтр остаётся OFF, и следующий слой
наследует базу без него. Всегда-включённые выходы (RSIOS, EMAX) остаются live
независимо от калибровки.

## Развилка по исходу

После прохождения всех слоёв — две ветки (как сегодня для UGLD/SFIN):

- **OOS-прибыльно** (pooled OOS PF > 1 с положительным compounded на всех/почти
  всех фолдах, профиль UGLD): `DefaultParams()` = откалиброванная конфигурация.
  Комментарий описывает фолды, pooled PF, compounded%, число сделок, какие гейты
  ON/OFF и почему.
- **OOS-убыточно** (pooled OOS PF ≤ 1 или нестабильность между фолдами, профиль
  SFIN): `DefaultParams()` = голое baseline-ядро (модальные значения cal_entry +
  всегда-включённые выходы, все опциональные фильтры OFF). Шапка комментария —
  `⚠️ CALIBRATION FAILED — DO NOT TRADE THIS TICKER LIVE`, чтобы результат был
  воспроизводим и тикер не попал в live-вселенную.

В обоих случаях EUTR остаётся **зарегистрированным**.

## Тестирование

- `go build ./...` — пакет компилируется, регистрация валидна.
- `go test ./internal/service/...` — существующие тесты registry/core зелёные.
- Новый пакет тривиален (только данные, без новой логики), отдельных юнит-тестов
  не требует — покрытие даёт существующий `reversion_registry`/`core`.
- walk-forward отчёты сохраняются под `reports/EUTR_<layer>/` как
  воспроизводимое свидетельство калибровки.

## Вне объёма (YAGNI)

- Изменения движка `core`.
- Live-интеграция EUTR в `internal/app` (торговый воркер).
- Новые измерения grid / новые типы фильтров.
- Рефакторинг существующих тикеров.
