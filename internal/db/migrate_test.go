package db

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"
	"testing/fstest"
)

func openTestDB(t *testing.T) *sql.DB {
	t.Helper()
	sqlDB, err := Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { sqlDB.Close() })
	return sqlDB
}

func TestMigrateAppliesEveryFileOnce(t *testing.T) {
	ctx := context.Background()
	sqlDB := openTestDB(t)

	fsys := fstest.MapFS{
		"0001_init.sql": &fstest.MapFile{Data: []byte(`
			CREATE TABLE a (id INTEGER PRIMARY KEY, name TEXT NOT NULL) STRICT;
			CREATE TABLE b (id INTEGER PRIMARY KEY) STRICT;
		`)},
		"0002_more.sql": &fstest.MapFile{Data: []byte(`
			CREATE TABLE c (id INTEGER PRIMARY KEY) STRICT;
		`)},
	}

	if err := Migrate(ctx, sqlDB, fsys); err != nil {
		t.Fatalf("first Migrate: %v", err)
	}
	if err := Migrate(ctx, sqlDB, fsys); err != nil {
		t.Fatalf("second Migrate must be idempotent: %v", err)
	}

	var versions int
	if err := sqlDB.QueryRowContext(ctx, `SELECT COUNT(*) FROM schema_migrations`).Scan(&versions); err != nil {
		t.Fatalf("count schema_migrations: %v", err)
	}
	if versions != 2 {
		t.Fatalf("schema_migrations = %d, want 2", versions)
	}
}

func TestMigrateRollsBackFailedFile(t *testing.T) {
	ctx := context.Background()
	sqlDB := openTestDB(t)

	fsys := fstest.MapFS{
		"0001_broken.sql": &fstest.MapFile{Data: []byte(`
			CREATE TABLE ok (id INTEGER PRIMARY KEY) STRICT;
			CREATE TABLE ok (id INTEGER PRIMARY KEY) STRICT;
		`)},
	}

	if err := Migrate(ctx, sqlDB, fsys); err == nil {
		t.Fatal("Migrate must fail on a broken file")
	}

	var versions int
	if err := sqlDB.QueryRowContext(ctx, `SELECT COUNT(*) FROM schema_migrations`).Scan(&versions); err != nil {
		t.Fatalf("count schema_migrations: %v", err)
	}
	if versions != 0 {
		t.Fatalf("failed migration must not be recorded, got %d", versions)
	}

	var tables int
	err := sqlDB.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM sqlite_master WHERE type = 'table' AND name = 'ok'`,
	).Scan(&tables)
	if err != nil {
		t.Fatalf("check table ok: %v", err)
	}
	if tables != 0 {
		t.Fatal("tables from a failed migration must be rolled back")
	}
}
