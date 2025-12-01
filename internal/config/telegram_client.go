package config

type TelegramClient struct {
	Token  string `config:"TELEGRAM"`
	ChatID []int64
}

func NewTelegramClientConfig() *TelegramClient {
	return &TelegramClient{
		ChatID: []int64{397653673, 784012062},
	}
}
