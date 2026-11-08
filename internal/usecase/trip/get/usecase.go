package get

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/Parcifallll/trip-go/internal/domain/trip"
)

type UseCase struct {
	trips tripRepository
}

func New(trips tripRepository) *UseCase {
	return &UseCase{trips: trips}
}

func (u *UseCase) Execute(ctx context.Context, id uuid.UUID) (trip.Trip, error) {
	t, err := u.trips.Get(ctx, id)
	if err != nil {
		return trip.Trip{}, fmt.Errorf("get trip: %w", err)
	}
	return t, nil
}
