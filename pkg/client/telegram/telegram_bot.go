package telegram

import (
	"fmt"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type Client interface {
	SendMessage(msg string) error
}

type telegramBotClientClient struct {
	clientApi *tgbotapi.BotAPI
	chatId    int64
}

func (b *telegramBotClientClient) SendMessage(msg string) error {
	ms := tgbotapi.NewMessage(b.chatId, msg)
	ms.ParseMode = "HTML"
	_, err := b.clientApi.Send(ms)

	if err != nil {
		return fmt.Errorf("failed to send message: %w", err)
	}

	return nil
}

func InitTelegramBot(token string, chatId int64) (Client, error) {
	bot, err := tgbotapi.NewBotAPI(token)

	if err != nil {
		return nil, err
	}

	bot.Debug = true

	return &telegramBotClientClient{clientApi: bot, chatId: chatId}, nil
}
