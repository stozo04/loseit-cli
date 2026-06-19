# ClawHub Inspection Standards

Rules for keeping `loseit-cli` clean through ClawHub's security inspection (the
scan that runs on publish — see `.github/workflows/publish-clawhub.yml`). ClawHub
grades the **skill** the way a reviewer grades an agent tool: it looks for
credential mishandling, privilege escalation, excessive permissions, and — the
category that has bitten us — **intent-code divergence**, where the prose
describes the tool as safer than the code actually is. Follow these rules so a
real change never trips a real finding, and so the few false positives are easy
to defend.

> **Golden rule:** a credential-collecting, token-caching health-data tool *will*
> draw scanner attention. Our job is to make every credential / file-write /
> network behavior **minimal, accurately documented, and provable with a test** —
> so any finding is either fixed or trivially shown to be a false positive.

---

## 1. "Read-only" must stay honest — the lesson that wrote this file

**Incident:** README/SKILL repeatedly described the tool as "read-only" and said
it "does **no writing** / **writes no files**" — while the same binary
authenticates with the user's credentials, **creates `~/.config/loseit/`**, and
**persists a reusable ~14-day `liauth` session token** to disk. ClawHub flagged
it **High / MCP Tool Poisoning** plus several **Intent-Code Divergence** findings:
the prominent "writes no files" claim can mislead a user or agent into granting
broader trust than warranted (e.g. running it where secret persistence is
disallowed, or failing to protect the token cache as a credential).

**The distinction we must always make:**

- loseit-cli **is** read-only **with respect to the user's Lose It account** — it
  never creates, changes, or deletes anything *in Lose It*, and it does no
  **application** storage (no daily log, no sync; the consumer stores nutrition).
- loseit-cli **is not** a no-local-writes tool. It **writes exactly one local
  file** — its session-token cache (`token_path`, default `~/.config/loseit/token`,
  mode `0600`) — and **reads** plaintext credentials from `config.json`.

**Rules:**

- **Never write "writes no files", "no files are written", or "it writes
  nothing"** in user-facing docs, code comments, or the SKILL frontmatter. Always
  qualify: *"read-only against your Lose It account; the one file it writes locally
  is its `0600` session-token cache."*
- **Disclose the token cache wherever you call the tool read-only.** README and
  SKILL each carry a **`Security & secrets`** section that names both secrets
  (`config.json`, the token file), their sensitivity, and how to protect them.
- **Keep the four descriptions consistent:** the SKILL `description`, the SKILL
  `permissions.files.write` block, the README/AGENTS prose, and the code comments
  (`main.go`, `internal/export/login.go`) must all tell the same story — token
  cache written, `0600`, credential. Divergence between any two of them is the
  exact finding.
- Pinned by `internal/cli/docs_security_test.go::TestUserDocsDoNotMisrepresentLocalWrites`.

## 2. Secrets on disk (least-privilege file permissions)

- Any file that can hold a credential — the **token file** and **`config.json`** —
  is owner-only. The token is written **`0600`** and its parent dir, if created,
  **`0700`**. See `internal/export/login.go::SaveToken` for the pattern:
  `os.OpenFile(..., 0o600)` **plus** a best-effort `f.Chmod(0o600)` to re-tighten a
  pre-existing file (`O_CREATE` only applies the mode to a *new* file; it will not
  re-restrict one that already exists with looser bits).
- Chmod failures on Windows are advisory — ignore them (best-effort), never fail
  the command over them.
- Secret files stay gitignored (`config.json`, `token`, `*.zip`) and are declared
  as such in `SKILL.md`. Never commit a real secret, even in `testdata/` or an
  example — `config.example.json` uses `you@example.com` placeholders only.
- The token is a **reusable session credential**, not a one-shot value. Say so, so
  users keep it off shared disks and out of backups.
- Pinned by `internal/export/login_test.go::TestSaveTokenWritesOwnerOnly` and
  `TestSaveTokenReTightensExistingFile`.

## 3. Never expose secrets in output or logs

- `config show` prints `password: <set>` (text) / `password_set: true` (JSON) —
  **never** the value; there is no `password` field in `configView`. `doctor`
  reports `tokenPresent` / `credentialsPresent` as booleans and **never** echoes
  the token value or password.
- No credential, token, `Cookie` header, or login form body may reach stdout,
  stderr, or a log line at any verbosity. `Login()` deliberately does **not** echo
  the response body on failure (it may carry detail).
- stdout is for parseable data (`--json`); human hints and logs go to stderr.
  Don't leak secrets across either.
- Pinned by `internal/cli/commands_test.go::TestConfigShowNeverRevealsPassword`
  and `TestDoctorNeverRevealsTokenOrPassword`.

## 4. Least privilege — permissions must match reality

- Keep the `SKILL.md` `metadata.openclaw.permissions` block **exactly** in sync
  with what the code does. Every declared `network`, `files.read`, and
  `files.write` entry must be real; the `files.write` entry **must** declare the
  token cache (under-declaring it is itself a finding — see §1). Remove anything
  the code no longer does; add anything new **before** publishing.
- Request the **narrowest** scope that works. No new file reads/writes, network
  hosts, env vars, or required binaries without updating `SKILL.md` and asking
  whether a narrower option exists.
- `requires.bins` stays `[]` — we ship a single static binary. Don't introduce a
  dependency on `sudo`, a shell, or an external tool, and never shell out to run a
  privileged command.

## 5. No code execution / injection surfaces; network stays first-party

- Don't pass user / config / env values into `exec.Command`, a shell, `eval`-like
  APIs, or template-to-code paths. We have **no `os/exec` usage** today — keep it
  that way unless there's a strong, reviewed reason.
- Treat the export ZIP and any fetched API data as **untrusted input**: validate,
  don't execute. CSV parsing tolerates ragged rows; it never interprets cell
  contents as code.
- All outbound calls are **HTTPS to the documented Lose It hosts**
  (`api.loseit.com/account/login`, `www.loseit.com/export/data`). The
  `LOSEIT_LOGIN_URL` / `LOSEIT_EXPORT_URL` overrides exist only for tests
  (httptest servers) and break-glass; they are not a feature to point at arbitrary
  hosts, and the defaults are baked in.

## 6. The write path lives in the consumer — keep it that way

- loseit-cli is the **dumb extractor**: get the data out, intact, to stdout. It has
  **no** knowledge of `DAILY_LOG.json`, no `sync`, no upsert. The nutrition-storage
  procedure (GOAL.md §11) belongs to **personal-workout-ai**, runs **in response to
  the user's own action**, and only sets the day's `nutrition` key. Never port a
  write path back into this repo.
- If a future feature surfaces more of the export (weights, exercise, steps…),
  keep it **read → stdout**; do not add a store/upsert. Surface it in a *separate*
  PR so the auth surface stays clean.

## 7. Comments and naming around security code

- Scanners do keyword/heuristic matching. Write security comments to **explain the
  safeguard** (what we deliberately do *not* do and why) — e.g. the `SaveToken`
  re-tighten comment, or `Login()`'s "don't echo the body" note. A good comment
  doubles as the reviewer's answer.
- The `G101` / `gosec` `//nolint` markers on `EnvPassword` / `EnvToken` in
  `internal/config/config.go` are there because those are **env-var names, not
  embedded secrets** — keep the explanatory comment with any such marker.
- Don't "hide" from the scanner by deleting honest comments — fix the behavior; the
  clearer comment is a side effect.

## 8. Community-friendly paths — never machine-specific

Paths in **tracked docs, comments, examples, tests, CI, `Makefile`, and
`.goreleaser.yaml`** must be **generic and portable**: no absolute paths, no home
directories, no usernames, no drive letters (e.g. `C:\Users\NAME\…`,
`/home/NAME/…`). Use repo-relative paths, placeholders (`$HOME/bin`, `~`,
`$(go env GOPATH)/bin`), or runtime resolution:

- Go: `os.UserHomeDir()`, `os.UserConfigDir()`, `os.TempDir()`, `t.TempDir()` in
  tests, `filepath.Join` — never a literal home dir. `internal/export/expandUser`
  and `internal/config/discoverConfigPath` model this.
- Tests write only under `t.TempDir()`.
- Examples/sample configs use neutral placeholders — `you@example.com`,
  `~/Downloads/loseit-export.zip` — never a real local path or username.
- `CLAUDE.md` and `GOAL.md` predate this rule and still contain a few owner-local
  paths; when you touch a line, generalize it. Don't bake new machine-specific
  paths in.

---

## Pre-publish checklist

Run before merging anything that touches config, auth, file I/O, network, docs, or
`SKILL.md`:

- [ ] No doc, comment, or SKILL field claims the tool "writes no files" / "writes
      nothing" — the token cache is disclosed and a `Security & secrets` section is
      present (README + SKILL).
- [ ] Every secret-bearing file written `0600` (dir `0700`), with the `Chmod`
      re-tighten pattern; secrets stay gitignored.
- [ ] No secret printed to stdout/stderr/logs at any verbosity; password masked in
      `config show`; `doctor` echoes no token/password.
- [ ] `SKILL.md` permissions/env/network block matches the code exactly — and the
      `files.write` entry declares the token cache.
- [ ] No new `os/exec`, shell-out, or non-HTTPS / user-supplied network target; no
      write path (DAILY_LOG / sync / upsert) added to this repo.
- [ ] No machine-specific paths (home dirs, usernames, drive letters) in code,
      docs, comments, or commit messages on lines you touched.
- [ ] `go build ./... && go vet ./... && go test -race ./...` green; `gofumpt -l .`
      empty; `golangci-lint run` clean.
- [ ] New security behavior is pinned by an **immutable regression test** (below).
- [ ] **Live end-to-end test** run (CLAUDE.md's one hard rule) before the PR.

## Immutable tests for security behavior

Every security guard gets a test that **fails loudly if the safeguard is removed**.
Current set:

- `internal/export/login_test.go`
  - `TestSaveTokenWritesOwnerOnly` — token file `0600`, created dir `0700`.
  - `TestSaveTokenReTightensExistingFile` — re-saving over a loose file restores
    `0600` (removing the `Chmod` fails this).
- `internal/cli/commands_test.go`
  - `TestConfigShowNeverRevealsPassword` — neither `config show` nor `--json`
    emits the password value.
  - `TestDoctorNeverRevealsTokenOrPassword` — `doctor` never echoes the token or
    password.
- `internal/cli/docs_security_test.go`
  - `TestUserDocsDoNotMisrepresentLocalWrites` — README + SKILL never carry the
    banned "writes no files" phrasings, always disclose the token cache, and keep a
    `Security & secrets` section.

When you add a safeguard, add the matching test in the **same PR**. Name it so the
guarantee is obvious, and assert the **negative** (the bad thing does NOT happen),
not just the happy path.

## What the ClawHub scan checks (keep all of these green)

The scanner sweeps these categories. loseit-cli currently passes them all; the
rules above are what keeps them passing. Re-check the relevant ones whenever you
touch that surface:

| Category | Sub-checks | Where we stay clean |
|---|---|---|
| **MCP Tool Poisoning** | Hidden Instructions, Unicode Deception, Parameter Description Injection | §1 — honest, consistent descriptions; no hidden directives in SKILL. |
| **Intent-Code Divergence** | (read-only vs. token write) | §1 — qualify "read-only"; disclose the token cache. |
| **Missing User Warnings** | (plaintext secrets, autonomous writes) | §1–§2, §6 — `Security & secrets` section; consumer write path is user-initiated. |
| **Data Exfiltration** | External Transmission, Env Harvesting, File Enumeration | §3, §5 — HTTPS to Lose It only; read only the documented files; no secret in output. |
| **Prompt Injection** | Instruction Override, Hidden Instructions, Exfiltration Commands | No instruction-bearing content in code/docs; export data is parsed, never executed. |
| **Privilege Escalation** | Excessive Permissions, Sudo/Root, Credential Access | §2, §4 — `0600`/`0700`, no `sudo`/shell, minimal credential reads. |
| **Supply Chain** | Unpinned Deps, External Script Fetching, Obfuscated Code | Pure stdlib + cobra; `go.mod` pinned; `goreleaser` builds; no fetched scripts. |
| **Excessive Agency** | Unrestricted Tool Access, Autonomous Decisions, Scope Creep | §6 — dumb extractor; no write path; new domains ship as separate PRs. |
| **Output Handling** | Unvalidated Output Injection, Cross-Context, Unbounded | stdout is bounded JSON/table; stderr separate; no secret crosses either. |
| **System Prompt Leakage** | Direct/Indirect/Tool-Based | No system prompt embedded; nothing to leak. |
| **Memory Poisoning** | Persistent Context Injection, Context Stuffing | No persisted agent memory in this repo. |
| **Tool Misuse** | Parameter Abuse, Chaining Abuse, Unsafe Defaults | §4–§5 — narrow flags, HTTPS defaults, no arbitrary URLs/files. |
| **Rogue Agent** | Self-Modification, Session Persistence | Token cache is the only persistence — declared, `0600`, expiring (§1–§2). |
| **Trigger Abuse** | Overly Broad / Shadow / Keyword-Baiting Triggers | SKILL `description` matches behavior; no bait keywords. |
| **Behavioral AST** | `exec()` / `eval()` / Dynamic Import | None — no `os/exec`, no dynamic code (§5). |
| **Taint Tracking** | Direct / Variable-Mediated Taint, Credential Exfil Chain | Credentials flow only into the login POST; never into output or a sink (§3). |
| **YARA Signatures** | Malware / Webshell / Cryptominer | N/A — single-purpose extractor. |
| **MCP Least Privilege** | Underdeclared Capability, Wildcard Permission, Missing Declaration | §4 — permissions block matches code; token write declared, no wildcards. |

## Handling a finding that is a genuine false positive

If a finding can't be fixed because the behavior is essential and already minimal
(e.g. "this credential-collecting tool reads credentials"):

1. Confirm the behavior really is minimal and documented (permissions in
   `SKILL.md`, the `Security & secrets` section, precedence in code comments).
2. Prefer a small **hardening** that lowers the finding's severity/confidence over
   doing nothing — that is how the original "writes no files" High became a
   non-issue: we kept the (unavoidable) token write but disclosed it, tightened it,
   and pinned it with tests.
3. Record the rationale here and in the PR so the next reviewer doesn't
   re-litigate it. Never silence a scanner by obfuscating honest code.
