# Docker integration seed fixture creation is still chatty

**Type:** performance | **Priority:** Low

## Problem

The shared seed repository is only built once, but the seed creation path still performs many small Docker exec and file-copy operations.

**File:**
- `internal/testutil/workspace.go:71-151` - `createRepoSeed()` still issues multiple `WriteFile`, `git add`, and `git commit` steps
- `internal/testutil/container.go:101-115` - each `WriteFile()` still creates a host temp file and copies it into the container

This is not the main bottleneck anymore, but it still adds avoidable setup latency when the seed is first created.

## Proposed Solution

Reduce the seed build chatter by batching seed file creation and commits, or by generating the seed repository from a tarball/template instead of many small `WriteFile()` calls.

Keep the current seed data and branch layout intact so the integration assertions do not change.
