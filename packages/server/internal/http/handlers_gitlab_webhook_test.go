package httpapi

import (
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"
)

// postGitlabWebhook posts a raw JSON body to a repo's GitLab webhook with an
// optional X-Gitlab-Token header.
func postGitlabWebhook(t *testing.T, url, token, body string) *http.Response {
	t.Helper()
	req, err := http.NewRequest(http.MethodPost, url, strings.NewReader(body))
	if err != nil {
		t.Fatalf("build webhook request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("X-Gitlab-Token", token)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("POST gitlab webhook: %v", err)
	}
	return resp
}

// seedRepo creates an account, provider, and repo over HTTP and returns the repo id.
func seedRepo(t *testing.T, baseURL string) string {
	t.Helper()
	acctResp := postJSON(t, baseURL+"/accounts", map[string]any{"name": "a", "baseUrl": "https://gitlab.com", "token": "t"})
	var acct struct{ ID string }
	decodeBody(t, acctResp, &acct)

	provResp := postJSON(t, baseURL+"/providers", map[string]any{"name": "p", "kind": "openai-compat", "model": "m", "apiKey": "k"})
	var prov struct{ ID string }
	decodeBody(t, provResp, &prov)

	repoResp := postJSON(t, baseURL+"/repos", map[string]any{"name": "web", "url": "https://gitlab.com/g/p", "accountId": acct.ID, "providerId": prov.ID})
	if repoResp.StatusCode != http.StatusCreated {
		t.Fatalf("create repo status = %d", repoResp.StatusCode)
	}
	var repoObj struct{ ID string }
	decodeBody(t, repoResp, &repoObj)
	return repoObj.ID
}

// countReviews returns how many reviews the repo currently has.
func countReviews(t *testing.T, baseURL, repoID string) int {
	t.Helper()
	resp, err := http.Get(baseURL + "/repos/" + repoID + "/reviews")
	if err != nil {
		t.Fatalf("GET reviews: %v", err)
	}
	var list []map[string]any
	decodeBody(t, resp, &list)
	return len(list)
}

// waitReviewCount polls until the repo has at least want reviews or the deadline
// passes (the webhook dispatches review creation on a goroutine).
func waitReviewCount(t *testing.T, baseURL, repoID string, want int) int {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	var n int
	for time.Now().Before(deadline) {
		n = countReviews(t, baseURL, repoID)
		if n >= want {
			return n
		}
		time.Sleep(20 * time.Millisecond)
	}
	return n
}

func mrEventBody(action string, iid int, oldrev string) string {
	oldrevField := ""
	if oldrev != "" {
		oldrevField = fmt.Sprintf(`"oldrev":%q,`, oldrev)
	}
	return fmt.Sprintf(
		`{"object_kind":"merge_request","object_attributes":{"iid":%d,"action":%q,%s"state":"opened","target_branch":"main","source_branch":"feat"},"project":{"id":1,"path_with_namespace":"g/p"}}`,
		iid, action, oldrevField)
}

// enableWebhook turns the repo's webhook on and returns its secret.
func enableWebhook(t *testing.T, baseURL, repoID string) string {
	t.Helper()
	resp := sendJSON(t, http.MethodPatch, baseURL+"/repos/"+repoID+"/webhook", map[string]any{"enabled": true})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("enable webhook status = %d, want 200", resp.StatusCode)
	}
	var repoObj struct {
		WebhookEnabled bool   `json:"webhookEnabled"`
		WebhookSecret  string `json:"webhookSecret"`
		WebhookPath    string `json:"webhookPath"`
	}
	decodeBody(t, resp, &repoObj)
	if !repoObj.WebhookEnabled || repoObj.WebhookSecret == "" {
		t.Fatalf("enable webhook response = %+v, want enabled with a secret", repoObj)
	}
	if repoObj.WebhookPath != "/webhooks/gitlab/"+repoID {
		t.Fatalf("webhookPath = %q, want /webhooks/gitlab/%s", repoObj.WebhookPath, repoID)
	}
	return repoObj.WebhookSecret
}

// enableWebhookRequireConfirmation turns the repo's webhook on WITH the manual
// confirmation gate and returns its secret.
func enableWebhookRequireConfirmation(t *testing.T, baseURL, repoID string) string {
	t.Helper()
	resp := sendJSON(t, http.MethodPatch, baseURL+"/repos/"+repoID+"/webhook", map[string]any{"enabled": true, "requireConfirmation": true})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("enable webhook (confirm) status = %d, want 200", resp.StatusCode)
	}
	var repoObj struct {
		WebhookEnabled             bool   `json:"webhookEnabled"`
		WebhookRequireConfirmation bool   `json:"webhookRequireConfirmation"`
		WebhookSecret              string `json:"webhookSecret"`
	}
	decodeBody(t, resp, &repoObj)
	if !repoObj.WebhookEnabled || !repoObj.WebhookRequireConfirmation || repoObj.WebhookSecret == "" {
		t.Fatalf("enable webhook (confirm) response = %+v, want enabled+requireConfirmation with a secret", repoObj)
	}
	return repoObj.WebhookSecret
}

// listReviews returns the repo's reviews decoded with id and status.
func listReviews(t *testing.T, baseURL, repoID string) []struct {
	ID     string `json:"id"`
	Status string `json:"status"`
} {
	t.Helper()
	resp, err := http.Get(baseURL + "/repos/" + repoID + "/reviews")
	if err != nil {
		t.Fatalf("GET reviews: %v", err)
	}
	var list []struct {
		ID     string `json:"id"`
		Status string `json:"status"`
	}
	decodeBody(t, resp, &list)
	return list
}

// TestGitlabWebhookRequireConfirmationHoldsReview asserts that with the
// confirmation gate on, a webhook trigger creates a HELD review (awaiting_approval)
// and that POST /reviews/{id}/approve promotes it to pending.
func TestGitlabWebhookRequireConfirmationHoldsReview(t *testing.T) {
	srv := newTestServer(t)
	repoID := seedRepo(t, srv.URL)
	secret := enableWebhookRequireConfirmation(t, srv.URL, repoID)

	resp := postGitlabWebhook(t, srv.URL+"/webhooks/gitlab/"+repoID, secret, mrEventBody("open", 7, ""))
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("open webhook status = %d, want 200", resp.StatusCode)
	}
	resp.Body.Close()

	if n := waitReviewCount(t, srv.URL, repoID, 1); n != 1 {
		t.Fatalf("gated webhook created %d reviews, want 1", n)
	}
	list := listReviews(t, srv.URL, repoID)
	if len(list) != 1 || list[0].Status != "awaiting_approval" {
		t.Fatalf("gated webhook review = %+v, want a single awaiting_approval review", list)
	}
	reviewID := list[0].ID

	// Approve promotes the held review to pending.
	appResp := postJSON(t, srv.URL+"/reviews/"+reviewID+"/approve", nil)
	if appResp.StatusCode != http.StatusOK {
		t.Fatalf("approve status = %d, want 200", appResp.StatusCode)
	}
	var approved struct {
		Status string `json:"status"`
	}
	decodeBody(t, appResp, &approved)
	if approved.Status != "pending" {
		t.Fatalf("approved status = %q, want pending", approved.Status)
	}
}

// TestApproveReviewWrongStateConflict asserts approving a review that is not
// awaiting approval (a normally-created pending review) is a 409 conflict.
func TestApproveReviewWrongStateConflict(t *testing.T) {
	srv := newTestServer(t)
	repoID := seedRepo(t, srv.URL)

	revResp := postJSON(t, srv.URL+"/reviews", map[string]any{"repoId": repoID, "mrIid": 7, "mode": "fast"})
	if revResp.StatusCode != http.StatusCreated {
		t.Fatalf("create review status = %d, want 201", revResp.StatusCode)
	}
	var rev struct {
		ID string `json:"id"`
	}
	decodeBody(t, revResp, &rev)

	appResp := postJSON(t, srv.URL+"/reviews/"+rev.ID+"/approve", nil)
	if appResp.StatusCode != http.StatusConflict {
		t.Fatalf("approve on pending review status = %d, want 409", appResp.StatusCode)
	}
	appResp.Body.Close()
}

// TestGitlabWebhookDisabledRepo asserts a repo with the webhook disabled answers
// 200 and never creates a review.
func TestGitlabWebhookDisabledRepo(t *testing.T) {
	srv := newTestServer(t)
	repoID := seedRepo(t, srv.URL)

	resp := postGitlabWebhook(t, srv.URL+"/webhooks/gitlab/"+repoID, "anything", mrEventBody("open", 7, ""))
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("disabled webhook status = %d, want 200", resp.StatusCode)
	}
	resp.Body.Close()

	if n := countReviews(t, srv.URL, repoID); n != 0 {
		t.Fatalf("disabled repo created %d reviews, want 0", n)
	}
}

// TestGitlabWebhookWrongToken asserts an enabled repo rejects a mismatched token
// with 401 and creates no review.
func TestGitlabWebhookWrongToken(t *testing.T) {
	srv := newTestServer(t)
	repoID := seedRepo(t, srv.URL)
	enableWebhook(t, srv.URL, repoID)

	resp := postGitlabWebhook(t, srv.URL+"/webhooks/gitlab/"+repoID, "wrong-token", mrEventBody("open", 7, ""))
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("wrong-token webhook status = %d, want 401", resp.StatusCode)
	}
	resp.Body.Close()

	if n := countReviews(t, srv.URL, repoID); n != 0 {
		t.Fatalf("wrong-token webhook created %d reviews, want 0", n)
	}
}

// TestGitlabWebhookOpenCreatesReview asserts a valid token + an "open" action
// creates a review.
func TestGitlabWebhookOpenCreatesReview(t *testing.T) {
	srv := newTestServer(t)
	repoID := seedRepo(t, srv.URL)
	secret := enableWebhook(t, srv.URL, repoID)

	resp := postGitlabWebhook(t, srv.URL+"/webhooks/gitlab/"+repoID, secret, mrEventBody("open", 7, ""))
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("open webhook status = %d, want 200", resp.StatusCode)
	}
	resp.Body.Close()

	if n := waitReviewCount(t, srv.URL, repoID, 1); n != 1 {
		t.Fatalf("open webhook created %d reviews, want 1", n)
	}
}

// TestGitlabWebhookUpdateWithoutOldrev asserts a metadata-only update (no oldrev)
// does not trigger a review.
func TestGitlabWebhookUpdateWithoutOldrev(t *testing.T) {
	srv := newTestServer(t)
	repoID := seedRepo(t, srv.URL)
	secret := enableWebhook(t, srv.URL, repoID)

	resp := postGitlabWebhook(t, srv.URL+"/webhooks/gitlab/"+repoID, secret, mrEventBody("update", 7, ""))
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("update-no-oldrev webhook status = %d, want 200", resp.StatusCode)
	}
	resp.Body.Close()

	// Give any (erroneous) dispatch a chance to run, then assert nothing appeared.
	time.Sleep(150 * time.Millisecond)
	if n := countReviews(t, srv.URL, repoID); n != 0 {
		t.Fatalf("metadata-only update created %d reviews, want 0", n)
	}
}

// TestGitlabWebhookUpdateWithOldrev asserts an update carrying new commits
// (oldrev set) does trigger a review.
func TestGitlabWebhookUpdateWithOldrev(t *testing.T) {
	srv := newTestServer(t)
	repoID := seedRepo(t, srv.URL)
	secret := enableWebhook(t, srv.URL, repoID)

	resp := postGitlabWebhook(t, srv.URL+"/webhooks/gitlab/"+repoID, secret, mrEventBody("update", 7, "abc123"))
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("update-with-oldrev webhook status = %d, want 200", resp.StatusCode)
	}
	resp.Body.Close()

	if n := waitReviewCount(t, srv.URL, repoID, 1); n != 1 {
		t.Fatalf("update-with-oldrev created %d reviews, want 1", n)
	}
}

// TestGitlabWebhookUnknownRepo asserts an unknown repo id is a 404.
func TestGitlabWebhookUnknownRepo(t *testing.T) {
	srv := newTestServer(t)
	resp := postGitlabWebhook(t, srv.URL+"/webhooks/gitlab/does-not-exist", "x", mrEventBody("open", 7, ""))
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("unknown-repo webhook status = %d, want 404", resp.StatusCode)
	}
	resp.Body.Close()
}

// TestGitlabWebhookAuthExempt asserts the auth middleware exempts the webhook
// path: with password auth enabled, an unauthenticated POST reaches the handler
// (which returns 404 for an unknown repo) rather than being blocked with a 401.
func TestGitlabWebhookAuthExempt(t *testing.T) {
	srv := newTestServerWithAuth(t, "hunter2")
	resp := postGitlabWebhook(t, srv.URL+"/webhooks/gitlab/does-not-exist", "x", mrEventBody("open", 7, ""))
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("auth-enabled webhook (no cookie) status = %d, want 404 (exempt, reached handler), not 401", resp.StatusCode)
	}
	resp.Body.Close()
}
