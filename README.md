# ww — Workspace Worktree Manager

A fast CLI tool for managing git worktrees across multiple repositories. Where existing tools handle single-repo worktree operations, `ww` fills the gap of **coordinated multi-repo worktree management** — creating, listing, and cleaning up worktrees across N repos from a single command.

## Install

### Homebrew

```sh
brew tap yoskeoka/ww
brew install ww
```

### Go

```sh
go install github.com/yoskeoka/ww/cmd/ww@latest
```

### From Source

```sh
git clone https://github.com/yoskeoka/ww.git
cd ww
go build -o ww ./cmd/ww
```

## Quick Start

`ww` works out of the box in any git repository — no configuration required.

### Create a worktree

```sh
ww create feat/my-feature
# Created worktree at /path/to/repo@feat-my-feature (branch: feat/my-feature)
```

This creates a new branch from `default_base` when configured, otherwise from `origin/HEAD`, and sets up a worktree for it. If the branch already exists, it checks out the existing branch.

To check out a branch that exists only on the remote, use the explicit Git-native path:

```sh
ww create --guess-remote feat/existing-pr-branch
```

This refreshes `origin` first, then asks Git to resolve and check out the same-named remote branch with `git worktree add --guess-remote`.

### List worktrees

```sh
ww list
# PATH                                BRANCH           HEAD     STATUS
# /path/to/repo (main worktree)      main             abc1234  active
# /path/to/repo@feat-my-feature      feat/my-feature  def5678  active
```

### Remove a worktree

```sh
ww remove feat/my-feature
# Removed worktree at /path/to/repo@feat-my-feature
# Deleted branch feat/my-feature
```

This removes the worktree directory and deletes the local branch. Use `--keep-branch` to keep the branch.

### Clean up stale worktrees

```sh
ww clean
```

Removes all worktrees whose branches are already merged or whose remote tracking branches no longer exist. Use `--dry-run` to preview what would be cleaned.

### Use interactive mode

```sh
ww i
```

Interactive mode provides a guided prompt flow for the most common human-facing operations:

- `create`: pick a repo and branch, preview the target path, then create
- `list`: browse existing worktrees and print the selected worktree path (equivalent to `ww cd`) for shell navigation, or remove them
- `clean`: review cleanable worktrees before deletion

Interactive mode is a thin wrapper over the standard CLI. It requires TTYs on stdin and stderr (stdout may be redirected) and is intended for people, not `--json` automation.

## Shell Integration

`ww` never changes your shell's current directory directly. Instead, it prints paths that shell wrappers or command substitution can consume.

### Navigate to a worktree with `ww cd`

`ww cd` prints the absolute path of a worktree. Combine it with a shell wrapper to navigate:

```sh
# Add to your .bashrc or .zshrc
wcd() {
  local dir
  dir="$(ww cd "$@")" && cd "$dir"
}
```

Then use it:

```sh
wcd feat/my-feature    # cd into the worktree for feat/my-feature
wcd                    # cd into the most recently created worktree
```

### Create and enter a worktree in one step

Use `ww create -q` (quiet mode) with command substitution:

```sh
cd "$(ww create -q feat/my-feature)"
```

Quiet mode suppresses human-readable output and prints only the created worktree path, making it ideal for shell composition.

## Workspace Mode

When `ww` detects multiple git repositories under a common parent directory, it automatically enters **workspace mode**. This lets you manage worktrees across all repos from a single command.

### Example workspace structure

```
my-workspace/
├── frontend/          # git repo
├── backend/           # git repo
├── shared-lib/        # git repo
└── .ww.toml           # optional workspace config
```

### Cross-repo operations

In workspace mode, `ww list` shows worktrees from all detected repos:

```sh
ww list
# REPO        PATH                                         BRANCH        HEAD     STATUS
# frontend    /path/to/my-workspace/frontend (main worktree)   main     abc1234  active
# backend     /path/to/my-workspace/backend (main worktree)    main     def5678  active
# shared-lib  /path/to/my-workspace/shared-lib (main worktree) main     789abcd  active
```

Use `--repo` to target a specific repo from anywhere in the workspace:

```sh
ww create feat/auth --repo backend
ww remove feat/auth --repo backend
ww cd feat/auth --repo backend
```

`ww clean` operates across all repos in the workspace automatically.

### Workspace worktree layout

Worktrees in workspace mode are created under `.worktrees/` at the workspace root:

```
my-workspace/
├── frontend/
├── backend/
├── .worktrees/
│   ├── frontend@feat-auth/
│   └── backend@feat-auth/
└── .ww.toml
```

## Configuration

`ww` is configured via a `.ww.toml` file. The file is discovered by searching upward from the current directory. If no file is found, sensible defaults are used.

```toml
# Parent directory for worktrees (default: mode-dependent)
worktree_dir = ".worktrees"

# Base ref for new branches (default: auto-detect via origin/HEAD)
default_base = "origin/main"

# Files to copy from main worktree to new worktrees
copy_files = [
    ".env",
    ".vscode/settings.json",
]

# Files to symlink from main worktree to new worktrees
symlink_files = [
    "node_modules",
]

# Shell command to run after worktree creation
post_create_hook = "npm install"
```

| Field | Default | Description |
|-------|---------|-------------|
| `worktree_dir` | mode-dependent | Parent directory for worktrees. Defaults to `.worktrees` in workspace mode, sibling layout in single-repo mode. |
| `default_base` | `""` (auto-detect) | Base ref for new branches. Empty means auto-detect via `origin/HEAD`. |
| `copy_files` | `[]` | Files to deep-copy from main worktree into new worktrees. Missing sources are silently skipped. |
| `symlink_files` | `[]` | Files to symlink from main worktree into new worktrees. Missing sources are silently skipped. |
| `post_create_hook` | `""` | Shell command run in the new worktree directory after creation. |

## Commands

| Command | Description |
|---------|-------------|
| `ww create <branch>` | Create a new worktree for a branch |
| `ww list` | List all worktrees (across workspace in workspace mode) |
| `ww remove <branch>` | Remove a worktree and delete its branch |
| `ww clean` | Remove all merged or stale worktrees |
| `ww i` | Start the interactive create/list/clean flow |
| `ww cd [branch]` | Print a worktree path for shell navigation |
| `ww version` | Print version information |

### Global flags

| Flag | Description |
|------|-------------|
| `--json` | Output as NDJSON (one JSON object per line) |
| `--dry-run` | Show planned actions without executing |
| `--version` | Print version and exit |

### Worktree status values

`ww list` classifies each worktree with a status:

| Status | Meaning |
|--------|---------|
| `active` | Main worktree, or a branch that is neither merged nor stale |
| `merged` | Branch is merged into the base branch |
| `stale` | Remote tracking branch no longer exists and branch is not merged |
| `unknown` | Base branch could not be determined |

`ww clean` removes worktrees with `merged` or `stale` status. `active` and `unknown` worktrees are never cleaned automatically.

## Machine-Readable Output

All commands support `--json` for scripting and AI agent integration:

```sh
ww list --json
# {"repo":"myapp","path":"/path/to/myapp","branch":"main","head":"abc1234","main":true,"status":"active"}
# {"repo":"myapp","path":"/path/to/myapp@feat-auth","branch":"feat/auth","head":"def5678","status":"merged"}
```

```sh
ww create feat/x --json
# {"path":"/path/to/myapp@feat-x","branch":"feat/x","created":true,"base":"origin/main"}
```

For detailed specifications, see [docs/specs/](docs/specs/).

## License

[MIT](LICENSE)
