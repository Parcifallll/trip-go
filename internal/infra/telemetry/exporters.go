package telemetry

import (
	"context"
	"fmt"

	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

const protocolHTTPProtobuf = "http/protobuf"

func newTraceExporter(ctx context.Context, protocol string) (sdktrace.SpanExporter, error) {
	if protocol != protocolHTTPProtobuf {
		return nil, fmt.Errorf("unsupported OTLP protocol %q", protocol)
	}
	return otlptracehttp.New(ctx)
}

func newMetricExporter(ctx context.Context, protocol string) (sdkmetric.Exporter, error) {
	if protocol != protocolHTTPProtobuf {
		return nil, fmt.Errorf("unsupported OTLP protocol %q", protocol)
	}
	return otlpmetrichttp.New(ctx)
}
