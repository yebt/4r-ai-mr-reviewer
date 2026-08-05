// Package telegram coordinates Telegram target persistence with the encrypted
// secret store, so a bot token is always stored and removed alongside its
// target. It also sends test and review-finished notifications.
package telegram

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	tgapi "github.com/webcloster-dev/ai-reviewer/internal/adapters/telegram"
	"github.com/webcloster-dev/ai-reviewer/internal/domain/secret"
	tgdomain "github.com/webcloster-dev/ai-reviewer/internal/domain/telegram"
	"github.com/webcloster-dev/ai-reviewer/internal/id"
)

// Service manages telegram targets and their bot tokens together.
type Service struct {
	repo    tgdomain.Repository
	secrets secret.Store
}

// NewService wires the telegram service.
func NewService(repo tgdomain.Repository, secrets secret.Store) *Service {
	return &Service{repo: repo, secrets: secrets}
}

// AddInput describes a new telegram target.
type AddInput struct {
	Name      string
	BotToken  string
	ChatID    string
	ThreadID  string
	IsDefault bool
}

// Add encrypts the bot token and records the target. Inputs are trimmed before
// validation and storage so stray whitespace or control characters (e.g. a
// trailing newline on a pasted token) never reach the Bot API request path.
func (s *Service) Add(ctx context.Context, in AddInput) (tgdomain.Target, error) {
	name := strings.TrimSpace(in.Name)
	botToken := strings.TrimSpace(in.BotToken)
	chatID := strings.TrimSpace(in.ChatID)
	threadID := strings.TrimSpace(in.ThreadID)
	if name == "" || botToken == "" || chatID == "" {
		return tgdomain.Target{}, fmt.Errorf("telegram: name, botToken and chatId are required")
	}
	t := tgdomain.Target{
		ID:        id.New(),
		Name:      name,
		ChatID:    chatID,
		ThreadID:  threadID,
		CreatedAt: time.Now().UTC(),
	}
	t.TokenRef = "telegram:" + t.ID + ":token"

	if err := s.secrets.Set(ctx, t.TokenRef, []byte(botToken)); err != nil {
		return tgdomain.Target{}, err
	}
	if err := s.repo.Create(ctx, t); err != nil {
		// Roll back the orphaned secret so we never leak a dangling token.
		_ = s.secrets.Delete(ctx, t.TokenRef)
		return tgdomain.Target{}, err
	}
	if in.IsDefault {
		if err := s.repo.SetDefault(ctx, t.ID); err != nil {
			// The row and secret already exist; roll both back so we don't leave an
			// orphaned non-default target behind.
			_ = s.repo.Delete(ctx, t.ID)
			_ = s.secrets.Delete(ctx, t.TokenRef)
			return tgdomain.Target{}, err
		}
		t.IsDefault = true
	}
	return t, nil
}

// UpdateInput describes an edit to an existing telegram target. BotToken is
// optional: a nil or blank pointer keeps the stored token untouched, while a
// non-empty value re-encrypts (rotates) the token under the same TokenRef.
type UpdateInput struct {
	Name      string
	BotToken  *string
	ChatID    string
	ThreadID  string
	IsDefault bool
}

// Update edits an existing target's name, chat and thread, optionally rotating
// its bot token, and mirrors Add's default handling (setting default clears the
// previous one). It returns tgdomain.ErrNotFound when the target is unknown.
// Inputs are trimmed before validation and storage, matching Add.
func (s *Service) Update(ctx context.Context, id string, in UpdateInput) (tgdomain.Target, error) {
	existing, err := s.repo.Get(ctx, id)
	if err != nil {
		return tgdomain.Target{}, err
	}
	name := strings.TrimSpace(in.Name)
	chatID := strings.TrimSpace(in.ChatID)
	threadID := strings.TrimSpace(in.ThreadID)
	if name == "" || chatID == "" {
		return tgdomain.Target{}, fmt.Errorf("telegram: name and chatId are required")
	}

	// Rotate the token only when a new non-empty value is supplied; otherwise the
	// existing encrypted token stays untouched under the same TokenRef.
	if in.BotToken != nil {
		if botToken := strings.TrimSpace(*in.BotToken); botToken != "" {
			if err := s.secrets.Set(ctx, existing.TokenRef, []byte(botToken)); err != nil {
				return tgdomain.Target{}, err
			}
		}
	}

	updated := existing
	updated.Name = name
	updated.ChatID = chatID
	updated.ThreadID = threadID
	if err := s.repo.Update(ctx, updated); err != nil {
		return tgdomain.Target{}, err
	}

	// Mirror Add: only a request to become default is acted on (it clears the
	// previous default). Un-defaulting is not expressed here, just as Add cannot.
	if in.IsDefault && !existing.IsDefault {
		if err := s.repo.SetDefault(ctx, id); err != nil {
			return tgdomain.Target{}, err
		}
		updated.IsDefault = true
	}
	return updated, nil
}

// Duplicate creates a new target copying the source's chat, thread and bot
// token, named "<name> (copy)". The copy is never default and never the bot, and
// its token is decrypted from the source and re-encrypted under the copy's own
// TokenRef so the duplicate is immediately functional. It returns
// tgdomain.ErrNotFound when the source is unknown.
func (s *Service) Duplicate(ctx context.Context, srcID string) (tgdomain.Target, error) {
	src, err := s.repo.Get(ctx, srcID)
	if err != nil {
		return tgdomain.Target{}, err
	}
	token, err := s.secrets.Get(ctx, src.TokenRef)
	if err != nil {
		return tgdomain.Target{}, err
	}

	dup := tgdomain.Target{
		ID:        id.New(),
		Name:      src.Name + " (copy)",
		ChatID:    src.ChatID,
		ThreadID:  src.ThreadID,
		CreatedAt: time.Now().UTC(),
	}
	dup.TokenRef = "telegram:" + dup.ID + ":token"

	if err := s.secrets.Set(ctx, dup.TokenRef, token); err != nil {
		return tgdomain.Target{}, err
	}
	if err := s.repo.Create(ctx, dup); err != nil {
		// Roll back the orphaned secret so we never leak a dangling token.
		_ = s.secrets.Delete(ctx, dup.TokenRef)
		return tgdomain.Target{}, err
	}
	return dup, nil
}

// List returns all telegram targets.
func (s *Service) List(ctx context.Context) ([]tgdomain.Target, error) {
	return s.repo.List(ctx)
}

// Get returns one telegram target.
func (s *Service) Get(ctx context.Context, id string) (tgdomain.Target, error) {
	return s.repo.Get(ctx, id)
}

// Exists reports whether a telegram target with the given id exists. A missing
// target is a clean (false, nil); any other lookup error propagates.
func (s *Service) Exists(ctx context.Context, id string) (bool, error) {
	if _, err := s.repo.Get(ctx, id); err != nil {
		if errors.Is(err, tgdomain.ErrNotFound) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// SetDefault makes id the default target.
func (s *Service) SetDefault(ctx context.Context, id string) error {
	return s.repo.SetDefault(ctx, id)
}

// SetBot designates id as the interactive-bot target whose token drives the
// webhook receiver.
func (s *Service) SetBot(ctx context.Context, id string) error {
	return s.repo.SetBot(ctx, id)
}

// Bot returns the interactive-bot target, or tgdomain.ErrNotFound if none is
// designated (the bot is dormant).
func (s *Service) Bot(ctx context.Context) (tgdomain.Target, error) {
	return s.repo.GetBot(ctx)
}

// BotToken resolves the decrypted bot token of the designated interactive-bot
// target, returning the target itself as a convenient chat fallback. It returns
// tgdomain.ErrNotFound when no bot target is set (the bot is dormant).
func (s *Service) BotToken(ctx context.Context) (token string, chatFallback tgdomain.Target, err error) {
	t, err := s.repo.GetBot(ctx)
	if err != nil {
		return "", tgdomain.Target{}, err
	}
	tok, err := s.token(ctx, t.ID)
	if err != nil {
		return "", tgdomain.Target{}, err
	}
	return tok, t, nil
}

// AllowedChatIDs returns the chat IDs of every configured target. It is the
// allowlist the interactive bot authorizes incoming updates against.
func (s *Service) AllowedChatIDs(ctx context.Context) ([]string, error) {
	ts, err := s.repo.List(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]string, 0, len(ts))
	for _, t := range ts {
		out = append(out, t.ChatID)
	}
	return out, nil
}

// Remove deletes the target and its bot token.
func (s *Service) Remove(ctx context.Context, id string) error {
	t, err := s.repo.Get(ctx, id)
	if err != nil {
		return err
	}
	if err := s.repo.Delete(ctx, id); err != nil {
		return err
	}
	return s.secrets.Delete(ctx, t.TokenRef)
}

// token returns the decrypted bot token for a target.
func (s *Service) token(ctx context.Context, id string) (string, error) {
	t, err := s.repo.Get(ctx, id)
	if err != nil {
		return "", err
	}
	b, err := s.secrets.Get(ctx, t.TokenRef)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// SendTo delivers text to the given target, loading its chat, thread and bot
// token. It is the single send path used by both notifications fan-out and the
// test message. Text is delivered with parse_mode HTML so the rich, pre-escaped
// notification messages render as formatted text; callers are responsible for
// html-escaping any interpolated value (see the reviews and routines services).
func (s *Service) SendTo(ctx context.Context, targetID, text string) error {
	t, err := s.repo.Get(ctx, targetID)
	if err != nil {
		return err
	}
	token, err := s.token(ctx, targetID)
	if err != nil {
		return err
	}
	return tgapi.SendMessageHTML(ctx, token, t.ChatID, t.ThreadID, text, nil)
}

// SendTest sends a fixed test message to the given target so the user can
// confirm the bot token and chat are configured correctly.
func (s *Service) SendTest(ctx context.Context, id string) error {
	return s.SendTo(ctx, id, "✅ ai-reviewer test message")
}

// Resolve calls the Bot API getUpdates method and returns the chats (and forum
// topic threads) the bot has recently seen, so the UI can offer a pick list
// instead of asking the user to copy chat/thread IDs by hand. The token is
// trimmed before use and never persisted; this is a stateless lookup.
func (s *Service) Resolve(ctx context.Context, botToken string) ([]tgapi.ResolvedChat, error) {
	token := strings.TrimSpace(botToken)
	if token == "" {
		return nil, fmt.Errorf("telegram: bot token is required")
	}
	return tgapi.ResolveChats(ctx, token)
}
