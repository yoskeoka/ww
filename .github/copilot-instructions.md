# Copilot Instructions for ww

## Spec-Code Parity

This project follows strict spec-code parity. When reviewing or suggesting changes:

- Code must match the specifications in `docs/specs/`.
- See `docs/spec-code-mapping.md` for which specs map to which code directories and test files.
- If code changes are proposed, check whether the corresponding spec needs updating.
- If spec changes are proposed, check whether the corresponding code/tests need updating.

## Project Context

- `ww` is a git worktree manager CLI written in Go.
- Execution plans live in `docs/exec-plan/todo/` while active; completed plans are deleted and retrievable from the implementation PR or Git history.
- Issues are tracked in `docs/issues/` while unresolved; resolved linked issues are deleted with the matching plan.
- Design decisions are indexed in `docs/design-decisions/README.md` as numbered ADRs under `docs/design-decisions/adr/`, with `docs/design-decisions/core-beliefs.md` holding the fundamental principles.

## Project Structure

See Project Structure section in `AGENTS.md` for an overview of the code organization.
