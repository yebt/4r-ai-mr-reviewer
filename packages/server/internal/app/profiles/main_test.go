package profiles

import (
	"os"
	"testing"
)

// TestMain points the AI client at its buffered (non-streaming) path for this
// package's tests, whose stubs serve a single JSON completion rather than SSE.
// The streaming transport itself is covered in internal/adapters/ai.
func TestMain(m *testing.M) {
	os.Setenv("AIR_AI_STREAM", "false")
	os.Exit(m.Run())
}
