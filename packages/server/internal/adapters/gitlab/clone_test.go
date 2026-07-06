package gitlab

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// runGit runs a git command in dir, failing the test on error.
func runGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
}

// makeSourceRepo builds a small local repo on branch main with one commit.
func makeSourceRepo(t *testing.T) string {
	t.Helper()
	src := t.TempDir()
	runGit(t, src, "init", "-b", "main")
	if err := os.WriteFile(filepath.Join(src, "README.md"), []byte("hello"), 0o644); err != nil {
		t.Fatal(err)
	}
	runGit(t, src, "add", ".")
	runGit(t, src, "-c", "user.email=t@t", "-c", "user.name=t", "commit", "-m", "init")
	return src
}

func TestCloneLocalRepo(t *testing.T) {
	src := makeSourceRepo(t)
	work := t.TempDir()

	c := NewCloner("") // file:// needs no token
	dest, err := c.Clone(context.Background(), "file://"+src, "main", work)
	if err != nil {
		t.Fatalf("Clone: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dest, "README.md")); err != nil {
		t.Fatalf("expected cloned file: %v", err)
	}
}

func TestCloneUnknownRefFails(t *testing.T) {
	src := makeSourceRepo(t)
	work := t.TempDir()

	c := NewCloner("")
	if _, err := c.Clone(context.Background(), "file://"+src, "does-not-exist", work); err == nil {
		t.Fatal("expected error cloning a missing ref")
	}
}

func TestCloneEmptyURL(t *testing.T) {
	c := NewCloner("")
	if _, err := c.Clone(context.Background(), "", "main", t.TempDir()); err == nil {
		t.Fatal("expected error for empty URL")
	}
}
