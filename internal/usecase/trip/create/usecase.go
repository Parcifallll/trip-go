package create

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"

	"github.com/Parcifallll/trip-go/internal/domain/trip"
	"github.com/Parcifallll/trip-go/internal/model"
)

const reasonCreated = "trip created"

type UseCase struct {
	trips   tripRepository
	history historyRepository
	tx      txManager
	metrics metrics
	logger  *slog.Logger
	now     func() time.Time
	newID   func() uuid.UUID
}

func New(
	trips tripRepository,
	history historyRepository,
	tx txManager,
	metrics metrics,
	logger *slog.Logger,
	now func() time.Time,
	newID func() uuid.UUID,
) *UseCase {
	return &UseCase{
		trips:   trips,
		history: history,
		tx:      tx,
		metrics: metrics,
		logger:  logger,
		now:     now,
		newID:   newID,
	}
}

func (u *UseCase) Execute(ctx context.Context, in model.CreateTripInput) (trip.Trip, error) {
	startedAt := u.now().UTC().Truncate(time.Microsecond)

	t, err := trip.New(u.newID(), in.UserID, in.DriverID, in.StartPoint, in.EndPoint, in.Price, startedAt)
	if err != nil {
		return trip.Trip{}, fmt.Errorf("new trip: %w", err)
	}

	err = u.tx.Do(ctx, func(ctx context.Context) error {
		if err := u.trips.Save(ctx, t); err != nil {
			return fmt.Errorf("save trip: %w", err)
		}

		change := model.StatusChange{TripID: t.ID, To: t.Status, Reason: reasonCreated}
		if err := u.history.Add(ctx, change); err != nil {
			return fmt.Errorf("add status history: %w", err)
		}

		return nil
	})
	if err != nil {
		return trip.Trip{}, fmt.Errorf("create trip: %w", err)
	}

	u.metrics.TripCreated(ctx)
	u.logger.InfoContext(ctx, "trip created",
		slog.String("trip_id", t.ID.String()),
		slog.String("driver_id", t.DriverID.String()),
	)

	return t, nil
}
