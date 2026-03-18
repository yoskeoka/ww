# 007: Workspace Discovery + Path Layout

**Objective:** Implement the workspace detection algorithm and worktree path layout changes described in the Phase 2 design (see `docs/design-decisions/adr.md` for key decisions). This is the foundation for all Phase 2 features.

**Covers:** Workspace detection (design doc "Workspace Discovery" section), worktree path layout (design doc "Worktree Path Layout" section), CLI prerequisites relaxation (non-git workspace roots).

## Context

Phase 1 assumes `ww` runs inside a single git repo. Phase 2 adds workspace awareness: detecting when `ww` is inside a meta-repo (parent directory with multiple git children) and adjusting behavior accordingly.

The detection algorithm must handle:
- Current directory is a git repo with git siblings → parent is workspace root
- Current directory is non-git with git children → current directory is workspace root
- Current directory is a standalone git repo → single-repo mode (Phase 1 compatible)

When workspace mode is active, the default `worktree_dir` changes to `.worktrees` under the workspace root, centralizing all worktrees.

## Sub-tasks

- [ ] [parallel] **New package `workspace/`**: Create `workspace/workspace.go` with:
  - `Workspace` struct: `Root string`, `Repos []Repo`, `Mode` (workspace vs single-repo)
  - `Repo` struct: `Name string`, `Path string` (absolute path to repo root)
  - `Detect(startDir string) (*Workspace, error)` — implements the 7-step detection algorithm from design doc
  - Unit tests with `t.TempDir()` covering all detection scenarios (git parent, non-git parent with siblings, standalone repo, non-git with children, nested edge cases)
- [ ] [parallel] **Spec update `docs/specs/workspace-discovery.md`**: New spec documenting:
  - Detection algorithm (7 steps)
  - `Workspace` and `Repo` types
  - Worktree path layout per mode (workspace: `.worktrees/<repo>@<branch>`, single-repo: sibling)
  - Edge cases: non-git workspace root, child repos never become workspace roots
  - CLI prerequisites change: `ww` no longer requires being inside a git repo when workspace mode is detected
- [ ] [depends on: New package] **Update `worktree.Manager` path logic**: Modify `WorktreePath()` to accept workspace context and compute paths differently per mode:
  - Workspace mode default: `<workspace_root>/.worktrees/<repo>@<branch>`
  - Single-repo mode default: `<repo-parent>/<repo>@<branch>` (unchanged)
  - Explicit `worktree_dir` in `.ww.toml` overrides both
- [ ] [depends on: New package] **Integrate into CLI (`cmd/ww/main.go`)**: Update `newManager()` (or introduce a higher-level entry point) to run workspace detection before manager creation. Pass workspace context through to Manager.
- [ ] [depends on: Spec update] **Update `docs/specs/cli-commands.md`**: Relax prerequisites section — `ww` can run from a non-git directory if it's a workspace root.
- [ ] [depends on: Spec update] **Update `docs/specs/configuration.md`**: Document `worktree_dir` default change in workspace mode.
- [ ] [depends on: Integrate into CLI] **Update `docs/spec-code-mapping.md`**: Add rows for `workspace/` package and `docs/specs/workspace-discovery.md`.

## Code Changes

| File | Change |
|------|--------|
| `workspace/workspace.go` | New — workspace detection algorithm |
| `workspace/workspace_test.go` | New — unit tests for detection |
| `worktree/worktree.go` | Update `WorktreePath()` for workspace-aware path computation |
| `cmd/ww/main.go` | Integrate workspace detection into CLI startup |

## Spec Changes

| File | Change |
|------|--------|
| `docs/specs/workspace-discovery.md` | New — workspace detection and path layout spec |
| `docs/specs/cli-commands.md` | Relax prerequisites (non-git workspace root) |
| `docs/specs/configuration.md` | Document `worktree_dir` default per mode |
| `docs/spec-code-mapping.md` | Add `workspace/` mapping |

## Design Decisions

- **New `workspace/` package** (not inside `worktree/`): Workspace detection is a separate concern from worktree operations. Keeping it in its own package maintains the existing layered architecture (`git/` → `worktree/` → `cmd/ww/`), adding `workspace/` as a peer to `worktree/`.
- **`Detect()` uses `git rev-parse` via `git.Runner`**: Reuses existing git abstraction rather than duplicating `.git` detection logic.

## Verification

- `make test` passes
- `make lint` passes
- Unit tests cover all 7 detection steps + edge cases
- Existing Phase 1 integration tests still pass (single-repo mode unchanged)
