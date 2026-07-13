# Dual version strategy: SemVer releases plus commit-aware dev builds

## Status

Accepted on 2026-04-04.

## Context

Phase 3 adds first-release packaging and Homebrew distribution. The previous `ww version` contract only exposed a commit hash, which is enough for local dev builds but not for stable public releases or package managers.

Several options were considered:

- **Commit-hash only**: Keep the current model for every build. Rejected because package managers and users need stable, ordered release versions.
- **SemVer only**: Require every build to carry a release version. Rejected because local builds and untagged CI snapshots still need a useful identifier without pretending to be a release.
- **Dual strategy (chosen)**: Use SemVer tags for releases and keep commit-aware dev output for untagged builds.

GoReleaser was also considered against ad hoc shell scripts for packaging. Shell scripts were rejected because cross-platform archives, GitHub Releases, and Homebrew tap publishing would become harder to audit and maintain.

## Decision

Use a dual version strategy:

- Tagged release builds inject `Version=<SemVer tag>` and `CommitHash=<short hash>`.
- Untagged builds leave `Version` empty and inject only `CommitHash`.
- `ww version` prefers the SemVer tag when present, otherwise renders `dev+<short-hash>` or `dev`.
- `ww version --json` exposes stable `version` and `commit` fields, using `version=dev` for untagged builds.
- Use GoReleaser as the release automation entry point for cross-platform archive builds, GitHub Release publication, and Homebrew tap publishing to `yoskeoka/homebrew-ww`.

## Consequences

- **Positive**: Public releases get stable SemVer identifiers starting at `v0.3.0`.
- **Positive**: Local and CI dev builds stay commit-identifiable without fabricating a release version.
- **Positive**: Release metadata and distribution settings remain auditable in-repo.
- **Negative**: Build tooling now has to manage two metadata paths (`Version` and `CommitHash`) instead of one.
