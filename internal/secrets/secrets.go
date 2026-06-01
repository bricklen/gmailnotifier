// Package secrets resolves passwords for accounts from one of several
// backends: macOS Keychain, 1Password (via the `op` CLI), or — only as a
// legacy migration path — plaintext that the caller already holds.
//
// Passwords are never logged and never written to disk by this package.
package secrets

import (
	"errors"
	"fmt"

	"github.com/bricklen/gmailnotifier/internal/config"
)

// Service is the Keychain service name. Keep this stable; changing it
// orphans every previously stored password.
const Service = "gmailnotifier"

// ErrNotFound is returned when no password exists for the account.
var ErrNotFound = errors.New("secret not found")

// Store stores a password for the account in the Keychain.
// For 1Password-backed accounts, no Keychain entry is written — the
// reference in the config points at the 1Password item instead.
func Store(a config.Account, password string) error {
	switch a.Source {
	case config.SourceKeychain:
		return keychainSet(a.Email, password)
	case config.SourceOnePassword:
		return fmt.Errorf("cannot store password for 1password-backed account %q; add the item in 1Password instead", a.Email)
	case config.SourcePlaintext:
		return fmt.Errorf("cannot store password for plaintext-backed account %q; migrate to keychain first", a.Email)
	default:
		return fmt.Errorf("unknown source %q", a.Source)
	}
}

// Resolve returns the password for the account by consulting its backend.
func Resolve(a config.Account) (string, error) {
	switch a.Source {
	case config.SourceKeychain:
		p, err := keychainGet(a.Email)
		if err != nil {
			return "", fmt.Errorf("keychain lookup for %s: %w", a.Email, err)
		}
		return p, nil
	case config.SourceOnePassword:
		p, err := opRead(a.Reference)
		if err != nil {
			return "", fmt.Errorf("1password lookup for %s: %w", a.Email, err)
		}
		return p, nil
	case config.SourcePlaintext:
		if a.Password == "" {
			return "", fmt.Errorf("%w: plaintext-backed account %s has no in-memory password (legacy file missing?)", ErrNotFound, a.Email)
		}
		return a.Password, nil
	default:
		return "", fmt.Errorf("unknown source %q for %s", a.Source, a.Email)
	}
}

// Delete removes any Keychain entry for the account. Safe to call even if
// no entry exists.
func Delete(email string) error {
	return keychainDelete(email)
}
