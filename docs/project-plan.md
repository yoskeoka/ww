# Project Plan: ww (Workspace Worktree)

## Goal

Build a fast, portable CLI tool (`ww`) that manages git worktrees across multiple repositories in a meta-repo workspace. Where existing tools handle single-repo worktree management well, `ww` fills the gap of **coordinated multi-repo worktree operations** — creating, listing, and cleaning up worktrees across N repos from a single command.

## Significance

### Problem

When working in a meta-repo environment with many child repositories, parallel development (feature branches, bug fixes, AI agent sessions) requires frequent git worktree operations. Current pain points:

1. **Repetitive setup**: Each new worktree needs .gitignore'd files copied, dependencies installed, and configs applied — multiplied by the number of repos involved.
2. **No multi-repo coordination**: Existing tools (worktrunk, gwq, wtp, twig, ha) only manage worktrees within a single repository. Nobody orchestrates worktrees across a workspace of repos.
3. **Non-portable workflows**: Developers who use meta-repo patterns across personal and work projects must re-create the same worktree management scripts in each environment.

### Value

- **Speed**: Compiled Go binary with deterministic behavior — worktree creation across repos completes in seconds.
- **Single pane of glass**: One command to see all active worktrees across all managed repos.
- **Portability**: A single binary that works in any meta-repo workspace. Bring the tool, not the scripts.
- **AI-agent friendly**: Designed for workflows where multiple AI agents work on different branches simultaneously.

### Competitive Landscape

| Category | Existing tools | Multi-repo? |
|----------|---------------|-------------|
| Single-repo worktree CLI | worktrunk (Rust), gwq (Go), wtp (Go), twig (Go), ha (Shell) | No |
| Multi-repo batch ops | gita (Python), meta (JS), Google repo (Python) | No worktree support |
| Multi-repo + worktree | workspace-manager (Go, niche) | Yes, but limited adoption |

`ww` targets the unserved intersection: multi-repo worktree coordination with modern single-repo UX.

## Requirements

### Functional Requirements

#### Core (MVP)

- **FR-1**: Manage a workspace configuration that lists multiple git repositories.
- **FR-2**: Create worktrees across one or more repos with a single command (`ww create <branch> [--repos repo1,repo2,...]`).
- **FR-3**: List all active worktrees across all managed repos (`ww list`).
- **FR-4**: Remove worktrees across repos with cleanup (`ww remove <branch>`).
- **FR-5**: Copy/symlink gitignored files (`.env`, IDE configs) into new worktrees automatically, configured per-repo.
- **FR-6**: Run post-create hooks (e.g., dependency install) per-repo.

#### Enhanced

- **FR-7**: Status overview — show branch state (ahead/behind, dirty) across all worktrees.
- **FR-8**: Clean merged/stale worktrees in bulk (`ww clean`).
- **FR-9**: Single-repo mode — `ww` should work fine in a standalone repo, not only in meta-repo contexts.
- **FR-10**: Shell integration — output that enables `cd` into created worktrees (e.g., `cd $(ww create feat/x)`).

### Non-Functional Requirements

- **NFR-1**: Written in Go. Single static binary, no runtime dependencies.
- **NFR-2**: Fast — worktree creation for a single repo should add negligible overhead over raw `git worktree add`.
- **NFR-3**: Git operations use `git` CLI internally (not a Go git library) for maximum compatibility.
- **NFR-4**: Configuration via a simple file (TOML or YAML) in the workspace root.
- **NFR-5**: Works on macOS and Linux. Windows is not a priority.
- **NFR-6**: Installable via `go install` and Homebrew.

## Milestones

- [ ] Phase 1 (MVP): Single-repo worktree management — create, list, remove with post-create hooks and gitignored file handling. Validate core UX.
- [ ] Phase 2: Multi-repo coordination — workspace config, create/list/remove across repos, unified status view.
- [ ] Phase 3: Polish — `ww clean`, shell integration (`cd` support), Homebrew formula, documentation.

## Design Principles

1. **Git-native**: Use `git` CLI under the hood. Don't reimplement git behavior.
2. **Convention over configuration**: Sensible defaults (worktree path = `<repo>@<branch>`), minimal required config.
3. **Single-repo first**: Phase 1 must work perfectly in a single repo. Multi-repo is an extension, not a prerequisite.
4. **Composable**: Output machine-readable data (JSON flag) for scripting and AI agent integration.

## References

- [twig](https://github.com/708u/twig) — Best-in-class single-repo worktree UX, symlink-based config sharing (Go)
- [ha](https://github.com/kawarimidoll/ha) — Shell function approach, `repo@branch` flat path layout
- [gwq](https://github.com/d-kuro/gwq) — Global directory hierarchy, fzf integration (Go)
- [worktrunk](https://github.com/max-sixty/worktrunk) — `.worktreeinclude`, hook system (Rust)
- [workspace-manager](https://github.com/go-go-golems/workspace-manager) — Closest multi-repo precedent (Go)
- [Zenn: git worktree tools survey](https://zenn.dev/kawarimidoll/articles/9a77555122b3d5)
- [Zenn: twig introduction](https://zenn.dev/progate/articles/2e1e90796d82f0)
