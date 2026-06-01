// Package output formats results for the SwiftBar (and xbar) plugin
// protocol: lines above `---` are cycled in the menu bar, lines below
// appear in the dropdown when the user clicks the icon.
package output

import (
	"fmt"
	"io"
	"os"
	"sort"
	"strings"

	"github.com/bricklen/gmailnotifier/internal/mail"
)

const (
	iconNoMail  = ":envelope:"
	iconHasMail = ":e-mail:"

	maxSubjectLen = 60
	maxSenderLen  = 30
)

// Render writes the menu bar text + dropdown for the given results to w.
// execPath is the absolute path to this binary, used to wire up the
// "Configure accounts…" dropdown item.
func Render(w io.Writer, results []mail.Result, execPath string) {
	total := 0
	failures := 0
	for _, r := range results {
		if r.Err != nil {
			failures++
			continue
		}
		total += r.UnreadCount
	}

	// Menu bar line.
	switch {
	case total > 0:
		fmt.Fprintf(w, "%s %d\n", iconHasMail, total)
	case failures > 0 && failures == len(results):
		// Every account failed — surface that in the bar.
		fmt.Fprintf(w, "%s !\n", iconNoMail)
	default:
		fmt.Fprintf(w, "%s 0\n", iconNoMail)
	}

	fmt.Fprintln(w, "---")

	// Per-account summary, sorted for stable ordering.
	sorted := append([]mail.Result(nil), results...)
	sort.SliceStable(sorted, func(i, j int) bool { return sorted[i].Email < sorted[j].Email })

	for _, r := range sorted {
		if r.Err != nil {
			fmt.Fprintf(w, "%s: %s | color=red font=Menlo size=12\n",
				r.Email, escapeMenu(r.Err.Error()))
			continue
		}
		color := "navy"
		if r.UnreadCount == 0 {
			color = "gray"
		}
		fmt.Fprintf(w, "%s: %d unread | color=%s font=Menlo size=12\n",
			r.Email, r.UnreadCount, color)
	}

	// Preview lines for accounts with unread mail.
	hasAnyPreview := false
	for _, r := range sorted {
		if r.Err != nil || len(r.Preview) == 0 {
			continue
		}
		if !hasAnyPreview {
			fmt.Fprintln(w, "---")
			hasAnyPreview = true
		}
		fmt.Fprintf(w, "%s | font=Menlo size=11 color=#555555\n", r.Email)
		for _, m := range r.Preview {
			fmt.Fprintf(w, "  %s — %s | font=Menlo size=11\n",
				truncate(m.From, maxSenderLen),
				escapeMenu(truncate(m.Subject, maxSubjectLen)))
		}
		if r.UnreadCount > len(r.Preview) {
			fmt.Fprintf(w, "  … and %d more | font=Menlo size=11 color=gray\n",
				r.UnreadCount-len(r.Preview))
		}
	}

	// Footer.
	fmt.Fprintln(w, "---")
	fmt.Fprintln(w, "Open Gmail | href=https://mail.google.com/mail")
	if execPath != "" {
		// SwiftBar will run this shell command when the menu item is clicked.
		// `terminal=false` keeps it from popping a Terminal window — the
		// setup wizard opens its own Terminal via osascript when it needs
		// to prompt the user interactively.
		fmt.Fprintf(w, "Configure accounts… | shell=%q param1=setup terminal=false\n", execPath)
	}
	fmt.Fprintln(w, "Refresh | refresh=true")
}

// RenderError prints an error state to stdout. Used when the plugin can't
// even load its config — we still want the menu bar to show *something*.
func RenderError(msg string, execPath string) {
	fmt.Fprintf(os.Stdout, "%s !\n", iconNoMail)
	fmt.Println("---")
	fmt.Printf("gmailnotifier error: %s | color=red\n", escapeMenu(msg))
	if execPath != "" {
		fmt.Printf("Configure accounts… | shell=%q param1=setup terminal=false\n", execPath)
	}
	fmt.Println("Refresh | refresh=true")
}

// RenderNoAccounts is shown when the config is empty — guides the user
// straight into setup.
func RenderNoAccounts(execPath string) {
	fmt.Println(":envelope: setup")
	fmt.Println("---")
	fmt.Println("No Gmail accounts configured yet. | color=gray")
	if execPath != "" {
		fmt.Printf("Add an account… | shell=%q param1=setup terminal=false\n", execPath)
	}
	fmt.Println("Refresh | refresh=true")
}

func truncate(s string, n int) string {
	s = strings.TrimSpace(s)
	if len(s) <= n {
		return s
	}
	if n <= 1 {
		return s[:n]
	}
	return s[:n-1] + "…"
}

// escapeMenu strips characters that would confuse SwiftBar's key=value
// parameter parser when they end up in a menu label. Pipes are the main
// issue — anything after `|` becomes a parameter list.
func escapeMenu(s string) string {
	s = strings.ReplaceAll(s, "|", "/")
	s = strings.ReplaceAll(s, "\n", " ")
	s = strings.ReplaceAll(s, "\r", " ")
	return s
}
