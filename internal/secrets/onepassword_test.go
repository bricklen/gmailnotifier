package secrets

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// fakeOp installs a shell-script `op` that returns a fixed value, on PATH.
func fakeOp(t *testing.T, stdout, stderr string, exitCode int) {
	t.Helper()
	dir := t.TempDir()
	script := "#!/bin/sh\n"
	if stderr != "" {
		script += `printf '%s' "` + stderr + `" >&2` + "\n"
	}
	if stdout != "" {
		script += `printf '%s' "` + stdout + `"` + "\n"
	}
	script += "exit " + itoa(exitCode) + "\n"
	path := filepath.Join(dir, "op")
	if err := os.WriteFile(path, []byte(script), 0o755); err != nil {
		t.Fatalf("write fake op: %v", err)
	}
	// Reset cache and prepend the fake to PATH.
	opPath = ""
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
	t.Cleanup(func() { opPath = "" })
}

func itoa(i int) string {
	if i == 0 {
		return "0"
	}
	neg := ""
	if i < 0 {
		neg = "-"
		i = -i
	}
	var digits []byte
	for i > 0 {
		digits = append([]byte{byte('0' + i%10)}, digits...)
		i /= 10
	}
	return neg + string(digits)
}

func TestOPReadHappyPath(t *testing.T) {
	fakeOp(t, "super-secret", "", 0)
	got, err := opRead("op://Personal/Gmail/password")
	if err != nil {
		t.Fatalf("opRead: %v", err)
	}
	if got != "super-secret" {
		t.Errorf("got %q, want %q", got, "super-secret")
	}
}

func TestOPReadRejectsBadReference(t *testing.T) {
	fakeOp(t, "should-not-be-called", "", 0)
	for _, ref := range []string{
		"",
		"Personal/Gmail/password",
		"https://example.com",
		"; rm -rf /",
	} {
		if _, err := opRead(ref); err == nil {
			t.Errorf("expected error for reference %q", ref)
		}
	}
}

func TestOPReadPropagatesError(t *testing.T) {
	fakeOp(t, "", "no such item", 1)
	_, err := opRead("op://x/y/z")
	if err == nil || !strings.Contains(err.Error(), "no such item") {
		t.Errorf("expected wrapped stderr, got %v", err)
	}
}

func TestOPAvailableMissing(t *testing.T) {
	// Point PATH at an empty dir.
	dir := t.TempDir()
	opPath = ""
	t.Setenv("PATH", dir)
	t.Cleanup(func() { opPath = "" })
	if OPAvailable() {
		t.Error("OPAvailable() = true with empty PATH")
	}
}
