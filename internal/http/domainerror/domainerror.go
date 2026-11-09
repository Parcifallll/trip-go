package domainerror

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/Parcifallll/trip-go/internal/domain/trip"
	api "github.com/Parcifallll/trip-go/internal/generated"
)

const problemBase = "https://tripgo.example/problems/"

func Problem(err error) (int, api.Problem) {
	var validationErr *trip.ValidationError

	switch {
	case errors.As(err, &validationErr):
		detail := fmt.Sprintf("Request validation failed: %s %s", validationErr.Field, validationErr.Reason)
		return http.StatusBadRequest, problem(http.StatusBadRequest, "invalid-request", "Invalid request", "invalid_request", detail)
	case errors.Is(err, trip.ErrTripNotFound):
		return http.StatusNotFound, problem(http.StatusNotFound, "trip-not-found", "Trip not found", "trip_not_found", "Trip was not found")
	case errors.Is(err, trip.ErrTripCompleted):
		return http.StatusConflict, problem(http.StatusConflict, "trip-completed", "Trip completed", "trip_completed", "Operation is not allowed for a completed trip")
	case errors.Is(err, trip.ErrDriverBusy):
		return http.StatusConflict, problem(http.StatusConflict, "driver-busy", "Driver busy", "driver_busy", "Driver already has an active trip")
	default:
		return http.StatusInternalServerError, problem(http.StatusInternalServerError, "internal-error", "Internal Server Error", "internal_error", "Internal server error")
	}
}

func Write(w http.ResponseWriter, r *http.Request, logger *slog.Logger, err error) {
	status, body := Problem(err)

	path := r.URL.Path
	body.Instance = &path

	if status >= http.StatusInternalServerError {
		logger.ErrorContext(r.Context(), "request failed", slog.Any("error", err))
	}

	WriteProblem(w, status, body)
}

func WriteProblem(w http.ResponseWriter, status int, body api.Problem) {
	w.Header().Set("Content-Type", "application/problem+json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func problem(status int32, slug, title, code, detail string) api.Problem {
	return api.Problem{
		Type:   problemBase + slug,
		Title:  title,
		Status: status,
		Detail: &detail,
		Code:   code,
	}
}
