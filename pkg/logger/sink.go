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
