package notification

import "testing"

// TestReleaseFinishedIsAValidEvent guards the wiring the release routine relies
// on: release.finished must be a known event and appear in the Events list the
// UI subscribes rules to.
func TestReleaseFinishedIsAValidEvent(t *testing.T) {
	if !ValidEvent(EventReleaseFinished) {
		t.Fatalf("ValidEvent(%q) = false, want true", EventReleaseFinished)
	}

	found := false
	for _, ev := range Events {
		if ev == EventReleaseFinished {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("Events = %v, want it to include %q", Events, EventReleaseFinished)
	}
}
