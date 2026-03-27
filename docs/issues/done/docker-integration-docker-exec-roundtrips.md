# Docker integration helpers still make many exec round-trips

**Type:** performance | **Priority:** Medium

## Problem

The test helpers still cross the Docker boundary for nearly every repository interaction.

**Files:**
- `internal/testutil/container.go:145-206` - `Exec()` is used for every `git`, `ww`, `test`, and `cat` call
- `internal/testutil/workspace.go:56-149` - repo setup and assertions rely on many small helper calls

Parallel execution hides some of the latency, but the helper layer still pays a separate `docker exec` for each command and each filesystem check.

## Proposed Solution

Batch where possible:
- combine repeated shell checks into one helper call
- prefer one repo-level command that verifies multiple outcomes at once
- reduce `ReadFile` / `PathExists` polling when an existing command output already proves the state

Do not remove the current helpers unless a replacement keeps the tests equally clear and reliable.
