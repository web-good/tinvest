package telegram

import (
	"fmt"
	"net/http"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type Client interface {
	SendMessage(msg string) error
	SendMessageToChat(chatID int64, msg string) error
}

type telegramBotClientClient struct {
	clientAPI *tgbotapi.BotAPI
	chatIDs   []int64
}

func (b *telegramBotClientClient) SendMessage(msg string) error {
	var errMsg string

	for _, chatID := range b.chatIDs {
		if err := b.SendMessageToChat(chatID, msg); err != nil {
			errMsg += fmt.Sprintf("Failed to send to chat %d: %v\n", chatID, err)
		}
	}
	if errMsg != "" {
		return fmt.Errorf("%s", errMsg)
	}
	return nil
}

func (b *telegramBotClientClient) SendMessageToChat(chatID int64, msg string) error {
	ms := tgbotapi.NewMessage(chatID, msg)
	ms.ParseMode = "HTML"
	_, err := b.clientAPI.Send(ms)
	return err
}

func InitTelegramBot(token string, chatID []int64) (Client, error) {
	bot, err := tgbotapi.NewBotAPI(token)

	if err != nil {
		return nil, err
	}

	bot.Debug = true

	return &telegramBotClientClient{clientAPI: bot, chatIDs: chatID}, nil
}

func InitTelegramBotProxy(token string, chatID []int64, proxyURL string) (Client, error) {
	bot, err := tgbotapi.NewBotAPIWithClient(token, "https://v0-telegram-proxy-api.vercel.app/bot%s/%s", &http.Client{})

	if err != nil {
		return nil, err
	}

	bot.Debug = true

	return &telegramBotClientClient{clientAPI: bot, chatIDs: chatID}, nil
}
