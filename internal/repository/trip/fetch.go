package trip

import (
	"context"
	"errors"
	"fmt"

	"github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	domain "github.com/Parcifallll/trip-go/internal/domain/trip"
)

func eqID(id uuid.UUID) squirrel.Eq {
	return squirrel.Eq{colID: id}
}

func squirrelNow() squirrel.Sqlizer {
	return squirrel.Expr("now()")
}

func (r *Repository) Get(ctx context.Context, id uuid.UUID) (domain.Trip, error) {
	return r.fetch(ctx, id, false)
}

func (r *Repository) GetForUpdate(ctx context.Context, id uuid.UUID) (domain.Trip, error) {
	return r.fetch(ctx, id, true)
}

func (r *Repository) fetch(ctx context.Context, id uuid.UUID, forUpdate bool) (domain.Trip, error) {
	builder := r.sb.Select(tripColumns...).From(tripsTable).Where(eqID(id))
	if forUpdate {
		builder = builder.Suffix("FOR UPDATE")
	}

	query, args, err := builder.ToSql()
	if err != nil {
		return domain.Trip{}, fmt.Errorf("build select trip: %w", err)
	}

	ctx, cancel := context.WithTimeout(ctx, r.timeout)
	defer cancel()

	t, err := scanTrip(r.executor(ctx).QueryRow(ctx, query, args...))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.Trip{}, domain.ErrTripNotFound
		}
		return domain.Trip{}, fmt.Errorf("scan trip: %w", err)
	}

	return t, nil
}
