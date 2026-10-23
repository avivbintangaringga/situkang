package attendance

import (
	"context"
	"database/sql"
	"fmt"
)

// ProjectRef is a compact project tag shown next to attendance.
type ProjectRef struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

// DayRow is one worker on the attendance screen for a single date.
type DayRow struct {
	WorkerID int64        `json:"worker_id"`
	Name     string       `json:"name"`
	Role     string       `json:"role"`
	DailyFee int64        `json:"daily_fee"`
	Quarters int          `json:"quarters"` // 0 means absent
	Snapshot int64        `json:"snapshot"`
	Earned   int64        `json:"earned"`
	Locked   bool         `json:"locked"`
	Projects []ProjectRef `json:"projects"`
}

// WeekEntry is one recorded day for a worker.
type WeekEntry struct {
	ID       int64        `json:"id"`
	Date     string       `json:"date"`
	Quarters int          `json:"quarters"`
	Earned   int64        `json:"earned"`
	Locked   bool         `json:"locked"`
	Projects []ProjectRef `json:"projects"`
}

type record struct {
	ID       int64
	WorkerID int64
	Quarters int
	Snapshot int64
	RecapID  *int64
}

func day(ctx context.Context, db *sql.DB, date string) ([]DayRow, error) {
	const query = `
		SELECT w.id, w.name, w.role, w.daily_fee,
		       a.id, a.quarters, a.daily_fee_snapshot, a.recap_id
		FROM workers w
		LEFT JOIN attendances a ON a.worker_id = w.id AND a.date = ?
		WHERE w.active = 1
		ORDER BY w.name COLLATE NOCASE`

	rows, err := db.QueryContext(ctx, query, date)
	if err != nil {
		return nil, fmt.Errorf("load attendance day: %w", err)
	}
	defer rows.Close()

	result := make([]DayRow, 0)
	for rows.Next() {
		var (
			row      DayRow
			id       sql.NullInt64
			quarters sql.NullInt64
			snapshot sql.NullInt64
			recapID  sql.NullInt64
		)
		if err := rows.Scan(&row.WorkerID, &row.Name, &row.Role, &row.DailyFee,
			&id, &quarters, &snapshot, &recapID); err != nil {
			return nil, fmt.Errorf("scan attendance day: %w", err)
		}
		row.Projects = []ProjectRef{}
		row.Snapshot = row.DailyFee
		if quarters.Valid {
			row.Snapshot = snapshot.Int64
			row.Quarters = int(quarters.Int64)
			row.Earned = earned(snapshot.Int64, row.Quarters)
			row.Locked = recapID.Valid
		}
		result = append(result, row)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("load attendance day: %w", err)
	}
	if err := rows.Close(); err != nil {
		return nil, fmt.Errorf("load attendance day: %w", err)
	}

	tags, err := tagRefsByWorker(ctx, db, date)
	if err != nil {
		return nil, err
	}
	for i := range result {
		if projects, ok := tags[result[i].WorkerID]; ok {
			result[i].Projects = projects
		}
	}
	return result, nil
}

func recordsForDate(ctx context.Context, q queryer, date string) (map[int64]record, error) {
	const query = `SELECT id, worker_id, quarters, daily_fee_snapshot, recap_id FROM attendances WHERE date = ?`

	rows, err := q.QueryContext(ctx, query, date)
	if err != nil {
		return nil, fmt.Errorf("load attendance records: %w", err)
	}
	defer rows.Close()

	records := make(map[int64]record)
	for rows.Next() {
		var (
			rec     record
			recapID sql.NullInt64
		)
		if err := rows.Scan(&rec.ID, &rec.WorkerID, &rec.Quarters, &rec.Snapshot, &recapID); err != nil {
			return nil, fmt.Errorf("scan attendance record: %w", err)
		}
		if recapID.Valid {
			value := recapID.Int64
			rec.RecapID = &value
		}
		records[rec.WorkerID] = rec
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("load attendance records: %w", err)
	}
	return records, nil
}

func insert(ctx context.Context, q execer, workerID int64, date string, quarters int, snapshot int64, now string) (int64, error) {
	const query = `INSERT INTO attendances (worker_id, date, quarters, daily_fee_snapshot, created_at) VALUES (?, ?, ?, ?, ?)`

	res, err := q.ExecContext(ctx, query, workerID, date, quarters, snapshot, now)
	if err != nil {
		return 0, fmt.Errorf("insert attendance: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("insert attendance: %w", err)
	}
	return id, nil
}

func updateQuarters(ctx context.Context, q execer, id int64, quarters int) error {
	if _, err := q.ExecContext(ctx, `UPDATE attendances SET quarters = ? WHERE id = ?`, quarters, id); err != nil {
		return fmt.Errorf("update attendance: %w", err)
	}
	return nil
}

func remove(ctx context.Context, q execer, id int64) error {
	if _, err := q.ExecContext(ctx, `DELETE FROM attendances WHERE id = ?`, id); err != nil {
		return fmt.Errorf("delete attendance: %w", err)
	}
	return nil
}

func replaceTags(ctx context.Context, q execer, attendanceID int64, projectIDs []int64) error {
	if _, err := q.ExecContext(ctx, `DELETE FROM attendance_projects WHERE attendance_id = ?`, attendanceID); err != nil {
		return fmt.Errorf("clear attendance projects: %w", err)
	}
	for _, projectID := range projectIDs {
		if _, err := q.ExecContext(ctx,
			`INSERT INTO attendance_projects (attendance_id, project_id) VALUES (?, ?)`,
			attendanceID, projectID); err != nil {
			return fmt.Errorf("tag attendance project: %w", err)
		}
	}
	return nil
}

func tagIDsByWorker(ctx context.Context, q queryer, date string) (map[int64][]int64, error) {
	const query = `
		SELECT a.worker_id, ap.project_id
		FROM attendances a
		JOIN attendance_projects ap ON ap.attendance_id = a.id
		WHERE a.date = ?
		ORDER BY ap.project_id`

	rows, err := q.QueryContext(ctx, query, date)
	if err != nil {
		return nil, fmt.Errorf("load attendance tags: %w", err)
	}
	defer rows.Close()

	tags := make(map[int64][]int64)
	for rows.Next() {
		var (
			workerID  int64
			projectID int64
		)
		if err := rows.Scan(&workerID, &projectID); err != nil {
			return nil, fmt.Errorf("scan attendance tag: %w", err)
		}
		tags[workerID] = append(tags[workerID], projectID)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("load attendance tags: %w", err)
	}
	return tags, nil
}

func tagRefsByWorker(ctx context.Context, q queryer, date string) (map[int64][]ProjectRef, error) {
	const query = `
		SELECT a.worker_id, p.id, p.name
		FROM attendances a
		JOIN attendance_projects ap ON ap.attendance_id = a.id
		JOIN projects p ON p.id = ap.project_id
		WHERE a.date = ?
		ORDER BY p.name COLLATE NOCASE`

	rows, err := q.QueryContext(ctx, query, date)
	if err != nil {
		return nil, fmt.Errorf("load attendance project tags: %w", err)
	}
	defer rows.Close()

	tags := make(map[int64][]ProjectRef)
	for rows.Next() {
		var (
			workerID int64
			project  ProjectRef
		)
		if err := rows.Scan(&workerID, &project.ID, &project.Name); err != nil {
			return nil, fmt.Errorf("scan attendance project tag: %w", err)
		}
		tags[workerID] = append(tags[workerID], project)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("load attendance project tags: %w", err)
	}
	return tags, nil
}

func weekTagRefs(ctx context.Context, q queryer, workerID int64, from, to string) (map[int64][]ProjectRef, error) {
	const query = `
		SELECT ap.attendance_id, p.id, p.name
		FROM attendance_projects ap
		JOIN projects p ON p.id = ap.project_id
		JOIN attendances a ON a.id = ap.attendance_id
		WHERE a.worker_id = ? AND a.date BETWEEN ? AND ?
		ORDER BY p.name COLLATE NOCASE`

	rows, err := q.QueryContext(ctx, query, workerID, from, to)
	if err != nil {
		return nil, fmt.Errorf("load week project tags: %w", err)
	}
	defer rows.Close()

	tags := make(map[int64][]ProjectRef)
	for rows.Next() {
		var (
			attendanceID int64
			project      ProjectRef
		)
		if err := rows.Scan(&attendanceID, &project.ID, &project.Name); err != nil {
			return nil, fmt.Errorf("scan week project tag: %w", err)
		}
		tags[attendanceID] = append(tags[attendanceID], project)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("load week project tags: %w", err)
	}
	return tags, nil
}

func projectExists(ctx context.Context, q queryer, id int64) (bool, error) {
	var count int
	if err := q.QueryRowContext(ctx, `SELECT COUNT(*) FROM projects WHERE id = ?`, id).Scan(&count); err != nil {
		return false, fmt.Errorf("check project: %w", err)
	}
	return count > 0, nil
}

func workerFee(ctx context.Context, q queryer, workerID int64) (int64, bool, error) {
	const query = `SELECT daily_fee, active FROM workers WHERE id = ?`

	var (
		fee    int64
		active bool
	)
	err := q.QueryRowContext(ctx, query, workerID).Scan(&fee, &active)
	return fee, active, err
}

func week(ctx context.Context, db *sql.DB, workerID int64, from, to string) ([]WeekEntry, error) {
	const query = `
		SELECT id, date, quarters, daily_fee_snapshot, recap_id
		FROM attendances
		WHERE worker_id = ? AND date BETWEEN ? AND ?
		ORDER BY date`

	rows, err := db.QueryContext(ctx, query, workerID, from, to)
	if err != nil {
		return nil, fmt.Errorf("load attendance week: %w", err)
	}
	defer rows.Close()

	entries := make([]WeekEntry, 0)
	for rows.Next() {
		var (
			entry    WeekEntry
			snapshot int64
			recapID  sql.NullInt64
		)
		if err := rows.Scan(&entry.ID, &entry.Date, &entry.Quarters, &snapshot, &recapID); err != nil {
			return nil, fmt.Errorf("scan attendance week: %w", err)
		}
		entry.Projects = []ProjectRef{}
		entry.Earned = earned(snapshot, entry.Quarters)
		entry.Locked = recapID.Valid
		entries = append(entries, entry)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("load attendance week: %w", err)
	}
	if err := rows.Close(); err != nil {
		return nil, fmt.Errorf("load attendance week: %w", err)
	}

	tags, err := weekTagRefs(ctx, db, workerID, from, to)
	if err != nil {
		return nil, err
	}
	for i := range entries {
		if projects, ok := tags[entries[i].ID]; ok {
			entries[i].Projects = projects
		}
	}
	return entries, nil
}

func earned(snapshot int64, quarters int) int64 {
	return snapshot * int64(quarters) / 4
}

type queryer interface {
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

type execer interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
}
