# Configuration Specification

## Overview

`ww` reads configuration from two optional layers:

- a user-owned global config file at `$XDG_CONFIG_HOME/ww/config.toml` when `XDG_CONFIG_HOME` is set, otherwise the current user's default home config location for `ww` (`.config/ww/config.toml` under the home directory)
- a repo-local `.ww.toml` file discovered from the current working directory

If no config file is found in either layer, sensible defaults are used (zero-config mode).

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

Global config uses the same TOML fields, but lives in `config.toml` at the global path above instead of repo-local `.ww.toml`.

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

Both repo-local `.ww.toml` and user-owned global `config.toml` are treated as **trusted input**, the same trust model as `.gitconfig`. The `post_create_hook` value is passed directly to `sh -c` without sanitization because it is authored by a trusted config owner. Users should review repository-local config before using an untrusted repository, just as they would review `.gitconfig` aliases.

## Config Search

### Global Config Search

1. If `XDG_CONFIG_HOME` is set, the global config path is `$XDG_CONFIG_HOME/ww/config.toml`.
2. Otherwise, the global config path is the current user's default home config location for `ww`: `.config/ww/config.toml` under the home directory.
3. If that file does not exist, the global layer is absent.

### Repo-Local Config Search

1. Start from the current working directory.
2. Look for `.ww.toml` in the current directory.
3. If not found, move to the parent directory and repeat.
4. Stop at the filesystem root.
5. If not found via upward search, check caller-provided fallback directories (for example, the main worktree's root directory or the detected workspace root).
6. If no file is found, the repo-local layer is absent.

### Layering and Precedence

When both config layers are present:

1. Load the global config first.
2. Load the repo-local config second.
3. Resolve values per configuration field, not per file:
   - if a field is defined only in global config, use the global value
   - if a field is defined only in repo-local config, use the repo-local value
   - if a field is defined in both, the repo-local value replaces the global value completely
4. Replacement uses the same rule for every field type:
   - scalar fields replace scalar fields
   - array fields replace the entire array without append or merge behavior
   - hook/script fields replace the entire hook value without composition

There is no implicit deep merge, list append, or hook concatenation.

### Sandbox Config Search

When sandbox mode is enabled by `--sandbox` or by an already-loaded `sandbox = true` config value:

1. Determine the sandbox boundary before loading the final config:
   - if the current working directory has immediate child git repositories, the boundary is the current working directory
   - otherwise, if the current working directory is inside git, the boundary is that repository's main working tree root
   - otherwise, config loading uses defaults and command setup returns `not a git repository`
2. Search for repo-local `.ww.toml` from the current working directory upward, stopping at the sandbox boundary.
3. If the current working directory is a secondary worktree that is not a descendant of the main working tree root, the main working tree root may be checked as an explicit fallback because git defines it as the repository's primary root.
4. Other repo-local fallback directories outside the sandbox boundary are ignored.
5. Global config still uses its explicit user-owned path. Sandbox mode does not change the global config path or turn it into an upward search.
6. If neither config layer is found, use defaults.

### Sandboxed Agent Guidance

For agent sandboxes, prefer granting read access to the global config path while denying write access to that path and any referenced hook/config assets unless write access is intentionally required for the task. This keeps user-owned defaults available without making global config a mutable sandbox target.

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
