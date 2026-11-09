package trip

import (
	"context"
	"fmt"
	"time"

	"github.com/Masterminds/squirrel"

	"github.com/Parcifallll/trip-go/internal/model"
)

type HistoryRepository struct {
	pool    executor
	sb      squirrel.StatementBuilderType
	timeout time.Duration
}

func NewHistory(pool executor, queryTimeout time.Duration) *HistoryRepository {
	return &HistoryRepository{
		pool:    pool,
		sb:      squirrel.StatementBuilder.PlaceholderFormat(squirrel.Dollar),
		timeout: queryTimeout,
	}
}

func (r *HistoryRepository) Add(ctx context.Context, change model.StatusChange) error {
	var from *string
	if change.From != nil {
		s := string(*change.From)
		from = &s
	}

	var reason *string
	if change.Reason != "" {
		reason = &change.Reason
	}

	query, args, err := r.sb.Insert(statusHistoryTable).
		Columns("trip_id", "from_status", "to_status", "reason").
		Values(change.TripID, from, string(change.To), reason).
		ToSql()
	if err != nil {
		return fmt.Errorf("build insert status history: %w", err)
	}

	ctx, cancel := context.WithTimeout(ctx, r.timeout)
	defer cancel()

	if _, err := executorFromContext(ctx, r.pool).Exec(ctx, query, args...); err != nil {
		return fmt.Errorf("insert status history: %w", err)
	}

	return nil
}
