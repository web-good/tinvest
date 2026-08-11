package errorlog

import (
	"fmt"
	"html"
	"log/slog"
	"sort"
	"strings"
	"time"

	"tinvest/pkg/logger"
)

const (
	// maxMessageLen — потолок длины сообщения в рунах. Telegram режет на 4096,
	// остаток — запас на служебные строки.
	maxMessageLen = 3800
	// maxClassLen — потолок длины имени класса в строке сводки.
	maxClassLen = 200
	// topSummaryClasses — сколько классов печатается построчно в сводке.
	topSummaryClasses = 3
)

// classCount — класс ошибки и число подавленных срабатываний.
type classCount struct {
	key   string
	count int
}

// formatEvent собирает сообщение об одной ошибке. Бот шлёт с ParseMode HTML,
// поэтому весь текст события экранируется, а служебные теги ставятся вручную.
func formatEvent(event logger.ErrorEvent, appEnv string, loc *time.Location) string {
	var b strings.Builder
	b.WriteString("🔴 <b>ERROR</b> · ")
	b.WriteString(html.EscapeString(appEnv))
	b.WriteString(" · ")
	b.WriteString(event.Time.In(loc).Format("15:04:05"))
	b.WriteString("\n")
	b.WriteString(html.EscapeString(event.Message))

	if attrs := flattenAttrs("", event.Attrs); len(attrs) > 0 {
		b.WriteString("\n")
		b.WriteString(html.EscapeString(strings.Join(attrs, " ")))
	}

	return truncateEscaped(b.String(), maxMessageLen)
}

// flattenAttrs разворачивает атрибуты в строки key=value; вложенные группы
// получают префикс родителя через точку.
func flattenAttrs(prefix string, attrs []slog.Attr) []string {
	out := make([]string, 0, len(attrs))
	for _, attr := range attrs {
		key := attr.Key
		if prefix != "" {
			key = prefix + "." + key
		}
		value := attr.Value.Resolve()
		if value.Kind() == slog.KindGroup {
			out = append(out, flattenAttrs(key, value.Group())...)

			continue
		}
		out = append(out, key+"="+value.String())
	}

	return out
}

// formatSummary собирает сводку подавленного за период. Пустая строка означает,
// что слать нечего. classes ожидаются отсортированными по убыванию count.
func formatSummary(classes []classCount, dropped int) string {
	total := 0
	for _, c := range classes {
		total += c.count
	}
	if total == 0 && dropped == 0 {
		return ""
	}

	var b strings.Builder
	if total > 0 {
		fmt.Fprintf(&b, "⚠️ <b>подавлено %d ошибок за минуту</b>", total)
		for i, c := range classes {
			if i == topSummaryClasses {
				fmt.Fprintf(&b, "\n+ ещё %d классов ошибок", len(classes)-topSummaryClasses)

				break
			}
			fmt.Fprintf(&b, "\n%d × %s", c.count, truncateEscaped(html.EscapeString(c.key), maxClassLen))
		}
	}
	if dropped > 0 {
		if b.Len() > 0 {
			b.WriteString("\n")
		}
		fmt.Fprintf(&b, "+ %d событий не попали в очередь (переполнение буфера)", dropped)
	}

	return truncateEscaped(b.String(), maxMessageLen)
}

// sortedClasses раскладывает счётчики подавленного по убыванию; при равенстве
// счётчиков порядок задаётся ключом, чтобы сводка была воспроизводимой.
func sortedClasses(m map[string]int) []classCount {
	if len(m) == 0 {
		return nil
	}

	out := make([]classCount, 0, len(m))
	for key, count := range m {
		out = append(out, classCount{key: key, count: count})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].count != out[j].count {
			return out[i].count > out[j].count
		}

		return out[i].key < out[j].key
	})

	return out
}

// dedupKey сводит сообщение к классу ошибки. В этом репозитории тикер, id
// заявки и текст gRPC-ошибки зашиты в сам текст сообщения, поэтому полный
// текст как ключ давал бы новый класс на каждую попытку и дедуп не ловил бы
// ретрай-шторм. Берём префикс до первого двоеточия — это имя подсистемы,
// которое пишут все вызывающие («rsi_pullback: ...», «backtest: ...»).
// Плата: разные по сути ошибки одной подсистемы склеиваются в один класс —
// первая уходит сообщением целиком, остальные попадают в сводку со счётчиком.
func dedupKey(message string) string {
	if i := strings.IndexByte(message, ':'); i >= 0 {
		return message[:i]
	}

	return truncateEscaped(message, maxClassLen)
}

// truncateEscaped обрезает уже экранированную строку до limit рун. Хвост
// откатывается до границы HTML-entity: обрывок вида &qu ломает разбор на
// стороне Telegram и сообщение не уходит вовсе.
func truncateEscaped(s string, limit int) string {
	runes := []rune(s)
	if len(runes) <= limit {
		return s
	}

	out := string(runes[:limit-1])
	if i := strings.LastIndexByte(out, '&'); i >= 0 && !strings.Contains(out[i:], ";") {
		out = out[:i]
	}

	return out + "…"
}
