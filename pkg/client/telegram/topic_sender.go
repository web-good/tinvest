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
