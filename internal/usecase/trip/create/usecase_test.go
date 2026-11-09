package create_test

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/google/uuid"
	"go.uber.org/mock/gomock"

	"github.com/Parcifallll/trip-go/internal/domain/trip"
	"github.com/Parcifallll/trip-go/internal/model"
	"github.com/Parcifallll/trip-go/internal/usecase/trip/create"
)

var (
	tripID   = uuid.MustParse("11111111-1111-1111-1111-111111111111")
	userID   = uuid.MustParse("22222222-2222-2222-2222-222222222222")
	driverID = uuid.MustParse("33333333-3333-3333-3333-333333333333")
	now      = time.Date(2026, 11, 8, 12, 0, 0, 0, time.UTC)
	errBoom  = errors.New("boom")
)

type fixture struct {
	trips   *MocktripRepository
	history *MockhistoryRepository
	tx      *MocktxManager
	metrics *Mockmetrics
	uc      *create.UseCase
}

func newFixture(t *testing.T) *fixture {
	t.Helper()

	ctrl := gomock.NewController(t)
	f := &fixture{
		trips:   NewMocktripRepository(ctrl),
		history: NewMockhistoryRepository(ctrl),
		tx:      NewMocktxManager(ctrl),
		metrics: NewMockmetrics(ctrl),
	}
	f.uc = create.New(
		f.trips, f.history, f.tx, f.metrics,
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		func() time.Time { return now },
		func() uuid.UUID { return tripID },
	)

	return f
}

func (f *fixture) expectTx() {
	f.tx.EXPECT().Do(gomock.Any(), gomock.Any()).DoAndReturn(
		func(ctx context.Context, fn func(context.Context) error) error {
			return fn(ctx)
		},
	)
}

func input() model.CreateTripInput {
	return model.CreateTripInput{
		UserID:     userID,
		DriverID:   driverID,
		StartPoint: trip.Coordinates{Latitude: 55.75, Longitude: 37.61},
		EndPoint:   trip.Coordinates{Latitude: 55.80, Longitude: 37.70},
		Price:      1500,
	}
}

func expectedTrip() trip.Trip {
	in := input()

	return trip.Trip{
		ID:         tripID,
		UserID:     in.UserID,
		DriverID:   in.DriverID,
		StartPoint: in.StartPoint,
		EndPoint:   in.EndPoint,
		Price:      in.Price,
		Status:     trip.StatusActive,
		StartedAt:  now,
	}
}

func TestUseCase_Execute_Success(t *testing.T) {
	f := newFixture(t)
	want := expectedTrip()

	f.expectTx()
	f.trips.EXPECT().Save(gomock.Any(), want).Return(nil)
	f.history.EXPECT().Add(gomock.Any(), model.StatusChange{
		TripID: tripID,
		To:     trip.StatusActive,
		Reason: "trip created",
	}).Return(nil)
	f.metrics.EXPECT().TripCreated(gomock.Any())

	got, err := f.uc.Execute(context.Background(), input())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != want {
		t.Errorf("trip = %+v, want %+v", got, want)
	}
}

func TestUseCase_Execute_DriverBusy(t *testing.T) {
	f := newFixture(t)

	f.expectTx()
	f.trips.EXPECT().Save(gomock.Any(), expectedTrip()).Return(trip.ErrDriverBusy)

	got, err := f.uc.Execute(context.Background(), input())

	if !errors.Is(err, trip.ErrDriverBusy) {
		t.Fatalf("error = %v, want %v", err, trip.ErrDriverBusy)
	}
	if got.ID != uuid.Nil {
		t.Errorf("trip = %+v, want zero value", got)
	}
}

func TestUseCase_Execute_InvalidInput(t *testing.T) {
	f := newFixture(t)
	in := input()
	in.StartPoint.Latitude = 100

	_, err := f.uc.Execute(context.Background(), in)

	var validationErr *trip.ValidationError
	if !errors.As(err, &validationErr) {
		t.Fatalf("error = %v, want *ValidationError", err)
	}
	if validationErr.Field != "start_point.latitude" {
		t.Errorf("field = %q, want %q", validationErr.Field, "start_point.latitude")
	}
}

func TestUseCase_Execute_SaveError(t *testing.T) {
	f := newFixture(t)

	f.expectTx()
	f.trips.EXPECT().Save(gomock.Any(), expectedTrip()).Return(errBoom)

	_, err := f.uc.Execute(context.Background(), input())

	if !errors.Is(err, errBoom) {
		t.Fatalf("error = %v, want %v", err, errBoom)
	}
}

func TestUseCase_Execute_HistoryError(t *testing.T) {
	f := newFixture(t)

	f.expectTx()
	f.trips.EXPECT().Save(gomock.Any(), expectedTrip()).Return(nil)
	f.history.EXPECT().Add(gomock.Any(), gomock.Any()).Return(errBoom)

	_, err := f.uc.Execute(context.Background(), input())

	if !errors.Is(err, errBoom) {
		t.Fatalf("error = %v, want %v", err, errBoom)
	}
}
