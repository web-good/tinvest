package news

import (
	"fmt"
	"strings"
	"testing"

	"tinvest/pkg/client/rss"
)

func TestFormatDigest_Empty(t *testing.T) {
	if got := formatDigest(nil); got != nil {
		t.Fatalf("formatDigest(nil) = %v, want nil", got)
	}
}

func TestFormatDigest_SingleMessage(t *testing.T) {
	items := []rss.Item{
		{Title: "Новость раз", Link: "https://example.com/1"},
		{Title: "Новость два", Link: "https://example.com/2"},
	}
	msgs := formatDigest(items)
	if len(msgs) != 1 {
		t.Fatalf("len(msgs) = %d, want 1", len(msgs))
	}
	want := "📰 Новости рынка\n" +
		`• <a href="https://example.com/1">Новость раз</a>` + "\n" +
		`• <a href="https://example.com/2">Новость два</a>`
	if msgs[0] != want {
		t.Errorf("msg = %q\nwant %q", msgs[0], want)
	}
}

func TestFormatDigest_EscapesHTML(t *testing.T) {
	items := []rss.Item{{Title: `Отчёт <AFKS> & "прогноз"`, Link: "https://example.com/1"}}
	msg := formatDigest(items)[0]
	if strings.Contains(msg, "<AFKS>") {
		t.Errorf("заголовок не заэскейпен: %q", msg)
	}
	if !strings.Contains(msg, "&lt;AFKS&gt; &amp;") {
		t.Errorf("нет ожидаемых HTML-энтити: %q", msg)
	}
}

func TestFormatDigest_TruncatesOversizedTitle(t *testing.T) {
	// Один пункт, чей заголовок сам по себе больше 4096: заголовок должен быть
	// обрезан (с «…»), якорь остаться целым, сообщение — одно и в лимите.
	items := []rss.Item{{
		Title: strings.Repeat("х", 5000),
		Link:  "https://example.com/long",
	}}
	msgs := formatDigest(items)
	if len(msgs) != 1 {
		t.Fatalf("len(msgs) = %d, want 1", len(msgs))
	}
	msg := msgs[0]
	if len(msg) > messageLimit {
		t.Errorf("len(msg) = %d > лимита %d", len(msg), messageLimit)
	}
	if !strings.Contains(msg, "…") {
		t.Errorf("нет признака обрезки «…»")
	}
	lines := strings.Split(msg, "\n")
	item := lines[len(lines)-1]
	if !strings.HasPrefix(item, "• ") || !strings.HasSuffix(item, "</a>") {
		t.Errorf("порванный якорь: %.80q", item)
	}
}

func TestFormatDigest_OversizedLinkFallsBackToPlainText(t *testing.T) {
	// Ссылка длиннее 4096: якорь не влезает даже с пустым заголовком —
	// ждём пункт без <a> (просто текст), одно сообщение в лимите.
	items := []rss.Item{{
		Title: "Короткая новость",
		Link:  "https://example.com/" + strings.Repeat("a", 5000),
	}}
	msgs := formatDigest(items)
	if len(msgs) != 1 {
		t.Fatalf("len(msgs) = %d, want 1", len(msgs))
	}
	msg := msgs[0]
	if len(msg) > messageLimit {
		t.Errorf("len(msg) = %d > лимита %d", len(msg), messageLimit)
	}
	lines := strings.Split(msg, "\n")
	item := lines[len(lines)-1]
	if strings.Contains(item, "<a") {
		t.Errorf("ожидали пункт без якоря: %.80q", item)
	}
	if !strings.Contains(item, "Короткая новость") {
		t.Errorf("пропал заголовок: %.80q", item)
	}
}

func TestFormatDigest_LineJustOverLineLimitFitsFirstMessage(t *testing.T) {
	// Пункт длиной в (lineLimit, messageLimit] сам по себе укладывается в
	// messageLimit, но вместе с заголовком — нет. До фикса триггер обрезки
	// смотрел на messageLimit, а не lineLimit, так что такой пункт не
	// обрезался: заголовок уходил отдельным сообщением, а пункт — вторым.
	// После фикса пункт обрезается, и всё влезает в одно сообщение.
	link := "https://example.com/1"
	prefixLen := len(fmt.Sprintf(`• <a href="%s">`, link))
	suffixLen := len("</a>")
	const targetLineLen = messageLimit - 10 // строго между lineLimit и messageLimit
	titleLen := targetLineLen - prefixLen - suffixLen

	items := []rss.Item{{Title: strings.Repeat("a", titleLen), Link: link}}
	rawLine := fmt.Sprintf(`• <a href="%s">%s</a>`, link, strings.Repeat("a", titleLen))
	if len(rawLine) <= lineLimit || len(rawLine) > messageLimit {
		t.Fatalf("тестовая установка неверна: len(rawLine) = %d, want in (%d, %d]", len(rawLine), lineLimit, messageLimit)
	}

	msgs := formatDigest(items)
	if len(msgs) != 1 {
		t.Fatalf("len(msgs) = %d, want 1", len(msgs))
	}
	if !strings.HasPrefix(msgs[0], digestHeader+"\n") {
		t.Errorf("первое сообщение должно содержать заголовок и пункт вместе: %.80q", msgs[0])
	}
	if len(msgs[0]) > messageLimit {
		t.Errorf("len(msgs[0]) = %d > лимита %d", len(msgs[0]), messageLimit)
	}
}

func TestFormatDigest_SplitsAtMessageLimit(t *testing.T) {
	// 100 пунктов по ~100 символам — заведомо больше 4096, ждём несколько
	// сообщений, каждое в лимите, пункты не порваны, все на месте.
	var items []rss.Item
	for i := range 100 {
		items = append(items, rss.Item{
			Title: fmt.Sprintf("Новость %03d %s", i, strings.Repeat("х", 60)),
			Link:  fmt.Sprintf("https://example.com/%03d", i),
		})
	}
	msgs := formatDigest(items)
	if len(msgs) < 2 {
		t.Fatalf("len(msgs) = %d, want >= 2", len(msgs))
	}

	total := 0
	for i, m := range msgs {
		if len(m) > messageLimit {
			t.Errorf("msgs[%d]: len %d > лимита %d", i, len(m), messageLimit)
		}
		if i == 0 && !strings.HasPrefix(m, "📰 Новости рынка\n") {
			t.Errorf("первое сообщение без заголовка: %q", m[:40])
		}
		if i > 0 && strings.Contains(m, "📰") {
			t.Errorf("заголовок должен быть только в первом сообщении, msgs[%d]", i)
		}
		for _, line := range strings.Split(m, "\n") {
			if strings.HasPrefix(line, "• ") {
				if !strings.HasSuffix(line, "</a>") {
					t.Errorf("порванный пункт: %q", line)
				}
				total++
			}
		}
	}
	if total != len(items) {
		t.Errorf("пунктов во всех сообщениях %d, want %d", total, len(items))
	}
}
