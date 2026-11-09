package get_test

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
	"time"

	"github.com/google/uuid"
	"go.uber.org/mock/gomock"

	"github.com/Parcifallll/trip-go/internal/domain/trip"
	api "github.com/Parcifallll/trip-go/internal/generated"
	"github.com/Parcifallll/trip-go/internal/http/trip/get"
)

var tripID = uuid.MustParse("11111111-1111-1111-1111-111111111111")

func existingTrip() trip.Trip {
	return trip.Trip{
		ID:         tripID,
		UserID:     uuid.MustParse("22222222-2222-2222-2222-222222222222"),
		DriverID:   uuid.MustParse("33333333-3333-3333-3333-333333333333"),
		StartPoint: trip.Coordinates{Latitude: 55.75, Longitude: 37.61},
		EndPoint:   trip.Coordinates{Latitude: 55.80, Longitude: 37.70},
		Price:      1500,
		Status:     trip.StatusActive,
		StartedAt:  time.Date(2026, 11, 8, 12, 0, 0, 0, time.UTC),
	}
}

func serve(t *testing.T, uc *MockuseCase) *httptest.ResponseRecorder {
	t.Helper()

	handler := get.New(uc, slog.New(slog.NewTextHandler(io.Discard, nil)))
	req := httptest.NewRequest(http.MethodGet, "/api/v1/trips/"+tripID.String(), nil)
	rec := httptest.NewRecorder()

	handler.Handle(rec, req, tripID)

	return rec
}

func TestHandler_Handle_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	uc := NewMockuseCase(ctrl)
	uc.EXPECT().Execute(gomock.Any(), tripID).Return(existingTrip(), nil)

	rec := serve(t, uc)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	var got api.Trip
	if err := json.NewDecoder(rec.Body).Decode(&got); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if got.Id != tripID {
		t.Errorf("id = %s, want %s", got.Id, tripID)
	}
	if got.StartPoint.Latitude != 55.75 || got.EndPoint.Longitude != 37.70 {
		t.Errorf("coordinates are not converted: %+v %+v", got.StartPoint, got.EndPoint)
	}
}

func TestHandler_Handle_Errors(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		wantStatus int
		wantCode   string
	}{
		{
			name:       "not found",
			err:        trip.ErrTripNotFound,
			wantStatus: http.StatusNotFound,
			wantCode:   "trip_not_found",
		},
		{
			name:       "wrapped not found",
			err:        fmt.Errorf("get trip: %w", trip.ErrTripNotFound),
			wantStatus: http.StatusNotFound,
			wantCode:   "trip_not_found",
		},
		{
			name:       "internal error",
			err:        errors.New("dial tcp 10.0.0.5:5432: password=secret"),
			wantStatus: http.StatusInternalServerError,
			wantCode:   "internal_error",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			uc := NewMockuseCase(ctrl)
			uc.EXPECT().Execute(gomock.Any(), tripID).Return(trip.Trip{}, tc.err)

			rec := serve(t, uc)

			if rec.Code != tc.wantStatus {
				t.Fatalf("status = %d, want %d", rec.Code, tc.wantStatus)
			}
			if strings.Contains(rec.Body.String(), "secret") {
				t.Errorf("response leaks error details: %s", rec.Body.String())
			}

			var p api.Problem
			if err := json.NewDecoder(rec.Body).Decode(&p); err != nil {
				t.Fatalf("decode problem: %v", err)
			}
			if p.Code != tc.wantCode {
				t.Errorf("code = %q, want %q", p.Code, tc.wantCode)
			}
		})
	}
}
