package cashadvance

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
)

// CashAdvance is a kas bon: money a worker took in advance.
type CashAdvance struct {
	ID       int64  `json:"id"`
	WorkerID int64  `json:"worker_id"`
	Date     string `json:"date"`
	Amount   int64  `json:"amount"`
	Note     string `json:"note"`
	Locked   bool   `json:"locked"`
}

func list(ctx context.Context, db *sql.DB, workerID int64, from, to string) ([]CashAdvance, error) {
	query := `SELECT id, worker_id, date, amount, COALESCE(note, ''), recap_id FROM cash_advances WHERE 1 = 1`
	args := make([]any, 0, 3)

	if workerID > 0 {
		query += ` AND worker_id = ?`
		args = append(args, workerID)
	}
	if from != "" {
		query += ` AND date >= ?`
		args = append(args, from)
	}
	if to != "" {
		query += ` AND date <= ?`
		args = append(args, to)
	}
	query += ` ORDER BY date DESC, id DESC`

	rows, err := db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list cash advances: %w", err)
	}
	defer rows.Close()

	items := make([]CashAdvance, 0)
	for rows.Next() {
		var (
			item    CashAdvance
			recapID sql.NullInt64
		)
		if err := rows.Scan(&item.ID, &item.WorkerID, &item.Date, &item.Amount, &item.Note, &recapID); err != nil {
			return nil, fmt.Errorf("scan cash advance: %w", err)
		}
		item.Locked = recapID.Valid
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list cash advances: %w", err)
	}
	return items, nil
}

func get(ctx context.Context, db *sql.DB, id int64) (CashAdvance, error) {
	const query = `SELECT id, worker_id, date, amount, COALESCE(note, ''), recap_id FROM cash_advances WHERE id = ?`

	var (
		item    CashAdvance
		recapID sql.NullInt64
	)
	err := db.QueryRowContext(ctx, query, id).Scan(&item.ID, &item.WorkerID, &item.Date, &item.Amount, &item.Note, &recapID)
	if err != nil {
		return CashAdvance{}, err
	}
	item.Locked = recapID.Valid
	return item, nil
}

func insert(ctx context.Context, db *sql.DB, workerID int64, date string, amount int64, note, now string) (CashAdvance, error) {
	const query = `INSERT INTO cash_advances (worker_id, date, amount, note, created_at) VALUES (?, ?, ?, ?, ?)`

	res, err := db.ExecContext(ctx, query, workerID, date, amount, nullableNote(note), now)
	if err != nil {
		return CashAdvance{}, fmt.Errorf("insert cash advance: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return CashAdvance{}, fmt.Errorf("insert cash advance: %w", err)
	}
	return get(ctx, db, id)
}

func update(ctx context.Context, db *sql.DB, id int64, date string, amount int64, note string) (CashAdvance, error) {
	const query = `UPDATE cash_advances SET date = ?, amount = ?, note = ? WHERE id = ?`

	if _, err := db.ExecContext(ctx, query, date, amount, nullableNote(note), id); err != nil {
		return CashAdvance{}, fmt.Errorf("update cash advance: %w", err)
	}
	return get(ctx, db, id)
}

func remove(ctx context.Context, db *sql.DB, id int64) error {
	if _, err := db.ExecContext(ctx, `DELETE FROM cash_advances WHERE id = ?`, id); err != nil {
		return fmt.Errorf("delete cash advance: %w", err)
	}
	return nil
}

func workerExists(ctx context.Context, db *sql.DB, workerID int64) (bool, error) {
	var count int
	err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM workers WHERE id = ?`, workerID).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("check worker: %w", err)
	}
	return count > 0, nil
}

func nullableNote(note string) any {
	if strings.TrimSpace(note) == "" {
		return nil
	}
	return strings.TrimSpace(note)
}
