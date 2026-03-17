# Configuration Specification

## Overview

`ww` reads configuration from a `.ww.toml` file. The file is located by searching upward from the current working directory. If no file is found, sensible defaults are used (zero-config mode).

## File Format

TOML format. Example:

```toml
worktree_dir = ".worktrees"
default_base = "origin/main"

copy_files = [
    ".env",
    ".vscode/settings.json",
]

symlink_files = [
    "node_modules",
]

post_create_hook = "npm install"
```

## Fields

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `worktree_dir` | string | `""` | Parent directory for worktrees. Empty = sibling layout (worktrees created next to the repo). Non-empty = all worktrees under this directory. |
| `default_base` | string | `""` | Base ref for new branches. Empty = auto-detect via `origin/HEAD`. |
| `copy_files` | string[] | `[]` | Files/directories to deep-copy from main worktree to new worktrees. Missing sources are silently skipped. |
| `symlink_files` | string[] | `[]` | Files/directories to symlink from main worktree to new worktrees. Missing sources are silently skipped. |
| `post_create_hook` | string | `""` | Shell command to run in the new worktree directory after creation. Empty = no hook. |

## Trust Model

`.ww.toml` is treated as **trusted input**, the same trust model as `.gitconfig`. The `post_create_hook` value is passed directly to `sh -c` without sanitization because it is authored by the repository owner. Users should review `.ww.toml` before using an untrusted repository, just as they would review `.gitconfig` aliases.

## Config Search

1. Start from the current working directory.
2. Look for `.ww.toml` in the current directory.
3. If not found, move to the parent directory and repeat.
4. Stop at the filesystem root.
5. If not found via upward search, check caller-provided fallback directories (e.g., the main worktree's root directory).
6. If no file is found, use defaults.

## Worktree Path Layout

### Sibling layout (default, `worktree_dir = ""`)

Worktrees are created as siblings of the repo directory:

```
myapp/              # main repo
myapp@feat-auth/    # worktree
myapp@fix-bug/      # worktree
```

Path formula: `<repo-parent>/<repo-name>@<sanitized-branch>`

### Workspace layout (`worktree_dir = ".worktrees"`)

Worktrees are created under the specified directory:

```
workspace/
├── repo/                   # main repo
├── .worktrees/
│   ├── repo@feat-auth/     # worktree
│   └── repo@fix-bug/       # worktree
└── .ww.toml
```

Path formula: `<worktree_dir>/<repo-name>@<sanitized-branch>`

## Branch Name Sanitization for Paths

Branch names are sanitized for use in directory names:
- `/` is replaced with `-`

Example: `feat/my-feature` becomes `feat-my-feature` in directory names.
