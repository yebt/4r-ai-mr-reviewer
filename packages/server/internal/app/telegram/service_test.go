package telegram

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/webcloster-dev/ai-reviewer/internal/adapters/crypto"
	"github.com/webcloster-dev/ai-reviewer/internal/adapters/sqlite"
	tgapi "github.com/webcloster-dev/ai-reviewer/internal/adapters/telegram"
	"github.com/webcloster-dev/ai-reviewer/internal/domain/secret"
)

func newService(t *testing.T) *Service {
	t.Helper()
	db, err := sqlite.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	salt, _ := crypto.NewSalt()
	key, _ := crypto.DeriveKey("pw", salt)
	cipher, _ := crypto.NewCipher(key)
	secrets := sqlite.NewSecretStore(db, cipher)
	return NewService(sqlite.NewTelegramStore(db), secrets)
}

// capturedRequest records what the Bot API adapter posted.
type capturedRequest struct {
	path string
	body map[string]any
}

// stubTelegram spins up an httptest server standing in for the Bot API and
// redirects the adapter at it. It returns a pointer that receives the last
// request and a cleanup that restores the adapter base URL.
func stubTelegram(t *testing.T, ok bool) *capturedRequest {
	t.Helper()
	got := &capturedRequest{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got.path = r.URL.Path
		b, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(b, &got.body)
		w.Header().Set("Content-Type", "application/json")
		if !ok {
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]any{"ok": false, "description": "chat not found"})
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"ok": true})
	}))
	t.Cleanup(srv.Close)
	restore := tgapi.SetBaseURL(srv.URL)
	t.Cleanup(restore)
	return got
}

func TestAddStoresRowAndEncryptsToken(t *testing.T) {
	ctx := context.Background()
	s := newService(t)

	tg, err := s.Add(ctx, AddInput{Name: "team", BotToken: "bot-secret", ChatID: "-100"})
	if err != nil {
		t.Fatalf("Add: %v", err)
	}
	if tg.ID == "" || tg.TokenRef != "telegram:"+tg.ID+":token" {
		t.Fatalf("unexpected target: %+v", tg)
	}

	// The token must be stored in the secret store, not on the row.
	token, err := s.token(ctx, tg.ID)
	if err != nil {
		t.Fatalf("token: %v", err)
	}
	if token != "bot-secret" {
		t.Fatalf("token = %q, want bot-secret", token)
	}

	got, err := s.Get(ctx, tg.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.Name != "team" || got.ChatID != "-100" {
		t.Fatalf("row mismatch: %+v", got)
	}
}

func TestAddRequiresFields(t *testing.T) {
	s := newService(t)
	if _, err := s.Add(context.Background(), AddInput{Name: "x", ChatID: "1"}); err == nil {
		t.Fatal("expected error when botToken is missing")
	}
	if _, err := s.Add(context.Background(), AddInput{BotToken: "t", ChatID: "1"}); err == nil {
		t.Fatal("expected error when name is missing")
	}
	if _, err := s.Add(context.Background(), AddInput{Name: "x", BotToken: "t"}); err == nil {
		t.Fatal("expected error when chatId is missing")
	}
}

func TestListAndSetDefault(t *testing.T) {
	ctx := context.Background()
	s := newService(t)

	first, _ := s.Add(ctx, AddInput{Name: "a", BotToken: "t1", ChatID: "1"})
	second, _ := s.Add(ctx, AddInput{Name: "b", BotToken: "t2", ChatID: "2", IsDefault: true})

	list, err := s.List(ctx)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(list) != 2 {
		t.Fatalf("List len = %d, want 2", len(list))
	}

	def, err := s.repo.GetDefault(ctx)
	if err != nil {
		t.Fatalf("GetDefault: %v", err)
	}
	if def.ID != second.ID {
		t.Fatalf("default id = %s, want %s", def.ID, second.ID)
	}

	if err := s.SetDefault(ctx, first.ID); err != nil {
		t.Fatalf("SetDefault: %v", err)
	}
	def, _ = s.repo.GetDefault(ctx)
	if def.ID != first.ID {
		t.Fatalf("default id = %s, want %s after switch", def.ID, first.ID)
	}
}

func TestRemoveDeletesRowAndSecret(t *testing.T) {
	ctx := context.Background()
	s := newService(t)
	tg, _ := s.Add(ctx, AddInput{Name: "a", BotToken: "t1", ChatID: "1"})

	if err := s.Remove(ctx, tg.ID); err != nil {
		t.Fatalf("Remove: %v", err)
	}
	if _, err := s.Get(ctx, tg.ID); err == nil {
		t.Fatal("row should be gone after Remove")
	}
	if _, err := s.secrets.Get(ctx, tg.TokenRef); !errors.Is(err, secret.ErrNotFound) {
		t.Fatalf("secret after remove: got %v, want secret.ErrNotFound", err)
	}
}

func TestSendTestPostsChatAndText(t *testing.T) {
	ctx := context.Background()
	s := newService(t)
	got := stubTelegram(t, true)

	tg, _ := s.Add(ctx, AddInput{Name: "a", BotToken: "bot-tok", ChatID: "-100", ThreadID: "7"})
	if err := s.SendTest(ctx, tg.ID); err != nil {
		t.Fatalf("SendTest: %v", err)
	}
	if got.body["chat_id"] != "-100" {
		t.Fatalf("chat_id = %v, want -100", got.body["chat_id"])
	}
	if got.body["message_thread_id"] != "7" {
		t.Fatalf("message_thread_id = %v, want 7", got.body["message_thread_id"])
	}
	if _, ok := got.body["text"].(string); !ok || got.body["text"] == "" {
		t.Fatalf("text missing: %v", got.body["text"])
	}
}

func TestSendTestPropagatesFailure(t *testing.T) {
	ctx := context.Background()
	s := newService(t)
	_ = stubTelegram(t, false)

	tg, _ := s.Add(ctx, AddInput{Name: "a", BotToken: "bot-tok", ChatID: "-100"})
	err := s.SendTest(ctx, tg.ID)
	if err == nil {
		t.Fatal("expected error when Bot API returns ok=false")
	}
	// The token must never leak into the error.
	if contains(err.Error(), "bot-tok") {
		t.Fatalf("error leaked the token: %v", err)
	}
}

func TestNotifyNoDefaultIsNoOp(t *testing.T) {
	ctx := context.Background()
	s := newService(t)
	// A stub is installed but must never be hit when there is no default.
	got := stubTelegram(t, true)

	// A target exists but is not the default.
	_, _ = s.Add(ctx, AddInput{Name: "a", BotToken: "t", ChatID: "1"})

	if err := s.Notify(ctx, "hello"); err != nil {
		t.Fatalf("Notify with no default should be a no-op nil, got %v", err)
	}
	if got.path != "" {
		t.Fatalf("Notify should not call the Bot API when no default is set, hit %q", got.path)
	}
}

func TestNotifySendsToDefault(t *testing.T) {
	ctx := context.Background()
	s := newService(t)
	got := stubTelegram(t, true)

	tg, _ := s.Add(ctx, AddInput{Name: "a", BotToken: "t", ChatID: "999", IsDefault: true})
	_ = tg

	if err := s.Notify(ctx, "review done"); err != nil {
		t.Fatalf("Notify: %v", err)
	}
	if got.body["chat_id"] != "999" {
		t.Fatalf("chat_id = %v, want 999", got.body["chat_id"])
	}
	if got.body["text"] != "review done" {
		t.Fatalf("text = %v, want 'review done'", got.body["text"])
	}
	// No thread configured: the field must be omitted entirely.
	if _, present := got.body["message_thread_id"]; present {
		t.Fatalf("message_thread_id should be omitted when no thread is set")
	}
}

// stubGetUpdates spins up an httptest server that answers getUpdates with the
// given raw JSON payload and redirects the adapter at it. It returns a pointer
// that receives the path the adapter hit and a cleanup that restores the base URL.
func stubGetUpdates(t *testing.T, payload string) *capturedRequest {
	t.Helper()
	got := &capturedRequest{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got.path = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, payload)
	}))
	t.Cleanup(srv.Close)
	restore := tgapi.SetBaseURL(srv.URL)
	t.Cleanup(restore)
	return got
}

func TestResolveReturnsChatsAndThreads(t *testing.T) {
	ctx := context.Background()
	s := newService(t)

	// One plain supergroup message and one topic message in the same chat.
	payload := `{
		"ok": true,
		"result": [
			{"message": {"chat": {"id": -1001234567890, "type": "supergroup", "title": "Team"}}},
			{"message": {
				"message_thread_id": 42,
				"is_topic_message": true,
				"reply_to_message": {"forum_topic_created": {"name": "Alerts"}},
				"chat": {"id": -1001234567890, "type": "supergroup", "title": "Team"}
			}}
		]
	}`
	got := stubGetUpdates(t, payload)

	chats, err := s.Resolve(ctx, "  bot-tok  ")
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if got.path == "" {
		t.Fatal("Resolve should have called the Bot API")
	}
	if len(chats) != 1 {
		t.Fatalf("chats len = %d, want 1", len(chats))
	}
	c := chats[0]
	if c.ChatID != "-1001234567890" {
		t.Fatalf("chatId = %q, want -1001234567890", c.ChatID)
	}
	if c.Title != "Team" || c.Type != "supergroup" {
		t.Fatalf("chat = %+v, want title Team type supergroup", c)
	}
	if len(c.Threads) != 1 {
		t.Fatalf("threads len = %d, want 1", len(c.Threads))
	}
	if c.Threads[0].ThreadID != "42" || c.Threads[0].Name != "Alerts" {
		t.Fatalf("thread = %+v, want {42 Alerts}", c.Threads[0])
	}
}

func TestResolveEmptyTokenDoesNotHitNetwork(t *testing.T) {
	ctx := context.Background()
	s := newService(t)
	// Installing a stub proves the empty-token guard short-circuits before any call.
	got := stubGetUpdates(t, `{"ok": true, "result": []}`)

	if _, err := s.Resolve(ctx, "   "); err == nil {
		t.Fatal("expected error for empty bot token")
	}
	if got.path != "" {
		t.Fatalf("Resolve with empty token must not call the Bot API, hit %q", got.path)
	}
}

func contains(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
