# `ww cd` Cannot Find a Just-Created Worktree

## Summary

During workflow dogfooding from the workspace root, `ww create <branch>` created the worktree successfully, but `ww cd <branch>` from the same cwd immediately failed to resolve that same worktree.

- repo: `ww`
- triggering workspace: `vibe-coding-workspace`
- cwd: workspace root `vibe-coding-workspace/`
- expected:
  - `ww create feat/kb-bilingual-rendering` succeeds
  - `ww cd feat/kb-bilingual-rendering` immediately returns the created worktree path
- actual:
  - `ww create feat/kb-bilingual-rendering` succeeded and reported `/home/yoske/src/github.com/yoskeoka/vibe-coding-workspace/.worktrees/vibe-coding-workspace@feat-kb-bilingual-rendering`
  - `ww cd feat/kb-bilingual-rendering` then failed with `no worktree found for branch "feat/kb-bilingual-rendering"`

## Impact

This breaks the documented `ww create` -> `ww cd` handoff used by the workspace workflow and weakens shell-oriented task startup for Git-backed workspace roots.

## External Context

The original dogfooding report was first captured in:

- `vibe-coding-workspace/docs/issues/ww-cd-cannot-find-just-created-worktree.md`

That workspace issue should be treated as the source report. This `ww` issue tracks the product-side investigation and fix work inside the `ww` repository.
