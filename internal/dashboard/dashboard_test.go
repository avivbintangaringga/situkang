package dashboard

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
    id     INTEGER PRIMARY KEY,
    status TEXT NOT NULL
) STRICT;

CREATE TABLE attendances (
    id                 INTEGER PRIMARY KEY,
    worker_id          INTEGER NOT NULL REFERENCES workers(id),
    date               TEXT NOT NULL,
    quarters           INTEGER NOT NULL,
    daily_fee_snapshot INTEGER NOT NULL,
    recap_id           INTEGER REFERENCES recaps(id),
    created_at         TEXT NOT NULL
) STRICT;

CREATE TABLE cash_advances (
    id         INTEGER PRIMARY KEY,
    worker_id  INTEGER NOT NULL REFERENCES workers(id),
    date       TEXT NOT NULL,
    amount     INTEGER NOT NULL,
    recap_id   INTEGER REFERENCES recaps(id),
    created_at TEXT NOT NULL
) STRICT;
`

func testService(t *testing.T) (*Service, *sql.DB) {
	t.Helper()

	sqlDB, err := db.Open(filepath.Join(t.TempDir(), "dashboard.db"))
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}
	t.Cleanup(func() { sqlDB.Close() })

	if _, err := sqlDB.ExecContext(context.Background(), testSchema); err != nil {
		t.Fatalf("create test schema: %v", err)
	}

	return NewService(sqlDB, clock.Fixed(time.Date(2026, 1, 5, 8, 0, 0, 0, time.UTC))), sqlDB
}

func addWorker(t *testing.T, sqlDB *sql.DB, id int64, name string, active int) {
	t.Helper()
	_, err := sqlDB.ExecContext(context.Background(),
		`INSERT INTO workers (id, name, role, daily_fee, active, created_at) VALUES (?, ?, 'Tukang', 100000, ?, '2026-01-01T00:00:00Z')`,
		id, name, active)
	if err != nil {
		t.Fatalf("add worker: %v", err)
	}
}

func TestAggregation(t *testing.T) {
	ctx := context.Background()
	svc, sqlDB := testService(t)

	addWorker(t, sqlDB, 1, "Budi", 1)
	addWorker(t, sqlDB, 2, "Andi", 1)
	addWorker(t, sqlDB, 3, "Selesai", 1)

	if _, err := sqlDB.ExecContext(ctx,
		`INSERT INTO attendances (id, worker_id, date, quarters, daily_fee_snapshot, recap_id, created_at)
		 VALUES (1, 1, '2026-01-05', 4, 100000, NULL, '2026-01-05T00:00:00Z')`); err != nil {
		t.Fatalf("add attendance: %v", err)
	}
	if _, err := sqlDB.ExecContext(ctx,
		`INSERT INTO attendances (id, worker_id, date, quarters, daily_fee_snapshot, recap_id, created_at)
		 VALUES (2, 1, '2026-01-06', 2, 120000, NULL, '2026-01-06T00:00:00Z')`); err != nil {
		t.Fatalf("add attendance: %v", err)
	}
	if _, err := sqlDB.ExecContext(ctx,
		`INSERT INTO cash_advances (id, worker_id, date, amount, recap_id, created_at)
		 VALUES (1, 1, '2026-01-06', 50000, NULL, '2026-01-06T00:00:00Z')`); err != nil {
		t.Fatalf("add advance: %v", err)
	}

	if _, err := sqlDB.ExecContext(ctx,
		`INSERT INTO recaps (id, status) VALUES (1, 'final')`); err != nil {
		t.Fatalf("add recap: %v", err)
	}
	if _, err := sqlDB.ExecContext(ctx,
		`INSERT INTO attendances (id, worker_id, date, quarters, daily_fee_snapshot, recap_id, created_at)
		 VALUES (3, 3, '2026-01-05', 4, 100000, 1, '2026-01-05T00:00:00Z')`); err != nil {
		t.Fatalf("add settled attendance: %v", err)
	}

	if _, err := sqlDB.ExecContext(ctx,
		`INSERT INTO recaps (id, status) VALUES (2, 'draft')`); err != nil {
		t.Fatalf("add draft: %v", err)
	}

	result, err := svc.Get(ctx)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}

	if len(result.Workers) != 1 {
		t.Fatalf("workers = %+v, want only Budi", result.Workers)
	}
	line := result.Workers[0]
	if line.WorkerID != 1 || line.Name != "Budi" {
		t.Fatalf("line = %+v", line)
	}
	if line.Quarters != 6 {
		t.Fatalf("quarters = %d, want 6", line.Quarters)
	}
	if line.Gross != 160_000 {
		t.Fatalf("gross = %d, want 160000", line.Gross)
	}
	if line.Advances != 50_000 {
		t.Fatalf("advances = %d, want 50000", line.Advances)
	}
	if line.Net != 110_000 {
		t.Fatalf("net = %d, want 110000", line.Net)
	}
	if result.Totals.Quarters != 6 || result.Totals.Gross != 160_000 ||
		result.Totals.Advances != 50_000 || result.Totals.Net != 110_000 {
		t.Fatalf("totals = %+v", result.Totals)
	}
	if result.DraftRecapID == nil || *result.DraftRecapID != 2 {
		t.Fatalf("draft recap = %v, want 2", result.DraftRecapID)
	}
}

func TestInactiveWorkerWithUnsettledItemsIsIncluded(t *testing.T) {
	ctx := context.Background()
	svc, sqlDB := testService(t)

	addWorker(t, sqlDB, 1, "Budi", 0)
	if _, err := sqlDB.ExecContext(ctx,
		`INSERT INTO attendances (id, worker_id, date, quarters, daily_fee_snapshot, recap_id, created_at)
		 VALUES (1, 1, '2026-01-05', 4, 100000, NULL, '2026-01-05T00:00:00Z')`); err != nil {
		t.Fatalf("add attendance: %v", err)
	}

	result, err := svc.Get(ctx)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if len(result.Workers) != 1 || result.Workers[0].WorkerID != 1 {
		t.Fatalf("workers = %+v, want the inactive worker", result.Workers)
	}
	if result.DraftRecapID != nil {
		t.Fatalf("draft recap = %v, want nil", result.DraftRecapID)
	}
}

func TestEmptyDashboard(t *testing.T) {
	svc, _ := testService(t)

	result, err := svc.Get(context.Background())
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if len(result.Workers) != 0 || result.Totals.Net != 0 {
		t.Fatalf("result = %+v, want empty", result)
	}
}
