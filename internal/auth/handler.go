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
