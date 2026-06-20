package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// repoRoot walks up from the test's working directory to the module root (the
// dir containing go.mod), so the doc assertions below don't depend on where the
// test binary is run from.
func repoRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("go.mod not found walking up from the test working dir")
		}
		dir = parent
	}
}

// TestUserDocsDoNotMisrepresentLocalWrites is a documentation regression guard
// for the ClawHub "intent-code divergence" findings (see
// .claude/CLAWHUB_STANDARDS.md §1). loseit-cli is read-only *against the user's
// Lose It account*, but it DOES write a session-token cache — a reusable ~14-day
// credential — to disk, and reads plaintext credentials from config.json. A
// blanket "writes no files" claim misrepresents that and was flagged by the
// scanner, so it must never reappear in the user-facing docs; and the disclosing
// fix (the token-cache mention + a Security section) must stay present.
func TestUserDocsDoNotMisrepresentLocalWrites(t *testing.T) {
	root := repoRoot(t)
	// Phrasings that falsely imply the tool persists nothing locally.
	banned := []string{"writes no files", "no files are written", "it writes nothing"}

	for _, name := range []string{"README.md", "SKILL.md"} {
		b, err := os.ReadFile(filepath.Join(root, name))
		if err != nil {
			t.Fatalf("read %s: %v", name, err)
		}
		text := string(b)
		lower := strings.ToLower(text)

		for _, phrase := range banned {
			if strings.Contains(lower, phrase) {
				t.Errorf("%s contains the misleading phrase %q — qualify it: read-only "+
					"against Lose It, but it writes a 0600 session-token cache", name, phrase)
			}
		}
		// The fix must remain: disclose the on-disk token credential...
		if !strings.Contains(lower, "token cache") {
			t.Errorf("%s must disclose the session-token cache it writes to disk", name)
		}
		// ...and keep a Security section warning the secrets are sensitive.
		if !strings.Contains(text, "Security & secrets") {
			t.Errorf("%s must keep a \"Security & secrets\" section covering config.json + the token file", name)
		}
	}
}

// TestDevDocsDoNotFrameScopeExpansion guards the ClawHub "Description-Behavior
// Mismatch" finding (see .claude/CLAWHUB_STANDARDS.md §6): loseit-cli reads only
// the two nutrition CSVs, so its docs must not frame collecting the rest of the
// export (bodyweight, profile, photos, …) as a goal or "natural next step". The
// nutrition-only / data-minimization framing must stay.
func TestDevDocsDoNotFrameScopeExpansion(t *testing.T) {
	root := repoRoot(t)
	b, err := os.ReadFile(filepath.Join(root, "CLAUDE.md"))
	if err != nil {
		t.Fatalf("read CLAUDE.md: %v", err)
	}
	lower := strings.ToLower(string(b))
	for _, banned := range []string{"natural next step", "extract-all"} {
		if strings.Contains(lower, banned) {
			t.Errorf("CLAUDE.md frames scope expansion (%q) — the tool is nutrition-only by design", banned)
		}
	}
	if !strings.Contains(lower, "data minimization") {
		t.Error("CLAUDE.md must keep the nutrition-only / data-minimization framing")
	}
}
