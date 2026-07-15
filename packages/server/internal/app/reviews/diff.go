package reviews

import (
	"strconv"
	"strings"

	"github.com/webcloster-dev/ai-reviewer/internal/adapters/gitlab"
)

// addedLines parses each file's unified diff and returns, per file, the set of
// NEW-side line numbers that are ADDED ('+') lines. Only added lines are safe to
// anchor an inline GitLab discussion to with just position[new_line]; context or
// deleted lines require old_line/old_path and otherwise trigger a 400.
//
// The outer map is keyed by NewPath (falling back to OldPath when NewPath is
// empty). Parsing is deliberately defensive: malformed hunk headers are skipped
// without panicking, and GitLab's headerless diff (starting directly at "@@") is
// handled.
func addedLines(ch gitlab.Changes) map[string]map[int]bool {
	out := make(map[string]map[int]bool, len(ch.Files))
	for _, f := range ch.Files {
		key := f.NewPath
		if key == "" {
			key = f.OldPath
		}
		if key == "" {
			continue
		}
		lines := parseAddedLines(f.Diff)
		if len(lines) == 0 {
			continue
		}
		out[key] = lines
	}
	return out
}

// parseAddedLines walks a single unified diff and returns the new-side line
// numbers of its added ('+') lines.
func parseAddedLines(diff string) map[int]bool {
	added := make(map[int]bool)
	newLine := 0
	inHunk := false

	for _, line := range strings.Split(diff, "\n") {
		switch {
		case strings.HasPrefix(line, "@@"):
			start, ok := parseHunkNewStart(line)
			if !ok {
				inHunk = false
				continue
			}
			newLine = start
			inHunk = true
		case strings.HasPrefix(line, "+++ "), strings.HasPrefix(line, "--- "),
			strings.HasPrefix(line, "diff "), strings.HasPrefix(line, "index "),
			strings.HasPrefix(line, `\ `):
			// Diff metadata: not part of the line-number accounting.
			continue
		case !inHunk:
			// Ignore any content before the first valid hunk header.
			continue
		case strings.HasPrefix(line, "+"):
			// Added line on the new side.
			added[newLine] = true
			newLine++
		case strings.HasPrefix(line, "-"):
			// Deleted line: present only on the old side, does not advance new.
			continue
		default:
			// Context (leading space) or blank line: present on the new side but
			// not added.
			newLine++
		}
	}
	return added
}

// parseHunkNewStart extracts the new-side start line from a hunk header of the
// form "@@ -a,b +c,d @@" (the count is optional, e.g. "+c"). It returns false
// for a malformed header so the caller can skip the hunk safely.
func parseHunkNewStart(header string) (int, bool) {
	plus := strings.IndexByte(header, '+')
	if plus < 0 {
		return 0, false
	}
	rest := header[plus+1:]
	// The new-side span ends at a comma (start,count) or whitespace (start only).
	end := strings.IndexAny(rest, ", \t")
	if end >= 0 {
		rest = rest[:end]
	}
	start, err := strconv.Atoi(rest)
	if err != nil {
		return 0, false
	}
	return start, true
}
