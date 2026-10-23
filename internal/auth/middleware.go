package auth

import (
	"context"
	"errors"
	"net/http"

	"situkang/internal/httpx"
)

type userContextKey struct{}

// RequireSession rejects requests without a valid session cookie and stores the
// signed-in user on the request context.
func (s *Service) RequireSession(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie(SessionCookieName)
		if err != nil {
			httpx.Error(w, http.StatusUnauthorized, "Sesi Anda telah berakhir. Silakan masuk kembali.")
			return
		}

		user, err := s.UserForToken(r.Context(), cookie.Value)
		switch {
		case errors.Is(err, ErrInvalidSession):
			httpx.Error(w, http.StatusUnauthorized, "Sesi Anda telah berakhir. Silakan masuk kembali.")
			return
		case err != nil:
			httpx.Error(w, http.StatusInternalServerError, "Terjadi kesalahan pada server.")
			return
		}

		ctx := context.WithValue(r.Context(), userContextKey{}, user)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// UserFrom returns the user attached by RequireSession.
func UserFrom(ctx context.Context) (User, bool) {
	user, ok := ctx.Value(userContextKey{}).(User)
	return user, ok
}
