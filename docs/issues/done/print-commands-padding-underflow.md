# printCommands padding can underflow

**Source**: PR #3 review
**File**: `cmd/ww/main.go:92`
**Severity**: Low

## Description

`strings.Repeat(" ", 14-len(cmd.name))` will panic if a command name exceeds 14 characters. Currently safe (longest is "version" at 7 chars) but fragile for future additions.

## Fix

Use `fmt.Fprintf` with a width specifier (e.g., `%-14s`) or add a max/guard:

```go
pad := 14 - len(cmd.name)
if pad < 1 {
    pad = 1
}
```
