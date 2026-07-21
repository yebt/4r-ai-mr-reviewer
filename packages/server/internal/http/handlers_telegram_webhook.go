package httpapi

import (
	"context"
	"crypto/subtle"
	"log"
	"net/http"
	"time"

	tgapi "github.com/webcloster-dev/ai-reviewer/internal/adapters/telegram"
)

// telegramWebhook receives Telegram Bot API updates.
//
// It is dormant until enabled: when no webhook secret is configured, it returns
// 200 without processing so nothing happens even if updates arrive. When a
// secret is configured, the request must carry a matching
// X-Telegram-Bot-Api-Secret-Token header (compared in constant time) or it is
// rejected with 401.
//
// Dispatch runs on a detached goroutine with its own bounded context and a
// panic recovery: Telegram expects a fast 200 and retries on non-2xx, so the
// handler must never block on the dispatcher nor let it crash the request.
func (s *Server) telegramWebhook(w http.ResponseWriter, r *http.Request) {
	// Dormant: no secret configured means the receiver is inert.
	if s.telegramWebhookSecret == "" {
		w.WriteHeader(http.StatusOK)
		return
	}

	got := r.Header.Get("X-Telegram-Bot-Api-Secret-Token")
	if subtle.ConstantTimeCompare([]byte(got), []byte(s.telegramWebhookSecret)) != 1 {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	var up tgapi.Update
	if err := decode(r, &up); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// Dispatch off the request path so we can answer Telegram immediately. The
	// bot may be nil if the process was wired without a dispatcher.
	if s.bot != nil {
		go func() {
			defer func() {
				if rec := recover(); rec != nil {
					log.Printf("telegram webhook: dispatch panic: %v", rec)
				}
			}()
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			if err := s.bot.HandleUpdate(ctx, up); err != nil {
				log.Printf("telegram webhook: handle update: %v", err)
			}
		}()
	}

	w.WriteHeader(http.StatusOK)
}
