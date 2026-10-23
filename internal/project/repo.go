package project

import (
	"context"
	"database/sql"
	"fmt"
)

// Project is a construction project with an assigned set of workers. Projects
// never change the amount a worker earns.
type Project struct {
	ID        int64   `json:"id"`
	Name      string  `json:"name"`
	StartDate *string `json:"start_date"`
	EndDate   *string `json:"end_date"`
	Status    string  `json:"status"`
	Notes     string  `json:"notes"`
	CreatedAt string  `json:"created_at"`
	WorkerIDs []int64 `json:"worker_ids"`
}

// TimelineProject is one timeline row: the project span plus the days its
// assigned workers attended in the requested month.
type TimelineProject struct {
	ID         int64    `json:"id"`
	Name       string   `json:"name"`
	Status     string   `json:"status"`
	StartDate  *string  `json:"start_date"`
	EndDate    *string  `json:"end_date"`
	TaggedDays []string `json:"tagged_days"`
}

// Timeline is the read-only monthly project view.
type Timeline struct {
	Month    string            `json:"month"`
	Days     int               `json:"days"`
	Projects []TimelineProject `json:"projects"`
}

// CalendarWorker is one assigned worker who attended on a calendar day.
type CalendarWorker struct {
	WorkerID int64  `json:"worker_id"`
	Name     string `json:"name"`
	Quarters int    `json:"quarters"`
}

// CalendarDay is one day with attendance by assigned workers.
type CalendarDay struct {
	Date    string           `json:"date"`
	Workers []CalendarWorker `json:"workers"`
}

// ProjectCalendar is the monthly calendar view for one project.
type ProjectCalendar struct {
	Month     string        `json:"month"`
	Days      []CalendarDay `json:"days"`
	WorkerIDs []int64       `json:"worker_ids"`
}

type projectData struct {
	Name      string
	StartDate *string
	EndDate   *string
	Status    string
	Notes     string
	CreatedAt string
}

// dbtx is the subset of *sql.DB and *sql.Tx used by the repository.
type dbtx interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

const selectColumns = `id, name, start_date, end_date, status, COALESCE(notes, ''), created_at`

func list(ctx context.Context, db *sql.DB) ([]Project, error) {
	rows, err := db.QueryContext(ctx, `SELECT `+selectColumns+` FROM projects ORDER BY name COLLATE NOCASE`)
	if err != nil {
		return nil, fmt.Errorf("list projects: %w", err)
	}

	projects := make([]Project, 0)
	for rows.Next() {
		item, err := scanProject(rows)
		if err != nil {
			rows.Close()
			return nil, err
		}
		projects = append(projects, item)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return nil, fmt.Errorf("list projects: %w", err)
	}
	if err := rows.Close(); err != nil {
		return nil, fmt.Errorf("list projects: %w", err)
	}

	assignments, err := workerIDsByProject(ctx, db)
	if err != nil {
		return nil, err
	}
	for i := range projects {
		ids := assignments[projects[i].ID]
		if ids == nil {
			ids = []int64{}
		}
		projects[i].WorkerIDs = ids
	}
	return projects, nil
}

func get(ctx context.Context, db *sql.DB, id int64) (Project, error) {
	row := db.QueryRowContext(ctx, `SELECT `+selectColumns+` FROM projects WHERE id = ?`, id)
	item, err := scanProject(row)
	if err != nil {
		return Project{}, err
	}
	workerIDs, err := assignedWorkerIDs(ctx, db, id)
	if err != nil {
		return Project{}, err
	}
	item.WorkerIDs = workerIDs
	return item, nil
}

type scanner interface {
	Scan(dest ...any) error
}

func scanProject(row scanner) (Project, error) {
	var (
		item       Project
		start, end sql.NullString
	)
	if err := row.Scan(&item.ID, &item.Name, &start, &end, &item.Status, &item.Notes, &item.CreatedAt); err != nil {
		return Project{}, err
	}
	if start.Valid {
		item.StartDate = &start.String
	}
	if end.Valid {
		item.EndDate = &end.String
	}
	item.WorkerIDs = []int64{}
	return item, nil
}

func insert(ctx context.Context, q dbtx, data projectData) (int64, error) {
	const query = `
		INSERT INTO projects (name, start_date, end_date, status, notes, created_at)
		VALUES (?, ?, ?, ?, ?, ?)`

	res, err := q.ExecContext(ctx, query, data.Name, data.StartDate, data.EndDate,
		data.Status, nullableText(data.Notes), data.CreatedAt)
	if err != nil {
		return 0, fmt.Errorf("insert project: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("insert project: %w", err)
	}
	return id, nil
}

func update(ctx context.Context, q dbtx, id int64, data projectData) error {
	const query = `
		UPDATE projects SET name = ?, start_date = ?, end_date = ?, status = ?, notes = ?
		WHERE id = ?`

	res, err := q.ExecContext(ctx, query, data.Name, data.StartDate, data.EndDate,
		data.Status, nullableText(data.Notes), id)
	if err != nil {
		return fmt.Errorf("update project: %w", err)
	}
	if affected, err := res.RowsAffected(); err == nil && affected == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func remove(ctx context.Context, db *sql.DB, id int64) error {
	res, err := db.ExecContext(ctx, `DELETE FROM projects WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("delete project: %w", err)
	}
	if affected, err := res.RowsAffected(); err == nil && affected == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func replaceWorkers(ctx context.Context, q dbtx, projectID int64, workerIDs []int64) error {
	if _, err := q.ExecContext(ctx, `DELETE FROM project_workers WHERE project_id = ?`, projectID); err != nil {
		return fmt.Errorf("clear project workers: %w", err)
	}
	for _, workerID := range workerIDs {
		if _, err := q.ExecContext(ctx,
			`INSERT INTO project_workers (project_id, worker_id) VALUES (?, ?)`,
			projectID, workerID); err != nil {
			return fmt.Errorf("assign project worker: %w", err)
		}
	}
	return nil
}

func assignedWorkerIDs(ctx context.Context, q dbtx, projectID int64) ([]int64, error) {
	rows, err := q.QueryContext(ctx,
		`SELECT worker_id FROM project_workers WHERE project_id = ? ORDER BY worker_id`, projectID)
	if err != nil {
		return nil, fmt.Errorf("load project workers: %w", err)
	}
	defer rows.Close()

	ids := make([]int64, 0)
	for rows.Next() {
		var workerID int64
		if err := rows.Scan(&workerID); err != nil {
			return nil, fmt.Errorf("scan project worker: %w", err)
		}
		ids = append(ids, workerID)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("load project workers: %w", err)
	}
	return ids, nil
}

func workerIDsByProject(ctx context.Context, q dbtx) (map[int64][]int64, error) {
	rows, err := q.QueryContext(ctx,
		`SELECT project_id, worker_id FROM project_workers ORDER BY project_id, worker_id`)
	if err != nil {
		return nil, fmt.Errorf("load project workers: %w", err)
	}
	defer rows.Close()

	assignments := make(map[int64][]int64)
	for rows.Next() {
		var (
			projectID int64
			workerID  int64
		)
		if err := rows.Scan(&projectID, &workerID); err != nil {
			return nil, fmt.Errorf("scan project worker: %w", err)
		}
		assignments[projectID] = append(assignments[projectID], workerID)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("load project workers: %w", err)
	}
	return assignments, nil
}

func workersExist(ctx context.Context, q dbtx, workerIDs []int64) (bool, error) {
	for _, workerID := range workerIDs {
		var count int
		if err := q.QueryRowContext(ctx,
			`SELECT COUNT(*) FROM workers WHERE id = ?`, workerID).Scan(&count); err != nil {
			return false, fmt.Errorf("check worker: %w", err)
		}
		if count == 0 {
			return false, nil
		}
	}
	return true, nil
}

func timeline(ctx context.Context, db *sql.DB, from, to string) ([]TimelineProject, error) {
	rows, err := db.QueryContext(ctx,
		`SELECT `+selectColumns+` FROM projects ORDER BY name COLLATE NOCASE`)
	if err != nil {
		return nil, fmt.Errorf("load timeline projects: %w", err)
	}

	projects := make([]TimelineProject, 0)
	index := make(map[int64]int)
	for rows.Next() {
		item, err := scanProject(rows)
		if err != nil {
			rows.Close()
			return nil, err
		}
		index[item.ID] = len(projects)
		projects = append(projects, TimelineProject{
			ID:         item.ID,
			Name:       item.Name,
			Status:     item.Status,
			StartDate:  item.StartDate,
			EndDate:    item.EndDate,
			TaggedDays: []string{},
		})
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return nil, fmt.Errorf("load timeline projects: %w", err)
	}
	if err := rows.Close(); err != nil {
		return nil, fmt.Errorf("load timeline projects: %w", err)
	}
	if len(projects) == 0 {
		return projects, nil
	}

	dayRows, err := db.QueryContext(ctx, `
		SELECT pw.project_id, a.date
		FROM project_workers pw
		JOIN attendances a ON a.worker_id = pw.worker_id
		WHERE a.date BETWEEN ? AND ?
		GROUP BY pw.project_id, a.date
		ORDER BY a.date`, from, to)
	if err != nil {
		return nil, fmt.Errorf("load timeline days: %w", err)
	}
	defer dayRows.Close()

	for dayRows.Next() {
		var (
			projectID int64
			date      string
		)
		if err := dayRows.Scan(&projectID, &date); err != nil {
			return nil, fmt.Errorf("scan timeline day: %w", err)
		}
		if position, ok := index[projectID]; ok {
			projects[position].TaggedDays = append(projects[position].TaggedDays, date)
		}
	}
	if err := dayRows.Err(); err != nil {
		return nil, fmt.Errorf("load timeline days: %w", err)
	}
	return projects, nil
}

func nullableText(value string) any {
	if value == "" {
		return nil
	}
	return value
}

type calendarEntry struct {
	Date     string
	WorkerID int64
	Name     string
	Quarters int
}

func calendarDays(ctx context.Context, db *sql.DB, projectID int64, from, to string) ([]calendarEntry, error) {
	const query = `
		SELECT a.date, a.worker_id, w.name, a.quarters
		FROM attendances a
		JOIN project_workers pw ON pw.worker_id = a.worker_id
		JOIN workers w ON w.id = a.worker_id
		WHERE pw.project_id = ? AND a.date BETWEEN ? AND ?
		ORDER BY a.date, w.name COLLATE NOCASE`

	rows, err := db.QueryContext(ctx, query, projectID, from, to)
	if err != nil {
		return nil, fmt.Errorf("load project calendar: %w", err)
	}
	defer rows.Close()

	entries := make([]calendarEntry, 0)
	for rows.Next() {
		var entry calendarEntry
		if err := rows.Scan(&entry.Date, &entry.WorkerID, &entry.Name, &entry.Quarters); err != nil {
			return nil, fmt.Errorf("scan project calendar: %w", err)
		}
		entries = append(entries, entry)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("load project calendar: %w", err)
	}
	return entries, nil
}
