package create

import (
	"log/slog"
	"net/http"

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

func (h *Handler) Handle(w http.ResponseWriter, r *http.Request) {
	input, err := parseRequest(w, r)
	if err != nil {
		domainerror.Write(w, r, h.logger, err)
		return
	}

	created, err := h.uc.Execute(r.Context(), input)
	if err != nil {
		domainerror.Write(w, r, h.logger, err)
		return
	}

	w.Header().Set("Location", "/api/v1/trips/"+created.ID.String())
	response.JSON(w, http.StatusCreated, dto.FromTrip(created))
}
