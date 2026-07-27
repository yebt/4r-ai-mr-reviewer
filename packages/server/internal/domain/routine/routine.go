// Package routine models merge-request routines and the preflight that reports,
// ahead of time, which routine actions a repo's token and access level permit.
//
// The preflight is fail-closed and three-state. A check is "ok" only when every
// fact it depends on was read AND satisfies the requirement; "fail" when a fact
// was read and does not satisfy it; and "unknown" when a fact could not be read,
// so the check cannot be proven safe. A missing scope or an insufficient (but
// definitively read) access level is always a hard "fail" — never masked by an
// unrelated unknown. This avoids the false green a fail-open preflight would give
// when it cannot verify token scope or branch/tag protection.
package routine

import "fmt"

// Capability names a distinct write action a routine may perform on a merge
// request or repository.
type Capability string

const (
	CapComment    Capability = "comment"
	CapAwardEmoji Capability = "award_emoji"
	CapCreateTag  Capability = "create_tag"
	CapCreateMR   Capability = "create_mr"
	CapMergeMR    Capability = "merge_mr"
)

// CheckStatus is the three-state verdict for a capability.
type CheckStatus string

const (
	// StatusOK means every dependent fact was read and the capability is permitted.
	StatusOK CheckStatus = "ok"
	// StatusFail means a fact was read and the capability is definitively denied.
	StatusFail CheckStatus = "fail"
	// StatusUnknown means a required fact could not be read, so the capability
	// cannot be proven available; treat it as not-yet-verified, not permitted.
	StatusUnknown CheckStatus = "unknown"
)

// Check is the verdict for a single capability: its three-state status and a
// short human-readable reason.
type Check struct {
	Capability Capability
	Label      string
	Status     CheckStatus
	Detail     string
}

// Preflight is the computed report for a repo: the token scopes and access level
// observed, the default branch, and a Check per capability.
type Preflight struct {
	TokenScopes     []string
	ScopesKnown     bool // false when the token-self lookup failed (scopes unknown)
	AccessLevel     int
	AccessLevelName string
	DefaultBranch   string
	Checks          []Check
}

// PreflightInput is the raw, network-free input to Evaluate. It carries only the
// facts gathered from GitLab so the decision logic can be unit-tested in isolation.
type PreflightInput struct {
	Scopes                []string
	ScopesKnown           bool // false when token scopes could not be read
	AccessLevel           int
	DefaultBranch         string
	ProtectedBranches     map[string]int // branch name -> highest required merge access level (0 if none)
	BranchProtectionKnown bool           // false when the protected-branches lookup failed
	ProtectedTagsMinLevel int            // highest create access level among protected tag rules; 0 if no protected tags
	TagProtectionKnown    bool           // false when the protected-tags lookup failed
}

// GitLab member access levels.
const (
	levelNone       = 0
	levelGuest      = 10
	levelReporter   = 20
	levelDeveloper  = 30
	levelMaintainer = 40
	levelOwner      = 50
)

// accessLevelName maps a numeric GitLab access level to its role name.
func accessLevelName(level int) string {
	switch level {
	case levelNone:
		return "None"
	case levelGuest:
		return "Guest"
	case levelReporter:
		return "Reporter"
	case levelDeveloper:
		return "Developer"
	case levelMaintainer:
		return "Maintainer"
	case levelOwner:
		return "Owner"
	default:
		return "None"
	}
}

// hasScope reports whether scopes contains the target scope.
func hasScope(scopes []string, target string) bool {
	for _, s := range scopes {
		if s == target {
			return true
		}
	}
	return false
}

// max returns the larger of a and b.
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// scopeState is the three-state verdict for the token's "api" scope, derived once
// and shared by every write check.
type scopeState int

const (
	scopeOK      scopeState = iota // scopes were read and include "api"
	scopeFail                      // scopes were read and do NOT include "api"
	scopeUnknown                   // scopes could not be read
)

// Evaluate computes the preflight from raw inputs. It is pure (no network, no
// context) so the permission rules can be tested directly. It is fail-closed:
//
//   - Scope is derived once. Unknown scopes never grant "ok" — at best a check is
//     "unknown". A read scope set that lacks "api" is a hard "fail".
//   - Each write capability has a required access level. A definitively read
//     access level below the requirement is a hard "fail", even when the
//     requirement is only a lower bound (protection unreadable).
//   - When access qualifies but a dependent fact is unverified — scopes unknown,
//     or branch/tag protection unreadable — the check is "unknown", not "ok".
//   - comment / award_emoji require Reporter (20); create_mr requires Developer
//     (30); merge_mr requires max(Developer, the default branch's protection
//     level); create_tag requires max(Developer, the highest protected tag level).
func Evaluate(in PreflightInput) Preflight {
	// Derive the scope state once.
	scope := scopeUnknown
	if in.ScopesKnown {
		if hasScope(in.Scopes, "api") {
			scope = scopeOK
		} else {
			scope = scopeFail
		}
	}

	// writeCheck combines the shared scope state with a capability's access
	// requirement into a three-state Check. protectedNote describes a protection
	// rule that raised (or could raise) the requirement; it is appended to a fail
	// reason and, when the check passes, reported as the qualifying note.
	writeCheck := func(cap Capability, label string, access, required int, requiredKnown bool, protectedNote string) Check {
		switch {
		case scope == scopeFail:
			return Check{Capability: cap, Label: label, Status: StatusFail, Detail: "token is missing the 'api' scope"}
		case access < required:
			detail := fmt.Sprintf("your access level %s(%d) is below the required %s(%d)",
				accessLevelName(access), access, accessLevelName(required), required)
			if protectedNote != "" {
				detail += "; " + protectedNote
			}
			return Check{Capability: cap, Label: label, Status: StatusFail, Detail: detail}
		case scope == scopeUnknown:
			return Check{Capability: cap, Label: label, Status: StatusUnknown,
				Detail: "token scopes could not be read, so 'api' scope is unverified"}
		case !requiredKnown:
			return Check{Capability: cap, Label: label, Status: StatusUnknown,
				Detail: "branch/tag protection could not be read; you meet the base level but protected rules may require a higher one"}
		default:
			detail := ""
			if protectedNote != "" {
				detail = protectedNote + "; your access level qualifies"
			}
			return Check{Capability: cap, Label: label, Status: StatusOK, Detail: detail}
		}
	}

	// merge_mr: the default branch's protection may raise the required level.
	mergeRequired := levelDeveloper
	mergeNote := ""
	if lvl := in.ProtectedBranches[in.DefaultBranch]; lvl > 0 {
		mergeRequired = max(levelDeveloper, lvl)
		mergeNote = fmt.Sprintf("default branch %q is protected", in.DefaultBranch)
	}

	// create_tag: protected tags may raise the required level.
	tagRequired := levelDeveloper
	tagNote := ""
	if in.ProtectedTagsMinLevel > 0 {
		tagRequired = max(levelDeveloper, in.ProtectedTagsMinLevel)
		tagNote = "tags are protected"
	}

	checks := []Check{
		writeCheck(CapComment, "Comment on MR", in.AccessLevel, levelReporter, true, ""),
		writeCheck(CapAwardEmoji, "Award emoji on MR", in.AccessLevel, levelReporter, true, ""),
		writeCheck(CapCreateMR, "Create merge request", in.AccessLevel, levelDeveloper, true, ""),
		writeCheck(CapMergeMR, "Merge merge request", in.AccessLevel, mergeRequired, in.BranchProtectionKnown, mergeNote),
		writeCheck(CapCreateTag, "Create tag", in.AccessLevel, tagRequired, in.TagProtectionKnown, tagNote),
	}

	return Preflight{
		TokenScopes:     in.Scopes,
		ScopesKnown:     in.ScopesKnown,
		AccessLevel:     in.AccessLevel,
		AccessLevelName: accessLevelName(in.AccessLevel),
		DefaultBranch:   in.DefaultBranch,
		Checks:          checks,
	}
}
