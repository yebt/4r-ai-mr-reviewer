package logx

import (
	"bytes"
	"strings"
	"testing"
	"time"
)

func TestClassify(t *testing.T) {
	cases := []struct {
		msg  string
		want level
	}{
		// Real lines from the server: routine status is INFO, not an error.
		{`routines: run abc blocked at step "verify": pipeline not finished`, levelInfo},
		{`routines: run abc awaiting confirmation at step "confirm"`, levelInfo},
		{`ai-reviewer: listening on :8080`, levelInfo},
		// Failures are errors.
		{`jobs: review abc failed (attempt 1): ai: read body: context deadline exceeded`, levelError},
		{`gitlab webhook: dispatch panic: boom`, levelError},
		{`http 500: cannot reach upstream`, levelError},
		// Recoverable hiccups and notices are warnings.
		{`ai: anthropic thinking rejected (400), retrying without thinking`, levelWarn},
		{`ai-reviewer: WARNING — API authentication is DISABLED`, levelWarn},
		{`gitlab webhook: skipped review (already active)`, levelWarn},
	}
	for _, tc := range cases {
		if got := classify(tc.msg); got != tc.want {
			t.Errorf("classify(%q) = %v, want %v", tc.msg, got, tc.want)
		}
	}
}

func TestFormatNoColor(t *testing.T) {
	w := &writer{color: false}
	ts := time.Date(2026, 8, 26, 16, 18, 8, 0, time.UTC)
	got := w.format(ts, levelError, "jobs: review failed")
	want := "2026/08/26 16:18:08 ERROR jobs: review failed\n"
	if got != want {
		t.Fatalf("format = %q, want %q", got, want)
	}
	if strings.Contains(got, "\x1b[") {
		t.Fatal("no-color format must not contain ANSI escapes")
	}
}

func TestFormatColorWrapsLevelTag(t *testing.T) {
	w := &writer{color: true}
	ts := time.Date(2026, 8, 26, 16, 18, 8, 0, time.UTC)
	got := w.format(ts, levelError, "boom")
	if !strings.Contains(got, red+"ERROR"+reset) {
		t.Fatalf("colored ERROR tag missing in %q", got)
	}
	if !strings.Contains(got, dim+"2026/08/26 16:18:08"+reset) {
		t.Fatalf("dimmed timestamp missing in %q", got)
	}
	if !strings.HasSuffix(got, "boom\n") {
		t.Fatalf("message/newline missing in %q", got)
	}
}

func TestWriterReportsFullWrite(t *testing.T) {
	var buf bytes.Buffer
	w := &writer{out: &buf, color: false}
	in := []byte("jobs: review failed\n")
	n, err := w.Write(in)
	if err != nil {
		t.Fatalf("Write: %v", err)
	}
	// The log package treats a short write as an error; we must report len(p).
	if n != len(in) {
		t.Fatalf("n = %d, want %d", n, len(in))
	}
	out := buf.String()
	if !strings.HasPrefix(out, "20") || !strings.Contains(out, " ERROR jobs: review failed\n") {
		t.Fatalf("unexpected line: %q", out)
	}
}

func TestColorEnabledOverrides(t *testing.T) {
	var notATerminal bytes.Buffer

	t.Setenv("NO_COLOR", "")
	t.Setenv("AIR_LOG_COLOR", "always")
	if !colorEnabled(&notATerminal) {
		t.Error("AIR_LOG_COLOR=always must force color even off a terminal")
	}

	t.Setenv("AIR_LOG_COLOR", "never")
	if colorEnabled(&notATerminal) {
		t.Error("AIR_LOG_COLOR=never must disable color")
	}

	t.Setenv("AIR_LOG_COLOR", "")
	t.Setenv("NO_COLOR", "1")
	if colorEnabled(&notATerminal) {
		t.Error("NO_COLOR must disable color")
	}

	t.Setenv("NO_COLOR", "")
	// A non-*os.File writer (a buffer) is never a terminal, so auto-detect is off.
	if colorEnabled(&notATerminal) {
		t.Error("a non-terminal writer must default to no color")
	}
}
