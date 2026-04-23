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

`--no-upward-search` is too narrow because it only describes one mechanism. The desired mode also changes the default worktree base when operating on a single repository from the repo parent to the repo-local `.worktrees` directory and avoids using parent directories to discover containing workspaces.

Options:

- `--no-upward-search`: precise for config/workspace traversal, but misleading once worktree placement also changes.
- `--local-only`: communicates current-repo operation, but can be confused with network/remotes and does not clearly explain sandbox motivation.
- `--contained`: communicates bounded filesystem behavior, but is a less common CLI term and may be vague without reading docs.
- `--sandbox`: directly names the target environment and gives room for the full behavior set: no parent discovery, no parent-based sibling workspace use, repo-local defaults when operating on one repo.

Recommendation: implement a global `--sandbox` flag and matching `.ww.toml` field named `sandbox = true`. The flag should take precedence over config. During execution, update the project plan to rename FR-26 from `--no-upward-search` to sandbox-constrained mode so future work does not preserve the misleading name.

This must not become a forced single-repo flag. If sandbox mode disabled downward scanning from a workspace root, the behavior would be better named `--single-repo` or similar. The intended contract is bounded filesystem access: use the current directory and descendants when they are the active workspace, but do not read or depend on parent directories.

Sandbox mode is also not a strict policy engine that rejects every path outside the sandbox boundary. Its purpose is to make the default discovery and placement strategy usable when `ww` has been used with `--sandbox` from the start. If an absolute config value or existing git worktree relationship explicitly points outside the sandbox boundary, `ww` should follow that user intent and let the underlying filesystem or git operation fail normally when the real sandbox blocks it. Relative `worktree_dir` values that escape their anchor with `..` remain rejected in sandbox mode, matching current behavior; loosening that rule globally is tracked as follow-up work in `docs/issues/relax-relative-worktree-dir-escape.md`.

## Behavior Contract

When sandbox mode is enabled:

- `ww` must not search above the current repository root for `.ww.toml`.
- `ww` must not inspect parent or grandparent directories while detecting a containing workspace.
- `ww` must not scan sibling repositories from a parent workspace candidate.
- `ww` may still scan immediate child directories of the current working directory. If the current working directory itself is a workspace root with child repositories, sandbox mode should preserve workspace mode and allow `--repo <name>`.
- If the current working directory is inside a child repository, `ww` must not walk upward to discover the containing workspace. In that case, sandbox mode operates only on the current repository and `--repo <name>` is rejected because no workspace repo list is available.
- If the current working directory is not inside git but has immediate child repositories, treat the current directory as a sandbox-bounded workspace root.
- If the current working directory is neither inside git nor a current-directory workspace root, return `not a git repository`.
- If `worktree_dir` is unset while operating on one repo outside a detected current-directory workspace, worktrees default to `<repo_root>/.worktrees/<repo>@<branch>` instead of `<repo_parent>/<repo>@<branch>`.
- If `worktree_dir` is unset in a sandbox-bounded workspace root, keep the workspace layout at `<cwd>/.worktrees/<repo>@<branch>`.
- If `worktree_dir` is relative, resolve it against the active sandbox boundary: the current-directory workspace root in workspace mode, or the repo root in single-repo mode.
- Relative `worktree_dir` values that escape the active anchor with `..` remain rejected in sandbox mode, preserving current path-safety behavior.
- Absolute `worktree_dir` values remain accepted even when they point outside the active sandbox boundary because they are explicit user intent from config. In a real sandbox, an out-of-bound path should fail with the underlying operation-not-permitted or access error instead of being pre-rejected by sandbox mode.
- Existing git worktree relationships, including a secondary worktree whose main working tree is outside the sandbox boundary, should also be followed rather than rejected solely by sandbox mode. If resolving or using that relationship needs access outside the real sandbox, surface the underlying git/filesystem error.
- Existing copy/symlink/hook behavior remains unchanged after the target path is resolved.

Sandbox-mode config lookup algorithm:

1. Determine the sandbox boundary before loading config:
   - if the current working directory has immediate child git repositories, the boundary is the current working directory
   - otherwise, if the current working directory is inside git, resolve the repository's main working tree root and use that as the boundary
   - otherwise, there is no valid boundary and `ww` returns `not a git repository`
2. Search for `.ww.toml` from the current working directory upward, but stop at the sandbox boundary.
3. If the current working directory is a secondary worktree that is not a descendant of the main working tree root, check the main working tree root as an explicit fallback directory. Do not reject this solely because it is outside the current checkout; it is the git-defined repository root for existing `ww` behavior.
4. Do not check fallback directories outside the sandbox boundary except for the explicit main working tree fallback described above.
5. If no config is found within those locations, use defaults.

Compatibility expectation: a user who previously used `ww` without sandbox mode may already have absolute config paths or git worktree relationships that point outside the sandbox-friendly defaults. Sandbox mode should not hide or reinterpret that state. It should attempt the configured operation and rely on the actual sandbox to report permission failures, making it clear to the user that the existing setup needs an unsandboxed run or migration.

## Spec Changes

Update specs before code:

- `docs/specs/cli-commands.md`
  - Add `--sandbox` to global flags.
  - Document that workspace-sensitive commands only use the current-directory workspace root or current repository in sandbox mode.
  - Document `--repo` behavior in sandbox mode: allowed from a current-directory workspace root, rejected when sandbox mode resolves only the current repository.
- `docs/specs/workspace-discovery.md`
  - Add a sandbox mode section explaining that parent/grandparent containing workspace detection is skipped, while current-directory child repo scanning remains allowed.
  - Add sandbox mode worktree path defaults.
- `docs/specs/configuration.md`
  - Add `sandbox = true`.
  - Define config lookup order in sandbox mode: search upward from the current working directory only until the active sandbox boundary.
  - Define relative `worktree_dir` anchoring to the current-directory workspace root in workspace mode or repo root in single-repo mode.
  - Document that absolute `worktree_dir` values are honored even if they point outside the sandbox-friendly default area, with actual sandbox denial surfaced as the underlying filesystem/git error.
  - Document that relative `worktree_dir` values that escape their anchor with `..` remain rejected in sandbox mode.
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
  - In sandbox mode, first scan only the current directory's immediate children. If child repositories are found, return `ModeWorkspace` rooted at the current directory.
  - If the current directory is inside git and no current-directory workspace root is found, return `ModeSingleRepo` for the current repository.
  - Skip candidate parent/grandparent detection and parent-based sibling scans that would touch parent directories.
- `internal/config/config.go`
  - Add `Sandbox bool` to `Config`.
  - Add load options so sandbox mode disables upward search beyond the repository root.
  - Preserve fallback behavior only for explicit in-scope directories.
- `worktree/worktree.go`
  - Add a manager/config flag for sandbox-constrained path layout.
  - Change default and relative `worktree_dir` anchoring to repo root when sandbox mode is enabled.
  - Preserve existing relative `worktree_dir` escape rejection in sandbox mode.
  - Do not add sandbox-mode validation that rejects absolute configured paths solely because they are outside the active sandbox boundary.
- Command modules that accept `--repo`
  - Ensure `--repo` errors clearly in sandbox mode.
- Tests
  - Add unit tests for config search, workspace detection, and worktree path resolution.
  - Add integration tests for `create`, `create -q`, `list`, and `--repo` rejection in sandbox mode.

## Sub-tasks

- [x] Update specs and project plan for sandbox-constrained mode.
- [x] Add ADR entry documenting why sandbox mode overrides existing workspace discovery and single-repo path defaults.
- [x] Implement global `--sandbox` flag and configuration field.
- [x] Implement sandbox-aware workspace detection without parent/grandparent scans while preserving current-directory child repo scans.
- [x] Implement sandbox-aware config search and path anchoring.
- [x] Preserve absolute out-of-bound config paths and existing git worktree relationships, while keeping relative `worktree_dir` escape rejection.
- [x] Add command-level `--repo` rejection coverage.
- [x] Add unit and integration tests.
- [x] Run `make test` and any narrower Go test commands needed while iterating.
- [x] Move this plan to `docs/exec-plan/done/` during execution.

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
  - absolute `worktree_dir` outside the sandbox-friendly default is attempted and reports the underlying filesystem/git error if blocked
  - relative `worktree_dir` escaping with `..` remains rejected in sandbox mode
  - config lookup from a secondary worktree checks the main working tree root fallback
  - `ww --sandbox list` from a workspace root includes immediate child repos
  - `ww --sandbox list` from inside a child repo does not include parent workspace siblings
  - `ww --sandbox create --repo other feat/x` works from a current-directory workspace root
  - `ww --sandbox create --repo other feat/x` returns a clear sandbox-mode error from inside a single child repo
