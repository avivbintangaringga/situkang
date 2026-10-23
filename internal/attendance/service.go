// Package attendance records daily worker attendance with a fee snapshot.
package attendance

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"situkang/internal/clock"
)

var (
	ErrLocked          = errors.New("attendance locked")
	ErrInvalidDate     = errors.New("invalid date")
	ErrInvalidQuarters = errors.New("invalid quarters")
	ErrInvalidWorker   = errors.New("invalid worker")
	ErrDuplicateWorker = errors.New("duplicate worker in day")
	ErrInvalidRange    = errors.New("invalid date range")
)

// Item is one worker entry in a day save.
type Item struct {
	WorkerID int64 `json:"worker_id"`
	Quarters int   `json:"quarters"`
}

// Day is the attendance screen payload for one date.
type Day struct {
	Date        string   `json:"date"`
	Workers     []DayRow `json:"workers"`
	TotalEarned int64    `json:"total_earned"`
	Present     int      `json:"present"`
}

// Service holds the attendance business rules.
type Service struct {
	db    *sql.DB
	clock clock.Clock
}

// NewService builds the attendance service.
func NewService(db *sql.DB, clk clock.Clock) *Service {
	return &Service{db: db, clock: clk}
}

// Day returns every active worker with their record for date (if any).
func (s *Service) Day(ctx context.Context, date string) (Day, error) {
	if _, err := clock.ParseDate(date); err != nil {
		return Day{}, ErrInvalidDate
	}

	rows, err := day(ctx, s.db, date)
	if err != nil {
		return Day{}, err
	}

	result := Day{Date: date, Workers: rows}
	for _, row := range rows {
		result.TotalEarned += row.Earned
		if row.Quarters > 0 {
			result.Present++
		}
	}
	return result, nil
}

// SaveDay replaces the whole day: listed workers get the given quarters and
// omitted workers lose their unlocked record. Locked records are never touched.
func (s *Service) SaveDay(ctx context.Context, date string, items []Item) (Day, error) {
	if _, err := clock.ParseDate(date); err != nil {
		return Day{}, ErrInvalidDate
	}

	seen := make(map[int64]bool, len(items))
	for _, item := range items {
		if item.Quarters < 1 || item.Quarters > 4 {
			return Day{}, ErrInvalidQuarters
		}
		if seen[item.WorkerID] {
			return Day{}, ErrDuplicateWorker
		}
		seen[item.WorkerID] = true
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return Day{}, fmt.Errorf("begin attendance day: %w", err)
	}
	defer tx.Rollback()

	existing, err := recordsForDate(ctx, tx, date)
	if err != nil {
		return Day{}, err
	}

	now := s.clock.Now().UTC().Format(time.RFC3339)
	for _, item := range items {
		fee, active, err := workerFee(ctx, tx, item.WorkerID)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return Day{}, ErrInvalidWorker
			}
			return Day{}, err
		}
		if !active {
			return Day{}, ErrInvalidWorker
		}

		current, ok := existing[item.WorkerID]
		if !ok {
			if _, err := insert(ctx, tx, item.WorkerID, date, item.Quarters, fee, now); err != nil {
				return Day{}, err
			}
			continue
		}

		if current.RecapID != nil {
			if current.Quarters != item.Quarters {
				return Day{}, ErrLocked
			}
			continue
		}
		if current.Quarters != item.Quarters {
			if err := updateQuarters(ctx, tx, current.ID, item.Quarters); err != nil {
				return Day{}, err
			}
		}
	}

	for workerID, current := range existing {
		if seen[workerID] || current.RecapID != nil {
			continue
		}
		if err := remove(ctx, tx, current.ID); err != nil {
			return Day{}, err
		}
	}

	if err := tx.Commit(); err != nil {
		return Day{}, fmt.Errorf("commit attendance day: %w", err)
	}

	return s.Day(ctx, date)
}

// Week returns a worker's attendance between from and to, inclusive.
func (s *Service) Week(ctx context.Context, workerID int64, from, to string) ([]WeekEntry, error) {
	if workerID <= 0 {
		return nil, ErrInvalidWorker
	}

	start, err := clock.ParseDate(from)
	if err != nil {
		return nil, ErrInvalidDate
	}
	end, err := clock.ParseDate(to)
	if err != nil {
		return nil, ErrInvalidDate
	}
	if end.Before(start) {
		return nil, ErrInvalidRange
	}
	if end.Sub(start) > 62*24*time.Hour {
		return nil, ErrInvalidRange
	}

	return week(ctx, s.db, workerID, from, to)
}

// Today returns the current Jakarta date as YYYY-MM-DD.
func (s *Service) Today() string {
	return clock.FormatDate(s.clock.Today())
}
