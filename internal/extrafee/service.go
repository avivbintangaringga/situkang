// Package extrafee manages standalone daily bonuses and deductions.
package extrafee

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"

	"situkang/internal/clock"
)

var (
	ErrNotFound            = errors.New("extra fee not found")
	ErrLocked              = errors.New("extra fee locked")
	ErrInvalidAmount       = errors.New("invalid amount")
	ErrInvalidDate         = errors.New("invalid date")
	ErrInvalidWorker       = errors.New("invalid worker")
	ErrDescriptionRequired = errors.New("description required")
)

const maxDescriptionLength = 200

// Input is the payload for creating an extra fee.
type Input struct {
	WorkerID    int64
	Date        string
	Amount      int64
	Description string
}

// UpdateInput changes the fields that may be edited after creation.
type UpdateInput struct {
	Date        string
	Amount      int64
	Description string
}

// Service holds the extra fee business rules.
type Service struct {
	db    *sql.DB
	clock clock.Clock
}

// NewService builds the extra fee service.
func NewService(db *sql.DB, clk clock.Clock) *Service {
	return &Service{db: db, clock: clk}
}

// List returns extra fees, optionally filtered by worker and date range.
func (s *Service) List(ctx context.Context, workerID int64, from, to string) ([]ExtraFee, error) {
	if from != "" {
		if _, err := clock.ParseDate(from); err != nil {
			return nil, ErrInvalidDate
		}
	}
	if to != "" {
		if _, err := clock.ParseDate(to); err != nil {
			return nil, ErrInvalidDate
		}
	}
	return list(ctx, s.db, workerID, from, to)
}

// Get returns one extra fee.
func (s *Service) Get(ctx context.Context, id int64) (ExtraFee, error) {
	item, err := get(ctx, s.db, id)
	if errors.Is(err, sql.ErrNoRows) {
		return ExtraFee{}, ErrNotFound
	}
	return item, err
}

// Create stores a new extra fee for a worker, unsettled until a recap picks it
// up.
func (s *Service) Create(ctx context.Context, in Input) (ExtraFee, error) {
	date, amount, description, err := validate(in.Date, in.Amount, in.Description)
	if err != nil {
		return ExtraFee{}, err
	}

	exists, err := workerExists(ctx, s.db, in.WorkerID)
	if err != nil {
		return ExtraFee{}, err
	}
	if !exists {
		return ExtraFee{}, ErrInvalidWorker
	}

	return insert(ctx, s.db, in.WorkerID, date, amount, description,
		s.clock.Now().UTC().Format(time.RFC3339))
}

// Update edits an extra fee that is not settled in a final recap yet.
func (s *Service) Update(ctx context.Context, id int64, in UpdateInput) (ExtraFee, error) {
	current, err := s.Get(ctx, id)
	if err != nil {
		return ExtraFee{}, err
	}
	if current.Locked {
		return ExtraFee{}, ErrLocked
	}

	date, amount, description, err := validate(in.Date, in.Amount, in.Description)
	if err != nil {
		return ExtraFee{}, err
	}

	return update(ctx, s.db, id, date, amount, description)
}

// Delete removes an extra fee that is not settled yet.
func (s *Service) Delete(ctx context.Context, id int64) error {
	current, err := s.Get(ctx, id)
	if err != nil {
		return err
	}
	if current.Locked {
		return ErrLocked
	}
	return remove(ctx, s.db, id)
}

func validate(date string, amount int64, description string) (string, int64, string, error) {
	if _, err := clock.ParseDate(date); err != nil {
		return "", 0, "", ErrInvalidDate
	}
	if amount == 0 {
		return "", 0, "", ErrInvalidAmount
	}
	description = strings.TrimSpace(description)
	if description == "" || len([]rune(description)) > maxDescriptionLength {
		return "", 0, "", ErrDescriptionRequired
	}
	return date, amount, description, nil
}
