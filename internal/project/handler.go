package project

import (
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"situkang/internal/httpx"
)

// Handler exposes the project endpoints.
type Handler struct {
	svc *Service
}

// NewHandler builds the project HTTP layer.
func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

// List returns every project.
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	items, err := h.svc.List(r.Context())
	if err != nil {
		writeError(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, items)
}

// Get returns one project.
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	id, ok := projectID(w, r)
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

type request struct {
	Name      string `json:"name"`
	StartDate string `json:"start_date"`
	EndDate   string `json:"end_date"`
	Status    string `json:"status"`
	Notes     string `json:"notes"`
}

// Create adds a project.
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

// Update replaces the editable fields of a project.
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	id, ok := projectID(w, r)
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

// Delete removes an unused project.
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	id, ok := projectID(w, r)
	if !ok {
		return
	}
	if err := h.svc.Delete(r.Context(), id); err != nil {
		writeError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// Timeline returns the read-only monthly timeline.
func (h *Handler) Timeline(w http.ResponseWriter, r *http.Request) {
	month := r.URL.Query().Get("month")
	if month == "" {
		month = h.svc.clock.Now().Format("2006-01")
	}
	result, err := h.svc.Timeline(r.Context(), month)
	if err != nil {
		writeError(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, result)
}

func (r request) input() Input {
	return Input{
		Name:      r.Name,
		StartDate: r.StartDate,
		EndDate:   r.EndDate,
		Status:    r.Status,
		Notes:     r.Notes,
	}
}

func projectID(w http.ResponseWriter, r *http.Request) (int64, bool) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil || id <= 0 {
		httpx.Error(w, http.StatusNotFound, "Proyek tidak ditemukan.")
		return 0, false
	}
	return id, true
}

func writeError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ErrInvalidName):
		httpx.Error(w, http.StatusBadRequest, "Nama proyek wajib diisi (maksimal 100 karakter).")
	case errors.Is(err, ErrInvalidStatus):
		httpx.Error(w, http.StatusBadRequest, "Status proyek tidak valid.")
	case errors.Is(err, ErrInvalidDate), errors.Is(err, ErrInvalidRange):
		httpx.Error(w, http.StatusBadRequest, "Tanggal proyek tidak valid.")
	case errors.Is(err, ErrNotesTooLong):
		httpx.Error(w, http.StatusBadRequest, "Catatan maksimal 500 karakter.")
	case errors.Is(err, ErrInvalidMonth):
		httpx.Error(w, http.StatusBadRequest, "Bulan tidak valid.")
	case errors.Is(err, ErrNotFound):
		httpx.Error(w, http.StatusNotFound, "Proyek tidak ditemukan.")
	case errors.Is(err, ErrInUse):
		httpx.Error(w, http.StatusConflict,
			"Proyek masih dipakai di absensi. Lepas tanda proyek dari absensi terlebih dahulu.")
	default:
		slog.Error("project request failed", "error", err)
		httpx.Error(w, http.StatusInternalServerError, "Terjadi kesalahan pada server.")
	}
}
