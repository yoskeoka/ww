# gh-aw Review Comments Duplicate on Each Push

## Problem

When pushing multiple commits to a PR branch, gh-aw workflows (plan-review, impl-review, spec-code-sync) create a new PR review on each `synchronize` event. This results in multiple stale reviews cluttering the PR conversation.

## Impact

- Reviewers must scroll past outdated reviews to find the current one
- Confusing when early reviews flagged issues that were already fixed in later pushes

## Proposed Fix

Change from `pulls.createReview()` to a PR comment-based approach with find-and-update:

1. Add a hidden HTML marker to each workflow's comment (e.g., `<!-- gh-aw:plan-review -->`)
2. On each run, search for an existing comment with that marker
3. If found, update it via `issues.updateComment()`; if not, create via `issues.createComment()`

This keeps exactly one comment per workflow per PR, always showing the latest result.

## Alternative

Dismiss old reviews before creating new ones. Keeps the review mechanism but dismissed reviews still show in the timeline.

## Files to Change

- `.github/workflows/plan-review.md` — `submit_pr_review` safe-output script
- `.github/workflows/impl-review.md` — same
- `.github/workflows/spec-code-sync.md` — check if same pattern applies
- Run `gh aw compile` after editing `.md` files
