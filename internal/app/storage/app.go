package storageapp

import (
	"log/slog"

	"github.com/BariVakhidov/sso/internal/storage/postgres"
)

type App struct {
	Storage *postgres.Storage
	log     *slog.Logger
	dbAddr  string
}

func MustCreateApp(dbAddr string, log *slog.Logger) *App {
	postgres, err := postgres.New(dbAddr)
	if err != nil {
		panic(err)
	}

	return &App{
		log:     log,
		Storage: postgres,
		dbAddr:  dbAddr,
	}
}

func (a *App) Stop() {
	const op = "storageapp.Stop"
	a.log.With(slog.String("op", op)).Info("stopping storage app")
	a.Storage.ClosePool()
}
