package auth

import (
	"errors"
	"log/slog"
	"net"
	"net/http"
	"strings"
	"time"

	"situkang/internal/httpx"
)

// Handler exposes the auth endpoints.
type Handler struct {
	svc *Service
}

// NewHandler builds the auth HTTP layer.
func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

type loginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// Login verifies credentials and starts a session.
func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := httpx.DecodeJSON(w, r, &req); err != nil {
		httpx.DecodeError(w, err)
		return
	}

	user, err := h.svc.Authenticate(r.Context(), req.Username, req.Password)
	if err != nil {
		if errors.Is(err, ErrInvalidCredentials) {
			httpx.Error(w, http.StatusUnauthorized, "Nama pengguna atau kata sandi salah.")
			return
		}
		slog.Error("login failed", "error", err)
		httpx.Error(w, http.StatusInternalServerError, "Terjadi kesalahan pada server.")
		return
	}

	token, expiresAt, err := h.svc.CreateSession(r.Context(), user.ID)
	if err != nil {
		slog.Error("create session failed", "error", err)
		httpx.Error(w, http.StatusInternalServerError, "Terjadi kesalahan pada server.")
		return
	}

	setSessionCookie(w, r, token, expiresAt)
	httpx.JSON(w, http.StatusOK, user)
}

// Logout ends the current session.
func (h *Handler) Logout(w http.ResponseWriter, r *http.Request) {
	if cookie, err := r.Cookie(SessionCookieName); err == nil {
		if err := h.svc.DeleteSession(r.Context(), cookie.Value); err != nil {
			slog.Error("delete session failed", "error", err)
			httpx.Error(w, http.StatusInternalServerError, "Terjadi kesalahan pada server.")
			return
		}
	}

	clearSessionCookie(w, r)
	w.WriteHeader(http.StatusNoContent)
}

// Me returns the signed-in user.
func (h *Handler) Me(w http.ResponseWriter, r *http.Request) {
	user, ok := UserFrom(r.Context())
	if !ok {
		httpx.Error(w, http.StatusUnauthorized, "Sesi Anda telah berakhir. Silakan masuk kembali.")
		return
	}
	httpx.JSON(w, http.StatusOK, user)
}

type createUserRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// ListUsers returns every admin account.
func (h *Handler) ListUsers(w http.ResponseWriter, r *http.Request) {
	accounts, err := h.svc.ListUsers(r.Context())
	if err != nil {
		slog.Error("list users failed", "error", err)
		httpx.Error(w, http.StatusInternalServerError, "Terjadi kesalahan pada server.")
		return
	}
	httpx.JSON(w, http.StatusOK, accounts)
}

// CreateUser adds another admin account.
func (h *Handler) CreateUser(w http.ResponseWriter, r *http.Request) {
	var req createUserRequest
	if err := httpx.DecodeJSON(w, r, &req); err != nil {
		httpx.DecodeError(w, err)
		return
	}

	user, err := h.svc.CreateUser(r.Context(), req.Username, req.Password)
	if err != nil {
		writeCredentialsError(w, err)
		return
	}
	httpx.JSON(w, http.StatusCreated, user)
}

type updateUsernameRequest struct {
	Username string `json:"username"`
}

// UpdateUsername renames the signed-in account.
func (h *Handler) UpdateUsername(w http.ResponseWriter, r *http.Request) {
	current, ok := UserFrom(r.Context())
	if !ok {
		httpx.Error(w, http.StatusUnauthorized, "Sesi Anda telah berakhir. Silakan masuk kembali.")
		return
	}

	var req updateUsernameRequest
	if err := httpx.DecodeJSON(w, r, &req); err != nil {
		httpx.DecodeError(w, err)
		return
	}

	user, err := h.svc.RenameUser(r.Context(), current.ID, req.Username)
	if err != nil {
		writeCredentialsError(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, user)
}

type changePasswordRequest struct {
	CurrentPassword string `json:"current_password"`
	NewPassword     string `json:"new_password"`
}

// ChangePassword updates the signed-in account's password.
func (h *Handler) ChangePassword(w http.ResponseWriter, r *http.Request) {
	current, ok := UserFrom(r.Context())
	if !ok {
		httpx.Error(w, http.StatusUnauthorized, "Sesi Anda telah berakhir. Silakan masuk kembali.")
		return
	}

	var req changePasswordRequest
	if err := httpx.DecodeJSON(w, r, &req); err != nil {
		httpx.DecodeError(w, err)
		return
	}

	var token string
	if cookie, err := r.Cookie(SessionCookieName); err == nil {
		token = cookie.Value
	}

	err := h.svc.ChangePassword(r.Context(), current.ID, token, req.CurrentPassword, req.NewPassword)
	switch {
	case errors.Is(err, ErrWrongPassword):
		httpx.Error(w, http.StatusBadRequest, "Kata sandi saat ini salah.")
		return
	case errors.Is(err, ErrWeakPassword):
		httpx.Error(w, http.StatusBadRequest, "Kata sandi baru minimal 8 karakter.")
		return
	case errors.Is(err, ErrInvalidSession):
		httpx.Error(w, http.StatusUnauthorized, "Sesi Anda telah berakhir. Silakan masuk kembali.")
		return
	case err != nil:
		slog.Error("change password failed", "error", err)
		httpx.Error(w, http.StatusInternalServerError, "Terjadi kesalahan pada server.")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func writeCredentialsError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ErrInvalidUsername):
		httpx.Error(w, http.StatusBadRequest, "Nama pengguna harus 3-50 karakter.")
	case errors.Is(err, ErrWeakPassword):
		httpx.Error(w, http.StatusBadRequest, "Kata sandi minimal 8 karakter.")
	case errors.Is(err, ErrUsernameTaken):
		httpx.Error(w, http.StatusConflict, "Nama pengguna sudah dipakai.")
	default:
		slog.Error("account request failed", "error", err)
		httpx.Error(w, http.StatusInternalServerError, "Terjadi kesalahan pada server.")
	}
}

func setSessionCookie(w http.ResponseWriter, r *http.Request, token string, expiresAt time.Time) {
	http.SetCookie(w, &http.Cookie{
		Name:     SessionCookieName,
		Value:    token,
		Path:     "/",
		Expires:  expiresAt,
		MaxAge:   int(time.Until(expiresAt).Seconds()),
		HttpOnly: true,
		Secure:   cookieSecure(r),
		SameSite: http.SameSiteLaxMode,
	})
}

func clearSessionCookie(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name:     SessionCookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   cookieSecure(r),
		SameSite: http.SameSiteLaxMode,
	})
}

// cookieSecure keeps the cookie usable over plain HTTP on localhost while
// requiring HTTPS everywhere else.
func cookieSecure(r *http.Request) bool {
	if r.TLS != nil {
		return true
	}

	host := r.Host
	if h, _, err := net.SplitHostPort(host); err == nil {
		host = h
	}
	host = strings.Trim(host, "[]")

	switch host {
	case "localhost", "127.0.0.1", "::1":
		return false
	default:
		return true
	}
}
