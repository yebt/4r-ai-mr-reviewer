package sqlite

import (
	"bytes"
	"context"
	"errors"
	"path/filepath"
	"testing"

	"github.com/webcloster-dev/ai-reviewer/internal/adapters/crypto"
	"github.com/webcloster-dev/ai-reviewer/internal/domain/secret"
)

func newTestStore(t *testing.T) *SecretStore {
	t.Helper()
	db, err := Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	salt, _ := crypto.NewSalt()
	key, _ := crypto.DeriveKey("test-password", salt)
	cipher, err := crypto.NewCipher(key)
	if err != nil {
		t.Fatalf("NewCipher: %v", err)
	}
	return NewSecretStore(db, cipher)
}

func TestSecretStoreSetGet(t *testing.T) {
	ctx := context.Background()
	s := newTestStore(t)

	want := []byte("glpat-token-123")
	if err := s.Set(ctx, "gitlab.pat", want); err != nil {
		t.Fatalf("Set: %v", err)
	}
	got, err := s.Get(ctx, "gitlab.pat")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("Get = %q, want %q", got, want)
	}
}

func TestSecretStoreCiphertextAtRest(t *testing.T) {
	ctx := context.Background()
	s := newTestStore(t)
	secretValue := []byte("do-not-store-in-plaintext")
	if err := s.Set(ctx, "k", secretValue); err != nil {
		t.Fatalf("Set: %v", err)
	}

	var stored []byte
	if err := s.db.QueryRowContext(ctx, `SELECT ciphertext FROM secrets WHERE name = 'k'`).Scan(&stored); err != nil {
		t.Fatalf("read raw: %v", err)
	}
	if bytes.Contains(stored, secretValue) {
		t.Fatal("plaintext found in the database column")
	}
}

func TestSecretStoreGetMissing(t *testing.T) {
	s := newTestStore(t)
	_, err := s.Get(context.Background(), "nope")
	if !errors.Is(err, secret.ErrNotFound) {
		t.Fatalf("Get missing: got %v, want ErrNotFound", err)
	}
}

func TestSecretStoreSetOverwrites(t *testing.T) {
	ctx := context.Background()
	s := newTestStore(t)
	_ = s.Set(ctx, "k", []byte("first"))
	_ = s.Set(ctx, "k", []byte("second"))

	got, err := s.Get(ctx, "k")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if string(got) != "second" {
		t.Fatalf("expected overwrite, got %q", got)
	}
}

func TestSecretStoreExistsDeleteList(t *testing.T) {
	ctx := context.Background()
	s := newTestStore(t)

	_ = s.Set(ctx, "a", []byte("1"))
	_ = s.Set(ctx, "b", []byte("2"))

	if ok, _ := s.Exists(ctx, "a"); !ok {
		t.Fatal("Exists(a) = false, want true")
	}
	if ok, _ := s.Exists(ctx, "missing"); ok {
		t.Fatal("Exists(missing) = true, want false")
	}

	infos, err := s.List(ctx)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(infos) != 2 {
		t.Fatalf("List len = %d, want 2", len(infos))
	}

	if err := s.Delete(ctx, "a"); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if ok, _ := s.Exists(ctx, "a"); ok {
		t.Fatal("Exists(a) after delete = true, want false")
	}
	// Deleting a missing secret is a no-op, not an error.
	if err := s.Delete(ctx, "a"); err != nil {
		t.Fatalf("Delete missing: %v", err)
	}
}
