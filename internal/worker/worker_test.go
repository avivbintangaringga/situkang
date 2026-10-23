package worker

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
    daily_fee  INTEGER NOT NULL CHECK (daily_fee >= 0),
    active     INTEGER NOT NULL DEFAULT 1 CHECK (active IN (0, 1)),
    created_at TEXT NOT NULL
) STRICT;
`

func testService(t *testing.T) (*Service, *sql.DB) {
	t.Helper()

	sqlDB, err := db.Open(filepath.Join(t.TempDir(), "worker.db"))
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}
	t.Cleanup(func() { sqlDB.Close() })

	if _, err := sqlDB.ExecContext(context.Background(), testSchema); err != nil {
		t.Fatalf("create test schema: %v", err)
	}

	return NewService(sqlDB, clock.Fixed(time.Date(2026, 1, 5, 8, 0, 0, 0, time.UTC))), sqlDB
}

func TestCreateAndGet(t *testing.T) {
	ctx := context.Background()
	svc, _ := testService(t)

	created, err := svc.Create(ctx, Input{Name: "  Budi  ", Role: "Tukang Batu", DailyFee: 150_000})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if created.Name != "Budi" {
		t.Fatalf("name = %q, want trimmed %q", created.Name, "Budi")
	}
	if !created.Active {
		t.Fatal("new worker must be active")
	}
	if created.DailyFee != 150_000 {
		t.Fatalf("daily_fee = %d, want 150000", created.DailyFee)
	}

	got, err := svc.Get(ctx, created.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got != created {
		t.Fatalf("Get = %+v, want %+v", got, created)
	}
}

func TestCreateValidation(t *testing.T) {
	ctx := context.Background()
	svc, _ := testService(t)

	cases := []struct {
		name  string
		input Input
		want  error
	}{
		{"empty name", Input{Name: "   ", Role: "Tukang", DailyFee: 1}, ErrInvalidName},
		{"long name", Input{Name: strings.Repeat("a", 101), Role: "Tukang", DailyFee: 1}, ErrInvalidName},
		{"empty role", Input{Name: "Budi", Role: " ", DailyFee: 1}, ErrInvalidRole},
		{"negative fee", Input{Name: "Budi", Role: "Tukang", DailyFee: -1}, ErrInvalidFee},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := svc.Create(ctx, tc.input); err != tc.want {
				t.Fatalf("error = %v, want %v", err, tc.want)
			}
		})
	}
}

func TestZeroFeeIsAllowed(t *testing.T) {
	ctx := context.Background()
	svc, _ := testService(t)

	if _, err := svc.Create(ctx, Input{Name: "Budi", Role: "Kenek", DailyFee: 0}); err != nil {
		t.Fatalf("Create with zero fee: %v", err)
	}
}

func TestUpdateKeepsActiveWhenOmitted(t *testing.T) {
	ctx := context.Background()
	svc, _ := testService(t)

	created, err := svc.Create(ctx, Input{Name: "Budi", Role: "Tukang", DailyFee: 100_000})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	updated, err := svc.Update(ctx, created.ID, Input{Name: "Budi Santoso", Role: "Mandor", DailyFee: 200_000})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if updated.Name != "Budi Santoso" || updated.Role != "Mandor" || updated.DailyFee != 200_000 {
		t.Fatalf("updated = %+v", updated)
	}
	if !updated.Active {
		t.Fatal("nil Active must keep the current state")
	}

	inactive := false
	updated, err = svc.Update(ctx, created.ID, Input{Name: "Budi Santoso", Role: "Mandor", DailyFee: 200_000, Active: &inactive})
	if err != nil {
		t.Fatalf("Update to inactive: %v", err)
	}
	if updated.Active {
		t.Fatal("explicit Active=false must deactivate")
	}
}

func TestUpdateMissingWorker(t *testing.T) {
	ctx := context.Background()
	svc, _ := testService(t)

	if _, err := svc.Update(ctx, 999, Input{Name: "Budi", Role: "Tukang"}); err != ErrNotFound {
		t.Fatalf("error = %v, want ErrNotFound", err)
	}
}

func TestDeleteIsSoft(t *testing.T) {
	ctx := context.Background()
	svc, sqlDB := testService(t)

	created, err := svc.Create(ctx, Input{Name: "Budi", Role: "Tukang", DailyFee: 100_000})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	if err := svc.Delete(ctx, created.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	var count int
	if err := sqlDB.QueryRowContext(ctx, `SELECT COUNT(*) FROM workers WHERE id = ?`, created.ID).Scan(&count); err != nil {
		t.Fatalf("count workers: %v", err)
	}
	if count != 1 {
		t.Fatalf("worker row must survive soft delete, count = %d", count)
	}

	active, err := svc.List(ctx, "", false)
	if err != nil {
		t.Fatalf("List active: %v", err)
	}
	if len(active) != 0 {
		t.Fatalf("active list = %d rows, want 0", len(active))
	}

	all, err := svc.List(ctx, "", true)
	if err != nil {
		t.Fatalf("List all: %v", err)
	}
	if len(all) != 1 || all[0].Active {
		t.Fatalf("all list = %+v, want one inactive worker", all)
	}

	if err := svc.Delete(ctx, 999); err != ErrNotFound {
		t.Fatalf("delete missing error = %v, want ErrNotFound", err)
	}
}

func TestListSearch(t *testing.T) {
	ctx := context.Background()
	svc, _ := testService(t)

	for _, w := range []Input{
		{Name: "Budi", Role: "Tukang Batu", DailyFee: 100_000},
		{Name: "Andi", Role: "Tukang Kayu", DailyFee: 120_000},
		{Name: "100% Ahli", Role: "Kenek", DailyFee: 80_000},
	} {
		if _, err := svc.Create(ctx, w); err != nil {
			t.Fatalf("Create %q: %v", w.Name, err)
		}
	}

	got, err := svc.List(ctx, "kayu", false)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(got) != 1 || got[0].Name != "Andi" {
		t.Fatalf("search kayu = %+v, want Andi", got)
	}

	got, err = svc.List(ctx, "%", false)
	if err != nil {
		t.Fatalf("List literal percent: %v", err)
	}
	if len(got) != 1 || got[0].Name != "100% Ahli" {
		t.Fatalf("search %% = %+v, want the literal percent worker", got)
	}
}

func newTestRouter(t *testing.T, svc *Service) chi.Router {
	t.Helper()
	h := NewHandler(svc)
	r := chi.NewRouter()
	r.Get("/workers", h.List)
	r.Post("/workers", h.Create)
	r.Get("/workers/{id}", h.Get)
	r.Put("/workers/{id}", h.Update)
	r.Delete("/workers/{id}", h.Delete)
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

func TestHandlerCreateAndValidation(t *testing.T) {
	svc, _ := testService(t)
	r := newTestRouter(t, svc)

	rec := doJSON(t, r, http.MethodPost, "/workers", `{"name":"Budi","role":"Tukang Batu","daily_fee":150000}`)
	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201 (body %s)", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `"name":"Budi"`) {
		t.Fatalf("body = %s", rec.Body.String())
	}

	rec = doJSON(t, r, http.MethodPost, "/workers", `{"name":"","role":"Tukang","daily_fee":1}`)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("empty name status = %d, want 400", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "Nama pekerja wajib diisi") {
		t.Fatalf("body = %s", rec.Body.String())
	}

	rec = doJSON(t, r, http.MethodPost, "/workers", `{"name":"Budi","role":"Tukang","daily_fee":-5}`)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("negative fee status = %d, want 400", rec.Code)
	}
}

func TestHandlerNotFoundAndDelete(t *testing.T) {
	svc, _ := testService(t)
	r := newTestRouter(t, svc)

	rec := doJSON(t, r, http.MethodGet, "/workers/99", "")
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", rec.Code)
	}

	rec = doJSON(t, r, http.MethodGet, "/workers/abc", "")
	if rec.Code != http.StatusNotFound {
		t.Fatalf("bad id status = %d, want 404", rec.Code)
	}

	rec = doJSON(t, r, http.MethodPost, "/workers", `{"name":"Budi","role":"Tukang","daily_fee":1000}`)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create status = %d", rec.Code)
	}

	rec = doJSON(t, r, http.MethodDelete, "/workers/1", "")
	if rec.Code != http.StatusNoContent {
		t.Fatalf("delete status = %d, want 204", rec.Code)
	}

	rec = doJSON(t, r, http.MethodGet, "/workers/1", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("get after delete status = %d, want 200", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), `"active":false`) {
		t.Fatalf("body = %s, want an inactive worker", rec.Body.String())
	}

	rec = doJSON(t, r, http.MethodGet, "/workers", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("list status = %d, want 200", rec.Code)
	}
	if rec.Body.String() != "[]\n" {
		t.Fatalf("active list body = %q, want []", rec.Body.String())
	}
}
