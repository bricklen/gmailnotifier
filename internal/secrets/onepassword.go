package secrets

import (
	"errors"
	"fmt"
	"os/exec"
	"strings"
)

// opPath is a package-level variable so tests can swap in a fake binary.
var opPath = ""

// OPAvailable reports whether the 1Password CLI (`op`) is installed and
// resolvable on $PATH. Result is cached for the lifetime of the process.
func OPAvailable() bool {
	return opLocate() != ""
}

// OPPath returns the resolved absolute path to the `op` binary, or "" if
// not installed.
func OPPath() string { return opLocate() }

func opLocate() string {
	if opPath != "" {
		return opPath
	}
	p, err := exec.LookPath("op")
	if err != nil {
		return ""
	}
	opPath = p
	return opPath
}

// opRead reads a secret from 1Password using `op read <reference>`.
// `reference` must be a secret reference in the form `op://vault/item/field`.
//
// We refuse references that don't start with the `op://` scheme to avoid
// passing arbitrary strings to the CLI. The `op` binary itself enforces
// vault permissions and biometric/desktop-app authorization.
func opRead(reference string) (string, error) {
	ref := strings.TrimSpace(reference)
	if !strings.HasPrefix(ref, "op://") {
		return "", fmt.Errorf("invalid 1password reference %q (expected op://vault/item/field)", reference)
	}
	bin := opLocate()
	if bin == "" {
		return "", errors.New("1password CLI (op) not found on $PATH")
	}
	cmd := exec.Command(bin, "read", "--no-newline", ref)
	out, err := cmd.Output()
	if err != nil {
		var ee *exec.ExitError
		if errors.As(err, &ee) {
			return "", fmt.Errorf("op read failed (exit %d): %s",
				ee.ExitCode(), strings.TrimSpace(string(ee.Stderr)))
		}
		return "", fmt.Errorf("op read: %w", err)
	}
	return string(out), nil
}
