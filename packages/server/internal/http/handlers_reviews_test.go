package httpapi

import (
	"context"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"github.com/webcloster-dev/ai-reviewer/internal/adapters/crypto"
	"github.com/webcloster-dev/ai-reviewer/internal/adapters/sqlite"
	"github.com/webcloster-dev/ai-reviewer/internal/app/accounts"
	apprepos "github.com/webcloster-dev/ai-reviewer/internal/app/repos"
	"github.com/webcloster-dev/ai-reviewer/internal/app/reviews"
	"github.com/webcloster-dev/ai-reviewer/internal/domain/review"
	"github.com/webcloster-dev/ai-reviewer/internal/id"
	"github.com/webcloster-dev/ai-reviewer/internal/jobs"
	"github.com/webcloster-dev/ai-reviewer/internal/review/engine"
	"github.com/webcloster-dev/ai-reviewer/internal/review/skills"
)

// reviewsTestHarness wires a full Server over a real sqlite DB and exposes the
// review store directly, so a test can seed reviews (including archived ones)
// by writing straight to the store instead of depending on the async job
// runner actually executing a created review to a terminal status.
type reviewsTestHarness struct {
	srv   *httptest.Server
	store *sqlite.ReviewStore
}

func newReviewsTestHarness(t *testing.T) reviewsTestHarness {
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
	repoSvc := apprepos.NewService(sqlite.NewRepoStore(db), sqlite.NewAccountRepo(db), sqlite.NewProviderRepo(db))
	reviewStore := sqlite.NewReviewStore(db)
	set, _ := skills.Load("")
	reviewSvc := reviews.NewService(reviewStore, sqlite.NewRepoStore(db), accountSvc, nil, engine.New(set), 0)
	runner := jobs.NewRunner(sqlite.NewJobStore(db), reviewSvc.Handle, jobs.WithLogger(log.New(io.Discard, "", 0)))
	reviewSvc.AttachRunner(runner)

	srv := httptest.NewServer(NewServer(accountSvc, nil, nil, repoSvc, reviewSvc, nil, nil, nil, nil, nil, set, nil, "", nil, false, nil).Routes())
	t.Cleanup(srv.Close)
	return reviewsTestHarness{srv: srv, store: reviewStore}
}

// addRepo creates an account + repo directly through the wired services and
// returns the repo id.
func (h reviewsTestHarness) addRepo(t *testing.T, name string) string {
	t.Helper()
	acctResp := postJSON(t, h.srv.URL+"/accounts", map[string]any{"name": name + "-acct", "baseUrl": "https://gitlab.com", "token": "t"})
	var acct struct{ ID string }
	decodeBody(t, acctResp, &acct)

	repoResp := postJSON(t, h.srv.URL+"/repos", map[string]any{"name": name, "url": "https://gitlab.com/g/" + name, "accountId": acct.ID})
	if repoResp.StatusCode != http.StatusCreated {
		t.Fatalf("create repo %q status = %d, want 201", name, repoResp.StatusCode)
	}
	var repoObj struct{ ID string }
	decodeBody(t, repoResp, &repoObj)
	return repoObj.ID
}

// seedReview writes a review straight to the store (bypassing the async job
// runner) and returns its id. archived, when true, is applied via
// SetArchived right after creation.
func (h reviewsTestHarness) seedReview(t *testing.T, repoID string, mrIID int, archived bool) string {
	t.Helper()
	ctx := context.Background()
	rv := review.Review{ID: id.New(), RepoID: repoID, MRIID: mrIID, Status: review.StatusPending}
	if err := h.store.Create(ctx, rv); err != nil {
		t.Fatalf("seed review: create: %v", err)
	}
	if archived {
		if err := h.store.SetArchived(ctx, rv.ID, true); err != nil {
			t.Fatalf("seed review: set archived: %v", err)
		}
	}
	return rv.ID
}

// TestListRecentReviewsOverHTTP verifies GET /reviews returns reviews newest
// first, spanning multiple repos, with a best-effort repoName on each item —
// while the per-repo /repos/{id}/reviews path stays unaffected (no repoName).
func TestListRecentReviewsOverHTTP(t *testing.T) {
	h := newReviewsTestHarness(t)
	repoA := h.addRepo(t, "repo-a")
	repoB := h.addRepo(t, "repo-b")

	h.seedReview(t, repoA, 1, false)
	time.Sleep(2 * time.Millisecond)
	h.seedReview(t, repoB, 2, false)

	resp, err := http.Get(h.srv.URL + "/reviews")
	if err != nil {
		t.Fatalf("GET reviews: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("list recent reviews status = %d, want 200", resp.StatusCode)
	}
	var list []struct {
		MRIID    int    `json:"mrIid"`
		RepoID   string `json:"repoId"`
		RepoName string `json:"repoName"`
	}
	decodeBody(t, resp, &list)
	if len(list) != 2 {
		t.Fatalf("recent reviews len = %d, want 2", len(list))
	}
	// Newest first: the repoB (mrIid=2) review was created last.
	if list[0].MRIID != 2 || list[0].RepoName != "repo-b" {
		t.Fatalf("list[0] = %+v, want mrIid 2 repoName repo-b", list[0])
	}
	if list[1].MRIID != 1 || list[1].RepoName != "repo-a" {
		t.Fatalf("list[1] = %+v, want mrIid 1 repoName repo-a", list[1])
	}

	// The per-repo path does not carry repoName.
	perRepoResp, err := http.Get(h.srv.URL + "/repos/" + repoA + "/reviews")
	if err != nil {
		t.Fatalf("GET repo reviews: %v", err)
	}
	var perRepo []map[string]any
	decodeBody(t, perRepoResp, &perRepo)
	if len(perRepo) != 1 {
		t.Fatalf("per-repo reviews len = %d, want 1", len(perRepo))
	}
	if _, ok := perRepo[0]["repoName"]; ok {
		t.Errorf("per-repo review carries repoName = %v, want omitted", perRepo[0]["repoName"])
	}
}

// TestListRecentReviewsKeysetPaginationOverHTTP seeds more reviews than a
// page (limit=2) and pages through GET /reviews with the returned
// X-Next-Cursor, asserting: newest-first order, no duplicates, no gaps, and
// the last page omits the header. It also proves ?archived= scopes the same
// keyset pagination to archived-only reviews, seeded independently of the
// active ones.
func TestListRecentReviewsKeysetPaginationOverHTTP(t *testing.T) {
	h := newReviewsTestHarness(t)
	repoID := h.addRepo(t, "web")

	const total = 5
	const limit = 2
	created := make([]string, 0, total)
	for i := 0; i < total; i++ {
		created = append(created, h.seedReview(t, repoID, 100+i, false))
		time.Sleep(2 * time.Millisecond)
	}

	seen := paginateReviews(t, h.srv.URL, "", limit)
	if len(seen) != total {
		t.Fatalf("paginated over %d reviews, want %d (no gaps/duplicates): %v", len(seen), total, seen)
	}
	seenSet := make(map[string]int, len(seen))
	for _, rid := range seen {
		seenSet[rid]++
	}
	for _, rid := range created {
		if seenSet[rid] != 1 {
			t.Errorf("review %s seen %d times, want exactly 1", rid, seenSet[rid])
		}
	}
	// Newest-first: pagination order is the reverse of creation order.
	for i, rid := range seen {
		want := created[total-1-i]
		if rid != want {
			t.Fatalf("seen[%d] = %s, want %s (newest-first across pages)", i, rid, want)
		}
	}

	// ?archived= scopes the same pagination to archived-only reviews, seeded
	// independently so this is a genuine filter check, not a status flip.
	const archivedTotal = 3
	archivedIDs := make([]string, 0, archivedTotal)
	for i := 0; i < archivedTotal; i++ {
		archivedIDs = append(archivedIDs, h.seedReview(t, repoID, 200+i, true))
		time.Sleep(2 * time.Millisecond)
	}

	seenArchived := paginateReviews(t, h.srv.URL, "archived=1", limit)
	if len(seenArchived) != archivedTotal {
		t.Fatalf("archived pagination len = %d, want %d: %v", len(seenArchived), archivedTotal, seenArchived)
	}
	for i, rid := range seenArchived {
		want := archivedIDs[archivedTotal-1-i]
		if rid != want {
			t.Fatalf("archived seen[%d] = %s, want %s (newest-first)", i, rid, want)
		}
	}
	// None of the active reviews leak into the archived listing.
	activeSet := make(map[string]bool, len(created))
	for _, rid := range created {
		activeSet[rid] = true
	}
	for _, rid := range seenArchived {
		if activeSet[rid] {
			t.Fatalf("archived listing leaked active review %s", rid)
		}
	}
}

// paginateReviews walks GET {baseURL}/reviews?{extraQuery}&limit=N following
// X-Next-Cursor until a page omits the header, returning every review id seen
// in response order.
func paginateReviews(t *testing.T, baseURL, extraQuery string, limit int) []string {
	t.Helper()
	var seen []string
	cursor := ""
	for page := 0; ; page++ {
		if page > 50 {
			t.Fatalf("pagination did not terminate after %d pages", page)
		}
		url := baseURL + "/reviews?limit=" + strconv.Itoa(limit)
		if extraQuery != "" {
			url += "&" + extraQuery
		}
		if cursor != "" {
			url += "&cursor=" + cursor
		}
		resp, err := http.Get(url)
		if err != nil {
			t.Fatalf("GET reviews page %d: %v", page, err)
		}
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("page %d status = %d, want 200", page, resp.StatusCode)
		}
		var list []struct {
			ID string `json:"id"`
		}
		decodeBody(t, resp, &list)
		next := resp.Header.Get("X-Next-Cursor")
		if len(list) > limit {
			t.Fatalf("page %d returned %d rows, want at most %d", page, len(list), limit)
		}
		for _, it := range list {
			seen = append(seen, it.ID)
		}
		if next == "" {
			if len(list) == limit {
				t.Fatalf("page %d was full (%d rows) but omitted X-Next-Cursor", page, len(list))
			}
			break
		}
		cursor = next
	}
	return seen
}

// TestListRecentReviewsBadCursorOverHTTP maps a malformed ?cursor to 400.
func TestListRecentReviewsBadCursorOverHTTP(t *testing.T) {
	h := newReviewsTestHarness(t)
	resp, err := http.Get(h.srv.URL + "/reviews?cursor=not-a-valid-cursor!!")
	if err != nil {
		t.Fatalf("GET reviews: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("bad cursor status = %d, want 400", resp.StatusCode)
	}
}
