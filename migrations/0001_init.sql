-- 0001_init.sql — initial schema for SiTukang.
-- Money is INTEGER rupiah. Dates are TEXT 'YYYY-MM-DD'. Timestamps are RFC3339 UTC.

CREATE TABLE users (
    id            INTEGER PRIMARY KEY,
    username      TEXT NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    created_at    TEXT NOT NULL
) STRICT;

CREATE TABLE sessions (
    id         TEXT PRIMARY KEY,
    user_id    INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    expires_at TEXT NOT NULL,
    created_at TEXT NOT NULL
) STRICT;

CREATE TABLE settings (
    key   TEXT PRIMARY KEY,
    value TEXT NOT NULL
) STRICT;

CREATE TABLE workers (
    id         INTEGER PRIMARY KEY,
    name       TEXT NOT NULL,
    role       TEXT NOT NULL,
    daily_fee  INTEGER NOT NULL CHECK (daily_fee >= 0),
    active     INTEGER NOT NULL DEFAULT 1 CHECK (active IN (0, 1)),
    created_at TEXT NOT NULL
) STRICT;

CREATE TABLE projects (
    id         INTEGER PRIMARY KEY,
    name       TEXT NOT NULL,
    start_date TEXT,
    end_date   TEXT,
    status     TEXT NOT NULL CHECK (status IN ('aktif', 'selesai', 'ditunda')),
    notes      TEXT,
    created_at TEXT NOT NULL
) STRICT;

CREATE TABLE recaps (
    id           INTEGER PRIMARY KEY,
    mode         TEXT NOT NULL CHECK (mode IN ('since_monday', 'all_unsettled', 'custom')),
    start_date   TEXT,
    end_date     TEXT,
    status       TEXT NOT NULL CHECK (status IN ('draft', 'final')),
    created_at   TEXT NOT NULL,
    finalized_at TEXT
) STRICT;

CREATE TABLE attendances (
    id                 INTEGER PRIMARY KEY,
    worker_id          INTEGER NOT NULL REFERENCES workers(id),
    date               TEXT NOT NULL,
    quarters           INTEGER NOT NULL CHECK (quarters BETWEEN 1 AND 4),
    daily_fee_snapshot INTEGER NOT NULL CHECK (daily_fee_snapshot >= 0),
    recap_id           INTEGER REFERENCES recaps(id),
    created_at         TEXT NOT NULL,
    UNIQUE (worker_id, date)
) STRICT;

CREATE TABLE attendance_projects (
    attendance_id INTEGER NOT NULL REFERENCES attendances(id) ON DELETE CASCADE,
    project_id    INTEGER NOT NULL REFERENCES projects(id),
    PRIMARY KEY (attendance_id, project_id)
) STRICT;

CREATE TABLE cash_advances (
    id              INTEGER PRIMARY KEY,
    worker_id       INTEGER NOT NULL REFERENCES workers(id),
    date            TEXT NOT NULL,
    amount          INTEGER NOT NULL CHECK (amount > 0),
    note            TEXT,
    recap_id        INTEGER REFERENCES recaps(id),
    source_recap_id INTEGER REFERENCES recaps(id),
    created_at      TEXT NOT NULL
) STRICT;

CREATE TABLE recap_lines (
    id            INTEGER PRIMARY KEY,
    recap_id      INTEGER NOT NULL REFERENCES recaps(id) ON DELETE CASCADE,
    worker_id     INTEGER NOT NULL REFERENCES workers(id),
    worker_name   TEXT NOT NULL,
    worker_role   TEXT NOT NULL,
    days_quarters INTEGER NOT NULL CHECK (days_quarters >= 0),
    gross         INTEGER NOT NULL,
    advances      INTEGER NOT NULL,
    extras        INTEGER NOT NULL,
    net           INTEGER NOT NULL
) STRICT;

CREATE TABLE recap_paid (
    recap_id  INTEGER NOT NULL REFERENCES recaps(id) ON DELETE CASCADE,
    worker_id INTEGER NOT NULL REFERENCES workers(id),
    paid      INTEGER NOT NULL DEFAULT 1 CHECK (paid IN (0, 1)),
    PRIMARY KEY (recap_id, worker_id)
) STRICT;

CREATE TABLE extra_fees (
    id          INTEGER PRIMARY KEY,
    worker_id   INTEGER NOT NULL REFERENCES workers(id),
    recap_id    INTEGER REFERENCES recaps(id) ON DELETE CASCADE,
    amount      INTEGER NOT NULL CHECK (amount <> 0),
    description TEXT NOT NULL,
    created_at  TEXT NOT NULL
) STRICT;

CREATE INDEX idx_attendances_worker_date ON attendances (worker_id, date);
CREATE INDEX idx_attendances_recap ON attendances (recap_id);
CREATE INDEX idx_cash_advances_worker_recap ON cash_advances (worker_id, recap_id);
CREATE INDEX idx_cash_advances_source_recap ON cash_advances (source_recap_id);
CREATE INDEX idx_extra_fees_recap ON extra_fees (recap_id);
CREATE INDEX idx_extra_fees_worker ON extra_fees (worker_id);
CREATE INDEX idx_recap_paid_recap ON recap_paid (recap_id);
CREATE INDEX idx_recap_lines_recap ON recap_lines (recap_id);

INSERT INTO settings (key, value) VALUES ('working_days', '1,2,3,4,5,6');
