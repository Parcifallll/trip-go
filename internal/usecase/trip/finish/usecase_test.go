package finish_test

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
	"github.com/Parcifallll/trip-go/internal/usecase/trip/finish"
)

var (
	tripID    = uuid.MustParse("11111111-1111-1111-1111-111111111111")
	userID    = uuid.MustParse("22222222-2222-2222-2222-222222222222")
	driverID  = uuid.MustParse("33333333-3333-3333-3333-333333333333")
	startedAt = time.Date(2026, 11, 8, 12, 0, 0, 0, time.UTC)
	now       = time.Date(2026, 11, 8, 12, 30, 0, 0, time.UTC)
	errBoom   = errors.New("boom")
)

type fixture struct {
	trips   *MocktripRepository
	history *MockhistoryRepository
	tx      *MocktxManager
	metrics *Mockmetrics
	uc      *finish.UseCase
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
	f.uc = finish.New(
		f.trips, f.history, f.tx, f.metrics,
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		func() time.Time { return now },
	)

	f.tx.EXPECT().Do(gomock.Any(), gomock.Any()).DoAndReturn(
		func(ctx context.Context, fn func(context.Context) error) error {
			return fn(ctx)
		},
	)

	return f
}

func activeTrip() trip.Trip {
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

func completedTrip() trip.Trip {
	tr := activeTrip()
	finishedAt := now.Add(-time.Minute)
	tr.Status = trip.StatusCompleted
	tr.FinishedAt = &finishedAt

	return tr
}

func TestUseCase_Execute_Success(t *testing.T) {
	f := newFixture(t)

	finishedAt := now
	want := activeTrip()
	want.Status = trip.StatusCompleted
	want.FinishedAt = &finishedAt
	from := trip.StatusActive

	f.trips.EXPECT().GetForUpdate(gomock.Any(), tripID).Return(activeTrip(), nil)
	f.trips.EXPECT().Update(gomock.Any(), want).Return(nil)
	f.history.EXPECT().Add(gomock.Any(), model.StatusChange{
		TripID: tripID,
		From:   &from,
		To:     trip.StatusCompleted,
		Reason: "trip finished by driver",
	}).Return(nil)
	f.metrics.EXPECT().TripCompleted(gomock.Any())

	got, err := f.uc.Execute(context.Background(), tripID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Status != trip.StatusCompleted {
		t.Errorf("status = %q, want %q", got.Status, trip.StatusCompleted)
	}
	if got.FinishedAt == nil || !got.FinishedAt.Equal(now) {
		t.Errorf("finished_at = %v, want %v", got.FinishedAt, now)
	}
}

func TestUseCase_Execute_AlreadyCompleted(t *testing.T) {
	f := newFixture(t)

	f.trips.EXPECT().GetForUpdate(gomock.Any(), tripID).Return(completedTrip(), nil)

	_, err := f.uc.Execute(context.Background(), tripID)

	if !errors.Is(err, trip.ErrTripCompleted) {
		t.Fatalf("error = %v, want %v", err, trip.ErrTripCompleted)
	}
}

func TestUseCase_Execute_NotFound(t *testing.T) {
	f := newFixture(t)

	f.trips.EXPECT().GetForUpdate(gomock.Any(), tripID).Return(trip.Trip{}, trip.ErrTripNotFound)

	_, err := f.uc.Execute(context.Background(), tripID)

	if !errors.Is(err, trip.ErrTripNotFound) {
		t.Fatalf("error = %v, want %v", err, trip.ErrTripNotFound)
	}
}

func TestUseCase_Execute_UpdateError(t *testing.T) {
	f := newFixture(t)

	f.trips.EXPECT().GetForUpdate(gomock.Any(), tripID).Return(activeTrip(), nil)
	f.trips.EXPECT().Update(gomock.Any(), gomock.Any()).Return(errBoom)

	_, err := f.uc.Execute(context.Background(), tripID)

	if !errors.Is(err, errBoom) {
		t.Fatalf("error = %v, want %v", err, errBoom)
	}
}

func TestUseCase_Execute_HistoryError(t *testing.T) {
	f := newFixture(t)

	f.trips.EXPECT().GetForUpdate(gomock.Any(), tripID).Return(activeTrip(), nil)
	f.trips.EXPECT().Update(gomock.Any(), gomock.Any()).Return(nil)
	f.history.EXPECT().Add(gomock.Any(), gomock.Any()).Return(errBoom)

	_, err := f.uc.Execute(context.Background(), tripID)

	if !errors.Is(err, errBoom) {
		t.Fatalf("error = %v, want %v", err, errBoom)
	}
}
