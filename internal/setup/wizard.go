// Package setup runs the interactive first-run / reconfigure wizard.
//
// The wizard is invoked as `gmailnotifier setup`. It:
//   - Detects 1Password CLI (`op`) availability and offers integration.
//   - Detects a legacy `.creds_gmail` file and offers one-shot migration.
//   - Detects the SwiftBar plugin folder and offers to install the binary there.
//   - Prompts for each new account's email + password via native macOS dialogs.
//   - Tests the IMAP connection before saving.
//   - Stores the password in the macOS Keychain.
package setup

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/bricklen/gmailnotifier/internal/config"
	"github.com/bricklen/gmailnotifier/internal/mail"
	"github.com/bricklen/gmailnotifier/internal/secrets"
)

// Run is the main entry point.
func Run(ctx context.Context) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	if err := maybeMigrateLegacy(ctx, cfg); err != nil {
		return err
	}
	if err := maybeInstallToSwiftBar(); err != nil {
		// Non-fatal — user can still run the binary directly.
		_ = Notify("gmailnotifier", "Could not auto-install to SwiftBar: "+err.Error())
	}

	for {
		action, err := mainMenu(cfg)
		if err != nil {
			if err == ErrDialogCancelled {
				return nil
			}
			return err
		}
		switch action {
		case actionAdd:
			if err := addAccount(ctx, cfg); err != nil && err != ErrDialogCancelled {
				_ = Notify("gmailnotifier", "Add account failed: "+err.Error())
			}
		case actionRemove:
			if err := removeAccount(cfg); err != nil && err != ErrDialogCancelled {
				_ = Notify("gmailnotifier", "Remove account failed: "+err.Error())
			}
		case actionTest:
			if err := testAccounts(ctx, cfg); err != nil && err != ErrDialogCancelled {
				_ = Notify("gmailnotifier", "Test failed: "+err.Error())
			}
		case actionDone:
			return nil
		}
	}
}

const (
	actionAdd    = "Add a Gmail account"
	actionRemove = "Remove an account"
	actionTest   = "Test all accounts"
	actionDone   = "Done"
)

func mainMenu(cfg *config.Config) (string, error) {
	var summary strings.Builder
	if len(cfg.Accounts) == 0 {
		summary.WriteString("No accounts configured yet.\n\n")
	} else {
		summary.WriteString("Configured accounts:\n")
		for _, a := range cfg.Accounts {
			summary.WriteString(fmt.Sprintf("  • %s  [%s]\n", a.Email, a.Source))
		}
		summary.WriteString("\n")
	}
	summary.WriteString("What would you like to do?")
	return Choose("gmailnotifier setup", summary.String(),
		[]string{actionAdd, actionRemove, actionTest, actionDone})
}

func addAccount(ctx context.Context, cfg *config.Config) error {
	email, err := AskString("Add Gmail account",
		"Enter the Gmail address to monitor (e.g. you@gmail.com):", "")
	if err != nil {
		return err
	}
	email = strings.TrimSpace(email)
	if !looksLikeEmail(email) {
		return fmt.Errorf("%q does not look like an email address", email)
	}

	// Backend choice.
	options := []string{"Store in macOS Keychain (recommended)"}
	if secrets.OPAvailable() {
		options = append(options, "Use 1Password (op CLI detected)")
	}
	choice, err := Choose("Password storage",
		"How should gmailnotifier obtain the App Password for "+email+"?",
		options)
	if err != nil {
		return err
	}

	acct := config.Account{Email: email}
	var password string

	if strings.HasPrefix(choice, "Use 1Password") {
		ref, err := AskString("1Password reference",
			"Enter the 1Password secret reference for "+email+":\n"+
				"(Format: op://VAULT/ITEM/FIELD, e.g. op://Personal/Gmail/password)",
			"op://Personal/Gmail/password")
		if err != nil {
			return err
		}
		acct.Source = config.SourceOnePassword
		acct.Reference = strings.TrimSpace(ref)
		// Resolve once to verify the reference is valid + accessible.
		p, err := secrets.Resolve(acct)
		if err != nil {
			return fmt.Errorf("could not read from 1Password: %w", err)
		}
		password = p
	} else {
		acct.Source = config.SourceKeychain
		p, err := AskPassword("Gmail App Password",
			"Enter the Gmail App Password for "+email+".\n\n"+
				"This is NOT your regular Google password. Generate one at:\n"+
				"https://myaccount.google.com/apppasswords")
		if err != nil {
			return err
		}
		password = p
	}

	if password == "" {
		return fmt.Errorf("empty password")
	}

	// Test before saving.
	res := mail.Check(ctx, acct.Email, password)
	if res.Err != nil {
		ok, _ := Confirm("Connection test failed",
			fmt.Sprintf("IMAP login failed:\n%s\n\nSave the account anyway?", res.Err))
		if !ok {
			return res.Err
		}
	}

	// Persist.
	if acct.Source == config.SourceKeychain {
		if err := secrets.Store(acct, password); err != nil {
			return fmt.Errorf("store in keychain: %w", err)
		}
	}
	cfg.Upsert(acct)
	if err := cfg.Save(); err != nil {
		return fmt.Errorf("save config: %w", err)
	}

	msg := fmt.Sprintf("Saved %s.", acct.Email)
	if res.Err == nil {
		msg += fmt.Sprintf("\nIMAP connection OK (%d unread).", res.UnreadCount)
	}
	return Notify("gmailnotifier", msg)
}

func removeAccount(cfg *config.Config) error {
	if len(cfg.Accounts) == 0 {
		return Notify("gmailnotifier", "No accounts to remove.")
	}
	items := make([]string, 0, len(cfg.Accounts))
	for _, a := range cfg.Accounts {
		items = append(items, a.Email)
	}
	pick, err := Choose("Remove account", "Choose an account to remove:", items)
	if err != nil {
		return err
	}
	ok, err := Confirm("Confirm",
		"Remove "+pick+"?\n\nThis will also delete the Keychain entry (if any).")
	if err != nil || !ok {
		return err
	}
	cfg.Remove(pick)
	if err := cfg.Save(); err != nil {
		return err
	}
	_ = secrets.Delete(pick)
	return Notify("gmailnotifier", "Removed "+pick+".")
}

func testAccounts(ctx context.Context, cfg *config.Config) error {
	if len(cfg.Accounts) == 0 {
		return Notify("gmailnotifier", "No accounts configured.")
	}
	var b strings.Builder
	for _, a := range cfg.Accounts {
		pwd, err := secrets.Resolve(a)
		if err != nil {
			fmt.Fprintf(&b, "✗ %s — %s\n", a.Email, err)
			continue
		}
		res := mail.Check(ctx, a.Email, pwd)
		if res.Err != nil {
			fmt.Fprintf(&b, "✗ %s — %s\n", a.Email, res.Err)
		} else {
			fmt.Fprintf(&b, "✓ %s — %d unread\n", a.Email, res.UnreadCount)
		}
	}
	return Notify("Connection test results", b.String())
}

// maybeMigrateLegacy looks for a .creds_gmail file and offers to migrate
// its contents into the Keychain in one click.
func maybeMigrateLegacy(ctx context.Context, cfg *config.Config) error {
	paths, _ := config.LegacyPaths()
	var found string
	for _, p := range paths {
		if p == "" {
			continue
		}
		if _, err := os.Stat(p); err == nil {
			found = p
			break
		}
	}
	if found == "" {
		return nil
	}
	ok, err := Confirm("Migrate legacy credentials",
		fmt.Sprintf("Found a legacy plaintext credentials file at:\n%s\n\nMigrate these accounts into the macOS Keychain and delete the plaintext file?", found))
	if err != nil || !ok {
		return nil
	}
	f, err := os.Open(found)
	if err != nil {
		return fmt.Errorf("open legacy creds: %w", err)
	}
	defer f.Close()
	accts, err := config.ParseLegacy(f)
	if err != nil {
		return fmt.Errorf("parse legacy creds: %w", err)
	}
	migrated := 0
	for _, a := range accts {
		ka := config.Account{Email: a.Email, Source: config.SourceKeychain}
		if err := secrets.Store(ka, a.Password); err != nil {
			_ = Notify("Migration warning",
				fmt.Sprintf("Could not store %s in Keychain: %s", a.Email, err))
			continue
		}
		cfg.Upsert(ka)
		migrated++
	}
	if err := cfg.Save(); err != nil {
		return fmt.Errorf("save migrated config: %w", err)
	}
	// Overwrite the legacy file with zeros before unlinking, so the
	// disk blocks don't sit around with cleartext.
	_ = wipeAndRemove(found)
	return Notify("Migration complete",
		fmt.Sprintf("Migrated %d account(s) into the Keychain and removed:\n%s", migrated, found))
}

func wipeAndRemove(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		return err
	}
	if info.Mode().IsRegular() && info.Size() > 0 {
		zero := make([]byte, info.Size())
		_ = os.WriteFile(path, zero, info.Mode().Perm())
	}
	return os.Remove(path)
}

// maybeInstallToSwiftBar offers to install this binary into the user's
// SwiftBar plugin folder so SwiftBar runs it. It also sweeps any prior
// gmailnotifier* files in that folder so we never end up with two icons
// in the menu bar.
//
// If the running binary is already inside the plugin folder (i.e. the
// user installed via `make install` and ran setup from there), the
// install step is a no-op.
func maybeInstallToSwiftBar() error {
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	exe, _ = filepath.Abs(exe)

	pluginDir := swiftBarPluginDir()
	if pluginDir == "" {
		// SwiftBar prefs not set; nothing to do silently.
		return nil
	}

	// If we're already running from inside the plugin folder, do nothing
	// — installing again would just delete ourselves mid-run.
	if filepath.Dir(exe) == pluginDir {
		return nil
	}

	target := filepath.Join(pluginDir, "gmailnotifier.30s")
	existing := findExistingPluginFiles(pluginDir)

	// If the only thing there is a symlink that already points at us,
	// nothing to do.
	if len(existing) == 1 && existing[0] == target {
		if linkTarget, err := os.Readlink(target); err == nil && linkTarget == exe {
			return nil
		}
	}

	prompt := fmt.Sprintf("Install the gmailnotifier plugin into SwiftBar?\n\nSource:  %s\nTarget:  %s", exe, target)
	if len(existing) > 0 {
		prompt += "\n\nThis will replace:"
		for _, p := range existing {
			prompt += "\n  • " + p
		}
	}
	ok, err := Confirm("Install plugin", prompt)
	if err != nil || !ok {
		return nil
	}
	if err := os.MkdirAll(pluginDir, 0o755); err != nil {
		return err
	}
	// Remove any prior installations so SwiftBar can never end up running
	// two copies of the plugin.
	for _, p := range existing {
		if err := os.Remove(p); err != nil {
			return fmt.Errorf("remove %s: %w", p, err)
		}
	}
	// Symlink so future rebuilds are picked up automatically.
	if err := os.Symlink(exe, target); err != nil {
		return fmt.Errorf("symlink: %w", err)
	}
	// Nudge SwiftBar to rescan.
	_ = exec.Command("/usr/bin/open", "-g", "swiftbar://refreshallplugins").Start()
	return Notify("Installed",
		"gmailnotifier is now active in SwiftBar.\n\nClick the menu bar icon to see your accounts.")
}

// findExistingPluginFiles returns the absolute paths of any gmailnotifier*
// regular files or symlinks at the top level of pluginDir. Subdirectories
// are not traversed.
func findExistingPluginFiles(pluginDir string) []string {
	entries, err := os.ReadDir(pluginDir)
	if err != nil {
		return nil
	}
	var out []string
	for _, e := range entries {
		name := e.Name()
		if !strings.HasPrefix(name, "gmailnotifier") {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		// Skip directories (e.g. the empty .cgo folder some users have
		// leftover from very old installs).
		if info.IsDir() {
			continue
		}
		out = append(out, filepath.Join(pluginDir, name))
	}
	return out
}

// swiftBarPluginDir returns the user's configured plugin directory by
// reading SwiftBar's preferences via `defaults read`. Returns "" if
// SwiftBar isn't configured.
func swiftBarPluginDir() string {
	out, err := exec.Command("/usr/bin/defaults", "read", "com.ameba.SwiftBar", "PluginDirectory").Output()
	if err == nil {
		dir := strings.TrimSpace(string(out))
		if dir != "" {
			return os.ExpandEnv(strings.ReplaceAll(dir, "~", os.Getenv("HOME")))
		}
	}
	// Fall back to the documented default.
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	candidate := filepath.Join(home, "Library", "Application Support", "SwiftBar", "Plugins")
	if st, err := os.Stat(candidate); err == nil && st.IsDir() {
		return candidate
	}
	return ""
}

func looksLikeEmail(s string) bool {
	at := strings.IndexByte(s, '@')
	if at <= 0 || at == len(s)-1 {
		return false
	}
	if strings.ContainsAny(s, " \t\r\n") {
		return false
	}
	if !strings.Contains(s[at+1:], ".") {
		return false
	}
	return true
}
