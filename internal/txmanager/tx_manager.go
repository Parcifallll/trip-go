package txmanager

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type TxManager interface {
	Do(ctx context.Context, fn func(ctx context.Context) error) error
}

type txManager struct {
	pool *pgxpool.Pool
}

func NewTxManager(pool *pgxpool.Pool) TxManager {
	return &txManager{pool: pool}
}

type txKey string

// TxKey is the context key for storing transaction
const TxKey txKey = "txmanager.tx"

func (m *txManager) Do(ctx context.Context, fn func(ctx context.Context) error) error {
	if tx := GetTx(ctx); tx != nil {
		return fn(ctx)
	}

	tx, err := m.pool.BeginTx(ctx, pgx.TxOptions{
		IsoLevel: pgx.ReadCommitted,
	})
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}

	rollbackCtx := context.WithoutCancel(ctx)

	defer func() {
		if p := recover(); p != nil {
			_ = tx.Rollback(rollbackCtx)
			panic(p)
		}
	}()

	if err := fn(context.WithValue(ctx, TxKey, tx)); err != nil {
		if rbErr := tx.Rollback(rollbackCtx); rbErr != nil {
			return fmt.Errorf("rollback: %w (original: %w)", rbErr, err)
		}
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit: %w", err)
	}

	return nil
}

func GetTx(ctx context.Context) pgx.Tx {
	if tx, ok := ctx.Value(TxKey).(pgx.Tx); ok {
		return tx
	}
	return nil
}
