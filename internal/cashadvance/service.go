// Package cashadvance manages kas bon: advances taken by workers.
package cashadvance

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"situkang/internal/clock"
)

var (
	ErrNotFound      = errors.New("cash advance not found")
	ErrLocked        = errors.New("cash advance locked")
	ErrInvalidAmount = errors.New("invalid amount")
	ErrInvalidDate   = errors.New("invalid date")
	ErrInvalidWorker = errors.New("invalid worker")
	ErrNoteTooLong   = errors.New("note too long")
)

const maxNoteLength = 200

// Input is the payload for creating a cash advance.
type Input struct {
	WorkerID int64
	Date     string
	Amount   int64
	Note     string
}

// UpdateInput changes the fields that may be edited after creation.
type UpdateInput struct {
	Date   string
	Amount int64
	Note   string
}

// Service holds the kas bon business rules.
type Service struct {
	db    *sql.DB
	clock clock.Clock
}

// NewService builds the cash advance service.
func NewService(db *sql.DB, clk clock.Clock) *Service {
	return &Service{db: db, clock: clk}
}

// List returns cash advances, optionally filtered by worker and date range.
func (s *Service) List(ctx context.Context, workerID int64, from, to string) ([]CashAdvance, error) {
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

// Get returns one cash advance.
func (s *Service) Get(ctx context.Context, id int64) (CashAdvance, error) {
	item, err := get(ctx, s.db, id)
	if errors.Is(err, sql.ErrNoRows) {
		return CashAdvance{}, ErrNotFound
	}
	return item, err
}

// Create stores a new kas bon for a worker.
func (s *Service) Create(ctx context.Context, in Input) (CashAdvance, error) {
	if _, err := clock.ParseDate(in.Date); err != nil {
		return CashAdvance{}, ErrInvalidDate
	}
	if in.Amount <= 0 {
		return CashAdvance{}, ErrInvalidAmount
	}
	if len([]rune(in.Note)) > maxNoteLength {
		return CashAdvance{}, ErrNoteTooLong
	}

	exists, err := workerExists(ctx, s.db, in.WorkerID)
	if err != nil {
		return CashAdvance{}, err
	}
	if !exists {
		return CashAdvance{}, ErrInvalidWorker
	}

	return insert(ctx, s.db, in.WorkerID, in.Date, in.Amount, in.Note, s.clock.Now().UTC().Format(time.RFC3339))
}

// Update edits a kas bon that is not settled in a final recap yet.
func (s *Service) Update(ctx context.Context, id int64, in UpdateInput) (CashAdvance, error) {
	current, err := s.Get(ctx, id)
	if err != nil {
		return CashAdvance{}, err
	}
	if current.Locked {
		return CashAdvance{}, ErrLocked
	}

	if _, err := clock.ParseDate(in.Date); err != nil {
		return CashAdvance{}, ErrInvalidDate
	}
	if in.Amount <= 0 {
		return CashAdvance{}, ErrInvalidAmount
	}
	if len([]rune(in.Note)) > maxNoteLength {
		return CashAdvance{}, ErrNoteTooLong
	}

	return update(ctx, s.db, id, in.Date, in.Amount, in.Note)
}

// Delete removes a kas bon that is not settled yet.
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
