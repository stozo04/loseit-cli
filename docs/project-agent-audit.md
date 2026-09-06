# Project agent audit, 2026-09-06

Scope: loseit-cli only, following the [model guide](https://developers.openai.com/api/docs/guides/latest-model?model=gpt-6-astra) and [OpenLoop PR 176](https://github.com/stozo04/OpenLoop/pull/176). Read AGENTS, CLAUDE, README, SKILL, RELEASING, ClawHub standards, local review notes, Makefile, CI and publishing/package configuration. Excluded dependencies, builds, archives, and nested worktrees. No existing task PR/branch existed.

Moved the complete frozen machine contract to docs/MACHINE_CONTRACT.md and preserved CLAUDE guidance in docs/PROJECT_INSTRUCTIONS.md. Root pointers match and Cursor discovers all shared documents. Generalized owner-specific Go paths. Preserved nutrition-only data minimization, fixed endpoints, privacy rules, live-test gate, required security tests, and owner-controlled releases. Added relevant retained review decisions without modifying or committing the original untracked review note. Consumer names describe integration context, not external instruction dependencies.

Mirrored the existing root SKILL package and its machine-contract reference into all three harness trees (2 files each). No competing harness copies existed. Root distributable SKILL and reference bytes also match the mirrors. Added the Windows ACL caveat and trusted explicit config guidance from the retained review. ClawHub standards remain in their native existing location, read through shared project instructions by all agents. Publishing workflows remain provider-specific and unchanged.

Reused the reference checker and 16 regression cases with remote-default resolution; make check and CI include it. Retargeted the existing scope security test to the moved document without weakening its assertions, and expanded the secret-literal guard to shared documents. GoReleaser includes the documents and referenced standards/release instructions so packaged pointers resolve.

Validation: make check passes (16 checker cases, tidy, formatting, vet, zero lint issues, race tests in 4 packages; 2 packages have no tests). Explicit binary build succeeds. The final diagnostic-text edit passed the CLI tests again. Sync passes for 2 files x 3; root pointers, packaged skill/reference equality, and link targets verified. Staged diff inspected and git diff --check passes.

Required live check passes: login returned success and a temporary session cache was independently confirmed; a one-year days --json selection returned populated nutrition objects. The initial 30-day selection was empty, so it did not satisfy the gate. No credentials or health values were printed or saved as evidence. Temporary token cache removed. Original local config and review note remain untouched in the original checkout.

Skipped: release archive cross-build, ClawHub publishing/security scan (publishes externally), interactive agent evaluation, merge/release/deployment. No product behavior changed and no publication workflow was dispatched.
