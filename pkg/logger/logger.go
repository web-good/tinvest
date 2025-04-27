package logger

import (
	"context"
	"log/slog"
	"os"
)

type OptionsStrategy interface {
	Options() *slog.Handler
}

type any = interface{}

var logger *slog.Logger

func Init() {
	logger = slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
}

func Info(msg string, fields ...any) {
	logger.Info(msg, fields...)
}

func InfoContext(ctx context.Context, msg string, fields ...any) {
	logger.InfoContext(ctx, msg, fields...)
}

func ErrorContext(ctx context.Context, msg string, fields ...any) {
	logger.ErrorContext(ctx, msg, fields...)
}

func DebugContext(ctx context.Context, msg string, fields ...any) {
	logger.DebugContext(ctx, msg, fields...)
}
