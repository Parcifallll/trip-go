package create_test

import (
	"encoding/json"
	"errors"
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
	"github.com/Parcifallll/trip-go/internal/http/trip/create"
	"github.com/Parcifallll/trip-go/internal/model"
)

const validBody = `{
	"user_id": "22222222-2222-2222-2222-222222222222",
	"driver_id": "33333333-3333-3333-3333-333333333333",
	"start_point": {"latitude": 55.75, "longitude": 37.61},
	"end_point": {"latitude": 55.80, "longitude": 37.70},
	"price": 1500
}`

const bodyWithoutPrice = `{
	"user_id": "22222222-2222-2222-2222-222222222222",
	"driver_id": "33333333-3333-3333-3333-333333333333",
	"start_point": {"latitude": 55.75, "longitude": 37.61},
	"end_point": {"latitude": 55.80, "longitude": 37.70}
}`

var (
	tripID   = uuid.MustParse("11111111-1111-1111-1111-111111111111")
	userID   = uuid.MustParse("22222222-2222-2222-2222-222222222222")
	driverID = uuid.MustParse("33333333-3333-3333-3333-333333333333")
)

func validInput() model.CreateTripInput {
	return model.CreateTripInput{
		UserID:     userID,
		DriverID:   driverID,
		StartPoint: trip.Coordinates{Latitude: 55.75, Longitude: 37.61},
		EndPoint:   trip.Coordinates{Latitude: 55.80, Longitude: 37.70},
		Price:      1500,
	}
}

func createdTrip() trip.Trip {
	in := validInput()

	return trip.Trip{
		ID:         tripID,
		UserID:     in.UserID,
		DriverID:   in.DriverID,
		StartPoint: in.StartPoint,
		EndPoint:   in.EndPoint,
		Price:      in.Price,
		Status:     trip.StatusActive,
		StartedAt:  time.Date(2026, 11, 8, 12, 0, 0, 0, time.UTC),
	}
}

func serve(t *testing.T, uc *MockuseCase, body string) *httptest.ResponseRecorder {
	t.Helper()

	handler := create.New(uc, slog.New(slog.NewTextHandler(io.Discard, nil)))
	req := httptest.NewRequest(http.MethodPost, "/api/v1/trips", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	handler.Handle(rec, req)

	return rec
}

func decodeProblem(t *testing.T, rec *httptest.ResponseRecorder) api.Problem {
	t.Helper()

	var p api.Problem
	if err := json.NewDecoder(rec.Body).Decode(&p); err != nil {
		t.Fatalf("decode problem: %v", err)
	}

	return p
}

func TestHandler_Handle_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	uc := NewMockuseCase(ctrl)
	uc.EXPECT().Execute(gomock.Any(), validInput()).Return(createdTrip(), nil)

	rec := serve(t, uc, validBody)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusCreated)
	}
	if loc := rec.Header().Get("Location"); loc != "/api/v1/trips/"+tripID.String() {
		t.Errorf("location = %q, want %q", loc, "/api/v1/trips/"+tripID.String())
	}

	var got api.Trip
	if err := json.NewDecoder(rec.Body).Decode(&got); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if got.Id != tripID {
		t.Errorf("id = %s, want %s", got.Id, tripID)
	}
	if string(got.Status) != string(trip.StatusActive) {
		t.Errorf("status = %q, want %q", got.Status, trip.StatusActive)
	}
	if got.Price != 1500 {
		t.Errorf("price = %d, want 1500", got.Price)
	}
}

func TestHandler_Handle_InvalidBody(t *testing.T) {
	tests := []struct {
		name string
		body string
	}{
		{name: "empty body", body: ""},
		{name: "malformed json", body: "{"},
		{name: "not an object", body: "[]"},
		{name: "null", body: "null"},
		{name: "empty object", body: "{}"},
		{name: "unknown field", body: strings.Replace(validBody, `"price": 1500`, `"price": 1500, "extra": 1`, 1)},
		{name: "missing price", body: bodyWithoutPrice},
		{name: "invalid user id", body: strings.Replace(validBody, "22222222-2222-2222-2222-222222222222", "not-a-uuid", 1)},
		{name: "missing coordinate", body: strings.Replace(validBody, `"latitude": 55.75, `, "", 1)},
		{name: "price is a string", body: strings.Replace(validBody, `"price": 1500`, `"price": "1500"`, 1)},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			uc := NewMockuseCase(ctrl)

			rec := serve(t, uc, tc.body)

			if rec.Code != http.StatusBadRequest {
				t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
			}
			if p := decodeProblem(t, rec); p.Code != "invalid_request" {
				t.Errorf("code = %q, want invalid_request", p.Code)
			}
		})
	}
}

func TestHandler_Handle_UseCaseErrors(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		wantStatus int
		wantCode   string
	}{
		{
			name:       "validation error",
			err:        &trip.ValidationError{Field: "start_point.latitude", Reason: "must be within [-90, 90]"},
			wantStatus: http.StatusBadRequest,
			wantCode:   "invalid_request",
		},
		{
			name:       "driver busy",
			err:        trip.ErrDriverBusy,
			wantStatus: http.StatusConflict,
			wantCode:   "driver_busy",
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
			uc.EXPECT().Execute(gomock.Any(), validInput()).Return(trip.Trip{}, tc.err)

			rec := serve(t, uc, validBody)

			if rec.Code != tc.wantStatus {
				t.Fatalf("status = %d, want %d", rec.Code, tc.wantStatus)
			}
			if strings.Contains(rec.Body.String(), "secret") {
				t.Errorf("response leaks error details: %s", rec.Body.String())
			}
			if p := decodeProblem(t, rec); p.Code != tc.wantCode {
				t.Errorf("code = %q, want %q", p.Code, tc.wantCode)
			}
		})
	}
}
