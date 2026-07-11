package profiles

import (
	"context"
	"errors"
	"path/filepath"
	"testing"

	"github.com/webcloster-dev/ai-reviewer/internal/adapters/sqlite"
	"github.com/webcloster-dev/ai-reviewer/internal/domain/profile"
)

func newService(t *testing.T) *Service {
	t.Helper()
	db, err := sqlite.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return NewService(sqlite.NewProfileStore(db))
}

func mustAdd(t *testing.T, s *Service, in AddInput) profile.Profile {
	t.Helper()
	p, err := s.Add(context.Background(), in)
	if err != nil {
		t.Fatalf("Add(%s): %v", in.Name, err)
	}
	return p
}

func TestAddSetsIDAndTimestamps(t *testing.T) {
	s := newService(t)
	p := mustAdd(t, s, AddInput{Name: "casual", Language: "es-AR", Formality: "casual", Emojis: true, Samples: []string{"che"}})

	if p.ID == "" {
		t.Fatal("Add should assign an id")
	}
	if p.CreatedAt.IsZero() || p.UpdatedAt.IsZero() {
		t.Fatalf("timestamps not set: %+v", p)
	}
	if p.StyleGuide != "" {
		t.Fatalf("styleGuide should stay empty, got %q", p.StyleGuide)
	}
}

func TestAddRequiresName(t *testing.T) {
	s := newService(t)
	if _, err := s.Add(context.Background(), AddInput{Name: ""}); err == nil {
		t.Fatal("expected name required error")
	}
}

func TestAddListGet(t *testing.T) {
	ctx := context.Background()
	s := newService(t)
	p := mustAdd(t, s, AddInput{Name: "one", Samples: []string{"a", "b"}})

	list, err := s.List(ctx)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(list) != 1 || list[0].ID != p.ID {
		t.Fatalf("List = %+v", list)
	}

	got, err := s.Get(ctx, p.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if len(got.Samples) != 2 || got.Samples[0] != "a" {
		t.Fatalf("samples not persisted: %+v", got.Samples)
	}
}

func TestUpdate(t *testing.T) {
	ctx := context.Background()
	s := newService(t)
	p := mustAdd(t, s, AddInput{Name: "old", Language: "en", Emojis: false, Samples: []string{"x"}})

	updated, err := s.Update(ctx, p.ID, UpdateInput{
		Name: "new", Language: "es", Formality: "formal", Emojis: true, Samples: []string{"y", "z"},
	})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if updated.Name != "new" || updated.Language != "es" || updated.Formality != "formal" || !updated.Emojis {
		t.Fatalf("fields not updated: %+v", updated)
	}
	if len(updated.Samples) != 2 || updated.Samples[1] != "z" {
		t.Fatalf("samples not updated: %+v", updated.Samples)
	}
	if !updated.UpdatedAt.After(p.UpdatedAt) && updated.UpdatedAt.Before(p.UpdatedAt) {
		t.Fatalf("updatedAt not advanced")
	}
}

func TestUpdateRequiresName(t *testing.T) {
	ctx := context.Background()
	s := newService(t)
	p := mustAdd(t, s, AddInput{Name: "one"})
	if _, err := s.Update(ctx, p.ID, UpdateInput{Name: ""}); err == nil {
		t.Fatal("expected name required error")
	}
}

func TestUpdateUnknown(t *testing.T) {
	s := newService(t)
	if _, err := s.Update(context.Background(), "nope", UpdateInput{Name: "x"}); !errors.Is(err, profile.ErrNotFound) {
		t.Fatalf("got %v, want ErrNotFound", err)
	}
}

func TestDelete(t *testing.T) {
	ctx := context.Background()
	s := newService(t)
	p := mustAdd(t, s, AddInput{Name: "gone"})

	if err := s.Delete(ctx, p.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if _, err := s.Get(ctx, p.ID); !errors.Is(err, profile.ErrNotFound) {
		t.Fatalf("Get after delete = %v, want ErrNotFound", err)
	}
}
