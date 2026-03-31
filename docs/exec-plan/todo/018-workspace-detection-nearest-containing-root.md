**Execution**: Use `/execute-task` to implement this plan.

# 018: Workspace detection via nearest containing root

**Objective**: Adopt the B design for workspace detection so `ww` resolves the nearest containing workspace root within a bounded 3-layer window. This aligns `ww list`, `ww clean`, and workspace-aware create/remove behavior with human expectations in 1-2-3 level directory layouts, while keeping detection deterministic and non-recursive.

## Background

The current implementation added a grandparent guard that can reject an intuitive parent workspace root when the grandparent also contains multiple git repos. In practice this makes `ww` fall back to single-repo mode in layouts where humans still expect a containing workspace to be detected.

The accepted design is:

- Anchor detection on the current repo's **main worktree root**
- If the current directory itself already qualifies as a workspace root, accept it immediately
- Otherwise search upward by at most two levels for candidates that:
  - contain the current main repo root
  - expose at least two immediate child **real git repositories**
- If multiple candidates qualify, choose the nearest containing candidate
- Managed git worktrees must not count as real git repositories

## Spec Changes

### `docs/specs/workspace-discovery.md`

- Replace the current grandparent-rejection wording with the B algorithm
- Define the bounded search window explicitly:
  - current directory as an immediate workspace-root candidate
  - main repo root
  - parent of main repo root
  - grandparent of main repo root
- Define "nearest containing workspace root" as the tie-break rule
- Clarify that git worktrees are excluded from "real git repository" detection
- Clarify that detection remains non-recursive beyond the bounded window

### `docs/specs/cli-commands.md`

- Update any wording that assumes a simpler parent-only workspace detection model, if present
- Confirm that workspace-sensitive commands (`list`, `clean`, `create --repo`, `remove --repo`) use the detected nearest containing workspace root

### `docs/specs/configuration.md`

- Check whether configuration search wording assumes the older workspace detection behavior
- Update only if the B detection rule changes how callers pass fallback directories

## Code Changes

### `workspace/workspace.go`

- Refactor `Detect()` and helper functions around the new bounded-window search
- Remove the current grandparent rejection logic
- Add helper logic to:
  - resolve candidate directories
  - test whether a candidate contains the current main repo root
  - count immediate child real git repositories
  - pick the nearest qualifying candidate
- Keep the existing exclusion for git worktree checkouts from `.git/worktrees/...`

### `cmd/ww/main.go`

- Verify `newManager()` still passes the correct workspace root and main repo root after the detection change
- Adjust only if workspace-root fallback handling for non-git roots needs to be updated

### Future-facing note

- Do not implement clone-mode support in this plan
- Preserve a clear seam for future FR-16 work so clone-based managed directories can be excluded from workspace-member detection via a managed marker such as `.ww-metadata`

## Test Changes

### `workspace/workspace_test.go`

- Add unit test: current repo inside a git parent workspace, with a grandparent that also contains multiple repos, still resolves to the parent workspace
- Add unit test: both parent and grandparent qualify; nearest containing candidate wins
- Add unit test: current directory already qualifies as a workspace root and is selected immediately
- Keep existing tests that ensure git worktree siblings do not trigger workspace detection

### `integration_test.go`

- Add integration test mirroring the real meta-repo shape:
  - grandparent contains multiple repos
  - parent is also a git-backed workspace root with multiple child repos
  - running `ww list` from the child repo resolves workspace mode and shows sibling repos
- Add integration test from a deeper path / worktree path if needed to confirm anchoring on `mainRoot`

## Sub-tasks

- [ ] Update workspace discovery spec to describe the B algorithm and tie-break rule
- [ ] Refactor `workspace.Detect()` to implement nearest-containing-root detection within the bounded window
- [ ] [parallel] Add unit tests for parent-vs-grandparent qualification and immediate workspace-root selection
- [ ] [parallel] Add integration coverage for the meta-repo case that currently falls back to single-repo mode
- [ ] [depends on: spec update, implementation, tests] Verify workspace-aware commands still behave correctly with the new root selection

## Verification

- `go test ./workspace`
- `go test ./...`
- Manual verification in a real layout similar to:
  - level 1: grandparent containing multiple repos
  - level 2: git-backed workspace root containing multiple child repos
  - level 3: child repo where `ww list` should resolve the level-2 workspace
- Confirm that worktree siblings are still ignored and do not create false workspace roots

## Expected Outcome

- Running `ww` from a child repo inside a practical 1-2-3 level meta-repo layout resolves the nearest containing workspace root instead of falling back to single-repo mode
- Detection remains deterministic, bounded, and compatible with the current "ignore git worktree siblings" invariant
