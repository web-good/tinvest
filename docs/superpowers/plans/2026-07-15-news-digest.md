# Ежечасный дайджест новостей рынка — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Раз в час постить дайджест свежих новостей российского фондового рынка (RSS smart-lab.ru) в тему «Новости» Telegram-форум-группы.

**Architecture:** Новый RSS-клиент `pkg/client/rss` (stdlib `encoding/xml`, без новых зависимостей) + сервис `internal/service/news` (in-memory дедуп по `PubDate`+GUID, HTML-дайджест с нарезкой по лимиту 4096) + cron-воркер `5 * * * *` в `runProd` по образцу Golden X/Reversion. Отправка через существующий `telegram.NewTopicSender`.

**Tech Stack:** Go 1.25, `net/http` + `encoding/xml` (stdlib), `pkg/scheduler` (robfig/cron), `go-telegram/bot` (уже в проекте), mockery v2, mage.

**Spec:** `docs/superpowers/specs/2026-07-15-news-digest-design.md`

## Global Constraints

- Никаких новых зависимостей в `go.mod`.
- Тесты гоняются с `-race`: `go test -race ./...` (входит в `./bin/mage ci`).
- `go build ./...` падает на `magefiles` — собирать `go build ./internal/... ./pkg/... ./cmd/...`.
- После изменения `.mockery.yaml` или мокаемых интерфейсов: `./bin/mage mocks`.
- Финальная проверка каждой задачи: `./bin/mage ci` (lint + test -race + mock-drift) — тот же гейт, что в CI.
- Комментарии в коде — на русском, в стиле существующих пакетов (см. `pkg/client/telegram/topic_sender.go`).
- Дефолтный URL ленты: `https://smart-lab.ru/news/rss/` (точно как здесь, со слэшем в конце).
- Cron-выражение прод-воркера: `5 * * * *`.

---

### Task 1: `pkg/client/rss` — клиент и парсер RSS 2.0

**Files:**
- Create: `pkg/client/rss/rss.go`
- Create: `pkg/client/rss/rss_test.go`
- Modify: `.mockery.yaml` (добавить пакет `tinvest/pkg/client/rss`)

**Interfaces:**
- Consumes: ничего из других задач.
- Produces (на это опираются Task 2 и 3):
  - `type Item struct { Title, Link, GUID string; PubDate time.Time }`
  - `type Fetcher interface { Fetch(ctx context.Context) ([]Item, error) }`
  - `func NewClient(url string) *Client` — `*Client` реализует `Fetcher`
  - мок `mocks.NewMockFetcher(t)` в `pkg/client/rss/mocks`

- [ ] **Step 1: Написать падающие тесты**

Создать `pkg/client/rss/rss_test.go`:

```go
package rss

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// Урезанная копия реальной ленты smart-lab (проверена 2026-07-15):
// RFC1123Z-даты, guid, одна запись с битой датой, одна без guid.
const feedFixture = `<?xml version="1.0" encoding="UTF-8"?>
<rss xmlns:atom="http://www.w3.org/2005/Atom" version="2.0">
<channel>
<title>Лента всех новостей акций</title>
<link>https://smart-lab.ru/news/</link>
<item>
<title>Минфин повысил план по расходам &amp; дефициту</title>
<link>https://smart-lab.ru/mobile/topic/1111111</link>
<guid>https://smart-lab.ru/mobile/topic/1111111</guid>
<pubDate>Wed, 15 Jul 2026 21:00:22 +0300</pubDate>
</item>
<item>
<title>Запись с битой датой — пропускается</title>
<link>https://smart-lab.ru/mobile/topic/2222222</link>
<guid>https://smart-lab.ru/mobile/topic/2222222</guid>
<pubDate>вчера вечером</pubDate>
</item>
<item>
<title>Запись без guid — фолбэк на link</title>
<link>https://smart-lab.ru/mobile/topic/3333333</link>
<pubDate>Wed, 15 Jul 2026 20:48:16 +0300</pubDate>
</item>
</channel>
</rss>`

func TestParse_ValidFeed(t *testing.T) {
	items, err := parse([]byte(feedFixture))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("len(items) = %d, want 2 (запись с битой датой пропущена)", len(items))
	}

	first := items[0]
	if first.Title != "Минфин повысил план по расходам & дефициту" {
		t.Errorf("Title = %q (XML-энтити должны быть раскрыты)", first.Title)
	}
	if first.GUID != "https://smart-lab.ru/mobile/topic/1111111" {
		t.Errorf("GUID = %q", first.GUID)
	}
	wantPub := time.Date(2026, 7, 15, 21, 0, 22, 0, time.FixedZone("", 3*3600))
	if !first.PubDate.Equal(wantPub) {
		t.Errorf("PubDate = %v, want %v", first.PubDate, wantPub)
	}

	if items[1].GUID != "https://smart-lab.ru/mobile/topic/3333333" {
		t.Errorf("пустой guid должен фолбэчиться на link, got %q", items[1].GUID)
	}
}

func TestParse_EmptyFeed(t *testing.T) {
	empty := `<?xml version="1.0"?><rss version="2.0"><channel><title>x</title></channel></rss>`
	items, err := parse([]byte(empty))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(items) != 0 {
		t.Fatalf("len(items) = %d, want 0", len(items))
	}
}

func TestParse_BrokenXML(t *testing.T) {
	if _, err := parse([]byte("это не xml")); err == nil {
		t.Fatal("ожидалась ошибка на невалидном XML")
	}
}

func TestFetch_OK(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if ua := r.Header.Get("User-Agent"); ua == "" || ua == "Go-http-client/1.1" {
			t.Errorf("нужен браузерный User-Agent, got %q", ua)
		}
		_, _ = w.Write([]byte(feedFixture))
	}))
	defer srv.Close()

	items, err := NewClient(srv.URL).Fetch(context.Background())
	if err != nil {
		t.Fatalf("Fetch: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("len(items) = %d, want 2", len(items))
	}
}

func TestFetch_Non200(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}))
	defer srv.Close()

	if _, err := NewClient(srv.URL).Fetch(context.Background()); err == nil {
		t.Fatal("ожидалась ошибка на статус 403")
	}
}
```

- [ ] **Step 2: Прогнать тесты — убедиться, что падают**

Run: `go test ./pkg/client/rss/ -run . -v`
Expected: compile error — `parse`, `NewClient` не определены.

- [ ] **Step 3: Реализация**

Создать `pkg/client/rss/rss.go`:

```go
// Package rss загружает и разбирает RSS 2.0-ленты (encoding/xml, без внешних
// зависимостей). Используется сервисом новостного дайджеста.
package rss

import (
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Item — одна запись ленты. GUID уникален в пределах ленты (при пустом guid
// в XML подставляется link).
type Item struct {
	Title   string
	Link    string
	GUID    string
	PubDate time.Time
}

// Fetcher загружает и разбирает RSS-ленту.
type Fetcher interface {
	Fetch(ctx context.Context) ([]Item, error)
}

const (
	fetchTimeout = 30 * time.Second
	// maxFeedSize ограничивает читаемое тело ответа: лента smart-lab ~200 КБ,
	// 5 МБ — защита от бесконечного/огромного ответа.
	maxFeedSize = 5 << 20
)

// Client — HTTP-реализация Fetcher для одного URL.
type Client struct {
	url  string
	http *http.Client
}

func NewClient(url string) *Client {
	return &Client{url: url, http: &http.Client{Timeout: fetchTimeout}}
}

func (c *Client) Fetch(ctx context.Context) ([]Item, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.url, nil)
	if err != nil {
		return nil, fmt.Errorf("rss: build request: %w", err)
	}
	// Без браузерного User-Agent часть лент (smart-lab) отвечает 403.
	req.Header.Set("User-Agent", "Mozilla/5.0 (X11; Linux x86_64)")

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("rss: fetch %s: %w", c.url, err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("rss: %s returned status %d", c.url, resp.StatusCode)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxFeedSize))
	if err != nil {
		return nil, fmt.Errorf("rss: read body: %w", err)
	}

	return parse(body)
}

type rssDoc struct {
	Channel struct {
		Items []rssItem `xml:"item"`
	} `xml:"channel"`
}

type rssItem struct {
	Title   string `xml:"title"`
	Link    string `xml:"link"`
	GUID    string `xml:"guid"`
	PubDate string `xml:"pubDate"`
}

func parse(data []byte) ([]Item, error) {
	var doc rssDoc
	if err := xml.Unmarshal(data, &doc); err != nil {
		return nil, fmt.Errorf("rss: parse feed: %w", err)
	}

	items := make([]Item, 0, len(doc.Channel.Items))
	for _, it := range doc.Channel.Items {
		pub, err := parseDate(it.PubDate)
		if err != nil {
			// Запись без валидной даты не может участвовать в окне
			// дедупликации — пропускаем её целиком.
			continue
		}
		guid := it.GUID
		if guid == "" {
			guid = it.Link
		}
		items = append(items, Item{Title: it.Title, Link: it.Link, GUID: guid, PubDate: pub})
	}

	return items, nil
}

func parseDate(s string) (time.Time, error) {
	// smart-lab/Finam/MOEX отдают RFC1123Z; RFC1123 — на всякий случай.
	for _, layout := range []string{time.RFC1123Z, time.RFC1123} {
		if t, err := time.Parse(layout, s); err == nil {
			return t, nil
		}
	}

	return time.Time{}, fmt.Errorf("rss: unsupported pubDate %q", s)
}
```

- [ ] **Step 4: Прогнать тесты — зелёные**

Run: `go test -race ./pkg/client/rss/ -v`
Expected: PASS (5 тестов).

- [ ] **Step 5: Мок для Fetcher**

В `.mockery.yaml` добавить в `packages:` (по алфавиту рядом с `tinvest/pkg/client/grpc`):

```yaml
  tinvest/pkg/client/rss:
    interfaces:
      Fetcher:
```

Run: `./bin/mage mocks`
Expected: появился `pkg/client/rss/mocks/mock_Fetcher.go`.

- [ ] **Step 6: Commit**

```bash
git add pkg/client/rss/ .mockery.yaml
git commit -m "feat(rss): RSS 2.0 fetcher client for news digest"
```

---

### Task 2: `internal/service/news` — форматирование дайджеста

**Files:**
- Create: `internal/service/news/format.go`
- Create: `internal/service/news/format_test.go`

**Interfaces:**
- Consumes: `rss.Item` из Task 1.
- Produces (на это опирается Task 3): `func formatDigest(items []rss.Item) []string` — пакетная (unexported) функция; nil при пустом входе; каждое сообщение ≤ 4096 символов; заголовок «📰 Новости рынка» только в первом.

- [ ] **Step 1: Написать падающие тесты**

Создать `internal/service/news/format_test.go`:

```go
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

func TestFormatDigest_SplitsAtMessageLimit(t *testing.T) {
	// 100 пунктов по ~100 символов — заведомо больше 4096, ждём несколько
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
```

- [ ] **Step 2: Прогнать тесты — убедиться, что падают**

Run: `go test ./internal/service/news/ -v`
Expected: compile error — `formatDigest`, `messageLimit` не определены.

- [ ] **Step 3: Реализация**

Создать `internal/service/news/format.go`:

```go
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
```

- [ ] **Step 4: Прогнать тесты — зелёные**

Run: `go test -race ./internal/service/news/ -v`
Expected: PASS (4 теста).

- [ ] **Step 5: Commit**

```bash
git add internal/service/news/
git commit -m "feat(news): digest formatting with HTML escaping and 4096 chunking"
```

---

### Task 3: `internal/service/news` — сервис Run с дедупликацией

**Files:**
- Create: `internal/service/news/news.go`
- Create: `internal/service/news/news_test.go`

**Interfaces:**
- Consumes: `rss.Fetcher`, `rss.Item`, мок `rssmocks.NewMockFetcher(t)` (Task 1); `formatDigest` (Task 2); `telegram.Client` + мок `NewMockClient` из `pkg/client/telegram/mocks` (уже в репо); `logger.Init()`, `logger.InfoContext`.
- Produces (на это опираются Task 4 и 5):
  - `func NewService(fetcher rss.Fetcher, tg telegram.Client) *Service`
  - `func (s *Service) Run(ctx context.Context) error` — одна итерация fetch→digest→send.

- [ ] **Step 1: Написать падающие тесты**

Создать `internal/service/news/news_test.go`:

```go
package news

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	"tinvest/pkg/client/rss"
	rssmocks "tinvest/pkg/client/rss/mocks"
	tgmocks "tinvest/pkg/client/telegram/mocks"
	"tinvest/pkg/logger"
)

// TestMain инициализирует пакетный логгер: Run зовёт logger.InfoContext,
// который паникует на nil-логгере (прецедент — telegram_commands).
func TestMain(m *testing.M) {
	logger.Init()
	os.Exit(m.Run())
}

var baseTime = time.Date(2026, 7, 15, 12, 0, 0, 0, time.UTC)

func item(guid string, pub time.Time) rss.Item {
	return rss.Item{Title: "n " + guid, Link: "https://example.com/" + guid, GUID: guid, PubDate: pub}
}

// newTestService фиксирует "сейчас" = baseTime: окно первого запуска —
// [baseTime-1h, ...].
func newTestService(f rss.Fetcher, tg *tgmocks.MockClient) *Service {
	s := NewService(f, tg)
	s.now = func() time.Time { return baseTime }
	return s
}

func TestRun_FirstRunPostsOnlyLastHour(t *testing.T) {
	f := rssmocks.NewMockFetcher(t)
	f.EXPECT().Fetch(context.Background()).Return([]rss.Item{
		item("fresh", baseTime.Add(-10*time.Minute)),
		item("stale", baseTime.Add(-2*time.Hour)),
	}, nil).Once()

	tg := tgmocks.NewMockClient(t)
	var sent string
	tg.EXPECT().SendMessage(mockAnyString(&sent)).Return(nil).Once()

	if err := newTestService(f, tg).Run(context.Background()); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if !strings.Contains(sent, "fresh") || strings.Contains(sent, "stale") {
		t.Errorf("в дайджесте только свежие записи, got %q", sent)
	}
}

func TestRun_SecondRunSkipsAlreadySent(t *testing.T) {
	first := item("a", baseTime.Add(-10*time.Minute))
	second := item("b", baseTime.Add(-5*time.Minute))

	f := rssmocks.NewMockFetcher(t)
	f.EXPECT().Fetch(context.Background()).Return([]rss.Item{first}, nil).Once()
	tg := tgmocks.NewMockClient(t)
	var sent string
	tg.EXPECT().SendMessage(mockAnyString(&sent)).Return(nil).Once()

	svc := newTestService(f, tg)
	if err := svc.Run(context.Background()); err != nil {
		t.Fatalf("Run 1: %v", err)
	}

	// Второй запуск: лента отдаёт old+new — уйти должна только new.
	f.EXPECT().Fetch(context.Background()).Return([]rss.Item{second, first}, nil).Once()
	tg.EXPECT().SendMessage(mockAnyString(&sent)).Return(nil).Once()
	if err := svc.Run(context.Background()); err != nil {
		t.Fatalf("Run 2: %v", err)
	}
	if strings.Contains(sent, `>n a<`) || !strings.Contains(sent, `>n b<`) {
		t.Errorf("повтор уже отправленной записи, got %q", sent)
	}
}

func TestRun_BoundaryGUIDDedup(t *testing.T) {
	// Две записи с одинаковым PubDate приходят в разных итерациях: вторая
	// не должна потеряться (PubDate == lastSeen) и первая не должна
	// повториться (GUID уже отправлен).
	pub := baseTime.Add(-10 * time.Minute)
	a, b := item("a", pub), item("b", pub)

	f := rssmocks.NewMockFetcher(t)
	f.EXPECT().Fetch(context.Background()).Return([]rss.Item{a}, nil).Once()
	tg := tgmocks.NewMockClient(t)
	var sent string
	tg.EXPECT().SendMessage(mockAnyString(&sent)).Return(nil).Once()

	svc := newTestService(f, tg)
	if err := svc.Run(context.Background()); err != nil {
		t.Fatalf("Run 1: %v", err)
	}

	f.EXPECT().Fetch(context.Background()).Return([]rss.Item{b, a}, nil).Once()
	tg.EXPECT().SendMessage(mockAnyString(&sent)).Return(nil).Once()
	if err := svc.Run(context.Background()); err != nil {
		t.Fatalf("Run 2: %v", err)
	}
	if strings.Contains(sent, `>n a<`) || !strings.Contains(sent, `>n b<`) {
		t.Errorf("граница по GUID отработала неверно, got %q", sent)
	}
}

func TestRun_ChronologicalOrder(t *testing.T) {
	// RSS отдаёт новые сверху — в дайджесте порядок хронологический.
	f := rssmocks.NewMockFetcher(t)
	f.EXPECT().Fetch(context.Background()).Return([]rss.Item{
		item("newer", baseTime.Add(-5*time.Minute)),
		item("older", baseTime.Add(-30*time.Minute)),
	}, nil).Once()
	tg := tgmocks.NewMockClient(t)
	var sent string
	tg.EXPECT().SendMessage(mockAnyString(&sent)).Return(nil).Once()

	if err := newTestService(f, tg).Run(context.Background()); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if strings.Index(sent, "older") > strings.Index(sent, "newer") {
		t.Errorf("ожидался хронологический порядок, got %q", sent)
	}
}

func TestRun_NoNewItems_NoSend(t *testing.T) {
	f := rssmocks.NewMockFetcher(t)
	f.EXPECT().Fetch(context.Background()).Return([]rss.Item{
		item("stale", baseTime.Add(-2*time.Hour)),
	}, nil).Once()
	tg := tgmocks.NewMockClient(t) // без EXPECT: любой SendMessage завалит тест

	if err := newTestService(f, tg).Run(context.Background()); err != nil {
		t.Fatalf("Run: %v", err)
	}
}

func TestRun_FetchError_ReturnsError(t *testing.T) {
	f := rssmocks.NewMockFetcher(t)
	f.EXPECT().Fetch(context.Background()).Return(nil, errors.New("boom")).Once()
	tg := tgmocks.NewMockClient(t)

	if err := newTestService(f, tg).Run(context.Background()); err == nil {
		t.Fatal("ожидалась ошибка fetch")
	}
}

func TestRun_SendError_DoesNotAdvanceWindow(t *testing.T) {
	it := item("a", baseTime.Add(-10*time.Minute))
	f := rssmocks.NewMockFetcher(t)
	f.EXPECT().Fetch(context.Background()).Return([]rss.Item{it}, nil).Twice()

	tg := tgmocks.NewMockClient(t)
	var sent string
	tg.EXPECT().SendMessage(mockAnyString(&sent)).Return(errors.New("tg down")).Once()

	svc := newTestService(f, tg)
	if err := svc.Run(context.Background()); err == nil {
		t.Fatal("ожидалась ошибка отправки")
	}

	// Повторный запуск шлёт ту же запись заново — окно не сдвинулось.
	tg.EXPECT().SendMessage(mockAnyString(&sent)).Return(nil).Once()
	if err := svc.Run(context.Background()); err != nil {
		t.Fatalf("Run 2: %v", err)
	}
	if !strings.Contains(sent, `>n a<`) {
		t.Errorf("запись потеряна после ошибки отправки, got %q", sent)
	}
}
```

И вспомогательный матчер в том же файле (mockery expecter принимает
`mock.MatchedBy` через интерфейсный аргумент):

```go
// mockAnyString принимает любое строковое сообщение и запоминает последнее
// в *dst — так проверяем содержимое дайджеста.
func mockAnyString(dst *string) any {
	return mock.MatchedBy(func(s string) bool {
		*dst = s
		return true
	})
}
```

(добавить `"github.com/stretchr/testify/mock"` в импорты).

- [ ] **Step 2: Прогнать тесты — убедиться, что падают**

Run: `go test ./internal/service/news/ -v`
Expected: compile error — `NewService`, `Service.now` не определены.

- [ ] **Step 3: Реализация**

Создать `internal/service/news/news.go`:

```go
// Package news раз в запуск публикует дайджест свежих записей RSS-ленты
// новостей фондового рынка в Telegram-тему «Новости».
// Дизайн: docs/superpowers/specs/2026-07-15-news-digest-design.md.
package news

import (
	"context"
	"fmt"
	"log/slog"
	"sort"
	"time"

	"tinvest/pkg/client/rss"
	"tinvest/pkg/client/telegram"
	"tinvest/pkg/logger"
)

// startupLookback — окно первого запуска после старта процесса: постим только
// новости за последний час, а не всю ленту.
const startupLookback = time.Hour

// Service хранит состояние дедупликации в памяти процесса: после рестарта
// возможен редкий повтор записей на границе окна — принято осознанно,
// хранилище ради этого не заводим.
type Service struct {
	fetcher rss.Fetcher
	tg      telegram.Client
	now     func() time.Time // подменяется в тестах

	// lastSeen — максимальный PubDate уже отправленных записей;
	// boundaryGUIDs — GUID отправленных записей с PubDate == lastSeen
	// (отсекают дубли ровно на границе окна).
	lastSeen      time.Time
	boundaryGUIDs map[string]struct{}
}

// NewService строит сервис. tg должен быть уже привязан к теме «Новости»
// (telegram.NewTopicSender).
func NewService(fetcher rss.Fetcher, tg telegram.Client) *Service {
	return &Service{
		fetcher:       fetcher,
		tg:            tg,
		now:           time.Now,
		boundaryGUIDs: map[string]struct{}{},
	}
}

// Run — одна итерация: fetch → отбор новых записей → отправка дайджеста →
// сдвиг окна. При любой ошибке окно не сдвигается: следующий запуск повторит
// невышедшие записи.
func (s *Service) Run(ctx context.Context) error {
	items, err := s.fetcher.Fetch(ctx)
	if err != nil {
		return fmt.Errorf("news: fetch feed: %w", err)
	}

	if s.lastSeen.IsZero() {
		s.lastSeen = s.now().Add(-startupLookback)
	}

	fresh := s.selectNew(items)
	if len(fresh) == 0 {
		logger.InfoContext(ctx, "news: новых записей нет")

		return nil
	}

	// RSS отдаёт новые сверху — в дайджесте порядок хронологический.
	sort.Slice(fresh, func(i, j int) bool { return fresh[i].PubDate.Before(fresh[j].PubDate) })

	for _, msg := range formatDigest(fresh) {
		if err := s.tg.SendMessage(msg); err != nil {
			return fmt.Errorf("news: send digest: %w", err)
		}
	}

	s.advance(fresh)
	logger.InfoContext(ctx, "news: дайджест отправлен", slog.Int("items", len(fresh)))

	return nil
}

func (s *Service) selectNew(items []rss.Item) []rss.Item {
	var fresh []rss.Item
	for _, it := range items {
		switch {
		case it.PubDate.After(s.lastSeen):
			fresh = append(fresh, it)
		case it.PubDate.Equal(s.lastSeen):
			if _, sent := s.boundaryGUIDs[it.GUID]; !sent {
				fresh = append(fresh, it)
			}
		}
	}

	return fresh
}

// advance сдвигает окно на максимальный PubDate отправленных записей и
// перезаполняет множество GUID на новой границе.
func (s *Service) advance(sent []rss.Item) {
	maxPub := s.lastSeen
	for _, it := range sent {
		if it.PubDate.After(maxPub) {
			maxPub = it.PubDate
		}
	}
	if maxPub.After(s.lastSeen) {
		s.lastSeen = maxPub
		s.boundaryGUIDs = map[string]struct{}{}
	}
	for _, it := range sent {
		if it.PubDate.Equal(s.lastSeen) {
			s.boundaryGUIDs[it.GUID] = struct{}{}
		}
	}
}
```

- [ ] **Step 4: Прогнать тесты — зелёные**

Run: `go test -race ./internal/service/news/ -v`
Expected: PASS (7 тестов Run + 4 теста format).

- [ ] **Step 5: Commit**

```bash
git add internal/service/news/
git commit -m "feat(news): digest service with in-memory PubDate/GUID dedupe"
```

---

### Task 4: Конфиг и wiring в service_provider

**Files:**
- Create: `internal/config/news.go`
- Create: `internal/config/news_test.go`
- Modify: `internal/config/config.go` (поле `News`)
- Modify: `internal/config/telegram_client.go` (поле `TopicNews`)
- Modify: `internal/app/init_config.go` (строка `News: config.NewNewsConfig(),`)
- Modify: `internal/service_provider/client.go` (геттер `GetNewsSender`)
- Modify: `internal/service_provider/service.go` (поле + геттер `GetNewsService`)
- Modify: `env/local.env.example`, `env/prod.env.example`

**Interfaces:**
- Consumes: `news.NewService` (Task 3), `rss.NewClient` (Task 1), существующие `topicSender(threadID int, name string)`, `NewTopicSender`.
- Produces (на это опирается Task 5): `(s *ServiceProvider) GetNewsService() (*news.Service, error)`.

- [ ] **Step 1: Написать падающий тест дефолта конфига**

Создать `internal/config/news_test.go`:

```go
package config

import "testing"

func TestNewNewsConfig_DefaultFeedURL(t *testing.T) {
	cfg := NewNewsConfig()
	if cfg.FeedURL != "https://smart-lab.ru/news/rss/" {
		t.Fatalf("FeedURL = %q, want smart-lab default", cfg.FeedURL)
	}
}
```

Run: `go test ./internal/config/ -run TestNewNewsConfig -v`
Expected: compile error — `NewNewsConfig` не определён.

- [ ] **Step 2: Конфиг**

Создать `internal/config/news.go`:

```go
package config

// NewsConfig — настройки новостного дайджеста.
type NewsConfig struct {
	// FeedURL — RSS-лента новостей рынка; дефолт — «Лента всех новостей
	// акций» smart-lab (агрегатор Интерфакс/Reuters/эмитенты).
	FeedURL string `config:"NEWS_FEED_URL"`
}

func NewNewsConfig() *NewsConfig {
	return &NewsConfig{FeedURL: "https://smart-lab.ru/news/rss/"}
}
```

В `internal/config/config.go` добавить поле в `Config` после `Reversion`:

```go
	Reversion      *ReversionConfig
	News           *NewsConfig
```

В `internal/config/telegram_client.go` добавить поле после `TopicReversion`:

```go
	TopicReversion int    `config:"TELEGRAM_TOPIC_REVERSION"`
	TopicNews      int    `config:"TELEGRAM_TOPIC_NEWS"`
```

В `internal/app/init_config.go` в литерале `cfg := &config.Config{...}` добавить после `Reversion: config.NewReversionConfig(),`:

```go
		News:           config.NewNewsConfig(),
```

Run: `go test ./internal/config/ -run TestNewNewsConfig -v`
Expected: PASS.

- [ ] **Step 3: Wiring в service_provider**

В `internal/service_provider/client.go` добавить после `GetReversionSender`:

```go
func (s *ServiceProvider) GetNewsSender() (telegram.Client, error) {
	return s.topicSender(s.appConfig.TelegramClient.TopicNews, "news")
}
```

В `internal/service_provider/service.go`:

в импорты добавить:

```go
	"tinvest/internal/service/news"
	"tinvest/pkg/client/rss"
```

в структуру `service` добавить поле:

```go
	newsService           *news.Service
```

и геттер в конец файла (паттерн `GetTelegramCommands` — с возвратом ошибки):

```go
func (s *ServiceProvider) GetNewsService() (*news.Service, error) {
	if serviceProvider.service.newsService != nil {
		return serviceProvider.service.newsService, nil
	}
	sender, err := s.GetNewsSender()
	if err != nil {
		return nil, err
	}
	serviceProvider.service.newsService = news.NewService(
		rss.NewClient(s.appConfig.News.FeedURL),
		sender,
	)

	return serviceProvider.service.newsService, nil
}
```

- [ ] **Step 4: env-примеры**

В `env/local.env.example` и `env/prod.env.example` после строки `TELEGRAM_TOPIC_REVERSION=` добавить:

```
TELEGRAM_TOPIC_NEWS=
# NEWS_FEED_URL можно не задавать: дефолт — https://smart-lab.ru/news/rss/
# NEWS_FEED_URL=
```

Важно: `NEWS_FEED_URL` в примерах закомментирован — незакомментированная
пустая строка перекрыла бы дефолт пустым значением (confita пишет и пустые
env-переменные).

- [ ] **Step 5: Сборка и тесты**

Run: `go build ./internal/... ./pkg/... ./cmd/... && go test -race ./internal/config/ ./internal/service_provider/ ./internal/service/news/`
Expected: сборка ОК, тесты PASS.

- [ ] **Step 6: Commit**

```bash
git add internal/config/ internal/app/init_config.go internal/service_provider/ env/local.env.example env/prod.env.example
git commit -m "feat(news): config (TELEGRAM_TOPIC_NEWS, NEWS_FEED_URL) and DI wiring"
```

---

### Task 5: Cron-воркер, запуск в app, документация

**Files:**
- Create: `internal/service/news/scheduler/scheduler.go`
- Create: `internal/service/news/scheduler/scheduler_test.go`
- Modify: `internal/app/app.go` (`runProd` — cron-воркер, `runDev` — one-shot)
- Modify: `CLAUDE.md` (упоминание новостного дайджеста)

**Interfaces:**
- Consumes: `GetNewsService() (*news.Service, error)` (Task 4); `news.Service.Run(ctx) error` (Task 3); `pkg/scheduler.NewScheduler()`.
- Produces: `scheduler.NewSchedulerService(service Runner) *SchedulerService` с методом `Run(ctx context.Context, cronExpr string) error` (блокируется до отмены ctx).

- [ ] **Step 1: Написать падающий тест**

Создать `internal/service/news/scheduler/scheduler_test.go` (по образцу
`reversion/live/scheduler/scheduler_test.go`, но с локальным стабом вместо
mockery — интерфейс `Runner` объявлен в этом же пакете):

```go
package scheduler

import (
	"context"
	"os"
	"testing"
	"time"

	"tinvest/pkg/logger"
)

func TestMain(m *testing.M) {
	logger.Init()
	os.Exit(m.Run())
}

type stubRunner struct{}

func (stubRunner) Run(context.Context) error { return nil }

func TestSchedulerService_ReturnsOnContextCancel(t *testing.T) {
	sch := NewSchedulerService(stubRunner{})

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- sch.Run(ctx, "5 * * * *") }()

	cancel()
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("Run returned %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Run did not return after context cancel")
	}
}

func TestSchedulerService_BadCronExpr(t *testing.T) {
	if err := NewSchedulerService(stubRunner{}).Run(context.Background(), "не крон"); err == nil {
		t.Fatal("ожидалась ошибка на невалидном cron-выражении")
	}
}
```

Run: `go test ./internal/service/news/scheduler/ -v`
Expected: compile error — пакет не существует.

- [ ] **Step 2: Реализация**

Создать `internal/service/news/scheduler/scheduler.go`:

```go
// Package scheduler запускает новостной дайджест по cron-расписанию.
package scheduler

import (
	"context"
	"time"

	"tinvest/pkg/logger"
	"tinvest/pkg/scheduler"
)

// Runner — одна итерация дайджеста (news.Service.Run).
type Runner interface {
	Run(ctx context.Context) error
}

type SchedulerService struct {
	sh      scheduler.Scheduler
	service Runner
}

// NewSchedulerService оборачивает Runner: Run регистрирует cron-job и
// блокируется до отмены контекста.
func NewSchedulerService(service Runner) *SchedulerService {
	return &SchedulerService{
		sh:      scheduler.NewScheduler(),
		service: service,
	}
}

func (s *SchedulerService) Run(ctx context.Context, cronExpr string) error {
	jobTicker := time.NewTicker(time.Hour)
	defer jobTicker.Stop()

	err := s.sh.AddJob(cronExpr, func() {
		// Паника в итерации не должна ронять процесс (прецедент —
		// telegram_commands.runExclusive).
		defer func() {
			if r := recover(); r != nil {
				logger.ErrorContext(ctx, "паника в воркере News", r)
			}
		}()
		logger.InfoContext(ctx, "Воркер News начал работу")
		if err := s.service.Run(ctx); err != nil {
			logger.ErrorContext(ctx, "Ошибка в ходе работы job News", err.Error())
		}
	})
	if err != nil {
		return err
	}

	s.sh.Start()
	defer s.sh.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-jobTicker.C:
			logger.InfoContext(ctx, "Worker News is running")
		}
	}
}
```

Run: `go test -race ./internal/service/news/scheduler/ -v`
Expected: PASS (2 теста).

- [ ] **Step 3: Подключить в app.go**

В `internal/app/app.go`:

в импорты добавить:

```go
	newsscheduler "tinvest/internal/service/news/scheduler"
```

В `runProd` заменить `wg.Add(5)` на `wg.Add(6)` и добавить горутину (после
горутины telegram commands, до Golden X):

```go
	go func() {
		defer wg.Done()
		svc, err := a.sp.GetNewsService()
		if err != nil {
			logger.ErrorContext(ctx, "news service init failed", err.Error())
			return
		}
		// 5-я минута часа: дайджест собирает полный прошедший час.
		if err := newsscheduler.NewSchedulerService(svc).Run(ctx, "5 * * * *"); err != nil {
			logger.ErrorContext(ctx, "Error in worker News digest", err.Error())
		}
	}()
```

В `runDev` заменить `wg.Add(1)` на `wg.Add(2)` и добавить one-shot горутину
(после горутины listener):

```go
	go func() {
		defer wg.Done()
		svc, err := a.sp.GetNewsService()
		if err != nil {
			logger.ErrorContext(ctx, "news service init failed", err.Error())
			return
		}
		// Dev: один немедленный прогон для ручной проверки дайджеста.
		if err := svc.Run(ctx); err != nil {
			logger.ErrorContext(ctx, "news digest run failed", err.Error())
		}
	}()
```

Run: `go build ./internal/... ./pkg/... ./cmd/...`
Expected: сборка ОК.

- [ ] **Step 4: Обновить CLAUDE.md**

В `CLAUDE.md`:

1. В разделе `## Layout`, в списке `internal/service` после строки
   `- notification/purchase_shares` добавить:

```markdown
  - `news/` — ежечасный дайджест новостей рынка (RSS smart-lab) в тему Telegram; `news/scheduler/` — cron-обёртка
```

2. В `## Development Notes` в последнем буллете про темы форума дописать в
   конец (перед ссылкой на spec):

```markdown
push-дайджест новостей рынка приходит в тему «Новости» раз в час (`TELEGRAM_TOPIC_NEWS`, источник `NEWS_FEED_URL`, дефолт — RSS smart-lab); см. `docs/superpowers/specs/2026-07-15-news-digest-design.md`
```

3. В `## Layout`, в списке `pkg` дописать `rss` в перечисление `client`
   (`client` (grpc, telegram, rss)).

- [ ] **Step 5: Полный гейт**

Run: `./bin/mage ci`
Expected: lint OK, все тесты PASS, mock-drift check OK.

- [ ] **Step 6: Commit**

```bash
git add internal/service/news/scheduler/ internal/app/app.go CLAUDE.md
git commit -m "feat(news): hourly cron worker in prod, one-shot dev run"
```

---

## Ручные шаги после реализации (пользователь)

1. Создать тему «Новости» в супергруппе (бот уже админ).
2. Узнать `message_thread_id` темы (ссылка на любое её сообщение:
   `https://t.me/c/<chat>/<thread_id>/<msg>`), прописать
   `TELEGRAM_TOPIC_NEWS=<thread_id>` в `env/local.env` и prod-env.
3. Проверка в dev: `go run ./cmd/main` — при старте придёт дайджест за
   последний час в тему «Новости».
