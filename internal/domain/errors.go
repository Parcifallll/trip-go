package domain

import "errors"

var (
	ErrDriverBusy     = errors.New("driver already has an active trip")
	ErrTripNotFound   = errors.New("trip not found")
	ErrTripCompleted  = errors.New("trip already completed")
	ErrInvalidRequest = errors.New("invalid request")
)
