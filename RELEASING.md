# Releasing loseit-cli

Releases are cut by tagging a commit `vX.Y.Z`. A `v*` tag triggers the **Release**
workflow (`.github/workflows/release.yml`), which runs **GoReleaser**: it cross-compiles
the binary for linux/darwin/windows × amd64/arm64, stamps the version via ldflags,
generates `checksums.txt`, and publishes a GitHub Release with the archives attached.

Versioning is [semver](https://semver.org): bump **major** for a breaking change to the
CLI surface / JSON contract (see `AGENTS.md`), **minor** for new commands or flags,
**patch** for fixes. The version is *your* decision — there is intentionally no auto-tagging.

## Before you release

- The change is merged to `main` and **CI is green** on `main`.
- Locally: `make check` passes (tidy, fmt, vet, lint, race tests).
- You've decided the version bump (major/minor/patch).

## How to release

### A. From the GitHub UI (preferred)

**Actions → Release → "Run workflow"** → enter the version (e.g. `v1.2.3`) → Run.

The workflow validates the version, creates and pushes the tag at the current `main`
commit, and runs GoReleaser. (The tag is pushed with the built-in `GITHUB_TOKEN`, which
does not start a second workflow run, so there is no duplicate release.)

### B. From the terminal

```sh
git checkout main && git pull
git tag -a v1.2.3 -m "v1.2.3"
git push origin v1.2.3
```

The pushed tag triggers the same Release workflow.

## After it runs

- Confirm the **GitHub Release** appears with six archives + `checksums.txt`:
  `gh release view v1.2.3`
- Confirm a downloaded binary reports the right version: `loseit-cli version`.
- `go install github.com/stozo04/loseit-cli/cmd/loseit-cli@latest` picks up the new tag.

Pre-release tags (e.g. `v1.2.3-rc.1`) are auto-marked as prereleases by GoReleaser
(`prerelease: auto` in `.goreleaser.yaml`).

## Release notes & changelog (auto-generated)

GoReleaser builds the release body automatically on every `vX.Y.Z` tag — there is no
`CHANGELOG.md` to hand-maintain:

- **Changelog** — commit subjects since the previous tag, **grouped by [Conventional
  Commit](https://www.conventionalcommits.org) type**: `feat:` → **Features**, `fix:` →
  **Bug fixes**, everything else → **Other changes**. Commits prefixed `docs:`, `test:`,
  or `chore:` are excluded.
- **Footer** — a static install snippet (`go install …@<tag>` + a pointer to
  `SKILL.md`/`AGENTS.md`), appended to every release.

So **commit subjects are the release notes** — write Conventional Commits and squash-merge
each PR with a clean `feat:`/`fix:` title so one PR = one tidy changelog line. Config lives
in `.goreleaser.yaml` → `changelog.groups` and `release.footer`. (There is no per-release
header; add any release-specific narrative by editing the GitHub Release after it posts.)

## ClawHub (separate)

The ClawHub skill is **not** tied to releases. It republishes whenever a bundled file
(`SKILL.md`, `LICENSE`, `config.example.json`) or `.clawhubignore` changes on `main`, via `.github/workflows/publish-clawhub.yml` (needs the `CLAWHUB_TOKEN_LOSEIT`
repo secret). Only those three files are published; `.claude/CLAWHUB_STANDARDS.md` §9
explains why. Bumping the CLI version does not require a ClawHub republish. Every publish to the live
listing (a merge to `main` or a `target=live` branch run) waits in one `clawhub-publish-live` queue
(`queue: max`, first in, first out), so none cancels or replaces another and a queued main publish
is never dropped.

To scan a branch without merging, run **Deploy to ClawHub (from branch)** with `target=scan-copy`. It
publishes to `loseit-scan-<run id>-<attempt>`, waits for the scan, puts the verdict in the run summary
and the full report in the `clawhub-scan-report` artifact, then deletes the copy (it keeps the copy if
the report could not be saved). The weekly **Sweep ClawHub scan copies** workflow deletes any copy a
finished run left behind. Both use `scripts/clawhub-scan-copies.sh`, which refuses to delete anything
outside the `loseit-scan-` prefix. `make test-scripts` tests it against a fake `clawhub`.
