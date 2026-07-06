package vault

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
)

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
