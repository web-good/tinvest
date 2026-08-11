# Дублирование ERROR-логов в Telegram «General» — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Каждая запись уровня ERROR, кроме вывода в stdout, дублируется сообщением в тему «General» Telegram-группы — с дедупом, лимитом частоты и минутной сводкой подавленного.

**Architecture:** `pkg/logger.Init()` оборачивает свой `TextHandler` в `teeHandler`: тот делегирует все записи базовому хендлеру без изменений и дополнительно отдаёт записи `>= slog.LevelError` в `ErrorSink`, установленный через `logger.SetErrorSink`. Реализация sink живёт в `internal/service/notification/errorlog`: неблокирующий `Publish` кладёт событие в буферизованный канал, одна горутина-воркер применяет дедуп и лимит и шлёт сообщения через `telegram.Client`, привязанный к General (`threadID = 0`). Sink подключается шагом `initErrorLogSink` в `internal/app` после инициализации бота, поэтому CLI-утилиты, зовущие тот же `logger.Init()`, остаются без Telegram-копии.

**Tech Stack:** Go 1.25, `log/slog`, `html` (экранирование), `github.com/go-telegram/bot` через существующий `pkg/client/telegram`, `pkg/closer`, стандартный `testing`.

**Спека:** `docs/superpowers/specs/2026-08-11-error-log-telegram-design.md`

## Global Constraints

- Дублируется только уровень `slog.LevelError`. `Warn` и `Info` — нет.
- Поток в stdout не меняется ни по объёму, ни по формату: дедуп и лимиты действуют исключительно на Telegram-копию.
- Пакет `pkg/logger` не импортирует `internal/**` и ничего не знает про Telegram.
- Код в `internal/service/notification/errorlog` **никогда** не вызывает `pkg/logger` — собственные сбои пишутся в `os.Stderr` через `fmt.Fprintf`. Это защита от бесконечной рекурсии «ошибка отправки → лог ERROR → отправка».
- `Publish` обязан не блокировать вызывающую горутину ни при каких условиях.
- Бот шлёт с `ParseMode: HTML` — весь динамический текст проходит через `html.EscapeString`, обрезка не имеет права разорвать HTML-entity.
- Новых переменных окружения не добавляется. General — это `threadID = 0` в существующем `Bot.SendMessageToTopic`.
- Константы: буфер канала 256, окно дедупа 5 минут, лимит 10 отправок за минутное окно, период сводки 1 минута, потолок сообщения 3800 рун, топ-3 класса в сводке.
- Комментарии в коде — на русском, как в остальном репозитории.
- Приёмочный гейт всей работы — `./bin/mage ci` (lint + `go test -race ./...` + проверка дрейфа моков).

## File Structure

| Файл | Ответственность |
|---|---|
| `pkg/logger/sink.go` (создать) | `ErrorEvent`, `ErrorSink`, `SetErrorSink`, `teeHandler` |
| `pkg/logger/sink_test.go` (создать) | Тесты маршрутизации уровней и сохранения stdout |
| `pkg/logger/logger.go` (изменить) | `Init()` собирает `teeHandler` поверх `TextHandler` |
| `internal/service/notification/errorlog/format.go` (создать) | Чистые функции текста: событие, сводка, экранирование, обрезка |
| `internal/service/notification/errorlog/format_test.go` (создать) | Тесты формата |
| `internal/service/notification/errorlog/errorlog.go` (создать) | `Sink`: `New`, `Publish`, `Run`, отправка, дедуп, лимит, сводка |
| `internal/service/notification/errorlog/errorlog_test.go` (создать) | Тесты доставки, отсутствия блокировки и рекурсии |
| `internal/service/notification/errorlog/throttle_test.go` (создать) | Тесты дедупа, лимита и сводки |
| `internal/service_provider/client.go` (изменить) | `GetErrorLogSender()` — Client на General |
| `internal/app/init_error_log_sink.go` (создать) | Сборка sink, запуск воркера, регистрация в closer |
| `internal/app/app.go` (изменить) | Шаг `initErrorLogSink` в списке инициализации |
| `CLAUDE.md` (изменить) | Строка про дублирование ERROR-логов в General |

---

### Task 1: `pkg/logger` — ErrorSink и teeHandler

**Files:**
- Create: `pkg/logger/sink.go`
- Create: `pkg/logger/sink_test.go`
- Modify: `pkg/logger/logger.go:17-19` (функция `Init`)

**Interfaces:**
- Consumes: ничего (первая задача).
- Produces: `logger.ErrorEvent{Time time.Time; Message string; Attrs []slog.Attr}`, `logger.ErrorSink` с методом `Publish(ErrorEvent)`, функция `logger.SetErrorSink(s ErrorSink)`. На них опираются Task 2, 3, 4, 5.

- [ ] **Step 1: Написать падающие тесты**

Создать `pkg/logger/sink_test.go`:

```go
package logger

import (
	"bytes"
	"context"
	"log/slog"
	"strings"
	"sync"
	"testing"
)

// fakeSink собирает события; обращения возможны из другой горутины.
type fakeSink struct {
	mu     sync.Mutex
	events []ErrorEvent
}

func (f *fakeSink) Publish(ev ErrorEvent) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.events = append(f.events, ev)
}

func (f *fakeSink) snapshot() []ErrorEvent {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]ErrorEvent(nil), f.events...)
}

// initWithBuffer собирает тот же teeHandler, что и Init, но с выводом в буфер:
// проверять stdout процесса переносимо нельзя.
func initWithBuffer(t *testing.T) *bytes.Buffer {
	t.Helper()
	buf := &bytes.Buffer{}
	base := slog.NewTextHandler(buf, &slog.HandlerOptions{Level: slog.LevelDebug})
	logger = slog.New(&teeHandler{base: base})
	t.Cleanup(func() { SetErrorSink(nil) })
	return buf
}

func TestErrorReachesSink(t *testing.T) {
	initWithBuffer(t)
	sink := &fakeSink{}
	SetErrorSink(sink)

	ErrorContext(context.Background(), "не удалось выставить ордер", slog.String("ticker", "UGLD"))

	got := sink.snapshot()
	if len(got) != 1 {
		t.Fatalf("получено %d событий, ожидалось 1", len(got))
	}
	if got[0].Message != "не удалось выставить ордер" {
		t.Errorf("Message = %q", got[0].Message)
	}
	if len(got[0].Attrs) != 1 || got[0].Attrs[0].Key != "ticker" {
		t.Errorf("Attrs = %v, ожидался один атрибут ticker", got[0].Attrs)
	}
	if got[0].Time.IsZero() {
		t.Error("Time не заполнено")
	}
}

func TestInfoAndWarnDoNotReachSink(t *testing.T) {
	initWithBuffer(t)
	sink := &fakeSink{}
	SetErrorSink(sink)

	Info("информационное")
	Warn("предупреждение")
	InfoContext(context.Background(), "ещё информационное")
	DebugContext(context.Background(), "отладочное")

	if got := sink.snapshot(); len(got) != 0 {
		t.Fatalf("в sink попало %d событий, ожидалось 0: %v", len(got), got)
	}
}

func TestStdoutKeepsErrorWhenSinkActive(t *testing.T) {
	buf := initWithBuffer(t)
	SetErrorSink(&fakeSink{})

	ErrorContext(context.Background(), "ошибка отправки", slog.String("ticker", "UGLD"))

	out := buf.String()
	if !strings.Contains(out, "level=ERROR") {
		t.Errorf("в выводе нет level=ERROR: %q", out)
	}
	if !strings.Contains(out, "ошибка отправки") {
		t.Errorf("в выводе нет текста сообщения: %q", out)
	}
	if !strings.Contains(out, "ticker=UGLD") {
		t.Errorf("в выводе нет атрибута: %q", out)
	}
}

func TestErrorWithoutSinkDoesNotPanic(t *testing.T) {
	buf := initWithBuffer(t)
	SetErrorSink(nil)

	ErrorContext(context.Background(), "ошибка без sink")

	if !strings.Contains(buf.String(), "ошибка без sink") {
		t.Errorf("запись не попала в вывод: %q", buf.String())
	}
}
```

- [ ] **Step 2: Убедиться, что тесты не компилируются/падают**

Run: `go test ./pkg/logger/ -run TestError -v`
Expected: FAIL — `undefined: ErrorEvent`, `undefined: teeHandler`, `undefined: SetErrorSink`.

- [ ] **Step 3: Реализовать sink.go**

Создать `pkg/logger/sink.go`:

```go
package logger

import (
	"context"
	"log/slog"
	"sync/atomic"
	"time"
)

// ErrorEvent — запись уровня ERROR, снятая со slog.Record для внешнего получателя.
type ErrorEvent struct {
	Time    time.Time
	Message string
	Attrs   []slog.Attr
}

// ErrorSink принимает ERROR-записи для доставки за пределы stdout.
// Publish зовётся из горячего пути логирования и обязан не блокировать.
type ErrorSink interface {
	Publish(ErrorEvent)
}

// errorSink хранится атомарно: устанавливается на старте приложения, читается
// из любых горутин, которые пишут логи.
var errorSink atomic.Pointer[ErrorSink]

// SetErrorSink подключает получателя ERROR-записей. Отдельный вызов, а не
// параметр Init: логгер поднимается раньше Telegram-бота, от которого зависит
// получатель. nil отключает дублирование — так живут CLI-утилиты.
func SetErrorSink(s ErrorSink) {
	if s == nil {
		errorSink.Store(nil)

		return
	}
	errorSink.Store(&s)
}

// teeHandler отдаёт каждую запись базовому хендлеру и дублирует ERROR в sink.
// WithAttrs/WithGroup сохраняют базовое поведение; накопленные ими атрибуты в
// ErrorEvent не попадают — глобальное API пакета этих методов не использует.
type teeHandler struct {
	base slog.Handler
}

func (h *teeHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.base.Enabled(ctx, level)
}

func (h *teeHandler) Handle(ctx context.Context, record slog.Record) error {
	if record.Level >= slog.LevelError {
		if p := errorSink.Load(); p != nil {
			(*p).Publish(eventFromRecord(record))
		}
	}

	return h.base.Handle(ctx, record)
}

func (h *teeHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &teeHandler{base: h.base.WithAttrs(attrs)}
}

func (h *teeHandler) WithGroup(name string) slog.Handler {
	return &teeHandler{base: h.base.WithGroup(name)}
}

func eventFromRecord(record slog.Record) ErrorEvent {
	event := ErrorEvent{Time: record.Time, Message: record.Message}
	record.Attrs(func(attr slog.Attr) bool {
		event.Attrs = append(event.Attrs, attr)

		return true
	})

	return event
}
```

- [ ] **Step 4: Подключить teeHandler в Init**

В `pkg/logger/logger.go` заменить тело `Init`:

```go
func Init() {
	base := slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug})
	logger = slog.New(&teeHandler{base: base})
}
```

- [ ] **Step 5: Прогнать тесты**

Run: `go test ./pkg/logger/ -race -v`
Expected: PASS, все четыре теста.

- [ ] **Step 6: Коммит**

```bash
git add pkg/logger/sink.go pkg/logger/sink_test.go pkg/logger/logger.go
git commit -m "feat(logger): хук ErrorSink на записи уровня ERROR"
```

---

### Task 2: Форматирование сообщений

**Files:**
- Create: `internal/service/notification/errorlog/format.go`
- Create: `internal/service/notification/errorlog/format_test.go`

**Interfaces:**
- Consumes: `logger.ErrorEvent` из Task 1.
- Produces:
  - `type classCount struct { key string; count int }`
  - `func formatEvent(event logger.ErrorEvent, appEnv string, loc *time.Location) string`
  - `func formatSummary(classes []classCount, dropped int) string` — пустая строка, когда слать нечего
  - `func truncateEscaped(s string, limit int) string`
  - константы `maxMessageLen = 3800`, `topSummaryClasses = 3`

  На них опираются Task 3 и Task 4.

- [ ] **Step 1: Написать падающие тесты**

Создать `internal/service/notification/errorlog/format_test.go`:

```go
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
```

- [ ] **Step 2: Убедиться, что тесты падают**

Run: `go test ./internal/service/notification/errorlog/ -v`
Expected: FAIL — `undefined: formatEvent`, `undefined: formatSummary`, `undefined: classCount`.

- [ ] **Step 3: Реализовать format.go**

Создать `internal/service/notification/errorlog/format.go`:

```go
package errorlog

import (
	"fmt"
	"html"
	"log/slog"
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
```

- [ ] **Step 4: Прогнать тесты**

Run: `go test ./internal/service/notification/errorlog/ -race -v`
Expected: PASS, все девять тестов.

- [ ] **Step 5: Коммит**

```bash
git add internal/service/notification/errorlog/format.go internal/service/notification/errorlog/format_test.go
git commit -m "feat(errorlog): формат сообщений об ошибках и сводки"
```

---

### Task 3: Sink — неблокирующий Publish, воркер, отправка

**Files:**
- Create: `internal/service/notification/errorlog/errorlog.go`
- Create: `internal/service/notification/errorlog/errorlog_test.go`

**Interfaces:**
- Consumes: `logger.ErrorEvent`, `logger.SetErrorSink` (Task 1); `formatEvent`, `formatSummary`, `classCount` (Task 2); `telegram.Client` из `pkg/client/telegram` — метод `SendMessage(msg string) error`.
- Produces:
  - `type Sink struct` с полями `tg telegram.Client`, `appEnv string`, `loc *time.Location`, `now func() time.Time`, `events chan logger.ErrorEvent`, `dropped atomic.Int64`, `ticker *time.Ticker`, `summaryTick <-chan time.Time`
  - `func New(tg telegram.Client, appEnv string) *Sink`
  - `func (s *Sink) Publish(event logger.ErrorEvent)`
  - `func (s *Sink) Run(ctx context.Context)`
  - `func (s *Sink) send(msg string)`
  - `func (s *Sink) handle(event logger.ErrorEvent)` — в этой задаче отправляет всё подряд; дедуп и лимит добавляет Task 4
  - `func (s *Sink) flushSummary()` — в этой задаче отчитывается только о переполнении буфера; счётчики подавленного добавляет Task 4
  - константы `eventBufferSize = 256`, `summaryPeriod = time.Minute`

  На них опирается Task 4 (правит `handle`/`flushSummary`) и Task 5 (зовёт `New`/`Run`).

- [ ] **Step 1: Написать падающие тесты**

Создать `internal/service/notification/errorlog/errorlog_test.go`:

```go
package errorlog

import (
	"context"
	"strings"
	"sync"
	"testing"
	"time"

	"tinvest/pkg/logger"
)

// fakeSender — telegram.Client, копящий отправленные сообщения.
// err задаётся до старта воркера и после не меняется.
type fakeSender struct {
	mu   sync.Mutex
	msgs []string
	err  error
	sent chan struct{}
}

func newFakeSender() *fakeSender {
	return &fakeSender{sent: make(chan struct{}, 64)}
}

func (f *fakeSender) SendMessage(msg string) error {
	f.mu.Lock()
	f.msgs = append(f.msgs, msg)
	err := f.err
	f.mu.Unlock()
	select {
	case f.sent <- struct{}{}:
	default:
	}

	return err
}

func (f *fakeSender) SendMessageToChat(_ int64, msg string) error { return f.SendMessage(msg) }

func (f *fakeSender) SendMessageToTopic(_ int64, _ int, msg string) error { return f.SendMessage(msg) }

func (f *fakeSender) snapshot() []string {
	f.mu.Lock()
	defer f.mu.Unlock()

	return append([]string(nil), f.msgs...)
}

// waitSent ждёт n отправок или падает по таймауту.
func (f *fakeSender) waitSent(t *testing.T, n int) {
	t.Helper()
	deadline := time.After(2 * time.Second)
	for i := 0; i < n; i++ {
		select {
		case <-f.sent:
		case <-deadline:
			t.Fatalf("дождались %d отправок из %d: %v", i, n, f.snapshot())
		}
	}
}

// newTestSink собирает Sink с ручным тиком сводки и фиксированными часами.
func newTestSink(t *testing.T, sender *fakeSender, now func() time.Time) (*Sink, chan time.Time) {
	t.Helper()
	sink := New(sender, "prod")
	sink.ticker.Stop()
	tick := make(chan time.Time, 1)
	sink.summaryTick = tick
	sink.now = now

	return sink, tick
}

func TestSinkDeliversEventToTelegram(t *testing.T) {
	sender := newFakeSender()
	fixed := time.Date(2026, 8, 11, 11, 23, 5, 0, time.UTC)
	sink, _ := newTestSink(t, sender, func() time.Time { return fixed })

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go sink.Run(ctx)

	sink.Publish(logger.ErrorEvent{Time: fixed, Message: "не удалось выставить ордер"})

	sender.waitSent(t, 1)
	msgs := sender.snapshot()
	if len(msgs) != 1 || !strings.Contains(msgs[0], "не удалось выставить ордер") {
		t.Fatalf("отправлено %v", msgs)
	}
	if !strings.Contains(msgs[0], "🔴 <b>ERROR</b> · prod · 14:23:05") {
		t.Errorf("нет ожидаемого заголовка: %q", msgs[0])
	}
}

func TestPublishDoesNotBlockWhenBufferIsFull(t *testing.T) {
	sender := newFakeSender()
	fixed := time.Date(2026, 8, 11, 11, 0, 0, 0, time.UTC)
	// Run не запускаем: канал заведомо переполняется.
	sink, _ := newTestSink(t, sender, func() time.Time { return fixed })

	done := make(chan struct{})
	go func() {
		defer close(done)
		for i := 0; i < eventBufferSize+5; i++ {
			sink.Publish(logger.ErrorEvent{Time: fixed, Message: "переполнение"})
		}
	}()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("Publish заблокировался на полном буфере")
	}

	if got := sink.dropped.Load(); got != 5 {
		t.Errorf("dropped = %d, ожидалось 5", got)
	}
}

func TestSendFailureDoesNotRecurse(t *testing.T) {
	sender := newFakeSender()
	sender.err = errFakeSend
	fixed := time.Date(2026, 8, 11, 11, 0, 0, 0, time.UTC)
	sink, _ := newTestSink(t, sender, func() time.Time { return fixed })

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go sink.Run(ctx)

	// Подключаем sink к настоящему логгеру: если бы воркер логировал свой сбой
	// через pkg/logger, ERROR вернулся бы сюда и породил бесконечный цикл.
	logger.Init()
	logger.SetErrorSink(sink)
	t.Cleanup(func() { logger.SetErrorSink(nil) })

	logger.ErrorContext(ctx, "ошибка обращения к API")

	sender.waitSent(t, 1)
	time.Sleep(200 * time.Millisecond)

	if got := len(sender.snapshot()); got != 1 {
		t.Fatalf("отправок %d, ожидалась ровно 1 — сбой отправки не должен порождать новые события", got)
	}
}

func TestRunStopsOnContextCancel(t *testing.T) {
	sender := newFakeSender()
	fixed := time.Date(2026, 8, 11, 11, 0, 0, 0, time.UTC)
	sink, _ := newTestSink(t, sender, func() time.Time { return fixed })

	ctx, cancel := context.WithCancel(context.Background())
	stopped := make(chan struct{})
	go func() {
		sink.Run(ctx)
		close(stopped)
	}()

	cancel()
	select {
	case <-stopped:
	case <-time.After(2 * time.Second):
		t.Fatal("Run не завершился после отмены контекста")
	}
}
```

Добавить в тот же файл объявление ошибки-заглушки:

```go
var errFakeSend = errors.New("telegram недоступен")
```

(и импорт `"errors"`).

- [ ] **Step 2: Убедиться, что тесты падают**

Run: `go test ./internal/service/notification/errorlog/ -run TestSink -v`
Expected: FAIL — `undefined: New`, `undefined: Sink`, `undefined: eventBufferSize`.

- [ ] **Step 3: Реализовать errorlog.go**

Создать `internal/service/notification/errorlog/errorlog.go`:

```go
// Package errorlog дублирует ERROR-логи приложения в тему Telegram «General».
//
// Пакет намеренно не использует pkg/logger: собственные сбои пишутся в stderr.
// Иначе ошибка отправки в Telegram логировалась бы как ERROR, возвращалась в
// этот же sink и порождала бесконечный цикл отправок.
package errorlog

import (
	"context"
	"fmt"
	"os"
	"sync/atomic"
	"time"

	"tinvest/pkg/client/telegram"
	"tinvest/pkg/logger"
)

const (
	// eventBufferSize — глубина очереди событий. Переживает всплеск ретраев и
	// при этом не даёт очереди расти неограниченно.
	eventBufferSize = 256
	// summaryPeriod — период отправки сводки подавленного.
	summaryPeriod = time.Minute
)

// Sink принимает ERROR-записи логгера и доставляет их в Telegram.
// Реализует logger.ErrorSink.
type Sink struct {
	tg     telegram.Client
	appEnv string
	loc    *time.Location
	// now — источник времени для дедупа и лимита; подменяется в тестах.
	now func() time.Time

	events  chan logger.ErrorEvent
	dropped atomic.Int64

	ticker      *time.Ticker
	summaryTick <-chan time.Time
}

// New собирает sink поверх готового Client, уже привязанного к нужному чату и теме.
func New(tg telegram.Client, appEnv string) *Sink {
	// Ошибка загрузки зоны не должна ронять приложение: без МСК время в
	// сообщении будет в UTC, что хуже, но не смертельно.
	loc, err := time.LoadLocation("Europe/Moscow")
	if err != nil {
		loc = time.UTC
	}
	ticker := time.NewTicker(summaryPeriod)

	return &Sink{
		tg:          tg,
		appEnv:      appEnv,
		loc:         loc,
		now:         time.Now,
		events:      make(chan logger.ErrorEvent, eventBufferSize),
		ticker:      ticker,
		summaryTick: ticker.C,
	}
}

// Publish кладёт событие в очередь и немедленно возвращается. Переполненная
// очередь означает потерю события для Telegram (в stdout оно уже есть);
// счётчик попадёт в ближайшую сводку.
func (s *Sink) Publish(event logger.ErrorEvent) {
	select {
	case s.events <- event:
	default:
		s.dropped.Add(1)
	}
}

// Run крутит единственную горутину доставки. Последовательная отправка
// сохраняет порядок сообщений в теме.
func (s *Sink) Run(ctx context.Context) {
	defer s.ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case event := <-s.events:
			s.handle(event)
		case <-s.summaryTick:
			s.flushSummary()
		}
	}
}

// handle решает судьбу одного события.
func (s *Sink) handle(event logger.ErrorEvent) {
	s.send(formatEvent(event, s.appEnv, s.loc))
}

// flushSummary отправляет сводку подавленного за период.
func (s *Sink) flushSummary() {
	dropped := int(s.dropped.Swap(0))
	if msg := formatSummary(nil, dropped); msg != "" {
		s.send(msg)
	}
}

// send пишет в Telegram. Сбой уходит прямо в stderr — см. комментарий к пакету.
func (s *Sink) send(msg string) {
	if err := s.tg.SendMessage(msg); err != nil {
		fmt.Fprintf(os.Stderr, "errorlog: отправка в telegram не удалась: %v\n", err)
	}
}
```

- [ ] **Step 4: Прогнать тесты пакета**

Run: `go test ./internal/service/notification/errorlog/ -race -v`
Expected: PASS — тесты формата из Task 2 и четыре новых.

- [ ] **Step 5: Коммит**

```bash
git add internal/service/notification/errorlog/errorlog.go internal/service/notification/errorlog/errorlog_test.go
git commit -m "feat(errorlog): доставка ERROR-записей в telegram без блокировки логгера"
```

---

### Task 4: Дедуп, лимит частоты и сводка

**Files:**
- Modify: `internal/service/notification/errorlog/errorlog.go` (поля `Sink`, `New`, `handle`, `flushSummary`)
- Create: `internal/service/notification/errorlog/throttle_test.go`

**Interfaces:**
- Consumes: `Sink`, `New`, `Publish`, `Run`, `send` (Task 3); `formatSummary`, `classCount` (Task 2); тестовые хелперы `fakeSender`, `newTestSink` из `errorlog_test.go` (Task 3) — они в том же пакете.
- Produces: новые поля `Sink`: `lastSent map[string]time.Time`, `suppressed map[string]int`, `windowStart time.Time`, `sentInWindow int`; функция `sortedClasses(m map[string]int) []classCount`; константы `dedupWindow = 5 * time.Minute`, `maxPerWindow = 10`, `limitWindow = time.Minute`.

- [ ] **Step 1: Написать падающие тесты**

Создать `internal/service/notification/errorlog/throttle_test.go`:

```go
package errorlog

import (
	"context"
	"strings"
	"sync"
	"testing"
	"time"

	"tinvest/pkg/logger"
)

// fakeClock — управляемые часы: воркер читает время только через now().
type fakeClock struct {
	mu sync.Mutex
	t  time.Time
}

func newFakeClock(t time.Time) *fakeClock { return &fakeClock{t: t} }

func (c *fakeClock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()

	return c.t
}

func (c *fakeClock) advance(d time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.t = c.t.Add(d)
}

func TestDuplicateWithinWindowIsSuppressed(t *testing.T) {
	sender := newFakeSender()
	clock := newFakeClock(time.Date(2026, 8, 11, 11, 0, 0, 0, time.UTC))
	sink, _ := newTestSink(t, sender, clock.Now)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go sink.Run(ctx)

	event := logger.ErrorEvent{Time: clock.Now(), Message: "не удалось получить свечи"}
	sink.Publish(event)
	sender.waitSent(t, 1)

	sink.Publish(event)
	sink.Publish(event)
	time.Sleep(200 * time.Millisecond)

	if got := len(sender.snapshot()); got != 1 {
		t.Fatalf("отправок %d, ожидалась 1: повтор в окне дедупа должен подавляться", got)
	}

	clock.advance(dedupWindow + time.Second)
	sink.Publish(event)
	sender.waitSent(t, 1)

	if got := len(sender.snapshot()); got != 2 {
		t.Fatalf("отправок %d, ожидалось 2: после окна дедупа сообщение должно пройти", got)
	}
}

func TestPerWindowLimit(t *testing.T) {
	sender := newFakeSender()
	clock := newFakeClock(time.Date(2026, 8, 11, 11, 0, 0, 0, time.UTC))
	sink, _ := newTestSink(t, sender, clock.Now)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go sink.Run(ctx)

	// Сообщения различаются, поэтому дедуп не срабатывает — режет только лимит.
	for i := 0; i < maxPerWindow; i++ {
		sink.Publish(logger.ErrorEvent{Time: clock.Now(), Message: "ошибка " + string(rune('A'+i))})
	}
	sender.waitSent(t, maxPerWindow)

	sink.Publish(logger.ErrorEvent{Time: clock.Now(), Message: "ошибка сверх лимита"})
	time.Sleep(200 * time.Millisecond)

	if got := len(sender.snapshot()); got != maxPerWindow {
		t.Fatalf("отправок %d, ожидалось %d: сообщение сверх лимита должно подавляться", got, maxPerWindow)
	}

	clock.advance(limitWindow + time.Second)
	sink.Publish(logger.ErrorEvent{Time: clock.Now(), Message: "ошибка в новом окне"})
	sender.waitSent(t, 1)

	if got := len(sender.snapshot()); got != maxPerWindow+1 {
		t.Fatalf("отправок %d, ожидалось %d: в новом окне лимит должен обнулиться", got, maxPerWindow+1)
	}
}

func TestSummaryReportsSuppressedAndDropped(t *testing.T) {
	sender := newFakeSender()
	clock := newFakeClock(time.Date(2026, 8, 11, 11, 0, 0, 0, time.UTC))
	sink, tick := newTestSink(t, sender, clock.Now)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go sink.Run(ctx)

	first := logger.ErrorEvent{Time: clock.Now(), Message: "не удалось получить свечи"}
	second := logger.ErrorEvent{Time: clock.Now(), Message: "сбой чтения стейта"}
	sink.Publish(first)
	sink.Publish(second)
	sender.waitSent(t, 2)

	// По три повтора каждого — все шесть подавляются дедупом.
	for i := 0; i < 3; i++ {
		sink.Publish(first)
		sink.Publish(second)
	}
	time.Sleep(200 * time.Millisecond)
	sink.dropped.Store(4)

	tick <- clock.Now()
	sender.waitSent(t, 1)

	msgs := sender.snapshot()
	summary := msgs[len(msgs)-1]
	for _, want := range []string{
		"подавлено 6 ошибок за минуту",
		"3 × не удалось получить свечи",
		"3 × сбой чтения стейта",
		"+ 4 событий не попали в очередь",
	} {
		if !strings.Contains(summary, want) {
			t.Errorf("в сводке нет %q:\n%s", want, summary)
		}
	}
}

func TestSummarySilentWhenNothingSuppressed(t *testing.T) {
	sender := newFakeSender()
	clock := newFakeClock(time.Date(2026, 8, 11, 11, 0, 0, 0, time.UTC))
	sink, tick := newTestSink(t, sender, clock.Now)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go sink.Run(ctx)

	tick <- clock.Now()
	time.Sleep(200 * time.Millisecond)

	if got := sender.snapshot(); len(got) != 0 {
		t.Fatalf("при пустой сводке отправок быть не должно, получено: %v", got)
	}
}

func TestSummaryResetsCountersBetweenPeriods(t *testing.T) {
	sender := newFakeSender()
	clock := newFakeClock(time.Date(2026, 8, 11, 11, 0, 0, 0, time.UTC))
	sink, tick := newTestSink(t, sender, clock.Now)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go sink.Run(ctx)

	event := logger.ErrorEvent{Time: clock.Now(), Message: "повторяющаяся ошибка"}
	sink.Publish(event)
	sender.waitSent(t, 1)
	sink.Publish(event)
	time.Sleep(200 * time.Millisecond)

	tick <- clock.Now()
	sender.waitSent(t, 1)

	tick <- clock.Now()
	time.Sleep(200 * time.Millisecond)

	if got := len(sender.snapshot()); got != 2 {
		t.Fatalf("отправок %d, ожидалось 2: вторая сводка должна быть пустой", got)
	}
}
```

- [ ] **Step 2: Убедиться, что тесты падают**

Run: `go test ./internal/service/notification/errorlog/ -run 'TestDuplicate|TestPerWindow|TestSummary' -v`
Expected: FAIL — `undefined: dedupWindow`, `undefined: maxPerWindow`, `undefined: limitWindow`; тесты дедупа падают, потому что `handle` пока шлёт всё подряд.

- [ ] **Step 3: Добавить константы и поля состояния**

В `internal/service/notification/errorlog/errorlog.go` дополнить блок констант:

```go
const (
	// eventBufferSize — глубина очереди событий. Переживает всплеск ретраев и
	// при этом не даёт очереди расти неограниченно.
	eventBufferSize = 256
	// summaryPeriod — период отправки сводки подавленного.
	summaryPeriod = time.Minute
	// dedupWindow — сколько молчим о повторе того же класса ошибки.
	dedupWindow = 5 * time.Minute
	// limitWindow и maxPerWindow — потолок отправок за окно. Telegram режет
	// группу на ~20 сообщениях в минуту, а туда же пишут стратегии, новости и
	// отчёты портфеля: половина лимита остаётся им.
	limitWindow  = time.Minute
	maxPerWindow = 10
)
```

Добавить в структуру `Sink` поля (все читаются и пишутся только из горутины `Run`, поэтому синхронизация не нужна):

```go
	// Состояние тротлинга. Живёт целиком внутри Run — единственной горутины,
	// которая его трогает.
	lastSent     map[string]time.Time
	suppressed   map[string]int
	windowStart  time.Time
	sentInWindow int
```

Инициализировать их в `New`, в возвращаемом литерале:

```go
		lastSent:    make(map[string]time.Time),
		suppressed:  make(map[string]int),
```

- [ ] **Step 4: Реализовать дедуп и лимит в handle**

Заменить `handle` и `flushSummary` в `errorlog.go`:

```go
// handle решает судьбу одного события: дедуп по классу, затем общий лимит.
// Ключ дедупа — только текст сообщения: в ретрай-цикле атрибуты меняются
// (тикер, id заявки, текст gRPC-ошибки), и ключ с ними не поймал бы дубли.
func (s *Sink) handle(event logger.ErrorEvent) {
	now := s.now()
	s.rollWindow(now)

	key := event.Message
	if last, ok := s.lastSent[key]; ok && now.Sub(last) < dedupWindow {
		s.suppressed[key]++

		return
	}
	if s.sentInWindow >= maxPerWindow {
		s.suppressed[key]++

		return
	}

	s.lastSent[key] = now
	s.sentInWindow++
	s.send(formatEvent(event, s.appEnv, s.loc))
}

// rollWindow сбрасывает счётчик отправок при переходе в новое окно.
func (s *Sink) rollWindow(now time.Time) {
	if s.windowStart.IsZero() || now.Sub(s.windowStart) >= limitWindow {
		s.windowStart = now
		s.sentInWindow = 0
	}
}

// flushSummary отправляет сводку подавленного и обнуляет счётчики. Сводка не
// расходует лимит окна: она одна за период и обязана дойти — иначе тишина в
// чате означала бы «ошибок нет» там, где они просто подавлены.
func (s *Sink) flushSummary() {
	dropped := int(s.dropped.Swap(0))
	classes := sortedClasses(s.suppressed)
	s.suppressed = make(map[string]int)

	if msg := formatSummary(classes, dropped); msg != "" {
		s.send(msg)
	}
}
```

- [ ] **Step 5: Реализовать sortedClasses**

Добавить в `internal/service/notification/errorlog/format.go`:

```go
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
```

Добавить импорт `"sort"` в `format.go`.

- [ ] **Step 6: Прогнать весь пакет**

Run: `go test ./internal/service/notification/errorlog/ -race -v`
Expected: PASS — тесты формата, доставки и все пять новых тестов тротлинга.

- [ ] **Step 7: Коммит**

```bash
git add internal/service/notification/errorlog/errorlog.go internal/service/notification/errorlog/format.go internal/service/notification/errorlog/throttle_test.go
git commit -m "feat(errorlog): дедуп, лимит частоты и минутная сводка подавленного"
```

---

### Task 5: Подключение в приложении и документация

**Files:**
- Modify: `internal/service_provider/client.go` (после метода `GetNewsSender`)
- Create: `internal/app/init_error_log_sink.go`
- Modify: `internal/app/app.go:64-70` (список `inits` в `initializationLoop`)
- Modify: `CLAUDE.md` (раздел `Development Notes`)

**Interfaces:**
- Consumes: `errorlog.New`, `(*errorlog.Sink).Run` (Task 3, 4); `logger.SetErrorSink` (Task 1); существующие `telegram.NewTopicSender`, `(*ServiceProvider).GetTelegramBot`, `closer.Add`.
- Produces: `(*ServiceProvider).GetErrorLogSender() (telegram.Client, error)`, `(*App).initErrorLogSink(ctx context.Context) error`.

Автотестов у пакетов `internal/app` и `internal/service_provider` в репозитории нет — это чистый wiring; приёмка задачи опирается на компиляцию, `./bin/mage ci` и ручной прогон из Step 6.

- [ ] **Step 1: Добавить GetErrorLogSender**

В `internal/service_provider/client.go` после `GetNewsSender` добавить:

```go
// GetErrorLogSender строит Client для темы General (threadID 0) — туда
// дублируются ERROR-логи. Отдельный метод, а не topicSender: у General нет
// собственного id темы, и предупреждение «topic id is not set» здесь ложное.
func (s *ServiceProvider) GetErrorLogSender() (telegram.Client, error) {
	base, err := s.GetTelegramBot()
	if err != nil {
		return nil, err
	}

	return telegram.NewTopicSender(base, s.appConfig.TelegramClient.GroupChatID, 0), nil
}
```

- [ ] **Step 2: Написать шаг инициализации**

Создать `internal/app/init_error_log_sink.go`:

```go
package app

import (
	"context"

	"tinvest/internal/service/notification/errorlog"
	"tinvest/pkg/closer"
	"tinvest/pkg/logger"
)

// initErrorLogSink подключает дублирование ERROR-логов в тему General.
// Шаг обязан идти после initTelegramBotClient: sink нужен готовый бот.
func (a *App) initErrorLogSink(ctx context.Context) error {
	if a.config.TelegramClient.Token == "" || a.config.TelegramClient.GroupChatID == 0 {
		logger.Warn("error log telegram sink disabled: TELEGRAM/TELEGRAM_GROUP_CHAT_ID are not set")

		return nil
	}

	sender, err := a.sp.GetErrorLogSender()
	if err != nil {
		return err
	}

	sink := errorlog.New(sender, a.config.AppEnv)
	sinkCtx, cancel := context.WithCancel(ctx)
	closer.Add(func() error {
		cancel()

		return nil
	})

	go sink.Run(sinkCtx)
	logger.SetErrorSink(sink)
	logger.InfoContext(ctx, "error log telegram sink enabled")

	return nil
}
```

- [ ] **Step 3: Включить шаг в список инициализации**

В `internal/app/app.go`, в `initializationLoop`, дополнить срез `inits`:

```go
	inits := []func(context.Context) error{
		a.initConfig,
		a.initCollection,
		a.initServiceProvider,
		a.initGrpcClient,
		a.initTelegramBotClient,
		a.initErrorLogSink,
	}
```

- [ ] **Step 4: Проверить сборку и весь тестовый набор**

Run: `go build ./internal/... ./pkg/... ./cmd/... && go test ./... -race`
Expected: сборка без ошибок, тесты PASS.

- [ ] **Step 5: Прогнать приёмочный гейт**

Run: `./bin/mage ci`
Expected: lint без замечаний, `go test -race ./...` PASS, проверка дрейфа моков PASS.

- [ ] **Step 6: Ручная проверка на живом боте**

Run: `APP_ENV=dev go run ./cmd/main`

Ожидается: в теме General появляются сообщения об ошибках вида `🔴 <b>ERROR</b> · dev · ЧЧ:ММ:СС`, и те же записи по-прежнему видны в stdout.

Если за прогон ошибок не случилось, форсировать проверку так: временно испортить `RSI_PULLBACK_TOKEN` в `env/local.env` (например, дописать символ) — воркер залогирует ошибку обращения к API, и она должна прийти в General. Портить `TELEGRAM`/`TELEGRAM_GROUP_CHAT_ID` для этой цели нельзя: они ломают сам канал доставки. После проверки вернуть токен обратно и убедиться, что ошибки прекратились.

- [ ] **Step 7: Обновить CLAUDE.md**

В `CLAUDE.md`, в разделе `## Development Notes`, после абзаца про Telegram-темы добавить:

```markdown
- Все записи уровня ERROR дублируются сообщением в тему «General» Telegram-группы (`internal/service/notification/errorlog`, подключается шагом `initErrorLogSink` в `internal/app`). В stdout поток логов не меняется; в Telegram действуют дедуп по тексту сообщения (5 минут), лимит 10 сообщений в минуту и минутная сводка подавленного. Спека: `docs/superpowers/specs/2026-08-11-error-log-telegram-design.md`.
```

- [ ] **Step 8: Коммит**

```bash
git add internal/service_provider/client.go internal/app/init_error_log_sink.go internal/app/app.go CLAUDE.md
git commit -m "feat(app): дублирование ERROR-логов в тему Telegram General"
```

---

## Порядок и зависимости

Задачи строго последовательны: Task 2 использует `logger.ErrorEvent` из Task 1, Task 3 — функции формата из Task 2, Task 4 правит код Task 3 и переиспользует его тестовые хелперы, Task 5 собирает всё в приложении. Параллелить нечего.
