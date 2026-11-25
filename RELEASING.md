# Releasing loseit-cli

A release is cut by merging the open release PR, titled `chore(main): release X.Y.Z`.
Merging ordinary PRs only updates `main`; merging the release PR is the deliberate publish step.

## How it works

1. Merge feature and fix PRs into `main` with Conventional Commit titles.
2. [release-please](https://github.com/googleapis/release-please-action) (`.github/workflows/release-please.yml`)
   runs on every push to `main` and keeps one release PR open with the next version and a
   preview of the notes. It only changes `.github/.release-please-manifest.json`.
3. When you want to ship, merge the release PR as an admin ("Merge without waiting for
   requirements to be met"). The **Release Please** workflow tags `vX.Y.Z`, creates the GitHub
   Release, and calls the **Release** workflow (`.github/workflows/release.yml`) in the same run.
   That runs **GoReleaser**, which cross-compiles linux/darwin/windows × amd64/arm64, stamps the
   version via ldflags, attaches six archives plus `checksums.txt`, and writes the release notes.

```sh
gh pr list --label "autorelease: pending"
gh run watch
gh release view vX.Y.Z
```

Versioning is [semver](https://semver.org) and comes from the commits since the last release:
`fix:` bumps **patch**, `feat:` bumps **minor**, `feat!:` or a `BREAKING CHANGE:` footer bumps
**major** (a breaking change to the CLI surface or JSON contract in `docs/MACHINE_CONTRACT.md`).
Commits like `ci:`, `docs:`, `test:` and `chore:` alone don't open a release PR. To force a
version, land a commit on `main` with a `Release-As: X.Y.Z` footer.

A **major** bump also renames the Go module: v2.x must live at
`github.com/stozo04/loseit-cli/v2`, so update `go.mod`, every import, the `go install` commands
and `SKILL.md`'s install entry in a PR **before** merging a release PR that proposes 2.0.0.
Without it the Go proxy rejects the tag and `go install …@latest` keeps serving v1.

The current version lives in `.github/.release-please-manifest.json` and the config in
`.github/release-please-config.json` (`release-type: go`, no component in the tag, no
`CHANGELOG.md`). Both sit under `.github/`, so ClawHub never ships them.

**Why GoReleaser runs inside the Release Please workflow:** a tag created with the workflow's
own `GITHUB_TOKEN` does not start other workflows, so the tag push would never fire
`release.yml` on its own. The Release Please workflow calls it directly (`workflow_call`).

**Why the release PR shows required checks as "Expected":** GitHub starts no workflows for a PR
opened with `GITHUB_TOKEN`. The release PR only bumps the manifest, and the code it releases
already passed CI on `main`, so an admin merge past the waiting checks is the intended path. To
get CI on release PRs instead, give release-please a PAT or GitHub App token; that also makes the
tag push start `release.yml` by itself, so the `goreleaser` job in `release-please.yml` would
then have to go.

## Before you merge the release PR

- CI is green on `main`.
- Locally: `make check` passes (tidy, fmt, vet, lint, race tests).
- The proposed version is the one you want. If not, land a `Release-As:` commit first.

## Manual fallback: push a tag

Pushing a `v*` tag still runs `release.yml` on its own:

```sh
git checkout main && git pull
git tag -a v1.2.3 -m "v1.2.3"
git push origin v1.2.3
```

Then set `.github/.release-please-manifest.json` to the same version in a PR, or release-please
will propose a version based on its last release.

## After it runs

- Confirm the **GitHub Release** appears with six archives + `checksums.txt`:
  `gh release view v1.2.3`
- Confirm a downloaded binary reports the right version: `loseit-cli version`.
- `go install github.com/stozo04/loseit-cli/cmd/loseit-cli@latest` picks up the new tag.

Pre-release tags (e.g. `v1.2.3-rc.1`, pushed by hand) are auto-marked as prereleases by
GoReleaser (`prerelease: auto` in `.goreleaser.yaml`).

## Release notes & changelog (auto-generated)

There is no `CHANGELOG.md` to hand-maintain. The release PR body previews the notes; on every
`vX.Y.Z` tag GoReleaser replaces the GitHub Release body (`release.mode: replace`) with:

- **Changelog** — commit subjects since the previous tag, **grouped by [Conventional
  Commit](https://www.conventionalcommits.org) type**: `feat:` → **Features**, `fix:` →
  **Bug fixes**, everything else → **Other changes**. Commits prefixed `docs:`, `test:`,
  or `chore:` (with or without a scope, such as `chore(deps):`) are excluded.
- **Footer** — a static install snippet (`go install …@<tag>` + a pointer to
  `SKILL.md`/`AGENTS.md`), appended to every release.

So **commit subjects are the release notes and pick the version** — write Conventional Commits
and squash-merge each PR with a clean `feat:`/`fix:` title so one PR = one tidy changelog line.
Config lives in `.goreleaser.yaml` → `changelog.groups` and `release.footer`. (There is no
per-release header; add any release-specific narrative by editing the GitHub Release after it posts.)

## ClawHub (separate)

The ClawHub skill is **not** tied to releases. It republishes whenever a bundled file
(`SKILL.md`, `LICENSE`, `config.example.json`) or `.clawhubignore` changes on `main`, via `.github/workflows/publish-clawhub.yml` (needs the `CLAWHUB_TOKEN_LOSEIT`
repo secret). Only those three files are published; `.claude/CLAWHUB_STANDARDS.md` §9
explains why. Bumping the CLI version does not require a ClawHub republish, but the
SKILL `install` entry installs `@latest`, so merge the release PR after behavior changes land.

ClawHub hides a new version until its security scans finish (about 10 minutes), and the CLI picks the
next version from the latest public one. When two publishes land inside that window, the second gets
`Version X already exists`. `scripts/publish-clawhub.sh`, which both ClawHub workflows call, retries
as the next patch version (up to five attempts) and fails on any other error. Every publish to the live
listing (a merge to `main` or a `target=live` branch run) waits in one `clawhub-publish-live` queue
(`queue: max`, first in, first out), so none cancels or replaces another and a queued main publish
is never dropped.

To scan a branch without merging, run **Deploy to ClawHub (from branch)** with `target=scan-copy`. It
publishes to `loseit-scan-<run id>-<attempt>`, waits for the scan, puts the verdict in the run summary
and the full report in the `clawhub-scan-report` artifact, then deletes the copy (it keeps the copy if
the report could not be saved). The weekly **Sweep ClawHub scan copies** workflow deletes any copy a
finished run left behind. Both use `scripts/clawhub-scan-copies.sh`, which refuses to delete anything
outside the `loseit-scan-` prefix. `make test-scripts` tests both scripts against a fake `clawhub`.

## Prerequisites (one-time)

- Repo setting **Settings → Actions → General → Workflow permissions → "Allow GitHub Actions to
  create and approve pull requests"** must be on, or release-please can't open the release PR.
