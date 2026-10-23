package settings

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
CREATE TABLE settings (
    key   TEXT PRIMARY KEY,
    value TEXT NOT NULL
) STRICT;
`

func testService(t *testing.T) (*Service, *sql.DB) {
	t.Helper()

	sqlDB, err := db.Open(filepath.Join(t.TempDir(), "settings.db"))
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}
	t.Cleanup(func() { sqlDB.Close() })

	if _, err := sqlDB.ExecContext(context.Background(), testSchema); err != nil {
		t.Fatalf("create test schema: %v", err)
	}

	return NewService(sqlDB, clock.Fixed(time.Date(2026, 1, 5, 8, 0, 0, 0, time.UTC))), sqlDB
}

func TestGetDefault(t *testing.T) {
	svc, _ := testService(t)

	result, err := svc.Get(context.Background())
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if len(result.WorkingDays) != 6 || result.WorkingDays[0] != 1 || result.WorkingDays[5] != 6 {
		t.Fatalf("working days = %v, want Monday-Saturday", result.WorkingDays)
	}
}

func TestUpdateRoundTrip(t *testing.T) {
	ctx := context.Background()
	svc, sqlDB := testService(t)

	updated, err := svc.Update(ctx, []int{7, 1, 3})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if len(updated.WorkingDays) != 3 || updated.WorkingDays[0] != 1 ||
		updated.WorkingDays[1] != 3 || updated.WorkingDays[2] != 7 {
		t.Fatalf("updated = %v, want sorted", updated.WorkingDays)
	}

	var raw string
	if err := sqlDB.QueryRowContext(ctx,
		`SELECT value FROM settings WHERE key = 'working_days'`).Scan(&raw); err != nil {
		t.Fatalf("read stored value: %v", err)
	}
	if raw != "1,3,7" {
		t.Fatalf("stored = %q, want 1,3,7", raw)
	}

	got, err := svc.Get(ctx)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if len(got.WorkingDays) != 3 || got.WorkingDays[2] != 7 {
		t.Fatalf("Get = %v", got.WorkingDays)
	}
}

func TestUpdateValidation(t *testing.T) {
	ctx := context.Background()
	svc, _ := testService(t)

	cases := []struct {
		name string
		days []int
		want error
	}{
		{"empty", []int{}, ErrEmptyWorkingDays},
		{"nil", nil, ErrEmptyWorkingDays},
		{"zero", []int{0, 1}, ErrInvalidWorkingDays},
		{"eight", []int{8}, ErrInvalidWorkingDays},
		{"duplicate", []int{1, 1}, ErrInvalidWorkingDays},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := svc.Update(ctx, tc.days); err != tc.want {
				t.Fatalf("error = %v, want %v", err, tc.want)
			}
		})
	}
}

func TestHandler(t *testing.T) {
	svc, _ := testService(t)
	h := NewHandler(svc)
	r := chi.NewRouter()
	r.Get("/settings", h.Get)
	r.Put("/settings", h.Update)

	req := httptest.NewRequest(http.MethodGet, "/settings", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"working_days":[1,2,3,4,5,6]`) {
		t.Fatalf("get status = %d body = %s", rec.Code, rec.Body.String())
	}

	req = httptest.NewRequest(http.MethodPut, "/settings", strings.NewReader(`{"working_days":[1,2,3]}`))
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"working_days":[1,2,3]`) {
		t.Fatalf("put status = %d body = %s", rec.Code, rec.Body.String())
	}

	req = httptest.NewRequest(http.MethodPut, "/settings", strings.NewReader(`{"working_days":[]}`))
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest || !strings.Contains(rec.Body.String(), "minimal satu hari") {
		t.Fatalf("empty status = %d body = %s", rec.Code, rec.Body.String())
	}
}
