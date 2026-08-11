package errorlog

import (
	"log/slog"
	"strings"
	"testing"
	"time"

	"tinvest/pkg/logger"
)

func mskTest(t *testing.T) *time.Location {
	t.Helper()
	loc, err := time.LoadLocation("Europe/Moscow")
	if err != nil {
		t.Fatalf("не загрузилась зона Europe/Moscow: %v", err)
	}

	return loc
}

func TestFormatEventHeaderAndAttrs(t *testing.T) {
	loc := mskTest(t)
	event := logger.ErrorEvent{
		// 11:23:05 UTC = 14:23:05 МСК — время в сообщении обязано быть московским.
		Time:    time.Date(2026, 8, 11, 11, 23, 5, 0, time.UTC),
		Message: "не удалось выставить стоп-заявку",
		Attrs: []slog.Attr{
			slog.String("ticker", "UGLD"),
			slog.Int("attempt", 3),
		},
	}

	got := formatEvent(event, "prod", loc)

	for _, want := range []string{
		"🔴 <b>ERROR</b> · prod · 14:23:05",
		"не удалось выставить стоп-заявку",
		"ticker=UGLD",
		"attempt=3",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("в сообщении нет %q:\n%s", want, got)
		}
	}
}

func TestFormatEventEscapesHTML(t *testing.T) {
	loc := mskTest(t)
	event := logger.ErrorEvent{
		Time:    time.Date(2026, 8, 11, 11, 0, 0, 0, time.UTC),
		Message: "ошибка <nil> & сбой",
		Attrs:   []slog.Attr{slog.String("err", `desc = "unavailable" <b>`)},
	}

	got := formatEvent(event, "dev", loc)

	if strings.Contains(got, "<nil>") {
		t.Errorf("угловые скобки в тексте не экранированы:\n%s", got)
	}
	if !strings.Contains(got, "&lt;nil&gt;") {
		t.Errorf("нет экранированного <nil>:\n%s", got)
	}
	if strings.Count(got, "<b>") != 1 {
		t.Errorf("в сообщении должен остаться ровно один служебный тег <b>:\n%s", got)
	}
}

func TestFormatEventFlattensGroups(t *testing.T) {
	loc := mskTest(t)
	event := logger.ErrorEvent{
		Time:    time.Date(2026, 8, 11, 11, 0, 0, 0, time.UTC),
		Message: "сбой заявки",
		Attrs: []slog.Attr{
			slog.Group("order", slog.String("id", "123"), slog.String("side", "buy")),
		},
	}

	got := formatEvent(event, "prod", loc)

	if !strings.Contains(got, "order.id=123") || !strings.Contains(got, "order.side=buy") {
		t.Errorf("группа не развёрнута через точку:\n%s", got)
	}
}

func TestFormatEventTruncatesLongMessage(t *testing.T) {
	loc := mskTest(t)
	event := logger.ErrorEvent{
		Time:    time.Date(2026, 8, 11, 11, 0, 0, 0, time.UTC),
		Message: strings.Repeat("я", 5000),
	}

	got := formatEvent(event, "prod", loc)

	if runes := []rune(got); len(runes) > maxMessageLen {
		t.Fatalf("длина %d рун превышает потолок %d", len(runes), maxMessageLen)
	}
	if !strings.HasSuffix(got, "…") {
		t.Error("обрезанное сообщение должно оканчиваться многоточием")
	}
}

func TestTruncateEscapedDoesNotSplitEntity(t *testing.T) {
	// Строка из кавычек после экранирования состоит из &quot; — обрезка обязана
	// откатиться к границе entity, иначе Telegram отвергнет сообщение.
	escaped := strings.Repeat("&quot;", 100)

	for limit := 10; limit < 60; limit++ {
		got := truncateEscaped(escaped, limit)
		body := strings.TrimSuffix(got, "…")
		if i := strings.LastIndexByte(body, '&'); i >= 0 && !strings.Contains(body[i:], ";") {
			t.Fatalf("limit=%d: обрезка разорвала entity: %q", limit, got)
		}
		if len([]rune(got)) > limit {
			t.Fatalf("limit=%d: длина %d больше лимита", limit, len([]rune(got)))
		}
	}
}

func TestFormatSummaryCountsAndTop3(t *testing.T) {
	classes := []classCount{
		{key: "не удалось получить свечи", count: 23},
		{key: "ошибка отправки в Telegram", count: 19},
		{key: "не удалось выставить стоп-заявку", count: 5},
		{key: "сбой чтения стейта", count: 2},
		{key: "таймаут gRPC", count: 1},
	}

	got := formatSummary(classes, 12)

	for _, want := range []string{
		"подавлено 50 ошибок за минуту",
		"23 × не удалось получить свечи",
		"19 × ошибка отправки в Telegram",
		"5 × не удалось выставить стоп-заявку",
		"+ ещё 2 классов ошибок",
		"+ 12 событий не попали в очередь (переполнение буфера)",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("в сводке нет %q:\n%s", want, got)
		}
	}
	if strings.Contains(got, "сбой чтения стейта") {
		t.Errorf("четвёртый класс не должен печататься построчно:\n%s", got)
	}
}

func TestFormatSummaryDroppedOnly(t *testing.T) {
	got := formatSummary(nil, 7)

	if !strings.Contains(got, "7 событий не попали в очередь") {
		t.Errorf("нет строки про переполнение буфера:\n%s", got)
	}
	if strings.Contains(got, "подавлено") {
		t.Errorf("без подавленных классов заголовка о подавлении быть не должно:\n%s", got)
	}
}

func TestFormatSummaryEmpty(t *testing.T) {
	if got := formatSummary(nil, 0); got != "" {
		t.Errorf("пустая сводка должна быть пустой строкой, получено %q", got)
	}
}

func TestFormatSummaryEscapesClassNames(t *testing.T) {
	got := formatSummary([]classCount{{key: "сбой <nil>", count: 2}}, 0)

	if strings.Contains(got, "<nil>") {
		t.Errorf("имя класса не экранировано:\n%s", got)
	}
}
