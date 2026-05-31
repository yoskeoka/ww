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

Status legend: `implemented` = shipped in current mainline behavior, `partial` = some but not all promised scope shipped, `planned` = intentionally not started yet, `blocked` = intentionally deferred due to external constraints.

### Functional Requirements

#### Core (MVP)

- **FR-1** (`partial`): Detect workspace automatically by scanning bounded parent/child directories for git repos. Zero-config detection is shipped, but explicit workspace declaration via `workspace = true` in config is not yet implemented.
- **FR-2** (`implemented`): Create a worktree for a single repo (`ww create <branch>`). Support `--repo` to target any detected repo in the workspace.
- **FR-3** (`implemented`): List all worktrees across the workspace (`ww list`). Show REPO and STATUS columns. Support `--cleanable` filtering for `merged`/`stale` worktrees.
- **FR-4** (`implemented`): Remove a worktree from a single repo (`ww remove <branch>`). Support `--repo` to target any detected repo in the workspace.
- **FR-5** (`implemented`): Copy/symlink selected gitignored files (`.env`, IDE configs, dependency directories) into new worktrees automatically via config.
- **FR-6** (`implemented`): Run a post-create hook (for example, dependency install) from config after worktree creation.

#### Enhanced (Phase 2)

- **FR-7** (`implemented`): STATUS column in `ww list` — `merged` (branch merged into base), `stale` (remote tracking set but remote branch gone + unmerged), `active` (neither).
- **FR-8** (`implemented`): Clean merged/stale worktrees in bulk (`ww clean`). Safe delete by default, `--force` for dirty worktrees.
- **FR-9** (`implemented`): Single-repo mode — when no workspace is detected, `ww` works on the current repo only.
- **FR-10** (`implemented`): Shell integration — output that enables `cd` into created or existing worktrees (`ww create -q`, `ww cd`).

#### Post-Phase 2

Post-Phase 2 (originally tracked as `--no-upward-search`) is complete. The planned follow-up landed via the sandbox implementation, so this item is no longer an open phase. Any remaining sandbox-related refinements should be tracked as separate follow-up tasks rather than under Post-Phase 2.

- **FR-26** (`implemented`): Sandbox-constrained mode (`--sandbox` flag or `sandbox = true` in `.ww.toml`) — completed via the sandbox implementation. `ww` can operate in filesystem-sandboxed environments that cannot reliably read or use parent directories by skipping parent/grandparent containing workspace detection, skipping parent-based sibling scans, limiting config lookup to the active sandbox boundary, and using repo-local `.worktrees` placement for single-repo defaults. It still supports current-directory workspace roots by scanning immediate child repositories, so `--repo` remains available when the user starts at a bounded workspace root.

#### Future

- **FR-16** (`planned`): Alternative isolation via `git clone --reference --dissociate` instead of `git worktree add`. Useful when full independence from the main repo is needed (for example, long-running AI agent tasks). To avoid clone-based workspaces being misdetected as real workspace member repos, `ww`-managed clones should carry an explicit managed marker such as `.ww-metadata`.
- **FR-17** (`partial`): Lifecycle hooks beyond post-create — `post_create_hook` is shipped, but `pre-create`, `pre-remove`, and `post-remove` are not yet implemented.
- **FR-18** (`partial`): Inject environment variables into hooks — `WW_BRANCH` and `WW_WORKTREE_PATH` are shipped today; `WW_REPO_NAME` and `WW_WORKTREE_INDEX` are not yet implemented.
- **FR-19** (`planned`): Multi-repo batch worktree operations — `ww create feat/x --repos ai-arena,ww` to create worktrees across multiple repos simultaneously.
- **FR-20** (`implemented`): `ww cd` — shell navigation between worktrees and workspace root.
- **FR-21** (`planned`): Child repo `.ww.toml` override — child repos can override workspace-level `copy_files`, `post_create_hook`, and related settings.
- **FR-22** (`planned`): Recursive workspace detection — respect nested workspace structures beyond the current bounded model.
- **FR-23** (`planned`): Time-based stale detection — mark worktrees as stale after N days since last commit. Configurable via `--stale-days`.
- **FR-24** (`implemented`): Human interactive mode — provide a guided mode for people using `ww` directly, including interactive repo/branch selection, preview-oriented create/remove/clean flows, and confirmation for destructive actions without requiring shell composition or raw flag memorization.
- **FR-25** (`blocked`): Sandboxed environment full compatibility — enable `ww` to operate end-to-end within filesystem-sandboxed AI agent environments (for example, Claude Code) even when the underlying git worktree flow touches `.git/config`, `.gitmodules`, submodules, or platform-specific sandbox artifacts. `ww` already addresses bounded discovery via FR-26, but fully reliable compatibility still depends partly on external sandbox behavior. Claude Code Issue #13195 is now closed, so it should no longer be treated as an active open blocker by itself; however, closing that issue does not yet prove that all `git worktree add` cases needed by `ww` are solved across real sandboxed repos. Track this as a verification-and-gap item rather than as purely internal feature work.
- **FR-27** (`planned`): Global config file support — allow `ww` to load user-owned global configuration such as `$HOME/.ww/config.toml` so teams or individuals can avoid committing `.ww.toml` into every target repository.
- **FR-28** (`planned`): Global project-target matching — allow global config to select per-project settings using the main worktree as the stable target anchor, so similar repository layouts or tech stacks can share one centrally maintained config set.
- **FR-29** (`planned`): Hook-driven materialization profiles — allow global config to define reusable copy/symlink/setup profiles for sandbox-friendly worktree setup, including patterns such as linking `.env`, dependency directories, or tool caches from the main worktree when direct copying is disallowed or expensive.
- **FR-30** (`planned`): Safe global-config guidance for sandboxed agents — document recommended permission patterns for `$HOME/.ww/*` (for example, allow reads but deny writes for agent sandboxes) so global hook/config workflows reduce prompt-injection and config-tampering risk.

#### Agent-Friendly CLI Design

- **FR-11** (`implemented`): `--dry-run` for mutation commands (create, remove, clean) — validate and show what would happen without executing.
- **FR-12** (`partial`): `--json` on standard non-interactive commands — machine-readable output is shipped broadly, but interactive mode intentionally rejects `--json` and the exact shape is JSON or NDJSON depending on command.
- **FR-13** (`planned`): `--fields` to limit output fields (for example, `ww list --json --fields path,branch,dirty`), reducing context window cost for AI agents.
- **FR-14** (`planned`): `ww schema <command>` — runtime introspection exposing available params, flags, and types as JSON. Agents discover capabilities without parsing `--help`.
- **FR-15** (`planned`, low priority): Ship optional agent skill files or equivalent packaged guidance for environments that need stronger operator conventions than built-in help and examples provide.

### Non-Functional Requirements

- **NFR-1** (`implemented`): Written in Go. Distributed as a single binary with no bundled runtime dependencies beyond the host tooling it intentionally invokes, chiefly `git` and an optional shell for configured hooks.
- **NFR-2** (`partial`): Fast — the tool remains lightweight in normal use, but the project plan does not yet carry explicit benchmark-backed proof for every command path.
- **NFR-3** (`implemented`): Git operations use the `git` CLI internally (not a Go git library) for maximum compatibility.
- **NFR-4** (`partial`): Configuration via a simple file. Repo-local TOML config is shipped today; future work should extend this with optional global config rather than replacing the simple-file model.
- **NFR-5** (`implemented`): Works on macOS and Linux. Windows is not a priority.
- **NFR-6** (`implemented`): Installable via `go install` and Homebrew.
- **NFR-7** (`implemented`): Hardened input validation — reject invalid branch names, path traversals, and control characters. Assume agent-generated inputs can be adversarial.

## Milestones

- [x] Phase 1 (MVP): Single-repo worktree management — create, list, remove with post-create hooks and gitignored file handling.
- [x] Phase 2: Workspace discovery (bounded auto-detect for practical parent/child git repo layouts), cross-repo `ww list` with STATUS (`active`/`merged`/`stale`), `--cleanable` filter, `ww clean`, `--repo` flag for create/remove.
- [x] Post-Phase 2: sandbox-constrained mode for sandboxed environments (FR-26). Originally tracked as `--no-upward-search`, and completed via the sandbox implementation. Any further sandbox refinements should be handled as separate follow-up tasks rather than this phase.
- [x] Phase 3: Polish — shell integration (`ww cd`, `cd $(ww create feat/x)`), SemVer release automation starting at `v0.3.0`, Homebrew tap distribution, documentation.
- [x] Phase 4: Human interactive mode — add a people-first interactive flow for common operations such as create, list, remove, clean, and worktree selection without requiring users to remember the full flag surface.
- [ ] Phase 5: Global config and hook workflow portability — add user-owned global config, per-project target matching, reusable hook/materialization profiles, and safe sandbox guidance so agent workflows do not require committing `.ww.toml` into every repository.
- [ ] Phase 6: Lifecycle hook expansion — extend today's post-create-only model with broader hook phases and richer hook context so setup/teardown workflows remain expressible without ad hoc wrapper scripts.
- [ ] Phase 7 (hardening): Hook trust hardening — once the hook/config surface is powerful enough, add first-run confirmation, config-change detection, sandbox execution options, and dangerous-pattern warnings.
- [ ] Future: Sandboxed environment full compatibility (FR-25). Treat this as a verification-and-gap track across real Claude Code sandbox cases, not as a single internal feature that `ww` can finish in isolation.

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
