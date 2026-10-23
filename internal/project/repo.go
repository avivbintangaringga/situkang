package project

import (
	"context"
	"database/sql"
	"fmt"
)

// Project is a construction project. Attendance rows are tagged with projects;
// projects never change the amount a worker earns.
type Project struct {
	ID        int64   `json:"id"`
	Name      string  `json:"name"`
	StartDate *string `json:"start_date"`
	EndDate   *string `json:"end_date"`
	Status    string  `json:"status"`
	Notes     string  `json:"notes"`
	CreatedAt string  `json:"created_at"`
}

// TimelineProject is one timeline row: the project span plus the days it was
// actually tagged on attendance in the requested month.
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

type projectData struct {
	Name      string
	StartDate *string
	EndDate   *string
	Status    string
	Notes     string
	CreatedAt string
}

const selectColumns = `id, name, start_date, end_date, status, COALESCE(notes, ''), created_at`

func list(ctx context.Context, db *sql.DB) ([]Project, error) {
	rows, err := db.QueryContext(ctx, `SELECT `+selectColumns+` FROM projects ORDER BY name COLLATE NOCASE`)
	if err != nil {
		return nil, fmt.Errorf("list projects: %w", err)
	}
	defer rows.Close()

	projects := make([]Project, 0)
	for rows.Next() {
		item, err := scanProject(rows)
		if err != nil {
			return nil, err
		}
		projects = append(projects, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list projects: %w", err)
	}
	return projects, nil
}

func get(ctx context.Context, db *sql.DB, id int64) (Project, error) {
	row := db.QueryRowContext(ctx, `SELECT `+selectColumns+` FROM projects WHERE id = ?`, id)
	return scanProject(row)
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
	return item, nil
}

func insert(ctx context.Context, db *sql.DB, data projectData) (Project, error) {
	const query = `
		INSERT INTO projects (name, start_date, end_date, status, notes, created_at)
		VALUES (?, ?, ?, ?, ?, ?)`

	res, err := db.ExecContext(ctx, query, data.Name, data.StartDate, data.EndDate,
		data.Status, nullableText(data.Notes), data.CreatedAt)
	if err != nil {
		return Project{}, fmt.Errorf("insert project: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return Project{}, fmt.Errorf("insert project: %w", err)
	}
	return get(ctx, db, id)
}

func update(ctx context.Context, db *sql.DB, id int64, data projectData) (Project, error) {
	const query = `
		UPDATE projects SET name = ?, start_date = ?, end_date = ?, status = ?, notes = ?
		WHERE id = ?`

	res, err := db.ExecContext(ctx, query, data.Name, data.StartDate, data.EndDate,
		data.Status, nullableText(data.Notes), id)
	if err != nil {
		return Project{}, fmt.Errorf("update project: %w", err)
	}
	if affected, err := res.RowsAffected(); err == nil && affected == 0 {
		return Project{}, sql.ErrNoRows
	}
	return get(ctx, db, id)
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

func usageCount(ctx context.Context, db *sql.DB, id int64) (int, error) {
	var count int
	err := db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM attendance_projects WHERE project_id = ?`, id).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count project usage: %w", err)
	}
	return count, nil
}

func timeline(ctx context.Context, db *sql.DB, from, to string) ([]TimelineProject, error) {
	rows, err := db.QueryContext(ctx,
		`SELECT `+selectColumns+` FROM projects ORDER BY name COLLATE NOCASE`)
	if err != nil {
		return nil, fmt.Errorf("load timeline projects: %w", err)
	}
	defer rows.Close()

	projects := make([]TimelineProject, 0)
	index := make(map[int64]int)
	for rows.Next() {
		item, err := scanProject(rows)
		if err != nil {
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
		return nil, fmt.Errorf("load timeline projects: %w", err)
	}
	if len(projects) == 0 {
		return projects, nil
	}

	tagRows, err := db.QueryContext(ctx, `
		SELECT ap.project_id, a.date
		FROM attendance_projects ap
		JOIN attendances a ON a.id = ap.attendance_id
		WHERE a.date BETWEEN ? AND ?
		GROUP BY ap.project_id, a.date
		ORDER BY a.date`, from, to)
	if err != nil {
		return nil, fmt.Errorf("load timeline tags: %w", err)
	}
	defer tagRows.Close()

	for tagRows.Next() {
		var (
			projectID int64
			date      string
		)
		if err := tagRows.Scan(&projectID, &date); err != nil {
			return nil, fmt.Errorf("scan timeline tag: %w", err)
		}
		if position, ok := index[projectID]; ok {
			projects[position].TaggedDays = append(projects[position].TaggedDays, date)
		}
	}
	if err := tagRows.Err(); err != nil {
		return nil, fmt.Errorf("load timeline tags: %w", err)
	}
	return projects, nil
}

func nullableText(value string) any {
	if value == "" {
		return nil
	}
	return value
}
