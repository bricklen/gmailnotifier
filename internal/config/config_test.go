package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func withTempConfigDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	// On darwin os.UserConfigDir returns ~/Library/Application Support and
	// ignores XDG_CONFIG_HOME, so also redirect HOME.
	t.Setenv("HOME", dir)
	return dir
}

func TestSaveLoadRoundtrip(t *testing.T) {
	withTempConfigDir(t)

	c := &Config{Accounts: []Account{
		{Email: "a@gmail.com", Source: SourceKeychain},
		{Email: "b@acme.com", Source: SourceOnePassword, Reference: "op://Personal/Gmail b/password"},
	}}
	if err := c.Save(); err != nil {
		t.Fatalf("Save: %v", err)
	}

	// File should be 0600.
	p, err := Path()
	if err != nil {
		t.Fatalf("Path: %v", err)
	}
	st, err := os.Stat(p)
	if err != nil {
		t.Fatalf("Stat: %v", err)
	}
	if mode := st.Mode().Perm(); mode != 0o600 {
		t.Errorf("config file perm = %o, want 0600", mode)
	}

	got, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(got.Accounts) != 2 {
		t.Fatalf("got %d accounts, want 2", len(got.Accounts))
	}
	if got.Accounts[0].Email != "a@gmail.com" || got.Accounts[0].Source != SourceKeychain {
		t.Errorf("account 0 mismatch: %+v", got.Accounts[0])
	}
	if got.Accounts[1].Reference != "op://Personal/Gmail b/password" {
		t.Errorf("account 1 reference mismatch: %q", got.Accounts[1].Reference)
	}
}

func TestSaveStripsInMemoryPassword(t *testing.T) {
	withTempConfigDir(t)
	c := &Config{Accounts: []Account{
		{Email: "a@gmail.com", Source: SourceKeychain, Password: "should-not-persist"},
	}}
	if err := c.Save(); err != nil {
		t.Fatalf("Save: %v", err)
	}
	p, _ := Path()
	data, _ := os.ReadFile(p)
	if strings.Contains(string(data), "should-not-persist") {
		t.Errorf("plaintext password leaked to disk:\n%s", data)
	}
}

func TestLoadMissingFile(t *testing.T) {
	withTempConfigDir(t)
	c, err := Load()
	if err != nil {
		t.Fatalf("Load on missing file: %v", err)
	}
	if len(c.Accounts) != 0 {
		t.Errorf("expected empty config, got %+v", c)
	}
}

func TestValidate(t *testing.T) {
	cases := []struct {
		name    string
		c       Config
		wantErr string
	}{
		{"missing email", Config{Accounts: []Account{{Source: SourceKeychain}}}, "email is required"},
		{"missing source", Config{Accounts: []Account{{Email: "a@b.com"}}}, "source is required"},
		{"unknown source", Config{Accounts: []Account{{Email: "a@b.com", Source: "wat"}}}, "unknown source"},
		{"1password without ref", Config{Accounts: []Account{{Email: "a@b.com", Source: SourceOnePassword}}}, "requires a reference"},
		{"duplicate email", Config{Accounts: []Account{
			{Email: "a@b.com", Source: SourceKeychain},
			{Email: "a@b.com", Source: SourceKeychain},
		}}, "duplicate email"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.c.Validate()
			if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
				t.Errorf("Validate() err = %v, want substring %q", err, tc.wantErr)
			}
		})
	}
}

func TestUpsertAndRemove(t *testing.T) {
	c := &Config{}
	c.Upsert(Account{Email: "a@x.com", Source: SourceKeychain})
	c.Upsert(Account{Email: "b@x.com", Source: SourceKeychain})
	if len(c.Accounts) != 2 {
		t.Fatalf("len=%d", len(c.Accounts))
	}
	// Replace existing.
	c.Upsert(Account{Email: "a@x.com", Source: SourceOnePassword, Reference: "op://x/y/z"})
	if a := c.Find("a@x.com"); a == nil || a.Source != SourceOnePassword {
		t.Errorf("upsert did not replace: %+v", a)
	}
	if !c.Remove("a@x.com") {
		t.Error("Remove returned false for existing email")
	}
	if c.Find("a@x.com") != nil {
		t.Error("Find returned non-nil after Remove")
	}
	if c.Remove("nope@x.com") {
		t.Error("Remove returned true for missing email")
	}
}

func TestParseLegacy(t *testing.T) {
	input := `# comment
a@gmail.com|password1

b@gmail.com|pass|word|with|pipes
c@gsuite.example|c-pass
`
	got, err := ParseLegacy(strings.NewReader(input))
	if err != nil {
		t.Fatalf("ParseLegacy: %v", err)
	}
	if len(got) != 3 {
		t.Fatalf("got %d accounts, want 3", len(got))
	}
	if got[1].Password != "pass|word|with|pipes" {
		t.Errorf("legacy parser dropped pipes in password: %q", got[1].Password)
	}
	for _, a := range got {
		if a.Source != SourcePlaintext {
			t.Errorf("%s: source = %s, want plaintext", a.Email, a.Source)
		}
	}
}

func TestParseLegacyMalformed(t *testing.T) {
	for _, line := range []string{
		"no-pipe-at-all",
		"|missing-email",
		"missing-pass|",
	} {
		if _, err := ParseLegacy(strings.NewReader(line)); err == nil {
			t.Errorf("expected error for %q", line)
		}
	}
}

// TestDirPermissions ensures the config directory is created as 0700.
func TestDirPermissions(t *testing.T) {
	tmp := withTempConfigDir(t)
	dir, err := Dir()
	if err != nil {
		t.Fatalf("Dir: %v", err)
	}
	if !strings.HasPrefix(dir, tmp) {
		t.Logf("note: config dir %q not inside temp %q (expected on darwin if HOME not honored)", dir, tmp)
	}
	st, err := os.Stat(dir)
	if err != nil {
		t.Fatalf("Stat: %v", err)
	}
	if !st.IsDir() {
		t.Fatalf("%s is not a directory", dir)
	}
	if mode := st.Mode().Perm(); mode != 0o700 {
		t.Errorf("dir perm = %o, want 0700", mode)
	}
	// Sanity: writing a file under it works.
	if err := os.WriteFile(filepath.Join(dir, "ok"), []byte("hi"), 0o600); err != nil {
		t.Errorf("write under config dir: %v", err)
	}
}
