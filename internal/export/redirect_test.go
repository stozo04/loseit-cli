package export

import (
	"context"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/stozo04/loseit-cli/internal/config"
)

func credentialSink(t *testing.T) (*httptest.Server, *atomic.Int32) {
	t.Helper()
	var hits atomic.Int32
	sink := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		hits.Add(1)
		http.SetCookie(w, &http.Cookie{Name: loginCookie, Value: goodToken})
	}))
	t.Cleanup(sink.Close)
	return sink, &hits
}

func redirectingServer(t *testing.T, status int, location string) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, location, status)
	}))
	t.Cleanup(srv.Close)
	return srv
}

func TestLoginNeverResendsCredentialsOnRedirect(t *testing.T) {
	for _, status := range []int{http.StatusTemporaryRedirect, http.StatusPermanentRedirect, http.StatusFound} {
		sink, hits := credentialSink(t)
		srv := redirectingServer(t, status, sink.URL+"/collect")
		cfg := testCfg(srv, filepath.Join(t.TempDir(), "token"), true)

		_, err := Login(context.Background(), cfg)
		if err == nil {
			t.Fatalf("HTTP %d: expected Login to fail on a redirect", status)
		}
		if !strings.Contains(err.Error(), "redirect") {
			t.Errorf("HTTP %d: error should say the login endpoint redirected: %v", status, err)
		}
		if got := hits.Load(); got != 0 {
			t.Errorf("HTTP %d: redirect target received %d request(s); credentials must never be resent", status, got)
		}
	}
}

func TestFetchZipRefusesRedirectsThatLeaveLoseItOverHTTPS(t *testing.T) {
	cases := map[string]struct {
		location string
		want     string
	}{
		"other host":     {"https://evil.example/collect", "non-lose it"},
		"http downgrade": {"http://www.loseit.com/export/data", "non-https"},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Setenv(config.EnvToken, "saved-cookie")
			srv := redirectingServer(t, http.StatusFound, tc.location)
			cfg := &config.Config{
				ExportURL: srv.URL + "/export/data",
				LoginURL:  config.DefaultLoginURL,
			}
			_, err := FetchZip(context.Background(), cfg)
			if err == nil {
				t.Fatal("expected FetchZip to refuse the redirect")
			}
			if !strings.Contains(strings.ToLower(err.Error()), tc.want) {
				t.Errorf("error should name the refused redirect (%q): %v", tc.want, err)
			}
		})
	}
}
