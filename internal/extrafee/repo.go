package extrafee

import (
	"context"
	"database/sql"
	"fmt"
)

// ExtraFee is a standalone bonus or deduction for one worker on one date.
type ExtraFee struct {
	ID          int64  `json:"id"`
	WorkerID    int64  `json:"worker_id"`
	Date        string `json:"date"`
	Amount      int64  `json:"amount"`
	Description string `json:"description"`
	Locked      bool   `json:"locked"`
}

const selectFields = `
	SELECT ef.id, ef.worker_id, ef.date, ef.amount, ef.description,
		EXISTS (
			SELECT 1 FROM recaps r WHERE r.id = ef.recap_id AND r.status = 'final'
		) AS locked
	FROM extra_fees ef`

func list(ctx context.Context, db *sql.DB, workerID int64, from, to string) ([]ExtraFee, error) {
	query := selectFields + ` WHERE 1 = 1`
	args := make([]any, 0, 3)

	if workerID > 0 {
		query += ` AND ef.worker_id = ?`
		args = append(args, workerID)
	}
	if from != "" {
		query += ` AND ef.date >= ?`
		args = append(args, from)
	}
	if to != "" {
		query += ` AND ef.date <= ?`
		args = append(args, to)
	}
	query += ` ORDER BY ef.date DESC, ef.id DESC`

	rows, err := db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list extra fees: %w", err)
	}
	defer rows.Close()

	items := make([]ExtraFee, 0)
	for rows.Next() {
		var item ExtraFee
		if err := rows.Scan(&item.ID, &item.WorkerID, &item.Date, &item.Amount,
			&item.Description, &item.Locked); err != nil {
			return nil, fmt.Errorf("scan extra fee: %w", err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list extra fees: %w", err)
	}
	return items, nil
}

func get(ctx context.Context, db *sql.DB, id int64) (ExtraFee, error) {
	query := selectFields + ` WHERE ef.id = ?`

	var item ExtraFee
	err := db.QueryRowContext(ctx, query, id).Scan(&item.ID, &item.WorkerID, &item.Date,
		&item.Amount, &item.Description, &item.Locked)
	if err != nil {
		return ExtraFee{}, err
	}
	return item, nil
}

func insert(ctx context.Context, db *sql.DB, workerID int64, date string, amount int64, description, now string) (ExtraFee, error) {
	const query = `
		INSERT INTO extra_fees (worker_id, date, amount, description, created_at)
		VALUES (?, ?, ?, ?, ?)`

	res, err := db.ExecContext(ctx, query, workerID, date, amount, description, now)
	if err != nil {
		return ExtraFee{}, fmt.Errorf("insert extra fee: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return ExtraFee{}, fmt.Errorf("insert extra fee: %w", err)
	}
	return get(ctx, db, id)
}

func update(ctx context.Context, db *sql.DB, id int64, date string, amount int64, description string) (ExtraFee, error) {
	const query = `UPDATE extra_fees SET date = ?, amount = ?, description = ? WHERE id = ?`

	if _, err := db.ExecContext(ctx, query, date, amount, description, id); err != nil {
		return ExtraFee{}, fmt.Errorf("update extra fee: %w", err)
	}
	return get(ctx, db, id)
}

func remove(ctx context.Context, db *sql.DB, id int64) error {
	if _, err := db.ExecContext(ctx, `DELETE FROM extra_fees WHERE id = ?`, id); err != nil {
		return fmt.Errorf("delete extra fee: %w", err)
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
