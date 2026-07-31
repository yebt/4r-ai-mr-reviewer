package routine

import "testing"

func TestNextTag(t *testing.T) {
	tests := []struct {
		name       string
		existing   []string
		bump       string
		prerelease string
		want       string
	}{
		{name: "no tags, patch", existing: nil, bump: "patch", want: "v0.0.1"},
		{name: "no tags, minor", existing: nil, bump: "minor", want: "v0.1.0"},
		{name: "no tags, major", existing: nil, bump: "major", want: "v1.0.0"},
		{name: "v-prefix by default", existing: []string{}, bump: "patch", want: "v0.0.1"},

		{name: "patch bump", existing: []string{"1.2.3"}, bump: "patch", want: "1.2.4"},
		{name: "minor bump resets patch", existing: []string{"1.2.3"}, bump: "minor", want: "1.3.0"},
		{name: "major bump resets minor and patch", existing: []string{"1.2.3"}, bump: "major", want: "2.0.0"},

		{name: "v-prefix preserved", existing: []string{"v1.2.3"}, bump: "patch", want: "v1.2.4"},
		{name: "v-prefix preserved on major", existing: []string{"v0.9.9"}, bump: "major", want: "v1.0.0"},

		{name: "prerelease suffix appended", existing: []string{"1.2.3"}, bump: "minor", prerelease: "development", want: "1.3.0-development"},
		{name: "prerelease with v-prefix", existing: []string{"v2.0.0"}, bump: "patch", prerelease: "development", want: "v2.0.1-development"},

		{
			name:     "non-semver tags are ignored",
			existing: []string{"latest", "release-candidate", "v1.0.0", "nightly"},
			bump:     "patch",
			want:     "v1.0.1",
		},
		{
			name:     "highest wins across many",
			existing: []string{"1.0.0", "1.2.0", "1.1.9", "0.9.0"},
			bump:     "patch",
			want:     "1.2.1",
		},
		{
			name:     "prerelease does not beat its release core",
			existing: []string{"1.2.3", "1.2.3-rc1"},
			bump:     "patch",
			want:     "1.2.4",
		},
		{
			name:     "release core outranks prerelease regardless of order",
			existing: []string{"1.2.3-rc1", "1.2.3"},
			bump:     "patch",
			want:     "1.2.4",
		},
		{
			name:     "prerelease of a higher core still counts as highest",
			existing: []string{"1.2.3", "1.2.4-rc1"},
			bump:     "patch",
			want:     "1.2.5",
		},
		{
			// An out-of-int-range numeric component makes the tag unrepresentable;
			// it must be ignored (not misparsed). Paired with a valid tag, the valid
			// one must drive the bump — if the overflow tag were misparsed as a huge
			// major it would dominate, and if misparsed as 0 it would be lower.
			name:     "overflow numeric component is ignored, not misparsed",
			existing: []string{"v1.2.3", "v999999999999999999999999999999.0.0"},
			bump:     "patch",
			want:     "v1.2.4",
		},
		{
			name:     "overflow tag ignored while a real tag still wins",
			existing: []string{"1.2.3", "999999999999999999999999999999.0.0"},
			bump:     "patch",
			want:     "1.2.4",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NextTag(tt.existing, tt.bump, tt.prerelease)
			if err != nil {
				t.Fatalf("NextTag: unexpected error: %v", err)
			}
			if got != tt.want {
				t.Fatalf("NextTag(%v, %q, %q) = %q, want %q", tt.existing, tt.bump, tt.prerelease, got, tt.want)
			}
		})
	}
}

func TestNextTagInvalidBump(t *testing.T) {
	if _, err := NextTag([]string{"1.0.0"}, "nope", ""); err == nil {
		t.Fatal("expected an error for an invalid bump")
	}
}
