package get_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"go.uber.org/mock/gomock"

	"github.com/Parcifallll/trip-go/internal/domain/trip"
	"github.com/Parcifallll/trip-go/internal/usecase/trip/get"
)

var (
	tripID  = uuid.MustParse("11111111-1111-1111-1111-111111111111")
	errBoom = errors.New("boom")
)

func TestUseCase_Execute(t *testing.T) {
	existing := trip.Trip{
		ID:        tripID,
		UserID:    uuid.MustParse("22222222-2222-2222-2222-222222222222"),
		DriverID:  uuid.MustParse("33333333-3333-3333-3333-333333333333"),
		Price:     1500,
		Status:    trip.StatusActive,
		StartedAt: time.Date(2026, 11, 8, 12, 0, 0, 0, time.UTC),
	}

	tests := []struct {
		name    string
		repoRes trip.Trip
		repoErr error
		want    trip.Trip
		wantErr error
	}{
		{name: "found", repoRes: existing, want: existing},
		{name: "not found", repoErr: trip.ErrTripNotFound, wantErr: trip.ErrTripNotFound},
		{name: "repository error", repoErr: errBoom, wantErr: errBoom},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			repo := NewMocktripRepository(ctrl)
			repo.EXPECT().Get(gomock.Any(), tripID).Return(tc.repoRes, tc.repoErr)

			got, err := get.New(repo).Execute(context.Background(), tripID)

			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("error = %v, want %v", err, tc.wantErr)
			}
			if got != tc.want {
				t.Errorf("trip = %+v, want %+v", got, tc.want)
			}
		})
	}
}
