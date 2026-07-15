package reviews

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/webcloster-dev/ai-reviewer/internal/adapters/gitlab"
	"github.com/webcloster-dev/ai-reviewer/internal/domain/repo"
	"github.com/webcloster-dev/ai-reviewer/internal/domain/review"
)

// Selection chooses which findings to publish: All, or specific 0-based indices
// into the review's stored findings. IncludeSummary is a tri-state override for
// posting the summary note: nil defaults to posting only on the first publish,
// while a non-nil value forces (true) or suppresses (false) the summary.
//
// SummaryOverride and FindingOverrides carry humanized text that REPLACES the
// generated body as-is: a non-nil SummaryOverride is posted instead of
// formatSummary(rv), and FindingOverrides[i] (keyed by finding index) is posted
// instead of formatFinding(rv.Findings[i]). Overrides are self-contained
// comments in the user's voice, so no dimension/severity header is prepended.
type Selection struct {
	All              bool
	Indices          []int
	IncludeSummary   *bool
	SummaryOverride  *string
	FindingOverrides map[int]string
}

// Publish posts the review summary and the selected findings to the merge
// request. Findings with a file and line become inline discussions; the rest
// become general notes. Successfully posted findings are marked published.
func (s *Service) Publish(ctx context.Context, reviewID string, sel Selection) error {
	rv, err := s.reviews.Get(ctx, reviewID)
	if err != nil {
		return err
	}
	if rv.Status != review.StatusDone {
		return fmt.Errorf("reviews: cannot publish a review in status %q", rv.Status)
	}
	rp, err := s.repos.Get(ctx, rv.RepoID)
	if err != nil {
		return err
	}
	gl, projectID, _, err := s.gitlabFor(ctx, rp)
	if err != nil {
		return err
	}

	ch, err := gl.MergeRequestChanges(ctx, projectID, rv.MRIID)
	if err != nil {
		return err
	}
	refs := ch.DiffRefs
	// Added lines per file: only these are safe to anchor an inline discussion
	// to with just position[new_line]. A finding on a context/deleted line falls
	// back to a general note (avoids GitLab's "line_code can't be blank" 400).
	added := addedLines(ch)

	postSummary := !rv.SummaryPublished // default: only if not already posted
	if sel.IncludeSummary != nil {
		postSummary = *sel.IncludeSummary // explicit override (re-selectable)
	}
	if postSummary {
		body := formatSummary(rv)
		if sel.SummaryOverride != nil {
			body = *sel.SummaryOverride // humanized summary replaces the generated body
		}
		if err := gl.CreateNote(ctx, projectID, rv.MRIID, body); err != nil {
			return err
		}
		// The note is on the MR now. Retry recording it a few times so a transient
		// DB failure does not leave SummaryPublished false and re-post on the next
		// publish (mirrors the findings loop persisting posted state before failing).
		if err := s.markSummaryPublishedRetry(ctx, reviewID); err != nil {
			return err
		}
	}

	indices := resolveIndices(sel, rv.Findings)
	published := make([]int, 0, len(indices))
	for _, i := range indices {
		f := rv.Findings[i]
		body := formatFinding(f)
		if text, ok := sel.FindingOverrides[i]; ok {
			body = text // humanized text replaces the generated body as-is
		}

		var perr error
		if f.File != "" && f.Line > 0 && added[f.File][f.Line] {
			perr = gl.CreateInlineDiscussion(ctx, projectID, rv.MRIID, body, gitlab.Position{
				BaseSHA:  refs.BaseSHA,
				StartSHA: refs.StartSHA,
				HeadSHA:  refs.HeadSHA,
				NewPath:  f.File,
				NewLine:  f.Line,
			})
		} else {
			// General note: the finding isn't anchored to a diff line, so prepend
			// the location header to keep the file/line from being lost.
			if loc := findingLocation(f); loc != "" {
				body = "**File:** " + loc + "\n\n" + body
			}
			perr = gl.CreateNote(ctx, projectID, rv.MRIID, body)
		}
		if perr != nil {
			// Persist what did post so a retry does not double-comment those.
			_ = s.reviews.MarkFindingsPublished(ctx, reviewID, published)
			return perr
		}
		published = append(published, i)
	}
	return s.reviews.MarkFindingsPublished(ctx, reviewID, published)
}

// markSummaryPublishedRetry records the posted summary, retrying a few times with
// a short backoff. The summary note is already on the merge request by the time
// this runs, so a transient failure to mark it must not silently allow a re-post
// on a later publish. Returns the last error if every attempt fails.
func (s *Service) markSummaryPublishedRetry(ctx context.Context, reviewID string) error {
	const attempts = 3
	var err error
	for i := 0; i < attempts; i++ {
		if err = s.reviews.MarkSummaryPublished(ctx, reviewID); err == nil {
			return nil
		}
		if i == attempts-1 {
			break
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(time.Duration(i+1) * 50 * time.Millisecond):
		}
	}
	return err
}

// gitlabFor builds a GitLab client for a repo's account and returns the client,
// the encoded project path, and the token (for deep-mode cloning).
func (s *Service) gitlabFor(ctx context.Context, rp repo.Repo) (*gitlab.Client, string, string, error) {
	acc, err := s.accounts.Get(ctx, rp.AccountID)
	if err != nil {
		return nil, "", "", err
	}
	token, err := s.accounts.Token(ctx, rp.AccountID)
	if err != nil {
		return nil, "", "", err
	}
	projectID, err := gitlab.ProjectPath(rp.URL)
	if err != nil {
		return nil, "", "", err
	}
	return gitlab.NewClient(acc.BaseURL, token), projectID, token, nil
}

// resolveIndices expands a Selection into valid, in-range finding indices.
//
// "All" posts only findings that are not yet published, so a repeated
// "publish all" never re-comments what is already on the merge request.
// Explicit indices are honored as-is: selecting a specific finding is a
// deliberate re-selection and may re-post one that was already published.
func resolveIndices(sel Selection, findings []review.Finding) []int {
	if sel.All {
		out := make([]int, 0, len(findings))
		for i, f := range findings {
			if !f.Published {
				out = append(out, i)
			}
		}
		return out
	}
	out := make([]int, 0, len(sel.Indices))
	for _, i := range sel.Indices {
		if i >= 0 && i < len(findings) {
			out = append(out, i)
		}
	}
	return out
}

var dimensionLabel = map[review.Dimension]string{
	review.Risk:        "R1 Risk",
	review.Readability: "R2 Readability",
	review.Reliability: "R3 Reliability",
	review.Resilience:  "R4 Resilience",
}

// findingLocation renders a finding's file location for a general note, e.g.
// "path/to/file.go:42" (or just the path when Line == 0). It returns "" when the
// finding has no file, so the caller can skip the location header entirely.
func findingLocation(f review.Finding) string {
	if f.File == "" {
		return ""
	}
	if f.Line > 0 {
		return fmt.Sprintf("%s:%d", f.File, f.Line)
	}
	return f.File
}

func formatFinding(f review.Finding) string {
	var b strings.Builder
	fmt.Fprintf(&b, "**[%s · %s]** %s\n\n", dimensionLabel[f.Dimension], strings.ToUpper(string(f.Severity)), f.Issue)
	if f.Why != "" {
		fmt.Fprintf(&b, "**Why:** %s\n\n", f.Why)
	}
	if f.Fix != "" {
		fmt.Fprintf(&b, "**Suggested fix:** %s\n", f.Fix)
	}
	if f.Blocking {
		b.WriteString("\n_Blocking._")
	}
	return b.String()
}

func formatSummary(rv review.Review) string {
	blocking := 0
	for _, f := range rv.Findings {
		if f.Blocking {
			blocking++
		}
	}
	var b strings.Builder
	fmt.Fprintf(&b, "## 4R Review — %s (score %d/100)\n\n", recommendationLabel(rv.Recommendation), rv.Score)
	if rv.Summary != "" {
		b.WriteString(rv.Summary)
		b.WriteString("\n\n")
	}
	fmt.Fprintf(&b, "%d finding(s), %d blocking.", len(rv.Findings), blocking)
	return b.String()
}

func recommendationLabel(r review.Recommendation) string {
	switch r {
	case review.Approve:
		return "Approve"
	case review.RequestChanges:
		return "Request changes"
	default:
		return "Comment"
	}
}
