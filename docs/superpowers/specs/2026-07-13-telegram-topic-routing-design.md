# Роутинг Telegram-уведомлений по темам форума + команды по запросу

Дата: 2026-07-13
Статус: дизайн утверждён, ждёт план реализации

## Проблема

Все уведомления (7+ источников: торговые стратегии, скринер облигаций, отчёты по
портфелю) летят одним потоком во все чаты из захардкоженного списка
`TelegramClient.ChatID` (`[397653673, 784012062]`). Лента нечитаемая, отключить
или найти отдельную тему нельзя. Портфельные отчёты приходят по расписанию, хотя
нужны только по запросу.

## Решение (утверждено)

Одна супергруппа Telegram с включёнными темами (форум-режим). Каждый источник
уведомлений пишет в свою тему. Портфельные расчёты снимаются с расписания и
запускаются командами боту; ответ приходит в ту тему, откуда вызвана команда.

Перед этим — фаза 0: удаление мёртвых стратегий (см. ниже), после которой
push-источников остаётся два.

### Распределение источников

| Источник | Режим | Куда |
|---|---|---|
| Golden X | push, расписание (как сейчас) | тема «Golden X» |
| Reversion live | push, расписание | тема «Reversion» |
| Анализ бонд-портфеля | pull, `/bonds_portfolio` | тема, откуда вызвали |
| Доходность YTD (XIRR) | pull, `/yield` | тема, откуда вызвали |
| Скринер облигаций (стратегия Bonds) | pull, `/bonds_screener` | тема, откуда вызвали |
| Системные сообщения, ошибки, фолбэк при недоставке | push | General |

Торговые стратегии продолжают торговать по расписанию без каких-либо изменений
своей логики — меняется только адресат сообщений. General — штатное место всех
системных/сервисных сообщений и ошибок (отдельная тема не нужна: General в
форум-группе существует всегда).

### Фаза 0: удаление мёртвых стратегий

Стратегии MACD-RSI, SuperTrend, Scalping, Scalping RSI и EMA200 не
используются — удаляются до миграции Telegram (меньше call-sites переносить).

Удаляется целиком:
- `internal/service/trading_strategy/macd_rsi/`
- `internal/service/trading_strategy/super_trend/`
- `internal/service/trading_strategy/scalping_rsi/`
- `internal/service/trading_strategy/ema200/` (не подключена к расписанию,
  только мёртвый геттер в DI; индикатор EMA живёт в
  `internal/service/instrument/ema` и не затрагивается)

`scalping` удаляется **частично** — только live-слой:
- удалить: корневые файлы пакета (`trade.go`, `trade_test.go`, `types.go`,
  `registry.go`), `dto/`, `scheduler/`, `notification/`
- **оставить**: `scalping/model/` и `scalping/strategy/**` — это разделяемое
  ядро, его импортируют reversion core/live, levels, momentum и весь
  бэктест-движок (`internal/domain/backtest`, `internal/service/backtest`)

Сопутствующая зачистка:
- `internal/service_provider/service.go` — поля и геттеры пяти сервисов
- `internal/app/app.go` — воркеры и импорты (`scalpingdto`, `scalpingscheduler`
  и др.), `internal/app/init_config.go` — `Scalping: config.NewScalpingConfig()`
- `internal/config/scalping.go` + поле `Scalping` в `config.go`
- моки удалённых интерфейсов, затем `./bin/mage mocks` и `./bin/mage ci`
- упоминания стратегий в `CLAUDE.md` (список стратегий)

### Библиотека

Текущая `go-telegram-bot-api/v5 v5.5.1` заморожена на Bot API ~5.x: нет
`MessageThreadID` ни на отправку, ни в парсинге входящих сообщений — ответить
«в ту же тему» на её структурах нельзя.

Решение: заменить библиотеку внутри `pkg/client/telegram` на
`github.com/go-telegram/bot` (активно поддерживается, zero-dependency,
актуальный Bot API: `SendMessageParams.MessageThreadID`, темы во входящих
апдейтах, встроенный роутер команд, long-polling). Замена заперта в одном
пакете: все потребители зависят от нашего интерфейса `telegram.Client`,
их код не меняется. Старая зависимость удаляется из `go.mod`.

`InitTelegramBotProxy` удаляется, а не переносится: на момент миграции он нигде
не вызывался (мёртвый код).

### Отправка (push): topic-sender per стратегия

Интерфейс `telegram.Client` сохраняется. Добавляется конструктор:

```go
// NewTopicSender возвращает Client, у которого SendMessage(msg)
// шлёт в groupChatID с message_thread_id = threadID.
func NewTopicSender(b *coreBot, groupChatID int64, threadID int64) Client
```

В `service_provider` каждая стратегия получает свой topic-bound экземпляр
(`GetGoldenXTelegram()`, … или общий `GetTopicSender(topic enum)`), поэтому
call-sites стратегий (`s.tgClient.SendMessage(...)`) не трогаются.
`SendMessageToChat(chatID, msg)` остаётся в интерфейсе для совместимости.

`threadID = 0` означает General (Telegram трактует отсутствие
`message_thread_id` как основную ленту группы).

### Команды (pull)

Новый компонент `internal/service/telegram_commands`:

- Запускается горутиной в `internal/app` (dev и prod одинаково), long-polling
  `getUpdates`.
- Роутер команд: `/bonds_portfolio`, `/yield`, `/bonds_screener`, `/help`.
- Хендлер получает из апдейта `chatID` + `messageThreadID` и передаёт их
  сервису — существующие сигнатуры `BondsPortfolio(ctx, chatID)` /
  `PortfolioYieldYTD(ctx, chatID)` расширяются до отправки через topic-aware
  sender (или принимают уже связанный `telegram.Client`).
- Долгие расчёты: хендлер сразу отвечает «⏳ Считаю…» и продолжает в горутине;
  на команду ставится mutex — повторный вызов до завершения получает
  «уже выполняется».
- Эти три воркера удаляются из расписания (`runDev`/`runProd`).

Авторизация: whitelist `AllowedUserIDs` в конфиге (два текущих user ID).
Команды от чужих `From.ID` молча игнорируются (без ответа, чтобы не
подтверждать существование бота). Команды принимаются и из супергруппы,
и из лички с ботом — ответ идёт туда, откуда пришла команда.

### Конфигурация

```go
type TelegramClient struct {
    Token          string `config:"TELEGRAM"`
    GroupChatID    int64  `config:"TELEGRAM_GROUP_CHAT_ID"` // супергруппа-форум
    TopicGoldenX   int    `config:"TELEGRAM_TOPIC_GOLDEN_X"`
    TopicReversion int    `config:"TELEGRAM_TOPIC_REVERSION"`
    AllowedUserIDs []int64 // whitelist для команд
}
```

- Плоские поля `TopicGoldenX`/`TopicReversion` вместо `map[string]int64` —
  проще для `confita` (нет парсинга произвольных ключей из ENV).
- ID тем и группы заполняются в `env/local.env` (confita); `AllowedUserIDs`
  дефолтится в `NewTelegramClientConfig()`.
- Незаданная (нулевая) тема → фолбэк в General + warning в лог (сообщение не
  должно пропасть).
- Поле `ChatID []int64` и `Profiles` удаляются после миграции всех
  потребителей (рассылка веером больше не нужна: второй получатель — участник
  группы).

Настройка на стороне Telegram (одноразовая, вручную):
1. Создать супергруппу, включить «Темы», добавить бота админом.
2. Создать темы по списку выше.
3. Узнать `message_thread_id` каждой темы: скопировать ссылку на любое её
   сообщение — `https://t.me/c/<internal_chat_id>/<thread_id>/<msg_id>`.
4. `GroupChatID` = `-100<internal_chat_id>`.

### Обработка ошибок

- Ошибка отправки в тему → лог + одна повторная попытка в General с префиксом
  темы (критичные сигналы reversion live не должны теряться).
- Ошибка выполнения команды → ответ в ту же тему: «❌ Ошибка: …» (текст ошибки
  без внутренних деталей) + полный лог.
- Падение long-polling горутины → рестарт с бэкоффом, сервисное сообщение в
  General.

### Тестирование

- Юнит-тесты роутера команд: whitelist (свой/чужой user ID), диспатч по
  командам, mutex «уже выполняется», неизвестная команда.
- Юнит-тесты topic-sender: проставление `message_thread_id`, фолбэк в General.
- Мок `telegram.Client` перегенерировать (`./bin/mage mocks`), прогнать
  `./bin/mage ci`.
- Ручная проверка в dev: сообщение каждой стратегии попадает в свою тему,
  команды отвечают в вызвавшую тему.

## Отвергнутые альтернативы

- **Отдельная группа на стратегию** — 6–8 чатов в списке, каждого получателя
  добавлять в каждую; форум-темы дают то же разделение в одном чате.
- **Остаться на go-telegram-bot-api v5 (вариант A)** — отправка в темы только
  через ручные params, входящий `message_thread_id` не парсится, «ответить в ту
  же тему» потребовало бы разбора сырого JSON.
- **Несколько ботов (бот на тему)** — умножает токены/инстансы без выгоды.
- **Убрать push полностью (всё по командам)** — сигналы сделок торговых
  стратегий критичны по времени, их нельзя переводить на pull.
