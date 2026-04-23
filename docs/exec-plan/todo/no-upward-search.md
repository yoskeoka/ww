# Sandbox-Constrained Mode

**Execution**: Use `/execute-task` to implement this plan.

## Objective

Make `ww` usable in filesystem-sandboxed environments that cannot reliably read or use parent directories of the current repository. This supersedes the narrower `--no-upward-search` framing from FR-26: the behavior must constrain workspace discovery, config lookup, and default worktree placement, not only disable upward search.

The branch and plan filename remain `no-upward-search` for traceability to the project-plan requirement, but the public user-facing concept should be sandbox-constrained operation.

## Context

Relevant existing decisions:

- `docs/design-decisions/adr.md` records the current workspace detection anchor as a bounded parent/grandparent search. That is appropriate for normal local use, but it intentionally reads parent directories and sibling repo candidates.
- `docs/design-decisions/adr.md` records workspace-mode worktrees under `<workspace_root>/.worktrees` and single-repo worktrees as siblings of the main repo. The sibling single-repo default is not suitable when parent directories are blocked or should be avoided.
- `docs/design-decisions/core-beliefs.md` prioritizes correctness and spec/code parity. This change needs explicit spec coverage before code changes because it alters command behavior, config discovery, workspace discovery, and path defaults.

## User-Facing Naming

`--no-upward-search` is too narrow because it only describes one mechanism. The desired mode also changes the default worktree base from the repo parent to the repo-local `.worktrees` directory and avoids sibling workspace enumeration.

Options:

- `--no-upward-search`: precise for config/workspace traversal, but misleading once worktree placement also changes.
- `--local-only`: communicates current-repo operation, but can be confused with network/remotes and does not clearly explain sandbox motivation.
- `--contained`: communicates bounded filesystem behavior, but is a less common CLI term and may be vague without reading docs.
- `--sandbox`: directly names the target environment and gives room for the full behavior set: no parent discovery, no sibling workspace use, repo-local defaults.

Recommendation: implement a global `--sandbox` flag and matching `.ww.toml` field named `sandbox = true`. The flag should take precedence over config. During execution, update the project plan to rename FR-26 from `--no-upward-search` to sandbox-constrained mode so future work does not preserve the misleading name.

## Behavior Contract

When sandbox mode is enabled:

- `ww` must not search above the current repository root for `.ww.toml`.
- `ww` must not inspect parent or grandparent directories while detecting a containing workspace.
- `ww` must not scan sibling repositories from a parent workspace candidate.
- `ww` must operate as single-repo mode for the current repository.
- `--repo <name>` must be rejected because no workspace repo list is available.
- If `worktree_dir` is unset, worktrees default to `<repo_root>/.worktrees/<repo>@<branch>` instead of `<repo_parent>/<repo>@<branch>`.
- If `worktree_dir` is relative, resolve it against `<repo_root>` instead of the repo parent.
- Absolute `worktree_dir` values remain accepted because they are explicit user intent.
- Existing copy/symlink/hook behavior remains unchanged after the target path is resolved.

Open edge to verify during implementation: running from a secondary worktree currently resolves back to the main working tree. Sandbox mode should keep that behavior only if the required git commands do not force parent directory filesystem reads outside the allowed sandbox. If this cannot be guaranteed, document the limitation and add an actionable error.

## Spec Changes

Update specs before code:

- `docs/specs/cli-commands.md`
  - Add `--sandbox` to global flags.
  - Document that workspace-sensitive commands force single-repo operation in sandbox mode.
  - Document `--repo` rejection in sandbox mode.
- `docs/specs/workspace-discovery.md`
  - Add a sandbox mode section explaining that containing workspace detection and sibling enumeration are skipped.
  - Add sandbox mode worktree path defaults.
- `docs/specs/configuration.md`
  - Add `sandbox = true`.
  - Define config lookup order in sandbox mode: current directory/repo-local config only, no parent walk.
  - Define relative `worktree_dir` anchoring to repo root in sandbox mode.
- `docs/specs/shell-integration.md`
  - Confirm `ww create -q --sandbox <branch>` prints the repo-local `.worktrees` path.
- `docs/project-plan.md`
  - Rename FR-26 from `--no-upward-search` to sandbox-constrained mode and describe the broader behavior.
- `docs/design-decisions/adr.md`
  - Add an ADR entry for sandbox-constrained mode overriding normal workspace discovery and single-repo path layout.

## Code Changes

- `cmd/ww/main.go`
  - Add global `--sandbox` parsing and pass the value into manager construction.
  - Thread the mode into workspace detection and config loading.
- `workspace/workspace.go`
  - Add an options struct, e.g. `DetectOptions{Sandbox bool}`.
  - In sandbox mode, resolve only the current repo and return `ModeSingleRepo`; skip candidate parent/grandparent detection and child/sibling scans that would touch parent directories.
- `internal/config/config.go`
  - Add `Sandbox bool` to `Config`.
  - Add load options so sandbox mode disables upward search beyond the repository root.
  - Preserve fallback behavior only for explicit in-scope directories.
- `worktree/worktree.go`
  - Add a manager/config flag for sandbox-constrained path layout.
  - Change default and relative `worktree_dir` anchoring to repo root when sandbox mode is enabled.
- Command modules that accept `--repo`
  - Ensure `--repo` errors clearly in sandbox mode.
- Tests
  - Add unit tests for config search, workspace detection, and worktree path resolution.
  - Add integration tests for `create`, `create -q`, `list`, and `--repo` rejection in sandbox mode.

## Sub-tasks

- [ ] Update specs and project plan for sandbox-constrained mode.
- [ ] Add ADR entry documenting why sandbox mode overrides existing workspace discovery and single-repo path defaults.
- [ ] Implement global `--sandbox` flag and configuration field.
- [ ] Implement sandbox-aware workspace detection without parent/grandparent/sibling scans.
- [ ] Implement sandbox-aware config search and path anchoring.
- [ ] Add command-level `--repo` rejection coverage.
- [ ] Add unit and integration tests.
- [ ] Run `make test` and any narrower Go test commands needed while iterating.
- [ ] Move this plan to `docs/exec-plan/done/` during execution.

## Parallelism

- [parallel] Workspace detection tests and config search tests can be drafted independently once the spec wording is in place.
- [parallel] Worktree path unit tests can be drafted independently from CLI wiring.
- [depends on: specs and ADR] CLI wiring should wait until the public flag/config names are confirmed.
- [depends on: CLI wiring, detection, config, paths] Integration tests should be finalized after the behavior is implemented.

## Verification

- `go test ./workspace ./internal/config ./worktree`
- `make test`
- Targeted integration tests covering:
  - `ww --sandbox create feat/x` creates `<repo_root>/.worktrees/<repo>@feat-x`
  - `ww --sandbox create -q feat/x` prints only that path
  - `ww --sandbox list` does not include sibling workspace repos
  - `ww --sandbox create --repo other feat/x` returns a clear sandbox-mode error
