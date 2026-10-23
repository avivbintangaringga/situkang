package worker

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
)

// Worker is a tukang. Workers never sign in; they are records only.
type Worker struct {
	ID        int64  `json:"id"`
	Name      string `json:"name"`
	Role      string `json:"role"`
	DailyFee  int64  `json:"daily_fee"`
	Active    bool   `json:"active"`
	CreatedAt string `json:"created_at"`
}

const selectColumns = `id, name, role, daily_fee, active, created_at`

func list(ctx context.Context, db *sql.DB, search string, includeInactive bool) ([]Worker, error) {
	query := `SELECT ` + selectColumns + ` FROM workers WHERE 1 = 1`
	args := make([]any, 0, 2)

	if search != "" {
		query += ` AND (name LIKE ? ESCAPE '\' OR role LIKE ? ESCAPE '\')`
		like := "%" + escapeLike(search) + "%"
		args = append(args, like, like)
	}
	if !includeInactive {
		query += ` AND active = 1`
	}
	query += ` ORDER BY active DESC, name COLLATE NOCASE`

	rows, err := db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list workers: %w", err)
	}
	defer rows.Close()

	workers := make([]Worker, 0)
	for rows.Next() {
		var w Worker
		if err := rows.Scan(&w.ID, &w.Name, &w.Role, &w.DailyFee, &w.Active, &w.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan worker: %w", err)
		}
		workers = append(workers, w)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list workers: %w", err)
	}
	return workers, nil
}

func get(ctx context.Context, db *sql.DB, id int64) (Worker, error) {
	query := `SELECT ` + selectColumns + ` FROM workers WHERE id = ?`

	var w Worker
	err := db.QueryRowContext(ctx, query, id).Scan(&w.ID, &w.Name, &w.Role, &w.DailyFee, &w.Active, &w.CreatedAt)
	return w, err
}

func insert(ctx context.Context, db *sql.DB, data workerData) (Worker, error) {
	const query = `INSERT INTO workers (name, role, daily_fee, active, created_at) VALUES (?, ?, ?, ?, ?)`

	res, err := db.ExecContext(ctx, query, data.Name, data.Role, data.DailyFee, data.Active, data.CreatedAt)
	if err != nil {
		return Worker{}, fmt.Errorf("insert worker: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return Worker{}, fmt.Errorf("insert worker: %w", err)
	}
	return get(ctx, db, id)
}

func update(ctx context.Context, db *sql.DB, id int64, data workerData) (Worker, error) {
	const query = `UPDATE workers SET name = ?, role = ?, daily_fee = ?, active = ? WHERE id = ?`

	res, err := db.ExecContext(ctx, query, data.Name, data.Role, data.DailyFee, data.Active, id)
	if err != nil {
		return Worker{}, fmt.Errorf("update worker: %w", err)
	}
	if affected, err := res.RowsAffected(); err == nil && affected == 0 {
		return Worker{}, sql.ErrNoRows
	}
	return get(ctx, db, id)
}

func setActive(ctx context.Context, db *sql.DB, id int64, active bool) error {
	const query = `UPDATE workers SET active = ? WHERE id = ?`

	res, err := db.ExecContext(ctx, query, active, id)
	if err != nil {
		return fmt.Errorf("set worker active: %w", err)
	}
	if affected, err := res.RowsAffected(); err == nil && affected == 0 {
		return sql.ErrNoRows
	}
	return nil
}

// escapeLike stops % and _ in user input from acting as wildcards.
func escapeLike(s string) string {
	return strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`).Replace(s)
}
