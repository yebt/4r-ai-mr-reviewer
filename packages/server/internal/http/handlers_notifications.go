package httpapi

import (
	"errors"
	"net/http"
	"time"

	"github.com/webcloster-dev/ai-reviewer/internal/app/notifications"
	"github.com/webcloster-dev/ai-reviewer/internal/domain/notification"
)

// --- notifications ---

func (s *Server) listNotificationEvents(w http.ResponseWriter, _ *http.Request) {
	events := s.notifications.Events()
	out := make([]string, 0, len(events))
	out = append(out, events...)
	writeJSON(w, http.StatusOK, map[string][]string{"events": out})
}

func (s *Server) listNotificationRules(w http.ResponseWriter, r *http.Request) {
	rules, err := s.notifications.ListRules(r.Context())
	if err != nil {
		writeErr(w, err, http.StatusInternalServerError)
		return
	}
	out := make([]notificationRuleResp, 0, len(rules))
	for _, rule := range rules {
		out = append(out, toNotificationRule(rule))
	}
	writeJSON(w, http.StatusOK, out)
}

func (s *Server) createNotificationRule(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Event        string `json:"event"`
		NotifierID   string `json:"notifierId"`
		NotifierKind string `json:"notifierKind"`
	}
	if err := decode(r, &in); err != nil {
		writeErr(w, err, http.StatusBadRequest)
		return
	}
	kind := in.NotifierKind
	if kind == "" {
		kind = notification.NotifierTelegram
	}
	rule, err := s.notifications.AddRule(r.Context(), in.Event, kind, in.NotifierID)
	if err != nil {
		// A duplicate is a conflict; every other validation failure is a 400.
		if errors.Is(err, notifications.ErrDuplicateRule) {
			writeErr(w, err, http.StatusConflict)
			return
		}
		writeErr(w, err, http.StatusBadRequest)
		return
	}
	writeJSON(w, http.StatusCreated, toNotificationRule(rule))
}

func (s *Server) patchNotificationRule(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Enabled bool `json:"enabled"`
	}
	if err := decode(r, &in); err != nil {
		writeErr(w, err, http.StatusBadRequest)
		return
	}
	id := r.PathValue("id")
	if err := s.notifications.SetRuleEnabled(r.Context(), id, in.Enabled); err != nil {
		writeErr(w, err, http.StatusBadRequest)
		return
	}
	rule, err := s.notifications.GetRule(r.Context(), id)
	if err != nil {
		writeErr(w, err, http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, toNotificationRule(rule))
}

func (s *Server) deleteNotificationRule(w http.ResponseWriter, r *http.Request) {
	if err := s.notifications.RemoveRule(r.Context(), r.PathValue("id")); err != nil {
		writeErr(w, err, http.StatusBadRequest)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// --- response DTO ---

type notificationRuleResp struct {
	ID           string    `json:"id"`
	Event        string    `json:"event"`
	NotifierKind string    `json:"notifierKind"`
	NotifierID   string    `json:"notifierId"`
	Enabled      bool      `json:"enabled"`
	CreatedAt    time.Time `json:"createdAt"`
}

func toNotificationRule(rule notification.Rule) notificationRuleResp {
	return notificationRuleResp{
		ID: rule.ID, Event: rule.Event, NotifierKind: rule.NotifierKind,
		NotifierID: rule.NotifierID, Enabled: rule.Enabled, CreatedAt: rule.CreatedAt,
	}
}
