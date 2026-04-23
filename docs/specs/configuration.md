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
sandbox = false
```

## Fields

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `worktree_dir` | string | mode-dependent | Parent directory for worktrees. In workspace mode the default is `.worktrees`; in single-repo mode the default is sibling layout next to the repo. An explicit value overrides the default in both modes. |
| `default_base` | string | `""` | Base ref for new branches. Empty = auto-detect via `origin/HEAD`. When set, this is the authoritative base for both branch creation and status classification. When empty and `origin/HEAD` cannot be detected, `ww list` degrades to `unknown` status instead of failing. |
| `copy_files` | string[] | `[]` | Files/directories to deep-copy from main worktree to new worktrees. Missing sources are silently skipped; other errors emit a warning to stderr. |
| `symlink_files` | string[] | `[]` | Files/directories to symlink from main worktree to new worktrees. Missing sources are silently skipped; other errors emit a warning to stderr. |
| `post_create_hook` | string | `""` | Shell command to run in the new worktree directory after creation. Empty = no hook. |
| `sandbox` | bool | `false` | Constrain workspace/config discovery and single-repo worktree defaults to the current sandbox boundary. The `--sandbox` CLI flag takes precedence and enables sandbox mode even when this field is absent or false. |

## Trust Model

`.ww.toml` is treated as **trusted input**, the same trust model as `.gitconfig`. The `post_create_hook` value is passed directly to `sh -c` without sanitization because it is authored by the repository owner. Users should review `.ww.toml` before using an untrusted repository, just as they would review `.gitconfig` aliases.

## Config Search

1. Start from the current working directory.
2. Look for `.ww.toml` in the current directory.
3. If not found, move to the parent directory and repeat.
4. Stop at the filesystem root.
5. If not found via upward search, check caller-provided fallback directories (e.g., the main worktree's root directory or the detected workspace root).
6. If no file is found, use defaults.

### Sandbox Config Search

When sandbox mode is enabled by `--sandbox` or by an already-loaded `sandbox = true` config value:

1. Determine the sandbox boundary before loading the final config:
   - if the current working directory has immediate child git repositories, the boundary is the current working directory
   - otherwise, if the current working directory is inside git, the boundary is that repository's main working tree root
   - otherwise, config loading uses defaults and command setup returns `not a git repository`
2. Search from the current working directory upward, stopping at the sandbox boundary.
3. If the current working directory is a secondary worktree that is not a descendant of the main working tree root, the main working tree root may be checked as an explicit fallback because git defines it as the repository's primary root.
4. Other fallback directories outside the sandbox boundary are ignored.
5. If no config is found within those locations, use defaults.

## Worktree Path Layout

### Sibling layout (`worktree_dir = ""` in single-repo mode)

Worktrees are created as siblings of the repo directory:

```text
myapp/              # main repo
myapp@feat-auth/    # worktree
myapp@fix-bug/      # worktree
```

Path formula: `<repo-parent>/<repo-name>@<sanitized-branch>`

### Sandbox single-repo layout (`worktree_dir = ""`)

In sandbox single-repo mode, the default avoids parent-directory placement:

```text
myapp/                        # main repo
├── .worktrees/
│   ├── myapp@feat-auth/      # worktree
│   └── myapp@fix-bug/        # worktree
└── .ww.toml
```

Path formula: `<repo-root>/.worktrees/<repo-name>@<sanitized-branch>`

### Workspace layout (`worktree_dir = ".worktrees"` in workspace mode)

Worktrees are created under the specified directory:

```text
workspace/
├── repo/                   # main repo
├── .worktrees/
│   ├── repo@feat-auth/     # worktree
│   └── repo@fix-bug/       # worktree
└── .ww.toml
```

Path formula: `<worktree_dir>/<repo-name>@<sanitized-branch>`

Relative `worktree_dir` values are resolved against the active anchor: the workspace root in workspace mode, the repository parent in normal single-repo mode, or the repository root in sandbox single-repo mode. Relative values that escape the active anchor with `..` are rejected. Absolute values are used as explicit user intent and are not rejected solely for pointing outside the sandbox-friendly default area.

## Branch Name Sanitization for Paths

Branch names are sanitized for use in directory names:
- `/` is replaced with `-`

Example: `feat/my-feature` becomes `feat-my-feature` in directory names.
