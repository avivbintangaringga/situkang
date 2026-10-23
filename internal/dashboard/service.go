// Package dashboard aggregates what is still owed to each worker.
package dashboard

import (
	"context"
	"database/sql"
	"sort"
	"strings"

	"situkang/internal/clock"
)

// Service holds the dashboard business rules.
type Service struct {
	db    *sql.DB
	clock clock.Clock
}

// NewService builds the dashboard service.
func NewService(db *sql.DB, clk clock.Clock) *Service {
	return &Service{db: db, clock: clk}
}

// Get returns every worker with unsettled attendance or kas bon, plus totals
// and the current draft recap id (if any).
func (s *Service) Get(ctx context.Context) (Dashboard, error) {
	attendance, err := attendanceTotals(ctx, s.db)
	if err != nil {
		return Dashboard{}, err
	}
	advances, err := advanceTotals(ctx, s.db)
	if err != nil {
		return Dashboard{}, err
	}
	workers, err := allWorkers(ctx, s.db)
	if err != nil {
		return Dashboard{}, err
	}
	draft, err := draftRecapID(ctx, s.db)
	if err != nil {
		return Dashboard{}, err
	}
	days, err := recordedDays(ctx, s.db)
	if err != nil {
		return Dashboard{}, err
	}

	pending := make(map[int64]bool, len(attendance)+len(advances))
	for id := range attendance {
		pending[id] = true
	}
	for id := range advances {
		pending[id] = true
	}

	result := Dashboard{
		Workers:      make([]WorkerTotal, 0, len(pending)),
		DraftRecapID: draft,
	}
	result.Totals.Days = days
	for id := range pending {
		worker := workers[id]
		line := WorkerTotal{
			WorkerID: id,
			Name:     worker.Name,
			Role:     worker.Role,
			Quarters: attendance[id].Quarters,
			Gross:    attendance[id].Gross,
			Advances: advances[id],
		}
		line.Net = line.Gross - line.Advances

		result.Totals.Quarters += line.Quarters
		result.Totals.Gross += line.Gross
		result.Totals.Advances += line.Advances
		result.Totals.Net += line.Net
		result.Workers = append(result.Workers, line)
	}

	sort.Slice(result.Workers, func(i, j int) bool {
		return strings.ToLower(result.Workers[i].Name) < strings.ToLower(result.Workers[j].Name)
	})
	return result, nil
}
