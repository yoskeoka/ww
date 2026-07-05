# Lessons Learned

Keep this file focused. When the lesson count grows past 10, suggest cleanup so solved or redundant lessons can be removed.

Agent note: do not add a lesson when the same mistake is already tracked by an issue or an approved plan that will fix it.

## L-006: Workspace discovery must ignore worktree sibling markers

- **Mistake**: Treated every `.git`-bearing sibling under the repo parent as a workspace member, which made an existing worktree sibling look like a second repo.
- **Pattern**: Discovery logic counted repository markers without distinguishing main repo checkouts from git worktrees created by the tool itself.
- **Rule**: When scanning candidate workspace members, treat `.git` files that point into `/.git/worktrees/` as worktree checkouts and exclude them from workspace membership checks. Add a test that creates a worktree sibling and verifies it does not flip the repo into workspace mode.
- **Applied**: `workspace/workspace.go`, workspace detection tests, and any future path-discovery logic that scans parent directories for repo members.

---

## L-007: `git branch --merged` marks other-worktree branches with `+`

- **Mistake**: Parsed `git branch --merged <base>` assuming only the current worktree would be marked, which missed branches checked out in a different worktree.
- **Pattern**: Git command output changes based on worktree ownership; the same branch can appear with `*` in the current worktree or `+` when it is active elsewhere.
- **Rule**: When parsing merged-branch output, strip both `*` and `+` prefixes before comparing branch names. Add a test that keeps a branch checked out in a sibling worktree and verifies it still counts as merged.
- **Applied**: `git/git.go::MergedBranches`, worktree status resolution, and any future parsers for branch lists from Git.

---

## L-008: Shared CLI semantics must be decided before widening a reused flag

- **Mistake**: Treated review feedback about `--force` on `ww remove` as a narrow spec-code mismatch, when the real question was whether `ww clean` and `ww remove` are contractually the same deletion operation at different scales.
- **Pattern**: A new bulk command can quietly broaden an existing command's semantics if both commands reuse the same implementation path but the shared flag contract is not made explicit first.
- **Rule**: When one command is intended to mean "bulk application of another command," decide and document whether shared flags are semantically identical before changing either implementation or spec. If the answer is yes, update both contracts together in the same change.
- **Applied**: `ww clean` / `ww remove`, and any future single-item vs bulk command pairs that share flags or deletion semantics.

---

## L-011: Config layering semantics must be stated per key before implementation

- **Mistake**: Started from a file-level "repo-local wins" framing before the user clarified that global/local precedence must be evaluated independently for each config field.
- **Pattern**: Layered config work becomes ambiguous when "override" is discussed without concrete examples for mixed global/local coverage and without stating how arrays or hooks behave.
- **Rule**: For layered config changes in `ww`, specify precedence per config key with explicit mixed-case examples before coding. Treat arrays and hook/script fields with the same full-replacement rule unless an explicit additive design is approved.
- **Applied**: Phase 5 global config work, future project-target/profile layering, and any config feature that combines multiple sources.
