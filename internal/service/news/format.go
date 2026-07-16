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
	// lineLimit — бюджет ужатого пункта: с запасом под заголовок дайджеста и
	// перевод строки, чтобы пункт гарантированно влез даже в первое сообщение.
	lineLimit = messageLimit - len(digestHeader) - 1
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
		if len(line) > lineLimit {
			line = shortenLine(it, lineLimit)
		}
		if len(cur)+1+len(line) > messageLimit {
			msgs = append(msgs, cur)
			cur = line

			continue
		}
		cur += "\n" + line
	}

	return append(msgs, cur)
}

// shortenLine ужимает слишком длинный пункт: лимит держится безусловно.
// Обрезаем заголовок по рунам (с «…» на месте обрезки), ссылку не рвём;
// строку перерисовываем после каждого шага, потому что эскейпинг нелинеен по
// длине (путь редкий, цикл не страшен). Если якорь не влезает даже с пустым
// заголовком (аномально длинная ссылка) — отдаём пункт без ссылки: простой
// текст в лимите лучше порванного тега или отклонённого сообщения.
func shortenLine(it rss.Item, limit int) string {
	runes := []rune(it.Title)
	for len(runes) > 0 {
		runes = runes[:len(runes)-1]
		line := fmt.Sprintf(
			"• <a href=\"%s\">%s…</a>",
			html.EscapeString(it.Link),
			html.EscapeString(string(runes)),
		)
		if len(line) <= limit {
			return line
		}
	}

	// Патологический fallback: сама ссылка длиннее лимита.
	runes = []rune(it.Title)
	for {
		line := "• " + html.EscapeString(string(runes))
		if len(line) <= limit || len(runes) == 0 {
			return line
		}
		runes = runes[:len(runes)-1]
	}
}
