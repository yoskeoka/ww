# post_create_hook: display hook command before execution output

GitHub: https://github.com/yoskeoka/ww/issues/37

## Problem

When `post_create_hook` runs during `ww create`, the hook's output appears without context. For example, with `post_create_hook = "git submodule update --init --recursive"`, the output looks like:

```
Cloning into '.../.claude/vendor/workflow'...
Submodule path '.claude/vendor/workflow': checked out 'd6590ea...'
Created worktree at /path/to/ww@feat-branch (branch: feat/branch)
```

It's unclear what triggered the clone output.

## Proposal

Display the hook command before its output:

```
Running post_create_hook: git submodule update --init --recursive
Cloning into '.../.claude/vendor/workflow'...
Submodule path '.claude/vendor/workflow': checked out 'd6590ea...'
Created worktree at /path/to/ww@feat-branch (branch: feat/branch)
```

## Priority

Low — cosmetic improvement for UX clarity.
