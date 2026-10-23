-- 0004_recap_shares.sql - public read-only share links for finalized recaps.

CREATE TABLE recap_shares (
    recap_id   INTEGER PRIMARY KEY REFERENCES recaps(id) ON DELETE CASCADE,
    token      TEXT NOT NULL UNIQUE,
    created_at TEXT NOT NULL
) STRICT;

CREATE INDEX idx_recap_shares_token ON recap_shares (token);
