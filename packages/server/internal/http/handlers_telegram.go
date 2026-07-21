package httpapi

import (
	"errors"
	"log"
	"net/http"
	"time"

	tgapi "github.com/webcloster-dev/ai-reviewer/internal/adapters/telegram"
	apptelegram "github.com/webcloster-dev/ai-reviewer/internal/app/telegram"
	"github.com/webcloster-dev/ai-reviewer/internal/domain/notification"
	tgdomain "github.com/webcloster-dev/ai-reviewer/internal/domain/telegram"
)

// --- telegram ---

func (s *Server) createTelegram(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Name      string `json:"name"`
		BotToken  string `json:"botToken"`
		ChatID    string `json:"chatId"`
		ThreadID  string `json:"threadId"`
		IsDefault bool   `json:"isDefault"`
	}
	if err := decode(r, &in); err != nil {
		writeErr(w, err, http.StatusBadRequest)
		return
	}
	t, err := s.telegram.Add(r.Context(), apptelegram.AddInput{
		Name: in.Name, BotToken: in.BotToken, ChatID: in.ChatID,
		ThreadID: in.ThreadID, IsDefault: in.IsDefault,
	})
	if err != nil {
		writeErr(w, err, http.StatusBadRequest)
		return
	}
	writeJSON(w, http.StatusCreated, toTelegram(t))
}

func (s *Server) listTelegram(w http.ResponseWriter, r *http.Request) {
	ts, err := s.telegram.List(r.Context())
	if err != nil {
		writeErr(w, err, http.StatusInternalServerError)
		return
	}
	out := make([]telegramResp, 0, len(ts))
	for _, t := range ts {
		out = append(out, toTelegram(t))
	}
	writeJSON(w, http.StatusOK, out)
}

func (s *Server) setDefaultTelegram(w http.ResponseWriter, r *http.Request) {
	if err := s.telegram.SetDefault(r.Context(), r.PathValue("id")); err != nil {
		writeErr(w, err, http.StatusBadRequest)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) setBotTelegram(w http.ResponseWriter, r *http.Request) {
	if err := s.telegram.SetBot(r.Context(), r.PathValue("id")); err != nil {
		writeErr(w, err, http.StatusBadRequest)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) deleteTelegram(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if err := s.telegram.Remove(r.Context(), id); err != nil {
		writeErr(w, err, http.StatusBadRequest)
		return
	}
	// Best-effort: drop any notification rules pointing at the deleted target so
	// no orphaned rules remain. A failure here must not fail the delete.
	if err := s.notifications.RemoveRulesForNotifier(r.Context(), notification.NotifierTelegram, id); err != nil {
		log.Printf("notifications: drop rules for telegram target %s: %v", id, err)
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) testTelegram(w http.ResponseWriter, r *http.Request) {
	if err := s.telegram.SendTest(r.Context(), r.PathValue("id")); err != nil {
		// A missing target is a 404; a delivery failure is an upstream (502)
		// error whose message (never the token) is safe to surface.
		writeErr(w, err, http.StatusBadGateway)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "sent"})
}

func (s *Server) resolveTelegram(w http.ResponseWriter, r *http.Request) {
	var in struct {
		BotToken string `json:"botToken"`
	}
	if err := decode(r, &in); err != nil {
		writeErr(w, err, http.StatusBadRequest)
		return
	}
	if in.BotToken == "" {
		writeErr(w, errors.New("botToken is required"), http.StatusBadRequest)
		return
	}
	chats, err := s.telegram.Resolve(r.Context(), in.BotToken)
	if err != nil {
		// The Bot API is an upstream dependency: a failure here (bad token,
		// network) is a 502. The message never contains the token.
		writeErr(w, err, http.StatusBadGateway)
		return
	}
	writeJSON(w, http.StatusOK, toTelegramResolve(chats))
}

// --- response DTO ---

type telegramResp struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	ChatID    string    `json:"chatId"`
	ThreadID  string    `json:"threadId"`
	IsDefault bool      `json:"isDefault"`
	IsBot     bool      `json:"isBot"`
	CreatedAt time.Time `json:"createdAt"`
}

func toTelegram(t tgdomain.Target) telegramResp {
	return telegramResp{
		ID: t.ID, Name: t.Name, ChatID: t.ChatID, ThreadID: t.ThreadID,
		IsDefault: t.IsDefault, IsBot: t.IsBot, CreatedAt: t.CreatedAt,
	}
}

// telegramResolveResp is the resolve endpoint payload. It carries only the
// discovered chats/threads — never the bot token supplied in the request.
type telegramResolveResp struct {
	Chats []telegramChatResp `json:"chats"`
}

type telegramChatResp struct {
	ChatID  string               `json:"chatId"`
	Title   string               `json:"title"`
	Type    string               `json:"type"`
	Threads []telegramThreadResp `json:"threads"`
}

type telegramThreadResp struct {
	ThreadID string `json:"threadId"`
	Name     string `json:"name"`
}

func toTelegramResolve(chats []tgapi.ResolvedChat) telegramResolveResp {
	out := telegramResolveResp{Chats: make([]telegramChatResp, 0, len(chats))}
	for _, c := range chats {
		threads := make([]telegramThreadResp, 0, len(c.Threads))
		for _, th := range c.Threads {
			threads = append(threads, telegramThreadResp{ThreadID: th.ThreadID, Name: th.Name})
		}
		out.Chats = append(out.Chats, telegramChatResp{
			ChatID: c.ChatID, Title: c.Title, Type: c.Type, Threads: threads,
		})
	}
	return out
}
