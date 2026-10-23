package situkang

import (
	"context"
	"io/fs"
	"path/filepath"
	"testing"
	"testing/fstest"

	"situkang/internal/db"
)

// TestProjectWorkersMigrationBackfillsAssignments applies the first two
// migrations, seeds per-day attendance tags, then runs the full set so the
// assignment table is populated before attendance_projects is dropped.
func TestProjectWorkersMigrationBackfillsAssignments(t *testing.T) {
	ctx := context.Background()
	sqlDB, err := db.Open(filepath.Join(t.TempDir(), "migration.db"))
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { sqlDB.Close() })

	all, err := Migrations()
	if err != nil {
		t.Fatalf("load migrations: %v", err)
	}

	early := fstest.MapFS{}
	for _, name := range []string{"0001_init.sql", "0002_attendance_fractions.sql"} {
		body, err := fs.ReadFile(all, name)
		if err != nil {
			t.Fatalf("read %s: %v", name, err)
		}
		early[name] = &fstest.MapFile{Data: body}
	}
	if err := db.Migrate(ctx, sqlDB, early); err != nil {
		t.Fatalf("apply early migrations: %v", err)
	}

	seed := []string{
		`INSERT INTO workers (id, name, role, daily_fee, created_at) VALUES (1, 'Budi', 'Tukang', 100000, '2026-01-01T00:00:00Z')`,
		`INSERT INTO projects (id, name, status, created_at) VALUES (1, 'Rumah', 'aktif', '2026-01-01T00:00:00Z')`,
		`INSERT INTO attendances (id, worker_id, date, quarters, daily_fee_snapshot, created_at) VALUES (1, 1, '2026-01-05', 4, 100000, '2026-01-05T00:00:00Z')`,
		`INSERT INTO attendances (id, worker_id, date, quarters, daily_fee_snapshot, created_at) VALUES (2, 1, '2026-01-06', 4, 100000, '2026-01-06T00:00:00Z')`,
		`INSERT INTO attendance_projects (attendance_id, project_id) VALUES (1, 1)`,
		`INSERT INTO attendance_projects (attendance_id, project_id) VALUES (2, 1)`,
	}
	for _, query := range seed {
		if _, err := sqlDB.ExecContext(ctx, query); err != nil {
			t.Fatalf("seed: %v", err)
		}
	}

	if err := db.Migrate(ctx, sqlDB, all); err != nil {
		t.Fatalf("apply full migrations: %v", err)
	}

	var assignments int
	if err := sqlDB.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM project_workers WHERE project_id = 1 AND worker_id = 1`).Scan(&assignments); err != nil {
		t.Fatalf("count assignments: %v", err)
	}
	if assignments != 1 {
		t.Fatalf("assignments = %d, want 1 distinct pair", assignments)
	}

	var legacy int
	if err := sqlDB.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM sqlite_master WHERE type = 'table' AND name = 'attendance_projects'`).Scan(&legacy); err != nil {
		t.Fatalf("check legacy table: %v", err)
	}
	if legacy != 0 {
		t.Fatal("attendance_projects must be dropped")
	}
}
