package telegram

// Белоящичные тесты Bot.SendMessageToTopic: поля Bot неэкспортируемые, а
// InitTelegramBot не принимает опций, поэтому тестовый Bot собирается здесь
// напрямую поверх tgbot.New с WithServerURL/WithSkipGetMe.

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	tgbot "github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

// sentRequest — распарсенное тело одного запроса sendMessage.
type sentRequest struct {
	threadID string // form-значение message_thread_id ("" если поле не слалось)
	text     string
}

// fakeTelegram поднимает httptest-сервер, который отвечает на sendMessage по
// правилу respond и потокобезопасно копит распарсенные запросы.
type fakeTelegram struct {
	mu       sync.Mutex
	requests []sentRequest
	respond  func(req sentRequest) (status int, body string)
}

func (f *fakeTelegram) handler(w http.ResponseWriter, r *http.Request) {
	if !strings.HasSuffix(r.URL.Path, "/sendMessage") {
		http.Error(w, `{"ok":false,"error_code":404,"description":"unexpected method"}`, http.StatusNotFound)
		return
	}
	if err := r.ParseMultipartForm(1 << 20); err != nil {
		http.Error(w, `{"ok":false,"error_code":400,"description":"bad form"}`, http.StatusBadRequest)
		return
	}
	req := sentRequest{
		threadID: r.FormValue("message_thread_id"),
		text:     r.FormValue("text"),
	}

	f.mu.Lock()
	f.requests = append(f.requests, req)
	f.mu.Unlock()

	status, body := f.respond(req)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, _ = w.Write([]byte(body))
}

func (f *fakeTelegram) sent() []sentRequest {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]sentRequest, len(f.requests))
	copy(out, f.requests)
	return out
}

const (
	okBody   = `{"ok":true,"result":{"message_id":1}}`
	failBody = `{"ok":false,"error_code":400,"description":"message thread not found"}`
)

func newTestBot(t *testing.T, fake *fakeTelegram) *Bot {
	t.Helper()

	ts := httptest.NewServer(http.HandlerFunc(fake.handler))
	t.Cleanup(ts.Close)

	api, err := tgbot.New("test-token", tgbot.WithServerURL(ts.URL), tgbot.WithSkipGetMe())
	if err != nil {
		t.Fatalf("tgbot.New: %v", err)
	}

	return &Bot{api: api, defaultChatID: 1}
}

func TestBotSendMessageToTopicSuccess(t *testing.T) {
	fake := &fakeTelegram{respond: func(sentRequest) (int, string) {
		return http.StatusOK, okBody
	}}
	b := newTestBot(t, fake)

	if err := b.SendMessageToTopic(-1001234, 42, "hello"); err != nil {
		t.Fatalf("SendMessageToTopic: %v", err)
	}

	sent := fake.sent()
	if len(sent) != 1 {
		t.Fatalf("expected 1 request, got %d: %+v", len(sent), sent)
	}
	if sent[0].threadID != "42" {
		t.Errorf("message_thread_id = %q, want %q", sent[0].threadID, "42")
	}
	if sent[0].text != "hello" {
		t.Errorf("text = %q, want %q", sent[0].text, "hello")
	}
}

func TestBotSendMessageToTopicFallsBackToGeneral(t *testing.T) {
	fake := &fakeTelegram{respond: func(req sentRequest) (int, string) {
		if req.threadID != "" {
			return http.StatusBadRequest, failBody
		}
		return http.StatusOK, okBody
	}}
	b := newTestBot(t, fake)

	if err := b.SendMessageToTopic(-1001234, 42, "hello"); err != nil {
		t.Fatalf("SendMessageToTopic: fallback should swallow topic error, got %v", err)
	}

	sent := fake.sent()
	if len(sent) != 2 {
		t.Fatalf("expected 2 requests (topic + fallback), got %d: %+v", len(sent), sent)
	}
	if sent[1].threadID != "" {
		t.Errorf("fallback request has message_thread_id = %q, want none", sent[1].threadID)
	}
	if !strings.HasPrefix(sent[1].text, "⚠️ тема 42") {
		t.Errorf("fallback text = %q, want prefix %q", sent[1].text, "⚠️ тема 42")
	}
	if !strings.HasSuffix(sent[1].text, "hello") {
		t.Errorf("fallback text = %q, want original message preserved", sent[1].text)
	}
}

func TestBotSendMessageToTopicFallbackAlsoFails(t *testing.T) {
	fake := &fakeTelegram{respond: func(sentRequest) (int, string) {
		return http.StatusBadRequest, failBody
	}}
	b := newTestBot(t, fake)

	err := b.SendMessageToTopic(-1001234, 42, "hello")
	if err == nil {
		t.Fatal("expected error when both topic and fallback sends fail")
	}
	if !strings.Contains(err.Error(), "fallback to General failed") {
		t.Errorf("error = %q, want it to mention the failed fallback", err)
	}

	if got := len(fake.sent()); got != 2 {
		t.Fatalf("expected 2 requests, got %d", got)
	}
}

// TestPollingSurvivesHangingLongPoll: Telegram держит getUpdates открытым почти
// весь pollTimeout, поэтому HTTP-клиенту нужен запас сверху. Если его нет,
// клиент рвёт соединение раньше ответа и апдейты не доходят вообще.
func TestPollingSurvivesHangingLongPoll(t *testing.T) {
	const (
		poll        = 200 * time.Millisecond
		httpTimeout = 3 * time.Second
		serverDelay = 600 * time.Millisecond // дольше poll: сервер «висит» на long poll
	)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasSuffix(r.URL.Path, "/getUpdates") {
			http.Error(w, `{"ok":false,"error_code":404,"description":"unexpected method"}`, http.StatusNotFound)
			return
		}
		select {
		case <-time.After(serverDelay):
		case <-r.Context().Done():
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true,"result":[{"update_id":1,"message":{"message_id":1,"date":1,` +
			`"chat":{"id":1,"type":"private"},"text":"/ping"}}]}`))
	}))
	t.Cleanup(ts.Close)

	opts := append(pollingOptions(poll, httpTimeout), tgbot.WithServerURL(ts.URL), tgbot.WithSkipGetMe())
	api, err := tgbot.New("test-token", opts...)
	if err != nil {
		t.Fatalf("tgbot.New: %v", err)
	}

	delivered := make(chan string, 1)
	api.RegisterHandlerMatchFunc(
		func(u *models.Update) bool { return u.Message != nil },
		func(_ context.Context, _ *tgbot.Bot, u *models.Update) {
			select {
			case delivered <- u.Message.Text:
			default:
			}
		},
	)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go api.Start(ctx)

	select {
	case got := <-delivered:
		if got != "/ping" {
			t.Errorf("delivered text = %q, want %q", got, "/ping")
		}
	case <-time.After(5 * time.Second):
		t.Fatal("апдейт не доставлен: HTTP-клиент обрывает long poll раньше, чем сервер отвечает")
	}
}

func TestPollingTimeoutsHaveHeadroom(t *testing.T) {
	if headroom := pollHTTPTimeout - pollTimeout; headroom < 10*time.Second {
		t.Errorf("запас HTTP-таймаута над pollTimeout = %v, want >= 10s: %v против %v",
			headroom, pollHTTPTimeout, pollTimeout)
	}
}

func TestBotSendMessageToChatDoesNotFallBack(t *testing.T) {
	fake := &fakeTelegram{respond: func(sentRequest) (int, string) {
		return http.StatusBadRequest, failBody
	}}
	b := newTestBot(t, fake)

	err := b.SendMessageToChat(-1001234, "hello")
	if err == nil {
		t.Fatal("expected error for threadID=0 send")
	}

	if got := len(fake.sent()); got != 1 {
		t.Fatalf("expected 1 request (no fallback for General), got %d", got)
	}
}
