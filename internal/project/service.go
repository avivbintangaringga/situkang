// Package project manages construction projects and their attendance tags.
package project

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"

	"situkang/internal/clock"
)

var (
	ErrNotFound      = errors.New("project not found")
	ErrInvalidName   = errors.New("invalid project name")
	ErrInvalidStatus = errors.New("invalid project status")
	ErrInvalidDate   = errors.New("invalid project date")
	ErrInvalidRange  = errors.New("invalid project date range")
	ErrNotesTooLong  = errors.New("project notes too long")
	ErrInUse         = errors.New("project is tagged on attendance")
	ErrInvalidMonth  = errors.New("invalid month")
)

const (
	maxNameLength  = 100
	maxNotesLength = 500
)

// Input is the editable payload of a project.
type Input struct {
	Name      string
	StartDate string
	EndDate   string
	Status    string
	Notes     string
}

// Service holds the project business rules.
type Service struct {
	db    *sql.DB
	clock clock.Clock
}

// NewService builds the project service.
func NewService(db *sql.DB, clk clock.Clock) *Service {
	return &Service{db: db, clock: clk}
}

// List returns all projects ordered by name.
func (s *Service) List(ctx context.Context) ([]Project, error) {
	return list(ctx, s.db)
}

// Get returns one project or ErrNotFound.
func (s *Service) Get(ctx context.Context, id int64) (Project, error) {
	if id <= 0 {
		return Project{}, ErrNotFound
	}
	item, err := get(ctx, s.db, id)
	if errors.Is(err, sql.ErrNoRows) {
		return Project{}, ErrNotFound
	}
	return item, err
}

// Create stores a project.
func (s *Service) Create(ctx context.Context, in Input) (Project, error) {
	data, err := validate(in)
	if err != nil {
		return Project{}, err
	}
	data.CreatedAt = s.clock.Now().UTC().Format(time.RFC3339)
	return insert(ctx, s.db, data)
}

// Update replaces the editable fields of a project.
func (s *Service) Update(ctx context.Context, id int64, in Input) (Project, error) {
	if id <= 0 {
		return Project{}, ErrNotFound
	}
	data, err := validate(in)
	if err != nil {
		return Project{}, err
	}
	updated, err := update(ctx, s.db, id, data)
	if errors.Is(err, sql.ErrNoRows) {
		return Project{}, ErrNotFound
	}
	return updated, err
}

// Delete removes an unused project. Projects tagged on attendance are kept so
// history is not silently rewritten.
func (s *Service) Delete(ctx context.Context, id int64) error {
	if id <= 0 {
		return ErrNotFound
	}
	if _, err := s.Get(ctx, id); err != nil {
		return err
	}
	used, err := usageCount(ctx, s.db, id)
	if err != nil {
		return err
	}
	if used > 0 {
		return ErrInUse
	}
	if err := remove(ctx, s.db, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrNotFound
		}
		return err
	}
	return nil
}

// Timeline returns the month view: every relevant project plus the days it was
// tagged on attendance within that month.
func (s *Service) Timeline(ctx context.Context, month string) (Timeline, error) {
	start, err := time.Parse("2006-01", month)
	if err != nil {
		return Timeline{}, ErrInvalidMonth
	}
	first := time.Date(start.Year(), start.Month(), 1, 0, 0, 0, 0, time.UTC)
	days := daysInMonth(first)
	last := first.AddDate(0, 0, days-1)

	all, err := timeline(ctx, s.db, clock.FormatDate(first), clock.FormatDate(last))
	if err != nil {
		return Timeline{}, err
	}

	result := Timeline{
		Month:    month,
		Days:     days,
		Projects: make([]TimelineProject, 0, len(all)),
	}
	for _, item := range all {
		if relevant(item, first, last) {
			result.Projects = append(result.Projects, item)
		}
	}
	return result, nil
}

func relevant(item TimelineProject, first, last time.Time) bool {
	if len(item.TaggedDays) > 0 {
		return true
	}
	if item.StartDate == nil && item.EndDate == nil {
		return false
	}
	if item.StartDate != nil {
		start, err := clock.ParseDate(*item.StartDate)
		if err == nil && start.After(last) {
			return false
		}
	}
	if item.EndDate != nil {
		end, err := clock.ParseDate(*item.EndDate)
		if err == nil && end.Before(first) {
			return false
		}
	}
	return true
}

func daysInMonth(first time.Time) int {
	return first.AddDate(0, 1, -1).Day()
}

func validate(in Input) (projectData, error) {
	name := strings.TrimSpace(in.Name)
	if name == "" || len([]rune(name)) > maxNameLength {
		return projectData{}, ErrInvalidName
	}

	status := strings.TrimSpace(in.Status)
	switch status {
	case "aktif", "selesai", "ditunda":
	default:
		return projectData{}, ErrInvalidStatus
	}

	notes := strings.TrimSpace(in.Notes)
	if len([]rune(notes)) > maxNotesLength {
		return projectData{}, ErrNotesTooLong
	}

	var start, end *string
	if raw := strings.TrimSpace(in.StartDate); raw != "" {
		parsed, err := clock.ParseDate(raw)
		if err != nil {
			return projectData{}, ErrInvalidDate
		}
		value := clock.FormatDate(parsed)
		start = &value
	}
	if raw := strings.TrimSpace(in.EndDate); raw != "" {
		parsed, err := clock.ParseDate(raw)
		if err != nil {
			return projectData{}, ErrInvalidDate
		}
		value := clock.FormatDate(parsed)
		end = &value
	}
	if start != nil && end != nil && *end < *start {
		return projectData{}, ErrInvalidRange
	}

	return projectData{
		Name:      name,
		StartDate: start,
		EndDate:   end,
		Status:    status,
		Notes:     notes,
	}, nil
}
