package app

import (
	"fmt"
	"log/slog"
	"time"

	grpcapp "github.com/BariVakhidov/sso/internal/app/grpc"
	prometheusapp "github.com/BariVakhidov/sso/internal/app/prometheus"
	storageapp "github.com/BariVakhidov/sso/internal/app/storage"
	redisapp "github.com/BariVakhidov/sso/internal/app/storage/redis"
	"github.com/BariVakhidov/sso/internal/config"
	authservice "github.com/BariVakhidov/sso/internal/services/auth"
)

type App struct {
	grpcServer   *grpcapp.App
	metrics      *prometheusapp.App
	storage      *storageapp.App
	redisStorage *redisapp.App
}

func New(log *slog.Logger, grpcPort int, storagePath string, ttl time.Duration, addr config.Addr) *App {
	metrics := prometheusapp.New(log, 9090)
	//TODO: configs
	storage := storageapp.MustCreateApp(fmt.Sprintf("postgres://postgres:password@%s/sso", addr.Db), log)

	redisApp := redisapp.New(log, addr.Redis, time.Minute*10)

	authService := authservice.New(
		log,
		storage.Storage,
		storage.Storage,
		storage.Storage,
		redisApp.Storage,
		ttl,
		metrics.FailedLoginsCounter,
	)

	grpcappOpts := grpcapp.AppOpts{
		Log:         log,
		Port:        grpcPort,
		StoragePath: storagePath,
		TTL:         ttl,
	}
	grpcApp := grpcapp.New(grpcappOpts, authService, metrics, metrics.RecoveryOpt, metrics.MetricsInterceptor)

	return &App{grpcServer: grpcApp, storage: storage, metrics: metrics, redisStorage: redisApp}
}

func (a *App) MustRun() {
	go a.grpcServer.MustRun()
	go a.metrics.MustRun()
}

func (a *App) Stop() error {
	a.grpcServer.Stop()
	a.storage.Stop()
	return a.redisStorage.Stop()
}
