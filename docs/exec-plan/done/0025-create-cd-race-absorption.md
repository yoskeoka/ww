# 0025: Create/Cd Race Absorption in `ww cd`
**Execution**: Use `/execute-task` to implement this plan.

Addresses: `docs/issues/done/ww-create-and-cd-run-in-parallel-can-fail.md`

## Objective

Reduce the impact of `ww create` / `ww cd` startup races for the same repo and
branch without depending on every caller or AI agent to serialize those steps
correctly.

This plan covers only the product-side mitigation for the observed
`multi_tool_use.parallel` failure by making named `ww cd` lookups more tolerant
of a narrow just-created race window.

## Context

- `docs/issues/ww-create-and-cd-run-in-parallel-can-fail.md` records the real
  incident: `ww create --repo ai-arena plan/dungeon-sidecar-boundary` and
  `ww cd --repo ai-arena plan/dungeon-sidecar-boundary` were started at the
  same time, `create` succeeded, and `cd` failed with
  `no worktree found for branch "plan/dungeon-sidecar-boundary"`.
- `docs/project-plan.md` says `ww` is AI-agent friendly by default. Requiring
  each caller to learn and restate sequencing rules is a poor fit for that goal.

## Expected Outcome

- `ww cd <branch>` tolerates a narrow just-created race window instead of
  failing immediately when the worktree registration appears moments later.
- The retry budget is explicit and reviewable in advance: named `ww cd <branch>`
  performs the normal immediate lookup first, then retries up to 5 additional
  times with 100ms intervals before returning the existing error, for a maximum
  added wait of 500ms.
- The real observed race is covered by regression tests with a bounded,
  intentional retry contract rather than an unbounded poll loop.

## Scope

- Add a bounded retry path to `ww cd <branch>` for the specific
  "just created but not yet discoverable" case.
- Add regression coverage for the parallel-start incident shape.

## Out of Scope

- General-purpose cross-process locking or persistent coordination files
- Broad changes to workspace detection or worktree layout
- Requiring new flags such as `--wait` for the default safe path
- Help / README / command-guidance improvements about create-and-enter flows

## Code Changes

- Update `cmd/ww/sub_cd.go` and any related helper paths so named `ww cd`
  performs a short, bounded retry before returning
  `no worktree found for branch "<branch>"`.
- If needed, factor the retry logic into a small helper in `cmd/ww/` or
  `worktree/` without broad refactoring.
- Add or extend automated coverage in:
  - `integration_test.go`
  - any focused unit tests near the chosen retry helper

## Spec Changes

- Update `docs/specs/cli-commands.md`:
  - `ww cd [branch]` should document the bounded retry behavior for a
    just-created named worktree lookup
- Update `docs/specs/shell-integration.md` only as needed to record the retry
  behavior for named `ww cd` lookups.

## Design Notes

- Keep the retry bounded and small. The goal is to absorb a narrow race window,
  not to hide genuine lookup errors indefinitely.
- Recommended contract for this plan:
  - apply only to named `ww cd <branch>` lookups
  - keep the current immediate first lookup
  - after a named lookup misses, retry up to 5 times with 100ms intervals
  - preserve the current error text after the retry budget is exhausted
  - do not apply the retry to no-argument recency lookup
- Prefer the retry only for named `ww cd <branch>` lookups. No-argument
  recency lookup should stay simple unless the investigation shows the same race
  applies there in practice.
- Preserve current error text after the retry budget is exhausted so existing
  failure handling remains understandable.

## Sub-tasks

- [x] Confirm the bounded retry contract stays:
  immediate first lookup, then up to 5 retries with 100ms intervals for named
  `ww cd <branch>` misses only
- [x] [parallel] Update `docs/specs/cli-commands.md` and
  `docs/specs/shell-integration.md` with the retry contract
- [x] [depends on: retry contract] Implement the bounded retry in the named
  `ww cd` path
- [x] [depends on: implementation] Add regression coverage for a parallel
  `create` / `cd` startup on the same branch and repo
- [x] [depends on: implementation] Confirm normal `ww cd` failures still return
  promptly for a truly missing branch after the bounded retry budget ends

## Verification

- `go test ./...`
- Focused test coverage for:
  - successful named `ww cd` after a just-created race window
  - still-failing named `ww cd` for a genuinely missing branch
- Manual smoke check:
  - `cd "$(ww create -q feat/plan-check)"`
  - `ww cd feat/plan-check`

## Risks

- An overly long retry budget would make genuine user mistakes feel sluggish.
- An overly short retry budget would not materially improve the observed race.
- If the retry is placed too low in shared worktree lookup logic, it could
  unintentionally affect unrelated commands.
