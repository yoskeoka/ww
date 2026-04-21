# interactive PTY smoke tests time out waiting for prompt

## Summary

`make test-all` can fail in host integration tests because the interactive PTY
smoke tests time out waiting for the initial `Select action` prompt while
captured stderr is empty.

Observed failures:

- `TestInteractivePTYQuitSmoke`: timed out waiting for `Select action`
- `TestInteractivePTYListOpenSmoke`: timed out waiting for `Select action`

This was observed while executing
`fix/worktree-remove-fails-with-submodules` after unrelated shell-quoting
changes in `worktree/worktree.go`.

## Impact

The failure blocks full `make test-all` verification even when focused packages,
short tests, and lint pass. Because the timeout occurs before any task-specific
submodule removal path is exercised, it appears unrelated to the submodule
worktree removal fix.

## Proposed Solution

Investigate the PTY harness and prompt rendering path used by
`integration_pty_test.go` and `internal/testutil/pty_unix.go`.

Concrete checks:

- Confirm whether `ww i` is starting and exiting early before prompt rendering.
- Capture PTY output incrementally when `WaitForOutput` times out so failures
  include enough context.
- Review whether the 5 second prompt wait is too aggressive under concurrent
  host integration load.
- If the prompt can legitimately render later, make the PTY smoke tests use a
  more robust bounded wait without weakening the non-TTY guards.

## Priority

Medium. The issue affects full integration confidence and PR verification, but
it is isolated to interactive PTY smoke coverage and does not indicate a
submodule removal regression.
