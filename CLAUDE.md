# CLAUDE.md — loseit-cli (start here)

You're in **loseit-cli**: a small, **read-only** Go binary whose ONE job is **authenticate to Lose It! and extract the user's data**, emitting JSON on stdout. It does **no** storing, no business logic, no opinions about what the data means. The **consumer** — the `personal-workout-ai` project — deciphers the output and keeps only what it needs.

**Design rule: stay dumb.** One job: get the data out, intact. If you're tempted to add logic about *what the data means* or *where it should go*, that belongs in the consumer, not here.

**Runtime has no Claude.** The binary runs unattended — an agent or a cron invokes it, and Claude is NOT present. So anything that must happen at runtime — above all, recovering from an expired token — must live in the **Go code**, not in a Claude session. Claude is here only during a Claude Code session (like now) to build/fix/document. This file exists so the *next* session boots with full context.

## The one hard rule for changes

**Always run a live end-to-end test before opening a PR.** Mocks lie; Lose It's real endpoints are what matter. Build the binary, run `login` + `days --json` against the **real** API with **real** credentials, confirm real data returns. Only then push + open the PR. (`main` is PR-protected — see Guardrails.)

> ⚠️ **Privacy — the live test uses real credentials and real health data.** Keep them out of anything shareable: do **not** paste config.json, the token, login requests, or the export/`days` output into commits, PRs, issues, screenshots, chat logs, or CI logs; do **not** run the live test in shared/hosted CI. Use a **local** config.json (gitignored) or env vars that you clear afterward; redact email/token/nutrition before sharing any debug output. The export ZIP and `days` JSON are personal health data — treat them as sensitive. Prefer a throwaway/test Lose It account if you can.

## AUTH — read this fully (it is the fragile part)

### The flow (verified 2026-06-18)

1. **Login:** `POST https://api.loseit.com/account/login`, `Content-Type: application/x-www-form-urlencoded`, body `username=<email>&password=<pw>&grant_type=password`.
   - It's an OAuth **password-grant** endpoint. **The reCAPTCHA the web login form attaches (`captcha_token`/`captcha_site_key`) is NOT required by the API** — a plain POST authenticates. (Confirmed: bogus creds with *no* captcha return `{"error":"invalid_grant"}`, i.e. a credential error, not a captcha rejection.)
   - On success (HTTP 200) the response sets cookies: **`liauth`** and `fn_auth` (same value), plus `fn_authed=1`. `liauth` is an **ES384 JWT** — `Domain=loseit.com`, `HttpOnly`, **~14-day expiry** (`Max-Age=1209600`; JWT `exp`−`iat`).
2. **Extract:** `GET https://www.loseit.com/export/data` with `Cookie: liauth=<t>; fn_auth=<t>` → returns a **ZIP of CSVs** (the user's full export).
3. Parse the CSVs → emit JSON.

### Self-heal (in code — `internal/export/`)

- `Login()` (login.go) — does the POST, extracts `liauth` from `Set-Cookie`, returns it. `SaveToken()` writes it to `token_path` (0600).
- `FetchZip()` (export.go) — the **self-healing** fetch:
  1. No saved token? → `Login()` + save.
  2. Do the export GET.
  3. Response isn't a valid ZIP (`errExpired` — an expired cookie makes Lose It serve an HTML login page)? → `Login()` again and **retry once**.
  - So a stale/expired token fixes itself at runtime with no human — **provided `email`/`password` are configured.** With only a manually-saved cookie (no creds), it works until the cookie dies, then exits 2 telling the caller to `login` or use `--zip`.

### Config (`internal/config/`)

- `email` + `password` (or env `LOSEIT_EMAIL` / `LOSEIT_PASSWORD`) — the credentials. Live in `config.json`, which is **gitignored**. Never commit, never print.
- `token_path` (default `~/.config/loseit/token`; env `LOSEIT_TOKEN_PATH`) — where the liauth cookie is cached.
- **`login_url` / `export_url` are NOT config or env keys — by design.** They carry the credentials and session cookie, so to stop untrusted input (a hostile env var, or a `config.json` in the CWD) redirecting them to an attacker host, they're compiled-in constants only (`api.loseit.com/account/login`, `www.loseit.com/export/data`). `Login`/`fetchWithToken` also assert the URL is first-party (`*.loseit.com`, HTTPS) before sending anything. A Lose It endpoint move is a code change, not a runtime override. See `internal/config` security note + `assertFirstPartyURL` (export.go).
- `config show` redacts the password (emits `password_set` bool only) and shows the resolved `login_url`/`export_url` constants. `doctor` reports `tokenPresent` / `credentialsPresent` (no network).

### When auth breaks — debugging playbook (the failures the binary CANNOT self-heal)

The binary heals an *expired token*. It **cannot** heal Lose It *changing the login* (new required fields, enforced captcha, moved endpoint, renamed cookie). Symptoms: `login failed (HTTP …)`, or "logged in but the export still wasn't a ZIP."

1. `loseit-cli doctor` — confirm creds/token present and the URLs are right. Re-run with `-v` for stderr logs.
2. **Reproduce the real login** to see what changed (the method that worked here): log into loseit.com in a browser, capture the `POST api.loseit.com/account/login` request (DevTools → Network, or "Save all as HAR"), and compare its params + the response `Set-Cookie` against what `Login()` sends/expects.
3. Common breakages → fixes:
   - **Captcha now enforced** → headless login is no longer possible (a CLI can't mint a reCAPTCHA token). No clean fix — fall back to `--zip` (manual export) or a manually-grabbed cookie, and document the regression loudly.
   - **Cookie renamed / new domain** → update `loginCookie` (login.go) and the `Cookie` header in `fetchWithToken` (export.go).
   - **Endpoint moved / params changed** → update the `DefaultLoginURL` / `DefaultExportURL` constant (config.go) + the form fields in `Login()`, and rebuild. (There is no runtime URL override by design — see Config above. If the new host is still under `loseit.com`, `assertFirstPartyURL` already allows it; a move off `loseit.com` also needs that allowlist updated.)
4. Re-run the **live test** before PRing the fix.

## What it extracts today vs. the full export

`days` surfaces **nutrition only** — from `food-logs.csv` + `daily-calorie-summary.csv`. But the export ZIP contains far more: `weights.csv` (bodyweight), `exercise-logs.csv`, `steps.csv`, `fasting-logs.csv`, `profile.csv`, per-nutrient series (`protein.csv`/`fat.csv`/`carbohydrates.csv`/`daily-values.csv`), `notes.csv`, `recipes.csv`, custom foods/exercises, and food/progress photos. Per the **extract-all, consumer-deciphers** design, surfacing more of these (e.g. a generic dump, or per-domain commands) is the natural next step — keep it a *separate* PR so auth stays clean.

## Code map

- `internal/export/login.go` — `Login`, `SaveToken`, `loginAndSave`, `loginCookie`.
- `internal/export/export.go` — `FetchZip` (auto-login + retry), `fetchWithToken`, `errExpired`, `ReadToken`, `LoadZipBytes`, ZIP/CSV reading.
- `internal/nutrition/` — per-day aggregation = the `days --json` contract (banker's rounding; see AGENTS.md).
- `internal/cli/` — cobra commands: `days`, `login`, `doctor`, `config`, `version`, `completion`; `root.go`, `exit.go`, `output.go`.
- `internal/config/` — discovery + precedence (flag > env > config.json > defaults).

## Build / test

- Go is **not** on PATH on this machine; prepend `C:\Program Files\Go\bin` (or `C:\Users\gates\go-sdk\go\bin`). Lint/format helpers (gofumpt, golangci-lint, goreleaser) are in `C:\Users\gates\go\bin`.
- `make check` — or run directly: `go build ./...`, `go vet ./...`, `gofumpt -l .` (empty = clean), `golangci-lint run`, `go test -race ./...`.
- **Then the live end-to-end test. Then the PR.**

## Guardrails

- `main` is **PR-protected**. **Never push to main, merge, tag, or release without the owner's explicit OK** (GOAL.md §13). Work on a branch and open a PR he merges.
- Never commit `config.json`, the token file, or `*.zip` exports. Never print the `liauth` cookie or the password.

## Pointers

- `AGENTS.md` — machine contract (command shapes, JSON, exit codes, auth model).
- `README.md` — user-facing setup. `GOAL.md` — original rewrite spec + guardrails. `RELEASING.md` — release pipeline. `SKILL.md` — ClawHub skill.
- `.claude/CLAWHUB_STANDARDS.md` — **read before touching auth, config, file I/O, network, or docs.** The rules that keep the ClawHub security scan green (chief one: never call the tool "writes no files" — it caches a `0600` session token; always qualify "read-only" as *against your Lose It account*). Each rule is pinned by an immutable regression test.
