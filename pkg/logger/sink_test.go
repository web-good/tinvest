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
