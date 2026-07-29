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
	// MergeCommitSHA / SquashCommitSHA are the commit(s) produced when the MR is
	// merged; routines tag the merge commit rather than a moving branch head.
	MergeCommitSHA  string `json:"merge_commit_sha"`
	SquashCommitSHA string `json:"squash_commit_sha"`
	WebURL          string `json:"web_url"`
	Author          Author `json:"author"`
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

// GetMergeRequest returns a single merge request by IID. Routines need its
// target/source branches and state to decide the tag ref and prerelease suffix.
func (c *Client) GetMergeRequest(ctx context.Context, projectID string, iid int) (MergeRequest, error) {
	path := fmt.Sprintf("/projects/%s/merge_requests/%s",
		url.PathEscape(projectID), strconv.Itoa(iid))

	var mr MergeRequest
	if err := c.getJSON(ctx, path, nil, &mr); err != nil {
		return MergeRequest{}, err
	}
	return mr, nil
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

// Tag is a repository tag.
type Tag struct {
	Name    string `json:"name"`
	Message string `json:"message"`
	Target  string `json:"target"`
}

// ListTags returns a project's tags, newest version first.
func (c *Client) ListTags(ctx context.Context, projectID string) ([]Tag, error) {
	path := fmt.Sprintf("/projects/%s/repository/tags", url.PathEscape(projectID))
	q := url.Values{"order_by": {"version"}, "sort": {"desc"}, "per_page": {"100"}}

	var tags []Tag
	if err := c.getJSON(ctx, path, q, &tags); err != nil {
		return nil, err
	}
	return tags, nil
}

// Pipeline is a CI pipeline attached to a merge request.
type Pipeline struct {
	ID     int    `json:"id"`
	Status string `json:"status"`
	Ref    string `json:"ref"`
	SHA    string `json:"sha"`
}

// MergeRequestPipelines returns the pipelines run for a merge request.
func (c *Client) MergeRequestPipelines(ctx context.Context, projectID string, iid int) ([]Pipeline, error) {
	path := fmt.Sprintf("/projects/%s/merge_requests/%s/pipelines",
		url.PathEscape(projectID), strconv.Itoa(iid))

	var pipelines []Pipeline
	if err := c.getJSON(ctx, path, nil, &pipelines); err != nil {
		return nil, err
	}
	return pipelines, nil
}

// TokenInfo describes the personal access token in use, used to check its scopes.
type TokenInfo struct {
	Name    string   `json:"name"`
	Scopes  []string `json:"scopes"`
	Active  bool     `json:"active"`
	Revoked bool     `json:"revoked"`
}

// TokenSelf returns metadata about the current personal access token. It fails
// with 401/404 for OAuth, job, or deploy tokens, so callers must tolerate an
// error here (token scopes are simply unknown).
func (c *Client) TokenSelf(ctx context.Context) (TokenInfo, error) {
	var info TokenInfo
	if err := c.getJSON(ctx, "/personal_access_tokens/self", nil, &info); err != nil {
		return TokenInfo{}, err
	}
	return info, nil
}

// AccessInfo is a single access-level grant on a project or group.
type AccessInfo struct {
	AccessLevel int `json:"access_level"`
}

// ProjectPermissions carries the caller's effective access on a project, either
// directly (project_access) or inherited from the group (group_access). Either
// may be null when the caller has no membership at that level.
type ProjectPermissions struct {
	ProjectAccess *AccessInfo `json:"project_access"`
	GroupAccess   *AccessInfo `json:"group_access"`
}

// Project is the subset of a GitLab project the preflight needs.
type Project struct {
	DefaultBranch string             `json:"default_branch"`
	Permissions   ProjectPermissions `json:"permissions"`
}

// Project returns a single project, including the caller's permissions.
func (c *Client) Project(ctx context.Context, projectID string) (Project, error) {
	path := fmt.Sprintf("/projects/%s", url.PathEscape(projectID))

	var p Project
	if err := c.getJSON(ctx, path, nil, &p); err != nil {
		return Project{}, err
	}
	return p, nil
}

// AccessLevelRule is one access-level entry within a protection rule.
type AccessLevelRule struct {
	AccessLevel            int    `json:"access_level"`
	AccessLevelDescription string `json:"access_level_description"`
}

// ProtectedBranch is a branch protection rule and the access levels allowed to
// merge into it.
type ProtectedBranch struct {
	Name              string            `json:"name"`
	MergeAccessLevels []AccessLevelRule `json:"merge_access_levels"`
}

// ProtectedBranches returns the project's branch protection rules.
func (c *Client) ProtectedBranches(ctx context.Context, projectID string) ([]ProtectedBranch, error) {
	path := fmt.Sprintf("/projects/%s/protected_branches", url.PathEscape(projectID))

	var branches []ProtectedBranch
	if err := c.getJSON(ctx, path, nil, &branches); err != nil {
		return nil, err
	}
	return branches, nil
}

// ProtectedTag is a tag protection rule and the access levels allowed to create
// matching tags.
type ProtectedTag struct {
	Name               string            `json:"name"`
	CreateAccessLevels []AccessLevelRule `json:"create_access_levels"`
}

// ProtectedTags returns the project's tag protection rules.
func (c *Client) ProtectedTags(ctx context.Context, projectID string) ([]ProtectedTag, error) {
	path := fmt.Sprintf("/projects/%s/protected_tags", url.PathEscape(projectID))

	var tags []ProtectedTag
	if err := c.getJSON(ctx, path, nil, &tags); err != nil {
		return nil, err
	}
	return tags, nil
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
