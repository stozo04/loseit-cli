package export

import (
	"context"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/stozo04/loseit-cli/internal/config"
)

// loginCookie is the session cookie the login response sets; it is what the
// export endpoint authenticates against. (Lose It sets liauth and fn_auth to the
// same JWT value; we carry liauth and send both — see fetchWithToken.)
const loginCookie = "liauth"

// Login authenticates against Lose It's first-party password-grant endpoint and
// returns the liauth session-cookie value. The request is a plain form POST of
// username/password/grant_type=password; the reCAPTCHA token the web form
// attaches is NOT required by the API (verified 2026-06-18). Any failure is an
// auth/export error (exit 2). The credential values are never logged.
func Login(ctx context.Context, cfg *config.Config) (string, error) {
	if cfg.Email == "" || cfg.Password == "" {
		return "", newErr(
			"No Lose It credentials. Set %s/%s, or add \"email\"/\"password\" to config.json.",
			config.EnvEmail, config.EnvPassword,
		)
	}

	form := url.Values{
		"username":   {cfg.Email},
		"password":   {cfg.Password},
		"grant_type": {"password"},
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, cfg.LoginURL,
		strings.NewReader(form.Encode()))
	if err != nil {
		return "", newErr("building login request: %v", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("User-Agent", userAgent)

	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", newErr("login request failed: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		// Bad credentials come back as a non-200 (the API returns 404/invalid_grant
		// for an unknown account); don't echo the body, which may carry detail.
		return "", newErr("login failed (HTTP %d) — check your Lose It email and password.", resp.StatusCode)
	}

	for _, c := range resp.Cookies() {
		if c.Name == loginCookie && c.Value != "" {
			return c.Value, nil
		}
	}
	return "", newErr("login succeeded but no %s cookie was returned — Lose It may have changed its login.", loginCookie)
}

// loginAndSave logs in and persists the resulting token to token_path, returning
// the token. Used by FetchZip for auto-login and by the `login` command.
func loginAndSave(ctx context.Context, cfg *config.Config) (string, error) {
	token, err := Login(ctx, cfg)
	if err != nil {
		return "", err
	}
	if err := SaveToken(cfg, token); err != nil {
		return "", err
	}
	return token, nil
}

// SaveToken writes the session token to the configured token_path (creating
// parent dirs), 0600 because it is a credential. A trailing newline matches the
// manual "echo cookie > token" convention.
func SaveToken(cfg *config.Config, token string) error {
	path := expandUser(cfg.TokenPath)
	if dir := filepath.Dir(path); dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0o700); err != nil {
			return newErr("creating token dir %s: %v", dir, err)
		}
	}
	if err := os.WriteFile(path, []byte(token+"\n"), 0o600); err != nil {
		return newErr("writing token to %s: %v", path, err)
	}
	return nil
}
