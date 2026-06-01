// Package mail checks Gmail accounts for unread messages via IMAPS.
//
// The connection is always TLS (port 993) with TLSv1.2 as the minimum
// negotiated version and Gmail's hostname verified against the server's
// certificate. We use the official Gmail IMAP endpoint, imap.gmail.com.
package mail

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/emersion/go-imap/v2"
	"github.com/emersion/go-imap/v2/imapclient"
)

// Server is the Gmail IMAP endpoint. Exposed as a variable so tests can
// retarget it at a fake server.
var Server = "imap.gmail.com:993"

// DefaultTimeout caps each per-account check. SwiftBar reaps long-running
// plugins, so we'd rather fail one account fast than hang the whole bar.
const DefaultTimeout = 10 * time.Second

// MaxPreview is the maximum number of unread message previews to fetch per
// account. The dropdown shows at most this many; the count itself is
// independent of this limit.
const MaxPreview = 8

// Message is a tiny view of an unread email used for the dropdown preview.
type Message struct {
	From    string
	Subject string
}

// Result is the outcome of one account check.
type Result struct {
	Email       string
	UnreadCount int
	Preview     []Message
	Err         error
}

// Check connects to Gmail IMAP, authenticates with the given password, and
// returns the unread count and a short preview of the most recent unread
// messages. Any error is returned in Result.Err rather than panicking, so a
// single bad account never takes the whole notifier down.
func Check(ctx context.Context, email, password string) Result {
	r := Result{Email: email}

	ctx, cancel := context.WithTimeout(ctx, DefaultTimeout)
	defer cancel()

	host, _, err := net.SplitHostPort(Server)
	if err != nil {
		r.Err = fmt.Errorf("invalid server %q: %w", Server, err)
		return r
	}

	dialer := &net.Dialer{Timeout: DefaultTimeout}
	conn, err := tls.DialWithDialer(dialer, "tcp", Server, &tls.Config{
		ServerName: host,
		MinVersion: tls.VersionTLS12,
	})
	if err != nil {
		r.Err = fmt.Errorf("tls dial %s: %w", Server, err)
		return r
	}

	client := imapclient.New(conn, nil)
	defer func() { _ = client.Close() }()

	// Tie the IMAP commands to the context's deadline as best we can —
	// go-imap doesn't take a context per command, so we close the
	// connection out from under it if the context fires.
	done := make(chan struct{})
	defer close(done)
	go func() {
		select {
		case <-ctx.Done():
			_ = conn.Close()
		case <-done:
		}
	}()

	if err := client.Login(email, password).Wait(); err != nil {
		r.Err = sanitizeAuthErr(err)
		return r
	}
	defer func() { _ = client.Logout().Wait() }()

	// Gmail's "All Mail" includes messages from every label; selecting
	// INBOX matches what most users mean by "unread email." Selected in
	// read-only mode so we don't accidentally mark anything as seen.
	mbox, err := client.Select("INBOX", &imap.SelectOptions{ReadOnly: true}).Wait()
	if err != nil {
		r.Err = fmt.Errorf("select INBOX: %w", err)
		return r
	}
	if mbox == nil {
		r.Err = errors.New("select INBOX: empty mailbox descriptor")
		return r
	}

	// Search for unseen messages.
	searchData, err := client.Search(&imap.SearchCriteria{
		NotFlag: []imap.Flag{imap.FlagSeen},
	}, nil).Wait()
	if err != nil {
		r.Err = fmt.Errorf("search unseen: %w", err)
		return r
	}
	if searchData == nil {
		return r
	}
	seqs := searchData.AllSeqNums()
	r.UnreadCount = len(seqs)
	if r.UnreadCount == 0 {
		return r
	}

	// Fetch envelopes for the most recent N unread messages. Sequence
	// numbers within INBOX are oldest→newest, so take the tail.
	previewCount := r.UnreadCount
	if previewCount > MaxPreview {
		previewCount = MaxPreview
	}
	tail := seqs[len(seqs)-previewCount:]
	set := imap.SeqSetNum(tail...)

	msgs, err := client.Fetch(set, &imap.FetchOptions{Envelope: true}).Collect()
	if err != nil {
		// Preview is best-effort — return the count even if envelopes fail.
		return r
	}
	for _, m := range msgs {
		if m.Envelope == nil {
			continue
		}
		r.Preview = append(r.Preview, Message{
			From:    formatFrom(m.Envelope),
			Subject: strings.TrimSpace(m.Envelope.Subject),
		})
	}
	// Reverse so newest first.
	for i, j := 0, len(r.Preview)-1; i < j; i, j = i+1, j-1 {
		r.Preview[i], r.Preview[j] = r.Preview[j], r.Preview[i]
	}
	return r
}

func formatFrom(env *imap.Envelope) string {
	if env == nil || len(env.From) == 0 {
		return ""
	}
	a := env.From[0]
	if a.Name != "" {
		return a.Name
	}
	if a.Mailbox != "" && a.Host != "" {
		return a.Mailbox + "@" + a.Host
	}
	return a.Mailbox
}

// sanitizeAuthErr returns a user-friendly message for the common Gmail
// "AUTHENTICATIONFAILED" response without leaking the password back into
// the error string (the underlying library is careful but better safe).
func sanitizeAuthErr(err error) error {
	if err == nil {
		return nil
	}
	msg := err.Error()
	low := strings.ToLower(msg)
	switch {
	case strings.Contains(low, "authenticationfailed"),
		strings.Contains(low, "invalid credentials"):
		return errors.New("authentication failed (check the Gmail App Password)")
	case strings.Contains(low, "application-specific password required"):
		return errors.New("Gmail requires an App Password (2FA accounts cannot use the login password)")
	default:
		return fmt.Errorf("login: %s", msg)
	}
}
