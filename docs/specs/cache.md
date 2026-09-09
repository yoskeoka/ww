# Persistent workspace-discovery cache

## Purpose

`ww` may persist validated workspace-discovery topology hints so separate
`ww list` and `ww clean` processes can avoid repeating unchanged path and
workspace-membership discovery. The cache is an optional optimization. A cache
miss must produce the same discovery result, command output, errors, and
sandbox boundary as an uncached invocation.

## Storage and permissions

- The cache root is `filepath.Join(os.UserCacheDir(), "ww")`.
- Unix uses `$XDG_CACHE_HOME/ww` when `XDG_CACHE_HOME` is set, otherwise
  `$HOME/.cache/ww`; other platforms follow Go's `os.UserCacheDir` contract.
- The cache is not stored below `XDG_CONFIG_HOME`, and normal execution does
  not fall back to `os.TempDir()`.
- The application directory is created with permissions no broader than `0700`
  and cache files with permissions no broader than `0600`.
- Each topology entry uses a path-derived hashed filename. Cache entries are
  replaced with an atomic same-directory rename, so concurrent writers are
  last-writer-wins and readers see either the old or new complete entry.
- If the user-cache directory cannot be resolved, created, read, written, or
  trusted, `ww` continues without caching and emits no cache diagnostic on
  normal stdout or stderr.

## Entry schema and trust boundary

An entry is versioned JSON containing only topology hints:

- canonical start-directory, current-worktree, main-worktree, and selected
  workspace-root paths
- sandbox mode and single-repo/workspace mode
- the deterministic normalized repository name/path list
- sorted immediate-child and `.git` marker fingerprints for the bounded
  candidates used during discovery

The entry must not contain worktree branch/head values, merged/stale/active
status, patch-equivalence results, or cleanable decisions. Those values remain
fresh Git-backed data on every command.

Before a hit is used, `ww` verifies that:

- the schema version is supported and the entry is valid JSON
- the current canonical start path and sandbox mode match the entry
- every cached path still exists, remains within the same bounded discovery
  candidates and sandbox boundary, and has not escaped through a symlink
- candidate directory entries and `.git` marker fingerprints still match
- repository names and paths remain normalized, deterministic, and contained
  by the cached workspace root

Any corrupt, unknown-version, stale, tampered, moved, deleted, or mismatched
entry is silently ignored. `ww` then runs the normal discovery algorithm and
may replace the entry with a newly validated hint. Cache data never expands
the paths that uncached discovery is allowed to inspect.

## Observable behavior

Successful cache hits return the same `workspace.Workspace` as the equivalent
uncached discovery. A hit may only skip discovery of the current worktree,
main worktree, workspace root, and unchanged immediate-child repository set;
`ww list` and `ww clean` still load current worktree/branch/head/status data.

The cache is shared across short-lived `ww` invocations. Tests redirect the
user-cache environment to test-owned temporary state and do not read or write
the operator's real cache.
