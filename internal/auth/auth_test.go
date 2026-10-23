package auth

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"situkang/internal/clock"
	"situkang/internal/db"
)

const testSchema = `
CREATE TABLE users (
    id            INTEGER PRIMARY KEY,
    username      TEXT NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    created_at    TEXT NOT NULL
) STRICT;

CREATE TABLE sessions (
    id         TEXT PRIMARY KEY,
    user_id    INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    expires_at TEXT NOT NULL,
    created_at TEXT NOT NULL
) STRICT;
`

func testService(t *testing.T, clk clock.Clock) (*Service, *sql.DB) {
	t.Helper()

	sqlDB, err := db.Open(filepath.Join(t.TempDir(), "auth.db"))
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}
	t.Cleanup(func() { sqlDB.Close() })

	if _, err := sqlDB.ExecContext(context.Background(), testSchema); err != nil {
		t.Fatalf("create test schema: %v", err)
	}

	return NewService(sqlDB, clk), sqlDB
}

func TestCreateUserAndAuthenticate(t *testing.T) {
	ctx := context.Background()
	svc, _ := testService(t, clock.New(time.UTC))

	user, err := svc.CreateUser(ctx, "  admin  ", "rahasia123")
	if err != nil {
		t.Fatalf("CreateUser: %v", err)
	}
	if user.Username != "admin" {
		t.Fatalf("username = %q, want %q", user.Username, "admin")
	}

	got, err := svc.Authenticate(ctx, "admin", "rahasia123")
	if err != nil {
		t.Fatalf("Authenticate: %v", err)
	}
	if got.ID != user.ID {
		t.Fatalf("id = %d, want %d", got.ID, user.ID)
	}
}

func TestCreateUserValidation(t *testing.T) {
	ctx := context.Background()
	svc, _ := testService(t, clock.New(time.UTC))

	if _, err := svc.CreateUser(ctx, "ab", "rahasia123"); !errors.Is(err, ErrInvalidUsername) {
		t.Fatalf("short username error = %v, want ErrInvalidUsername", err)
	}
	if _, err := svc.CreateUser(ctx, "admin", "pendek"); !errors.Is(err, ErrWeakPassword) {
		t.Fatalf("short password error = %v, want ErrWeakPassword", err)
	}
}

func TestCreateUserRejectsDuplicate(t *testing.T) {
	ctx := context.Background()
	svc, _ := testService(t, clock.New(time.UTC))

	if _, err := svc.CreateUser(ctx, "admin", "rahasia123"); err != nil {
		t.Fatalf("first CreateUser: %v", err)
	}
	if _, err := svc.CreateUser(ctx, "admin", "rahasia456"); !errors.Is(err, ErrUsernameTaken) {
		t.Fatalf("duplicate error = %v, want ErrUsernameTaken", err)
	}
}

func TestAuthenticateWrongPassword(t *testing.T) {
	ctx := context.Background()
	svc, _ := testService(t, clock.New(time.UTC))

	if _, err := svc.CreateUser(ctx, "admin", "rahasia123"); err != nil {
		t.Fatalf("CreateUser: %v", err)
	}

	if _, err := svc.Authenticate(ctx, "admin", "salah"); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("error = %v, want ErrInvalidCredentials", err)
	}
	if _, err := svc.Authenticate(ctx, "tidak-ada", "rahasia123"); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("unknown user error = %v, want ErrInvalidCredentials", err)
	}
}

func TestSessionStoresOnlyTokenHash(t *testing.T) {
	ctx := context.Background()
	svc, sqlDB := testService(t, clock.New(time.UTC))

	user, err := svc.CreateUser(ctx, "admin", "rahasia123")
	if err != nil {
		t.Fatalf("CreateUser: %v", err)
	}

	token, expiresAt, err := svc.CreateSession(ctx, user.ID)
	if err != nil {
		t.Fatalf("CreateSession: %v", err)
	}
	if token == "" {
		t.Fatal("token is empty")
	}
	if !expiresAt.After(time.Now()) {
		t.Fatalf("expiresAt = %v, want a future time", expiresAt)
	}

	var stored string
	if err := sqlDB.QueryRowContext(ctx, `SELECT id FROM sessions`).Scan(&stored); err != nil {
		t.Fatalf("read session: %v", err)
	}
	if stored == token {
		t.Fatal("raw token must not be stored")
	}
	if stored != hashToken(token) {
		t.Fatalf("stored id = %q, want the token hash", stored)
	}

	got, err := svc.UserForToken(ctx, token)
	if err != nil {
		t.Fatalf("UserForToken: %v", err)
	}
	if got.ID != user.ID {
		t.Fatalf("user id = %d, want %d", got.ID, user.ID)
	}
}

func TestUserForTokenRejectsExpiredSession(t *testing.T) {
	ctx := context.Background()
	start := time.Date(2026, 1, 5, 8, 0, 0, 0, time.UTC)
	svc, _ := testService(t, clock.Fixed(start))

	user, err := svc.CreateUser(ctx, "admin", "rahasia123")
	if err != nil {
		t.Fatalf("CreateUser: %v", err)
	}
	token, _, err := svc.CreateSession(ctx, user.ID)
	if err != nil {
		t.Fatalf("CreateSession: %v", err)
	}

	// Same service, later clock: the session must be rejected.
	expired := NewService(svc.db, clock.Fixed(start.Add(sessionDuration+time.Minute)))
	if _, err := expired.UserForToken(ctx, token); !errors.Is(err, ErrInvalidSession) {
		t.Fatalf("error = %v, want ErrInvalidSession", err)
	}
}

func TestDeleteSessionInvalidatesToken(t *testing.T) {
	ctx := context.Background()
	svc, _ := testService(t, clock.New(time.UTC))

	user, err := svc.CreateUser(ctx, "admin", "rahasia123")
	if err != nil {
		t.Fatalf("CreateUser: %v", err)
	}
	token, _, err := svc.CreateSession(ctx, user.ID)
	if err != nil {
		t.Fatalf("CreateSession: %v", err)
	}

	if err := svc.DeleteSession(ctx, token); err != nil {
		t.Fatalf("DeleteSession: %v", err)
	}
	if _, err := svc.UserForToken(ctx, token); !errors.Is(err, ErrInvalidSession) {
		t.Fatalf("error = %v, want ErrInvalidSession", err)
	}
}

func TestRequireSessionRejectsMissingCookie(t *testing.T) {
	svc, _ := testService(t, clock.New(time.UTC))

	called := false
	next := http.HandlerFunc(func(http.ResponseWriter, *http.Request) { called = true })
	handler := svc.RequireSession(next)

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/auth/me", nil))

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
	if called {
		t.Fatal("next handler must not run without a session")
	}
	if !strings.Contains(rec.Body.String(), "Sesi Anda telah berakhir") {
		t.Fatalf("body = %q, want the Indonesian session message", rec.Body.String())
	}
}

func TestLoginHandlerSetsCookie(t *testing.T) {
	ctx := context.Background()
	svc, _ := testService(t, clock.New(time.UTC))
	if _, err := svc.CreateUser(ctx, "admin", "rahasia123"); err != nil {
		t.Fatalf("CreateUser: %v", err)
	}

	h := NewHandler(svc)
	body := strings.NewReader(`{"username":"admin","password":"rahasia123"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/auth/login", body)
	req.Header.Set("Content-Type", "application/json")
	req.Host = "localhost:8080"

	rec := httptest.NewRecorder()
	h.Login(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body %s)", rec.Code, rec.Body.String())
	}

	cookies := rec.Result().Cookies()
	if len(cookies) != 1 {
		t.Fatalf("cookies = %d, want 1", len(cookies))
	}
	cookie := cookies[0]
	if cookie.Name != SessionCookieName {
		t.Fatalf("cookie name = %q, want %q", cookie.Name, SessionCookieName)
	}
	if !cookie.HttpOnly {
		t.Fatal("cookie must be HttpOnly")
	}
	if cookie.SameSite != http.SameSiteLaxMode {
		t.Fatalf("SameSite = %v, want Lax", cookie.SameSite)
	}
	if cookie.Secure {
		t.Fatal("cookie must not be Secure on localhost")
	}
	if cookie.MaxAge < int((29 * 24 * time.Hour).Seconds()) {
		t.Fatalf("MaxAge = %d, want about 30 days", cookie.MaxAge)
	}

	var user User
	if err := json.Unmarshal(rec.Body.Bytes(), &user); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if user.Username != "admin" {
		t.Fatalf("username = %q, want admin", user.Username)
	}
}

func TestLoginHandlerRejectsBadInput(t *testing.T) {
	ctx := context.Background()
	svc, _ := testService(t, clock.New(time.UTC))
	if _, err := svc.CreateUser(ctx, "admin", "rahasia123"); err != nil {
		t.Fatalf("CreateUser: %v", err)
	}
	h := NewHandler(svc)

	t.Run("wrong password", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/auth/login",
			strings.NewReader(`{"username":"admin","password":"salah"}`))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		h.Login(rec, req)

		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("status = %d, want 401", rec.Code)
		}
		if len(rec.Result().Cookies()) != 0 {
			t.Fatal("failed login must not set a cookie")
		}
	})

	t.Run("not json", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/auth/login",
			strings.NewReader(`username=admin`))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		rec := httptest.NewRecorder()
		h.Login(rec, req)

		if rec.Code != http.StatusUnsupportedMediaType {
			t.Fatalf("status = %d, want 415", rec.Code)
		}
	})

	t.Run("malformed json", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/auth/login",
			strings.NewReader(`{"username":`))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		h.Login(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want 400", rec.Code)
		}
	})
}

func TestCookieSecureOutsideLocalhost(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/api/auth/login", nil)
	req.Host = "situkang.example.com"
	if !cookieSecure(req) {
		t.Fatal("cookie must be Secure on a non-localhost host")
	}

	req.Host = "127.0.0.1:8080"
	if cookieSecure(req) {
		t.Fatal("cookie must not be Secure on 127.0.0.1")
	}
}
