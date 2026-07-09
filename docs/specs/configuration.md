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

pre_create_hook = "npm run pre-create"
post_create_hook = "npm install"
pre_remove_hook = "npm run pre-remove"
post_remove_hook = "npm run post-remove"
sandbox = false
```

Global config uses the same TOML fields, but lives in `config.toml` at the global path above instead of repo-local `.ww.toml`.

Global config may also declare ordered project-specific overrides:

```toml
worktree_dir = ".worktrees"

[materialization_profiles.dev_setup]
copy_files = [".env"]
symlink_files = ["node_modules"]
post_create_hook = "make setup"

[[projects]]
root = "/home/user/src/workspace/ww"
materialization_profile = "dev_setup"

[[projects]]
root_prefix = "/home/user/src/workspace"
default_base = "origin/main"
```

## Fields

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `worktree_dir` | string | mode-dependent | Parent directory for worktrees. In workspace mode the default is `.worktrees`; in single-repo mode the default is sibling layout next to the repo. An explicit value overrides the default in both modes. |
| `default_base` | string | `""` | Base ref for new branches. Empty = auto-detect via `origin/HEAD`. When set, this is the authoritative base for both branch creation and status classification. When empty and `origin/HEAD` cannot be detected, `ww list` degrades to `unknown` status instead of failing. |
| `copy_files` | string[] | `[]` | Files/directories to deep-copy from main worktree to new worktrees. Missing sources are silently skipped; other errors emit a warning to stderr. |
| `symlink_files` | string[] | `[]` | Files/directories to symlink from main worktree to new worktrees. Missing sources are silently skipped; other errors emit a warning to stderr. |
| `pre_create_hook` | string | `""` | Shell command to run before a worktree is created. Empty = no hook. |
| `post_create_hook` | string | `""` | Shell command to run in the new worktree directory after creation. Empty = no hook. |
| `pre_remove_hook` | string | `""` | Shell command to run before a worktree is removed. Empty = no hook. |
| `post_remove_hook` | string | `""` | Shell command to run after a worktree has been removed and branch cleanup has been attempted. Empty = no hook. |
| `materialization_profile` | string | `""` | Select one named global materialization profile and expand it into `copy_files`, `symlink_files`, and `post_create_hook` for this config layer. Empty = no profile selection. |
| `sandbox` | bool | `false` | Constrain workspace/config discovery and single-repo worktree defaults to the current sandbox boundary. The `--sandbox` CLI flag takes precedence and enables sandbox mode even when this field is absent or false. |

## Materialization Profiles

Global config may define reusable materialization profiles under `[materialization_profiles.<name>]`.

Each profile may contain only these fields:

| Field | Type | Description |
|-------|------|-------------|
| `copy_files` | string[] | Replacement value for `copy_files` when the profile is selected |
| `symlink_files` | string[] | Replacement value for `symlink_files` when the profile is selected |
| `post_create_hook` | string | Replacement value for `post_create_hook` when the profile is selected |

Profile definitions are global-config-only. Repo-local `.ww.toml` files may reference a named profile with `materialization_profile`, but they cannot define new profiles.

Profile selection is allowed in:

- the base global config
- ordered `[[projects]]` blocks in the global config
- repo-local `.ww.toml`

The selected profile expands into ordinary materialization fields before command execution. Expansion is configuration sugar only; runtime worktree behavior still consumes the resolved `copy_files`, `symlink_files`, and `post_create_hook` values exactly as if they had been written directly in the selected config layer.

Within a single config layer, `materialization_profile` must not be combined with direct `copy_files`, `symlink_files`, or `post_create_hook` values. That combination is rejected during config loading to avoid hidden composition.

Profile lookup is single-select only. `ww` does not stack or merge multiple profiles in one load.

## Global Project Blocks

Global config may contain zero or more `[[projects]]` blocks. Each block adds project-specific overrides on top of the base global config.

Each `[[projects]]` block must define exactly one targeting field:

| Field | Type | Description |
|-------|------|-------------|
| `root` | string | Exact absolute path of the repository main worktree root to match. |
| `root_prefix` | string | Absolute path prefix of the repository main worktree root to match. Prefix matching is path-segment aware, so `/workspace/repo` does not match `/workspace/repo2`. |

Project blocks may also use any ordinary config fields from the table above. Those fields use the same full-replacement semantics as any other config layer.

Invalid project blocks are rejected during config loading. Examples of invalid blocks include:

- both `root` and `root_prefix` are set
- neither `root` nor `root_prefix` is set
- the selected targeting value is empty or not an absolute path

Invalid materialization profile configuration is also rejected during config loading. Examples include:

- `materialization_profile` names a profile that does not exist in the global config
- the same config layer sets both `materialization_profile` and one of `copy_files`, `symlink_files`, or `post_create_hook`
- a named profile defines none of `copy_files`, `symlink_files`, or `post_create_hook`

## Trust Model

Both repo-local `.ww.toml` and user-owned global `config.toml` are treated as **trusted input**, the same trust model as `.gitconfig`. Lifecycle hook values (`pre_create_hook`, `post_create_hook`, `pre_remove_hook`, `post_remove_hook`) are passed directly to `sh -c` without sanitization because they are authored by trusted config owners. Users should review repository-local config before using an untrusted repository, just as they would review `.gitconfig` aliases.

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

For explicit target-repository flows such as `ww create --repo <name>` from a
workspace root or another repo worktree, the repo-local search restarts from
the selected repository's main worktree root. In that mode, `ww` uses the
selected repository root for upward search, fallback directories, sandbox
boundary checks, and global `[[projects]]` matching instead of reusing the
caller's already-loaded repo-local config context.

### Layering and Precedence

When both config layers are present:

1. Load the global config first.
2. Expand any selected `materialization_profile` in the base global config into ordinary materialization fields for that layer.
3. If the global config contains `[[projects]]` blocks, evaluate them against the repository main worktree root in declared order and select the first matching block.
4. Expand any selected `materialization_profile` in the chosen project block into ordinary materialization fields for that layer.
5. Merge the selected project block, if any, onto the base global config.
6. Load the repo-local config second.
7. If the repo-local config selects `materialization_profile`, expand it using the global profile table before applying repo-local precedence.
8. Resolve values per configuration field, not per file:
   - if a field is defined only in global config, use the global value
   - if a field is defined only in the selected project block, use the project value
   - if a field is defined only in repo-local config, use the repo-local value
   - if a field is defined in both the base global config and the selected project block, the project value replaces the base global value completely
   - if a field is defined in repo-local config and any global layer, the repo-local value replaces the global value completely
9. Replacement uses the same rule for every field type:
   - scalar fields replace scalar fields
   - array fields replace the entire array without append or merge behavior
   - hook/script fields replace the entire hook value without composition

There is no implicit deep merge, list append, or hook concatenation.

If no `[[projects]]` block matches the repository main worktree root, the base global config remains unchanged.

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
