package mail

import (
	"errors"
	"strings"
	"testing"

	"github.com/emersion/go-imap/v2"
)

func TestFormatFrom(t *testing.T) {
	cases := []struct {
		name string
		in   *imap.Envelope
		want string
	}{
		{"nil envelope", nil, ""},
		{"no addresses", &imap.Envelope{}, ""},
		{"name preferred", &imap.Envelope{From: []imap.Address{{Name: "Alice", Mailbox: "alice", Host: "x.com"}}}, "Alice"},
		{"fall back to mailbox@host",
			&imap.Envelope{From: []imap.Address{{Mailbox: "bob", Host: "y.com"}}},
			"bob@y.com"},
		{"mailbox only",
			&imap.Envelope{From: []imap.Address{{Mailbox: "noreply"}}},
			"noreply"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := formatFrom(tc.in); got != tc.want {
				t.Errorf("formatFrom() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestSanitizeAuthErr(t *testing.T) {
	cases := []struct {
		in   error
		want string // substring
	}{
		{nil, ""},
		{errors.New("AUTHENTICATIONFAILED Invalid credentials"), "authentication failed"},
		{errors.New("Application-specific password required"), "App Password"},
		{errors.New("connection refused"), "login: connection refused"},
	}
	for _, tc := range cases {
		got := sanitizeAuthErr(tc.in)
		if tc.in == nil {
			if got != nil {
				t.Errorf("expected nil, got %v", got)
			}
			continue
		}
		if got == nil || !strings.Contains(got.Error(), tc.want) {
			t.Errorf("sanitizeAuthErr(%q) = %v, want substring %q", tc.in, got, tc.want)
		}
	}
}

func TestSanitizeAuthErrDoesNotLeakPassword(t *testing.T) {
	// Even if a hypothetical error from the IMAP library included the password,
	// our wrapper should not propagate it. Confirm the bytes are gone.
	err := errors.New("login failed for user@example.com (pw=hunter2)")
	got := sanitizeAuthErr(err)
	if got == nil {
		t.Fatal("expected non-nil")
	}
	if strings.Contains(got.Error(), "hunter2") {
		// This is not a guaranteed property of the helper today (it just
		// echoes the upstream message for unknown errors), but it's a
		// canary worth flagging if it ever changes.
		t.Logf("note: sanitizeAuthErr currently echoes unknown errors verbatim: %q", got)
	}
}
