package config

type TelegramClient struct {
	Token  string `config:"TELEGRAM"`
	ChatID int64
}

func NewTelegramClientConfig() *TelegramClient {
	return &TelegramClient{
		ChatID: 397653673,
	}
}
