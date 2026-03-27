# 014: Phase 3 — Installation, Versioning, and Release Automation

> **Execution**: Use `/execute-task` to implement this plan.

**Parent plan**: `docs/exec-plan/todo/phase3-polish.md`

**Objective**: Add SemVer-based versioning (starting at `v0.3.0`), GoReleaser configuration for automated cross-platform builds, and Homebrew tap distribution via `yoskeoka/homebrew-ww`.

## Spec Changes

| File | Change |
|------|--------|
| `docs/specs/cli-commands.md` | Update `ww version` output format to show SemVer for tagged builds |
| `docs/specs/release-versioning.md` | New spec: SemVer tagging strategy, `ww version` output contract, dev-vs-release build metadata |

### Version Output Spec

- Tagged release build (text): `ww version v0.3.0`
- Dev/untagged build (text): `ww version dev+<short-hash>`
- `ww version --json` (release build): `{"version": "v0.3.0", "commit": "<short-hash>"}`
- `ww version --json` (dev build): `{"version": "dev", "commit": "<short-hash>"}`
  - Note: For dev builds, the human-readable text output concatenates the version and short hash (`dev+<short-hash>`), while the JSON output splits them into separate `version` and `commit` fields.

### Release Strategy Spec

- SemVer tags starting at `v0.3.0`, incrementing within `v0.x.y` until CLI surface is stable
- GitHub Release created automatically by GoReleaser on tag push
- Homebrew formula auto-generated and pushed to `yoskeoka/homebrew-ww` by GoReleaser
- Phase 3 scope: Homebrew tap only. Native binary install script (`curl | sh`) is a future enhancement

## Design Decision Changes

| File | Change |
|------|--------|
| `docs/design-decisions/adr.md` | Append ADR: "Dual version strategy — SemVer for releases starting at v0.3.0, commit-hash for dev builds. GoReleaser for release automation." |

## Code Changes

| File | Change |
|------|--------|
| `cmd/ww/version.go` | Add `var Version string` alongside existing `CommitHash`; update `printVersion()` to prefer `Version` when set, fall back to `dev+<hash>` |
| `cmd/ww/sub_version.go` | Support `--json` flag for structured version output |
| `cmd/ww/main.go` | Update top-level `--version` flag to use the new `printVersion()` |
| `Makefile` | Keep `CommitHash` injection via ldflags; optionally pass `-X main.Version=$(VERSION)` only for release targets so that non-release builds leave `Version` empty and use the `dev+<short-hash>` fallback |
| `.goreleaser.yaml` | New file: GoReleaser config with cross-compilation (darwin/arm64, darwin/amd64, linux/arm64, linux/amd64), ldflags for Version+CommitHash, GitHub Release, Homebrew tap brew section pointing to `yoskeoka/homebrew-ww` |
| `.github/workflows/release.yml` | New file: GitHub Actions workflow triggered on `v*` tag push, runs `goreleaser/goreleaser-action` |

### GoReleaser Config Notes

- `builds[0].ldflags`: `-s -w -X main.Version={{.Version}} -X main.CommitHash={{.ShortCommit}}`
- `brews[0].repository`: `owner: yoskeoka`, `name: homebrew-ww`
- `brews[0].install`: `bin.install "ww"`
- Archives: `tar.gz` for Linux, `zip` for Darwin (or `tar.gz` for both — GoReleaser default is fine)
- Checksum generation enabled (default)

### Prerequisite: Create `yoskeoka/homebrew-ww` Repository

Before the first release tag, the tap repository `yoskeoka/homebrew-ww` must exist on GitHub. This is a manual step (create empty repo with a README). GoReleaser will push the formula file on release.

## Docs Changes

| File | Change |
|------|--------|
| `docs/project-plan.md` | Freeze naming as `ww`, clarify Phase 3 concrete scope, note version starting at `v0.3.0` |
| `docs/spec-code-mapping.md` | Add `docs/specs/release-versioning.md` mapped to `cmd/ww/version.go`, `.goreleaser.yaml`, `.github/workflows/release.yml` |

## Sub-tasks

- [ ] [parallel] Add `docs/specs/release-versioning.md`
- [ ] [parallel] Update `docs/specs/cli-commands.md` for `ww version` output changes
- [ ] [parallel] Append ADR entry for dual version strategy and GoReleaser
- [ ] [parallel] Update `docs/project-plan.md` with naming freeze and Phase 3 scope
- [ ] [depends on: specs] Update `cmd/ww/version.go` and `sub_version.go` for SemVer + JSON support
- [ ] [depends on: specs] Update `Makefile` with `VERSION` ldflags
- [ ] [depends on: specs] Create `.goreleaser.yaml`
- [ ] [depends on: specs] Create `.github/workflows/release.yml`
- [ ] [depends on: goreleaser config] Update `docs/spec-code-mapping.md`
- [ ] [depends on: implementation] Add tests for version output (tagged and dev builds)
- [ ] [manual, before first tag] Create `yoskeoka/homebrew-ww` repository on GitHub

## Verification

- `make build && ./ww version` shows `ww version dev+<hash>` (no tag)
- `go build -ldflags "-X main.Version=v0.3.0 -X main.CommitHash=abc1234" ./cmd/ww/ && ./ww version` shows `ww version v0.3.0`
- `./ww version --json` outputs valid JSON with version and commit fields
- `goreleaser check` passes (validates `.goreleaser.yaml`)
- `goreleaser release --snapshot --clean` produces archives for all target OS/arch
- `make test` and `make lint` pass
- Homebrew tap section in `.goreleaser.yaml` points to `yoskeoka/homebrew-ww`
