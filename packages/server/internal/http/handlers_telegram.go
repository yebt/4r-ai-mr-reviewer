package httpapi

import (
	"net/http"
	"time"

	apptelegram "github.com/webcloster-dev/ai-reviewer/internal/app/telegram"
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

func (s *Server) deleteTelegram(w http.ResponseWriter, r *http.Request) {
	if err := s.telegram.Remove(r.Context(), r.PathValue("id")); err != nil {
		writeErr(w, err, http.StatusBadRequest)
		return
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

// --- response DTO ---

type telegramResp struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	ChatID    string    `json:"chatId"`
	ThreadID  string    `json:"threadId"`
	IsDefault bool      `json:"isDefault"`
	CreatedAt time.Time `json:"createdAt"`
}

func toTelegram(t tgdomain.Target) telegramResp {
	return telegramResp{
		ID: t.ID, Name: t.Name, ChatID: t.ChatID, ThreadID: t.ThreadID,
		IsDefault: t.IsDefault, CreatedAt: t.CreatedAt,
	}
}
