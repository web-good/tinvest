package news

import (
	"fmt"
	"html"

	"tinvest/pkg/client/rss"
)

const (
	digestHeader = "📰 Новости рынка"
	// messageLimit — лимит Telegram на длину текста одного сообщения.
	messageLimit = 4096
)

// formatDigest собирает дайджест в одно или несколько сообщений (ParseMode
// HTML): пункты не рвутся посередине, заголовок — только в первом сообщении.
// Пустой вход → nil (постить нечего).
func formatDigest(items []rss.Item) []string {
	if len(items) == 0 {
		return nil
	}

	var msgs []string
	cur := digestHeader
	for _, it := range items {
		line := fmt.Sprintf(
			"• <a href=\"%s\">%s</a>",
			html.EscapeString(it.Link),
			html.EscapeString(it.Title),
		)
		if len(cur)+1+len(line) > messageLimit {
			msgs = append(msgs, cur)
			cur = line

			continue
		}
		cur += "\n" + line
	}

	return append(msgs, cur)
}
