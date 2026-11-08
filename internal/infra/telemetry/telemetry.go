package telemetry

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"runtime/debug"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
)

const (
	instrumentationName = "github.com/Parcifallll/trip-go"
	metricExportPeriod  = 10 * time.Second
)

type Config struct {
	ServiceName  string
	LogLevel     string
	OTLPProtocol string
}

type Telemetry struct {
	logger *slog.Logger
	tp     *sdktrace.TracerProvider
	mp     *sdkmetric.MeterProvider
}

func Init(ctx context.Context, cfg Config) (*Telemetry, error) {
	logger, err := newLogger(cfg.LogLevel, cfg.ServiceName)
	if err != nil {
		return nil, err
	}

	res, err := resource.New(ctx,
		resource.WithTelemetrySDK(),
		resource.WithAttributes(
			attribute.String("service.name", cfg.ServiceName),
			attribute.String("service.version", serviceVersion()),
		),
		resource.WithFromEnv(),
	)
	if err != nil {
		return nil, fmt.Errorf("create resource: %w", err)
	}

	traceExporter, err := newTraceExporter(ctx, cfg.OTLPProtocol)
	if err != nil {
		return nil, fmt.Errorf("create trace exporter: %w", err)
	}

	metricExporter, err := newMetricExporter(ctx, cfg.OTLPProtocol)
	if err != nil {
		return nil, fmt.Errorf("create metric exporter: %w", err)
	}

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(traceExporter),
		sdktrace.WithResource(res),
	)
	mp := sdkmetric.NewMeterProvider(
		sdkmetric.WithReader(sdkmetric.NewPeriodicReader(metricExporter, sdkmetric.WithInterval(metricExportPeriod))),
		sdkmetric.WithResource(res),
	)

	otel.SetErrorHandler(otel.ErrorHandlerFunc(func(err error) {
		logger.Warn("telemetry error", slog.Any("error", err))
	}))

	return &Telemetry{logger: logger, tp: tp, mp: mp}, nil
}

func (t *Telemetry) Logger() *slog.Logger {
	return t.logger
}

func (t *Telemetry) TracerProvider() trace.TracerProvider {
	return t.tp
}

func (t *Telemetry) MeterProvider() metric.MeterProvider {
	return t.mp
}

func (t *Telemetry) Tracer() trace.Tracer {
	return t.tp.Tracer(instrumentationName)
}

func (t *Telemetry) Meter() metric.Meter {
	return t.mp.Meter(instrumentationName)
}

func (t *Telemetry) Shutdown(ctx context.Context) error {
	return errors.Join(t.tp.Shutdown(ctx), t.mp.Shutdown(ctx))
}

func serviceVersion() string {
	if info, ok := debug.ReadBuildInfo(); ok && info.Main.Version != "" && info.Main.Version != "(devel)" {
		return info.Main.Version
	}
	return "dev"
}
