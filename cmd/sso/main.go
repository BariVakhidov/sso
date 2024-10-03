package main

import (
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/BariVakhidov/sso/internal/app"
	"github.com/BariVakhidov/sso/internal/config"
	"github.com/BariVakhidov/sso/internal/lib/logger/sl"
)

const (
	envLocal = "local"
	envDev   = "development"
	envProd  = "production"
)

func main() {
	//load config
	cfg := config.MustLoad()
	//setup logger
	logger := setupLogger(cfg.Env)
	logger.Info("starting application", slog.String("env", cfg.Env))

	application := app.New(logger, cfg.GRPC.Port, cfg.StoragePath, cfg.TokenTTL)

	go application.MustRun()

	//graceful shutdown
	stopChan := make(chan os.Signal, 1)
	signal.Notify(stopChan, syscall.SIGTERM, syscall.SIGINT)

	sign := <-stopChan
	logger.Info("stopping application", slog.String("signal", sign.String()))
	if err := application.Stop(); err != nil {
		logger.Info("failed to stop application", slog.String("signal", sign.String()), sl.Err(err))
		return
	}
	logger.Info("application stopped", slog.String("signal", sign.String()))
}

func setupLogger(env string) *slog.Logger {
	var logger *slog.Logger

	switch env {
	case envLocal:
		logger = slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	case envDev:
		logger = slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	case envProd:
		logger = slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	}

	return logger
}
