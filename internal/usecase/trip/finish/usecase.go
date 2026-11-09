package finish

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"

	"github.com/Parcifallll/trip-go/internal/domain/trip"
	"github.com/Parcifallll/trip-go/internal/model"
)

const reasonFinished = "trip finished by driver"

type UseCase struct {
	trips   tripRepository
	history historyRepository
	tx      txManager
	metrics metrics
	logger  *slog.Logger
	now     func() time.Time
}

func New(
	trips tripRepository,
	history historyRepository,
	tx txManager,
	metrics metrics,
	logger *slog.Logger,
	now func() time.Time,
) *UseCase {
	return &UseCase{
		trips:   trips,
		history: history,
		tx:      tx,
		metrics: metrics,
		logger:  logger,
		now:     now,
	}
}

func (u *UseCase) Execute(ctx context.Context, id uuid.UUID) (trip.Trip, error) {
	var result trip.Trip

	err := u.tx.Do(ctx, func(ctx context.Context) error {
		t, err := u.trips.GetForUpdate(ctx, id)
		if err != nil {
			return fmt.Errorf("get trip for update: %w", err)
		}

		from := t.Status
		if err := t.Finish(u.now().UTC().Truncate(time.Microsecond)); err != nil {
			return fmt.Errorf("finish trip: %w", err)
		}

		if err := u.trips.Update(ctx, t); err != nil {
			return fmt.Errorf("update trip: %w", err)
		}

		change := model.StatusChange{TripID: t.ID, From: &from, To: t.Status, Reason: reasonFinished}
		if err := u.history.Add(ctx, change); err != nil {
			return fmt.Errorf("add status history: %w", err)
		}

		result = t
		return nil
	})
	if err != nil {
		return trip.Trip{}, err
	}

	u.metrics.TripCompleted(ctx)
	u.logger.InfoContext(ctx, "trip completed", slog.String("trip_id", result.ID.String()))

	return result, nil
}
