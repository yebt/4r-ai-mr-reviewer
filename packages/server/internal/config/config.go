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
	// HTTPAddr is the address the API server listens on.
	HTTPAddr string
	// SkillsDir optionally overrides the embedded 4R rule files.
	SkillsDir string
	// Password unlocks the secret vault at startup. Empty selects key-file mode.
	Password string
}

// Load reads configuration from the environment.
func Load() Config {
	dbPath := envOr("AIR_DB_PATH", "ai-reviewer.db")
	return Config{
		DBPath:      dbPath,
		KeyfilePath: envOr("AIR_KEYFILE_PATH", dbPath+".key"),
		// Bind to loopback by default so the API (which acts on stored GitLab
		// PATs and provider keys, with no auth of its own) is not exposed on all
		// interfaces. Containerized deploys set AIR_HTTP_ADDR=:8080 explicitly.
		HTTPAddr:  envOr("AIR_HTTP_ADDR", "127.0.0.1:8080"),
		SkillsDir: os.Getenv("AIR_SKILLS_DIR"),
		Password:  os.Getenv("AIR_PASSWORD"),
	}
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
