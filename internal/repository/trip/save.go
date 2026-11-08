package trip

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5/pgconn"

	domain "github.com/Parcifallll/trip-go/internal/domain/trip"
)

const (
	pgUniqueViolation = "23505"
	driverActiveIndex = "trips_driver_active_idx"
)

func (r *Repository) Save(ctx context.Context, t domain.Trip) error {
	query, args, err := r.sb.Insert(tripsTable).
		Columns(
			colID, colUserID, colDriverID,
			colStartLatitude, colStartLongitude, colEndLatitude, colEndLongitude,
			colPrice, colStatus, colStartedAt,
		).
		Values(
			t.ID, t.UserID, t.DriverID,
			t.StartPoint.Latitude, t.StartPoint.Longitude, t.EndPoint.Latitude, t.EndPoint.Longitude,
			t.Price, string(t.Status), t.StartedAt.UTC(),
		).
		ToSql()
	if err != nil {
		return fmt.Errorf("build insert trip: %w", err)
	}

	ctx, cancel := context.WithTimeout(ctx, r.timeout)
	defer cancel()

	if _, err := r.executor(ctx).Exec(ctx, query, args...); err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == pgUniqueViolation && pgErr.ConstraintName == driverActiveIndex {
			return domain.ErrDriverBusy
		}
		return fmt.Errorf("insert trip: %w", err)
	}

	return nil
}

func (r *Repository) Update(ctx context.Context, t domain.Trip) error {
	var finishedAt any
	if t.FinishedAt != nil {
		finishedAt = t.FinishedAt.UTC()
	}

	query, args, err := r.sb.Update(tripsTable).
		Set(colStatus, string(t.Status)).
		Set(colFinishedAt, finishedAt).
		Set(colUpdatedAt, squirrelNow()).
		Where(eqID(t.ID)).
		ToSql()
	if err != nil {
		return fmt.Errorf("build update trip: %w", err)
	}

	ctx, cancel := context.WithTimeout(ctx, r.timeout)
	defer cancel()

	tag, err := r.executor(ctx).Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("update trip: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrTripNotFound
	}

	return nil
}
