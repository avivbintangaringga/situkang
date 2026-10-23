// Package settings stores application-wide preferences such as working days.
package settings

import (
	"context"
	"database/sql"
	"errors"
	"sort"

	"situkang/internal/clock"
)

var (
	ErrEmptyWorkingDays   = errors.New("working days must not be empty")
	ErrInvalidWorkingDays = errors.New("working days must be unique numbers from 1 to 7")
)

// Settings is the public settings payload.
type Settings struct {
	WorkingDays []int `json:"working_days"`
}

// Service holds the settings business rules.
type Service struct {
	db    *sql.DB
	clock clock.Clock
}

// NewService builds the settings service.
func NewService(db *sql.DB, clk clock.Clock) *Service {
	return &Service{db: db, clock: clk}
}

// Get returns the stored settings, falling back to Monday-Saturday.
func (s *Service) Get(ctx context.Context) (Settings, error) {
	raw, err := getValue(ctx, s.db, workingDaysKey)
	if errors.Is(err, sql.ErrNoRows) {
		return Settings{WorkingDays: []int{1, 2, 3, 4, 5, 6}}, nil
	}
	if err != nil {
		return Settings{}, err
	}

	days, err := parseDays(raw)
	if err != nil {
		return Settings{WorkingDays: []int{1, 2, 3, 4, 5, 6}}, nil
	}
	return Settings{WorkingDays: normalized(days)}, nil
}

// Update validates and persists the working days.
func (s *Service) Update(ctx context.Context, days []int) (Settings, error) {
	if len(days) == 0 {
		return Settings{}, ErrEmptyWorkingDays
	}

	seen := make(map[int]bool, len(days))
	for _, day := range days {
		if day < 1 || day > 7 || seen[day] {
			return Settings{}, ErrInvalidWorkingDays
		}
		seen[day] = true
	}

	sorted := normalized(days)
	if err := setValue(ctx, s.db, workingDaysKey, formatDays(sorted)); err != nil {
		return Settings{}, err
	}
	return Settings{WorkingDays: sorted}, nil
}

func normalized(days []int) []int {
	seen := make(map[int]bool, len(days))
	result := make([]int, 0, len(days))
	for _, day := range days {
		if day < 1 || day > 7 || seen[day] {
			continue
		}
		seen[day] = true
		result = append(result, day)
	}
	sort.Ints(result)
	return result
}
