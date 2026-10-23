package recap

import (
	"context"
	"database/sql"
	"path/filepath"
	"strings"
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
    id           INTEGER PRIMARY KEY,
    mode         TEXT NOT NULL,
    start_date   TEXT,
    end_date     TEXT,
    status       TEXT NOT NULL,
    created_at   TEXT NOT NULL,
    finalized_at TEXT
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

CREATE TABLE recap_lines (
    id            INTEGER PRIMARY KEY,
    recap_id      INTEGER NOT NULL REFERENCES recaps(id),
    worker_id     INTEGER NOT NULL REFERENCES workers(id),
    worker_name   TEXT NOT NULL,
    worker_role   TEXT NOT NULL,
    days_quarters INTEGER NOT NULL,
    gross         INTEGER NOT NULL,
    advances      INTEGER NOT NULL,
    extras        INTEGER NOT NULL,
    net           INTEGER NOT NULL
) STRICT;

CREATE TABLE recap_paid (
    recap_id  INTEGER NOT NULL REFERENCES recaps(id) ON DELETE CASCADE,
    worker_id INTEGER NOT NULL REFERENCES workers(id),
    paid      INTEGER NOT NULL DEFAULT 1,
    PRIMARY KEY (recap_id, worker_id)
) STRICT;

CREATE TABLE extra_fees (
    id          INTEGER PRIMARY KEY,
    worker_id   INTEGER NOT NULL REFERENCES workers(id),
    recap_id    INTEGER REFERENCES recaps(id) ON DELETE CASCADE,
    amount      INTEGER NOT NULL,
    description TEXT NOT NULL,
    created_at  TEXT NOT NULL
) STRICT;

CREATE TABLE recap_shares (
    recap_id   INTEGER PRIMARY KEY REFERENCES recaps(id) ON DELETE CASCADE,
    token      TEXT NOT NULL UNIQUE,
    created_at TEXT NOT NULL
) STRICT;
`

// 2026-01-12 is a Monday.
var monday = time.Date(2026, 1, 12, 9, 0, 0, 0, time.UTC)

func testService(t *testing.T) (*Service, *sql.DB) {
	t.Helper()

	sqlDB, err := db.Open(filepath.Join(t.TempDir(), "recap.db"))
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}
	t.Cleanup(func() { sqlDB.Close() })

	if _, err := sqlDB.ExecContext(context.Background(), testSchema); err != nil {
		t.Fatalf("create schema: %v", err)
	}

	return NewService(sqlDB, clock.Fixed(monday)), sqlDB
}

func seedWorker(t *testing.T, sqlDB *sql.DB, id int64, name string, fee int64) {
	t.Helper()
	_, err := sqlDB.ExecContext(context.Background(),
		`INSERT INTO workers (id, name, role, daily_fee, active, created_at) VALUES (?, ?, 'Tukang', ?, 1, '2026-01-01T00:00:00Z')`,
		id, name, fee)
	if err != nil {
		t.Fatalf("seed worker: %v", err)
	}
}

func seedAttendance(t *testing.T, sqlDB *sql.DB, workerID int64, date string, quarters int, snapshot int64) {
	t.Helper()
	_, err := sqlDB.ExecContext(context.Background(),
		`INSERT INTO attendances (worker_id, date, quarters, daily_fee_snapshot, created_at) VALUES (?, ?, ?, ?, '2026-01-01T00:00:00Z')`,
		workerID, date, quarters, snapshot)
	if err != nil {
		t.Fatalf("seed attendance: %v", err)
	}
}

func seedAdvance(t *testing.T, sqlDB *sql.DB, workerID int64, date string, amount int64) {
	t.Helper()
	_, err := sqlDB.ExecContext(context.Background(),
		`INSERT INTO cash_advances (worker_id, date, amount, note, created_at) VALUES (?, ?, ?, 'bon', '2026-01-01T00:00:00Z')`,
		workerID, date, amount)
	if err != nil {
		t.Fatalf("seed advance: %v", err)
	}
}

func TestCreateModesAndWarnings(t *testing.T) {
	ctx := context.Background()
	svc, sqlDB := testService(t)
	seedWorker(t, sqlDB, 1, "Budi", 100_000)
	seedAttendance(t, sqlDB, 1, "2026-01-05", 4, 100_000)
	seedAttendance(t, sqlDB, 1, "2026-01-08", 4, 100_000)
	seedAttendance(t, sqlDB, 1, "2026-01-12", 4, 100_000)

	recap, err := svc.Create(ctx, CreateInput{Mode: "since_monday"})
	if err != nil {
		t.Fatalf("Create since_monday: %v", err)
	}
	if recap.StartDate == nil || *recap.StartDate != "2026-01-12" {
		t.Fatalf("start = %v, want 2026-01-12", recap.StartDate)
	}

	detail, err := svc.Detail(ctx, recap.ID)
	if err != nil {
		t.Fatalf("Detail: %v", err)
	}
	if len(detail.Workers) != 1 {
		t.Fatalf("workers = %d, want 1", len(detail.Workers))
	}
	if got := len(detail.Workers[0].Attendance); got != 1 {
		t.Fatalf("attendance = %d, want 1", got)
	}
	if detail.Workers[0].Warning != 2 {
		t.Fatalf("warning = %d, want 2", detail.Workers[0].Warning)
	}

	if _, err := svc.Create(ctx, CreateInput{Mode: "all_unsettled"}); err != ErrDraftExists {
		t.Fatalf("second draft error = %v, want ErrDraftExists", err)
	}

	if err := svc.DeleteDraft(ctx, recap.ID); err != nil {
		t.Fatalf("DeleteDraft: %v", err)
	}

	all, err := svc.Create(ctx, CreateInput{Mode: "all_unsettled"})
	if err != nil {
		t.Fatalf("Create all_unsettled: %v", err)
	}
	allDetail, err := svc.Detail(ctx, all.ID)
	if err != nil {
		t.Fatalf("Detail all: %v", err)
	}
	if got := len(allDetail.Workers[0].Attendance); got != 3 {
		t.Fatalf("all_unsettled attendance = %d, want 3", got)
	}
	if allDetail.Workers[0].Warning != 0 || all.StartDate != nil {
		t.Fatalf("all_unsettled must not warn: %+v", allDetail.Workers[0])
	}

	if err := svc.DeleteDraft(ctx, all.ID); err != nil {
		t.Fatalf("DeleteDraft: %v", err)
	}

	start, end := "2026-01-05", "2026-01-08"
	custom, err := svc.Create(ctx, CreateInput{Mode: "custom", StartDate: &start, EndDate: &end})
	if err != nil {
		t.Fatalf("Create custom: %v", err)
	}
	customDetail, err := svc.Detail(ctx, custom.ID)
	if err != nil {
		t.Fatalf("Detail custom: %v", err)
	}
	if got := len(customDetail.Workers[0].Attendance); got != 2 {
		t.Fatalf("custom attendance = %d, want 2", got)
	}
}

func TestCreateValidation(t *testing.T) {
	ctx := context.Background()
	svc, _ := testService(t)

	if _, err := svc.Create(ctx, CreateInput{Mode: "bogus"}); err != ErrInvalidMode {
		t.Fatalf("mode error = %v, want ErrInvalidMode", err)
	}
	if _, err := svc.Create(ctx, CreateInput{Mode: "custom"}); err != ErrInvalidRange {
		t.Fatalf("custom without dates = %v, want ErrInvalidRange", err)
	}
	start, end := "2026-01-10", "2026-01-05"
	if _, err := svc.Create(ctx, CreateInput{Mode: "custom", StartDate: &start, EndDate: &end}); err != ErrInvalidRange {
		t.Fatalf("reversed range = %v, want ErrInvalidRange", err)
	}
}

func TestDraftCalculation(t *testing.T) {
	ctx := context.Background()
	svc, sqlDB := testService(t)
	seedWorker(t, sqlDB, 1, "Budi", 100_000)
	seedAttendance(t, sqlDB, 1, "2026-01-05", 4, 100_000)
	seedAttendance(t, sqlDB, 1, "2026-01-06", 2, 100_000)
	seedAttendance(t, sqlDB, 1, "2026-01-07", 1, 100_000)
	seedAdvance(t, sqlDB, 1, "2026-01-08", 50_000)

	recap, err := svc.Create(ctx, CreateInput{Mode: "all_unsettled"})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if _, err := svc.AddFee(ctx, recap.ID, 1, 10_000, "Lembur"); err != nil {
		t.Fatalf("AddFee bonus: %v", err)
	}
	if _, err := svc.AddFee(ctx, recap.ID, 1, -3_000, "Denda"); err != nil {
		t.Fatalf("AddFee deduction: %v", err)
	}

	detail, err := svc.Detail(ctx, recap.ID)
	if err != nil {
		t.Fatalf("Detail: %v", err)
	}
	line := detail.Workers[0]
	if line.Quarters != 7 {
		t.Fatalf("quarters = %d, want 7", line.Quarters)
	}
	if line.Gross != 175_000 {
		t.Fatalf("gross = %d, want 175000", line.Gross)
	}
	if line.Advances != 50_000 {
		t.Fatalf("advances = %d, want 50000", line.Advances)
	}
	if line.Extras != 7_000 {
		t.Fatalf("extras = %d, want 7000", line.Extras)
	}
	if line.Net != 132_000 {
		t.Fatalf("net = %d, want 132000", line.Net)
	}
	if !line.Paid {
		t.Fatal("paid must default to true")
	}
	if detail.Totals.Net != 132_000 {
		t.Fatalf("totals net = %d", detail.Totals.Net)
	}
	if detail.PeriodStart != "2026-01-05" || detail.PeriodEnd != "2026-01-08" {
		t.Fatalf("period = %s..%s", detail.PeriodStart, detail.PeriodEnd)
	}
}

func TestFinalizeSettlesPaidAndDetachesUnpaid(t *testing.T) {
	ctx := context.Background()
	svc, sqlDB := testService(t)
	seedWorker(t, sqlDB, 1, "Budi", 100_000)
	seedWorker(t, sqlDB, 2, "Andi", 100_000)
	seedAttendance(t, sqlDB, 1, "2026-01-05", 4, 100_000)
	seedAttendance(t, sqlDB, 2, "2026-01-05", 4, 100_000)
	seedAdvance(t, sqlDB, 1, "2026-01-06", 150_000)

	recap, err := svc.Create(ctx, CreateInput{Mode: "all_unsettled"})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if err := svc.SetPaid(ctx, recap.ID, 2, false); err != nil {
		t.Fatalf("SetPaid: %v", err)
	}
	if _, err := svc.AddFee(ctx, recap.ID, 2, 5_000, "Bonus"); err != nil {
		t.Fatalf("AddFee: %v", err)
	}

	if err := svc.Finalize(ctx, recap.ID); err != nil {
		t.Fatalf("Finalize: %v", err)
	}

	final, err := svc.Detail(ctx, recap.ID)
	if err != nil {
		t.Fatalf("Detail final: %v", err)
	}
	if final.Recap.Status != "final" {
		t.Fatalf("status = %s, want final", final.Recap.Status)
	}

	// Budi is paid: negative net carries forward as an unsettled kas bon.
	if len(final.Workers) != 1 {
		t.Fatalf("final lines = %d, want 1 (only paid workers)", len(final.Workers))
	}
	budi := final.Workers[0]
	if budi.Name != "Budi" || budi.Net != -50_000 {
		t.Fatalf("budi line = %+v, want net -50000", budi)
	}
	if budi.Gross != 100_000 || budi.Advances != 150_000 {
		t.Fatalf("budi totals = %+v", budi)
	}

	var autoAmount, sourceRecap int64
	var autoRecap sql.NullInt64
	var note string
	if err := sqlDB.QueryRowContext(ctx,
		`SELECT amount, source_recap_id, recap_id, note FROM cash_advances WHERE source_recap_id = ?`,
		recap.ID).Scan(&autoAmount, &sourceRecap, &autoRecap, &note); err != nil {
		t.Fatalf("load carried balance: %v", err)
	}
	if autoAmount != 50_000 {
		t.Fatalf("carried amount = %d, want 50000", autoAmount)
	}
	if sourceRecap != recap.ID || autoRecap.Valid {
		t.Fatalf("carried advance must be unsettled and linked to recap %d", recap.ID)
	}
	if !strings.Contains(note, "Saldo sebelumnya (rekap #") {
		t.Fatalf("note = %q", note)
	}

	// Andi is unpaid: his attendance stays unsettled and his fee is detached.
	var andiRecap sql.NullInt64
	if err := sqlDB.QueryRowContext(ctx,
		`SELECT recap_id FROM attendances WHERE worker_id = 2`).Scan(&andiRecap); err != nil {
		t.Fatalf("load andi attendance: %v", err)
	}
	if andiRecap.Valid {
		t.Fatal("unpaid worker's attendance must stay unsettled")
	}

	var feeRecap sql.NullInt64
	if err := sqlDB.QueryRowContext(ctx,
		`SELECT recap_id FROM extra_fees WHERE worker_id = 2`).Scan(&feeRecap); err != nil {
		t.Fatalf("load fee: %v", err)
	}
	if feeRecap.Valid {
		t.Fatal("unpaid worker's extra fee must be detached")
	}

	// Final recap keeps a frozen snapshot that survives item settlement.
	if final.Totals.Net != -50_000 || len(final.Workers[0].Attendance) != 1 {
		t.Fatalf("final detail = %+v", final)
	}
}

func TestSetPaidPersistsInDetail(t *testing.T) {
	ctx := context.Background()
	svc, sqlDB := testService(t)
	seedWorker(t, sqlDB, 1, "Budi", 100_000)
	seedAttendance(t, sqlDB, 1, "2026-01-05", 4, 100_000)

	recap, err := svc.Create(ctx, CreateInput{Mode: "all_unsettled"})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if err := svc.SetPaid(ctx, recap.ID, 1, false); err != nil {
		t.Fatalf("SetPaid: %v", err)
	}

	detail, err := svc.Detail(ctx, recap.ID)
	if err != nil {
		t.Fatalf("Detail: %v", err)
	}
	if detail.Workers[0].Paid {
		t.Fatal("paid flag must persist as false")
	}
}

func TestReopenRestoresDraft(t *testing.T) {
	ctx := context.Background()
	svc, sqlDB := testService(t)
	seedWorker(t, sqlDB, 1, "Budi", 100_000)
	seedAttendance(t, sqlDB, 1, "2026-01-05", 4, 100_000)
	seedAdvance(t, sqlDB, 1, "2026-01-06", 150_000)

	recap, err := svc.Create(ctx, CreateInput{Mode: "all_unsettled"})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if _, err := svc.AddFee(ctx, recap.ID, 1, 5_000, "Bonus"); err != nil {
		t.Fatalf("AddFee: %v", err)
	}
	if err := svc.SetPaid(ctx, recap.ID, 1, true); err != nil {
		t.Fatalf("SetPaid: %v", err)
	}
	if err := svc.Finalize(ctx, recap.ID); err != nil {
		t.Fatalf("Finalize: %v", err)
	}

	if err := svc.Reopen(ctx, recap.ID); err != nil {
		t.Fatalf("Reopen: %v", err)
	}

	reopened, err := svc.Detail(ctx, recap.ID)
	if err != nil {
		t.Fatalf("Detail: %v", err)
	}
	if reopened.Recap.Status != "draft" {
		t.Fatalf("status = %s, want draft", reopened.Recap.Status)
	}
	if len(reopened.Workers[0].Attendance) != 1 {
		t.Fatalf("attendance must return to the draft")
	}
	if len(reopened.Workers[0].ExtraFees) != 1 {
		t.Fatalf("extra fees must stay attached to the reopened draft")
	}
	if !reopened.Workers[0].Paid {
		t.Fatalf("paid toggles must survive reopen")
	}

	var autoCount, lineCount int
	if err := sqlDB.QueryRowContext(ctx, `SELECT COUNT(*) FROM cash_advances WHERE source_recap_id = ?`, recap.ID).Scan(&autoCount); err != nil {
		t.Fatalf("count carried: %v", err)
	}
	if autoCount != 0 {
		t.Fatalf("carried balance must be deleted on reopen, got %d", autoCount)
	}
	if err := sqlDB.QueryRowContext(ctx, `SELECT COUNT(*) FROM recap_lines WHERE recap_id = ?`, recap.ID).Scan(&lineCount); err != nil {
		t.Fatalf("count lines: %v", err)
	}
	if lineCount != 0 {
		t.Fatalf("recap lines must be deleted on reopen, got %d", lineCount)
	}

	if err := svc.Reopen(ctx, recap.ID); err != ErrNotFinal {
		t.Fatalf("reopen draft error = %v, want ErrNotFinal", err)
	}
}

func TestReopenOnlyLatestFinalAndNoDraft(t *testing.T) {
	ctx := context.Background()
	svc, sqlDB := testService(t)
	seedWorker(t, sqlDB, 1, "Budi", 100_000)
	seedAttendance(t, sqlDB, 1, "2026-01-05", 4, 100_000)

	first, err := svc.Create(ctx, CreateInput{Mode: "all_unsettled"})
	if err != nil {
		t.Fatalf("Create first: %v", err)
	}
	if err := svc.Finalize(ctx, first.ID); err != nil {
		t.Fatalf("Finalize first: %v", err)
	}

	second, err := svc.Create(ctx, CreateInput{Mode: "all_unsettled"})
	if err != nil {
		t.Fatalf("Create second: %v", err)
	}
	if err := svc.Finalize(ctx, second.ID); err != nil {
		t.Fatalf("Finalize second: %v", err)
	}

	if err := svc.Reopen(ctx, first.ID); err != ErrNotLatest {
		t.Fatalf("reopen old recap = %v, want ErrNotLatest", err)
	}

	third, err := svc.Create(ctx, CreateInput{Mode: "all_unsettled"})
	if err != nil {
		t.Fatalf("Create third: %v", err)
	}
	if err := svc.Reopen(ctx, second.ID); err != ErrDraftExists {
		t.Fatalf("reopen with draft = %v, want ErrDraftExists", err)
	}
	if err := svc.DeleteDraft(ctx, third.ID); err != nil {
		t.Fatalf("DeleteDraft: %v", err)
	}
	if err := svc.Reopen(ctx, second.ID); err != nil {
		t.Fatalf("Reopen second: %v", err)
	}
}

func TestDeleteDraftRemovesExtraFees(t *testing.T) {
	ctx := context.Background()
	svc, sqlDB := testService(t)
	seedWorker(t, sqlDB, 1, "Budi", 100_000)

	recap, err := svc.Create(ctx, CreateInput{Mode: "all_unsettled"})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if _, err := svc.AddFee(ctx, recap.ID, 1, 10_000, "Bonus"); err != nil {
		t.Fatalf("AddFee: %v", err)
	}

	if err := svc.DeleteDraft(ctx, recap.ID); err != nil {
		t.Fatalf("DeleteDraft: %v", err)
	}

	var fees int
	if err := sqlDB.QueryRowContext(ctx, `SELECT COUNT(*) FROM extra_fees`).Scan(&fees); err != nil {
		t.Fatalf("count fees: %v", err)
	}
	if fees != 0 {
		t.Fatalf("extra fees = %d, want 0", fees)
	}
	if _, err := svc.Detail(ctx, recap.ID); err != ErrNotFound {
		t.Fatalf("detail after delete = %v, want ErrNotFound", err)
	}
}

func TestExtraFeeValidation(t *testing.T) {
	ctx := context.Background()
	svc, sqlDB := testService(t)
	seedWorker(t, sqlDB, 1, "Budi", 100_000)

	recap, err := svc.Create(ctx, CreateInput{Mode: "all_unsettled"})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	if _, err := svc.AddFee(ctx, recap.ID, 1, 0, "Bonus"); err != ErrInvalidAmount {
		t.Fatalf("zero amount = %v, want ErrInvalidAmount", err)
	}
	if _, err := svc.AddFee(ctx, recap.ID, 1, 1000, "  "); err != ErrDescriptionRequired {
		t.Fatalf("empty description = %v, want ErrDescriptionRequired", err)
	}
	if _, err := svc.AddFee(ctx, recap.ID, 99, 1000, "Bonus"); err != ErrInvalidWorker {
		t.Fatalf("unknown worker = %v, want ErrInvalidWorker", err)
	}

	fee, err := svc.AddFee(ctx, recap.ID, 1, 1000, "Bonus")
	if err != nil {
		t.Fatalf("AddFee: %v", err)
	}
	if err := svc.UpdateFee(ctx, recap.ID, fee.ID, 2000, "Bonus lebih"); err != nil {
		t.Fatalf("UpdateFee: %v", err)
	}
	if err := svc.DeleteFee(ctx, recap.ID, fee.ID); err != nil {
		t.Fatalf("DeleteFee: %v", err)
	}
	if err := svc.DeleteFee(ctx, recap.ID, fee.ID); err != ErrNotFound {
		t.Fatalf("delete twice = %v, want ErrNotFound", err)
	}
}

func TestFinalizeRejectsNonDraft(t *testing.T) {
	ctx := context.Background()
	svc, sqlDB := testService(t)
	seedWorker(t, sqlDB, 1, "Budi", 100_000)
	seedAttendance(t, sqlDB, 1, "2026-01-05", 4, 100_000)

	recap, err := svc.Create(ctx, CreateInput{Mode: "all_unsettled"})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if err := svc.Finalize(ctx, recap.ID); err != nil {
		t.Fatalf("Finalize: %v", err)
	}
	if err := svc.Finalize(ctx, recap.ID); err != ErrNotDraft {
		t.Fatalf("second finalize = %v, want ErrNotDraft", err)
	}
	if err := svc.DeleteDraft(ctx, recap.ID); err != ErrNotDraft {
		t.Fatalf("delete final = %v, want ErrNotDraft", err)
	}
}

func TestListOrdersByCreatedAtAndFormatsName(t *testing.T) {
	ctx := context.Background()
	svc, sqlDB := testService(t)

	if _, err := sqlDB.ExecContext(ctx,
		`INSERT INTO recaps (id, mode, status, created_at) VALUES
			(1, 'all_unsettled', 'final', '2026-10-09T03:00:00Z'),
			(2, 'all_unsettled', 'final', '2026-10-10T17:30:00Z'),
			(3, 'since_monday', 'final', '2026-10-10T10:00:00Z')`); err != nil {
		t.Fatalf("seed recaps: %v", err)
	}

	recaps, err := svc.List(ctx)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(recaps) != 3 {
		t.Fatalf("recaps = %d, want 3", len(recaps))
	}
	if recaps[0].ID != 2 || recaps[1].ID != 3 || recaps[2].ID != 1 {
		t.Fatalf("order = [%d %d %d], want [2 3 1]", recaps[0].ID, recaps[1].ID, recaps[2].ID)
	}
	if recaps[0].Name != "Rekap Minggu, 11 Oktober 2026" {
		t.Fatalf("name = %q, want Jakarta date", recaps[0].Name)
	}
	if recaps[2].Name != "Rekap Jumat, 9 Oktober 2026" {
		t.Fatalf("name = %q", recaps[2].Name)
	}

	created, err := svc.Create(ctx, CreateInput{Mode: "all_unsettled"})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if created.Name != "Rekap Senin, 12 Januari 2026" {
		t.Fatalf("created name = %q", created.Name)
	}
}

func TestRecapNameFallback(t *testing.T) {
	if got := recapName("2026-10-10"); got != "Rekap Sabtu, 10 Oktober 2026" {
		t.Fatalf("date-only name = %q", got)
	}
	if got := recapName("not a date"); got != "Rekap" {
		t.Fatalf("invalid name = %q", got)
	}
}

// newFinalRecap seeds one worker attendance and finalizes a recap.
func newFinalRecap(t *testing.T, svc *Service, sqlDB *sql.DB) int64 {
	t.Helper()
	seedWorker(t, sqlDB, 1, "Budi", 100_000)
	seedAttendance(t, sqlDB, 1, "2026-01-05", 4, 100_000)

	recap, err := svc.Create(context.Background(), CreateInput{Mode: "all_unsettled"})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if err := svc.Finalize(context.Background(), recap.ID); err != nil {
		t.Fatalf("Finalize: %v", err)
	}
	return recap.ID
}

func TestCreateShareIsIdempotent(t *testing.T) {
	ctx := context.Background()
	svc, sqlDB := testService(t)
	seedWorker(t, sqlDB, 1, "Budi", 100_000)
	seedAttendance(t, sqlDB, 1, "2026-01-05", 4, 100_000)

	draft, err := svc.Create(ctx, CreateInput{Mode: "all_unsettled"})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	first, err := svc.CreateShare(ctx, draft.ID)
	if err != nil {
		t.Fatalf("CreateShare: %v", err)
	}
	if len(first) != 43 {
		t.Fatalf("token length = %d, want 43 base64url chars", len(first))
	}

	second, err := svc.CreateShare(ctx, draft.ID)
	if err != nil {
		t.Fatalf("CreateShare again: %v", err)
	}
	if second != first {
		t.Fatalf("token changed: %q then %q", first, second)
	}

	stored, err := svc.ShareToken(ctx, draft.ID)
	if err != nil {
		t.Fatalf("ShareToken: %v", err)
	}
	if stored != first {
		t.Fatalf("ShareToken = %q, want %q", stored, first)
	}

	var count int
	if err := sqlDB.QueryRowContext(ctx, `SELECT COUNT(*) FROM recap_shares`).Scan(&count); err != nil {
		t.Fatalf("count shares: %v", err)
	}
	if count != 1 {
		t.Fatalf("share rows = %d, want 1", count)
	}

	if _, err := svc.CreateShare(ctx, 999); err != ErrNotFound {
		t.Fatalf("unknown recap share error = %v, want ErrNotFound", err)
	}
}

func TestRevokeShare(t *testing.T) {
	ctx := context.Background()
	svc, sqlDB := testService(t)
	recapID := newFinalRecap(t, svc, sqlDB)

	token, err := svc.CreateShare(ctx, recapID)
	if err != nil {
		t.Fatalf("CreateShare: %v", err)
	}

	if err := svc.RevokeShare(ctx, recapID); err != nil {
		t.Fatalf("RevokeShare: %v", err)
	}
	if _, err := svc.PublicDetail(ctx, token); err != ErrShareNotFound {
		t.Fatalf("public detail after revoke = %v, want ErrShareNotFound", err)
	}
	if _, err := svc.ShareToken(ctx, recapID); err != ErrShareNotFound {
		t.Fatalf("share token after revoke = %v, want ErrShareNotFound", err)
	}

	if err := svc.RevokeShare(ctx, 999); err != ErrNotFound {
		t.Fatalf("revoke unknown recap = %v, want ErrNotFound", err)
	}

	// Revoking again is a no-op.
	if err := svc.RevokeShare(ctx, recapID); err != nil {
		t.Fatalf("RevokeShare twice: %v", err)
	}
}

func TestPublicDetailShareLifecycle(t *testing.T) {
	ctx := context.Background()
	svc, sqlDB := testService(t)
	recapID := newFinalRecap(t, svc, sqlDB)

	token, err := svc.CreateShare(ctx, recapID)
	if err != nil {
		t.Fatalf("CreateShare: %v", err)
	}

	detail, err := svc.PublicDetail(ctx, token)
	if err != nil {
		t.Fatalf("PublicDetail: %v", err)
	}
	if detail.Recap.ID != recapID || detail.Recap.Status != "final" {
		t.Fatalf("public recap = %+v", detail.Recap)
	}
	if len(detail.Workers) != 1 || detail.Workers[0].Net != 100_000 {
		t.Fatalf("public workers = %+v", detail.Workers)
	}
	if detail.PeriodStart != "2026-01-05" || detail.PeriodEnd != "2026-01-05" {
		t.Fatalf("public period = %s..%s", detail.PeriodStart, detail.PeriodEnd)
	}

	if _, err := svc.PublicDetail(ctx, "unknown-token"); err != ErrShareNotFound {
		t.Fatalf("unknown token = %v, want ErrShareNotFound", err)
	}
	if _, err := svc.PublicDetail(ctx, ""); err != ErrShareNotFound {
		t.Fatalf("empty token = %v, want ErrShareNotFound", err)
	}

	if err := svc.Reopen(ctx, recapID); err != nil {
		t.Fatalf("Reopen: %v", err)
	}
	if _, err := svc.PublicDetail(ctx, token); err != nil {
		t.Fatalf("public detail after reopen: %v", err)
	}
}
