# Persistent discovery hints use the user cache directory and fail open

## Status

Accepted on 2026-09-09.

## Context

Each `ww list` or `ww clean` process currently repeats path identity queries and
bounded immediate-child repository validation. Reusing those results can make
the slow environment materially faster, but ordinary Git commands can change
worktrees, branches, remotes, and commits outside `ww`.

The cache therefore needs a location that is non-essential application state,
a format that can be rejected without affecting the command, and a trust
boundary that cannot turn cached paths into a broader discovery scope.

## Decision

- Store persistent discovery hints under `filepath.Join(os.UserCacheDir(),
  "ww")`. Do not use the explicit config directory or a normal temporary
  directory fallback.
- Use application-directory permissions no broader than `0700`, file
  permissions no broader than `0600`, hashed path-derived filenames, and
  atomic same-directory replacement.
- Store only canonical topology paths, sandbox/mode, normalized repository
  paths, and bounded directory/`.git` marker fingerprints. Never store dynamic
  worktree status or cleanability decisions.
- Validate schema, path identity, containment, sandbox boundaries, and
  fingerprints before using an entry. Any resolution, permission, I/O, parse,
  schema, or validation failure is a silent cache miss.
- Pass the optional cache through discovery and preserve the existing
  uncached discovery path when it is absent or unusable.

## Consequences

- **Positive**: separate CLI processes can reuse unchanged topology without
  making cache availability a prerequisite for normal operation.
- **Positive**: configuration and sandbox boundaries remain authoritative;
  cached data cannot add a repository or cause parent traversal.
- **Positive**: atomic replacement keeps concurrent writers safe without a
  lock protocol or stale status data.
- **Negative**: cache writes and fingerprint validation add filesystem work on
  cold invocations, so budgeted benchmarks must keep a cold profile and warm
  measurements must be reported separately.
- **Negative**: topology changes and marker identity changes invalidate hints
  and return to the normal Git-backed path.
