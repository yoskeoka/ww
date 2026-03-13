# Import order in worktree.go

**Source**: PR #3 review
**File**: `worktree/worktree.go:6-8`
**Severity**: Low (style)

## Description

The imports list `internal/config` before `git`:

```go
import (
    ...
    "github.com/yoskeoka/ww/internal/config"
    "github.com/yoskeoka/ww/git"
    "github.com/yoskeoka/ww/validate"
)
```

Standard Go convention is alphabetical order within each import group. Should be:

```go
import (
    ...
    "github.com/yoskeoka/ww/git"
    "github.com/yoskeoka/ww/internal/config"
    "github.com/yoskeoka/ww/validate"
)
```

## Fix

Run `goimports` or reorder manually. Consider adding `goimports` to the CI lint step.
