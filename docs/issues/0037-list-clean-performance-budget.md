# `ww list` / `ww clean` performance cannot be measured or managed

**Type:** performance | **Priority:** High

## Problem

In real use, `ww list` and `ww clean` can take about ten seconds in a workspace with many secondary worktrees or when non-sandbox discovery scans multiple workspace candidates.

External observation suggests workspace discovery and worktree enumeration are likely contributors, but they are not the only possible cause. The same command path also calculates status: base resolution, merged / patch-equivalent detection, branch remote lookup, and `ls-remote`. Optimizing discovery alone without measurement could target the wrong dominant cost.

## Required resolution

This issue is not resolved by an isolated speedup. It is resolved only when all of the following are true:

- A reproducible non-sandbox workspace fixture uses real git repositories and `git worktree` checkouts.
- Command-boundary benchmarks measure `ww list` and non-destructive `ww clean --dry-run`, with standard Go pprof output available for investigation.
- Fixture construction is excluded from measurements, while repository count, secondary worktree count, and a local remote status path are represented deterministically.
- PR CI evaluates a defined performance budget; an exceeded budget is a visible red advisory check that does not become a branch-protection requirement.
- The initial budget and method, including investigation of the suspected PR and possible cumulative regressions, are documented.

## Non-goals

- Changing workspace discovery or status logic based only on hypothesis rather than benchmark evidence.
- Adding scheduled measurement or automatic GitHub issue creation in the initial implementation. If normal PR measurement consistently exceeds one minute, a separate issue may move the work to daily execution and add issue automation.

## Follow-up direction

`docs/exec-plan/todo/0037-ww-performance-budget.md` plans the benchmark, budget evaluator, and PR advisory CI. A concrete optimization must be split into a separate execution plan only after profiling identifies a hot path.
