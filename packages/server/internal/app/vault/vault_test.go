package vault

import (
	"context"
	"errors"
	"path/filepath"
	"testing"

	"github.com/webcloster-dev/ai-reviewer/internal/adapters/crypto"
)

// fakeRekeyer stands in for the secret store: it applies the meta updates to the
// shared MetaStore (so the vault's Verify/Status see the new key material) and
// records the last cipher it was handed.
type fakeRekeyer struct {
	meta *fakeMeta
	last *crypto.Cipher
}

func (r *fakeRekeyer) Rekey(_ context.Context, newCipher *crypto.Cipher, meta map[string][]byte) error {
	for k, v := range meta {
		r.meta.m[k] = v
	}
	r.last = newCipher
	return nil
}

func TestServiceChangeSecret(t *testing.T) {
	ctx := context.Background()
	meta := newFakeMeta()
	v := New(meta, filepath.Join(t.TempDir(), "master.key"))
	if _, err := v.Initialize(ctx, "old-pass"); err != nil {
		t.Fatalf("Initialize: %v", err)
	}
	svc := NewService(v, &fakeRekeyer{meta: meta})

	// The current password gates the change.
	if _, err := svc.ChangeSecret(ctx, "wrong", "new-pass"); !errors.Is(err, ErrWrongPassword) {
		t.Fatalf("ChangeSecret with wrong old password = %v, want ErrWrongPassword", err)
	}

	// Change the password: still protected, and only the NEW password verifies.
	st, err := svc.ChangeSecret(ctx, "old-pass", "new-pass")
	if err != nil {
		t.Fatalf("ChangeSecret: %v", err)
	}
	if !st.PasswordProtected {
		t.Fatal("want password-protected after a password change")
	}
	if err := v.Verify(ctx, "new-pass"); err != nil {
		t.Fatalf("new password should verify: %v", err)
	}
	if err := v.Verify(ctx, "old-pass"); !errors.Is(err, ErrWrongPassword) {
		t.Fatalf("old password should no longer verify: %v", err)
	}

	// Switch to key-file mode (empty new password): no longer password-protected.
	st, err = svc.ChangeSecret(ctx, "new-pass", "")
	if err != nil {
		t.Fatalf("ChangeSecret to key-file: %v", err)
	}
	if st.PasswordProtected {
		t.Fatal("want key-file mode (not password-protected)")
	}
}

// fakeMeta is an in-memory MetaStore for unit tests.
type fakeMeta struct{ m map[string][]byte }

func newFakeMeta() *fakeMeta { return &fakeMeta{m: map[string][]byte{}} }

func (f *fakeMeta) Get(_ context.Context, key string) ([]byte, bool, error) {
	v, ok := f.m[key]
	return v, ok, nil
}

func (f *fakeMeta) Set(_ context.Context, key string, value []byte) error {
	f.m[key] = value
	return nil
}

func newVault(t *testing.T) *Vault {
	t.Helper()
	return New(newFakeMeta(), filepath.Join(t.TempDir(), "master.key"))
}

func TestPasswordModeRoundTrip(t *testing.T) {
	ctx := context.Background()
	v := newVault(t)

	initCipher, err := v.Initialize(ctx, "hunter2")
	if err != nil {
		t.Fatalf("Initialize: %v", err)
	}
	sealed, _ := initCipher.Seal([]byte("token"))

	unlocked, err := v.Unlock(ctx, "hunter2")
	if err != nil {
		t.Fatalf("Unlock: %v", err)
	}
	got, err := unlocked.Open(sealed)
	if err != nil {
		t.Fatalf("Open with unlocked cipher: %v", err)
	}
	if string(got) != "token" {
		t.Fatalf("round trip mismatch: %q", got)
	}
}

func TestWrongPassword(t *testing.T) {
	ctx := context.Background()
	v := newVault(t)
	if _, err := v.Initialize(ctx, "right"); err != nil {
		t.Fatalf("Initialize: %v", err)
	}
	if _, err := v.Unlock(ctx, "wrong"); !errors.Is(err, ErrWrongPassword) {
		t.Fatalf("Unlock wrong password: got %v, want ErrWrongPassword", err)
	}
}

func TestKeyfileModeIgnoresPassword(t *testing.T) {
	ctx := context.Background()
	v := newVault(t)

	initCipher, err := v.Initialize(ctx, "")
	if err != nil {
		t.Fatalf("Initialize keyfile: %v", err)
	}
	sealed, _ := initCipher.Seal([]byte("data"))

	unlocked, err := v.Unlock(ctx, "anything-is-ignored")
	if err != nil {
		t.Fatalf("Unlock keyfile: %v", err)
	}
	got, err := unlocked.Open(sealed)
	if err != nil || string(got) != "data" {
		t.Fatalf("keyfile round trip failed: got %q err %v", got, err)
	}
}

func TestStatusReflectsMode(t *testing.T) {
	ctx := context.Background()

	pw := newVault(t)
	if st, _ := pw.Status(ctx); st.Initialized {
		t.Fatal("fresh vault should not be initialized")
	}
	_, _ = pw.Initialize(ctx, "pw")
	if st, _ := pw.Status(ctx); !st.Initialized || !st.PasswordProtected {
		t.Fatalf("password vault status = %+v", st)
	}

	kf := newVault(t)
	_, _ = kf.Initialize(ctx, "")
	if st, _ := kf.Status(ctx); !st.Initialized || st.PasswordProtected {
		t.Fatalf("keyfile vault status = %+v", st)
	}
}

func TestInitializeTwiceFails(t *testing.T) {
	ctx := context.Background()
	v := newVault(t)
	if _, err := v.Initialize(ctx, "pw"); err != nil {
		t.Fatalf("first Initialize: %v", err)
	}
	if _, err := v.Initialize(ctx, "pw"); !errors.Is(err, ErrAlreadyInit) {
		t.Fatalf("second Initialize: got %v, want ErrAlreadyInit", err)
	}
}

func TestUnlockBeforeInitFails(t *testing.T) {
	v := newVault(t)
	if _, err := v.Unlock(context.Background(), "pw"); !errors.Is(err, ErrNotInitialized) {
		t.Fatalf("Unlock before init: got %v, want ErrNotInitialized", err)
	}
}
