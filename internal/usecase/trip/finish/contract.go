//go:generate go tool mockgen -source ${GOFILE} -destination mocks_test.go -package ${GOPACKAGE}_test
package finish

import (
	"context"

	"github.com/google/uuid"

	"github.com/Parcifallll/trip-go/internal/domain/trip"
	"github.com/Parcifallll/trip-go/internal/model"
)

type (
	tripRepository interface {
		GetForUpdate(ctx context.Context, id uuid.UUID) (trip.Trip, error)
		Update(ctx context.Context, t trip.Trip) error
	}

	historyRepository interface {
		Add(ctx context.Context, change model.StatusChange) error
	}

	txManager interface {
		Do(ctx context.Context, fn func(ctx context.Context) error) error
	}

	metrics interface {
		TripCompleted(ctx context.Context)
	}
)
