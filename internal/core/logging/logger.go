package logging

import (
	"log/slog"
	"os"
)

var Logger *slog.Logger

func Init() {
	opts := &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}
	handler := slog.NewJSONHandler(os.Stdout, opts)
	Logger = slog.New(handler)
}

func Info(msg string, args ...any) {
	if Logger == nil {
		Init()
	}
	Logger.Info(msg, args...)
}

func Error(msg string, args ...any) {
	if Logger == nil {
		Init()
	}
	Logger.Error(msg, args...)
}

func Debug(msg string, args ...any) {
	if Logger == nil {
		Init()
	}
	Logger.Debug(msg, args...)
}
