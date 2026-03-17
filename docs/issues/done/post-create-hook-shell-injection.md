# post_create_hook shell injection risk

**Source**: PR #3 review
**File**: `worktree/worktree.go:282`
**Severity**: Low (config is user-controlled)

## Description

The `post_create_hook` value is passed directly to `sh -c` without sanitization. Since this comes from the user's own `.ww.toml` config file (similar to `.gitconfig` aliases), this is acceptable for now, but it should be documented that the config file is trusted input.

## Action

- Document in `docs/specs/configuration.md` that `.ww.toml` is treated as trusted input (same trust model as `.gitconfig`).
- Consider whether shared/workspace-level configs need any additional safeguards in future multi-repo support.
