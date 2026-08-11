package errorlog

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"tinvest/pkg/logger"
)

var errFakeSend = errors.New("telegram недоступен")

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
