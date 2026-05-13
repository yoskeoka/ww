# `ww create` and `ww cd` Can Fail When Started in Parallel for the Same Branch

## Summary

During workflow dogfooding from the workspace root, an AI agent used `multi_tool_use.parallel` to start
`ww create --repo ai-arena plan/dungeon-sidecar-boundary` and `ww cd --repo ai-arena plan/dungeon-sidecar-boundary`
at the same time instead of waiting for `create` to finish first.

- repo: `ai-arena`
- triggering workspace: `vibe-coding-workspace`
- cwd: workspace root `vibe-coding-workspace/`
- commands started in parallel via `multi_tool_use.parallel`:
  - `ww create --repo ai-arena plan/dungeon-sidecar-boundary`
  - `ww cd --repo ai-arena plan/dungeon-sidecar-boundary`
- actual:
  - `ww create --repo ai-arena plan/dungeon-sidecar-boundary` succeeded and reported:
    `/home/yoske/src/github.com/yoskeoka/vibe-coding-workspace/.worktrees/ai-arena@plan-dungeon-sidecar-boundary`
  - `ww cd --repo ai-arena plan/dungeon-sidecar-boundary` failed with:
    `no worktree found for branch "plan/dungeon-sidecar-boundary"`

## Impact

AI agents may try to optimize startup by parallelizing adjacent shell steps. When `ww create` and `ww cd` for the same
branch are started concurrently, the `cd` side can observe a not-yet-created worktree and fail even though `create`
eventually succeeds.

## Notes

- This report records only the observed commands and result.
- It does not claim that `ww create` followed by `ww cd` fails when run sequentially.
