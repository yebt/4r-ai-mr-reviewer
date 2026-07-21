package bot

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/webcloster-dev/ai-reviewer/internal/adapters/gitlab"
	tgapi "github.com/webcloster-dev/ai-reviewer/internal/adapters/telegram"
	"github.com/webcloster-dev/ai-reviewer/internal/domain/repo"
	"github.com/webcloster-dev/ai-reviewer/internal/domain/review"
	tgdomain "github.com/webcloster-dev/ai-reviewer/internal/domain/telegram"
)

// --- fakes ---

type sentMessage struct {
	token    string
	chatID   string
	text     string
	keyboard [][]tgapi.InlineButton
}

type fakeTelegram struct {
	sent    []sentMessage
	answers int
}

func (f *fakeTelegram) SendMessageHTML(_ context.Context, token, chatID, _, text string, keyboard [][]tgapi.InlineButton) error {
	f.sent = append(f.sent, sentMessage{token: token, chatID: chatID, text: text, keyboard: keyboard})
	return nil
}

func (f *fakeTelegram) AnswerCallbackQuery(_ context.Context, _, _, _ string) error {
	f.answers++
	return nil
}

type fakeConfig struct {
	token   string
	allowed []string
	noBot   bool
}

func (c fakeConfig) BotToken(context.Context) (string, tgdomain.Target, error) {
	if c.noBot {
		return "", tgdomain.Target{}, tgdomain.ErrNotFound
	}
	return c.token, tgdomain.Target{ChatID: "1"}, nil
}

func (c fakeConfig) AllowedChatIDs(context.Context) ([]string, error) {
	return c.allowed, nil
}

type fakeRepos struct{ repos []repo.Repo }

func (r fakeRepos) List(context.Context) ([]repo.Repo, error) { return r.repos, nil }

type createCall struct {
	repoID string
	mrIID  int
	mode   review.ContextMode
}

type fakeReviews struct {
	byRepo    map[string][]review.Review
	byID      map[string]review.Review
	mrs       map[string][]gitlab.MergeRequest
	created   []createCall
	createErr error // when set, Create fails (records the call first)
	getErr    error // when set, Get fails
}

func (r *fakeReviews) List(_ context.Context, repoID string) ([]review.Review, error) {
	return r.byRepo[repoID], nil
}

func (r *fakeReviews) Get(_ context.Context, reviewID string) (review.Review, error) {
	if r.getErr != nil {
		return review.Review{}, r.getErr
	}
	rv, ok := r.byID[reviewID]
	if !ok {
		return review.Review{}, review.ErrNotFound
	}
	return rv, nil
}

func (r *fakeReviews) Create(_ context.Context, repoID string, mrIID int, mode review.ContextMode, _, _ string) (review.Review, error) {
	r.created = append(r.created, createCall{repoID: repoID, mrIID: mrIID, mode: mode})
	if r.createErr != nil {
		return review.Review{}, r.createErr
	}
	rv := review.Review{ID: "rvnew", RepoID: repoID, MRIID: mrIID, ContextMode: mode, Status: review.StatusPending}
	return rv, nil
}

func (r *fakeReviews) ListOpenMergeRequests(_ context.Context, repoID string) ([]gitlab.MergeRequest, error) {
	return r.mrs[repoID], nil
}

// --- helpers ---

func msgUpdate(chatID int64, text string) tgapi.Update {
	return tgapi.Update{Message: &tgapi.Message{Chat: &tgapi.Chat{ID: chatID}, Text: text}}
}

func cbUpdate(chatID int64, data string) tgapi.Update {
	return tgapi.Update{CallbackQuery: &tgapi.CallbackQuery{ID: "cb1", Message: &tgapi.Message{Chat: &tgapi.Chat{ID: chatID}}, Data: data}}
}

func newService(tg *fakeTelegram, cfg fakeConfig, rv *fakeReviews, rp fakeRepos) *Service {
	return NewService(tg, cfg, rv, rp)
}

// --- tests ---

func TestUnauthorizedChatIsIgnored(t *testing.T) {
	tg := &fakeTelegram{}
	cfg := fakeConfig{token: "T", allowed: []string{"100"}}
	svc := newService(tg, cfg, &fakeReviews{}, fakeRepos{})

	if err := svc.HandleUpdate(context.Background(), msgUpdate(999, "/repos")); err != nil {
		t.Fatalf("HandleUpdate: %v", err)
	}
	if len(tg.sent) != 0 {
		t.Fatalf("unauthorized chat produced %d sends, want 0", len(tg.sent))
	}
}

func TestDormantBotIgnoresUpdate(t *testing.T) {
	tg := &fakeTelegram{}
	cfg := fakeConfig{allowed: []string{"100"}, noBot: true}
	svc := newService(tg, cfg, &fakeReviews{}, fakeRepos{})

	if err := svc.HandleUpdate(context.Background(), msgUpdate(100, "/repos")); err != nil {
		t.Fatalf("HandleUpdate: %v", err)
	}
	if len(tg.sent) != 0 {
		t.Fatalf("dormant bot produced %d sends, want 0", len(tg.sent))
	}
}

func TestReposCommandListsRepos(t *testing.T) {
	tg := &fakeTelegram{}
	cfg := fakeConfig{token: "T", allowed: []string{"100"}}
	rp := fakeRepos{repos: []repo.Repo{{ID: "r1", Name: "web"}, {ID: "r2", Name: "api"}}}
	svc := newService(tg, cfg, &fakeReviews{}, rp)

	if err := svc.HandleUpdate(context.Background(), msgUpdate(100, "/repos")); err != nil {
		t.Fatalf("HandleUpdate: %v", err)
	}
	if len(tg.sent) != 1 {
		t.Fatalf("sends = %d, want 1", len(tg.sent))
	}
	m := tg.sent[0]
	if m.token != "T" || m.chatID != "100" {
		t.Fatalf("send addressed to token=%q chat=%q, want T/100", m.token, m.chatID)
	}
	if len(m.keyboard) != 2 {
		t.Fatalf("keyboard rows = %d, want 2", len(m.keyboard))
	}
	if m.keyboard[0][0].Data != "repo:r1" || m.keyboard[1][0].Data != "repo:r2" {
		t.Fatalf("unexpected repo button data: %q, %q", m.keyboard[0][0].Data, m.keyboard[1][0].Data)
	}
}

func TestRepoCallbackListsMergeRequests(t *testing.T) {
	tg := &fakeTelegram{}
	cfg := fakeConfig{token: "T", allowed: []string{"100"}}
	rv := &fakeReviews{mrs: map[string][]gitlab.MergeRequest{
		"r1": {{IID: 7, Title: "Add feature"}},
	}}
	svc := newService(tg, cfg, rv, fakeRepos{})

	if err := svc.HandleUpdate(context.Background(), cbUpdate(100, "repo:r1")); err != nil {
		t.Fatalf("HandleUpdate: %v", err)
	}
	if tg.answers != 1 {
		t.Fatalf("answers = %d, want 1 (callback acknowledged)", tg.answers)
	}
	if len(tg.sent) != 1 {
		t.Fatalf("sends = %d, want 1", len(tg.sent))
	}
	kb := tg.sent[0].keyboard
	// One MR button + a "Recent reviews" button.
	if len(kb) != 2 {
		t.Fatalf("keyboard rows = %d, want 2", len(kb))
	}
	if kb[0][0].Data != "mr:r1:7" {
		t.Fatalf("MR button data = %q, want mr:r1:7", kb[0][0].Data)
	}
	if kb[1][0].Data != "rr:r1" {
		t.Fatalf("last button data = %q, want rr:r1", kb[1][0].Data)
	}
}

func TestGoCallbackCreatesReview(t *testing.T) {
	tg := &fakeTelegram{}
	cfg := fakeConfig{token: "T", allowed: []string{"100"}}
	rv := &fakeReviews{}
	svc := newService(tg, cfg, rv, fakeRepos{})

	if err := svc.HandleUpdate(context.Background(), cbUpdate(100, "go:r1:7:fast")); err != nil {
		t.Fatalf("HandleUpdate: %v", err)
	}
	if len(rv.created) != 1 {
		t.Fatalf("create calls = %d, want 1", len(rv.created))
	}
	c := rv.created[0]
	if c.repoID != "r1" || c.mrIID != 7 || c.mode != review.ModeFast {
		t.Fatalf("create call = %+v, want {r1 7 fast}", c)
	}
	if len(tg.sent) != 1 {
		t.Fatalf("sends = %d, want 1", len(tg.sent))
	}
	kb := tg.sent[0].keyboard
	if len(kb) != 1 || kb[0][0].Data != "rv:rvnew" {
		t.Fatalf("expected a View button rv:rvnew, got %+v", kb)
	}
}

func TestGoCallbackDeepMode(t *testing.T) {
	tg := &fakeTelegram{}
	cfg := fakeConfig{token: "T", allowed: []string{"100"}}
	rv := &fakeReviews{}
	svc := newService(tg, cfg, rv, fakeRepos{})

	if err := svc.HandleUpdate(context.Background(), cbUpdate(100, "go:r1:9:deep")); err != nil {
		t.Fatalf("HandleUpdate: %v", err)
	}
	if len(rv.created) != 1 || rv.created[0].mode != review.ModeDeep {
		t.Fatalf("expected one deep create, got %+v", rv.created)
	}
}

func TestReviewCallbackRendersContent(t *testing.T) {
	tg := &fakeTelegram{}
	cfg := fakeConfig{token: "T", allowed: []string{"100"}}
	rv := &fakeReviews{byID: map[string]review.Review{
		"rv1": {
			ID:             "rv1",
			MRIID:          7,
			Status:         review.StatusDone,
			Recommendation: review.RequestChanges,
			Score:          85,
			Summary:        "Looks risky",
			Findings: []review.Finding{
				{Dimension: review.Risk, Severity: review.SeverityHigh, File: "a.go", Line: 12, Issue: "SQL injection"},
			},
		},
	}}
	svc := newService(tg, cfg, rv, fakeRepos{})

	if err := svc.HandleUpdate(context.Background(), cbUpdate(100, "rv:rv1")); err != nil {
		t.Fatalf("HandleUpdate: %v", err)
	}
	if len(tg.sent) != 1 {
		t.Fatalf("sends = %d, want 1", len(tg.sent))
	}
	text := tg.sent[0].text
	for _, want := range []string{"request_changes", "85/100", "done", "Looks risky", "SQL injection", "a.go:12", "risk", "HIGH"} {
		if !strings.Contains(text, want) {
			t.Fatalf("rendered review missing %q:\n%s", want, text)
		}
	}
}

func TestReviewsCommandMergesAndSorts(t *testing.T) {
	tg := &fakeTelegram{}
	cfg := fakeConfig{token: "T", allowed: []string{"100"}}
	now := time.Now().UTC()
	rp := fakeRepos{repos: []repo.Repo{{ID: "r1", Name: "web"}, {ID: "r2", Name: "api"}}}
	rv := &fakeReviews{byRepo: map[string][]review.Review{
		"r1": {{ID: "old", RepoID: "r1", MRIID: 1, Status: review.StatusDone, CreatedAt: now.Add(-time.Hour)}},
		"r2": {{ID: "new", RepoID: "r2", MRIID: 2, Status: review.StatusPending, CreatedAt: now}},
	}}
	svc := newService(tg, cfg, rv, rp)

	if err := svc.HandleUpdate(context.Background(), msgUpdate(100, "/reviews")); err != nil {
		t.Fatalf("HandleUpdate: %v", err)
	}
	kb := tg.sent[0].keyboard
	if len(kb) != 2 {
		t.Fatalf("keyboard rows = %d, want 2", len(kb))
	}
	// Newest first.
	if kb[0][0].Data != "rv:new" || kb[1][0].Data != "rv:old" {
		t.Fatalf("reviews not sorted newest-first: %q, %q", kb[0][0].Data, kb[1][0].Data)
	}
	if !strings.Contains(kb[0][0].Text, "api") || !strings.Contains(kb[0][0].Text, "!2") {
		t.Fatalf("review label = %q, want repo name and MR iid", kb[0][0].Text)
	}
}

// TestMalformedCallbackFallsBack drives HandleUpdate with unknown or truncated
// callback_data. Each was already acknowledged, so the actor must still get a
// guiding fallback message rather than silence.
func TestMalformedCallbackFallsBack(t *testing.T) {
	const fallback = "Couldn't process that button — send /menu to start over."
	for _, data := range []string{"go:r1:7", "mr:r1", "nope:x", ""} {
		t.Run(data, func(t *testing.T) {
			tg := &fakeTelegram{}
			cfg := fakeConfig{token: "T", allowed: []string{"100"}}
			rv := &fakeReviews{}
			svc := newService(tg, cfg, rv, fakeRepos{})

			if err := svc.HandleUpdate(context.Background(), cbUpdate(100, data)); err != nil {
				t.Fatalf("HandleUpdate(%q): %v", data, err)
			}
			if len(tg.sent) != 1 {
				t.Fatalf("data %q: sends = %d, want 1", data, len(tg.sent))
			}
			if tg.sent[0].text != fallback {
				t.Fatalf("data %q: text = %q, want fallback message", data, tg.sent[0].text)
			}
			if len(rv.created) != 0 {
				t.Fatalf("data %q: unexpected review creation", data)
			}
		})
	}
}

// TestHandlerErrorSurfacesToUser verifies that when a routed handler fails (a
// failed Create, or a stale button pointing at a since-deleted entity), the
// actor still receives a generic error reply instead of silence.
func TestHandlerErrorSurfacesToUser(t *testing.T) {
	const generic = "Something went wrong — please try again."

	t.Run("create error", func(t *testing.T) {
		tg := &fakeTelegram{}
		cfg := fakeConfig{token: "T", allowed: []string{"100"}}
		rv := &fakeReviews{createErr: errors.New("boom")}
		svc := newService(tg, cfg, rv, fakeRepos{})

		if err := svc.HandleUpdate(context.Background(), cbUpdate(100, "go:r1:7:fast")); err != nil {
			t.Fatalf("HandleUpdate: %v", err)
		}
		if len(tg.sent) != 1 || tg.sent[0].text != generic {
			t.Fatalf("sends = %+v, want single generic error reply", tg.sent)
		}
	})

	t.Run("get error", func(t *testing.T) {
		tg := &fakeTelegram{}
		cfg := fakeConfig{token: "T", allowed: []string{"100"}}
		rv := &fakeReviews{getErr: errors.New("gone")}
		svc := newService(tg, cfg, rv, fakeRepos{})

		if err := svc.HandleUpdate(context.Background(), cbUpdate(100, "rv:x")); err != nil {
			t.Fatalf("HandleUpdate: %v", err)
		}
		if len(tg.sent) != 1 || tg.sent[0].text != generic {
			t.Fatalf("sends = %+v, want single generic error reply", tg.sent)
		}
	})
}

// TestGoCallbackUnknownModeDefaultsFast documents the contract: any mode that is
// not exactly "deep" falls back to the safe, cheaper fast mode.
func TestGoCallbackUnknownModeDefaultsFast(t *testing.T) {
	tg := &fakeTelegram{}
	cfg := fakeConfig{token: "T", allowed: []string{"100"}}
	rv := &fakeReviews{}
	svc := newService(tg, cfg, rv, fakeRepos{})

	if err := svc.HandleUpdate(context.Background(), cbUpdate(100, "go:r1:7:bogus")); err != nil {
		t.Fatalf("HandleUpdate: %v", err)
	}
	if len(rv.created) != 1 || rv.created[0].mode != review.ModeFast {
		t.Fatalf("create calls = %+v, want one fast-mode create", rv.created)
	}
}

// TestEmptyAllowlistIgnoresEverything asserts the "reject everything when nothing
// is configured" contract: with no targets (empty allowlist), any update is
// ignored — no reply, no side effects.
func TestEmptyAllowlistIgnoresEverything(t *testing.T) {
	tg := &fakeTelegram{}
	cfg := fakeConfig{token: "T", allowed: nil}
	rv := &fakeReviews{}
	rp := fakeRepos{repos: []repo.Repo{{ID: "r1", Name: "web"}}}
	svc := newService(tg, cfg, rv, rp)

	updates := []tgapi.Update{
		msgUpdate(100, "/repos"),
		cbUpdate(100, "go:r1:7:fast"),
	}
	for _, up := range updates {
		if err := svc.HandleUpdate(context.Background(), up); err != nil {
			t.Fatalf("HandleUpdate: %v", err)
		}
	}
	if len(tg.sent) != 0 {
		t.Fatalf("empty allowlist produced %d sends, want 0", len(tg.sent))
	}
	if len(rv.created) != 0 {
		t.Fatalf("empty allowlist produced %d creates, want 0", len(rv.created))
	}
}
