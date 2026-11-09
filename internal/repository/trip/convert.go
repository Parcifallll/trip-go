package trip

import (
	"time"

	"github.com/jackc/pgx/v5"

	domain "github.com/Parcifallll/trip-go/internal/domain/trip"
)

var tripColumns = []string{
	colID, colUserID, colDriverID,
	colStartLatitude, colStartLongitude, colEndLatitude, colEndLongitude,
	colPrice, colStatus, colStartedAt, colFinishedAt, colLastPositionAt,
}

func scanTrip(row pgx.Row) (domain.Trip, error) {
	var (
		t              domain.Trip
		status         string
		finishedAt     *time.Time
		lastPositionAt *time.Time
	)

	err := row.Scan(
		&t.ID, &t.UserID, &t.DriverID,
		&t.StartPoint.Latitude, &t.StartPoint.Longitude, &t.EndPoint.Latitude, &t.EndPoint.Longitude,
		&t.Price, &status, &t.StartedAt, &finishedAt, &lastPositionAt,
	)
	if err != nil {
		return domain.Trip{}, err
	}

	t.Status = domain.Status(status)
	t.StartedAt = t.StartedAt.UTC()
	t.FinishedAt = utcPtr(finishedAt)
	t.LastPositionAt = utcPtr(lastPositionAt)

	return t, nil
}

func utcPtr(v *time.Time) *time.Time {
	if v == nil {
		return nil
	}
	utc := v.UTC()
	return &utc
}
