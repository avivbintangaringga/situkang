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

CREATE TABLE attendance_projects (
    attendance_id INTEGER NOT NULL REFERENCES attendances(id) ON DELETE CASCADE,
    project_id    INTEGER NOT NULL REFERENCES projects(id),
    PRIMARY KEY (attendance_id, project_id)
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

func TestCreateGetUpdate(t *testing.T) {
	ctx := context.Background()
	svc, _ := testService(t)

	created, err := svc.Create(ctx, Input{
		Name:      "  Rumah Budi  ",
		Status:    "aktif",
		StartDate: "2026-01-05",
		EndDate:   "2026-02-05",
		Notes:     " Renovasi ",
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
		Name:   "Rumah Budi 2",
		Status: "selesai",
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

func TestDeleteAndInUse(t *testing.T) {
	ctx := context.Background()
	svc, sqlDB := testService(t)

	created, err := svc.Create(ctx, Input{Name: "Rumah", Status: "aktif"})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if err := svc.Delete(ctx, created.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if _, err := svc.Get(ctx, created.ID); err != ErrNotFound {
		t.Fatalf("error = %v, want ErrNotFound", err)
	}

	used, err := svc.Create(ctx, Input{Name: "Gudang", Status: "aktif"})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if _, err := sqlDB.ExecContext(ctx,
		`INSERT INTO workers (id, name, role, daily_fee, active, created_at) VALUES (1, 'Budi', 'Tukang', 100000, 1, '2026-01-01T00:00:00Z')`); err != nil {
		t.Fatalf("add worker: %v", err)
	}
	if _, err := sqlDB.ExecContext(ctx,
		`INSERT INTO attendances (id, worker_id, date, quarters, daily_fee_snapshot, created_at) VALUES (1, 1, '2026-01-05', 4, 100000, '2026-01-05T00:00:00Z')`); err != nil {
		t.Fatalf("add attendance: %v", err)
	}
	if _, err := sqlDB.ExecContext(ctx,
		`INSERT INTO attendance_projects (attendance_id, project_id) VALUES (1, ?)`, used.ID); err != nil {
		t.Fatalf("tag project: %v", err)
	}

	if err := svc.Delete(ctx, used.ID); err != ErrInUse {
		t.Fatalf("delete used error = %v, want ErrInUse", err)
	}
}

func TestTimeline(t *testing.T) {
	ctx := context.Background()
	svc, sqlDB := testService(t)

	current, err := svc.Create(ctx, Input{
		Name: "Rumah", Status: "aktif", StartDate: "2026-01-05", EndDate: "2026-01-20",
	})
	if err != nil {
		t.Fatalf("Create current: %v", err)
	}
	if _, err := svc.Create(ctx, Input{
		Name: "Lama", Status: "selesai", StartDate: "2025-01-05", EndDate: "2025-02-20",
	}); err != nil {
		t.Fatalf("Create old: %v", err)
	}

	if _, err := sqlDB.ExecContext(ctx,
		`INSERT INTO workers (id, name, role, daily_fee, active, created_at) VALUES (1, 'Budi', 'Tukang', 100000, 1, '2026-01-01T00:00:00Z')`); err != nil {
		t.Fatalf("add worker: %v", err)
	}
	if _, err := sqlDB.ExecContext(ctx,
		`INSERT INTO attendances (id, worker_id, date, quarters, daily_fee_snapshot, created_at) VALUES (1, 1, '2026-01-07', 4, 100000, '2026-01-07T00:00:00Z')`); err != nil {
		t.Fatalf("add attendance: %v", err)
	}
	if _, err := sqlDB.ExecContext(ctx,
		`INSERT INTO attendance_projects (attendance_id, project_id) VALUES (1, ?)`, current.ID); err != nil {
		t.Fatalf("tag project: %v", err)
	}

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
	if len(result.Projects[0].TaggedDays) != 1 || result.Projects[0].TaggedDays[0] != "2026-01-07" {
		t.Fatalf("tagged days = %v", result.Projects[0].TaggedDays)
	}

	if _, err := svc.Timeline(ctx, "2026-13"); err != ErrInvalidMonth {
		t.Fatalf("invalid month error = %v, want ErrInvalidMonth", err)
	}
}

func newTestRouter(t *testing.T, svc *Service) chi.Router {
	t.Helper()
	h := NewHandler(svc)
	r := chi.NewRouter()
	r.Get("/projects", h.List)
	r.Post("/projects", h.Create)
	r.Get("/projects/timeline", h.Timeline)
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

	rec := doJSON(t, r, http.MethodPost, "/projects", `{"name":"Rumah","status":"aktif","start_date":"2026-01-05","end_date":"2026-02-05","notes":""}`)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create status = %d, want 201 (body %s)", rec.Code, rec.Body.String())
	}

	rec = doJSON(t, r, http.MethodPost, "/projects", `{"name":"","status":"aktif"}`)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("invalid name status = %d, want 400", rec.Code)
	}

	rec = doJSON(t, r, http.MethodGet, "/projects/99", "")
	if rec.Code != http.StatusNotFound {
		t.Fatalf("missing status = %d, want 404", rec.Code)
	}

	rec = doJSON(t, r, http.MethodGet, "/projects/timeline?month=2026-01", "")
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"days":31`) {
		t.Fatalf("timeline status = %d body = %s", rec.Code, rec.Body.String())
	}

	if _, err := sqlDB.ExecContext(context.Background(),
		`INSERT INTO workers (id, name, role, daily_fee, active, created_at) VALUES (1, 'Budi', 'Tukang', 100000, 1, '2026-01-01T00:00:00Z')`); err != nil {
		t.Fatalf("add worker: %v", err)
	}
	if _, err := sqlDB.ExecContext(context.Background(),
		`INSERT INTO attendances (id, worker_id, date, quarters, daily_fee_snapshot, created_at) VALUES (1, 1, '2026-01-05', 4, 100000, '2026-01-05T00:00:00Z')`); err != nil {
		t.Fatalf("add attendance: %v", err)
	}
	if _, err := sqlDB.ExecContext(context.Background(),
		`INSERT INTO attendance_projects (attendance_id, project_id) VALUES (1, 1)`); err != nil {
		t.Fatalf("tag project: %v", err)
	}

	rec = doJSON(t, r, http.MethodDelete, "/projects/1", "")
	if rec.Code != http.StatusConflict {
		t.Fatalf("delete used status = %d, want 409", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "masih dipakai") {
		t.Fatalf("body = %s", rec.Body.String())
	}
}
