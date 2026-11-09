package middleware

import (
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"runtime/debug"

	"github.com/Parcifallll/trip-go/internal/http/domainerror"
)

var errPanic = errors.New("panic recovered")

func Recover(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				rec := recover()
				if rec == nil {
					return
				}

				if err, ok := rec.(error); ok && errors.Is(err, http.ErrAbortHandler) {
					panic(rec)
				}

				logger.ErrorContext(r.Context(), "panic recovered",
					slog.String("method", r.Method),
					slog.String("path", r.URL.Path),
					slog.String("panic", fmt.Sprint(rec)),
					slog.String("stack", string(debug.Stack())),
				)

				status, body := domainerror.Problem(errPanic)
				path := r.URL.Path
				body.Instance = &path
				domainerror.WriteProblem(w, status, body)
			}()

			next.ServeHTTP(w, r)
		})
	}
}
