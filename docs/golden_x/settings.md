# Параметры Golden X

Все настраиваемые ручки стратегии собраны в одной структуре `dto.Settings`. Дефолтные значения возвращает функция `golden_x.DefaultSettings()`.

## Где задаются

| Файл | Что внутри |
|---|---|
| `internal/service/trading_strategy/golden_x/dto/settings.go` | Структура `Settings` с описанием каждого поля |
| `internal/service/trading_strategy/golden_x/settings.go` | `DefaultSettings()` — дефолтные значения |
| `internal/service/trading_strategy/golden_x/settings_test.go` | Drift-тест: фиксирует значения по умолчанию, чтобы случайные правки ловились в CI |

Контракт `DefaultSettings()`: вызов `Detect()` с этими значениями должен давать byte-identical вывод по сравнению с пре-D2 кодом (так зафиксировано в комментарии к функции).

## Полная таблица параметров

| Имя | Тип | Default | Назначение | Где применяется |
|---|---|---:|---|---|
| `BuyGreen` | float64 | **5** | Перцентиль для 🟢 сильной покупки (RSI < P5) | `percentile.go:49`, `percentile.go:34` |
| `BuyYellow` | float64 | **15** | Перцентиль для 🟡 слабой покупки (RSI < P15) | `percentile.go:49`, `percentile.go:34` |
| `SellYellow` | float64 | **80** | Перцентиль для 🟠 частичной продажи (только Dividend) | `percentile.go:62`, `percentile.go:82` |
| `SellOrange` | float64 | **90** | Перцентиль для 🔴 продажи (оба режима) | `percentile.go:62`, `percentile.go:82` |
| `SellRed` | float64 | **95** | Перцентиль для 🚨 финального выхода (только Dividend) | `percentile.go:62`, `percentile.go:82` |
| `ATRPeriod` | int | **14** | Период ATR для расчёта стопа | `detector.go:74` |
| `ATRMultiplierDividend` | float64 | **2.0** | Множитель ATR для стопа Dividend (`Stop = Close − 2.0×ATR`) | `stop.go:9` |
| `ATRMultiplierGrowth` | float64 | **1.5** | Множитель ATR для стопа Growth | `stop.go:9` |
| `VolumeSMALookback` | int | **20** | Окно SMA для проверки объёма | `detector.go:64` |
| `VolumeMultiplier` | float64 | **1.5** | Множитель: `Volume > 1.5 × SMA20` → подтверждение 🔊 | `detector.go:64` |
| `TrendEMAPeriod` | int | **200** | Период EMA для фильтра тренда (обе стратегии) | `trend_filter.go:42` |
| `AdaptiveWindowMin` | int | **100** | Минимум недель RSI для адаптивных перцентилей (включая текущую формирующуюся); меньше — `ErrAdaptiveInsufficientHistory` | `detector.go:26` |
| `AdaptiveWindowMax` | int | **200** | Максимум недель RSI (включая текущую формирующуюся); история обрезается до последних N значений | `detector.go:26` |
| `DivergenceLookbackWeeks` | int | **52** | Глубина поиска фрактального минимума для дивергенции | `detector.go:51` |

## Фунд-бонус к Score (не часть `Settings`)

`Score` каждой покупки может получать +0..+3 от абсолютных полос композита дивидендного скринера (см. [strategy.md §Score сигнала](strategy.md#score-сигнала) и `docs/dividend/screener.md`). Это не поле `dto.Settings` — оно подключается снаружи через `golden_x.WithRankProvider(p dividend.RankProvider)` (см. `internal/service_provider/service.go:GetGoldenXTradingService`), а данные обновляются скринером `internal/service/screener/dividend` раз в ~24ч. Без вызова `WithRankProvider` бонус всегда 0 (no-op) — `Score` ведёт себя как раньше.

## Per-share параметр: `RSILength`

Кроме общих `Settings`, каждая бумага имеет свой период RSI — `RSILength`. Он задан в списке тикеров `shares/shares.go` и подбирается эмпирически: для волатильных бумаг короче (резвее реагирует), для стабильных длиннее (меньше шума).

### Список Dividend (🥇)

| Тикер | ID | RSILength |
|---|---|---:|
| Сургутнефтегаз - прив | `a797f14a-…cc00` | 10 |
| Татнефть - прив | `efdb54d3-…39f6` | 9 |
| Роснефть | `fd417230-…ec6b` | 9 |
| Лукойл | `02cfdf61-…af3` | 9 |
| Сбер Банк - прив | `c190ff1f-…9ca5` | 8 |
| Северсталь | `fa6aae10-…d096` | 11 |
| НЛМК | `161eb0d0-…b508` | 8 |
| ММК | `7132b1c9-…61d9` | 9 |
| ФосАгро | `9978b56f-…d194` | 7 |
| Транс нефть | `653d47e9-…4694` | 9 |
| Банк Санкт-Петербург | `1e19953d-…4029` | 8 |

### Список Growth (🥈)

| Тикер | ID | RSILength |
|---|---|---:|
| Мать и дитя | `0d53d29a-…cb46` | 7 |
| Газпром | `962e2a95-…643a` | 8 |
| Яндекс | `7de75794-…3822` | 7 |
| Полюс | `10620843-…d3e1` | 7 |
| Т-Технологии | `87db07bc-…1d7b` | 8 |
| НОВАТЭК | `0da66728-…67ee` | 9 |

Полные ID — в `internal/service/trading_strategy/golden_x/shares/shares.go`.

## Рецепты тюнинга

**Хочу больше сигналов покупки.** Поднять `BuyYellow` с 15 до 20 — расширит жёлтую зону. Или `BuyGreen` с 5 до 8 — увеличит частоту 🟢. Минус: больше ложных срабатываний.

**Хочу более глубокий стоп (давать акции дышать).** Поднять `ATRMultiplierDividend` (например, с 2.0 до 2.5) и/или `ATRMultiplierGrowth` (с 1.5 до 2.0). Минус: бо́льшие убытки на стопе.

**Не доверяю подтверждению объёмом.** Поднять `VolumeMultiplier` (например, до 2.0) — будет требовать более явный всплеск объёма. Или просто игнорировать значок 🔊 в алерте.

**Дивергенция слишком короткая / длинная.** Менять `DivergenceLookbackWeeks` — 52 недели (год) это компромисс. Увеличить до 78 (1.5 года) — больше шансов поймать дивергенцию, но риск зацепить нерелевантный pivot.

**Хочу более «свежие» пороги.** Уменьшить `AdaptiveWindowMax` (например, до 150) — пороги будут больше зависеть от недавнего поведения. Минус: на молодых тикерах сужает выборку и делает пороги неустойчивыми.

**Хочу EMA50 вместо EMA200 для Growth.** Уменьшить `TrendEMAPeriod` до 50. Тренд-фильтр станет агрессивнее (чаще будет менять ✅↔🚫).

## Где применить новые значения

В коде дефолты прописаны в `settings.go`:

```go
func DefaultSettings() dto.Settings {
    return dto.Settings{
        BuyGreen:                5,
        BuyYellow:               15,
        // ... остальные
    }
}
```

Способы изменить:

1. **Глобально** — поправить `DefaultSettings()`. Затронет и live, и бэктест.
2. **Только бэктест** — добавить новую функцию вида `AlternativeSettings()` и переключить вызов в `cmd/backtest/main.go:83`. Полезно для AB-сравнения.
3. **На лету для одного прогона** — реализовать новый CLI-флаг и парсить значения. Сейчас не сделано (см. [backtest.md §5.6](backtest.md#где-менять-параметры-стратегии)).

При любой правке `DefaultSettings()` — обновить `settings_test.go`, иначе CI зафейлит.

## Per-share `RSILength` — как менять

Редактировать `shares/shares.go`:

```go
c.Add(collection.Instrument{
    ID:        "uuid-from-tinkoff-api",
    RSILength: 9,                    // эмпирически 7..11
    Name:      "Название бумаги",
})
```

Подбор `RSILength`: запустить бэктест ([backtest.md](backtest.md)) для разных значений на одном тикере, выбрать тот, что даёт лучший WinRate / Cumulative% при приемлемой просадке.
