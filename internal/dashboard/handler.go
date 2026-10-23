package dashboard

import (
	"log/slog"
	"net/http"

	"situkang/internal/httpx"
)

// Handler exposes the dashboard endpoint.
type Handler struct {
	svc *Service
}

// NewHandler builds the dashboard HTTP layer.
func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

// Get returns the Beranda unsettled summary.
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	result, err := h.svc.Get(r.Context())
	if err != nil {
		slog.Error("dashboard request failed", "error", err)
		httpx.Error(w, http.StatusInternalServerError, "Terjadi kesalahan pada server.")
		return
	}
	httpx.JSON(w, http.StatusOK, result)
}
