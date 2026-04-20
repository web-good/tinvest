package config

type TelegramClient struct {
	Token    string `config:"TELEGRAM"`
	ChatID   []int64
	Profiles Profiles
}

func NewTelegramClientConfig() *TelegramClient {
	return &TelegramClient{
		ChatID:   []int64{397653673, 784012062},
		Profiles: Profiles{},
	}
}
