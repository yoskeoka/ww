# Child repos are never workspace roots (no recursive nesting)

## Status

Accepted on 2026-03-18.

## Context

If a child repo could itself be treated as a workspace root (e.g., a child that has its own git children), workspace detection could recurse indefinitely or produce ambiguous results.

## Decision

Child repos are never treated as workspace roots. Detection stops at one level. This is an explicit invariant enforced in the detection algorithm.

## Consequences

- **Positive**: Deterministic detection, no ambiguity, simple mental model.
- **Negative**: Nested workspace-in-workspace layouts are not supported. Accepted — FR-22 reserves this for the future.
