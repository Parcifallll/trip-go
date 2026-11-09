package container

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Parcifallll/trip-go/internal/config"
	"github.com/Parcifallll/trip-go/internal/http/health"
	"github.com/Parcifallll/trip-go/internal/http/server"
	createhandler "github.com/Parcifallll/trip-go/internal/http/trip/create"
	finishhandler "github.com/Parcifallll/trip-go/internal/http/trip/finish"
	gethandler "github.com/Parcifallll/trip-go/internal/http/trip/get"
	"github.com/Parcifallll/trip-go/internal/infra/postgres"
	"github.com/Parcifallll/trip-go/internal/infra/postgres/transaction"
	"github.com/Parcifallll/trip-go/internal/infra/telemetry"
	triprepo "github.com/Parcifallll/trip-go/internal/repository/trip"
	createusecase "github.com/Parcifallll/trip-go/internal/usecase/trip/create"
	finishusecase "github.com/Parcifallll/trip-go/internal/usecase/trip/finish"
	getusecase "github.com/Parcifallll/trip-go/internal/usecase/trip/get"
)

type Container struct {
	cfg       *config.Config
	logger    *slog.Logger
	telemetry *telemetry.Telemetry
	pool      *pgxpool.Pool
	server    *server.Server
}

func New(ctx context.Context, cfg *config.Config) (*Container, error) {
	tel, err := telemetry.Init(ctx, telemetry.Config{
		ServiceName:  cfg.OTELServiceName,
		LogLevel:     cfg.LogLevel,
		OTLPProtocol: cfg.OTELExporterProtocol,
	})
	if err != nil {
		return nil, fmt.Errorf("init telemetry: %w", err)
	}

	c, err := build(ctx, cfg, tel)
	if err != nil {
		shutdownCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), cfg.ShutdownTimeout)
		defer cancel()
		_ = tel.Shutdown(shutdownCtx)
		return nil, err
	}

	return c, nil
}

func build(ctx context.Context, cfg *config.Config, tel *telemetry.Telemetry) (*Container, error) {
	logger := tel.Logger()

	queryTracer, err := postgres.NewQueryTracer(tel.TracerProvider(), tel.MeterProvider())
	if err != nil {
		return nil, fmt.Errorf("create query tracer: %w", err)
	}

	pool, err := postgres.NewPool(ctx, postgres.Config{
		URL:             cfg.DatabaseURL,
		MaxConns:        cfg.DatabaseMaxConns,
		MinConns:        cfg.DatabaseMinConns,
		MaxConnLifetime: cfg.DatabaseMaxConnLifetime,
		ConnectTimeout:  cfg.DatabaseConnectTimeout,
	}, queryTracer)
	if err != nil {
		return nil, fmt.Errorf("create postgres pool: %w", err)
	}

	tripMetrics, err := telemetry.NewTripMetrics(tel.MeterProvider())
	if err != nil {
		pool.Close()
		return nil, fmt.Errorf("create trip metrics: %w", err)
	}

	txManager := transaction.NewManager(pool)
	tripRepo := triprepo.New(pool, cfg.DatabaseQueryTimeout)
	historyRepo := triprepo.NewHistory(pool, cfg.DatabaseQueryTimeout)

	createUC := createusecase.New(tripRepo, historyRepo, txManager, tripMetrics, logger, time.Now, uuid.New)
	getUC := getusecase.New(tripRepo)
	finishUC := finishusecase.New(tripRepo, historyRepo, txManager, tripMetrics, logger, time.Now)

	srv, err := server.New(
		server.Config{
			Addr:              cfg.HTTPAddr,
			ReadTimeout:       cfg.HTTPReadTimeout,
			ReadHeaderTimeout: cfg.HTTPReadHeaderTimeout,
			WriteTimeout:      cfg.HTTPWriteTimeout,
			IdleTimeout:       cfg.HTTPIdleTimeout,
		},
		server.Handlers{
			CreateTrip: createhandler.New(createUC, logger),
			GetTrip:    gethandler.New(getUC, logger),
			FinishTrip: finishhandler.New(finishUC, logger),
			Health:     health.New(pool, cfg.DatabaseQueryTimeout),
		},
		logger,
		tel.Tracer(),
		tel.Meter(),
	)
	if err != nil {
		pool.Close()
		return nil, fmt.Errorf("create http server: %w", err)
	}

	return &Container{
		cfg:       cfg,
		logger:    logger,
		telemetry: tel,
		pool:      pool,
		server:    srv,
	}, nil
}

func (c *Container) Run(ctx context.Context) error {
	serverErr := make(chan error, 1)

	go func() {
		c.logger.InfoContext(ctx, "http server starting", slog.String("addr", c.cfg.HTTPAddr))
		if err := c.server.Start(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErr <- err
		}
	}()

	var runErr error

	select {
	case <-ctx.Done():
		c.logger.InfoContext(ctx, "shutdown signal received")
	case err := <-serverErr:
		runErr = fmt.Errorf("http server: %w", err)
		c.logger.ErrorContext(ctx, "http server failed", slog.Any("error", err))
	}

	return errors.Join(runErr, c.shutdown(ctx))
}

func (c *Container) shutdown(ctx context.Context) error {
	shutdownCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), c.cfg.ShutdownTimeout)
	defer cancel()

	var errs []error

	if err := c.server.Shutdown(shutdownCtx); err != nil {
		c.logger.WarnContext(shutdownCtx, "graceful shutdown timed out, forcing exit", slog.Any("error", err))
		errs = append(errs, fmt.Errorf("shutdown http server: %w", err))
		if closeErr := c.server.Close(); closeErr != nil {
			errs = append(errs, fmt.Errorf("close http server: %w", closeErr))
		}
	}

	c.pool.Close()
	c.logger.InfoContext(shutdownCtx, "service stopped")

	if err := c.telemetry.Shutdown(shutdownCtx); err != nil {
		errs = append(errs, fmt.Errorf("shutdown telemetry: %w", err))
	}

	return errors.Join(errs...)
}
