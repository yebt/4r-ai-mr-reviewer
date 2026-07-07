// Package skills loads the 4R review rule sets. The canonical versions are
// embedded in the binary; an optional override directory lets the user iterate
// on the rules at runtime without recompiling.
package skills

import (
	"embed"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

//go:embed builtin/*.md
var builtin embed.FS

// Set holds the four rule texts.
type Set struct {
	Risk        string
	Readability string
	Reliability string
	Resilience  string
}

var files = []struct {
	name string
	get  func(*Set) *string
}{
	{"r-risk.md", func(s *Set) *string { return &s.Risk }},
	{"r-readability.md", func(s *Set) *string { return &s.Readability }},
	{"r-reliability.md", func(s *Set) *string { return &s.Reliability }},
	{"r-resilience.md", func(s *Set) *string { return &s.Resilience }},
}

// Load returns the rule set. For each rule, a file of the same name in
// overrideDir (when overrideDir is non-empty and the file exists) wins over the
// embedded default.
func Load(overrideDir string) (Set, error) {
	var set Set
	for _, f := range files {
		text, err := loadOne(f.name, overrideDir)
		if err != nil {
			return Set{}, err
		}
		*f.get(&set) = text
	}
	return set, nil
}

func loadOne(name, overrideDir string) (string, error) {
	if overrideDir != "" {
		if b, err := os.ReadFile(filepath.Join(overrideDir, name)); err == nil {
			return string(b), nil
		}
		// A missing or unreadable override falls back to the embedded copy.
	}
	b, err := builtin.ReadFile("builtin/" + name)
	if err != nil {
		return "", fmt.Errorf("skills: load %s: %w", name, err)
	}
	return string(b), nil
}

// Combined renders all four rule sets into one block, in R1..R4 order.
func (s Set) Combined() string {
	return strings.Join([]string{s.Risk, s.Readability, s.Reliability, s.Resilience}, "\n\n")
}
