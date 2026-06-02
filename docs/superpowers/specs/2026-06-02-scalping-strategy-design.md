# Дизайн: консервативный часовой скальпинг по российским акциям

Дата: 2026-06-02

## Цель

Новая торговая стратегия скальпинга для российских акций (RUB) на часовом
таймфрейме с консервативным риском. Стратегия не выставляет ордера, а присылает
в Telegram алерты «покупай» (BUY) и «продавай» (SELL). Источник правды об
открытых позициях — выделенный брокерский счёт, ID которого задаётся через env.

## Принципы стратегии

Консервативный скальпинг = вход только **по тренду на откате**, а не ловля ножей.
Это отличает новую стратегию от существующего `scalping_rsi` (покупает любой
RSI < 29 без фильтра тренда).

### Сигнал входа (BUY)

Все условия одновременно на 1h:

1. **Фильтр тренда**: текущая цена выше EMA200 (бумага в восходящем тренде) —
   торгуем только лонг по тренду.
2. **Откат с разворотом**: RSI(14) пересёк уровень 35 снизу вверх (покупаем
   разворот отката, а не падающий нож).
3. **Волатильность**: бумага входит в топ-N самых волатильных (отбор универсума).

### Сигнал выхода (SELL)

TP/SL считаются от **реальной средней цены позиции** (`PurchasePrice`) на
выделенном счёте, ATR — текущий на момент запуска:

- **Take-profit** = `PurchasePrice` + `AtrTakeProfitMult` × ATR (по умолчанию 1.5×)
- **Stop-loss**   = `PurchasePrice` − `AtrStopLossMult` × ATR (по умолчанию 1.0×)
- R:R ≈ 1.5 (асимметрия в нашу пользу).

SELL-алерт шлётся, если по бумаге есть позиция и `Price ≥ TP` или `Price ≤ SL`.

### Управление позициями (без собственного хранилища состояния)

Состояние не храним сами — читаем портфель выделенного счёта каждый запуск:

- `GetPortfolio(cfg.Scalping.AccountID)` → карта `instrumentID → Position`.
- **BUY-алерт** только если по бумаге **нет** открытой позиции и не достигнут
  лимит `MaxOpenPositions`.
- **SELL-алерт** только если позиция **есть** и сработал TP/SL.

Это переживает перезапуск процесса: позиции всегда восстанавливаются из брокера.

## Архитектура

Новый пакет `internal/service/trading_strategy/scalping` по образцу `golden_x`.
Существующий `scalping_rsi` не трогаем — новая стратегия самостоятельна.

Подпакеты:

- `model/` — `Settings` (+ `DefaultSettings()`), `Signal` (BUY/SELL),
  `TradeDecision`.
- `dto/` — входной DTO `Trade{ Interval, ChatID }`.
- `specification/` — `TrendEMA` (цена > EMA200), `RsiReversal` (RSI пересёк
  порог снизу вверх).
- `universe/` — отбор топ-N волатильных акций через сервисы `volatility`/`atr`.
- `notification/` — формирование агрегированного BUY/SELL Telegram-сообщения.
- `scheduler/` — обёртка для cron-запуска в prod (как у остальных стратегий).
- корень: `types.go` (интерфейсы + `service` + `NewService`), `trade.go`
  (оркестрация).

### Переиспользуемые блоки (уже есть в проекте)

- `internal/service/instrument/atr` — ATR (период 14), `TechAnalyse`.
- `internal/service/instrument/volatility` — `CalculateVolatility`.
- `internal/service/instrument/ema` — EMA `TechAnalyse`.
- `internal/service/instrument/rsi` — `CalculateRSI`.
- `pkg/client/grpc.OperationsServiceClient.GetPortfolio(ctx, accountID)` →
  `[]*model.Position` (`ShareID`, `PurchasePrice`, `Price`, `Quantity`).
- `pkg/client/grpc.InstrumentsServiceClient.Shares` — список акций.
- `enum.Hour1` (= 4).

## Поток данных (один запуск `Trade`)

1. Получить все RUB-акции → `universe` отбирает топ-N самых волатильных (по
   ATR% / исторической волатильности за окно).
2. Прочитать `GetPortfolio(cfg.Scalping.AccountID)` → карта открытых позиций.
3. Для каждой бумаги из универсума:
   - **Есть позиция** → посчитать ATR, TP/SL от `PurchasePrice`. Если
     `Price ≥ TP` или `Price ≤ SL` → SELL-сигнал.
   - **Нет позиции** (и `len(open) < MaxOpenPositions`) → проверить вход:
     `цена > EMA200` И RSI(14) пересёк 35 снизу вверх → BUY-сигнал (с
     расчётными TP/SL для информации).
4. Собранные сигналы → **одно агрегированное** Telegram-сообщение за запуск
   (как в golden_x).

## Настройки (`model.Settings` + `DefaultSettings()`)

Алгоритмические кнопки — экспортируемые поля (как в golden_x):

| Поле | По умолчанию | Назначение |
|------|--------------|------------|
| `EmaPeriod` | 200 | период EMA для фильтра тренда |
| `RsiPeriod` | 14 | период RSI |
| `RsiReversalLevel` | 35 | уровень разворота RSI снизу вверх |
| `AtrTakeProfitMult` | 1.5 | множитель ATR для take-profit |
| `AtrStopLossMult` | 1.0 | множитель ATR для stop-loss |
| `UniverseSize` | 10 | сколько самых волатильных бумаг отбирать |
| `MaxOpenPositions` | 5 | лимит одновременно открытых позиций |
| `Interval` | `Hour1` | таймфрейм |

## Конфиг (env)

Новый `internal/config/scalping.go`:

```go
type ScalpingConfig struct {
    AccountID string `config:"SCALPING_ACCOUNT_ID,required,backend=env"`
}

func NewScalpingConfig() *ScalpingConfig {
    return &ScalpingConfig{}
}
```

Подключается в `Config` рядом с `PortfolioYield`. Счёт для скальпинга задаётся
через `env/local.env` переменной `SCALPING_ACCOUNT_ID`.

## Запуск

- `dev`: горутина в `internal/app/app.go:runDev` (как другие стратегии).
- `prod`: cron каждый час в торговую сессию MOEX через `pkg/scheduler`
  (`scheduler/trade.go`).

## Тестирование

Table-driven юнит-тесты (по образцу `golden_x/*_test.go`):

- `specification`: фильтр тренда (цена vs EMA200), RSI-разворот (пересечение
  снизу вверх vs прочие случаи).
- расчёт TP/SL от `PurchasePrice` и ATR.
- логика решения BUY/SELL в зависимости от наличия позиции и `MaxOpenPositions`.
- отбор универсума (сортировка по волатильности, ограничение топ-N).

## Вне области (YAGNI)

- Автоматическое выставление ордеров (только алерты).
- Трейлинг-стоп / частичные выходы.
- Собственное персистентное хранилище позиций (используем брокерский счёт).
- Шорты (только лонг по тренду).
