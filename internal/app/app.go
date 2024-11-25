package app

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	grpcapp "github.com/BariVakhidov/sso/internal/app/grpc"
	prometheusapp "github.com/BariVakhidov/sso/internal/app/prometheus"
	storageapp "github.com/BariVakhidov/sso/internal/app/storage"
	redisapp "github.com/BariVakhidov/sso/internal/app/storage/redis"
	jaegerapp "github.com/BariVakhidov/sso/internal/app/tracing/jaeger"
	"github.com/BariVakhidov/sso/internal/config"
	"github.com/BariVakhidov/sso/internal/kafka"
	"github.com/BariVakhidov/sso/internal/lib/logger/sl"
	authservice "github.com/BariVakhidov/sso/internal/services/auth"
	eventsender "github.com/BariVakhidov/sso/internal/services/event_sender"
	"go.opentelemetry.io/otel"
)

const (
	eventsLimit       = 100
	producingInterval = time.Millisecond * 1000
)

type App struct {
	grpcServer     *grpcapp.App
	metrics        *prometheusapp.App
	storage        *storageapp.App
	redisStorage   *redisapp.App
	eventSender    *eventsender.Sender
	tracerProvider *jaegerapp.App
}

func New(log *slog.Logger, grpcPort int, storagePath string, ttl time.Duration, addr config.Addr) *App {
	metrics := prometheusapp.New(log, 9090)
	brokers := []string{"host.docker.internal:29092"}
	topic := "user_created"
	kafkaPublisher := kafka.NewKafkaProducer(brokers, topic)

	//TODO: configs
	storage := storageapp.MustCreateApp(fmt.Sprintf("postgres://postgres:password@%s/sso", addr.Db), log)
	tracerProvider, err := jaegerapp.New(log, "host.docker.internal:4318")
	if err != nil {
		log.Error("failed to create tracerApp", sl.Err(err))
	}
	tracer := otel.Tracer("sso")
	redisApp := redisapp.New(log, addr.Redis, time.Minute*10)

	eventSender := eventsender.NewSender(log, kafkaPublisher, storage.Storage)

	authService := authservice.New(
		log,
		storage.Storage,
		storage.Storage,
		storage.Storage,
		redisApp.Storage,
		ttl,
		metrics.FailedLoginsCounter,
		tracer,
	)

	grpcappOpts := grpcapp.AppOpts{
		Log:         log,
		Port:        grpcPort,
		StoragePath: storagePath,
		TTL:         ttl,
	}
	grpcApp := grpcapp.New(grpcappOpts, authService, metrics, metrics.RecoveryOpt, metrics.MetricsInterceptor)

	return &App{
		grpcServer:     grpcApp,
		storage:        storage,
		metrics:        metrics,
		redisStorage:   redisApp,
		eventSender:    eventSender,
		tracerProvider: tracerProvider,
	}
}

func (a *App) MustRun() {
	go a.grpcServer.MustRun()
	go a.metrics.MustRun()
	go a.eventSender.StartProducing(context.Background(), eventsLimit, producingInterval)
}

func (a *App) Stop() error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()
	a.grpcServer.Stop()
	a.storage.Stop()
	a.eventSender.StopSending()
	a.tracerProvider.Stop(ctx)
	return a.redisStorage.Stop()
}
