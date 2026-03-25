# Roadmap Update: Phase 2 Completion and Interactive Mode

> **Execution**: Use `/execute-task` to implement this plan.

**Objective:** Update the `ww` project roadmap to reflect Phase 2 completion and insert a new roadmap phase for a human-oriented interactive mode.

## Context

Phase 2 workspace features are implemented and verified in completed exec-plans `007`, `009`, `010`, and `011`, but `docs/project-plan.md` still marks Phase 2 as incomplete.

The roadmap also needs a new phase between the existing Phase 3 and the current nice-to-have Phase 4. The new phase captures a human-oriented interactive mode so `ww` serves both AI-agent workflows and direct human operation.

Past decisions reviewed before this change:

- `core-beliefs.md`: keep AI-first context, but do not block additional human UX when it stays aligned with correctness and spec-first workflow.
- `adr.md`: no prior ADR conflicts with adding a human-interactive phase; existing decisions remain compatible because the CLI stays git-native and workspace-aware.

## Docs Changes

| File | Change |
|------|--------|
| `docs/project-plan.md` | Mark Phase 2 complete, add a new Phase 4 for human interactive mode, renumber the existing nice-to-have phase, and add a requirement entry for interactive mode |

## Spec Changes

None in this plan. This is a roadmap and requirement update only.

## Code Changes

None in this plan.

## Sub-tasks

- [ ] Update `docs/project-plan.md` milestones to mark Phase 2 complete
- [ ] Add a new roadmap phase between current Phase 3 and current Phase 4 for human interactive mode
- [ ] Add or update functional requirements so the new phase has explicit scope instead of milestone-only wording
- [ ] Verify the roadmap still reads coherently after phase renumbering

## Parallelism

No meaningful parallel work. The project-plan update is a single coherent document change.

## Verification

- `docs/project-plan.md` reflects the implemented Phase 2 status
- The roadmap contains a distinct Phase 4 for human interactive mode
- The former nice-to-have Phase 4 is renumbered consistently
