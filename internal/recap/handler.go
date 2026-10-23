package recap

import (
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"situkang/internal/httpx"
)

// Handler exposes the recap endpoints.
type Handler struct {
	svc *Service
}

// NewHandler builds the recap HTTP layer.
func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

// List returns every recap.
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	recaps, err := h.svc.List(r.Context())
	if err != nil {
		writeError(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, recaps)
}

type createRequest struct {
	Mode      string  `json:"mode"`
	StartDate *string `json:"start_date"`
	EndDate   *string `json:"end_date"`
}

// Create starts the single allowed draft.
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var req createRequest
	if err := httpx.DecodeJSON(w, r, &req); err != nil {
		httpx.DecodeError(w, err)
		return
	}

	created, err := h.svc.Create(r.Context(), CreateInput{
		Mode:      req.Mode,
		StartDate: req.StartDate,
		EndDate:   req.EndDate,
	})
	if err != nil {
		writeError(w, err)
		return
	}
	httpx.JSON(w, http.StatusCreated, created)
}

// Detail returns the live draft or the frozen final recap.
func (h *Handler) Detail(w http.ResponseWriter, r *http.Request) {
	id, ok := recapID(w, r)
	if !ok {
		return
	}

	detail, err := h.svc.Detail(r.Context(), id)
	if err != nil {
		writeError(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, detail)
}

// Delete removes a draft.
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	id, ok := recapID(w, r)
	if !ok {
		return
	}

	if err := h.svc.DeleteDraft(r.Context(), id); err != nil {
		writeError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

type shareResponse struct {
	Token string `json:"token"`
	URL   string `json:"url"`
}

func newShareResponse(token string) shareResponse {
	return shareResponse{Token: token, URL: "/bagikan/" + token}
}

// GetShare returns the existing public link for a final recap.
func (h *Handler) GetShare(w http.ResponseWriter, r *http.Request) {
	id, ok := recapID(w, r)
	if !ok {
		return
	}

	token, err := h.svc.ShareToken(r.Context(), id)
	if err != nil {
		writeError(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, newShareResponse(token))
}

// CreateShare creates or returns the public link for a final recap.
func (h *Handler) CreateShare(w http.ResponseWriter, r *http.Request) {
	id, ok := recapID(w, r)
	if !ok {
		return
	}

	token, err := h.svc.CreateShare(r.Context(), id)
	if err != nil {
		writeError(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, newShareResponse(token))
}

// RevokeShare removes the public link of a recap.
func (h *Handler) RevokeShare(w http.ResponseWriter, r *http.Request) {
	id, ok := recapID(w, r)
	if !ok {
		return
	}

	if err := h.svc.RevokeShare(r.Context(), id); err != nil {
		writeError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// PublicDetail returns a shared recap without requiring a session.
func (h *Handler) PublicDetail(w http.ResponseWriter, r *http.Request) {
	detail, err := h.svc.PublicDetail(r.Context(), chi.URLParam(r, "token"))
	if err != nil {
		writeError(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, detail)
}

type paidRequest struct {
	WorkerID int64 `json:"worker_id"`
	Paid     bool  `json:"paid"`
}

// SetPaid toggles a worker's paid flag in the draft.
func (h *Handler) SetPaid(w http.ResponseWriter, r *http.Request) {
	id, ok := recapID(w, r)
	if !ok {
		return
	}

	var req paidRequest
	if err := httpx.DecodeJSON(w, r, &req); err != nil {
		httpx.DecodeError(w, err)
		return
	}

	if err := h.svc.SetPaid(r.Context(), id, req.WorkerID, req.Paid); err != nil {
		writeError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// Finalize settles the recap.
func (h *Handler) Finalize(w http.ResponseWriter, r *http.Request) {
	id, ok := recapID(w, r)
	if !ok {
		return
	}

	if err := h.svc.Finalize(r.Context(), id); err != nil {
		writeError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// Reopen turns the latest final recap back into a draft.
func (h *Handler) Reopen(w http.ResponseWriter, r *http.Request) {
	id, ok := recapID(w, r)
	if !ok {
		return
	}

	if err := h.svc.Reopen(r.Context(), id); err != nil {
		writeError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

type feeRequest struct {
	WorkerID    int64  `json:"worker_id"`
	Amount      int64  `json:"amount"`
	Description string `json:"description"`
}

// AddFee attaches an extra fee to the draft.
func (h *Handler) AddFee(w http.ResponseWriter, r *http.Request) {
	id, ok := recapID(w, r)
	if !ok {
		return
	}

	var req feeRequest
	if err := httpx.DecodeJSON(w, r, &req); err != nil {
		httpx.DecodeError(w, err)
		return
	}

	fee, err := h.svc.AddFee(r.Context(), id, req.WorkerID, req.Amount, req.Description)
	if err != nil {
		writeError(w, err)
		return
	}
	httpx.JSON(w, http.StatusCreated, fee)
}

type updateFeeRequest struct {
	Amount      int64  `json:"amount"`
	Description string `json:"description"`
}

// UpdateFee edits an extra fee.
func (h *Handler) UpdateFee(w http.ResponseWriter, r *http.Request) {
	id, ok := recapID(w, r)
	if !ok {
		return
	}
	feeID, ok := parsedID(w, r, "feeId", "Tambahan tidak ditemukan.")
	if !ok {
		return
	}

	var req updateFeeRequest
	if err := httpx.DecodeJSON(w, r, &req); err != nil {
		httpx.DecodeError(w, err)
		return
	}

	if err := h.svc.UpdateFee(r.Context(), id, feeID, req.Amount, req.Description); err != nil {
		writeError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// DeleteFee removes an extra fee.
func (h *Handler) DeleteFee(w http.ResponseWriter, r *http.Request) {
	id, ok := recapID(w, r)
	if !ok {
		return
	}
	feeID, ok := parsedID(w, r, "feeId", "Tambahan tidak ditemukan.")
	if !ok {
		return
	}

	if err := h.svc.DeleteFee(r.Context(), id, feeID); err != nil {
		writeError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ExportExcel streams the recap workbook.
func (h *Handler) ExportExcel(w http.ResponseWriter, r *http.Request) {
	id, ok := recapID(w, r)
	if !ok {
		return
	}

	detail, err := h.svc.Detail(r.Context(), id)
	if err != nil {
		writeError(w, err)
		return
	}

	buffer, err := Excel(detail)
	if err != nil {
		slog.Error("build excel failed", "error", err)
		httpx.Error(w, http.StatusInternalServerError, "Terjadi kesalahan pada server.")
		return
	}

	w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"rekap-%d.xlsx\"", id))
	_, _ = w.Write(buffer.Bytes())
}

// WhatsApp returns the plain text recap message.
func (h *Handler) WhatsApp(w http.ResponseWriter, r *http.Request) {
	id, ok := recapID(w, r)
	if !ok {
		return
	}

	detail, err := h.svc.Detail(r.Context(), id)
	if err != nil {
		writeError(w, err)
		return
	}

	text := AllWhatsAppText(detail)
	if raw := r.URL.Query().Get("worker_id"); raw != "" {
		workerID, err := strconv.ParseInt(raw, 10, 64)
		if err != nil || workerID <= 0 {
			httpx.Error(w, http.StatusBadRequest, "Pekerja tidak ditemukan.")
			return
		}
		text = ""
		for _, line := range detail.Workers {
			if line.WorkerID == workerID {
				text = WhatsAppText(line)
				break
			}
		}
		if text == "" {
			httpx.Error(w, http.StatusNotFound, "Pekerja tidak ada pada rekap ini.")
			return
		}
	}

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	_, _ = w.Write([]byte(text))
}

func recapID(w http.ResponseWriter, r *http.Request) (int64, bool) {
	return parsedID(w, r, "id", "Rekap tidak ditemukan.")
}

func parsedID(w http.ResponseWriter, r *http.Request, param, message string) (int64, bool) {
	id, err := strconv.ParseInt(chi.URLParam(r, param), 10, 64)
	if err != nil || id <= 0 {
		httpx.Error(w, http.StatusNotFound, message)
		return 0, false
	}
	return id, true
}

func writeError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ErrNotFound):
		httpx.Error(w, http.StatusNotFound, "Rekap tidak ditemukan.")
	case errors.Is(err, ErrDraftExists):
		httpx.Error(w, http.StatusConflict,
			"Masih ada rekap draft. Selesaikan atau hapus rekap tersebut dulu.")
	case errors.Is(err, ErrNotDraft):
		httpx.Error(w, http.StatusConflict,
			"Rekap ini sudah final. Buka kembali rekap untuk mengubahnya.")
	case errors.Is(err, ErrNotFinal):
		httpx.Error(w, http.StatusConflict, "Rekap ini belum final.")
	case errors.Is(err, ErrShareNotFound):
		httpx.Error(w, http.StatusNotFound,
			"Tautan tidak ditemukan atau rekap sudah tidak berlaku.")
	case errors.Is(err, ErrNotLatest):
		httpx.Error(w, http.StatusConflict,
			"Hanya rekap final terakhir yang dapat dibuka kembali.")
	case errors.Is(err, ErrLocked):
		httpx.Error(w, http.StatusConflict,
			"Data sudah masuk rekap dan terkunci. Buka kembali rekap untuk mengubahnya.")
	case errors.Is(err, ErrInvalidMode):
		httpx.Error(w, http.StatusBadRequest, "Mode rekap tidak valid.")
	case errors.Is(err, ErrInvalidRange):
		httpx.Error(w, http.StatusBadRequest, "Rentang tanggal tidak valid.")
	case errors.Is(err, ErrInvalidWorker):
		httpx.Error(w, http.StatusBadRequest, "Pekerja tidak ditemukan.")
	case errors.Is(err, ErrInvalidAmount):
		httpx.Error(w, http.StatusBadRequest, "Nominal tambahan tidak boleh nol.")
	case errors.Is(err, ErrDescriptionRequired):
		httpx.Error(w, http.StatusBadRequest,
			"Keterangan tambahan wajib diisi (maksimal 200 karakter).")
	default:
		slog.Error("recap request failed", "error", err)
		httpx.Error(w, http.StatusInternalServerError, "Terjadi kesalahan pada server.")
	}
}
