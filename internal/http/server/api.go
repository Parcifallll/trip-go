package server

import (
	"net/http"

	api "github.com/Parcifallll/trip-go/internal/generated"
)

type apiServer struct {
	handlers Handlers
}

func (s *apiServer) CreateTrip(w http.ResponseWriter, r *http.Request, _ api.CreateTripParams) {
	s.handlers.CreateTrip.Handle(w, r)
}

func (s *apiServer) GetTrip(w http.ResponseWriter, r *http.Request, tripID api.TripId) {
	s.handlers.GetTrip.Handle(w, r, tripID)
}

func (s *apiServer) FinishTrip(w http.ResponseWriter, r *http.Request, tripID api.TripId) {
	s.handlers.FinishTrip.Handle(w, r, tripID)
}

func (s *apiServer) Health(w http.ResponseWriter, r *http.Request) {
	s.handlers.Health.Health(w, r)
}

func (s *apiServer) Ready(w http.ResponseWriter, r *http.Request) {
	s.handlers.Health.Ready(w, r)
}

func (s *apiServer) ListTripPositions(w http.ResponseWriter, _ *http.Request, _ api.TripId) {
	w.WriteHeader(http.StatusNotImplemented)
}

func (s *apiServer) CreateTripPosition(w http.ResponseWriter, _ *http.Request, _ api.TripId) {
	w.WriteHeader(http.StatusNotImplemented)
}
