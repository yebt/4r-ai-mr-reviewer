// Package telegram sends messages through the Telegram Bot API. It is the
// outbound adapter used to notify a configured chat when a review finishes.
package telegram

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// baseURL is the Telegram Bot API root. It is a package var so tests can point
// SendMessage at an httptest server.
var baseURL = "https://api.telegram.org"

// SetBaseURL overrides the Bot API root and returns a function that restores the
// previous value. It exists so tests (including tests in other packages that go
// through this adapter) can redirect SendMessage to an httptest server.
func SetBaseURL(u string) func() {
	prev := baseURL
	baseURL = u
	return func() { baseURL = prev }
}

// httpClient bounds every Bot API call so a stuck request can never block the
// caller (the review worker) indefinitely.
var httpClient = &http.Client{Timeout: 10 * time.Second}

// SendMessage posts text to a Telegram chat via the Bot API sendMessage method.
// threadID targets a forum topic when non-empty; it is omitted otherwise. The
// bot token is used in the request path but is never logged or returned in an
// error.
func SendMessage(ctx context.Context, botToken, chatID, threadID, text string) error {
	payload := map[string]any{
		"chat_id": chatID,
		"text":    text,
	}
	if threadID != "" {
		payload["message_thread_id"] = threadID
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("telegram: marshal request: %w", err)
	}

	url := fmt.Sprintf("%s/bot%s/sendMessage", baseURL, botToken)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		// Never wrap the underlying error: it would embed the request URL, which
		// contains the bot token, and this error is logged on 5xx by the HTTP layer.
		return fmt.Errorf("telegram: build request failed")
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("telegram: send request: %w", err)
	}
	defer resp.Body.Close()

	var out struct {
		OK          bool   `json:"ok"`
		Description string `json:"description"`
	}
	// Best-effort decode; a non-2xx status is an error regardless of the body.
	_ = json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&out)

	if resp.StatusCode < 200 || resp.StatusCode >= 300 || !out.OK {
		if out.Description != "" {
			return fmt.Errorf("telegram: send failed (status %d): %s", resp.StatusCode, out.Description)
		}
		return fmt.Errorf("telegram: send failed (status %d)", resp.StatusCode)
	}
	return nil
}
