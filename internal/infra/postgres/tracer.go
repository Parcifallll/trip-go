package postgres

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/exaring/otelpgx"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
)

const meterName = "github.com/Parcifallll/trip-go/infra/postgres"

var (
	operationRe = regexp.MustCompile(`^\s*([A-Za-z]+)`)
	tableRe     = regexp.MustCompile(`(?i)\b(?:from|into|update)\s+"?([a-z_][a-z0-9_.]*)"?`)
)

type queryStateKey struct{}

type queryState struct {
	start time.Time
	attrs []attribute.KeyValue
}

type queryTracer struct {
	inner    pgx.QueryTracer
	duration metric.Float64Histogram
}

func NewQueryTracer(tp trace.TracerProvider, mp metric.MeterProvider) (pgx.QueryTracer, error) {
	duration, err := mp.Meter(meterName).Float64Histogram(
		"db.client.operation.duration",
		metric.WithUnit("s"),
		metric.WithDescription("Duration of database client operations."),
		metric.WithExplicitBucketBoundaries(0.001, 0.005, 0.01, 0.05, 0.1, 0.5, 1, 5, 10),
	)
	if err != nil {
		return nil, fmt.Errorf("create db duration histogram: %w", err)
	}

	return &queryTracer{
		inner:    otelpgx.NewTracer(otelpgx.WithTracerProvider(tp)),
		duration: duration,
	}, nil
}

func (t *queryTracer) TraceQueryStart(ctx context.Context, conn *pgx.Conn, data pgx.TraceQueryStartData) context.Context {
	ctx = t.inner.TraceQueryStart(ctx, conn, data)

	attrs := queryAttributes(data.SQL)
	trace.SpanFromContext(ctx).SetAttributes(attrs...)

	return context.WithValue(ctx, queryStateKey{}, queryState{start: time.Now(), attrs: attrs})
}

func (t *queryTracer) TraceQueryEnd(ctx context.Context, conn *pgx.Conn, data pgx.TraceQueryEndData) {
	state, ok := ctx.Value(queryStateKey{}).(queryState)

	attrs := make([]attribute.KeyValue, len(state.attrs), len(state.attrs)+1)
	copy(attrs, state.attrs)
	if data.Err != nil {
		errType := attribute.String("error.type", errorType(data.Err))
		attrs = append(attrs, errType)
		trace.SpanFromContext(ctx).SetAttributes(errType)
	}

	t.inner.TraceQueryEnd(ctx, conn, data)

	if ok {
		t.duration.Record(ctx, time.Since(state.start).Seconds(), metric.WithAttributes(attrs...))
	}
}

func queryAttributes(sql string) []attribute.KeyValue {
	attrs := make([]attribute.KeyValue, 0, 2)

	if m := operationRe.FindStringSubmatch(sql); m != nil {
		attrs = append(attrs, attribute.String("db.operation.name", strings.ToUpper(m[1])))
	}
	if m := tableRe.FindStringSubmatch(sql); m != nil {
		attrs = append(attrs, attribute.String("db.collection.name", m[1]))
	}

	return attrs
}

func errorType(err error) string {
	var pgErr *pgconn.PgError
	switch {
	case errors.As(err, &pgErr):
		return pgErr.Code
	case errors.Is(err, context.DeadlineExceeded):
		return "timeout"
	case errors.Is(err, context.Canceled):
		return "canceled"
	default:
		return fmt.Sprintf("%T", err)
	}
}
