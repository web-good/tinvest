package config

type TelegramClient struct {
	Token            string `config:"TELEGRAM"`
	GroupChatID      int64  `config:"TELEGRAM_GROUP_CHAT_ID"`
	TopicGoldenX     int    `config:"TELEGRAM_TOPIC_GOLDEN_X"`
	TopicReversion   int    `config:"TELEGRAM_TOPIC_REVERSION"`
	TopicRSIPullback int    `config:"TELEGRAM_TOPIC_RSI_PULLBACK"`
	TopicNews        int    `config:"TELEGRAM_TOPIC_NEWS"`
	AllowedUserIDs   []int64
}

func NewTelegramClientConfig() *TelegramClient {
	return &TelegramClient{
		AllowedUserIDs: []int64{397653673, 784012062},
	}
}
