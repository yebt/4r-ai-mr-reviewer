// Package reviewctx builds the material a review runs against, in one of two
// modes: Fast (diff + touched files over the API) or Deep (a shallow clone for
// surrounding context). Both produce an engine.Input.
package reviewctx

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/webcloster-dev/ai-reviewer/internal/adapters/gitlab"
	"github.com/webcloster-dev/ai-reviewer/internal/review/engine"
)

// Target identifies the merge request to gather context for.
type Target struct {
	RepoURL   string
	ProjectID string
	MRIID     int
}

// Strategy gathers review context. cleanup releases any resources (e.g. a
// clone) and is always safe to call, even on error.
type Strategy interface {
	Build(ctx context.Context, t Target) (engine.Input, func(), error)
}

// noop is an empty cleanup function.
func noop() {}

// FastStrategy fetches the diff and touched files over the REST API only.
type FastStrategy struct {
	client *gitlab.Client
}

// NewFastStrategy builds a fast-context strategy.
func NewFastStrategy(client *gitlab.Client) FastStrategy {
	return FastStrategy{client: client}
}

// Build returns the diff-only context.
func (s FastStrategy) Build(ctx context.Context, t Target) (engine.Input, func(), error) {
	ch, err := s.client.MergeRequestChanges(ctx, t.ProjectID, t.MRIID)
	if err != nil {
		return engine.Input{}, noop, err
	}
	return engine.Input{
		MRIID:       t.MRIID,
		Title:       ch.Title,
		Description: ch.Description,
		Diff:        renderDiff(ch),
	}, noop, nil
}

// DeepStrategy clones the repo and includes the full content of changed files
// alongside the diff, giving the model surrounding context.
type DeepStrategy struct {
	client *gitlab.Client
	cloner *gitlab.Cloner
}

// NewDeepStrategy builds a deep-context strategy.
func NewDeepStrategy(client *gitlab.Client, cloner *gitlab.Cloner) DeepStrategy {
	return DeepStrategy{client: client, cloner: cloner}
}

// Build clones the MR's source branch and appends changed-file contents.
func (s DeepStrategy) Build(ctx context.Context, t Target) (engine.Input, func(), error) {
	ch, err := s.client.MergeRequestChanges(ctx, t.ProjectID, t.MRIID)
	if err != nil {
		return engine.Input{}, noop, err
	}

	workDir, err := os.MkdirTemp("", "air-clone-*")
	if err != nil {
		return engine.Input{}, noop, fmt.Errorf("reviewctx: temp dir: %w", err)
	}
	cleanup := func() { os.RemoveAll(workDir) }

	checkout, err := s.cloner.Clone(ctx, t.RepoURL, ch.SourceBranch, workDir)
	if err != nil {
		cleanup()
		return engine.Input{}, noop, err
	}

	var b strings.Builder
	b.WriteString(renderDiff(ch))
	b.WriteString("\n\n=== Full content of changed files ===\n")
	for _, f := range ch.Files {
		if f.DeletedFile || f.NewPath == "" {
			continue
		}
		content, err := os.ReadFile(filepath.Join(checkout, f.NewPath))
		if err != nil {
			continue // file may be absent (e.g. binary or moved); skip quietly
		}
		fmt.Fprintf(&b, "\n--- %s ---\n%s\n", f.NewPath, content)
	}

	return engine.Input{
		MRIID:       t.MRIID,
		Title:       ch.Title,
		Description: ch.Description,
		Diff:        b.String(),
	}, cleanup, nil
}

// renderDiff flattens per-file diffs into a single labelled block.
func renderDiff(ch gitlab.Changes) string {
	var b strings.Builder
	for _, f := range ch.Files {
		path := f.NewPath
		if path == "" {
			path = f.OldPath
		}
		fmt.Fprintf(&b, "diff: %s\n%s\n", path, f.Diff)
	}
	return b.String()
}
