package middleware

import (
	"fmt"
	"net/http"
	"strconv"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

func Tracing(tracer trace.Tracer, resolve RouteResolver) func(http.Handler) http.Handler {
	propagator := propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{})

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			route := resolve(r)

			ctx := propagator.Extract(r.Context(), propagation.HeaderCarrier(r.Header))
			ctx, span := tracer.Start(ctx, r.Method+" "+route,
				trace.WithSpanKind(trace.SpanKindServer),
				trace.WithAttributes(
					attribute.String("http.request.method", r.Method),
					attribute.String("http.route", route),
					attribute.String("url.path", r.URL.Path),
				),
			)
			ctx = withRoute(ctx, route)

			sw := wrapStatus(w)

			defer func() {
				rec := recover()

				status := sw.Status()
				if rec != nil {
					status = http.StatusInternalServerError
					span.RecordError(fmt.Errorf("panic: %v", rec))
				}

				span.SetAttributes(attribute.Int("http.response.status_code", status))
				if status >= http.StatusInternalServerError {
					span.SetStatus(codes.Error, http.StatusText(status))
					span.SetAttributes(attribute.String("error.type", strconv.Itoa(status)))
				}
				span.End()

				if rec != nil {
					panic(rec)
				}
			}()

			next.ServeHTTP(sw, r.WithContext(ctx))
		})
	}
}
