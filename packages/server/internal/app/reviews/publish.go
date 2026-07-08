package reviews

import (
	"context"
	"fmt"
	"strings"

	"github.com/webcloster-dev/ai-reviewer/internal/adapters/gitlab"
	"github.com/webcloster-dev/ai-reviewer/internal/domain/repo"
	"github.com/webcloster-dev/ai-reviewer/internal/domain/review"
)

// Selection chooses which findings to publish: All, or specific 0-based indices
// into the review's stored findings.
type Selection struct {
	All     bool
	Indices []int
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

	if err := gl.CreateNote(ctx, projectID, rv.MRIID, formatSummary(rv)); err != nil {
		return err
	}

	indices := resolveIndices(sel, len(rv.Findings))
	published := make([]int, 0, len(indices))
	for _, i := range indices {
		f := rv.Findings[i]
		body := formatFinding(f)

		var perr error
		if f.File != "" && f.Line > 0 {
			perr = gl.CreateInlineDiscussion(ctx, projectID, rv.MRIID, body, gitlab.Position{
				BaseSHA:  refs.BaseSHA,
				StartSHA: refs.StartSHA,
				HeadSHA:  refs.HeadSHA,
				NewPath:  f.File,
				NewLine:  f.Line,
			})
		} else {
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
func resolveIndices(sel Selection, n int) []int {
	if sel.All {
		out := make([]int, n)
		for i := range out {
			out[i] = i
		}
		return out
	}
	out := make([]int, 0, len(sel.Indices))
	for _, i := range sel.Indices {
		if i >= 0 && i < n {
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
