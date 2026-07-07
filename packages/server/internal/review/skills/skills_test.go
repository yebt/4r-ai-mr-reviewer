package skills

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadBuiltin(t *testing.T) {
	set, err := Load("")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if !strings.Contains(set.Risk, "R1") || !strings.Contains(set.Resilience, "R4") {
		t.Fatal("embedded rules missing expected content")
	}
	if !strings.Contains(set.Combined(), "R2") {
		t.Fatal("Combined should include all four sets")
	}
}

func TestOverrideWins(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "r-risk.md"), []byte("CUSTOM RISK RULES"), 0o644); err != nil {
		t.Fatal(err)
	}

	set, err := Load(dir)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if set.Risk != "CUSTOM RISK RULES" {
		t.Fatalf("override ignored: %q", set.Risk)
	}
	// Rules without an override file still come from the embedded copy.
	if !strings.Contains(set.Readability, "R2") {
		t.Fatal("non-overridden rule should fall back to embedded")
	}
}
