package prometheusapp

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"runtime/debug"

	authgrpc "github.com/BariVakhidov/sso/internal/grpc/auth"
	"github.com/BariVakhidov/sso/internal/lib/logger/sl"
	grpcprom "github.com/grpc-ecosystem/go-grpc-middleware/providers/prometheus"
	"github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/recovery"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type App struct {
	log                 *slog.Logger
	port                int
	reg                 *prometheus.Registry
	serverMetrics       *grpcprom.ServerMetrics
	RecoveryOpt         recovery.Option
	MetricsInterceptor  grpc.UnaryServerInterceptor
	FailedLoginsCounter *prometheus.CounterVec
}

func New(log *slog.Logger, port int) *App {
	// Setup metrics.
	srvMetrics := grpcprom.NewServerMetrics(
		grpcprom.WithServerHandlingTimeHistogram(
			grpcprom.WithHistogramBuckets([]float64{0.001, 0.01, 0.1, 0.3, 0.6, 1, 3, 6, 9, 20, 30, 60, 90, 120}),
		),
	)
	reg := prometheus.NewRegistry()
	reg.MustRegister(srvMetrics)

	exemplarFromContext := func(ctx context.Context) prometheus.Labels {
		if span := trace.SpanContextFromContext(ctx); span.IsSampled() {
			return prometheus.Labels{"traceID": span.TraceID().String()}
		}
		return nil
	}

	panicsTotal := promauto.With(reg).NewCounter(prometheus.CounterOpts{
		Name: "grpc_req_panics_recovered_total",
		Help: "Total number of gRPC requests recovered from internal panic.",
	})
	failedLogins := promauto.With(reg).NewCounterVec(prometheus.CounterOpts{
		Name: "failed_login_attempts_total",
		Help: "Total number of failed login attempts.",
	}, []string{"email", "ip"})

	grpcPanicRecoveryHandler := recovery.WithRecoveryHandler(func(p any) (err error) {
		panicsTotal.Inc()
		log.Error("msg", "recovered from panic", "panic", p, "stack", debug.Stack())
		return status.Errorf(codes.Internal, authgrpc.ErrInternal)
	})

	metricsInterceptor := srvMetrics.UnaryServerInterceptor(grpcprom.WithExemplarFromContext(exemplarFromContext))

	return &App{
		log:                 log,
		port:                port,
		reg:                 reg,
		serverMetrics:       srvMetrics,
		RecoveryOpt:         grpcPanicRecoveryHandler,
		MetricsInterceptor:  metricsInterceptor,
		FailedLoginsCounter: failedLogins,
	}
}

func (a *App) MustRun() {
	err := a.Run()
	if errors.Is(err, http.ErrServerClosed) {
		a.log.Info("Prometheus server closed", sl.Err(err))
	} else if err != nil {
		a.log.Error("Failed to start Prometheus", sl.Err(err))
		panic(err)
	}
}

func (a *App) Run() error {
	const op = "prometheusapp.Run"
	log := a.log.With(slog.String("op", op), slog.Int("port", a.port))

	log.Info("exposing Prometheus metrics")

	http.Handle("/metrics", promhttp.HandlerFor(
		a.reg,
		promhttp.HandlerOpts{
			// Opt into OpenMetrics e.g. to support exemplars.
			EnableOpenMetrics: true,
		},
	))

	return http.ListenAndServe(fmt.Sprintf(":%d", a.port), nil)
}

func (a *App) Initialize(srv *grpc.Server) {
	a.serverMetrics.InitializeMetrics(srv)
}
