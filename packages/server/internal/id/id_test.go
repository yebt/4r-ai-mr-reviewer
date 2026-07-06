package id

import "testing"

func TestNewLength(t *testing.T) {
	if got := len(New()); got != 32 {
		t.Fatalf("id length = %d, want 32", got)
	}
}

func TestNewUniqueness(t *testing.T) {
	seen := make(map[string]struct{}, 1000)
	for range 1000 {
		v := New()
		if _, dup := seen[v]; dup {
			t.Fatalf("duplicate id generated: %s", v)
		}
		seen[v] = struct{}{}
	}
}
