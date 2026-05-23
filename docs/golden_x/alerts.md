# Алерты Golden X — расшифровка

Стратегия шлёт уведомления через Telegram Bot API в HTML-режиме. Один прогон шедулера порождает **одно** сообщение:

- **Trade Message** — основной алерт с покупками и продажами. Формируется в `notification.Trade()` (`notification/notifications.go:12`). Начинается с блока легенды, объясняющего все emoji.

## Каталог иконок

Все emoji, используемые в сообщениях:

| Emoji | Что означает | Условие | Источник |
|---|---|---|---|
| 🥇 | Стратегия **Dividend (Gold)** | `kind = StrategyKindDividend` | `dto/strategy_kind.go:11` |
| 🥈 | Стратегия **Growth** | `kind = StrategyKindGrowth` | `dto/strategy_kind.go:13` |
| 🟢 | Сильная покупка | `RSI < P5` | `notification/notifications.go:107` |
| 🟡 | Слабая покупка | `P5 ≤ RSI < P15` | `notification/notifications.go:109` |
| 🟠 | Частичная продажа (Gold only) | `RSI > P80` | `notification/notifications.go:139` |
| 🔴 | Продажа | `RSI > P90` | `notification/notifications.go:137` |
| 🚨 | Финальный выход / тревога (Gold only) | `RSI > P95` | `notification/notifications.go:135` |
| ✅ | Тренд за нас | `Close > EMA200_W` | `dto/trend_status.go:18` |
| 🚫 | Тренд против | `Close ≤ EMA200_W` | `dto/trend_status.go:20` |
| 📈 | Бычья RSI-дивергенция | `bullishDivergence() = true` | `notification/notifications.go:73` |
| 🔊 | Объём подтверждён | `Volume > VolumeMultiplier × SMA20` | `notification/notifications.go:82` |

Сравнения RSI с порогами **строгие** (`<` и `>`), не `≤`/`≥`. Если RSI ровно равен порогу — соответствующий emoji не показывается.

## Шаблон Trade-сообщения

Реальный HTML-шаблон собирается через `strings.Builder` в `notification/notifications.go:12-55`. Логически:

```
[МЕДАЛЬ]

[ЛЕГЕНДА — блок legendBlock с расшифровкой всех emoji]

<u><b>Сигналы на покупку:</b></u>

<code>• <b>Акция:</b> ИМЯ [tier] [trend] [divergence] [volume]
  <b>RSI Value:</b>NN  (p5=N.N, p15=N.N)
  <b>Stop:</b> ЦЕНА (−N.N%)

• <b>Акция:</b> ИМЯ2 ...
</code>

<u><b>Сигналы на продажу:</b></u>

<code>• <b>Акция:</b> ИМЯ [sell-tier]
  <b>RSI Value:</b>NN  (p80=N.N, p90=N.N, p95=N.N)

• <b>Акция:</b> ИМЯ2 ...
</code>
```

### Пример заполненного сообщения (Dividend)

```
🥇

Легенда:
🟢 сильно перепродан
🟡 перепродан
🟠 перекуплен
🔴 сильно перекуплен
🚨 экстремум сверху
✅ тренд за нас
🚫 тренд против
📈 бычья дивергенция
🔊 подтверждение объёмом

Сигналы на покупку:

• Акция: Лукойл 🟢 ✅ 📈 🔊
  RSI Value:18  (p5=22.4, p15=29.1)
  Stop: 6482.50 (−6.2%)

• Акция: Татнефть - прив 🟡 🚫
  RSI Value:27  (p5=24.0, p15=31.0)
  Stop: 642.30 (−4.8%)

Сигналы на продажу:

• Акция: ФосАгро 🚨
  RSI Value:88  (p80=70.5, p90=78.2, p95=82.9)

• Акция: ММК 🟠
  RSI Value:74  (p80=70.5, p90=78.2, p95=82.9)
```

### Пример заполненного сообщения (Growth)

```
🥈

Легенда:
...

Сигналы на покупку:

• Акция: Яндекс 🟢 ✅ 📈
  RSI Value:19  (p5=23.0, p15=30.5)
  Stop: 3240.10 (−5.7%)

• Акция: Газпром 🟡 🚫
  RSI Value:28  (p5=24.0, p15=31.0)
  Stop: 122.80 (−4.5%)

Сигналы на продажу:

• Акция: Полюс 🔴
  RSI Value:91  (p80=72.0, p90=80.0, p95=86.0)
```

В обеих стратегиях в buy-секции виден маркер тренда (✅/🚫), т.к. `UseTrendFilter: true` для обоих типов.

## Порядок элементов в строке

```
• <b>Акция:</b> ИМЯ [tier emoji][trend mark][divergence][volume]
```

Конкретно (`notification/notifications.go:36`):

```go
log.InstrumentName
  + tierEmoji(log.RSIValue, thresholds[id])    // 🟢 / 🟡 / пусто
  + trendMark                                   // " ✅" / " 🚫" / пусто
  + divergenceBadge(divergences[id])            // " 📈" / пусто
  + volumeBadge(volumesConfirmed[id])           // " 🔊" / пусто
```

Все «значки» опциональны: если соответствующее условие не выполнено, ничего не печатается. То есть «голая» строка `• Акция: ХХХ` без emoji означает «нет ни сигнала, ни подтверждений» (в Trade-сообщение такая строка обычно не попадает — секции формируются только когда есть что показать).

## Формат полей

| Поле | Формат | Пример |
|---|---|---|
| Имя акции | как есть из `domain.Item.InstrumentName` | `Лукойл` |
| RSI Value | **целое** (truncation через `int(...)`) | `27` |
| `p5` / `p15` | `%.1f` (один знак после запятой) | `(p5=22.4, p15=29.1)` |
| `p80` / `p90` / `p95` | `%.1f` | `(p80=70.5, p90=78.2, p95=82.9)` |
| Stop price | `%.2f` | `6482.50` |
| Stop distance | `%.1f%%` со знаком минус | `−6.2%` |

Значения, у которых вся структура нулевая (`Thresholds{}`, `SellThresholds{}`, `Stop{}`), вообще не печатаются — это «тихое» поведение для случаев нехватки истории.

## Доставка

- Метод: Telegram Bot API.
- Режим: `ParseMode=HTML` — теги `<u>`, `<b>`, `<code>` рендерятся клиентом.
- Получатели: все `chatIds` из конфига (`pkg/client/telegram/telegram_bot.go:22-26`) получают идентичное сообщение.
- Где инициализируется бот: `internal/app/app.go` (см. вызов `InitTelegramBot`).

В live прод одновременно крутятся две стратегии (Dividend + Growth), и каждая шлёт свои сообщения с собственной медалью (🥇 / 🥈).
