# Lessons Learned

## GitHub Actions Script (actions/github-script) — API Patterns

### L-001: Octokit `listReviews` is not paginated by default

- **Mistake**: Used `octokit.rest.pulls.listReviews({ ... })` and iterated over the result, missing reviews beyond the first page (~30).
- **Pattern**: Mistakenly assumed paginated APIs return all results in one call.
- **Rule**: When iterating over potentially large result sets from GitHub API (reviews, comments, check runs), always use `octokit.paginate.iterator(octokit.rest.pulls.listReviews, { ... })` instead of a direct call.
- **Applied**: Any `actions/github-script` step that loops over `listReviews`, `listComments`, `listCheckRunsForRef`, etc.

---

### L-002: Missing `issues: write` permission when using Issues Comments API

- **Mistake**: Added `github.rest.issues.listComments()` / `github.rest.issues.deleteComment()` in a job that only had `pull-requests: write` — the API calls failed at runtime.
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
- **Rule**: When implementing find-and-update for bot comments: collect ALL matches, update the **newest** one, and delete the rest (best-effort, wrapped in try/catch). Sort by `id` ascending and treat the last element as newest.
- **Applied**: Route B fallback comment logic in all `gh-aw` review workflows.

---

### L-006: Workspace discovery must ignore worktree sibling markers

- **Mistake**: Treated every `.git`-bearing sibling under the repo parent as a workspace member, which made an existing worktree sibling look like a second repo.
- **Pattern**: Discovery logic counted repository markers without distinguishing main repo checkouts from git worktrees created by the tool itself.
- **Rule**: When scanning candidate workspace members, treat `.git` files that point into `/.git/worktrees/` as worktree checkouts and exclude them from workspace membership checks. Add a test that creates a worktree sibling and verifies it does not flip the repo into workspace mode.
- **Applied**: `workspace/workspace.go`, workspace detection tests, and any future path-discovery logic that scans parent directories for repo members.

---

### L-007: `git branch --merged` marks other-worktree branches with `+`

- **Mistake**: Parsed `git branch --merged <base>` assuming only the current worktree would be marked, which missed branches checked out in a different worktree.
- **Pattern**: Git command output changes based on worktree ownership; the same branch can appear with `*` in the current worktree or `+` when it is active elsewhere.
- **Rule**: When parsing merged-branch output, strip both `*` and `+` prefixes before comparing branch names. Add a test that keeps a branch checked out in a sibling worktree and verifies it still counts as merged.
- **Applied**: `git/git.go::MergedBranches`, worktree status resolution, and any future parsers for branch lists from Git.

---

### L-008: Shared CLI semantics must be decided before widening a reused flag

- **Mistake**: Treated review feedback about `--force` on `ww remove` as a narrow spec-code mismatch, when the real question was whether `ww clean` and `ww remove` are contractually the same deletion operation at different scales.
- **Pattern**: A new bulk command can quietly broaden an existing command's semantics if both commands reuse the same implementation path but the shared flag contract is not made explicit first.
- **Rule**: When one command is intended to mean "bulk application of another command," decide and document whether shared flags are semantically identical before changing either implementation or spec. If the answer is yes, update both contracts together in the same change.
- **Applied**: `ww clean` / `ww remove`, and any future single-item vs bulk command pairs that share flags or deletion semantics.

---

### L-009: Follow-up execution must re-check whether the prerequisite PR is already merged

- **Mistake**: Continued working from the assumption that PR `ww#87` was still pending, and only the user's correction made it explicit that the plan PR had already merged.
- **Pattern**: Carrying state forward from the previous turn without re-verifying merge status can make workflow decisions stale, especially when the user has advanced the repository between turns.
- **Rule**: At the start of any `/execute-task` follow-up, explicitly verify whether the referenced plan PR has already merged or whether `origin/main` already contains the prerequisite docs move, before deciding whether more planning work is needed.
- **Applied**: Workflow transitions between `/plan-execution` and `/execute-task`, especially when a user references a recently created PR or says "it's already merged".

---

### L-010: Do not plan around unstable git internals when stable `rev-parse` contracts exist

- **Mistake**: Considered using raw `.git` file-pointer inspection as part of the linked-worktree exclusion approach, even though the same distinction could be expressed through stable `git rev-parse` outputs.
- **Pattern**: Reaching for filesystem implementation details first can make both plans and code depend on git's current storage format rather than its supported command interface.
- **Rule**: When distinguishing repository states in `ww`, prefer stable git CLI contracts such as `rev-parse --show-toplevel`, `--git-dir`, and `--git-common-dir` over inspecting `.git` file contents directly. Only rely on raw `.git` internals when no stable git command can express the required distinction.
- **Applied**: Workspace-member detection, linked-worktree exclusion, and any future repo-shape validation logic.
