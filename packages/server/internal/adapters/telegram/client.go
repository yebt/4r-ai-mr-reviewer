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
	"strconv"
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

// ResolvedThread is a forum topic thread the bot has seen inside a chat.
type ResolvedThread struct {
	ThreadID string
	Name     string
}

// ResolvedChat is a chat the bot has recently seen, along with any forum topic
// threads observed inside it. It lets the UI offer a pick list instead of
// asking the user to copy chat/thread IDs by hand.
type ResolvedChat struct {
	ChatID  string
	Title   string
	Type    string
	Threads []ResolvedThread
}

// ResolveChats calls the Bot API getUpdates method and distils the recently
// seen chats (and forum topic threads) from the returned updates. The bot token
// is used in the request path but is never logged or returned in an error.
//
// An empty update list yields an empty slice and a nil error: "the bot has seen
// nothing yet" is a valid, non-error state the UI can render.
func ResolveChats(ctx context.Context, botToken string) ([]ResolvedChat, error) {
	url := fmt.Sprintf("%s/bot%s/getUpdates", baseURL, botToken)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		// Never wrap the underlying error: it would embed the request URL, which
		// contains the bot token.
		return nil, fmt.Errorf("telegram: build request failed")
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("telegram: get updates: %w", err)
	}
	defer resp.Body.Close()

	// A single update carries at most one message-like object. Only the fields
	// needed to identify a chat and a forum topic thread are decoded.
	type forumTopicCreated struct {
		Name string `json:"name"`
	}
	type messageLike struct {
		MessageThreadID   int64              `json:"message_thread_id"`
		IsTopicMessage    bool               `json:"is_topic_message"`
		ForumTopicCreated *forumTopicCreated `json:"forum_topic_created"`
		ReplyToMessage    *struct {
			ForumTopicCreated *forumTopicCreated `json:"forum_topic_created"`
		} `json:"reply_to_message"`
		Chat struct {
			ID        int64  `json:"id"`
			Title     string `json:"title"`
			Type      string `json:"type"`
			Username  string `json:"username"`
			FirstName string `json:"first_name"`
		} `json:"chat"`
	}
	type update struct {
		Message       *messageLike `json:"message"`
		ChannelPost   *messageLike `json:"channel_post"`
		EditedMessage *messageLike `json:"edited_message"`
	}

	var out struct {
		OK          bool     `json:"ok"`
		Description string   `json:"description"`
		Result      []update `json:"result"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&out); err != nil {
		return nil, fmt.Errorf("telegram: decode updates: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 || !out.OK {
		if out.Description != "" {
			return nil, fmt.Errorf("telegram: get updates failed (status %d): %s", resp.StatusCode, out.Description)
		}
		return nil, fmt.Errorf("telegram: get updates failed (status %d)", resp.StatusCode)
	}

	// order preserves first-seen chat order for a stable response; byID indexes
	// into chats so a chat seen across multiple updates is merged in place.
	chats := make([]ResolvedChat, 0, len(out.Result))
	byID := make(map[string]int, len(out.Result))
	// threadSeen[chatID][threadID] guards against duplicate threads.
	threadSeen := make(map[string]map[string]int)

	for _, u := range out.Result {
		msg := u.Message
		if msg == nil {
			msg = u.ChannelPost
		}
		if msg == nil {
			msg = u.EditedMessage
		}
		if msg == nil {
			continue
		}

		chatID := strconv.FormatInt(msg.Chat.ID, 10)
		title := firstNonEmpty(msg.Chat.Title, msg.Chat.Username, msg.Chat.FirstName, chatID)

		idx, ok := byID[chatID]
		if !ok {
			idx = len(chats)
			byID[chatID] = idx
			chats = append(chats, ResolvedChat{
				ChatID:  chatID,
				Title:   title,
				Type:    msg.Chat.Type,
				Threads: make([]ResolvedThread, 0),
			})
			threadSeen[chatID] = make(map[string]int)
		} else if chats[idx].Title == chatID && title != chatID {
			// Fill in a better title if the first sighting fell back to the ID.
			chats[idx].Title = title
		}

		// A thread is only recorded for genuine topic messages.
		if msg.MessageThreadID == 0 || !msg.IsTopicMessage {
			continue
		}
		threadID := strconv.FormatInt(msg.MessageThreadID, 10)
		name := ""
		if msg.ReplyToMessage != nil && msg.ReplyToMessage.ForumTopicCreated != nil {
			name = msg.ReplyToMessage.ForumTopicCreated.Name
		} else if msg.ForumTopicCreated != nil {
			name = msg.ForumTopicCreated.Name
		}

		if tIdx, seen := threadSeen[chatID][threadID]; seen {
			if chats[idx].Threads[tIdx].Name == "" && name != "" {
				chats[idx].Threads[tIdx].Name = name
			}
			continue
		}
		threadSeen[chatID][threadID] = len(chats[idx].Threads)
		chats[idx].Threads = append(chats[idx].Threads, ResolvedThread{ThreadID: threadID, Name: name})
	}

	return chats, nil
}

// firstNonEmpty returns the first non-empty string, or "" if all are empty.
func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if v != "" {
			return v
		}
	}
	return ""
}
