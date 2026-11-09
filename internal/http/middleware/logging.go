package middleware

import (
	"log/slog"
	"net/http"
	"time"
)

func Logging(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			sw := wrapStatus(w)

			defer func() {
				rec := recover()

				status := sw.Status()
				if rec != nil {
					status = http.StatusInternalServerError
				}

				level := slog.LevelInfo
				if status >= http.StatusInternalServerError {
					level = slog.LevelError
				}

				logger.LogAttrs(r.Context(), level, "request handled",
					slog.String("method", r.Method),
					slog.String("route", RouteFromContext(r.Context())),
					slog.Int("status", status),
					slog.Int64("duration_ms", time.Since(start).Milliseconds()),
				)

				if rec != nil {
					panic(rec)
				}
			}()

			next.ServeHTTP(sw, r)
		})
	}
}
