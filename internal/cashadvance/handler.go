package cashadvance

import (
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"situkang/internal/httpx"
)

// Handler exposes the kas bon endpoints.
type Handler struct {
	svc *Service
}

// NewHandler builds the cash advance HTTP layer.
func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

// List returns cash advances, optionally filtered.
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()

	var workerID int64
	if raw := query.Get("worker_id"); raw != "" {
		parsed, err := strconv.ParseInt(raw, 10, 64)
		if err != nil || parsed <= 0 {
			httpx.Error(w, http.StatusBadRequest, "Pekerja tidak ditemukan.")
			return
		}
		workerID = parsed
	}

	items, err := h.svc.List(r.Context(), workerID, query.Get("from"), query.Get("to"))
	if err != nil {
		writeError(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, items)
}

type createRequest struct {
	WorkerID int64  `json:"worker_id"`
	Date     string `json:"date"`
	Amount   int64  `json:"amount"`
	Note     string `json:"note"`
}

// Create records a new kas bon.
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var req createRequest
	if err := httpx.DecodeJSON(w, r, &req); err != nil {
		httpx.DecodeError(w, err)
		return
	}

	created, err := h.svc.Create(r.Context(), Input{
		WorkerID: req.WorkerID,
		Date:     req.Date,
		Amount:   req.Amount,
		Note:     req.Note,
	})
	if err != nil {
		writeError(w, err)
		return
	}
	httpx.JSON(w, http.StatusCreated, created)
}

type updateRequest struct {
	Date   string `json:"date"`
	Amount int64  `json:"amount"`
	Note   string `json:"note"`
}

// Update edits a kas bon that is still unsettled.
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	id, ok := advanceID(w, r)
	if !ok {
		return
	}

	var req updateRequest
	if err := httpx.DecodeJSON(w, r, &req); err != nil {
		httpx.DecodeError(w, err)
		return
	}

	updated, err := h.svc.Update(r.Context(), id, UpdateInput{
		Date:   req.Date,
		Amount: req.Amount,
		Note:   req.Note,
	})
	if err != nil {
		writeError(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, updated)
}

// Delete removes an unsettled kas bon.
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	id, ok := advanceID(w, r)
	if !ok {
		return
	}

	if err := h.svc.Delete(r.Context(), id); err != nil {
		writeError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func advanceID(w http.ResponseWriter, r *http.Request) (int64, bool) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil || id <= 0 {
		httpx.Error(w, http.StatusNotFound, "Kas bon tidak ditemukan.")
		return 0, false
	}
	return id, true
}

func writeError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ErrLocked):
		httpx.Error(w, http.StatusConflict,
			"Data sudah masuk rekap dan terkunci. Buka kembali rekap untuk mengubahnya.")
	case errors.Is(err, ErrNotFound):
		httpx.Error(w, http.StatusNotFound, "Kas bon tidak ditemukan.")
	case errors.Is(err, ErrInvalidAmount):
		httpx.Error(w, http.StatusBadRequest, "Jumlah kas bon harus lebih dari nol.")
	case errors.Is(err, ErrInvalidDate):
		httpx.Error(w, http.StatusBadRequest, "Tanggal tidak valid.")
	case errors.Is(err, ErrInvalidWorker):
		httpx.Error(w, http.StatusBadRequest, "Pekerja tidak ditemukan.")
	case errors.Is(err, ErrNoteTooLong):
		httpx.Error(w, http.StatusBadRequest, "Catatan maksimal 200 karakter.")
	default:
		slog.Error("cash advance request failed", "error", err)
		httpx.Error(w, http.StatusInternalServerError, "Terjadi kesalahan pada server.")
	}
}
