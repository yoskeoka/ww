# Missing * (glob) in branch name validation

**Source**: PR #3 review
**File**: `validate/validate.go`
**Severity**: Low

## Description

`git check-ref-format` rejects `*` in ref names, but the `BranchName` validation function does not check for it. A branch name containing `*` would pass validation but fail at the git level.

## Fix

Add `*` to the invalid character check:

```go
if strings.Contains(name, "*") {
    return fmt.Errorf("branch name contains invalid character '*': %q", name)
}
```

Also add `"has*glob"` to the invalid test cases in `validate_test.go`.
