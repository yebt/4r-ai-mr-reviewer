package gitlab

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

// CreateNote posts a general (non-anchored) note on a merge request, used for
// the review summary and for findings that have no specific line.
func (c *Client) CreateNote(ctx context.Context, projectID string, iid int, body string) error {
	path := fmt.Sprintf("/projects/%s/merge_requests/%s/notes",
		url.PathEscape(projectID), strconv.Itoa(iid))
	return c.postForm(ctx, path, url.Values{"body": {body}})
}

// Position anchors an inline discussion to a line on the new side of the diff.
type Position struct {
	BaseSHA  string
	StartSHA string
	HeadSHA  string
	NewPath  string
	NewLine  int
}

// CreateInlineDiscussion posts a discussion anchored to a file and line.
func (c *Client) CreateInlineDiscussion(ctx context.Context, projectID string, iid int, body string, pos Position) error {
	path := fmt.Sprintf("/projects/%s/merge_requests/%s/discussions",
		url.PathEscape(projectID), strconv.Itoa(iid))
	form := url.Values{
		"body":                    {body},
		"position[position_type]": {"text"},
		"position[base_sha]":      {pos.BaseSHA},
		"position[start_sha]":     {pos.StartSHA},
		"position[head_sha]":      {pos.HeadSHA},
		"position[new_path]":      {pos.NewPath},
		"position[new_line]":      {strconv.Itoa(pos.NewLine)},
	}
	return c.postForm(ctx, path, form)
}

// AwardEmoji awards a named emoji reaction to a merge request. It is IDEMPOTENT:
// GitLab rejects a duplicate award, so an APIError that indicates the emoji is
// already present (status 404 or 409 with "already" in the body) is treated as
// success. This lets a routine re-run without failing on a reaction it set on a
// previous pass.
func (c *Client) AwardEmoji(ctx context.Context, projectID string, iid int, name string) error {
	path := fmt.Sprintf("/projects/%s/merge_requests/%s/award_emoji",
		url.PathEscape(projectID), strconv.Itoa(iid))
	err := c.doForm(ctx, http.MethodPost, path, url.Values{"name": {name}}, nil)
	if err != nil && alreadyAwarded(err) {
		return nil
	}
	return err
}

// alreadyAwarded reports whether an award_emoji APIError means the reaction was
// already present. GitLab answers a duplicate award with 404 or 409 and a body
// mentioning it is "already" awarded/taken.
func alreadyAwarded(err error) bool {
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		return false
	}
	if apiErr.Status != http.StatusNotFound && apiErr.Status != http.StatusConflict {
		return false
	}
	return strings.Contains(strings.ToLower(apiErr.Body), "already")
}

// CreateTag creates a tag pointing at ref (a branch, tag, or commit SHA). The
// annotation message is omitted when empty, producing a lightweight tag.
func (c *Client) CreateTag(ctx context.Context, projectID, tagName, ref, message string) error {
	path := fmt.Sprintf("/projects/%s/repository/tags", url.PathEscape(projectID))
	form := url.Values{"tag_name": {tagName}, "ref": {ref}}
	if message != "" {
		form.Set("message", message)
	}
	return c.doForm(ctx, http.MethodPost, path, form, nil)
}

// CreateMergeRequest opens a merge request from sourceBranch to targetBranch and
// returns the created MR (its IID is needed to drive it further).
func (c *Client) CreateMergeRequest(ctx context.Context, projectID, sourceBranch, targetBranch, title, description string) (MergeRequest, error) {
	path := fmt.Sprintf("/projects/%s/merge_requests", url.PathEscape(projectID))
	form := url.Values{
		"source_branch": {sourceBranch},
		"target_branch": {targetBranch},
		"title":         {title},
		"description":   {description},
	}
	var mr MergeRequest
	if err := c.doForm(ctx, http.MethodPost, path, form, &mr); err != nil {
		return MergeRequest{}, err
	}
	return mr, nil
}

// MergeOptions controls how a merge request is merged.
type MergeOptions struct {
	MergeWhenPipelineSucceeds bool
	RemoveSourceBranch        bool
	SHA                       string // if set, GitLab merges only when the MR head still matches
}

// MergeMergeRequest merges a merge request and returns the resulting MR.
func (c *Client) MergeMergeRequest(ctx context.Context, projectID string, iid int, opts MergeOptions) (MergeRequest, error) {
	path := fmt.Sprintf("/projects/%s/merge_requests/%s/merge",
		url.PathEscape(projectID), strconv.Itoa(iid))
	form := url.Values{
		"merge_when_pipeline_succeeds": {strconv.FormatBool(opts.MergeWhenPipelineSucceeds)},
		"should_remove_source_branch":  {strconv.FormatBool(opts.RemoveSourceBranch)},
	}
	if opts.SHA != "" {
		form.Set("sha", opts.SHA)
	}
	var mr MergeRequest
	if err := c.doForm(ctx, http.MethodPut, path, form, &mr); err != nil {
		return MergeRequest{}, err
	}
	return mr, nil
}

func (c *Client) postForm(ctx context.Context, path string, form url.Values) error {
	return c.doForm(ctx, http.MethodPost, path, form, nil)
}

// doForm sends a form-urlencoded body (POST/PUT) with the account token and, when
// out is non-nil, JSON-decodes the response into it. A non-2xx status returns an
// APIError. It is the write-side counterpart to getJSON.
func (c *Client) doForm(ctx context.Context, method, path string, form url.Values, out any) error {
	endpoint := c.baseURL + "/api/v4" + path
	req, err := http.NewRequestWithContext(ctx, method, endpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return fmt.Errorf("gitlab: build request: %w", err)
	}
	req.Header.Set("PRIVATE-TOKEN", c.token)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "ai-reviewer")

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("gitlab: request %s: %w", path, err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return &APIError{Status: resp.StatusCode, Body: strings.TrimSpace(string(body))}
	}
	if out != nil && len(body) > 0 {
		// A 2xx with an empty body is a success, not a decode error: some write
		// endpoints (e.g. a duplicate-tolerant POST) answer 2xx with no payload.
		if err := json.Unmarshal(body, out); err != nil {
			return fmt.Errorf("gitlab: decode %s: %w", path, err)
		}
	}
	return nil
}
