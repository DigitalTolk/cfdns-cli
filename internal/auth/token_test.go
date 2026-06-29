package auth

import (
	"errors"
	"testing"

	"github.com/zalando/go-keyring"
)

func TestStoreLoadDeleteRoundTrip(t *testing.T) {
	keyring.MockInit() // in-memory keychain, no real OS access

	// Nothing stored yet.
	if _, err := Load(); !errors.Is(err, ErrNoToken) {
		t.Fatalf("expected ErrNoToken before storing, got %v", err)
	}

	if err := Store("tok-123"); err != nil {
		t.Fatalf("Store: %v", err)
	}

	got, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got != "tok-123" {
		t.Fatalf("Load returned %q, want %q", got, "tok-123")
	}

	if err := Delete(); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if _, err := Load(); !errors.Is(err, ErrNoToken) {
		t.Fatalf("expected ErrNoToken after delete, got %v", err)
	}
}

func TestDeleteMissingIsNotAnError(t *testing.T) {
	keyring.MockInit()

	if err := Delete(); err != nil {
		t.Fatalf("deleting a missing token should be a no-op, got %v", err)
	}
}

func TestLoadSurfacesKeyringErrors(t *testing.T) {
	keyring.MockInitWithError(errors.New("keychain locked"))
	t.Cleanup(keyring.MockInit) // restore a clean mock for other tests

	_, err := Load()
	if err == nil {
		t.Fatal("expected an error when the keychain backend fails")
	}
	if errors.Is(err, ErrNoToken) {
		t.Fatalf("a backend failure must not be reported as ErrNoToken, got %v", err)
	}
}
