package worker

import (
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"situkang/internal/httpx"
)

// Handler exposes the worker endpoints.
type Handler struct {
	svc *Service
}

// NewHandler builds the worker HTTP layer.
func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

type request struct {
	Name     string `json:"name"`
	Role     string `json:"role"`
	DailyFee int64  `json:"daily_fee"`
	Active   *bool  `json:"active"`
}

// List returns active workers, or all of them with include_inactive=1.
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	search := r.URL.Query().Get("q")
	includeInactive := r.URL.Query().Get("include_inactive") == "1"

	workers, err := h.svc.List(r.Context(), search, includeInactive)
	if err != nil {
		slog.Error("list workers failed", "error", err)
		httpx.Error(w, http.StatusInternalServerError, "Terjadi kesalahan pada server.")
		return
	}
	httpx.JSON(w, http.StatusOK, workers)
}

// Get returns one worker.
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	id, ok := workerID(w, r)
	if !ok {
		return
	}

	item, err := h.svc.Get(r.Context(), id)
	if err != nil {
		writeError(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, item)
}

// Create adds a worker.
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var req request
	if err := httpx.DecodeJSON(w, r, &req); err != nil {
		httpx.DecodeError(w, err)
		return
	}

	created, err := h.svc.Create(r.Context(), req.input())
	if err != nil {
		writeError(w, err)
		return
	}
	httpx.JSON(w, http.StatusCreated, created)
}

// Update replaces the editable fields of a worker.
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	id, ok := workerID(w, r)
	if !ok {
		return
	}

	var req request
	if err := httpx.DecodeJSON(w, r, &req); err != nil {
		httpx.DecodeError(w, err)
		return
	}

	updated, err := h.svc.Update(r.Context(), id, req.input())
	if err != nil {
		writeError(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, updated)
}

// Delete soft-deletes a worker.
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	id, ok := workerID(w, r)
	if !ok {
		return
	}

	if err := h.svc.Delete(r.Context(), id); err != nil {
		writeError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (r request) input() Input {
	return Input{
		Name:     r.Name,
		Role:     r.Role,
		DailyFee: r.DailyFee,
		Active:   r.Active,
	}
}

func workerID(w http.ResponseWriter, r *http.Request) (int64, bool) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil || id <= 0 {
		httpx.Error(w, http.StatusNotFound, "Pekerja tidak ditemukan.")
		return 0, false
	}
	return id, true
}

func writeError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ErrInvalidName):
		httpx.Error(w, http.StatusBadRequest, "Nama pekerja wajib diisi (maksimal 100 karakter).")
	case errors.Is(err, ErrInvalidRole):
		httpx.Error(w, http.StatusBadRequest, "Peran pekerja wajib diisi (maksimal 50 karakter).")
	case errors.Is(err, ErrInvalidFee):
		httpx.Error(w, http.StatusBadRequest, "Upah harian tidak boleh negatif.")
	case errors.Is(err, ErrNotFound), errors.Is(err, ErrInvalidInput):
		httpx.Error(w, http.StatusNotFound, "Pekerja tidak ditemukan.")
	default:
		slog.Error("worker request failed", "error", err)
		httpx.Error(w, http.StatusInternalServerError, "Terjadi kesalahan pada server.")
	}
}
