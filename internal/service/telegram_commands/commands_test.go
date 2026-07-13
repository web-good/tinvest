package telegram_commands

import (
	"context"
	"sync"
	"testing"
	"time"

	"tinvest/pkg/client/telegram"
)

type fakeSender struct {
	mu   sync.Mutex
	msgs []string
}

func (f *fakeSender) SendMessage(msg string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.msgs = append(f.msgs, msg)
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
