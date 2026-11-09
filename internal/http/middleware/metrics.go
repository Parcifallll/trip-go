package middleware

import (
	"fmt"
	"net/http"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

func Metrics(meter metric.Meter) (func(http.Handler) http.Handler, error) {
	duration, err := meter.Float64Histogram(
		"http.server.request.duration",
		metric.WithUnit("s"),
		metric.WithDescription("Duration of HTTP server requests."),
		metric.WithExplicitBucketBoundaries(0.005, 0.01, 0.025, 0.05, 0.075, 0.1, 0.25, 0.5, 0.75, 1, 2.5, 5, 7.5, 10),
	)
	if err != nil {
		return nil, fmt.Errorf("create request duration histogram: %w", err)
	}

	active, err := meter.Int64UpDownCounter(
		"http.server.active_requests",
		metric.WithUnit("{request}"),
		metric.WithDescription("Number of requests being handled right now."),
	)
	if err != nil {
		return nil, fmt.Errorf("create active requests counter: %w", err)
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			start := time.Now()
			sw := wrapStatus(w)

			method := attribute.String("http.request.method", r.Method)
			route := attribute.String("http.route", RouteFromContext(ctx))
			activeAttrs := metric.WithAttributes(method, route)

			active.Add(ctx, 1, activeAttrs)

			defer func() {
				rec := recover()

				status := sw.Status()
				if rec != nil {
					status = http.StatusInternalServerError
				}

				active.Add(ctx, -1, activeAttrs)
				duration.Record(ctx, time.Since(start).Seconds(), metric.WithAttributes(
					method,
					route,
					attribute.Int("http.response.status_code", status),
				))

				if rec != nil {
					panic(rec)
				}
			}()

			next.ServeHTTP(sw, r)
		})
	}, nil
}
