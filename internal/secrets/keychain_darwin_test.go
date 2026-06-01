//go:build darwin && keychain

// These tests touch the real macOS Keychain. They are gated behind the
// `keychain` build tag so CI doesn't trigger interactive prompts. Run
// locally with:  go test -tags=keychain ./internal/secrets/...
package secrets

import (
	"errors"
	"testing"
)

func TestKeychainRoundtrip(t *testing.T) {
	const email = "gmailnotifier-test@example.invalid"
	const pw = "round-trip-password-🌟"

	t.Cleanup(func() { _ = keychainDelete(email) })

	if err := keychainSet(email, pw); err != nil {
		t.Fatalf("set: %v", err)
	}
	got, err := keychainGet(email)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got != pw {
		t.Errorf("got %q, want %q", got, pw)
	}
	if err := keychainDelete(email); err != nil {
		t.Errorf("delete: %v", err)
	}
	if _, err := keychainGet(email); !errors.Is(err, ErrNotFound) {
		t.Errorf("after delete, get err = %v, want ErrNotFound", err)
	}
}
