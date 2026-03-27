# Docker integration tests still pay per-test clone overhead

**Type:** performance | **Priority:** Medium

## Problem

The integration suite now runs tests in parallel, but each test still creates a fresh repository by cloning the shared seed repository in `SetupRepo()`.

**Files:**
- `internal/testutil/workspace.go:41-166` - `SetupRepo()` still clones a repo per test via `cloneRepoSeed()`
- `integration_test.go:37-39` - every Docker integration test goes through `setupRepo()`

This is much faster than rebuilding the repo from scratch, but it is still a measurable per-test cost and remains the largest remaining setup bottleneck.

## Proposed Solution

Replace the per-test `git clone` with a cheaper materialization strategy, such as:
- `git worktree add` from a shared bare seed repo, or
- a local template/copy mechanism that avoids full clone metadata

Measure the result against the current clone-based path before choosing the implementation.
