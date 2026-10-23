package settings

import (
	"errors"
	"log/slog"
	"net/http"

	"situkang/internal/httpx"
)

// Handler exposes the settings endpoints.
type Handler struct {
	svc *Service
}

// NewHandler builds the settings HTTP layer.
func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

// Get returns the current settings.
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	result, err := h.svc.Get(r.Context())
	if err != nil {
		writeError(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, result)
}

type updateRequest struct {
	WorkingDays         *[]int `json:"working_days"`
	AttendanceFractions *[]int `json:"attendance_fractions"`
}

// Update replaces the working days.
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	var req updateRequest
	if err := httpx.DecodeJSON(w, r, &req); err != nil {
		httpx.DecodeError(w, err)
		return
	}

	result, err := h.svc.Update(r.Context(), req.WorkingDays, req.AttendanceFractions)
	if err != nil {
		writeError(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, result)
}

func writeError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ErrEmptyWorkingDays):
		httpx.Error(w, http.StatusBadRequest, "Pilih minimal satu hari kerja.")
	case errors.Is(err, ErrInvalidWorkingDays):
		httpx.Error(w, http.StatusBadRequest, "Hari kerja tidak valid.")
	case errors.Is(err, ErrInvalidFractions):
		httpx.Error(w, http.StatusBadRequest, "Bagian hari tidak valid.")
	default:
		slog.Error("settings request failed", "error", err)
		httpx.Error(w, http.StatusInternalServerError, "Terjadi kesalahan pada server.")
	}
}
