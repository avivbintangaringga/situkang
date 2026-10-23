package cashadvance

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"
	"time"

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

CREATE TABLE cash_advances (
    id              INTEGER PRIMARY KEY,
    worker_id       INTEGER NOT NULL REFERENCES workers(id),
    date            TEXT NOT NULL,
    amount          INTEGER NOT NULL,
    note            TEXT,
    recap_id        INTEGER REFERENCES recaps(id),
    source_recap_id INTEGER REFERENCES recaps(id),
    created_at      TEXT NOT NULL
) STRICT;
`

func testService(t *testing.T) (*Service, *sql.DB) {
	t.Helper()

	sqlDB, err := db.Open(filepath.Join(t.TempDir(), "kasbon.db"))
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}
	t.Cleanup(func() { sqlDB.Close() })

	if _, err := sqlDB.ExecContext(context.Background(), testSchema); err != nil {
		t.Fatalf("create schema: %v", err)
	}
	if _, err := sqlDB.ExecContext(context.Background(),
		`INSERT INTO workers (id, name, role, daily_fee, active, created_at) VALUES (1, 'Budi', 'Tukang', 100000, 1, '2026-01-01T00:00:00Z')`); err != nil {
		t.Fatalf("seed worker: %v", err)
	}

	return NewService(sqlDB, clock.Fixed(time.Date(2026, 1, 5, 8, 0, 0, 0, time.UTC))), sqlDB
}

func TestCreateAndList(t *testing.T) {
	ctx := context.Background()
	svc, _ := testService(t)

	created, err := svc.Create(ctx, Input{WorkerID: 1, Date: "2026-01-03", Amount: 50_000, Note: " Beli rokok "})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if created.Note != "Beli rokok" {
		t.Fatalf("note = %q, want trimmed", created.Note)
	}
	if created.Locked {
		t.Fatal("new kas bon must be unlocked")
	}

	items, err := svc.List(ctx, 1, "", "")
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(items) != 1 || items[0].Amount != 50_000 {
		t.Fatalf("items = %+v", items)
	}

	if items, err := svc.List(ctx, 2, "", ""); err != nil || len(items) != 0 {
		t.Fatalf("other worker list = %+v, err = %v", items, err)
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
		{"zero amount", Input{WorkerID: 1, Date: "2026-01-03", Amount: 0}, ErrInvalidAmount},
		{"negative", Input{WorkerID: 1, Date: "2026-01-03", Amount: -1}, ErrInvalidAmount},
		{"bad date", Input{WorkerID: 1, Date: "2026-02-31", Amount: 1000}, ErrInvalidDate},
		{"missing worker", Input{WorkerID: 99, Date: "2026-01-03", Amount: 1000}, ErrInvalidWorker},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := svc.Create(ctx, tc.input); err != tc.want {
				t.Fatalf("error = %v, want %v", err, tc.want)
			}
		})
	}
}

func TestUpdateAndDelete(t *testing.T) {
	ctx := context.Background()
	svc, _ := testService(t)

	created, err := svc.Create(ctx, Input{WorkerID: 1, Date: "2026-01-03", Amount: 50_000})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	updated, err := svc.Update(ctx, created.ID, UpdateInput{Date: "2026-01-04", Amount: 75_000, Note: "Naik"})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if updated.Amount != 75_000 || updated.Date != "2026-01-04" {
		t.Fatalf("updated = %+v", updated)
	}

	if err := svc.Delete(ctx, created.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if _, err := svc.Get(ctx, created.ID); err != ErrNotFound {
		t.Fatalf("Get after delete = %v, want ErrNotFound", err)
	}
}

func TestLockedCashAdvanceCannotChange(t *testing.T) {
	ctx := context.Background()
	svc, sqlDB := testService(t)

	created, err := svc.Create(ctx, Input{WorkerID: 1, Date: "2026-01-03", Amount: 50_000})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	if _, err := sqlDB.ExecContext(ctx,
		`INSERT INTO recaps (id, mode, status, created_at) VALUES (1, 'all_unsettled', 'final', '2026-01-05T00:00:00Z')`); err != nil {
		t.Fatalf("insert recap: %v", err)
	}
	if _, err := sqlDB.ExecContext(ctx, `UPDATE cash_advances SET recap_id = 1`); err != nil {
		t.Fatalf("lock: %v", err)
	}

	if _, err := svc.Update(ctx, created.ID, UpdateInput{Date: "2026-01-04", Amount: 60_000}); err != ErrLocked {
		t.Fatalf("update error = %v, want ErrLocked", err)
	}
	if err := svc.Delete(ctx, created.ID); err != ErrLocked {
		t.Fatalf("delete error = %v, want ErrLocked", err)
	}
}
