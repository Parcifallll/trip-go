package domainerror_test

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Parcifallll/trip-go/internal/domain/trip"
	api "github.com/Parcifallll/trip-go/internal/generated"
	"github.com/Parcifallll/trip-go/internal/http/domainerror"
)

const tripNotFound = "trip_not_found"

func TestProblem(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		wantStatus int
		wantCode   string
	}{
		{
			name:       "validation error",
			err:        &trip.ValidationError{Field: "price", Reason: "must be greater than or equal to 0"},
			wantStatus: http.StatusBadRequest,
			wantCode:   "invalid_request",
		},
		{
			name:       "wrapped validation error",
			err:        fmt.Errorf("new trip: %w", &trip.ValidationError{Field: "id", Reason: "must not be empty"}),
			wantStatus: http.StatusBadRequest,
			wantCode:   "invalid_request",
		},
		{
			name:       "trip not found",
			err:        trip.ErrTripNotFound,
			wantStatus: http.StatusNotFound,
			wantCode:   "trip_not_found",
		},
		{
			name:       "wrapped trip not found",
			err:        fmt.Errorf("get trip: %w", trip.ErrTripNotFound),
			wantStatus: http.StatusNotFound,
			wantCode:   tripNotFound,
		},
		{
			name:       "trip completed",
			err:        trip.ErrTripCompleted,
			wantStatus: http.StatusConflict,
			wantCode:   "trip_completed",
		},
		{
			name:       "wrapped trip completed",
			err:        fmt.Errorf("finish trip: %w", trip.ErrTripCompleted),
			wantStatus: http.StatusConflict,
			wantCode:   "trip_completed",
		},
		{
			name:       "driver busy",
			err:        trip.ErrDriverBusy,
			wantStatus: http.StatusConflict,
			wantCode:   "driver_busy",
		},
		{
			name:       "wrapped driver busy",
			err:        fmt.Errorf("create trip: %w", trip.ErrDriverBusy),
			wantStatus: http.StatusConflict,
			wantCode:   "driver_busy",
		},
		{
			name:       "unknown error",
			err:        errors.New("connection refused"),
			wantStatus: http.StatusInternalServerError,
			wantCode:   "internal_error",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			status, body := domainerror.Problem(tc.err)

			if status != tc.wantStatus {
				t.Errorf("status = %d, want %d", status, tc.wantStatus)
			}
			if int(body.Status) != tc.wantStatus {
				t.Errorf("body status = %d, want %d", body.Status, tc.wantStatus)
			}
			if body.Code != tc.wantCode {
				t.Errorf("code = %q, want %q", body.Code, tc.wantCode)
			}
			if body.Title == "" || body.Type == "" {
				t.Errorf("title and type must be set, got %+v", body)
			}
		})
	}
}

func TestWrite(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	t.Run("domain error", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/api/v1/trips/42", nil)

		domainerror.Write(rec, req, logger, trip.ErrTripNotFound)

		if rec.Code != http.StatusNotFound {
			t.Errorf("status = %d, want %d", rec.Code, http.StatusNotFound)
		}
		if ct := rec.Header().Get("Content-Type"); ct != "application/problem+json" {
			t.Errorf("content type = %q, want application/problem+json", ct)
		}

		var body api.Problem
		if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		if body.Code != "trip_not_found" {
			t.Errorf("code = %q, want trip_not_found", body.Code)
		}
		if body.Instance == nil || *body.Instance != "/api/v1/trips/42" {
			t.Errorf("instance = %v, want /api/v1/trips/42", body.Instance)
		}
	})

	t.Run("internal error does not leak details", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/api/v1/trips", nil)

		domainerror.Write(rec, req, logger, errors.New("dial tcp 10.0.0.5:5432: password=secret"))

		if rec.Code != http.StatusInternalServerError {
			t.Errorf("status = %d, want %d", rec.Code, http.StatusInternalServerError)
		}
		if strings.Contains(rec.Body.String(), "secret") || strings.Contains(rec.Body.String(), "10.0.0.5") {
			t.Errorf("response leaks error details: %s", rec.Body.String())
		}
	})
}
