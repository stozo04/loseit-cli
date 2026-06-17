---
name: loseit
description: >
  Read your Lose It! nutrition from any agent. Parses the Lose It data export and
  emits per-day nutrition as JSON — calories, protein, carbs, fat, fiber, a
  per-meal breakdown, and Lose It's own budget / under / exercise-adjustment
  figures. A self-contained, read-only extractor: it does NO writing — no daily
  log, no sync — the caller decides what to do with the data. Single static
  binary, no Python or other runtime. Two ways to supply the export: a downloaded
  ZIP (--zip, no token, the reliable headless path) or a cookie fetch using your
  liauth session cookie. NOTE: the liauth cookie expires with no auto-refresh, so
  hands-off use means dropping in a freshly downloaded export ZIP.
metadata:
  openclaw:
    emoji: 🥗
    homepage: https://github.com/stozo04/loseit-cli
    primaryEnv: LOSEIT_TOKEN
    permissions:
      network:
        - "Lose It! export endpoint (www.loseit.com/export/data, HTTPS) — download your own data export (read-only) using the liauth session cookie"
      files.read:
        - "config.json — optional settings (token_path, export_url), in the working directory or next to the binary"
        - "token file — the liauth session cookie (default ~/.config/loseit/token; overridable via LOSEIT_TOKEN_PATH)"
        - "--zip file — a downloaded Lose It export ZIP, when the --zip path is used"
      files.write:
        - "NONE — loseit-cli writes nothing. It only reads the export and prints JSON to stdout."
    requires:
      bins: []
      env: []
    envVars:
      - name: LOSEIT_TOKEN
        description: "The liauth session cookie value (cookie-fetch path). Optional — the --zip path needs no token."
        required: false
      - name: LOSEIT_CONFIG
        description: "Path to a config.json (alternative to discovery)."
        required: false
      - name: LOSEIT_TOKEN_PATH
        description: "Override the token file path (default ~/.config/loseit/token)."
        required: false
      - name: LOSEIT_EXPORT_URL
        description: "Override the export endpoint (default https://www.loseit.com/export/data)."
        required: false
---

# Lose It Nutrition — read-only nutrition extractor

Read your **Lose It!** nutrition from any agent. This is a read-only extractor: it obtains the
Lose It data export, parses the food log, and prints **per-day nutrition as JSON** — calories,
protein, carbs, fat, fiber, a per-meal breakdown, plus Lose It's own budget/under/exercise
figures. It does **no writing** — no daily log, no sync; you get the data and store whatever you
care about. Single static binary — **no Python or other runtime**.

> ⚠️ **Read-only.** It only reads your own Lose It export and prints JSON. It writes no files.

> ⏳ **The cookie wrinkle.** The `liauth` session cookie **expires with no auto-refresh**. For
> hands-off/agent use, the reliable path is **`--zip`** with a freshly downloaded export ZIP — not
> the cookie fetch. Don't chase a dead cookie; drop in a fresh export instead.

## Install

```bash
# A) Download a release for your OS/arch and put it on PATH:
#    https://github.com/stozo04/loseit-cli/releases
# B) Or with Go (1.24+):
go install github.com/stozo04/loseit-cli/cmd/loseit-cli@latest
```

## Two ways to supply the export

1. **Downloaded ZIP (no token — recommended for agents):** export your data from Lose It
   (Settings → Export), then:
   ```bash
   loseit-cli days --zip ~/Downloads/loseit-export.zip --json
   ```
2. **Cookie fetch:** save your `liauth` cookie (loseit.com → F12 → Application → Cookies) to
   `~/.config/loseit/token` (or set `LOSEIT_TOKEN`), then:
   ```bash
   loseit-cli days --json
   ```
   The cookie expires periodically with no refresh — re-grab it, or just use `--zip`.

## Commands

```bash
loseit-cli days --zip export.zip --days 7          # human table for the last 7 days
loseit-cli days --zip export.zip --json --days 7    # the frozen per-day JSON contract
loseit-cli days --zip export.zip --date 2026-06-16 --days 1
loseit-cli config show                              # resolved config
loseit-cli doctor                                   # config + token presence (no network)
loseit-cli version                                  # build metadata (also --version)
loseit-cli completion bash|zsh|fish|powershell      # shell completion
```

### `days --json` output

A JSON object keyed by ISO date → nutrition object (empty selection → `{}`):

```json
{
  "2026-06-16": {
    "source": "Lose It export",
    "calories_food": 505,
    "protein_g": 75,
    "carbs_g": 36,
    "fat_g": 6,
    "fiber_g": 3,
    "meals": [
      { "meal": "Breakfast", "calories": 225, "protein_g": 23, "carbs_g": 36, "fat_g": 0,
        "items": [ { "name": "Greek Yogurt", "qty": "1 cup", "calories": 120 } ] }
    ],
    "loseit_budget": 1663,
    "loseit_under": 1158,
    "exercise_adjustment": 120
  }
}
```

`loseit_budget` / `loseit_under` / `exercise_adjustment` appear only when the daily summary provides
them, after `meals`. Numbers are integers (banker's rounding). See **AGENTS.md** for the full
contract and exit codes.

## Conventions

- **stdout is parseable**; human hints and logs go to **stderr**, never interleaved.
- **Exit codes:** `0` success, `2` export/parse failure (no/expired token, bad ZIP), `64` usage
  error, `78` config error.
- **Read-only / writes nothing:** the tool only reads the export and prints JSON.
- **Secrets:** the token file and `config.json` are gitignored — never commit them, and the cookie
  value is never printed.
