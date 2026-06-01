package output

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/bricklen/gmailnotifier/internal/mail"
)

func TestRenderUnread(t *testing.T) {
	var buf bytes.Buffer
	Render(&buf, []mail.Result{
		{Email: "a@gmail.com", UnreadCount: 3, Preview: []mail.Message{
			{From: "Alice", Subject: "Hello"},
			{From: "Bob", Subject: "Re: project"},
		}},
		{Email: "b@gmail.com", UnreadCount: 0},
	}, "/usr/local/bin/gmailnotifier.30s")
	out := buf.String()

	if !strings.Contains(out, ":e-mail: 3") {
		t.Errorf("expected total count in header, got:\n%s", out)
	}
	if !strings.Contains(out, "a@gmail.com: 3 unread") {
		t.Errorf("expected per-account count for a, got:\n%s", out)
	}
	if !strings.Contains(out, "b@gmail.com: 0 unread") {
		t.Errorf("expected per-account count for b, got:\n%s", out)
	}
	if !strings.Contains(out, "Alice") || !strings.Contains(out, "Hello") {
		t.Errorf("expected preview lines, got:\n%s", out)
	}
	if !strings.Contains(out, "Configure accounts…") {
		t.Errorf("expected configure menu item, got:\n%s", out)
	}
}

func TestRenderNoMail(t *testing.T) {
	var buf bytes.Buffer
	Render(&buf, []mail.Result{
		{Email: "a@gmail.com", UnreadCount: 0},
	}, "/p")
	if !strings.Contains(buf.String(), ":envelope: 0") {
		t.Errorf("expected empty-state icon, got:\n%s", buf.String())
	}
}

func TestRenderAllFailures(t *testing.T) {
	var buf bytes.Buffer
	Render(&buf, []mail.Result{
		{Email: "a@gmail.com", Err: errors.New("network down")},
	}, "/p")
	out := buf.String()
	if !strings.Contains(out, ":envelope: !") {
		t.Errorf("expected error icon, got:\n%s", out)
	}
	if !strings.Contains(out, "network down") {
		t.Errorf("expected error text, got:\n%s", out)
	}
}

func TestEscapeMenuStripsPipes(t *testing.T) {
	got := escapeMenu("hello | world\nnext")
	if strings.ContainsAny(got, "|\n") {
		t.Errorf("escapeMenu left dangerous chars: %q", got)
	}
}

func TestTruncate(t *testing.T) {
	cases := []struct{ in, want string }{
		{"short", "short"},
		{strings.Repeat("a", 70), strings.Repeat("a", 59) + "…"},
		{"  trim  ", "trim"},
	}
	for _, c := range cases {
		if got := truncate(c.in, 60); got != c.want {
			t.Errorf("truncate(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestRenderEscapesPreviewPipes(t *testing.T) {
	var buf bytes.Buffer
	Render(&buf, []mail.Result{
		{Email: "a@gmail.com", UnreadCount: 1, Preview: []mail.Message{
			{From: "Evil", Subject: "click here | shell=/bin/rm"},
		}},
	}, "/p")
	out := buf.String()
	if strings.Contains(out, "Subject: click here | shell=/bin/rm") {
		t.Errorf("unsafe pipe leaked into menu line:\n%s", out)
	}
	// The subject text itself should appear (with the pipe replaced).
	if !strings.Contains(out, "click here / shell=/bin/rm") {
		t.Errorf("subject not escaped as expected:\n%s", out)
	}
}
