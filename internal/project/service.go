// Package project manages construction projects and their worker assignments.
package project

import (
	"context"
	"database/sql"
	"errors"
	"sort"
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
	ErrInvalidWorker = errors.New("invalid project worker")
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
	WorkerIDs []int64
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

// Create stores a project and its worker assignments.
func (s *Service) Create(ctx context.Context, in Input) (Project, error) {
	data, err := validate(in)
	if err != nil {
		return Project{}, err
	}
	workerIDs, err := normalizeWorkerIDs(in.WorkerIDs)
	if err != nil {
		return Project{}, err
	}
	data.CreatedAt = s.clock.Now().UTC().Format(time.RFC3339)

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return Project{}, err
	}
	defer tx.Rollback()

	ok, err := workersExist(ctx, tx, workerIDs)
	if err != nil {
		return Project{}, err
	}
	if !ok {
		return Project{}, ErrInvalidWorker
	}

	id, err := insert(ctx, tx, data)
	if err != nil {
		return Project{}, err
	}
	if err := replaceWorkers(ctx, tx, id, workerIDs); err != nil {
		return Project{}, err
	}
	if err := tx.Commit(); err != nil {
		return Project{}, err
	}
	return s.Get(ctx, id)
}

// Update replaces the editable fields and the worker assignments of a project.
func (s *Service) Update(ctx context.Context, id int64, in Input) (Project, error) {
	if id <= 0 {
		return Project{}, ErrNotFound
	}
	data, err := validate(in)
	if err != nil {
		return Project{}, err
	}
	workerIDs, err := normalizeWorkerIDs(in.WorkerIDs)
	if err != nil {
		return Project{}, err
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return Project{}, err
	}
	defer tx.Rollback()

	ok, err := workersExist(ctx, tx, workerIDs)
	if err != nil {
		return Project{}, err
	}
	if !ok {
		return Project{}, ErrInvalidWorker
	}

	if err := update(ctx, tx, id, data); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Project{}, ErrNotFound
		}
		return Project{}, err
	}
	if err := replaceWorkers(ctx, tx, id, workerIDs); err != nil {
		return Project{}, err
	}
	if err := tx.Commit(); err != nil {
		return Project{}, err
	}
	return s.Get(ctx, id)
}

// Delete removes a project. Its assignment rows are removed by cascade.
func (s *Service) Delete(ctx context.Context, id int64) error {
	if id <= 0 {
		return ErrNotFound
	}
	if _, err := s.Get(ctx, id); err != nil {
		return err
	}
	if err := remove(ctx, s.db, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrNotFound
		}
		return err
	}
	return nil
}

// Timeline returns the month view: every relevant project plus the days its
// assigned workers attended within that month.
func (s *Service) Timeline(ctx context.Context, month string) (Timeline, error) {
	first, err := parseMonth(month)
	if err != nil {
		return Timeline{}, err
	}
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

// Calendar returns the days in a month when the project's assigned workers
// attended, together with the assigned worker ids.
func (s *Service) Calendar(ctx context.Context, id int64, month string) (ProjectCalendar, error) {
	first, err := parseMonth(month)
	if err != nil {
		return ProjectCalendar{}, err
	}
	if id <= 0 {
		return ProjectCalendar{}, ErrNotFound
	}
	project, err := s.Get(ctx, id)
	if err != nil {
		return ProjectCalendar{}, err
	}

	days := daysInMonth(first)
	last := first.AddDate(0, 0, days-1)
	entries, err := calendarDays(ctx, s.db, id, clock.FormatDate(first), clock.FormatDate(last))
	if err != nil {
		return ProjectCalendar{}, err
	}

	result := ProjectCalendar{
		Month:     month,
		Days:      make([]CalendarDay, 0),
		WorkerIDs: project.WorkerIDs,
	}
	index := make(map[string]int)
	for _, entry := range entries {
		position, ok := index[entry.Date]
		if !ok {
			position = len(result.Days)
			index[entry.Date] = position
			result.Days = append(result.Days, CalendarDay{Date: entry.Date, Workers: []CalendarWorker{}})
		}
		result.Days[position].Workers = append(result.Days[position].Workers, CalendarWorker{
			WorkerID: entry.WorkerID,
			Name:     entry.Name,
			Quarters: entry.Quarters,
		})
	}
	return result, nil
}

func parseMonth(month string) (time.Time, error) {
	if len(month) != 7 || month[4] != '-' {
		return time.Time{}, ErrInvalidMonth
	}
	parsed, err := time.Parse("2006-01", month)
	if err != nil || parsed.Format("2006-01") != month {
		return time.Time{}, ErrInvalidMonth
	}
	return time.Date(parsed.Year(), parsed.Month(), 1, 0, 0, 0, 0, time.UTC), nil
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

// normalizeWorkerIDs sorts and deduplicates worker ids, rejecting non-positive
// ids.
func normalizeWorkerIDs(ids []int64) ([]int64, error) {
	seen := make(map[int64]bool, len(ids))
	result := make([]int64, 0, len(ids))
	for _, id := range ids {
		if id <= 0 {
			return nil, ErrInvalidWorker
		}
		if seen[id] {
			continue
		}
		seen[id] = true
		result = append(result, id)
	}
	sort.Slice(result, func(i, j int) bool { return result[i] < result[j] })
	return result, nil
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
