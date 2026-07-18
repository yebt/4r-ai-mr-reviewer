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

func TestValidateRef(t *testing.T) {
	valid := []string{"main", "feature/login", "release-1.2.3", "v2", "a_b.c/d-e"}
	for _, ref := range valid {
		if err := validateRef(ref); err != nil {
			t.Errorf("validateRef(%q) = %v, want nil", ref, err)
		}
	}
	invalid := []string{"-foo", "--upload-pack=evil", "a b", "a;b", "a..b", "a$b", "a|b", "a\tb"}
	for _, ref := range invalid {
		if err := validateRef(ref); err == nil {
			t.Errorf("validateRef(%q) = nil, want error", ref)
		}
	}
}

func TestCloneRejectsInjectingRef(t *testing.T) {
	c := NewCloner("")
	if _, err := c.Clone(context.Background(), "https://example.com/x.git", "--upload-pack=evil", t.TempDir()); err == nil {
		t.Fatal("expected error for option-like ref")
	}
}

func TestIsAuthDenied(t *testing.T) {
	// The exact message GitLab returned when the token lacked read_repository.
	gitlabDenied := "remote: HTTP Basic: Access denied. If a password was provided " +
		"for Git authentication, the password was incorrect.\n" +
		"fatal: Authentication failed for 'https://gitlab.com/group/project.git/'"
	denied := []string{
		gitlabDenied,
		"fatal: Authentication failed for 'https://gitlab.com/x.git/'",
		"remote: HTTP Basic: Access denied.",
	}
	for _, s := range denied {
		if !isAuthDenied(s) {
			t.Errorf("isAuthDenied(%q) = false, want true", s)
		}
	}

	notDenied := []string{
		"fatal: Remote branch does-not-exist not found in upstream origin",
		"fatal: unable to access 'https://gitlab.com/x.git/': Could not resolve host",
		"",
	}
	for _, s := range notDenied {
		if isAuthDenied(s) {
			t.Errorf("isAuthDenied(%q) = true, want false", s)
		}
	}
}
