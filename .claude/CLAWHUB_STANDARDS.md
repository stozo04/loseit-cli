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

## 3a. No secret-shaped literals in docs — even placeholders

**Incident (2026-07-06):** a ClawHub deploy flagged **Critical /
`suspicious.exposed_secret_literal`** on the former CLAUDE.md auth-flow line, which spelled
out the login form body as a single key=value string with angle-bracket
placeholder values (and the export step showed the session cookies the same way).
No real secret was present — but a heuristic scanner cannot tell a placeholder
assignment from a hardcoded credential, and the shape invites a future edit to
paste a real value into the same spot. Fixed by rewording, not by exception.

**Rules:**

- In tracked docs (CLAUDE.md, README.md, SKILL.md, AGENTS.md), never write a
  credential-bearing key (`password`, `username`, `liauth`, `fn_auth`, `token`)
  immediately followed by `=` and a value — not even an obvious placeholder.
  Describe request/cookie fields in **prose** instead: "form fields `username`,
  `password`, and `grant_type` (the OAuth literal)"; "sends the `liauth` and
  `fn_auth` cookies, both set to the session token".
- Env-var *names* (`LOSEIT_PASSWORD`) and JSON examples with placeholder values
  (`"password": "..."` in config.example.json) are fine — the ban is on
  assignment-shaped literals for the bare credential keys.
- Pinned by `internal/cli/docs_security_test.go::TestDocsCarryNoSecretShapedLiterals`.

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

## 5. The credential endpoints are fixed — untrusted input can't redirect them

**Incident:** the env vars `LOSEIT_LOGIN_URL` / `LOSEIT_EXPORT_URL` (and the
matching `login_url` / `export_url` config.json keys) let the login POST and the
export GET be repointed at an arbitrary host. ClawHub flagged it **Medium ×2
(Description-Behavior Mismatch + Context-Inappropriate Capability)**: those two
requests carry the user's **email/password** and **liauth session cookie**, so an
attacker who can influence the environment or drop a `config.json` in the working
directory could redirect them to their own server and harvest the credentials and
exported health data. That is broader than a "Lose It only" extractor should be.

**Rules:**

- **The login/export URLs are compiled-in constants** (`DefaultLoginURL` /
  `DefaultExportURL`) and have **no env or config override** — there is no
  `LOSEIT_LOGIN_URL`/`LOSEIT_EXPORT_URL`, and `fileConfig` deliberately does not
  decode `login_url`/`export_url`. A Lose It endpoint move is a **code change +
  rebuild**, never a runtime knob. (docs/PROJECT_INSTRUCTIONS.md's auth playbook already assumes this.)
- **Defense in depth:** before sending anything, `Login` and `fetchWithToken` call
  `assertFirstPartyURL` — credentials/cookies may only go to `*.loseit.com` over
  HTTPS (loopback is allowed *only* so tests can use an httptest server). If a
  future change ever lets an untrusted URL reach there, credentials still can't be
  exfiltrated off-domain.
- **Tests never redirect via env/config.** Production endpoints can't be changed by
  untrusted input, so tests inject a loopback URL through a pre-resolved
  `config.Config` (the `newRootCmd(app)` seam in `cli`, or a direct struct in
  `export`) — never by setting `LOSEIT_*_URL`.
- Pinned by `internal/config/config_test.go::TestURLEndpointsAreNotOverridableByUntrustedInput`
  + `TestDefaultEndpointsAreFirstPartyHTTPS`, and
  `internal/export/login_test.go::TestLoginRefusesNonFirstPartyURL` +
  `TestFetchZipRefusesNonFirstPartyURL`.

## 5a. No code execution / injection surfaces

- Don't pass user / config / env values into `exec.Command`, a shell, `eval`-like
  APIs, or template-to-code paths. We have **no `os/exec` usage** today — keep it
  that way unless there's a strong, reviewed reason.
- Treat the export ZIP and any fetched API data as **untrusted input**: validate,
  don't execute. CSV parsing tolerates ragged rows; it never interprets cell
  contents as code.

## 5b. Testing with real credentials is a privacy hazard — warn and contain it

docs/PROJECT_INSTRUCTIONS.md's "one hard rule" requires a **live** end-to-end test with **real**
credentials and **real** health data. That is necessary (mocks miss Lose It auth
changes) but risky, so the instruction must always carry the privacy guard:

- Never paste `config.json`, the token, a login request, or the export / `days`
  output into commits, PRs, issues, screenshots, chat, or **CI logs**; never run
  the live test in shared/hosted CI.
- Use a **local, gitignored** config.json or env vars cleared afterward; redact
  email / token / nutrition before sharing any debug output. Prefer a throwaway
  Lose It account where possible.
- The export ZIP and `days` JSON are personal health data — treat them as secrets
  (§2–§3 apply to them too).

## 5c. Redirects never carry credentials off Lose It

**Incident (2026-11-13):** ClawHub's A.I.G scanner flagged **Warning / T09
Insecure Skill Coding Practices** on `internal/export/login.go`: `assertFirstPartyURL`
checked only the first URL, and the default `http.Client` follows redirects, replaying
a POST body on 307/308. A redirect from the login endpoint would have resent the email
and password to any host; a redirect from the export endpoint could carry the session
cookie to `http://` Lose It in cleartext. A test proved both before the fix.

**Rules:**

- The login POST follows **no** redirects (`CheckRedirect: refuseRedirects`); any 3xx
  fails with "refusing to resend your credentials". Lose It's login answers directly
  (no redirect, verified 2026-11-13).
- The export GET follows a redirect only if the target passes `assertFirstPartyURL`
  (`firstPartyRedirectsOnly`, at most 5 hops). Lose It redirects a cookieless export
  request to `https://www.loseit.com/account/…`, which stays allowed. The test-only
  loopback exemption in `assertFirstPartyURL` does not extend to redirects: a hop to a
  loopback address is accepted only when the original request was already loopback.
- Every new credential- or cookie-bearing request sets a `CheckRedirect` the same way.
- Pinned by `internal/export/redirect_test.go::TestLoginNeverResendsCredentialsOnRedirect`
  (307, 308 and 302 to a second server, which must receive nothing) and
  `TestFetchZipRefusesRedirectsThatLeaveLoseItOverHTTPS` (off-domain and `http://`), and
  `TestExportRedirectFromLoseItToLoopbackIsRefused`.

## 6. Nutrition-only scope + the write path lives in the consumer

**Incident:** a dev-doc section described the export ZIP's *other* contents
(bodyweight, exercise, steps, profile, notes, photos…) and framed surfacing them as
"the natural next step" under an "extract-all" philosophy. ClawHub flagged it
**Medium / Description-Behavior Mismatch**: a skill advertised as a *nutrition*
reader that frames broad collection of sensitive personal data as the plan is a
scope-expansion + privacy risk to any integrating agent.

**Rules:**

- **Read only the two nutrition CSVs** (`food-logs.csv`, `daily-calorie-summary.csv`)
  and emit only nutrition. The rest of the export ZIP is **deliberately ignored** —
  never parsed, stored, transmitted, or emitted. That is **data minimization**, not
  a gap to fill. Say it that way in the docs.
- **Never frame scope expansion as a goal / roadmap / "natural next step."** If a
  real need arises, it is a **separate, security-reviewed PR** adding an
  **explicit, opt-in, off-by-default** command for that *one* domain, with the
  `SKILL.md` permissions updated to declare it — so advertised scope always equals
  actual behavior. Pinned by
  `internal/cli/docs_security_test.go::TestDevDocsDoNotFrameScopeExpansion`.
- loseit-cli is the **dumb extractor**: get the *nutrition* data out, intact, to
  stdout. It has **no** knowledge of `DAILY_LOG.json`, no `sync`, no upsert. The
  nutrition-storage procedure belongs to the **consumer** (personal-workout-ai),
  runs **in response to the user's own action**, and only sets the day's
  `nutrition` key. Never port a write path back into this repo.
- **Keep auth docs in sync with the shipped self-healing model.** The tool *has* a
  `login` command and *auto-refreshes* an expired `liauth` cookie via
  email/password. No doc may claim "no login command" or "no auto-refresh" — that
  stale framing (from the original spec, now deleted) was a High Intent-Code
  Divergence finding.

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
- Shared project instructions use portable tool locations. Keep new paths portable too.

## 9. The published bundle is a whitelist — only the skill, never the repo

**Incident (2026-11-13):** the publish workflows ran `clawhub skill publish .`, and
the clawhub CLI uploads every non-dotfile that is not gitignored. So the "skill"
ClawHub scanned was the whole repo: 43 files, including `internal/cli/completion.go`,
`internal/version`, the Go tests, `scripts/sync-harness-skills.py`, and the agent
docs. Its reviewer graded each file against the nutrition-extractor description and
flagged **6 × High / MCP Tool Poisoning** (dev files whose purpose isn't
the declared one), **High / Vague Triggers** (`docs/OPERATING_INSTRUCTIONS.md`, the
repo's own agent rules), **Skill Enumeration** and **subprocess** findings (the
harness-sync script), plus duplicate **External Transmission** hits on copied docs.

**Rules:**

- `.clawhubignore` is a **whitelist**: `/*` excludes everything, then `!/<file>`
  lines allow back exactly `SKILL.md`, `LICENSE`, and `config.example.json`. The
  Go source, tests, dev scripts, and agent docs stay on GitHub, where the SKILL
  `homepage` points.
- The shipped SKILL.md avoids phrases SkillSpector's static rules match even inside a
  warning: say "a directory other users can read", never `world-readable`/`world-writable`
  (TM3 Unsafe Defaults). Pinned by
  `internal/cli/docs_security_test.go::TestShippedSkillAvoidsUnsafeDefaultsPhrasing`.
- `SKILL.md` links to anything outside the bundle with an absolute GitHub URL — a
  relative link to an excluded file is broken for ClawHub users.
- Adding a file to the bundle is a deliberate change: update `.clawhubignore`, the
  `EXPECTED_BUNDLE` list in `scripts/check-clawhub-bundle.mjs`, and this section in
  the same PR, and make sure the new file matches the SKILL description.
- Both publish workflows pin `clawhub` and run the bundle check before publishing,
  so a CLI upgrade that changes file selection fails loudly instead of shipping the
  repo. Bump the pinned version in all three workflows together.
- Pinned by `scripts/check-clawhub-bundle.mjs` (`make check-clawhub`, CI job
  `clawhub bundle`), which runs the clawhub CLI's own `listSkillFiles` against the
  checkout and fails unless the bundle is exactly the whitelist.

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
- [ ] No assignment-shaped credential literals in docs (a credential key + `=` +
      value, even a placeholder) — describe request/cookie fields in prose (§3a).
- [ ] `SKILL.md` permissions/env/network block matches the code exactly — and the
      `files.write` entry declares the token cache.
- [ ] Every credential/cookie-bearing `http.Client` sets `CheckRedirect` (§5c).
- [ ] Login/export URLs stay compiled-in: no `LOSEIT_*_URL` env, no
      `login_url`/`export_url` config key, no flag — and `assertFirstPartyURL` still
      guards both requests. Tests inject loopback via a pre-built config, never env.
- [ ] Any instruction to test with real credentials carries the privacy warning
      (no real secrets/health data in commits, PRs, logs, screenshots, or CI).
- [ ] No new `os/exec`, shell-out, or non-HTTPS / user-supplied network target; no
      write path (DAILY_LOG / sync / upsert) added to this repo.
- [ ] No machine-specific paths (home dirs, usernames, drive letters) in code,
      docs, comments, or commit messages on lines you touched.
- [ ] `make check-clawhub` passes: the ClawHub bundle is exactly `SKILL.md`,
      `LICENSE`, `config.example.json` (§9).
- [ ] `go build ./... && go vet ./... && go test -race ./...` green; `gofumpt -l .`
      empty; `golangci-lint run` clean.
- [ ] New security behavior is pinned by an **immutable regression test** (below).
- [ ] **Live end-to-end test** run (docs/PROJECT_INSTRUCTIONS.md's one hard rule) before the PR.

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
  - `TestDevDocsDoNotFrameScopeExpansion` — docs/PROJECT_INSTRUCTIONS.md never frames collecting the
    rest of the export as a goal / "natural next step"; keeps the data-minimization
    framing.
  - `TestDocsCarryNoSecretShapedLiterals` — tracked docs never write a credential
    key as a key=value literal, even with a placeholder value (the ClawHub
    `suspicious.exposed_secret_literal` guard, §3a).
- `internal/config/config_test.go`
  - `TestURLEndpointsAreNotOverridableByUntrustedInput` — a hostile env var and a
    hostile `config.json` both fail to repoint the login/export URLs.
  - `TestDefaultEndpointsAreFirstPartyHTTPS` — the compiled-in endpoints are HTTPS
    and within `loseit.com`.
- `internal/export/login_test.go`
  - `TestLoginRefusesNonFirstPartyURL` / `TestFetchZipRefusesNonFirstPartyURL` —
    `assertFirstPartyURL` blocks sending credentials/cookies off-domain.

- `internal/export/redirect_test.go`
  - `TestLoginNeverResendsCredentialsOnRedirect` — a redirected login is refused and
    the redirect target receives nothing (§5c).
  - `TestFetchZipRefusesRedirectsThatLeaveLoseItOverHTTPS` — the export never follows
    a redirect off `*.loseit.com` or to `http://` (§5c).
  - `TestExportRedirectFromLoseItToLoopbackIsRefused` — a real Lose It response can't
    redirect the client to a local address (§5c).
- `internal/cli/docs_security_test.go::TestShippedSkillAvoidsUnsafeDefaultsPhrasing` —
  the shipped SKILL.md carries no `world-readable`-style phrase (§9).
- `scripts/check-clawhub-bundle.mjs`
  - fails unless clawhub's own file selection yields exactly the whitelisted
    bundle (§9).

When you add a safeguard, add the matching test in the **same PR**. Name it so the
guarantee is obvious, and assert the **negative** (the bad thing does NOT happen),
not just the happy path.

## What the ClawHub scan checks (keep all of these green)

The scanner sweeps these categories. loseit-cli currently passes them all; the
rules above are what keeps them passing. Re-check the relevant ones whenever you
touch that surface:

| Category | Sub-checks | Where we stay clean |
|---|---|---|
| **MCP Tool Poisoning** | Hidden Instructions, Unicode Deception, Parameter Description Injection | §1, §9 — honest, consistent descriptions; no hidden directives in SKILL; only skill files are published, so no unrelated code is graded against the description. |
| **Intent-Code Divergence** | (read-only vs. token write; declared vs. real behavior) | §1, §5 — qualify "read-only"; disclose the token cache; endpoints are fixed, not redirectable. |
| **Missing User Warnings** | (plaintext secrets, autonomous writes, real-cred testing) | §1–§2, §5b, §6 — `Security & secrets` section; consumer write path is user-initiated; live-test privacy warning. |
| **Data Exfiltration** | External Transmission, Env Harvesting, File Enumeration | §3, §5, §5c — credentials/cookies go only to `*.loseit.com` over HTTPS (fixed endpoints + `assertFirstPartyURL`, enforced on every redirect hop); read only documented files; no secret in output. |
| **Prompt Injection** | Instruction Override, Hidden Instructions, Exfiltration Commands | No instruction-bearing content in code/docs; export data is parsed, never executed. |
| **Privilege Escalation** | Excessive Permissions, Sudo/Root, Credential Access | §2, §4 — `0600`/`0700`, no `sudo`/shell, minimal credential reads. |
| **Supply Chain** | Unpinned Deps, External Script Fetching, Obfuscated Code | Pure stdlib + cobra; `go.mod` pinned; `goreleaser` builds; no fetched scripts. |
| **Excessive Agency** | Unrestricted Tool Access, Autonomous Decisions, Scope Creep | §5–§6 — dumb extractor; no write path; endpoints can't be repointed; new domains ship as separate PRs. |
| **Output Handling** | Unvalidated Output Injection, Cross-Context, Unbounded | stdout is bounded JSON/table; stderr separate; no secret crosses either. |
| **System Prompt Leakage** | Direct/Indirect/Tool-Based | No system prompt embedded; nothing to leak. |
| **Memory Poisoning** | Persistent Context Injection, Context Stuffing | No persisted agent memory in this repo. |
| **Tool Misuse** | Parameter Abuse, Chaining Abuse, Unsafe Defaults | §4–§5 — narrow flags, fixed first-party endpoints, no arbitrary URLs/files. |
| **Rogue Agent** | Self-Modification, Session Persistence | Token cache is the only persistence — declared, `0600`, expiring (§1–§2). |
| **Trigger Abuse** | Overly Broad / Shadow / Keyword-Baiting Triggers | SKILL `description` matches behavior; no bait keywords; repo agent rules (`docs/OPERATING_INSTRUCTIONS.md`) stay out of the bundle (§9). |
| **Behavioral AST** | `exec()` / `eval()` / Dynamic Import | None in the skill — no `os/exec`, no dynamic code (§5a); dev scripts that use `subprocess` stay out of the bundle (§9). |
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
