package telegram

import (
	"fmt"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"net/http"
)

type Client interface {
	SendMessage(msg string) error
	SendMessageToChat(chatID int64, msg string) error
}

type telegramBotClientClient struct {
	clientApi *tgbotapi.BotAPI
	chatIds   []int64
}

func (b *telegramBotClientClient) SendMessage(msg string) error {
	var errMsg string

	for _, chatID := range b.chatIds {
		if err := b.SendMessageToChat(chatID, msg); err != nil {
			errMsg += fmt.Sprintf("Failed to send to chat %d: %v\n", chatID, err)
		}
	}
	if errMsg != "" {
		return fmt.Errorf(errMsg)
	}
	return nil
}

func (b *telegramBotClientClient) SendMessageToChat(chatID int64, msg string) error {
	ms := tgbotapi.NewMessage(chatID, msg)
	ms.ParseMode = "HTML"
	_, err := b.clientApi.Send(ms)
	return err
}

func InitTelegramBot(token string, chatId []int64) (Client, error) {
	bot, err := tgbotapi.NewBotAPI(token)

	if err != nil {
		return nil, err
	}

	bot.Debug = true

	return &telegramBotClientClient{clientApi: bot, chatIds: chatId}, nil
}

func InitTelegramBotProxy(token string, chatId []int64, proxyURL string) (Client, error) {
	bot, err := tgbotapi.NewBotAPIWithClient(token, "https://v0-telegram-proxy-api.vercel.app/bot%s/%s", &http.Client{})

	if err != nil {
		return nil, err
	}

	bot.Debug = true

	return &telegramBotClientClient{clientApi: bot, chatIds: chatId}, nil
}
