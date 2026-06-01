//go:build !darwin

package secrets

import "errors"

// On non-darwin platforms the Keychain backend is a no-op stub so the
// package still builds (useful for CI cross-compile checks and tests).
// The binary itself only ships for darwin.

func keychainSet(_, _ string) error {
	return errors.New("keychain backend is only available on macOS")
}

func keychainGet(_ string) (string, error) {
	return "", errors.New("keychain backend is only available on macOS")
}

func keychainDelete(_ string) error {
	return errors.New("keychain backend is only available on macOS")
}
