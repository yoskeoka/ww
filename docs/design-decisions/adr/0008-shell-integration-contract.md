# Phase 3 shell integration contract: explicit path-only interfaces

## Status

Accepted on 2026-03-29.

## Context

Phase 3 adds shell-oriented navigation workflows such as `wcd` wrappers and `cd $(ww create -q feat/x)`. The binary cannot change the parent shell's current working directory directly, so the CLI contract needs to stay composable instead of pretending to mutate shell state.

Several interface shapes were considered:

- **Change default `ww create` text output to path-only**: Rejected — it would silently break existing human-oriented output and make the default UX less clear.
- **Add a path-printing subcommand plus an opt-in quiet mode**: Chosen — it preserves the current defaults while enabling shell composition.
- **Use `--path` / `--print-path` instead of `-q` / `--quiet`**: Rejected — `--path` sounds like an input location flag, and `--print-path` is too verbose for frequent interactive shell use.

## Decision

- Add `ww cd` as a read-only path resolver that prints the absolute worktree path.
- Add `-q` / `--quiet` to `ww create` so shell users can request path-only output explicitly.
- Keep path-oriented success output on `stdout`.
- Keep human-readable context and progress on `stderr` whenever shell-oriented flows are in use.
- Do not claim that the binary itself can change the parent shell's cwd; shell wrappers such as `wcd() { cd "$(ww cd "$@")"; }` are the supported pattern.

## Consequences

- **Positive**: Shell workflows stay explicit and scriptable without breaking the existing default output contract of `ww create`.
- **Positive**: The stdout/stderr split makes command substitution and agent usage predictable.
- **Negative**: There are now two related navigation entry points (`ww cd` and `ww create -q`) instead of one magical default. Accepted — the explicitness is the point.
