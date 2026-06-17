package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stozo04/loseit-cli/internal/config"
)

// run executes the root command with args, capturing stdout and stderr.
func run(t *testing.T, args ...string) (stdout, stderr string, err error) {
	t.Helper()
	root := NewRootCmd()
	var out, errBuf bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&errBuf)
	root.SetArgs(args)
	err = root.Execute()
	return out.String(), errBuf.String(), err
}

// writeConfig writes a config.json into a temp dir and returns its path.
func writeConfig(t *testing.T, cfg map[string]any) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	b, _ := json.Marshal(cfg)
	if err := os.WriteFile(path, b, 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

// normalizeLF strips carriage returns so Windows checkouts compare byte-for-byte
// against the LF goldens.
func normalizeLF(b []byte) []byte {
	return bytes.ReplaceAll(b, []byte("\r"), nil)
}

func assertGolden(t *testing.T, name string, got []byte) {
	t.Helper()
	path := filepath.Join("..", "..", "testdata", "golden", name)
	// UPDATE_GOLDEN=1 rewrites the golden from the current output (goldens are
	// stored LF).
	if os.Getenv("UPDATE_GOLDEN") != "" {
		if err := os.WriteFile(path, normalizeLF(got), 0o644); err != nil {
			t.Fatalf("update golden %s: %v", name, err)
		}
		return
	}
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read golden %s: %v", name, err)
	}
	g, w := normalizeLF(got), normalizeLF(want)
	if !bytes.Equal(g, w) {
		t.Errorf("output does not match golden %s\n--- got (%d bytes) ---\n%s\n--- want (%d bytes) ---\n%s",
			name, len(g), g, len(w), w)
	}
}

func TestVersionJSON(t *testing.T) {
	stdout, _, err := run(t, "version", "--json")
	if err != nil {
		t.Fatalf("version: %v", err)
	}
	var info map[string]any
	if err := json.Unmarshal([]byte(stdout), &info); err != nil {
		t.Fatalf("version --json not valid JSON: %v\n%s", err, stdout)
	}
	for _, k := range []string{"version", "commit", "date", "go"} {
		if _, ok := info[k]; !ok {
			t.Errorf("version JSON missing key %q", k)
		}
	}
}

func TestConfigShowJSONDefaults(t *testing.T) {
	cfgPath := writeConfig(t, map[string]any{}) // empty file → defaults apply.
	stdout, _, err := run(t, "--config", cfgPath, "config", "show", "--json")
	if err != nil {
		t.Fatalf("config show: %v", err)
	}
	var view map[string]any
	if err := json.Unmarshal([]byte(stdout), &view); err != nil {
		t.Fatalf("config show --json invalid: %v", err)
	}
	if view["export_url"] != config.DefaultExportURL {
		t.Errorf("export_url = %v, want %s", view["export_url"], config.DefaultExportURL)
	}
	if view["token_path"] != config.DefaultTokenPath {
		t.Errorf("token_path = %v, want %s", view["token_path"], config.DefaultTokenPath)
	}
}

func TestDoctorNoTokenExitsZeroWithHint(t *testing.T) {
	t.Setenv(config.EnvToken, "") // ensure no env token.
	missing := filepath.Join(t.TempDir(), "absent-token")
	cfgPath := writeConfig(t, map[string]any{"token_path": missing})

	stdout, stderr, err := run(t, "--config", cfgPath, "doctor")
	if err != nil {
		t.Fatalf("doctor should exit 0 without a token, got: %v", err)
	}
	var report map[string]any
	if err := json.Unmarshal([]byte(stdout), &report); err != nil {
		t.Fatalf("doctor JSON invalid: %v\n%s", err, stdout)
	}
	if report["tokenPresent"] != false {
		t.Errorf("tokenPresent = %v, want false", report["tokenPresent"])
	}
	if !strings.Contains(stderr, "No Lose It token") {
		t.Errorf("stderr missing hint: %q", stderr)
	}
}

func TestDoctorWithTokenPresent(t *testing.T) {
	t.Setenv(config.EnvToken, "liauth-cookie")
	cfgPath := writeConfig(t, map[string]any{})
	stdout, _, err := run(t, "--config", cfgPath, "doctor")
	if err != nil {
		t.Fatalf("doctor: %v", err)
	}
	var report map[string]any
	if err := json.Unmarshal([]byte(stdout), &report); err != nil {
		t.Fatal(err)
	}
	if report["tokenPresent"] != true {
		t.Errorf("tokenPresent = %v, want true", report["tokenPresent"])
	}
}

func TestCompletionBash(t *testing.T) {
	stdout, _, err := run(t, "completion", "bash")
	if err != nil {
		t.Fatalf("completion bash: %v", err)
	}
	if !strings.Contains(stdout, "loseit-cli") {
		t.Error("bash completion output does not mention loseit-cli")
	}
}
