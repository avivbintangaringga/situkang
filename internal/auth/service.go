// Package auth handles admin accounts, sessions, and the session middleware.
package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"

	"situkang/internal/clock"
)

const (
	// SessionCookieName is the cookie that carries the raw session token.
	SessionCookieName = "situkang_session"

	sessionDuration   = 30 * 24 * time.Hour
	minPasswordLength = 8
	minUsernameLength = 3
	maxUsernameLength = 50
)

var (
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrInvalidSession     = errors.New("invalid session")
	ErrUsernameTaken      = errors.New("username already taken")
	ErrInvalidUsername    = errors.New("invalid username")
	ErrWeakPassword       = errors.New("password too short")
)

// Service owns the auth business rules: hashing, tokens, and expiry.
type Service struct {
	db    *sql.DB
	clock clock.Clock
}

// NewService builds a Service. The clock is injectable so tests can expire
// sessions without sleeping.
func NewService(db *sql.DB, clk clock.Clock) *Service {
	return &Service{db: db, clock: clk}
}

// CreateUser hashes the password and stores a new admin account.
func (s *Service) CreateUser(ctx context.Context, username, password string) (User, error) {
	username = strings.TrimSpace(username)
	if len(username) < minUsernameLength || len(username) > maxUsernameLength {
		return User{}, ErrInvalidUsername
	}
	if len(password) < minPasswordLength {
		return User{}, ErrWeakPassword
	}

	if _, err := findUserByUsername(ctx, s.db, username); err == nil {
		return User{}, ErrUsernameTaken
	} else if !errors.Is(err, sql.ErrNoRows) {
		return User{}, err
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return User{}, err
	}

	return insertUser(ctx, s.db, username, string(hash), s.clock.Now())
}

// Authenticate checks a username and password pair.
func (s *Service) Authenticate(ctx context.Context, username, password string) (User, error) {
	rec, err := findUserByUsername(ctx, s.db, strings.TrimSpace(username))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return User{}, ErrInvalidCredentials
		}
		return User{}, err
	}

	if err := bcrypt.CompareHashAndPassword([]byte(rec.PasswordHash), []byte(password)); err != nil {
		return User{}, ErrInvalidCredentials
	}
	return rec.User, nil
}

// CreateSession stores a new session and returns the raw token for the cookie.
// Only the SHA-256 hash of the token is persisted.
func (s *Service) CreateSession(ctx context.Context, userID int64) (string, time.Time, error) {
	token, err := newToken()
	if err != nil {
		return "", time.Time{}, err
	}

	now := s.clock.Now()
	expiresAt := now.Add(sessionDuration)
	if err := insertSession(ctx, s.db, hashToken(token), userID, expiresAt, now); err != nil {
		return "", time.Time{}, err
	}

	return token, expiresAt, nil
}

// UserForToken resolves a raw cookie token to its user, expiring it if needed.
func (s *Service) UserForToken(ctx context.Context, token string) (User, error) {
	if token == "" {
		return User{}, ErrInvalidSession
	}

	user, expiresAt, err := findSession(ctx, s.db, hashToken(token))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return User{}, ErrInvalidSession
		}
		return User{}, err
	}

	if !s.clock.Now().Before(expiresAt) {
		_ = deleteSession(ctx, s.db, hashToken(token))
		return User{}, ErrInvalidSession
	}

	return user, nil
}

// DeleteSession removes the session behind a raw cookie token.
func (s *Service) DeleteSession(ctx context.Context, token string) error {
	if token == "" {
		return nil
	}
	return deleteSession(ctx, s.db, hashToken(token))
}

func newToken() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}

func hashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}
