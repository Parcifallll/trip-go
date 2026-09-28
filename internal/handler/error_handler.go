package handler

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/Parcifallll/trip-go/internal/domain"
	"github.com/Parcifallll/trip-go/internal/generated"
)

type Problem = api.Problem

func WriteProblem(w http.ResponseWriter, p Problem) {
	w.Header().Set("Content-Type", "application/problem+json")
	w.WriteHeader(int(p.Status))
	_ = json.NewEncoder(w).Encode(p)
}

func MapError(err error, instance string) Problem {
	switch {
	case errors.Is(err, domain.ErrDriverBusy):
		return Problem{
			Type:     "https://tripgo.example/problems/driver-busy",
			Title:    "Driver busy",
			Status:   409,
			Detail:   new("Driver already has an active trip"),
			Code:     "driver_busy",
			Instance: &instance,
		}
	case errors.Is(err, domain.ErrTripNotFound):
		return Problem{
			Type:     "https://tripgo.example/problems/trip-not-found",
			Title:    "Trip not found",
			Status:   404,
			Detail:   new("Trip was not found"),
			Code:     "trip_not_found",
			Instance: &instance,
		}
	case errors.Is(err, domain.ErrTripCompleted):
		return Problem{
			Type:     "https://tripgo.example/problems/trip-completed",
			Title:    "Trip completed",
			Status:   409,
			Detail:   new("Operation is not allowed for a completed trip"),
			Code:     "trip_completed",
			Instance: &instance,
		}
	case errors.Is(err, domain.ErrInvalidRequest):
		return Problem{
			Type:     "https://tripgo.example/problems/invalid-request",
			Title:    "Invalid request",
			Status:   400,
			Detail:   new("Request validation failed"),
			Code:     "invalid_request",
			Instance: &instance,
		}
	default:
		return Problem{
			Type:     "https://tripgo.example/problems/internal-error",
			Title:    "Internal Server Error",
			Status:   500,
			Detail:   new("Internal server error"),
			Code:     "internal_error",
			Instance: &instance,
		}
	}
}

func WriteError(w http.ResponseWriter, err error, instance string) {
	WriteProblem(w, MapError(err, instance))
}
