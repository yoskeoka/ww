# Copilot Instructions for ww

## Spec-Code Parity

This project follows strict spec-code parity. When reviewing or suggesting changes:

- Code must match the specifications in `docs/specs/`.
- See `docs/spec-code-mapping.md` for which specs map to which code directories and test files.
- If code changes are proposed, check whether the corresponding spec needs updating.
- If spec changes are proposed, check whether the corresponding code/tests need updating.

## Project Context

- `ww` is a git worktree manager CLI written in Go.
- Execution plans live in `docs/exec-plan/todo/` (active) and `docs/exec-plan/done/` (completed).
- Issues are tracked in `docs/issues/` (active) and `docs/issues/done/` (resolved).
