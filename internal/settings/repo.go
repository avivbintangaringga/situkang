package settings

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"
	"strings"
)

const workingDaysKey = "working_days"

func getValue(ctx context.Context, db *sql.DB, key string) (string, error) {
	var value string
	err := db.QueryRowContext(ctx, `SELECT value FROM settings WHERE key = ?`, key).Scan(&value)
	return value, err
}

func setValue(ctx context.Context, db *sql.DB, key, value string) error {
	const query = `
		INSERT INTO settings (key, value) VALUES (?, ?)
		ON CONFLICT (key) DO UPDATE SET value = excluded.value`

	if _, err := db.ExecContext(ctx, query, key, value); err != nil {
		return fmt.Errorf("save setting %s: %w", key, err)
	}
	return nil
}

func parseDays(raw string) ([]int, error) {
	parts := strings.Split(raw, ",")
	days := make([]int, 0, len(parts))
	for _, part := range parts {
		value := strings.TrimSpace(part)
		if value == "" {
			continue
		}
		day, err := strconv.Atoi(value)
		if err != nil {
			return nil, fmt.Errorf("parse working days: %w", err)
		}
		days = append(days, day)
	}
	return days, nil
}

func formatDays(days []int) string {
	parts := make([]string, len(days))
	for i, day := range days {
		parts[i] = strconv.Itoa(day)
	}
	return strings.Join(parts, ",")
}
