package app

import (
	"log/slog"
	"time"

	grpcapp "github.com/BariVakhidov/sso/internal/app/grpc"
	"github.com/BariVakhidov/sso/internal/services/auth"
	"github.com/BariVakhidov/sso/internal/storage/sqllite"
)

type App struct {
	GRPCServer *grpcapp.App
}

func New(log *slog.Logger, grpcPort int, storagePath string, ttl time.Duration) *App {
	sqlite, err := sqllite.New(storagePath)
	if err != nil {
		panic(err)
	}

	auth := auth.New(log, sqlite, sqlite, sqlite, ttl)

	grpcappOpts := grpcapp.AppOpts{
		Log:         log,
		Port:        grpcPort,
		StoragePath: storagePath,
		TTL:         ttl,
	}
	grpcApp := grpcapp.New(grpcappOpts, auth)

	return &App{GRPCServer: grpcApp}
}
