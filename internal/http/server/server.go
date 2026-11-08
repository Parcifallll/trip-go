package server

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"

	"github.com/Parcifallll/trip-go/internal/domain/trip"
	api "github.com/Parcifallll/trip-go/internal/generated"
	"github.com/Parcifallll/trip-go/internal/http/domainerror"
	"github.com/Parcifallll/trip-go/internal/http/health"
	"github.com/Parcifallll/trip-go/internal/http/middleware"
	createhandler "github.com/Parcifallll/trip-go/internal/http/trip/create"
	finishhandler "github.com/Parcifallll/trip-go/internal/http/trip/finish"
	gethandler "github.com/Parcifallll/trip-go/internal/http/trip/get"
)

type Config struct {
	Addr              string
	ReadTimeout       time.Duration
	ReadHeaderTimeout time.Duration
	WriteTimeout      time.Duration
	IdleTimeout       time.Duration
}

type Handlers struct {
	CreateTrip *createhandler.Handler
	GetTrip    *gethandler.Handler
	FinishTrip *finishhandler.Handler
	Health     *health.Handler
}

type Server struct {
	httpServer *http.Server
}

func New(
	cfg Config,
	handlers Handlers,
	logger *slog.Logger,
	tracer trace.Tracer,
	meter metric.Meter,
) (*Server, error) {
	mux := chi.NewRouter()

	api.HandlerWithOptions(&apiServer{handlers: handlers}, api.ChiServerOptions{
		BaseRouter:       mux,
		ErrorHandlerFunc: paramErrorHandler(logger),
	})

	metricsMiddleware, err := middleware.Metrics(meter)
	if err != nil {
		return nil, fmt.Errorf("create metrics middleware: %w", err)
	}

	root := middleware.Chain(
		mux,
		middleware.Recover(logger),
		middleware.Tracing(tracer, routeResolver(mux)),
		middleware.Logging(logger),
		metricsMiddleware,
	)

	return &Server{
		httpServer: &http.Server{
			Addr:              cfg.Addr,
			Handler:           root,
			ReadTimeout:       cfg.ReadTimeout,
			ReadHeaderTimeout: cfg.ReadHeaderTimeout,
			WriteTimeout:      cfg.WriteTimeout,
			IdleTimeout:       cfg.IdleTimeout,
			ErrorLog:          slog.NewLogLogger(logger.Handler(), slog.LevelError),
		},
	}, nil
}

func (s *Server) Start() error {
	return s.httpServer.ListenAndServe()
}

func (s *Server) Shutdown(ctx context.Context) error {
	return s.httpServer.Shutdown(ctx)
}

func (s *Server) Close() error {
	return s.httpServer.Close()
}

func routeResolver(mux *chi.Mux) middleware.RouteResolver {
	return func(r *http.Request) string {
		rctx := chi.NewRouteContext()
		if mux.Match(rctx, r.Method, r.URL.Path) {
			if pattern := rctx.RoutePattern(); pattern != "" {
				return pattern
			}
		}
		return "unmatched"
	}
}

func paramErrorHandler(logger *slog.Logger) func(http.ResponseWriter, *http.Request, error) {
	return func(w http.ResponseWriter, r *http.Request, err error) {
		field := "request"

		var invalidFormat *api.InvalidParamFormatError
		var tooMany *api.TooManyValuesForParamError
		switch {
		case errors.As(err, &invalidFormat):
			field = invalidFormat.ParamName
		case errors.As(err, &tooMany):
			field = tooMany.ParamName
		}

		domainerror.Write(w, r, logger, &trip.ValidationError{Field: field, Reason: "has an invalid value"})
	}
}
