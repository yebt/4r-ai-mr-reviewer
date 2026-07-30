// Package config loads runtime configuration from the environment with
// sensible defaults for local development.
package config

import (
	"os"
	"strconv"
	"time"
)

// maxReasoningBudget caps AIR_REASONING_BUDGET defensively (mirroring the ai
// package's retryAfterCap / maxResponseBytes bounds), so a mistyped value cannot
// request an unbounded thinking budget.
const maxReasoningBudget = 32768

// Merge-wait bounds for the release routine. The default is 10 minutes; the
// ceiling caps AIR_ROUTINE_MERGE_TIMEOUT so a mistyped value cannot pin the
// single worker on one run indefinitely.
const (
	defaultMergeWaitSeconds = 600
	maxMergeWaitSeconds     = 3600
)

// Session-lifetime bounds for API auth. The default is 7 days; the value is
// clamped to [minAuthSessionHours, maxAuthSessionHours] so a mistyped env value
// cannot mint effectively-immortal or already-dead sessions.
const (
	defaultAuthSessionHours = 168 // 7 days
	minAuthSessionHours     = 1
	maxAuthSessionHours     = 8760 // 365 days
)

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
	// AuthPassword opts the HTTP API into password + signed-cookie session auth.
	// Empty disables auth entirely (all routes pass through); non-empty enforces
	// it. This is distinct from Password (the vault password) on purpose.
	AuthPassword string
	// AuthSessionHours is the session-cookie lifetime in hours (clamped to a sane
	// range). It only matters when AuthPassword is set.
	AuthSessionHours int
	// TrustProxyHeaders opts into trusting client-supplied X-Forwarded-* headers
	// (X-Forwarded-Proto for the cookie Secure decision, X-Forwarded-For for
	// login rate limiting). Leave false unless a trusted TLS-terminating proxy
	// sits in front and sets these from a trusted source.
	TrustProxyHeaders bool
	// ReviewConcurrency is how many reviews run at once. Bounds LLM concurrency
	// and resource use; keep it modest.
	ReviewConcurrency int
	// ReasoningBudget controls reasoning capture per 4R phase. 0 disables reasoning
	// capture; a positive value is the Anthropic thinking-token budget and also
	// enables capture of reasoning fields returned by OpenAI-compatible providers.
	ReasoningBudget int
	// TelegramWebhookSecret gates the interactive Telegram webhook receiver.
	// Empty keeps the receiver dormant (it returns 200 without processing).
	TelegramWebhookSecret string
	// MergeWaitTimeout bounds how long a release routine waits for its MR to be
	// merged before it blocks (and can be resumed to keep waiting).
	MergeWaitTimeout time.Duration
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
		HTTPAddr:              envOr("AIR_HTTP_ADDR", "127.0.0.1:8080"),
		SkillsDir:             os.Getenv("AIR_SKILLS_DIR"),
		Password:              os.Getenv("AIR_PASSWORD"),
		AuthPassword:          os.Getenv("AIR_AUTH_PASSWORD"),
		AuthSessionHours:      clampAuthSessionHours(envInt("AIR_AUTH_SESSION_HOURS", defaultAuthSessionHours)),
		TrustProxyHeaders:     envBool("AIR_TRUST_PROXY", false),
		ReviewConcurrency:     envInt("AIR_REVIEW_CONCURRENCY", 2),
		ReasoningBudget:       clampReasoningBudget(envInt("AIR_REASONING_BUDGET", 0)),
		TelegramWebhookSecret: os.Getenv("AIR_TELEGRAM_WEBHOOK_SECRET"),
		MergeWaitTimeout:      mergeWaitTimeout(envInt("AIR_ROUTINE_MERGE_TIMEOUT", defaultMergeWaitSeconds)),
	}
}

// mergeWaitTimeout converts a seconds value into a duration, clamping it to
// maxMergeWaitSeconds so a mistyped env value cannot stall the worker.
func mergeWaitTimeout(secs int) time.Duration {
	if secs > maxMergeWaitSeconds {
		secs = maxMergeWaitSeconds
	}
	return time.Duration(secs) * time.Second
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// clampReasoningBudget caps a positive budget at maxReasoningBudget. 0 (disabled)
// and any already-in-range value pass through unchanged.
func clampReasoningBudget(v int) int {
	if v > maxReasoningBudget {
		return maxReasoningBudget
	}
	return v
}

func envInt(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			return n
		}
	}
	return fallback
}

// envBool parses key as a boolean (strconv.ParseBool: 1/t/true/0/f/false, etc.),
// falling back on an unset or unparseable value.
func envBool(key string, fallback bool) bool {
	if v := os.Getenv(key); v != "" {
		if b, err := strconv.ParseBool(v); err == nil {
			return b
		}
	}
	return fallback
}

// clampAuthSessionHours bounds the session lifetime to [minAuthSessionHours,
// maxAuthSessionHours] so a mistyped value cannot mint unbounded sessions.
func clampAuthSessionHours(h int) int {
	if h < minAuthSessionHours {
		return minAuthSessionHours
	}
	if h > maxAuthSessionHours {
		return maxAuthSessionHours
	}
	return h
}
