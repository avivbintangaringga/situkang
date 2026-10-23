// Package worker manages tukang records: names, roles, daily fees, soft delete.
package worker

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"

	"situkang/internal/clock"
)

var (
	ErrNotFound     = errors.New("worker not found")
	ErrInvalidName  = errors.New("invalid worker name")
	ErrInvalidRole  = errors.New("invalid worker role")
	ErrInvalidFee   = errors.New("invalid daily fee")
	ErrInvalidInput = errors.New("invalid worker id")
)

const (
	maxNameLength = 100
	maxRoleLength = 50
)

type workerData struct {
	Name      string
	Role      string
	DailyFee  int64
	Active    bool
	CreatedAt string
}

// Input is the validated payload accepted by Create and Update.
type Input struct {
	Name     string
	Role     string
	DailyFee int64
	// Active is optional on create (defaults to true) and on update (keeps the
	// current value when nil).
	Active *bool
}

// Service holds the worker business rules.
type Service struct {
	db    *sql.DB
	clock clock.Clock
}

// NewService builds the worker service.
func NewService(db *sql.DB, clk clock.Clock) *Service {
	return &Service{db: db, clock: clk}
}

// List returns workers matching search. Inactive workers are included only when
// explicitly requested.
func (s *Service) List(ctx context.Context, search string, includeInactive bool) ([]Worker, error) {
	return list(ctx, s.db, strings.TrimSpace(search), includeInactive)
}

// Get returns one worker or ErrNotFound.
func (s *Service) Get(ctx context.Context, id int64) (Worker, error) {
	if id <= 0 {
		return Worker{}, ErrInvalidInput
	}
	w, err := get(ctx, s.db, id)
	if errors.Is(err, sql.ErrNoRows) {
		return Worker{}, ErrNotFound
	}
	return w, err
}

// Create stores a new active worker.
func (s *Service) Create(ctx context.Context, in Input) (Worker, error) {
	name, role, err := validate(in)
	if err != nil {
		return Worker{}, err
	}

	active := true
	if in.Active != nil {
		active = *in.Active
	}

	return insert(ctx, s.db, workerData{
		Name:      name,
		Role:      role,
		DailyFee:  in.DailyFee,
		Active:    active,
		CreatedAt: s.clock.Now().UTC().Format(time.RFC3339),
	})
}

// Update replaces a worker's editable fields. A nil Active keeps the current
// value so partial clients cannot deactivate a worker by accident.
func (s *Service) Update(ctx context.Context, id int64, in Input) (Worker, error) {
	if id <= 0 {
		return Worker{}, ErrInvalidInput
	}
	name, role, err := validate(in)
	if err != nil {
		return Worker{}, err
	}

	current, err := s.Get(ctx, id)
	if err != nil {
		return Worker{}, err
	}

	active := current.Active
	if in.Active != nil {
		active = *in.Active
	}

	updated, err := update(ctx, s.db, id, workerData{
		Name:     name,
		Role:     role,
		DailyFee: in.DailyFee,
		Active:   active,
	})
	if errors.Is(err, sql.ErrNoRows) {
		return Worker{}, ErrNotFound
	}
	return updated, err
}

// Delete soft-deletes a worker: active = 0, the row and all history stay.
func (s *Service) Delete(ctx context.Context, id int64) error {
	if id <= 0 {
		return ErrInvalidInput
	}
	if _, err := s.Get(ctx, id); err != nil {
		return err
	}
	if err := setActive(ctx, s.db, id, false); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrNotFound
		}
		return err
	}
	return nil
}

func validate(in Input) (string, string, error) {
	name := strings.TrimSpace(in.Name)
	if name == "" || len([]rune(name)) > maxNameLength {
		return "", "", ErrInvalidName
	}

	role := strings.TrimSpace(in.Role)
	if role == "" || len([]rune(role)) > maxRoleLength {
		return "", "", ErrInvalidRole
	}

	if in.DailyFee < 0 {
		return "", "", ErrInvalidFee
	}

	return name, role, nil
}
