// Package settings stores application-wide preferences such as working days
// and which attendance fractions are enabled.
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
	ErrInvalidFractions   = errors.New("attendance fractions must be unique numbers from 1 to 3")
)

const (
	defaultWorkingDays = "1,2,3,4,5,6"
	defaultFractions   = "1,2,3"
)

// Settings is the public settings payload.
type Settings struct {
	WorkingDays         []int `json:"working_days"`
	AttendanceFractions []int `json:"attendance_fractions"`
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

// Get returns the stored settings with sensible defaults.
func (s *Service) Get(ctx context.Context) (Settings, error) {
	days, err := s.readDays(ctx, workingDaysKey, defaultWorkingDays, false)
	if err != nil {
		return Settings{}, err
	}
	fractions, err := s.readDays(ctx, fractionsKey, defaultFractions, true)
	if err != nil {
		return Settings{}, err
	}
	return Settings{WorkingDays: days, AttendanceFractions: fractions}, nil
}

func (s *Service) readDays(ctx context.Context, key, fallback string, fraction bool) ([]int, error) {
	raw, err := getValue(ctx, s.db, key)
	if errors.Is(err, sql.ErrNoRows) {
		parsed, _ := parseDays(fallback)
		return parsed, nil
	}
	if err != nil {
		return nil, err
	}

	parsed, err := parseDays(raw)
	if err != nil {
		parsed, _ = parseDays(fallback)
		return parsed, nil
	}

	normalized := normalizedValues(parsed, 1, 7, false)
	if fraction {
		normalized = normalizedValues(parsed, 1, 3, true)
	}
	if len(normalized) == 0 && fraction {
		return []int{}, nil
	}
	if len(normalized) == 0 {
		parsed, _ = parseDays(fallback)
		return parsed, nil
	}
	return normalized, nil
}

// Update saves the fields that are provided and returns the resulting settings.
// A nil pointer keeps the current value.
func (s *Service) Update(ctx context.Context, workingDays, fractions *[]int) (Settings, error) {
	current, err := s.Get(ctx)
	if err != nil {
		return Settings{}, err
	}

	days := current.WorkingDays
	if workingDays != nil {
		if len(*workingDays) == 0 {
			return Settings{}, ErrEmptyWorkingDays
		}
		seen := make(map[int]bool, len(*workingDays))
		for _, day := range *workingDays {
			if day < 1 || day > 7 || seen[day] {
				return Settings{}, ErrInvalidWorkingDays
			}
			seen[day] = true
		}
		days = normalizedValues(*workingDays, 1, 7, false)
	}

	enabled := current.AttendanceFractions
	if fractions != nil {
		if len(*fractions) > 3 {
			return Settings{}, ErrInvalidFractions
		}
		seen := make(map[int]bool, len(*fractions))
		for _, value := range *fractions {
			if value < 1 || value > 3 || seen[value] {
				return Settings{}, ErrInvalidFractions
			}
			seen[value] = true
		}
		enabled = normalizedValues(*fractions, 1, 3, true)
	}

	if err := setValue(ctx, s.db, workingDaysKey, formatDays(days)); err != nil {
		return Settings{}, err
	}
	if err := setValue(ctx, s.db, fractionsKey, formatDays(enabled)); err != nil {
		return Settings{}, err
	}

	return Settings{WorkingDays: days, AttendanceFractions: enabled}, nil
}

// normalizedValues filters, de-duplicates, and sorts values in [min, max].
// When allowEmpty is true an empty result is meaningful (all fractions off).
func normalizedValues(values []int, min, max int, allowEmpty bool) []int {
	seen := make(map[int]bool, len(values))
	result := make([]int, 0, len(values))
	for _, value := range values {
		if value < min || value > max || seen[value] {
			continue
		}
		seen[value] = true
		result = append(result, value)
	}
	sort.Ints(result)
	if len(result) == 0 && !allowEmpty {
		return nil
	}
	return result
}
