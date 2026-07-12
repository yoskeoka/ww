# Global config uses explicit user path with per-key local override

## Status

Accepted on 2026-06-02.

## Context

Phase 5 adds user-owned global config so `ww` workflows do not require every repository to commit `.ww.toml`. The open design questions were where that global config should live, whether repo-local config should still win, and whether matching values should be merged or replaced.

## Decision

- Global config lives at `$XDG_CONFIG_HOME/ww/config.toml` when `XDG_CONFIG_HOME` is set, otherwise `$HOME/.config/ww/config.toml`.
- Repo-local `.ww.toml` remains the higher-priority layer.
- Precedence is evaluated per configuration field, not per file.
- When both layers define the same field, the repo-local value replaces the global value completely.
- Complete replacement applies uniformly to scalars, arrays, and hook/script fields. `ww` does not append arrays or compose hook strings implicitly.
- Sandbox mode continues to honor the explicit global config path. Sandbox boundaries only constrain repo-local upward search and fallback directories.

## Consequences

- **Positive**: The mental model stays simple: global config provides defaults, repo-local config overrides only the fields it mentions, and each overridden field is fully replaced.
- **Positive**: Existing repositories keep their current `.ww.toml` authority and behavior when no global config exists.
- **Positive**: Hook/materialization behavior stays deterministic because there is no hidden merge logic.
- **Negative**: Users must repeat a full array or hook value when overriding only part of it.
- **Negative**: Future additive reuse needs explicit syntax instead of piggybacking on implicit merge rules.
