package sqlite

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/webcloster-dev/ai-reviewer/internal/domain/profile"
	"github.com/webcloster-dev/ai-reviewer/internal/id"
)

func newProfileStore(t *testing.T) *ProfileStore {
	t.Helper()
	db, err := Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return NewProfileStore(db)
}

func TestProfileCreateGet(t *testing.T) {
	ctx := context.Background()
	s := newProfileStore(t)

	now := time.Now().UTC()
	p := profile.Profile{
		ID: id.New(), Name: "casual-es", Language: "es-AR", Formality: "casual",
		Emojis: true, Samples: []string{"che, mirá esto", "buenísimo el cambio"},
		CreatedAt: now, UpdatedAt: now,
	}
	if err := s.Create(ctx, p); err != nil {
		t.Fatalf("Create: %v", err)
	}

	got, err := s.Get(ctx, p.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.Name != "casual-es" || got.Language != "es-AR" || got.Formality != "casual" {
		t.Fatalf("unexpected profile: %+v", got)
	}
	if !got.Emojis {
		t.Fatalf("emojis = %v, want true", got.Emojis)
	}
	if len(got.Samples) != 2 || got.Samples[0] != "che, mirá esto" {
		t.Fatalf("samples not persisted: %+v", got.Samples)
	}
	if got.StyleGuide != "" {
		t.Fatalf("styleGuide should be empty, got %q", got.StyleGuide)
	}
}

func TestProfileSaveRoundTrip(t *testing.T) {
	ctx := context.Background()
	s := newProfileStore(t)

	now := time.Now().UTC()
	p := profile.Profile{
		ID: id.New(), Name: "one", Language: "en", Emojis: false,
		Samples: []string{"first"}, CreatedAt: now, UpdatedAt: now,
	}
	if err := s.Create(ctx, p); err != nil {
		t.Fatalf("Create: %v", err)
	}

	p.Name = "one-edited"
	p.Language = "es"
	p.Formality = "formal"
	p.Emojis = true
	p.Samples = []string{"a", "b", "c"}
	p.UpdatedAt = now.Add(time.Minute)
	if err := s.Save(ctx, p); err != nil {
		t.Fatalf("Save: %v", err)
	}

	got, err := s.Get(ctx, p.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.Name != "one-edited" || got.Language != "es" || got.Formality != "formal" || !got.Emojis {
		t.Fatalf("fields not updated: %+v", got)
	}
	if len(got.Samples) != 3 || got.Samples[2] != "c" {
		t.Fatalf("samples not updated: %+v", got.Samples)
	}
}

func TestProfileSaveMissing(t *testing.T) {
	s := newProfileStore(t)
	p := profile.Profile{ID: "nope", Name: "x", CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC()}
	if err := s.Save(context.Background(), p); !errors.Is(err, profile.ErrNotFound) {
		t.Fatalf("got %v, want ErrNotFound", err)
	}
}

func TestProfileGetMissing(t *testing.T) {
	s := newProfileStore(t)
	if _, err := s.Get(context.Background(), "nope"); !errors.Is(err, profile.ErrNotFound) {
		t.Fatalf("got %v, want ErrNotFound", err)
	}
}

func TestProfileDelete(t *testing.T) {
	ctx := context.Background()
	s := newProfileStore(t)

	now := time.Now().UTC()
	p := profile.Profile{ID: id.New(), Name: "gone", CreatedAt: now, UpdatedAt: now}
	if err := s.Create(ctx, p); err != nil {
		t.Fatalf("Create: %v", err)
	}
	if err := s.Delete(ctx, p.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if _, err := s.Get(ctx, p.ID); !errors.Is(err, profile.ErrNotFound) {
		t.Fatalf("Get after delete = %v, want ErrNotFound", err)
	}
}

func TestProfileDeleteMissing(t *testing.T) {
	s := newProfileStore(t)
	if err := s.Delete(context.Background(), "nope"); !errors.Is(err, profile.ErrNotFound) {
		t.Fatalf("got %v, want ErrNotFound", err)
	}
}

func TestProfileList(t *testing.T) {
	ctx := context.Background()
	s := newProfileStore(t)

	now := time.Now().UTC()
	_ = s.Create(ctx, profile.Profile{ID: id.New(), Name: "beta", CreatedAt: now, UpdatedAt: now})
	_ = s.Create(ctx, profile.Profile{ID: id.New(), Name: "alpha", CreatedAt: now, UpdatedAt: now})

	list, err := s.List(ctx)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(list) != 2 {
		t.Fatalf("len = %d, want 2", len(list))
	}
	// Ordered by name.
	if list[0].Name != "alpha" || list[1].Name != "beta" {
		t.Fatalf("expected name order, got %+v", list)
	}
}
