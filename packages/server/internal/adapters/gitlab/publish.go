package gitlab

import (
	"context"
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

func (c *Client) postForm(ctx context.Context, path string, form url.Values) error {
	endpoint := c.baseURL + "/api/v4" + path
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return fmt.Errorf("gitlab: build request: %w", err)
	}
	req.Header.Set("PRIVATE-TOKEN", c.token)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
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
	return nil
}
