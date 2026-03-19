# gh-aw PR Reviews Duplicate on Each Push

## Problem

When pushing multiple commits to a PR branch, gh-aw workflows (plan-review, impl-review, spec-code-sync) create a new PR review on each `synchronize` event. This results in multiple stale reviews cluttering the PR conversation.

## Impact

- Reviewers must scroll past outdated reviews to find the current one
- Confusing when early reviews flagged issues that were already fixed in later pushes

## Proposed Fix

Both routes need fixing — the review submission path (happy path) and the comment fallback path (when review submission fails).

### Route A: Review submission (happy path)

Before creating a new review, dismiss existing reviews from `github-actions[bot]`:

1. List reviews on the PR via `pulls.listReviews()`
2. Filter to reviews by `github-actions[bot]`
3. Dismiss each via `pulls.dismissReview()` with a message like "Superseded by updated review"
4. Create the new review via `pulls.createReview()`

This is the primary fix. Dismissed reviews remain in the timeline but are visually collapsed.

### Route B: Comment fallback (when review submission fails)

The existing fallback creates a comment via `issues.createComment()`. Update this to find-and-update:

1. Add a hidden HTML marker to each workflow's comment (e.g., `<!-- gh-aw:plan-review -->`)
2. On each run, use the GitHub REST API to list all PR comments (with pagination) and filter them client-side for that marker to find an existing comment ID
3. If found, update it via `issues.updateComment()`; if not, create via `issues.createComment()`

This keeps exactly one fallback comment per workflow per PR.

## Files to Change

- `.github/workflows/plan-review.md` — `submit_pr_review` safe-output script (both routes)
- `.github/workflows/impl-review.md` — same
- `.github/workflows/spec-code-sync.md` — check if same pattern applies
- Run `gh aw compile` after editing `.md` files
