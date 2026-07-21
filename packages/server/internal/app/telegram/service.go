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
// test message.
func (s *Service) SendTo(ctx context.Context, targetID, text string) error {
	t, err := s.repo.Get(ctx, targetID)
	if err != nil {
		return err
	}
	token, err := s.token(ctx, targetID)
	if err != nil {
		return err
	}
	return tgapi.SendMessage(ctx, token, t.ChatID, t.ThreadID, text)
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
