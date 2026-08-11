package telegram

import (
	"context"
	"fmt"
	"net/http"
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

const (
	sendTimeout = 30 * time.Second

	// pollTimeout — сколько Telegram держит открытым getUpdates;
	// pollHTTPTimeout — потолок ожидания у HTTP-клиента. Второй заведомо
	// больше первого: у библиотеки они по умолчанию равны (минута против
	// минуты), и любая сетевая задержка выше секунды роняет каждый long poll
	// в «Client.Timeout exceeded while awaiting headers».
	pollTimeout     = 30 * time.Second
	pollHTTPTimeout = 60 * time.Second
)

// pollingOptions собирает опции long-polling.
func pollingOptions(poll, httpTimeout time.Duration) []tgbot.Option {
	return []tgbot.Option{
		// No-op default handler: иначе библиотека дампит в лог каждый update,
		// не пойманный зарегистрированными хендлерами.
		tgbot.WithDefaultHandler(func(_ context.Context, _ *tgbot.Bot, _ *models.Update) {}),
		tgbot.WithHTTPClient(poll, &http.Client{Timeout: httpTimeout}),
	}
}

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
	// Свой таймаут: если первичная отправка «медленно упала» и съела бюджет
	// первого контекста, фолбэк всё равно должен успеть уйти.
	fctx, fcancel := context.WithTimeout(context.Background(), sendTimeout)
	defer fcancel()

	_, ferr := b.api.SendMessage(fctx, &tgbot.SendMessageParams{
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
	api, err := tgbot.New(token, pollingOptions(pollTimeout, pollHTTPTimeout)...)
	if err != nil {
		return nil, err
	}

	return &Bot{api: api, defaultChatID: defaultChatID}, nil
}
