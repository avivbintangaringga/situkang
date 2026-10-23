package recap

import (
	"context"
	"database/sql"
	"fmt"
)

// Recap is a weekly pay period. Only one draft may exist at a time.
type Recap struct {
	ID          int64   `json:"id"`
	Name        string  `json:"name"`
	Mode        string  `json:"mode"`
	StartDate   *string `json:"start_date"`
	EndDate     *string `json:"end_date"`
	Status      string  `json:"status"`
	CreatedAt   string  `json:"created_at"`
	FinalizedAt *string `json:"finalized_at"`
}

// AttendanceItem is one settled day inside a recap line.
type AttendanceItem struct {
	ID       int64  `json:"id"`
	Date     string `json:"date"`
	Quarters int    `json:"quarters"`
	Earned   int64  `json:"earned"`
}

// AdvanceItem is one kas bon inside a recap line.
type AdvanceItem struct {
	ID     int64  `json:"id"`
	Date   string `json:"date"`
	Amount int64  `json:"amount"`
	Note   string `json:"note"`
}

// ExtraFee is a signed bonus or deduction attached to a recap.
type ExtraFee struct {
	ID          int64  `json:"id"`
	WorkerID    int64  `json:"worker_id"`
	Amount      int64  `json:"amount"`
	Description string `json:"description"`
}

// WorkerLine is one worker's row in a recap.
type WorkerLine struct {
	WorkerID     int64            `json:"worker_id"`
	Name         string           `json:"name"`
	Role         string           `json:"role"`
	Quarters     int              `json:"quarters"`
	Gross        int64            `json:"gross"`
	Advances     int64            `json:"advances"`
	Extras       int64            `json:"extras"`
	Net          int64            `json:"net"`
	Paid         bool             `json:"paid"`
	Warning      int              `json:"warning"`
	Attendance   []AttendanceItem `json:"attendance"`
	CashAdvances []AdvanceItem    `json:"cash_advances"`
	ExtraFees    []ExtraFee       `json:"extra_fees"`
}

// Totals aggregates every worker line.
type Totals struct {
	Gross    int64 `json:"gross"`
	Advances int64 `json:"advances"`
	Extras   int64 `json:"extras"`
	Net      int64 `json:"net"`
}

// Detail is the full draft view.
type Detail struct {
	Recap       Recap        `json:"recap"`
	Workers     []WorkerLine `json:"workers"`
	Totals      Totals       `json:"totals"`
	PeriodStart string       `json:"period_start"`
	PeriodEnd   string       `json:"period_end"`
}

type attRow struct {
	ID       int64
	WorkerID int64
	Date     string
	Quarters int
	Snapshot int64
}

type advRow struct {
	ID       int64
	WorkerID int64
	Date     string
	Amount   int64
	Note     string
}

type feeRow struct {
	ID          int64
	WorkerID    int64
	Amount      int64
	Description string
}

type workerRow struct {
	ID   int64
	Name string
	Role string
}

type queryer interface {
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

type execer interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
}

func getRecap(ctx context.Context, q queryer, id int64) (Recap, error) {
	const query = `SELECT id, mode, start_date, end_date, status, created_at, finalized_at FROM recaps WHERE id = ?`

	var (
		recap       Recap
		start, end  sql.NullString
		finalizedAt sql.NullString
	)
	err := q.QueryRowContext(ctx, query, id).Scan(&recap.ID, &recap.Mode, &start, &end,
		&recap.Status, &recap.CreatedAt, &finalizedAt)
	if err != nil {
		return Recap{}, err
	}
	if start.Valid {
		recap.StartDate = &start.String
	}
	if end.Valid {
		recap.EndDate = &end.String
	}
	if finalizedAt.Valid {
		recap.FinalizedAt = &finalizedAt.String
	}
	recap.Name = recapName(recap.CreatedAt)
	return recap, nil
}

func listRecaps(ctx context.Context, db *sql.DB) ([]Recap, error) {
	rows, err := db.QueryContext(ctx,
		`SELECT id, mode, start_date, end_date, status, created_at, finalized_at FROM recaps ORDER BY created_at DESC, id DESC`)
	if err != nil {
		return nil, fmt.Errorf("list recaps: %w", err)
	}
	defer rows.Close()

	recaps := make([]Recap, 0)
	for rows.Next() {
		var (
			recap       Recap
			start, end  sql.NullString
			finalizedAt sql.NullString
		)
		if err := rows.Scan(&recap.ID, &recap.Mode, &start, &end, &recap.Status,
			&recap.CreatedAt, &finalizedAt); err != nil {
			return nil, fmt.Errorf("scan recap: %w", err)
		}
		if start.Valid {
			recap.StartDate = &start.String
		}
		if end.Valid {
			recap.EndDate = &end.String
		}
		if finalizedAt.Valid {
			recap.FinalizedAt = &finalizedAt.String
		}
		recap.Name = recapName(recap.CreatedAt)
		recaps = append(recaps, recap)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list recaps: %w", err)
	}
	return recaps, nil
}

func draftExists(ctx context.Context, q queryer) (bool, error) {
	var count int
	if err := q.QueryRowContext(ctx, `SELECT COUNT(*) FROM recaps WHERE status = 'draft'`).Scan(&count); err != nil {
		return false, fmt.Errorf("check draft: %w", err)
	}
	return count > 0, nil
}

func latestFinalID(ctx context.Context, q queryer) (int64, error) {
	var id int64
	err := q.QueryRowContext(ctx, `SELECT COALESCE(MAX(id), 0) FROM recaps WHERE status = 'final'`).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("latest final recap: %w", err)
	}
	return id, nil
}

func insertRecap(ctx context.Context, q execer, mode string, start, end *string, now string) (int64, error) {
	res, err := q.ExecContext(ctx,
		`INSERT INTO recaps (mode, start_date, end_date, status, created_at) VALUES (?, ?, ?, 'draft', ?)`,
		mode, start, end, now)
	if err != nil {
		return 0, fmt.Errorf("insert recap: %w", err)
	}
	return res.LastInsertId()
}

func setStatus(ctx context.Context, q execer, id int64, status string, finalizedAt *string) error {
	if _, err := q.ExecContext(ctx,
		`UPDATE recaps SET status = ?, finalized_at = ? WHERE id = ?`, status, finalizedAt, id); err != nil {
		return fmt.Errorf("update recap status: %w", err)
	}
	return nil
}

func deleteRecap(ctx context.Context, q execer, id int64) error {
	if _, err := q.ExecContext(ctx, `DELETE FROM recaps WHERE id = ?`, id); err != nil {
		return fmt.Errorf("delete recap: %w", err)
	}
	return nil
}

func rangeClause(start, end *string) (string, []any) {
	clause := ""
	args := make([]any, 0, 2)
	if start != nil {
		clause += " AND date >= ?"
		args = append(args, *start)
	}
	if end != nil {
		clause += " AND date <= ?"
		args = append(args, *end)
	}
	return clause, args
}

func unsettledAttendances(ctx context.Context, q queryer, start, end *string) ([]attRow, error) {
	clause, args := rangeClause(start, end)
	query := `SELECT id, worker_id, date, quarters, daily_fee_snapshot FROM attendances WHERE recap_id IS NULL` + clause + ` ORDER BY date, id`

	rows, err := q.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("load recap attendances: %w", err)
	}
	defer rows.Close()

	items := make([]attRow, 0)
	for rows.Next() {
		var item attRow
		if err := rows.Scan(&item.ID, &item.WorkerID, &item.Date, &item.Quarters, &item.Snapshot); err != nil {
			return nil, fmt.Errorf("scan recap attendance: %w", err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("load recap attendances: %w", err)
	}
	return items, nil
}

func attendanceWarnings(ctx context.Context, q queryer, start string) (map[int64]int, error) {
	const query = `SELECT worker_id, COUNT(*) FROM attendances WHERE recap_id IS NULL AND date < ? GROUP BY worker_id`

	rows, err := q.QueryContext(ctx, query, start)
	if err != nil {
		return nil, fmt.Errorf("load attendance warnings: %w", err)
	}
	defer rows.Close()

	warnings := make(map[int64]int)
	for rows.Next() {
		var (
			workerID int64
			count    int
		)
		if err := rows.Scan(&workerID, &count); err != nil {
			return nil, fmt.Errorf("scan attendance warning: %w", err)
		}
		warnings[workerID] = count
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("load attendance warnings: %w", err)
	}
	return warnings, nil
}

func unsettledAdvances(ctx context.Context, q queryer, start, end *string) ([]advRow, error) {
	clause, args := rangeClause(start, end)
	query := `SELECT id, worker_id, date, amount, COALESCE(note, '') FROM cash_advances WHERE recap_id IS NULL` + clause + ` ORDER BY date, id`

	rows, err := q.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("load recap advances: %w", err)
	}
	defer rows.Close()

	items := make([]advRow, 0)
	for rows.Next() {
		var item advRow
		if err := rows.Scan(&item.ID, &item.WorkerID, &item.Date, &item.Amount, &item.Note); err != nil {
			return nil, fmt.Errorf("scan recap advance: %w", err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("load recap advances: %w", err)
	}
	return items, nil
}

func feesForRecap(ctx context.Context, q queryer, recapID int64) ([]feeRow, error) {
	rows, err := q.QueryContext(ctx,
		`SELECT id, worker_id, amount, description FROM extra_fees WHERE recap_id = ? ORDER BY id`, recapID)
	if err != nil {
		return nil, fmt.Errorf("load extra fees: %w", err)
	}
	defer rows.Close()

	items := make([]feeRow, 0)
	for rows.Next() {
		var item feeRow
		if err := rows.Scan(&item.ID, &item.WorkerID, &item.Amount, &item.Description); err != nil {
			return nil, fmt.Errorf("scan extra fee: %w", err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("load extra fees: %w", err)
	}
	return items, nil
}

func paidMap(ctx context.Context, q queryer, recapID int64) (map[int64]bool, error) {
	rows, err := q.QueryContext(ctx, `SELECT worker_id, paid FROM recap_paid WHERE recap_id = ?`, recapID)
	if err != nil {
		return nil, fmt.Errorf("load recap paid: %w", err)
	}
	defer rows.Close()

	paid := make(map[int64]bool)
	for rows.Next() {
		var (
			workerID int64
			value    bool
		)
		if err := rows.Scan(&workerID, &value); err != nil {
			return nil, fmt.Errorf("scan recap paid: %w", err)
		}
		paid[workerID] = value
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("load recap paid: %w", err)
	}
	return paid, nil
}

func allWorkers(ctx context.Context, q queryer) (map[int64]workerRow, error) {
	rows, err := q.QueryContext(ctx, `SELECT id, name, role FROM workers`)
	if err != nil {
		return nil, fmt.Errorf("load workers: %w", err)
	}
	defer rows.Close()

	workers := make(map[int64]workerRow)
	for rows.Next() {
		var worker workerRow
		if err := rows.Scan(&worker.ID, &worker.Name, &worker.Role); err != nil {
			return nil, fmt.Errorf("scan worker: %w", err)
		}
		workers[worker.ID] = worker
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("load workers: %w", err)
	}
	return workers, nil
}

func upsertPaid(ctx context.Context, q execer, recapID, workerID int64, paid bool) error {
	const query = `
		INSERT INTO recap_paid (recap_id, worker_id, paid) VALUES (?, ?, ?)
		ON CONFLICT (recap_id, worker_id) DO UPDATE SET paid = excluded.paid`

	if _, err := q.ExecContext(ctx, query, recapID, workerID, paid); err != nil {
		return fmt.Errorf("save recap paid: %w", err)
	}
	return nil
}

func insertFee(ctx context.Context, q execer, recapID, workerID, amount int64, description, now string) (int64, error) {
	res, err := q.ExecContext(ctx,
		`INSERT INTO extra_fees (worker_id, recap_id, amount, description, created_at) VALUES (?, ?, ?, ?, ?)`,
		workerID, recapID, amount, description, now)
	if err != nil {
		return 0, fmt.Errorf("insert extra fee: %w", err)
	}
	return res.LastInsertId()
}

func updateFee(ctx context.Context, q execer, id, amount int64, description string) error {
	if _, err := q.ExecContext(ctx,
		`UPDATE extra_fees SET amount = ?, description = ? WHERE id = ?`, amount, description, id); err != nil {
		return fmt.Errorf("update extra fee: %w", err)
	}
	return nil
}

func deleteFee(ctx context.Context, q execer, id int64) error {
	if _, err := q.ExecContext(ctx, `DELETE FROM extra_fees WHERE id = ?`, id); err != nil {
		return fmt.Errorf("delete extra fee: %w", err)
	}
	return nil
}

func feeOwner(ctx context.Context, q queryer, id int64) (int64, int64, error) {
	var recapID, workerID int64
	err := q.QueryRowContext(ctx, `SELECT recap_id, worker_id FROM extra_fees WHERE id = ?`, id).Scan(&recapID, &workerID)
	return recapID, workerID, err
}

func workerExists(ctx context.Context, q queryer, workerID int64) (bool, error) {
	var count int
	if err := q.QueryRowContext(ctx, `SELECT COUNT(*) FROM workers WHERE id = ?`, workerID).Scan(&count); err != nil {
		return false, fmt.Errorf("check worker: %w", err)
	}
	return count > 0, nil
}

func settleAttendances(ctx context.Context, q execer, ids []int64, recapID int64) error {
	for _, id := range ids {
		if _, err := q.ExecContext(ctx, `UPDATE attendances SET recap_id = ? WHERE id = ?`, recapID, id); err != nil {
			return fmt.Errorf("settle attendance %d: %w", id, err)
		}
	}
	return nil
}

func settleAdvances(ctx context.Context, q execer, ids []int64, recapID int64) error {
	for _, id := range ids {
		if _, err := q.ExecContext(ctx, `UPDATE cash_advances SET recap_id = ? WHERE id = ?`, recapID, id); err != nil {
			return fmt.Errorf("settle advance %d: %w", id, err)
		}
	}
	return nil
}

func insertLine(ctx context.Context, q execer, recapID int64, line WorkerLine) error {
	const query = `
		INSERT INTO recap_lines
			(recap_id, worker_id, worker_name, worker_role, days_quarters, gross, advances, extras, net)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`

	if _, err := q.ExecContext(ctx, query, recapID, line.WorkerID, line.Name, line.Role,
		line.Quarters, line.Gross, line.Advances, line.Extras, line.Net); err != nil {
		return fmt.Errorf("insert recap line: %w", err)
	}
	return nil
}

func insertAutoAdvance(ctx context.Context, q execer, workerID, recapID, amount int64, note, date, now string) error {
	const query = `
		INSERT INTO cash_advances (worker_id, date, amount, note, source_recap_id, created_at)
		VALUES (?, ?, ?, ?, ?, ?)`

	if _, err := q.ExecContext(ctx, query, workerID, date, amount, note, recapID, now); err != nil {
		return fmt.Errorf("insert carried balance: %w", err)
	}
	return nil
}

func detachWorkerFees(ctx context.Context, q execer, recapID, workerID int64) error {
	if _, err := q.ExecContext(ctx,
		`UPDATE extra_fees SET recap_id = NULL WHERE recap_id = ? AND worker_id = ?`, recapID, workerID); err != nil {
		return fmt.Errorf("detach extra fees: %w", err)
	}
	return nil
}

func clearRecapItems(ctx context.Context, q execer, recapID int64) error {
	if _, err := q.ExecContext(ctx,
		`UPDATE attendances SET recap_id = NULL WHERE recap_id = ?`, recapID); err != nil {
		return fmt.Errorf("reopen attendances: %w", err)
	}
	if _, err := q.ExecContext(ctx,
		`UPDATE cash_advances SET recap_id = NULL WHERE recap_id = ?`, recapID); err != nil {
		return fmt.Errorf("reopen advances: %w", err)
	}
	return nil
}

func deleteAutoAdvances(ctx context.Context, q execer, recapID int64) error {
	if _, err := q.ExecContext(ctx,
		`DELETE FROM cash_advances WHERE source_recap_id = ?`, recapID); err != nil {
		return fmt.Errorf("delete carried balance: %w", err)
	}
	return nil
}

func deleteLines(ctx context.Context, q execer, recapID int64) error {
	if _, err := q.ExecContext(ctx, `DELETE FROM recap_lines WHERE recap_id = ?`, recapID); err != nil {
		return fmt.Errorf("delete recap lines: %w", err)
	}
	return nil
}

func deleteFeesForRecap(ctx context.Context, q execer, recapID int64) error {
	if _, err := q.ExecContext(ctx, `DELETE FROM extra_fees WHERE recap_id = ?`, recapID); err != nil {
		return fmt.Errorf("delete extra fees: %w", err)
	}
	return nil
}

func deletePaid(ctx context.Context, q execer, recapID int64) error {
	if _, err := q.ExecContext(ctx, `DELETE FROM recap_paid WHERE recap_id = ?`, recapID); err != nil {
		return fmt.Errorf("delete recap paid: %w", err)
	}
	return nil
}

type lineRow struct {
	WorkerID int64
	Name     string
	Role     string
	Quarters int
	Gross    int64
	Advances int64
	Extras   int64
	Net      int64
}

func recapLineRows(ctx context.Context, q queryer, recapID int64) ([]lineRow, error) {
	const query = `
		SELECT worker_id, worker_name, worker_role, days_quarters, gross, advances, extras, net
		FROM recap_lines WHERE recap_id = ? ORDER BY worker_name COLLATE NOCASE`

	rows, err := q.QueryContext(ctx, query, recapID)
	if err != nil {
		return nil, fmt.Errorf("load recap lines: %w", err)
	}
	defer rows.Close()

	lines := make([]lineRow, 0)
	for rows.Next() {
		var line lineRow
		if err := rows.Scan(&line.WorkerID, &line.Name, &line.Role, &line.Quarters,
			&line.Gross, &line.Advances, &line.Extras, &line.Net); err != nil {
			return nil, fmt.Errorf("scan recap line: %w", err)
		}
		lines = append(lines, line)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("load recap lines: %w", err)
	}
	return lines, nil
}

func settledAttendances(ctx context.Context, q queryer, recapID int64) ([]attRow, error) {
	const query = `
		SELECT id, worker_id, date, quarters, daily_fee_snapshot
		FROM attendances WHERE recap_id = ? ORDER BY date, id`

	rows, err := q.QueryContext(ctx, query, recapID)
	if err != nil {
		return nil, fmt.Errorf("load settled attendances: %w", err)
	}
	defer rows.Close()

	items := make([]attRow, 0)
	for rows.Next() {
		var item attRow
		if err := rows.Scan(&item.ID, &item.WorkerID, &item.Date, &item.Quarters, &item.Snapshot); err != nil {
			return nil, fmt.Errorf("scan settled attendance: %w", err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("load settled attendances: %w", err)
	}
	return items, nil
}

func settledAdvances(ctx context.Context, q queryer, recapID int64) ([]advRow, error) {
	const query = `
		SELECT id, worker_id, date, amount, COALESCE(note, '')
		FROM cash_advances WHERE recap_id = ? ORDER BY date, id`

	rows, err := q.QueryContext(ctx, query, recapID)
	if err != nil {
		return nil, fmt.Errorf("load settled advances: %w", err)
	}
	defer rows.Close()

	items := make([]advRow, 0)
	for rows.Next() {
		var item advRow
		if err := rows.Scan(&item.ID, &item.WorkerID, &item.Date, &item.Amount, &item.Note); err != nil {
			return nil, fmt.Errorf("scan settled advance: %w", err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("load settled advances: %w", err)
	}
	return items, nil
}
