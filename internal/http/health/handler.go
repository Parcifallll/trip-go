package health

import (
	"context"
	"net/http"
	"time"

	api "github.com/Parcifallll/trip-go/internal/generated"
	"github.com/Parcifallll/trip-go/internal/http/response"
)

type Handler struct {
	db          pinger
	pingTimeout time.Duration
}

func New(db pinger, pingTimeout time.Duration) *Handler {
	return &Handler{db: db, pingTimeout: pingTimeout}
}

func (h *Handler) Health(w http.ResponseWriter, _ *http.Request) {
	response.JSON(w, http.StatusOK, api.HealthResponse{Status: api.Ok})
}

func (h *Handler) Ready(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), h.pingTimeout)
	defer cancel()

	if err := h.db.Ping(ctx); err != nil {
		response.JSON(w, http.StatusServiceUnavailable, api.HealthResponse{Status: api.Unavailable})
		return
	}

	response.JSON(w, http.StatusOK, api.HealthResponse{Status: api.Ok})
}
