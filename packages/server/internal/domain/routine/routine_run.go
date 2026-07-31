package routine

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// Kind names a routine's fixed action sequence.
type Kind string

// KindApproveAndTag reacts to an MR with emojis, comments, and creates the next
// version tag.
const KindApproveAndTag Kind = "approve_and_tag"

// KindRelease verifies an MR, reacts, approves, computes the next version, pauses
// for a merge-confirmation decision, then merges (or waits) and tags.
const KindRelease Kind = "release"

// RunStatus is the lifecycle state of a routine run.
type RunStatus string

const (
	// RunPending is queued, waiting for the worker to claim it.
	RunPending RunStatus = "pending"
	// RunRunning is actively executing steps.
	RunRunning RunStatus = "running"
	// RunBlocked paused on a failed step and can be resumed once the cause is
	// fixed; already-completed steps are skipped on resume.
	RunBlocked RunStatus = "blocked"
	// RunAwaitingConfirmation paused on an interactive gate, waiting for an
	// out-of-band decision (Confirm) to flip it back to pending. Unlike blocked,
	// this is not an error state and is resolved by Confirm, not Resume.
	RunAwaitingConfirmation RunStatus = "awaiting_confirmation"
	// RunDone completed every step.
	RunDone RunStatus = "done"
	// RunCancelled was aborted by an out-of-band Cancel. It is terminal (like
	// RunDone): the worker never claims it and it can no longer transition, but
	// unlike a running run it is deletable.
	RunCancelled RunStatus = "cancelled"
)

// StepStatus is the state of a single step within a run's ledger.
type StepStatus string

const (
	StepPending StepStatus = "pending"
	StepRunning StepStatus = "running"
	StepDone    StepStatus = "done"
	StepFailed  StepStatus = "failed"
	StepSkipped StepStatus = "skipped"
)

// Step is one checkpointed action in a run. Because routine actions are
// irreversible, a step's Status is persisted after each transition so a resume
// can skip the ones already done.
type Step struct {
	Name      string
	Status    StepStatus
	Detail    string
	UpdatedAt time.Time
}

// Run is a resumable, checkpointed sequence of GitLab actions.
type Run struct {
	ID     string
	Kind   Kind
	RepoID string
	MRIID  int
	Status RunStatus
	// Params is the immutable, kind-specific input chosen at creation.
	Params json.RawMessage
	// State is the mutable, kind-specific accumulator persisted across steps.
	State json.RawMessage
	// Steps is the ordered ledger; each entry checkpoints one action.
	Steps     []Step
	LastError string
	CreatedAt time.Time
	UpdatedAt time.Time
}

// ErrRunNotFound is returned when a routine run does not exist.
var ErrRunNotFound = errors.New("routine: run not found")

// ErrRunFinalized is returned by Save when the run's PERSISTED status is already
// terminal (cancelled or done), so the conditional (compare-and-set) update
// matched no rows. It is the synchronization signal that a concurrent Cancel — or
// a run that already completed — has finalized the row: the caller must STOP and
// must not revive it. The DB row, not the in-memory copy, is the source of truth
// for whether a run may still transition.
var ErrRunFinalized = errors.New("routine: run is finalized")

// ErrNotResumable is returned when a run cannot be resumed (it is not blocked).
var ErrNotResumable = errors.New("routine: run is not resumable")

// ErrRunNotCancelable is returned when a run cannot be cancelled because it has
// already reached a terminal status (done or cancelled). The HTTP layer maps it
// to 409 Conflict.
var ErrRunNotCancelable = errors.New("routine: run is not cancelable")

// ErrRunActive is returned when a run cannot be deleted because it is actively
// executing (running); deleting it mid-flight would orphan in-progress work.
var ErrRunActive = errors.New("routine: run is active")

// ErrNotAwaitingConfirmation is returned when Confirm targets a run that is not
// paused on the confirmation gate.
var ErrNotAwaitingConfirmation = errors.New("routine: run is not awaiting confirmation")

// ErrInvalidConfirmationDecision is returned when Confirm is given a decision
// other than "merge" or "wait". It lets the HTTP layer map an invalid decision
// to 400 while routing unknown-run and unexpected errors through the normal
// not-found/500 fallback.
var ErrInvalidConfirmationDecision = errors.New("routine: invalid confirmation decision (want merge or wait)")

// ErrDuplicateRun is returned when an active run (pending, running, or blocked)
// already exists for the same repo, merge request, and kind.
var ErrDuplicateRun = errors.New("routine: an active run already exists for this merge request")

// RunStore persists routine runs.
type RunStore interface {
	// Create inserts a new run (typically pending).
	Create(ctx context.Context, run Run) error
	// Get returns a run by id, or ErrRunNotFound if missing.
	Get(ctx context.Context, id string) (Run, error)
	// Delete removes a run by id, or ErrRunNotFound if missing.
	Delete(ctx context.Context, id string) error
	// ListByRepo returns a repo's runs, newest first.
	ListByRepo(ctx context.Context, repoID string) ([]Run, error)
	// ListRecent returns the most recent runs across all repos, newest first,
	// capped at limit.
	ListRecent(ctx context.Context, limit int) ([]Run, error)
	// Save persists status, steps, state, last_error and updated_at as a
	// compare-and-set: it never overwrites a row whose persisted status is already
	// terminal (cancelled or done). It returns ErrRunFinalized when the row is
	// terminal and ErrRunNotFound when the row is missing.
	Save(ctx context.Context, run Run) error
	// ClaimPending atomically flips the oldest pending run to running and returns
	// it. found is false when nothing was claimable.
	ClaimPending(ctx context.Context) (run Run, found bool, err error)
	// RequeueRunning resets running runs back to pending (startup crash
	// recovery); safe because steps are checkpointed.
	RequeueRunning(ctx context.Context) error
}

// semverTag matches MAJOR.MINOR.PATCH with an optional leading "v" and an
// optional "-prerelease" suffix. Anything else is not a version tag.
var semverTag = regexp.MustCompile(`^(v?)(\d+)\.(\d+)\.(\d+)(?:-(.+))?$`)

// parsedTag is a decoded semver tag.
type parsedTag struct {
	hasV                bool
	major, minor, patch int
	hasPre              bool
}

// parseTag decodes a semver tag, reporting false when it is not MAJOR.MINOR.PATCH
// (optionally v-prefixed, optionally with a -suffix).
func parseTag(tag string) (parsedTag, bool) {
	m := semverTag.FindStringSubmatch(strings.TrimSpace(tag))
	if m == nil {
		return parsedTag{}, false
	}
	// A numeric component out of int range is not a version we can reason about;
	// treat the whole tag as non-semver rather than silently misparsing it as 0.
	major, err := strconv.Atoi(m[2])
	if err != nil {
		return parsedTag{}, false
	}
	minor, err := strconv.Atoi(m[3])
	if err != nil {
		return parsedTag{}, false
	}
	patch, err := strconv.Atoi(m[4])
	if err != nil {
		return parsedTag{}, false
	}
	return parsedTag{
		hasV:   m[1] == "v",
		major:  major,
		minor:  minor,
		patch:  patch,
		hasPre: m[5] != "",
	}, true
}

// higher reports whether a ranks strictly above b. Cores are compared field by
// field; on an equal core a release (no prerelease) outranks a prerelease, so a
// prerelease never beats its own release core.
func (a parsedTag) higher(b parsedTag) bool {
	switch {
	case a.major != b.major:
		return a.major > b.major
	case a.minor != b.minor:
		return a.minor > b.minor
	case a.patch != b.patch:
		return a.patch > b.patch
	case a.hasPre != b.hasPre:
		return !a.hasPre
	default:
		return false
	}
}

// NextTag finds the highest semver tag among existing, applies bump
// ("major"|"minor"|"patch"), and appends a "-<prerelease>" suffix when
// prerelease != "". It preserves a leading "v" if the highest existing tag had
// one. If no semver tag exists it starts from 0.0.0. Tags that aren't
// MAJOR.MINOR.PATCH (optionally v-prefixed, optionally with a -suffix) are
// ignored. It returns an error on an invalid bump.
func NextTag(existing []string, bump, prerelease string) (string, error) {
	switch bump {
	case "major", "minor", "patch":
	default:
		return "", fmt.Errorf("routine: invalid bump %q (want major, minor or patch)", bump)
	}

	var best parsedTag
	found := false
	for _, tag := range existing {
		pt, ok := parseTag(tag)
		if !ok {
			continue
		}
		if !found || pt.higher(best) {
			best, found = pt, true
		}
	}

	major, minor, patch := best.major, best.minor, best.patch
	switch bump {
	case "major":
		major, minor, patch = major+1, 0, 0
	case "minor":
		minor, patch = minor+1, 0
	case "patch":
		patch++
	}

	var b strings.Builder
	// Default to a "v" prefix; when a base tag was found, follow its convention
	// (an existing non-"v" tag keeps producing non-"v" tags).
	if !found || best.hasV {
		b.WriteByte('v')
	}
	fmt.Fprintf(&b, "%d.%d.%d", major, minor, patch)
	if prerelease != "" {
		b.WriteByte('-')
		b.WriteString(prerelease)
	}
	return b.String(), nil
}
