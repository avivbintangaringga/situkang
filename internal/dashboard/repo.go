package dashboard

import (
	"context"
	"database/sql"
	"fmt"
)

// WorkerTotal is one worker's unsettled balance.
type WorkerTotal struct {
	WorkerID int64  `json:"worker_id"`
	Name     string `json:"name"`
	Role     string `json:"role"`
	Quarters int    `json:"quarters"`
	Gross    int64  `json:"gross"`
	Advances int64  `json:"advances"`
	Net      int64  `json:"net"`
}

// Totals aggregates every unsettled worker.
type Totals struct {
	Days     int   `json:"days"`
	Quarters int   `json:"quarters"`
	Gross    int64 `json:"gross"`
	Advances int64 `json:"advances"`
	Net      int64 `json:"net"`
}

// Dashboard is the Beranda payload.
type Dashboard struct {
	Workers      []WorkerTotal `json:"workers"`
	Totals       Totals        `json:"totals"`
	DraftRecapID *int64        `json:"draft_recap_id"`
}

type attendanceTotal struct {
	Quarters int
	Gross    int64
}

type workerRow struct {
	Name string
	Role string
}

func attendanceTotals(ctx context.Context, db *sql.DB) (map[int64]attendanceTotal, error) {
	const query = `
		SELECT worker_id, SUM(quarters), SUM(daily_fee_snapshot * quarters / 4)
		FROM attendances
		WHERE recap_id IS NULL
		GROUP BY worker_id`

	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("load unsettled attendance totals: %w", err)
	}
	defer rows.Close()

	totals := make(map[int64]attendanceTotal)
	for rows.Next() {
		var (
			workerID int64
			total    attendanceTotal
		)
		if err := rows.Scan(&workerID, &total.Quarters, &total.Gross); err != nil {
			return nil, fmt.Errorf("scan attendance total: %w", err)
		}
		totals[workerID] = total
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("load unsettled attendance totals: %w", err)
	}
	return totals, nil
}

// recordedDays counts the distinct dates with unsettled attendance, so two
// workers on the same day still count as one recorded day.
func recordedDays(ctx context.Context, db *sql.DB) (int, error) {
	var days int
	err := db.QueryRowContext(ctx,
		`SELECT COUNT(DISTINCT date) FROM attendances WHERE recap_id IS NULL`).Scan(&days)
	if err != nil {
		return 0, fmt.Errorf("count recorded days: %w", err)
	}
	return days, nil
}

func advanceTotals(ctx context.Context, db *sql.DB) (map[int64]int64, error) {
	const query = `
		SELECT worker_id, SUM(amount)
		FROM cash_advances
		WHERE recap_id IS NULL
		GROUP BY worker_id`

	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("load unsettled advance totals: %w", err)
	}
	defer rows.Close()

	totals := make(map[int64]int64)
	for rows.Next() {
		var (
			workerID int64
			amount   int64
		)
		if err := rows.Scan(&workerID, &amount); err != nil {
			return nil, fmt.Errorf("scan advance total: %w", err)
		}
		totals[workerID] = amount
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("load unsettled advance totals: %w", err)
	}
	return totals, nil
}

func allWorkers(ctx context.Context, db *sql.DB) (map[int64]workerRow, error) {
	rows, err := db.QueryContext(ctx, `SELECT id, name, role FROM workers`)
	if err != nil {
		return nil, fmt.Errorf("load dashboard workers: %w", err)
	}
	defer rows.Close()

	workers := make(map[int64]workerRow)
	for rows.Next() {
		var (
			id   int64
			item workerRow
		)
		if err := rows.Scan(&id, &item.Name, &item.Role); err != nil {
			return nil, fmt.Errorf("scan dashboard worker: %w", err)
		}
		workers[id] = item
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("load dashboard workers: %w", err)
	}
	return workers, nil
}

func draftRecapID(ctx context.Context, db *sql.DB) (*int64, error) {
	var id int64
	err := db.QueryRowContext(ctx,
		`SELECT id FROM recaps WHERE status = 'draft' ORDER BY id LIMIT 1`).Scan(&id)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("load draft recap: %w", err)
	}
	return &id, nil
}
