package auth

import (
	"context"
	"database/sql"
	"time"
)

// User is the account that may sign in. It deliberately carries no password.
type User struct {
	ID       int64  `json:"id"`
	Username string `json:"username"`
}

// Account is the account row shown in the user management screen.
type Account struct {
	ID        int64  `json:"id"`
	Username  string `json:"username"`
	CreatedAt string `json:"created_at"`
}

type userRecord struct {
	User
	PasswordHash string
}

func findUserByUsername(ctx context.Context, db *sql.DB, username string) (userRecord, error) {
	const query = `SELECT id, username, password_hash FROM users WHERE username = ?`

	var rec userRecord
	err := db.QueryRowContext(ctx, query, username).Scan(&rec.ID, &rec.Username, &rec.PasswordHash)
	return rec, err
}

func findUserByID(ctx context.Context, db *sql.DB, id int64) (User, error) {
	const query = `SELECT id, username FROM users WHERE id = ?`

	var user User
	err := db.QueryRowContext(ctx, query, id).Scan(&user.ID, &user.Username)
	return user, err
}

func findUserRecordByID(ctx context.Context, db *sql.DB, id int64) (userRecord, error) {
	const query = `SELECT id, username, password_hash FROM users WHERE id = ?`

	var rec userRecord
	err := db.QueryRowContext(ctx, query, id).Scan(&rec.ID, &rec.Username, &rec.PasswordHash)
	return rec, err
}

func listUsers(ctx context.Context, db *sql.DB) ([]Account, error) {
	const query = `SELECT id, username, created_at FROM users ORDER BY username`

	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	accounts := make([]Account, 0)
	for rows.Next() {
		var account Account
		if err := rows.Scan(&account.ID, &account.Username, &account.CreatedAt); err != nil {
			return nil, err
		}
		accounts = append(accounts, account)
	}
	return accounts, rows.Err()
}

func updateUsername(ctx context.Context, db *sql.DB, id int64, username string) error {
	_, err := db.ExecContext(ctx, `UPDATE users SET username = ? WHERE id = ?`, username, id)
	return err
}

func updateUserPassword(ctx context.Context, tx *sql.Tx, id int64, passwordHash string) error {
	_, err := tx.ExecContext(ctx, `UPDATE users SET password_hash = ? WHERE id = ?`, passwordHash, id)
	return err
}

func insertUser(ctx context.Context, db *sql.DB, username, passwordHash string, now time.Time) (User, error) {
	const query = `INSERT INTO users (username, password_hash, created_at) VALUES (?, ?, ?)`

	res, err := db.ExecContext(ctx, query, username, passwordHash, now.UTC().Format(time.RFC3339))
	if err != nil {
		return User{}, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return User{}, err
	}
	return User{ID: id, Username: username}, nil
}

func insertSession(ctx context.Context, db *sql.DB, id string, userID int64, expiresAt, now time.Time) error {
	const query = `INSERT INTO sessions (id, user_id, expires_at, created_at) VALUES (?, ?, ?, ?)`

	_, err := db.ExecContext(ctx, query, id, userID,
		expiresAt.UTC().Format(time.RFC3339), now.UTC().Format(time.RFC3339))
	return err
}

func findSession(ctx context.Context, db *sql.DB, id string) (User, time.Time, error) {
	const query = `
		SELECT u.id, u.username, s.expires_at
		FROM sessions s
		JOIN users u ON u.id = s.user_id
		WHERE s.id = ?`

	var (
		user      User
		expiresAt string
	)
	if err := db.QueryRowContext(ctx, query, id).Scan(&user.ID, &user.Username, &expiresAt); err != nil {
		return User{}, time.Time{}, err
	}

	parsed, err := time.Parse(time.RFC3339, expiresAt)
	if err != nil {
		return User{}, time.Time{}, err
	}
	return user, parsed, nil
}

func deleteSession(ctx context.Context, db *sql.DB, id string) error {
	_, err := db.ExecContext(ctx, `DELETE FROM sessions WHERE id = ?`, id)
	return err
}

// deleteOtherSessions removes every session of a user except keepID, which is
// the hashed id of the session that must stay signed in.
func deleteOtherSessions(ctx context.Context, tx *sql.Tx, userID int64, keepID string) error {
	_, err := tx.ExecContext(ctx,
		`DELETE FROM sessions WHERE user_id = ? AND id <> ?`, userID, keepID)
	return err
}
