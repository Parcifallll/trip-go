package handler

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/Parcifallll/trip-go/internal/domain"
	"github.com/Parcifallll/trip-go/internal/generated"
	"github.com/Parcifallll/trip-go/internal/repository"
	"github.com/Parcifallll/trip-go/internal/txmanager"
	"github.com/google/uuid"
)

type TripHandler struct {
	txManager         txmanager.TxManager
	tripRepo          *repository.TripRepository
	statusHistoryRepo *repository.StatusHistoryRepository
}

func NewTripHandler(
	txManager txmanager.TxManager,
	tripRepo *repository.TripRepository,
	statusHistoryRepo *repository.StatusHistoryRepository,
) *TripHandler {
	return &TripHandler{
		txManager:         txManager,
		tripRepo:          tripRepo,
		statusHistoryRepo: statusHistoryRepo,
	}
}

func (h *TripHandler) CreateTrip(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	instance := "/api/v1/trips"

	body := api.CreateTripJSONRequestBody{}
	if err := decodeJSONBody(r, &body); err != nil {
		WriteError(w, domain.ErrInvalidRequest, instance)
		return
	}

	userID, err := uuid.Parse(body.UserId.String())
	if err != nil {
		WriteError(w, domain.ErrInvalidRequest, instance)
		return
	}
	driverID, err := uuid.Parse(body.DriverId.String())
	if err != nil {
		WriteError(w, domain.ErrInvalidRequest, instance)
		return
	}

	if !validLatLon(body.StartPoint.Latitude, body.StartPoint.Longitude) ||
		!validLatLon(body.EndPoint.Latitude, body.EndPoint.Longitude) {
		WriteError(w, domain.ErrInvalidRequest, instance)
		return
	}

	if body.Price < 0 {
		WriteError(w, domain.ErrInvalidRequest, instance)
		return
	}

	tripID := uuid.New()
	now := time.Now().UTC()

	err = h.txManager.Do(ctx, func(ctx context.Context) error {
		hasActive, err := h.tripRepo.CheckDriverActive(ctx, driverID)
		if err != nil {
			return err
		}
		if hasActive {
			return domain.ErrDriverBusy
		}

		trip := repository.Trip{
			ID:             tripID,
			UserID:         userID,
			DriverID:       driverID,
			StartLatitude:  body.StartPoint.Latitude,
			StartLongitude: body.StartPoint.Longitude,
			EndLatitude:    body.EndPoint.Latitude,
			EndLongitude:   body.EndPoint.Longitude,
			Price:          body.Price,
			Status:         "active",
			StartedAt:      now,
		}
		if err := h.tripRepo.Create(ctx, trip); err != nil {
			return err
		}

		reason := "trip created"
		if err := h.statusHistoryRepo.Insert(ctx, repository.StatusHistoryEntry{
			TripID:     tripID,
			FromStatus: nil,
			ToStatus:   "active",
			Reason:     &reason,
		}); err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		log.Printf("CreateTrip: transaction error: %v", err)
		WriteError(w, err, instance)
		return
	}

	createdTrip, err := h.tripRepo.Get(ctx, tripID)
	if err != nil {
		log.Printf("CreateTrip: get after create error: %v", err)
		WriteError(w, err, instance)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Location", "/api/v1/trips/"+tripID.String())
	w.WriteHeader(http.StatusCreated)
	_ = encodeJSON(w, toAPITrip(createdTrip))
}

func (h *TripHandler) GetTrip(w http.ResponseWriter, r *http.Request, tripID api.TripId) {
	ctx := r.Context()
	instance := "/api/v1/trips/" + tripID.String()

	id, err := uuid.Parse(tripID.String())
	if err != nil {
		WriteError(w, domain.ErrInvalidRequest, instance)
		return
	}

	trip, err := h.tripRepo.Get(ctx, id)
	if err != nil {
		log.Printf("GetTrip: repo error: %v", err)
		WriteError(w, err, instance)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = encodeJSON(w, toAPITrip(trip))
}

func (h *TripHandler) FinishTrip(w http.ResponseWriter, r *http.Request, tripID api.TripId) {
	ctx := r.Context()
	instance := "/api/v1/trips/" + tripID.String() + "/finish"

	id, err := uuid.Parse(tripID.String())
	if err != nil {
		WriteError(w, domain.ErrInvalidRequest, instance)
		return
	}

	finishedAt := time.Now().UTC()

	err = h.txManager.Do(ctx, func(ctx context.Context) error {
		// Finish trip (atomic check + update)
		if err := h.tripRepo.Finish(ctx, id, finishedAt); err != nil {
			return err
		}

		reason := "trip finished by driver"
		if err := h.statusHistoryRepo.Insert(ctx, repository.StatusHistoryEntry{
			TripID:     id,
			FromStatus: strPtr("active"),
			ToStatus:   "completed",
			Reason:     &reason,
		}); err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		log.Printf("FinishTrip: transaction error: %v", err)
		WriteError(w, err, instance)
		return
	}

	trip, err := h.tripRepo.Get(ctx, id)
	if err != nil {
		log.Printf("FinishTrip: get after finish error: %v", err)
		WriteError(w, err, instance)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = encodeJSON(w, toAPITrip(trip))
}

func validLatLon(lat, lon float64) bool {
	return lat >= -90 && lat <= 90 && lon >= -180 && lon <= 180
}

func strPtr(s string) *string {
	return &s
}

func toAPITrip(t *repository.Trip) api.Trip {
	return api.Trip{
		Id:             t.ID,
		UserId:         t.UserID,
		DriverId:       t.DriverID,
		StartPoint:     api.Coordinates{Latitude: t.StartLatitude, Longitude: t.StartLongitude},
		EndPoint:       api.Coordinates{Latitude: t.EndLatitude, Longitude: t.EndLongitude},
		Price:          t.Price,
		Status:         api.TripStatus(t.Status),
		StartedAt:      t.StartedAt,
		FinishedAt:     t.FinishedAt,
		LastPositionAt: t.LastPositionAt,
	}
}

func decodeJSONBody(r *http.Request, v any) error {
	return json.NewDecoder(r.Body).Decode(v)
}

func encodeJSON(w http.ResponseWriter, v any) error {
	return json.NewEncoder(w).Encode(v)
}
