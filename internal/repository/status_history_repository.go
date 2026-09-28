package repository

import (
	"context"
	"fmt"

	"github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type StatusHistoryRepository struct {
	pool *pgxpool.Pool
	sb   squirrel.StatementBuilderType
}

func NewStatusHistoryRepository(pool *pgxpool.Pool) *StatusHistoryRepository {
	return &StatusHistoryRepository{
		pool: pool,
		sb:   squirrel.StatementBuilder.PlaceholderFormat(squirrel.Dollar),
	}
}

type StatusHistoryEntry struct {
	TripID     uuid.UUID
	FromStatus *string
	ToStatus   string
	Reason     *string
}

func (r *StatusHistoryRepository) Insert(ctx context.Context, entry StatusHistoryEntry) error {
	sql, args, err := r.sb.Insert("trip_status_history").
		Columns("trip_id", "from_status", "to_status", "reason").
		Values(entry.TripID, entry.FromStatus, entry.ToStatus, entry.Reason).
		ToSql()
	if err != nil {
		return fmt.Errorf("build insert status history: %w", err)
	}

	_, err = r.getExecutor(ctx).Exec(ctx, sql, args...)
	if err != nil {
		return fmt.Errorf("insert status history: %w", err)
	}
	return nil
}

func (r *StatusHistoryRepository) getExecutor(ctx context.Context) executor {
	return executorFromCtx(ctx, r.pool)
}
