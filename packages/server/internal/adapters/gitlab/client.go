// Package gitlab talks to the GitLab REST API v4 and clones repositories.
// It provides the two context modes reviews run in: "fast" (diff + touched
// files over HTTP) and, via Cloner, "deep-lite" (a shallow local clone).
package gitlab

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// MergeRequest is the subset of a GitLab MR the reviewer needs.
type MergeRequest struct {
	IID          int    `json:"iid"`
	Title        string `json:"title"`
	Description  string `json:"description"`
	State        string `json:"state"`
	SourceBranch string `json:"source_branch"`
	TargetBranch string `json:"target_branch"`
	SHA          string `json:"sha"`
	WebURL       string `json:"web_url"`
	Author       Author `json:"author"`
}

// Author is the MR author.
type Author struct {
	Username string `json:"username"`
	Name     string `json:"name"`
}

// FileChange is a single changed file within an MR.
type FileChange struct {
	OldPath     string `json:"old_path"`
	NewPath     string `json:"new_path"`
	Diff        string `json:"diff"`
	NewFile     bool   `json:"new_file"`
	RenamedFile bool   `json:"renamed_file"`
	DeletedFile bool   `json:"deleted_file"`
}

// DiffRefs are the SHAs needed to anchor an inline comment to a diff line.
type DiffRefs struct {
	BaseSHA  string `json:"base_sha"`
	StartSHA string `json:"start_sha"`
	HeadSHA  string `json:"head_sha"`
}

// Changes is an MR together with its per-file diffs (the "fast" context).
type Changes struct {
	MergeRequest
	DiffRefs DiffRefs     `json:"diff_refs"`
	Files    []FileChange `json:"changes"`
}

// Client is a GitLab REST API v4 client scoped to one account (base URL + token).
type Client struct {
	baseURL string
	token   string
	http    *http.Client
}

// NewClient builds a client for a GitLab instance (e.g. https://gitlab.com)
// authenticated with a personal access token.
func NewClient(baseURL, token string) *Client {
	return &Client{
		baseURL: strings.TrimRight(baseURL, "/"),
		token:   token,
		http:    &http.Client{Timeout: 30 * time.Second},
	}
}

// APIError is returned for non-2xx responses.
type APIError struct {
	Status int
	Body   string
}

func (e *APIError) Error() string {
	return fmt.Sprintf("gitlab: unexpected status %d: %s", e.Status, e.Body)
}

// ListOpenMergeRequests returns the opened MRs of a project. projectID may be a
// numeric ID or a namespaced path ("group/project"); it is URL-encoded here.
func (c *Client) ListOpenMergeRequests(ctx context.Context, projectID string) ([]MergeRequest, error) {
	path := fmt.Sprintf("/projects/%s/merge_requests", url.PathEscape(projectID))
	q := url.Values{"state": {"opened"}, "per_page": {"100"}}

	var mrs []MergeRequest
	if err := c.getJSON(ctx, path, q, &mrs); err != nil {
		return nil, err
	}
	return mrs, nil
}

// MergeRequestChanges returns an MR with its per-file diffs.
func (c *Client) MergeRequestChanges(ctx context.Context, projectID string, iid int) (Changes, error) {
	path := fmt.Sprintf("/projects/%s/merge_requests/%s/changes",
		url.PathEscape(projectID), strconv.Itoa(iid))

	var ch Changes
	if err := c.getJSON(ctx, path, nil, &ch); err != nil {
		return Changes{}, err
	}
	return ch, nil
}

func (c *Client) getJSON(ctx context.Context, path string, query url.Values, out any) error {
	endpoint := c.baseURL + "/api/v4" + path
	if len(query) > 0 {
		endpoint += "?" + query.Encode()
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return fmt.Errorf("gitlab: build request: %w", err)
	}
	req.Header.Set("PRIVATE-TOKEN", c.token)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "ai-reviewer")

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("gitlab: request %s: %w", path, err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
	if err != nil {
		return fmt.Errorf("gitlab: read body: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return &APIError{Status: resp.StatusCode, Body: strings.TrimSpace(string(body))}
	}
	if err := json.Unmarshal(body, out); err != nil {
		return fmt.Errorf("gitlab: decode %s: %w", path, err)
	}
	return nil
}
