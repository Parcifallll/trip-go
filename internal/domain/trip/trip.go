package trip

import (
	"math"
	"time"

	"github.com/google/uuid"
)

type Status string

const (
	StatusActive    Status = "active"
	StatusCompleted Status = "completed"
)
const reasonEmpty = "must not be empty"

const (
	minLatitude  = -90.0
	maxLatitude  = 90.0
	minLongitude = -180.0
	maxLongitude = 180.0
)

type Coordinates struct {
	Latitude  float64
	Longitude float64
}

func (c Coordinates) Validate(field string) error {
	if math.IsNaN(c.Latitude) || c.Latitude < minLatitude || c.Latitude > maxLatitude {
		return &ValidationError{Field: field + ".latitude", Reason: "must be within [-90, 90]"}
	}
	if math.IsNaN(c.Longitude) || c.Longitude < minLongitude || c.Longitude > maxLongitude {
		return &ValidationError{Field: field + ".longitude", Reason: "must be within [-180, 180]"}
	}
	return nil
}

type Trip struct {
	ID             uuid.UUID
	UserID         uuid.UUID
	DriverID       uuid.UUID
	StartPoint     Coordinates
	EndPoint       Coordinates
	Price          int64
	Status         Status
	StartedAt      time.Time
	FinishedAt     *time.Time
	LastPositionAt *time.Time
}

func New(
	id, userID, driverID uuid.UUID,
	start, end Coordinates,
	price int64,
	startedAt time.Time,
) (Trip, error) {
	t := Trip{
		ID:         id,
		UserID:     userID,
		DriverID:   driverID,
		StartPoint: start,
		EndPoint:   end,
		Price:      price,
		Status:     StatusActive,
		StartedAt:  startedAt,
	}
	if err := t.Validate(); err != nil {
		return Trip{}, err
	}
	return t, nil
}

func (t *Trip) Validate() error {
	if t.ID == uuid.Nil {
		return &ValidationError{Field: "id", Reason: reasonEmpty}
	}
	if t.UserID == uuid.Nil {
		return &ValidationError{Field: "user_id", Reason: reasonEmpty}
	}
	if t.DriverID == uuid.Nil {
		return &ValidationError{Field: "driver_id", Reason: reasonEmpty}
	}
	if err := t.StartPoint.Validate("start_point"); err != nil {
		return err
	}
	if err := t.EndPoint.Validate("end_point"); err != nil {
		return err
	}
	if t.Price < 0 {
		return &ValidationError{Field: "price", Reason: "must be greater than or equal to 0"}
	}

	switch t.Status {
	case StatusActive:
		if t.FinishedAt != nil {
			return &ValidationError{Field: "finished_at", Reason: "must be empty for an active trip"}
		}
	case StatusCompleted:
		if t.FinishedAt == nil {
			return &ValidationError{Field: "finished_at", Reason: "must be set for a completed trip"}
		}
	default:
		return &ValidationError{Field: "status", Reason: "unknown status"}
	}

	return nil
}

func (t *Trip) Finish(at time.Time) error {
	if t.Status != StatusActive {
		return ErrTripCompleted
	}
	t.Status = StatusCompleted
	t.FinishedAt = &at
	return nil
}
