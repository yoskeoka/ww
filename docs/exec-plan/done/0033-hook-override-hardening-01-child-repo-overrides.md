# 0033: Hook Override Hardening 01 Child Repo Overrides
> **Execution**: Use `/execute-task` to implement this plan. After implementation is complete, use `/review-task` to prepare and create the PR.

**Shared branch**: `plan/hook-override-hardening`

## Objective

When `ww` targets a child repo from a workspace context, it must resolve that child repo's effective config instead of reusing only the caller's already-loaded config. Completion means `ww create --repo ai-arena ...` and related preview paths honor `ai-arena/.ww.toml`, so repo-local `copy_files`, `symlink_files`, hooks, and future repo-scoped overrides apply even when the command started from the workspace root or another repo worktree.

## Context

- `ai-arena/.ww.toml` currently declares `copy_files = [".env"]` and `symlink_files = ["operator-ui/node_modules"]`, but those settings are skipped when `ww` is invoked from the workspace and then redirected with `--repo ai-arena`.
- The current manager handoff clones `base.Config` into the selected repo manager, so repo-local config lookup never re-runs for the actual target repo.
- This plan is the prerequisite for sandbox recovery flows because any later explicit replay command must materialize the target repo's own config, not the workspace caller's config.

## Existing Implementation References

- `internal/config/config.go`
  - `Config`, lines 15-23
  - `LoadWithOptions`, lines 40-73
  - `decodeFields`, lines 127-154
  - `mergeConfig`, lines 156-178
- `cmd/ww/main.go`
  - `newManagerWithOptions`, lines 139-189
  - `loadManagerContext`, lines 191-219
- `cmd/ww/helpers.go`
  - `managerForSelectedRepo`, lines 23-35
  - `managerForRepo`, lines 38-63
- `cmd/ww/config_test.go`
  - `TestNewManagerWithoutGlobalConfigKeepsRepoLocalBehavior`, lines 11-26
  - `TestNewManagerLoadsGlobalConfigAndLocalOverridesPerKey`, lines 28-73
  - `TestNewManagerUsesMainWorktreeRootForGlobalProjectMatching`, lines 75-117
- `internal/config/config_test.go`
  - sandbox search coverage, lines 170-234
  - repo-local/global overlay coverage, lines 280-409
- `docs/specs/configuration.md`
  - config overview and layering contract, lines 5-140
- `docs/specs/cli-commands.md`
  - `ww create` target-repo behavior, lines 69-95

## Code Change Map

- `internal/config/config.go` (MODIFY)
  - config loading entrypoints and/or options so `ww` can resolve the effective config for an explicit target repo root instead of only the caller directory
- `cmd/ww/main.go` (MODIFY)
  - manager/context setup so repo-root, workspace-root, sandbox boundary, and fallback lookup stay correct when the selected repo differs from the caller repo
- `cmd/ww/helpers.go` (MODIFY)
  - `managerForRepo` so selected child repos receive freshly resolved config rather than a shallow copy of `base.Config`
- `cmd/ww/config_test.go` (MODIFY)
  - add target-repo selection coverage for workspace-root and secondary-worktree call sites
- `internal/config/config_test.go` (MODIFY)
  - add loader coverage for child-repo-local `.ww.toml` overlays in workspace and sandbox-aware paths

## Spec Changes

- `docs/specs/configuration.md`
  - document that explicit child-repo selection resolves the target repo's repo-local `.ww.toml`, with the same layering and replacement rules as direct invocation inside that repo
- `docs/specs/cli-commands.md`
  - document that `ww create --repo <name>` and other repo-targeted flows use the selected repo's effective config for copy/symlink/hook behavior

## Sub-tasks

- [ ] Add a target-repo-aware config resolution path that can load effective config for a selected child repo without breaking current single-repo behavior
- [ ] Update manager construction for `--repo` flows so the selected repo manager uses that resolved config and the correct main repo root
- [ ] Add regression tests covering:
  - workspace root -> `--repo child` config resolution
  - secondary worktree -> `--repo child` config resolution
  - sandbox-aware fallback behavior when the selected repo has its own `.ww.toml`
- [ ] Update spec language for target-repo config resolution and verify it matches the shipped behavior

## Design Decisions

- Preserve the existing full-replacement semantics per field; this plan changes *which repo-local file is loaded*, not how config keys merge.
- Keep config resolution centralized in loader/manager setup rather than sprinkling per-command override logic across `create`, `remove`, `cd`, or interactive flows.

## Parallelism

- This plan can execute in parallel with `0034-hook-override-hardening-02-lifecycle-context.md`.
- `0035-hook-override-hardening-03-sandbox-replay.md` depends on this plan because replay must materialize the selected repo's own effective config.
