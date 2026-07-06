// Package config loads runtime configuration from the environment with
// sensible defaults for local development.
package config

import "os"

// Config holds process-wide runtime settings.
type Config struct {
	// DBPath is the SQLite database file path.
	DBPath string
	// KeyfilePath is where the master key lives when no app password is set.
	KeyfilePath string
}

// Load reads configuration from the environment.
func Load() Config {
	dbPath := envOr("AIR_DB_PATH", "ai-reviewer.db")
	return Config{
		DBPath:      dbPath,
		KeyfilePath: envOr("AIR_KEYFILE_PATH", dbPath+".key"),
	}
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
