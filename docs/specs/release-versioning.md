# Release and Versioning Specification

## Overview

`ww` uses SemVer tags for public releases and commit-aware dev metadata for untagged builds.

## Release Tags

- Public releases use git tags in the form `vMAJOR.MINOR.PATCH`.
- The first public release is `v0.3.0`.
- While the CLI surface is still stabilizing, releases stay within the `v0.x.y` range.
- Pushing a `v*` tag triggers automated release packaging and publication.

## Build Metadata Contract

- Tagged release builds inject both:
  - `Version`: the SemVer tag, for example `v0.3.0`
  - `CommitHash`: the short git commit hash, for example `abc1234`
- Untagged builds leave `Version` empty and inject only `CommitHash`.
- If `CommitHash` is unavailable, dev builds fall back to `dev`.

## `ww version` Output

### Text output

- Tagged release build: `ww version v0.3.0`
- Dev build with commit hash: `ww version dev+abc1234`
- Dev build without commit hash: `ww version dev`

### JSON output

- Tagged release build:
  - `{"version":"v0.3.0","commit":"abc1234"}`
- Dev build with commit hash:
  - `{"version":"dev","commit":"abc1234"}`
- Dev build without commit hash:
  - `{"version":"dev","commit":"dev"}`

For dev builds, the human-readable text output combines the dev marker and commit hash (`dev+<short-hash>`), while JSON keeps them separate so scripts can rely on stable fields.

## Release Automation

- GoReleaser is the source of truth for release packaging.
- Release builds target:
  - `darwin/arm64`
  - `darwin/amd64`
  - `linux/arm64`
  - `linux/amd64`
- GoReleaser creates a GitHub Release on tag push.
- GoReleaser publishes a Homebrew formula to `yoskeoka/homebrew-ww`.
- Phase 3 scope includes Homebrew tap distribution only. A native install script is out of scope for this phase.
