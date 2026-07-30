package httpapi

import (
	"bytes"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/webcloster-dev/ai-reviewer/internal/adapters/crypto"
	"github.com/webcloster-dev/ai-reviewer/internal/adapters/sqlite"
	"github.com/webcloster-dev/ai-reviewer/internal/app/accounts"
	"github.com/webcloster-dev/ai-reviewer/internal/app/bot"
	apphumanize "github.com/webcloster-dev/ai-reviewer/internal/app/humanize"
	"github.com/webcloster-dev/ai-reviewer/internal/app/notifications"
	"github.com/webcloster-dev/ai-reviewer/internal/app/profiles"
	"github.com/webcloster-dev/ai-reviewer/internal/app/providers"
	apprepos "github.com/webcloster-dev/ai-reviewer/internal/app/repos"
	"github.com/webcloster-dev/ai-reviewer/internal/app/reviews"
	"github.com/webcloster-dev/ai-reviewer/internal/app/routines"
	apptelegram "github.com/webcloster-dev/ai-reviewer/internal/app/telegram"
	"github.com/webcloster-dev/ai-reviewer/internal/auth"
	"github.com/webcloster-dev/ai-reviewer/internal/jobs"
	"github.com/webcloster-dev/ai-reviewer/internal/review/engine"
	"github.com/webcloster-dev/ai-reviewer/internal/review/skills"
)

func newTestServer(t *testing.T) *httptest.Server {
	t.Helper()
	return newTestServerFull(t, "", "")
}

// newTestServerWithSecret wires a full test server, gating the Telegram webhook
// with the given secret ("" keeps the receiver dormant).
func newTestServerWithSecret(t *testing.T, webhookSecret string) *httptest.Server {
	t.Helper()
	return newTestServerFull(t, webhookSecret, "")
}

// newTestServerWithAuth wires a full test server with API auth enabled under the
// given password ("" disables it).
func newTestServerWithAuth(t *testing.T, authPassword string) *httptest.Server {
	t.Helper()
	return newTestServerFull(t, "", authPassword)
}

// newTestServerFull wires a full test server, gating the Telegram webhook with
// webhookSecret and the API with authPassword (both "" leave them off). Session
// tokens use a one-hour lifetime.
func newTestServerFull(t *testing.T, webhookSecret, authPassword string) *httptest.Server {
	t.Helper()
	return newTestServerFullWithLifetime(t, webhookSecret, authPassword, time.Hour)
}

// newTestServerWithAuthLifetime wires an auth-enabled test server whose session
// tokens expire after lifetime. A past/negative lifetime yields already-expired
// tokens, exercising the expiry-rejection path without sleeping.
func newTestServerWithAuthLifetime(t *testing.T, authPassword string, lifetime time.Duration) *httptest.Server {
	t.Helper()
	return newTestServerFullWithLifetime(t, "", authPassword, lifetime)
}

// newTestServerFullWithLifetime is newTestServerFull with an explicit session
// lifetime seam.
func newTestServerFullWithLifetime(t *testing.T, webhookSecret, authPassword string, lifetime time.Duration) *httptest.Server {
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
	reviewSvc := reviews.NewService(sqlite.NewReviewStore(db), sqlite.NewRepoStore(db), accountSvc, providerSvc, engine.New(set), 0)
	routinesSvc := routines.NewService(sqlite.NewRepoStore(db), accountSvc, sqlite.NewRoutineRunStore(db), 10*time.Minute, nil, log.New(io.Discard, "", 0))
	humanizeSvc := apphumanize.NewService(sqlite.NewReviewStore(db), sqlite.NewProfileStore(db), sqlite.NewHumanizationStore(db), providerSvc, log.New(io.Discard, "", 0))
	telegramSvc := apptelegram.NewService(sqlite.NewTelegramStore(db), secrets)
	notificationsSvc := notifications.NewService(sqlite.NewNotificationRuleStore(db), telegramSvc)
	runner := jobs.NewRunner(sqlite.NewJobStore(db), reviewSvc.Handle, jobs.WithLogger(log.New(io.Discard, "", 0)))
	reviewSvc.AttachRunner(runner)
	botSvc := bot.NewService(bot.NewAPIClient(), telegramSvc, reviewSvc, repoSvc)

	authMgr := auth.NewManager(authPassword, lifetime)
	srv := httptest.NewServer(NewServer(accountSvc, providerSvc, profileSvc, repoSvc, reviewSvc, routinesSvc, humanizeSvc, telegramSvc, notificationsSvc, set, botSvc, webhookSecret, authMgr, false).Routes())
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

// TestNewServerNilAuthManagerPassthrough is the FIX-5 regression: a Server built
// with a nil *auth.Manager must substitute a disabled manager so s.auth is never
// nil and the session gate passes every request through instead of nil-panicking.
func TestNewServerNilAuthManagerPassthrough(t *testing.T) {
	var set skills.Set
	s := NewServer(nil, nil, nil, nil, nil, nil, nil, nil, nil, set, nil, "", nil, false)
	if s.auth == nil {
		t.Fatal("NewServer left s.auth nil for a nil authMgr; want a substituted disabled manager")
	}

	gated := s.requireAuth(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	rec := httptest.NewRecorder()
	gated.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/accounts", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("nil auth manager passthrough = %d, want 200", rec.Code)
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

// TestGetReviewReasoningsField pins the wire contract for the reasonings DTO
// field: GET /reviews/{id} must expose it, shaped [{phase, content}], and it
// must be an empty array (never null) when no reasoning was captured.
func TestGetReviewReasoningsField(t *testing.T) {
	srv := newTestServer(t)

	acctResp := postJSON(t, srv.URL+"/accounts", map[string]any{"name": "a", "baseUrl": "https://gitlab.com", "token": "t"})
	var acct struct{ ID string }
	decodeBody(t, acctResp, &acct)

	provResp := postJSON(t, srv.URL+"/providers", map[string]any{"name": "p", "kind": "openai-compat", "model": "m", "apiKey": "k"})
	var prov struct{ ID string }
	decodeBody(t, provResp, &prov)

	repoResp := postJSON(t, srv.URL+"/repos", map[string]any{"name": "web", "url": "https://gitlab.com/g/p", "accountId": acct.ID, "providerId": prov.ID})
	var repoObj struct{ ID string }
	decodeBody(t, repoResp, &repoObj)

	revResp := postJSON(t, srv.URL+"/reviews", map[string]any{"repoId": repoObj.ID, "mrIid": 7, "mode": "fast"})
	var rev struct {
		ID string `json:"id"`
	}
	decodeBody(t, revResp, &rev)

	getResp, err := http.Get(srv.URL + "/reviews/" + rev.ID)
	if err != nil {
		t.Fatalf("GET /reviews/{id}: %v", err)
	}
	if getResp.StatusCode != http.StatusOK {
		t.Fatalf("get review status = %d, want 200", getResp.StatusCode)
	}

	// Inspect the raw JSON so null and [] are distinguishable.
	var raw map[string]json.RawMessage
	decodeBody(t, getResp, &raw)
	rawReasonings, ok := raw["reasonings"]
	if !ok {
		t.Fatal("review response is missing the reasonings field")
	}
	if string(rawReasonings) != "[]" {
		t.Fatalf("reasonings = %s, want [] (empty array, not null) when none captured", rawReasonings)
	}

	// It must decode into the documented [{phase, content}] shape.
	var reasonings []struct {
		Phase   string `json:"phase"`
		Content string `json:"content"`
	}
	if err := json.Unmarshal(rawReasonings, &reasonings); err != nil {
		t.Fatalf("reasonings not shaped [{phase, content}]: %v", err)
	}
	if len(reasonings) != 0 {
		t.Fatalf("reasonings len = %d, want 0", len(reasonings))
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

func TestTelegramUpdateOverHTTP(t *testing.T) {
	srv := newTestServer(t)

	resp := postJSON(t, srv.URL+"/telegram", map[string]any{"name": "team", "botToken": "bot-secret", "chatId": "-100"})
	var created map[string]any
	decodeBody(t, resp, &created)
	id, _ := created["id"].(string)

	// Update name/chat, omit the token (it must be kept server-side).
	upd := sendJSON(t, http.MethodPut, srv.URL+"/telegram/"+id, map[string]any{
		"name": "renamed", "chatId": "-200", "threadId": "9", "isDefault": true,
	})
	if upd.StatusCode != http.StatusOK {
		t.Fatalf("update status = %d, want 200", upd.StatusCode)
	}
	var updated map[string]any
	decodeBody(t, upd, &updated)
	if updated["name"] != "renamed" || updated["chatId"] != "-200" || updated["threadId"] != "9" {
		t.Fatalf("update did not persist fields: %+v", updated)
	}
	if updated["isDefault"] != true {
		t.Fatalf("isDefault = %v, want true", updated["isDefault"])
	}
	// The response must never leak a token.
	if _, leaked := updated["botToken"]; leaked {
		t.Fatal("update response leaked the bot token")
	}
	if _, leaked := updated["token"]; leaked {
		t.Fatal("update response leaked the token")
	}
}

func TestTelegramUpdateUnknownIs404(t *testing.T) {
	srv := newTestServer(t)
	resp := sendJSON(t, http.MethodPut, srv.URL+"/telegram/nope", map[string]any{"name": "x", "chatId": "1"})
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("update unknown status = %d, want 404", resp.StatusCode)
	}
}

func TestTelegramDuplicateOverHTTP(t *testing.T) {
	srv := newTestServer(t)

	resp := postJSON(t, srv.URL+"/telegram", map[string]any{"name": "team", "botToken": "bot-secret", "chatId": "-100", "isDefault": true})
	var created map[string]any
	decodeBody(t, resp, &created)
	id, _ := created["id"].(string)

	dup := postJSON(t, srv.URL+"/telegram/"+id+"/duplicate", nil)
	if dup.StatusCode != http.StatusCreated {
		t.Fatalf("duplicate status = %d, want 201", dup.StatusCode)
	}
	var copyTarget map[string]any
	decodeBody(t, dup, &copyTarget)
	if copyTarget["id"] == created["id"] {
		t.Fatal("duplicate must have a new id")
	}
	if copyTarget["name"] != "team (copy)" {
		t.Fatalf("name = %v, want 'team (copy)'", copyTarget["name"])
	}
	if copyTarget["isDefault"] != false || copyTarget["isBot"] != false {
		t.Fatalf("copy must not be default or bot: %+v", copyTarget)
	}
	if _, leaked := copyTarget["botToken"]; leaked {
		t.Fatal("duplicate response leaked the bot token")
	}

	// Two targets now exist.
	listResp, _ := http.Get(srv.URL + "/telegram")
	var list []map[string]any
	decodeBody(t, listResp, &list)
	if len(list) != 2 {
		t.Fatalf("list len = %d, want 2 after duplicate", len(list))
	}
}

func TestTelegramDuplicateUnknownIs404(t *testing.T) {
	srv := newTestServer(t)
	resp := postJSON(t, srv.URL+"/telegram/nope/duplicate", nil)
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("duplicate unknown status = %d, want 404", resp.StatusCode)
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

// postWebhook posts a raw JSON body to the webhook with an optional secret
// header, so the gate can be exercised with matching and mismatching tokens.
func postWebhook(t *testing.T, url, secretHeader, body string) *http.Response {
	t.Helper()
	req, err := http.NewRequest(http.MethodPost, url, strings.NewReader(body))
	if err != nil {
		t.Fatalf("build webhook request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if secretHeader != "" {
		req.Header.Set("X-Telegram-Bot-Api-Secret-Token", secretHeader)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("POST webhook: %v", err)
	}
	return resp
}

const validReposUpdate = `{"update_id":1,"message":{"message_id":1,"chat":{"id":100},"from":{"id":5},"text":"/repos"}}`

func TestTelegramWebhook(t *testing.T) {
	// Dormant: an empty secret makes the receiver inert — any POST is a 200 and
	// nothing is processed (even without a secret header).
	dormant := newTestServer(t)
	resp := postWebhook(t, dormant.URL+"/telegram/webhook", "", validReposUpdate)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("dormant webhook status = %d, want 200", resp.StatusCode)
	}
	resp.Body.Close()

	// Enabled: a configured secret gates the endpoint.
	srv := newTestServerWithSecret(t, "s3cret")

	// Wrong header → 401.
	bad := postWebhook(t, srv.URL+"/telegram/webhook", "nope", validReposUpdate)
	if bad.StatusCode != http.StatusUnauthorized {
		t.Fatalf("bad-secret webhook status = %d, want 401", bad.StatusCode)
	}
	bad.Body.Close()

	// Correct header + a valid update → 200 (dispatch runs on a goroutine).
	ok := postWebhook(t, srv.URL+"/telegram/webhook", "s3cret", validReposUpdate)
	if ok.StatusCode != http.StatusOK {
		t.Fatalf("valid webhook status = %d, want 200", ok.StatusCode)
	}
	ok.Body.Close()

	// Malformed body with the correct secret → 400.
	bad400 := postWebhook(t, srv.URL+"/telegram/webhook", "s3cret", "{not json")
	if bad400.StatusCode != http.StatusBadRequest {
		t.Fatalf("malformed webhook status = %d, want 400", bad400.StatusCode)
	}
	bad400.Body.Close()
}

func TestTelegramSetBot(t *testing.T) {
	srv := newTestServer(t)

	resp := postJSON(t, srv.URL+"/telegram", map[string]any{"name": "bot", "botToken": "bot-secret", "chatId": "-100"})
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("create telegram status = %d, want 201", resp.StatusCode)
	}
	var created map[string]any
	decodeBody(t, resp, &created)
	if created["isBot"] != false {
		t.Fatalf("new target isBot = %v, want false", created["isBot"])
	}
	id, _ := created["id"].(string)

	botResp := postJSON(t, srv.URL+"/telegram/"+id+"/bot", nil)
	if botResp.StatusCode != http.StatusOK {
		t.Fatalf("set bot status = %d, want 200", botResp.StatusCode)
	}
	botResp.Body.Close()

	listResp, _ := http.Get(srv.URL + "/telegram")
	var list []map[string]any
	decodeBody(t, listResp, &list)
	if len(list) != 1 || list[0]["isBot"] != true {
		t.Fatalf("telegram list after set bot = %+v, want single target with isBot true", list)
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
