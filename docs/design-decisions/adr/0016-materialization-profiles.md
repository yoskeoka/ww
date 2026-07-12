# Materialization profiles expand a single named global profile per config layer

## Status

Accepted on 2026-07-07.

## Context

Phase 5-03 adds reusable materialization patterns so `ww` users can centralize common `copy_files`, `symlink_files`, and `post_create_hook` setups in global config instead of repeating the same payload across repositories. The open design question was whether profiles should be stackable or additive, or whether they should remain a thin expansion layer on top of the existing full-override config model.

## Decision

- Global config may define named `[materialization_profiles.<name>]` entries.
- A config layer may select at most one profile with `materialization_profile = "<name>"`.
- Profile selection expands into ordinary materialization fields for that layer before normal cross-layer precedence is applied.
- The same config layer must not combine `materialization_profile` with direct `copy_files`, `symlink_files`, or `post_create_hook` values.
- `ww` does not stack or merge multiple profiles implicitly.

## Consequences

- **Positive**: Reuse stays explicit and easy to reason about because profile selection is just configuration sugar over the existing resolved fields.
- **Positive**: The established full-replacement rule remains intact for arrays and hooks, so users do not need a second mental model for profile-specific merge behavior.
- **Negative**: Users who want to combine multiple reusable patterns must duplicate or factor them manually until a future explicit composition design exists.
