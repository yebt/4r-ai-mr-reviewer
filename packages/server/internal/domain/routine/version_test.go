package routine

import "testing"

func TestClassifyCommit(t *testing.T) {
	cases := []struct {
		name     string
		message  string
		wantFeat bool
		wantFix  bool
	}{
		{"feat", "feat: x", true, false},
		{"fix with scope", "fix(api): y", false, true},
		{"feat breaking", "feat!: z", true, false},
		{"feat scope breaking", "feat(core)!: z", true, false},
		{"upper fix", "FIX: upper", false, true},
		{"mixed case feat", "FeAt: casing", true, false},
		{"chore no match", "chore: nope", false, false},
		{"fixup must not match", "fixup something", false, false},
		{"feature must not match", "feature: nope", false, false},
		{"multiline only subject", "feat: add thing\n\nfix: this body line is ignored", true, false},
		{"multiline fix body ignored", "chore: cleanup\n\nfeat: not counted", false, false},
		{"empty", "", false, false},
		{"leading space trimmed", "  fix: trimmed", false, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			gotFeat, gotFix := ClassifyCommit(tc.message)
			if gotFeat != tc.wantFeat || gotFix != tc.wantFix {
				t.Fatalf("ClassifyCommit(%q) = (feat=%v, fix=%v), want (feat=%v, fix=%v)",
					tc.message, gotFeat, gotFix, tc.wantFeat, tc.wantFix)
			}
		})
	}
}

func TestNextRelease(t *testing.T) {
	cases := []struct {
		name       string
		lastTag    string
		subjects   []string
		mode       BumpMode
		wantNext   string
		wantCounts CommitCounts
	}{
		{
			// The user's exact example: 3 feats raise minor 4->7; 2 fixes after the
			// last feat set the patch to 2. Total counts are {Feat:3, Fix:6}.
			name:    "canonical minor example",
			lastTag: "v1.4.2",
			subjects: []string{
				"feat: a", "fix: b", "fix: c", "fix: d",
				"feat: e", "fix: f", "feat: g", "fix: h", "fix: i",
			},
			mode:       BumpMinor,
			wantNext:   "v1.7.2",
			wantCounts: CommitCounts{Feat: 3, Fix: 6},
		},
		{
			name:       "minor no feats only fixes",
			lastTag:    "v2.5.3",
			subjects:   []string{"fix: a", "fix: b", "fix: c", "fix: d"},
			mode:       BumpMinor,
			wantNext:   "v2.5.7",
			wantCounts: CommitCounts{Feat: 0, Fix: 4},
		},
		{
			name:       "empty lastTag minor no prefix",
			lastTag:    "",
			subjects:   []string{"feat: a", "fix: b", "fix: c"},
			mode:       BumpMinor,
			wantNext:   "0.1.2",
			wantCounts: CommitCounts{Feat: 1, Fix: 2},
		},
		{
			name:       "major forced ignores commits",
			lastTag:    "v1.4.2",
			subjects:   []string{"feat: a", "fix: b"},
			mode:       BumpMajor,
			wantNext:   "v2.0.0",
			wantCounts: CommitCounts{Feat: 1, Fix: 1},
		},
		{
			name:    "patch ignores feats counts all fixes",
			lastTag: "v1.4.2",
			subjects: []string{
				"feat: a", "feat: b", "feat: c",
				"fix: d", "fix: e", "fix: f", "fix: g", "fix: h",
			},
			mode:       BumpPatch,
			wantNext:   "v1.4.7",
			wantCounts: CommitCounts{Feat: 3, Fix: 5},
		},
		{
			name:       "no v prefix preserved",
			lastTag:    "3.2.1",
			subjects:   []string{"feat: a"},
			mode:       BumpMinor,
			wantNext:   "3.3.0",
			wantCounts: CommitCounts{Feat: 1, Fix: 0},
		},
		{
			name:       "v prefix preserved with fixes only",
			lastTag:    "v0.9.0",
			subjects:   []string{"fix: a"},
			mode:       BumpMinor,
			wantNext:   "v0.9.1",
			wantCounts: CommitCounts{Feat: 0, Fix: 1},
		},
		{
			name:       "prerelease suffix ignored for base numbers",
			lastTag:    "v1.4.2-dev",
			subjects:   []string{"feat: a"},
			mode:       BumpMinor,
			wantNext:   "v1.5.0",
			wantCounts: CommitCounts{Feat: 1, Fix: 0},
		},
		{
			name:       "non-semver lastTag treated as zero base",
			lastTag:    "not-a-version",
			subjects:   []string{"feat: a", "fix: b"},
			mode:       BumpMinor,
			wantNext:   "0.1.1",
			wantCounts: CommitCounts{Feat: 1, Fix: 1},
		},
		{
			name:       "no commits minor is a no-op bump",
			lastTag:    "v1.4.2",
			subjects:   nil,
			mode:       BumpMinor,
			wantNext:   "v1.4.2",
			wantCounts: CommitCounts{Feat: 0, Fix: 0},
		},
		{
			// Non-conventional subjects are neither feat nor fix and do not move
			// the version.
			name:       "non-conventional commits ignored",
			lastTag:    "v1.4.2",
			subjects:   []string{"chore: x", "docs: y", "fixup z"},
			mode:       BumpMinor,
			wantNext:   "v1.4.2",
			wantCounts: CommitCounts{Feat: 0, Fix: 0},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			gotNext, gotCounts, err := NextRelease(tc.lastTag, tc.subjects, tc.mode)
			if err != nil {
				t.Fatalf("NextRelease: unexpected error %v", err)
			}
			if gotNext != tc.wantNext {
				t.Errorf("next = %q, want %q", gotNext, tc.wantNext)
			}
			if gotCounts != tc.wantCounts {
				t.Errorf("counts = %+v, want %+v", gotCounts, tc.wantCounts)
			}
		})
	}
}

func TestNextReleaseInvalidMode(t *testing.T) {
	_, _, err := NextRelease("v1.0.0", []string{"feat: a"}, BumpMode("bogus"))
	if err == nil {
		t.Fatal("expected error for invalid mode, got nil")
	}
}
