# Git Operations Specification

## Overview

`ww` wraps the `git` CLI for all git operations. It does not use a Go git library. All operations are performed by shelling out to the `git` binary via `os/exec`.

## Git Wrapper (`git/`)

### Interface

The `Runner` struct provides a thin wrapper around `git` CLI execution.

```go
type Runner struct {
    GitBin string // path to git binary, defaults to "git"
    Dir    string // working directory for git commands
}
```

### Operations

#### `Run(args ...string) (string, error)`

Execute an arbitrary git command. Returns stdout as a string. Returns an error if the command fails (non-zero exit code), wrapping stderr in the error message.

#### `WorktreeAdd(path, branch, base string) error`

Create a new worktree with a new branch:
```
git worktree add -b <branch> <path> <base>
```

#### `WorktreeAddExisting(path, branch string) error`

Create a worktree for an existing branch:
```
git worktree add <path> <branch>
```

#### `WorktreeList() ([]WorktreeEntry, error)`

List all worktrees using porcelain format:
```
git worktree list --porcelain
```

Parse the output into structured entries:
```go
type WorktreeEntry struct {
    Path   string
    Head   string // abbreviated commit hash
    Branch string // e.g., "refs/heads/main" -> "main"
    Bare   bool
}
```

#### `WorktreeRemove(path string) error`

Remove a worktree:
```
git worktree remove <path>
```

#### `BranchDelete(branch string) error`

Delete a local branch:
```
git branch -d <branch>
```

Uses `-d` (safe delete) to prevent deleting unmerged branches. If the branch has unmerged work, git will refuse and the error is surfaced to the user.

#### `BranchExists(branch string) bool`

Check if a local branch exists:
```
git rev-parse --verify refs/heads/<branch>
```

Returns true if exit code is 0.

#### `DefaultBranch() (string, error)`

Detect the default branch:
```
git symbolic-ref refs/remotes/origin/HEAD
```

Parse the output to extract the branch name (e.g., `refs/remotes/origin/main` -> `origin/main`).

#### `Fetch() error`

Fetch from origin:
```
git fetch origin
```

#### `RepoName() (string, error)`

Get the repository name from the toplevel directory:
```
git rev-parse --show-toplevel
```

Returns the basename of the result.

## Error Handling

All git errors include:
- The git command that was run
- The stderr output from git
- The exit code

Errors are wrapped with context to make debugging straightforward.
