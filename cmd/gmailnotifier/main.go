// Command gmailnotifier is a SwiftBar (and xbar-compatible) plugin that
// shows the number of unread Gmail messages across one or more accounts.
//
// Usage:
//
//	gmailnotifier            # run the menu-bar check (default; what SwiftBar invokes)
//	gmailnotifier setup      # interactive setup / reconfigure wizard
//	gmailnotifier version    # print version and exit
//
// SwiftBar / xbar metadata follows. The `.30s.` middle component of the
// installed filename (gmailnotifier.30s.bin) tells the host how often to
// invoke this binary; the metadata below is read by the "About" panel.
//
//<xbar.title>Gmail Notifier</xbar.title>
//<xbar.version>v2.0.1</xbar.version>
//<xbar.author>bricklen</xbar.author>
//<xbar.author.github>bricklen</xbar.author.github>
//<xbar.desc>Encrypted multi-account Gmail unread-mail notifier (IMAP + Keychain + optional 1Password).</xbar.desc>
//<xbar.image>https://raw.githubusercontent.com/bricklen/gmailnotifier/master/docs/logo.png</xbar.image>
//<xbar.abouturl>https://github.com/bricklen/gmailnotifier</xbar.abouturl>
//<swiftbar.hideRunInTerminal>true</swiftbar.hideRunInTerminal>
//<swiftbar.hideDisablePlugin>false</swiftbar.hideDisablePlugin>
package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/bricklen/gmailnotifier/internal/config"
	"github.com/bricklen/gmailnotifier/internal/mail"
	"github.com/bricklen/gmailnotifier/internal/output"
	"github.com/bricklen/gmailnotifier/internal/secrets"
	"github.com/bricklen/gmailnotifier/internal/setup"
)

// version is overridden at build time via -ldflags.
var version = "dev"

func main() {
	ctx := context.Background()

	exePath := ""
	if p, err := os.Executable(); err == nil {
		exePath, _ = filepath.Abs(p)
	}

	args := os.Args[1:]
	cmd := ""
	if len(args) > 0 {
		cmd = args[0]
	}

	switch cmd {
	case "setup", "configure":
		if err := setup.Run(ctx); err != nil {
			fmt.Fprintln(os.Stderr, "setup:", err)
			os.Exit(1)
		}
	case "version", "--version", "-v":
		fmt.Println("gmailnotifier", version)
	case "", "check":
		runCheck(ctx, exePath)
	default:
		fmt.Fprintf(os.Stderr, "unknown command %q\n\nUsage:\n  gmailnotifier            # menu-bar check\n  gmailnotifier setup      # interactive setup\n  gmailnotifier version    # print version\n", cmd)
		os.Exit(2)
	}
}

func runCheck(ctx context.Context, exePath string) {
	cfg, err := config.Load()
	if err != nil {
		output.RenderError(err.Error(), exePath)
		return
	}
	// Auto-migrate legacy file if present and no accounts configured yet.
	if len(cfg.Accounts) == 0 {
		if migrated, _ := autoLoadLegacy(cfg); migrated > 0 {
			// keep going with the in-memory legacy accounts; setup wizard
			// will offer to move them to Keychain.
		}
	}
	if len(cfg.Accounts) == 0 {
		output.RenderNoAccounts(exePath)
		return
	}

	// Check accounts concurrently — much faster, and a slow account
	// never blocks a fast one.
	results := make([]mail.Result, len(cfg.Accounts))
	var wg sync.WaitGroup
	for i, a := range cfg.Accounts {
		wg.Add(1)
		go func(i int, a config.Account) {
			defer wg.Done()
			defer func() {
				if r := recover(); r != nil {
					results[i] = mail.Result{Email: a.Email, Err: fmt.Errorf("panic: %v", r)}
				}
			}()
			pwd, err := secrets.Resolve(a)
			if err != nil {
				results[i] = mail.Result{Email: a.Email, Err: err}
				return
			}
			results[i] = mail.Check(ctx, a.Email, pwd)
		}(i, a)
	}
	wg.Wait()

	output.Render(os.Stdout, results, exePath)
}

// autoLoadLegacy reads accounts out of a legacy .creds_gmail (if found)
// into cfg as in-memory plaintext entries. It does NOT save to disk —
// the setup wizard handles the actual migration with user confirmation.
func autoLoadLegacy(cfg *config.Config) (int, error) {
	paths, _ := config.LegacyPaths()
	for _, p := range paths {
		if p == "" {
			continue
		}
		f, err := os.Open(p)
		if err != nil {
			continue
		}
		accts, err := config.ParseLegacy(f)
		f.Close()
		if err != nil {
			return 0, err
		}
		for _, a := range accts {
			cfg.Upsert(a)
		}
		return len(accts), nil
	}
	return 0, nil
}
