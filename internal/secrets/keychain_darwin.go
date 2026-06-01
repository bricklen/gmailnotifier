//go:build darwin

package secrets

import (
	"errors"
	"fmt"
	"os/exec"
	"strings"
)

// keychainSet stores password under (service=Service, account=email) in the
// default login Keychain. If an item already exists, it is replaced.
//
// Implementation note: we shell out to /usr/bin/security rather than link
// against the Security framework via cgo. This keeps the build pure-Go,
// cross-compilable from any arch, and easy to audit — every interaction
// with the Keychain is visible as a process invocation.
func keychainSet(email, password string) error {
	// Best-effort delete first so add-generic-password doesn't complain.
	_ = keychainDelete(email)
	cmd := exec.Command(
		"/usr/bin/security",
		"add-generic-password",
		"-a", email,
		"-s", Service,
		"-w", password,
		"-U", // update if exists
		"-T", "", // no apps are pre-authorized (each access prompts the user once)
	)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("security add-generic-password: %w: %s", err, strings.TrimSpace(string(out)))
	}
	return nil
}

func keychainGet(email string) (string, error) {
	cmd := exec.Command(
		"/usr/bin/security",
		"find-generic-password",
		"-a", email,
		"-s", Service,
		"-w", // print password to stdout, nothing else
	)
	out, err := cmd.Output()
	if err != nil {
		var ee *exec.ExitError
		if errors.As(err, &ee) {
			// security exits 44 when the item is not found.
			if ee.ExitCode() == 44 {
				return "", ErrNotFound
			}
			return "", fmt.Errorf("security find-generic-password (exit %d): %s",
				ee.ExitCode(), strings.TrimSpace(string(ee.Stderr)))
		}
		return "", err
	}
	// `security -w` prints the password followed by a trailing newline.
	return strings.TrimRight(string(out), "\n"), nil
}

func keychainDelete(email string) error {
	cmd := exec.Command(
		"/usr/bin/security",
		"delete-generic-password",
		"-a", email,
		"-s", Service,
	)
	out, err := cmd.CombinedOutput()
	if err != nil {
		var ee *exec.ExitError
		if errors.As(err, &ee) && ee.ExitCode() == 44 {
			return nil // already absent
		}
		return fmt.Errorf("security delete-generic-password: %w: %s", err, strings.TrimSpace(string(out)))
	}
	return nil
}
