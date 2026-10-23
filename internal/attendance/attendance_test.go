package attendance

import (
	"context"
	"database/sql"
	"encoding/json"
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

CREATE TABLE recaps (
    id         INTEGER PRIMARY KEY,
    mode       TEXT NOT NULL,
    status     TEXT NOT NULL,
    created_at TEXT NOT NULL
) STRICT;

CREATE TABLE attendances (
    id                 INTEGER PRIMARY KEY,
    worker_id          INTEGER NOT NULL REFERENCES workers(id),
    date               TEXT NOT NULL,
    quarters           INTEGER NOT NULL,
    daily_fee_snapshot INTEGER NOT NULL,
    recap_id           INTEGER REFERENCES recaps(id),
    created_at         TEXT NOT NULL,
    UNIQUE (worker_id, date)
) STRICT;

CREATE TABLE projects (
    id    INTEGER PRIMARY KEY,
    name  TEXT NOT NULL
) STRICT;

CREATE TABLE project_workers (
    project_id INTEGER NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    worker_id  INTEGER NOT NULL REFERENCES workers(id),
    PRIMARY KEY (project_id, worker_id)
) STRICT;
`

var testDay = time.Date(2026, 1, 5, 9, 0, 0, 0, time.UTC) // Monday

func testService(t *testing.T) (*Service, *sql.DB) {
	t.Helper()

	sqlDB, err := db.Open(filepath.Join(t.TempDir(), "attendance.db"))
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}
	t.Cleanup(func() { sqlDB.Close() })

	if _, err := sqlDB.ExecContext(context.Background(), testSchema); err != nil {
		t.Fatalf("create test schema: %v", err)
	}

	return NewService(sqlDB, clock.Fixed(testDay)), sqlDB
}

func addWorker(t *testing.T, sqlDB *sql.DB, id int64, name string, fee int64) {
	t.Helper()
	_, err := sqlDB.ExecContext(context.Background(),
		`INSERT INTO workers (id, name, role, daily_fee, active, created_at) VALUES (?, ?, 'Tukang', ?, 1, '2026-01-01T00:00:00Z')`,
		id, name, fee)
	if err != nil {
		t.Fatalf("add worker: %v", err)
	}
}

func TestSaveDayAndEarnings(t *testing.T) {
	ctx := context.Background()
	svc, sqlDB := testService(t)
	addWorker(t, sqlDB, 1, "Budi", 100_000)
	addWorker(t, sqlDB, 2, "Andi", 100_000)

	result, err := svc.SaveDay(ctx, "2026-01-05", []Item{
		{WorkerID: 1, Quarters: 4},
		{WorkerID: 2, Quarters: 1},
	})
	if err != nil {
		t.Fatalf("SaveDay: %v", err)
	}

	if result.TotalEarned != 125_000 {
		t.Fatalf("total = %d, want 125000", result.TotalEarned)
	}
	if result.Present != 2 {
		t.Fatalf("present = %d, want 2", result.Present)
	}

	byID := map[int64]DayRow{}
	for _, row := range result.Workers {
		byID[row.WorkerID] = row
	}
	if byID[1].Earned != 100_000 || byID[1].Quarters != 4 {
		t.Fatalf("budi = %+v", byID[1])
	}
	if byID[2].Earned != 25_000 {
		t.Fatalf("andi earned = %d, want 25000", byID[2].Earned)
	}
}

func TestDayListsAbsentWorkersWithoutRecords(t *testing.T) {
	svc, sqlDB := testService(t)
	addWorker(t, sqlDB, 1, "Budi", 100_000)

	result, err := svc.Day(context.Background(), "2026-01-05")
	if err != nil {
		t.Fatalf("Day: %v", err)
	}
	if len(result.Workers) != 1 {
		t.Fatalf("workers = %d, want 1", len(result.Workers))
	}
	if result.Workers[0].Quarters != 0 || result.Workers[0].Locked {
		t.Fatalf("row = %+v, want absent and unlocked", result.Workers[0])
	}
}

func TestSnapshotIsFrozen(t *testing.T) {
	ctx := context.Background()
	svc, sqlDB := testService(t)
	addWorker(t, sqlDB, 1, "Budi", 100_000)

	if _, err := svc.SaveDay(ctx, "2026-01-05", []Item{{WorkerID: 1, Quarters: 4}}); err != nil {
		t.Fatalf("SaveDay: %v", err)
	}

	// Raise the fee, then change the recorded quarters.
	if _, err := sqlDB.ExecContext(ctx, `UPDATE workers SET daily_fee = 200_000 WHERE id = 1`); err != nil {
		t.Fatalf("update fee: %v", err)
	}
	result, err := svc.SaveDay(ctx, "2026-01-05", []Item{{WorkerID: 1, Quarters: 2}})
	if err != nil {
		t.Fatalf("SaveDay again: %v", err)
	}
	if result.Workers[0].Earned != 50_000 {
		t.Fatalf("earned = %d, want 50000 from the frozen snapshot", result.Workers[0].Earned)
	}
}

func TestOmittedWorkerIsRemoved(t *testing.T) {
	ctx := context.Background()
	svc, sqlDB := testService(t)
	addWorker(t, sqlDB, 1, "Budi", 100_000)
	addWorker(t, sqlDB, 2, "Andi", 100_000)

	if _, err := svc.SaveDay(ctx, "2026-01-05", []Item{
		{WorkerID: 1, Quarters: 4},
		{WorkerID: 2, Quarters: 4},
	}); err != nil {
		t.Fatalf("SaveDay: %v", err)
	}

	result, err := svc.SaveDay(ctx, "2026-01-05", []Item{{WorkerID: 1, Quarters: 4}})
	if err != nil {
		t.Fatalf("SaveDay: %v", err)
	}
	for _, row := range result.Workers {
		if row.WorkerID == 2 && row.Quarters != 0 {
			t.Fatalf("omitted worker still present: %+v", row)
		}
	}

	var count int
	if err := sqlDB.QueryRowContext(ctx, `SELECT COUNT(*) FROM attendances`).Scan(&count); err != nil {
		t.Fatalf("count: %v", err)
	}
	if count != 1 {
		t.Fatalf("attendance rows = %d, want 1", count)
	}
}

func TestLockedRecordRejectsChanges(t *testing.T) {
	ctx := context.Background()
	svc, sqlDB := testService(t)
	addWorker(t, sqlDB, 1, "Budi", 100_000)

	if _, err := svc.SaveDay(ctx, "2026-01-05", []Item{{WorkerID: 1, Quarters: 4}}); err != nil {
		t.Fatalf("SaveDay: %v", err)
	}
	if _, err := sqlDB.ExecContext(ctx,
		`INSERT INTO recaps (id, mode, status, created_at) VALUES (1, 'all_unsettled', 'final', '2026-01-05T00:00:00Z')`); err != nil {
		t.Fatalf("insert recap: %v", err)
	}
	if _, err := sqlDB.ExecContext(ctx, `UPDATE attendances SET recap_id = 1`); err != nil {
		t.Fatalf("lock attendance: %v", err)
	}

	if _, err := svc.SaveDay(ctx, "2026-01-05", []Item{{WorkerID: 1, Quarters: 2}}); err != ErrLocked {
		t.Fatalf("error = %v, want ErrLocked", err)
	}

	// Omitting a locked worker must be a no-op, not an error.
	result, err := svc.SaveDay(ctx, "2026-01-05", []Item{})
	if err != nil {
		t.Fatalf("SaveDay with omitted locked worker: %v", err)
	}
	if result.Workers[0].Quarters != 4 || !result.Workers[0].Locked {
		t.Fatalf("locked record must survive: %+v", result.Workers[0])
	}

	// Sending the same quarters is also not a change.
	if _, err := svc.SaveDay(ctx, "2026-01-05", []Item{{WorkerID: 1, Quarters: 4}}); err != nil {
		t.Fatalf("SaveDay unchanged locked: %v", err)
	}
}

func TestSaveDayValidation(t *testing.T) {
	ctx := context.Background()
	svc, sqlDB := testService(t)
	addWorker(t, sqlDB, 1, "Budi", 100_000)

	cases := []struct {
		name  string
		date  string
		items []Item
		want  error
	}{
		{"bad date", "2026-02-31", nil, ErrInvalidDate},
		{"quarters zero", "2026-01-05", []Item{{WorkerID: 1, Quarters: 0}}, ErrInvalidQuarters},
		{"quarters five", "2026-01-05", []Item{{WorkerID: 1, Quarters: 5}}, ErrInvalidQuarters},
		{"duplicate", "2026-01-05", []Item{{WorkerID: 1, Quarters: 1}, {WorkerID: 1, Quarters: 2}}, ErrDuplicateWorker},
		{"unknown worker", "2026-01-05", []Item{{WorkerID: 99, Quarters: 4}}, ErrInvalidWorker},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := svc.SaveDay(ctx, tc.date, tc.items); err != tc.want {
				t.Fatalf("error = %v, want %v", err, tc.want)
			}
		})
	}

	if _, err := sqlDB.ExecContext(ctx, `UPDATE workers SET active = 0 WHERE id = 1`); err != nil {
		t.Fatalf("deactivate: %v", err)
	}
	if _, err := svc.SaveDay(ctx, "2026-01-05", []Item{{WorkerID: 1, Quarters: 4}}); err != ErrInvalidWorker {
		t.Fatalf("inactive worker error = %v, want ErrInvalidWorker", err)
	}
}

func TestWeek(t *testing.T) {
	ctx := context.Background()
	svc, sqlDB := testService(t)
	addWorker(t, sqlDB, 1, "Budi", 100_000)

	for _, date := range []string{"2026-01-05", "2026-01-07"} {
		if _, err := svc.SaveDay(ctx, date, []Item{{WorkerID: 1, Quarters: 4}}); err != nil {
			t.Fatalf("SaveDay %s: %v", date, err)
		}
	}

	entries, err := svc.Week(ctx, 1, "2026-01-05", "2026-01-11")
	if err != nil {
		t.Fatalf("Week: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("entries = %d, want 2", len(entries))
	}
	if entries[0].Date != "2026-01-05" || entries[1].Date != "2026-01-07" {
		t.Fatalf("dates = %v", entries)
	}
	if entries[0].Earned != 100_000 {
		t.Fatalf("earned = %d", entries[0].Earned)
	}

	if _, err := svc.Week(ctx, 1, "2026-01-11", "2026-01-05"); err != ErrInvalidRange {
		t.Fatalf("reversed range error = %v, want ErrInvalidRange", err)
	}
}

func TestHandlerLockedMessage(t *testing.T) {
	ctx := context.Background()
	svc, sqlDB := testService(t)
	addWorker(t, sqlDB, 1, "Budi", 100_000)
	if _, err := svc.SaveDay(ctx, "2026-01-05", []Item{{WorkerID: 1, Quarters: 4}}); err != nil {
		t.Fatalf("SaveDay: %v", err)
	}
	if _, err := sqlDB.ExecContext(ctx,
		`INSERT INTO recaps (id, mode, status, created_at) VALUES (1, 'all_unsettled', 'final', '2026-01-05T00:00:00Z')`); err != nil {
		t.Fatalf("insert recap: %v", err)
	}
	if _, err := sqlDB.ExecContext(ctx, `UPDATE attendances SET recap_id = 1`); err != nil {
		t.Fatalf("lock: %v", err)
	}

	h := NewHandler(svc)
	req := httptest.NewRequest(http.MethodPut, "/attendances/day/2026-01-05",
		strings.NewReader(`{"items":[{"worker_id":1,"quarters":3}]}`))
	req.Header.Set("Content-Type", "application/json")
	router := chi.NewRouter()
	router.Put("/attendances/day/{date}", h.SaveDay)

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusConflict {
		t.Fatalf("status = %d, want 409", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "Data sudah masuk rekap dan terkunci") {
		t.Fatalf("body = %s", rec.Body.String())
	}
}

func addProject(t *testing.T, sqlDB *sql.DB, id int64, name string, workerIDs ...int64) {
	t.Helper()
	if _, err := sqlDB.ExecContext(context.Background(),
		`INSERT INTO projects (id, name) VALUES (?, ?)`, id, name); err != nil {
		t.Fatalf("add project: %v", err)
	}
	for _, workerID := range workerIDs {
		if _, err := sqlDB.ExecContext(context.Background(),
			`INSERT INTO project_workers (project_id, worker_id) VALUES (?, ?)`, id, workerID); err != nil {
			t.Fatalf("assign project %d to worker %d: %v", id, workerID, err)
		}
	}
}

func TestCalendar(t *testing.T) {
	ctx := context.Background()
	svc, sqlDB := testService(t)
	addWorker(t, sqlDB, 1, "Budi", 100_000)
	addWorker(t, sqlDB, 2, "Andi", 120_000)
	addWorker(t, sqlDB, 3, "Citra", 80_000)

	addProject(t, sqlDB, 1, "Renovasi", 1, 2)
	addProject(t, sqlDB, 2, "Gudang", 1)

	if _, err := svc.SaveDay(ctx, "2026-01-05", []Item{
		{WorkerID: 1, Quarters: 4},
		{WorkerID: 2, Quarters: 2},
	}); err != nil {
		t.Fatalf("SaveDay 2026-01-05: %v", err)
	}
	if _, err := svc.SaveDay(ctx, "2026-01-06", []Item{{WorkerID: 3, Quarters: 3}}); err != nil {
		t.Fatalf("SaveDay 2026-01-06: %v", err)
	}

	h := NewHandler(svc)
	router := chi.NewRouter()
	router.Get("/attendances/calendar", h.Calendar)

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/attendances/calendar?month=2026-01", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200: %s", rec.Code, rec.Body.String())
	}

	var result Calendar
	if err := json.Unmarshal(rec.Body.Bytes(), &result); err != nil {
		t.Fatalf("decode calendar: %v", err)
	}

	if result.Month != "2026-01" {
		t.Fatalf("month = %q, want 2026-01", result.Month)
	}
	if result.Totals.Days != 2 {
		t.Fatalf("days = %d, want 2", result.Totals.Days)
	}
	if result.Totals.Workers != 3 {
		t.Fatalf("workers = %d, want 3", result.Totals.Workers)
	}
	if result.Totals.Earned != 220_000 {
		t.Fatalf("earned = %d, want 220000", result.Totals.Earned)
	}
	if len(result.Days) != 2 || result.Days[0].Date != "2026-01-05" || result.Days[1].Date != "2026-01-06" {
		t.Fatalf("days = %+v", result.Days)
	}

	first := result.Days[0]
	if len(first.Workers) != 2 {
		t.Fatalf("workers on 2026-01-05 = %d, want 2", len(first.Workers))
	}
	for _, worker := range first.Workers {
		switch worker.WorkerID {
		case 1:
			if worker.Earned != 100_000 {
				t.Fatalf("budi earned = %d, want 100000", worker.Earned)
			}
			if len(worker.Projects) != 2 ||
				worker.Projects[0].Name != "Gudang" ||
				worker.Projects[1].Name != "Renovasi" {
				t.Fatalf("budi projects = %+v, want Gudang then Renovasi", worker.Projects)
			}
		case 2:
			if worker.Earned != 60_000 {
				t.Fatalf("andi earned = %d, want 60000", worker.Earned)
			}
			if len(worker.Projects) != 1 || worker.Projects[0].Name != "Renovasi" {
				t.Fatalf("andi projects = %+v, want Renovasi", worker.Projects)
			}
		default:
			t.Fatalf("unexpected worker on 2026-01-05: %+v", worker)
		}
	}

	second := result.Days[1]
	if len(second.Workers) != 1 || second.Workers[0].WorkerID != 3 {
		t.Fatalf("workers on 2026-01-06 = %+v", second.Workers)
	}
	if second.Workers[0].Earned != 60_000 {
		t.Fatalf("citra earned = %d, want 60000", second.Workers[0].Earned)
	}
	if second.Workers[0].Projects == nil || len(second.Workers[0].Projects) != 0 {
		t.Fatalf("citra projects = %+v, want empty list", second.Workers[0].Projects)
	}
}

func TestCalendarRangeAcrossMonths(t *testing.T) {
	ctx := context.Background()
	svc, sqlDB := testService(t)
	addWorker(t, sqlDB, 1, "Budi", 100_000)
	addWorker(t, sqlDB, 2, "Andi", 120_000)

	if _, err := svc.SaveDay(ctx, "2026-01-30", []Item{{WorkerID: 1, Quarters: 4}}); err != nil {
		t.Fatalf("SaveDay 2026-01-30: %v", err)
	}
	if _, err := svc.SaveDay(ctx, "2026-01-31", []Item{{WorkerID: 2, Quarters: 2}}); err != nil {
		t.Fatalf("SaveDay 2026-01-31: %v", err)
	}
	if _, err := svc.SaveDay(ctx, "2026-02-02", []Item{{WorkerID: 1, Quarters: 3}}); err != nil {
		t.Fatalf("SaveDay 2026-02-02: %v", err)
	}

	h := NewHandler(svc)
	router := chi.NewRouter()
	router.Get("/attendances/calendar", h.Calendar)

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet,
		"/attendances/calendar?from=2026-01-30&to=2026-02-01", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200: %s", rec.Code, rec.Body.String())
	}

	var result Calendar
	if err := json.Unmarshal(rec.Body.Bytes(), &result); err != nil {
		t.Fatalf("decode calendar: %v", err)
	}
	if result.Month != "" {
		t.Fatalf("month = %q, want empty for an explicit range", result.Month)
	}
	if result.Totals.Days != 2 {
		t.Fatalf("days = %d, want 2", result.Totals.Days)
	}
	if result.Totals.Workers != 2 {
		t.Fatalf("workers = %d, want 2", result.Totals.Workers)
	}
	if result.Totals.Earned != 160_000 {
		t.Fatalf("earned = %d, want 160000", result.Totals.Earned)
	}
	if len(result.Days) != 2 ||
		result.Days[0].Date != "2026-01-30" ||
		result.Days[1].Date != "2026-01-31" {
		t.Fatalf("days = %+v", result.Days)
	}
}

func TestCalendarRangeRejectsReversedDates(t *testing.T) {
	svc, _ := testService(t)
	h := NewHandler(svc)
	router := chi.NewRouter()
	router.Get("/attendances/calendar", h.Calendar)

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet,
		"/attendances/calendar?from=2026-02-01&to=2026-01-01", nil))

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400: %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "Tanggal tidak valid") {
		t.Fatalf("body = %s", rec.Body.String())
	}
}

func TestCalendarRangeCapsAt62Days(t *testing.T) {
	svc, _ := testService(t)
	h := NewHandler(svc)
	router := chi.NewRouter()
	router.Get("/attendances/calendar", h.Calendar)

	allowed := httptest.NewRecorder()
	router.ServeHTTP(allowed, httptest.NewRequest(http.MethodGet,
		"/attendances/calendar?from=2026-01-01&to=2026-03-03", nil))
	if allowed.Code != http.StatusOK {
		t.Fatalf("62-day status = %d, want 200: %s", allowed.Code, allowed.Body.String())
	}

	tooLong := httptest.NewRecorder()
	router.ServeHTTP(tooLong, httptest.NewRequest(http.MethodGet,
		"/attendances/calendar?from=2026-01-01&to=2026-03-04", nil))
	if tooLong.Code != http.StatusBadRequest {
		t.Fatalf("63-day status = %d, want 400: %s", tooLong.Code, tooLong.Body.String())
	}
	if !strings.Contains(tooLong.Body.String(), "maksimal 62 hari") {
		t.Fatalf("body = %s", tooLong.Body.String())
	}
}

func TestCalendarRejectsInvalidMonth(t *testing.T) {
	svc, _ := testService(t)
	h := NewHandler(svc)
	router := chi.NewRouter()
	router.Get("/attendances/calendar", h.Calendar)

	for _, month := range []string{"", "2026-1", "2026-13", "2026-00", "januari"} {
		t.Run(month, func(t *testing.T) {
			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/attendances/calendar?month="+month, nil))
			if rec.Code != http.StatusBadRequest {
				t.Fatalf("status = %d, want 400: %s", rec.Code, rec.Body.String())
			}
			if !strings.Contains(rec.Body.String(), "Bulan tidak valid") {
				t.Fatalf("body = %s", rec.Body.String())
			}
		})
	}
}
