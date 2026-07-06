// Package config loads runtime configuration from the environment with
// sensible defaults for local development.
package config

import "os"

// Config holds process-wide runtime settings.
type Config struct {
	// DBPath is the SQLite database file path.
	DBPath string
}

// Load reads configuration from the environment.
func Load() Config {
	return Config{
		DBPath: envOr("AIR_DB_PATH", "ai-reviewer.db"),
	}
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
