# Golden X: убрать dedup из уведомлений

## Проблема

Сейчас уведомления стратегии Golden X включают только те акции, у которых изменился snapshot (tier, trend, divergence, volume) — логика `alertState.ShouldAlert` в `dedup.go`. Акции, стабильно находящиеся в зоне покупки/продажи, исчезают из уведомлений после первого алерта.

## Цель

Каждый тик стратегии отправляет уведомление со **всеми** акциями, чей RSI находится в зоне покупки (`< p15`) или продажи (`> p80/p90/p95`), независимо от изменений в snapshot.

## Дизайн

### 1. Перенос `alertTier` и tier-констант в `percentile.go`

Тип `alertTier` и константы `tierNone..tierSellRed` переносятся в `percentile.go`, где уже живут `tierFromAdaptive` и `sellTierFromAdaptive`.

### 2. Удаление dedup-логики

- Удалить `dedup.go` (содержит `alertSnapshot`, `alertState`, `newAlertState`, `ShouldAlert`)
- Удалить `dedup_test.go`

### 3. Очистка `types.go`

- Убрать поле `state *alertState` из структуры `service`
- Убрать вызов `newAlertState()` из конструктора `NewService`

### 4. Упрощение `trade.go`

Убрать из цикла:
- Создание `alertSnapshot`
- Проверку `s.state.ShouldAlert(share.ID, snap)`

Акции с `tierNone` по-прежнему не попадают в уведомление. Все остальные попадают в `buyInfo` или `sellInfo` безусловно.

### Затрагиваемые файлы

| Файл | Действие |
|------|----------|
| `dedup.go` | Удалить |
| `dedup_test.go` | Удалить |
| `percentile.go` | Добавить `alertTier` + tier-константы |
| `types.go` | Убрать `state` поле |
| `trade.go` | Убрать dedup-проверку из цикла |

### Что не меняется

- `detector.go` — использует `tierFromAdaptive`/`tierGreen`/`tierYellow`/`tierNone` из `percentile.go`, работает без изменений
- `notification/notifications.go` — формат уведомления не меняется
- Backtesting — не использует dedup
