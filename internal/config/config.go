// Package config loads the server configuration from environment variables.
package config

import (
	"fmt"
	"os"
	"time"
)

// Config holds everything the server needs to boot.
type Config struct {
	Port          string
	DBPath        string
	SessionSecret string
	Location      *time.Location
}

// Load reads the environment and applies the defaults from AGENTS.md.
func Load() (Config, error) {
	cfg := Config{
		Port:          env("PORT", "8080"),
		DBPath:        env("DB_PATH", "./data/situkang.db"),
		SessionSecret: os.Getenv("SESSION_SECRET"),
	}

	tz := env("TZ", "Asia/Jakarta")
	loc, err := time.LoadLocation(tz)
	if err != nil {
		return Config{}, fmt.Errorf("invalid timezone %q: %w", tz, err)
	}
	cfg.Location = loc

	return cfg, nil
}

func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
