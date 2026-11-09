package trip

import (
	"context"
	"time"

	"github.com/Masterminds/squirrel"

	"github.com/Parcifallll/trip-go/internal/infra/postgres/transaction"
)

const (
	tripsTable         = "trips"
	statusHistoryTable = "trip_status_history"

	colID             = "id"
	colUserID         = "user_id"
	colDriverID       = "driver_id"
	colStartLatitude  = "start_latitude"
	colStartLongitude = "start_longitude"
	colEndLatitude    = "end_latitude"
	colEndLongitude   = "end_longitude"
	colPrice          = "price"
	colStatus         = "status"
	colStartedAt      = "started_at"
	colFinishedAt     = "finished_at"
	colLastPositionAt = "last_position_at"
	colUpdatedAt      = "updated_at"
)

type Repository struct {
	pool    executor
	sb      squirrel.StatementBuilderType
	timeout time.Duration
}

func New(pool executor, queryTimeout time.Duration) *Repository {
	return &Repository{
		pool:    pool,
		sb:      squirrel.StatementBuilder.PlaceholderFormat(squirrel.Dollar),
		timeout: queryTimeout,
	}
}

func (r *Repository) executor(ctx context.Context) executor {
	return executorFromContext(ctx, r.pool)
}

func executorFromContext(ctx context.Context, pool executor) executor {
	if tx := transaction.GetTx(ctx); tx != nil {
		return tx
	}
	return pool
}
