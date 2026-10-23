package project

import (
	"context"
	"database/sql"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"

	"situkang/internal/clock"
	"situkang/internal/db"
)

const testSchema = `
CREATE TABLE workers (
    id         INTEGER PRIMARY KEY,
    name       TEXT NOT NULL,
    role       TEXT NOT NULL,
    daily_fee  INTEGER NOT NULL,
    active     INTEGER NOT NULL DEFAULT 1,
    created_at TEXT NOT NULL
) STRICT;

CREATE TABLE projects (
    id         INTEGER PRIMARY KEY,
    name       TEXT NOT NULL,
    start_date TEXT,
    end_date   TEXT,
    status     TEXT NOT NULL,
    notes      TEXT,
    created_at TEXT NOT NULL
) STRICT;

CREATE TABLE attendances (
    id                 INTEGER PRIMARY KEY,
    worker_id          INTEGER NOT NULL REFERENCES workers(id),
    date               TEXT NOT NULL,
    quarters           INTEGER NOT NULL,
    daily_fee_snapshot INTEGER NOT NULL,
    recap_id           INTEGER,
    created_at         TEXT NOT NULL
) STRICT;

CREATE TABLE project_workers (
    project_id INTEGER NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    worker_id  INTEGER NOT NULL REFERENCES workers(id),
    PRIMARY KEY (project_id, worker_id)
) STRICT;
`

func testService(t *testing.T) (*Service, *sql.DB) {
	t.Helper()

	sqlDB, err := db.Open(filepath.Join(t.TempDir(), "project.db"))
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}
	t.Cleanup(func() { sqlDB.Close() })

	if _, err := sqlDB.ExecContext(context.Background(), testSchema); err != nil {
		t.Fatalf("create test schema: %v", err)
	}

	return NewService(sqlDB, clock.Fixed(time.Date(2026, 1, 5, 8, 0, 0, 0, time.UTC))), sqlDB
}

func addWorker(t *testing.T, sqlDB *sql.DB, id int64, name string) {
	t.Helper()
	if _, err := sqlDB.ExecContext(context.Background(),
		`INSERT INTO workers (id, name, role, daily_fee, active, created_at) VALUES (?, ?, 'Tukang', 100000, 1, '2026-01-01T00:00:00Z')`,
		id, name); err != nil {
		t.Fatalf("add worker: %v", err)
	}
}

func addAttendance(t *testing.T, sqlDB *sql.DB, id, workerID int64, date string) {
	t.Helper()
	if _, err := sqlDB.ExecContext(context.Background(),
		`INSERT INTO attendances (id, worker_id, date, quarters, daily_fee_snapshot, created_at) VALUES (?, ?, ?, 4, 100000, '2026-01-01T00:00:00Z')`,
		id, workerID, date); err != nil {
		t.Fatalf("add attendance: %v", err)
	}
}

func TestCreateGetUpdate(t *testing.T) {
	ctx := context.Background()
	svc, sqlDB := testService(t)
	addWorker(t, sqlDB, 1, "Budi")
	addWorker(t, sqlDB, 2, "Andi")

	created, err := svc.Create(ctx, Input{
		Name:      "  Rumah Budi  ",
		Status:    "aktif",
		StartDate: "2026-01-05",
		EndDate:   "2026-02-05",
		Notes:     " Renovasi ",
		WorkerIDs: []int64{2, 1, 2},
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if created.Name != "Rumah Budi" || created.Notes != "Renovasi" {
		t.Fatalf("created = %+v, want trimmed values", created)
	}
	if created.StartDate == nil || *created.StartDate != "2026-01-05" {
		t.Fatalf("start = %v", created.StartDate)
	}
	if len(created.WorkerIDs) != 2 || created.WorkerIDs[0] != 1 || created.WorkerIDs[1] != 2 {
		t.Fatalf("worker ids = %v, want deduplicated [1 2]", created.WorkerIDs)
	}

	got, err := svc.Get(ctx, created.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.ID != created.ID || got.Name != created.Name || got.Status != created.Status ||
		got.Notes != created.Notes || got.CreatedAt != created.CreatedAt {
		t.Fatalf("Get = %+v, want %+v", got, created)
	}
	if got.StartDate == nil || *got.StartDate != "2026-01-05" {
		t.Fatalf("Get start = %v", got.StartDate)
	}
	if got.EndDate == nil || *got.EndDate != "2026-02-05" {
		t.Fatalf("Get end = %v", got.EndDate)
	}

	updated, err := svc.Update(ctx, created.ID, Input{
		Name:      "Rumah Budi 2",
		Status:    "selesai",
		WorkerIDs: []int64{2},
	})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if updated.Name != "Rumah Budi 2" || updated.Status != "selesai" {
		t.Fatalf("updated = %+v", updated)
	}
	if updated.StartDate != nil || updated.EndDate != nil {
		t.Fatalf("dates must be cleared, got %+v", updated)
	}
	if len(updated.WorkerIDs) != 1 || updated.WorkerIDs[0] != 2 {
		t.Fatalf("worker ids after update = %v, want [2]", updated.WorkerIDs)
	}
}

func TestWorkerAssignmentReplace(t *testing.T) {
	ctx := context.Background()
	svc, sqlDB := testService(t)
	addWorker(t, sqlDB, 1, "Budi")
	addWorker(t, sqlDB, 2, "Andi")
	addWorker(t, sqlDB, 3, "Cici")

	created, err := svc.Create(ctx, Input{
		Name: "Rumah", Status: "aktif", WorkerIDs: []int64{1, 2},
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	updated, err := svc.Update(ctx, created.ID, Input{
		Name: "Rumah", Status: "aktif", WorkerIDs: []int64{3},
	})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if len(updated.WorkerIDs) != 1 || updated.WorkerIDs[0] != 3 {
		t.Fatalf("worker ids = %v, want [3]", updated.WorkerIDs)
	}

	items, err := svc.List(ctx)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(items) != 1 || len(items[0].WorkerIDs) != 1 || items[0].WorkerIDs[0] != 3 {
		t.Fatalf("list = %+v, want only worker 3", items)
	}
}

func TestInvalidWorkerAssignment(t *testing.T) {
	ctx := context.Background()
	svc, sqlDB := testService(t)
	addWorker(t, sqlDB, 1, "Budi")

	if _, err := svc.Create(ctx, Input{
		Name: "Rumah", Status: "aktif", WorkerIDs: []int64{1, 99},
	}); err != ErrInvalidWorker {
		t.Fatalf("create error = %v, want ErrInvalidWorker", err)
	}

	created, err := svc.Create(ctx, Input{
		Name: "Rumah", Status: "aktif", WorkerIDs: []int64{1},
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if _, err := svc.Update(ctx, created.ID, Input{
		Name: "Rumah", Status: "aktif", WorkerIDs: []int64{99},
	}); err != ErrInvalidWorker {
		t.Fatalf("update error = %v, want ErrInvalidWorker", err)
	}

	got, err := svc.Get(ctx, created.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if len(got.WorkerIDs) != 1 || got.WorkerIDs[0] != 1 {
		t.Fatalf("assignments changed on failed update: %v", got.WorkerIDs)
	}
}

func TestValidation(t *testing.T) {
	ctx := context.Background()
	svc, _ := testService(t)

	cases := []struct {
		name  string
		input Input
		want  error
	}{
		{"empty name", Input{Name: " ", Status: "aktif"}, ErrInvalidName},
		{"long name", Input{Name: strings.Repeat("a", 101), Status: "aktif"}, ErrInvalidName},
		{"bad status", Input{Name: "Rumah", Status: "baru"}, ErrInvalidStatus},
		{"empty status", Input{Name: "Rumah"}, ErrInvalidStatus},
		{"bad start", Input{Name: "Rumah", Status: "aktif", StartDate: "2026-02-31"}, ErrInvalidDate},
		{"bad end", Input{Name: "Rumah", Status: "aktif", EndDate: "kemarin"}, ErrInvalidDate},
		{"reversed range", Input{Name: "Rumah", Status: "aktif", StartDate: "2026-02-05", EndDate: "2026-01-05"}, ErrInvalidRange},
		{"long notes", Input{Name: "Rumah", Status: "aktif", Notes: strings.Repeat("a", 501)}, ErrNotesTooLong},
		{"zero worker", Input{Name: "Rumah", Status: "aktif", WorkerIDs: []int64{0}}, ErrInvalidWorker},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := svc.Create(ctx, tc.input); err != tc.want {
				t.Fatalf("error = %v, want %v", err, tc.want)
			}
		})
	}

	if _, err := svc.Update(ctx, 999, Input{Name: "Rumah", Status: "aktif"}); err != ErrNotFound {
		t.Fatalf("update missing error = %v, want ErrNotFound", err)
	}
}

func TestDeleteRemovesAssignments(t *testing.T) {
	ctx := context.Background()
	svc, sqlDB := testService(t)
	addWorker(t, sqlDB, 1, "Budi")

	created, err := svc.Create(ctx, Input{
		Name: "Rumah", Status: "aktif", WorkerIDs: []int64{1},
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if err := svc.Delete(ctx, created.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if _, err := svc.Get(ctx, created.ID); err != ErrNotFound {
		t.Fatalf("error = %v, want ErrNotFound", err)
	}

	var count int
	if err := sqlDB.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM project_workers WHERE project_id = ?`, created.ID).Scan(&count); err != nil {
		t.Fatalf("count assignments: %v", err)
	}
	if count != 0 {
		t.Fatalf("assignments = %d, want 0 after cascade delete", count)
	}
}

func TestTimeline(t *testing.T) {
	ctx := context.Background()
	svc, sqlDB := testService(t)
	addWorker(t, sqlDB, 1, "Budi")
	addWorker(t, sqlDB, 2, "Andi")

	current, err := svc.Create(ctx, Input{
		Name: "Rumah", Status: "aktif", StartDate: "2026-01-05", EndDate: "2026-01-20",
		WorkerIDs: []int64{1},
	})
	if err != nil {
		t.Fatalf("Create current: %v", err)
	}
	if _, err := svc.Create(ctx, Input{
		Name: "Lama", Status: "selesai", StartDate: "2025-01-05", EndDate: "2025-02-20",
	}); err != nil {
		t.Fatalf("Create old: %v", err)
	}

	addAttendance(t, sqlDB, 1, 1, "2026-01-07")
	addAttendance(t, sqlDB, 2, 1, "2026-01-08")
	addAttendance(t, sqlDB, 3, 2, "2026-01-09")

	result, err := svc.Timeline(ctx, "2026-01")
	if err != nil {
		t.Fatalf("Timeline: %v", err)
	}
	if result.Days != 31 || result.Month != "2026-01" {
		t.Fatalf("result = %+v", result)
	}
	if len(result.Projects) != 1 || result.Projects[0].Name != "Rumah" {
		t.Fatalf("projects = %+v, want only Rumah", result.Projects)
	}
	days := result.Projects[0].TaggedDays
	if len(days) != 2 || days[0] != "2026-01-07" || days[1] != "2026-01-08" {
		t.Fatalf("tagged days = %v, want assigned worker attendance days", days)
	}
	if result.Projects[0].ID != current.ID {
		t.Fatalf("project id = %d, want %d", result.Projects[0].ID, current.ID)
	}

	if _, err := svc.Timeline(ctx, "2026-13"); err != ErrInvalidMonth {
		t.Fatalf("invalid month error = %v, want ErrInvalidMonth", err)
	}
	if _, err := svc.Timeline(ctx, "2026-1"); err != ErrInvalidMonth {
		t.Fatalf("unpadded month error = %v, want ErrInvalidMonth", err)
	}
}

func TestCalendar(t *testing.T) {
	ctx := context.Background()
	svc, sqlDB := testService(t)
	addWorker(t, sqlDB, 1, "Budi")
	addWorker(t, sqlDB, 2, "Andi")

	created, err := svc.Create(ctx, Input{
		Name: "Rumah", Status: "aktif", StartDate: "2026-01-05", EndDate: "2026-01-20",
		WorkerIDs: []int64{1, 2},
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	addAttendance(t, sqlDB, 1, 1, "2026-01-07")
	addAttendance(t, sqlDB, 2, 2, "2026-01-07")
	addAttendance(t, sqlDB, 3, 1, "2026-01-08")

	result, err := svc.Calendar(ctx, created.ID, "2026-01")
	if err != nil {
		t.Fatalf("Calendar: %v", err)
	}
	if result.Month != "2026-01" {
		t.Fatalf("month = %q", result.Month)
	}
	if len(result.WorkerIDs) != 2 || result.WorkerIDs[0] != 1 || result.WorkerIDs[1] != 2 {
		t.Fatalf("worker ids = %v", result.WorkerIDs)
	}
	if len(result.Days) != 2 {
		t.Fatalf("days = %+v, want 2", result.Days)
	}
	if result.Days[0].Date != "2026-01-07" || len(result.Days[0].Workers) != 2 {
		t.Fatalf("first day = %+v", result.Days[0])
	}
	if result.Days[0].Workers[0].Name != "Andi" || result.Days[0].Workers[0].Quarters != 4 {
		t.Fatalf("first worker = %+v", result.Days[0].Workers[0])
	}
	if result.Days[1].Date != "2026-01-08" || len(result.Days[1].Workers) != 1 {
		t.Fatalf("second day = %+v", result.Days[1])
	}

	if _, err := svc.Calendar(ctx, created.ID, "2026-1"); err != ErrInvalidMonth {
		t.Fatalf("invalid month = %v, want ErrInvalidMonth", err)
	}
	if _, err := svc.Calendar(ctx, 99, "2026-01"); err != ErrNotFound {
		t.Fatalf("missing project = %v, want ErrNotFound", err)
	}
}

func newTestRouter(t *testing.T, svc *Service) chi.Router {
	t.Helper()
	h := NewHandler(svc)
	r := chi.NewRouter()
	r.Get("/projects", h.List)
	r.Post("/projects", h.Create)
	r.Get("/projects/timeline", h.Timeline)
	r.Get("/projects/{id}/calendar", h.Calendar)
	r.Get("/projects/{id}", h.Get)
	r.Put("/projects/{id}", h.Update)
	r.Delete("/projects/{id}", h.Delete)
	return r
}

func doJSON(t *testing.T, r http.Handler, method, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	return rec
}

func TestHandler(t *testing.T) {
	svc, sqlDB := testService(t)
	r := newTestRouter(t, svc)
	addWorker(t, sqlDB, 1, "Budi")

	rec := doJSON(t, r, http.MethodPost, "/projects", `{"name":"Rumah","status":"aktif","start_date":"2026-01-05","end_date":"2026-02-05","notes":"","worker_ids":[1]}`)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create status = %d, want 201 (body %s)", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `"worker_ids":[1]`) {
		t.Fatalf("create body = %s, want worker_ids", rec.Body.String())
	}

	rec = doJSON(t, r, http.MethodGet, "/projects", "")
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"worker_ids":[1]`) {
		t.Fatalf("list status = %d body = %s", rec.Code, rec.Body.String())
	}

	rec = doJSON(t, r, http.MethodPost, "/projects", `{"name":"Tanpa Tukang","status":"aktif"}`)
	if rec.Code != http.StatusCreated || !strings.Contains(rec.Body.String(), `"worker_ids":[]`) {
		t.Fatalf("empty worker list status = %d body = %s", rec.Code, rec.Body.String())
	}

	rec = doJSON(t, r, http.MethodPost, "/projects", `{"name":"","status":"aktif"}`)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("invalid name status = %d, want 400", rec.Code)
	}

	rec = doJSON(t, r, http.MethodPost, "/projects", `{"name":"Gudang","status":"aktif","worker_ids":[99]}`)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("invalid worker status = %d, want 400", rec.Code)
	}

	rec = doJSON(t, r, http.MethodGet, "/projects/99", "")
	if rec.Code != http.StatusNotFound {
		t.Fatalf("missing status = %d, want 404", rec.Code)
	}

	rec = doJSON(t, r, http.MethodGet, "/projects/timeline?month=2026-01", "")
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"days":31`) {
		t.Fatalf("timeline status = %d body = %s", rec.Code, rec.Body.String())
	}

	rec = doJSON(t, r, http.MethodGet, "/projects/1/calendar?month=2026-01", "")
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"month":"2026-01"`) {
		t.Fatalf("calendar status = %d body = %s", rec.Code, rec.Body.String())
	}

	rec = doJSON(t, r, http.MethodGet, "/projects/1/calendar?month=2026-1", "")
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("invalid calendar month status = %d, want 400", rec.Code)
	}

	rec = doJSON(t, r, http.MethodDelete, "/projects/1", "")
	if rec.Code != http.StatusNoContent {
		t.Fatalf("delete status = %d, want 204 (body %s)", rec.Code, rec.Body.String())
	}
}
