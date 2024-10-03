package grpcapp

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"time"

	"github.com/BariVakhidov/sso/internal/grpc/auth"
	authgrpc "github.com/BariVakhidov/sso/internal/grpc/auth"
	"github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/logging"
	"github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/recovery"
	"google.golang.org/grpc"
)

type AppOpts struct {
	Log         *slog.Logger
	Port        int
	StoragePath string
	TTL         time.Duration
}

type Metrics interface {
	Initialize(srv *grpc.Server)
}

type App struct {
	AppOpts
	gRPCServer *grpc.Server
}

func New(opts AppOpts, auth auth.Auth, metrics Metrics, recoveryOpt recovery.Option, metricsInterceptor grpc.UnaryServerInterceptor) *App {
	logOpts := []logging.Option{
		logging.WithLogOnEvents(logging.PayloadSent, logging.PayloadReceived),
	}

	gRPCServer := grpc.NewServer(grpc.ChainUnaryInterceptor(
		metricsInterceptor,
		logging.UnaryServerInterceptor(InterceptorLogger(opts.Log), logOpts...),
		recovery.UnaryServerInterceptor(recoveryOpt),
	))

	metrics.Initialize(gRPCServer)

	authgrpc.Register(gRPCServer, auth)

	return &App{gRPCServer: gRPCServer, AppOpts: opts}
}

// MustRun runs gRPC server and panic if any error occurs
func (a *App) MustRun() {
	if err := a.Run(); err != nil {
		panic(err)
	}
}

func (a *App) Run() error {
	const op = "grpcapp.Run"
	log := a.Log.With(slog.String("op", op), slog.Int("port", a.Port))

	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", a.Port))
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	log.Info("gRPC server is running", slog.String("addr", listener.Addr().String()))

	if err := a.gRPCServer.Serve(listener); err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	return nil
}

func (a *App) Stop() {
	const op = "grpcapp.Stop"

	a.Log.With(slog.String("op", op), slog.Int("port", a.Port)).
		Info("stopping gRPC server", slog.Int("port", a.Port))

	a.gRPCServer.GracefulStop()
}

// InterceptorLogger adapts slog logger to interceptor logger.
// This code is simple enough to be copied and not imported.
func InterceptorLogger(l *slog.Logger) logging.Logger {
	return logging.LoggerFunc(func(ctx context.Context, lvl logging.Level, msg string, fields ...any) {
		l.Log(ctx, slog.Level(lvl), msg, fields...)
	})
}
