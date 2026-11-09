package trip_test

import (
	"errors"
	"math"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/Parcifallll/trip-go/internal/domain/trip"
)

var (
	tripID    = uuid.MustParse("11111111-1111-1111-1111-111111111111")
	userID    = uuid.MustParse("22222222-2222-2222-2222-222222222222")
	driverID  = uuid.MustParse("33333333-3333-3333-3333-333333333333")
	startedAt = time.Date(2026, 11, 8, 12, 0, 0, 0, time.UTC)
	finishAt  = time.Date(2026, 11, 8, 12, 30, 0, 0, time.UTC)
)

func validTrip() trip.Trip {
	return trip.Trip{
		ID:         tripID,
		UserID:     userID,
		DriverID:   driverID,
		StartPoint: trip.Coordinates{Latitude: 55.75, Longitude: 37.61},
		EndPoint:   trip.Coordinates{Latitude: 55.80, Longitude: 37.70},
		Price:      1500,
		Status:     trip.StatusActive,
		StartedAt:  startedAt,
	}
}

func TestTrip_Finish(t *testing.T) {
	t.Run("active trip is completed", func(t *testing.T) {
		tr := validTrip()

		if err := tr.Finish(finishAt); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if tr.Status != trip.StatusCompleted {
			t.Errorf("status = %q, want %q", tr.Status, trip.StatusCompleted)
		}
		if tr.FinishedAt == nil || !tr.FinishedAt.Equal(finishAt) {
			t.Errorf("finished_at = %v, want %v", tr.FinishedAt, finishAt)
		}
	})

	t.Run("completed trip cannot be finished again", func(t *testing.T) {
		tr := validTrip()
		if err := tr.Finish(finishAt); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		later := finishAt.Add(time.Hour)
		err := tr.Finish(later)

		if !errors.Is(err, trip.ErrTripCompleted) {
			t.Fatalf("error = %v, want %v", err, trip.ErrTripCompleted)
		}
		if !tr.FinishedAt.Equal(finishAt) {
			t.Errorf("finished_at changed to %v, want %v", tr.FinishedAt, finishAt)
		}
	})
}

func TestNew(t *testing.T) {
	t.Run("valid data", func(t *testing.T) {
		want := validTrip()

		got, err := trip.New(want.ID, want.UserID, want.DriverID, want.StartPoint, want.EndPoint, want.Price, want.StartedAt)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got.Status != trip.StatusActive {
			t.Errorf("status = %q, want %q", got.Status, trip.StatusActive)
		}
		if got.FinishedAt != nil {
			t.Errorf("finished_at = %v, want nil", got.FinishedAt)
		}
		if got.ID != want.ID || got.UserID != want.UserID || got.DriverID != want.DriverID ||
			got.StartPoint != want.StartPoint || got.EndPoint != want.EndPoint ||
			got.Price != want.Price || !got.StartedAt.Equal(want.StartedAt) {
			t.Errorf("trip = %+v, want %+v", got, want)
		}
	})

	t.Run("invalid data", func(t *testing.T) {
		want := validTrip()

		got, err := trip.New(want.ID, want.UserID, want.DriverID, want.StartPoint, want.EndPoint, -1, want.StartedAt)

		var validationErr *trip.ValidationError
		if !errors.As(err, &validationErr) {
			t.Fatalf("error = %v, want *ValidationError", err)
		}
		if validationErr.Field != "price" {
			t.Errorf("field = %q, want %q", validationErr.Field, "price")
		}
		if got.ID != uuid.Nil {
			t.Errorf("trip = %+v, want zero value", got)
		}
	})
}

func TestTrip_Validate(t *testing.T) {
	tests := []struct {
		name      string
		mutate    func(tr *trip.Trip)
		wantField string
	}{
		{name: "valid active trip", mutate: func(tr *trip.Trip) {}},
		{
			name: "valid completed trip",
			mutate: func(tr *trip.Trip) {
				at := finishAt
				tr.Status = trip.StatusCompleted
				tr.FinishedAt = &at
			},
		},
		{
			name: "boundary coordinates",
			mutate: func(tr *trip.Trip) {
				tr.StartPoint = trip.Coordinates{Latitude: 90, Longitude: 180}
				tr.EndPoint = trip.Coordinates{Latitude: -90, Longitude: -180}
			},
		},
		{name: "zero price", mutate: func(tr *trip.Trip) { tr.Price = 0 }},
		{name: "empty id", mutate: func(tr *trip.Trip) { tr.ID = uuid.Nil }, wantField: "id"},
		{name: "empty user id", mutate: func(tr *trip.Trip) { tr.UserID = uuid.Nil }, wantField: "user_id"},
		{name: "empty driver id", mutate: func(tr *trip.Trip) { tr.DriverID = uuid.Nil }, wantField: "driver_id"},
		{
			name:      "start latitude too high",
			mutate:    func(tr *trip.Trip) { tr.StartPoint.Latitude = 90.0001 },
			wantField: "start_point.latitude",
		},
		{
			name:      "start longitude too low",
			mutate:    func(tr *trip.Trip) { tr.StartPoint.Longitude = -180.0001 },
			wantField: "start_point.longitude",
		},
		{
			name:      "end latitude too low",
			mutate:    func(tr *trip.Trip) { tr.EndPoint.Latitude = -91 },
			wantField: "end_point.latitude",
		},
		{
			name:      "end longitude too high",
			mutate:    func(tr *trip.Trip) { tr.EndPoint.Longitude = 181 },
			wantField: "end_point.longitude",
		},
		{
			name:      "nan latitude",
			mutate:    func(tr *trip.Trip) { tr.StartPoint.Latitude = math.NaN() },
			wantField: "start_point.latitude",
		},
		{name: "negative price", mutate: func(tr *trip.Trip) { tr.Price = -1 }, wantField: "price"},
		{
			name: "active trip with finish time",
			mutate: func(tr *trip.Trip) {
				at := finishAt
				tr.FinishedAt = &at
			},
			wantField: "finished_at",
		},
		{
			name:      "completed trip without finish time",
			mutate:    func(tr *trip.Trip) { tr.Status = trip.StatusCompleted },
			wantField: "finished_at",
		},
		{
			name:      "unknown status",
			mutate:    func(tr *trip.Trip) { tr.Status = trip.Status("paused") },
			wantField: "status",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tr := validTrip()
			tc.mutate(&tr)

			err := tr.Validate()

			if tc.wantField == "" {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				return
			}

			var validationErr *trip.ValidationError
			if !errors.As(err, &validationErr) {
				t.Fatalf("error = %v, want *ValidationError", err)
			}
			if validationErr.Field != tc.wantField {
				t.Errorf("field = %q, want %q", validationErr.Field, tc.wantField)
			}
		})
	}
}
