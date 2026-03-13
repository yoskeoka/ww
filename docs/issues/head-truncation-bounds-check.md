# HEAD truncation assumes 40+ char hash

**Source**: PR #3 review
**File**: `git/git.go:62`
**Severity**: Low

## Description

`current.Head = strings.TrimPrefix(line, "HEAD ")[:7]` will panic if the HEAD value has fewer than 7 characters after the prefix. This shouldn't happen with valid git output, but a bounds check would make the code more robust.

## Fix

```go
head := strings.TrimPrefix(line, "HEAD ")
if len(head) > 7 {
    head = head[:7]
}
current.Head = head
```
