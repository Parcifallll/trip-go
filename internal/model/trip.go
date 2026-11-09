package model

import (
	"github.com/google/uuid"

	"github.com/Parcifallll/trip-go/internal/domain/trip"
)

type CreateTripInput struct {
	UserID     uuid.UUID
	DriverID   uuid.UUID
	StartPoint trip.Coordinates
	EndPoint   trip.Coordinates
	Price      int64
}

type StatusChange struct {
	TripID uuid.UUID
	From   *trip.Status
	To     trip.Status
	Reason string
}
