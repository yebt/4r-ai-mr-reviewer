package reviews

import (
	"testing"

	"github.com/webcloster-dev/ai-reviewer/internal/adapters/gitlab"
)

func TestAddedLinesMultiHunkMultiFile(t *testing.T) {
	// a.go: two hunks. First hunk starts at new line 10; second at new line 30.
	aGo := "@@ -8,3 +10,4 @@\n" +
		" context ten\n" + // new line 10, context
		"+added eleven\n" + // new line 11, ADDED
		"-deleted old\n" + // old side only, no new-line advance
		" context twelve\n" + // new line 12, context
		"@@ -25,2 +30,3 @@\n" +
		"+added thirty\n" + // new line 30, ADDED
		" context thirtyone\n" + // new line 31, context
		"+added thirtytwo\n" // new line 32, ADDED

	// b.go: a fresh-file style hunk with no count on the new side (+1).
	bGo := "@@ -0,0 +1 @@\n" +
		"+only line\n" // new line 1, ADDED

	ch := gitlab.Changes{
		Files: []gitlab.FileChange{
			{OldPath: "a.go", NewPath: "a.go", Diff: aGo},
			{OldPath: "b.go", NewPath: "b.go", Diff: bGo},
		},
	}

	got := addedLines(ch)

	wantA := map[int]bool{11: true, 30: true, 32: true}
	assertLineSet(t, "a.go", got["a.go"], wantA)

	wantB := map[int]bool{1: true}
	assertLineSet(t, "b.go", got["b.go"], wantB)

	// Context and deleted lines must be excluded.
	for _, ctxLine := range []int{10, 12, 31} {
		if got["a.go"][ctxLine] {
			t.Fatalf("a.go: line %d is context/deleted but was recorded as added", ctxLine)
		}
	}
}

func TestAddedLinesFallsBackToOldPath(t *testing.T) {
	ch := gitlab.Changes{
		Files: []gitlab.FileChange{
			{OldPath: "gone.go", NewPath: "", Diff: "@@ -1,0 +1,1 @@\n+x\n"},
		},
	}
	got := addedLines(ch)
	if !got["gone.go"][1] {
		t.Fatalf("expected key fallback to OldPath with line 1 added, got %v", got)
	}
}

func TestAddedLinesMalformedHunkSkipped(t *testing.T) {
	// A malformed hunk header (no new-side marker) must be skipped without panic;
	// the valid hunk that follows is still parsed.
	diff := "@@ this is broken @@\n" +
		"+ignored because no valid hunk\n" +
		"@@ -1,0 +5,1 @@\n" +
		"+real added\n"
	ch := gitlab.Changes{Files: []gitlab.FileChange{{NewPath: "c.go", Diff: diff}}}
	got := addedLines(ch)
	if want := map[int]bool{5: true}; !equalLineSet(got["c.go"], want) {
		t.Fatalf("c.go: got %v, want %v", got["c.go"], want)
	}
}

func assertLineSet(t *testing.T, file string, got, want map[int]bool) {
	t.Helper()
	if !equalLineSet(got, want) {
		t.Fatalf("%s added lines = %v, want %v", file, got, want)
	}
}

func equalLineSet(got, want map[int]bool) bool {
	if len(got) != len(want) {
		return false
	}
	for k, v := range want {
		if got[k] != v {
			return false
		}
	}
	return true
}
