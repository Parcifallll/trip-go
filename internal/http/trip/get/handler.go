package get

import (
	"log/slog"
	"net/http"

	"github.com/google/uuid"

	"github.com/Parcifallll/trip-go/internal/http/domainerror"
	"github.com/Parcifallll/trip-go/internal/http/response"
	"github.com/Parcifallll/trip-go/internal/http/trip/dto"
)

type Handler struct {
	uc     useCase
	logger *slog.Logger
}

func New(uc useCase, logger *slog.Logger) *Handler {
	return &Handler{uc: uc, logger: logger}
}

func (h *Handler) Handle(w http.ResponseWriter, r *http.Request, id uuid.UUID) {
	found, err := h.uc.Execute(r.Context(), id)
	if err != nil {
		domainerror.Write(w, r, h.logger, err)
		return
	}

	response.JSON(w, http.StatusOK, dto.FromTrip(found))
}
