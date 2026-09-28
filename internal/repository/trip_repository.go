package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/Masterminds/squirrel"
	"github.com/Parcifallll/trip-go/internal/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

const pgUniqueViolation = "23505"

type TripRepository struct {
	pool *pgxpool.Pool
	sb   squirrel.StatementBuilderType
}

func NewTripRepository(pool *pgxpool.Pool) *TripRepository {
	return &TripRepository{
		pool: pool,
		sb:   squirrel.StatementBuilder.PlaceholderFormat(squirrel.Dollar),
	}
}

type Trip struct {
	ID             uuid.UUID
	UserID         uuid.UUID
	DriverID       uuid.UUID
	StartLatitude  float64
	StartLongitude float64
	EndLatitude    float64
	EndLongitude   float64
	Price          int64
	Status         string
	StartedAt      time.Time
	FinishedAt     *time.Time
	LastPositionAt *time.Time
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

func (r *TripRepository) Create(ctx context.Context, trip Trip) error {
	sql, args, err := r.sb.Insert("trips").
		Columns("id", "user_id", "driver_id", "start_latitude", "start_longitude",
			"end_latitude", "end_longitude", "price", "status", "started_at").
		Values(trip.ID, trip.UserID, trip.DriverID, trip.StartLatitude, trip.StartLongitude,
			trip.EndLatitude, trip.EndLongitude, trip.Price, trip.Status, trip.StartedAt.UTC()).
		ToSql()
	if err != nil {
		return fmt.Errorf("build insert trip: %w", err)
	}

	_, err = r.getExecutor(ctx).Exec(ctx, sql, args...)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == pgUniqueViolation {
			return domain.ErrDriverBusy
		}
		return fmt.Errorf("insert trip: %w", err)
	}
	return nil
}

func (r *TripRepository) Get(ctx context.Context, id uuid.UUID) (*Trip, error) {
	sql, args, err := r.sb.Select("id", "user_id", "driver_id", "start_latitude", "start_longitude",
		"end_latitude", "end_longitude", "price", "status", "started_at", "finished_at",
		"last_position_at", "created_at", "updated_at").
		From("trips").
		Where(squirrel.Eq{"id": id}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build select trip: %w", err)
	}

	row := r.getExecutor(ctx).QueryRow(ctx, sql, args...)

	var t Trip
	var finishedAt, lastPositionAt *time.Time
	err = row.Scan(&t.ID, &t.UserID, &t.DriverID, &t.StartLatitude, &t.StartLongitude,
		&t.EndLatitude, &t.EndLongitude, &t.Price, &t.Status, &t.StartedAt,
		&finishedAt, &lastPositionAt, &t.CreatedAt, &t.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrTripNotFound
		}
		return nil, fmt.Errorf("scan trip: %w", err)
	}
	t.FinishedAt = finishedAt
	t.LastPositionAt = lastPositionAt

	return &t, nil
}

func (r *TripRepository) Finish(ctx context.Context, id uuid.UUID, finishedAt time.Time) error {
	sql, args, err := r.sb.Update("trips").
		Set("status", "completed").
		Set("finished_at", finishedAt.UTC()).
		Set("updated_at", squirrel.Expr("now()")).
		Where(squirrel.Eq{"id": id}).
		Where(squirrel.Eq{"status": "active"}).
		ToSql()
	if err != nil {
		return fmt.Errorf("build finish trip: %w", err)
	}

	cmdTag, err := r.getExecutor(ctx).Exec(ctx, sql, args...)
	if err != nil {
		return fmt.Errorf("exec finish trip: %w", err)
	}

	if cmdTag.RowsAffected() == 0 {
		exists, err := r.Exists(ctx, id)
		if err != nil {
			return fmt.Errorf("check trip exists: %w", err)
		}
		if !exists {
			return domain.ErrTripNotFound
		}
		return domain.ErrTripCompleted
	}
	return nil
}

func (r *TripRepository) Exists(ctx context.Context, id uuid.UUID) (bool, error) {
	sql, args, err := r.sb.Select("1").From("trips").Where(squirrel.Eq{"id": id}).ToSql()
	if err != nil {
		return false, fmt.Errorf("build exists: %w", err)
	}

	var exists bool
	err = r.getExecutor(ctx).QueryRow(ctx, "SELECT EXISTS("+sql+")", args...).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("scan exists: %w", err)
	}
	return exists, nil
}

func (r *TripRepository) CheckDriverActive(ctx context.Context, driverID uuid.UUID) (bool, error) {
	sql, args, err := r.sb.Select("1").
		From("trips").
		Where(squirrel.Eq{"driver_id": driverID, "status": "active"}).
		ToSql()
	if err != nil {
		return false, fmt.Errorf("build check driver active: %w", err)
	}

	var exists bool
	err = r.getExecutor(ctx).QueryRow(ctx, "SELECT EXISTS("+sql+")", args...).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("scan check driver active: %w", err)
	}
	return exists, nil
}

func (r *TripRepository) getExecutor(ctx context.Context) executor {
	return executorFromCtx(ctx, r.pool)
}
