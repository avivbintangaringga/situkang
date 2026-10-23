package attendance

import (
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"situkang/internal/httpx"
)

// Handler exposes the attendance endpoints.
type Handler struct {
	svc *Service
}

// NewHandler builds the attendance HTTP layer.
func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

// Day returns the attendance screen data, defaulting to today.
func (h *Handler) Day(w http.ResponseWriter, r *http.Request) {
	date := r.URL.Query().Get("date")
	if date == "" {
		date = h.svc.Today()
	}

	result, err := h.svc.Day(r.Context(), date)
	if err != nil {
		writeError(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, result)
}

type saveDayRequest struct {
	Items []Item `json:"items"`
}

// SaveDay replaces one day's attendance.
func (h *Handler) SaveDay(w http.ResponseWriter, r *http.Request) {
	date := chi.URLParam(r, "date")

	var req saveDayRequest
	if err := httpx.DecodeJSON(w, r, &req); err != nil {
		httpx.DecodeError(w, err)
		return
	}
	if req.Items == nil {
		req.Items = []Item{}
	}

	result, err := h.svc.SaveDay(r.Context(), date, req.Items)
	if err != nil {
		writeError(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, result)
}

// Week returns one worker's attendance for a date range.
func (h *Handler) Week(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()

	workerID, err := strconv.ParseInt(query.Get("worker_id"), 10, 64)
	if err != nil || workerID <= 0 {
		httpx.Error(w, http.StatusBadRequest, "Pekerja tidak ditemukan.")
		return
	}

	entries, err := h.svc.Week(r.Context(), workerID, query.Get("from"), query.Get("to"))
	if err != nil {
		writeError(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, entries)
}

// Calendar returns recorded attendance for one month.
func (h *Handler) Calendar(w http.ResponseWriter, r *http.Request) {
	result, err := h.svc.Calendar(r.Context(), r.URL.Query().Get("month"))
	if err != nil {
		writeError(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, result)
}

func writeError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ErrLocked):
		httpx.Error(w, http.StatusConflict,
			"Data sudah masuk rekap dan terkunci. Buka kembali rekap untuk mengubahnya.")
	case errors.Is(err, ErrInvalidDate), errors.Is(err, ErrInvalidRange):
		httpx.Error(w, http.StatusBadRequest, "Tanggal tidak valid.")
	case errors.Is(err, ErrInvalidMonth):
		httpx.Error(w, http.StatusBadRequest, "Bulan tidak valid.")
	case errors.Is(err, ErrInvalidQuarters):
		httpx.Error(w, http.StatusBadRequest, "Bagian hari harus antara 1 dan 4.")
	case errors.Is(err, ErrInvalidWorker):
		httpx.Error(w, http.StatusBadRequest, "Pekerja tidak ditemukan atau sudah nonaktif.")
	case errors.Is(err, ErrDuplicateWorker):
		httpx.Error(w, http.StatusBadRequest, "Pekerja muncul lebih dari sekali.")
	default:
		slog.Error("attendance request failed", "error", err)
		httpx.Error(w, http.StatusInternalServerError, "Terjadi kesalahan pada server.")
	}
}
