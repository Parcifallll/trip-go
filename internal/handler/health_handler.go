package handler

import (
	"context"
	"net/http"
	"time"

	"github.com/Parcifallll/trip-go/internal/generated"
)

type Pinger interface {
	Ping(ctx context.Context) error
}

type HealthHandler struct {
	db Pinger
}

func NewHealthHandler(db Pinger) *HealthHandler {
	return &HealthHandler{db: db}
}

func (h *HealthHandler) Health(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	_ = encodeJSON(w, api.HealthResponse{Status: api.Ok})
}

func (h *HealthHandler) Ready(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()

	err := h.db.Ping(ctx)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusServiceUnavailable)
		_ = encodeJSON(w, api.HealthResponse{Status: api.Unavailable})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = encodeJSON(w, api.HealthResponse{Status: api.Ok})
}
