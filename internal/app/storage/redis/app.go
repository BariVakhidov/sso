package redisapp

import (
	"log/slog"
	"time"

	"github.com/BariVakhidov/sso/internal/storage/redis"
)

type App struct {
	Storage *redis.Storage
	log     *slog.Logger
}

func New(log *slog.Logger, addr string, ttl time.Duration) *App {
	redisStorage := redis.New(addr, ttl)

	return &App{Storage: redisStorage, log: log}
}

func (a *App) Stop() error {
	const op = "redisapp.Stop"
	a.log.With(slog.String("op", op)).Info("stopping redis app")
	return a.Storage.Stop()
}
