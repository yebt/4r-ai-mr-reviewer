// Package bot is the interactive Telegram bot receiver: it turns incoming
// webhook updates (commands and inline-button taps) into replies with
// inline-keyboard navigation over repos, merge requests and reviews.
//
// It is deliberately decoupled from concrete backends through narrow interfaces
// so it can be driven end to end in tests with fakes. It never edits messages;
// every reply is a fresh message.
package bot

import (
	"context"
	"errors"
	"fmt"
	"log"
	"sort"
	"strconv"
	"strings"

	"github.com/webcloster-dev/ai-reviewer/internal/adapters/gitlab"
	tgapi "github.com/webcloster-dev/ai-reviewer/internal/adapters/telegram"
	"github.com/webcloster-dev/ai-reviewer/internal/domain/repo"
	"github.com/webcloster-dev/ai-reviewer/internal/domain/review"
	tgdomain "github.com/webcloster-dev/ai-reviewer/internal/domain/telegram"
)

// TelegramPort is the outbound Bot API surface the dispatcher needs. It is kept
// narrow so tests can record sends against a fake.
type TelegramPort interface {
	SendMessageHTML(ctx context.Context, token, chatID, threadID, text string, keyboard [][]tgapi.InlineButton) error
	AnswerCallbackQuery(ctx context.Context, token, cbID, text string) error
}

// Config resolves the interactive-bot token and the chat allowlist. It is
// satisfied by *telegram.Service.
type Config interface {
	// BotToken returns the designated bot's token (and its target as a chat
	// fallback), or tgdomain.ErrNotFound when no bot is designated (dormant).
	BotToken(ctx context.Context) (token string, chatFallback tgdomain.Target, err error)
	// AllowedChatIDs returns the chat IDs authorized to drive the bot.
	AllowedChatIDs(ctx context.Context) ([]string, error)
}

// ReposLister lists tracked repositories. Satisfied by *repos.Service.
type ReposLister interface {
	List(ctx context.Context) ([]repo.Repo, error)
}

// ReviewsPort is the review surface the dispatcher needs. Satisfied by
// *reviews.Service.
type ReviewsPort interface {
	List(ctx context.Context, repoID string) ([]review.Review, error)
	Get(ctx context.Context, reviewID string) (review.Review, error)
	Create(ctx context.Context, repoID string, mrIID int, mode review.ContextMode, providerID, model string) (review.Review, error)
	ListOpenMergeRequests(ctx context.Context, repoID string) ([]gitlab.MergeRequest, error)
}

// Service dispatches Telegram updates to bot actions.
type Service struct {
	tg      TelegramPort
	cfg     Config
	reviews ReviewsPort
	repos   ReposLister
}

// NewService wires the dispatcher.
func NewService(tg TelegramPort, cfg Config, reviews ReviewsPort, repos ReposLister) *Service {
	return &Service{tg: tg, cfg: cfg, reviews: reviews, repos: repos}
}

// APIClient is the production TelegramPort backed by the Bot API adapter.
type APIClient struct{}

// NewAPIClient returns a TelegramPort that calls the real Bot API.
func NewAPIClient() APIClient { return APIClient{} }

// SendMessageHTML forwards to the Bot API adapter.
func (APIClient) SendMessageHTML(ctx context.Context, token, chatID, threadID, text string, keyboard [][]tgapi.InlineButton) error {
	return tgapi.SendMessageHTML(ctx, token, chatID, threadID, text, keyboard)
}

// AnswerCallbackQuery forwards to the Bot API adapter.
func (APIClient) AnswerCallbackQuery(ctx context.Context, token, cbID, text string) error {
	return tgapi.AnswerCallbackQuery(ctx, token, cbID, text)
}

// HandleUpdate processes one incoming update: it authorizes the actor against
// the allowlist, resolves the bot token (staying dormant when none is set), and
// routes the message command or callback-button tap to an action.
//
// Unauthorized actors and a dormant bot are ignored silently (logged, no
// reply); only genuine infrastructure errors are returned.
func (s *Service) HandleUpdate(ctx context.Context, up tgapi.Update) error {
	chatID := actorChatID(up)
	if chatID == "" {
		return nil
	}

	allowed, err := s.cfg.AllowedChatIDs(ctx)
	if err != nil {
		return err
	}
	if !contains(allowed, chatID) {
		log.Printf("bot: ignoring update from unauthorized chat %s", chatID)
		return nil
	}

	token, _, err := s.cfg.BotToken(ctx)
	if err != nil {
		if errors.Is(err, tgdomain.ErrNotFound) {
			log.Print("bot: no bot target designated; ignoring update (dormant)")
			return nil
		}
		return err
	}

	var routeErr error
	switch {
	case up.CallbackQuery != nil:
		// Acknowledge the tap first so the client stops its spinner, then route.
		_ = s.tg.AnswerCallbackQuery(ctx, token, up.CallbackQuery.ID, "")
		routeErr = s.routeCallback(ctx, token, chatID, up.CallbackQuery.Data)
	case up.Message != nil && strings.HasPrefix(up.Message.Text, "/"):
		routeErr = s.routeCommand(ctx, token, chatID, up.Message.Text)
	default:
		return nil
	}

	// A handler failing (e.g. reviews.Create failed, or a stale button pointed at
	// a since-deleted entity) would otherwise be silent: the HTTP layer already
	// answered 200 and the callback spinner was already acknowledged. Surface a
	// generic reply to the actor so the tap never dead-ends, then swallow the
	// error — there is no client left to receive a non-nil return.
	if routeErr != nil {
		log.Printf("bot: handling update for chat %s: %v", chatID, routeErr)
		if err := s.send(ctx, token, chatID, "Something went wrong — please try again.", nil); err != nil {
			log.Printf("bot: sending error reply to chat %s: %v", chatID, err)
		}
	}
	return nil
}

// routeCommand handles a slash command message.
func (s *Service) routeCommand(ctx context.Context, token, chatID, text string) error {
	cmd := strings.Fields(text)[0]
	// Strip a "@botname" suffix (Telegram appends it in group chats).
	if i := strings.IndexByte(cmd, '@'); i >= 0 {
		cmd = cmd[:i]
	}
	switch cmd {
	case "/start", "/menu":
		return s.send(ctx, token, chatID, "<b>ai-reviewer</b>\nChoose a section.", [][]tgapi.InlineButton{
			{{Text: "Repos", Data: "nav:repos"}},
			{{Text: "Reviews", Data: "nav:reviews"}},
		})
	case "/repos":
		return s.sendRepos(ctx, token, chatID)
	case "/reviews":
		return s.sendReviews(ctx, token, chatID)
	default:
		return s.send(ctx, token, chatID, "Unknown command. Try /menu.", nil)
	}
}

// routeCallback routes an inline-button tap by its callback_data scheme.
func (s *Service) routeCallback(ctx context.Context, token, chatID, data string) error {
	parts := strings.Split(data, ":")
	switch parts[0] {
	case "nav":
		if len(parts) >= 2 && parts[1] == "repos" {
			return s.sendRepos(ctx, token, chatID)
		}
		if len(parts) >= 2 && parts[1] == "reviews" {
			return s.sendReviews(ctx, token, chatID)
		}
	case "repo":
		if len(parts) >= 2 {
			return s.sendRepoView(ctx, token, chatID, parts[1])
		}
	case "rr":
		if len(parts) >= 2 {
			return s.sendRepoReviews(ctx, token, chatID, parts[1])
		}
	case "mr":
		if len(parts) >= 3 {
			return s.sendMRView(ctx, token, chatID, parts[1], parts[2])
		}
	case "go":
		if len(parts) >= 4 {
			return s.doReview(ctx, token, chatID, parts[1], parts[2], parts[3])
		}
	case "rv":
		if len(parts) >= 2 {
			return s.sendReviewContent(ctx, token, chatID, parts[1])
		}
	}
	// Unknown prefix, or a known prefix with too few segments (e.g. "go:r1:7",
	// "mr:r1", "rr", ""): the tap was already acknowledged, so without a reply
	// the user would see nothing. Guide them back to a known state.
	return s.send(ctx, token, chatID, "Couldn't process that button — send /menu to start over.", nil)
}

// sendRepos lists tracked repositories as buttons.
func (s *Service) sendRepos(ctx context.Context, token, chatID string) error {
	repos, err := s.repos.List(ctx)
	if err != nil {
		return err
	}
	if len(repos) == 0 {
		return s.send(ctx, token, chatID, "No repositories configured.", nil)
	}
	kb := make([][]tgapi.InlineButton, 0, len(repos))
	for _, rp := range repos {
		kb = append(kb, []tgapi.InlineButton{{Text: rp.Name, Data: "repo:" + rp.ID}})
	}
	return s.send(ctx, token, chatID, "<b>Repositories</b>", kb)
}

// sendReviews lists the most recent reviews across all repos.
func (s *Service) sendReviews(ctx context.Context, token, chatID string) error {
	repos, err := s.repos.List(ctx)
	if err != nil {
		return err
	}
	names := make(map[string]string, len(repos))
	var all []review.Review
	for _, rp := range repos {
		names[rp.ID] = rp.Name
		rvs, err := s.reviews.List(ctx, rp.ID)
		if err != nil {
			return err
		}
		all = append(all, rvs...)
	}
	sort.Slice(all, func(i, j int) bool { return all[i].CreatedAt.After(all[j].CreatedAt) })
	if len(all) > 10 {
		all = all[:10]
	}
	if len(all) == 0 {
		return s.send(ctx, token, chatID, "No reviews yet.", nil)
	}
	kb := make([][]tgapi.InlineButton, 0, len(all))
	for _, rv := range all {
		label := fmt.Sprintf("%s !%d %s", names[rv.RepoID], rv.MRIID, rv.Status)
		kb = append(kb, []tgapi.InlineButton{{Text: label, Data: "rv:" + rv.ID}})
	}
	return s.send(ctx, token, chatID, "<b>Recent reviews</b>", kb)
}

// sendRepoView shows a repo's open merge requests, plus a shortcut to its
// recent reviews.
func (s *Service) sendRepoView(ctx context.Context, token, chatID, repoID string) error {
	mrs, err := s.reviews.ListOpenMergeRequests(ctx, repoID)
	if err != nil {
		return err
	}
	kb := make([][]tgapi.InlineButton, 0, len(mrs)+1)
	for _, mr := range mrs {
		label := fmt.Sprintf("!%d %s", mr.IID, truncate(mr.Title, 40))
		kb = append(kb, []tgapi.InlineButton{{Text: label, Data: fmt.Sprintf("mr:%s:%d", repoID, mr.IID)}})
	}
	kb = append(kb, []tgapi.InlineButton{{Text: "Recent reviews", Data: "rr:" + repoID}})
	text := "<b>Open merge requests</b>"
	if len(mrs) == 0 {
		text = "No open merge requests."
	}
	return s.send(ctx, token, chatID, text, kb)
}

// sendRepoReviews lists one repo's reviews as buttons.
func (s *Service) sendRepoReviews(ctx context.Context, token, chatID, repoID string) error {
	rvs, err := s.reviews.List(ctx, repoID)
	if err != nil {
		return err
	}
	if len(rvs) == 0 {
		return s.send(ctx, token, chatID, "No reviews for this repository yet.", nil)
	}
	kb := make([][]tgapi.InlineButton, 0, len(rvs))
	for _, rv := range rvs {
		label := fmt.Sprintf("!%d %s", rv.MRIID, rv.Status)
		kb = append(kb, []tgapi.InlineButton{{Text: label, Data: "rv:" + rv.ID}})
	}
	return s.send(ctx, token, chatID, "<b>Reviews</b>", kb)
}

// sendMRView offers to review one merge request in fast or deep mode.
func (s *Service) sendMRView(ctx context.Context, token, chatID, repoID, iid string) error {
	text := fmt.Sprintf("<b>MR !%s</b>\nChoose a review mode.", htmlEscape(iid))
	kb := [][]tgapi.InlineButton{{
		{Text: "Review · fast", Data: fmt.Sprintf("go:%s:%s:fast", repoID, iid)},
		{Text: "Review · deep", Data: fmt.Sprintf("go:%s:%s:deep", repoID, iid)},
	}}
	return s.send(ctx, token, chatID, text, kb)
}

// doReview queues a review for a merge request and replies with a shortcut to
// view it.
func (s *Service) doReview(ctx context.Context, token, chatID, repoID, iid, mode string) error {
	n, err := strconv.Atoi(iid)
	if err != nil {
		return s.send(ctx, token, chatID, "Invalid merge request number.", nil)
	}
	// Only an explicit "deep" selects the deep context mode; any other value
	// (including an unknown/malformed one) intentionally defaults to fast, the
	// safe and cheaper default.
	cmode := review.ModeFast
	if mode == "deep" {
		cmode = review.ModeDeep
	}
	rv, err := s.reviews.Create(ctx, repoID, n, cmode, "", "")
	if err != nil {
		return err
	}
	text := fmt.Sprintf("Review queued for !%d.", n)
	kb := [][]tgapi.InlineButton{{{Text: "View review", Data: "rv:" + rv.ID}}}
	return s.send(ctx, token, chatID, text, kb)
}

// sendReviewContent renders a review's outcome as an HTML message.
func (s *Service) sendReviewContent(ctx context.Context, token, chatID, reviewID string) error {
	rv, err := s.reviews.Get(ctx, reviewID)
	if err != nil {
		return err
	}
	kb := [][]tgapi.InlineButton{{{Text: "Back to reviews", Data: "nav:reviews"}}}
	return s.send(ctx, token, chatID, renderReview(rv), kb)
}

// send is the single reply path: an HTML message to the actor's chat (no forum
// thread for now).
func (s *Service) send(ctx context.Context, token, chatID, text string, keyboard [][]tgapi.InlineButton) error {
	return s.tg.SendMessageHTML(ctx, token, chatID, "", text, keyboard)
}

// renderReview builds the HTML body for a single review: a header
// (recommendation · score · status), the summary, and a compact findings list.
func renderReview(rv review.Review) string {
	var b strings.Builder
	rec := rv.Recommendation
	if rec == "" {
		rec = "—"
	}
	fmt.Fprintf(&b, "<b>Review !%d</b>\n%s · %d/100 · %s", rv.MRIID, htmlEscape(string(rec)), rv.Score, htmlEscape(string(rv.Status)))
	if rv.Summary != "" {
		fmt.Fprintf(&b, "\n\n%s", htmlEscape(rv.Summary))
	}
	if len(rv.Findings) > 0 {
		b.WriteString("\n")
		for _, f := range rv.Findings {
			loc := htmlEscape(f.File)
			if f.Line > 0 {
				loc = fmt.Sprintf("%s:%d", loc, f.Line)
			}
			fmt.Fprintf(&b, "\n• %s · %s · %s — %s",
				htmlEscape(string(f.Dimension)),
				htmlEscape(strings.ToUpper(string(f.Severity))),
				loc,
				htmlEscape(f.Issue))
		}
	}
	return b.String()
}

// actorChatID resolves the chat an update originates from, as a string.
func actorChatID(up tgapi.Update) string {
	if up.Message != nil && up.Message.Chat != nil {
		return strconv.FormatInt(up.Message.Chat.ID, 10)
	}
	if up.CallbackQuery != nil && up.CallbackQuery.Message != nil && up.CallbackQuery.Message.Chat != nil {
		return strconv.FormatInt(up.CallbackQuery.Message.Chat.ID, 10)
	}
	return ""
}

func contains(xs []string, want string) bool {
	for _, x := range xs {
		if x == want {
			return true
		}
	}
	return false
}

// truncate shortens s to at most n runes, appending an ellipsis when cut.
func truncate(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	if n <= 1 {
		return string(r[:n])
	}
	return string(r[:n-1]) + "…"
}

// htmlEscape escapes the three characters that are significant in Telegram's
// HTML parse mode so model/user text can never break the markup.
func htmlEscape(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	return s
}
