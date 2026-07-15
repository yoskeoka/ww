# 0037: `ww list` / `ww clean` Performance Budget
> **Execution**: Use `/execute-task` to implement this plan. After implementation is complete, use `/review-task` to prepare and create the PR.

**Branch:** `plan/ww-performance-budget`

**Addresses:** `docs/issues/0037-list-clean-performance-budget.md`

## Objective

Provide a reproducible end-to-end performance measurement for `ww list` and `ww clean --dry-run` using a non-sandbox workspace fixture containing real git repositories, linked worktrees, and a local remote. Evaluate a defined budget on every pull request; an exceeded budget must be a visible red advisory check that prompts investigation of the likely PR or a cumulative regression, while remaining outside the repository's required branch-protection checks.

Completion requires a reusable fixture, command-boundary `go test -bench` benchmarks, a development entry point for pprof, an initial machine-readable budget, PR advisory CI, and a documented testing contract. Any concrete speedup identified by measurement is out of scope and must be split into a separate plan or issue.

## Context and agreed approach

- Do not assume workspace discovery is the only delay. `workspace.DetectWithOptions` scans the start directory, main repository, parent, and grandparent candidates; each candidate can inspect immediate child repositories.
- In workspace mode, `Manager.List` processes every repository serially. Status calculation for each repository includes `git worktree list`, base resolution, merged and patch-equivalent detection, upstream lookup for unmerged branches, and `ls-remote` once per remote.
- Use Go benchmarks as the primary surface. A shared fixture factory creates the fixture once, and the benchmark invokes the actual CLI only after `b.ResetTimer`. This excludes setup cost and preserves standard `go test` support for `-cpuprofile` and `-memprofile`.
- Do not add an independent fixture-generation script in this phase. Keep the fixture as a reusable Go helper so the benchmark command is also the source of truth for local profiling and regression investigation.
- Benchmark `clean` only with `--dry-run`, exercising enumeration, cleanable filtering, and repository targeting without destructive benchmark iterations.

## Existing Implementation References

- `workspace/workspace.go`
  - `DetectWithOptions`, lines 50-129: initial child scan, non-sandbox main-worktree resolution, and containing-workspace fallback.
  - `detectContainingWorkspace` / `candidateDirs`, lines 131-170: bounded start/main/parent/grandparent scanning.
  - `isContainingWorkspaceRoot` / `reposAtWorkspaceRoot`, lines 173-217: immediate child repository enumeration per candidate.
- `cmd/ww/main.go`
  - `newManagerWithOptions`, lines 144-195: command startup including config pre-load and sandbox re-resolution.
  - `loadManagerContext`, lines 197-224: workspace detection, main worktree resolution, and config load.
- `cmd/ww/sub_list.go`, lines 17-66: `ww list` calls `Manager.List` after manager startup and applies `--cleanable` afterwards.
- `cmd/ww/sub_clean.go`, lines 35-63 and 66-72: `ww clean` obtains cleanable items through the same `Manager.List` path.
- `worktree/worktree.go`
  - `Manager.List` / `listWorkspace`, lines 353-453: serial enumeration of every workspace repository.
  - `listRepo`, lines 456-540: worktree, merge, patch-equivalent, and remote-status aggregation.
  - `listRepoFast`, lines 543-565: an existing status-free path that is a future comparison point, not a command-semantic change in this plan.
- `internal/testutil/host.go`, lines 16-60 and 111-160: host-native binary / git isolation and actual CLI process execution.
- `docs/specs/testing.md`, lines 3-21: current test targets and host-native integration-harness contract.
- `Makefile`, lines 9-22: existing canonical `test` / `test-all` command surface.
- `.github/workflows/ci.yml`, lines 1-19: PR CI and existing required jobs.

## Code Change Map

- `internal/testutil/performance_fixture.go` (NEW)
  - Reuse the host-native integration harness to create and clean up a workspace root, multiple child git repositories, several secondary worktrees per repository, a local bare remote, and upstream branches.
  - Define a fixed named scale profile that substantially exercises discovery and status work, then return read-only metadata such as workspace root and expected counts.
- `cmd/ww/performance_test.go` (NEW)
  - Add actual-binary benchmarks for `list` and `clean --dry-run`.
  - Keep setup outside the timer; measure command startup through completed output and assert expected counts and successful execution.
  - Publish stable `BenchmarkWorkspaceList` and `BenchmarkWorkspaceCleanDryRun` names, usable with `-benchmem`, `-cpuprofile`, and `-memprofile`.
- `tools/check-performance-budget.go` (NEW)
  - Run the canonical benchmark command a fixed number of times, parse each command's `ns/op`, and compare results with the budget file.
  - On an exceeded budget, print the benchmark name, observed value, budget, and reproduction command before returning non-zero. Distinguish an evaluator/fixture failure from a budget regression.
- `tools/performance-budget.json` (NEW)
  - Store the fixture profile, benchmark command, repetition count, and each command's initial budget in reviewable version control. Set values only from repeated baseline runs on GitHub-hosted runners with sufficient noise margin.
- `Makefile` (MODIFY)
  - Add canonical `make perf-check` and a development-only `make perf-profile` accepting a profile artifact path. Do not implicitly add benchmarks to existing `test` or `test-all` commands.
- `.github/workflows/ci.yml` (MODIFY)
  - Add a dedicated pull-request performance job running `make perf-check`.
  - Let a budget exceedance fail that job visibly. Keep it advisory by ensuring repository branch protection does not require this check; do not use `continue-on-error`, which would obscure a true regression.
  - Record the job duration in the job summary. If baseline data shows normal PR runs consistently exceed one minute, create a separate follow-up for daily scheduling and GitHub issue creation; do not add scheduling or automation in this change.
- `docs/specs/testing.md` (MODIFY)
  - Describe the fixture realism, measurement boundary, pprof command, budget contract, advisory semantics, and one-minute follow-up policy as a black-box testing contract.
- `docs/issues/0037-list-clean-performance-budget.md` (MODIFY)
  - After baseline adoption, record the measured values and any resulting optimization or daily-migration issue; move the issue to `done/` only after every completion condition is met.

## Spec Changes

Extend `docs/specs/testing.md` with the following black-box contract:

- Performance coverage uses a host fixture with actual git repositories, linked worktrees, and a local remote rather than a fake git API.
- `ww list` and `ww clean --dry-run` benchmarks emit comparable command-boundary elapsed time with fixture setup excluded.
- Developers can collect standard Go CPU and memory profiles through a canonical Make target.
- PR CI compares each benchmark with a version-controlled budget. Passing measurements are green; an exceeded budget is a visible red advisory check that prompts review but is not added to branch protection's required checks.
- The fixture profile, runner, and repetition count are versioned with the budget. Only if normal PR execution is consistently slower than one minute should a separate follow-up consider daily measurement and automatic GitHub issue creation.

## Sub-tasks

1. Fixture and benchmark foundation
   - Design a performance-fixture API that reuses `HostEnv` git-config isolation and actual-binary execution.
   - Create a multi-repository workspace with a local bare remote and several secondary worktrees. Assert that a child-repository start directory uses non-sandbox workspace discovery.
   - Fix the named fixture profile and status shape (merged, active, and upstream-tracking branches), and create it outside the benchmark timer.
   - Add `list` and `clean --dry-run` benchmarks that assert fixture-specific output and exit status.

2. Baseline, profile, and budget evaluator
   - Run the benchmark repeatedly on a GitHub-hosted Ubuntu runner and record baseline variation for each command.
   - Add canonical Makefile support for local pprof collection.
   - Define a non-flaky initial budget and fixed repetition count from baseline evidence rather than a guessed constant.
   - Test that the evaluator distinguishes within-budget results, budget exceedance, missing benchmark output, and fixture failure.

3. Advisory PR CI and documentation
   - Add the pull-request performance job and verify that it is visibly red on an intentional budget exceedance without becoming a required branch-protection check or blocking the existing `test` job.
   - Update `docs/specs/testing.md` with the measurement, profile, budget, advisory, and one-minute fallback policy.
   - Inspect the implementation PR job duration. Keep PR execution if normal duration is within one minute; otherwise create a separate issue for daily measurement and GitHub issue automation.

## Dependencies and Parallelism

- Sub-task 1's fixture API and stable benchmark names are prerequisites for the other work.
- Baseline collection and evaluator work follow the fixture/benchmark foundation.
- Once the budget-file and evaluator interface are settled, CI workflow and documentation can proceed in parallel.
- This plan establishes measurement and management only. A profile-identified optimization must use a separate plan.

## Verification

- `make test`
- `make test-all`
- `make lint`
- Run `make perf-check` multiple times and confirm fixture setup is excluded while both benchmarks compare against the budget.
- Run `make perf-profile`, confirm CPU and memory profiles are written to the requested locations, and inspect them with `go tool pprof`.
- Cover evaluator handling of within-budget values, budget exceedance, missing output, and fixture failure.
- In a PR workflow, verify the performance job runs on pull requests only and that an intentional budget exceedance is visibly red but does not enter the required-check set or block the existing `test` job.
