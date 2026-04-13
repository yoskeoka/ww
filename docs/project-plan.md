# Project Plan: ww (Workspace Worktree)

## Naming

Public command name is frozen as **ww** (workspace worktree) for the first release line starting at `v0.3.0`.

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

- **FR-1**: Detect workspace automatically by scanning parent and child directories for git repos. Support `workspace = true` in `.ww.toml` for explicit declaration.
- **FR-2**: Create a worktree for a single repo (`ww create <branch>`). Support `--repo` flag to target any repo in the workspace.
- **FR-3**: List all worktrees across the workspace (`ww list`). Show REPO and STATUS columns. Support `--cleanable` filter for `merged`/`stale` worktrees.
- **FR-4**: Remove a worktree from a single repo (`ww remove <branch>`). Support `--repo` flag to target any repo in the workspace.
- **FR-5**: Copy/symlink gitignored files (`.env`, IDE configs) into new worktrees automatically, configured per-repo.
- **FR-6**: Run post-create hooks (e.g., dependency install) per-repo.

#### Enhanced (Phase 2)

- **FR-7**: STATUS column in `ww list` — `merged` (branch merged into base), `stale` (remote tracking set but remote branch gone + unmerged), `active` (neither).
- **FR-8**: Clean merged/stale worktrees in bulk (`ww clean`). Safe delete by default, `--force` for dirty worktrees.
- **FR-9**: Single-repo mode — when no workspace is detected, `ww` works on the current repo only (Phase 1 compatible).
- **FR-10**: Shell integration — output that enables `cd` into created worktrees (e.g., `cd $(ww create feat/x)`).

#### Post-Phase 2

- **FR-26**: `--no-upward-search` flag (or `.ww.toml` equivalent) — disable upward search for workspace detection and config discovery. In sandboxed environments (e.g., Claude Code), the process may not have permission to read parent directories or scan siblings. This flag constrains `ww` to the current repo only, skipping parent directory walks for `.ww.toml` config, workspace root detection, and sibling repo scanning. Implementation is small: skip the upward walk and sibling enumeration in the existing workspace discovery and config search logic.

#### Future

- **FR-16**: Alternative isolation via `git clone --reference --dissociate` instead of `git worktree add`. Useful when full independence from the main repo is needed (e.g., AI agent orchestrators running long tasks). Configurable per-repo or per-command flag. To avoid clone-based workspaces being misdetected as real workspace member repos, `ww`-managed clones should carry an explicit managed marker such as `.ww-metadata`.
- **FR-17**: Lifecycle hooks beyond post-create — support `pre-create`, `post-create`, `pre-remove`, and `post-remove` hooks per-repo. Enables container orchestration (e.g., `docker compose up` on create, DB cleanup + `docker compose down` on remove).
- **FR-18**: Inject environment variables into hooks — `WW_BRANCH`, `WW_WORKTREE_PATH`, `WW_REPO_NAME`, `WW_WORKTREE_INDEX` (numeric, for port offset derivation). Enables worktree-aware compose files without hardcoding.
- **FR-19**: Multi-repo batch worktree operations — `ww create feat/x --repos ai-arena,ww` to create worktrees across multiple repos simultaneously. Useful when child repos have dependencies on each other.
- **FR-20**: `ww cd` — shell navigation between worktrees and workspace root.
- **FR-21**: Child repo `.ww.toml` override — child repos can override workspace-level `copy_files`, `post_create_hook` etc.
- **FR-22**: Recursive workspace detection — respect `workspace = true` in child repos to support nested workspace structures.
- **FR-23**: Time-based stale detection — mark worktrees as stale after N days since last commit. Configurable via `--stale-days`.
- **FR-24**: Human interactive mode — provide a guided mode for people using `ww` directly, including interactive repo/branch selection, preview-oriented create/remove/clean flows, and confirmation for destructive actions without requiring shell composition or raw flag memorization.
- **FR-25**: Sandboxed environment compatibility — enable `ww` to operate fully within filesystem-sandboxed AI agent environments (e.g., Claude Code). The core issue is that `git worktree add` fails on repos with submodules because the sandbox blocks creation of `.gitmodules` and writes to `.git/config` (`Operation not permitted`). This is not a `ww` bug but an interaction between the sandbox's filesystem restrictions and git's internal operations. **Precondition for work**: upstream Claude Code sandbox issues are resolved or root cause is definitively identified — [Issue #13195](https://github.com/anthropics/claude-code/issues/13195) (`.git/config` write blocked), [Issue #21942](https://github.com/anthropics/claude-code/issues/21942) (`com.apple.provenance` xattr causing EPERM). FR-26 (`--no-upward-search`) addresses one aspect (parent directory access), but the `.gitmodules`/`.git/config` write failures are outside `ww`'s control. See `docs/issues/sandbox-worktree-compatibility.md` for full investigation and reference links.

#### Agent-Friendly CLI Design

- **FR-11**: `--dry-run` flag for mutation commands (create, remove, clean) — validate and show what would happen without executing.
- **FR-12**: `--json` flag on all commands — output NDJSON (one JSON object per line) for stream-friendly machine consumption.
- **FR-13**: `--fields` flag to limit output fields (e.g., `ww list --json --fields path,branch,dirty`), reducing context window cost for AI agents.
- **FR-14**: `ww schema <command>` — runtime introspection exposing available params, flags, and types as JSON. Agents discover capabilities without parsing `--help`.
- **FR-15**: Ship agent skill files (e.g., `.claude/skills/ww-operator`) encoding invariants agents cannot infer from help text (e.g., "always use `--dry-run` before mutations").

### Non-Functional Requirements

- **NFR-1**: Written in Go. Single static binary, no runtime dependencies.
- **NFR-2**: Fast — worktree creation for a single repo should add negligible overhead over raw `git worktree add`.
- **NFR-3**: Git operations use `git` CLI internally (not a Go git library) for maximum compatibility.
- **NFR-4**: Configuration via a simple file (TOML or YAML) in the workspace root.
- **NFR-5**: Works on macOS and Linux. Windows is not a priority.
- **NFR-6**: Installable via `go install` and Homebrew.
- **NFR-7**: Hardened input validation — reject invalid branch names, path traversals, control characters. Assume agent-generated inputs can be adversarial.

## Milestones

- [x] Phase 1 (MVP): Single-repo worktree management — create, list, remove with post-create hooks and gitignored file handling.
- [x] Phase 2: Workspace discovery (auto-detect parent/child git repos, `workspace = true`), cross-repo `ww list` with STATUS (`active`/`merged`/`stale`), `--cleanable` filter, `ww clean`, `--repo` flag for create/remove.
- [ ] Post-Phase 2: `--no-upward-search` flag for sandboxed environments (FR-26). Small scope, independent of Phase 2 workspace discovery.
- [x] Phase 3: Polish — shell integration (`ww cd`, `cd $(ww create feat/x)`), SemVer release automation starting at `v0.3.0`, Homebrew tap distribution, documentation.
- [x] Phase 4: Human interactive mode — add a people-first interactive flow for common operations such as create, list, remove, clean, and worktree selection without requiring users to remember the full flag surface.
- [ ] Phase 5 (nice-to-have): Hook trust hardening — first-run confirmation prompt, config change detection, sandbox execution, dangerous pattern warning.
- [ ] Future: Sandboxed environment full compatibility (FR-25). Blocked on upstream Claude Code sandbox issue resolution.

## Design Principles

1. **Git-native**: Use `git` CLI under the hood. Don't reimplement git behavior.
2. **Convention over configuration**: Sensible defaults (worktree path = `<repo>@<branch>`), minimal required config.
3. **Single-repo first**: Phase 1 must work perfectly in a single repo. Multi-repo is an extension, not a prerequisite.
4. **Composable**: Output machine-readable data (JSON flag) for scripting and AI agent integration.
5. **Agent-friendly by default**: Structured output, runtime schema introspection, dry-run safety, and hardened input validation. Design for both human and AI agent operators from day one.

## References

- [twig](https://github.com/708u/twig) — Best-in-class single-repo worktree UX, symlink-based config sharing (Go)
- [ha](https://github.com/kawarimidoll/ha) — Shell function approach, `repo@branch` flat path layout
- [gwq](https://github.com/d-kuro/gwq) — Global directory hierarchy, fzf integration (Go)
- [worktrunk](https://github.com/max-sixty/worktrunk) — `.worktreeinclude`, hook system (Rust)
- [workspace-manager](https://github.com/go-go-golems/workspace-manager) — Closest multi-repo precedent (Go)
- [Zenn: git worktree tools survey](https://zenn.dev/kawarimidoll/articles/9a77555122b3d5)
- [Zenn: twig introduction](https://zenn.dev/progate/articles/2e1e90796d82f0)
- [Rewrite Your CLI for AI Agents](https://justin.poehnelt.com/posts/rewrite-your-cli-for-ai-agents/) — Agent-friendly CLI design patterns (JSON payloads, schema introspection, dry-run, input validation)
