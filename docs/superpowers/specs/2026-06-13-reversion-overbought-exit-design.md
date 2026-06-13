# Reversion: выход по перекупленности (take-profit)

Дата: 2026-06-13
Ветка: feat/reversion-rsi-dip
Пакет: `internal/service/trading_strategy/reversion/strategy/core`

## Цель

Добавить в mean-reversion RSI buy-the-dip стратегию четвёртый сигнал выхода:
закрывать открытый лонг, когда **оба** осциллятора одновременно находятся в зоне
перекупленности — `RSI ≥ RSIOverbought` И `Stoch %D ≥ StochOverbought`.

Это take-profit «по happy-path»: фиксируем прибыль на сильном движении вверх.
Существующий выход RSI50 (кросс RSI вниз через 50) **сохраняется без изменений** —
он страхует прибыль при затухании импульса и новым условием не заменяется.

## Решения (согласовано)

- **Триггер — уровневый**, а не кроссовый: выход срабатывает на любом баре, где
  обе зоны достигнуты одновременно (`rsiNow >= RSIOverbought && stochNow >= StochOverbought`).
  Не ждём пробоя зоны вниз.
- **Флаг с дефолтом 1**: реализуется как `UseOverbought int` (0/1), по умолчанию `1`
  во всех тикерах — то есть включено везде. Флаг (а не «вшито всегда») сохраняет
  паттерн `Params`, где каждый тумблер — `int/float64` для грид-калибровки, и позволяет
  отключить/свипать условие без изменения дефолтного поведения.
- **Пороги — параметры**: `RSIOverbought` (default 70) и `StochOverbought` (default 80),
  по аналогии с уже существующими `RSIOversold`/`StochOversold`.
- **Тег сигнала**: `OB`.
- **Приоритет**: новая ветка — **первая** в switch `manage()` (высший приоритет).

## Изменения

### 1. `core.Params` (core.go:32-47)

Добавить три поля:

```go
UseOverbought   int     // 1 = выход при одновременной перекупленности RSI и Stoch; 0 = выкл
RSIOverbought   float64 // RSI зона перекупленности для выхода (default 70)
StochOverbought float64 // Stoch %D зона перекупленности для выхода (default 80)
```

### 2. `manage()` (core.go:334-357)

Добавить новую ветку **первой** в switch:

```go
case s.p.UseOverbought == 1 && in.rsiOK && in.stochOK &&
    in.rsiNow >= s.p.RSIOverbought && in.stochNow >= s.p.StochOverbought:
    sig.Kind, sig.Reason = model.SignalSell, "OB"
    sig.ExitReason = fmt.Sprintf("OB: RSI %.2f ≥ %.0f и Stoch %.2f ≥ %.0f — обе зоны перекупленности",
        in.rsiNow, s.p.RSIOverbought, in.stochNow, s.p.StochOverbought)
```

Гард `in.rsiOK && in.stochOK` защищает от нулевых значений в период прогрева
индикаторов (мирроринг дисциплины остальных веток).

**Анализ конфликтов с существующими ветками:**
- RSI50 (`crossDown` через 50): при `RSI ≥ 70` кросс вниз через 50 невозможен — взаимоисключающи.
- RSIOS (пробой зоны перепроданности ~30 вниз): при `RSI ≥ 70` невозможен — взаимоисключающи.
- ATRSL (цена ниже ATR-стопа): сценарий «глубоко в прибыли по осцилляторам, но цена под стопом»
  практически невозможен; даже при со-триггере фиксация прибыли по перекупленности — желаемое поведение.
- EMAX (медвежий кросс EMA на лагающих линиях): единственный теоретически возможный со-триггер.
  В этом редком случае берём take-profit по перекупленности — это и есть цель. Поэтому OB стоит первой.

### 3. Дефолты по тикерам

Добавить `UseOverbought: 1, RSIOverbought: 70, StochOverbought: 80` в 8 per-ticker
файлов **и** в generic-базлайн (иначе тикеры без своего конфига получат OB выключенным,
что противоречит «включено для всех»):

- `reversion/strategy/rusal/rusal.go`
- `reversion/strategy/ydex/ydex.go`
- `reversion/strategy/gazp/gazp.go`
- `reversion/strategy/afks/afks.go`
- `reversion/strategy/nvtk/nvtk.go`
- `reversion/strategy/mdmg/mdmg.go`
- `reversion/strategy/sber/sber.go`
- `reversion/strategy/plzl/plzl.go`
- `internal/service/backtest/reversion_registry.go` → `genericReversionDefaults()` (reversion_registry.go:51)

Калибровка свипает новые поля автоматически: `ParseParams` (reversion_registry.go:27-32)
анмаршалит частичный JSON-оверрайд поверх дефолтов, поэтому экспортируемых полей
достаточно — доработок кода калибровки не требуется.

### 4. Док-комментарий пакета (core.go:1-13)

Обновить: «exits on one of three signals» → четыре сигнала, добавить описание OB-выхода
(«OB: both RSI and Stochastic %D simultaneously in their overbought zones — take-profit»).

## Тесты (TDD)

Табличные тесты в `core_test.go`:

1. Обе зоны достигнуты (`RSI ≥ 70`, `Stoch ≥ 80`) → `Sell`, reason `OB`.
2. Только `RSI ≥ 70`, `Stoch < 80` → нет выхода (held).
3. Только `Stoch ≥ 80`, `RSI < 70` → нет выхода (held).
4. `UseOverbought=0` при обеих зонах → нет выхода (held).
5. Приоритет: при одновременной перекупленности OB не перетирается другими ветками
   (проверяем `reason == "OB"`).
6. Прогрев: `rsiOK==false` или `stochOK==false` → OB не срабатывает.

## Вне области (не трогаем)

- `Explain()` — описывает только гейты входа; выходы там не показываются.
- Stochastic в логике входа — без изменений (двойное подтверждение перепроданности).
- RSI50 / RSIOS / ATRSL / EMAX — без изменений.

## Совместимость

Дефолт `UseOverbought: 1` включает условие у всех тикеров немедленно — поведение
существующих бэктестов изменится (как и было согласовано). Калибровочные гриды смогут
свипать `UseOverbought`/`RSIOverbought`/`StochOverbought` рефлексией без доработок.
