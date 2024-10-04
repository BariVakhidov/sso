package logger

import (
	"log/slog"
	"os"
)

const (
	envLocal = "local"
	envDev   = "development"
	envProd  = "production"
)

type Logger struct {
	Log *slog.Logger
}

func New(env string) *Logger {
	var logger *slog.Logger

	switch env {
	case envLocal:
		logger = slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	case envDev:
		logger = slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	case envProd:
		logger = slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	}

	return &Logger{Log: logger}
}
