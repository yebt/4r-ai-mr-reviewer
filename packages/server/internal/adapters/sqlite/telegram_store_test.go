package sqlite

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/webcloster-dev/ai-reviewer/internal/domain/telegram"
)

func newTelegramStore(t *testing.T) *TelegramStore {
	t.Helper()
	db, err := Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return NewTelegramStore(db)
}

func mustCreateTarget(t *testing.T, s *TelegramStore, name string) telegram.Target {
	t.Helper()
	tg := telegram.Target{
		ID:        "id-" + name,
		Name:      name,
		ChatID:    "chat-" + name,
		ThreadID:  "",
		TokenRef:  "telegram:id-" + name + ":token",
		CreatedAt: time.Now().UTC().Truncate(time.Second),
	}
	if err := s.Create(context.Background(), tg); err != nil {
		t.Fatalf("Create(%s): %v", name, err)
	}
	return tg
}

func TestTelegramStoreRoundTrip(t *testing.T) {
	ctx := context.Background()
	s := newTelegramStore(t)

	want := telegram.Target{
		ID:        "t1",
		Name:      "team",
		ChatID:    "-100123",
		ThreadID:  "42",
		TokenRef:  "telegram:t1:token",
		CreatedAt: time.Now().UTC().Truncate(time.Second),
	}
	if err := s.Create(ctx, want); err != nil {
		t.Fatalf("Create: %v", err)
	}

	got, err := s.Get(ctx, "t1")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.Name != want.Name || got.ChatID != want.ChatID || got.ThreadID != want.ThreadID || got.TokenRef != want.TokenRef {
		t.Fatalf("round-trip mismatch: got %+v, want %+v", got, want)
	}
	if !got.CreatedAt.Equal(want.CreatedAt) {
		t.Fatalf("createdAt mismatch: got %v, want %v", got.CreatedAt, want.CreatedAt)
	}

	list, err := s.List(ctx)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("List len = %d, want 1", len(list))
	}
}

func TestTelegramStoreSetDefaultSwitches(t *testing.T) {
	ctx := context.Background()
	s := newTelegramStore(t)
	first := mustCreateTarget(t, s, "first")
	second := mustCreateTarget(t, s, "second")

	if err := s.SetDefault(ctx, first.ID); err != nil {
		t.Fatalf("SetDefault(first): %v", err)
	}
	def, err := s.GetDefault(ctx)
	if err != nil {
		t.Fatalf("GetDefault: %v", err)
	}
	if def.ID != first.ID {
		t.Fatalf("default id = %s, want %s", def.ID, first.ID)
	}

	// Switching the default must clear the previous one (single-default index).
	if err := s.SetDefault(ctx, second.ID); err != nil {
		t.Fatalf("SetDefault(second): %v", err)
	}
	def, _ = s.GetDefault(ctx)
	if def.ID != second.ID {
		t.Fatalf("default id = %s, want %s", def.ID, second.ID)
	}
	reloaded, _ := s.Get(ctx, first.ID)
	if reloaded.IsDefault {
		t.Fatal("former default should have been cleared")
	}
}

func TestTelegramStoreGetDefaultEmpty(t *testing.T) {
	s := newTelegramStore(t)
	if _, err := s.GetDefault(context.Background()); !errors.Is(err, telegram.ErrNotFound) {
		t.Fatalf("GetDefault with none set: got %v, want ErrNotFound", err)
	}
}

func TestTelegramStoreSetDefaultUnknown(t *testing.T) {
	s := newTelegramStore(t)
	if err := s.SetDefault(context.Background(), "nope"); !errors.Is(err, telegram.ErrNotFound) {
		t.Fatalf("SetDefault(unknown): got %v, want ErrNotFound", err)
	}
}
