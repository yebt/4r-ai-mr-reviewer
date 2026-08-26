// Package logx wraps the standard logger's output with a colored, structured
// line format. It infers a severity level from the message (the codebase logs
// through the flat stdlib log.Printf with semantic prefixes, not levels), tags
// each line with it, and colors the tag so real failures stand out from routine
// status. Color is applied only when writing to a terminal and NO_COLOR is
// unset, so file/journald output stays clean; AIR_LOG_COLOR forces the choice.
package logx

import (
	"io"
	"log"
	"os"
	"strings"
	"sync"
	"time"
)

// timeLayout matches the stdlib default (date + time), so nothing is lost versus
// log.LstdFlags — only reformatted and colored.
const timeLayout = "2006/01/02 15:04:05"

// ANSI escapes. Kept local so the package has no dependency.
const (
	reset  = "\x1b[0m"
	dim    = "\x1b[90m" // bright black / grey
	red    = "\x1b[1;31m"
	yellow = "\x1b[33m"
	green  = "\x1b[32m"
)

// level is the inferred severity of a log line.
type level int

const (
	levelInfo level = iota
	levelWarn
	levelError
)

// tag is the fixed-width label printed for the level (aligned across lines).
func (l level) tag() string {
	switch l {
	case levelError:
		return "ERROR"
	case levelWarn:
		return "WARN "
	default:
		return "INFO "
	}
}

func (l level) color() string {
	switch l {
	case levelError:
		return red
	case levelWarn:
		return yellow
	default:
		return green
	}
}

// errorHints and warnHints are matched case-insensitively against the message.
// Error is checked first, so "retry failed" reads as an error, not a warning.
var (
	errorHints = []string{"panic", "fatal", "failed", "fail:", " error", "error:", "cannot", "unable", " err=", "err:"}
	warnHints  = []string{"warning", "warn", "retrying", "deprecated", "skipped"}
)

// classify infers the severity of a message from its wording.
func classify(msg string) level {
	m := strings.ToLower(msg)
	for _, h := range errorHints {
		if strings.Contains(m, h) {
			return levelError
		}
	}
	for _, h := range warnHints {
		if strings.Contains(m, h) {
			return levelWarn
		}
	}
	return levelInfo
}

// Setup routes the standard logger through a colored, structured writer. It
// clears the stdlib flags (this writer supplies its own timestamp) and returns
// the writer. Call once at startup, before the first log line.
func Setup() io.Writer {
	w := NewWriter(os.Stderr)
	log.SetFlags(0)
	log.SetOutput(w)
	return w
}

// writer formats and colors each log line as it is written.
type writer struct {
	mu    sync.Mutex
	out   io.Writer
	color bool
}

// NewWriter builds a writer over out, deciding once whether to emit color.
func NewWriter(out io.Writer) io.Writer {
	return &writer{out: out, color: colorEnabled(out)}
}

// Write receives one already-assembled log line (stdlib flags are cleared, so it
// is just the message plus a trailing newline), classifies it, and re-emits it
// as "timestamp LEVEL message".
func (w *writer) Write(p []byte) (int, error) {
	msg := strings.TrimRight(string(p), "\n")
	lvl := classify(msg)
	line := w.format(time.Now(), lvl, msg)

	w.mu.Lock()
	defer w.mu.Unlock()
	if _, err := io.WriteString(w.out, line); err != nil {
		return 0, err
	}
	// Report the full input as consumed: the log package treats a short write as
	// an error, and the reformatting is our concern, not the caller's.
	return len(p), nil
}

func (w *writer) format(t time.Time, lvl level, msg string) string {
	ts := t.Format(timeLayout)
	tag := lvl.tag()
	if !w.color {
		return ts + " " + tag + " " + msg + "\n"
	}
	return dim + ts + reset + " " + lvl.color() + tag + reset + " " + msg + "\n"
}

// colorEnabled decides whether to emit ANSI color: AIR_LOG_COLOR wins if set
// (always/never), then NO_COLOR disables it, otherwise color is on only when out
// is a character device (a terminal).
func colorEnabled(out io.Writer) bool {
	switch strings.ToLower(strings.TrimSpace(os.Getenv("AIR_LOG_COLOR"))) {
	case "always", "1", "true", "yes":
		return true
	case "never", "0", "false", "no":
		return false
	}
	if os.Getenv("NO_COLOR") != "" {
		return false
	}
	f, ok := out.(*os.File)
	if !ok {
		return false
	}
	st, err := f.Stat()
	return err == nil && st.Mode()&os.ModeCharDevice != 0
}
