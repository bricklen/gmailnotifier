// Package config loads and persists the gmailnotifier account list.
//
// Accounts are stored in TOML at ~/Library/Application Support/gmailnotifier/accounts.toml.
// The config file never contains a password — only a reference describing
// where to find one (Keychain, 1Password, or legacy plaintext).
package config

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/pelletier/go-toml/v2"
)

// Source identifies where an account's password lives.
type Source string

const (
	SourceKeychain    Source = "keychain"
	SourceOnePassword Source = "1password"
	SourcePlaintext   Source = "plaintext" // legacy migration only
)

// Account represents one Gmail account to check.
type Account struct {
	Email  string `toml:"email"`
	Source Source `toml:"source"`
	// Reference is meaningful only when Source == SourceOnePassword.
	// Example: "op://Personal/Gmail/password"
	Reference string `toml:"reference,omitempty"`
	// Password is only set transiently when migrating from legacy plaintext.
	// It is never serialized to disk by Save().
	Password string `toml:"-"`
}

// Config is the on-disk representation.
type Config struct {
	Accounts []Account `toml:"account"`
}

const (
	appDirName  = "gmailnotifier"
	configName  = "accounts.toml"
	legacyCreds = ".creds_gmail"
)

// Dir returns the per-user config directory, creating it if needed (0700).
func Dir() (string, error) {
	base, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("locate user config dir: %w", err)
	}
	dir := filepath.Join(base, appDirName)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", fmt.Errorf("create %s: %w", dir, err)
	}
	return dir, nil
}

// Path returns the absolute config file path.
func Path() (string, error) {
	dir, err := Dir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, configName), nil
}

// Load reads and parses the config file. Returns an empty Config and no error
// if the file does not yet exist.
func Load() (*Config, error) {
	p, err := Path()
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(p)
	if errors.Is(err, os.ErrNotExist) {
		return &Config{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", p, err)
	}
	var c Config
	if err := toml.Unmarshal(data, &c); err != nil {
		return nil, fmt.Errorf("parse %s: %w", p, err)
	}
	if err := c.Validate(); err != nil {
		return nil, err
	}
	return &c, nil
}

// Save writes the config atomically with 0600 perms (owner read/write only).
func (c *Config) Save() error {
	if err := c.Validate(); err != nil {
		return err
	}
	p, err := Path()
	if err != nil {
		return err
	}
	// Defensive copy that strips any in-memory Password before serializing.
	out := Config{Accounts: make([]Account, len(c.Accounts))}
	for i, a := range c.Accounts {
		a.Password = ""
		out.Accounts[i] = a
	}
	data, err := toml.Marshal(out)
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}
	dir := filepath.Dir(p)
	tmp, err := os.CreateTemp(dir, ".accounts.toml.*")
	if err != nil {
		return fmt.Errorf("create temp file: %w", err)
	}
	tmpPath := tmp.Name()
	// Best-effort cleanup if anything below fails.
	defer func() {
		_ = os.Remove(tmpPath)
	}()
	if err := tmp.Chmod(0o600); err != nil {
		tmp.Close()
		return fmt.Errorf("chmod temp file: %w", err)
	}
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return fmt.Errorf("write temp file: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close temp file: %w", err)
	}
	if err := os.Rename(tmpPath, p); err != nil {
		return fmt.Errorf("rename %s -> %s: %w", tmpPath, p, err)
	}
	return nil
}

// Validate checks for obvious config errors.
func (c *Config) Validate() error {
	seen := make(map[string]struct{}, len(c.Accounts))
	for i, a := range c.Accounts {
		if strings.TrimSpace(a.Email) == "" {
			return fmt.Errorf("account[%d]: email is required", i)
		}
		if _, dup := seen[a.Email]; dup {
			return fmt.Errorf("account[%d]: duplicate email %q", i, a.Email)
		}
		seen[a.Email] = struct{}{}
		switch a.Source {
		case SourceKeychain, SourcePlaintext:
			// no extra fields required
		case SourceOnePassword:
			if strings.TrimSpace(a.Reference) == "" {
				return fmt.Errorf("account[%d] (%s): 1password source requires a reference", i, a.Email)
			}
		case "":
			return fmt.Errorf("account[%d] (%s): source is required (keychain|1password|plaintext)", i, a.Email)
		default:
			return fmt.Errorf("account[%d] (%s): unknown source %q", i, a.Email, a.Source)
		}
	}
	return nil
}

// Find returns a pointer to the account with the given email, or nil.
func (c *Config) Find(email string) *Account {
	for i := range c.Accounts {
		if c.Accounts[i].Email == email {
			return &c.Accounts[i]
		}
	}
	return nil
}

// Upsert adds or replaces an account by email.
func (c *Config) Upsert(a Account) {
	for i := range c.Accounts {
		if c.Accounts[i].Email == a.Email {
			c.Accounts[i] = a
			return
		}
	}
	c.Accounts = append(c.Accounts, a)
}

// Remove deletes the account with the given email. Returns true if removed.
func (c *Config) Remove(email string) bool {
	for i := range c.Accounts {
		if c.Accounts[i].Email == email {
			c.Accounts = append(c.Accounts[:i], c.Accounts[i+1:]...)
			return true
		}
	}
	return false
}

// LegacyPaths returns candidate locations for the historical .creds_gmail
// file: alongside the running executable (the original layout) and in the
// new config directory. Callers should try each in order.
func LegacyPaths() ([]string, error) {
	var out []string
	exe, err := os.Executable()
	if err == nil {
		out = append(out, filepath.Join(filepath.Dir(exe), legacyCreds))
	}
	dir, err := Dir()
	if err != nil {
		return out, err
	}
	out = append(out, filepath.Join(dir, legacyCreds))
	return out, nil
}

// ParseLegacy reads a pipe-delimited "email|password" file and returns the
// accounts it contains. Lines are tolerated to contain pipes in the password
// (everything after the first pipe is the password).
func ParseLegacy(r io.Reader) ([]Account, error) {
	var accts []Account
	scanner := bufio.NewScanner(r)
	lineNo := 0
	for scanner.Scan() {
		lineNo++
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		idx := strings.IndexByte(line, '|')
		if idx <= 0 || idx == len(line)-1 {
			return nil, fmt.Errorf("legacy creds line %d: expected 'email|password'", lineNo)
		}
		email := strings.TrimSpace(line[:idx])
		pass := line[idx+1:]
		if email == "" || pass == "" {
			return nil, fmt.Errorf("legacy creds line %d: empty email or password", lineNo)
		}
		accts = append(accts, Account{
			Email:    email,
			Source:   SourcePlaintext,
			Password: pass,
		})
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("read legacy creds: %w", err)
	}
	return accts, nil
}
