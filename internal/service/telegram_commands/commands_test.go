package telegram_commands

import (
	"context"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"tinvest/pkg/client/telegram"
	"tinvest/pkg/logger"
)

// TestMain инициализирует пакетный логгер: recover-путь runExclusive зовёт
// logger.ErrorContext, который паникует на nil-логгере (прецедент —
// reversion/live/service_test.go).
func TestMain(m *testing.M) {
	logger.Init()
	os.Exit(m.Run())
}

type fakeSender struct {
	mu   sync.Mutex
	msgs []string
	sent chan string // опционально: сигнал о каждом отправленном сообщении
}

func (f *fakeSender) SendMessage(msg string) error {
	f.mu.Lock()
	f.msgs = append(f.msgs, msg)
	f.mu.Unlock()
	if f.sent != nil {
		f.sent <- msg
	}
	return nil
}
func (f *fakeSender) SendMessageToChat(int64, string) error       { return nil }
func (f *fakeSender) SendMessageToTopic(int64, int, string) error { return nil }

func (f *fakeSender) count() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.msgs)
}

type fakeYield struct {
	calls   chan telegram.Client
	release chan struct{}
}

func (f *fakeYield) PortfolioYieldYTD(_ context.Context, tg telegram.Client) error {
	f.calls <- tg
	if f.release != nil {
		<-f.release
	}
	return nil
}

type fakeAnalyze struct{ called bool }

func (f *fakeAnalyze) BondsPortfolio(context.Context, telegram.Client) error {
	f.called = true
	return nil
}

type fakeBonds struct{ called bool }

func (f *fakeBonds) Trade(context.Context, telegram.Client) error {
	f.called = true
	return nil
}

func newTestCommands(y *fakeYield, sender *fakeSender) *Commands {
	factory := func(chatID int64, threadID int) telegram.Client { return sender }
	return New(&fakeAnalyze{}, y, &fakeBonds{}, factory, []int64{111})
}

func TestHandleIgnoresUnknownUser(t *testing.T) {
	y := &fakeYield{calls: make(chan telegram.Client, 1)}
	sender := &fakeSender{}
	c := newTestCommands(y, sender)

	if c.Handle(context.Background(), "/yield", 1, 2, 999) {
		t.Fatal("чужой userID должен игнорироваться")
	}
	if sender.count() != 0 {
		t.Fatal("чужому пользователю не должно уходить сообщений")
	}
}

func TestHandleDispatchesYield(t *testing.T) {
	y := &fakeYield{calls: make(chan telegram.Client, 1)}
	sender := &fakeSender{}
	c := newTestCommands(y, sender)

	if !c.Handle(context.Background(), "/yield", 1, 2, 111) {
		t.Fatal("команда авторизованного пользователя должна обрабатываться")
	}
	select {
	case <-y.calls:
	case <-time.After(2 * time.Second):
		t.Fatal("PortfolioYieldYTD не вызван")
	}
	if sender.count() == 0 {
		t.Fatal("нет ack-сообщения «Считаю»")
	}
}

func TestHandleRejectsConcurrentDuplicate(t *testing.T) {
	y := &fakeYield{calls: make(chan telegram.Client, 1), release: make(chan struct{})}
	sender := &fakeSender{}
	c := newTestCommands(y, sender)

	c.Handle(context.Background(), "/yield", 1, 2, 111)
	<-y.calls // первый запуск внутри расчёта

	before := sender.count()
	c.Handle(context.Background(), "/yield", 1, 2, 111)
	if sender.count() != before+1 {
		t.Fatal("повторная команда должна получить ответ «уже выполняется»")
	}
	close(y.release)

	select {
	case y.calls <- nil: // канал свободен — второго вызова сервиса не было
	default:
	}
}

func TestHandleUnknownCommandIgnored(t *testing.T) {
	y := &fakeYield{calls: make(chan telegram.Client, 1)}
	sender := &fakeSender{}
	c := newTestCommands(y, sender)

	if c.Handle(context.Background(), "/unknown", 1, 2, 111) {
		t.Fatal("неизвестная команда должна игнорироваться")
	}
}

type panickyYield struct{}

func (panickyYield) PortfolioYieldYTD(context.Context, telegram.Client) error {
	panic("boom: unexpected API response")
}

func waitMsg(t *testing.T, ch chan string) string {
	t.Helper()
	select {
	case m := <-ch:
		return m
	case <-time.After(2 * time.Second):
		t.Fatal("сообщение не пришло за 2s")
		return ""
	}
}

func TestHandleRecoversFromPanic(t *testing.T) {
	sender := &fakeSender{sent: make(chan string, 8)}
	factory := func(int64, int) telegram.Client { return sender }
	c := New(&fakeAnalyze{}, panickyYield{}, &fakeBonds{}, factory, []int64{111})

	if !c.Handle(context.Background(), "/yield", 1, 2, 111) {
		t.Fatal("команда авторизованного пользователя должна обрабатываться")
	}
	_ = waitMsg(t, sender.sent) // ack «Считаю»
	errMsg := waitMsg(t, sender.sent)
	if !strings.HasPrefix(errMsg, "❌") {
		t.Fatalf("после паники пользователю должно уйти сообщение «❌…», got %q", errMsg)
	}

	// Паникующий запуск завершён (❌ шлётся после снятия running-флага) —
	// команда должна быть снова доступна, без «уже выполняется».
	if !c.Handle(context.Background(), "/yield", 1, 2, 111) {
		t.Fatal("после паники команда должна обрабатываться снова")
	}
	ack2 := waitMsg(t, sender.sent)
	if strings.Contains(ack2, "уже выполняется") {
		t.Fatalf("повторный запуск после паники получил «уже выполняется»: %q", ack2)
	}
	_ = waitMsg(t, sender.sent) // «❌» второго запуска: горутина дожита до конца
}

func TestHandleStripsBotMention(t *testing.T) {
	y := &fakeYield{calls: make(chan telegram.Client, 1)}
	sender := &fakeSender{}
	c := newTestCommands(y, sender)

	if !c.Handle(context.Background(), "/yield@MyTradingBot", 1, 2, 111) {
		t.Fatal("команда с @упоминанием бота должна обрабатываться")
	}
	select {
	case <-y.calls:
	case <-time.After(2 * time.Second):
		t.Fatal("PortfolioYieldYTD не вызван")
	}
}
