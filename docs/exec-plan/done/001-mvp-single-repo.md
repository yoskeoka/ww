# 001: MVP — Single-Repo Worktree Management

## Objective

Implement Phase 1 of the project plan: a working `ww` CLI that manages git worktrees within a single repository. Covers FR-2 through FR-6, FR-11, FR-12, NFR-1 through NFR-5, and NFR-7.

After this plan, `ww` should be usable as a standalone single-repo worktree manager with basic agent-friendly output.

## Design Decisions

### Configuration format: TOML

Use TOML (`.ww.toml`) for workspace configuration. Rationale:
- Go has excellent TOML support (`BurntSushi/toml`)
- TOML is the standard for Go/Rust CLI tool configs (cargo, goreleaser)
- Simpler than YAML for flat key-value + tables

### Worktree path layout: `<repo>@<branch>` with configurable parent dir

Directory naming follows `ha`'s `@` convention. Slash in branch names replaced with `-`.

Parent directory is configurable via `worktree_dir` in `.ww.toml`:

- **Workspace mode** (`worktree_dir = ".worktrees"`): All worktrees under a single directory, easy to `.gitignore`.
  ```
  vibe-coding-workspace/
  ├── ww/                        # real repo
  ├── ai-arena/                  # real repo
  ├── .worktrees/                 # one .gitignore entry
  │   ├── ww@feat-auth/
  │   └── ai-arena@feat-auth/
  └── .ww.toml
  ```
- **Single-repo mode** (`worktree_dir` omitted or `""`): Flat sibling layout (ha-style).
  ```
  myapp/
  myapp@feat-auth/
  myapp@fix-bug/
  ```

Default: `""` (sibling layout). Zero-config for single-repo users.

### CLI framework: custom subcommand pattern + pflag

Based on `yoskeoka/go-templates/cli/subcommand` template. Uses a custom `command` struct with recursive subcommand dispatch. Replace `flag.FlagSet` with `pflag.FlagSet` for POSIX-style `--flag` support. No cobra dependency.

### Git execution: shell out to `git`

Use `os/exec` to call `git` directly (NFR-3). Wrap in a thin internal package for testability.

### Branch creation: always from configurable default

Default to creating worktree branches from `origin/HEAD` (auto-detected default branch). Configurable via `default_base` in `.ww.toml`.

## Spec Changes

Create initial specs:

- `docs/specs/cli-commands.md` — Command interface (create, list, remove), flags, output formats
- `docs/specs/configuration.md` — `.ww.toml` schema and defaults
- `docs/specs/git-operations.md` — How ww wraps git worktree/branch operations

## Sub-tasks

### 1. Project scaffolding + CI
- [ ] [parallel] Initialize Go module (`go mod init github.com/yoskeoka/ww`)
- [ ] [parallel] Set up project structure: `cmd/ww/main.go`, `internal/git/`, `internal/config/`, `internal/worktree/`
- [ ] [parallel] Add `.gitignore` for Go binaries
- [ ] [parallel] Create spec files in `docs/specs/`
- [ ] [parallel] GitHub Actions CI: `go test ./...` on push and PR, Makefile with `build`/`test` targets

### 2. Core: git wrapper
- [ ] [depends on: scaffolding] `internal/git/` — thin wrapper around `git` CLI execution
  - `RunGit(args ...string) (stdout, stderr, error)`
  - `WorktreeAdd(path, branch, base string) error` — new branch: `git worktree add -b <branch> <path> <base>`; existing branch: `git worktree add <path> <branch>`
  - `WorktreeList() ([]Worktree, error)`
  - `WorktreeRemove(path string) error`
  - `BranchDelete(branch string) error`
  - `DefaultBranch() (string, error)`

### 3. Core: configuration
- [ ] [depends on: scaffolding] `internal/config/` — load `.ww.toml`
  - Config struct: `worktree_dir`, `default_base`, `copy_files []string`, `symlink_files []string`, `post_create_hook string`
  - Search upward from CWD to find `.ww.toml`
  - Sensible defaults when no config file exists (zero-config single-repo usage)

### 4. Core: worktree operations
- [ ] [depends on: git wrapper, configuration] `internal/worktree/` — business logic
  - `Create(branch string, opts CreateOpts) (WorktreeInfo, error)` — create worktree, copy/symlink files, run hook
  - `List() ([]WorktreeInfo, error)` — list worktrees with metadata
  - `Remove(branch string, opts RemoveOpts) error` — remove worktree + optionally delete branch

### 5. Core: input validation (NFR-7)
- [ ] [parallel with task 4] `internal/validate/` — branch name and path validation
  - Reject path traversals (`../`, absolute paths outside workspace)
  - Reject control characters (ASCII < 0x20)
  - Validate branch names against `git check-ref-format` rules

### 6. CLI: command wiring
- [ ] [depends on: worktree operations, validation] `cmd/ww/main.go` — CLI entry point based on `yoskeoka/go-templates/cli/subcommand` pattern with `pflag`
  - Custom `command` struct with `pflag.FlagSet` per subcommand
  - `globalOpts` carries `--json` and `--dry-run` flags, plus `io.Writer` for output
  - `ww create <branch>` — create worktree
  - `ww list` — list worktrees
  - `ww remove <branch>` — remove worktree
  - `ww version` — print version

### 7. File operations: copy and symlink
- [ ] [depends on: worktree operations] Implement `copy_files` and `symlink_files` from config
  - Copy: deep copy listed files/dirs from main worktree to new worktree
  - Symlink: create symlinks for listed files/dirs
  - Skip silently if source doesn't exist (not an error)

### 8. Post-create hook
- [ ] [depends on: worktree operations] Implement `post_create_hook` from config
  - Execute shell command in new worktree's directory
  - Pass `WW_BRANCH` and `WW_WORKTREE_PATH` as env vars (early FR-18 subset)
  - Stream stdout/stderr to user
  - Non-zero exit = warning (don't fail the create)

### 9. Output formatting
- [ ] [depends on: CLI wiring] Implement `--json` flag (FR-12)
  - Default: human-readable table/text output
  - `--json`: NDJSON, one object per line
  - `--dry-run`: show planned actions without executing (FR-11)

### 10. Tests
- [ ] [parallel, ongoing] Unit tests for each `internal/` package (every task from #2 onward includes tests)
- [ ] [depends on: CLI wiring] Integration tests — create temp git repo, run `ww` commands, verify state

## Out of Scope (deferred to later plans)

- Multi-repo workspace coordination (Phase 2)
- `ww clean` (FR-8)
- `ww schema` (FR-14)
- `--fields` flag (FR-13)
- Shell `cd` integration (FR-10)
- Agent skill files (FR-15)
- Homebrew formula (NFR-6)
- Clone-based isolation (FR-16)
- Full lifecycle hooks (FR-17)

## Verification

- [ ] `ww create feat/test` creates a new branch + worktree together (1 command = 1 branch + 1 worktree)
- [ ] `ww create feat/test` when branch `feat/test` already exists: checks out the existing branch as a worktree (no new branch)
- [ ] `ww list` shows the created worktree with path and branch
- [ ] `ww remove feat/test` removes worktree and branch
- [ ] `ww list --json` outputs valid NDJSON
- [ ] `ww create --dry-run feat/test` shows plan without creating anything
- [ ] Config-defined `copy_files` are present in new worktree
- [ ] Config-defined `symlink_files` are symlinked in new worktree
- [ ] `post_create_hook` runs after worktree creation
- [ ] Invalid branch names are rejected with clear error message
- [ ] Works with zero config (no `.ww.toml`)
