# Lessons Learned

## GitHub Actions Script (actions/github-script) — API Patterns

### L-001: Octokit `listReviews` is not paginated by default

- **Mistake**: Used `octokit.rest.pulls.listReviews({ ... })` and iterated over the result, missing reviews beyond the first page (~30).
- **Pattern**: Mistakenly assumed paginated APIs return all results in one call.
- **Rule**: When iterating over potentially large result sets from GitHub API (reviews, comments, check runs), always use `octokit.paginate.iterator(octokit.rest.pulls.listReviews, { ... })` instead of a direct call.
- **Applied**: Any `actions/github-script` step that loops over `listReviews`, `listComments`, `listCheckRunsForRef`, etc.

---

### L-002: Missing `issues: write` permission when using Issues Comments API

- **Mistake**: Added `issues.listComments()` / `issues.deleteComment()` in a job that only had `pull-requests: write` — the API calls failed at runtime.
- **Pattern**: PR comments and issue comments share the Issues API endpoint; `pull-requests: write` does not grant write access to the Issues API.
- **Rule**: When a `github-script` step calls any `octokit.rest.issues.*` method that writes (createComment, deleteComment, updateComment), add `issues: write` to the job-level `permissions` block.
- **Applied**: All `.github/workflows/*.md` jobs that use Route B (fallback comment find-and-update).

---

### L-003: `r.body` from PR Reviews API can be `null`

- **Mistake**: Called `r.body.includes(marker)` directly, causing a TypeError when the review was submitted without a body.
- **Pattern**: GitHub API returns `null` for optional string fields (body, description, etc.) rather than an empty string.
- **Rule**: Always guard optional string fields from the GitHub API with a nullish fallback: `(r.body || '').includes(marker)`. Never call string methods directly on API response fields without a null check.
- **Applied**: Any `github-script` that inspects `review.body`, `comment.body`, `issue.body`, etc.

---

### L-004: `c.user` can be `null` for deleted or ghost GitHub users

- **Mistake**: Accessed `c.user.login` without null check, causing TypeError for comments left by accounts that were later deleted.
- **Pattern**: The `user` object in any GitHub API response is nullable when the account no longer exists.
- **Rule**: Always guard `user` before accessing its properties: `c.user && c.user.login === 'github-actions[bot]'`. Apply to any loop over comments, reviews, or reactions.
- **Applied**: Any `github-script` step that filters by `c.user.login` or accesses `c.user.type`.

---

### L-005: Old duplicate comments must be deleted, not just ignored

- **Mistake**: Route B found an existing fallback comment and updated it — but if multiple matching comments existed (e.g., from earlier workflow versions), older ones were left behind, still cluttering the PR.
- **Pattern**: "Find and update" logic only works if there is at most one match. When multiple matches are possible (bot pushed before the de-duplication fix was deployed), the oldest ones pile up.
- **Rule**: When implementing find-and-update for bot comments: collect ALL matches, update the **newest** one, and delete the rest (best-effort, wrapped in try/catch). Sort by `id` descending to identify newest.
- **Applied**: Route B fallback comment logic in all `gh-aw` review workflows.
