package main

import (
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/BariVakhidov/sso/internal/app"
	"github.com/BariVakhidov/sso/internal/config"
	"github.com/BariVakhidov/sso/internal/lib/logger/sl"
	"github.com/BariVakhidov/sso/internal/logger"
)

func main() {
	//load config
	cfg := config.MustLoad()
	//setup logger
	logger := logger.New(cfg.Env)
	logger.Log.Info("starting application", slog.String("env", cfg.Env))

	application := app.New(logger.Log, cfg.GRPC.Port, cfg.StoragePath, cfg.TokenTTL, cfg.Addr)

	go application.MustRun()

	//graceful shutdown
	stopChan := make(chan os.Signal, 1)
	signal.Notify(stopChan, syscall.SIGTERM, syscall.SIGINT)

	sign := <-stopChan
	logger.Log.Info("stopping application", slog.String("signal", sign.String()))
	if err := application.Stop(); err != nil {
		logger.Log.Info("failed to stop application", slog.String("signal", sign.String()), sl.Err(err))
		return
	}
	logger.Log.Info("application stopped", slog.String("signal", sign.String()))
}
