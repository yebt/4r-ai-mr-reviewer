package routine

import (
	"strings"
	"testing"
)

// checkByCap finds the Check for a capability in a preflight result.
func checkByCap(t *testing.T, p Preflight, cap Capability) Check {
	t.Helper()
	for _, c := range p.Checks {
		if c.Capability == cap {
			return c
		}
	}
	t.Fatalf("no check for capability %q", cap)
	return Check{}
}

func TestEvaluate(t *testing.T) {
	tests := []struct {
		name       string
		in         PreflightInput
		wantStatus map[Capability]CheckStatus
		// wantDetail asserts a substring is present in a capability's Detail.
		wantDetail map[Capability]string
	}{
		{
			name: "api and Owner: everything ok",
			in: PreflightInput{
				Scopes: []string{"api"}, ScopesKnown: true,
				AccessLevel:   levelOwner,
				DefaultBranch: "main",
				// Protection was read (empty): merge/tag can be a definitive ok.
				BranchProtectionKnown: true, TagProtectionKnown: true,
			},
			wantStatus: map[Capability]CheckStatus{
				CapComment: StatusOK, CapAwardEmoji: StatusOK, CapCreateMR: StatusOK,
				CapMergeMR: StatusOK, CapCreateTag: StatusOK,
			},
		},
		{
			name: "missing api: every write fails",
			in: PreflightInput{
				Scopes: []string{"read_api"}, ScopesKnown: true,
				AccessLevel:           levelOwner,
				DefaultBranch:         "main",
				BranchProtectionKnown: true, TagProtectionKnown: true,
			},
			wantStatus: map[Capability]CheckStatus{
				CapComment: StatusFail, CapAwardEmoji: StatusFail, CapCreateMR: StatusFail,
				CapMergeMR: StatusFail, CapCreateTag: StatusFail,
			},
			wantDetail: map[Capability]string{
				CapComment: "missing the 'api' scope",
			},
		},
		{
			name: "Reporter: comment and emoji ok, the rest fail",
			in: PreflightInput{
				Scopes: []string{"api"}, ScopesKnown: true,
				AccessLevel:           levelReporter,
				DefaultBranch:         "main",
				BranchProtectionKnown: true, TagProtectionKnown: true,
			},
			wantStatus: map[Capability]CheckStatus{
				CapComment: StatusOK, CapAwardEmoji: StatusOK, CapCreateMR: StatusFail,
				CapMergeMR: StatusFail, CapCreateTag: StatusFail,
			},
			wantDetail: map[Capability]string{
				CapCreateMR: "below the required Developer(30)",
			},
		},
		{
			name: "Developer on Maintainer-protected main: merge fails, create_mr ok",
			in: PreflightInput{
				Scopes: []string{"api"}, ScopesKnown: true,
				AccessLevel:           levelDeveloper,
				DefaultBranch:         "main",
				ProtectedBranches:     map[string]int{"main": levelMaintainer},
				BranchProtectionKnown: true, TagProtectionKnown: true,
			},
			wantStatus: map[Capability]CheckStatus{
				CapCreateMR: StatusOK, CapMergeMR: StatusFail, CapCreateTag: StatusOK,
			},
			wantDetail: map[Capability]string{
				CapMergeMR: "below the required Maintainer(40)",
			},
		},
		{
			name: "boundary: access equals required protection level (merge ok, proves < not <=)",
			in: PreflightInput{
				Scopes: []string{"api"}, ScopesKnown: true,
				AccessLevel:           levelMaintainer,
				DefaultBranch:         "main",
				ProtectedBranches:     map[string]int{"main": levelMaintainer},
				BranchProtectionKnown: true, TagProtectionKnown: true,
			},
			wantStatus: map[Capability]CheckStatus{
				CapMergeMR: StatusOK,
			},
			wantDetail: map[Capability]string{
				CapMergeMR: "your access level qualifies",
			},
		},
		{
			name: "protection on a non-default branch is ignored",
			in: PreflightInput{
				Scopes: []string{"api"}, ScopesKnown: true,
				AccessLevel:           levelDeveloper,
				DefaultBranch:         "main",
				ProtectedBranches:     map[string]int{"develop": levelMaintainer},
				BranchProtectionKnown: true, TagProtectionKnown: true,
			},
			wantStatus: map[Capability]CheckStatus{
				CapMergeMR: StatusOK,
			},
		},
		{
			name: "empty default branch with protection read: merge ok at base level",
			in: PreflightInput{
				Scopes: []string{"api"}, ScopesKnown: true,
				AccessLevel:           levelDeveloper,
				DefaultBranch:         "",
				BranchProtectionKnown: true, TagProtectionKnown: true,
			},
			wantStatus: map[Capability]CheckStatus{
				CapMergeMR: StatusOK,
			},
		},
		{
			name: "no access (None): every write fails",
			in: PreflightInput{
				Scopes: []string{"api"}, ScopesKnown: true,
				AccessLevel:           levelNone,
				DefaultBranch:         "main",
				BranchProtectionKnown: true, TagProtectionKnown: true,
			},
			wantStatus: map[Capability]CheckStatus{
				CapComment: StatusFail, CapAwardEmoji: StatusFail, CapCreateMR: StatusFail,
				CapMergeMR: StatusFail, CapCreateTag: StatusFail,
			},
			wantDetail: map[Capability]string{
				CapComment: "your access level None(0)",
			},
		},
		{
			name: "scopes unknown, Owner access: every write is unknown (scope unverified)",
			in: PreflightInput{
				Scopes: nil, ScopesKnown: false,
				AccessLevel:           levelOwner,
				DefaultBranch:         "main",
				BranchProtectionKnown: true, TagProtectionKnown: true,
			},
			wantStatus: map[Capability]CheckStatus{
				CapComment: StatusUnknown, CapAwardEmoji: StatusUnknown, CapCreateMR: StatusUnknown,
				CapMergeMR: StatusUnknown, CapCreateTag: StatusUnknown,
			},
			wantDetail: map[Capability]string{
				CapComment: "unverified",
				CapMergeMR: "unverified",
			},
		},
		{
			name: "scopes unknown but access too low: still a definitive fail",
			in: PreflightInput{
				Scopes: nil, ScopesKnown: false,
				AccessLevel:           levelNone,
				DefaultBranch:         "main",
				BranchProtectionKnown: true, TagProtectionKnown: true,
			},
			wantStatus: map[Capability]CheckStatus{
				CapComment: StatusFail, CapAwardEmoji: StatusFail, CapCreateMR: StatusFail,
				CapMergeMR: StatusFail, CapCreateTag: StatusFail,
			},
		},
		{
			name: "protection unreadable: merge/tag unknown, create_mr still ok",
			in: PreflightInput{
				Scopes: []string{"api"}, ScopesKnown: true,
				AccessLevel:   levelDeveloper,
				DefaultBranch: "main",
				// Both protection lookups failed → flags stay false.
				BranchProtectionKnown: false, TagProtectionKnown: false,
			},
			wantStatus: map[Capability]CheckStatus{
				CapComment: StatusOK, CapAwardEmoji: StatusOK, CapCreateMR: StatusOK,
				CapMergeMR: StatusUnknown, CapCreateTag: StatusUnknown,
			},
			wantDetail: map[Capability]string{
				CapMergeMR:   "protection could not be read",
				CapCreateTag: "protection could not be read",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Evaluate(tt.in)
			if len(got.Checks) != 5 {
				t.Fatalf("expected 5 checks, got %d", len(got.Checks))
			}
			for cap, want := range tt.wantStatus {
				c := checkByCap(t, got, cap)
				if c.Status != want {
					t.Errorf("capability %q: Status = %q, want %q (detail %q)", cap, c.Status, want, c.Detail)
				}
				// A fail or unknown must always carry a reason.
				if c.Status != StatusOK && c.Detail == "" {
					t.Errorf("capability %q: %q status has empty detail", cap, c.Status)
				}
			}
			for cap, want := range tt.wantDetail {
				c := checkByCap(t, got, cap)
				if !strings.Contains(c.Detail, want) {
					t.Errorf("capability %q: detail = %q, want to contain %q", cap, c.Detail, want)
				}
			}
		})
	}
}

// TestEvaluateChecksOrder pins the emitted check order:
// comment, award_emoji, create_mr, merge_mr, create_tag.
func TestEvaluateChecksOrder(t *testing.T) {
	got := Evaluate(PreflightInput{
		Scopes: []string{"api"}, ScopesKnown: true,
		AccessLevel: levelOwner, DefaultBranch: "main",
		BranchProtectionKnown: true, TagProtectionKnown: true,
	})
	want := []Capability{CapComment, CapAwardEmoji, CapCreateMR, CapMergeMR, CapCreateTag}
	if len(got.Checks) != len(want) {
		t.Fatalf("got %d checks, want %d", len(got.Checks), len(want))
	}
	for i, c := range got.Checks {
		if c.Capability != want[i] {
			t.Errorf("check[%d] = %q, want %q", i, c.Capability, want[i])
		}
	}
}

func TestAccessLevelName(t *testing.T) {
	cases := map[int]string{
		0: "None", 10: "Guest", 20: "Reporter", 30: "Developer",
		40: "Maintainer", 50: "Owner", 99: "None",
	}
	for level, want := range cases {
		if got := accessLevelName(level); got != want {
			t.Errorf("accessLevelName(%d) = %q, want %q", level, got, want)
		}
	}
}
