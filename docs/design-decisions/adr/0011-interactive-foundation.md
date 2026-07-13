# Phase 4 interactive foundation: lightweight prompt surface with strict CLI parity

## Status

Accepted on 2026-04-08.

## Context

Phase 4 adds a human-oriented interactive entry point, `ww i`. The main risk is allowing the interactive UI to drift into a separate capability surface with behavior that cannot be expressed through the normal CLI. That would weaken `ww`'s AI/scripting contract and create two inconsistent execution models.

The earlier Phase 3 shell integration ADR established a strict stream contract: machine/path-oriented output belongs on `stdout`, and human-readable context belongs on `stderr`. The interactive mode foundation needs to preserve that contract while introducing a prompt-based flow.

Several interaction models were considered:

- **Full-screen TUI**: Rejected for the MVP. It is heavier than needed and adds more UI state than the current scope requires.
- **Interactive-only behavior**: Rejected. Any action that mutates state or selects an externally meaningful result must remain available through the standard CLI.
- **Lightweight prompt flow (chosen)**: Use `ww i` as a guided entry point over existing CLI behavior, with fixed top-level actions and explicit stream routing.

For prompt tooling, the project direction is to use `huh` for the interactive flow implementation because it fits grouped, guided prompts better than single-prompt libraries. The foundation step keeps the session and dispatch contracts independent from any single prompt adapter so later child plans can wire `huh` without changing the command contract.

## Decision

- `ww i` is a lightweight prompt flow, not a separate full-screen application.
- Interactive mode is an orchestration layer over existing command/business logic and must obey strict non-interactive CLI parity.
- The fixed MVP top-level actions are `create`, `list`, `clean`, and `quit`.
- Interactive prompt rendering and human-readable context use `stderr`.
- `stdout` remains reserved for path-only or machine-consumable action results.
- `ww i --json` is rejected; users needing machine-readable output must use the standard non-interactive commands.
- The implementation direction for prompt UI is `huh`, but the foundation layer keeps session abstractions small so prompt-library changes do not alter the public CLI contract.

## Consequences

- **Positive**: Interactive mode stays additive and cannot become the only place where a capability exists.
- **Positive**: The Phase 3 shell contract survives intact, including future path-only `stdout` output for interactive `open`.
- **Positive**: The fixed action model and shared session abstractions make later child plans testable without requiring a real terminal in unit tests.
- **Negative**: The foundation step adds placeholder dispatch before the full child flows exist.
- **Negative**: Keeping parity with the non-interactive CLI may delay interactive UX ideas that require new command flags or commands first.
