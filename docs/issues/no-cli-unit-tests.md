# No unit tests for cmd/ww/ CLI wiring

**GitHub:** https://github.com/yoskeoka/ww/issues/9
**Type:** enhancement | **Priority:** Low

## Problem

The `cmd/ww/` package has no test files. Integration tests cover end-to-end behavior, but individual subcommand flag parsing, error handling, and output formatting aren't unit-tested.

**File:** `cmd/ww/*.go` — all `_test.go` files are missing

## Proposed Solution

Add unit tests for flag parsing edge cases, `newManager()` error paths, `outputJSON()` formatting, and `printCommands()` output. Use a test `io.Writer` to capture output without shelling out.
