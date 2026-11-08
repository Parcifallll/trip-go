package repository

import (
	"context"

	"github.com/Parcifallll/trip-go/internal/txmanager"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type executor interface {
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

func executorFromCtx(ctx context.Context, pool *pgxpool.Pool) executor {
	if tx := txmanager.GetTx(ctx); tx != nil {
		return tx
	}
	return pool
}
