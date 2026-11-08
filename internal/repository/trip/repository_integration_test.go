//go:build integration

package trip_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Parcifallll/trip-go/internal/config"
	domain "github.com/Parcifallll/trip-go/internal/domain/trip"
	"github.com/Parcifallll/trip-go/internal/infra/postgres"
	"github.com/Parcifallll/trip-go/internal/infra/postgres/transaction"
	"github.com/Parcifallll/trip-go/internal/model"
	triprepo "github.com/Parcifallll/trip-go/internal/repository/trip"
)

var errBoom = errors.New("boom")

type env struct {
	pool    *pgxpool.Pool
	trips   *triprepo.Repository
	history *triprepo.HistoryRepository
	tx      *transaction.Manager
}

func newEnv(t *testing.T) *env {
	t.Helper()

	t.Chdir("../../..")

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	pool, err := postgres.NewPool(context.Background(), postgres.Config{
		URL:             cfg.DatabaseURL,
		MaxConns:        cfg.DatabaseMaxConns,
		MinConns:        cfg.DatabaseMinConns,
		MaxConnLifetime: cfg.DatabaseMaxConnLifetime,
		ConnectTimeout:  cfg.DatabaseConnectTimeout,
	}, nil)
	if err != nil {
		t.Fatalf("connect to database: %v", err)
	}
	t.Cleanup(pool.Close)

	return &env{
		pool:    pool,
		trips:   triprepo.New(pool, cfg.DatabaseQueryTimeout),
		history: triprepo.NewHistory(pool, cfg.DatabaseQueryTimeout),
		tx:      transaction.NewManager(pool),
	}
}

func (e *env) track(t *testing.T, ids ...uuid.UUID) {
	t.Helper()

	t.Cleanup(func() {
		for _, id := range ids {
			if _, err := e.pool.Exec(context.Background(), "DELETE FROM trips WHERE id = $1", id); err != nil {
				t.Errorf("cleanup trip %s: %v", id, err)
			}
		}
	})
}

func (e *env) historyCount(t *testing.T, tripID uuid.UUID) int {
	t.Helper()

	var n int
	err := e.pool.QueryRow(context.Background(),
		"SELECT count(*) FROM trip_status_history WHERE trip_id = $1", tripID).Scan(&n)
	if err != nil {
		t.Fatalf("count status history: %v", err)
	}

	return n
}

func newTrip() domain.Trip {
	return domain.Trip{
		ID:         uuid.New(),
		UserID:     uuid.New(),
		DriverID:   uuid.New(),
		StartPoint: domain.Coordinates{Latitude: 55.75, Longitude: 37.61},
		EndPoint:   domain.Coordinates{Latitude: 55.80, Longitude: 37.70},
		Price:      1500,
		Status:     domain.StatusActive,
		StartedAt:  time.Now().UTC().Truncate(time.Microsecond),
	}
}

func sameTimePtr(a, b *time.Time) bool {
	if a == nil || b == nil {
		return a == b
	}

	return a.Equal(*b)
}

func requireSameTrip(t *testing.T, want, got domain.Trip) {
	t.Helper()

	if got.ID != want.ID || got.UserID != want.UserID || got.DriverID != want.DriverID ||
		got.StartPoint != want.StartPoint || got.EndPoint != want.EndPoint ||
		got.Price != want.Price || got.Status != want.Status ||
		!got.StartedAt.Equal(want.StartedAt) ||
		!sameTimePtr(got.FinishedAt, want.FinishedAt) ||
		!sameTimePtr(got.LastPositionAt, want.LastPositionAt) {
		t.Fatalf("trip = %+v, want %+v", got, want)
	}
}

func TestRepository_SaveAndGet(t *testing.T) {
	e := newEnv(t)
	ctx := context.Background()
	want := newTrip()
	e.track(t, want.ID)

	if err := e.trips.Save(ctx, want); err != nil {
		t.Fatalf("save: %v", err)
	}

	got, err := e.trips.Get(ctx, want.ID)
	if err != nil {
		t.Fatalf("get: %v", err)
	}

	requireSameTrip(t, want, got)
}

func TestRepository_GetNotFound(t *testing.T) {
	e := newEnv(t)

	_, err := e.trips.Get(context.Background(), uuid.New())

	if !errors.Is(err, domain.ErrTripNotFound) {
		t.Fatalf("error = %v, want %v", err, domain.ErrTripNotFound)
	}
}

func TestRepository_SaveDriverBusy(t *testing.T) {
	e := newEnv(t)
	ctx := context.Background()

	first := newTrip()
	second := newTrip()
	second.DriverID = first.DriverID
	e.track(t, first.ID, second.ID)

	if err := e.trips.Save(ctx, first); err != nil {
		t.Fatalf("save first: %v", err)
	}

	err := e.trips.Save(ctx, second)

	if !errors.Is(err, domain.ErrDriverBusy) {
		t.Fatalf("error = %v, want %v", err, domain.ErrDriverBusy)
	}
}

func TestRepository_SaveDriverFreeAfterFinish(t *testing.T) {
	e := newEnv(t)
	ctx := context.Background()

	first := newTrip()
	second := newTrip()
	second.DriverID = first.DriverID
	e.track(t, first.ID, second.ID)

	if err := e.trips.Save(ctx, first); err != nil {
		t.Fatalf("save first: %v", err)
	}

	finishedAt := first.StartedAt.Add(time.Minute)
	if err := first.Finish(finishedAt); err != nil {
		t.Fatalf("finish: %v", err)
	}
	if err := e.trips.Update(ctx, first); err != nil {
		t.Fatalf("update: %v", err)
	}

	if err := e.trips.Save(ctx, second); err != nil {
		t.Fatalf("save second: %v", err)
	}
}

func TestRepository_Update(t *testing.T) {
	e := newEnv(t)
	ctx := context.Background()
	want := newTrip()
	e.track(t, want.ID)

	if err := e.trips.Save(ctx, want); err != nil {
		t.Fatalf("save: %v", err)
	}

	if err := want.Finish(want.StartedAt.Add(time.Minute)); err != nil {
		t.Fatalf("finish: %v", err)
	}
	if err := e.trips.Update(ctx, want); err != nil {
		t.Fatalf("update: %v", err)
	}

	got, err := e.trips.Get(ctx, want.ID)
	if err != nil {
		t.Fatalf("get: %v", err)
	}

	requireSameTrip(t, want, got)
}

func TestRepository_UpdateNotFound(t *testing.T) {
	e := newEnv(t)

	err := e.trips.Update(context.Background(), newTrip())

	if !errors.Is(err, domain.ErrTripNotFound) {
		t.Fatalf("error = %v, want %v", err, domain.ErrTripNotFound)
	}
}

func TestRepository_GetForUpdateInTransaction(t *testing.T) {
	e := newEnv(t)
	ctx := context.Background()
	want := newTrip()
	e.track(t, want.ID)

	if err := e.trips.Save(ctx, want); err != nil {
		t.Fatalf("save: %v", err)
	}

	err := e.tx.Do(ctx, func(ctx context.Context) error {
		got, err := e.trips.GetForUpdate(ctx, want.ID)
		if err != nil {
			return err
		}
		requireSameTrip(t, want, got)

		return nil
	})
	if err != nil {
		t.Fatalf("transaction: %v", err)
	}
}

func TestTransaction_Commit(t *testing.T) {
	e := newEnv(t)
	ctx := context.Background()
	tr := newTrip()
	e.track(t, tr.ID)

	err := e.tx.Do(ctx, func(ctx context.Context) error {
		if err := e.trips.Save(ctx, tr); err != nil {
			return err
		}

		return e.history.Add(ctx, model.StatusChange{TripID: tr.ID, To: domain.StatusActive, Reason: "trip created"})
	})
	if err != nil {
		t.Fatalf("transaction: %v", err)
	}

	got, err := e.trips.Get(ctx, tr.ID)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	requireSameTrip(t, tr, got)

	if n := e.historyCount(t, tr.ID); n != 1 {
		t.Errorf("history records = %d, want 1", n)
	}
}

func TestTransaction_RollbackOnError(t *testing.T) {
	tests := []struct {
		name   string
		second func(ctx context.Context, e *env, tr domain.Trip) error
	}{
		{
			name: "error after both inserts",
			second: func(ctx context.Context, e *env, tr domain.Trip) error {
				err := e.history.Add(ctx, model.StatusChange{TripID: tr.ID, To: domain.StatusActive, Reason: "trip created"})
				if err != nil {
					return err
				}

				return errBoom
			},
		},
		{
			name: "second insert fails",
			second: func(ctx context.Context, e *env, tr domain.Trip) error {
				return e.history.Add(ctx, model.StatusChange{TripID: tr.ID, To: domain.Status("unknown")})
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			e := newEnv(t)
			ctx := context.Background()
			tr := newTrip()
			e.track(t, tr.ID)

			err := e.tx.Do(ctx, func(ctx context.Context) error {
				if err := e.trips.Save(ctx, tr); err != nil {
					return err
				}

				return tc.second(ctx, e, tr)
			})
			if err == nil {
				t.Fatal("expected an error, got nil")
			}

			if _, err := e.trips.Get(ctx, tr.ID); !errors.Is(err, domain.ErrTripNotFound) {
				t.Errorf("trip must be rolled back, get error = %v", err)
			}
			if n := e.historyCount(t, tr.ID); n != 0 {
				t.Errorf("history records = %d, want 0", n)
			}
		})
	}
}
