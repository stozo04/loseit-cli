package config

import (
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestURLEndpointsAreNotOverridableByUntrustedInput pins the security guarantee
// that the credential/cookie-bearing login and export endpoints cannot be
// repointed by an attacker-influenceable environment variable or by a config.json
// dropped in the working directory. Both inputs are ignored; the compiled-in
// first-party defaults win. Reintroducing an env or config override for these
// URLs fails this test.
func TestURLEndpointsAreNotOverridableByUntrustedInput(t *testing.T) {
	const evilExport = "https://evil.example/export"
	const evilLogin = "https://evil.example/login"

	// A hostile environment...
	t.Setenv("LOSEIT_EXPORT_URL", evilExport)
	t.Setenv("LOSEIT_LOGIN_URL", evilLogin)

	// ...and a hostile config.json in the working directory.
	cfgPath := filepath.Join(t.TempDir(), "config.json")
	body := `{"export_url":"` + evilExport + `","login_url":"` + evilLogin + `"}`
	if err := os.WriteFile(cfgPath, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(Options{ConfigPath: cfgPath})
	if err != nil {
		t.Fatal(err)
	}
	if cfg.ExportURL != DefaultExportURL {
		t.Errorf("export URL overridden to %q; want first-party default %q", cfg.ExportURL, DefaultExportURL)
	}
	if cfg.LoginURL != DefaultLoginURL {
		t.Errorf("login URL overridden to %q; want first-party default %q", cfg.LoginURL, DefaultLoginURL)
	}
}

// TestDefaultEndpointsAreFirstPartyHTTPS pins that the compiled-in endpoints are
// HTTPS and within Lose It's own domain, so the constants themselves can never be
// quietly repointed off-domain.
func TestDefaultEndpointsAreFirstPartyHTTPS(t *testing.T) {
	for _, raw := range []string{DefaultLoginURL, DefaultExportURL} {
		u, err := url.Parse(raw)
		if err != nil {
			t.Fatalf("default endpoint %q invalid: %v", raw, err)
		}
		if u.Scheme != "https" {
			t.Errorf("default endpoint %q must be HTTPS", raw)
		}
		host := u.Hostname()
		if host != "loseit.com" && !strings.HasSuffix(host, ".loseit.com") {
			t.Errorf("default endpoint host %q must be within loseit.com", host)
		}
	}
}
