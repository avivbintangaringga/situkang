-- 0005_extra_fee_dates.sql - standalone extra fees are recorded per date.
-- SQLite cannot add a NOT NULL column to a populated table, so rebuild it.

CREATE TABLE extra_fees_new (
    id          INTEGER PRIMARY KEY,
    worker_id   INTEGER NOT NULL REFERENCES workers(id),
    date        TEXT NOT NULL,
    recap_id    INTEGER REFERENCES recaps(id) ON DELETE CASCADE,
    amount      INTEGER NOT NULL CHECK (amount <> 0),
    description TEXT NOT NULL,
    created_at  TEXT NOT NULL
) STRICT;

INSERT INTO extra_fees_new (id, worker_id, date, recap_id, amount, description, created_at)
SELECT id, worker_id, substr(created_at, 1, 10), recap_id, amount, description, created_at
FROM extra_fees;

DROP TABLE extra_fees;

ALTER TABLE extra_fees_new RENAME TO extra_fees;

CREATE INDEX idx_extra_fees_recap ON extra_fees (recap_id);
CREATE INDEX idx_extra_fees_worker ON extra_fees (worker_id);
CREATE INDEX idx_extra_fees_date ON extra_fees (date);
