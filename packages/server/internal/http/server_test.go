package httpapi

import (
	"bytes"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/webcloster-dev/ai-reviewer/internal/adapters/crypto"
	"github.com/webcloster-dev/ai-reviewer/internal/adapters/sqlite"
	"github.com/webcloster-dev/ai-reviewer/internal/app/accounts"
	apphumanize "github.com/webcloster-dev/ai-reviewer/internal/app/humanize"
	"github.com/webcloster-dev/ai-reviewer/internal/app/notifications"
	"github.com/webcloster-dev/ai-reviewer/internal/app/profiles"
	"github.com/webcloster-dev/ai-reviewer/internal/app/providers"
	apprepos "github.com/webcloster-dev/ai-reviewer/internal/app/repos"
	"github.com/webcloster-dev/ai-reviewer/internal/app/reviews"
	apptelegram "github.com/webcloster-dev/ai-reviewer/internal/app/telegram"
	"github.com/webcloster-dev/ai-reviewer/internal/jobs"
	"github.com/webcloster-dev/ai-reviewer/internal/review/engine"
	"github.com/webcloster-dev/ai-reviewer/internal/review/skills"
)

func newTestServer(t *testing.T) *httptest.Server {
	t.Helper()
	db, err := sqlite.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	salt, _ := crypto.NewSalt()
	key, _ := crypto.DeriveKey("pw", salt)
	cipher, _ := crypto.NewCipher(key)
	secrets := sqlite.NewSecretStore(db, cipher)

	accountSvc := accounts.NewService(sqlite.NewAccountRepo(db), secrets)
	providerSvc := providers.NewService(sqlite.NewProviderRepo(db), secrets)
	profileSvc := profiles.NewService(sqlite.NewProfileStore(db), providerSvc, log.New(io.Discard, "", 0))
	repoSvc := apprepos.NewService(sqlite.NewRepoStore(db), sqlite.NewAccountRepo(db), sqlite.NewProviderRepo(db))
	set, _ := skills.Load("")
	reviewSvc := reviews.NewService(sqlite.NewReviewStore(db), sqlite.NewRepoStore(db), accountSvc, providerSvc, engine.New(set))
	humanizeSvc := apphumanize.NewService(sqlite.NewReviewStore(db), sqlite.NewProfileStore(db), sqlite.NewHumanizationStore(db), providerSvc, log.New(io.Discard, "", 0))
	telegramSvc := apptelegram.NewService(sqlite.NewTelegramStore(db), secrets)
	notificationsSvc := notifications.NewService(sqlite.NewNotificationRuleStore(db), telegramSvc)
	runner := jobs.NewRunner(sqlite.NewJobStore(db), reviewSvc.Handle, jobs.WithLogger(log.New(io.Discard, "", 0)))
	reviewSvc.AttachRunner(runner)

	srv := httptest.NewServer(NewServer(accountSvc, providerSvc, profileSvc, repoSvc, reviewSvc, humanizeSvc, telegramSvc, notificationsSvc, set).Routes())
	t.Cleanup(srv.Close)
	return srv
}

func postJSON(t *testing.T, url string, body any) *http.Response {
	t.Helper()
	b, _ := json.Marshal(body)
	resp, err := http.Post(url, "application/json", bytes.NewReader(b))
	if err != nil {
		t.Fatalf("POST %s: %v", url, err)
	}
	return resp
}

func decodeBody(t *testing.T, resp *http.Response, dst any) {
	t.Helper()
	defer resp.Body.Close()
	if err := json.NewDecoder(resp.Body).Decode(dst); err != nil {
		t.Fatalf("decode: %v", err)
	}
}

func TestHealth(t *testing.T) {
	srv := newTestServer(t)
	resp, err := http.Get(srv.URL + "/health")
	if err != nil {
		t.Fatalf("GET /health: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
}

func TestAccountLifecycleOverHTTP(t *testing.T) {
	srv := newTestServer(t)

	resp := postJSON(t, srv.URL+"/accounts", map[string]any{"name": "acc", "baseUrl": "https://gitlab.com", "token": "glpat"})
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("create account status = %d, want 201", resp.StatusCode)
	}
	var created struct{ ID string }
	decodeBody(t, resp, &created)
	if created.ID == "" {
		t.Fatal("created account has no id")
	}

	listResp, _ := http.Get(srv.URL + "/accounts")
	var list []map[string]any
	decodeBody(t, listResp, &list)
	if len(list) != 1 {
		t.Fatalf("account list len = %d, want 1", len(list))
	}

	// The token must never be exposed in the API response.
	if _, leaked := list[0]["token"]; leaked {
		t.Fatal("account response leaked the token")
	}
}

func TestCreateReviewOverHTTP(t *testing.T) {
	srv := newTestServer(t)

	acctResp := postJSON(t, srv.URL+"/accounts", map[string]any{"name": "a", "baseUrl": "https://gitlab.com", "token": "t"})
	var acct struct{ ID string }
	decodeBody(t, acctResp, &acct)

	provResp := postJSON(t, srv.URL+"/providers", map[string]any{"name": "p", "kind": "openai-compat", "model": "m", "apiKey": "k"})
	var prov struct{ ID string }
	decodeBody(t, provResp, &prov)

	repoResp := postJSON(t, srv.URL+"/repos", map[string]any{"name": "web", "url": "https://gitlab.com/g/p", "accountId": acct.ID, "providerId": prov.ID})
	if repoResp.StatusCode != http.StatusCreated {
		t.Fatalf("create repo status = %d", repoResp.StatusCode)
	}
	var repoObj struct{ ID string }
	decodeBody(t, repoResp, &repoObj)

	revResp := postJSON(t, srv.URL+"/reviews", map[string]any{"repoId": repoObj.ID, "mrIid": 7, "mode": "fast"})
	if revResp.StatusCode != http.StatusCreated {
		t.Fatalf("create review status = %d, want 201", revResp.StatusCode)
	}
	var rev struct {
		ID     string `json:"id"`
		Status string `json:"status"`
	}
	decodeBody(t, revResp, &rev)
	if rev.Status != "pending" {
		t.Fatalf("review status = %q, want pending", rev.Status)
	}

	// It should show up under the repo's reviews.
	listResp, _ := http.Get(srv.URL + "/repos/" + repoObj.ID + "/reviews")
	var reviewsList []map[string]any
	decodeBody(t, listResp, &reviewsList)
	if len(reviewsList) != 1 {
		t.Fatalf("repo reviews len = %d, want 1", len(reviewsList))
	}
}

func TestTelegramLifecycleOverHTTP(t *testing.T) {
	srv := newTestServer(t)

	resp := postJSON(t, srv.URL+"/telegram", map[string]any{"name": "team", "botToken": "bot-secret", "chatId": "-100", "isDefault": true})
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("create telegram status = %d, want 201", resp.StatusCode)
	}
	var created map[string]any
	decodeBody(t, resp, &created)
	if created["id"] == "" || created["id"] == nil {
		t.Fatal("created telegram target has no id")
	}
	// The bot token must never be exposed in the API response.
	if _, leaked := created["botToken"]; leaked {
		t.Fatal("telegram response leaked the bot token")
	}
	if _, leaked := created["token"]; leaked {
		t.Fatal("telegram response leaked the token")
	}
	if created["isDefault"] != true {
		t.Fatalf("isDefault = %v, want true", created["isDefault"])
	}

	listResp, _ := http.Get(srv.URL + "/telegram")
	var list []map[string]any
	decodeBody(t, listResp, &list)
	if len(list) != 1 {
		t.Fatalf("telegram list len = %d, want 1", len(list))
	}

	id, _ := created["id"].(string)
	delResp, err := http.NewRequest(http.MethodDelete, srv.URL+"/telegram/"+id, nil)
	if err != nil {
		t.Fatalf("build delete: %v", err)
	}
	done, err := http.DefaultClient.Do(delResp)
	if err != nil {
		t.Fatalf("DELETE /telegram: %v", err)
	}
	if done.StatusCode != http.StatusNoContent {
		t.Fatalf("delete telegram status = %d, want 204", done.StatusCode)
	}
}

func TestCreateRepoRejectsUnknownAccount(t *testing.T) {
	srv := newTestServer(t)
	resp := postJSON(t, srv.URL+"/repos", map[string]any{"name": "web", "url": "https://gitlab.com/g/p", "accountId": "nope"})
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("status = %d, want 404 for unknown account", resp.StatusCode)
	}
}

// sendJSON issues a request with a JSON body for verbs Post does not cover
// (PATCH here) so the notification-rules wire contract can be exercised.
func sendJSON(t *testing.T, method, url string, body any) *http.Response {
	t.Helper()
	b, _ := json.Marshal(body)
	req, err := http.NewRequest(method, url, bytes.NewReader(b))
	if err != nil {
		t.Fatalf("build %s %s: %v", method, url, err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("%s %s: %v", method, url, err)
	}
	return resp
}

func doDelete(t *testing.T, url string) *http.Response {
	t.Helper()
	req, err := http.NewRequest(http.MethodDelete, url, nil)
	if err != nil {
		t.Fatalf("build delete %s: %v", url, err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("DELETE %s: %v", url, err)
	}
	return resp
}

// TestNotificationRulesLifecycleOverHTTP exercises the /notifications/* wire
// contract end to end, plus the delete-target -> rules cascade, at the HTTP
// boundary.
func TestNotificationRulesLifecycleOverHTTP(t *testing.T) {
	srv := newTestServer(t)

	// A notifier target to assign rules to.
	tgResp := postJSON(t, srv.URL+"/telegram", map[string]any{"name": "team", "botToken": "bot-secret", "chatId": "-100"})
	if tgResp.StatusCode != http.StatusCreated {
		t.Fatalf("create telegram status = %d, want 201", tgResp.StatusCode)
	}
	var target struct {
		ID string `json:"id"`
	}
	decodeBody(t, tgResp, &target)
	if target.ID == "" {
		t.Fatal("created telegram target has no id")
	}

	// Events list must advertise review.finished.
	evResp, _ := http.Get(srv.URL + "/notifications/events")
	if evResp.StatusCode != http.StatusOK {
		t.Fatalf("events status = %d, want 200", evResp.StatusCode)
	}
	var events struct {
		Events []string `json:"events"`
	}
	decodeBody(t, evResp, &events)
	if !containsString(events.Events, "review.finished") {
		t.Fatalf("events = %v, want it to contain review.finished", events.Events)
	}

	// Create a rule bound to the target.
	ruleResp := postJSON(t, srv.URL+"/notifications/rules", map[string]any{"event": "review.finished", "notifierId": target.ID})
	if ruleResp.StatusCode != http.StatusCreated {
		t.Fatalf("create rule status = %d, want 201", ruleResp.StatusCode)
	}
	var rule struct {
		ID           string `json:"id"`
		Event        string `json:"event"`
		NotifierKind string `json:"notifierKind"`
		NotifierID   string `json:"notifierId"`
		Enabled      bool   `json:"enabled"`
	}
	decodeBody(t, ruleResp, &rule)
	if rule.ID == "" || rule.Event != "review.finished" || rule.NotifierKind != "telegram" || rule.NotifierID != target.ID || !rule.Enabled {
		t.Fatalf("unexpected created rule: %+v", rule)
	}

	// It shows up in the list, enabled.
	listResp, _ := http.Get(srv.URL + "/notifications/rules")
	var rules []map[string]any
	decodeBody(t, listResp, &rules)
	if len(rules) != 1 || rules[0]["id"] != rule.ID || rules[0]["enabled"] != true {
		t.Fatalf("rules list = %+v, want the single enabled rule %s", rules, rule.ID)
	}

	// A duplicate (same event + target) is a 409 conflict.
	dupResp := postJSON(t, srv.URL+"/notifications/rules", map[string]any{"event": "review.finished", "notifierId": target.ID})
	if dupResp.StatusCode != http.StatusConflict {
		t.Fatalf("duplicate rule status = %d, want 409", dupResp.StatusCode)
	}
	dupResp.Body.Close()

	// A rule against a nonexistent target is a 400.
	missingResp := postJSON(t, srv.URL+"/notifications/rules", map[string]any{"event": "review.finished", "notifierId": "does-not-exist"})
	if missingResp.StatusCode != http.StatusBadRequest {
		t.Fatalf("missing-target rule status = %d, want 400", missingResp.StatusCode)
	}
	missingResp.Body.Close()

	// An unknown event is a 400.
	bogusResp := postJSON(t, srv.URL+"/notifications/rules", map[string]any{"event": "bogus.event", "notifierId": target.ID})
	if bogusResp.StatusCode != http.StatusBadRequest {
		t.Fatalf("bogus-event rule status = %d, want 400", bogusResp.StatusCode)
	}
	bogusResp.Body.Close()

	// Disable the rule; the response reflects enabled=false.
	patchResp := sendJSON(t, http.MethodPatch, srv.URL+"/notifications/rules/"+rule.ID, map[string]any{"enabled": false})
	if patchResp.StatusCode != http.StatusOK {
		t.Fatalf("patch rule status = %d, want 200", patchResp.StatusCode)
	}
	var patched map[string]any
	decodeBody(t, patchResp, &patched)
	if patched["enabled"] != false {
		t.Fatalf("patched enabled = %v, want false", patched["enabled"])
	}

	// Patching an unknown rule is a 404.
	unknownResp := sendJSON(t, http.MethodPatch, srv.URL+"/notifications/rules/nope", map[string]any{"enabled": true})
	if unknownResp.StatusCode != http.StatusNotFound {
		t.Fatalf("patch unknown rule status = %d, want 404", unknownResp.StatusCode)
	}
	unknownResp.Body.Close()

	// Deleting the target cascades: its rules must be gone.
	delResp := doDelete(t, srv.URL+"/telegram/"+target.ID)
	if delResp.StatusCode != http.StatusNoContent {
		t.Fatalf("delete telegram status = %d, want 204", delResp.StatusCode)
	}
	delResp.Body.Close()

	afterResp, _ := http.Get(srv.URL + "/notifications/rules")
	var afterRules []map[string]any
	decodeBody(t, afterResp, &afterRules)
	if len(afterRules) != 0 {
		t.Fatalf("rules after target delete = %+v, want none (cascade)", afterRules)
	}
}

func containsString(xs []string, want string) bool {
	for _, x := range xs {
		if x == want {
			return true
		}
	}
	return false
}
