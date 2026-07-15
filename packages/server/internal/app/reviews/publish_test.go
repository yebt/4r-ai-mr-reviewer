package reviews

import (
	"context"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/webcloster-dev/ai-reviewer/internal/adapters/crypto"
	"github.com/webcloster-dev/ai-reviewer/internal/adapters/sqlite"
	"github.com/webcloster-dev/ai-reviewer/internal/app/accounts"
	"github.com/webcloster-dev/ai-reviewer/internal/app/providers"
	appRepos "github.com/webcloster-dev/ai-reviewer/internal/app/repos"
	"github.com/webcloster-dev/ai-reviewer/internal/domain/provider"
	"github.com/webcloster-dev/ai-reviewer/internal/domain/review"
	"github.com/webcloster-dev/ai-reviewer/internal/jobs"
	"github.com/webcloster-dev/ai-reviewer/internal/review/engine"
	"github.com/webcloster-dev/ai-reviewer/internal/review/skills"
)

// aGoDiff makes new-side lines 1..5 of a.go all ADDED lines, so fixtures that
// point a finding at a.go:3 anchor an inline discussion.
const aGoDiff = "@@ -0,0 +1,5 @@\n" +
	"+line one\n" +
	"+line two\n" +
	"+line three\n" +
	"+line four\n" +
	"+line five\n"

// bGoDiff makes new-side line 2 of b.go an ADDED line while lines 1, 3 and 4 are
// CONTEXT lines. A finding on b.go:3 must fall back to a general note.
const bGoDiff = "@@ -1,3 +1,4 @@\n" +
	" context one\n" +
	"+added two\n" +
	" context three\n" +
	" context four\n"

// recordingGitLab serves /changes (with diff_refs), counts posted notes and
// inline discussions, and captures their "body" form values so tests can assert
// the exact text posted.
type recordingGitLab struct {
	*httptest.Server
	mu               sync.Mutex
	notes            int
	discussions      int
	noteBodies       []string
	discussionBodies []string
}

func newRecordingGitLab(t *testing.T) *recordingGitLab {
	g := &recordingGitLab{}
	g.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasSuffix(r.URL.Path, "/changes"):
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"iid":           7,
				"title":         "Add login",
				"source_branch": "feat",
				"diff_refs":     map[string]any{"base_sha": "b", "start_sha": "s", "head_sha": "h"},
				"changes": []map[string]any{
					// a.go: lines 1..5 are all ADDED, so a finding on a.go:3
					// (used by the fixtures) anchors as an inline discussion.
					{"old_path": "a.go", "new_path": "a.go", "diff": aGoDiff},
					// b.go: line 2 is ADDED; lines 1/3/4 are CONTEXT lines. Lets a
					// test target a context line and assert the general-note fallback.
					{"old_path": "b.go", "new_path": "b.go", "diff": bGoDiff},
				},
			})
		case strings.HasSuffix(r.URL.Path, "/discussions"):
			body := r.FormValue("body")
			g.mu.Lock()
			g.discussions++
			g.discussionBodies = append(g.discussionBodies, body)
			g.mu.Unlock()
			w.WriteHeader(http.StatusCreated)
		case strings.HasSuffix(r.URL.Path, "/notes"):
			body := r.FormValue("body")
			g.mu.Lock()
			g.notes++
			g.noteBodies = append(g.noteBodies, body)
			g.mu.Unlock()
			w.WriteHeader(http.StatusCreated)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(g.Close)
	return g
}

func TestPublishSelectedFindings(t *testing.T) {
	ctx := context.Background()

	gl := newRecordingGitLab(t)
	// One inline finding (file+line) and one general finding (no line).
	reviewJSON := `{"summary":"issues","findings":[
		{"dimension":"risk","severity":"high","file":"a.go","line":3,"issue":"secret","blocking":true},
		{"dimension":"readability","severity":"low","file":"","line":0,"issue":"vague description"}
	]}`
	aiSrv := aiStub(t, reviewJSON)
	defer aiSrv.Close()

	db, err := sqlite.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer db.Close()

	salt, _ := crypto.NewSalt()
	key, _ := crypto.DeriveKey("pw", salt)
	cipher, _ := crypto.NewCipher(key)
	secrets := sqlite.NewSecretStore(db, cipher)
	accountSvc := accounts.NewService(sqlite.NewAccountRepo(db), secrets)
	providerSvc := providers.NewService(sqlite.NewProviderRepo(db), secrets)
	repoSvc := appRepos.NewService(sqlite.NewRepoStore(db), sqlite.NewAccountRepo(db), sqlite.NewProviderRepo(db))
	reviewStore := sqlite.NewReviewStore(db)

	acc, _ := accountSvc.Add(ctx, "acc", gl.URL, "token")
	prov, _ := providerSvc.Add(ctx, providers.AddInput{Name: "p", Kind: provider.KindOpenAICompat, BaseURL: aiSrv.URL, Model: "m", APIKey: "k"})
	rp, _ := repoSvc.Add(ctx, appRepos.AddInput{Name: "web", URL: "https://gitlab.test/group/project", AccountID: acc.ID, ProviderID: prov.ID})

	set, _ := skills.Load("")
	svc := NewService(reviewStore, sqlite.NewRepoStore(db), accountSvc, providerSvc, engine.New(set))
	runner := jobs.NewRunner(sqlite.NewJobStore(db), svc.Handle, jobs.WithLogger(log.New(io.Discard, "", 0)))
	svc.AttachRunner(runner)

	rv, _ := svc.Create(ctx, rp.ID, 7, review.ModeFast)
	runner.Drain(ctx)

	if err := svc.Publish(ctx, rv.ID, Selection{All: true}); err != nil {
		t.Fatalf("Publish: %v", err)
	}

	gl.mu.Lock()
	notes, discussions := gl.notes, gl.discussions
	gl.mu.Unlock()

	// 1 summary note + 1 note for the line-less finding = 2 notes; 1 inline.
	if notes != 2 || discussions != 1 {
		t.Fatalf("posts = %d notes / %d discussions, want 2 / 1", notes, discussions)
	}

	got, _ := reviewStore.Get(ctx, rv.ID)
	for i, f := range got.Findings {
		if !f.Published {
			t.Fatalf("finding %d not marked published", i)
		}
	}
}

func ptr[T any](v T) *T { return &v }

// setupPublishTest wires a done review ready to publish, returning the service,
// the recording GitLab stub, and the review id.
func setupPublishTest(t *testing.T) (context.Context, *Service, *recordingGitLab, string) {
	t.Helper()
	// One inline finding (file+line) and one general finding (no line).
	reviewJSON := `{"summary":"issues","findings":[
		{"dimension":"risk","severity":"high","file":"a.go","line":3,"issue":"secret","blocking":true},
		{"dimension":"readability","severity":"low","file":"","line":0,"issue":"vague description"}
	]}`
	return setupPublishTestJSON(t, reviewJSON)
}

// setupPublishTestJSON is setupPublishTest with a caller-supplied review payload,
// so tests can drive specific finding file/line combinations against the shared
// stub diff (a.go lines 1..5 added, b.go line 2 added / 3 context).
func setupPublishTestJSON(t *testing.T, reviewJSON string) (context.Context, *Service, *recordingGitLab, string) {
	t.Helper()
	ctx := context.Background()

	gl := newRecordingGitLab(t)
	aiSrv := aiStub(t, reviewJSON)
	defer aiSrv.Close()

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
	repoSvc := appRepos.NewService(sqlite.NewRepoStore(db), sqlite.NewAccountRepo(db), sqlite.NewProviderRepo(db))
	reviewStore := sqlite.NewReviewStore(db)

	acc, _ := accountSvc.Add(ctx, "acc", gl.URL, "token")
	prov, _ := providerSvc.Add(ctx, providers.AddInput{Name: "p", Kind: provider.KindOpenAICompat, BaseURL: aiSrv.URL, Model: "m", APIKey: "k"})
	rp, _ := repoSvc.Add(ctx, appRepos.AddInput{Name: "web", URL: "https://gitlab.test/group/project", AccountID: acc.ID, ProviderID: prov.ID})

	set, _ := skills.Load("")
	svc := NewService(reviewStore, sqlite.NewRepoStore(db), accountSvc, providerSvc, engine.New(set))
	runner := jobs.NewRunner(sqlite.NewJobStore(db), svc.Handle, jobs.WithLogger(log.New(io.Discard, "", 0)))
	svc.AttachRunner(runner)

	rv, _ := svc.Create(ctx, rp.ID, 7, review.ModeFast)
	runner.Drain(ctx)

	return ctx, svc, gl, rv.ID
}

// TestPublishAllPostsEachThingOnce verifies that a repeated "publish all"
// re-posts nothing: the summary is suppressed after the first publish and
// already-published findings are skipped, so the MR is never spammed.
func TestPublishAllPostsEachThingOnce(t *testing.T) {
	ctx, svc, gl, rvID := setupPublishTest(t)

	if err := svc.Publish(ctx, rvID, Selection{All: true}); err != nil {
		t.Fatalf("first Publish: %v", err)
	}
	gl.mu.Lock()
	notes, discussions := gl.notes, gl.discussions
	gl.mu.Unlock()
	// 1 summary note + 1 note for the line-less finding = 2 notes; 1 inline.
	if notes != 2 || discussions != 1 {
		t.Fatalf("after first publish = %d notes / %d discussions, want 2 / 1", notes, discussions)
	}

	// Second "publish all" with no override: summary suppressed AND both findings
	// already published, so nothing new posts.
	if err := svc.Publish(ctx, rvID, Selection{All: true}); err != nil {
		t.Fatalf("second Publish: %v", err)
	}
	gl.mu.Lock()
	notes, discussions = gl.notes, gl.discussions
	gl.mu.Unlock()
	if notes != 2 || discussions != 1 {
		t.Fatalf("after second publish = %d notes / %d discussions, want 2 / 1 (nothing re-posted)", notes, discussions)
	}
}

// TestPublishExplicitIndexRepostsPublished verifies that explicitly selecting a
// finding re-posts it even if already published — the deliberate re-selection
// escape hatch (unlike "publish all", which skips published findings).
func TestPublishExplicitIndexRepostsPublished(t *testing.T) {
	ctx, svc, gl, rvID := setupPublishTest(t)

	if err := svc.Publish(ctx, rvID, Selection{All: true}); err != nil {
		t.Fatalf("first Publish: %v", err)
	}
	gl.mu.Lock()
	notes := gl.notes
	gl.mu.Unlock()
	if notes != 2 {
		t.Fatalf("after first publish = %d notes, want 2", notes)
	}

	// Re-select the line-less finding (index 1) explicitly; it re-posts as a note.
	if err := svc.Publish(ctx, rvID, Selection{Indices: []int{1}, IncludeSummary: ptr(false)}); err != nil {
		t.Fatalf("re-select Publish: %v", err)
	}
	gl.mu.Lock()
	notes = gl.notes
	gl.mu.Unlock()
	// +1 general finding note; no summary (explicitly suppressed).
	if notes != 3 {
		t.Fatalf("after re-select = %d notes, want 3 (finding re-posted, no summary)", notes)
	}
}

// TestPublishSummaryReselectable verifies IncludeSummary=true re-posts the
// summary even after it was already posted once.
func TestPublishSummaryReselectable(t *testing.T) {
	ctx, svc, gl, rvID := setupPublishTest(t)

	if err := svc.Publish(ctx, rvID, Selection{All: true}); err != nil {
		t.Fatalf("first Publish: %v", err)
	}
	gl.mu.Lock()
	notes := gl.notes
	gl.mu.Unlock()
	if notes != 2 {
		t.Fatalf("after first publish = %d notes, want 2", notes)
	}

	// Explicit override re-posts the summary. The findings are already published,
	// and "All" skips them, so only the summary note is added.
	if err := svc.Publish(ctx, rvID, Selection{All: true, IncludeSummary: ptr(true)}); err != nil {
		t.Fatalf("second Publish: %v", err)
	}
	gl.mu.Lock()
	notes = gl.notes
	gl.mu.Unlock()
	// +1 summary note = 3 total (findings skipped, already published).
	if notes != 3 {
		t.Fatalf("after re-selected publish = %d notes, want 3 (summary re-posted, findings skipped)", notes)
	}
}

// TestPublishUsesFindingOverride verifies that a FindingOverrides entry replaces
// the generated body for that finding as-is (no dimension/severity header), while
// a non-overridden finding still uses formatFinding output.
func TestPublishUsesFindingOverride(t *testing.T) {
	ctx, svc, gl, rvID := setupPublishTest(t)

	// Index 0 is the inline finding (a.go:3). Override its body; leave index 1
	// (the line-less general finding) untouched.
	sel := Selection{
		All:              true,
		FindingOverrides: map[int]string{0: "humanized inline text"},
	}
	if err := svc.Publish(ctx, rvID, sel); err != nil {
		t.Fatalf("Publish: %v", err)
	}

	gl.mu.Lock()
	discussionBodies := append([]string(nil), gl.discussionBodies...)
	noteBodies := append([]string(nil), gl.noteBodies...)
	gl.mu.Unlock()

	// The overridden inline finding posts as a discussion with the exact override.
	if len(discussionBodies) != 1 {
		t.Fatalf("discussions = %d, want 1", len(discussionBodies))
	}
	if discussionBodies[0] != "humanized inline text" {
		t.Fatalf("discussion body = %q, want %q", discussionBodies[0], "humanized inline text")
	}
	if strings.Contains(discussionBodies[0], "Risk") {
		t.Fatalf("override body should not carry the dimension header, got %q", discussionBodies[0])
	}

	// The non-overridden general finding still uses formatFinding output. Notes are
	// [summary, general finding]; the finding note carries the dimension label.
	found := false
	for _, b := range noteBodies {
		if strings.Contains(b, "Readability") && strings.Contains(b, "vague description") {
			found = true
		}
	}
	if !found {
		t.Fatalf("non-overridden finding note not found in %q", noteBodies)
	}
}

// TestPublishUsesSummaryOverride verifies that SummaryOverride replaces the
// generated summary body as-is.
func TestPublishUsesSummaryOverride(t *testing.T) {
	ctx, svc, gl, rvID := setupPublishTest(t)

	sel := Selection{
		All:             true,
		SummaryOverride: ptr("humanized summary"),
	}
	if err := svc.Publish(ctx, rvID, sel); err != nil {
		t.Fatalf("Publish: %v", err)
	}

	gl.mu.Lock()
	noteBodies := append([]string(nil), gl.noteBodies...)
	gl.mu.Unlock()

	// Summary is the first note posted.
	if len(noteBodies) == 0 {
		t.Fatalf("no notes posted")
	}
	if noteBodies[0] != "humanized summary" {
		t.Fatalf("summary body = %q, want %q", noteBodies[0], "humanized summary")
	}
}

// TestPublishContextLineFallsBackToNote verifies that a finding whose line is a
// CONTEXT line (present in the diff but not a '+' line) is posted as a general
// note — never an inline discussion, which would 400 — and that the note body
// carries the "**File:** <file>:<line>" location so context isn't lost.
func TestPublishContextLineFallsBackToNote(t *testing.T) {
	// b.go:3 is a context line in the stub diff (only b.go:2 is added).
	reviewJSON := `{"summary":"issues","findings":[
		{"dimension":"risk","severity":"high","file":"b.go","line":3,"issue":"context finding"}
	]}`
	ctx, svc, gl, rvID := setupPublishTestJSON(t, reviewJSON)

	if err := svc.Publish(ctx, rvID, Selection{All: true}); err != nil {
		t.Fatalf("Publish: %v", err)
	}

	gl.mu.Lock()
	discussions := gl.discussions
	noteBodies := append([]string(nil), gl.noteBodies...)
	gl.mu.Unlock()

	if discussions != 0 {
		t.Fatalf("discussions = %d, want 0 (context line must not post inline)", discussions)
	}
	found := false
	for _, b := range noteBodies {
		if strings.Contains(b, "**File:** b.go:3") && strings.Contains(b, "context finding") {
			found = true
		}
	}
	if !found {
		t.Fatalf("no note with location header for b.go:3 in %q", noteBodies)
	}
}

// TestPublishLinelessFindingCarriesFile verifies that a finding with Line == 0
// but a non-empty File posts a general note whose body carries "**File:** <file>"
// (no ":line" suffix).
func TestPublishLinelessFindingCarriesFile(t *testing.T) {
	reviewJSON := `{"summary":"issues","findings":[
		{"dimension":"readability","severity":"low","file":"pkg/util.go","line":0,"issue":"whole-file note"}
	]}`
	ctx, svc, gl, rvID := setupPublishTestJSON(t, reviewJSON)

	if err := svc.Publish(ctx, rvID, Selection{All: true}); err != nil {
		t.Fatalf("Publish: %v", err)
	}

	gl.mu.Lock()
	discussions := gl.discussions
	noteBodies := append([]string(nil), gl.noteBodies...)
	gl.mu.Unlock()

	if discussions != 0 {
		t.Fatalf("discussions = %d, want 0", discussions)
	}
	found := false
	for _, b := range noteBodies {
		if strings.Contains(b, "**File:** pkg/util.go") && !strings.Contains(b, "pkg/util.go:") {
			found = true
		}
	}
	if !found {
		t.Fatalf("no note with bare file header for pkg/util.go in %q", noteBodies)
	}
}

// TestPublishAddedLinePostsInline verifies that a finding on a genuine ADDED line
// still posts as an inline discussion.
func TestPublishAddedLinePostsInline(t *testing.T) {
	// b.go:2 is the single added line in the stub diff.
	reviewJSON := `{"summary":"issues","findings":[
		{"dimension":"risk","severity":"high","file":"b.go","line":2,"issue":"added-line finding"}
	]}`
	ctx, svc, gl, rvID := setupPublishTestJSON(t, reviewJSON)

	if err := svc.Publish(ctx, rvID, Selection{All: true}); err != nil {
		t.Fatalf("Publish: %v", err)
	}

	gl.mu.Lock()
	discussions := gl.discussions
	discussionBodies := append([]string(nil), gl.discussionBodies...)
	gl.mu.Unlock()

	if discussions != 1 {
		t.Fatalf("discussions = %d, want 1 (added line must post inline)", discussions)
	}
	if !strings.Contains(discussionBodies[0], "added-line finding") {
		t.Fatalf("inline body = %q, want to contain %q", discussionBodies[0], "added-line finding")
	}
	if strings.Contains(discussionBodies[0], "**File:**") {
		t.Fatalf("inline discussion must not carry a location header, got %q", discussionBodies[0])
	}
}

func TestPublishRejectsUnfinishedReview(t *testing.T) {
	ctx := context.Background()
	db, err := sqlite.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer db.Close()

	salt, _ := crypto.NewSalt()
	key, _ := crypto.DeriveKey("pw", salt)
	cipher, _ := crypto.NewCipher(key)
	secrets := sqlite.NewSecretStore(db, cipher)
	accountSvc := accounts.NewService(sqlite.NewAccountRepo(db), secrets)
	providerSvc := providers.NewService(sqlite.NewProviderRepo(db), secrets)
	reviewStore := sqlite.NewReviewStore(db)

	// Seed an account + repo + a pending review directly.
	acc, _ := accountSvc.Add(ctx, "acc", "https://gitlab.test", "t")
	repoSvc := appRepos.NewService(sqlite.NewRepoStore(db), sqlite.NewAccountRepo(db), sqlite.NewProviderRepo(db))
	prov, _ := providerSvc.Add(ctx, providers.AddInput{Name: "p", Kind: provider.KindOpenAICompat, Model: "m", APIKey: "k"})
	rp, _ := repoSvc.Add(ctx, appRepos.AddInput{Name: "web", URL: "https://gitlab.test/g/p", AccountID: acc.ID, ProviderID: prov.ID})

	pending := review.Review{ID: "rv1", RepoID: rp.ID, MRIID: 1, Status: review.StatusPending}
	if err := reviewStore.Create(ctx, pending); err != nil {
		t.Fatalf("seed review: %v", err)
	}

	set, _ := skills.Load("")
	svc := NewService(reviewStore, sqlite.NewRepoStore(db), accountSvc, providerSvc, engine.New(set))
	if err := svc.Publish(ctx, "rv1", Selection{All: true}); err == nil {
		t.Fatal("expected error publishing a non-done review")
	}
}
