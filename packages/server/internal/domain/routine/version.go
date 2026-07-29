package routine

import (
	"fmt"
	"regexp"
	"strings"
)

// BumpMode selects how NextRelease derives the next version from commits.
type BumpMode string

const (
	// BumpMajor forces a major release: (major+1).0.0 regardless of commits.
	BumpMajor BumpMode = "major"
	// BumpMinor derives the bump from conventional commits: feats raise the minor,
	// trailing fixes raise the patch. This is the primary mode.
	BumpMinor BumpMode = "minor"
	// BumpPatch treats every fix as a patch increment and ignores feats.
	BumpPatch BumpMode = "patch"
)

// CommitCounts is the TOTAL number of feat and fix commits found, for display
// ("Commits found"). It is independent of how the version is actually bumped.
type CommitCounts struct {
	Feat int
	Fix  int
}

// conventionalCommit matches a conventional-commit subject: a "feat" or "fix"
// type token (case-insensitive), an optional "(scope)", an optional "!", then a
// ":". The type token boundary (the group and/or "!" and/or ":") prevents
// "fixup ..." from matching "fix".
var conventionalCommit = regexp.MustCompile(`^(?i:(feat|fix))(?:\([^)]*\))?!?:`)

// ClassifyCommit reports whether a commit message is a conventional-commit feat
// or fix. It recognizes "feat"/"fix" optionally with a scope and/or "!" then ":"
// — e.g. "feat: x", "fix(scope): y", "feat!: z" (case-insensitive on the type
// token). Only the FIRST line (subject) is inspected; anything else is neither.
func ClassifyCommit(message string) (isFeat, isFix bool) {
	subject := message
	if i := strings.IndexByte(subject, '\n'); i >= 0 {
		subject = subject[:i]
	}
	subject = strings.TrimSpace(subject)

	m := conventionalCommit.FindStringSubmatch(subject)
	if m == nil {
		return false, false
	}
	switch strings.ToLower(m[1]) {
	case "feat":
		return true, false
	case "fix":
		return false, true
	default:
		return false, false
	}
}

// NextRelease computes the next version from the last tag and the ordered list of
// commit subjects (OLDEST first), applying the configured bump mode. It also
// returns the TOTAL feat/fix counts across all commits (for display).
//
// For BumpMinor (the primary case):
//   - minorIncrements = number of feat commits
//   - patchIncrements = number of fix commits after the LAST feat (or all fixes
//     when there are no feats)
//   - if minorIncrements > 0: next = major.(minor+minorIncrements).patchIncrements
//   - else:                   next = major.minor.(patch+patchIncrements)
//
// For BumpMajor: next = (major+1).0.0 (forced major release).
// For BumpPatch: next = major.minor.(patch+totalFixCount) (feats ignored).
//
// lastTag parsing: optional leading "v" (preserved in output), then
// MAJOR.MINOR.PATCH with an optional "-suffix" (ignored for the base numbers).
// If lastTag == "" the base is 0.0.0 and NO "v" prefix is emitted. A non-semver
// lastTag is treated as an empty base (0.0.0) without an error.
// It returns an error only for an invalid mode.
func NextRelease(lastTag string, commitSubjects []string, mode BumpMode) (next string, counts CommitCounts, err error) {
	switch mode {
	case BumpMajor, BumpMinor, BumpPatch:
	default:
		return "", CommitCounts{}, fmt.Errorf("routine: invalid mode %q (want major, minor or patch)", mode)
	}

	base, ok := parseTag(lastTag)
	hasV := ok && base.hasV // no "v" prefix for an empty/non-semver base

	// Total counts and the index of the last feat, in a single pass.
	lastFeatIdx := -1
	for i, subject := range commitSubjects {
		isFeat, isFix := ClassifyCommit(subject)
		if isFeat {
			counts.Feat++
			lastFeatIdx = i
		}
		if isFix {
			counts.Fix++
		}
	}

	major, minor, patch := base.major, base.minor, base.patch
	switch mode {
	case BumpMajor:
		major, minor, patch = major+1, 0, 0
	case BumpPatch:
		patch += counts.Fix
	case BumpMinor:
		if counts.Feat > 0 {
			// Patch resets to the number of fixes that land after the last feat.
			patchAfterLastFeat := 0
			for _, subject := range commitSubjects[lastFeatIdx+1:] {
				if _, isFix := ClassifyCommit(subject); isFix {
					patchAfterLastFeat++
				}
			}
			minor += counts.Feat
			patch = patchAfterLastFeat
		} else {
			patch += counts.Fix
		}
	}

	var b strings.Builder
	if hasV {
		b.WriteByte('v')
	}
	fmt.Fprintf(&b, "%d.%d.%d", major, minor, patch)
	return b.String(), counts, nil
}
