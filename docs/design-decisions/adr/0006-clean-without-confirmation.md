# `ww clean` has no confirmation prompt

## Status

Accepted on 2026-03-18.

## Context

Most CLI tools that delete data ask for confirmation (`rm -i`, `git clean -i`). However, `ww clean` targets a specific audience (developers and AI agents) and has explicit preview mechanisms.

## Decision

`ww clean` executes immediately without a confirmation prompt. Users preview with `ww list --cleanable` or `ww clean --dry-run`.

## Consequences

- **Positive**: Scriptable, agent-friendly, no interactive input required. Consistent with `ww remove` which also has no prompt.
- **Negative**: Risk of accidental deletion. Mitigated by safe defaults (`git branch -d` fails on unmerged branches) and `--force` being opt-in.
