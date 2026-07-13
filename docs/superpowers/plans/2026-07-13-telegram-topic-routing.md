# Telegram Topic Routing Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Удалить мёртвые стратегии, развести уведомления по темам форум-супергруппы Telegram и перевести портфельные расчёты на команды боту.

**Architecture:** Спека: `docs/superpowers/specs/2026-07-13-telegram-topic-routing-design.md`. Библиотека `go-telegram-bot-api/v5` заменяется на `github.com/go-telegram/bot` внутри `pkg/client/telegram`; интерфейс `telegram.Client` сохраняется, стратегии получают topic-bound экземпляры через DI. Новый компонент `internal/service/telegram_commands` слушает long-polling и диспатчит `/bonds_portfolio`, `/yield`, `/bonds_screener` с whitelist по user ID.

**Tech Stack:** Go 1.25, `github.com/go-telegram/bot` (+`/models`), mockery v2, mage.

## Global Constraints

- Ветка: `feat/telegram-topic-notifications` (уже создана, спека закоммичена).
- `go build ./...` падает на `magefiles` — проверять сборку так: `go build ./internal/... ./pkg/... ./cmd/...`.
- Финальный гейт: `./bin/mage ci` (lint + `go test -race ./...` + mock-drift). После изменения мокаемого интерфейса: `./bin/mage mocks`.
- **НЕ удалять** `internal/service/trading_strategy/scalping/model/` и `internal/service/trading_strategy/scalping/strategy/**` — их импортируют reversion core/live, levels, momentum и бэктест (`internal/domain/backtest`, `internal/service/backtest`).
- Не трогать логику торговли golden_x и reversion — меняется только адресат сообщений.
- Сообщения — ParseMode HTML (как сейчас).
- Коммиты завершать `Co-Authored-By: Claude Fable 5 <noreply@anthropic.com>`.

---

### Task 1: Удалить мёртвые стратегии macd_rsi, super_trend, scalping_rsi, ema200

**Files:**
- Delete: `internal/service/trading_strategy/macd_rsi/` (вся директория)
- Delete: `internal/service/trading_strategy/super_trend/` (вся директория)
- Delete: `internal/service/trading_strategy/scalping_rsi/` (вся директория)
- Delete: `internal/service/trading_strategy/ema200/` (вся директория)
- Modify: `internal/service_provider/service.go`
- Modify: `internal/app/app.go` (только закомментированные блоки)
- Modify: `CLAUDE.md`

**Interfaces:**
- Consumes: —
- Produces: репозиторий без четырёх пакетов; `service_provider.service` без полей `scalpingRsiTradingService`, `superTrendTradingService`, `ema200`.

- [ ] **Step 1: Убедиться, что внешних импортёров нет**

Run:
```bash
grep -rln "trading_strategy/macd_rsi\|trading_strategy/super_trend\|trading_strategy/scalping_rsi\|trading_strategy/ema200" --include="*.go" internal cmd pkg | grep -v "trading_strategy/macd_rsi/\|trading_strategy/super_trend/\|trading_strategy/scalping_rsi/\|trading_strategy/ema200/"
```
Expected: единственная строка `internal/service_provider/service.go`. Если появились другие — СТОП, доложить.

- [ ] **Step 2: Удалить директории**

```bash
git rm -r internal/service/trading_strategy/macd_rsi internal/service/trading_strategy/super_trend internal/service/trading_strategy/scalping_rsi internal/service/trading_strategy/ema200
```

- [ ] **Step 3: Зачистить service_provider/service.go**

Удалить импорты `ema200`, `scalping_rsi`, `super_trend`; поля структуры `service`: `scalpingRsiTradingService`, `superTrendTradingService`, `ema200`; целиком геттеры `GetScalpingRsiTradingService` (строки ~48-62), `GetSuperTrendTradingService` (~92-108), `Get200EmaService` (~110-124).

- [ ] **Step 4: Зачистить закомментированные блоки app.go**

В `internal/app/app.go` удалить закомментированные блоки, ссылающиеся на удалённые сервисы: в `runDev` блоки с `GetSuperTrendTradingService` (~88-105), `Get200EmaService` (~107-117), `GetMacdRsiTradingService` (~118-138), `GetScalpingRsiTradingService` (~140-149); в `runProd` два блока `GetMacdRsiTradingService` (~232-250). Импорты не менять (эти блоки закомментированы, импортов на них нет).

- [ ] **Step 5: Обновить CLAUDE.md**

В Overview заменить перечень стратегий: `It implements several trading strategies (MACD-RSI, SuperTrend, EMA200, Golden X, Bonds, Scalping RSI)` → `It implements trading strategies (Golden X, Reversion, Bonds screening, plus backtest-only Levels/Momentum)`. В Layout убрать `macd_rsi`, `super_trend`, `ema200`, `scalping_rsi` из списка `trading_strategy/`.

- [ ] **Step 6: Сборка и тесты**

Run: `go build ./internal/... ./pkg/... ./cmd/... && go test ./internal/... ./pkg/...`
Expected: OK, без упоминаний удалённых пакетов.

- [ ] **Step 7: Commit**

```bash
git add -A && git commit -m "chore: remove dead strategies macd_rsi, super_trend, scalping_rsi, ema200"
```

---

### Task 2: Удалить live-слой scalping и его конфиг

**Files:**
- Delete: `internal/service/trading_strategy/scalping/trade.go`, `trade_test.go`, `types.go`, `registry.go`
- Delete: `internal/service/trading_strategy/scalping/dto/`, `scheduler/`, `notification/` (директории)
- Keep (НЕ трогать): `internal/service/trading_strategy/scalping/model/`, `internal/service/trading_strategy/scalping/strategy/**`
- Modify: `internal/service_provider/service.go`, `internal/app/app.go`, `internal/app/init_config.go`
- Delete: `internal/config/scalping.go`
- Modify: `internal/config/config.go`, `env/local.env.example`, `CLAUDE.md`

**Interfaces:**
- Consumes: результат Task 1.
- Produces: пакет `scalping` содержит только `model/` и `strategy/`; `Config` без поля `Scalping`.

- [ ] **Step 1: Проверить импортёров корня scalping**

Run:
```bash
grep -rln "trading_strategy/scalping\"" --include="*.go" internal cmd | grep -v "/scalping/"
grep -rln "scalping/dto\|scalping/scheduler\|scalping/notification" --include="*.go" internal cmd | grep -v "trading_strategy/scalping/"
```
Expected: первая — только `internal/service_provider/service.go`; вторая — только `internal/app/app.go`. Иначе СТОП.

- [ ] **Step 2: Удалить live-слой**

```bash
git rm internal/service/trading_strategy/scalping/trade.go internal/service/trading_strategy/scalping/trade_test.go internal/service/trading_strategy/scalping/types.go internal/service/trading_strategy/scalping/registry.go
git rm -r internal/service/trading_strategy/scalping/dto internal/service/trading_strategy/scalping/scheduler internal/service/trading_strategy/scalping/notification
```

- [ ] **Step 3: Зачистить service_provider/service.go**

Удалить импорт `"tinvest/internal/service/trading_strategy/scalping"`, поле `scalpingTradingService scalping.Scalping`, геттер `GetScalpingTradingService` (~211-225).

- [ ] **Step 4: Зачистить app.go**

В `runDev` удалить оба воркера Scalping (строки ~205-225: `GetScalpingTradingService().Trade(...)` с `wg.Add(1)` каждый); удалить импорт `scalpingdto "tinvest/internal/service/trading_strategy/scalping/dto"`. В `runProd` удалить закомментированный блок `scalpingscheduler` (~314-325).

- [ ] **Step 5: Удалить конфиг scalping**

`git rm internal/config/scalping.go`. В `internal/config/config.go` удалить поле `Scalping *ScalpingConfig`. В `internal/app/init_config.go` удалить строку `Scalping: config.NewScalpingConfig(),`. В `env/local.env.example` удалить блок `# Dedicated brokerage account ID used by the scalping strategy` + `SCALPING_ACCOUNT_ID=`.

- [ ] **Step 6: Проверить mockery-конфиг**

В `.mockery.yaml` запись `tinvest/internal/service/trading_strategy/scalping/strategy: Strategy` относится к остающемуся пакету — НЕ удалять. Записей на удалённые файлы нет — изменений не требуется.

- [ ] **Step 7: Сборка и тесты**

Run: `go build ./internal/... ./pkg/... ./cmd/... && go test ./internal/... ./pkg/...`
Expected: OK. Бэктест-пакеты (`internal/service/backtest`, reversion, levels, momentum) собираются — они зависят только от `scalping/model` и `scalping/strategy`.

- [ ] **Step 8: Обновить CLAUDE.md**

В Layout: у `trading_strategy/` убрать `scalping` из списка живых стратегий, упомянуть: `scalping/model`, `scalping/strategy` — shared core для reversion/levels/momentum/backtest (live-слой удалён).

- [ ] **Step 9: Commit**

```bash
git add -A && git commit -m "chore: remove scalping live layer, keep shared model/strategy core"
```

---

### Task 3: Переписать pkg/client/telegram на github.com/go-telegram/bot

**Files:**
- Rewrite: `pkg/client/telegram/telegram_bot.go`
- Create: `pkg/client/telegram/topic_sender.go`
- Create: `pkg/client/telegram/topic_sender_test.go`
- Modify: `internal/service_provider/client.go`
- Regenerate: `pkg/client/telegram/mocks/mock_Client.go`
- Modify: `go.mod`/`go.sum` (`go get` + `go mod tidy`)

**Interfaces:**
- Consumes: —
- Produces:
  ```go
  type Client interface {
      SendMessage(msg string) error                                  // в дефолтный destination экземпляра
      SendMessageToChat(chatID int64, msg string) error              // в чат без темы
      SendMessageToTopic(chatID int64, threadID int, msg string) error
  }
  type Bot struct{ ... }                       // реализует Client; дефолт = (defaultChatID, General)
  func InitTelegramBot(token string, defaultChatID int64) (*Bot, error)
  func (b *Bot) API() *tgbot.Bot               // сырой доступ для listener'а команд (Task 6)
  func NewTopicSender(base Client, chatID int64, threadID int) Client
  ```
  ВАЖНО: `SendMessage` больше НЕ рассылает по списку чатов — шлёт в один дефолтный destination. `InitTelegramBotProxy` удаляется (нигде не вызывается: единственный референс — неиспользуемый `GetTelegramBotClientWithProxy`).

- [ ] **Step 1: Добавить зависимость**

```bash
go get github.com/go-telegram/bot@latest
```

- [ ] **Step 2: Переписать telegram_bot.go**

Полное новое содержимое `pkg/client/telegram/telegram_bot.go`:

```go
package telegram

import (
	"context"
	"fmt"
	"strconv"
	"time"

	tgbot "github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

// Client отправляет сообщения в Telegram. Экземпляр может быть привязан к
// destination (чат + тема форума): SendMessage шлёт именно туда.
type Client interface {
	SendMessage(msg string) error
	SendMessageToChat(chatID int64, msg string) error
	SendMessageToTopic(chatID int64, threadID int, msg string) error
}

const sendTimeout = 30 * time.Second

// Bot — клиент поверх go-telegram/bot. Реализует Client с destination по
// умолчанию (defaultChatID, General).
type Bot struct {
	api           *tgbot.Bot
	defaultChatID int64
}

// API отдаёт сырой bot для регистрации хендлеров команд и long-polling.
func (b *Bot) API() *tgbot.Bot { return b.api }

func (b *Bot) SendMessage(msg string) error {
	return b.SendMessageToTopic(b.defaultChatID, 0, msg)
}

func (b *Bot) SendMessageToChat(chatID int64, msg string) error {
	return b.SendMessageToTopic(chatID, 0, msg)
}

func (b *Bot) SendMessageToTopic(chatID int64, threadID int, msg string) error {
	ctx, cancel := context.WithTimeout(context.Background(), sendTimeout)
	defer cancel()

	_, err := b.api.SendMessage(ctx, &tgbot.SendMessageParams{
		ChatID:          chatID,
		MessageThreadID: threadID,
		Text:            msg,
		ParseMode:       models.ParseModeHTML,
	})
	if err == nil || threadID == 0 {
		return err
	}
	// Тема недоступна — фолбэк в General: сигнал не должен пропасть.
	_, ferr := b.api.SendMessage(ctx, &tgbot.SendMessageParams{
		ChatID:    chatID,
		Text:      "⚠️ тема " + strconv.Itoa(threadID) + " недоступна\n" + msg,
		ParseMode: models.ParseModeHTML,
	})
	if ferr != nil {
		return fmt.Errorf("send to topic %d failed: %w (fallback to General failed: %v)", threadID, err, ferr)
	}
	return nil
}

func InitTelegramBot(token string, defaultChatID int64) (*Bot, error) {
	api, err := tgbot.New(token)
	if err != nil {
		return nil, err
	}

	return &Bot{api: api, defaultChatID: defaultChatID}, nil
}
```

Примечание: если `tgbot.SendMessageParams.MessageThreadID` в текущей версии библиотеки имеет другой тип (проверить в `go doc github.com/go-telegram/bot.SendMessageParams`), привести threadID к нему; интерфейс `Client` оставить с `int`.

- [ ] **Step 3: Создать topic_sender.go**

```go
package telegram

// topicSender — Client, привязанный к конкретному чату и теме форума.
type topicSender struct {
	base     Client
	chatID   int64
	threadID int
}

// NewTopicSender возвращает Client, у которого SendMessage шлёт в
// (chatID, threadID). Остальные методы делегируются base.
func NewTopicSender(base Client, chatID int64, threadID int) Client {
	return &topicSender{base: base, chatID: chatID, threadID: threadID}
}

func (t *topicSender) SendMessage(msg string) error {
	return t.base.SendMessageToTopic(t.chatID, t.threadID, msg)
}

func (t *topicSender) SendMessageToChat(chatID int64, msg string) error {
	return t.base.SendMessageToChat(chatID, msg)
}

func (t *topicSender) SendMessageToTopic(chatID int64, threadID int, msg string) error {
	return t.base.SendMessageToTopic(chatID, threadID, msg)
}
```

- [ ] **Step 4: Перегенерировать мок Client**

Run: `./bin/mage mocks`
Expected: `pkg/client/telegram/mocks/mock_Client.go` получил метод `SendMessageToTopic`.

- [ ] **Step 5: Написать тест topic_sender**

`pkg/client/telegram/topic_sender_test.go`:

```go
package telegram_test

import (
	"testing"

	"tinvest/pkg/client/telegram"
	"tinvest/pkg/client/telegram/mocks"
)

func TestTopicSenderBindsSendMessageToTopic(t *testing.T) {
	m := mocks.NewMockClient(t)
	m.EXPECT().SendMessageToTopic(int64(-1001234), 42, "hello").Return(nil)

	s := telegram.NewTopicSender(m, -1001234, 42)
	if err := s.SendMessage("hello"); err != nil {
		t.Fatalf("SendMessage returned error: %v", err)
	}
}

func TestTopicSenderDelegatesExplicitDestinations(t *testing.T) {
	m := mocks.NewMockClient(t)
	m.EXPECT().SendMessageToChat(int64(7), "a").Return(nil)
	m.EXPECT().SendMessageToTopic(int64(8), 9, "b").Return(nil)

	s := telegram.NewTopicSender(m, -1001234, 42)
	if err := s.SendMessageToChat(7, "a"); err != nil {
		t.Fatalf("SendMessageToChat: %v", err)
	}
	if err := s.SendMessageToTopic(8, 9, "b"); err != nil {
		t.Fatalf("SendMessageToTopic: %v", err)
	}
}
```

- [ ] **Step 6: Запустить тест**

Run: `go test ./pkg/client/telegram/...`
Expected: PASS.

- [ ] **Step 7: Обновить service_provider/client.go**

Поле `telegramBot telegram.Client` в структуре `client` заменить на `telegramBot *telegram.Bot`. Геттеры:

```go
func (s *ServiceProvider) GetTelegramBot() (*telegram.Bot, error) {
	if serviceProvider.client.telegramBot != nil {
		return serviceProvider.client.telegramBot, nil
	}

	var err error
	serviceProvider.client.telegramBot, err = telegram.InitTelegramBot(
		s.appConfig.TelegramClient.Token,
		s.appConfig.TelegramClient.ChatID[0],
	)
	if err != nil {
		return nil, fmt.Errorf("could not init telegram bot: %w", err)
	}

	return serviceProvider.client.telegramBot, nil
}

func (s *ServiceProvider) GetTelegramBotClient() (telegram.Client, error) {
	return s.GetTelegramBot()
}
```

Удалить целиком `GetTelegramBotClientWithProxy` (референсов на него нет). `ChatID[0]` — временно, Task 4 заменит на `GroupChatID`.

- [ ] **Step 8: Убрать старую библиотеку**

```bash
go mod tidy
grep go-telegram-bot-api go.mod
```
Expected: grep пустой (старая зависимость ушла).

- [ ] **Step 9: Сборка и все тесты**

Run: `go build ./internal/... ./pkg/... ./cmd/... && go test ./internal/... ./pkg/...`
Expected: PASS (тесты reversion live и yield используют перегенерированный мок — компилируются без правок, т.к. старые методы интерфейса сохранены).

- [ ] **Step 10: Commit**

```bash
git add -A && git commit -m "feat(telegram): migrate to go-telegram/bot, add topic-aware Client"
```

---

### Task 4: Конфиг тем + DI topic-sender'ы + снятие портфельных воркеров с расписания

**Files:**
- Rewrite: `internal/config/telegram_client.go`
- Delete: `internal/config/profile.go`
- Modify: `internal/app/init_config.go`, `internal/service_provider/client.go`, `internal/service_provider/service.go`, `internal/app/app.go`, `env/local.env.example`
- Modify: `internal/service/trading_strategy/golden_x/scheduler/trade.go`
- Delete: `internal/service/portfolio/analyze/scheduler/`, `internal/service/portfolio/yield/scheduler/`, `internal/service/trading_strategy/bonds/scheduler/`

**Interfaces:**
- Consumes: `telegram.NewTopicSender`, `GetTelegramBot` из Task 3.
- Produces:
  ```go
  // internal/config
  type TelegramClient struct {
      Token          string `config:"TELEGRAM"`
      GroupChatID    int64  `config:"TELEGRAM_GROUP_CHAT_ID"`
      TopicGoldenX   int    `config:"TELEGRAM_TOPIC_GOLDEN_X"`
      TopicReversion int    `config:"TELEGRAM_TOPIC_REVERSION"`
      AllowedUserIDs []int64
  }
  // service_provider
  func (s *ServiceProvider) GetGoldenXSender() (telegram.Client, error)
  func (s *ServiceProvider) GetReversionSender() (telegram.Client, error)
  ```
  golden_x-scheduler теряет неиспользуемый tg-параметр: `NewSchedulerService(service golden_x.GoldenX) golden_x.GoldenX`.

- [ ] **Step 1: Переписать конфиг**

`internal/config/telegram_client.go` целиком:

```go
package config

type TelegramClient struct {
	Token          string `config:"TELEGRAM"`
	GroupChatID    int64  `config:"TELEGRAM_GROUP_CHAT_ID"`
	TopicGoldenX   int    `config:"TELEGRAM_TOPIC_GOLDEN_X"`
	TopicReversion int    `config:"TELEGRAM_TOPIC_REVERSION"`
	AllowedUserIDs []int64
}

func NewTelegramClientConfig() *TelegramClient {
	return &TelegramClient{
		AllowedUserIDs: []int64{397653673, 784012062},
	}
}
```

`git rm internal/config/profile.go`. В `internal/app/init_config.go` удалить блок парсинга PROFILES (строки ~58-62: `prJSON := os.Getenv("PROFILES")` … `}`) и, если после этого не используется, импорт `encoding/json`.

- [ ] **Step 2: env/local.env.example**

Удалить строку `PROFILES='...'`. Добавить:

```
# Telegram forum supergroup (ID вида -100XXXXXXXXXX) и ID тем (message_thread_id)
TELEGRAM_GROUP_CHAT_ID=
TELEGRAM_TOPIC_GOLDEN_X=
TELEGRAM_TOPIC_REVERSION=
```

- [ ] **Step 3: client.go — дефолт на GroupChatID**

В `GetTelegramBot` заменить `s.appConfig.TelegramClient.ChatID[0]` на `s.appConfig.TelegramClient.GroupChatID`.

- [ ] **Step 4: Topic-sender'ы в DI**

В `internal/service_provider/client.go` добавить:

```go
func (s *ServiceProvider) GetGoldenXSender() (telegram.Client, error) {
	return s.topicSender(s.appConfig.TelegramClient.TopicGoldenX, "golden_x")
}

func (s *ServiceProvider) GetReversionSender() (telegram.Client, error) {
	return s.topicSender(s.appConfig.TelegramClient.TopicReversion, "reversion")
}

// topicSender строит Client, привязанный к теме форума; при незаданном ID
// темы сообщения уходят в General (threadID 0), о чём предупреждаем в логе.
func (s *ServiceProvider) topicSender(threadID int, name string) (telegram.Client, error) {
	base, err := s.GetTelegramBot()
	if err != nil {
		return nil, err
	}
	if threadID == 0 {
		logger.Warn("telegram topic id is not set, sending to General", slog.String("topic", name))
	}

	return telegram.NewTopicSender(base, s.appConfig.TelegramClient.GroupChatID, threadID), nil
}
```

(импорты: `log/slog`, `tinvest/pkg/logger`).

- [ ] **Step 5: Привязать golden_x и reversion**

В `internal/service_provider/service.go`:
- `GetGoldenXTradingService`: `tgClient, _ := serviceProvider.GetTelegramBotClient()` → `tgClient, _ := serviceProvider.GetGoldenXSender()`.
- `GetReversionLiveService`: аналогично → `GetReversionSender()`.
- Остальные (`GetBondsTradingService`, `GetAnalyze`, `GetPortfolioYield`, `GetPurchaseSharesService`) пока не трогать — Task 5 меняет их сигнатуры.

- [ ] **Step 6: Убрать неиспользуемый tgClient из golden_x scheduler**

В `internal/service/trading_strategy/golden_x/scheduler/trade.go`: удалить поле `tgClient telegram.Client`, параметр конструктора и импорт `telegram`:

```go
func NewSchedulerService(service golden_x.GoldenX) golden_x.GoldenX {
	return &schedulerService{
		sh:      scheduler.NewScheduler(),
		service: service,
	}
}
```

- [ ] **Step 7: app.go — снять портфельные воркеры с расписания**

В `runDev`: удалить воркер `GetPortfolioYield().PortfolioYieldYTD(...)` (строки ~176-183) и закомментированный блок `GetAnalyze().BondsPortfolio` (~162-175).
В `runProd`: удалить три воркера — `bondsscheduler` (~251-257), `analyzescheduler` (~258-264), `yieldscheduler` (~265-271); в воркерах golden_x убрать получение `tgBot` и передавать только сервис: `scheduler.NewSchedulerService(a.sp.GetGoldenXTradingService()).Trade(...)`; `wg.Add(8)` → `wg.Add(4)` (2×golden_x + 2×reversion).
Удалить импорты `analyzescheduler`, `yieldscheduler`, `bondsscheduler`.

Затем удалить сами пакеты:
```bash
git rm -r internal/service/portfolio/analyze/scheduler internal/service/portfolio/yield/scheduler internal/service/trading_strategy/bonds/scheduler
```

- [ ] **Step 8: Сборка и тесты**

Run: `go build ./internal/... ./pkg/... ./cmd/... && go test ./internal/... ./pkg/...`
Expected: PASS.

- [ ] **Step 9: Commit**

```bash
git add -A && git commit -m "feat(telegram): topic config + per-strategy senders, unschedule portfolio workers"
```

---

### Task 5: Отчётные сервисы принимают destination-клиент

**Files:**
- Modify: `internal/service/portfolio/analyze/types.go`, `analyze.go`
- Modify: `internal/service/portfolio/yield/types.go`, `yield.go`, `yield_test.go`
- Modify: `internal/service/trading_strategy/bonds/types.go`, `trade.go`
- Modify: `internal/service_provider/service.go`

**Interfaces:**
- Consumes: `telegram.Client` (Task 3).
- Produces (на эти сигнатуры опирается Task 6):
  ```go
  type Analyze interface{ BondsPortfolio(ctx context.Context, tg telegram.Client) error }
  type Yield   interface{ PortfolioYieldYTD(ctx context.Context, tg telegram.Client) error }
  type Bonds   interface{ Trade(ctx context.Context, tg telegram.Client) error }
  ```
  Поле `tgClient` и параметр конструктора удаляются из всех трёх сервисов.

- [ ] **Step 1: Обновить тест yield (TDD — сначала тест)**

В `internal/service/portfolio/yield/yield_test.go`: вызов `svc.PortfolioYieldYTD(ctx, 12345)` → `svc.PortfolioYieldYTD(ctx, tgClient)`; ожидание мока `SendMessageToChat(mock.Anything, mock.Anything)` → `SendMessage(mock.Anything)`; из литерала `service{...}` убрать поле `tgClient` (в остальных тестах файла — аналогично).

- [ ] **Step 2: Прогнать тест — должен падать компиляцией**

Run: `go test ./internal/service/portfolio/yield/...`
Expected: FAIL (compile error) — сигнатура ещё старая.

- [ ] **Step 3: yield**

`types.go`: интерфейс `PortfolioYieldYTD(ctx context.Context, tg telegram.Client) error`; из `service` удалить поле `tgClient`, из `NewService` — параметр `tgClient telegram.Client`.
`yield.go`: сигнатура метода `(s *service) PortfolioYieldYTD(ctx context.Context, tg telegram.Client) error`; блок отправки:

```go
msg := notification.Send(y)
if tg != nil {
	if err := tg.SendMessage(msg); err != nil {
		return fmt.Errorf("failed to send telegram message: %w", err)
	}
}
```

- [ ] **Step 4: analyze**

`types.go`: `BondsPortfolio(ctx context.Context, tg telegram.Client) error`; удалить поле `tgClient` и параметр конструктора.
`analyze.go`: сигнатура `(s service) BondsPortfolio(ctx context.Context, tg telegram.Client) error`; оба места отправки `s.tgClient.SendMessageToChat(chatID, msg)` → `tg.SendMessage(msg)` (с сохранением nil-чека: `if tg != nil`).

- [ ] **Step 5: bonds**

`types.go`: интерфейс `Trade(ctx context.Context, tg telegram.Client) error`; удалить поле `tgClient` и параметр из `NewService`.
`trade.go`: сигнатура `(s *service) Trade(ctx context.Context, tg telegram.Client) error`; во всех четырёх вызовах `pipeline.Sender(..., s.tgClient, ...)` → `pipeline.Sender(..., tg, ...)`. `pipeline/sender.go` не меняется (уже принимает `telegram.Client` параметром).

- [ ] **Step 6: service_provider**

В `GetAnalyze`, `GetPortfolioYield`, `GetBondsTradingService` убрать `tgClient, _ := serviceProvider.GetTelegramBotClient()` и параметр `tgClient` из вызовов `NewService`.

- [ ] **Step 7: Сборка и тесты**

Run: `go build ./internal/... ./pkg/... ./cmd/... && go test ./internal/... ./pkg/...`
Expected: PASS, включая обновлённый yield_test.

- [ ] **Step 8: Commit**

```bash
git add -A && git commit -m "refactor(reports): analyze/yield/bonds take per-call telegram destination"
```

---

### Task 6: Команды бота — internal/service/telegram_commands

**Files:**
- Create: `internal/service/telegram_commands/commands.go`
- Create: `internal/service/telegram_commands/listener.go`
- Create: `internal/service/telegram_commands/commands_test.go`
- Modify: `internal/service_provider/service.go` (getter), `internal/app/app.go` (запуск listener в dev и prod)

**Interfaces:**
- Consumes: `analyze.Analyze`, `yield.Yield`, `bonds.Bonds` (сигнатуры Task 5); `*telegram.Bot.API()`, `telegram.NewTopicSender` (Task 3); `AllowedUserIDs` (Task 4).
- Produces:
  ```go
  type SenderFactory func(chatID int64, threadID int) telegram.Client
  func New(a analyze.Analyze, y yield.Yield, b bonds.Bonds, f SenderFactory, allowed []int64) *Commands
  func (c *Commands) Handle(ctx context.Context, text string, chatID int64, threadID int, userID int64) bool
  func NewListener(b *telegram.Bot, c *Commands) *Listener
  func (l *Listener) Run(ctx context.Context) // блокируется до отмены ctx
  ```

- [ ] **Step 1: Написать тесты роутера (TDD)**

`internal/service/telegram_commands/commands_test.go`:

```go
package telegram_commands

import (
	"context"
	"sync"
	"testing"
	"time"

	"tinvest/pkg/client/telegram"
)

type fakeSender struct {
	mu   sync.Mutex
	msgs []string
}

func (f *fakeSender) SendMessage(msg string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.msgs = append(f.msgs, msg)
	return nil
}
func (f *fakeSender) SendMessageToChat(int64, string) error       { return nil }
func (f *fakeSender) SendMessageToTopic(int64, int, string) error { return nil }

func (f *fakeSender) count() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.msgs)
}

type fakeYield struct {
	calls   chan telegram.Client
	release chan struct{}
}

func (f *fakeYield) PortfolioYieldYTD(_ context.Context, tg telegram.Client) error {
	f.calls <- tg
	if f.release != nil {
		<-f.release
	}
	return nil
}

type fakeAnalyze struct{ called bool }

func (f *fakeAnalyze) BondsPortfolio(context.Context, telegram.Client) error {
	f.called = true
	return nil
}

type fakeBonds struct{ called bool }

func (f *fakeBonds) Trade(context.Context, telegram.Client) error {
	f.called = true
	return nil
}

func newTestCommands(y *fakeYield, sender *fakeSender) *Commands {
	factory := func(chatID int64, threadID int) telegram.Client { return sender }
	return New(&fakeAnalyze{}, y, &fakeBonds{}, factory, []int64{111})
}

func TestHandleIgnoresUnknownUser(t *testing.T) {
	y := &fakeYield{calls: make(chan telegram.Client, 1)}
	sender := &fakeSender{}
	c := newTestCommands(y, sender)

	if c.Handle(context.Background(), "/yield", 1, 2, 999) {
		t.Fatal("чужой userID должен игнорироваться")
	}
	if sender.count() != 0 {
		t.Fatal("чужому пользователю не должно уходить сообщений")
	}
}

func TestHandleDispatchesYield(t *testing.T) {
	y := &fakeYield{calls: make(chan telegram.Client, 1)}
	sender := &fakeSender{}
	c := newTestCommands(y, sender)

	if !c.Handle(context.Background(), "/yield", 1, 2, 111) {
		t.Fatal("команда авторизованного пользователя должна обрабатываться")
	}
	select {
	case <-y.calls:
	case <-time.After(2 * time.Second):
		t.Fatal("PortfolioYieldYTD не вызван")
	}
	if sender.count() == 0 {
		t.Fatal("нет ack-сообщения «Считаю»")
	}
}

func TestHandleRejectsConcurrentDuplicate(t *testing.T) {
	y := &fakeYield{calls: make(chan telegram.Client, 1), release: make(chan struct{})}
	sender := &fakeSender{}
	c := newTestCommands(y, sender)

	c.Handle(context.Background(), "/yield", 1, 2, 111)
	<-y.calls // первый запуск внутри расчёта

	before := sender.count()
	c.Handle(context.Background(), "/yield", 1, 2, 111)
	if sender.count() != before+1 {
		t.Fatal("повторная команда должна получить ответ «уже выполняется»")
	}
	close(y.release)

	select {
	case y.calls <- nil: // канал свободен — второго вызова сервиса не было
	default:
	}
}

func TestHandleUnknownCommandIgnored(t *testing.T) {
	y := &fakeYield{calls: make(chan telegram.Client, 1)}
	sender := &fakeSender{}
	c := newTestCommands(y, sender)

	if c.Handle(context.Background(), "/unknown", 1, 2, 111) {
		t.Fatal("неизвестная команда должна игнорироваться")
	}
}

func TestHandleStripsBotMention(t *testing.T) {
	y := &fakeYield{calls: make(chan telegram.Client, 1)}
	sender := &fakeSender{}
	c := newTestCommands(y, sender)

	if !c.Handle(context.Background(), "/yield@MyTradingBot", 1, 2, 111) {
		t.Fatal("команда с @упоминанием бота должна обрабатываться")
	}
	select {
	case <-y.calls:
	case <-time.After(2 * time.Second):
		t.Fatal("PortfolioYieldYTD не вызван")
	}
}
```

- [ ] **Step 2: Прогнать — компиляция падает**

Run: `go test ./internal/service/telegram_commands/...`
Expected: FAIL — пакета ещё нет.

- [ ] **Step 3: Реализовать commands.go**

```go
package telegram_commands

import (
	"context"
	"strings"
	"sync"

	"tinvest/internal/service/portfolio/analyze"
	"tinvest/internal/service/portfolio/yield"
	"tinvest/internal/service/trading_strategy/bonds"
	"tinvest/pkg/client/telegram"
	"tinvest/pkg/logger"
)

// SenderFactory строит Client, привязанный к чату/теме, откуда пришла команда.
type SenderFactory func(chatID int64, threadID int) telegram.Client

type Commands struct {
	analyze        analyze.Analyze
	yield          yield.Yield
	bonds          bonds.Bonds
	newSender      SenderFactory
	allowedUserIDs map[int64]struct{}

	mu      sync.Mutex
	running map[string]bool
}

func New(a analyze.Analyze, y yield.Yield, b bonds.Bonds, f SenderFactory, allowed []int64) *Commands {
	ids := make(map[int64]struct{}, len(allowed))
	for _, id := range allowed {
		ids[id] = struct{}{}
	}

	return &Commands{
		analyze:        a,
		yield:          y,
		bonds:          b,
		newSender:      f,
		allowedUserIDs: ids,
		running:        make(map[string]bool),
	}
}

const helpText = `Доступные команды:
/bonds_portfolio — распределение облигаций в портфеле
/yield — доходность портфеля YTD (XIRR)
/bonds_screener — скринер облигаций к покупке
/help — этот список`

// Handle обрабатывает одно входящее сообщение. Возвращает false, если оно
// проигнорировано (не-whitelisted пользователь или неизвестная команда).
// Команды от чужих пользователей игнорируются молча.
func (c *Commands) Handle(ctx context.Context, text string, chatID int64, threadID int, userID int64) bool {
	if _, ok := c.allowedUserIDs[userID]; !ok {
		return false
	}
	cmd, _, _ := strings.Cut(strings.TrimSpace(text), " ")
	cmd, _, _ = strings.Cut(cmd, "@") // "/yield@MyBot" -> "/yield"
	tg := c.newSender(chatID, threadID)

	switch cmd {
	case "/help", "/start":
		_ = tg.SendMessage(helpText)
	case "/bonds_portfolio":
		c.runExclusive(ctx, cmd, tg, func(ctx context.Context) error {
			return c.analyze.BondsPortfolio(ctx, tg)
		})
	case "/yield":
		c.runExclusive(ctx, cmd, tg, func(ctx context.Context) error {
			return c.yield.PortfolioYieldYTD(ctx, tg)
		})
	case "/bonds_screener":
		c.runExclusive(ctx, cmd, tg, func(ctx context.Context) error {
			return c.bonds.Trade(ctx, tg)
		})
	default:
		return false
	}

	return true
}

// runExclusive подтверждает приём, выполняет расчёт в горутине и не
// допускает второй параллельный запуск той же команды.
func (c *Commands) runExclusive(ctx context.Context, cmd string, tg telegram.Client, fn func(context.Context) error) {
	c.mu.Lock()
	if c.running[cmd] {
		c.mu.Unlock()
		_ = tg.SendMessage("⏳ " + cmd + " уже выполняется")

		return
	}
	c.running[cmd] = true
	c.mu.Unlock()

	_ = tg.SendMessage("⏳ Считаю " + cmd + "…")

	go func() {
		defer func() {
			c.mu.Lock()
			c.running[cmd] = false
			c.mu.Unlock()
		}()
		if err := fn(ctx); err != nil {
			logger.ErrorContext(ctx, "telegram command failed", err.Error())
			_ = tg.SendMessage("❌ " + cmd + ": ошибка выполнения, подробности в логах")
		}
	}()
}
```

- [ ] **Step 4: Прогнать тесты роутера**

Run: `go test ./internal/service/telegram_commands/... -race`
Expected: PASS.

- [ ] **Step 5: Реализовать listener.go**

```go
package telegram_commands

import (
	"context"
	"strings"

	tgbot "github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"

	"tinvest/pkg/client/telegram"
)

// Listener подписывает роутер на входящие сообщения и держит long-polling.
type Listener struct {
	bot      *telegram.Bot
	commands *Commands
}

func NewListener(b *telegram.Bot, c *Commands) *Listener {
	return &Listener{bot: b, commands: c}
}

// Run блокируется до отмены ctx. Переподключение при сетевых ошибках
// long-polling библиотека go-telegram/bot выполняет сама.
func (l *Listener) Run(ctx context.Context) {
	l.bot.API().RegisterHandlerMatchFunc(func(update *models.Update) bool {
		return update.Message != nil && strings.HasPrefix(update.Message.Text, "/")
	}, l.handle)
	l.bot.API().Start(ctx)
}

func (l *Listener) handle(ctx context.Context, _ *tgbot.Bot, update *models.Update) {
	m := update.Message
	if m == nil || m.From == nil {
		return
	}
	l.commands.Handle(ctx, m.Text, m.Chat.ID, m.MessageThreadID, m.From.ID)
}
```

Примечание: точные имена (`RegisterHandlerMatchFunc`, `models.Update.Message.MessageThreadID`) сверить с установленной версией через `go doc github.com/go-telegram/bot Bot.RegisterHandlerMatchFunc` и `go doc github.com/go-telegram/bot/models Message` — при расхождении адаптировать, семантику сохранить.

- [ ] **Step 6: DI-getter**

В `internal/service_provider/service.go` добавить поле `telegramCommands *telegram_commands.Listener` в структуру `service` и геттер:

```go
func (s *ServiceProvider) GetTelegramCommands() (*telegram_commands.Listener, error) {
	if serviceProvider.service.telegramCommands != nil {
		return serviceProvider.service.telegramCommands, nil
	}
	bot, err := s.GetTelegramBot()
	if err != nil {
		return nil, err
	}
	factory := func(chatID int64, threadID int) telegram.Client {
		return telegram.NewTopicSender(bot, chatID, threadID)
	}
	cmds := telegram_commands.New(
		s.GetAnalyze(),
		s.GetPortfolioYield(),
		s.GetBondsTradingService(),
		factory,
		s.appConfig.TelegramClient.AllowedUserIDs,
	)
	serviceProvider.service.telegramCommands = telegram_commands.NewListener(bot, cmds)

	return serviceProvider.service.telegramCommands, nil
}
```

- [ ] **Step 7: Запуск в app.go**

Одинаковый блок в `runDev` и `runProd` (в prod поднять счётчик: `wg.Add(4)` → `wg.Add(5)`; в dev — `wg.Add(1)`):

```go
go func() {
	defer wg.Done()
	listener, err := a.sp.GetTelegramCommands()
	if err != nil {
		logger.ErrorContext(ctx, "telegram commands init failed", err.Error())
		return
	}
	logger.InfoContext(ctx, "telegram commands listener started")
	listener.Run(ctx)
}()
```

- [ ] **Step 8: Сборка и все тесты**

Run: `go build ./internal/... ./pkg/... ./cmd/... && go test ./internal/... ./pkg/... -race`
Expected: PASS.

- [ ] **Step 9: Commit**

```bash
git add -A && git commit -m "feat(telegram): command listener for portfolio reports and bonds screener"
```

---

### Task 7: Финальный гейт, документация, ручная настройка

**Files:**
- Modify: `CLAUDE.md`, `docs/superpowers/specs/2026-07-13-telegram-topic-routing-design.md` (синхронизация мелочей)
- Verify: `./bin/mage ci`

**Interfaces:**
- Consumes: всё предыдущее.
- Produces: зелёный CI-гейт, актуальные доки, инструкция по ручной настройке Telegram.

- [ ] **Step 1: Mock-drift и CI**

Run: `./bin/mage mocks && git diff --stat` — дрейфа быть не должно (моки регенерированы в Task 3). Затем `./bin/mage ci`.
Expected: lint + `go test -race ./...` + mock-check зелёные. Ошибки чинить до зелёного.

- [ ] **Step 2: Синхронизировать спеку с фактом**

В спеке поправить два пункта под реализацию: (а) конфиг тем — плоские поля `TopicGoldenX`/`TopicReversion` вместо `map[string]int64` (проще для confita); (б) `InitTelegramBotProxy` не переносится, а удалён — он нигде не вызывался.

- [ ] **Step 3: CLAUDE.md**

В Development Notes добавить одну строку: уведомления идут в форум-темы супергруппы (`TELEGRAM_GROUP_CHAT_ID`, `TELEGRAM_TOPIC_*`), портфельные отчёты — по командам боту (`/bonds_portfolio`, `/yield`, `/bonds_screener`), см. спеку `docs/superpowers/specs/2026-07-13-telegram-topic-routing-design.md`.

- [ ] **Step 4: Commit**

```bash
git add -A && git commit -m "docs(telegram): sync spec and CLAUDE.md with topic-routing implementation"
```

- [ ] **Step 5: Сообщить пользователю ручные шаги (в финальном отчёте, не в коде)**

1. Создать супергруппу, включить «Темы», добавить бота админом (право «Manage topics» не обязательно, «Post messages» есть у админа).
2. Создать темы «Golden X» и «Reversion».
3. В каждой теме отправить сообщение, скопировать его ссылку `https://t.me/c/<internal_id>/<thread_id>/<msg_id>`: `GroupChatID = -100<internal_id>`, `thread_id` — ID темы.
4. Заполнить в `env/local.env` / `env/prod.env`: `TELEGRAM_GROUP_CHAT_ID`, `TELEGRAM_TOPIC_GOLDEN_X`, `TELEGRAM_TOPIC_REVERSION`.
5. Проверка: запустить dev, в группе вызвать `/help`, затем `/yield` — ответ должен прийти в ту же тему.
